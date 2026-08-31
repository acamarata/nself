package tenant

// Purpose: the RLS coverage-matrix query, report-building, and remediation-SQL generation backing LintRLSFull in lint.go.
// Inputs: raw audit rows fetched from Postgres and an allowlist of expected exceptions.
// Outputs: a built LintRLSReport, a disabled-table count, and generated remediation SQL.
// Constraints: split out of lint.go as a pure move (CLI-R12); no behavior change.

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// lintRow is the JSON shape returned by the RLS audit query.
type lintRow struct {
	Schema      string `json:"schema"`
	Table       string `json:"table"`
	HasRLS      bool   `json:"has_rls"`
	PolicyCount int    `json:"policy_count"`
	HasUserID   bool   `json:"has_user_id"`
	HasTenantID bool   `json:"has_tenant_id"`
}

// buildLintReport converts raw query rows + allowlist into a LintRLSReport.
// Extracted for testability — no I/O, pure deterministic logic.
func buildLintReport(rows []lintRow, allowlist []AllowlistEntry) *LintRLSReport {
	// Build allowlist lookup.
	allowed := make(map[string]string) // "schema.table" -> reason
	for _, a := range allowlist {
		allowed[a.Schema+"."+a.Table] = a.Reason
	}

	report := &LintRLSReport{}
	for _, r := range rows {
		key := r.Schema + "." + r.Table
		lr := LintResult{
			Schema:      r.Schema,
			Table:       r.Table,
			HasRLS:      r.HasRLS,
			HasPolicy:   r.PolicyCount > 0,
			PolicyCount: r.PolicyCount,
			HasUserID:   r.HasUserID,
			HasTenantID: r.HasTenantID,
		}

		if reason, ok := allowed[key]; ok {
			lr.Allowlisted = true
			lr.Reason = reason
			lr.Pass = true
			lr.Message = fmt.Sprintf("SKIP: %s allowlisted (%s)", key, reason)
			report.Allowlisted++
		} else if r.HasRLS && r.PolicyCount > 0 {
			lr.Pass = true
			lr.Message = "OK"
			report.RLSEnabled++
		} else {
			lr.Pass = false
			parts := []string{}
			if !r.HasRLS {
				parts = append(parts, "RLS not enabled")
			}
			if r.PolicyCount == 0 {
				parts = append(parts, "no RLS policy found")
			}
			lr.Message = fmt.Sprintf("FAIL: %s — %s", key, strings.Join(parts, " and "))
			report.RLSDisabled++
			report.Violations++
		}
		report.Tables = append(report.Tables, lr)
	}
	report.TotalTables = len(report.Tables)
	return report
}

// fetchCoverageMatrix queries pg_policies to build a table x role matrix.
func fetchCoverageMatrix(ctx context.Context, container, user, db string, tables []lintRow) ([]CoverageEntry, error) {
	sql := `SELECT json_agg(row_to_json(t)) FROM (
		SELECT schemaname AS schema, tablename AS table,
		       policyname AS policy, roles
		FROM pg_policies
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY schemaname, tablename, policyname
	) t;`

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" || raw == "null" {
		return nil, nil
	}

	var policies []struct {
		Schema string   `json:"schema"`
		Table  string   `json:"table"`
		Policy string   `json:"policy"`
		Roles  []string `json:"roles"`
	}
	if err := json.Unmarshal([]byte(raw), &policies); err != nil {
		return nil, err
	}

	// Build lookup: schema.table -> role -> policy name.
	lookup := make(map[string]map[string]string)
	for _, p := range policies {
		key := p.Schema + "." + p.Table
		if lookup[key] == nil {
			lookup[key] = make(map[string]string)
		}
		for _, role := range p.Roles {
			lookup[key][role] = p.Policy
		}
		// {PUBLIC} means all roles.
		if len(p.Roles) == 0 {
			for _, role := range AllRoles {
				if lookup[key][role] == "" {
					lookup[key][role] = p.Policy
				}
			}
		}
	}

	var matrix []CoverageEntry
	for _, t := range tables {
		key := t.Schema + "." + t.Table
		entry := CoverageEntry{
			Schema: t.Schema,
			Table:  t.Table,
			Roles:  make(map[string]string),
		}
		for _, role := range AllRoles {
			if pol, ok := lookup[key][role]; ok {
				entry.Roles[role] = pol
			} else {
				entry.Roles[role] = "none"
			}
		}
		matrix = append(matrix, entry)
	}
	return matrix, nil
}

// DisabledTableCount returns the number of tables failing RLS lint.
// Useful for Prometheus metric emission.
func DisabledTableCount(report *LintRLSReport) int {
	if report == nil {
		return 0
	}
	return report.Violations
}

// GenerateRemediationSQL produces migration SQL for all failing tables in a report.
func GenerateRemediationSQL(report *LintRLSReport) string {
	if report == nil {
		return ""
	}
	var sb strings.Builder
	for _, t := range report.Tables {
		if t.Pass || t.Allowlisted {
			continue
		}
		qSchema := quoteIdent(t.Schema)
		qTable := quoteIdent(t.Table)
		sb.WriteString("-- Remediation for " + t.Schema + "." + t.Table + "\n")
		if !t.HasRLS {
			fmt.Fprintf(&sb, "ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;\n", qSchema, qTable)
			fmt.Fprintf(&sb, "ALTER TABLE %s.%s FORCE ROW LEVEL SECURITY;\n", qSchema, qTable)
		}
		if !t.HasPolicy {
			idCol := "tenant_id"
			setting := "hasura.user.id"
			if t.HasUserID && !t.HasTenantID {
				idCol = "user_id"
			}
			if t.HasTenantID {
				idCol = "tenant_id"
				setting = "hasura.user"
			}
			qIDCol := quoteIdent(idCol)
			// Policy name is a safe SQL identifier derived from table name;
			// replace any dots or special chars before quoting.
			safeName := strings.ReplaceAll(t.Table, ".", "_")
			policyOwner := quoteIdent(safeName + "_owner")
			policyAdmin := quoteIdent(safeName + "_admin")
			if idCol == "tenant_id" {
				fmt.Fprintf(&sb, "CREATE POLICY %s ON %s.%s USING (%s = (current_setting('%s', true)::jsonb->>'x-hasura-tenant-id')::uuid) WITH CHECK (%s = (current_setting('%s', true)::jsonb->>'x-hasura-tenant-id')::uuid);\n",
					policyOwner, qSchema, qTable, qIDCol, setting, qIDCol, setting)
			} else {
				fmt.Fprintf(&sb, "CREATE POLICY %s ON %s.%s USING (%s = current_setting('hasura.user.id', true)::uuid) WITH CHECK (%s = current_setting('hasura.user.id', true)::uuid);\n",
					policyOwner, qSchema, qTable, qIDCol, qIDCol)
			}
			fmt.Fprintf(&sb, "CREATE POLICY %s ON %s.%s FOR ALL TO nself_admin USING (true) WITH CHECK (true);\n",
				policyAdmin, qSchema, qTable)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
