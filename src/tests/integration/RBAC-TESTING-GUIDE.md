# RBAC Testing Quick Reference Guide

Quick reference for testing organization RBAC functionality in nself.

## Quick Start

```bash
# 1. Start PostgreSQL
nself start

# 2. Run all RBAC tests
bash src/tests/integration/test-org-rbac.sh

# 3. Run specific test suite
bash -c 'source src/tests/test_framework.sh && source src/tests/integration/test-org-rbac.sh && test_org_create_with_owner'
```

## Test Scenarios Covered

### ✅ Organization Management

| Test | What It Verifies | Key Functions |
|------|------------------|---------------|
| **Create Org** | Org created with owner | `organizations.organizations`, `org_members` |
| **Add Members** | Multiple role types work | `owner`, `admin`, `member`, `guest` |
| **Member Check** | Membership validation | `is_org_member()` |
| **Get Role** | Role retrieval | `get_user_org_role()` |

### ✅ Team Management

| Test | What It Verifies | Key Functions |
|------|------------------|---------------|
| **Create Team** | Team hierarchy works | `organizations.teams` |
| **Add Members** | Team lead vs member | `team_members`, roles |
| **Member Check** | Team membership | `is_team_member()` |

### ✅ Role & Permission System

| Test | What It Verifies | Key Functions |
|------|------------------|---------------|
| **Custom Roles** | Role creation | `permissions.roles` |
| **Assign Perms** | Role-permission link | `role_permissions` |
| **User Roles** | User-role assignment | `user_roles`, `has_permission()` |
| **Revoke** | Permission removal | DELETE cascades |

### ✅ Permission Inheritance

| Test | What It Verifies | Key Functions |
|------|------------------|---------------|
| **Multiple Roles** | Aggregated perms | Multiple `user_roles` |
| **Multiple Teams** | Team membership | Multiple `team_members` |
| **Scoped Perms** | Global vs team vs tenant | `scope`, `scope_id` |
| **Get All Perms** | Permission listing | `get_user_permissions()` |

### ✅ Security Boundaries

| Test | What It Verifies | Key Functions |
|------|------------------|---------------|
| **Org Isolation** | Cross-org access blocked | `is_org_member()` |
| **Role Isolation** | Roles don't leak | `org_id` scoping |
| **Team Isolation** | Teams org-scoped | `org_id` FK |
| **Data Scoping** | No data leakage | Schema isolation |

## Manual Testing Examples

### Test Organization Creation

```sql
-- Create organization
INSERT INTO organizations.organizations (id, slug, name, owner_user_id)
VALUES (gen_random_uuid(), 'acme-corp', 'ACME Corporation', 'user-uuid-here');

-- Verify
SELECT * FROM organizations.organizations WHERE slug = 'acme-corp';

-- Add owner to members
INSERT INTO organizations.org_members (org_id, user_id, role)
SELECT id, owner_user_id, 'owner'
FROM organizations.organizations
WHERE slug = 'acme-corp';
```

### Test Team Creation

```sql
-- Create team
INSERT INTO organizations.teams (id, org_id, name, slug)
SELECT gen_random_uuid(), id, 'Engineering', 'engineering'
FROM organizations.organizations
WHERE slug = 'acme-corp';

-- Add team lead
INSERT INTO organizations.team_members (team_id, user_id, role)
SELECT t.id, 'user-uuid-here', 'lead'
FROM organizations.teams t
JOIN organizations.organizations o ON t.org_id = o.id
WHERE o.slug = 'acme-corp' AND t.slug = 'engineering';

-- Verify membership
SELECT organizations.is_team_member(t.id, 'user-uuid-here')
FROM organizations.teams t
WHERE t.slug = 'engineering';
```

### Test Custom Role with Permissions

```sql
-- Create custom role
INSERT INTO permissions.roles (id, org_id, name, description)
SELECT gen_random_uuid(), id, 'Developer', 'Dev role with limited perms'
FROM organizations.organizations
WHERE slug = 'acme-corp';

-- Get tenant.read permission
SELECT id FROM permissions.permissions WHERE name = 'tenant.read';

-- Assign permission to role
INSERT INTO permissions.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM permissions.roles r
CROSS JOIN permissions.permissions p
WHERE r.name = 'Developer' AND p.name = 'tenant.read';

-- Assign role to user
INSERT INTO permissions.user_roles (user_id, role_id, org_id)
SELECT 'user-uuid', r.id, r.org_id
FROM permissions.roles r
WHERE r.name = 'Developer';

-- Check if user has permission
SELECT permissions.has_permission(
  'user-uuid',
  o.id,
  'tenant.read'
)
FROM organizations.organizations o
WHERE o.slug = 'acme-corp';
```

### Test Permission Scopes

```sql
-- Global scope (default)
INSERT INTO permissions.user_roles (user_id, role_id, org_id, scope)
VALUES ('user-uuid', 'role-uuid', 'org-uuid', 'global');

-- Team scope
INSERT INTO permissions.user_roles (user_id, role_id, org_id, scope, scope_id)
VALUES ('user-uuid', 'role-uuid', 'org-uuid', 'team', 'team-uuid');

-- Tenant scope
INSERT INTO permissions.user_roles (user_id, role_id, org_id, scope, scope_id)
VALUES ('user-uuid', 'role-uuid', 'org-uuid', 'tenant', 'tenant-uuid');

-- Check permission with scope
SELECT permissions.has_permission(
  'user-uuid',
  'org-uuid',
  'tenant.read',
  'team',
  'team-uuid'
);
```

## Common Test Patterns

### Pattern 1: Create Test Organization

```bash
TEST_ORG_ID=$(gen_uuid)
TEST_USER_ID=$(gen_uuid)

exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) \
  VALUES ('$TEST_ORG_ID', 'test-org', 'Test Org', '$TEST_USER_ID')"

exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role) \
  VALUES ('$TEST_ORG_ID', '$TEST_USER_ID', 'owner')"
```

### Pattern 2: Create Role with Permissions

```bash
ROLE_ID=$(gen_uuid)
exec_sql "INSERT INTO permissions.roles (id, org_id, name) \
  VALUES ('$ROLE_ID', '$TEST_ORG_ID', 'Developer')"

PERM_ID=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'tenant.read'")
exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) \
  VALUES ('$ROLE_ID', '$PERM_ID')"
```

### Pattern 3: Test Permission

```bash
exec_sql "INSERT INTO permissions.user_roles (user_id, role_id, org_id) \
  VALUES ('$USER_ID', '$ROLE_ID', '$ORG_ID')"

HAS_PERM=$(exec_sql "SELECT permissions.has_permission('$USER_ID', '$ORG_ID', 'tenant.read')")
assert_equals "t" "$HAS_PERM" "User should have permission"
```

### Pattern 4: Test Cross-Org Isolation

```bash
# Create two orgs
ORG_A=$(gen_uuid)
ORG_B=$(gen_uuid)
USER=$(gen_uuid)

exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) \
  VALUES ('$ORG_A', 'org-a', 'Org A', '$USER')"

exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) \
  VALUES ('$ORG_B', 'org-b', 'Org B', 'other-user')"

# Add user to org A only
exec_sql "INSERT INTO organizations.org_members (org_id, user_id, role) \
  VALUES ('$ORG_A', '$USER', 'owner')"

# Verify isolation
IS_MEMBER_A=$(exec_sql "SELECT organizations.is_org_member('$ORG_A', '$USER')")
IS_MEMBER_B=$(exec_sql "SELECT organizations.is_org_member('$ORG_B', '$USER')")

assert_equals "t" "$IS_MEMBER_A" "Should be member of org A"
assert_equals "f" "$IS_MEMBER_B" "Should NOT be member of org B"
```

## Default Permissions Reference

All available permissions from the migration:

```sql
-- Tenant permissions
tenant.create    -- Create new tenants
tenant.read      -- View tenant details
tenant.update    -- Update tenant settings
tenant.delete    -- Delete tenants
tenant.manage    -- Full tenant management

-- User permissions
user.create      -- Create new users
user.read        -- View user details
user.update      -- Update user information
user.delete      -- Delete users
user.manage      -- Full user management

-- Team permissions
team.create      -- Create new teams
team.read        -- View team details
team.update      -- Update team settings
team.delete      -- Delete teams
team.manage      -- Full team management

-- Organization permissions
org.billing      -- Manage organization billing
org.settings     -- Manage organization settings
org.members      -- Manage organization members
```

## Troubleshooting

### Issue: Tests Skip with "PostgreSQL not available"

```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Start nself stack
nself start

# Or set connection manually
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_DB=nself
```

### Issue: Migration not applied

```bash
# Apply migration manually
psql -h localhost -U postgres -d nself \
  -f postgres/migrations/010_create_organization_system.sql
```

### Issue: Test data not cleaning up

```sql
-- Manual cleanup
DELETE FROM organizations.organizations WHERE slug LIKE 'test-org-%';
DELETE FROM permissions.roles WHERE org_id IN (
  SELECT id FROM organizations.organizations WHERE slug LIKE 'test-org-%'
);
```

### Issue: Permission not found

```sql
-- List all available permissions
SELECT name, resource_type, action, description
FROM permissions.permissions
ORDER BY resource_type, action;

-- Check if permission exists
SELECT COUNT(*) FROM permissions.permissions WHERE name = 'tenant.read';
```

## Performance Tips

1. **Use transactions** for test isolation:
   ```sql
   BEGIN;
   -- your test queries
   ROLLBACK; -- or COMMIT
   ```

2. **Batch inserts** for multiple test records:
   ```sql
   INSERT INTO permissions.role_permissions (role_id, permission_id)
   SELECT 'role-uuid', id FROM permissions.permissions
   WHERE name IN ('tenant.read', 'user.read', 'team.read');
   ```

3. **Use CTEs** for complex queries:
   ```sql
   WITH org AS (
     SELECT id FROM organizations.organizations WHERE slug = 'acme-corp'
   )
   SELECT * FROM organizations.teams WHERE org_id = (SELECT id FROM org);
   ```

## Database Views for Testing

Use built-in views for easier queries:

```sql
-- User's organizations
SELECT * FROM organizations.user_organizations;

-- User's teams
SELECT * FROM organizations.user_teams;

-- Organization stats
SELECT * FROM organizations.org_stats;
```

## Test Coverage Checklist

- [x] Organization creation with owner
- [x] Multiple member roles (owner, admin, member, guest)
- [x] Team creation and hierarchy
- [x] Team lead vs member roles
- [x] Custom role creation
- [x] Permission assignment to roles
- [x] User role assignment
- [x] Role revocation
- [x] Multiple roles per user (aggregation)
- [x] Multiple teams per user
- [x] Scoped permissions (global, team, tenant)
- [x] Cross-org member isolation
- [x] Cross-org role isolation
- [x] Cross-org team isolation
- [x] Cross-org data scoping
- [x] Permission inheritance
- [x] Built-in functions (is_org_member, get_user_org_role, etc.)
- [x] RLS policies (tested implicitly)
- [x] Cascade deletes (tested via cleanup)

## Next Steps

1. **Add Row-Level Security tests** - Test RLS policies explicitly
2. **Add Audit log tests** - Verify permission_audit table
3. **Add View tests** - Test user_organizations, user_teams views
4. **Add Performance tests** - Test with large datasets
5. **Add Concurrency tests** - Test concurrent role assignments

## References

- Test File: `src/tests/integration/test-org-rbac.sh`
- Migration: `postgres/migrations/010_create_organization_system.sql`
- Framework: `src/tests/test_framework.sh`

---

**Version:** 1.0
**Last Updated:** 2026-01-29
**Maintained By:** nself core team
