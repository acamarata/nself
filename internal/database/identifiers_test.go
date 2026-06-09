package database

import (
	"strings"
	"testing"
)

func TestSanitizeIdentifier_valid(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"public", `"public"`},
		{"auth", `"auth"`},
		{"storage", `"storage"`},
		{"pgcrypto", `"pgcrypto"`},
		{"citext", `"citext"`},
		{"nself", `"nself"`},
		{"_private", `"_private"`},
		{"col_123", `"col_123"`},
		{"A", `"A"`},
	}
	for _, tc := range cases {
		got, err := SanitizeIdentifier(tc.input)
		if err != nil {
			t.Errorf("SanitizeIdentifier(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("SanitizeIdentifier(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestSanitizeIdentifier_invalid(t *testing.T) {
	cases := []string{
		"",
		"123starts_with_digit",
		"has space",
		"has-hyphen",
		"has.dot",
		"'; DROP TABLE np_users; --",
		`"quoted"`,
		"has\nnewline",
	}
	for _, s := range cases {
		got, err := SanitizeIdentifier(s)
		if err == nil {
			t.Errorf("SanitizeIdentifier(%q): expected error, got %q", s, got)
		}
	}
}

// TestSanitizeIdentifier_lengthBoundary verifies that identifiers at and
// beyond the PostgreSQL 63-byte NAMEDATALEN-1 limit are handled correctly.
// Boundary cases: 63-char (valid), 64-char (rejected), 65-char (rejected).
// This is an ADR-009 SQL-safety requirement (T01 — sql-allowlist-audit-cli).
func TestSanitizeIdentifier_lengthBoundary(t *testing.T) {
	// Exactly 63 characters — valid (PostgreSQL NAMEDATALEN-1 limit).
	exactly63 := strings.Repeat("a", 63)
	if _, err := SanitizeIdentifier(exactly63); err != nil {
		t.Errorf("SanitizeIdentifier(63-char): unexpected error: %v", err)
	}

	// 64 characters — must be rejected.
	over64 := strings.Repeat("a", 64)
	if _, err := SanitizeIdentifier(over64); err == nil {
		t.Errorf("SanitizeIdentifier(64-char): expected error for oversized identifier, got nil")
	}

	// 65 characters — must be rejected.
	over65 := strings.Repeat("a", 65)
	if _, err := SanitizeIdentifier(over65); err == nil {
		t.Errorf("SanitizeIdentifier(65-char): expected error for oversized identifier, got nil")
	}
}

// TestSanitizeIdentifier_nopanicOnBadInput verifies that calling SanitizeIdentifier
// with adversarial input returns an error and never panics (GAP-D11-01 regression
// guard: MustSanitizeIdentifier was removed; callers must handle error returns).
func TestSanitizeIdentifier_nopanicOnBadInput(t *testing.T) {
	badInputs := []string{
		"'; DROP TABLE np_users; --",
		"has space",
		"\x00null\x00byte",
		"tab\there",
	}
	for _, s := range badInputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("SanitizeIdentifier(%q) panicked: %v", s, r)
				}
			}()
			_, err := SanitizeIdentifier(s)
			if err == nil {
				t.Errorf("SanitizeIdentifier(%q): expected error for bad input, got nil", s)
			}
		}()
	}
}
