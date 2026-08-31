package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/nself-org/cli/internal/config"
)

// RLSTableInfo holds the RLS state for a single table returned by AuditRLS.
type RLSTableInfo struct {
	TableSchema string `json:"table_schema"`
	TableName   string `json:"table_name"`
	RLSEnabled  bool   `json:"rls_enabled"`
	ForceRLS    bool   `json:"force_rls"`
	PolicyCount int    `json:"policy_count"`
}

// auditRLSQuery returns all user-data tables from pg_catalog with their RLS state.
// Schemas in the exclusion list (system catalogs) are omitted. Only tables in
// schemas whose names start with "np_" or equal "public" are returned.
const auditRLSQuery = `
SELECT
    n.nspname AS table_schema,
    c.relname AS table_name,
    c.relrowsecurity AS rls_enabled,
    c.relforcerowsecurity AS force_rls,
    COALESCE((SELECT COUNT(*) FROM pg_policies p WHERE p.schemaname = n.nspname AND p.tablename = c.relname), 0) AS policy_count
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE c.relkind = 'r'
  AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast', 'hdb_catalog')
ORDER BY n.nspname, c.relname
`

// AuditRLS queries pg_catalog.pg_class and pg_policies to return the RLS state
// for every table in schemas starting with "np_" or in schema "public". It
// identifies tables that are missing RLS or have zero policies.
func AuditRLS(ctx context.Context, cfg *config.Config) ([]RLSTableInfo, error) {
	db, err := Open(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("audit rls: %w", err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(ctx, auditRLSQuery)
	if err != nil {
		return nil, fmt.Errorf("audit rls query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []RLSTableInfo
	for rows.Next() {
		var t RLSTableInfo
		var policyCount sql.NullInt64
		if err := rows.Scan(&t.TableSchema, &t.TableName, &t.RLSEnabled, &t.ForceRLS, &policyCount); err != nil {
			return nil, fmt.Errorf("audit rls scan: %w", err)
		}
		if policyCount.Valid {
			t.PolicyCount = int(policyCount.Int64)
		}
		// Only include tables in np_* schemas or public.
		if t.TableSchema != "public" && !strings.HasPrefix(t.TableSchema, "np_") {
			continue
		}
		results = append(results, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit rls rows: %w", err)
	}
	return results, nil
}

// EnableRLSOnTable runs ALTER TABLE to enable and force row-level security on
// the given schema.table. Schema and table names must come from AuditRLS output
// (validated against pg_catalog) and are quoted to prevent SQL injection.
func EnableRLSOnTable(ctx context.Context, cfg *config.Config, schema, table string) error {
	// Quote identifiers using double-quotes. Schema and table originate from
	// pg_catalog queries (AuditRLS), not from raw user input.
	quoted := fmt.Sprintf(`"%s"."%s"`, schema, table)
	sql := fmt.Sprintf(
		`ALTER TABLE %s ENABLE ROW LEVEL SECURITY; ALTER TABLE %s FORCE ROW LEVEL SECURITY;`,
		quoted, quoted,
	)
	slog.InfoContext(ctx, "rls_enable",
		"table_schema", schema,
		"table_name", table,
		"operation", "enable_rls",
	)
	if err := pipeSQLToContainer(ctx, cfg, sql); err != nil {
		slog.ErrorContext(ctx, "rls_enable_failed",
			"table_schema", schema,
			"table_name", table,
			"operation", "enable_rls",
			"error", err.Error(),
		)
		return fmt.Errorf("enable rls on %s.%s: %w", schema, table, err)
	}
	return nil
}

// hasUserIDColumn reports whether the given table has a column named user_id.
func hasUserIDColumn(ctx context.Context, cfg *config.Config, schema, table string) (bool, error) {
	q := `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2 AND column_name = 'user_id'
	)`
	db, err := Open(ctx, cfg)
	if err != nil {
		return false, fmt.Errorf("check user_id column: %w", err)
	}
	defer func() { _ = db.Close() }()

	var exists bool
	if err := db.QueryRowContext(ctx, q, schema, table).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user_id column query: %w", err)
	}
	return exists, nil
}

// ApplyDefaultRLSPolicies creates standard Hasura-compatible RLS policies on
// the given table. Policies use current_setting('hasura.user', true) for user
// identity and current_setting('hasura.role', true) for admin bypass.
//
// If the table has no user_id column the user-based predicates are omitted and
// only admin-bypass policies are applied.
//
// Schema and table names must originate from AuditRLS (pg_catalog validated).
func ApplyDefaultRLSPolicies(ctx context.Context, cfg *config.Config, schema, table string) error {
	hasUserID, err := hasUserIDColumn(ctx, cfg, schema, table)
	if err != nil {
		return fmt.Errorf("apply default rls policies: %w", err)
	}

	quoted := fmt.Sprintf(`"%s"."%s"`, schema, table)

	var sb strings.Builder

	// Drop any pre-existing policies with these names so the function is
	// idempotent across repeated invocations.
	for _, pol := range []string{
		table + "_select_own",
		table + "_insert_own",
		table + "_update_own",
		table + "_delete_own",
	} {
		fmt.Fprintf(&sb, "DROP POLICY IF EXISTS %q ON %s;\n", pol, quoted)
	}

	if hasUserID {
		// User-scoped predicate: owner check OR admin bypass.
		userPredicate := `(current_setting('hasura.user', true))::uuid = user_id OR current_setting('hasura.role', true) = 'admin'`

		fmt.Fprintf(&sb, "CREATE POLICY %q ON %s FOR SELECT USING (%s);\n",
			table+"_select_own", quoted, userPredicate)
		fmt.Fprintf(&sb, "CREATE POLICY %q ON %s FOR INSERT WITH CHECK (%s);\n",
			table+"_insert_own", quoted, userPredicate)
		fmt.Fprintf(&sb, "CREATE POLICY %q ON %s FOR UPDATE USING (%s) WITH CHECK (%s);\n",
			table+"_update_own", quoted, userPredicate, userPredicate)
		fmt.Fprintf(&sb, "CREATE POLICY %q ON %s FOR DELETE USING (%s);\n",
			table+"_delete_own", quoted, userPredicate)
	} else {
		// No user_id column: admin-only access for all operations.
		adminPredicate := `current_setting('hasura.role', true) = 'admin'`

		fmt.Fprintf(&sb, "CREATE POLICY %q ON %s FOR SELECT USING (%s);\n",
			table+"_select_own", quoted, adminPredicate)
		fmt.Fprintf(&sb, "CREATE POLICY %q ON %s FOR INSERT WITH CHECK (%s);\n",
			table+"_insert_own", quoted, adminPredicate)
		fmt.Fprintf(&sb, "CREATE POLICY %q ON %s FOR UPDATE USING (%s) WITH CHECK (%s);\n",
			table+"_update_own", quoted, adminPredicate, adminPredicate)
		fmt.Fprintf(&sb, "CREATE POLICY %q ON %s FOR DELETE USING (%s);\n",
			table+"_delete_own", quoted, adminPredicate)
	}

	slog.InfoContext(ctx, "rls_policies_apply",
		"table_schema", schema,
		"table_name", table,
		"operation", "apply_policies",
		"has_user_id", hasUserID,
	)
	if err := pipeSQLToContainer(ctx, cfg, sb.String()); err != nil {
		slog.ErrorContext(ctx, "rls_policies_apply_failed",
			"table_schema", schema,
			"table_name", table,
			"operation", "apply_policies",
			"error", err.Error(),
		)
		return fmt.Errorf("apply default rls policies on %s.%s: %w", schema, table, err)
	}
	return nil
}

// ApplyRLSBatch runs EnableRLSOnTable and ApplyDefaultRLSPolicies on each
// table in the provided list. Tables that already have RLS enabled and at least
// one policy are skipped. Returns counts of applied and skipped tables along
// with any non-fatal errors encountered.
func ApplyRLSBatch(ctx context.Context, cfg *config.Config, tables []RLSTableInfo) (applied int, skipped int, errs []error) {
	for _, t := range tables {
		// Skip tables that already have RLS enabled and at least one policy.
		if t.RLSEnabled && t.PolicyCount > 0 {
			skipped++
			continue
		}

		if err := EnableRLSOnTable(ctx, cfg, t.TableSchema, t.TableName); err != nil {
			errs = append(errs, fmt.Errorf("enable rls %s.%s: %w", t.TableSchema, t.TableName, err))
			continue
		}

		if err := ApplyDefaultRLSPolicies(ctx, cfg, t.TableSchema, t.TableName); err != nil {
			errs = append(errs, fmt.Errorf("apply policies %s.%s: %w", t.TableSchema, t.TableName, err))
			continue
		}

		applied++
	}
	slog.InfoContext(ctx, "rls_batch_complete",
		"operation", "batch_apply",
		"applied", applied,
		"skipped", skipped,
		"error_count", len(errs),
	)
	return applied, skipped, errs
}
