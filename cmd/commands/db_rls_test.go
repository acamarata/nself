package commands

// db_rls_test.go — unit tests for db rls helper functions.
//
// Purpose: cover the identifier validation and glob matching helpers
// that are used to prevent SQL injection in apply-table arguments.
// These tests run entirely offline — no database, no network, no Docker.

import (
	"testing"
)

// TestValidateIdentifier_HappyPath verifies that valid SQL identifiers are accepted.
func TestValidateIdentifier_HappyPath(t *testing.T) {
	valid := []string{
		"np_claw_messages",
		"public",
		"users",
		"my_table_123",
		"A",
		"Table1",
	}
	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			if err := validateIdentifier(name); err != nil {
				t.Errorf("validateIdentifier(%q) returned unexpected error: %v", name, err)
			}
		})
	}
}

// TestValidateIdentifier_ErrorPaths verifies that empty names and names containing
// characters not allowed in PostgreSQL unquoted identifiers are rejected.
// This guards apply-table against SQL injection via schema/table name arguments.
func TestValidateIdentifier_ErrorPaths(t *testing.T) {
	invalid := []string{
		"",
		"my-table",        // hyphen
		"table name",      // space
		"np_claw;drop",    // semicolon injection
		"np_claw\ndrop",   // newline injection
		"schema.table",    // dot is not valid in an unquoted identifier component
		"'quoted'",        // single quote
		`"dquote"`,        // double quote
	}
	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			if err := validateIdentifier(name); err == nil {
				t.Errorf("validateIdentifier(%q) returned nil; expected error for invalid identifier", name)
			}
		})
	}
}

// TestMatchesGlob_HappyPath verifies that glob matching works for the supported
// prefix*, *suffix, and exact patterns.
func TestMatchesGlob_HappyPath(t *testing.T) {
	cases := []struct {
		pattern string
		name    string
		want    bool
	}{
		{pattern: "*", name: "anything", want: true},
		{pattern: "np_claw_*", name: "np_claw_messages", want: true},
		{pattern: "np_claw_*", name: "np_claw_rooms", want: true},
		{pattern: "*_messages", name: "np_claw_messages", want: true},
		{pattern: "exact", name: "exact", want: true},
		// Non-matching cases.
		{pattern: "np_claw_*", name: "np_chat_messages", want: false},
		{pattern: "*_rooms", name: "np_claw_messages", want: false},
		{pattern: "exact", name: "different", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.pattern+"/"+tc.name, func(t *testing.T) {
			got := matchesGlob(tc.pattern, tc.name)
			if got != tc.want {
				t.Errorf("matchesGlob(%q, %q) = %v; want %v", tc.pattern, tc.name, got, tc.want)
			}
		})
	}
}

// TestDbRLSCmd_SubcommandsRegistered verifies that the four db rls subcommands
// are registered so that help output is not misleading.
func TestDbRLSCmd_SubcommandsRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, sub := range dbRLSCmd.Commands() {
		names[sub.Name()] = true
	}
	required := []string{"audit", "apply", "apply-table", "rollback"}
	for _, req := range required {
		if !names[req] {
			t.Errorf("db rls subcommand %q not registered", req)
		}
	}
}
