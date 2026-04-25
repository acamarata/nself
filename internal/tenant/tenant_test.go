package tenant

import (
	"strings"
	"testing"
)

// TestIsValidPlan verifies valid and invalid plan names.
func TestIsValidPlan(t *testing.T) {
	validCases := []string{"basic", "pro", "elite", "business", "business-plus", "enterprise"}
	for _, p := range validCases {
		if !IsValidPlan(p) {
			t.Errorf("IsValidPlan(%q) = false, want true", p)
		}
	}
	invalidCases := []string{"", "free", "starter", "BASIC", "Pro", "unknown"}
	for _, p := range invalidCases {
		if IsValidPlan(p) {
			t.Errorf("IsValidPlan(%q) = true, want false", p)
		}
	}
}

// TestValidPlansSlice verifies all expected plans are in ValidPlans.
func TestValidPlansSlice(t *testing.T) {
	expected := []Plan{PlanBasic, PlanPro, PlanElite, PlanBusiness, PlanBusinessPlus, PlanEnterprise}
	if len(ValidPlans) != len(expected) {
		t.Errorf("ValidPlans has %d entries, want %d", len(ValidPlans), len(expected))
	}
	for _, p := range expected {
		found := false
		for _, vp := range ValidPlans {
			if vp == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidPlans missing %q", p)
		}
	}
}

// TestGenerateRLSContainsTenantID verifies generated RLS SQL references tenant_id.
func TestGenerateRLSContainsTenantID(t *testing.T) {
	p := GenerateRLS("public", "np_workspaces")
	for _, sql := range []string{p.EnableSQL, p.IsolationSQL, p.AdminBypassSQL} {
		if sql == "" {
			t.Error("GenerateRLS produced empty SQL field")
		}
	}
	if !strings.Contains(p.IsolationSQL, "tenant_id") {
		t.Errorf("IsolationSQL missing tenant_id:\n%s", p.IsolationSQL)
	}
	if !strings.Contains(p.AdminBypassSQL, "nself_admin") {
		t.Errorf("AdminBypassSQL missing nself_admin role:\n%s", p.AdminBypassSQL)
	}
}

// TestGenerateRLSSQLSchemaTable verifies qualified table name appears in output.
func TestGenerateRLSSQLSchemaTable(t *testing.T) {
	sql := GenerateRLSSQL("myschema", "mytable")
	if !strings.Contains(sql, "myschema.mytable") {
		t.Errorf("GenerateRLSSQL missing qualified table name:\n%s", sql)
	}
	if !strings.Contains(sql, "ROW LEVEL SECURITY") {
		t.Errorf("GenerateRLSSQL missing ROW LEVEL SECURITY:\n%s", sql)
	}
}

// TestGenerateTenantColumnSQL verifies the ALTER TABLE and CREATE INDEX output.
func TestGenerateTenantColumnSQL(t *testing.T) {
	sql := GenerateTenantColumnSQL("public", "np_foo")
	if !strings.Contains(sql, "ADD COLUMN tenant_id UUID") {
		t.Errorf("GenerateTenantColumnSQL missing ADD COLUMN tenant_id UUID:\n%s", sql)
	}
	if !strings.Contains(sql, "CREATE INDEX") {
		t.Errorf("GenerateTenantColumnSQL missing CREATE INDEX:\n%s", sql)
	}
}

// TestHasuraPermissionFilter verifies the filter shape required by Hasura.
func TestHasuraPermissionFilter(t *testing.T) {
	f := HasuraPermissionFilter()
	tenantFilter, ok := f["tenant_id"]
	if !ok {
		t.Fatal("HasuraPermissionFilter: missing tenant_id key")
	}
	inner, ok := tenantFilter.(map[string]interface{})
	if !ok {
		t.Fatalf("HasuraPermissionFilter: tenant_id value has wrong type %T", tenantFilter)
	}
	eqVal, ok := inner["_eq"]
	if !ok {
		t.Fatal("HasuraPermissionFilter: missing _eq operator")
	}
	if eqVal != "X-Hasura-Tenant-Id" {
		t.Errorf("HasuraPermissionFilter._eq = %q, want %q", eqVal, "X-Hasura-Tenant-Id")
	}
}

// TestAllRolesOrder verifies all expected roles are declared.
func TestAllRolesOrder(t *testing.T) {
	if len(AllRoles) == 0 {
		t.Fatal("AllRoles is empty")
	}
	expected := []string{RoleAnonymous, RoleUser, RoleTenantMember, RoleTenantAdmin, RoleAdmin, RoleService}
	for i, r := range expected {
		if i >= len(AllRoles) || AllRoles[i] != r {
			t.Errorf("AllRoles[%d] = %q, want %q", i, AllRoles[i], r)
		}
	}
}
