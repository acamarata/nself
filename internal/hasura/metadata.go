// Package hasura provides the single step-function entry point that wires
// Hasura metadata application into the nself lifecycle (start/deploy), so
// that `<project>/hasura/metadata/` on disk is never silently ignored.
//
// Purpose: FIX-CLI-3 (P6 2026-09-04) — `nself build/start/deploy` never ran
// `hasura metadata apply`, so staging tracked 9 tables and prod 21 while the
// repo declared ~51 (46 np_/nself_/shopify_/idme_ tables never applied to
// either environment). This package closes that gap with one idempotent call.
// Inputs: a context, *config.Config, and the project directory.
// Outputs: an error only when metadata is present, apply fails, AND strict
// mode is on; otherwise a printed warning (dev default) or silent skip.
// Constraints: kept in its own package/file with a single call site in
// cmd/commands/start.go and the deploy paths, so it rebases cleanly against
// concurrent start*.go/internal/compose work tonight (coordination note,
// P6 crunch 2026-09-04).
package hasura

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/database"
	"github.com/nself-org/cli/internal/ui"
)

// applyFn/inconsistentFn/reachableFn are indirected for unit testing (no
// live Hasura required to exercise the strict/warn/skip decision tree).
type applyFn func(ctx context.Context, cfg *config.Config, projectDir string) error
type inconsistentFn func(ctx context.Context, cfg *config.Config) ([]string, error)
type reachableFn func(ctx context.Context, cfg *config.Config) bool

// ApplyIfPresent applies `<projectDir>/hasura/metadata/` to the running
// Hasura instance if that directory exists; otherwise it is a clean no-op.
//
// Idempotent: safe to call on every start/deploy — Hasura's replace_metadata
// API is itself idempotent (CLI-R already relies on this for `nself db
// hasura metadata apply`).
//
// Strictness: controlled by NSELF_HASURA_METADATA_STRICT (explicit override,
// "true"/"false"), else defaults to strict in staging/prod and warn-only in
// dev (cfg.Env). In warn mode a failed apply or inconsistency report prints
// but never fails the caller; in strict mode both fail it.
func ApplyIfPresent(ctx context.Context, cfg *config.Config, projectDir string) error {
	return applyIfPresent(ctx, cfg, projectDir, isHasuraReachable, database.HasuraApplyMetadata, database.HasuraGetInconsistentMetadata)
}

// isHasuraReachable reports whether *something* is listening on Hasura's
// configured port, distinguishing "Hasura isn't part of this stack / this
// run" (connection refused/timeout — skip cleanly, no warning) from "Hasura
// is part of the stack but returned an unhealthy status" (any HTTP response
// at all, including a 503 from HasuraHealthzHandler's degraded/down states —
// still worth attempting the apply, since a real instance may just be slow
// to come up). A short timeout keeps this cheap on every start/deploy.
func isHasuraReachable(ctx context.Context, cfg *config.Config) bool {
	port := cfg.Hasura.Port
	if port == 0 {
		port = 8080
	}
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fmt.Sprintf("http://localhost:%d/healthz", port), nil)
	if err != nil {
		return false
	}
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return true
}

// IsStrict reports whether metadata-apply failures should fail the caller,
// per NSELF_HASURA_METADATA_STRICT (explicit override) else cfg.Env
// (staging/prod = strict, dev = warn-only). Exported for reuse by the deploy
// paths, which need the same decision before deciding whether to fail a
// remote rolling restart.
func IsStrict(cfg *config.Config) bool {
	if raw := os.Getenv("NSELF_HASURA_METADATA_STRICT"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			return v
		}
		ui.Warn(fmt.Sprintf("NSELF_HASURA_METADATA_STRICT=%q is not a valid bool; ignoring", raw))
	}
	return cfg.Env == "staging" || cfg.IsProduction()
}

func applyIfPresent(ctx context.Context, cfg *config.Config, projectDir string, reachable reachableFn, apply applyFn, inconsistent inconsistentFn) error {
	metadataDir := filepath.Join(projectDir, "hasura", "metadata")
	metadataFile := filepath.Join(projectDir, "hasura", "metadata.json")
	if _, err := os.Stat(metadataDir); errors.Is(err, os.ErrNotExist) {
		if _, err := os.Stat(metadataFile); errors.Is(err, os.ErrNotExist) {
			return nil // No metadata tracked in this project — clean skip.
		}
	}

	if !reachable(ctx, cfg) {
		// Hasura isn't part of this run (e.g. a stack/profile that omits it,
		// or this call landing before the container is even scheduled) —
		// nothing to apply to. Never fails, strict mode included: this is not
		// a metadata problem, it's "there is no Hasura here right now."
		ui.Dimmed("  (hasura not reachable — skipping metadata apply)")
		return nil
	}

	strict := IsStrict(cfg)

	if err := apply(ctx, cfg, projectDir); err != nil {
		msg := fmt.Sprintf("hasura metadata apply failed: %v", err)
		if strict {
			return fmt.Errorf("%s (NSELF_HASURA_METADATA_STRICT=true; set false to warn-only)", msg)
		}
		ui.Warn(msg + " (continuing — set NSELF_HASURA_METADATA_STRICT=true to fail on this in dev)")
		return nil
	}

	names, err := inconsistent(ctx, cfg)
	if err != nil {
		// Reachability of get_inconsistent_metadata is best-effort; a failure
		// here must never mask a successful apply.
		ui.Dimmed(fmt.Sprintf("  (could not check metadata consistency: %v)", err))
		return nil
	}
	if len(names) > 0 {
		msg := fmt.Sprintf("hasura metadata applied with %d inconsistent object(s): %v", len(names), names)
		if strict {
			return fmt.Errorf("%s (NSELF_HASURA_METADATA_STRICT=true)", msg)
		}
		ui.Warn(msg)
		return nil
	}

	ui.Success("Hasura metadata applied (no inconsistencies)")
	return nil
}
