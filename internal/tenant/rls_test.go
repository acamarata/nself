package tenant

import (
	"strings"
	"testing"
)

// TestRLSIdentifierQuoting_RejectsInjection verifies that callers passing
// SQL meta-characters as schema/table names cannot break out of the
// identifier context. SEC-17. Identifiers must be double-quoted with any
// embedded `"` doubled per the SQL standard.
func TestRLSIdentifierQuoting_RejectsInjection(t *testing.T) {
	maliciousTable := `tenant"; DROP TABLE np_x --`
	maliciousSchema := `bad"schema`

	p := GenerateRLS(maliciousSchema, maliciousTable)

	// Every SQL field must contain the safely-quoted form, NOT the raw
	// injection. Quoted form: each `"` is escaped to `""` and the entire
	// identifier is wrapped in `"..."`.
	wantQuotedTable := `"tenant""; DROP TABLE np_x --"`
	wantQuotedSchema := `"bad""schema"`

	for name, sql := range map[string]string{
		"EnableSQL":      p.EnableSQL,
		"IsolationSQL":   p.IsolationSQL,
		"AdminBypassSQL": p.AdminBypassSQL,
	} {
		if !strings.Contains(sql, wantQuotedTable) {
			t.Errorf("%s missing safely-quoted table identifier %q\nGOT:\n%s",
				name, wantQuotedTable, sql)
		}
		if !strings.Contains(sql, wantQuotedSchema) {
			t.Errorf("%s missing safely-quoted schema identifier %q\nGOT:\n%s",
				name, wantQuotedSchema, sql)
		}
		// The unquoted, raw injection must NEVER appear as an identifier.
		// (We allow it inside the policy-name comment region only because
		// safeName replaces dots; double-quotes survive sanitize, so we
		// verify no unbalanced raw `"; DROP TABLE` sequence escapes the
		// identifier wrapper.)
		if strings.Contains(sql, `"; DROP TABLE np_x --`) &&
			!strings.Contains(sql, `""; DROP TABLE np_x --`) {
			t.Errorf("%s contains UNESCAPED injection sequence; identifier quoting failed:\n%s",
				name, sql)
		}
	}
}

// TestRLSIdentifierQuoting_TenantColumnSQL verifies the same defense applies
// to GenerateTenantColumnSQL. SEC-17.
func TestRLSIdentifierQuoting_TenantColumnSQL(t *testing.T) {
	maliciousSchema := `pub"; DROP SCHEMA public CASCADE; --`
	maliciousTable := `t"x`

	sql := GenerateTenantColumnSQL(maliciousSchema, maliciousTable)

	wantQuotedSchema := `"pub""; DROP SCHEMA public CASCADE; --"`
	wantQuotedTable := `"t""x"`

	if !strings.Contains(sql, wantQuotedSchema) {
		t.Errorf("missing safely-quoted schema %q\nGOT:\n%s", wantQuotedSchema, sql)
	}
	if !strings.Contains(sql, wantQuotedTable) {
		t.Errorf("missing safely-quoted table %q\nGOT:\n%s", wantQuotedTable, sql)
	}
	// Index name embeds the table; verify the dot-replaced safe form is
	// double-quoted as a single identifier (the embedded `"` is doubled).
	if !strings.Contains(sql, `"idx_t""x_tenant_id"`) {
		t.Errorf("index identifier not safely quoted; got:\n%s", sql)
	}
}

// TestRLSIdentifierQuoting_NoBareConcat is a guardrail ensuring the
// happy-path output uses quoted identifiers (not bare `schema.table`).
// SEC-17 regression guard.
func TestRLSIdentifierQuoting_NoBareConcat(t *testing.T) {
	p := GenerateRLS("public", "np_workspaces")

	// The qualified table must appear quoted in every SQL field.
	wantQualified := `"public"."np_workspaces"`
	for name, sql := range map[string]string{
		"EnableSQL":      p.EnableSQL,
		"IsolationSQL":   p.IsolationSQL,
		"AdminBypassSQL": p.AdminBypassSQL,
	} {
		if !strings.Contains(sql, wantQualified) {
			t.Errorf("%s missing quoted qualified identifier %q\nGOT:\n%s",
				name, wantQualified, sql)
		}
	}

	// Policy names must be quoted too.
	if !strings.Contains(p.IsolationSQL, `"np_workspaces_tenant_isolation"`) {
		t.Errorf("isolation policy name not quoted:\n%s", p.IsolationSQL)
	}
	if !strings.Contains(p.AdminBypassSQL, `"np_workspaces_admin_bypass"`) {
		t.Errorf("admin-bypass policy name not quoted:\n%s", p.AdminBypassSQL)
	}
}
