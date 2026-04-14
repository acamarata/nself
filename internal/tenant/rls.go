package tenant

import (
	"fmt"
	"strings"
)

// RLSPolicy holds the generated SQL for a single table's RLS policies.
type RLSPolicy struct {
	TableName      string
	EnableSQL      string
	IsolationSQL   string
	AdminBypassSQL string
}

// GenerateRLS produces the standard RLS policy SQL for a tenant-scoped table.
// The table must have a tenant_id UUID column. Two policies are created:
//   - {table}_tenant_isolation: restricts rows to the JWT tenant_id claim
//   - {table}_admin_bypass: allows nself_admin role full access
func GenerateRLS(schema, table string) *RLSPolicy {
	qualified := schema + "." + table
	safeName := strings.ReplaceAll(table, ".", "_")

	return &RLSPolicy{
		TableName: qualified,
		EnableSQL: fmt.Sprintf(
			"ALTER TABLE %s ENABLE ROW LEVEL SECURITY;\n"+
				"ALTER TABLE %s FORCE ROW LEVEL SECURITY;",
			qualified, qualified,
		),
		IsolationSQL: fmt.Sprintf(
			"CREATE POLICY %s_tenant_isolation ON %s\n"+
				"  USING (tenant_id = (current_setting('hasura.user')::jsonb->>'x-hasura-tenant-id')::uuid);",
			safeName, qualified,
		),
		AdminBypassSQL: fmt.Sprintf(
			"CREATE POLICY %s_admin_bypass ON %s\n"+
				"  FOR ALL\n"+
				"  TO nself_admin\n"+
				"  USING (true)\n"+
				"  WITH CHECK (true);",
			safeName, qualified,
		),
	}
}

// GenerateRLSSQL returns the complete SQL string for enabling RLS on a table.
func GenerateRLSSQL(schema, table string) string {
	p := GenerateRLS(schema, table)
	return strings.Join([]string{
		"-- RLS policies for " + p.TableName,
		p.EnableSQL,
		"",
		p.IsolationSQL,
		"",
		p.AdminBypassSQL,
	}, "\n")
}

// GenerateTenantColumnSQL returns SQL to add tenant_id column, index, and FK
// to an existing table that does not yet have one.
func GenerateTenantColumnSQL(schema, table string) string {
	qualified := schema + "." + table
	safeName := strings.ReplaceAll(table, ".", "_")

	return fmt.Sprintf(
		"ALTER TABLE %s ADD COLUMN tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE RESTRICT;\n"+
			"CREATE INDEX idx_%s_tenant_id ON %s (tenant_id);",
		qualified, safeName, qualified,
	)
}

// HasuraPermissionFilter returns the Hasura permission filter JSON for
// tenant-scoped tables. This is applied to select/insert/update/delete
// permissions for tenant_member and tenant_admin roles.
func HasuraPermissionFilter() map[string]interface{} {
	return map[string]interface{}{
		"tenant_id": map[string]interface{}{
			"_eq": "X-Hasura-Tenant-Id",
		},
	}
}

// JWTClaimsSchema returns the expected JWT claims structure for Hasura
// multi-tenancy. This is used to validate JWT configuration.
func JWTClaimsSchema() map[string]string {
	return map[string]string{
		"x-hasura-default-role":  "Required: default role for the user",
		"x-hasura-allowed-roles": "Required: array of allowed roles",
		"x-hasura-user-id":       "Required: user UUID",
		"x-hasura-tenant-id":     "Required: tenant UUID for tenant-scoped access",
		"x-hasura-tenant-plan":   "Optional: tenant plan tier for feature gating",
	}
}

// HasuraRolePermissions returns the permission inheritance model for all 6 roles.
func HasuraRolePermissions() map[string][]string {
	return map[string][]string{
		RoleAnonymous:    {"select"},                                // public read-only
		RoleUser:         {"select"},                                // authenticated, no tenant context
		RoleTenantMember: {"select", "insert"},                     // tenant read + create
		RoleTenantAdmin:  {"select", "insert", "update", "delete"}, // tenant full CRUD
		RoleAdmin:        {"select", "insert", "update", "delete"}, // platform admin, no RLS
		RoleService:      {"select", "insert", "update", "delete"}, // service-to-service, no RLS
	}
}
