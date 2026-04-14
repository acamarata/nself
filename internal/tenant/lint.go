package tenant

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// LintResult holds the outcome of an RLS lint check for a single table.
type LintResult struct {
	Schema      string   `json:"schema"`
	Table       string   `json:"table"`
	HasRLS      bool     `json:"rls_enabled"`
	HasPolicy   bool     `json:"has_policy"`
	PolicyCount int      `json:"policy_count"`
	Policies    []string `json:"policies,omitempty"`
	HasUserID   bool     `json:"has_user_id"`
	HasTenantID bool     `json:"has_tenant_id"`
	Allowlisted bool     `json:"allowlisted"`
	Reason      string   `json:"allowlist_reason,omitempty"`
	Pass        bool     `json:"pass"`
	Message     string   `json:"message"`
}

// LintRLSReport is the top-level report returned by LintRLSFull.
type LintRLSReport struct {
	Tables           []LintResult     `json:"tables"`
	TotalTables      int              `json:"total_tables"`
	RLSEnabled       int              `json:"rls_enabled"`
	RLSDisabled      int              `json:"rls_disabled"`
	Allowlisted      int              `json:"allowlisted"`
	Violations       int              `json:"violations"`
	CoverageMatrix   []CoverageEntry  `json:"coverage_matrix,omitempty"`
}

// CoverageEntry maps a table to its per-role RLS policy coverage.
type CoverageEntry struct {
	Schema string            `json:"schema"`
	Table  string            `json:"table"`
	Roles  map[string]string `json:"roles"` // role -> policy name or "none"
}

// AllowlistEntry represents a table explicitly exempt from RLS requirements.
type AllowlistEntry struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Reason string `json:"reason"`
}

// LintRLS scans pg_policies vs tables with a tenant_id column and reports
// any tables missing RLS policies. Returns non-nil error only on query failure;
// lint violations are reported in the results. This is the legacy function
// retained for backward compatibility.
func LintRLS(ctx context.Context, cfg *config.Config) ([]LintResult, error) {
	report, err := LintRLSFull(ctx, cfg, nil)
	if err != nil {
		return nil, err
	}
	return report.Tables, nil
}

// LintRLSFull performs an exhaustive RLS audit of every np_* table (and any
// table with user_id or tenant_id columns). Tables in the allowlist are marked
// as passing with an annotation. Returns a full report with coverage matrix.
func LintRLSFull(ctx context.Context, cfg *config.Config, allowlist []AllowlistEntry) (*LintRLSReport, error) {
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	// Query ALL user-data tables: np_* prefixed, or containing user_id/tenant_id.
	sql := `SELECT json_agg(row_to_json(t)) FROM (
		SELECT DISTINCT
			t.table_schema AS schema,
			t.table_name AS table,
			COALESCE(cls.relrowsecurity, false) AS has_rls,
			(SELECT count(*) FROM pg_policies p
			 WHERE p.schemaname = t.table_schema AND p.tablename = t.table_name
			) AS policy_count,
			EXISTS(
				SELECT 1 FROM information_schema.columns c2
				WHERE c2.table_schema = t.table_schema
				  AND c2.table_name = t.table_name
				  AND c2.column_name = 'user_id'
			) AS has_user_id,
			EXISTS(
				SELECT 1 FROM information_schema.columns c3
				WHERE c3.table_schema = t.table_schema
				  AND c3.table_name = t.table_name
				  AND c3.column_name = 'tenant_id'
			) AS has_tenant_id
		FROM information_schema.tables t
		JOIN pg_class cls ON cls.relname = t.table_name
		JOIN pg_namespace ns ON ns.oid = cls.relnamespace AND ns.nspname = t.table_schema
		WHERE t.table_type = 'BASE TABLE'
		  AND t.table_schema NOT IN ('pg_catalog', 'information_schema')
		  AND (
			t.table_name LIKE 'np\_%'
			OR EXISTS(
				SELECT 1 FROM information_schema.columns cx
				WHERE cx.table_schema = t.table_schema
				  AND cx.table_name = t.table_name
				  AND cx.column_name IN ('user_id', 'tenant_id')
			)
		  )
		ORDER BY t.table_schema, t.table_name
	) t;`

	cmd := exec.CommandContext(ctx, "docker", "exec", container,
		"psql", "-U", user, "-d", db, "-tAc", sql,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("querying RLS status: %w", err)
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" || raw == "null" {
		return &LintRLSReport{}, nil
	}

	var rows []struct {
		Schema      string `json:"schema"`
		Table       string `json:"table"`
		HasRLS      bool   `json:"has_rls"`
		PolicyCount int    `json:"policy_count"`
		HasUserID   bool   `json:"has_user_id"`
		HasTenantID bool   `json:"has_tenant_id"`
	}
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, fmt.Errorf("parsing RLS lint results: %w", err)
	}

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

	// Fetch coverage matrix (table x role).
	matrix, err := fetchCoverageMatrix(ctx, container, user, db, rows)
	if err == nil {
		report.CoverageMatrix = matrix
	}

	return report, nil
}

// fetchCoverageMatrix queries pg_policies to build a table x role matrix.
func fetchCoverageMatrix(ctx context.Context, container, user, db string, tables []struct {
	Schema      string `json:"schema"`
	Table       string `json:"table"`
	HasRLS      bool   `json:"has_rls"`
	PolicyCount int    `json:"policy_count"`
	HasUserID   bool   `json:"has_user_id"`
	HasTenantID bool   `json:"has_tenant_id"`
}) ([]CoverageEntry, error) {
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
		sb.WriteString("-- Remediation for " + t.Schema + "." + t.Table + "\n")
		if !t.HasRLS {
			sb.WriteString(fmt.Sprintf("ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;\n", t.Schema, t.Table))
			sb.WriteString(fmt.Sprintf("ALTER TABLE %s.%s FORCE ROW LEVEL SECURITY;\n", t.Schema, t.Table))
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
			safeName := strings.ReplaceAll(t.Table, ".", "_")
			if idCol == "tenant_id" {
				sb.WriteString(fmt.Sprintf(
					"CREATE POLICY %s_owner ON %s.%s USING (%s = (current_setting('%s', true)::jsonb->>'x-hasura-tenant-id')::uuid) WITH CHECK (%s = (current_setting('%s', true)::jsonb->>'x-hasura-tenant-id')::uuid);\n",
					safeName, t.Schema, t.Table, idCol, setting, idCol, setting,
				))
			} else {
				sb.WriteString(fmt.Sprintf(
					"CREATE POLICY %s_owner ON %s.%s USING (%s = current_setting('hasura.user.id', true)::uuid) WITH CHECK (%s = current_setting('hasura.user.id', true)::uuid);\n",
					safeName, t.Schema, t.Table, idCol, idCol,
				))
			}
			sb.WriteString(fmt.Sprintf(
				"CREATE POLICY %s_admin ON %s.%s FOR ALL TO nself_admin USING (true) WITH CHECK (true);\n",
				safeName, t.Schema, t.Table,
			))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
