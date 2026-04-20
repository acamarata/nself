package plugin

import (
	"testing"
)

// TestSanitizeSchemaName verifies that plugin names are safely transformed into
// Postgres schema names, and that SQL-significant characters are rejected.
// S43-T-SEC-02.
func TestSanitizeSchemaName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"ai", "np_ai"},
		{"claw-web", "np_claw_web"},
		{"NOTIFY", "np_notify"},
		{"feature-flags", "np_feature_flags"},
		// Malicious inputs must return the safe fallback "np_invalid".
		{"'; DROP TABLE np_common.schema_versions;--", "np_invalid"},
		{"../../etc/passwd", "np_invalid"},
		{"schema\x00injection", "np_invalid"},
		{`name"with"quotes`, "np_invalid"},
		{"name with spaces", "np_invalid"},
	}
	for _, tc := range cases {
		got := sanitizeSchemaName(tc.input)
		if got != tc.want {
			t.Errorf("sanitizeSchemaName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestQuoteIdent verifies that quoteIdent wraps identifiers in double quotes
// and escapes any embedded double-quote per the SQL standard. S43-T-SEC-02.
func TestQuoteIdent(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"np_ai", `"np_ai"`},
		{"np_claw_web_role", `"np_claw_web_role"`},
		// Belt-and-suspenders: if a double-quote somehow slips through it is
		// escaped, not injected.
		{`np_evil"drop`, `"np_evil""drop"`},
	}
	for _, tc := range cases {
		got := quoteIdent(tc.input)
		if got != tc.want {
			t.Errorf("quoteIdent(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
