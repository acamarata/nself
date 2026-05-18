package tenant

import (
	"strings"
	"testing"
)

// FuzzSanitize verifies that sanitize() never allows a raw single-quote through.
// After sanitize(), the output must not contain an unescaped single quote —
// every ' must appear as ''.
func FuzzSanitize(f *testing.F) {
	// Seed with SQL injection payloads.
	f.Add("normal")
	f.Add("O'Brien")
	f.Add("'; DROP TABLE tenants; --")
	f.Add("' OR '1'='1")
	f.Add("\\'; SELECT sleep(1); --")
	f.Add("' UNION SELECT NULL--")
	f.Add("")
	f.Add("safe-slug_123")
	f.Add("100.100.100.200")

	f.Fuzz(func(t *testing.T, s string) {
		out := sanitize(s)
		// After sanitize, every single-quote must be doubled.
		// Count raw lone single quotes: look for ' not preceded or followed by another '.
		// The simplest invariant: replace all '' with "" (escaped pairs), then no ' remains.
		stripped := strings.ReplaceAll(out, "''", "")
		if strings.Contains(stripped, "'") {
			t.Errorf("sanitize(%q) = %q: contains unescaped single quote after removing pairs", s, out)
		}
	})
}

// FuzzValidateUUID verifies that validateUUID() only accepts canonical UUIDs.
func FuzzValidateUUID(f *testing.F) {
	f.Add("550e8400-e29b-41d4-a716-446655440000")
	f.Add("00000000-0000-0000-0000-000000000000")
	f.Add("'; DROP TABLE t; --")
	f.Add("")
	f.Add("not-a-uuid")
	f.Add("XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX")

	f.Fuzz(func(t *testing.T, s string) {
		err := validateUUID(s)
		if err != nil {
			// Rejection is correct for invalid input; make sure we don't panic.
			return
		}
		// If accepted, verify it matches the UUID pattern exactly.
		if !uuidRegex.MatchString(s) {
			t.Errorf("validateUUID(%q) returned nil but string does not match uuidRegex", s)
		}
	})
}

// FuzzValidateSlug verifies that validateSlug() only accepts safe slugs.
func FuzzValidateSlug(f *testing.F) {
	f.Add("my-tenant")
	f.Add("tenant_123")
	f.Add("'; DROP TABLE t; --")
	f.Add("")
	f.Add("a b c")
	f.Add("../../../etc/passwd")

	f.Fuzz(func(t *testing.T, s string) {
		err := validateSlug(s)
		if err != nil {
			return
		}
		if !slugRegex.MatchString(s) {
			t.Errorf("validateSlug(%q) returned nil but string does not match slugRegex", s)
		}
		// Accepted slug must not contain any SQL-dangerous characters.
		dangerous := []string{"'", "\"", ";", "--", "/*", "*/", "\\", "\x00"}
		for _, d := range dangerous {
			if strings.Contains(s, d) {
				t.Errorf("validateSlug(%q) accepted a string containing dangerous character %q", s, d)
			}
		}
	})
}

// FuzzValidateDate verifies that validateDate() only accepts YYYY-MM-DD.
func FuzzValidateDate(f *testing.F) {
	f.Add("2024-01-15")
	f.Add("'; DROP TABLE t; --")
	f.Add("2024-1-1")
	f.Add("")
	f.Add("20240115")

	f.Fuzz(func(t *testing.T, s string) {
		err := validateDate(s)
		if err != nil {
			return
		}
		if !dateRegex.MatchString(s) {
			t.Errorf("validateDate(%q) returned nil but string does not match dateRegex", s)
		}
	})
}

// FuzzValidateReason verifies that validateReason() rejects dangerous characters.
func FuzzValidateReason(f *testing.F) {
	f.Add("payment failed")
	f.Add("'; DROP TABLE t; --")
	f.Add("suspended by admin")
	f.Add("")
	f.Add("reason with 'quotes'")

	f.Fuzz(func(t *testing.T, s string) {
		err := validateReason(s)
		if err != nil {
			return
		}
		if !reasonRegex.MatchString(s) {
			t.Errorf("validateReason(%q) returned nil but string does not match reasonRegex", s)
		}
		// Accepted reason must not contain SQL injection vectors.
		dangerous := []string{"'", "\"", "\\", "\x00", ";"}
		for _, d := range dangerous {
			if strings.Contains(s, d) {
				t.Errorf("validateReason(%q) accepted string containing dangerous character %q", s, d)
			}
		}
	})
}

// TestValidateUUID_known verifies exact known cases.
func TestValidateUUID_known(t *testing.T) {
	t.Parallel()
	valid := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"00000000-0000-0000-0000-000000000000",
		"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
	}
	for _, v := range valid {
		if err := validateUUID(v); err != nil {
			t.Errorf("expected valid UUID %q to pass, got: %v", v, err)
		}
	}
	invalid := []string{
		"",
		"not-a-uuid",
		"550e8400-e29b-41d4-a716-44665544000",  // too short
		"550e8400-e29b-41d4-a716-4466554400000", // too long
		"'; DROP TABLE t; --",
		"550e8400 e29b 41d4 a716 446655440000", // spaces not hyphens
	}
	for _, v := range invalid {
		if err := validateUUID(v); err == nil {
			t.Errorf("expected invalid UUID %q to fail, got nil", v)
		}
	}
}

// TestValidateSlug_known verifies exact known cases.
func TestValidateSlug_known(t *testing.T) {
	t.Parallel()
	valid := []string{"my-tenant", "tenant_123", "a", "ALLCAPS"}
	for _, v := range valid {
		if err := validateSlug(v); err != nil {
			t.Errorf("expected valid slug %q to pass, got: %v", v, err)
		}
	}
	invalid := []string{
		"",
		"has space",
		"has'quote",
		"semi;colon",
		strings.Repeat("a", 65), // too long
		"../../../etc",
	}
	for _, v := range invalid {
		if err := validateSlug(v); err == nil {
			t.Errorf("expected invalid slug %q to fail, got nil", v)
		}
	}
}
