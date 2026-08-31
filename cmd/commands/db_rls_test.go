package commands

// db_rls_test.go — Unit tests for the db rls command.
// P4-E5-W3-S06-T20: security command coverage gate (was 0% — now covers happy + error paths).
//
// Purpose: Verify command registration, flag parsing, and identifier validation.
// Inputs:  cobra command execution, identifier strings.
// Outputs: error on invalid input; nil on valid input.
// Constraints: Does not require a live DB; tests focus on CLI plumbing + validation.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nself-org/cli/internal/database"
)

// TestValidateIdentifier_Valid verifies that well-formed SQL identifiers pass.
func TestValidateIdentifier_Valid(t *testing.T) {
	t.Parallel()

	valid := []string{
		"public",
		"np_users",
		"my_table_123",
		"np_subscriptions",
		"schema1",
	}
	for _, id := range valid {
		id := id
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			if err := validateIdentifier(id); err != nil {
				t.Errorf("validateIdentifier(%q) unexpected error: %v", id, err)
			}
		})
	}
}

// TestValidateIdentifier_Invalid verifies that SQL injection patterns and
// invalid characters are rejected.
func TestValidateIdentifier_Invalid(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",           // empty
		"table;drop", // semicolon injection
		"has space",  // space
		"has-hyphen", // hyphen is not an ident char
		"has'quote",  // single quote
		"has\"quote", // double quote
	}
	for _, id := range invalid {
		id := id
		label := id
		if len(label) > 8 {
			label = label[:8]
		}
		t.Run("invalid:"+label, func(t *testing.T) {
			t.Parallel()
			if err := validateIdentifier(id); err == nil {
				t.Errorf("validateIdentifier(%q) expected error, got nil", id)
			}
		})
	}
}

// TestDBRLSCmd_Registration verifies that db rls commands are registered with
// the expected Use strings.
func TestDBRLSCmd_Registration(t *testing.T) {
	t.Parallel()

	if dbRLSCmd == nil {
		t.Fatal("dbRLSCmd is nil — command not registered")
	}
	if dbRLSCmd.Use != "rls" {
		t.Errorf("dbRLSCmd.Use = %q, want %q", dbRLSCmd.Use, "rls")
	}
	if dbRLSAuditCmd == nil {
		t.Fatal("dbRLSAuditCmd is nil — subcommand not registered")
	}
}

// TestDBRLSAuditCmd_HasRunE verifies the audit command uses RunE (not Run),
// per cobra conventions enforced by cli rules.
func TestDBRLSAuditCmd_HasRunE(t *testing.T) {
	t.Parallel()

	if dbRLSAuditCmd == nil {
		t.Fatal("dbRLSAuditCmd is nil — command not registered")
	}
	if dbRLSAuditCmd.RunE == nil {
		t.Error("dbRLSAuditCmd.RunE is nil — command must use RunE, not Run")
	}
}

// ── P6-E11-W2-S3-T18: security command test floor ──────────────────────
//
// The tests above prove validateIdentifier() itself rejects injection
// characters, but not that the apply-table/rollback COMMANDS actually call
// it before touching the database. A regression that removed the
// validateIdentifier() call from runDBRLSApplyTable/runDBRLSRollback (e.g.
// during a refactor) would pass every test above while reopening the exact
// injection vector this file exists to close. These tests exercise the
// real RunE path with a malicious argument and require no live database,
// because validation happens before database.EnableRLSOnTable/
// GenerateRollbackSQL are ever reached.

// TestRunDBRLSApplyTable_RejectsInjectionBeforeDBCall verifies the apply-table
// command path rejects a malicious schema/table argument without needing a
// database connection — proving validation runs before any DB call.
func TestRunDBRLSApplyTable_RejectsInjectionBeforeDBCall(t *testing.T) {
	t.Parallel()

	cases := [][2]string{
		{"public; DROP TABLE users;--", "claw_messages"},
		{"np_claw", "claw_messages; DROP TABLE users;--"},
	}
	for _, c := range cases {
		schema, table := c[0], c[1]
		t.Run(schema+"/"+table, func(t *testing.T) {
			t.Parallel()
			err := runDBRLSApplyTable(dbRLSApplyTableCmd, []string{schema, table})
			if err == nil {
				t.Fatalf("expected rejection for injection attempt (%q, %q), got nil", schema, table)
			}
			if !strings.Contains(err.Error(), "invalid") {
				t.Errorf("error = %q, want it to say the identifier is invalid", err.Error())
			}
		})
	}
}

// TestRunDBRLSRollback_RejectsInjectionBeforeGeneratingSQL verifies the
// rollback command path rejects a malicious argument and never reaches
// database.GenerateRollbackSQL (which has no escaping of its own — it
// trusts its caller to have validated the identifiers already).
func TestRunDBRLSRollback_RejectsInjectionBeforeGeneratingSQL(t *testing.T) {
	// Not t.Parallel(): captureStdout (config_test.go) mutates os.Stdout.
	out, err := captureStdout(t, func() error {
		return runDBRLSRollback(dbRLSRollbackCmd, []string{"public; DROP TABLE users;--", "claw_messages"})
	})
	if err == nil {
		t.Fatal("expected rejection for injection attempt in rollback schema arg, got nil")
	}
	if out != "" {
		t.Errorf("rollback printed SQL despite rejecting the identifier: %q", out)
	}
}

// TestRunDBRLSRollback_ValidIdentifiers_PrintsRealRollbackSQL verifies the
// positive path: valid identifiers produce the actual rollback SQL for that
// exact schema.table, matching database.GenerateRollbackSQL directly — this
// is a pure function, so no DB is needed either way.
func TestRunDBRLSRollback_ValidIdentifiers_PrintsRealRollbackSQL(t *testing.T) {
	// Not t.Parallel(): captureStdout (config_test.go) mutates os.Stdout.
	want := database.GenerateRollbackSQL("np_claw", "claw_messages")

	out, err := captureStdout(t, func() error {
		return runDBRLSRollback(dbRLSRollbackCmd, []string{"np_claw", "claw_messages"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != want {
		t.Errorf("rollback output = %q, want %q", out, want)
	}
	if !strings.Contains(out, "np_claw") || !strings.Contains(out, "claw_messages") {
		t.Errorf("rollback SQL does not reference the requested table: %q", out)
	}
}

// TestRunDBRLSAudit_NoLiveDB_ErrorPropagates and
// TestRunDBRLSApply_NoLiveDB_ErrorPropagates exercise the real connection-
// failure path against whatever Postgres config.Load resolves in this test
// environment (none is running). This does not prove RLS auditing/applying
// works against a real database — that needs a live Postgres instance,
// which is out of this ticket's scope (see the completion note's explicit
// non-coverage list) — but it does prove a DB-layer error surfaces as a
// wrapped, descriptive command error rather than being swallowed or
// panicking, which is the one property reachable without live
// infrastructure. A 5s timeout keeps this from hanging if the environment's
// resolved host/port ever points somewhere that blackholes instead of
// refusing the connection.
func TestRunDBRLSAudit_NoLiveDB_ErrorPropagates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dbRLSAuditCmd.SetContext(ctx)
	_ = dbRLSAuditCmd.Flags().Set("format", "table")

	err := runDBRLSAudit(dbRLSAuditCmd, nil)
	if err == nil {
		t.Skip("a live DB answered in this environment — nothing to assert about the error path")
	}
	if !strings.Contains(err.Error(), "db rls audit") {
		t.Errorf("error = %q, want it wrapped with 'db rls audit' context", err.Error())
	}
}

func TestRunDBRLSApply_NoLiveDB_ErrorPropagates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dbRLSApplyCmd.SetContext(ctx)
	_ = dbRLSApplyCmd.Flags().Set("dry-run", "false")
	_ = dbRLSApplyCmd.Flags().Set("pattern", "")

	err := runDBRLSApply(dbRLSApplyCmd, nil)
	if err == nil {
		t.Skip("a live DB answered in this environment — nothing to assert about the error path")
	}
	if !strings.Contains(err.Error(), "db rls apply") {
		t.Errorf("error = %q, want it wrapped with 'db rls apply' context", err.Error())
	}
}

// TestMatchesGlob verifies the pattern-matching used by `db rls apply
// --pattern` to select which tables get RLS applied. An incorrect glob
// match here means RLS is silently skipped on tables the operator intended
// to include, or applied (with its side effects) to tables they meant to
// exclude — either way this affects security posture, not just cosmetics.
func TestMatchesGlob(t *testing.T) {
	t.Parallel()

	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"*", "anything", true},
		{"np_claw_messages", "np_claw_messages", true},
		{"np_claw_messages", "np_claw_other", false},
		{"np_claw_*", "np_claw_messages", true},
		{"np_claw_*", "np_task_messages", false},
		{"*_messages", "np_claw_messages", true},
		{"*_messages", "np_claw_users", false},
		{"np_*_messages", "np_claw_messages", true}, // only first '*' honored per doc comment
	}
	for _, c := range cases {
		c := c
		t.Run(c.pattern+"/"+c.name, func(t *testing.T) {
			t.Parallel()
			got := matchesGlob(c.pattern, c.name)
			if got != c.want {
				t.Errorf("matchesGlob(%q, %q) = %v, want %v", c.pattern, c.name, got, c.want)
			}
		})
	}
}
