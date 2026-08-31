// Package tenant — sanitize_test.go covers S192:
// Table-driven adversarial inputs for all helper validators, fuzz targets,
// and 100% branch coverage push for helpers.go.
package tenant

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// --- S192-T01: Table-driven adversarial tests ---

// TestSanitize_TableDriven exercises SQL metachar escaping in sanitize().
func TestSanitize_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"no_quotes", "hello world", "hello world"},
		{"single_quote", "it's", "it''s"},
		{"double_single_quote", "it''s", "it''''s"},
		{"semicolon", "foo; DROP TABLE", "foo; DROP TABLE"}, // semicolons not escaped (belt-and-suspenders)
		{"sql_injection_classic", "' OR '1'='1", "'' OR ''1''=''1"},
		{"null_byte_passthrough", "foo\x00bar", "foo\x00bar"}, // sanitize doesn't strip nulls
		{"unicode_quotes", "’s", "’s"},                        // curly apostrophe — no escaping needed
		{"only_quotes", "''''", "''''''''"},
		{"long_value", strings.Repeat("a", 256), strings.Repeat("a", 256)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitize(tc.input)
			if got != tc.want {
				t.Errorf("sanitize(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestQuoteIdent_TableDriven exercises SQL identifier quoting in quoteIdent().
func TestQuoteIdent_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "tenant_id", `"tenant_id"`},
		{"uppercase", "TenantID", `"TenantID"`},
		{"with_space", "my schema", `"my schema"`},
		{"embedded_double_quote", `my"table`, `"my""table"`},
		{"two_double_quotes", `a""b`, `"a""""b"`},
		{"sql_injection", `"; DROP TABLE --`, `"""; DROP TABLE --"`},
		{"empty", "", `""`},
		{"dot_notation", "public.tenants", `"public.tenants"`},
		{"unicode", "schéma", `"schéma"`},
		{"backslash", `path\name`, `"path\name"`}, // backslashes pass through
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := quoteIdent(tc.input)
			if got != tc.want {
				t.Errorf("quoteIdent(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestTrimOutput_TableDriven exercises output trimming.
func TestTrimOutput_TableDriven(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{"empty", []byte{}, ""},
		{"plain", []byte("hello"), "hello"},
		{"leading_whitespace", []byte("  hello"), "hello"},
		{"trailing_newline", []byte("hello\n"), "hello"},
		{"both_sides", []byte("\n  result  \n"), "result"},
		{"tabs", []byte("\t\toutput\t"), "output"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := trimOutput(tc.input)
			if got != tc.want {
				t.Errorf("trimOutput(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestValidateUUID_TableDriven exercises UUID validation.
func TestValidateUUID_TableDriven(t *testing.T) {
	valid := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"00000000-0000-0000-0000-000000000000",
		"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
		"a1b2c3d4-e5f6-A1B2-C3D4-E5F6A1B2C3D4",
	}
	invalid := []string{
		"",
		"not-a-uuid",
		"550e8400-e29b-41d4-a716-44665544000",   // too short
		"550e8400-e29b-41d4-a716-4466554400001", // too long
		"550e8400_e29b_41d4_a716_446655440000",  // underscores not dashes
		"gg000000-0000-0000-0000-000000000000",  // non-hex char
		"' OR '1'='1' DROP TABLE tenants --",    // SQL injection attempt (no semicolon to avoid Go string parse issue)
		strings.Repeat("a", 36),                 // looks like UUID length but wrong format
	}
	for _, v := range valid {
		if err := validateUUID(v); err != nil {
			t.Errorf("validateUUID(%q) returned error for valid UUID: %v", v, err)
		}
	}
	for _, v := range invalid {
		if err := validateUUID(v); err == nil {
			t.Errorf("validateUUID(%q) returned nil for invalid UUID", v)
		}
	}
}

// TestValidateSlug_TableDriven exercises slug validation.
func TestValidateSlug_TableDriven(t *testing.T) {
	valid := []string{
		"tenant1",
		"my-tenant",
		"my_tenant",
		"UPPERCASE",
		"mix3d-Case_123",
		strings.Repeat("a", 64), // max length
		"a",                     // min length
	}
	invalid := []string{
		"",
		strings.Repeat("a", 65),    // too long
		"has space",                // space not allowed
		"semi;colon",               // semicolon
		"quote'here",               // single quote
		"dquote\"here",             // double quote
		"null\x00byte",             // null byte
		"backslash\\path",          // backslash
		"sql'; DROP TABLE tenants", // SQL injection
		"tab\there",                // tab character
	}
	for _, v := range valid {
		if err := validateSlug(v); err != nil {
			t.Errorf("validateSlug(%q) returned error for valid slug: %v", v, err)
		}
	}
	for _, v := range invalid {
		if err := validateSlug(v); err == nil {
			t.Errorf("validateSlug(%q) returned nil for invalid slug", v)
		}
	}
}

// TestValidateDate_TableDriven exercises YYYY-MM-DD date validation.
func TestValidateDate_TableDriven(t *testing.T) {
	valid := []string{
		"2026-04-24",
		"2000-01-01",
		"9999-12-31",
	}
	invalid := []string{
		"",
		"2026-4-24",     // missing zero pad
		"2026/04/24",    // slashes
		"24-04-2026",    // wrong order
		"2026-04-24T00", // with time
		"2026-130-1",    // wrong digit count in month field
		"' OR '1'='1",   // SQL injection
		"2026-04",       // missing day
		"abcd-ef-gh",    // non-digits
	}
	for _, v := range valid {
		if err := validateDate(v); err != nil {
			t.Errorf("validateDate(%q) returned error for valid date: %v", v, err)
		}
	}
	for _, v := range invalid {
		if err := validateDate(v); err == nil {
			t.Errorf("validateDate(%q) returned nil for invalid date", v)
		}
	}
}

// TestValidateMonth_TableDriven exercises YYYY-MM month validation.
func TestValidateMonth_TableDriven(t *testing.T) {
	valid := []string{"2026-04", "2000-01", "9999-12"}
	invalid := []string{
		"", "2026-4", "2026/04", "04-2026",
		"2026-04-01", "abcd-ef", "' OR 1=1",
	}
	for _, v := range valid {
		if err := validateMonth(v); err != nil {
			t.Errorf("validateMonth(%q) returned error for valid month: %v", v, err)
		}
	}
	for _, v := range invalid {
		if err := validateMonth(v); err == nil {
			t.Errorf("validateMonth(%q) returned nil for invalid month", v)
		}
	}
}

// TestValidateEventID_TableDriven exercises event ID validation.
func TestValidateEventID_TableDriven(t *testing.T) {
	valid := []string{
		"evt_1234567890",
		"ch_3ABC",
		"pi_test123",
		"idm:evt.123-foo",
		strings.Repeat("a", 128), // max length
	}
	invalid := []string{
		"",
		strings.Repeat("a", 129), // too long
		"has space",
		"quote'here",
		"semi;colon",
		"newline\nhere",
		"null\x00byte",
		"sql'; DROP TABLE",
	}
	for _, v := range valid {
		if err := validateEventID(v); err != nil {
			t.Errorf("validateEventID(%q) returned error for valid ID: %v", v, err)
		}
	}
	for _, v := range invalid {
		if err := validateEventID(v); err == nil {
			t.Errorf("validateEventID(%q) returned nil for invalid ID", v)
		}
	}
}

// TestValidateReason_TableDriven exercises free-text reason validation.
func TestValidateReason_TableDriven(t *testing.T) {
	valid := []string{
		"",
		"Payment failed",
		"Cancelled by admin",
		"Test run #1 (ok)",
		"user@domain.com",
		"reason: 100 done",
		strings.Repeat("a", 256), // max length
	}
	invalid := []string{
		strings.Repeat("a", 257), // too long
		"quote'here",             // single quote not allowed
		"dquote\"here",           // double quote not allowed
		"backslash\\path",        // backslash not allowed
		"semi;colon",             // semicolon not allowed
		"null\x00byte",           // null byte
		"newline\nhere",          // newline
		"tab\there",              // tab
		"em—dash",                // em dash (outside ASCII safe set)
	}
	for _, v := range valid {
		if err := validateReason(v); err != nil {
			t.Errorf("validateReason(%q) returned error for valid reason: %v", v, err)
		}
	}
	for _, v := range invalid {
		if err := validateReason(v); err == nil {
			t.Errorf("validateReason(%q) returned nil for invalid reason", v)
		}
	}
}

// TestValidateDuration_TableDriven exercises duration string validation.
func TestValidateDuration_TableDriven(t *testing.T) {
	valid := []string{
		"1s", "60s", "24h", "7d", "1m", "90m",
		"999999d", // max 6 digits
	}
	invalid := []string{
		"",
		"h",        // no digits
		"1",        // no unit
		"1w",       // unsupported unit
		"1H",       // uppercase unit
		"1 h",      // space
		"-1h",      // negative
		"1000000d", // 7 digits — too long
		"1.5h",     // decimal
		"'; DROP",  // SQL injection
	}
	for _, v := range valid {
		if err := validateDuration(v); err != nil {
			t.Errorf("validateDuration(%q) returned error for valid duration: %v", v, err)
		}
	}
	for _, v := range invalid {
		if err := validateDuration(v); err == nil {
			t.Errorf("validateDuration(%q) returned nil for invalid duration", v)
		}
	}
}

// --- S192-T01: Unicode and null byte adversarial inputs ---

// TestSanitize_UnicodeAndNullBytes exercises boundary inputs.
func TestSanitize_UnicodeAndNullBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"null_byte", "foo\x00bar"},
		{"unicode_smp", "𝕳𝖊𝖑𝖑𝖔"}, // supplementary plane chars
		{"rtl_text", "مرحبا"},
		{"zero_width", "a\u200bb"},   // zero-width space
		{"bom", "\xef\xbb\xbfhello"}, // byte order mark (UTF-8 BOM)
		{"sql_metachar_mix", "'; UPDATE tenants SET plan='enterprise' WHERE '1'='1"},
		{"newline_in_value", "line1\nline2"},
		{"crlf", "val\r\nmore"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitize(tc.input)
			// Must not panic and must be valid UTF-8 if input was valid UTF-8.
			if utf8.ValidString(tc.input) && !utf8.ValidString(got) {
				t.Errorf("sanitize(%q) produced invalid UTF-8: %q", tc.input, got)
			}
			// Single quotes must be doubled.
			if strings.Count(tc.input, "'") > 0 {
				if strings.Count(got, "''") != strings.Count(tc.input, "'") {
					t.Errorf("sanitize(%q) single-quote count mismatch: got %q", tc.input, got)
				}
			}
		})
	}
}

// TestQuoteIdent_SQLInjectionAttempts verifies SQL injection via identifiers is neutralised.
func TestQuoteIdent_SQLInjectionAttempts(t *testing.T) {
	attacks := []string{
		`"; DROP TABLE tenants; --`,
		`" OR 1=1 --`,
		`public"."malicious`,
		"table\x00hidden",
	}
	for _, attack := range attacks {
		quoted := quoteIdent(attack)
		// Must be surrounded by double quotes.
		if !strings.HasPrefix(quoted, `"`) || !strings.HasSuffix(quoted, `"`) {
			t.Errorf("quoteIdent(%q) = %q: must be double-quoted", attack, quoted)
		}
		// Must not contain an unescaped double quote inside the identifier body.
		inner := quoted[1 : len(quoted)-1]
		for i := 0; i < len(inner); i++ {
			if inner[i] == '"' {
				if i+1 >= len(inner) || inner[i+1] != '"' {
					t.Errorf("quoteIdent(%q): unescaped double quote at position %d in %q", attack, i, inner)
				} else {
					i++ // skip the escaped pair
				}
			}
		}
	}
}

// --- S192-T02: Fuzz targets ---

// FuzzSanitizeTenantID fuzzes the sanitize function with arbitrary input.
// Invariants:
//  1. Must not panic.
//  2. Result must be valid UTF-8 when input is valid UTF-8.
//  3. Single quotes in input must be doubled in output.
//  4. Result length must be >= input length (only additions, no removals).
func FuzzSanitizeTenantID(f *testing.F) {
	// Seed corpus.
	f.Add("")
	f.Add("normal-slug")
	f.Add("it's a test")
	f.Add("'; DROP TABLE tenants; --")
	f.Add("'' double '' quoted ''")
	f.Add("\x00null")
	f.Add("𝕳ello")
	f.Add(strings.Repeat("'", 100))
	f.Add("mix ' and \" and ; and \\")

	f.Fuzz(func(t *testing.T, input string) {
		got := sanitize(input)
		// Must not panic (already guaranteed by reaching here).
		// If input is valid UTF-8, output must be valid UTF-8.
		if utf8.ValidString(input) && !utf8.ValidString(got) {
			t.Errorf("sanitize produced invalid UTF-8 from valid input %q: got %q", input, got)
		}
		// Every ' in input must become '' in output.
		inputQuotes := strings.Count(input, "'")
		outputDoubles := strings.Count(got, "''")
		if inputQuotes > 0 && outputDoubles < inputQuotes {
			t.Errorf("sanitize(%q) = %q: expected at least %d '', got %d",
				input, got, inputQuotes, outputDoubles)
		}
		// Output must be at least as long as input.
		if len(got) < len(input) {
			t.Errorf("sanitize(%q) = %q: output shorter than input", input, got)
		}
	})
}

// FuzzMergeTenantEnv fuzzes the quoteIdent function with arbitrary identifiers.
// Invariants:
//  1. Must not panic.
//  2. Result must always be surrounded by double quotes.
//  3. No unescaped double quotes inside the identifier body.
func FuzzMergeTenantEnv(f *testing.F) {
	// Seed corpus.
	f.Add("")
	f.Add("tenants")
	f.Add(`table"name`)
	f.Add(`"already"quoted"`)
	f.Add("'; DROP TABLE --")
	f.Add("\x00null")
	f.Add(strings.Repeat(`"`, 50))
	f.Add("normal_identifier_123")
	f.Add("has space and \"quotes\"")

	f.Fuzz(func(t *testing.T, input string) {
		got := quoteIdent(input)
		// Must be surrounded by double quotes.
		if len(got) < 2 {
			t.Errorf("quoteIdent(%q) = %q: too short to have surrounding quotes", input, got)
			return
		}
		if got[0] != '"' || got[len(got)-1] != '"' {
			t.Errorf("quoteIdent(%q) = %q: must be surrounded by double quotes", input, got)
			return
		}
		// Inner part: every " must be paired as "".
		inner := got[1 : len(got)-1]
		for i := 0; i < len(inner); i++ {
			if inner[i] == '"' {
				if i+1 >= len(inner) || inner[i+1] != '"' {
					t.Errorf("quoteIdent(%q) = %q: unescaped \" at position %d", input, got, i+1)
					return
				}
				i++ // skip the pair
			}
		}
	})
}

// --- S192-T03: 100% branch coverage for helper validators ---

// TestValidateUUID_AllBranches verifies the match/no-match branches explicitly.
func TestValidateUUID_AllBranches(t *testing.T) {
	// Match branch.
	if err := validateUUID("00000000-0000-0000-0000-000000000000"); err != nil {
		t.Errorf("valid UUID returned error: %v", err)
	}
	// No-match branch.
	if err := validateUUID("not-a-uuid"); err == nil {
		t.Error("invalid UUID returned nil error")
	}
}

// TestValidateSlug_AllBranches verifies match and no-match.
func TestValidateSlug_AllBranches(t *testing.T) {
	if err := validateSlug("valid-slug"); err != nil {
		t.Errorf("valid slug returned error: %v", err)
	}
	if err := validateSlug(""); err == nil {
		t.Error("empty slug returned nil error")
	}
}

// TestValidateDate_AllBranches verifies match and no-match.
func TestValidateDate_AllBranches(t *testing.T) {
	if err := validateDate("2026-04-24"); err != nil {
		t.Errorf("valid date returned error: %v", err)
	}
	if err := validateDate("not-a-date"); err == nil {
		t.Error("invalid date returned nil error")
	}
}

// TestValidateMonth_AllBranches verifies match and no-match.
func TestValidateMonth_AllBranches(t *testing.T) {
	if err := validateMonth("2026-04"); err != nil {
		t.Errorf("valid month returned error: %v", err)
	}
	if err := validateMonth("2026-4"); err == nil {
		t.Error("short month returned nil error")
	}
}

// TestValidateEventID_AllBranches verifies match and no-match.
func TestValidateEventID_AllBranches(t *testing.T) {
	if err := validateEventID("evt_123"); err != nil {
		t.Errorf("valid event ID returned error: %v", err)
	}
	if err := validateEventID("has space"); err == nil {
		t.Error("event ID with space returned nil error")
	}
}

// TestValidateReason_AllBranches verifies match and no-match.
func TestValidateReason_AllBranches(t *testing.T) {
	if err := validateReason("ok reason"); err != nil {
		t.Errorf("valid reason returned error: %v", err)
	}
	if err := validateReason("bad'reason"); err == nil {
		t.Error("reason with single quote returned nil error")
	}
}

// TestValidateDuration_AllBranches verifies match and no-match.
func TestValidateDuration_AllBranches(t *testing.T) {
	if err := validateDuration("24h"); err != nil {
		t.Errorf("valid duration returned error: %v", err)
	}
	if err := validateDuration("1w"); err == nil {
		t.Error("unsupported unit returned nil error")
	}
}

// TestMigrations_SQLNotEmpty verifies all migration functions return non-empty SQL.
func TestMigrations_SQLNotEmpty(t *testing.T) {
	migrations := []struct {
		name string
		fn   func() string
	}{
		{"AuditLog", MigrationAuditLog},
		{"UsageDaily", MigrationUsageDaily},
		{"StripeOutbox", MigrationStripeOutbox},
		{"TenantsTable", MigrationTenantsTable},
	}
	for _, m := range migrations {
		t.Run(m.name, func(t *testing.T) {
			sql := m.fn()
			if sql == "" {
				t.Errorf("%s: migration SQL must not be empty", m.name)
			}
			if !strings.Contains(sql, "CREATE TABLE") && !strings.Contains(sql, "ALTER TABLE") {
				t.Errorf("%s: migration SQL should contain CREATE TABLE or ALTER TABLE", m.name)
			}
		})
	}
}

// TestJWTClaimsSchema_RequiredKeys verifies all required JWT claim keys are present.
func TestJWTClaimsSchema_RequiredKeys(t *testing.T) {
	schema := JWTClaimsSchema()
	required := []string{
		"x-hasura-default-role",
		"x-hasura-allowed-roles",
		"x-hasura-user-id",
		"x-hasura-tenant-id",
	}
	for _, key := range required {
		if _, ok := schema[key]; !ok {
			t.Errorf("JWTClaimsSchema missing required key %q", key)
		}
	}
}

// TestHasuraRolePermissions_AllRoles verifies every role has permissions defined.
func TestHasuraRolePermissions_AllRoles(t *testing.T) {
	perms := HasuraRolePermissions()
	for _, role := range AllRoles {
		if _, ok := perms[role]; !ok {
			t.Errorf("HasuraRolePermissions missing role %q", role)
		}
	}
}
