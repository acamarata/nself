// Package tenant — sqli_adversarial_test.go
//
// S10.T10 — SQL injection adversarial suite [CR-C]
//
// Strategy:
//   The tenant package uses a two-layer defence:
//     1. Strict whitelist validation (validateSlug, validateUUID, validateMonth,
//        validateDuration, validateReason, validateEventID) — rejects any value
//        that contains SQL meta-characters before it reaches SQL construction.
//     2. sanitize() — escapes single quotes as a belt-and-suspenders fallback.
//     3. quoteIdent() — double-quotes SQL identifiers for DDL paths (rls.go).
//
//   Create() additionally uses psql :'varname' server-side quoting (true
//   parameterization), so no fmt.Sprintf interpolation occurs there at all.
//
//   These tests verify:
//     a) All functions that accept external input reject injection payloads
//        at the validation gate (error returned, never reaching DB).
//     b) sanitize() correctly neutralises any quote that slips through.
//     c) quoteIdent() neutralises injection in identifier context.
//     d) GenerateRLS / GenerateTenantColumnSQL are safe when called with
//        adversarial schema/table names.
//
// Run: go test -run Sqli ./internal/tenant/...

package tenant

import (
	"strings"
	"testing"
)

// sqliPayloads is the canonical list of SQL injection strings used across
// all sub-tests. Every payload must be rejected by the relevant validator.
var sqliPayloads = []string{
	// Classic terminate-and-inject
	"'; DROP TABLE users; --",
	// Boolean bypass
	"' OR 1=1 --",
	// Time-based blind
	"'; SELECT pg_sleep(60); --",
	// Unicode full-width quote bypass (U+02BC MODIFIER LETTER APOSTROPHE)
	"ʼ; DROP TABLE tenants; --",
	// Comment-obfuscated UNION
	"/**/UNION/**/SELECT/**/1,2,3--",
	// Stacked queries via semicolon
	"valid; INSERT INTO tenants VALUES ('evil','basic');",
	// Null-byte injection
	"safe\x00'; DROP TABLE tenants; --",
	// Newline injection (SQL comment bypass)
	"value\n' OR '1'='1",
	// Double-encode attempt (percent-encoded single quote)
	"%27 OR %271%27=%271",
	// COPY-based injection
	"'; COPY tenants TO '/tmp/exfil'; --",
	// pg_read_file exfiltration
	"' UNION SELECT pg_read_file('/etc/passwd') --",
	// Batch via dollar-quoting
	"$$ DROP TABLE tenants; $$",
	// Comment-only injection
	"--' OR '1'='1",
	// CRLF injection
	"val\r\n' OR '1'='1",
}

// TestSqli_SlugRejectsAllPayloads verifies validateSlug rejects every
// injection payload. Slug is used in Destroy, Suspend, and BillingReport.
func TestSqli_SlugRejectsAllPayloads(t *testing.T) {
	for _, payload := range sqliPayloads {
		t.Run("slug/"+shortLabel(payload), func(t *testing.T) {
			err := validateSlug(payload)
			if err == nil {
				t.Errorf("validateSlug(%q): want error (SQL injection), got nil", payload)
			}
		})
	}
}

// TestSqli_UUIDRejectsAllPayloads verifies validateUUID rejects injection
// payloads. UUID is used in Audit (TenantID field) and destroy auditSQL.
func TestSqli_UUIDRejectsAllPayloads(t *testing.T) {
	for _, payload := range sqliPayloads {
		t.Run("uuid/"+shortLabel(payload), func(t *testing.T) {
			err := validateUUID(payload)
			if err == nil {
				t.Errorf("validateUUID(%q): want error (SQL injection), got nil", payload)
			}
		})
	}
}

// TestSqli_MonthRejectsAllPayloads verifies validateMonth rejects injection
// payloads. Month is used in BillingReport WHERE clause.
func TestSqli_MonthRejectsAllPayloads(t *testing.T) {
	for _, payload := range sqliPayloads {
		t.Run("month/"+shortLabel(payload), func(t *testing.T) {
			err := validateMonth(payload)
			if err == nil {
				t.Errorf("validateMonth(%q): want error (SQL injection), got nil", payload)
			}
		})
	}
}

// TestSqli_DurationRejectsAllPayloads verifies validateDuration rejects
// injection payloads. Duration feeds parseDuration() then sinceClause in
// Audit's WHERE clause via sanitize().
func TestSqli_DurationRejectsAllPayloads(t *testing.T) {
	for _, payload := range sqliPayloads {
		t.Run("duration/"+shortLabel(payload), func(t *testing.T) {
			err := validateDuration(payload)
			if err == nil {
				t.Errorf("validateDuration(%q): want error (SQL injection), got nil", payload)
			}
		})
	}
}

// TestSqli_ReasonRejectsAllPayloads verifies validateReason rejects
// injection payloads. Reason is interpolated into Suspend's UPDATE and
// audit_log INSERT via sanitize().
func TestSqli_ReasonRejectsAllPayloads(t *testing.T) {
	for _, payload := range sqliPayloads {
		t.Run("reason/"+shortLabel(payload), func(t *testing.T) {
			err := validateReason(payload)
			if err == nil {
				t.Errorf("validateReason(%q): want error (SQL injection), got nil", payload)
			}
		})
	}
}

// TestSqli_EventIDRejectsAllPayloads verifies validateEventID rejects
// injection payloads. EventID is interpolated into RetryStripeEvent's UPDATE.
func TestSqli_EventIDRejectsAllPayloads(t *testing.T) {
	for _, payload := range sqliPayloads {
		t.Run("eventid/"+shortLabel(payload), func(t *testing.T) {
			err := validateEventID(payload)
			if err == nil {
				t.Errorf("validateEventID(%q): want error (SQL injection), got nil", payload)
			}
		})
	}
}

// TestSqli_SanitizeMustDoubleAllQuotes verifies sanitize() correctly escapes
// every single quote. This is the belt-and-suspenders layer.
// Even if a value somehow passed validation (e.g. via a future code path),
// sanitize ensures it cannot terminate a SQL literal.
func TestSqli_SanitizeMustDoubleAllQuotes(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"classic_drop", "'; DROP TABLE users; --"},
		{"or_bypass", "' OR 1=1 --"},
		{"pg_sleep", "'; SELECT pg_sleep(60); --"},
		{"stacked_insert", "'; INSERT INTO tenants VALUES ('evil'); --"},
		{"exfiltrate", "' UNION SELECT pg_read_file('/etc/passwd') --"},
		{"double_encoded", "%27 OR %271%27=%271"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := sanitize(tc.input)
			// Verify every single quote from input is doubled in output.
			inputQuotes := strings.Count(tc.input, "'")
			if inputQuotes > 0 {
				outputDoubles := strings.Count(out, "''")
				if outputDoubles < inputQuotes {
					t.Errorf("sanitize(%q) = %q: %d single quotes in input, only %d '' pairs in output",
						tc.input, out, inputQuotes, outputDoubles)
				}
			}
			// Verify output has no lone unescaped single quote that could terminate a literal.
			// Strategy: replace all '' with X and check no lone ' remains.
			cleaned := strings.ReplaceAll(out, "''", "XX")
			if strings.Contains(cleaned, "'") {
				t.Errorf("sanitize(%q) = %q: lone unescaped single quote remains after escaping", tc.input, out)
			}
		})
	}
}

// TestSqli_QuoteIdentNeutralisesInjection verifies quoteIdent() used in
// GenerateRLS and GenerateTenantColumnSQL cannot be broken by injection payloads
// masquerading as schema or table names.
func TestSqli_QuoteIdentNeutralisesInjection(t *testing.T) {
	attacks := []string{
		`"; DROP TABLE tenants; --`,
		`" OR 1=1 --`,
		`public"."malicious`,
		`'; DROP TABLE tenants; --`,
		`table\x00hidden`,
		`/**/UNION/**/SELECT`,
	}
	for _, attack := range attacks {
		t.Run("quoteident/"+shortLabel(attack), func(t *testing.T) {
			got := quoteIdent(attack)
			// Must be surrounded by double-quotes.
			if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
				t.Errorf("quoteIdent(%q) = %q: must be surrounded by double quotes", attack, got)
			}
			// No unescaped double quote must appear inside the body.
			inner := got[1 : len(got)-1]
			for i := 0; i < len(inner); i++ {
				if inner[i] == '"' {
					if i+1 >= len(inner) || inner[i+1] != '"' {
						t.Errorf("quoteIdent(%q) = %q: unescaped double quote at body position %d", attack, got, i)
					} else {
						i++ // skip escaped pair
					}
				}
			}
		})
	}
}

// TestSqli_GenerateRLSWithAdversarialSchema verifies GenerateRLS produces
// safe SQL even when called with adversarial schema/table name strings.
// Because quoteIdent() wraps every identifier, SQL meta-chars are contained.
func TestSqli_GenerateRLSWithAdversarialSchema(t *testing.T) {
	adversarialNames := []struct {
		schema string
		table  string
	}{
		{`"; DROP TABLE tenants; --`, "np_workspaces"},
		{"public", `"; DROP TABLE tenants; --`},
		{`' OR 1=1 --`, `np_foo`},
		{"public", `table"injected`},
	}
	for _, tc := range adversarialNames {
		label := shortLabel(tc.schema + "." + tc.table)
		t.Run("rls/"+label, func(t *testing.T) {
			p := GenerateRLS(tc.schema, tc.table)
			// Output must not contain an un-quoted DROP, UNION, or similar keyword
			// that would only appear if injection escaped the identifier context.
			for _, kw := range []string{"DROP TABLE", "OR 1=1", "UNION SELECT"} {
				if containsUnquoted(p.EnableSQL, kw) ||
					containsUnquoted(p.IsolationSQL, kw) ||
					containsUnquoted(p.AdminBypassSQL, kw) {
					t.Errorf("GenerateRLS: SQL keyword %q appears unquoted in output for schema=%q table=%q\nEnable: %s\nIsolation: %s\nAdmin: %s",
						kw, tc.schema, tc.table, p.EnableSQL, p.IsolationSQL, p.AdminBypassSQL)
				}
			}
			// Every generated SQL must contain the ENABLE ROW LEVEL SECURITY token.
			if !strings.Contains(p.EnableSQL, "ENABLE ROW LEVEL SECURITY") {
				t.Errorf("GenerateRLS: EnableSQL missing ENABLE ROW LEVEL SECURITY: %s", p.EnableSQL)
			}
		})
	}
}

// TestSqli_GenerateTenantColumnSQLWithAdversarialTable verifies that the
// ALTER TABLE and CREATE INDEX paths do not propagate identifier injection.
func TestSqli_GenerateTenantColumnSQLWithAdversarialTable(t *testing.T) {
	adversarialTables := []string{
		`"; DROP TABLE tenants; --`,
		`' OR 1=1 --`,
		`table"injected`,
		`/**/UNION/**/SELECT`,
	}
	for _, table := range adversarialTables {
		t.Run("tenantcol/"+shortLabel(table), func(t *testing.T) {
			sql := GenerateTenantColumnSQL("public", table)
			for _, kw := range []string{"DROP TABLE", "OR 1=1", "UNION SELECT"} {
				if containsUnquoted(sql, kw) {
					t.Errorf("GenerateTenantColumnSQL: SQL keyword %q unquoted in output for table=%q\n%s",
						kw, table, sql)
				}
			}
			if !strings.Contains(sql, "ADD COLUMN tenant_id UUID") {
				t.Errorf("GenerateTenantColumnSQL output malformed for table=%q:\n%s", table, sql)
			}
		})
	}
}

// TestSqli_UnicodeSQLBypassRejection verifies that Unicode variants of SQL
// meta-characters (full-width apostrophe, modifier letter apostrophe, etc.)
// are rejected by slug/reason validators, since they may bypass naive
// single-quote-only filters.
func TestSqli_UnicodeSQLBypassRejection(t *testing.T) {
	unicodeAttacks := []struct {
		name  string
		value string
	}{
		// Full-width apostrophe U+FF07
		{"fullwidth_apostrophe_slug", "＇ OR 1=1"},
		// Modifier letter apostrophe U+02BC
		{"modifier_apostrophe_slug", "ʼ DROP TABLE tenants"},
		// Left single quotation mark U+2018
		{"left_sq_mark_slug", "‘; DROP"},
		// Right single quotation mark U+2019
		{"right_sq_mark_slug", "’ OR 1=1"},
	}
	for _, tc := range unicodeAttacks {
		t.Run("unicode/slug/"+tc.name, func(t *testing.T) {
			// Slug regex is ASCII-only ([a-zA-Z0-9_-]), so all unicode attacks fail.
			if err := validateSlug(tc.value); err == nil {
				t.Errorf("validateSlug(%q): want error for unicode bypass attempt, got nil", tc.value)
			}
		})
		t.Run("unicode/reason/"+tc.name, func(t *testing.T) {
			// reasonRegex is ASCII-safe set only; unicode outside the set is rejected.
			if err := validateReason(tc.value); err == nil {
				t.Errorf("validateReason(%q): want error for unicode bypass attempt, got nil", tc.value)
			}
		})
	}
}

// --- helpers ---

// shortLabel returns a concise, safe label for a test name from a payload.
func shortLabel(s string) string {
	const maxLen = 40
	out := make([]byte, 0, maxLen)
	for _, b := range []byte(s) {
		if len(out) >= maxLen {
			break
		}
		if b >= 32 && b < 127 && b != '/' && b != '"' {
			out = append(out, b)
		} else {
			out = append(out, '_')
		}
	}
	return string(out)
}

// containsUnquoted returns true if the SQL string contains the keyword
// outside of a double-quoted identifier context. This is a heuristic:
// it checks whether the keyword appears outside "..." blocks.
func containsUnquoted(sql, keyword string) bool {
	// Strip all double-quoted segments and check remainder.
	stripped := removeDoubleQuotedSegments(sql)
	return strings.Contains(strings.ToUpper(stripped), strings.ToUpper(keyword))
}

// removeDoubleQuotedSegments removes all "..."-delimited substrings from s,
// treating "" as an escaped quote inside the identifier (not a terminator).
func removeDoubleQuotedSegments(s string) string {
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inQuote {
			if ch == '"' {
				// Check for escaped double-quote ""
				if i+1 < len(s) && s[i+1] == '"' {
					i++ // skip the pair — still inside the identifier
				} else {
					inQuote = false // closing quote
				}
			}
			// Anything inside the identifier is omitted from the stripped version.
		} else {
			if ch == '"' {
				inQuote = true
			} else {
				b.WriteByte(ch)
			}
		}
	}
	return b.String()
}
