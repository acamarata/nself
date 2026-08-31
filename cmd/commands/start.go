package commands

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/database"
	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/embedded"
	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/health"
	"github.com/nself-org/cli/internal/lifecycle"
	"github.com/nself-org/cli/internal/migration"
	"github.com/nself-org/cli/internal/ports"
	"github.com/nself-org/cli/internal/search"
	"github.com/nself-org/cli/internal/ssl"
	"github.com/nself-org/cli/internal/telemetry"
	"github.com/nself-org/cli/internal/truststate"
	"github.com/nself-org/cli/internal/ui"

	"github.com/spf13/cobra"
)

// startCmd, its flags, and the RootCmd registration were extracted to
// start_cmd.go (T-P6-E2-W1-S1-T3) for 300-line compliance.

func runStart(cmd *cobra.Command, _ []string) error {
	opts, err := resolveStartOpts(cmd)
	if err != nil {
		return err
	}

	// Telemetry: capture start time and first-run state before execution.
	startTime := time.Now()
	firstRun := isFirstStart()

	// Telemetry: emit start_attempt immediately (opt-in only).
	if telemetry.IsOptedIn() {
		telemetry.Send("start_attempt", map[string]any{
			"first_run": firstRun,
		})
	}

	// Telemetry: deferred start_result emits on all exit paths (success or failure).
	var startErr error
	defer func() {
		if !telemetry.IsOptedIn() {
			return
		}
		props := map[string]any{
			"first_run":   firstRun,
			"duration_ms": time.Since(startTime).Milliseconds(),
			"success":     startErr == nil,
		}
		if startErr != nil {
			props["failure_category"] = classifyStartError(startErr)
		}
		telemetry.Send("start_result", props)
	}()

	// ── Plugin lifecycle: dormant banners (read-only — auto-removal is build-only) ──
	// Show warnings for dormant plugins so operators know to renew or rebuild.
	showDormantBannersOnStart(opts.quiet)

	cwd, startErr := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	allowLegacy, _ := cmd.Flags().GetBool("allow-legacy")

	projectDir, err := config.FindNSelfRoot(cwd)
	if err != nil {
		// Before reporting "no project found", check if this is a v0.9 directory.
		// FindNSelfRoot won't find a .env marker in a v0.9 project, so we must
		// probe cwd directly first.
		if count, names := migration.CheckLegacyProject(cwd); count >= migration.DetectionThreshold {
			if allowLegacy {
				ui.Warn(fmt.Sprintf("WARNING: v0.9 project detected (%d artifact(s): %s). Proceeding due to --allow-legacy (not recommended).", count, strings.Join(names, ", ")))
				projectDir = cwd
			} else {
				ui.Error(fmt.Sprintf("v0.9 project detected. Found %d legacy artifact(s): %s", count, strings.Join(names, ", ")))
				fmt.Fprintln(os.Stderr, "Run `nself migrate` first. See https://nself.org/docs/migrate/from-v0.9")
				startErr = fmt.Errorf("v0.9 project detected — run `nself migrate` first")
				return startErr
			}
		} else {
			return fmt.Errorf("no nself project found in current directory or parents. Run 'nself init' to create a project")
		}
	}

	// ── v0.9 artifact detection (S60-T02) ────────────────────────────────
	// Requires ≥2 of 5 heuristics to trigger (prevents false positives).
	// A single artifact warns but proceeds. ≥2 artifacts fail unless --allow-legacy.
	if count, names := migration.CheckLegacyProject(projectDir); count >= migration.DetectionThreshold {
		if allowLegacy {
			ui.Warn(fmt.Sprintf("WARNING: v0.9 project detected (%d artifact(s): %s). Proceeding due to --allow-legacy (not recommended).", count, strings.Join(names, ", ")))
		} else {
			ui.Error(fmt.Sprintf("v0.9 project detected. Found %d legacy artifact(s): %s", count, strings.Join(names, ", ")))
			fmt.Fprintln(os.Stderr, "Run `nself migrate` first. See https://nself.org/docs/migrate/from-v0.9")
			startErr = fmt.Errorf("v0.9 project detected — run `nself migrate` first")
			return startErr
		}
	} else if count == 1 {
		ui.Warn(fmt.Sprintf("One possible v0.9 artifact found (%s). Proceeding — run `nself migrate` if this is a v0.9 project.", names[0]))
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	lifecycle.TrapSignals(ctx, cancel, func() {}, 10*time.Second, os.Exit)

	totalSteps := 7
	currentStep := 0

	ui.CommandHeader("nSelf Start", "Booting your stack")

	// ── Docker preflight (first check, before anything else) ─────────
	if err := checkDockerAvailable(ctx); err != nil {
		return err
	}

	// ── AI auto-install (T-05-05) ──────────────────────────────────
	// If AI_AUTO_INSTALL=true (default in the AI config block), run doctor
	// --ai --yes --skip-pool to at least get Ollama running before the stack
	// boots.
	//
	// CLI-R18: the AI config block (and NSELF_MASTER_SECRET) used to live in
	// a dedicated .env.ai file, and its mere existence gated this block.
	// .env.ai is now folded into .env.secrets, so the gate is
	// NSELF_MASTER_SECRET presence in the resolved environment — the same
	// signal ("zero-config AI was set up for this project"), just read from
	// the cascade instead of the filesystem.
	if aiAutoInstall := os.Getenv("AI_AUTO_INSTALL"); aiAutoInstall == "" || strings.EqualFold(aiAutoInstall, "true") {
		if os.Getenv("NSELF_MASTER_SECRET") != "" {
			if !ollamaHealthy(ctx) {
				ui.Info("AI_AUTO_INSTALL: setting up local AI...")
				_ = runDoctorAI(ctx, doctorAIFlags{
					yes:        true,
					skipPool:   true,
					skipOllama: false,
					headless:   false,
					jsonOut:    false,
				})
			}
		}
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
			if _, err := build.Build(projectDir, build.BuildOptions{Profile: opts.profile}); err != nil {
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

	// ── DNS resolution check ─────────────────────────────────────────────
	dnsResolved := true
	if cfg.BaseDomain != "" && cfg.BaseDomain != "localhost" && cfg.BaseDomain != "127.0.0.1" {
		if _, err := net.LookupHost(cfg.BaseDomain); err != nil {
			dnsResolved = false
			ui.Warn(fmt.Sprintf("DNS not configured for %s — run 'nself dns-setup' or 'nself trust' to configure local DNS", cfg.BaseDomain))
		}
	}

	// ── BASE_DOMAIN change detection ─────────────────────────────────────
	if ts, err := truststate.Load(); err == nil {
		if ts.TrustedDomain != "" && ts.TrustedDomain != cfg.BaseDomain {
			ui.Warn(fmt.Sprintf("BASE_DOMAIN changed from %s to %s — SSL certificates need regeneration. Run 'nself trust' to reconfigure.", ts.TrustedDomain, cfg.BaseDomain))
		}
	}

	// ── Auto-trust prompt ────────────────────────────────────────────────
	if !dnsResolved {
		if ts, _ := truststate.Load(); ts.TrustedDomain != cfg.BaseDomain && isTerminal() {
			ui.Info("Would you like to run 'nself trust' now to configure local DNS? [Y/n]: ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(answer)
			if answer == "" || strings.ToLower(answer) == "y" || strings.ToLower(answer) == "yes" {
				if err := runDNSSetup(cmd, nil); err != nil {
					ui.Warn(fmt.Sprintf("DNS setup failed: %v", err))
				} else {
					_ = truststate.Save(truststate.TrustState{TrustedDomain: cfg.BaseDomain})
				}
			}
		}
	}

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

	// ── Read compose manifest (needed for port-ownership check) ─────
	// Read early — before the port check — so CheckAllPortsFiltered can
	// query only nself's own containers and avoid flagging them as conflicts.
	composeFiles, err := build.ReadComposeManifest(projectDir)
	if err != nil {
		if opts.verbose {
			ui.Warn(fmt.Sprintf("Could not read compose manifest: %v (using defaults)", err))
		}
		composeFiles = nil
	}

	// --skip-plugins: keep only the first entry (base docker-compose.yml).
	if opts.skipPlugins && len(composeFiles) > 1 {
		ui.Info("Skipping plugin compose files (--skip-plugins)")
		composeFiles = composeFiles[:1]
	}

	// ── Step 3: Port availability check ──────────────────────────────
	currentStep++
	ui.Step(currentStep, totalSteps, "Checking port availability")

	if opts.skipPortCheck {
		ui.Warn("Port check skipped (--skip-port-check)")
	} else {
		// Check the ports THIS stack will actually bind, read from the resolved
		// compose config, rather than a fixed list of nSelf defaults.
		//
		// A second project on the same host has to move off the defaults
		// (POSTGRES_PORT=5433, HASURA_PORT=8181, ...). Checking the default
		// list then blocks it on 5432 and 8080 — ports it was never going to
		// bind — purely because the first project holds them. That is not a
		// conflict, and it is what kept the ɳTask staging stack down: its
		// compose published 5433/8181/4001/6380/9010/9011, every one free,
		// while start refused over six defaults it does not use.
		checkPorts, portServiceMap, derr := docker.DeclaredHostPorts(ctx, projectDir, composeFiles...)
		if derr != nil || len(checkPorts) == 0 {
			// Fall back to the default list rather than check nothing. Checking
			// an empty list would be a port check that cannot fail, which is
			// worse than one that is occasionally too strict.
			if derr != nil {
				ui.Warn(fmt.Sprintf("Could not read published ports from compose (%v); using the default port list", derr))
			}
			checkPorts = docker.ReservedPorts
			portServiceMap = docker.DefaultPortServiceNames()
		}

		conflicts, err := docker.CheckAllPortsFiltered(ctx, checkPorts, projectDir, composeFiles...)
		if err != nil {
			ui.Warn(fmt.Sprintf("Port check error: %v", err))
		} else if len(conflicts) > 0 {
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

	compose := docker.NewCompose(composeFiles...)
	compose.EnvFiles = build.ComposeEnvFiles(projectDir)

	// ── Embedded PG path (--embedded-pg / NSELF_EMBEDDED_PG=true) ───
	// When embedded PG is requested, boot pglite via wasmtime instead of
	// the Docker postgres container. The WASM module listens on a Unix
	// domain socket; a wire-protocol bridge proxies Hasura's TCP traffic.
	//
	// startEmbeddedPGRuntime is conditionally compiled: the real implementation
	// lives in start_embedded_cgo.go (//go:build cgo) and a stub that returns
	// an error lives in start_embedded_nocgo.go (//go:build !cgo).
	if opts.embeddedPG {
		epgSp := ui.NewSpinner("Starting embedded PostgreSQL (pglite/wasmtime)...")
		epgSp.Start()

		runtimeDir := filepath.Join(projectDir, ".nself", "embedded-pg")
		if mkErr := os.MkdirAll(runtimeDir, 0o700); mkErr != nil {
			epgSp.Fail(fmt.Sprintf("Could not create embedded PG runtime dir: %v", mkErr))
			return fmt.Errorf("embedded-pg: mkdir runtime dir: %w", mkErr)
		}

		wasmPath, fetchErr := embedded.FetchOrCached(ctx, embedded.DefaultPGliteVersion)
		if fetchErr != nil {
			epgSp.Fail(fmt.Sprintf("Failed to fetch pglite WASM: %v", fetchErr))
			ui.UXError(
				"Embedded PG: pglite fetch failed",
				fetchErr.Error(),
				[]string{
					"Check network connectivity to ping.nself.org",
					"Try clearing the cache: rm -rf ~/.nself/cache/pglite",
					"Use standard PostgreSQL instead: remove --embedded-pg flag",
				},
			)
			return fmt.Errorf("embedded-pg: fetch wasm: %w", fetchErr)
		}

		epgCleanup, bridgeSockPath, epgErr := startEmbeddedPGRuntime(ctx, runtimeDir, wasmPath)
		if epgErr != nil {
			epgSp.Fail(fmt.Sprintf("Embedded PG failed to start: %v", epgErr))
			ui.UXError(
				"Embedded PG start failed",
				epgErr.Error(),
				[]string{
					"Embedded PostgreSQL (pglite/wasmtime) is EXPERIMENTAL and not yet functional — " +
						"the wasmtime shim does not yet implement the ~113 Emscripten host imports required.",
					"Use standard PostgreSQL instead: omit the --embedded-pg flag (nself start).",
					"Status: tracked in P104 residue; see FEATURES.md for details.",
				},
			)
			return epgErr
		}
		defer epgCleanup()

		epgSp.Success(fmt.Sprintf("Embedded PostgreSQL ready (pglite v%s / wasmtime v25.x)", embedded.DefaultPGliteVersion))
		ui.Info(fmt.Sprintf("  Socket: %s", bridgeSockPath))
		ui.Info("  Hasura will connect via UDS bridge — no Docker postgres container required")
	} else {
		// ── Standard Docker postgres path ───────────────────────────

		// ── First-run image pull ─────────────────────────────────────────
		// Detect first run via the .nself/.first-run-complete marker.
		// On first run, docker compose pull can take 1-3 minutes; show progress.
		firstRunMarker := filepath.Join(projectDir, ".nself", ".first-run-complete")
		if _, err := os.Stat(firstRunMarker); os.IsNotExist(err) {
			if !opts.quiet {
				ui.Info("First run detected — pulling Docker images (this takes 1-3 minutes on slow connections)...")
			}
			donePull := ui.FirstRunProgress(opts.quiet)
			pullCtx, pullCancel := context.WithTimeout(ctx, 10*time.Minute)
			_ = compose.ComposePull(pullCtx, projectDir)
			pullCancel()
			donePull()
			// Write the first-run marker so subsequent starts skip this step.
			if mkErr := os.MkdirAll(filepath.Dir(firstRunMarker), 0o700); mkErr == nil {
				f, fErr := os.OpenFile(firstRunMarker, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
				if fErr == nil {
					_ = f.Close()
				}
			}
		}

		// Remove stale hash-prefixed rename-leftover containers (interrupted
		// recreates) so they cannot hold ports or shadow the clean
		// <project>_<service> names (e.g. b6d7..._ntask_hasura, gap #21).
		_ = docker.RunPreStartCleanup(ctx, cfg.ProjectName)

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
	}

	// ── Step 5: Database initialization (Phase 2) ───────────────────
	// Run BEFORE other services start. The auth service requires the auth
	// schema to exist in PostgreSQL, so schemas and extensions must be
	// created while only postgres is running.
	// When --skip-db-init is set (CI/E2E mode) this entire step is bypassed:
	// migrations and seed are skipped, and the stack boots to a bare schema state.
	currentStep++
	ui.Step(currentStep, totalSteps, "Initializing database")

	if opts.skipDBInit {
		ui.Warn("Database initialization skipped (--skip-db-init / CI mode)")
	} else {
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
	}

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

	// Post-start cleanup: remove init containers (meilisearch-init)
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

	if opts.skipDBInit {
		// CI/E2E mode: wait only for the backend triad (postgres, hasura,
		// hasura-auth) to be healthy, then exit 0. Migrations and seed are
		// intentionally absent — the test suite manages schema state itself.
		if waitErr := waitCIReady(ctx, cfg, projectDir, opts.timeout, opts.verbose); waitErr != nil {
			startErr = fmt.Errorf("CI readiness gate failed: %w", waitErr)
			return startErr
		}
	} else if opts.skipHealthChecks {
		ui.Warn("Health checks skipped (--skip-health-checks)")
	} else {
		requiredPct := 80
		if opts.quick {
			requiredPct = 60
		}
		if cfg.HealthCheckRequired > 0 && !opts.quick {
			requiredPct = cfg.HealthCheckRequired
		}

		report := runHealthCheckLoop(ctx, cfg, projectDir, opts.timeout, requiredPct, opts.verbose)
		_ = report // Final report used inline by the loop for display.
	}

	// ── MeiliSearch index warm-up ────────────────────────────────────
	// Runs asynchronously so it never delays the startup sequence.
	// Activated only when MEILISEARCH_WARMUP_QUERIES is set and the
	// Search engine is meilisearch.
	if cfg.Search.Enabled && strings.EqualFold(cfg.Search.Engine, "meilisearch") {
		if warmupQueries := search.QueriesFromEnv(); len(warmupQueries) > 0 {
			go func() {
				warmupCfg := search.WarmupConfig{
					Host:        "localhost",
					Port:        cfg.Search.Port,
					MasterKey:   cfg.Search.MeiliSearch.MasterKey,
					IndexPrefix: cfg.Search.IndexPrefix,
					Queries:     warmupQueries,
				}
				result, _ := search.Warmup(context.Background(), warmupCfg)
				if opts.verbose {
					ui.Dimmed(fmt.Sprintf("  MeiliSearch warm-up: %d/%d queries succeeded",
						result.QueriesSucceeded, result.QueriesAttempted))
				}
			}()
		}
	}

	// ── Display URLs ─────────────────────────────────────────────────
	ui.Separator()

	// URL list: localhost:<port> endpoints for default local domains
	// (local.nself.org needs DNS/hosts setup and 502s on a fresh machine),
	// nginx-routed domain URLs for custom domains. See start_urls.go.
	ui.SummaryBox("nSelf Stack Running", stackURLs(cfg))

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

	// Telemetry: emit start_success (opt-in only).
	if telemetry.IsOptedIn() {
		telemetry.Send("start_success", map[string]any{
			"first_run":   firstRun,
			"duration_ms": time.Since(startTime).Milliseconds(),
		})
	}

	return nil
}
