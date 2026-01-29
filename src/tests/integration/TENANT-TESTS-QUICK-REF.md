# Tenant Isolation Tests - Quick Reference Card

## 🚀 Quick Start

```bash
# Start nself
nself start

# Run migrations
nself db migrate

# Run tenant tests
./src/tests/integration/test-tenant-isolation.sh
```

## 📊 Test Suites at a Glance

| Suite | Tests | Focus |
|-------|-------|-------|
| 1. Isolation | 7 | Tenant data separation |
| 2. Lifecycle | 7 | Status transitions |
| 3. Members | 8 | Access control |
| 4. Domains | 8 | Custom domains |
| 5. Security | 8 | RLS enforcement |
| 6. Views | 7 | Queries & stats |
| **TOTAL** | **45+** | **Complete coverage** |

## 🔍 What Gets Tested

### ✅ Tenant Operations
- [x] Create tenant
- [x] Update tenant
- [x] Suspend tenant
- [x] Reactivate tenant
- [x] Soft delete tenant
- [x] Restore tenant

### ✅ Member Management
- [x] Add member (owner, admin, member, guest)
- [x] Update member role
- [x] Remove member
- [x] Check membership
- [x] Get member role

### ✅ Domain Management
- [x] Add custom domain
- [x] Verify domain
- [x] Set primary domain
- [x] Add secondary domains
- [x] Remove domain

### ✅ Security
- [x] RLS policies enabled
- [x] Cross-tenant isolation
- [x] Schema isolation
- [x] Role-based access
- [x] Function existence

## 🎯 Key Functions Tested

```sql
-- Membership
is_tenant_member(tenant_id, user_id) → boolean

-- Roles
get_user_tenant_role(tenant_id, user_id) → text

-- Session
current_tenant_id() → uuid
current_user_id() → uuid

-- Schema Management
create_tenant_schema(tenant_id) → text
drop_tenant_schema(tenant_id) → boolean
```

## 📋 Test Data Created

```
Users:
  - user_a@test.com (Tenant A owner)
  - user_b@test.com (Tenant B owner)
  - user_c@test.com (Tenant A admin/member)

Tenants:
  - test-tenant-a (main test tenant)
  - test-tenant-b (isolation testing)

Domains:
  - tenant-a.example.com
  - tenant-a-alt.example.com
```

**Note**: All test data is automatically cleaned up.

## 🔐 RLS Policies Verified

| Table | Policies |
|-------|----------|
| tenants | SELECT, INSERT, UPDATE, DELETE |
| tenant_members | SELECT, ALL |
| tenant_domains | SELECT, ALL |
| tenant_settings | SELECT, ALL |

## ⚡ Expected Performance

- **Total time**: ~12 seconds
- **Setup**: ~3 seconds
- **Tests**: ~7 seconds
- **Cleanup**: ~2 seconds

## 🐛 Common Issues

### Database not running
```bash
# Error: PostgreSQL container is not running
# Fix:
nself start
```

### Schema not found
```bash
# Error: Tenant schema not found
# Fix:
nself db migrate
```

### Auth table missing
```bash
# Error: auth.users table not found
# Fix: Ensure auth migration ran or create manually:
CREATE TABLE auth.users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL
);
```

## 📈 Success Criteria

✅ All 45+ tests pass
✅ Zero failures
✅ Clean test data cleanup
✅ Exit code 0

## 🔄 Test Flow

```
Setup
  └─ Create users (A, B, C)
  └─ Verify database ready

Suite 1: Isolation
  └─ Create tenants A & B
  └─ Verify data separation

Suite 2: Lifecycle
  └─ Test status transitions
  └─ Test soft delete

Suite 3: Members
  └─ Add/remove members
  └─ Test role management

Suite 4: Domains
  └─ Add custom domains
  └─ Test verification

Suite 5: Security
  └─ Verify RLS enabled
  └─ Test schema isolation

Suite 6: Views
  └─ Test stats views
  └─ Test JSONB storage

Teardown
  └─ Delete test data
  └─ Clean up schemas
```

## 📚 File Locations

| File | Purpose |
|------|---------|
| `test-tenant-isolation.sh` | Main test script |
| `README-TENANT-TESTS.md` | Full documentation |
| `TENANT-TESTS-SUMMARY.md` | Technical details |
| `TENANT-TESTS-QUICK-REF.md` | This file |

## 🎓 Test Assertions

```bash
# Equality
assert_equals "expected" "$actual" "message"

# Inequality
assert_not_equals "not_this" "$actual" "message"

# Existence
assert_not_empty "$value" "should exist"

# Absence
assert_empty "$value" "should be empty"
```

## 🔧 Extending Tests

### Add test to existing suite
```bash
test_tenant_isolation() {
  # ... existing tests ...

  printf "Test 1.X: New test... "
  result=$(db_query_raw "SELECT ..." "$TEST_DB")
  assert_equals "expected" "$result" "description"
}
```

### Create new suite
```bash
test_new_feature() {
  printf "\n${YELLOW}=== Test Suite 7: New Feature ===${NC}\n\n"

  printf "Test 7.1: First test... "
  # test implementation
}

# Add to main()
test_new_feature
```

## 🎨 Output Colors

- 🟢 Green - Test passed
- 🔴 Red - Test failed
- 🟡 Yellow - Section headers

## 📞 Quick Help

```bash
# Run specific suite (manual edit required)
# Comment out suites in main()

# Debug mode
set -x
./src/tests/integration/test-tenant-isolation.sh

# Check database
nself db shell
\dt tenants.*

# View logs
nself logs postgres
```

## ✨ Pro Tips

1. **Run tests frequently** - Catch regressions early
2. **Check exit code** - Use in CI/CD pipelines
3. **Review failures** - Failed assertions show expected vs actual
4. **Clean environment** - Fresh `nself start` if tests behave oddly
5. **Read README** - Full docs in README-TENANT-TESTS.md

## 🎯 CI/CD Integration

```yaml
# GitHub Actions example
- name: Tenant Tests
  run: |
    nself start
    nself db migrate
    ./src/tests/integration/test-tenant-isolation.sh
```

## 📊 Coverage Summary

| Category | Coverage |
|----------|----------|
| Tables | 100% (6/6) |
| Functions | 100% (6/6) |
| RLS Policies | 100% (10/10) |
| Triggers | 100% (2/2) |
| Views | 100% (2/2) |

---

**Version**: 1.0.0
**Last Updated**: 2026-01-29
**Status**: ✅ Ready

*For detailed information, see README-TENANT-TESTS.md*
