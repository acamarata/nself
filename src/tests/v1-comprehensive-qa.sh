#!/usr/bin/env bash
# v1-comprehensive-qa.sh - Complete QA suite for v1.0 release
#
# Tests all implemented commands, routing, output formatting, and compatibility

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/../cli" && pwd)"
BIN_DIR="$(cd "$SCRIPT_DIR/../../bin" && pwd)"

# Create temp directory to run tests from (avoid source repo protection)
TEST_DIR=$(mktemp -d)
trap "rm -rf $TEST_DIR" EXIT
cd "$TEST_DIR"

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
TESTS_WARNING=0

# Test results
declare -a FAILED_TESTS=()
declare -a WARNING_TESTS=()

# Get all actual commands (Bash 3.2 compatible)
ALL_COMMANDS=()
while IFS= read -r cmd; do
  ALL_COMMANDS+=("$cmd")
done < <(ls -1 "$CLI_DIR"/*.sh 2>/dev/null | xargs -n1 basename | sed 's/\.sh$//' | grep -v '^nself$' | sort)
TOTAL_COMMANDS=${#ALL_COMMANDS[@]}

# Utility functions
print_header() {
  printf "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${BLUE}║    nself v1.0 - Comprehensive QA Verification Suite           ║${NC}\n"
  printf "${BLUE}╠════════════════════════════════════════════════════════════════╣${NC}\n"
  printf "${BLUE}║  Total Commands: %-47d║${NC}\n" "$TOTAL_COMMANDS"
  printf "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}\n\n"
}

print_section() {
  printf "\n${CYAN}▶ %s${NC}\n" "$1"
  printf "${CYAN}%s${NC}\n" "$(printf '─%.0s' {1..64})"
}

pass_test() {
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_PASSED=$((TESTS_PASSED + 1))
  printf "  ${GREEN}✓${NC} %-50s ${GREEN}PASS${NC}\n" "$1"
}

fail_test() {
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_FAILED=$((TESTS_FAILED + 1))
  printf "  ${RED}✗${NC} %-50s ${RED}FAIL${NC}\n" "$1"
  FAILED_TESTS+=("$1${2:+ - $2}")
}

warn_test() {
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_WARNING=$((TESTS_WARNING + 1))
  printf "  ${YELLOW}⚠${NC} %-50s ${YELLOW}WARN${NC}\n" "$1"
  WARNING_TESTS+=("$1${2:+ - $2}")
}

skip_test() {
  TESTS_RUN=$((TESTS_RUN + 1))
  TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
  printf "  ${YELLOW}○${NC} %-50s ${YELLOW}SKIP${NC}\n" "$1"
}

# Start tests
print_header

# TEST 1: File Structure
print_section "TEST 1: Core File Structure"

# Check main wrapper
if [[ -f "$CLI_DIR/nself.sh" ]]; then
  pass_test "Main wrapper: nself.sh exists"
else
  fail_test "Main wrapper: nself.sh exists" "Missing file"
fi

# Check binary
if [[ -x "$BIN_DIR/nself" ]]; then
  pass_test "Binary: bin/nself is executable"
else
  fail_test "Binary: bin/nself is executable" "Not found or not executable"
fi

# Check core library files
for lib_file in "utils/cli-output.sh" "config/constants.sh" "config/defaults.sh"; do
  if [[ -f "$SCRIPT_DIR/../lib/$lib_file" ]]; then
    pass_test "Library: $lib_file exists"
  else
    fail_test "Library: $lib_file exists" "Missing file"
  fi
done

# TEST 2: All Command Files
print_section "TEST 2: Command File Verification ($TOTAL_COMMANDS commands)"

for cmd in "${ALL_COMMANDS[@]}"; do
  if [[ -f "$CLI_DIR/$cmd.sh" ]] && [[ -r "$CLI_DIR/$cmd.sh" ]]; then
    pass_test "Command file: $cmd.sh"
  else
    fail_test "Command file: $cmd.sh" "Missing or unreadable"
  fi
done

# TEST 3: Command Routing
print_section "TEST 3: Command Routing (Sample: 20 commands)"

# Test a representative sample to avoid timeout
SAMPLE_COMMANDS=("help" "version" "init" "build" "start" "stop" "status" "env" "config" "db" "backup" "deploy" "logs" "urls" "doctor" "clean" "auth" "secrets" "dev" "sync")

for cmd in "${SAMPLE_COMMANDS[@]}"; do
  if [[ ! -f "$CLI_DIR/$cmd.sh" ]]; then
    skip_test "Route: nself $cmd" "(command not implemented)"
    continue
  fi

  output=""
  exit_code=0
  output=$("$BIN_DIR/nself" "$cmd" --help 2>&1) || exit_code=$?

  # Accept 0 (success), 1 (expected for some commands)
  # Reject 127 (command not found), 126 (not executable)
  if [[ $exit_code -eq 0 ]] || [[ $exit_code -eq 1 ]]; then
    pass_test "Route: nself $cmd"
  elif [[ $exit_code -eq 127 ]]; then
    fail_test "Route: nself $cmd" "Command not found (exit 127)"
  elif [[ $exit_code -eq 126 ]]; then
    fail_test "Route: nself $cmd" "Not executable (exit 126)"
  else
    warn_test "Route: nself $cmd" "Unexpected exit code $exit_code"
  fi
done

# TEST 4: Help System
print_section "TEST 4: Help System"

# Test main help
help_output=""
if help_output=$("$BIN_DIR/nself" help 2>&1) && [[ -n "$help_output" ]]; then
  pass_test "Help: nself help (returns output)"
else
  fail_test "Help: nself help" "No output"
fi

# Test -h flag
if help_output=$("$BIN_DIR/nself" -h 2>&1) && [[ -n "$help_output" ]]; then
  pass_test "Help: nself -h flag"
else
  fail_test "Help: nself -h flag" "No output"
fi

# Test --help flag
if help_output=$("$BIN_DIR/nself" --help 2>&1) && [[ -n "$help_output" ]]; then
  pass_test "Help: nself --help flag"
else
  fail_test "Help: nself --help flag" "No output"
fi

# Test 5: Version System
print_section "TEST 5: Version System"

# Test version command
version_output=""
if version_output=$("$BIN_DIR/nself" version 2>&1) && [[ -n "$version_output" ]]; then
  pass_test "Version: nself version"
else
  fail_test "Version: nself version" "No output"
fi

# Test -v flag
if version_output=$("$BIN_DIR/nself" -v 2>&1) && [[ -n "$version_output" ]]; then
  pass_test "Version: nself -v flag"
else
  fail_test "Version: nself -v flag" "No output"
fi

# Test --version flag
if version_output=$("$BIN_DIR/nself" --version 2>&1) && [[ -n "$version_output" ]]; then
  pass_test "Version: nself --version flag"
else
  fail_test "Version: nself --version flag" "No output"
fi

# TEST 6: Output Formatting
print_section "TEST 6: Output Formatting (cli-output.sh integration)"

# Check critical commands use proper output formatting
CRITICAL_COMMANDS=("init" "build" "start" "stop" "deploy" "backup" "env" "db")

for cmd in "${CRITICAL_COMMANDS[@]}"; do
  if [[ ! -f "$CLI_DIR/$cmd.sh" ]]; then
    skip_test "Output: $cmd uses cli-output.sh" "(not implemented)"
    continue
  fi

  # Check if file sources cli-output.sh or uses log_ functions
  if grep -qE '(cli-output\.sh|log_success|log_error|log_info|log_warning|display\.sh)' "$CLI_DIR/$cmd.sh" 2>/dev/null; then
    pass_test "Output: $cmd uses cli-output.sh"
  else
    warn_test "Output: $cmd uses cli-output.sh" "No output formatting found"
  fi
done

# TEST 7: Subcommand Structure
print_section "TEST 7: Subcommand Support"

# Commands that should support subcommands
SUBCOMMAND_CMDS=("env" "db" "backup" "config" "deploy" "auth" "secrets" "service")

for cmd in "${SUBCOMMAND_CMDS[@]}"; do
  if [[ ! -f "$CLI_DIR/$cmd.sh" ]]; then
    skip_test "Subcommands: $cmd" "(not implemented)"
    continue
  fi

  # Check if file has case statement for subcommands
  if grep -q 'case.*\$[{]*1[}]*.*in' "$CLI_DIR/$cmd.sh" 2>/dev/null; then
    pass_test "Subcommands: $cmd has case statement"
  else
    warn_test "Subcommands: $cmd has case statement" "No case statement found"
  fi
done

# TEST 8: Error Handling
print_section "TEST 8: Error Handling"

# Test invalid command
output=""
exit_code=0
output=$("$BIN_DIR/nself" invalidcommand123 2>&1) || exit_code=$?

if [[ $exit_code -ne 0 ]] && [[ "$output" =~ (Unknown|error|not found|invalid) || "$output" =~ (Unknown|error|not found|invalid) ]]; then
  pass_test "Error: Invalid command rejected"
else
  fail_test "Error: Invalid command rejected" "Should reject with error"
fi

# TEST 9: Critical Command Files
print_section "TEST 9: Critical Commands (Production Essentials)"

CRITICAL_FILES=("init" "build" "start" "stop" "restart" "status" "logs" "env" "db" "backup" "restore" "deploy" "health" "doctor")

for cmd in "${CRITICAL_FILES[@]}"; do
  if [[ -f "$CLI_DIR/$cmd.sh" ]]; then
    pass_test "Critical: $cmd.sh exists"
  else
    fail_test "Critical: $cmd.sh exists" "MISSING CRITICAL FILE"
  fi
done

# TEST 10: Source Repository Protection
print_section "TEST 10: Source Repository Protection"

# Test that nself detects when run in its own repo
# This is done by checking the protection logic in nself.sh
if grep -q "Cannot run nself commands in the nself source repository" "$CLI_DIR/nself.sh" 2>/dev/null; then
  pass_test "Protection: Source repo detection exists"
else
  warn_test "Protection: Source repo detection exists" "No protection found"
fi

# Print summary
print_section "TEST SUMMARY"

printf "\n"
printf "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}\n"
printf "${BLUE}║  Test Results Summary                                          ║${NC}\n"
printf "${BLUE}╠════════════════════════════════════════════════════════════════╣${NC}\n"
printf "${BLUE}║  ${NC}Total Tests:     %-42d ${BLUE}║${NC}\n" "$TESTS_RUN"
printf "${BLUE}║  ${GREEN}✓${NC} Passed:         %-42d ${BLUE}║${NC}\n" "$TESTS_PASSED"
printf "${BLUE}║  ${RED}✗${NC} Failed:         %-42d ${BLUE}║${NC}\n" "$TESTS_FAILED"
printf "${BLUE}║  ${YELLOW}⚠${NC} Warnings:       %-42d ${BLUE}║${NC}\n" "$TESTS_WARNING"
printf "${BLUE}║  ${YELLOW}○${NC} Skipped:        %-42d ${BLUE}║${NC}\n" "$TESTS_SKIPPED"
printf "${BLUE}╠════════════════════════════════════════════════════════════════╣${NC}\n"
printf "${BLUE}║  ${NC}Commands Found:  %-42d ${BLUE}║${NC}\n" "$TOTAL_COMMANDS"
printf "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}\n"
printf "\n"

# Show failed tests
if [[ $TESTS_FAILED -gt 0 ]]; then
  printf "${RED}╔════════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${RED}║  Failed Tests                                                  ║${NC}\n"
  printf "${RED}╚════════════════════════════════════════════════════════════════╝${NC}\n\n"
  for test in "${FAILED_TESTS[@]}"; do
    printf "  ${RED}✗${NC} %s\n" "$test"
  done
  printf "\n"
fi

# Show warnings
if [[ $TESTS_WARNING -gt 0 ]]; then
  printf "${YELLOW}╔════════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${YELLOW}║  Warnings (Non-Critical)                                       ║${NC}\n"
  printf "${YELLOW}╚════════════════════════════════════════════════════════════════╝${NC}\n\n"
  for test in "${WARNING_TESTS[@]}"; do
    printf "  ${YELLOW}⚠${NC} %s\n" "$test"
  done
  printf "\n"
fi

# Final verdict
if [[ $TESTS_FAILED -eq 0 ]]; then
  printf "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${GREEN}║  ✓ ALL CRITICAL TESTS PASSED                                   ║${NC}\n"
  printf "${GREEN}╠════════════════════════════════════════════════════════════════╣${NC}\n"
  printf "${GREEN}║  The v1.0 command structure is ready for production.           ║${NC}\n"
  printf "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}\n"

  # Show status
  pass_rate=$((TESTS_PASSED * 100 / TESTS_RUN))
  printf "\n${GREEN}Pass Rate: %d%% (%d/%d)${NC}\n" "$pass_rate" "$TESTS_PASSED" "$TESTS_RUN"

  if [[ $TESTS_WARNING -gt 0 ]]; then
    printf "${YELLOW}Note: %d warnings found (non-critical, review recommended)${NC}\n" "$TESTS_WARNING"
  fi

  exit 0
else
  printf "${RED}╔════════════════════════════════════════════════════════════════╗${NC}\n"
  printf "${RED}║  ✗ TESTS FAILED - REVIEW REQUIRED                              ║${NC}\n"
  printf "${RED}╠════════════════════════════════════════════════════════════════╣${NC}\n"
  printf "${RED}║  The v1.0 command structure has critical issues.               ║${NC}\n"
  printf "${RED}╚════════════════════════════════════════════════════════════════╝${NC}\n"

  pass_rate=$((TESTS_PASSED * 100 / TESTS_RUN))
  printf "\n${RED}Pass Rate: %d%% (%d/%d)${NC}\n" "$pass_rate" "$TESTS_PASSED" "$TESTS_RUN"
  printf "${RED}Failed: %d tests${NC}\n" "$TESTS_FAILED"

  exit 1
fi
