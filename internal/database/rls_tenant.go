package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// TenantRLSPattern describes the access-control model for a table.
type TenantRLSPattern string

const (
	// PatternUserOwned restricts access to rows whose user_id matches the
	// Hasura session variable hasura.user. Admins bypass via hasura.role.
	PatternUserOwned TenantRLSPattern = "user_owned"

	// PatternTenantScoped restricts access to rows whose tenant_id matches the
	// Hasura session variable hasura.tenant. Admins bypass via hasura.role.
	PatternTenantScoped TenantRLSPattern = "tenant_scoped"

	// PatternPublic makes every row visible to all sessions without restriction.
	PatternPublic TenantRLSPattern = "public"

	// PatternAdminOnly restricts all access to sessions where hasura.role = 'admin'.
	PatternAdminOnly TenantRLSPattern = "admin_only"
)

// TenantRLSConfig specifies the parameters needed to generate and apply RLS
// policies for a single table.
type TenantRLSConfig struct {
	Pattern      TenantRLSPattern
	Schema       string
	Table        string
	UserColumn   string // column compared to hasura.user (default: "user_id")
	TenantColumn string // column compared to hasura.tenant (default: "tenant_id")
}

// userColumn returns the effective user column name, defaulting to "user_id".
func (c TenantRLSConfig) userColumn() string {
	if c.UserColumn != "" {
		return c.UserColumn
	}
	return "user_id"
}

// tenantColumn returns the effective tenant column name, defaulting to "tenant_id".
func (c TenantRLSConfig) tenantColumn() string {
	if c.TenantColumn != "" {
		return c.TenantColumn
	}
	return "tenant_id"
}

// GenerateTenantRLSSQL generates the complete SQL block (ENABLE RLS + FORCE RLS +
// DROP/CREATE policies) for the given TenantRLSConfig. The returned SQL is safe
// to pipe directly to psql.
func GenerateTenantRLSSQL(cfg TenantRLSConfig) (string, error) {
	if cfg.Schema == "" {
		return "", fmt.Errorf("generate tenant rls sql: schema is required")
	}
	if cfg.Table == "" {
		return "", fmt.Errorf("generate tenant rls sql: table is required")
	}

	quoted := fmt.Sprintf(`"%s"."%s"`, cfg.Schema, cfg.Table)
	tbl := cfg.Table

	var sb strings.Builder

	// Enable and force RLS.
	sb.WriteString(fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY;\n", quoted))
	sb.WriteString(fmt.Sprintf("ALTER TABLE %s FORCE ROW LEVEL SECURITY;\n", quoted))

	// Drop pre-existing standard policies (idempotent).
	for _, pol := range []string{
		tbl + "_select_own",
		tbl + "_insert_own",
		tbl + "_update_own",
		tbl + "_delete_own",
	} {
		sb.WriteString(fmt.Sprintf("DROP POLICY IF EXISTS %q ON %s;\n", pol, quoted))
	}

	switch cfg.Pattern {
	case PatternUserOwned:
		col := cfg.userColumn()
		pred := fmt.Sprintf(
			`(current_setting('hasura.user', true))::uuid = %s OR current_setting('hasura.role', true) = 'admin'`,
			col,
		)
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR SELECT USING (%s);\n", tbl+"_select_own", quoted, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR INSERT WITH CHECK (%s);\n", tbl+"_insert_own", quoted, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR UPDATE USING (%s) WITH CHECK (%s);\n", tbl+"_update_own", quoted, pred, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR DELETE USING (%s);\n", tbl+"_delete_own", quoted, pred))

	case PatternTenantScoped:
		col := cfg.tenantColumn()
		pred := fmt.Sprintf(
			`(current_setting('hasura.tenant', true))::uuid = %s OR current_setting('hasura.role', true) = 'admin'`,
			col,
		)
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR SELECT USING (%s);\n", tbl+"_select_own", quoted, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR INSERT WITH CHECK (%s);\n", tbl+"_insert_own", quoted, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR UPDATE USING (%s) WITH CHECK (%s);\n", tbl+"_update_own", quoted, pred, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR DELETE USING (%s);\n", tbl+"_delete_own", quoted, pred))

	case PatternPublic:
		// Public: always return true — no filter on any operation.
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR SELECT USING (true);\n", tbl+"_select_own", quoted))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR INSERT WITH CHECK (true);\n", tbl+"_insert_own", quoted))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR UPDATE USING (true) WITH CHECK (true);\n", tbl+"_update_own", quoted))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR DELETE USING (true);\n", tbl+"_delete_own", quoted))

	case PatternAdminOnly:
		pred := `current_setting('hasura.role', true) = 'admin'`
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR SELECT USING (%s);\n", tbl+"_select_own", quoted, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR INSERT WITH CHECK (%s);\n", tbl+"_insert_own", quoted, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR UPDATE USING (%s) WITH CHECK (%s);\n", tbl+"_update_own", quoted, pred, pred))
		sb.WriteString(fmt.Sprintf("CREATE POLICY %q ON %s FOR DELETE USING (%s);\n", tbl+"_delete_own", quoted, pred))

	default:
		return "", fmt.Errorf("generate tenant rls sql: unknown pattern %q", cfg.Pattern)
	}

	return sb.String(), nil
}

// ApplyTenantRLS generates the RLS SQL for the given TenantRLSConfig and pipes
// it to the postgres container via psql.
func ApplyTenantRLS(ctx context.Context, dbcfg *config.Config, rlsCfg TenantRLSConfig) error {
	sql, err := GenerateTenantRLSSQL(rlsCfg)
	if err != nil {
		return fmt.Errorf("apply tenant rls: %w", err)
	}
	if err := pipeSQLToContainer(ctx, dbcfg, sql); err != nil {
		return fmt.Errorf("apply tenant rls on %s.%s: %w", rlsCfg.Schema, rlsCfg.Table, err)
	}
	return nil
}

// GenerateRollbackSQL returns SQL that removes all four standard RLS policies
// and disables row-level security on the given schema.table. The schema and
// table names must originate from AuditRLS (pg_catalog validated).
func GenerateRollbackSQL(schema, table string) string {
	quoted := fmt.Sprintf(`"%s"."%s"`, schema, table)
	var sb strings.Builder
	for _, pol := range []string{
		table + "_select_own",
		table + "_insert_own",
		table + "_update_own",
		table + "_delete_own",
	} {
		sb.WriteString(fmt.Sprintf("DROP POLICY IF EXISTS %q ON %s;\n", pol, quoted))
	}
	sb.WriteString(fmt.Sprintf("ALTER TABLE %s DISABLE ROW LEVEL SECURITY;\n", quoted))
	return sb.String()
}
