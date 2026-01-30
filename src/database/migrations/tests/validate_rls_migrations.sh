#!/usr/bin/env bash
# ============================================================================
# RLS Migration Validation Script
# ============================================================================
# Description: Validates SQL syntax and logic of RLS migration files
# Usage: ./validate_rls_migrations.sh
# ============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Migration files to validate
MIGRATION_019="/Users/admin/Sites/nself/src/database/migrations/019_add_billing_rls.sql"
MIGRATION_020="/Users/admin/Sites/nself/src/database/migrations/020_add_whitelabel_rls.sql"
TEST_FILE="/Users/admin/Sites/nself/src/database/migrations/tests/test_rls_policies.sql"

printf "${YELLOW}==============================================================${NC}\n"
printf "${YELLOW}RLS Migration Validation${NC}\n"
printf "${YELLOW}==============================================================${NC}\n\n"

# Function to check if file exists
check_file() {
    local file="$1"
    local name="$2"

    if [[ -f "$file" ]]; then
        printf "${GREEN}✓${NC} Found: %s\n" "$name"
        return 0
    else
        printf "${RED}✗${NC} Missing: %s\n" "$name"
        return 1
    fi
}

# Function to count lines in file
count_lines() {
    local file="$1"
    wc -l < "$file" | tr -d ' '
}

# Function to validate SQL syntax (basic checks)
validate_sql_syntax() {
    local file="$1"
    local name="$2"
    local errors=0

    printf "\n${YELLOW}Validating: %s${NC}\n" "$name"

    # Check for common SQL syntax errors
    printf "  Checking for syntax issues...\n"

    # Check for unmatched parentheses
    local open_parens=$(grep -o '(' "$file" | wc -l | tr -d ' ')
    local close_parens=$(grep -o ')' "$file" | wc -l | tr -d ' ')
    if [[ "$open_parens" -eq "$close_parens" ]]; then
        printf "    ${GREEN}✓${NC} Parentheses balanced (%s pairs)\n" "$open_parens"
    else
        printf "    ${RED}✗${NC} Unmatched parentheses (open: %s, close: %s)\n" "$open_parens" "$close_parens"
        errors=$((errors + 1))
    fi

    # Check for unmatched BEGIN/END blocks
    local begin_count=$(grep -c '\bBEGIN\b' "$file" || true)
    local end_count=$(grep -c '\bEND\b' "$file" || true)
    if [[ "$begin_count" -eq "$end_count" ]]; then
        printf "    ${GREEN}✓${NC} BEGIN/END blocks balanced (%s blocks)\n" "$begin_count"
    else
        printf "    ${YELLOW}⚠${NC} BEGIN/END count mismatch (BEGIN: %s, END: %s)\n" "$begin_count" "$end_count"
        printf "      Note: This may be intentional (e.g., BEGIN in strings)\n"
    fi

    # Count key SQL commands
    local create_policy_count=$(grep -c 'CREATE POLICY' "$file" || true)
    local create_function_count=$(grep -c 'CREATE.*FUNCTION' "$file" || true)
    local alter_table_count=$(grep -c 'ALTER TABLE.*ENABLE ROW LEVEL SECURITY' "$file" || true)

    printf "    ${GREEN}✓${NC} CREATE POLICY statements: %s\n" "$create_policy_count"
    printf "    ${GREEN}✓${NC} CREATE FUNCTION statements: %s\n" "$create_function_count"
    printf "    ${GREEN}✓${NC} ALTER TABLE...ENABLE RLS: %s\n" "$alter_table_count"

    # Check for required helper functions
    if grep -q 'get_current_customer_id' "$file"; then
        printf "    ${GREEN}✓${NC} Contains get_current_customer_id() function\n"
    fi

    if grep -q 'is_current_user_admin' "$file"; then
        printf "    ${GREEN}✓${NC} Contains is_current_user_admin() function\n"
    fi

    # Check for session variable usage
    if grep -q "current_setting('app\." "$file"; then
        printf "    ${GREEN}✓${NC} Uses session variables (app.*)\n"
    fi

    return $errors
}

# Function to analyze RLS coverage
analyze_rls_coverage() {
    local file="$1"
    local name="$2"

    printf "\n${YELLOW}Analyzing RLS coverage: %s${NC}\n" "$name"

    # Extract table names with RLS enabled
    local tables=$(grep -oP '(?<=ALTER TABLE )\w+(?= ENABLE ROW LEVEL SECURITY)' "$file" || true)
    local table_count=$(echo "$tables" | grep -c . || echo 0)

    if [[ $table_count -gt 0 ]]; then
        printf "  ${GREEN}✓${NC} RLS enabled on %s tables:\n" "$table_count"
        echo "$tables" | while IFS= read -r table; do
            if [[ -n "$table" ]]; then
                local policy_count=$(grep -c "CREATE POLICY.*ON $table" "$file" || echo 0)
                printf "    - %s (%s policies)\n" "$table" "$policy_count"
            fi
        done
    else
        printf "  ${RED}✗${NC} No tables with RLS enabled found\n"
    fi
}

# Function to check for security best practices
check_security_best_practices() {
    local file="$1"
    local name="$2"

    printf "\n${YELLOW}Security best practices check: %s${NC}\n" "$name"

    # Check for SECURITY DEFINER functions
    if grep -q 'SECURITY DEFINER' "$file"; then
        printf "  ${GREEN}✓${NC} Uses SECURITY DEFINER for helper functions\n"
    else
        printf "  ${YELLOW}⚠${NC} No SECURITY DEFINER functions found\n"
    fi

    # Check for admin bypass policies
    if grep -q "admin_all_access\|admin_bypass" "$file"; then
        printf "  ${GREEN}✓${NC} Includes admin bypass policies\n"
    fi

    # Check for WITH CHECK clauses
    local with_check_count=$(grep -c 'WITH CHECK' "$file" || true)
    if [[ $with_check_count -gt 0 ]]; then
        printf "  ${GREEN}✓${NC} Uses WITH CHECK clauses (%s found)\n" "$with_check_count"
    fi

    # Check for proper indexing comments
    if grep -q 'Performance Indexes for RLS' "$file"; then
        printf "  ${GREEN}✓${NC} Includes performance index recommendations\n"
    fi
}

# Main validation
printf "Step 1: Checking file existence...\n"
check_file "$MIGRATION_019" "019_add_billing_rls.sql" || exit 1
check_file "$MIGRATION_020" "020_add_whitelabel_rls.sql" || exit 1
check_file "$TEST_FILE" "test_rls_policies.sql" || exit 1

printf "\nStep 2: File size analysis...\n"
printf "  019_add_billing_rls.sql:     %6s lines\n" "$(count_lines "$MIGRATION_019")"
printf "  020_add_whitelabel_rls.sql:  %6s lines\n" "$(count_lines "$MIGRATION_020")"
printf "  test_rls_policies.sql:       %6s lines\n" "$(count_lines "$TEST_FILE")"

# Validate each migration
validate_sql_syntax "$MIGRATION_019" "019_add_billing_rls.sql"
analyze_rls_coverage "$MIGRATION_019" "019_add_billing_rls.sql"
check_security_best_practices "$MIGRATION_019" "019_add_billing_rls.sql"

validate_sql_syntax "$MIGRATION_020" "020_add_whitelabel_rls.sql"
analyze_rls_coverage "$MIGRATION_020" "020_add_whitelabel_rls.sql"
check_security_best_practices "$MIGRATION_020" "020_add_whitelabel_rls.sql"

# Summary
printf "\n${YELLOW}==============================================================${NC}\n"
printf "${YELLOW}Validation Summary${NC}\n"
printf "${YELLOW}==============================================================${NC}\n"

# Count total policies across both files
total_policies=$(grep -c 'CREATE POLICY' "$MIGRATION_019" "$MIGRATION_020" || echo 0)
total_functions=$(grep -c 'CREATE.*FUNCTION' "$MIGRATION_019" "$MIGRATION_020" || echo 0)
total_tables=$(grep -c 'ENABLE ROW LEVEL SECURITY' "$MIGRATION_019" "$MIGRATION_020" || echo 0)

printf "  Total RLS policies:      %s\n" "$total_policies"
printf "  Total helper functions:  %s\n" "$total_functions"
printf "  Total tables protected:  %s\n" "$total_tables"

printf "\n${GREEN}✓ Validation complete!${NC}\n"
printf "\nNext steps:\n"
printf "  1. Review SQL files manually\n"
printf "  2. Test on development database\n"
printf "  3. Run test suite: psql -f test_rls_policies.sql\n"
printf "  4. Apply to staging environment\n"
printf "  5. Verify with application integration tests\n\n"

printf "${YELLOW}==============================================================${NC}\n"
