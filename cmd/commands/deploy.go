package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/controlplane"
	"github.com/nself-org/cli/internal/deploy/bluegreen"
	"github.com/nself-org/cli/internal/maintenance"
	"github.com/nself-org/cli/internal/promote"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// remotePathRe allows safe remote path characters: alphanumeric, slash, hyphen, underscore, dot.
var remotePathRe = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)

// sshKeyRe allows safe filesystem path characters for the SSH key path.
// The key path is interpolated into the rsync "-e ssh -i %s ..." string, which
// rsync shell-interprets — so it must never contain shell metacharacters.
var sshKeyRe = regexp.MustCompile(`^[a-zA-Z0-9/_.~-]+$`)

// Deploy targets accepted by the CLI. Admin UI sends "production" instead of "prod".
var deployTargets = map[string]string{
	"local":      "local",
	"staging":    "staging",
	"prod":       "prod",
	"production": "prod",
}

// Deploy strategies.
var deployStrategies = map[string]bool{
	"rolling":    true,
	"blue-green": true,
	"canary":     true,
	"preview":    true,
}

var deployCmd = &cobra.Command{
	Use:   "deploy [target]",
	Short: "Deploy the stack to a target environment",
	Long: `Deploy the nSelf stack to local, staging, or production.

Executes: build (if needed) then start, with strategy-aware orchestration.

Targets:
  local       Equivalent to 'nself build && nself start' on this machine
  staging     Deploy to the staging environment (NSELF_DEPLOY_HOST_STAGING)
  production  Deploy to production (NSELF_DEPLOY_HOST_PROD, requires confirmation)

The target can be specified as a positional argument or via --env:
  nself deploy staging
  nself deploy --env staging

Examples:
  nself deploy local
  nself deploy --env staging --dry-run
  nself deploy staging --dry-run
  nself deploy production --strategy=blue-green
  nself deploy --env prod --force --follow
  nself deploy production --rolling --skip-health`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeploy,
}

var deployStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show deployment status for a target",
	RunE:  runDeployStatus,
}

var deployRollbackCmd = &cobra.Command{
	Use:   "rollback [target]",
	Short: "Roll back the last deployment for a target",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDeployRollback,
}

var deployLogsCmd = &cobra.Command{
	Use:   "logs [target]",
	Short: "Show deployment logs for a target",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDeployLogs,
}

var deployHealthCmd = &cobra.Command{
	Use:   "health [target]",
	Short: "Run health checks against a target deployment",
	Long: `Run health checks (nself doctor) against a target deployment.

With no [target] (or [target]=local), runs doctor checks against the local
docker daemon — unchanged from prior versions.

With [target]=staging|prod (or --server <name>), resolves that target's
primary server from .nself/control-plane.yaml (or NSELF_DEPLOY_HOST_<TARGET>)
and runs the check over SSH on the remote host itself, instead of silently
running local checks under a remote-sounding target name.

Note: remote checks require the target host's own nself CLI to support
'nself doctor' — an older remote CLI version returns a clear error naming
the likely version-drift cause rather than a raw SSH failure.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDeployHealth,
}

var deployCheckAccessCmd = &cobra.Command{
	Use:   "check-access",
	Short: "Verify access to configured deploy targets (deprecated: use 'nself env target probe')",
	Long: `Check SSH capability for all configured deploy environments.

Deprecated: this command performs a shallow env-var check for backward compatibility.
For full per-server SSH capability resolution use:

  nself env target probe [env]

The output format and exit code semantics are preserved for backward compatibility.`,
	RunE: runDeployCheckAccess,
}

// deployPromoteCmd promotes a canary deploy to 100% green traffic.
var deployPromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promote a canary deploy to 100% green traffic",
	Long: `Flip Nginx upstream weights from the current canary split to 100% green.
Use after manual inspection of a canary deploy.

Example:
  nself deploy --canary 10   # start canary at 10%
  nself deploy promote       # flip to 100% green after review`,
	RunE: runDeployPromote,
}

// deployEnvironmentsCmd lists environments and their servers with resolved
// capability.  This is the contract that the nSelf Admin companion consumes;
// absence of this endpoint caused Admin 500 errors (T04).
var deployEnvironmentsCmd = &cobra.Command{
	Use:   "environments",
	Short: "List configured deploy environments and server capabilities",
	Long: `List every environment defined in .nself/control-plane.yaml (or synthesized
from NSELF_DEPLOY_HOST_<TARGET> env vars) together with the resolved SSH
capability of each server.

Output is always JSON:

  {
    "environments": [
      {
        "name": "staging",
        "kind": "remote",
        "servers": [
          {
            "name": "staging-app",
            "role": "app",
            "capability": "manage",
            "reason": ""
          }
        ]
      }
    ]
  }

No SSH keys or credential material appear in the output.

Examples:
  nself deploy environments
  nself deploy environments | jq '.environments[].name'`,
	RunE: runDeployEnvironments,
}

func init() {
	f := deployCmd.Flags()
	f.String("strategy", "rolling", "Deploy strategy: rolling|blue-green|canary|preview")
	f.Bool("dry-run", false, "Preview the deploy without executing")
	f.Bool("force", false, "Skip confirmation prompts (prod requires --confirm or --force)")
	f.Bool("rolling", false, "Alias for --strategy=rolling")
	f.Bool("skip-health", false, "Skip post-deploy health checks")
	f.Bool("include-frontends", false, "Include frontend apps in the deploy")
	f.Bool("exclude-frontends", false, "Exclude frontend apps from the deploy")
	f.String("env", "", "Target environment: local|staging|prod (overrides positional arg; required env vars: NSELF_DEPLOY_HOST, NSELF_DEPLOY_USER, NSELF_DEPLOY_KEY_PATH)")
	f.Bool("follow", false, "Stream container logs after deploy until Ctrl-C (staging/prod only)")
	f.Bool("yes", false, "Skip prod confirmation prompt (alias for --force)")
	f.Bool("json", false, "Emit JSON output")

	// Blue/green canary flags (Y17 — blue_green_deploy feature flag).
	f.Int("canary", 0, "Start a canary deploy at N%% traffic to green (0 = full flip)")
	f.Bool("skip-canary", false, "Skip canary phase and flip directly to 100% green")
	f.Bool("force-migration", false, "Force deploy even with backward-incompatible migrations (disables canary)")

	deployStatusCmd.Flags().String("env", "", "Environment to check")
	deployStatusCmd.Flags().Bool("json", false, "Emit JSON output")
	deployStatusCmd.Flags().Bool("blue-green", false, "Show blue/green state alongside deployment status")

	// T05: --server selects a subset of servers for the pipeline.
	f.String("server", "", "Deploy to a specific server only (name from control-plane inventory)")

	deployStatusCmd.Flags().String("server", "", "Filter status output to a specific server")

	deployLogsCmd.Flags().String("server", "", "Stream logs from a specific server via SSH")

	deployHealthCmd.Flags().String("server", "", "Run health check on a specific server via SSH")
	deployHealthCmd.Flags().Bool("json", false, "Emit JSON output")

	deployCheckAccessCmd.Flags().Bool("json", false, "Emit JSON output (deprecated alias: use 'nself env target probe')")

	deployCmd.AddCommand(deployStatusCmd)
	deployCmd.AddCommand(deployRollbackCmd)
	deployCmd.AddCommand(deployLogsCmd)
	deployCmd.AddCommand(deployHealthCmd)
	deployCmd.AddCommand(deployCheckAccessCmd)
	deployCmd.AddCommand(deployPromoteCmd)
	deployCmd.AddCommand(deployEnvironmentsCmd)

	RootCmd.AddCommand(deployCmd)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// resolveTarget normalises "production" → "prod" and validates the value.
func resolveTarget(raw string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(raw))
	canonical, ok := deployTargets[t]
	if !ok {
		return "", fmt.Errorf("invalid target %q (allowed: local, staging, prod|production)", raw)
	}
	return canonical, nil
}

// projectRoot returns the nSelf project root, falling back to cwd if not found.
func projectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	root, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return cwd, nil
	}
	return root, nil
}

// runCLISelf invokes this same binary with args (used for build/start chaining).
// Falls back to exec.LookPath("nself") if os.Executable fails.
func runCLISelf(ctx context.Context, workdir string, args ...string) error {
	bin, err := os.Executable()
	if err != nil || bin == "" {
		bin, _ = exec.LookPath("nself")
	}
	if bin == "" {
		return fmt.Errorf("unable to locate nself binary")
	}
	c := exec.CommandContext(ctx, bin, args...)
	c.Dir = workdir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = os.Environ()
	return c.Run()
}

// ── T04: deploy environments ──────────────────────────────────────────────────

// envServerRow is the stable JSON schema for one server entry returned by
// "nself deploy environments". No secrets: SSHKeyRef value is never emitted.
type envServerRow struct {
	Name       string `json:"name"`
	Role       string `json:"role"`
	Capability string `json:"capability"`
	Reason     string `json:"reason,omitempty"`
}

// envEnvironmentRow is the stable JSON schema for one environment entry.
type envEnvironmentRow struct {
	Name    string         `json:"name"`
	Kind    string         `json:"kind"`
	Servers []envServerRow `json:"servers"`
}

// deployEnvironmentsOutput is the top-level JSON response consumed by Admin.
type deployEnvironmentsOutput struct {
	Environments []envEnvironmentRow `json:"environments"`
}

// runDeployEnvironments resolves the inventory and capability of every server,
// then emits the Admin-contract JSON.  No secret values are included.
func runDeployEnvironments(cmd *cobra.Command, _ []string) error {
	root, err := projectRoot()
	if err != nil {
		return err
	}

	inv, err := controlplane.Load(root)
	if err != nil {
		return fmt.Errorf("deploy environments: %w", err)
	}

	prober := controlplane.NewSSHProber(root, false)
	statuses := controlplane.Resolve(inv, prober)

	// Build a fast lookup: env/server → TargetStatus.
	type envServerKey struct{ env, server string }
	lookup := make(map[envServerKey]controlplane.TargetStatus, len(statuses))
	for _, ts := range statuses {
		lookup[envServerKey{ts.Env, ts.Server}] = ts
	}

	// Sort environments: local first, then alphabetically.
	envNames := make([]string, 0, len(inv.Environments))
	for name := range inv.Environments {
		envNames = append(envNames, name)
	}
	sort.Slice(envNames, func(i, j int) bool {
		if envNames[i] == "local" {
			return true
		}
		if envNames[j] == "local" {
			return false
		}
		return envNames[i] < envNames[j]
	})

	out := deployEnvironmentsOutput{
		Environments: make([]envEnvironmentRow, 0, len(envNames)),
	}
	for _, envName := range envNames {
		env := inv.Environments[envName]
		row := envEnvironmentRow{
			Name:    env.Name,
			Kind:    env.Kind,
			Servers: make([]envServerRow, 0, len(env.Servers)),
		}
		for _, srv := range env.Servers {
			ts := lookup[envServerKey{envName, srv.Name}]
			cap := string(ts.Capability)
			if cap == "" {
				cap = string(controlplane.CapHidden)
			}
			row.Servers = append(row.Servers, envServerRow{
				Name:       srv.Name,
				Role:       string(srv.Role),
				Capability: cap,
				Reason:     ts.Reason,
			})
		}
		out.Environments = append(out.Environments, row)
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("deploy environments: marshal: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// ── runDeploy ────────────────────────────────────────────────────────────────

// notYetImplementedStrategies lists strategies that fall back to rolling with
// an explicit warning. Tracked for v1.1.0.
// Note: blue-green and canary are now implemented via --canary N when the
// blue_green_deploy feature flag (Y17) is ON. The --strategy=blue-green/canary
// path still falls back to rolling for backwards compat with existing scripts.
var notYetImplementedStrategies = map[string]bool{
	"blue-green": true,
	"canary":     true,
	"preview":    true,
}

// deployServiceOrder defines the sequenced restart order for the rolling
// strategy. Services are restarted in dependency order so each layer is
// healthy before the next layer comes up.
var deployServiceOrder = []string{
	"postgres",
	"hasura",
	"auth",
	"storage",
	"plugins",
}

// runRollingRestart performs a per-service sequenced restart with health-
// gating between each service. It iterates over deployServiceOrder, calling
// "docker compose up -d <service>" per entry and waiting up to 60s for
// service_healthy before continuing. The deploy halts on first unhealthy
// service with a clear error and a pointer to nself logs.
func runRollingRestart(ctx context.Context, workdir string, jsonOut bool) ([]deployStep, error) {
	steps := []deployStep{}
	for _, svc := range deployServiceOrder {
		if !jsonOut {
			fmt.Printf("  [running] Restart %s (sequenced rolling)\n", svc)
		}
		// Restart the service.
		c := exec.CommandContext(ctx, "docker", "compose", "up", "-d", "--no-deps", svc)
		c.Dir = workdir
		c.Env = os.Environ()
		if out, err := c.CombinedOutput(); err != nil {
			steps = append(steps, deployStep{Name: fmt.Sprintf("Restart %s", svc), Status: "failed"})
			return steps, fmt.Errorf("rolling restart: service %s restart failed: %w\nOutput: %s\nRun 'nself logs %s' for details", svc, err, strings.TrimSpace(string(out)), svc)
		}

		// Health-gate: poll for service_healthy up to 60s.
		if !jsonOut {
			fmt.Printf("  [waiting] Waiting for service_healthy: %s (max 60s)\n", svc)
		}
		deadline := time.Now().Add(60 * time.Second)
		healthy := false
		for time.Now().Before(deadline) {
			out, err := exec.CommandContext(ctx, "docker", "compose", "ps", "--format", "{{.Name}}\t{{.Health}}", svc).Output()
			if err == nil {
				line := strings.TrimSpace(string(out))
				if strings.Contains(line, "healthy") && !strings.Contains(line, "unhealthy") {
					healthy = true
					break
				}
			}
			time.Sleep(2 * time.Second)
		}
		if !healthy {
			steps = append(steps, deployStep{Name: fmt.Sprintf("Restart %s", svc), Status: "unhealthy"})
			return steps, fmt.Errorf("rolling restart: service %s did not become healthy within 60s. Run 'nself logs %s' for details", svc, svc)
		}
		steps = append(steps, deployStep{Name: fmt.Sprintf("Restart %s", svc), Status: "done"})
		if !jsonOut {
			fmt.Printf("  [done] %s healthy\n", svc)
		}
	}
	return steps, nil
}

// runDeployHealthCheck calls nself doctor and gates the deploy result.
// Returns an error with the failed service name when any service is unhealthy.
func runDeployHealthCheck(ctx context.Context, workdir string, jsonOut bool) (deployStep, error) {
	if !jsonOut {
		fmt.Println("  [running] Health checks (calling nself health)")
	}
	bin, err := os.Executable()
	if err != nil || bin == "" {
		bin, _ = exec.LookPath("nself")
	}
	if bin == "" {
		return deployStep{Name: "Health checks", Status: "failed"}, fmt.Errorf("unable to locate nself binary for health check")
	}
	c := exec.CommandContext(ctx, bin, "doctor")
	c.Dir = workdir
	c.Env = os.Environ()
	out, err := c.CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		// Try to extract the failing service from doctor output.
		failedSvc := "unknown"
		for _, line := range strings.Split(output, "\n") {
			if strings.Contains(line, "unhealthy") || strings.Contains(line, "failed") {
				parts := strings.Fields(line)
				if len(parts) > 0 {
					failedSvc = parts[0]
					break
				}
			}
		}
		return deployStep{Name: "Health checks", Status: "failed"},
			fmt.Errorf("health check failed (service: %s). Run 'nself doctor --verbose' for details", failedSvc)
	}
	if !jsonOut {
		fmt.Println("  [done] Health checks passed")
	}
	return deployStep{Name: "Health checks", Status: "done"}, nil
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Resolve target: --env flag takes priority over the positional argument.
	envFlag, _ := cmd.Flags().GetString("env")
	var rawTarget string
	switch {
	case envFlag != "":
		rawTarget = envFlag
	case len(args) == 1:
		rawTarget = args[0]
	default:
		return fmt.Errorf("target environment required: pass it as an argument (nself deploy staging) or via --env (nself deploy --env staging)")
	}
	target, err := resolveTarget(rawTarget)
	if err != nil {
		return err
	}

	// Load the env file cascade for the resolved target into the current process
	// so that downstream helpers (build, health checks, SSH env) pick up the
	// correct environment variables. NSELF_DEPLOY_ENV is also set for subprocesses.
	// workdir may not be known yet; use a best-effort lookup here.
	if wd, wdErr := projectRoot(); wdErr == nil {
		loadDeployEnvCascade(wd, target)
	}

	strategy, _ := cmd.Flags().GetString("strategy")
	if rolling, _ := cmd.Flags().GetBool("rolling"); rolling {
		strategy = "rolling"
	}
	if !deployStrategies[strategy] {
		return fmt.Errorf("invalid strategy %q (allowed: rolling, blue-green, canary, preview)", strategy)
	}

	// Strategies other than rolling are not yet implemented. Fall back to
	// rolling with an explicit warning so users know the flag was accepted but
	// has no effect yet.
	if notYetImplementedStrategies[strategy] {
		if !func() bool { v, _ := cmd.Flags().GetBool("json"); return v }() {
			ui.Warn(fmt.Sprintf("Strategy %q is not yet implemented in v1.0.9. Tracked for v1.1.0; falling back to rolling. See .claude/docs/operations/deploy-strategies.md", strategy))
		}
		strategy = "rolling"
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		force = true
	}
	follow, _ := cmd.Flags().GetBool("follow")
	skipHealth, _ := cmd.Flags().GetBool("skip-health")
	jsonOut, _ := cmd.Flags().GetBool("json")
	includeFrontends, _ := cmd.Flags().GetBool("include-frontends")
	excludeFrontends, _ := cmd.Flags().GetBool("exclude-frontends")
	canaryPct, _ := cmd.Flags().GetInt("canary")
	skipCanary, _ := cmd.Flags().GetBool("skip-canary")
	forceMigration, _ := cmd.Flags().GetBool("force-migration")
	serverFilter, _ := cmd.Flags().GetString("server")

	workdir, err := projectRoot()
	if err != nil {
		return err
	}

	// Blue/green canary path (Y17 — blue_green_deploy feature flag).
	// When --canary N is passed and the flag is ON, route to the bluegreen package.
	// The feature flag check is intentionally lightweight: the env var
	// NSELF_FEATURE_BLUE_GREEN_DEPLOY=true mirrors what the feature-flags plugin
	// would return (nself flag list | grep blue_green_deploy). In production the
	// flag plugin is the authoritative source; the env var is the fallback for
	// environments without the flags plugin running.
	bgEnabled := os.Getenv("NSELF_FEATURE_BLUE_GREEN_DEPLOY") == "true"
	if (canaryPct > 0 || skipCanary) && bgEnabled {
		if !jsonOut {
			label := fmt.Sprintf("canary=%d%% skip-canary=%v force-migration=%v dry-run=%v", canaryPct, skipCanary, forceMigration, dryRun)
			ui.CommandHeader(fmt.Sprintf("nself deploy %s (blue/green)", target), label)
		}
		if target == "prod" && !dryRun && !force {
			return fmt.Errorf("production blue/green deploy requires --force. Re-run with --force once ready")
		}
		cfg := bluegreen.DeployConfig{
			ProjectRoot:    workdir,
			CanaryPercent:  canaryPct,
			SkipCanary:     skipCanary,
			ForceMigration: forceMigration,
			DryRun:         dryRun,
		}
		result := bluegreen.Deploy(cmd.Context(), cfg)
		if jsonOut {
			b, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(b))
		} else if result.RolledBack {
			ui.Error("Canary auto-rolled back: " + result.Error)
		} else if !result.Success {
			ui.Error("Blue/green deploy failed: " + result.Error)
		} else {
			ui.Success(fmt.Sprintf("Blue/green deploy complete in %s", result.Duration.Round(time.Millisecond)))
		}
		if !result.Success {
			return fmt.Errorf("%s", result.Error)
		}
		return nil
	}

	if !jsonOut {
		ui.CommandHeader(fmt.Sprintf("nself deploy %s", target), fmt.Sprintf("strategy=%s dry-run=%v include-frontends=%v exclude-frontends=%v", strategy, dryRun, includeFrontends, excludeFrontends))
	}

	// Production safety gate: require --force (or a "prod" confirm) when not in dry-run.
	if target == "prod" && !dryRun && !force {
		return fmt.Errorf("production deploy requires --force (or --dry-run). Re-run with --force once ready")
	}

	// T05: Control-plane pipeline path.
	// Active when .nself/control-plane.yaml exists OR when --server is specified.
	// When neither condition is true the legacy single-host path runs unchanged
	// (back-compat byte-identical guarantee per sprint spec).
	cpYamlPath := filepath.Join(workdir, ".nself", "control-plane.yaml")
	_, cpYamlErr := os.Stat(cpYamlPath)
	usePipeline := cpYamlErr == nil || serverFilter != ""

	if usePipeline {
		inv, loadErr := controlplane.Load(workdir)
		if loadErr != nil {
			return fmt.Errorf("deploy: load inventory: %w", loadErr)
		}

		// Apply server filter: remove all servers that do not match the requested name.
		if serverFilter != "" {
			inv = filterInventoryByServer(inv, serverFilter)
			if totalServers(inv) == 0 {
				return fmt.Errorf("deploy: --server %q not found in inventory", serverFilter)
			}
		}

		// --dry-run: print topology plan and exit without executing.
		if dryRun {
			prober := controlplane.NewSSHProber(workdir, false)
			statuses := controlplane.Resolve(inv, prober)
			if !jsonOut {
				fmt.Printf("  [dry-run] Topology plan for target %q:\n", target)
				for _, ts := range statuses {
					if ts.Capability == controlplane.CapHidden {
						continue
					}
					fmt.Printf("    %s/%s role=%-15s capability=%s", ts.Env, ts.Server, "", string(ts.Capability))
					if ts.Reason != "" {
						fmt.Printf(" reason=%q", ts.Reason)
					}
					fmt.Println()
				}
			} else {
				type dryRow struct {
					Env        string `json:"env"`
					Server     string `json:"server"`
					Capability string `json:"capability"`
					Reason     string `json:"reason,omitempty"`
				}
				var rows []dryRow
				for _, ts := range statuses {
					if ts.Capability == controlplane.CapHidden {
						continue
					}
					rows = append(rows, dryRow{Env: ts.Env, Server: ts.Server, Capability: string(ts.Capability), Reason: ts.Reason})
				}
				b, _ := json.MarshalIndent(map[string]interface{}{"dry_run": true, "topology": rows}, "", "  ")
				fmt.Println(string(b))
			}
			return nil
		}

		// Execute via topology-aware pipeline.
		prober := controlplane.NewSSHProber(workdir, false)
		composePath := filepath.Join(workdir, "docker-compose.yml")

		if !jsonOut {
			ui.CommandHeader(fmt.Sprintf("nself deploy %s (pipeline)", target), fmt.Sprintf("strategy=%s server=%s", strategy, serverFilter))
		}

		result, pipeErr := controlplane.Run(cmd.Context(), inv, prober, composePath)
		if pipeErr != nil {
			return fmt.Errorf("deploy pipeline: %w", pipeErr)
		}

		// Primary-skip gate: non-zero exit when primary was skipped.
		if result.PrimarySkipped {
			if !jsonOut {
				ui.Warn("Primary server was skipped (read-only capability) — deploy incomplete")
			} else {
				b, _ := json.MarshalIndent(result.Servers, "", "  ")
				fmt.Println(string(b))
			}
			return fmt.Errorf("deploy: primary server skipped (read-only capability); re-run once SSH access is restored")
		}

		if jsonOut {
			b, _ := json.MarshalIndent(result.Servers, "", "  ")
			fmt.Println(string(b))
		} else {
			for _, sr := range result.Servers {
				switch sr.Status {
				case "ok":
					ui.Success(fmt.Sprintf("  [ok] %s/%s", sr.Env, sr.Server))
				case "skipped":
					ui.Warn(fmt.Sprintf("  [skipped] %s/%s (read-only)", sr.Env, sr.Server))
				case "failed":
					ui.Error(fmt.Sprintf("  [failed] %s/%s: %v", sr.Env, sr.Server, sr.Err))
				}
			}
			ui.Success(fmt.Sprintf("Deploy %s (pipeline) complete", target))
		}
		return nil
	}

	steps := []deployStep{}
	start := time.Now()

	// Build
	if !dryRun {
		if !jsonOut {
			fmt.Println("  [running] Build images")
		}
		if err := runCLISelf(cmd.Context(), workdir, "build"); err != nil {
			return finalize(jsonOut, target, strategy, start, append(steps, deployStep{Name: "Build images", Status: "failed"}), err)
		}
	}
	steps = append(steps, deployStep{Name: "Build images", Status: stepStatus(dryRun, "done")})
	if !jsonOut && !dryRun {
		fmt.Println("  [done] Build images")
	}

	// Target-specific action
	switch target {
	case "local":
		if dryRun {
			if !jsonOut {
				fmt.Printf("  [dry-run] Would: docker compose up -d (rolling: %v, frontends: include=%v exclude=%v)\n", strategy, includeFrontends, excludeFrontends)
			}
			steps = append(steps, deployStep{Name: "Start local stack", Status: "pending"})
		} else {
			if !jsonOut {
				fmt.Println("  [running] Start local stack (rolling sequenced restart)")
			}
			restartSteps, restartErr := runRollingRestart(cmd.Context(), workdir, jsonOut)
			steps = append(steps, restartSteps...)
			if restartErr != nil {
				return finalize(jsonOut, target, strategy, start, steps, restartErr)
			}
		}

	case "staging", "prod":
		host := os.Getenv("NSELF_DEPLOY_HOST_" + strings.ToUpper(target))
		if host == "" {
			// Fall back to common env-var name patterns
			host = os.Getenv(strings.ToUpper(target) + "_DEPLOY_HOST")
		}

		if dryRun {
			if host != "" {
				if !jsonOut {
					fmt.Printf("  [dry-run] Would: ssh+rsync to %s then docker compose pull + rolling restart\n", host)
					fmt.Printf("  [dry-run] SSH key: %s\n", sshKeyPath())
					fmt.Printf("  [dry-run] Rolling restart order: %s\n", strings.Join(deployServiceOrder, " → "))
					fmt.Printf("  [dry-run] Frontends: include=%v exclude=%v\n", includeFrontends, excludeFrontends)
				}
				steps = append(steps, deployStep{Name: fmt.Sprintf("Push artefacts to %s", host), Status: "pending"})
				steps = append(steps, deployStep{Name: "Rolling restart (sequenced)", Status: "pending"})
			} else {
				if !jsonOut {
					fmt.Printf("  [dry-run] No NSELF_DEPLOY_HOST_%s set; would run locally\n", strings.ToUpper(target))
					fmt.Printf("  [dry-run] Set NSELF_DEPLOY_HOST_%s=user@host:/path to enable remote push\n", strings.ToUpper(target))
				}
				steps = append(steps, deployStep{Name: "Start stack (local host)", Status: "pending"})
			}
		} else if host != "" {
			// Remote push: rsync compose file + env + migrations, then pull images
			// and run the rolling restart on the remote host via ssh.
			if !jsonOut {
				fmt.Printf("  [running] Remote push to %s\n", host)
			}
			pushErr := remoteDeployPush(cmd.Context(), workdir, host, target, jsonOut)
			if pushErr != nil {
				steps = append(steps, deployStep{Name: fmt.Sprintf("Push artefacts to %s", host), Status: "failed"})
				return finalize(jsonOut, target, strategy, start, steps, pushErr)
			}
			steps = append(steps, deployStep{Name: fmt.Sprintf("Push artefacts to %s", host), Status: "done"})
		} else {
			// No host configured: run locally (matches v0.9.x behaviour when
			// deploy is triggered from a session on the target machine itself).
			if !jsonOut {
				fmt.Println("  [running] Start stack (rolling sequenced restart, local host)")
			}
			restartSteps, restartErr := runRollingRestart(cmd.Context(), workdir, jsonOut)
			steps = append(steps, restartSteps...)
			if restartErr != nil {
				return finalize(jsonOut, target, strategy, start, steps, restartErr)
			}
		}

		// Health gate (post-restart).
		if skipHealth {
			if !jsonOut {
				ui.Warn("Skipping health checks (--skip-health). Stack state unverified.")
			}
			steps = append(steps, deployStep{Name: "Health checks", Status: "skipped"})
		} else if !dryRun {
			healthStep, healthErr := runDeployHealthCheck(cmd.Context(), workdir, jsonOut)
			steps = append(steps, healthStep)
			if healthErr != nil {
				return finalize(jsonOut, target, strategy, start, steps, healthErr)
			}
		} else {
			steps = append(steps, deployStep{Name: "Health checks", Status: "pending"})
		}
	}

	// Auto-install daily disk-cleanup timer after successful staging/prod deploy (P98 T10.T07).
	// Non-fatal: a failure here warns but does not roll back the deploy.
	if !dryRun && (target == "staging" || target == "prod") {
		if timerErr := maintenance.InstallDailyTimer(); timerErr != nil {
			ui.Warn(fmt.Sprintf("daily maintenance timer install failed (non-fatal): %v", timerErr))
			ui.Warn("Run `nself maintenance schedule --daily` manually to enable disk-cleanup cron")
		}
	}

	if err := finalize(jsonOut, target, strategy, start, steps, nil); err != nil {
		return err
	}

	// --follow: stream container logs until Ctrl-C (staging/prod only).
	// Not supported for dry-run or local (local already has foreground start).
	if follow && !dryRun && (target == "staging" || target == "prod") {
		if !jsonOut {
			ui.Info("Following container logs (Ctrl-C to stop)...")
		}
		host := os.Getenv("NSELF_DEPLOY_HOST_" + strings.ToUpper(target))
		if host == "" {
			host = os.Getenv(strings.ToUpper(target) + "_DEPLOY_HOST")
		}
		if host != "" {
			// Remote follow: tail logs on the remote host via SSH.
			colonIdx := strings.LastIndex(host, ":")
			sshTarget := host
			remotePath := ""
			if colonIdx >= 0 {
				sshTarget = host[:colonIdx]
				remotePath = host[colonIdx+1:]
			}
			sshKey := sshKeyPath()
			logsCmd := "docker compose logs -f --tail=50"
			if remotePath != "" {
				logsCmd = fmt.Sprintf("cd %s && docker compose logs -f --tail=50", remotePath)
			}
			sc := exec.CommandContext(cmd.Context(), "ssh",
				"-i", sshKey,
				"-o", "StrictHostKeyChecking=accept-new",
				"-o", "ForwardAgent=no",
				sshTarget, logsCmd)
			sc.Stdout = os.Stdout
			sc.Stderr = os.Stderr
			_ = sc.Run() // Ctrl-C exits; non-zero exit is not an error from user perspective.
		} else {
			// Local host: stream logs directly.
			lc := exec.CommandContext(cmd.Context(), "docker", "compose", "logs", "-f", "--tail=50")
			lc.Dir = workdir
			lc.Stdout = os.Stdout
			lc.Stderr = os.Stderr
			_ = lc.Run()
		}
	}

	return nil
}

// filterInventoryByServer returns a shallow copy of inv retaining only the
// server whose Name matches serverName across all environments.
func filterInventoryByServer(inv *controlplane.Inventory, serverName string) *controlplane.Inventory {
	filtered := &controlplane.Inventory{
		SchemaVersion: inv.SchemaVersion,
		Project:       inv.Project,
		Environments:  make(map[string]controlplane.Environment, len(inv.Environments)),
	}
	for envName, env := range inv.Environments {
		var kept []controlplane.Server
		for _, srv := range env.Servers {
			if srv.Name == serverName {
				kept = append(kept, srv)
			}
		}
		if len(kept) > 0 {
			filtered.Environments[envName] = controlplane.Environment{
				Name:    env.Name,
				Kind:    env.Kind,
				Servers: kept,
			}
		}
	}
	return filtered
}

// totalServers returns the total number of servers across all environments.
func totalServers(inv *controlplane.Inventory) int {
	n := 0
	for _, env := range inv.Environments {
		n += len(env.Servers)
	}
	return n
}

// loadDeployEnvCascade loads the env file cascade for the given deploy target
// into the current process environment. Later files override earlier ones.
//
// Cascade per target:
//
//	local      → .env.dev + .env.local
//	staging    → .env.dev + .env.staging + .env.secrets
//	prod       → .env.dev + .env.prod   + .env.secrets
//
// Missing files are silently skipped. The NSELF_DEPLOY_ENV env var is set
// to the canonical target name so downstream helpers can introspect it.
//
// Gap #13 fix: this also sets ENV to the same canonical target name. ENV is
// the variable config.Load (internal/config/loader.go) actually keys its own
// file cascade on — before this fix, ENV was never set here, so the
// subsequent 'nself build' subprocess (spawned by runDeploy via runCLISelf)
// resolved config.Load's cascade against the default "dev" tier regardless
// of which target this process just loaded, silently baking dev-tier values
// (wrong POSTGRES_DB, wrong ports, etc.) into docker-compose.yml even for a
// staging/prod deploy.
func loadDeployEnvCascade(workdir, target string) {
	files := deployEnvCascadeFiles(workdir, target)
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			// Overload merges into os.Environ — missing files already skipped above.
			_ = godotenv.Overload(f)
		}
	}
	// Expose the resolved target so subprocesses and plugins can read it.
	_ = os.Setenv("NSELF_DEPLOY_ENV", target)
	// Gap #13: make config.Load's own cascade selection agree with the
	// cascade we just loaded into this process's environment.
	_ = os.Setenv("ENV", target)
}

// deployEnvCascadeFiles returns the ordered list of .env files that make up
// target's cascade, matching config.Load's own cascade order (internal/config/loader.go)
// so the set of files loaded here and the set config.Load merges in the
// 'nself build' subprocess are identical.
func deployEnvCascadeFiles(workdir, target string) []string {
	switch target {
	case "local":
		return []string{
			filepath.Join(workdir, ".env.dev"),
			filepath.Join(workdir, ".env.local"),
		}
	case "staging":
		return []string{
			filepath.Join(workdir, ".env.dev"),
			filepath.Join(workdir, ".env.staging"),
			filepath.Join(workdir, ".env.secrets"),
		}
	default: // "prod"
		return []string{
			filepath.Join(workdir, ".env.dev"),
			filepath.Join(workdir, ".env.prod"),
			filepath.Join(workdir, ".env.secrets"),
		}
	}
}

// sshKeyPath returns the SSH key path from NSELF_DEPLOY_SSH_KEY env or the
// default ~/.ssh/id_ed25519.
func sshKeyPath() string {
	if k := os.Getenv("NSELF_DEPLOY_SSH_KEY"); k != "" {
		return k
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "id_ed25519")
}

// remoteDeployPush rsyncs the compose file and env to the remote host, then
// pulls new images and runs a rolling restart via SSH.
// host format: "user@host:/remote/path"
func remoteDeployPush(ctx context.Context, workdir, host, target string, jsonOut bool) error {
	sshKey := sshKeyPath()

	// sshKey is interpolated into the rsync "-e" command, which rsync passes
	// through a shell. Reject any key path containing shell metacharacters to
	// prevent command injection via NSELF_DEPLOY_SSH_KEY.
	if sshKey != "" && !sshKeyRe.MatchString(sshKey) {
		return fmt.Errorf("NSELF_DEPLOY_SSH_KEY contains unsafe characters (got %q): only [a-zA-Z0-9/_.~-] allowed", sshKey)
	}

	// Split user@host:/path into ssh-target and remote-path.
	colonIdx := strings.LastIndex(host, ":")
	if colonIdx < 0 {
		return fmt.Errorf("NSELF_DEPLOY_HOST_%s format must be user@host:/remote/path (got %q)", strings.ToUpper(target), host)
	}
	sshTarget := host[:colonIdx]
	remotePath := host[colonIdx+1:]
	if remotePath == "" {
		return fmt.Errorf("NSELF_DEPLOY_HOST_%s remote path is empty (got %q)", strings.ToUpper(target), host)
	}
	if !remotePathRe.MatchString(remotePath) {
		return fmt.Errorf("NSELF_DEPLOY_HOST_%s remote path contains unsafe characters (got %q): only [a-zA-Z0-9/_.-] allowed", strings.ToUpper(target), remotePath)
	}

	// Gap #13 fix: the file that used to be rsynced here (.env.<target> alone,
	// e.g. .env.staging) is only ONE layer of the cascade that config.Load
	// actually merged to produce the docker-compose.yml being pushed
	// alongside it (CLI-R18 canonical order: .env -> .env.<target> ->
	// .env.secrets -> .env.local). Pushing just one layer left the remote's
	// env file mismatched with values baked into the compose file (wrong
	// POSTGRES_DB, wrong ports, etc.) whenever an earlier/later layer set them.
	//
	// Write a merged snapshot containing every value config.Load resolved
	// for this deploy, and push that as .env.<target> instead of the raw
	// single-layer file. This keeps the remote filename convention the
	// on-box CLI already expects, while guaranteeing its contents match
	// docker-compose.yml byte-for-byte in provenance.
	resolvedEnvPath, cleanupResolvedEnv, err := writeResolvedDeployEnv(workdir, target)
	if err != nil {
		return fmt.Errorf("preparing resolved .env for %s: %w", target, err)
	}
	defer cleanupResolvedEnv()

	// rsync compose + env files to the remote.
	// Agent forwarding is disabled via ForwardAgent=no in the -e ssh command —
	// it is an ssh option and must never appear in rsync argv (breaks rsync 3.x).
	rsyncArgs := []string{
		"-az",
		"-e", fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=accept-new -o ForwardAgent=no", sshKey),
		"docker-compose.yml",
		resolvedEnvPath,
		fmt.Sprintf("%s:%s/", sshTarget, remotePath),
	}
	if !jsonOut {
		fmt.Printf("  [running] rsync compose + resolved env to %s:%s\n", sshTarget, remotePath)
	}
	rc := exec.CommandContext(ctx, "rsync", rsyncArgs...)
	rc.Dir = workdir
	rc.Env = os.Environ()
	if out, err := rc.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync to %s failed: %w\n%s", sshTarget, err, strings.TrimSpace(string(out)))
	}
	// Rename the pushed snapshot to the expected .env.<target> name on the
	// remote (rsync above pushes it under its resolvedEnvPath basename).
	renameCmd := fmt.Sprintf("cd %s && mv %s .env.%s", remotePath, filepath.Base(resolvedEnvPath), target)
	rn := exec.CommandContext(ctx, "ssh",
		"-i", sshKey,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ForwardAgent=no",
		sshTarget, renameCmd)
	rn.Env = os.Environ()
	if out, err := rn.CombinedOutput(); err != nil {
		return fmt.Errorf("finalizing resolved .env on %s failed: %w\n%s", sshTarget, err, strings.TrimSpace(string(out)))
	}

	// Pull new images on the remote.
	sshPull := fmt.Sprintf("cd %s && docker compose pull", remotePath)
	if !jsonOut {
		fmt.Printf("  [running] docker compose pull on %s\n", sshTarget)
	}
	pc := exec.CommandContext(ctx, "ssh",
		"-i", sshKey,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ForwardAgent=no",
		sshTarget, sshPull)
	pc.Env = os.Environ()
	if out, err := pc.CombinedOutput(); err != nil {
		return fmt.Errorf("remote pull on %s failed: %w\n%s", sshTarget, err, strings.TrimSpace(string(out)))
	}

	// Discover which services exist in the pushed compose so the rolling order
	// only restarts real services (e.g. generated composes name object storage
	// "minio", not "storage" — restarting a nonexistent service aborts deploys).
	lsCmd := fmt.Sprintf("cd %s && docker compose config --services", remotePath)
	lc := exec.CommandContext(ctx, "ssh",
		"-i", sshKey,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ForwardAgent=no",
		sshTarget, lsCmd)
	lc.Env = os.Environ()
	remoteServices := map[string]bool{}
	if out, err := lc.CombinedOutput(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if s := strings.TrimSpace(line); s != "" {
				remoteServices[s] = true
			}
		}
	}

	// Rolling restart on the remote: sequence the services via SSH.
	for _, svc := range deployServiceOrder {
		if len(remoteServices) > 0 && !remoteServices[svc] {
			if !jsonOut {
				fmt.Printf("  [skip] %s not in remote compose — skipping\n", svc)
			}
			continue
		}
		restartCmd := fmt.Sprintf("cd %s && docker compose up -d --no-deps %s", remotePath, svc)
		if !jsonOut {
			fmt.Printf("  [running] Rolling restart: %s on %s\n", svc, sshTarget)
		}
		sc := exec.CommandContext(ctx, "ssh",
			"-i", sshKey,
			"-o", "StrictHostKeyChecking=accept-new",
			"-o", "ForwardAgent=no",
			sshTarget, restartCmd)
		sc.Env = os.Environ()
		if out, err := sc.CombinedOutput(); err != nil {
			return fmt.Errorf("remote rolling restart of %s failed: %w\n%s\nRun 'nself logs %s' on the remote host for details", svc, err, strings.TrimSpace(string(out)), svc)
		}
	}
	return nil
}

// ── subcommands ──────────────────────────────────────────────────────────────

func runDeployStatus(cmd *cobra.Command, args []string) error {
	env, _ := cmd.Flags().GetString("env")
	jsonOut, _ := cmd.Flags().GetBool("json")
	serverFilter, _ := cmd.Flags().GetString("server")

	// Determine base state from docker ps (local stack indicator).
	state := "unknown"
	if env == "" {
		state = "no-target"
	} else if _, err := resolveTarget(env); err != nil {
		return err
	}
	if _, err := exec.LookPath("docker"); err == nil {
		out, derr := exec.CommandContext(cmd.Context(), "docker", "ps", "--format", "{{.Names}}").Output()
		if derr == nil && strings.Contains(string(out), "postgres") {
			state = "running"
		} else if state != "no-target" {
			state = "not-running"
		}
	}

	// T06: If control-plane inventory exists, enrich with per-server capability.
	root, rootErr := projectRoot()
	if rootErr == nil {
		cpYaml := filepath.Join(root, ".nself", "control-plane.yaml")
		if _, err := os.Stat(cpYaml); err == nil {
			inv, loadErr := controlplane.Load(root)
			if loadErr == nil {
				prober := controlplane.NewSSHProber(root, false)
				statuses := controlplane.Resolve(inv, prober)

				type serverStatus struct {
					Env        string `json:"env"`
					Server     string `json:"server"`
					Role       string `json:"role"`
					Capability string `json:"capability"`
					Reason     string `json:"reason,omitempty"`
					LatencyMS  int    `json:"latency_ms,omitempty"`
				}
				var rows []serverStatus
				for _, ts := range statuses {
					if ts.Capability == controlplane.CapHidden {
						continue
					}
					if serverFilter != "" && ts.Server != serverFilter {
						continue
					}
					// Determine role from inventory.
					role := ""
					if envDef, ok := inv.Environments[ts.Env]; ok {
						for _, srv := range envDef.Servers {
							if srv.Name == ts.Server {
								role = string(srv.Role)
								break
							}
						}
					}
					rows = append(rows, serverStatus{
						Env:        ts.Env,
						Server:     ts.Server,
						Role:       role,
						Capability: string(ts.Capability),
						Reason:     ts.Reason,
						LatencyMS:  ts.LatencyMS,
					})
				}

				if jsonOut {
					b, _ := json.MarshalIndent(map[string]interface{}{
						"target":  env,
						"state":   state,
						"servers": rows,
					}, "", "  ")
					fmt.Println(string(b))
					return nil
				}
				fmt.Printf("target=%s state=%s\n", env, state)
				for _, row := range rows {
					fmt.Printf("  server=%s/%s role=%-15s capability=%s", row.Env, row.Server, row.Role, row.Capability)
					if row.LatencyMS > 0 {
						fmt.Printf(" latency=%dms", row.LatencyMS)
					}
					if row.Reason != "" {
						fmt.Printf(" reason=%q", row.Reason)
					}
					fmt.Println()
				}
				return nil
			}
		}
	}

	// Fallback: legacy single-host output.
	status := map[string]string{
		"target": env,
		"state":  state,
	}
	if jsonOut {
		b, _ := json.MarshalIndent(status, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("target=%s state=%s\n", status["target"], status["state"])
	return nil
}

func runDeployRollback(cmd *cobra.Command, args []string) error {
	target := "local"
	if len(args) == 1 {
		t, err := resolveTarget(args[0])
		if err != nil {
			return err
		}
		target = t
	}

	workdir, err := projectRoot()
	if err != nil {
		return err
	}

	// Blue/green rollback path (Y17).
	bgEnabled := os.Getenv("NSELF_FEATURE_BLUE_GREEN_DEPLOY") == "true"
	if bgEnabled {
		ui.Info(fmt.Sprintf("Rolling back blue/green deploy for target: %s", target))
		cfg := bluegreen.DeployConfig{ProjectRoot: workdir}
		result := bluegreen.Rollback(cmd.Context(), cfg)
		if !result.Success {
			return fmt.Errorf("blue/green rollback failed: %s", result.Error)
		}
		ui.Success(fmt.Sprintf("Blue/green rollback complete in %s — all traffic restored to blue", result.Duration.Round(time.Millisecond)))
		return nil
	}

	// PREVIEW: rollback is a PREVIEW feature. It restores the last promote snapshot
	// created by nself promote. DNS failback and full cluster-level rollback are not
	// yet automated — see cross-cutting.md §3 Lifecycle for the roadmap.
	ui.Warn("nself deploy rollback is a PREVIEW feature. It restores the last pre-promote backup snapshot. No DNS changes are made automatically.")
	ui.Info(fmt.Sprintf("Rolling back last deployment for target: %s", target))

	// DEP-04: wire to last promote tag written by nself promote.
	// promote.Rollback with an empty tag reads the latest promote record from
	// <projectDir>/.nself/promotions/ and restores the backup snapshot created
	// before that promotion was applied. This ensures rollback always targets
	// the last honest production-change surface (nself promote), not an
	// arbitrary git tag or manual state.
	if err := promote.Rollback(cmd.Context(), workdir, ""); err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	ui.Success(fmt.Sprintf("Rollback for %s completed — prior promote state restored", target))
	return nil
}

// runDeployPromote flips Nginx to 100% green after a manual canary review.
func runDeployPromote(cmd *cobra.Command, args []string) error {
	workdir, err := projectRoot()
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")

	ui.Info("Promoting canary to 100% green traffic...")
	cfg := bluegreen.DeployConfig{
		ProjectRoot: workdir,
		DryRun:      dryRun,
	}
	result := bluegreen.Promote(cmd.Context(), cfg)
	if !result.Success {
		return fmt.Errorf("promote failed: %s", result.Error)
	}
	ui.Success(fmt.Sprintf("Promoted to 100%% green in %s", result.Duration.Round(time.Millisecond)))
	return nil
}

func runDeployLogs(cmd *cobra.Command, args []string) error {
	workdir, err := projectRoot()
	if err != nil {
		return err
	}

	serverFilter, _ := cmd.Flags().GetString("server")
	if serverFilter != "" {
		// T06: Stream logs from a specific remote server via SSH.
		inv, loadErr := controlplane.Load(workdir)
		if loadErr != nil {
			return fmt.Errorf("deploy logs: load inventory: %w", loadErr)
		}
		for _, env := range inv.Environments {
			for _, srv := range env.Servers {
				if srv.Name != serverFilter {
					continue
				}
				if srv.Host == "" {
					// Local server: fall through to docker compose logs below.
					break
				}
				keyPath := os.Getenv(srv.SSHKeyRef)
				remotePath := srv.RemotePath
				if remotePath == "" {
					remotePath = "/opt/nself"
				}
				logsCmd := fmt.Sprintf("cd %s && docker compose logs --tail=200 -f", remotePath)
				sshTarget := srv.Host
				sc := exec.CommandContext(cmd.Context(), "ssh",
					"-i", keyPath,
					"-o", "BatchMode=yes",
					"-o", "ForwardAgent=no",
					"-o", "StrictHostKeyChecking=accept-new",
					sshTarget, logsCmd)
				sc.Stdout = os.Stdout
				sc.Stderr = os.Stderr
				return sc.Run()
			}
		}
		return fmt.Errorf("deploy logs: --server %q not found in inventory", serverFilter)
	}

	// Default: local docker compose logs.
	c := exec.CommandContext(cmd.Context(), "docker", "compose", "logs", "--tail=200")
	c.Dir = workdir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func runDeployHealth(cmd *cobra.Command, args []string) error {
	workdir, err := projectRoot()
	if err != nil {
		return err
	}

	serverFilter, _ := cmd.Flags().GetString("server")
	jsonOut, _ := cmd.Flags().GetBool("json")

	if serverFilter != "" {
		return runDeployHealthOnServer(cmd, workdir, serverFilter, jsonOut)
	}

	// Gap #12: "nself deploy health <target>" (positional arg, no --server)
	// previously ignored the target entirely and always ran local doctor —
	// silently misleading for staging/prod. Resolve the target the same way
	// runDeploy does and probe its primary server remotely when the target
	// names a configured remote environment. "local" (or no target) keeps
	// the original local-doctor behavior unchanged (back-compat).
	if len(args) == 1 {
		target, resolveErr := resolveTarget(args[0])
		if resolveErr != nil {
			return resolveErr
		}
		if target != "local" {
			inv, loadErr := controlplane.Load(workdir)
			if loadErr != nil {
				return fmt.Errorf("deploy health: load inventory: %w", loadErr)
			}
			if env, ok := inv.Environments[target]; ok {
				if srv, found := findDBTargetServer(env, ""); found && srv.Host != "" {
					return runDeployHealthOnServer(cmd, workdir, srv.Name, jsonOut)
				}
			}
			// No remote host configured for this target (e.g. only
			// NSELF_DEPLOY_HOST_<TARGET> style legacy env vars, no inventory
			// entry) — tell the user explicitly rather than silently running
			// local checks against the wrong environment's name.
			if host, ok := ResolveLegacyDeployHost(target); ok {
				return runDeployHealthOverSSH(cmd, host, jsonOut)
			}
			return fmt.Errorf("deploy health: no server configured for target %q (set NSELF_DEPLOY_HOST_%s or add it to .nself/control-plane.yaml)", target, strings.ToUpper(target))
		}
	}

	return runCLISelf(cmd.Context(), workdir, "doctor")
}

func runDeployCheckAccess(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

	// T07: Re-point to controlplane probe path when inventory is present.
	root, rootErr := projectRoot()
	if rootErr == nil {
		cpYaml := filepath.Join(root, ".nself", "control-plane.yaml")
		if _, err := os.Stat(cpYaml); err == nil {
			inv, loadErr := controlplane.Load(root)
			if loadErr == nil {
				prober := controlplane.NewSSHProber(root, false)
				statuses := controlplane.Resolve(inv, prober)

				if !jsonOut {
					ui.Warn("'nself deploy check-access' is deprecated. Use 'nself env target probe' for full per-server capability details.")
					fmt.Println()
				}

				type accessRow struct {
					Env        string `json:"env"`
					Server     string `json:"server"`
					Capability string `json:"capability"`
					Reason     string `json:"reason,omitempty"`
				}
				var rows []accessRow
				allOK := true
				for _, ts := range statuses {
					if ts.Capability == controlplane.CapHidden {
						continue
					}
					rows = append(rows, accessRow{
						Env:        ts.Env,
						Server:     ts.Server,
						Capability: string(ts.Capability),
						Reason:     ts.Reason,
					})
					if ts.Capability != controlplane.CapManage {
						allOK = false
					}
				}

				if jsonOut {
					b, _ := json.MarshalIndent(map[string]interface{}{"servers": rows, "all_ok": allOK}, "", "  ")
					fmt.Println(string(b))
					return nil
				}

				for _, row := range rows {
					if row.Capability == string(controlplane.CapManage) {
						ui.Success(fmt.Sprintf("%s/%s: %s", row.Env, row.Server, row.Capability))
					} else {
						ui.Warn(fmt.Sprintf("%s/%s: %s — %s", row.Env, row.Server, row.Capability, row.Reason))
					}
				}
				if allOK {
					ui.Success("All deploy targets reachable")
				}
				return nil
			}
		}
	}

	// Legacy fallback: shallow env-var check (back-compat for no-yaml installs).
	ok := true
	for _, name := range []string{"NSELF_DEPLOY_HOST_STAGING", "NSELF_DEPLOY_HOST_PROD"} {
		v := os.Getenv(name)
		if v == "" {
			ui.Warn(fmt.Sprintf("%s is not set (deploy to this target will run locally)", name))
			ok = false
			continue
		}
		ui.Success(fmt.Sprintf("%s = %s", name, v))
	}
	if !ok {
		return nil
	}
	ui.Success("All deploy targets reachable")
	return nil
}

// ── output helpers ───────────────────────────────────────────────────────────

type deployStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type deployResult struct {
	Target   string       `json:"target"`
	Strategy string       `json:"strategy"`
	Steps    []deployStep `json:"steps"`
	Duration int64        `json:"durationMs"`
	Success  bool         `json:"success"`
	Error    string       `json:"error,omitempty"`
}

func stepStatus(dryRun bool, ok string) string {
	if dryRun {
		return "pending"
	}
	return ok
}

func finalize(jsonOut bool, target, strategy string, start time.Time, steps []deployStep, err error) error {
	duration := time.Since(start).Milliseconds()
	success := err == nil
	if jsonOut {
		res := deployResult{
			Target:   target,
			Strategy: strategy,
			Steps:    steps,
			Duration: duration,
			Success:  success,
		}
		if err != nil {
			res.Error = err.Error()
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
		return err
	}
	if success {
		ui.Success(fmt.Sprintf("Deploy %s (%s) finished in %dms", target, strategy, duration))
	} else {
		ui.Error(fmt.Sprintf("Deploy %s (%s) failed after %dms: %v", target, strategy, duration, err))
	}
	return err
}
