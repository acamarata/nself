// S9.T03 — unit tests for OPS-DRILL-01.
package doctor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/database"
)

// writeEntry appends a DrillResult JSON line to the drill log under projectDir.
func writeEntry(t *testing.T, projectDir string, completedAt time.Time, success bool) {
	t.Helper()
	dir := filepath.Join(projectDir, ".nself")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir .nself: %v", err)
	}
	entry := database.DrillResult{
		StartedAt:     completedAt.Add(-time.Minute),
		CompletedAt:   completedAt,
		Success:       success,
		BackupFile:    "test.dump",
		TotalDuration: time.Minute,
		RTOTargetMet:  true,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	f, err := os.OpenFile(filepath.Join(dir, "drill-log.json"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(append(data, '\n')); err != nil {
		t.Fatalf("write log: %v", err)
	}
}

// TestCheckOPSDrill_NoLog: missing drill log → fail status.
func TestCheckOPSDrill_NoLog(t *testing.T) {
	dir := t.TempDir()
	res := CheckOPSDrill(context.Background(), dir)
	if res.Status != "fail" {
		t.Errorf("missing log: want fail, got %q", res.Status)
	}
	if !strings.Contains(res.Message, "no successful backup drill recorded") {
		t.Errorf("message missing expected phrase: %q", res.Message)
	}
	if res.FixCmd != "nself backup drill" {
		t.Errorf("fix-cmd: want 'nself backup drill', got %q", res.FixCmd)
	}
}

// TestCheckOPSDrill_Recent: drill within 7d → pass.
func TestCheckOPSDrill_Recent(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, time.Now().Add(-3*24*time.Hour), true)

	res := CheckOPSDrill(context.Background(), dir)
	if res.Status != "pass" {
		t.Errorf("recent drill: want pass, got %q (msg: %s)", res.Status, res.Message)
	}
}

// TestCheckOPSDrill_Stale: drill 10 days ago → warn.
func TestCheckOPSDrill_Stale(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, time.Now().Add(-10*24*time.Hour), true)

	res := CheckOPSDrill(context.Background(), dir)
	if res.Status != "warn" {
		t.Errorf("stale drill: want warn, got %q (msg: %s)", res.Status, res.Message)
	}
	if res.FixCmd == "" {
		t.Errorf("warn status should suggest fix command")
	}
}

// TestCheckOPSDrill_VeryStale: drill > 14 days → fail.
func TestCheckOPSDrill_VeryStale(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, time.Now().Add(-30*24*time.Hour), true)

	res := CheckOPSDrill(context.Background(), dir)
	if res.Status != "fail" {
		t.Errorf("very stale drill: want fail, got %q (msg: %s)", res.Status, res.Message)
	}
}

// TestCheckOPSDrill_OnlyFailureEntries: log has only failed drills → fail.
func TestCheckOPSDrill_OnlyFailureEntries(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, time.Now().Add(-1*time.Hour), false) // recent failure

	res := CheckOPSDrill(context.Background(), dir)
	if res.Status != "fail" {
		t.Errorf("only failures: want fail, got %q", res.Status)
	}
}
