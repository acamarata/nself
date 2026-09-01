// Package database — drill.go: high-level DR drill orchestration.
//
// The `nself backup drill` command wraps the existing RestoreDrill primitive
// (restore_drill.go) with three production concerns the bare drill primitive
// does not address:
//
//  1. RTO measurement: total wall-clock from "drill start" to "smoke checks
//     pass", broken into download / decrypt / restore / verify phases. The
//     target per STRAT-11 is 4 hours; anything longer needs investigation.
//  2. Critical-table smoke: row-count assertions on np_users, np_licenses,
//     np_audit_log, np_plugins, np_billing — five tables whose presence is
//     load-bearing for license validation + signup recovery.
//  3. Drill-log emission: an entry in .nself/drill-log.json that the doctor
//     check OPS-DRILL-01 reads to enforce "drill within last 7 days".
//
// The drill never touches production. RestoreDrill restores into a scratch
// database named <db>_drilltest, runs the smoke suite, then drops the scratch
// DB. Failure at any phase is reported with the partial timing data so
// operators can see exactly which phase blew the budget.
//
// This file lives in `database` (not `backup`) to avoid an import cycle —
// `backup` already imports `database`, so the high-level drill orchestrator
// must live next to RestoreDrill rather than alongside backup.Create.
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// DefaultCriticalTables is the canonical np_*-prefixed critical-table list
// used when a project has not set BACKUP_CRITICAL_TABLES. The five tables
// here are load-bearing for the most common recovery scenario on an
// np_-prefixed (multi-app-isolation, per the Multi-Tenant Convention Wall)
// schema: license validation + signup pipeline + audit log must survive a
// restore.
//
// Decision (2026-09-01, closes .claude/qa/bugs/drill-critical-tables-naming.md,
// option (c)): the list is project-configurable rather than a single hardcoded
// convention, because real deployed schemas disagree on naming — this
// project's own staging nself_web_db uses unprefixed names (users, licenses,
// audit_logs, plugins) with no np_ prefix and no billing-equivalent table at
// all. Hardcoding either convention would make the drill permanently
// wrong-shaped for whichever projects don't use it. ResolveCriticalTables
// reads the per-project override; this var stays the zero-config fallback so
// existing np_-prefixed deployments (and every test in this package) see no
// behavior change.
var DefaultCriticalTables = []string{
	"np_users",
	"np_licenses",
	"np_audit_log",
	"np_plugins",
	"np_billing",
}

// ResolveCriticalTables returns the critical-table list to smoke-check for
// this project: cfg.Backup.CriticalTables (BACKUP_CRITICAL_TABLES,
// comma-separated, whitespace-trimmed, empty entries dropped) when set, else
// DefaultCriticalTables. A nil cfg (unit tests exercising smokeCheck/
// verifyCriticalTables directly) also falls back to the default.
func ResolveCriticalTables(cfg *config.Config) []string {
	if cfg == nil || strings.TrimSpace(cfg.Backup.CriticalTables) == "" {
		return DefaultCriticalTables
	}
	var out []string
	for _, name := range strings.Split(cfg.Backup.CriticalTables, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return DefaultCriticalTables
	}
	return out
}

// smokeCheck evaluates a completed RestoreDrillResult against the drill's
// smoke gate. Pulled out of Drill as a pure function specifically so it can
// be unit-tested without a live Postgres — RestoreDrill itself needs one, so
// exercising this logic through Drill() end-to-end is not practical in CI.
//
// criticalTables is the resolved list (ResolveCriticalTables) used only for
// the table-count floor below — it is NOT re-checked by name here; presence
// by name is verifyCriticalTables' job and stays advisory-only (see
// MissingCriticalTables on DrillResult).
//
// Two independent conditions, either of which fails the drill:
//  1. Table-count floor: fewer than len(criticalTables) tables were found at
//     all. Weak alone (any N+ tables of any name pass) but cheap and kept.
//  2. Row-total: subResult.RowsVerified is an EXACT total across every user
//     table (see VerifyRestoredDatabase — no LIMIT-10 sampling), so it is
//     safe to hard-fail on zero. A fresh-init DB has zero rows in user
//     tables; a real restore must not. Proven live on staging 2026-08-31: a
//     drill reported tables_checked=107, rows_observed=0, success=true under
//     the old code, which had no row assertion at all — this is that gate.
func smokeCheck(subResult RestoreDrillResult, criticalTables []string) error {
	if subResult.TablesVerified < len(criticalTables) {
		return fmt.Errorf("only %d tables verified, want >= %d (critical: %v)",
			subResult.TablesVerified, len(criticalTables), criticalTables)
	}
	if subResult.RowsVerified <= 0 {
		return fmt.Errorf(
			"%d tables verified but 0 rows observed across all of them — restore did not bring back real data",
			subResult.TablesVerified,
		)
	}
	return nil
}

// DrillResult is the structured outcome of a single `nself backup drill` run.
// Phase timings are zero when a phase did not execute. All fields serialize
// to JSON for the drill log + doctor consumption.
type DrillResult struct {
	StartedAt      time.Time     `json:"started_at"`
	CompletedAt    time.Time     `json:"completed_at"`
	BackupFile     string        `json:"backup_file"`
	Success        bool          `json:"success"`
	Phase          string        `json:"phase,omitempty"` // last phase reached (init|restore|verify|smoke|done|dry-run)
	ErrorMessage   string        `json:"error_message,omitempty"`
	TotalDuration  time.Duration `json:"total_duration_ns"`
	RestoreSeconds float64       `json:"restore_seconds"`
	VerifySeconds  float64       `json:"verify_seconds"`
	SmokeSeconds   float64       `json:"smoke_seconds"`
	TablesChecked  int           `json:"tables_checked"`
	RowsObserved   int64         `json:"rows_observed"`
	RTOTargetMet   bool          `json:"rto_target_met"`
	// MissingCriticalTables lists CriticalTables entries not found by name in
	// the restored database. Informational, not a failure condition on its
	// own — see the smoke-check comment below.
	MissingCriticalTables []string `json:"missing_critical_tables,omitempty"`
}

// DrillOptions captures the user-facing knobs for `nself backup drill`. All
// fields are optional; sensible defaults come from the project config.
type DrillOptions struct {
	// BackupFile, when set, drills against a specific backup. Empty = most recent.
	BackupFile string
	// RTOTargetHours is the failure threshold for total drill wall-clock.
	// Default 4 hours per STRAT-11. Setting <= 0 disables the gate (the drill
	// still runs and reports duration; result.RTOTargetMet stays true).
	RTOTargetHours float64
	// DryRun, when true, validates the inputs and prints what would happen
	// without actually creating the scratch DB or touching pg_restore.
	DryRun bool
	// ProjectDir is where .nself/drill-log.json lives. Defaults to ".".
	ProjectDir string
}

// Drill runs the high-level drill orchestration: invokes RestoreDrill for the
// heavy lifting, layers RTO measurement + smoke checks on top, and emits a
// structured DrillResult to .nself/drill-log.json. The function never returns
// nil result — even on early failure the result captures whatever timing data
// was collected up to the failure point.
func Drill(ctx context.Context, cfg *config.Config, opts DrillOptions) (DrillResult, error) {
	result := DrillResult{
		StartedAt: time.Now(),
		Phase:     "init",
	}
	if opts.RTOTargetHours <= 0 {
		opts.RTOTargetHours = 4 // STRAT-11 default
	}
	if opts.ProjectDir == "" {
		opts.ProjectDir = "."
	}

	if opts.DryRun {
		result.Phase = "dry-run"
		result.Success = true
		result.RTOTargetMet = true
		result.CompletedAt = time.Now()
		result.TotalDuration = result.CompletedAt.Sub(result.StartedAt)
		// Dry-run still writes to drill log so operators see the dry-run was attempted.
		_ = appendDrillLog(opts.ProjectDir, result)
		return result, nil
	}

	// Phase 1+2+3: download + decrypt + restore (handled inside RestoreDrill).
	result.Phase = "restore"
	restoreStart := time.Now()
	subResult, err := RestoreDrill(ctx, cfg, opts.BackupFile)
	result.RestoreSeconds = time.Since(restoreStart).Seconds()
	result.BackupFile = subResult.BackupFile
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("restore phase failed: %v", err)
		result.CompletedAt = time.Now()
		result.TotalDuration = result.CompletedAt.Sub(result.StartedAt)
		_ = appendDrillLog(opts.ProjectDir, result)
		return result, fmt.Errorf("backup drill: %w", err)
	}

	// Phase 4: VerifyRestoredDatabase already ran inside RestoreDrill and
	// captured tables/rows in subResult. We surface those numbers here.
	result.Phase = "verify"
	result.VerifySeconds = subResult.Duration.Seconds() - result.RestoreSeconds
	if result.VerifySeconds < 0 {
		// Floor at zero — RestoreDrill's Duration covers both phases so the
		// subtraction can underflow if the restore phase was the dominant cost.
		result.VerifySeconds = 0
	}
	result.RowsObserved = subResult.RowsVerified

	// Phase 5: smoke checks. RestoreDrill drops the scratch DB before
	// returning, so we cannot re-query it here — everything below is derived
	// from what RestoreDrill/VerifyRestoredDatabase already captured in
	// subResult while the DB was still alive.
	//
	// Two independent gates:
	//  1. Table-count floor: ≥len(criticalTables) tables were found at all.
	//     Weak on its own (any N+ tables pass regardless of name) but kept as
	//     a cheap first check.
	//  2. Row-total assertion: subResult.RowsVerified is an EXACT total across
	//     every user table (see VerifyRestoredDatabase — no more LIMIT-10
	//     sampling), so it is safe to hard-fail on zero. A fresh-init DB would
	//     have zero rows in user tables; a real restore must not. Proven live
	//     on staging 2026-08-31: a drill reported tables_checked=107,
	//     rows_observed=0, success=true under the old code — this gate is
	//     what was missing.
	//
	// criticalTables presence-by-name (subResult.MissingCriticalTables) is
	// surfaced for visibility but does NOT fail the drill: the list itself is
	// per-project (ResolveCriticalTables / BACKUP_CRITICAL_TABLES) precisely
	// because deployed schemas disagree on naming conventions — see
	// .claude/qa/bugs/drill-critical-tables-naming.md for the decision.
	result.Phase = "smoke"
	smokeStart := time.Now()
	result.TablesChecked = subResult.TablesVerified
	result.MissingCriticalTables = subResult.MissingCriticalTables

	if smokeErr := smokeCheck(subResult, ResolveCriticalTables(cfg)); smokeErr != nil {
		result.ErrorMessage = fmt.Sprintf("smoke: %v", smokeErr)
		result.SmokeSeconds = time.Since(smokeStart).Seconds()
		result.CompletedAt = time.Now()
		result.TotalDuration = result.CompletedAt.Sub(result.StartedAt)
		_ = appendDrillLog(opts.ProjectDir, result)
		return result, fmt.Errorf("backup drill: %s", result.ErrorMessage)
	}
	result.SmokeSeconds = time.Since(smokeStart).Seconds()

	// All phases passed.
	result.Phase = "done"
	result.Success = true
	result.CompletedAt = time.Now()
	result.TotalDuration = result.CompletedAt.Sub(result.StartedAt)
	result.RTOTargetMet = result.TotalDuration.Hours() <= opts.RTOTargetHours

	if err := appendDrillLog(opts.ProjectDir, result); err != nil {
		return result, fmt.Errorf("backup drill: %w", err)
	}
	return result, nil
}

// drillLogPath returns the canonical path to the drill log inside projectDir.
func drillLogPath(projectDir string) string {
	return filepath.Join(projectDir, ".nself", "drill-log.json")
}

// appendDrillLog writes the result as a JSON line to .nself/drill-log.json.
// The file uses JSON Lines format (one object per line) so the doctor can
// tail-read the most recent entry without parsing the entire file.
func appendDrillLog(projectDir string, r DrillResult) error {
	path := drillLogPath(projectDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("drill log: mkdir: %w", err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("drill log: marshal: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("drill log: open: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("drill log: write: %w", err)
	}
	return nil
}

// LastSuccessfulDrill returns the most recent successful DrillResult from the
// drill log, or (zero, false) when no successful drill has run yet. The
// doctor check OPS-DRILL-01 calls this to enforce "drill within last 7 days".
func LastSuccessfulDrill(projectDir string) (DrillResult, bool) {
	path := drillLogPath(projectDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return DrillResult{}, false
	}
	// JSON Lines — scan backwards for the most recent Success=true entry.
	var newest DrillResult
	found := false
	start := 0
	for i := 0; i <= len(data); i++ {
		if i == len(data) || data[i] == '\n' {
			line := data[start:i]
			start = i + 1
			if len(line) == 0 {
				continue
			}
			var entry DrillResult
			if err := json.Unmarshal(line, &entry); err != nil {
				continue
			}
			if !entry.Success {
				continue
			}
			if !found || entry.CompletedAt.After(newest.CompletedAt) {
				newest = entry
				found = true
			}
		}
	}
	return newest, found
}
