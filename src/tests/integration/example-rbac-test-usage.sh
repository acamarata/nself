#!/usr/bin/env bash
# example-rbac-test-usage.sh - Example usage of RBAC integration tests
#
# This script demonstrates how to:
# 1. Run all RBAC tests
# 2. Run specific test suites
# 3. Run individual tests
# 4. Use the database helpers in your own tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "╔════════════════════════════════════════════════════════════╗"
echo "║ Organization RBAC Integration Tests - Usage Examples      ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""

# ============================================================================
# Example 1: Run All Tests
# ============================================================================

echo "Example 1: Run all RBAC tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Command:"
echo "  bash src/tests/integration/test-org-rbac.sh"
echo ""
echo "This runs all 19 tests across 5 test suites:"
echo "  - Suite 1: Organization Permission Tests (4 tests)"
echo "  - Suite 2: Team Permission Tests (3 tests)"
echo "  - Suite 3: Role Assignment Tests (4 tests)"
echo "  - Suite 4: Permission Inheritance Tests (4 tests)"
echo "  - Suite 5: Cross-Organization Security Tests (4 tests)"
echo ""
echo "Press Enter to run all tests, or Ctrl+C to skip..."
read -r

bash "$SCRIPT_DIR/test-org-rbac.sh"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ============================================================================
# Example 2: Run Specific Test Suite
# ============================================================================

echo "Example 2: Run a specific test suite only"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "To run only Suite 1 (Organization Permission Tests):"
echo ""
cat <<'EOF'
#!/bin/bash
source src/tests/test_framework.sh
source src/tests/integration/test-org-rbac.sh

# Run just the organization tests
test_org_create_with_owner
test_org_add_members
test_org_member_check
test_org_user_role

# Show summary
print_test_summary
EOF
echo ""
echo "Would you like to run just Suite 1? (y/n)"
read -r response
if [[ "$response" == "y" || "$response" == "Y" ]]; then
  bash -c '
    source '"$SCRIPT_DIR"'/../test_framework.sh
    source '"$SCRIPT_DIR"'/test-org-rbac.sh

    echo "Running Organization Permission Tests..."
    test_org_create_with_owner
    test_org_add_members
    test_org_member_check
    test_org_user_role

    print_test_summary
  '
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ============================================================================
# Example 3: Run Individual Test
# ============================================================================

echo "Example 3: Run a single test function"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "To run just one specific test (e.g., test_custom_role_create):"
echo ""
cat <<'EOF'
#!/bin/bash
source src/tests/test_framework.sh
source src/tests/integration/test-org-rbac.sh

# Run single test
test_custom_role_create

# Show summary
print_test_summary
EOF
echo ""
echo "Would you like to run test_cross_org_isolation? (y/n)"
read -r response
if [[ "$response" == "y" || "$response" == "Y" ]]; then
  bash -c '
    source '"$SCRIPT_DIR"'/../test_framework.sh
    source '"$SCRIPT_DIR"'/test-org-rbac.sh

    echo "Running test_cross_org_isolation..."
    test_cross_org_isolation

    print_test_summary
  '
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ============================================================================
# Example 4: Use Database Helpers
# ============================================================================

echo "Example 4: Use database helper functions in your own tests"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "The test file provides several database helper functions you can reuse:"
echo ""
echo "  exec_sql \"SQL_QUERY\"              - Execute SQL and return result"
echo "  exec_sql_file \"path/to/file.sql\"  - Execute SQL file"
echo "  is_postgres_available              - Check if PostgreSQL is accessible"
echo "  ensure_migration                   - Auto-apply migration if needed"
echo "  gen_uuid                           - Generate UUID (cross-platform)"
echo ""
echo "Example usage:"
echo ""
cat <<'EOF'
#!/bin/bash
source src/tests/integration/test-org-rbac.sh

# Check if PostgreSQL is available
if ! is_postgres_available; then
  echo "PostgreSQL not available"
  exit 1
fi

# Generate a UUID
org_id=$(gen_uuid)
echo "Generated org ID: $org_id"

# Execute SQL
exec_sql "SELECT COUNT(*) FROM organizations.organizations"

# Create test organization
exec_sql "INSERT INTO organizations.organizations (id, slug, name, owner_user_id) \
  VALUES ('$org_id', 'my-org', 'My Organization', '$(gen_uuid)')"

# Verify
count=$(exec_sql "SELECT COUNT(*) FROM organizations.organizations WHERE id = '$org_id'")
echo "Organizations with ID $org_id: $count"

# Cleanup
exec_sql "DELETE FROM organizations.organizations WHERE id = '$org_id'"
EOF
echo ""

# ============================================================================
# Example 5: Write Custom Test
# ============================================================================

echo ""
echo "Example 5: Write your own custom test"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Here's a template for adding your own test:"
echo ""
cat <<'EOF'
#!/bin/bash
source src/tests/test_framework.sh
source src/tests/integration/test-org-rbac.sh

test_my_custom_feature() {
  describe "Description of what this test verifies"

  # Setup
  setup_org_tests || return 0

  # Your test logic
  local result=$(exec_sql "SELECT ...")
  assert_equals "expected" "$result" "Assertion message"

  # Another assertion
  local another=$(exec_sql "SELECT ...")
  assert_equals "expected2" "$another" "Another check"

  # Cleanup
  teardown_org_tests
}

# Run your test
print_test_header "My Custom Tests"
test_my_custom_feature
print_test_summary
EOF
echo ""

# ============================================================================
# Example 6: CI/CD Integration
# ============================================================================

echo ""
echo "Example 6: CI/CD integration"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "GitHub Actions workflow example:"
echo ""
cat <<'EOF'
name: RBAC Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: nself
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3

      - name: Run RBAC Tests
        env:
          POSTGRES_HOST: localhost
          POSTGRES_PORT: 5432
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: nself
        run: |
          bash src/tests/integration/test-org-rbac.sh
EOF
echo ""

# ============================================================================
# Summary
# ============================================================================

echo ""
echo "╔════════════════════════════════════════════════════════════╗"
echo "║ Summary                                                    ║"
echo "╚════════════════════════════════════════════════════════════╝"
echo ""
echo "You can:"
echo "  1. Run all tests:        bash src/tests/integration/test-org-rbac.sh"
echo "  2. Run specific suites:  Source the file and call test functions"
echo "  3. Run individual tests: Source and call one test function"
echo "  4. Use database helpers: Source and use exec_sql, gen_uuid, etc."
echo "  5. Write custom tests:   Follow the test_* function pattern"
echo "  6. Integrate with CI/CD: Use in GitHub Actions, GitLab CI, etc."
echo ""
echo "Documentation:"
echo "  - Full README:     src/tests/integration/test-org-rbac.README.md"
echo "  - Quick Reference: src/tests/integration/RBAC-TESTING-GUIDE.md"
echo "  - This file:       src/tests/integration/example-rbac-test-usage.sh"
echo ""
echo "For more information, see the documentation files listed above."
echo ""
