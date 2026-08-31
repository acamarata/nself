package commands

// start_containers.go — Steps 4-6 (start PostgreSQL, initialize database,
// start remaining services) for `nself start`. Split out of start.go
// (T-P6-E2-W1-S1-T3) for 300-line compliance.
// Inputs:  ctx, opts, cfg, projectDir, composeFiles / compose.
// Outputs: startPostgresPhase returns the created *docker.Compose plus an
//          optional cleanup func (non-nil only on the embedded-PG path,
//          where the original code deferred it from runStart — the caller
//          must `defer cleanup()` itself to preserve that lifetime) and any
//          error. The other two return only error, matching the originals.
// Constraints: pure move, same checks/output/errors/order, no behavior
//              change. The embedded-PG cleanup must remain deferred from
//              runStart's scope, not this file's — see call sites in
//              start.go.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nself-org/cli/internal/build"
	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/database"
	"github.com/nself-org/cli/internal/docker"
	"github.com/nself-org/cli/internal/embedded"
	"github.com/nself-org/cli/internal/ui"
)

// startPostgresPhase starts PostgreSQL (Step 4): either the embedded
// pglite/wasmtime runtime (--embedded-pg / NSELF_EMBEDDED_PG=true) or the
// standard Docker postgres container. Returns the *docker.Compose to reuse
// for the remaining-services step, and — on the embedded-PG path only — a
// cleanup func the caller must defer from its own scope (runStart), since
// the original code deferred epgCleanup() directly from runStart.
//
// startEmbeddedPGRuntime is conditionally compiled: the real implementation
// lives in start_embedded_cgo.go (//go:build cgo) and a stub that returns
// an error lives in start_embedded_nocgo.go (//go:build !cgo).
func startPostgresPhase(ctx context.Context, opts startOpts, cfg *config.Config, projectDir string, composeFiles []string) (compose *docker.Compose, cleanup func(), err error) {
	compose = docker.NewCompose(composeFiles...)
	compose.EnvFiles = build.ComposeEnvFiles(projectDir)

	// ── Embedded PG path (--embedded-pg / NSELF_EMBEDDED_PG=true) ───
	// When embedded PG is requested, boot pglite via wasmtime instead of
	// the Docker postgres container. The WASM module listens on a Unix
	// domain socket; a wire-protocol bridge proxies Hasura's TCP traffic.
	if opts.embeddedPG {
		epgSp := ui.NewSpinner("Starting embedded PostgreSQL (pglite/wasmtime)...")
		epgSp.Start()

		runtimeDir := filepath.Join(projectDir, ".nself", "embedded-pg")
		if mkErr := os.MkdirAll(runtimeDir, 0o700); mkErr != nil {
			epgSp.Fail(fmt.Sprintf("Could not create embedded PG runtime dir: %v", mkErr))
			return compose, nil, fmt.Errorf("embedded-pg: mkdir runtime dir: %w", mkErr)
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
			return compose, nil, fmt.Errorf("embedded-pg: fetch wasm: %w", fetchErr)
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
			return compose, nil, epgErr
		}

		epgSp.Success(fmt.Sprintf("Embedded PostgreSQL ready (pglite v%s / wasmtime v25.x)", embedded.DefaultPGliteVersion))
		ui.Info(fmt.Sprintf("  Socket: %s", bridgeSockPath))
		ui.Info("  Hasura will connect via UDS bridge — no Docker postgres container required")
		return compose, epgCleanup, nil
	}

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
		return compose, nil, fmt.Errorf("compose up postgres: %w", err)
	}
	sp.Success("PostgreSQL container started")

	return compose, nil, nil
}

// initializeDatabasePhase runs database initialization (Step 5). Must run
// BEFORE other services start: the auth service requires the auth schema to
// exist in PostgreSQL, so schemas and extensions must be created while only
// postgres is running. When opts.skipDBInit is set (CI/E2E mode) this
// entire step is bypassed: migrations and seed are skipped, and the stack
// boots to a bare schema state.
func initializeDatabasePhase(ctx context.Context, cfg *config.Config, opts startOpts) error {
	if opts.skipDBInit {
		ui.Warn("Database initialization skipped (--skip-db-init / CI mode)")
		return nil
	}

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
	return nil
}

// startRemainingServicesPhase runs `docker compose up` for the rest of the
// stack (Step 6), then kicks off a best-effort background cleanup of exited
// init containers (meilisearch-init) and zombie containers from interrupted
// starts.
func startRemainingServicesPhase(ctx context.Context, compose *docker.Compose, projectDir string, cfg *config.Config) error {
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

	return nil
}
