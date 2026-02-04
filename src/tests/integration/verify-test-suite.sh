#!/usr/bin/env bash
# verify-test-suite.sh - Verify integration test suite is properly set up
#
# Checks all required files, permissions, and dependencies

set -euo pipefail

# Color definitions
readonly COLOR_RESET='\033[0m'
readonly COLOR_GREEN='\033[0;32m'
readonly COLOR_RED='\033[0;31m'
readonly COLOR_YELLOW='\033[0;33m'
readonly COLOR_BLUE='\033[0;34m'

# Test directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Verification results
CHECKS_PASSED=0
CHECKS_FAILED=0
CHECKS_WARNING=0

# ============================================================================
# Helper Functions
# ============================================================================

print_header() {
  printf "\n${COLOR_BLUE}=================================================================\n"
  printf "%s\n" "$1"
  printf "=================================================================${COLOR_RESET}\n\n"
}

check_pass() {
  printf "${COLOR_GREEN}✓${COLOR_RESET} %s\n" "$1"
  CHECKS_PASSED=$((CHECKS_PASSED + 1))
}

check_fail() {
  printf "${COLOR_RED}✗${COLOR_RESET} %s\n" "$1"
  CHECKS_FAILED=$((CHECKS_FAILED + 1))
}

check_warn() {
  printf "${COLOR_YELLOW}⚠${COLOR_RESET} %s\n" "$1"
  CHECKS_WARNING=$((CHECKS_WARNING + 1))
}

# ============================================================================
# Verification Checks
# ============================================================================

verify_test_files() {
  print_header "Verifying Test Files"

  local required_tests=(
    "test-full-deployment.sh"
    "test-multi-tenant-workflow.sh"
    "test-backup-restore-workflow.sh"
    "test-migration-workflow.sh"
    "test-monitoring-stack.sh"
    "test-custom-services-workflow.sh"
  )

  for test_file in "${required_tests[@]}"; do
    if [[ -f "$TEST_DIR/$test_file" ]]; then
      check_pass "Test file exists: $test_file"
    else
      check_fail "Test file missing: $test_file"
    fi
  done
}

verify_permissions() {
  print_header "Verifying File Permissions"

  local test_files=("$TEST_DIR"/test-*.sh)
  local all_executable=true

  for test_file in "${test_files[@]}"; do
    if [[ -x "$test_file" ]]; then
      check_pass "Executable: $(basename "$test_file")"
    else
      check_fail "Not executable: $(basename "$test_file")"
      all_executable=false
    fi
  done

  # Check master runner
  if [[ -x "$TEST_DIR/run-all-integration-tests.sh" ]]; then
    check_pass "Master runner is executable"
  else
    check_fail "Master runner not executable"
  fi

  # Check helpers
  if [[ -x "$TEST_DIR/utils/integration-helpers.sh" ]]; then
    check_pass "Integration helpers are executable"
  else
    check_fail "Integration helpers not executable"
  fi
}

verify_utilities() {
  print_header "Verifying Utility Files"

  # Check utils directory
  if [[ -d "$TEST_DIR/utils" ]]; then
    check_pass "Utils directory exists"
  else
    check_fail "Utils directory missing"
  fi

  # Check integration helpers
  if [[ -f "$TEST_DIR/utils/integration-helpers.sh" ]]; then
    check_pass "Integration helpers exist"

    # Verify key functions exist
    if grep -q "setup_test_project()" "$TEST_DIR/utils/integration-helpers.sh"; then
      check_pass "setup_test_project() function found"
    else
      check_fail "setup_test_project() function missing"
    fi

    if grep -q "cleanup_test_project()" "$TEST_DIR/utils/integration-helpers.sh"; then
      check_pass "cleanup_test_project() function found"
    else
      check_fail "cleanup_test_project() function missing"
    fi

    if grep -q "wait_for_service_healthy()" "$TEST_DIR/utils/integration-helpers.sh"; then
      check_pass "wait_for_service_healthy() function found"
    else
      check_fail "wait_for_service_healthy() function missing"
    fi
  else
    check_fail "Integration helpers missing"
  fi
}

verify_documentation() {
  print_header "Verifying Documentation"

  local docs=(
    "README.md"
    "INTEGRATION-TEST-SUMMARY.md"
    "QUICK-START.md"
  )

  for doc in "${docs[@]}"; do
    if [[ -f "$TEST_DIR/$doc" ]]; then
      check_pass "Documentation exists: $doc"
    else
      check_fail "Documentation missing: $doc"
    fi
  done
}

verify_ci_workflow() {
  print_header "Verifying CI/CD Configuration"

  local workflow_file="$TEST_DIR/../../.github/workflows/integration-tests.yml"

  if [[ -f "$workflow_file" ]]; then
    check_pass "CI workflow file exists"

    # Check workflow contains required jobs
    if grep -q "integration-tests:" "$workflow_file"; then
      check_pass "integration-tests job defined"
    else
      check_fail "integration-tests job missing"
    fi

    if grep -q "test-summary:" "$workflow_file"; then
      check_pass "test-summary job defined"
    else
      check_warn "test-summary job missing (optional)"
    fi
  else
    check_fail "CI workflow file missing"
  fi
}

verify_dependencies() {
  print_header "Verifying System Dependencies"

  # Check Docker
  if command -v docker >/dev/null 2>&1; then
    check_pass "Docker is installed"

    if docker ps >/dev/null 2>&1; then
      check_pass "Docker daemon is running"
    else
      check_fail "Docker daemon is not running"
    fi
  else
    check_fail "Docker is not installed"
  fi

  # Check Docker Compose
  if command -v docker-compose >/dev/null 2>&1; then
    check_pass "Docker Compose is installed"
  else
    check_fail "Docker Compose is not installed"
  fi

  # Check nself
  if command -v nself >/dev/null 2>&1; then
    check_pass "nself is in PATH"

    local nself_version
    nself_version=$(nself --version 2>&1 | head -1 || echo "unknown")
    printf "  Version: %s\n" "$nself_version"
  else
    check_warn "nself not in PATH (tests will use NSELF_ROOT)"
  fi

  # Check test framework
  local test_framework="$TEST_DIR/../test_framework.sh"
  if [[ -f "$test_framework" ]]; then
    check_pass "Test framework exists"
  else
    check_fail "Test framework missing"
  fi
}

verify_resources() {
  print_header "Verifying System Resources"

  # Check available disk space
  local available_space
  if command -v df >/dev/null 2>&1; then
    available_space=$(df -h . | awk 'NR==2 {print $4}')
    printf "Available disk space: %s\n" "$available_space"

    # Parse GB (rough check)
    local space_gb
    space_gb=$(echo "$available_space" | grep -o '[0-9]\+' | head -1 || echo "0")

    if [[ $space_gb -ge 20 ]]; then
      check_pass "Sufficient disk space (${space_gb}GB available, 20GB recommended)"
    elif [[ $space_gb -ge 10 ]]; then
      check_warn "Limited disk space (${space_gb}GB available, 20GB recommended)"
    else
      check_fail "Insufficient disk space (${space_gb}GB available, 20GB required)"
    fi
  else
    check_warn "Cannot check disk space"
  fi

  # Check available memory
  if command -v free >/dev/null 2>&1; then
    local available_mem
    available_mem=$(free -h | awk 'NR==2 {print $7}')
    printf "Available memory: %s\n" "$available_mem"
  elif command -v vm_stat >/dev/null 2>&1; then
    # macOS
    local free_pages
    free_pages=$(vm_stat | grep "Pages free" | awk '{print $3}' | tr -d '.')
    local available_gb=$((free_pages * 4096 / 1024 / 1024 / 1024))
    printf "Available memory: ~%dGB\n" "$available_gb"

    if [[ $available_gb -ge 8 ]]; then
      check_pass "Sufficient memory (${available_gb}GB available, 8GB recommended)"
    elif [[ $available_gb -ge 4 ]]; then
      check_warn "Limited memory (${available_gb}GB available, 8GB recommended)"
    else
      check_fail "Insufficient memory (${available_gb}GB available, 4GB required)"
    fi
  else
    check_warn "Cannot check available memory"
  fi
}

verify_test_structure() {
  print_header "Verifying Test Structure"

  # Count test cases in each file
  local test_files=(
    "test-full-deployment.sh:14"
    "test-multi-tenant-workflow.sh:10"
    "test-backup-restore-workflow.sh:11"
    "test-migration-workflow.sh:11"
    "test-monitoring-stack.sh:11"
    "test-custom-services-workflow.sh:13"
  )

  for entry in "${test_files[@]}"; do
    local file="${entry%:*}"
    local expected_tests="${entry#*:}"

    if [[ -f "$TEST_DIR/$file" ]]; then
      local actual_tests
      actual_tests=$(grep -c "^test_[0-9]\+_" "$TEST_DIR/$file" || echo "0")

      if [[ $actual_tests -eq $expected_tests ]]; then
        check_pass "$file has $actual_tests test cases"
      else
        check_warn "$file has $actual_tests tests (expected $expected_tests)"
      fi
    fi
  done
}

# ============================================================================
# Summary
# ============================================================================

print_summary() {
  print_header "Verification Summary"

  local total_checks=$((CHECKS_PASSED + CHECKS_FAILED + CHECKS_WARNING))

  printf "Total Checks: %d\n" "$total_checks"
  printf "${COLOR_GREEN}Passed: %d${COLOR_RESET}\n" "$CHECKS_PASSED"
  printf "${COLOR_RED}Failed: %d${COLOR_RESET}\n" "$CHECKS_FAILED"
  printf "${COLOR_YELLOW}Warnings: %d${COLOR_RESET}\n\n" "$CHECKS_WARNING"

  if [[ $CHECKS_FAILED -eq 0 ]]; then
    printf "${COLOR_GREEN}✓ Integration test suite is properly configured!${COLOR_RESET}\n\n"
    printf "Next steps:\n"
    printf "  1. Run all tests: ./run-all-integration-tests.sh\n"
    printf "  2. Run specific test: ./test-full-deployment.sh\n"
    printf "  3. Check documentation: cat README.md\n\n"
    return 0
  else
    printf "${COLOR_RED}✗ Integration test suite has issues that need attention.${COLOR_RESET}\n\n"
    printf "Fix the failed checks above before running tests.\n\n"
    return 1
  fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
  print_header "Integration Test Suite Verification"
  printf "Test Directory: %s\n" "$TEST_DIR"

  verify_test_files
  verify_permissions
  verify_utilities
  verify_documentation
  verify_ci_workflow
  verify_dependencies
  verify_resources
  verify_test_structure

  print_summary
}

# Run verification
main "$@"
