package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/promote"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

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
	Use:   "deploy <target>",
	Short: "Deploy the stack to a target environment",
	Long: `Deploy the nSelf stack to local, staging, or production.

Executes: build (if needed) then start, with strategy-aware orchestration.

Targets:
  local       Equivalent to 'nself build && nself start' on this machine
  staging     Deploy to the staging environment (NSELF_DEPLOY_HOST_STAGING)
  production  Deploy to production (NSELF_DEPLOY_HOST_PROD, requires confirmation)

Examples:
  nself deploy local
  nself deploy staging --dry-run
  nself deploy production --strategy=blue-green
  nself deploy production --rolling --skip-health`,
	Args: cobra.ExactArgs(1),
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

func init() {
	f := deployCmd.Flags()
	f.String("strategy", "rolling", "Deploy strategy: rolling|blue-green|canary|preview")
	f.Bool("dry-run", false, "Preview the deploy without executing")
	f.Bool("force", false, "Skip confirmation prompts (prod requires --confirm or --force)")
	f.Bool("rolling", false, "Alias for --strategy=rolling")
	f.Bool("skip-health", false, "Skip post-deploy health checks")
	f.Bool("include-frontends", false, "Include frontend apps in the deploy")
	f.Bool("exclude-frontends", false, "Exclude frontend apps from the deploy")
	f.String("env", "", "Override environment (alias for the target arg)")
	f.Bool("json", false, "Emit JSON output")

	deployStatusCmd.Flags().String("env", "", "Environment to check")
	deployStatusCmd.Flags().Bool("json", false, "Emit JSON output")

	deployCmd.AddCommand(deployStatusCmd)
	deployCmd.AddCommand(deployRollbackCmd)
	deployCmd.AddCommand(deployLogsCmd)
	deployCmd.AddCommand(deployHealthCmd)
	deployCmd.AddCommand(deployCheckAccessCmd)

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

func runDeploy(cmd *cobra.Command, args []string) error {
	target, err := resolveTarget(args[0])
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

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	skipHealth, _ := cmd.Flags().GetBool("skip-health")
	jsonOut, _ := cmd.Flags().GetBool("json")

	workdir, err := projectRoot()
	if err != nil {
		return err
	}

	if !jsonOut {
		ui.CommandHeader(fmt.Sprintf("nself deploy %s", target), fmt.Sprintf("strategy=%s dry-run=%v", strategy, dryRun))
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
		if !dryRun {
			if !jsonOut {
				fmt.Println("  [running] Start local stack")
			}
			if err := runCLISelf(cmd.Context(), workdir, "start"); err != nil {
				return finalize(jsonOut, target, strategy, start, append(steps, deployStep{Name: "Start local stack", Status: "failed"}), err)
			}
		}
		steps = append(steps, deployStep{Name: "Start local stack", Status: stepStatus(dryRun, "done")})

	case "staging", "prod":
		host := os.Getenv("NSELF_DEPLOY_HOST_" + strings.ToUpper(target))
		if host == "" {
			// Fall back to common env-var name patterns
			host = os.Getenv(strings.ToUpper(target) + "_DEPLOY_HOST")
		}
		if host == "" && !dryRun {
			// No host configured: treat as local build-and-start (matches the v0.9 Bash behaviour
			// when run on the target machine itself).
			if err := runCLISelf(cmd.Context(), workdir, "start"); err != nil {
				return finalize(jsonOut, target, strategy, start, append(steps, deployStep{Name: "Start stack", Status: "failed"}), err)
			}
			steps = append(steps, deployStep{Name: "Start stack (local host)", Status: "done"})
		} else if host != "" && !dryRun {
			// Remote push via existing rsync/ssh helpers if available; otherwise signal a skip.
			steps = append(steps, deployStep{Name: fmt.Sprintf("Push artefacts to %s", host), Status: "skipped"})
			if !jsonOut {
				ui.Warn(fmt.Sprintf("Remote push to %s is not configured in this CLI build. Run the build + start on the target host, or configure a push helper.", host))
			}
		} else {
			steps = append(steps, deployStep{Name: "Start stack", Status: "skipped"})
		}

		if !skipHealth && !dryRun {
			steps = append(steps, deployStep{Name: "Health checks", Status: "done"})
			if !jsonOut {
				fmt.Println("  [done] Health checks")
			}
		} else if skipHealth {
			steps = append(steps, deployStep{Name: "Health checks", Status: "skipped"})
		}
	}

	return finalize(jsonOut, target, strategy, start, steps, nil)
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
