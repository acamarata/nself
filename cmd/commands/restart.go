package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/health"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart [SERVICES...]",
	Short: "Smart restart with config change detection",
	Long: `Restart the nSelf stack with intelligent change detection.

Smart mode (default) compares .env modification time against container start
times and only rebuilds when configuration has changed. Use --all for a full
stop + start cycle.

Examples:
  nself restart              # Smart restart (rebuild only if .env changed)
  nself restart postgres     # Restart specific service
  nself restart --all        # Full stop + start cycle
  nself restart --no-build   # Skip image rebuild`,
	RunE: runRestart,
}

func init() {
	restartCmd.Flags().BoolP("smart", "s", true, "Detect changes, only restart affected (default)")
	restartCmd.Flags().BoolP("all", "a", false, "Full restart: stop all + start all")
	restartCmd.Flags().BoolP("verbose", "v", false, "Detailed output")
	restartCmd.Flags().Bool("skip-build", false, "Skip image rebuild")
	restartCmd.Flags().Bool("no-build", false, "Alias for --skip-build")

	RootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	ui.CommandHeader("nself restart", "Smart restart with change detection")

	smart, _ := cmd.Flags().GetBool("smart")
	all, _ := cmd.Flags().GetBool("all")
	verbose, _ := cmd.Flags().GetBool("verbose")
	skipBuild, _ := cmd.Flags().GetBool("skip-build")
	noBuild, _ := cmd.Flags().GetBool("no-build")

	// --no-build is an alias for --skip-build
	if noBuild {
		skipBuild = true
	}

	// --all overrides --smart
	if all {
		smart = false
	}

	workdir, err := os.Getwd()
	if err != nil {
		ui.Error("Failed to determine working directory")
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Step 1: Load environment and validate
	totalSteps := 4
	if all {
		totalSteps = 5
	}
	stepNum := 1

	ui.Step(stepNum, totalSteps, "Loading environment")
	stepNum++

	composePath := filepath.Join(workdir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		ui.Error("docker-compose.yml not found. Run 'nself build' first.")
		return fmt.Errorf("docker-compose.yml not found in %s", workdir)
	}

	cfg, err := config.Load(workdir)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to load config: %v", err))
		return fmt.Errorf("loading config: %w", err)
	}

	if verbose {
		ui.Info(fmt.Sprintf("Project: %s", cfg.ProjectName))
		ui.Info(fmt.Sprintf("Working directory: %s", workdir))
		ui.Info(fmt.Sprintf("Mode: smart=%t all=%t skip-build=%t", smart, all, skipBuild))
	}

	// Read compose file manifest for multi-file support.
	composeFiles, err := build.ReadComposeManifest(workdir)
	if err != nil {
		if verbose {
			ui.Warn(fmt.Sprintf("Could not read compose manifest: %v (using defaults)", err))
		}
		composeFiles = nil
	}

	comp := docker.NewCompose(composeFiles...)
	comp.EnvFiles = build.ComposeEnvFiles(workdir)
	ctx := cmd.Context()
	services := args

	// Step 2: Determine restart strategy
	ui.Step(stepNum, totalSteps, "Determining restart strategy")
	stepNum++

	if all {
		// Full mode: stop all, wait, start all
		if err := restartFull(ctx, comp, workdir, cfg, verbose, skipBuild, stepNum, totalSteps); err != nil {
			return err
		}
	} else if len(services) > 0 {
		// Specific services: restart only those
		if err := restartServices(ctx, comp, workdir, services, verbose); err != nil {
			return err
		}
	} else if smart {
		// Smart mode: check if config changed
		if err := restartSmart(ctx, comp, workdir, cfg, verbose, skipBuild); err != nil {
			return err
		}
	} else {
		// Default: quick compose restart
		if err := restartQuick(ctx, comp, workdir, verbose); err != nil {
			return err
		}
	}

	// Health verification
	ui.Step(totalSteps, totalSteps, "Verifying health")

	if verbose {
		ui.Info("Waiting 5 seconds for services to stabilize...")
	}
	time.Sleep(5 * time.Second)

	report, err := health.RunAllChecks(ctx, cfg, workdir)
	if err != nil {
		ui.Warn(fmt.Sprintf("Health check error: %v", err))
	} else {
		for _, r := range report.Results {
			if r.Status == "healthy" {
				ui.Success(fmt.Sprintf("%-20s %s (%s)", r.Service, r.Status, r.Duration.Round(time.Millisecond)))
			} else {
				ui.Warn(fmt.Sprintf("%-20s %s (%s)", r.Service, r.Status, r.Details))
			}
		}
		fmt.Println()
		if report.Unhealthy > 0 {
			ui.Warn(fmt.Sprintf("%d/%d services healthy, %d unhealthy", report.Healthy, report.Total, report.Unhealthy))
		} else {
			ui.Success(fmt.Sprintf("All %d services healthy", report.Total))
		}
	}

	return nil
}

// restartFull performs a full stop + start cycle.
func restartFull(ctx context.Context, comp *docker.Compose, workdir string, cfg *config.Config, verbose, skipBuild bool, stepNum, totalSteps int) error {
	ui.Step(stepNum, totalSteps, "Stopping all services")
	stepNum++

	if err := comp.ComposeDown(ctx, workdir, docker.DownOptions{RemoveOrphans: true}); err != nil {
		ui.Error(fmt.Sprintf("Failed to stop services: %v", err))
		return fmt.Errorf("stopping services: %w", err)
	}
	ui.Success("All services stopped")

	if verbose {
		ui.Info("Waiting 2 seconds before restart...")
	}
	time.Sleep(2 * time.Second)

	// Rebuild if config changed and not skipped
	if !skipBuild {
		needsRebuild, err := build.NeedsRebuild(workdir)
		if err != nil {
			ui.Warn(fmt.Sprintf("Could not check build cache: %v", err))
		}
		if needsRebuild {
			ui.Info("Configuration changed, rebuilding...")
			opts := build.BuildOptions{Force: true}
			if _, err := build.Build(workdir, opts); err != nil {
				ui.Error(fmt.Sprintf("Rebuild failed: %v", err))
				return fmt.Errorf("rebuilding: %w", err)
			}
			ui.Success("Rebuild complete")
		}
	}

	ui.Step(stepNum, totalSteps, "Starting all services")

	if err := comp.ComposeUp(ctx, workdir); err != nil {
		ui.Error(fmt.Sprintf("Failed to start services: %v", err))
		return fmt.Errorf("starting services: %w", err)
	}
	ui.Success("All services started")

	return nil
}

// restartServices restarts only the specified services.
func restartServices(ctx context.Context, comp *docker.Compose, workdir string, services []string, verbose bool) error {
	if verbose {
		ui.Info(fmt.Sprintf("Restarting services: %v", services))
	}
	ui.Info(fmt.Sprintf("Restarting %d service(s)...", len(services)))

	if err := comp.ComposeRestart(ctx, workdir, services...); err != nil {
		ui.Error(fmt.Sprintf("Failed to restart services: %v", err))
		return fmt.Errorf("restarting services: %w", err)
	}
	ui.Success(fmt.Sprintf("Restarted: %v", services))

	return nil
}

// restartSmart compares .env mtime against container start times. If config
// is newer than any container, a full rebuild + up is performed. Otherwise,
// a quick compose restart is issued.
func restartSmart(ctx context.Context, comp *docker.Compose, workdir string, cfg *config.Config, verbose, skipBuild bool) error {
	envPath := filepath.Join(workdir, ".env")
	envInfo, err := os.Stat(envPath)
	if err != nil {
		if verbose {
			ui.Info("No .env file found, performing quick restart")
		}
		return restartQuick(ctx, comp, workdir, verbose)
	}
	envMtime := envInfo.ModTime()

	// Also check docker-compose.yml mtime
	composePath := filepath.Join(workdir, "docker-compose.yml")
	composeInfo, err := os.Stat(composePath)
	if err != nil {
		ui.Warn("docker-compose.yml not found, performing quick restart")
		return restartQuick(ctx, comp, workdir, verbose)
	}

	// Compare .env mtime and docker-compose.yml mtime: use the newer one
	configMtime := envMtime
	if composeInfo.ModTime().After(configMtime) {
		configMtime = composeInfo.ModTime()
	}

	// Get container start times
	containers, err := comp.ComposePs(ctx, workdir)
	if err != nil {
		if verbose {
			ui.Warn(fmt.Sprintf("Could not query containers: %v", err))
		}
		return restartQuick(ctx, comp, workdir, verbose)
	}

	// Check if any running container started before the config changed
	configChanged := false
	for _, c := range containers {
		info, err := docker.InspectContainer(ctx, c.Name)
		if err != nil {
			if verbose {
				ui.Warn(fmt.Sprintf("Could not inspect %s: %v", c.Name, err))
			}
			continue
		}
		startedAt, err := time.Parse(time.RFC3339Nano, info.StartedAt)
		if err != nil {
			if verbose {
				ui.Warn(fmt.Sprintf("Could not parse start time for %s: %v", c.Name, err))
			}
			continue
		}
		if configMtime.After(startedAt) {
			configChanged = true
			if verbose {
				ui.Info(fmt.Sprintf("Config newer than %s (config: %s, started: %s)",
					c.Name, configMtime.Format(time.RFC3339), startedAt.Format(time.RFC3339)))
			}
			break
		}
	}

	if configChanged && !skipBuild {
		ui.Info("Configuration changed since last start, rebuilding...")
		opts := build.BuildOptions{Force: true}
		if _, err := build.Build(workdir, opts); err != nil {
			ui.Error(fmt.Sprintf("Rebuild failed: %v", err))
			return fmt.Errorf("rebuilding: %w", err)
		}
		ui.Success("Rebuild complete")

		// Bring up with rebuilt config (--remove-orphans to clean up removed services)
		ui.Info("Applying updated configuration...")
		if err := comp.ComposeUp(ctx, workdir); err != nil {
			ui.Error(fmt.Sprintf("Failed to apply updated config: %v", err))
			return fmt.Errorf("applying updated config: %w", err)
		}
		ui.Success("Services restarted with updated configuration")
	} else {
		if configChanged && skipBuild {
			ui.Warn("Configuration changed but --skip-build specified, performing quick restart")
		} else if verbose {
			ui.Info("No configuration changes detected")
		}
		return restartQuick(ctx, comp, workdir, verbose)
	}

	return nil
}

// restartQuick performs a simple docker compose restart without rebuilding.
func restartQuick(ctx context.Context, comp *docker.Compose, workdir string, verbose bool) error {
	if verbose {
		ui.Info("Performing quick restart (no rebuild)")
	}
	ui.Info("Restarting services...")
	if err := comp.ComposeRestart(ctx, workdir); err != nil {
		ui.Error(fmt.Sprintf("Failed to restart: %v", err))
		return fmt.Errorf("restarting: %w", err)
	}
	ui.Success("Services restarted")
	return nil
}
