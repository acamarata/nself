#!/usr/bin/env bash
# test-permissions.sh - File Permission Security Tests
# Verifies sensitive files have correct permissions

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

# Get file permissions in octal format (cross-platform)
get_perms() {
  local file="$1"
  if stat --version 2>/dev/null | grep -q GNU; then
    stat -c "%a" "$file"
  else
    stat -f "%OLp" "$file"
  fi
}

# Test 1: .env files permissions
test_env_permissions() {
  section "Test 1: .env Files Permissions"

  cd "$PROJECT_ROOT"

  # Check .env files (should be 600 or not exist)
  for env_file in .env .env.local .env.staging .env.prod .secrets; do
    if [[ -f "$env_file" ]]; then
      local perms
      perms=$(get_perms "$env_file")
      if [[ "$perms" == "600" ]] || [[ "$perms" == "400" ]]; then
        pass "$env_file has secure permissions ($perms)"
      else
        fail "$env_file has insecure permissions ($perms) - should be 600"
      fi
    else
      warn "$env_file does not exist (OK if not in use)"
    fi
  done
}

# Test 2: SSL key permissions
test_ssl_permissions() {
  section "Test 2: SSL Key Permissions"

  cd "$PROJECT_ROOT"

  # Check for .key and .pem files
  if [[ -d "ssl" ]]; then
    local key_files
    key_files=$(find ssl -name "*.key" -o -name "*.pem" 2>/dev/null || true)

    if [[ -n "$key_files" ]]; then
      while IFS= read -r key_file; do
        if [[ -f "$key_file" ]]; then
          local perms
          perms=$(get_perms "$key_file")
          if [[ "$perms" == "600" ]] || [[ "$perms" == "400" ]]; then
            pass "$key_file has secure permissions ($perms)"
          else
            fail "$key_file has insecure permissions ($perms)"
          fi
        fi
      done <<< "$key_files"
    else
      warn "No SSL keys found (OK for development)"
    fi
  else
    warn "ssl/ directory not found (OK for fresh install)"
  fi
}

# Test 3: Scripts are executable
test_scripts_executable() {
  section "Test 3: Shell Scripts Are Executable"

  # Check main CLI script
  if [[ -f "$PROJECT_ROOT/nself" ]]; then
    if [[ -x "$PROJECT_ROOT/nself" ]]; then
      pass "nself CLI is executable"
    else
      fail "nself CLI is not executable"
    fi
  fi

  # Check a few key scripts
  for script in src/cli/*.sh; do
    if [[ -f "$script" ]]; then
      if [[ -x "$script" ]]; then
        pass "$(basename "$script") is executable"
      else
        warn "$(basename "$script") is not executable (may be sourced)"
      fi
      break  # Just check one as example
    fi
  done
}

# Test 4: No world-writable files
test_no_world_writable() {
  section "Test 4: No World-Writable Files"

  cd "$PROJECT_ROOT"

  # Find world-writable files (excluding .git and node_modules)
  local writable
  writable=$(find . -type f -perm -002 ! -path "*/\.git/*" ! -path "*/node_modules/*" 2>/dev/null || true)

  if [[ -z "$writable" ]]; then
    pass "No world-writable files found"
  else
    fail "Found world-writable files:"
    printf "%s\n" "$writable"
  fi
}

# Main
main() {
  printf "\n${BLUE}╔════════════════════════════════════════════════════════╗${NC}\n"
  printf "${BLUE}║      File Permissions Security Test Suite             ║${NC}\n"
  printf "${BLUE}╚════════════════════════════════════════════════════════╝${NC}\n"

  test_env_permissions
  test_ssl_permissions
  test_scripts_executable
  test_no_world_writable

  # Summary
  printf "\n${BLUE}═══════════════════════════════════════════════════════${NC}\n"
  printf "Total: %d | ${GREEN}Passed: %d${NC} | ${RED}Failed: %d${NC}\n" "$TESTS_RUN" "$TESTS_PASSED" "$TESTS_FAILED"

  if [[ $TESTS_FAILED -eq 0 ]]; then
    printf "${GREEN}✓ All permission tests passed!${NC}\n"
    return 0
  else
    printf "${YELLOW}⚠ Run 'chmod 600 .env*' and 'chmod 600 ssl/*.key' to fix${NC}\n"
    return 1
  fi
}

main "$@"
