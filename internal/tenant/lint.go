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
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	HasRLS    bool   `json:"has_rls"`
	HasPolicy bool   `json:"has_policy"`
	Pass      bool   `json:"pass"`
	Message   string `json:"message"`
}

// LintRLS scans pg_policies vs tables with a tenant_id column and reports
// any tables missing RLS policies. Returns non-nil error only on query failure;
// lint violations are reported in the results.
func LintRLS(ctx context.Context, cfg *config.Config) ([]LintResult, error) {
	container := cfg.ProjectName + "_postgres"
	user := cfg.Postgres.User
	if user == "" {
		user = "postgres"
	}
	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}

	// Find all tables with a tenant_id column and check RLS status.
	sql := `SELECT json_agg(row_to_json(t)) FROM (
		SELECT
			c.table_schema AS schema,
			c.table_name AS table,
			COALESCE(cls.relrowsecurity, false) AS has_rls,
			EXISTS(
				SELECT 1 FROM pg_policies p
				WHERE p.schemaname = c.table_schema AND p.tablename = c.table_name
			) AS has_policy
		FROM information_schema.columns c
		JOIN pg_class cls ON cls.relname = c.table_name
		JOIN pg_namespace ns ON ns.oid = cls.relnamespace AND ns.nspname = c.table_schema
		WHERE c.column_name = 'tenant_id'
		  AND c.table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY c.table_schema, c.table_name
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
		return nil, nil
	}

	var rows []struct {
		Schema    string `json:"schema"`
		Table     string `json:"table"`
		HasRLS    bool   `json:"has_rls"`
		HasPolicy bool   `json:"has_policy"`
	}
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		return nil, fmt.Errorf("parsing RLS lint results: %w", err)
	}

	var results []LintResult
	for _, r := range rows {
		lr := LintResult{
			Schema:    r.Schema,
			Table:     r.Table,
			HasRLS:    r.HasRLS,
			HasPolicy: r.HasPolicy,
		}
		if r.HasRLS && r.HasPolicy {
			lr.Pass = true
			lr.Message = "OK"
		} else {
			lr.Pass = false
			parts := []string{}
			if !r.HasRLS {
				parts = append(parts, "RLS not enabled")
			}
			if !r.HasPolicy {
				parts = append(parts, "no RLS policy found")
			}
			lr.Message = fmt.Sprintf("FAIL: %s.%s has tenant_id but %s",
				r.Schema, r.Table, strings.Join(parts, " and "))
		}
		results = append(results, lr)
	}

	return results, nil
}
