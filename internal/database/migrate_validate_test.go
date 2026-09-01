package database

import (
	"strings"
	"testing"
)

// Purpose: unit-level coverage for wrapDryRunSQL's transaction envelope.
// validateMigrationSQL itself shells out to psql via pipeSQLToContainer and
// requires a live Postgres target (container or embedded-PG) to exercise
// end-to-end — not available in a sandboxed unit-test run — so this file
// covers the pure, independently-testable half: the exact envelope that
// makes the dry run safe (never commits) and faithful (runs the identical
// statements the real apply would).

func TestWrapDryRunSQL_WrapsBeginRollback(t *testing.T) {
	sql := "CREATE TABLE widgets (id serial primary key);"
	got := wrapDryRunSQL(sql)

	if !strings.HasPrefix(got, "BEGIN;\n") {
		t.Fatalf("wrapped SQL does not start with BEGIN;\\n: %q", got)
	}
	if !strings.HasSuffix(got, "\nROLLBACK;\n") {
		t.Fatalf("wrapped SQL does not end with \\nROLLBACK;\\n: %q", got)
	}
	if !strings.Contains(got, sql) {
		t.Fatalf("wrapped SQL does not contain the original statements verbatim: %q", got)
	}
}

func TestWrapDryRunSQL_PreservesInvalidConstructVerbatim(t *testing.T) {
	// The exact invalid construct named in
	// msg-2026-07-02-nself-migration-ledger-pk-bug.md's secondary finding:
	// Postgres does not support IF NOT EXISTS on ADD CONSTRAINT. This test
	// only proves the dry-run envelope passes the construct through
	// unmodified to whatever runs it (validateMigrationSQL -> psql); it does
	// not itself invoke Postgres.
	sql := "ALTER TABLE t ADD CONSTRAINT IF NOT EXISTS ck_t_name CHECK (name <> '');"
	got := wrapDryRunSQL(sql)

	if !strings.Contains(got, "ADD CONSTRAINT IF NOT EXISTS") {
		t.Fatalf("wrapped SQL lost the invalid construct under test: %q", got)
	}
}

func TestWrapDryRunSQL_NeverAppendsCommit(t *testing.T) {
	// Regression guard: a dry-run envelope that ends in COMMIT instead of
	// ROLLBACK would defeat the entire point of this validator (it would
	// apply the migration for real during "validation").
	got := wrapDryRunSQL("SELECT 1;")
	if strings.Contains(got, "COMMIT") {
		t.Fatalf("dry-run envelope must never COMMIT, got: %q", got)
	}
}
