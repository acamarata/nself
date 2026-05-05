package database

import (
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
