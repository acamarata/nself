package commands

import (
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
