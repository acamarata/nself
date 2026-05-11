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

// TestCriticalTables_Stable: changing the canonical critical-table list is a
// breaking change to the drill smoke suite. This test pins the list so any
// modification is intentional + reviewed.
func TestCriticalTables_Stable(t *testing.T) {
	want := []string{
		"np_users",
		"np_licenses",
		"np_audit_log",
		"np_plugins",
		"np_billing",
	}
	if len(CriticalTables) != len(want) {
		t.Fatalf("CriticalTables length changed: want %d, got %d", len(want), len(CriticalTables))
	}
	for i, name := range want {
		if CriticalTables[i] != name {
			t.Errorf("CriticalTables[%d]: want %q, got %q", i, name, CriticalTables[i])
		}
	}
}
