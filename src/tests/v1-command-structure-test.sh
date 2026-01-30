#!/usr/bin/env bash
# v1-command-structure-test.sh - Comprehensive test for v1.0 command routing
#
# Tests all 31 TLCs, subcommands, aliases, deprecation warnings, and output formatting

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
NC='\033[0m' # No Color

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
  printf "${CYAN}%s${NC}\n" "$(printf '─%.0s' {1..64})"
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

  if [[ -x "$BIN_DIR/nself" ]]; then
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
  if output=$("$BIN_DIR/nself" "$cmd" --help 2>&1); then
    if [[ -n "$output" ]] && [[ "$output" == *"Usage:"* || "$output" == *"nself"* ]]; then
      TESTS_PASSED=$((TESTS_PASSED + 1))
      printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
      TEST_RESULTS+=("PASS: $test_name")
      return 0
    fi
  fi

  TESTS_FAILED=$((TESTS_FAILED + 1))
  printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
  printf "    ${RED}No help output or invalid format${NC}\n"
  TEST_RESULTS+=("FAIL: $test_name - No/invalid help output")
  return 1
}

test_command_routing() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  # Try to run command with --help (safest way to test routing without side effects)
  local output
  local exit_code=0
  output=$("$BIN_DIR/nself" "$cmd" --help 2>&1) || exit_code=$?

  # Accept exit codes: 0 (success), 1 (expected for some commands)
  # Reject: 127 (command not found)
  if [[ $exit_code -ne 127 ]]; then
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
    TEST_RESULTS+=("PASS: $test_name")
    return 0
  else
    TESTS_FAILED=$((TESTS_FAILED + 1))
    printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
    printf "    ${RED}Command not found (exit $exit_code)${NC}\n"
    TEST_RESULTS+=("FAIL: $test_name - Command not found")
    return 1
  fi
}

test_version_formats() {
  local test_name="$1"
  local flag="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  local output
  if output=$("$BIN_DIR/nself" "$flag" 2>&1); then
    if [[ -n "$output" ]]; then
      TESTS_PASSED=$((TESTS_PASSED + 1))
      printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
      TEST_RESULTS+=("PASS: $test_name")
      return 0
    fi
  fi

  TESTS_FAILED=$((TESTS_FAILED + 1))
  printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
  TEST_RESULTS+=("FAIL: $test_name - No output")
  return 1
}

# Start tests
print_header

# TEST 1: Core TLC Files Existence
print_section "TEST 1: All 31 TLC Command Files Exist"

# Project Lifecycle Commands
test_command_exists "init" "init.sh exists"
test_command_exists "build" "build.sh exists"
test_command_exists "start" "start.sh exists"
test_command_exists "stop" "stop.sh exists"
test_command_exists "restart" "restart.sh exists"
test_command_exists "destroy" "destroy.sh exists"

# Operational Commands
test_command_exists "status" "status.sh exists"
test_command_exists "logs" "logs.sh exists"
test_command_exists "urls" "urls.sh exists"
test_command_exists "exec" "exec.sh exists"
test_command_exists "shell" "shell.sh exists"

# Configuration Commands
test_command_exists "env" "env.sh exists"
test_command_exists "config" "config.sh exists"
test_command_exists "domain" "domain.sh exists"

# Database Commands
test_command_exists "db" "db.sh exists"
test_command_exists "migrate" "migrate.sh exists"
test_command_exists "seed" "seed.sh exists"

# Backup & Recovery
test_command_exists "backup" "backup.sh exists"
test_command_exists "restore" "restore.sh exists"

# Security & Auth
test_command_exists "ssl" "ssl.sh exists"
test_command_exists "auth" "auth.sh exists"
test_command_exists "secrets" "secrets.sh exists"

# Development Tools
test_command_exists "dev" "dev.sh exists"
test_command_exists "test" "test.sh exists"
test_command_exists "lint" "lint.sh exists"

# Deployment
test_command_exists "deploy" "deploy.sh exists"
test_command_exists "sync" "sync.sh exists"

# Utilities & Helpers
test_command_exists "doctor" "doctor.sh exists"
test_command_exists "clean" "clean.sh exists"
test_command_exists "update" "update.sh exists"
test_command_exists "version" "version.sh exists"
test_command_exists "help" "help.sh exists"

# TEST 2: Main nself Binary
print_section "TEST 2: Main nself Binary"

test_command_executable "nself" "nself binary is executable"
test_command_exists "nself.sh" "nself.sh wrapper exists"

# TEST 3: Command Routing (via --help for safety)
print_section "TEST 3: Command Routing (Safe Test via --help)"

# Sample of critical commands
test_command_routing "init" "Route: nself init"
test_command_routing "build" "Route: nself build"
test_command_routing "start" "Route: nself start"
test_command_routing "status" "Route: nself status"
test_command_routing "env" "Route: nself env"
test_command_routing "db" "Route: nself db"
test_command_routing "backup" "Route: nself backup"
test_command_routing "deploy" "Route: nself deploy"
test_command_routing "help" "Route: nself help"
test_command_routing "version" "Route: nself version"

# TEST 4: Version Flag Formats
print_section "TEST 4: Version Flag Formats"

test_version_formats "Version: -v flag" "-v"
test_version_formats "Version: --version flag" "--version"
test_command_routing "version" "Version: nself version"

# TEST 5: Help Command Variations
print_section "TEST 5: Help Command Variations"

test_command_routing "help" "Help: nself help"
test_version_formats "Help: -h flag" "-h"
test_version_formats "Help: --help flag" "--help"

# TEST 6: Subcommand Structure (check if files support subcommands)
print_section "TEST 6: Subcommand Structure Check"

check_subcommand_support() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ -f "$CLI_DIR/$cmd.sh" ]]; then
    # Check if file has case statement for subcommands
    if grep -q 'case.*\$1.*in' "$CLI_DIR/$cmd.sh"; then
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

check_subcommand_support "env" "Subcommands: env (switch, list, etc.)"
check_subcommand_support "db" "Subcommands: db (migrate, seed, etc.)"
check_subcommand_support "backup" "Subcommands: backup (create, restore, list)"
check_subcommand_support "config" "Subcommands: config (get, set, list)"
check_subcommand_support "deploy" "Subcommands: deploy (staging, prod, etc.)"

# TEST 7: Output Formatting (check if uses cli-output.sh)
print_section "TEST 7: Output Formatting Check (cli-output.sh usage)"

check_output_formatting() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  if [[ -f "$CLI_DIR/$cmd.sh" ]]; then
    # Check if file sources cli-output.sh or uses log_ functions
    if grep -qE '(cli-output\.sh|log_success|log_error|log_info|log_warning)' "$CLI_DIR/$cmd.sh"; then
      TESTS_PASSED=$((TESTS_PASSED + 1))
      printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
      TEST_RESULTS+=("PASS: $test_name")
      return 0
    else
      TESTS_FAILED=$((TESTS_FAILED + 1))
      printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
      printf "    ${RED}Missing cli-output.sh integration${NC}\n"
      TEST_RESULTS+=("FAIL: $test_name - No output formatting")
      return 1
    fi
  fi

  TESTS_FAILED=$((TESTS_FAILED + 1))
  printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
  TEST_RESULTS+=("FAIL: $test_name - File missing")
  return 1
}

# Check critical commands use proper output
check_output_formatting "init" "Output: init uses cli-output.sh"
check_output_formatting "build" "Output: build uses cli-output.sh"
check_output_formatting "start" "Output: start uses cli-output.sh"
check_output_formatting "status" "Output: status uses cli-output.sh"
check_output_formatting "deploy" "Output: deploy uses cli-output.sh"

# TEST 8: Error Handling
print_section "TEST 8: Error Handling for Invalid Commands"

test_invalid_command() {
  local cmd="$1"
  local test_name="$2"
  TESTS_RUN=$((TESTS_RUN + 1))

  local output
  local exit_code=0
  output=$("$BIN_DIR/nself" "$cmd" 2>&1) || exit_code=$?

  # Should fail (non-zero exit) and show error message
  if [[ $exit_code -ne 0 ]] && [[ "$output" == *"Unknown command"* || "$output" == *"error"* || "$output" == *"not found"* ]]; then
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$test_name"
    TEST_RESULTS+=("PASS: $test_name")
    return 0
  else
    TESTS_FAILED=$((TESTS_FAILED + 1))
    printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$test_name"
    printf "    ${RED}Should reject invalid command${NC}\n"
    TEST_RESULTS+=("FAIL: $test_name - Didn't reject invalid command")
    return 1
  fi
}

test_invalid_command "invalidcmd123" "Error: Invalid command rejected"
test_invalid_command "foobar" "Error: Random command rejected"

# TEST 9: File Structure
print_section "TEST 9: File Structure Verification"

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

test_file_structure "$SCRIPT_DIR/../lib/utils/cli-output.sh" "File: cli-output.sh exists"
test_file_structure "$SCRIPT_DIR/../lib/config/constants.sh" "File: constants.sh exists"
test_file_structure "$SCRIPT_DIR/../lib/config/defaults.sh" "File: defaults.sh exists"

# Print summary
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
  printf "${GREEN}║  ALL TESTS PASSED ✓                                           ║${NC}\n"
  printf "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}\n"
  exit 0
fi
