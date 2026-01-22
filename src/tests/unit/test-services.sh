#!/usr/bin/env bash
# test-services.sh - Unit tests for service management commands (v0.4.2)
# Tests: email, search, functions, mlflow, metrics, monitor

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# ROOT_DIR is the nself project root (parent of src/)
ROOT_DIR="$(dirname "$(dirname "$(dirname "$SCRIPT_DIR")")")"

# Colors for output (cross-platform compatible)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
log_test() {
  printf "${YELLOW}TEST:${NC} %s\n" "$1"
}

pass() {
  TESTS_PASSED=$((TESTS_PASSED + 1))
  printf "  ${GREEN}✓${NC} %s\n" "$1"
}

fail() {
  TESTS_FAILED=$((TESTS_FAILED + 1))
  printf "  ${RED}✗${NC} %s\n" "$1"
}

run_test() {
  TESTS_RUN=$((TESTS_RUN + 1))
}

# Test that a file exists
assert_file_exists() {
  local file="$1"
  if [[ -f "$file" ]]; then
    pass "File exists: $(basename "$file")"
    return 0
  else
    fail "File missing: $file"
    return 1
  fi
}

# Test that a file contains a pattern
assert_file_contains() {
  local file="$1"
  local pattern="$2"
  local description="${3:-Pattern found}"
  if grep -q "$pattern" "$file" 2>/dev/null; then
    pass "$description"
    return 0
  else
    fail "$description (pattern not found: $pattern)"
    return 1
  fi
}

# Test that file does NOT contain a pattern (for compatibility checks)
assert_file_not_contains() {
  local file="$1"
  local pattern="$2"
  local description="${3:-Pattern not found}"
  if ! grep -q "$pattern" "$file" 2>/dev/null; then
    pass "$description"
    return 0
  else
    fail "$description (pattern found: $pattern)"
    return 1
  fi
}

# Test that a function is defined in a file
assert_function_exists() {
  local file="$1"
  local func="$2"
  if grep -qE "^${func}\(\)|^function ${func}" "$file" 2>/dev/null; then
    pass "Function exists: $func"
    return 0
  else
    fail "Function missing: $func in $(basename "$file")"
    return 1
  fi
}

# ============================================================================
# EMAIL COMMAND TESTS
# ============================================================================
test_email_command() {
  log_test "Email Command Tests"
  local file="$ROOT_DIR/src/cli/email.sh"

  run_test
  assert_file_exists "$file"

  run_test
  assert_function_exists "$file" "email_main"

  run_test
  assert_function_exists "$file" "validate_config"

  run_test
  assert_function_exists "$file" "smtp_preflight_check"

  run_test
  assert_function_exists "$file" "test_email"

  # Cross-platform compatibility
  run_test
  assert_file_not_contains "$file" 'echo -e' "No echo -e usage"

  # Check for provider templates
  run_test
  assert_file_contains "$file" "sendgrid" "SendGrid provider support"

  run_test
  assert_file_contains "$file" "aws-ses" "AWS SES provider support"

  run_test
  assert_file_contains "$file" "mailgun" "Mailgun provider support"

  echo ""
}

# ============================================================================
# SEARCH COMMAND TESTS
# ============================================================================
test_search_command() {
  log_test "Search Command Tests"
  local file="$ROOT_DIR/src/cli/search.sh"

  run_test
  assert_file_exists "$file"

  run_test
  assert_function_exists "$file" "search_enable"

  run_test
  assert_function_exists "$file" "search_status"

  run_test
  assert_function_exists "$file" "search_test"

  # Check for all search engines
  run_test
  assert_file_contains "$file" "postgres" "PostgreSQL search support"

  run_test
  assert_file_contains "$file" "meilisearch" "MeiliSearch support"

  run_test
  assert_file_contains "$file" "typesense" "Typesense support"

  run_test
  assert_file_contains "$file" "elasticsearch" "Elasticsearch support"

  run_test
  assert_file_contains "$file" "opensearch" "OpenSearch support"

  run_test
  assert_file_contains "$file" "sonic" "Sonic support"

  # Cross-platform compatibility
  run_test
  assert_file_not_contains "$file" 'echo -e' "No echo -e usage"

  run_test
  assert_file_contains "$file" "safe_sed_inline" "Uses safe_sed_inline"

  echo ""
}

# ============================================================================
# FUNCTIONS COMMAND TESTS
# ============================================================================
test_functions_command() {
  log_test "Functions Command Tests"
  local file="$ROOT_DIR/src/cli/functions.sh"

  run_test
  assert_file_exists "$file"

  run_test
  assert_function_exists "$file" "cmd_functions"

  run_test
  assert_function_exists "$file" "functions_create"

  run_test
  assert_function_exists "$file" "functions_deploy"

  run_test
  assert_function_exists "$file" "deploy_functions_local"

  run_test
  assert_function_exists "$file" "deploy_functions_production"

  run_test
  assert_function_exists "$file" "validate_functions"

  run_test
  assert_function_exists "$file" "create_typescript_function"

  # Check for function templates
  run_test
  assert_file_contains "$file" "basic" "Basic template"

  run_test
  assert_file_contains "$file" "webhook" "Webhook template"

  run_test
  assert_file_contains "$file" "api" "API template"

  run_test
  assert_file_contains "$file" "scheduled" "Scheduled template"

  # TypeScript support
  run_test
  assert_file_contains "$file" 'typescript' "TypeScript flag support"

  # Cross-platform compatibility
  run_test
  assert_file_not_contains "$file" 'echo -e' "No echo -e usage"

  run_test
  assert_file_contains "$file" "safe_sed_inline" "Uses safe_sed_inline"

  echo ""
}

# ============================================================================
# MLFLOW COMMAND TESTS
# ============================================================================
test_mlflow_command() {
  log_test "MLflow Command Tests"
  local file="$ROOT_DIR/src/cli/mlflow.sh"

  run_test
  assert_file_exists "$file"

  run_test
  assert_function_exists "$file" "cmd_mlflow"

  run_test
  assert_function_exists "$file" "mlflow_enable"

  run_test
  assert_function_exists "$file" "mlflow_status"

  run_test
  assert_function_exists "$file" "mlflow_test"

  run_test
  assert_function_exists "$file" "mlflow_experiments"

  run_test
  assert_function_exists "$file" "mlflow_runs"

  # Experiments subcommands
  run_test
  assert_file_contains "$file" "experiments create" "Experiments create"

  run_test
  assert_file_contains "$file" "experiments delete" "Experiments delete"

  # Cross-platform compatibility
  run_test
  assert_file_not_contains "$file" 'echo -e' "No echo -e usage"

  run_test
  assert_file_contains "$file" "safe_sed_inline" "Uses safe_sed_inline"

  echo ""
}

# ============================================================================
# METRICS COMMAND TESTS
# ============================================================================
test_metrics_command() {
  log_test "Metrics Command Tests"
  local file="$ROOT_DIR/src/cli/metrics.sh"

  run_test
  assert_file_exists "$file"

  run_test
  assert_function_exists "$file" "cmd_metrics"

  run_test
  assert_function_exists "$file" "enable_monitoring"

  run_test
  assert_function_exists "$file" "disable_monitoring"

  run_test
  assert_function_exists "$file" "show_monitoring_status"

  run_test
  assert_function_exists "$file" "manage_monitoring_profile"

  run_test
  assert_function_exists "$file" "configure_monitoring"

  # Check for monitoring profiles
  run_test
  assert_file_contains "$file" "minimal" "Minimal profile"

  run_test
  assert_file_contains "$file" "standard" "Standard profile"

  run_test
  assert_file_contains "$file" "full" "Full profile"

  # Cross-platform compatibility
  run_test
  assert_file_not_contains "$file" 'echo -e' "No echo -e usage"

  run_test
  assert_file_contains "$file" "safe_sed_inline" "Uses safe_sed_inline"

  echo ""
}

# ============================================================================
# MONITOR COMMAND TESTS
# ============================================================================
test_monitor_command() {
  log_test "Monitor Command Tests"
  local file="$ROOT_DIR/src/cli/monitor.sh"

  run_test
  assert_file_exists "$file"

  run_test
  assert_function_exists "$file" "cmd_monitor"

  run_test
  assert_function_exists "$file" "open_grafana_dashboard"

  run_test
  assert_function_exists "$file" "open_prometheus_ui"

  run_test
  assert_function_exists "$file" "show_service_status"

  run_test
  assert_function_exists "$file" "show_resource_usage"

  run_test
  assert_function_exists "$file" "color_text"

  # Cross-platform compatibility - color_text should use printf
  run_test
  assert_file_not_contains "$file" 'echo -e "\033' "No echo -e for colors"

  run_test
  assert_file_contains "$file" 'printf.*\\033' "Uses printf for colors"

  echo ""
}

# ============================================================================
# PLATFORM COMPATIBILITY TESTS
# ============================================================================
test_platform_compatibility() {
  log_test "Platform Compatibility Tests"
  local cli_dir="$ROOT_DIR/src/cli"

  # Check all 6 command files for cross-platform issues
  for file in email.sh search.sh functions.sh mlflow.sh metrics.sh monitor.sh; do
    local filepath="$cli_dir/$file"
    if [[ -f "$filepath" ]]; then
      run_test
      if ! grep -q 'echo -e' "$filepath" 2>/dev/null; then
        pass "$file: No echo -e usage"
      else
        fail "$file: Contains echo -e (not portable)"
      fi

      run_test
      if ! grep -q '\${[^}]*,,}' "$filepath" 2>/dev/null; then
        pass "$file: No Bash 4+ lowercase expansion"
      else
        fail "$file: Contains \${var,,} (Bash 4+ only)"
      fi

      run_test
      if ! grep -q '\${[^}]*\^\^}' "$filepath" 2>/dev/null; then
        pass "$file: No Bash 4+ uppercase expansion"
      else
        fail "$file: Contains \${var^^} (Bash 4+ only)"
      fi

      run_test
      if ! grep -q 'declare -A' "$filepath" 2>/dev/null; then
        pass "$file: No associative arrays"
      else
        fail "$file: Contains declare -A (Bash 4+ only)"
      fi
    fi
  done

  echo ""
}

# ============================================================================
# RUN ALL TESTS
# ============================================================================
main() {
  echo ""
  printf "${YELLOW}═══════════════════════════════════════════════════════════════${NC}\n"
  printf "${YELLOW}   nself v0.4.2 - Service Commands Unit Tests${NC}\n"
  printf "${YELLOW}═══════════════════════════════════════════════════════════════${NC}\n"
  echo ""

  # Run test suites
  test_email_command
  test_search_command
  test_functions_command
  test_mlflow_command
  test_metrics_command
  test_monitor_command
  test_platform_compatibility

  # Summary
  printf "${YELLOW}═══════════════════════════════════════════════════════════════${NC}\n"
  echo ""
  echo "Test Summary:"
  echo "  Total:  $TESTS_RUN"
  printf "  ${GREEN}Passed: $TESTS_PASSED${NC}\n"
  if [[ $TESTS_FAILED -gt 0 ]]; then
    printf "  ${RED}Failed: $TESTS_FAILED${NC}\n"
  else
    echo "  Failed: 0"
  fi
  echo ""

  if [[ $TESTS_FAILED -gt 0 ]]; then
    printf "${RED}Some tests failed!${NC}\n"
    exit 1
  else
    printf "${GREEN}All tests passed!${NC}\n"
    exit 0
  fi
}

# Run main
main "$@"
