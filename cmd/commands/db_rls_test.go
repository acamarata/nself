package commands

// db_rls_test.go — Unit tests for the db rls command.
// P4-E5-W3-S06-T20: security command coverage gate (was 0% — now covers happy + error paths).
//
// Purpose: Verify command registration, flag parsing, and identifier validation.
// Inputs:  cobra command execution, identifier strings.
// Outputs: error on invalid input; nil on valid input.
// Constraints: Does not require a live DB; tests focus on CLI plumbing + validation.

import (
	"testing"
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
