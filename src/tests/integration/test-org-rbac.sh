#!/usr/bin/env bash
# test-org-rbac.sh - Organization RBAC integration tests
# Tests organization hierarchy, team management, and permission system
#
# These tests verify the complete RBAC system including:
# - Organization creation and membership
# - Team management and hierarchy
# - Custom roles and permissions
# - Permission inheritance and aggregation
# - Cross-organization security boundaries

set -euo pipefail

# Source test framework
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$TEST_DIR/../test_framework.sh"

# ============================================================================
# Database Helper Functions
# ============================================================================

# Execute SQL and return result
exec_sql() {
  local sql="$1"
  local db="${POSTGRES_DB:-nself}"
  local user="${POSTGRES_USER:-postgres}"
  local host="${POSTGRES_HOST:-localhost}"
  local port="${POSTGRES_PORT:-5432}"

  PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h "$host" \
    -p "$port" \
    -U "$user" \
    -d "$db" \
    -t \
    -A \
    -c "$sql" 2>/dev/null || echo ""
}

# Execute SQL file
exec_sql_file() {
  local file="$1"
  local db="${POSTGRES_DB:-nself}"
  local user="${POSTGRES_USER:-postgres}"
  local host="${POSTGRES_HOST:-localhost}"
  local port="${POSTGRES_PORT:-5432}"

  PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h "$host" \
    -p "$port" \
    -U "$user" \
    -d "$db" \
    -f "$file" 2>/dev/null
}

# Check if PostgreSQL is available
is_postgres_available() {
  if ! command -v psql >/dev/null 2>&1; then
    return 1
  fi

  local db="${POSTGRES_DB:-nself}"
  local user="${POSTGRES_USER:-postgres}"
  local host="${POSTGRES_HOST:-localhost}"
  local port="${POSTGRES_PORT:-5432}"

  PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
    -h "$host" \
    -p "$port" \
    -U "$user" \
    -d "$db" \
    -c "SELECT 1" >/dev/null 2>&1
}

# Apply migration if needed
ensure_migration() {
  local migration_file="$TEST_DIR/../../postgres/migrations/010_create_organization_system.sql"

  if [[ ! -f "$migration_file" ]]; then
    skip "Migration file not found: $migration_file"
    return 1
  fi

  # Check if organizations schema exists
  local schema_exists=$(exec_sql "SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = 'organizations'")

  if [[ "$schema_exists" == "0" ]]; then
    printf "  Applying organization migration...\n"
    exec_sql_file "$migration_file"
  fi

  return 0
}

# Generate UUID
gen_uuid() {
  if command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr '[:upper:]' '[:lower:]'
  else
    # Fallback: use PostgreSQL to generate UUID
    exec_sql "SELECT gen_random_uuid()"
  fi
}

# ============================================================================
# Test Setup and Teardown
# ============================================================================

setup_org_tests() {
  # Check if PostgreSQL is available
  if ! is_postgres_available; then
    skip "PostgreSQL not available - integration tests require running database"
    return 1
  fi

  # Ensure migration is applied
  ensure_migration || return 1

  # Create test users
  TEST_USER_OWNER=$(gen_uuid)
  TEST_USER_ADMIN=$(gen_uuid)
  TEST_USER_MEMBER=$(gen_uuid)
  TEST_USER_GUEST=$(gen_uuid)
  TEST_USER_OTHER=$(gen_uuid)

  # Create test organization
  TEST_ORG_ID=$(gen_uuid)
  TEST_ORG_SLUG="test-org-$(date +%s)"

  exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) VALUES ('$TEST_ORG_ID', '$TEST_ORG_SLUG', 'Test Organization', '$TEST_USER_OWNER')" >/dev/null

  # Add owner to org_members
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role) VALUES ('$TEST_ORG_ID', '$TEST_USER_OWNER', 'owner')" >/dev/null

  return 0
}

teardown_org_tests() {
  # Clean up test data
  if [[ -n "${TEST_ORG_ID:-}" ]]; then
    exec_sql "DELETE FROM organizations.organizations WHERE id = '$TEST_ORG_ID'" >/dev/null 2>&1 || true
  fi

  # Clean up any other test orgs
  exec_sql "DELETE FROM organizations.organizations WHERE slug LIKE 'test-org-%'" >/dev/null 2>&1 || true

  # Clean up test permissions and roles
  exec_sql "DELETE FROM permissions.user_roles WHERE user_id IN ('$TEST_USER_OWNER', '$TEST_USER_ADMIN', '$TEST_USER_MEMBER', '$TEST_USER_GUEST', '$TEST_USER_OTHER')" >/dev/null 2>&1 || true
  exec_sql "DELETE FROM permissions.roles WHERE org_id = '$TEST_ORG_ID'" >/dev/null 2>&1 || true
}

# ============================================================================
# Test Suite 1: Organization Permission Tests
# ============================================================================

test_org_create_with_owner() {
  describe "Create organization with owner"

  setup_org_tests || return 0

  # Verify organization created
  local org_count=$(exec_sql "SELECT COUNT(*) FROM organizations.organizations WHERE id = '$TEST_ORG_ID'")
  assert_equals "1" "$org_count" "Organization should be created"

  # Verify owner is set
  local owner=$(exec_sql "SELECT owner_user_id FROM organizations.organizations WHERE id = '$TEST_ORG_ID'")
  assert_equals "$TEST_USER_OWNER" "$owner" "Owner should be set correctly"

  # Verify owner in org_members
  local member_count=$(exec_sql "SELECT COUNT(*) FROM organizations.org_members WHERE org_id = '$TEST_ORG_ID' AND user_id = '$TEST_USER_OWNER' AND role = 'owner'")
  assert_equals "1" "$member_count" "Owner should be in org_members with owner role"

  teardown_org_tests
}

test_org_add_members() {
  describe "Add members to organization with different roles"

  setup_org_tests || return 0

  # Add admin
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role, invited_by) VALUES ('$TEST_ORG_ID', '$TEST_USER_ADMIN', 'admin', '$TEST_USER_OWNER')" >/dev/null

  # Add regular member
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role, invited_by) VALUES ('$TEST_ORG_ID', '$TEST_USER_MEMBER', 'member', '$TEST_USER_OWNER')" >/dev/null

  # Add guest
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role, invited_by) VALUES ('$TEST_ORG_ID', '$TEST_USER_GUEST', 'guest', '$TEST_USER_ADMIN')" >/dev/null

  # Verify member count
  local member_count=$(exec_sql "SELECT COUNT(*) FROM organizations.org_members WHERE org_id = '$TEST_ORG_ID'")
  assert_equals "4" "$member_count" "Should have 4 members (owner, admin, member, guest)"

  # Verify roles
  local admin_role=$(exec_sql "SELECT role FROM organizations.org_members WHERE org_id = '$TEST_ORG_ID' AND user_id = '$TEST_USER_ADMIN'")
  assert_equals "admin" "$admin_role" "Admin should have admin role"

  local member_role=$(exec_sql "SELECT role FROM organizations.org_members WHERE org_id = '$TEST_ORG_ID' AND user_id = '$TEST_USER_MEMBER'")
  assert_equals "member" "$member_role" "Member should have member role"

  local guest_role=$(exec_sql "SELECT role FROM organizations.org_members WHERE org_id = '$TEST_ORG_ID' AND user_id = '$TEST_USER_GUEST'")
  assert_equals "guest" "$guest_role" "Guest should have guest role"

  teardown_org_tests
}

test_org_member_check() {
  describe "Check if user is organization member"

  setup_org_tests || return 0

  # Add member
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role) VALUES ('$TEST_ORG_ID', '$TEST_USER_MEMBER', 'member')" >/dev/null

  # Test is_org_member function
  local is_member=$(exec_sql "SELECT organizations.is_org_member('$TEST_ORG_ID', '$TEST_USER_MEMBER')")
  assert_equals "t" "$is_member" "Member should be recognized as org member"

  local is_not_member=$(exec_sql "SELECT organizations.is_org_member('$TEST_ORG_ID', '$TEST_USER_OTHER')")
  assert_equals "f" "$is_not_member" "Non-member should not be recognized as org member"

  teardown_org_tests
}

test_org_user_role() {
  describe "Get user's organization role"

  setup_org_tests || return 0

  # Add admin
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role) VALUES ('$TEST_ORG_ID', '$TEST_USER_ADMIN', 'admin')" >/dev/null

  # Test get_user_org_role function
  local owner_role=$(exec_sql "SELECT organizations.get_user_org_role('$TEST_ORG_ID', '$TEST_USER_OWNER')")
  assert_equals "owner" "$owner_role" "Should return owner role"

  local admin_role=$(exec_sql "SELECT organizations.get_user_org_role('$TEST_ORG_ID', '$TEST_USER_ADMIN')")
  assert_equals "admin" "$admin_role" "Should return admin role"

  local no_role=$(exec_sql "SELECT organizations.get_user_org_role('$TEST_ORG_ID', '$TEST_USER_OTHER')")
  assert_equals "" "$no_role" "Should return empty for non-member"

  teardown_org_tests
}

# ============================================================================
# Test Suite 2: Team Permission Tests
# ============================================================================

test_team_create() {
  describe "Create team within organization"

  setup_org_tests || return 0

  # Create team
  local team_id=$(gen_uuid)
  exec_sql "INSERT INTO organizations.teams (id, org_id, name, slug) VALUES ('$team_id', '$TEST_ORG_ID', 'Engineering Team', 'engineering')" >/dev/null

  # Verify team created
  local team_count=$(exec_sql "SELECT COUNT(*) FROM organizations.teams WHERE id = '$team_id'")
  assert_equals "1" "$team_count" "Team should be created"

  # Verify org_id is correct
  local team_org=$(exec_sql "SELECT org_id FROM organizations.teams WHERE id = '$team_id'")
  assert_equals "$TEST_ORG_ID" "$team_org" "Team should belong to correct organization"

  teardown_org_tests
}

test_team_add_members() {
  describe "Add team lead and members"

  setup_org_tests || return 0

  # Create team
  local team_id=$(gen_uuid)
  exec_sql "INSERT INTO organizations.teams (id, org_id, name, slug) VALUES ('$team_id', '$TEST_ORG_ID', 'Engineering Team', 'engineering')" >/dev/null

  # Add team lead
  exec_sql "INSERT INTO organizations.team_members (team_id, user_id, role, added_by) VALUES ('$team_id', '$TEST_USER_ADMIN', 'lead', '$TEST_USER_OWNER')" >/dev/null

  # Add team member
  exec_sql "INSERT INTO organizations.team_members (team_id, user_id, role, added_by) VALUES ('$team_id', '$TEST_USER_MEMBER', 'member', '$TEST_USER_ADMIN')" >/dev/null

  # Verify member count
  local member_count=$(exec_sql "SELECT COUNT(*) FROM organizations.team_members WHERE team_id = '$team_id'")
  assert_equals "2" "$member_count" "Should have 2 team members"

  # Verify lead role
  local lead_role=$(exec_sql "SELECT role FROM organizations.team_members WHERE team_id = '$team_id' AND user_id = '$TEST_USER_ADMIN'")
  assert_equals "lead" "$lead_role" "Team lead should have lead role"

  # Verify member role
  local member_role=$(exec_sql "SELECT role FROM organizations.team_members WHERE team_id = '$team_id' AND user_id = '$TEST_USER_MEMBER'")
  assert_equals "member" "$member_role" "Team member should have member role"

  teardown_org_tests
}

test_team_member_check() {
  describe "Check if user is team member"

  setup_org_tests || return 0

  # Create team
  local team_id=$(gen_uuid)
  exec_sql "INSERT INTO organizations.teams (id, org_id, name, slug) VALUES ('$team_id', '$TEST_ORG_ID', 'Engineering Team', 'engineering')" >/dev/null

  # Add member
  exec_sql "INSERT INTO organizations.team_members (team_id, user_id, role) VALUES ('$team_id', '$TEST_USER_MEMBER', 'member')" >/dev/null

  # Test is_team_member function
  local is_member=$(exec_sql "SELECT organizations.is_team_member('$team_id', '$TEST_USER_MEMBER')")
  assert_equals "t" "$is_member" "Member should be recognized as team member"

  local is_not_member=$(exec_sql "SELECT organizations.is_team_member('$team_id', '$TEST_USER_OTHER')")
  assert_equals "f" "$is_not_member" "Non-member should not be recognized as team member"

  teardown_org_tests
}

# ============================================================================
# Test Suite 3: Role Assignment Tests
# ============================================================================

test_custom_role_create() {
  describe "Create custom role with permissions"

  setup_org_tests || return 0

  # Create custom role
  local role_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name, description) VALUES ('$role_id', '$TEST_ORG_ID', 'Developer', 'Developer role with limited permissions')" >/dev/null

  # Verify role created
  local role_count=$(exec_sql "SELECT COUNT(*) FROM permissions.roles WHERE id = '$role_id'")
  assert_equals "1" "$role_count" "Role should be created"

  # Verify role belongs to org
  local role_org=$(exec_sql "SELECT org_id FROM permissions.roles WHERE id = '$role_id'")
  assert_equals "$TEST_ORG_ID" "$role_org" "Role should belong to correct organization"

  teardown_org_tests
}

test_role_assign_permissions() {
  describe "Assign permissions to role"

  setup_org_tests || return 0

  # Create custom role
  local role_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name, description) VALUES ('$role_id', '$TEST_ORG_ID', 'Developer', 'Developer role')" >/dev/null

  # Get some default permissions
  local perm_tenant_read=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'tenant.read' LIMIT 1")
  local perm_user_read=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'user.read' LIMIT 1")

  # Assign permissions to role
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm_tenant_read')" >/dev/null
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm_user_read')" >/dev/null

  # Verify permissions assigned
  local perm_count=$(exec_sql "SELECT COUNT(*) FROM permissions.role_permissions WHERE role_id = '$role_id'")
  assert_equals "2" "$perm_count" "Role should have 2 permissions"

  teardown_org_tests
}

test_user_role_assignment() {
  describe "Assign role to user and verify permissions"

  setup_org_tests || return 0

  # Create custom role
  local role_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name, description) VALUES ('$role_id', '$TEST_ORG_ID', 'Developer', 'Developer role')" >/dev/null

  # Get permission
  local perm_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'tenant.read' LIMIT 1")

  # Assign permission to role
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm_id')" >/dev/null

  # Assign role to user
  exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id, granted_by) VALUES ('$TEST_USER_MEMBER', '$role_id', '$TEST_ORG_ID', '$TEST_USER_OWNER')" >/dev/null

  # Verify user has role
  local user_role_count=$(exec_sql "SELECT COUNT(*) FROM permissions.user_roles WHERE user_id = '$TEST_USER_MEMBER' AND role_id = '$role_id'")
  assert_equals "1" "$user_role_count" "User should have role assigned"

  # Verify user has permission through role
  local has_perm=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$TEST_ORG_ID', 'tenant.read')")
  assert_equals "t" "$has_perm" "User should have tenant.read permission through role"

  teardown_org_tests
}

test_role_revoke() {
  describe "Revoke role and verify permissions removed"

  setup_org_tests || return 0

  # Create custom role
  local role_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name, description) VALUES ('$role_id', '$TEST_ORG_ID', 'Developer', 'Developer role')" >/dev/null

  # Get permission
  local perm_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'tenant.read' LIMIT 1")

  # Assign permission to role
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm_id')" >/dev/null

  # Assign role to user
  exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id) VALUES ('$TEST_USER_MEMBER', '$role_id', '$TEST_ORG_ID')" >/dev/null

  # Verify user has permission
  local has_perm_before=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$TEST_ORG_ID', 'tenant.read')")
  assert_equals "t" "$has_perm_before" "User should have permission before revoke"

  # Revoke role
  exec_sql "DELETE FROM permissions.user_roles WHERE user_id = '$TEST_USER_MEMBER' AND role_id = '$role_id'" >/dev/null

  # Verify permission removed
  local has_perm_after=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$TEST_ORG_ID', 'tenant.read')")
  assert_equals "f" "$has_perm_after" "User should not have permission after revoke"

  teardown_org_tests
}

# ============================================================================
# Test Suite 4: Permission Inheritance Tests
# ============================================================================

test_user_multiple_roles() {
  describe "User with multiple roles gets aggregated permissions"

  setup_org_tests || return 0

  # Create two roles
  local role1_id=$(gen_uuid)
  local role2_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name) VALUES ('$role1_id', '$TEST_ORG_ID', 'Role1')" >/dev/null
  exec_sql "INSERT INTO permissions.roles (id, org_id, name) VALUES ('$role2_id', '$TEST_ORG_ID', 'Role2')" >/dev/null

  # Get different permissions
  local perm1_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'tenant.read' LIMIT 1")
  local perm2_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'user.read' LIMIT 1")

  # Assign different permissions to each role
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role1_id', '$perm1_id')" >/dev/null
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role2_id', '$perm2_id')" >/dev/null

  # Assign both roles to user
  exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id) VALUES ('$TEST_USER_MEMBER', '$role1_id', '$TEST_ORG_ID')" >/dev/null
  exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id) VALUES ('$TEST_USER_MEMBER', '$role2_id', '$TEST_ORG_ID')" >/dev/null

  # Verify user has both permissions
  local has_perm1=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$TEST_ORG_ID', 'tenant.read')")
  assert_equals "t" "$has_perm1" "User should have tenant.read from role1"

  local has_perm2=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$TEST_ORG_ID', 'user.read')")
  assert_equals "t" "$has_perm2" "User should have user.read from role2"

  teardown_org_tests
}

test_user_multiple_teams() {
  describe "User in multiple teams inherits permissions from both"

  setup_org_tests || return 0

  # Create two teams
  local team1_id=$(gen_uuid)
  local team2_id=$(gen_uuid)
  exec_sql "INSERT INTO organizations.teams (id, org_id, name, slug) VALUES ('$team1_id', '$TEST_ORG_ID', 'Team1', 'team1')" >/dev/null
  exec_sql "INSERT INTO organizations.teams (id, org_id, name, slug) VALUES ('$team2_id', '$TEST_ORG_ID', 'Team2', 'team2')" >/dev/null

  # Add user to both teams
  exec_sql "INSERT INTO organizations.team_members (team_id, user_id, role) VALUES ('$team1_id', '$TEST_USER_MEMBER', 'member')" >/dev/null
  exec_sql "INSERT INTO organizations.team_members (team_id, user_id, role) VALUES ('$team2_id', '$TEST_USER_MEMBER', 'lead')" >/dev/null

  # Verify user is member of both teams
  local is_team1_member=$(exec_sql "SELECT organizations.is_team_member('$team1_id', '$TEST_USER_MEMBER')")
  assert_equals "t" "$is_team1_member" "User should be member of team1"

  local is_team2_member=$(exec_sql "SELECT organizations.is_team_member('$team2_id', '$TEST_USER_MEMBER')")
  assert_equals "t" "$is_team2_member" "User should be member of team2"

  # Verify different roles in different teams
  local team1_role=$(exec_sql "SELECT role FROM organizations.team_members WHERE team_id = '$team1_id' AND user_id = '$TEST_USER_MEMBER'")
  assert_equals "member" "$team1_role" "User should be member in team1"

  local team2_role=$(exec_sql "SELECT role FROM organizations.team_members WHERE team_id = '$team2_id' AND user_id = '$TEST_USER_MEMBER'")
  assert_equals "lead" "$team2_role" "User should be lead in team2"

  teardown_org_tests
}

test_scoped_permissions() {
  describe "Test scoped permissions (global vs team vs tenant)"

  setup_org_tests || return 0

  # Create role
  local role_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name) VALUES ('$role_id', '$TEST_ORG_ID', 'Developer')" >/dev/null

  # Get permission
  local perm_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'tenant.read' LIMIT 1")
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm_id')" >/dev/null

  # Create team
  local team_id=$(gen_uuid)
  exec_sql "INSERT INTO organizations.teams (id, org_id, name, slug) VALUES ('$team_id', '$TEST_ORG_ID', 'Team', 'team')" >/dev/null

  # Assign role with global scope
  exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id, scope) VALUES ('$TEST_USER_ADMIN', '$role_id', '$TEST_ORG_ID', 'global')" >/dev/null

  # Assign role with team scope
  exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id, scope, scope_id) VALUES ('$TEST_USER_MEMBER', '$role_id', '$TEST_ORG_ID', 'team', '$team_id')" >/dev/null

  # Verify global scoped user has permission
  local has_global=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_ADMIN', '$TEST_ORG_ID', 'tenant.read')")
  assert_equals "t" "$has_global" "User with global scope should have permission"

  # Verify team scoped user has permission in team context
  local has_team=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$TEST_ORG_ID', 'tenant.read', 'team', '$team_id')")
  assert_equals "t" "$has_team" "User with team scope should have permission in team context"

  # Verify team scoped user does NOT have permission globally
  local has_team_global=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$TEST_ORG_ID', 'tenant.read', 'global', NULL)")
  assert_equals "f" "$has_team_global" "User with team scope should NOT have permission globally"

  teardown_org_tests
}

test_get_user_permissions() {
  describe "Get all user permissions across roles"

  setup_org_tests || return 0

  # Create role
  local role_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name) VALUES ('$role_id', '$TEST_ORG_ID', 'Developer')" >/dev/null

  # Get multiple permissions
  local perm1_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'tenant.read' LIMIT 1")
  local perm2_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'user.read' LIMIT 1")
  local perm3_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'team.read' LIMIT 1")

  # Assign permissions to role
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm1_id')" >/dev/null
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm2_id')" >/dev/null
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm3_id')" >/dev/null

  # Assign role to user
  exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id) VALUES ('$TEST_USER_MEMBER', '$role_id', '$TEST_ORG_ID')" >/dev/null

  # Get all user permissions
  local perm_count=$(exec_sql "SELECT COUNT(*) FROM permissions.get_user_permissions('$TEST_USER_MEMBER', '$TEST_ORG_ID')")
  assert_equals "3" "$perm_count" "User should have 3 permissions"

  # Verify specific permission in list
  local has_tenant_read=$(exec_sql "SELECT COUNT(*) FROM permissions.get_user_permissions('$TEST_USER_MEMBER', '$TEST_ORG_ID') WHERE permission_name = 'tenant.read'")
  assert_equals "1" "$has_tenant_read" "tenant.read should be in user permissions"

  teardown_org_tests
}

# ============================================================================
# Test Suite 5: Cross-Organization Security Tests
# ============================================================================

test_cross_org_isolation() {
  describe "User in org_a cannot access org_b resources"

  setup_org_tests || return 0

  # Create second organization
  local org_b_id=$(gen_uuid)
  local org_b_slug="test-org-b-$(date +%s)"
  exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) VALUES ('$org_b_id', '$org_b_slug', 'Org B', '$TEST_USER_OTHER')" >/dev/null
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role) VALUES ('$org_b_id', '$TEST_USER_OTHER', 'owner')" >/dev/null

  # Add user to org_a only
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role) VALUES ('$TEST_ORG_ID', '$TEST_USER_MEMBER', 'member')" >/dev/null

  # Verify user is member of org_a
  local is_member_a=$(exec_sql "SELECT organizations.is_org_member('$TEST_ORG_ID', '$TEST_USER_MEMBER')")
  assert_equals "t" "$is_member_a" "User should be member of org_a"

  # Verify user is NOT member of org_b
  local is_member_b=$(exec_sql "SELECT organizations.is_org_member('$org_b_id', '$TEST_USER_MEMBER')")
  assert_equals "f" "$is_member_b" "User should NOT be member of org_b"

  # Cleanup org_b
  exec_sql "DELETE FROM organizations.organizations WHERE id = '$org_b_id'" >/dev/null

  teardown_org_tests
}

test_cross_org_role_isolation() {
  describe "Roles from org_a do not grant permissions in org_b"

  setup_org_tests || return 0

  # Create second organization
  local org_b_id=$(gen_uuid)
  local org_b_slug="test-org-b-$(date +%s)"
  exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) VALUES ('$org_b_id', '$org_b_slug', 'Org B', '$TEST_USER_OTHER')" >/dev/null
  exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role) VALUES ('$org_b_id', '$TEST_USER_OTHER', 'owner')" >/dev/null

  # Create role in org_a
  local role_a_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name) VALUES ('$role_a_id', '$TEST_ORG_ID', 'Developer')" >/dev/null

  # Assign permission to role
  local perm_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'tenant.read' LIMIT 1")
  exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_a_id', '$perm_id')" >/dev/null

  # Assign role to user in org_a
  exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id) VALUES ('$TEST_USER_MEMBER', '$role_a_id', '$TEST_ORG_ID')" >/dev/null

  # Verify user has permission in org_a
  local has_perm_a=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$TEST_ORG_ID', 'tenant.read')")
  assert_equals "t" "$has_perm_a" "User should have permission in org_a"

  # Verify user does NOT have permission in org_b
  local has_perm_b=$(exec_sql "SELECT permissions.has_permission('$TEST_USER_MEMBER', '$org_b_id', 'tenant.read')")
  assert_equals "f" "$has_perm_b" "User should NOT have permission in org_b"

  # Cleanup
  exec_sql "DELETE FROM organizations.organizations WHERE id = '$org_b_id'" >/dev/null

  teardown_org_tests
}

test_cross_org_team_isolation() {
  describe "Teams from org_a not accessible from org_b"

  setup_org_tests || return 0

  # Create second organization
  local org_b_id=$(gen_uuid)
  local org_b_slug="test-org-b-$(date +%s)"
  exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) VALUES ('$org_b_id', '$org_b_slug', 'Org B', '$TEST_USER_OTHER')" >/dev/null

  # Create team in org_a
  local team_a_id=$(gen_uuid)
  exec_sql "INSERT INTO organizations.teams (id, org_id, name, slug) VALUES ('$team_a_id', '$TEST_ORG_ID', 'Team A', 'team-a')" >/dev/null

  # Add user to team in org_a
  exec_sql "INSERT INTO organizations.team_members (team_id, user_id, role) VALUES ('$team_a_id', '$TEST_USER_MEMBER', 'member')" >/dev/null

  # Verify team belongs to org_a
  local team_org=$(exec_sql "SELECT org_id FROM organizations.teams WHERE id = '$team_a_id'")
  assert_equals "$TEST_ORG_ID" "$team_org" "Team should belong to org_a"

  # Verify team does NOT belong to org_b
  assert_not_equals "$org_b_id" "$team_org" "Team should NOT belong to org_b"

  # Verify user is team member
  local is_member=$(exec_sql "SELECT organizations.is_team_member('$team_a_id', '$TEST_USER_MEMBER')")
  assert_equals "t" "$is_member" "User should be member of team_a"

  # Cleanup
  exec_sql "DELETE FROM organizations.organizations WHERE id = '$org_b_id'" >/dev/null

  teardown_org_tests
}

test_org_data_scoping() {
  describe "Verify org-scoped data cannot leak across organizations"

  setup_org_tests || return 0

  # Create two organizations
  local org_b_id=$(gen_uuid)
  local org_b_slug="test-org-b-$(date +%s)"
  exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) VALUES ('$org_b_id', '$org_b_slug', 'Org B', '$TEST_USER_OTHER')" >/dev/null

  # Create roles in each org with same name
  local role_a_id=$(gen_uuid)
  local role_b_id=$(gen_uuid)
  exec_sql "INSERT INTO permissions.roles (id, org_id, name) VALUES ('$role_a_id', '$TEST_ORG_ID', 'Developer')" >/dev/null
  exec_sql "INSERT INTO permissions.roles (id, org_id, name) VALUES ('$role_b_id', '$org_b_id', 'Developer')" >/dev/null

  # Verify roles are distinct
  local role_a_org=$(exec_sql "SELECT org_id FROM permissions.roles WHERE id = '$role_a_id'")
  local role_b_org=$(exec_sql "SELECT org_id FROM permissions.roles WHERE id = '$role_b_id'")

  assert_equals "$TEST_ORG_ID" "$role_a_org" "Role A should belong to org_a"
  assert_equals "$org_b_id" "$role_b_org" "Role B should belong to org_b"
  assert_not_equals "$role_a_org" "$role_b_org" "Roles should belong to different orgs"

  # Verify cannot query roles across orgs
  local org_a_role_count=$(exec_sql "SELECT COUNT(*) FROM permissions.roles WHERE org_id = '$TEST_ORG_ID'")
  local org_b_role_count=$(exec_sql "SELECT COUNT(*) FROM permissions.roles WHERE org_id = '$org_b_id'")

  assert_equals "1" "$org_a_role_count" "Org A should have 1 role"
  assert_equals "1" "$org_b_role_count" "Org B should have 1 role"

  # Cleanup
  exec_sql "DELETE FROM organizations.organizations WHERE id = '$org_b_id'" >/dev/null

  teardown_org_tests
}

# ============================================================================
# Run All Tests
# ============================================================================

main() {
  print_test_header "Organization RBAC Integration Tests"

  # Check if PostgreSQL is available
  if ! is_postgres_available; then
    printf "\n⚠️  PostgreSQL is not available\n"
    printf "These integration tests require a running PostgreSQL instance.\n"
    printf "\nTo run these tests:\n"
    printf "1. Start PostgreSQL: nself start\n"
    printf "2. Or set connection vars: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB\n\n"
    exit 0
  fi

  printf "\nDatabase connection: OK\n"

  # Test Suite 1: Organization Permissions
  printf "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  printf "Test Suite 1: Organization Permission Tests\n"
  printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  test_org_create_with_owner
  test_org_add_members
  test_org_member_check
  test_org_user_role

  # Test Suite 2: Team Permissions
  printf "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  printf "Test Suite 2: Team Permission Tests\n"
  printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  test_team_create
  test_team_add_members
  test_team_member_check

  # Test Suite 3: Role Assignments
  printf "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  printf "Test Suite 3: Role Assignment Tests\n"
  printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  test_custom_role_create
  test_role_assign_permissions
  test_user_role_assignment
  test_role_revoke

  # Test Suite 4: Permission Inheritance
  printf "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  printf "Test Suite 4: Permission Inheritance Tests\n"
  printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  test_user_multiple_roles
  test_user_multiple_teams
  test_scoped_permissions
  test_get_user_permissions

  # Test Suite 5: Cross-Organization Security
  printf "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  printf "Test Suite 5: Cross-Organization Security Tests\n"
  printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
  test_cross_org_isolation
  test_cross_org_role_isolation
  test_cross_org_team_isolation
  test_org_data_scoping

  # Show final summary
  print_test_summary
}

# Run tests
main
