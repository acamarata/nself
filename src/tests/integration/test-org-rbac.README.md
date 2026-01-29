# Organization RBAC Integration Tests

Comprehensive integration tests for the nself organization hierarchy and role-based access control (RBAC) system.

## Overview

These tests verify the complete organization and permission system defined in migration `010_create_organization_system.sql`. They test:

- **Organization management** - Creating orgs, managing members, role assignments
- **Team hierarchy** - Team creation, membership, and permissions
- **Custom roles** - Creating roles, assigning permissions, scoping
- **Permission inheritance** - Multi-role users, aggregated permissions
- **Security boundaries** - Cross-organization isolation and data scoping

## Prerequisites

### Required Services

The tests require a running PostgreSQL instance with the organization migration applied:

```bash
# Start nself services (includes PostgreSQL)
nself start

# Or use custom PostgreSQL connection
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_DB=nself
```

### Required Tools

- `psql` - PostgreSQL client
- `bash 3.2+` - Shell interpreter
- `uuidgen` - UUID generation (optional, will use PostgreSQL fallback)

## Running the Tests

### Run All Tests

```bash
bash src/tests/integration/test-org-rbac.sh
```

### Expected Output

```
╔════════════════════════════════════════════════════════════╗
║ Organization RBAC Integration Tests
╚════════════════════════════════════════════════════════════╝

Database connection: OK

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Test Suite 1: Organization Permission Tests
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  → Create organization with owner
  ✓ Organization should be created
  ✓ Owner should be set correctly
  ✓ Owner should be in org_members with owner role

  → Add members to organization with different roles
  ✓ Should have 4 members (owner, admin, member, guest)
  ✓ Admin should have admin role
  ✓ Member should have member role
  ✓ Guest should have guest role

...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Test Summary:
  Passed:  XX
  Failed:  0
  Total:   XX
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ All tests passed!
```

## Test Suites

### Suite 1: Organization Permission Tests (4 tests)

Tests basic organization functionality:

- **test_org_create_with_owner** - Verify organization creation with owner
- **test_org_add_members** - Add members with different roles (owner, admin, member, guest)
- **test_org_member_check** - Test `is_org_member()` function
- **test_org_user_role** - Test `get_user_org_role()` function

### Suite 2: Team Permission Tests (3 tests)

Tests team hierarchy and membership:

- **test_team_create** - Create team within organization
- **test_team_add_members** - Add team leads and members
- **test_team_member_check** - Test `is_team_member()` function

### Suite 3: Role Assignment Tests (4 tests)

Tests custom role creation and assignment:

- **test_custom_role_create** - Create custom role with description
- **test_role_assign_permissions** - Assign permissions to role
- **test_user_role_assignment** - Assign role to user and verify permissions
- **test_role_revoke** - Revoke role and verify permissions removed

### Suite 4: Permission Inheritance Tests (4 tests)

Tests complex permission scenarios:

- **test_user_multiple_roles** - User with multiple roles gets aggregated permissions
- **test_user_multiple_teams** - User in multiple teams inherits from both
- **test_scoped_permissions** - Global vs team vs tenant scoped permissions
- **test_get_user_permissions** - Get all user permissions across roles

### Suite 5: Cross-Organization Security Tests (4 tests)

Tests security boundaries between organizations:

- **test_cross_org_isolation** - User in org_a cannot access org_b
- **test_cross_org_role_isolation** - Roles from org_a don't work in org_b
- **test_cross_org_team_isolation** - Teams from org_a not accessible from org_b
- **test_org_data_scoping** - Verify org-scoped data cannot leak

## Test Architecture

### Database Helper Functions

The test file includes helper functions for database operations:

```bash
# Execute SQL query
exec_sql "SELECT * FROM organizations.organizations"

# Execute SQL file
exec_sql_file "path/to/migration.sql"

# Check database availability
is_postgres_available

# Apply migration if needed
ensure_migration

# Generate UUID
gen_uuid
```

### Setup and Teardown

Each test uses proper setup/teardown to ensure isolation:

```bash
setup_org_tests() {
  # Check PostgreSQL availability
  # Apply migration if needed
  # Create test users (owner, admin, member, guest, other)
  # Create test organization
}

teardown_org_tests() {
  # Clean up test organization
  # Clean up test permissions
  # Clean up test roles
}
```

### Test User IDs

Tests use consistent test users:

- `TEST_USER_OWNER` - Organization owner
- `TEST_USER_ADMIN` - Organization admin
- `TEST_USER_MEMBER` - Regular member
- `TEST_USER_GUEST` - Guest user
- `TEST_USER_OTHER` - User from different org

## Database Schema Tested

### Organizations Schema

```sql
organizations.organizations
organizations.org_members
organizations.teams
organizations.team_members
organizations.org_tenants
```

### Permissions Schema

```sql
permissions.roles
permissions.permissions
permissions.role_permissions
permissions.user_roles
permissions.permission_audit
```

### Functions Tested

```sql
-- Organization functions
organizations.current_org_id()
organizations.is_org_member(org_id, user_id)
organizations.get_user_org_role(org_id, user_id)
organizations.is_team_member(team_id, user_id)

-- Permission functions
permissions.has_permission(user_id, org_id, permission_name, scope, scope_id)
permissions.get_user_permissions(user_id, org_id)
```

## Common Issues

### PostgreSQL Not Available

**Error:**
```
⚠️  PostgreSQL is not available
These integration tests require a running PostgreSQL instance.
```

**Solution:**
```bash
# Start nself services
nself start

# Or configure connection
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=postgres
export POSTGRES_DB=nself
```

### Migration Not Applied

**Error:**
```
Migration file not found: postgres/migrations/010_create_organization_system.sql
```

**Solution:**

Run from the nself root directory, or set the correct path:
```bash
cd /path/to/nself
bash src/tests/integration/test-org-rbac.sh
```

### Permission Denied

**Error:**
```
psql: FATAL: password authentication failed
```

**Solution:**

Set correct PostgreSQL credentials:
```bash
export POSTGRES_PASSWORD=your_password
```

## CI/CD Integration

These tests are designed to run in CI/CD pipelines:

```yaml
# GitHub Actions example
- name: Run RBAC Integration Tests
  env:
    POSTGRES_HOST: localhost
    POSTGRES_PORT: 5432
    POSTGRES_USER: postgres
    POSTGRES_PASSWORD: postgres
    POSTGRES_DB: test_db
  run: |
    bash src/tests/integration/test-org-rbac.sh
```

## Test Data Cleanup

Tests automatically clean up after themselves:

- All test organizations are deleted after each test
- All test roles and permissions are removed
- Test data uses timestamped slugs to avoid conflicts
- Failed tests may leave data (use `test-org-%` pattern to find)

### Manual Cleanup

If needed, manually clean up test data:

```sql
-- Clean up test organizations
DELETE FROM organizations.organizations WHERE slug LIKE 'test-org-%';

-- Clean up test roles
DELETE FROM permissions.roles WHERE name LIKE 'Developer%' AND NOT is_builtin;
```

## Extending the Tests

### Adding New Test Cases

Follow the existing pattern:

```bash
test_my_new_feature() {
  describe "Description of what this tests"

  setup_org_tests || return 0

  # Your test logic here
  local result=$(exec_sql "SELECT ...")
  assert_equals "expected" "$result" "Assertion message"

  teardown_org_tests
}
```

Add to the appropriate test suite in `main()`:

```bash
printf "\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
printf "Test Suite X: My New Tests\n"
printf "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
test_my_new_feature
```

### Testing New Permissions

Add to default permissions in migration:

```sql
INSERT INTO permissions.permissions (name, description, resource_type, action) VALUES
('resource.action', 'Description', 'resource', 'action');
```

Then test in suite:

```bash
local perm_id=$(exec_sql "SELECT id FROM permissions.permissions WHERE name = 'resource.action'")
exec_sql "INSERT INTO permissions.role_permissions (role_id, permission_id) VALUES ('$role_id', '$perm_id')"
```

## Performance Considerations

- Tests use database transactions for isolation
- Each test creates minimal test data (1 org, 5 users)
- Cleanup is automatic and efficient
- Average test suite runtime: ~5-10 seconds

## Security Testing

The test suite specifically validates:

1. **Vertical privilege escalation prevention** - Members cannot access admin functions
2. **Horizontal privilege escalation prevention** - Users cannot access other orgs
3. **Permission inheritance correctness** - Aggregated permissions work properly
4. **Scope isolation** - Team/tenant scoped permissions don't leak to global
5. **Role-based access control** - Permissions only granted through roles

## References

- Migration: `postgres/migrations/010_create_organization_system.sql`
- Test Framework: `src/tests/test_framework.sh`
- Related Tests: `src/tests/integration/test-roles.sh`

## License

Part of nself v0.6.0+ - Organization & RBAC System

---

**Last Updated:** 2026-01-29
**Test Coverage:** 19 integration tests across 5 suites
**Database Schema Version:** Migration 010
