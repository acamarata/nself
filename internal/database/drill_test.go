// S9.T03 — unit tests for the high-level backup drill orchestration.
package database

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// TestDrill_DryRun: --dry-run returns Success without touching the DB and
// still appends an entry to the drill log so operators see the run.
func TestDrill_DryRun(t *testing.T) {
	dir := t.TempDir()
	r, err := Drill(context.Background(), &config.Config{}, DrillOptions{
		DryRun:     true,
		ProjectDir: dir,
	})
	if err != nil {
		t.Fatalf("dry-run drill: %v", err)
	}
	if !r.Success {
		t.Errorf("dry-run: want Success=true, got false")
	}
	if r.Phase != "dry-run" {
		t.Errorf("dry-run: want Phase=dry-run, got %q", r.Phase)
	}
	if !r.RTOTargetMet {
		t.Errorf("dry-run: RTO target should be vacuously met")
	}

	// drill-log.json must exist and contain exactly one JSONL entry.
	logPath := filepath.Join(dir, ".nself", "drill-log.json")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read drill log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("dry-run: want 1 log line, got %d", len(lines))
	}
	var parsed DrillResult
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Errorf("dry-run log entry not valid JSON: %v", err)
	}
}

// TestLastSuccessfulDrill_Empty: no log file → (zero, false).
func TestLastSuccessfulDrill_Empty(t *testing.T) {
	dir := t.TempDir()
	_, ok := LastSuccessfulDrill(dir)
	if ok {
		t.Errorf("empty dir: want ok=false, got true")
	}
}

// TestLastSuccessfulDrill_PicksMostRecent: write three entries (two success,
// one failure) and verify the latest success wins.
func TestLastSuccessfulDrill_PicksMostRecent(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	entries := []DrillResult{
		{StartedAt: now.Add(-72 * time.Hour), CompletedAt: now.Add(-71 * time.Hour), Success: true, BackupFile: "old"},
		{StartedAt: now.Add(-48 * time.Hour), CompletedAt: now.Add(-47 * time.Hour), Success: false, BackupFile: "failed"},
		{StartedAt: now.Add(-24 * time.Hour), CompletedAt: now.Add(-23 * time.Hour), Success: true, BackupFile: "newest"},
	}
	for _, e := range entries {
		if err := appendDrillLog(dir, e); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	got, ok := LastSuccessfulDrill(dir)
	if !ok {
		t.Fatalf("want ok=true with successful entries present")
	}
	if got.BackupFile != "newest" {
		t.Errorf("want newest success entry, got %q", got.BackupFile)
	}
}

// TestLastSuccessfulDrill_IgnoresFailures: when only failure entries exist,
// returns (zero, false).
func TestLastSuccessfulDrill_IgnoresFailures(t *testing.T) {
	dir := t.TempDir()
	if err := appendDrillLog(dir, DrillResult{
		StartedAt:    time.Now(),
		CompletedAt:  time.Now(),
		Success:      false,
		ErrorMessage: "test failure",
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, ok := LastSuccessfulDrill(dir); ok {
		t.Errorf("want ok=false when only failures logged")
	}
}

// TestCriticalTables_Stable: changing the canonical default critical-table
// list is a breaking change to the drill smoke suite. This test pins the
// list so any modification is intentional + reviewed.
func TestCriticalTables_Stable(t *testing.T) {
	want := []string{
		"np_users",
		"np_licenses",
		"np_audit_log",
		"np_plugins",
		"np_billing",
	}
	if len(DefaultCriticalTables) != len(want) {
		t.Fatalf("DefaultCriticalTables length changed: want %d, got %d", len(want), len(DefaultCriticalTables))
	}
	for i, name := range want {
		if DefaultCriticalTables[i] != name {
			t.Errorf("DefaultCriticalTables[%d]: want %q, got %q", i, name, DefaultCriticalTables[i])
		}
	}
}

// TestResolveCriticalTables_DefaultsWhenUnset covers the zero-config path:
// no cfg, or a cfg with no BACKUP_CRITICAL_TABLES override, must return
// DefaultCriticalTables unchanged — every np_-prefixed deployment (and every
// other test in this file) depends on this staying true.
func TestResolveCriticalTables_DefaultsWhenUnset(t *testing.T) {
	if got := ResolveCriticalTables(nil); len(got) != len(DefaultCriticalTables) {
		t.Fatalf("ResolveCriticalTables(nil) = %v, want %v", got, DefaultCriticalTables)
	}
	if got := ResolveCriticalTables(&config.Config{}); len(got) != len(DefaultCriticalTables) {
		t.Fatalf("ResolveCriticalTables(empty cfg) = %v, want %v", got, DefaultCriticalTables)
	}
}

// TestResolveCriticalTables_OverrideSplitsAndTrims proves the
// BACKUP_CRITICAL_TABLES override (closes
// .claude/qa/bugs/drill-critical-tables-naming.md option (c)) is parsed as a
// comma-separated, whitespace-trimmed list — the unprefixed convention this
// project's own staging nself_web_db actually uses.
func TestResolveCriticalTables_OverrideSplitsAndTrims(t *testing.T) {
	cfg := &config.Config{Backup: config.BackupConfig{
		CriticalTables: "users, licenses,audit_logs ,plugins",
	}}
	got := ResolveCriticalTables(cfg)
	want := []string{"users", "licenses", "audit_logs", "plugins"}
	if len(got) != len(want) {
		t.Fatalf("ResolveCriticalTables = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ResolveCriticalTables[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestResolveCriticalTables_BlankOverrideFallsBackToDefault guards against a
// set-but-empty/whitespace-only override silently producing a zero-length
// critical-table list (which would make smokeCheck's table-count floor
// trivially always pass).
func TestResolveCriticalTables_BlankOverrideFallsBackToDefault(t *testing.T) {
	cfg := &config.Config{Backup: config.BackupConfig{CriticalTables: "  , , "}}
	got := ResolveCriticalTables(cfg)
	if len(got) != len(DefaultCriticalTables) {
		t.Fatalf("ResolveCriticalTables(blank override) = %v, want default %v", got, DefaultCriticalTables)
	}
}

// TestSmokeCheck_ZeroRowsFails is the inversion proof for the hollow-gate bug:
// a restore that reports plenty of tables but zero rows across all of them
// must FAIL the drill. Proven live on staging 2026-08-31 — a real drill
// returned tables_checked=107, rows_observed=0, success=true — because the
// old smoke gate only ever compared TablesVerified against len(CriticalTables)
// and never looked at RowsVerified at all. Against the code before this fix,
// this case reports err == nil (the bug); against the fix, it must fail.
func TestSmokeCheck_ZeroRowsFails(t *testing.T) {
	sub := RestoreDrillResult{
		Success:        true,
		TablesVerified: 107, // well over len(DefaultCriticalTables) — the old gate's only check
		RowsVerified:   0,   // fresh-init / nothing-really-restored signal
	}
	if err := smokeCheck(sub, DefaultCriticalTables); err == nil {
		t.Fatalf("smokeCheck: want error for 0 rows observed across %d tables, got nil (hollow gate)", sub.TablesVerified)
	}
}

// TestSmokeCheck_RealDataPasses: a restore with real rows in at least one
// table, and enough tables present, must pass. Guards against overcorrecting
// the zero-rows fix into a check that fails good restores too.
func TestSmokeCheck_RealDataPasses(t *testing.T) {
	sub := RestoreDrillResult{
		Success:        true,
		TablesVerified: 107,
		RowsVerified:   4213,
	}
	if err := smokeCheck(sub, DefaultCriticalTables); err != nil {
		t.Errorf("smokeCheck: want nil for a real restore with rows observed, got %v", err)
	}
}

// TestSmokeCheck_TooFewTablesFails preserves the pre-existing table-count
// floor: fewer than len(criticalTables) tables found still fails, even with a
// nonzero row count (e.g. one giant table).
func TestSmokeCheck_TooFewTablesFails(t *testing.T) {
	sub := RestoreDrillResult{
		Success:        true,
		TablesVerified: len(DefaultCriticalTables) - 1,
		RowsVerified:   999,
	}
	if err := smokeCheck(sub, DefaultCriticalTables); err == nil {
		t.Fatalf("smokeCheck: want error for too few tables verified, got nil")
	}
}
