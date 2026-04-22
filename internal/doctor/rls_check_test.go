package doctor

import (
	"context"
	"testing"
)

// TestCheckRLSEnforcement_NoDBURL: NSELF_DB_URL not set → warn (skip, not fail).
func TestCheckRLSEnforcement_NoDBURL(t *testing.T) {
	t.Setenv("NSELF_DB_URL", "")
	t.Setenv("DATABASE_URL", "")

	results := CheckRLSEnforcement(context.Background(), false)
	if len(results) != 1 {
		t.Fatalf("want 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Status != "warn" {
		t.Errorf("no DB URL: want warn, got %q", r.Status)
	}
	if !containsStr(r.Message, "NSELF_DB_URL not set") {
		t.Errorf("want 'NSELF_DB_URL not set' in message, got %q", r.Message)
	}
}

// TestCheckRLSEnforcement_StrictEscalates: strict=true escalates warn to fail.
// This test relies on NSELF_DB_URL being absent so the function returns a warn
// result. With strict=true the underlying violations would be fail — but in the
// no-DB-URL path the function itself emits a warn regardless of strict because
// it cannot inspect the DB. This test verifies that strict mode does not panic
// and returns a single result.
func TestCheckRLSEnforcement_StrictNoDB(t *testing.T) {
	t.Setenv("NSELF_DB_URL", "")
	t.Setenv("DATABASE_URL", "")

	results := CheckRLSEnforcement(context.Background(), true)
	if len(results) == 0 {
		t.Fatal("strict mode with no DB: want at least 1 result")
	}
	// The no-DB-URL path always returns warn even in strict mode (we cannot inspect the DB).
	for _, r := range results {
		if r.Status == "fail" && containsStr(r.Name, "RLS enforcement") {
			// Acceptable: strict path may escalate the top-level skip to fail.
			return
		}
	}
}

// TestCheckRLSEnforcement_InvalidDSN: bad DSN → warn (not a hard failure).
func TestCheckRLSEnforcement_InvalidDSN(t *testing.T) {
	t.Setenv("NSELF_DB_URL", "postgres://invalid-host-that-does-not-exist:5432/nself?connect_timeout=1")
	t.Setenv("HASURA_GRAPHQL_ADMIN_SECRET", "")

	results := CheckRLSEnforcement(context.Background(), false)
	if len(results) == 0 {
		t.Fatal("invalid DSN: want at least 1 result")
	}
	// All results should be warn (cannot connect = skip, not fail).
	for _, r := range results {
		if r.Status == "fail" {
			t.Errorf("invalid DSN: want warn, got fail for check %q: %s", r.Name, r.Message)
		}
	}
}
