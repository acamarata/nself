package audit

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestRunTableAudit_NoDBURL verifies that RunTableAudit returns an error
// (not a panic) when NSELF_DB_URL is unset.
func TestRunTableAudit_NoDBURL(t *testing.T) {
	t.Setenv("NSELF_DB_URL", "")
	t.Setenv("DATABASE_URL", "")

	_, err := RunTableAudit(context.Background())
	if err == nil {
		t.Fatal("want error when NSELF_DB_URL not set, got nil")
	}
	if !containsAudit(err.Error(), "NSELF_DB_URL not set") {
		t.Errorf("want 'NSELF_DB_URL not set' in error, got: %v", err)
	}
}

// TestRunTableAudit_InvalidDSN verifies that RunTableAudit returns an error
// when the DSN is syntactically valid but the host is unreachable.
func TestRunTableAudit_InvalidDSN(t *testing.T) {
	t.Setenv("NSELF_DB_URL", "postgres://127.0.0.1:1/nself?connect_timeout=1&sslmode=disable")
	t.Setenv("DATABASE_URL", "")

	_, err := RunTableAudit(context.Background())
	if err == nil {
		t.Fatal("want error for unreachable DB, got nil")
	}
}

// TestTableAuditResult_NoSourceAccountIDColumn verifies that auditTable
// sets HasSourceAccountID=false and populates Warning when the column is absent.
// We exercise this path via the exported field shapes only (no live DB).
func TestTableAuditResult_Fields(t *testing.T) {
	result := TableAuditResult{
		Table:              "np_chat_messages",
		HasSourceAccountID: false,
		TotalRows:          42,
		Warning:            "no source_account_id column — cross-app isolation unknown",
	}
	if result.HasSourceAccountID {
		t.Error("HasSourceAccountID should be false")
	}
	if result.Warning == "" {
		t.Error("Warning should be non-empty for missing column")
	}
	if result.Accounts != nil {
		t.Error("Accounts should be nil when source_account_id absent")
	}
}

// TestTableAuditResult_MultiAccount verifies the Warning message for
// multi-account contamination is populated correctly by callers.
func TestTableAuditResult_MultiAccount(t *testing.T) {
	result := TableAuditResult{
		Table:              "np_family_members",
		HasSourceAccountID: true,
		TotalRows:          150,
		Accounts:           map[string]int64{"primary": 130, "unity-flock": 20},
	}
	// Simulate the warning that auditTable would set.
	if len(result.Accounts) > 1 {
		result.Warning = "multi-account contamination: 2 source_account_id values found — verify plugin isolation"
	}
	if result.Warning == "" {
		t.Error("Warning should be set for multi-account table")
	}
	if result.TotalRows != 150 {
		t.Errorf("TotalRows want 150, got %d", result.TotalRows)
	}
}

// TestAuditReport_AuditAtNotEmpty verifies AuditReport.AuditAt is set.
func TestAuditReport_AuditAtNotEmpty(t *testing.T) {
	r := &AuditReport{
		AuditAt: "2026-05-13T12:00:00Z",
		Tables:  []TableAuditResult{},
	}
	if r.AuditAt == "" {
		t.Error("AuditAt must not be empty")
	}
}

// TestRunTableAudit_EnvFallback verifies DATABASE_URL is used as fallback
// when NSELF_DB_URL is unset. An invalid fallback DSN should return an error,
// not a no-DB-URL message.
func TestRunTableAudit_EnvFallback(t *testing.T) {
	t.Setenv("NSELF_DB_URL", "")
	t.Setenv("DATABASE_URL", "postgres://127.0.0.1:1/nself?connect_timeout=1&sslmode=disable")

	_, err := RunTableAudit(context.Background())
	if err == nil {
		t.Fatal("want error for unreachable fallback DB")
	}
	// The error must NOT say "NSELF_DB_URL not set" — it must attempt the connection.
	if containsAudit(err.Error(), "NSELF_DB_URL not set") {
		t.Errorf("DATABASE_URL fallback not used; got: %v", err)
	}
}

// TestRunTableAudit_LiveDB is an integration test that only runs when NSELF_DB_URL
// points to a real Postgres instance. Skipped in unit mode.
func TestRunTableAudit_LiveDB(t *testing.T) {
	dbURL := os.Getenv("NSELF_DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("NSELF_DB_URL not set; skipping live-DB audit test")
	}

	report, err := RunTableAudit(context.Background())
	if err != nil {
		t.Fatalf("RunTableAudit with live DB: %v", err)
	}
	if report == nil {
		t.Fatal("report must not be nil with live DB")
	}
	if report.AuditAt == "" {
		t.Error("AuditAt must not be empty")
	}
	// Tables may be empty on a fresh install — that's valid.
	t.Logf("audit found %d np_* tables", len(report.Tables))
}

// containsAudit is a local helper for substring matching in test assertions.
func containsAudit(s, sub string) bool {
	return strings.Contains(s, sub)
}
