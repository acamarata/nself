package gdpr

import (
	"strings"
	"testing"
)

// TestValidateTableStrategy_valid verifies that well-formed table and column
// names are accepted and returned as double-quoted SQL identifiers.
// Covers happy path including schema-qualified names via single identifiers.
// SPORT: MASTER-FEATURES.md — SQL injection hardening — cli
func TestValidateTableStrategy_valid(t *testing.T) {
	cases := []struct {
		table   string
		userCol string
	}{
		{"np_users", "id"},
		{"np_user_sessions", "user_id"},
		{"np_chat_messages", "author_id"},
		{"np_audit_events", "user_id"},
		{"plugin_data", "owner"},
		{"A", "B"},
		{"_private", "_col"},
	}
	for _, tc := range cases {
		tbl := TableStrategy{Table: tc.table, UserCol: tc.userCol}
		qtbl, qcol, err := validateTableStrategy(tbl)
		if err != nil {
			t.Errorf("validateTableStrategy(%q, %q): unexpected error: %v", tc.table, tc.userCol, err)
			continue
		}
		if !strings.HasPrefix(qtbl, `"`) || !strings.HasSuffix(qtbl, `"`) {
			t.Errorf("validateTableStrategy(%q): table not double-quoted: %s", tc.table, qtbl)
		}
		if !strings.HasPrefix(qcol, `"`) || !strings.HasSuffix(qcol, `"`) {
			t.Errorf("validateTableStrategy(%q): col not double-quoted: %s", tc.userCol, qcol)
		}
	}
}

// TestValidateTableStrategy_invalidTable ensures that injection-risky table
// names from the plugin registry are rejected with an error.
// Boundary cases: empty, SQL metacharacters (' ; -- %), oversized (>64 chars).
func TestValidateTableStrategy_invalidTable(t *testing.T) {
	cases := []struct {
		table string
		desc  string
	}{
		{"", "empty table"},
		{"'; DROP TABLE np_users; --", "SQL injection in table"},
		{"has space", "space in table"},
		{"has-hyphen", "hyphen in table"},
		{"has.dot", "dot in table (schema-qualified raw)"},
		{"has%wildcard", "percent in table"},
		{strings.Repeat("a", 65), "oversized table (>64 chars)"},
		{"123starts_digit", "starts with digit"},
		{`"already_quoted"`, "already double-quoted"},
	}
	for _, tc := range cases {
		tbl := TableStrategy{Table: tc.table, UserCol: "id"}
		_, _, err := validateTableStrategy(tbl)
		if err == nil {
			t.Errorf("validateTableStrategy(%q) [%s]: expected error, got nil", tc.table, tc.desc)
		}
	}
}

// TestValidateTableStrategy_invalidColumn ensures that injection-risky column
// names from the plugin registry are rejected with an error.
func TestValidateTableStrategy_invalidColumn(t *testing.T) {
	cases := []struct {
		col  string
		desc string
	}{
		{"", "empty col"},
		{"'; DROP TABLE np_users; --", "SQL injection in col"},
		{"user col", "space in col"},
		{"user-id", "hyphen in col"},
		{"user.id", "dot in col"},
		{"col%val", "percent in col"},
		{strings.Repeat("c", 65), "oversized col (>64 chars)"},
	}
	for _, tc := range cases {
		tbl := TableStrategy{Table: "np_users", UserCol: tc.col}
		_, _, err := validateTableStrategy(tbl)
		if err == nil {
			t.Errorf("validateTableStrategy(col=%q) [%s]: expected error, got nil", tc.col, tc.desc)
		}
	}
}

// TestValidateTableStrategy_coreTablesPass verifies that the built-in core
// tables (always safe, hardcoded) pass validation without error.
func TestValidateTableStrategy_coreTablesPass(t *testing.T) {
	for _, tbl := range coreUserErasureTables() {
		_, _, err := validateTableStrategy(tbl)
		if err != nil {
			t.Errorf("core table %q / col %q failed validation: %v", tbl.Table, tbl.UserCol, err)
		}
	}
}
