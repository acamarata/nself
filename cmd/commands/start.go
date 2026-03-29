package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/database"
	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/health"
	"github.com/nself-org/cli/internal/license"
	"github.com/nself-org/cli/internal/lifecycle"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/nself-org/cli/internal/ports"
	"github.com/nself-org/cli/internal/ssl"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"up"},
	Short:   "Boot your nSelf stack",
	Long: `Boot the nSelf stack with health checks and automatic database initialization.

Executes the startup sequence:
  1. Validate docker-compose.yml exists
  2. Load environment configuration
  3. Check port availability
  4. Start PostgreSQL
  5. Initialize database (schemas, extensions)
  6. Start remaining services
  7. Run health checks on all services
  8. Display service URLs`,
	RunE: runStart,
}

func init() {
	f := startCmd.Flags()
	f.BoolP("verbose", "v", false, "Show detailed Docker output")
	f.BoolP("debug", "d", false, "Show debug information")
	f.Bool("skip-health-checks", false, "Skip health validation after startup")
	f.Int("timeout", 120, "Health check timeout in seconds (range: 30-600)")
	f.Bool("fresh", false, "Force recreate all containers")
	f.Bool("force-recreate", false, "Alias for --fresh")
	f.Bool("clean-start", false, "Remove all containers before starting")
	f.Bool("quick", false, "Quick start (timeout=30, required=60%)")
	f.Bool("skip-port-check", false, "Skip port availability check")
	f.Bool("skip-build", false, "Skip automatic rebuild detection")
	f.Bool("watch", false, "Enable health auto-restart: poll services and restart unhealthy containers")

	RootCmd.AddCommand(startCmd)
}

// startOpts holds the resolved flags for the start command.
type startOpts struct {
	verbose          bool
	debug            bool
	skipHealthChecks bool
	timeout          int
	fresh            bool
	cleanStart       bool
	quick            bool
	skipPortCheck    bool
	skipBuild        bool
	watch            bool
}

func resolveStartOpts(cmd *cobra.Command) (startOpts, error) {
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")
	skipHealth, _ := cmd.Flags().GetBool("skip-health-checks")
	timeout, _ := cmd.Flags().GetInt("timeout")
	fresh, _ := cmd.Flags().GetBool("fresh")
	forceRecreate, _ := cmd.Flags().GetBool("force-recreate")
	cleanStart, _ := cmd.Flags().GetBool("clean-start")
	quick, _ := cmd.Flags().GetBool("quick")
	skipPortCheck, _ := cmd.Flags().GetBool("skip-port-check")
	skipBuild, _ := cmd.Flags().GetBool("skip-build")
	watch, _ := cmd.Flags().GetBool("watch")

	// --force-recreate is an alias for --fresh.
	if forceRecreate {
		fresh = true
	}

	// --quick overrides timeout and required percentage.
	if quick {
		timeout = 30
	}

	// Clamp timeout to valid range.
	if timeout < 30 {
		timeout = 30
	}
	if timeout > 600 {
		timeout = 600
	}

	return startOpts{
		verbose:          verbose,
		debug:            debug,
		skipHealthChecks: skipHealth,
		timeout:          timeout,
		fresh:            fresh,
		cleanStart:       cleanStart,
		quick:            quick,
		skipPortCheck:    skipPortCheck,
		skipBuild:        skipBuild,
		watch:            watch,
	}, nil
}

func runStart(cmd *cobra.Command, _ []string) error {
	opts, err := resolveStartOpts(cmd)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	projectDir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	lifecycle.TrapSignals(ctx, cancel, func() {}, 10*time.Second)

	totalSteps := 7
	currentStep := 0

	ui.CommandHeader("nSelf Start", "Booting your stack")

	// ── Docker preflight (first check, before anything else) ─────────
	if err := checkDockerAvailable(ctx); err != nil {
		return err
	}

	// ── Auto-build detection ────────────────────────────────────────
	// Run BEFORE the docker-compose.yml check because build creates it.
	if !opts.skipBuild {
		needsRebuild, err := build.NeedsRebuild(projectDir)
		if err != nil {
			return fmt.Errorf("checking build state: %w", err)
		}
		if needsRebuild {
			ui.Info("Configuration changed — rebuilding before start...")
			if _, err := build.Build(projectDir, build.BuildOptions{}); err != nil {
				return fmt.Errorf("auto-build failed: %w", err)
			}
			ui.Success("Build completed")
		}
	}

	// ── Step 1: Validate docker-compose.yml ──────────────────────────
	currentStep++
	ui.Step(currentStep, totalSteps, "Checking docker-compose.yml")

	composePath := filepath.Join(projectDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		ui.UXError(
			"docker-compose.yml not found",
			fmt.Sprintf("Looked in %s", projectDir),
			[]string{
				"Run 'nself build' to generate your compose configuration",
				"Make sure you are in the correct project directory",
			},
		)
		return fmt.Errorf("docker-compose.yml not found in %s: %w", projectDir, errs.ErrComposeNotFound)
	}
	ui.Success("docker-compose.yml found")

	// ── Step 2: Load configuration ───────────────────────────────────
	currentStep++
	ui.Step(currentStep, totalSteps, "Loading configuration")

	cfg, err := config.Load(projectDir)
	if err != nil {
		ui.UXError(
			"Failed to load configuration",
			err.Error(),
			[]string{
				"Check your .env file for syntax errors",
				"Run 'nself init' to regenerate configuration",
			},
		)
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply env-level overrides for skip-health-checks.
	if cfg.SkipHealthChecks {
		opts.skipHealthChecks = true
	}
	// Apply env-level timeout if flag was not explicitly set.
	if !cmd.Flags().Changed("timeout") && !opts.quick && cfg.HealthCheckTimeout > 0 {
		opts.timeout = cfg.HealthCheckTimeout
		if opts.timeout < 30 {
			opts.timeout = 30
		}
		if opts.timeout > 600 {
			opts.timeout = 600
		}
	}

	// Validate config before proceeding. The auto-build path already runs
	// validation inside build.Build(), but we must also validate here for
	// the --skip-build path or when docker-compose.yml already exists.
	if err := config.Validate(cfg); err != nil {
		ui.UXError(
			"Configuration validation failed",
			err.Error(),
			[]string{
				"Review your .env file and fix the reported issues",
				"Run 'nself build --check' to re-validate",
			},
		)
		return fmt.Errorf("config validation failed: %w", err)
	}

	if opts.verbose {
		ui.Info(fmt.Sprintf("Project: %s | Domain: %s | Env: %s", cfg.ProjectName, cfg.BaseDomain, cfg.Env))
	}
	ui.Success(fmt.Sprintf("Configuration loaded (%s)", cfg.Env))

	// ── License revalidation heartbeat ──────────────────────────────
	// Soft check: warn on revoked/expired licenses but never block
	// existing services from starting. Only new plugin installs are
	// blocked by invalid licenses.
	checkLicenseHeartbeat(ctx, cfg, opts.verbose)

	// ── Cert expiry preflight ────────────────────────────────────────
	certDirName := strings.ReplaceAll(cfg.BaseDomain, ".", "-")
	certPath := filepath.Join(projectDir, "certificates", certDirName, "fullchain.pem")
	if days, err := ssl.CheckCertExpiry(certPath); err != nil {
		ui.Warn(fmt.Sprintf("TLS certificate issue: %v", err))
	} else if days < 30 {
		ui.Warn(fmt.Sprintf("TLS certificate expires in %d days — renew soon", days))
	}

	// ── Step 3: Port availability check ──────────────────────────────
	currentStep++
	ui.Step(currentStep, totalSteps, "Checking port availability")

	if opts.skipPortCheck {
		ui.Warn("Port check skipped (--skip-port-check)")
	} else {
		conflicts, err := docker.CheckAllPorts(docker.ReservedPorts)
		if err != nil {
			ui.Warn(fmt.Sprintf("Port check error: %v", err))
		} else if len(conflicts) > 0 {
			// Map ports to their service names for clear diagnostics.
			portServiceMap := map[int]string{
				80:   "HTTP (Nginx)",
				443:  "HTTPS (Nginx)",
				5432: "PostgreSQL",
				8080: "Hasura GraphQL",
				4000: "Auth",
				6379: "Redis",
				9000: "MinIO",
				9001: "MinIO Console",
				7700: "MeiliSearch",
				3021: "nSelf Admin",
				1025: "Mailpit SMTP",
				8025: "Mailpit UI",
				3008: "Functions",
				5000: "MLflow",
			}

			var portList []string
			for _, c := range conflicts {
				svc := portServiceMap[c.Port]
				if svc == "" {
					svc = "unknown service"
				}
				// Attempt to identify the holder process for a richer message.
				holder, _ := ports.WhoHoldsPort(c.Port)
				var detail string
				if holder != nil {
					detail = fmt.Sprintf("Port %d (%s): %s", c.Port, svc,
						ports.FormatConflictMessage(c.Port, holder))
				} else {
					detail = fmt.Sprintf("Port %d (%s) is already in use", c.Port, svc)
				}
				ui.Error(detail)
				portList = append(portList, fmt.Sprintf("%d", c.Port))
			}

			suggestions := []string{
				fmt.Sprintf("Find what is using these ports: lsof -i :%s", strings.Join(portList, " -i :")),
				"Stop the conflicting services before starting nSelf",
				"Or change the conflicting ports in your .env file (e.g., NGINX_HTTP_PORT, HASURA_PORT, POSTGRES_EXPOSE_PORT)",
				"Use --skip-port-check to start anyway (not recommended)",
			}
			ui.UXError(
				fmt.Sprintf("Port conflicts detected (%d port(s))", len(conflicts)),
				fmt.Sprintf("Ports in use: %s", strings.Join(portList, ", ")),
				suggestions,
			)
			return fmt.Errorf("port conflicts detected: %d port(s) in use", len(conflicts))
		} else {
			ui.Success("All ports available")
		}
	}

	// ── Step 4: Start PostgreSQL (Phase 1) ──────────────────────────
	currentStep++
	ui.Step(currentStep, totalSteps, "Starting PostgreSQL")

	// Read compose file manifest for multi-file support.
	// If .nself/compose-files.txt exists, pass the listed files to docker compose
	// via -f flags. Otherwise fall back to Docker's default discovery.
	composeFiles, err := build.ReadComposeManifest(projectDir)
	if err != nil {
		if opts.verbose {
			ui.Warn(fmt.Sprintf("Could not read compose manifest: %v (using defaults)", err))
		}
		composeFiles = nil
	}

	compose := docker.NewCompose(composeFiles...)

	// Clean start: remove all containers first.
	if opts.cleanStart {
		sp := ui.NewSpinner("Removing existing containers...")
		sp.Start()
		if err := compose.ComposeDown(ctx, projectDir, docker.DownOptions{
			RemoveOrphans: true,
		}); err != nil {
			sp.Fail(fmt.Sprintf("Cleanup failed: %v", err))
			// Non-fatal: continue with startup.
		} else {
			sp.Success("Existing containers removed")
		}
	}

	// Fresh mode: force recreate by running down first.
	if opts.fresh && !opts.cleanStart {
		sp := ui.NewSpinner("Forcing container recreation...")
		sp.Start()
		if err := compose.ComposeDown(ctx, projectDir, docker.DownOptions{}); err != nil {
			sp.Fail(fmt.Sprintf("Recreation cleanup failed: %v", err))
			// Non-fatal: continue with startup.
		} else {
			sp.Success("Containers cleared for fresh start")
		}
	}

	// Phase 1: Start only postgres so schemas can be created before auth starts.
	sp := ui.NewSpinner("Starting PostgreSQL...")
	sp.Start()
	if err := compose.ComposeUp(ctx, projectDir, "postgres"); err != nil {
		sp.Fail("Failed to start PostgreSQL")
		ui.UXError(
			"Failed to start PostgreSQL",
			err.Error(),
			[]string{
				"Check Docker is running: docker info",
				"Review logs: nself logs postgres",
				"Try a clean start: nself start --clean-start",
				"Rebuild: nself build --force",
			},
		)
		return fmt.Errorf("compose up postgres: %w", err)
	}
	sp.Success("PostgreSQL container started")

	// ── Step 5: Database initialization (Phase 2) ───────────────────
	// Run BEFORE other services start. The auth service requires the auth
	// schema to exist in PostgreSQL, so schemas and extensions must be
	// created while only postgres is running.
	currentStep++
	ui.Step(currentStep, totalSteps, "Initializing database")

	dbSp := ui.NewSpinner("Waiting for PostgreSQL ready and initializing schemas...")
	dbSp.Start()

	dbCtx, dbCancel := context.WithTimeout(ctx, 90*time.Second)
	defer dbCancel()

	if err := database.InitializeDatabase(dbCtx, cfg); err != nil {
		dbSp.Fail("Database initialization failed")
		ui.UXError(
			"Database initialization failed",
			err.Error(),
			[]string{
				"Check PostgreSQL logs: nself logs postgres",
				"Verify POSTGRES_USER and POSTGRES_PASSWORD in .env",
				"Try restarting: nself restart",
			},
		)
		return fmt.Errorf("database init: %w", err)
	}
	dbSp.Success("Database initialized (schemas, extensions, grants)")

	// ── Step 6: Start remaining services (Phase 3) ──────────────────
	currentStep++
	ui.Step(currentStep, totalSteps, "Starting remaining services")

	svcSp := ui.NewSpinner("Running docker compose up...")
	svcSp.Start()
	if err := compose.ComposeUp(ctx, projectDir); err != nil {
		svcSp.Fail("Docker compose up failed")
		ui.UXError(
			"Failed to start services",
			err.Error(),
			[]string{
				"Check Docker is running: docker info",
				"Review logs: nself logs",
				"Try a clean start: nself start --clean-start",
				"Rebuild: nself build --force",
			},
		)
		return fmt.Errorf("compose up: %w", err)
	}
	svcSp.Success("All services started")

	// Post-start cleanup: remove init containers (minio-init, meilisearch-init)
	// that completed and are sitting in exited state, plus zombie containers
	// from interrupted starts.
	go func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = docker.RunPostStartCleanup(cleanupCtx, cfg.ProjectName)
	}()

	// ── Health auto-restart (--watch) ────────────────────────────────
	// Start the Restarter in a background goroutine. It polls running containers
	// and automatically restarts any that become unhealthy, up to MaxAttempts.
	if opts.watch {
		restartPolicy := health.DefaultRestartPolicy()
		dockerClient := health.NewShellDockerClient(cfg.ProjectName)
		restarter := health.NewRestarter(dockerClient, restartPolicy)
		if err := restarter.Start(ctx); err != nil {
			ui.Warn(fmt.Sprintf("Health auto-restart could not start: %v", err))
		} else {
			ui.Info(fmt.Sprintf("Health auto-restart enabled (poll: %s, max attempts: %d)",
				restartPolicy.PollInterval, restartPolicy.MaxAttempts))
		}
	}

	// ── Step 7: Health checks ────────────────────────────────────────
	currentStep++
	ui.Step(currentStep, totalSteps, "Running health checks")

	if opts.skipHealthChecks {
		ui.Warn("Health checks skipped (--skip-health-checks)")
	} else {
		requiredPct := 80
		if opts.quick {
			requiredPct = 60
		}
		if cfg.HealthCheckRequired > 0 && !opts.quick {
			requiredPct = cfg.HealthCheckRequired
		}

		report := runHealthCheckLoop(ctx, cfg, opts.timeout, requiredPct, opts.verbose)
		_ = report // Final report used inline by the loop for display.
	}

	// ── Display URLs ─────────────────────────────────────────────────
	ui.Separator()

	domain := cfg.BaseDomain
	if domain == "" {
		domain = "localhost"
	}

	urls := []string{
		fmt.Sprintf("GraphQL:  https://%s/v1/graphql", domain),
		fmt.Sprintf("Console:  https://%s/console", domain),
		fmt.Sprintf("Auth:     https://%s/v1/auth", domain),
	}

	if cfg.Minio.Enabled {
		urls = append(urls, fmt.Sprintf("Storage:  https://%s/v1/storage", domain))
	}
	if cfg.Mailpit.Enabled {
		urls = append(urls, fmt.Sprintf("Mail UI:  https://%s/mailpit", domain))
	}
	if cfg.Monitoring.GrafanaEnabled {
		urls = append(urls, fmt.Sprintf("Grafana:  https://%s/grafana", domain))
	}
	if cfg.Admin.Enabled {
		urls = append(urls, "Admin:    http://localhost:3021")
	}

	ui.SummaryBox("nSelf Stack Running", urls)

	if opts.debug {
		ui.Section("Debug Info")
		ui.Dimmed(fmt.Sprintf("  Project dir:  %s", projectDir))
		ui.Dimmed(fmt.Sprintf("  Project name: %s", cfg.ProjectName))
		ui.Dimmed(fmt.Sprintf("  Environment:  %s", cfg.Env))
		ui.Dimmed(fmt.Sprintf("  Timeout:      %ds", opts.timeout))
		ui.Dimmed(fmt.Sprintf("  Quick mode:   %v", opts.quick))
		ui.Dimmed(fmt.Sprintf("  Fresh:        %v", opts.fresh))
		ui.Dimmed(fmt.Sprintf("  Clean start:  %v", opts.cleanStart))
	}

	return nil
}

// runHealthCheckLoop polls service health until the required percentage is met
// or the timeout expires. It updates a spinner with live progress and prints
// per-service status on completion.
func runHealthCheckLoop(ctx context.Context, cfg *config.Config, timeoutSec, requiredPct int, verbose bool) *health.HealthReport {
	timeout := time.Duration(timeoutSec) * time.Second
	healthCtx, healthCancel := context.WithTimeout(ctx, timeout)
	defer healthCancel()

	const pollInterval = 3 * time.Second
	var lastReport *health.HealthReport

	// Show initial spinner.
	hcSp := ui.NewSpinner(fmt.Sprintf("Waiting for services... (timeout: %ds)", timeoutSec))
	hcSp.Start()

	for {
		report, err := health.RunAllChecks(healthCtx, cfg)
		if err != nil {
			// Transient error: retry unless we are out of time.
			select {
			case <-healthCtx.Done():
				hcSp.Fail(fmt.Sprintf("Health check timed out after %ds: %v", timeoutSec, err))
				return lastReport
			case <-time.After(pollInterval):
				continue
			}
		}

		lastReport = report
		actualPct := 0
		if report.Total > 0 {
			actualPct = (report.Healthy * 100) / report.Total
		}

		// Success: threshold met.
		if actualPct >= requiredPct {
			hcSp.Success(fmt.Sprintf("Health checks passed: %d/%d healthy (%d%%)",
				report.Healthy, report.Total, actualPct))
			printServiceDetails(report, verbose)
			return report
		}

		// All services healthy (edge case where total is 0 or rounding).
		if report.Healthy == report.Total && report.Total > 0 {
			hcSp.Success(fmt.Sprintf("All %d services healthy", report.Total))
			printServiceDetails(report, verbose)
			return report
		}

		// Not yet healthy: stop current spinner, print progress, start fresh spinner.
		hcSp.Stop()
		ui.Info(fmt.Sprintf("Waiting for services... %d/%d healthy (%d%%, need %d%%)",
			report.Healthy, report.Total, actualPct, requiredPct))
		hcSp = ui.NewSpinner(fmt.Sprintf("Retrying health checks... %d/%d healthy", report.Healthy, report.Total))
		hcSp.Start()

		// Wait before next poll, or bail if context is done.
		select {
		case <-healthCtx.Done():
			hcSp.Fail(fmt.Sprintf("Health checks below threshold after %ds: %d/%d healthy (%d%%, required %d%%): %s",
				timeoutSec, report.Healthy, report.Total, actualPct, requiredPct, errs.ErrHealthTimeout))
			// Show which services are still unhealthy.
			for _, r := range report.Results {
				if r.Status != "healthy" {
					ui.Warn(fmt.Sprintf("  %s %s: %s (%s)", "\u2717", r.Service, r.Status, r.Details))
				}
			}
			return report
		case <-time.After(pollInterval):
			// Continue polling.
		}
	}
}

// printServiceDetails prints per-service health results. In normal mode it only
// shows unhealthy services. In verbose mode it shows all services with response times.
func printServiceDetails(report *health.HealthReport, verbose bool) {
	if verbose {
		ui.Section("Service Health Details")
		for _, r := range report.Results {
			if r.Status == "healthy" {
				ui.Success(fmt.Sprintf("  %s %s: healthy (%s)", "\u2713", r.Service, r.Duration))
			} else {
				ui.Warn(fmt.Sprintf("  %s %s: %s (%s)", "\u2717", r.Service, r.Status, r.Details))
			}
		}
	} else {
		// Show only unhealthy services in non-verbose mode.
		for _, r := range report.Results {
			if r.Status != "healthy" {
				ui.Warn(fmt.Sprintf("  %s %s: %s (%s)", "\u2717", r.Service, r.Status, r.Details))
			}
		}
	}
}

// checkDockerAvailable verifies Docker is installed and running before starting
// the stack. It distinguishes between Docker not being installed (binary absent
// from PATH) and Docker being installed but the daemon not running.
func checkDockerAvailable(ctx context.Context) error {
	if _, lookErr := exec.LookPath("docker"); lookErr != nil {
		return fmt.Errorf("docker binary not found: %w", errs.ErrDockerNotInstalled)
	}
	dockerCmd := exec.CommandContext(ctx, "docker", "info")
	dockerCmd.Stdout = nil
	dockerCmd.Stderr = nil
	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("docker info failed — start Docker and try again: %w", errs.ErrDockerNotRunning)
	}
	return nil
}

// checkLicenseHeartbeat performs a soft license revalidation if the cached
// validation is older than 7 days. It warns on revoked or expired licenses
// but never blocks the startup sequence. Network errors are treated as
// informational only.
func checkLicenseHeartbeat(ctx context.Context, cfg *config.Config, verbose bool) {
	key, err := license.GetKey()
	if err != nil || key == "" {
		return
	}

	cacheDir := plugin.LicenseCacheDir()
	if !plugin.NeedsRevalidation(key, cacheDir) {
		if verbose {
			ui.Dimmed("  License cache is current (revalidation not needed)")
		}
		return
	}

	if verbose {
		ui.Info("License cache is stale — revalidating...")
	}

	pingURL := cfg.PluginSystem.PingURL
	if pingURL == "" {
		pingURL = "https://ping.nself.org"
	}

	revalCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	valid, err := plugin.ValidateLicenseRemote(revalCtx, key, pingURL)
	if err != nil {
		// Network error: informational only, don't block startup.
		ui.Info("License revalidation skipped (network unavailable)")
		if verbose {
			ui.Dimmed(fmt.Sprintf("  %v", err))
		}
		return
	}

	// Update the cache with the fresh result.
	_ = plugin.CacheLicense(key, valid, cacheDir)

	if valid {
		if verbose {
			ui.Success("License revalidated successfully")
		}
	} else {
		// License revoked or expired: warn but don't block existing services.
		ui.Warn("License validation failed — your license may be revoked or expired")
		ui.Warn("Existing services will continue running. New plugin installs may be blocked.")
		ui.Warn("Visit https://nself.org/pricing to check your subscription status.")
	}
}
