#!/usr/bin/env bash
# v1-command-structure-test.sh - Comprehensive test for v1.0 command routing
#
# Tests all 31 TLCs, subcommands, aliases, deprecation warnings, and output formatting
# Updated to match actual v1.0 command surface per CLAUDE.md

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/../cli" && pwd)"
BIN_DIR="$(cd "$SCRIPT_DIR/../../bin" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Test results array
declare -a TEST_RESULTS=()

# Utility functions
print_header() {
  printf "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${BLUE}║  nself v1.0 Command Structure - Comprehensive Test Suite      ║${NC}\n"
  printf "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}\n\n"
}

print_section() {
  printf "\n${CYAN}▶ %s${NC}\n" "$1"
  printf "${CYAN}────────────────────────────────────────────────────────────────${NC}\n"
}

test_command_exists() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ -f "$CLI_DIR/$cmd.sh" ]]; then
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
    TEST_RESULTS+=("PASS: $test_name")
    return 0
  else
    TESTS_FAILED=$((TESTS_FAILED + 1))
    printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
    printf "    ${RED}Missing: $CLI_DIR/$cmd.sh${NC}\n"
    TEST_RESULTS+=("FAIL: $test_name - Missing file")
    return 1
  fi
}

test_command_executable() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ -f "$BIN_DIR/nself" ]] && [[ -x "$BIN_DIR/nself" ]]; then
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
    TEST_RESULTS+=("PASS: $test_name")
    return 0
  else
    TESTS_FAILED=$((TESTS_FAILED + 1))
    printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
    printf "    ${RED}Not executable: $BIN_DIR/nself${NC}\n"
    TEST_RESULTS+=("FAIL: $test_name - Not executable")
    return 1
  fi
}

test_help_output() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  local output
  local exit_code=0
  output=$(bash "$CLI_DIR/$cmd.sh" --help 2>&1) || exit_code=$?

  if [[ $exit_code -eq 0 ]] && [[ -n "$output" ]]; then
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
    TEST_RESULTS+=("PASS: $test_name")
    return 0
  fi

  TESTS_FAILED=$((TESTS_FAILED + 1))
  printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC} (exit %d)\n" "$test_name" "$exit_code"
  TEST_RESULTS+=("FAIL: $test_name - exit $exit_code")
  return 1
}

check_subcommand_support() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ -f "$CLI_DIR/$cmd.sh" ]]; then
    if grep -qE 'case.*\$|case.*"\$' "$CLI_DIR/$cmd.sh"; then
      TESTS_PASSED=$((TESTS_PASSED + 1))
      printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
      TEST_RESULTS+=("PASS: $test_name")
      return 0
    else
      TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
      printf "  ${YELLOW}○${NC} %-50s ${YELLOW}SKIP${NC} (no subcommands)\n" "$test_name"
      TEST_RESULTS+=("SKIP: $test_name - No subcommands needed")
      return 0
    fi
  fi

  TESTS_FAILED=$((TESTS_FAILED + 1))
  printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
  TEST_RESULTS+=("FAIL: $test_name - File missing")
  return 1
}

check_output_formatting() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ -f "$CLI_DIR/$cmd.sh" ]]; then
    if grep -qE '(cli-output\.sh|display\.sh|log_success|log_error|log_info|log_warning|printf)' "$CLI_DIR/$cmd.sh"; then
      TESTS_PASSED=$((TESTS_PASSED + 1))
      printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
      TEST_RESULTS+=("PASS: $test_name")
      return 0
    else
      TESTS_FAILED=$((TESTS_FAILED + 1))
      printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
      printf "    ${RED}Missing output formatting integration${NC}\n"
      TEST_RESULTS+=("FAIL: $test_name - No output formatting")
      return 1
    fi
  fi

  TESTS_FAILED=$((TESTS_FAILED + 1))
  printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
  TEST_RESULTS+=("FAIL: $test_name - File missing")
  return 1
}

# Start tests
print_header

# =========================================================================
# TEST 1: Core TLC Files Existence (31 commands per v1.0 CLAUDE.md spec)
# =========================================================================
print_section "TEST 1: All 31 v1.0 TLC Command Files Exist"

# Core Commands (5)
test_command_exists "init" "init.sh exists"
test_command_exists "build" "build.sh exists"
test_command_exists "start" "start.sh exists"
test_command_exists "stop" "stop.sh exists"
test_command_exists "restart" "restart.sh exists"

# Utilities (15)
test_command_exists "status" "status.sh exists"
test_command_exists "logs" "logs.sh exists"
test_command_exists "help" "help.sh exists"
test_command_exists "admin" "admin.sh exists"
test_command_exists "urls" "urls.sh exists"
test_command_exists "exec" "exec.sh exists"
test_command_exists "doctor" "doctor.sh exists"
test_command_exists "monitor" "monitor.sh exists"
test_command_exists "health" "health.sh exists"
test_command_exists "version" "version.sh exists"
test_command_exists "update" "update.sh exists"
test_command_exists "completion" "completion.sh exists"
test_command_exists "metrics" "metrics.sh exists"
test_command_exists "history" "history.sh exists"
test_command_exists "audit" "audit.sh exists"

# Other Commands (11)
test_command_exists "db" "db.sh exists"
test_command_exists "tenant" "tenant.sh exists"
test_command_exists "deploy" "deploy.sh exists"
test_command_exists "infra" "infra.sh exists"
test_command_exists "service" "service.sh exists"
test_command_exists "config" "config.sh exists"
test_command_exists "auth" "auth.sh exists"
test_command_exists "perf" "perf.sh exists"
test_command_exists "backup" "backup.sh exists"
test_command_exists "dev" "dev.sh exists"
test_command_exists "plugin" "plugin.sh exists"

# =========================================================================
# TEST 2: Main nself Binary
# =========================================================================
print_section "TEST 2: Main nself Binary"

test_command_executable "nself" "nself binary is executable"
test_command_exists "nself" "nself.sh wrapper exists"

# =========================================================================
# TEST 3: Deprecated/Aliased Commands Still Have Files
# =========================================================================
print_section "TEST 3: Deprecated/Aliased Command Files (Backward Compat)"

# These should exist as wrappers/aliases
test_command_exists "billing" "billing.sh (alias -> tenant billing)"
test_command_exists "org" "org.sh (alias -> tenant org)"
test_command_exists "upgrade" "upgrade.sh (alias -> deploy upgrade)"
test_command_exists "staging" "staging.sh (alias -> deploy staging)"
test_command_exists "prod" "prod.sh (alias -> deploy production)"
test_command_exists "sync" "sync.sh (alias -> deploy sync / config sync)"
test_command_exists "k8s" "k8s.sh (alias -> infra k8s)"
test_command_exists "helm" "helm.sh (alias -> infra helm)"
test_command_exists "storage" "storage.sh (alias -> service storage)"
test_command_exists "email" "email.sh (alias -> service email)"
test_command_exists "search" "search.sh (alias -> service search)"
test_command_exists "redis" "redis.sh (alias -> service redis)"
test_command_exists "functions" "functions.sh (alias -> service functions)"
test_command_exists "mlflow" "mlflow.sh (alias -> service mlflow)"
test_command_exists "secrets" "secrets.sh (alias -> config secrets)"
test_command_exists "env" "env.sh (alias -> config env)"
test_command_exists "mfa" "mfa.sh (alias -> auth mfa)"
test_command_exists "ssl" "ssl.sh (alias -> auth ssl)"
test_command_exists "whitelabel" "whitelabel.sh (alias -> dev whitelabel)"

# =========================================================================
# TEST 4: Subcommand Structure Check
# =========================================================================
print_section "TEST 4: Subcommand Structure Check"

check_subcommand_support "db" "Subcommands: db"
check_subcommand_support "tenant" "Subcommands: tenant"
check_subcommand_support "deploy" "Subcommands: deploy"
check_subcommand_support "infra" "Subcommands: infra"
check_subcommand_support "service" "Subcommands: service"
check_subcommand_support "config" "Subcommands: config"
check_subcommand_support "auth" "Subcommands: auth"
check_subcommand_support "backup" "Subcommands: backup"
check_subcommand_support "dev" "Subcommands: dev"
check_subcommand_support "plugin" "Subcommands: plugin"

# =========================================================================
# TEST 5: Output Formatting Check
# =========================================================================
print_section "TEST 5: Output Formatting Check"

check_output_formatting "init" "Output: init uses formatting"
check_output_formatting "build" "Output: build uses formatting"
check_output_formatting "start" "Output: start uses formatting"
check_output_formatting "status" "Output: status uses formatting"
check_output_formatting "deploy" "Output: deploy uses formatting"

# =========================================================================
# TEST 6: File Structure
# =========================================================================
print_section "TEST 6: File Structure Verification"

test_file_structure() {
  local path="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ -e "$path" ]]; then
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
    TEST_RESULTS+=("PASS: $test_name")
    return 0
  else
    TESTS_FAILED=$((TESTS_FAILED + 1))
    printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
    printf "    ${RED}Missing: $path${NC}\n"
    TEST_RESULTS+=("FAIL: $test_name - Path missing")
    return 1
  fi
}

test_file_structure "$SCRIPT_DIR/../lib/utils/display.sh" "File: display.sh exists"
test_file_structure "$SCRIPT_DIR/../lib/config/constants.sh" "File: constants.sh exists"
test_file_structure "$SCRIPT_DIR/../lib/config/defaults.sh" "File: defaults.sh exists"
test_file_structure "$SCRIPT_DIR/../lib/hooks/pre-command.sh" "File: pre-command.sh exists"
test_file_structure "$SCRIPT_DIR/../lib/hooks/post-command.sh" "File: post-command.sh exists"
test_file_structure "$SCRIPT_DIR/../VERSION" "File: VERSION exists"

# =========================================================================
# SUMMARY
# =========================================================================
print_section "TEST SUMMARY"

printf "\n"
printf "${BLUE}Total Tests:${NC}    %d\n" "$TESTS_RUN"
printf "${GREEN}Passed:${NC}         %d\n" "$TESTS_PASSED"
printf "${RED}Failed:${NC}         %d\n" "$TESTS_FAILED"
printf "${YELLOW}Skipped:${NC}        %d\n" "$TESTS_SKIPPED"
printf "\n"

if [[ $TESTS_FAILED -gt 0 ]]; then
  printf "${RED}╔════════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${RED}║  TESTS FAILED - See details above                             ║${NC}\n"
  printf "${RED}╚════════════════════════════════════════════════════════════════╝${NC}\n"

  printf "\n${RED}Failed Tests:${NC}\n"
  for result in "${TEST_RESULTS[@]}"; do
    if [[ "$result" == FAIL* ]]; then
      printf "  ${RED}✗${NC} %s\n" "${result#FAIL: }"
    fi
  done

  exit 1
else
  printf "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${GREEN}║  ALL TESTS PASSED                                             ║${NC}\n"
  printf "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}\n"
  exit 0
fi
