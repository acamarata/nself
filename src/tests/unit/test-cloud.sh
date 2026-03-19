#!/usr/bin/env bash
# test-cloud.sh - Tests for nself cloud command
# Part of nself v0.9.9

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_FILE="$SCRIPT_DIR/../../cli/cloud.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0

# Test: Syntax validation
test_syntax() {
  TESTS_RUN=$((TESTS_RUN + 1))
  if bash -n "$SOURCE_FILE" 2>/dev/null; then
    printf "✓ Syntax valid\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Syntax validation failed\n"
    return 1
  fi
}

# Test: Help shows subcommands
test_help_subcommands() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local output
  output=$(bash "$SOURCE_FILE" 2>&1 || true)

  if echo "$output" | grep -q "status" && echo "$output" | grep -q "upgrade" && echo "$output" | grep -q "destroy"; then
    printf "✓ Help shows subcommands (status, upgrade, destroy)\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Help subcommands missing\n"
    return 1
  fi
}

# Test: Legacy provider commands reference infra provider
test_legacy_reference() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local output
  output=$(bash "$SOURCE_FILE" 2>&1 || true)

  if echo "$output" | grep -q "infra provider"; then
    printf "✓ Mentions 'nself infra provider' for legacy commands\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Legacy provider reference missing\n"
    return 1
  fi
}

# Run all tests
printf "Testing nself cloud...\n"
test_syntax
test_help_subcommands
test_legacy_reference

printf "\n"
printf "Tests passed: %d/%d\n" "$TESTS_PASSED" "$TESTS_RUN"

if [ "$TESTS_PASSED" -eq "$TESTS_RUN" ]; then
  printf "✅ All tests passed\n"
  exit 0
else
  printf "❌ Some tests failed\n"
  exit 1
fi
