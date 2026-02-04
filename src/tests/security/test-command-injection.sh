#!/usr/bin/env bash
# test-command-injection.sh - Command Injection Security Tests
# Tests for unsafe command execution patterns

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

pass() {
  printf "${GREEN}✓${NC} %s\n" "$1"
  TESTS_PASSED=$((TESTS_PASSED + 1))
  TESTS_RUN=$((TESTS_RUN + 1))
}

fail() {
  printf "${RED}✗${NC} %s\n" "$1"
  TESTS_FAILED=$((TESTS_FAILED + 1))
  TESTS_RUN=$((TESTS_RUN + 1))
}

warn() {
  printf "${YELLOW}⚠${NC} %s\n" "$1"
}

section() {
  printf "\n${BLUE}=== %s ===${NC}\n\n" "$1"
}

# Test 1: No unsafe eval usage
test_no_eval() {
  section "Test 1: No Unsafe eval Usage"

  # Look for eval with user input (dangerous patterns)
  local dangerous_eval
  dangerous_eval=$(grep -rE 'eval.*\$\{?input\}?|eval.*\$\{?user|eval.*\$\{?param' "$PROJECT_ROOT/src/lib" 2>/dev/null | wc -l | xargs)

  if [[ "$dangerous_eval" -eq 0 ]]; then
    pass "No dangerous eval with user input found"
  else
    fail "Found $dangerous_eval eval statements with user input"
  fi

  # Note: Safe eval usage for path expansion and array assignments is acceptable
  local safe_eval
  safe_eval=$(grep -rw "eval" "$PROJECT_ROOT/src/lib" 2>/dev/null | grep -E "echo|array_name" | wc -l | xargs)
  if [[ "$safe_eval" -gt 0 ]]; then
    warn "Found $safe_eval eval statements (for path expansion/arrays - acceptable)"
  fi
}

# Test 2: Proper variable quoting
test_variable_quoting() {
  section "Test 2: Variable Quoting in Commands"

  # Check docker exec calls
  local unquoted
  unquoted=$(grep -rE 'docker exec [^"]*\$[a-zA-Z_]+' "$PROJECT_ROOT/src/lib" 2>/dev/null | grep -v '"\$' | wc -l | xargs)

  if [[ "$unquoted" -eq 0 ]]; then
    pass "All docker exec calls properly quoted"
  else
    warn "Found $unquoted potentially unquoted docker exec calls"
  fi
}

# Test 3: No backticks with variables
test_no_backticks_with_vars() {
  section "Test 3: No Command Substitution with Unquoted Variables"

  local backtick_vars
  backtick_vars=$(grep -rE '`.*\$[a-zA-Z_]+.*`' "$PROJECT_ROOT/src/lib" 2>/dev/null | wc -l | xargs)

  if [[ "$backtick_vars" -eq 0 ]]; then
    pass "No backticks with variables"
  else
    fail "Found $backtick_vars backtick command substitutions with variables"
  fi
}

# Test 4: SSH command safety
test_ssh_safety() {
  section "Test 4: SSH Command Safety"

  # Check for heredocs (safe)
  local heredoc_count
  heredoc_count=$(grep -rE 'ssh.*<<' "$PROJECT_ROOT/src/lib" 2>/dev/null | wc -l | xargs)

  if [[ "$heredoc_count" -gt 0 ]]; then
    pass "Found $heredoc_count SSH calls using heredocs (safe)"
  fi

  # Check for single-quoted SSH commands (safe)
  local quoted_count
  quoted_count=$(grep -rE "ssh.*'.*'" "$PROJECT_ROOT/src/lib" 2>/dev/null | wc -l | xargs)

  if [[ "$quoted_count" -gt 0 ]]; then
    pass "Found $quoted_count SSH calls using single quotes (safe)"
  fi
}

# Main
main() {
  printf "\n${BLUE}╔════════════════════════════════════════════════════════╗${NC}\n"
  printf "${BLUE}║    Command Injection Security Test Suite              ║${NC}\n"
  printf "${BLUE}╚════════════════════════════════════════════════════════╝${NC}\n"

  test_no_eval
  test_variable_quoting
  test_no_backticks_with_vars
  test_ssh_safety

  # Summary
  printf "\n${BLUE}═══════════════════════════════════════════════════════${NC}\n"
  printf "Total: %d | ${GREEN}Passed: %d${NC} | ${RED}Failed: %d${NC}\n" "$TESTS_RUN" "$TESTS_PASSED" "$TESTS_FAILED"

  if [[ $TESTS_FAILED -eq 0 ]]; then
    printf "${GREEN}✓ All command injection tests passed!${NC}\n"
    return 0
  else
    return 1
  fi
}

main "$@"
