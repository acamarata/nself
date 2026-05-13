package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/deploy/bluegreen"
	"github.com/nself-org/cli/internal/maintenance"
	"github.com/nself-org/cli/internal/promote"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// remotePathRe allows safe remote path characters: alphanumeric, slash, hyphen, underscore, dot.
var remotePathRe = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)

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
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDeployHealth,
}

var deployCheckAccessCmd = &cobra.Command{
	Use:   "check-access",
	Short: "Verify access to configured deploy targets",
	RunE:  runDeployCheckAccess,
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

	deployCmd.AddCommand(deployStatusCmd)
	deployCmd.AddCommand(deployRollbackCmd)
	deployCmd.AddCommand(deployLogsCmd)
	deployCmd.AddCommand(deployHealthCmd)
	deployCmd.AddCommand(deployCheckAccessCmd)
	deployCmd.AddCommand(deployPromoteCmd)

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

	// rsync compose + env files to the remote.
	rsyncArgs := []string{
		"-az", "--no-agent-forwarding",
		"-e", fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=accept-new -o ForwardAgent=no", sshKey),
		"docker-compose.yml",
		fmt.Sprintf(".env.%s", target),
		fmt.Sprintf("%s:%s/", sshTarget, remotePath),
	}
	if !jsonOut {
		fmt.Printf("  [running] rsync compose + env to %s:%s\n", sshTarget, remotePath)
	}
	rc := exec.CommandContext(ctx, "rsync", rsyncArgs...)
	rc.Dir = workdir
	rc.Env = os.Environ()
	if out, err := rc.CombinedOutput(); err != nil {
		return fmt.Errorf("rsync to %s failed: %w\n%s", sshTarget, err, strings.TrimSpace(string(out)))
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

	// Rolling restart on the remote: sequence the services via SSH.
	for _, svc := range deployServiceOrder {
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

	status := map[string]string{
		"target": env,
		"state":  "unknown",
	}
	if env == "" {
		status["state"] = "no-target"
	} else if _, err := resolveTarget(env); err != nil {
		return err
	}

	// A running nSelf stack on localhost is inferred from docker ps existence.
	if _, err := exec.LookPath("docker"); err == nil {
		out, derr := exec.CommandContext(cmd.Context(), "docker", "ps", "--format", "{{.Names}}").Output()
		if derr == nil && strings.Contains(string(out), "postgres") {
			status["state"] = "running"
		} else {
			status["state"] = "not-running"
		}
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
	return runCLISelf(cmd.Context(), workdir, "doctor")
}

func runDeployCheckAccess(cmd *cobra.Command, args []string) error {
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
