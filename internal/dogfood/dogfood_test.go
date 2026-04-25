package dogfood

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSaveAndGetReport_RoundTrip verifies that a saved report can be retrieved
// as the latest report.
func TestSaveAndGetReport_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	report := &AuditReport{
		ID:        "dogfood-test-123",
		RanAt:     time.Now().UTC().Truncate(time.Second),
		PassCount: 18,
		FailCount: 2,
		WarnCount: 1,
		SkipCount: 0,
		Checks: []AuditCheck{
			{Name: "backup-recency", Status: "pass", Message: "backup < 24h", Section: "backup"},
			{Name: "license-valid", Status: "fail", Message: "license expired", Section: "license"},
		},
	}

	if err := SaveReport(dir, report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	got, err := GetLatestReport(dir)
	if err != nil {
		t.Fatalf("GetLatestReport: %v", err)
	}

	if got.ID != report.ID {
		t.Errorf("ID: got %q, want %q", got.ID, report.ID)
	}
	if got.PassCount != report.PassCount {
		t.Errorf("PassCount: got %d, want %d", got.PassCount, report.PassCount)
	}
	if len(got.Checks) != len(report.Checks) {
		t.Errorf("Checks len: got %d, want %d", len(got.Checks), len(report.Checks))
	}
}

// TestGetLatestReport_NoReports verifies that GetLatestReport returns an error
// when no reports exist.
func TestGetLatestReport_NoReports(t *testing.T) {
	dir := t.TempDir()
	// Create the dogfood dir but no reports.
	dfDir := filepath.Join(dir, ".nself", "dogfood")
	if err := os.MkdirAll(dfDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	_, err := GetLatestReport(dir)
	if err == nil {
		t.Error("GetLatestReport should return error when no reports exist")
	}
}

// TestSaveReport_WritesValidJSON verifies the report file contains valid JSON.
func TestSaveReport_WritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	report := &AuditReport{
		ID:        "dogfood-json-test",
		RanAt:     time.Now().UTC(),
		PassCount: 21,
		Checks:    []AuditCheck{},
	}

	if err := SaveReport(dir, report); err != nil {
		t.Fatalf("SaveReport: %v", err)
	}

	// Find the file.
	dfDir := filepath.Join(dir, ".nself", "dogfood")
	entries, err := os.ReadDir(dfDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no report file created")
	}

	data, err := os.ReadFile(filepath.Join(dfDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("report file is not valid JSON: %v", err)
	}
}

// TestAuditCheck_StatusValues verifies that documented status values are
// distinguishable strings.
func TestAuditCheck_StatusValues(t *testing.T) {
	statuses := []string{"pass", "fail", "warn", "skip"}
	seen := make(map[string]bool)
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate status value: %q", s)
		}
		seen[s] = true
	}
}
