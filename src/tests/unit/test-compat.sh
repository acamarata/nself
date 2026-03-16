#!/usr/bin/env bash
# test-compat.sh - Tests for nself compat
# Part of nself v0.9.9

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_FILE="$SCRIPT_DIR/../../cli/compat.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0

# Scratch directory for test fixtures
TMPDIR_COMPAT=""

setup() {
  TMPDIR_COMPAT="$(mktemp -d)"
}

teardown() {
  if [[ -n "$TMPDIR_COMPAT" ]] && [[ -d "$TMPDIR_COMPAT" ]]; then
    rm -rf "$TMPDIR_COMPAT"
  fi
}

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

# Test: Help text present
test_help() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local help_output
  help_output=$(bash "$SOURCE_FILE" --help 2>&1 || true)

  if printf '%s' "$help_output" | grep -qi "usage\|check\|compat"; then
    printf "✓ Help text present\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Help text missing or incomplete\n"
    return 1
  fi
}

# Test: compat check with no files exits cleanly
test_check_no_files() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local output
  # Run in a temp directory that has no .env or docker-compose.yml
  output=$(cd "$TMPDIR_COMPAT" && bash "$SOURCE_FILE" check 2>&1 || true)

  # Should mention skipping files and show "All checks passed" (no script files either)
  if printf '%s' "$output" | grep -qi "not found\|passed\|skipping"; then
    printf "✓ Clean run with no files succeeds\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Clean run output unexpected: %s\n" "$output"
    return 1
  fi
}

# Test: detects CS_N 3-field format in .env
test_cs_three_field_warning() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local tdir
  tdir="$(mktemp -d)"
  printf 'CS_1=myservice:node:3000\n' > "$tdir/.env"
  local output
  output=$(cd "$tdir" && bash "$SOURCE_FILE" check 2>&1 || true)
  rm -rf "$tdir"

  if printf '%s' "$output" | grep -qi "3-field\|route\|v1.1.0"; then
    printf "✓ Detects CS_N 3-field format deprecation\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Did not detect CS_N 3-field format: %s\n" "$output"
    return 1
  fi
}

# Test: passes CS_N 4-field format without warning
test_cs_four_field_ok() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local tdir
  tdir="$(mktemp -d)"
  printf 'CS_1=myservice:node:3000:api\n' > "$tdir/.env"
  local output
  output=$(cd "$tdir" && bash "$SOURCE_FILE" check 2>&1 || true)
  rm -rf "$tdir"

  if printf '%s' "$output" | grep -qi "3-field\|route\|v1.1.0"; then
    printf "✗ Incorrectly warned on valid 4-field CS_N format\n"
    return 1
  else
    printf "✓ 4-field CS_N format passes without warning\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  fi
}

# Test: detects legacy-compose in docker-compose.yml
test_legacy_compose_issue() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local tdir
  tdir="$(mktemp -d)"
  printf 'version: "3"\n# legacy-compose format\nservices:\n  postgres:\n    image: postgres:15\n' > "$tdir/docker-compose.yml"
  local output
  output=$(cd "$tdir" && bash "$SOURCE_FILE" check 2>&1 || true)
  rm -rf "$tdir"

  if printf '%s' "$output" | grep -qi "legacy.compose\|removed"; then
    printf "✓ Detects legacy-compose in docker-compose.yml\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Did not detect legacy-compose issue: %s\n" "$output"
    return 1
  fi
}

# Test: JSON output is valid structure
test_json_output() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local tdir
  tdir="$(mktemp -d)"
  local output
  output=$(cd "$tdir" && bash "$SOURCE_FILE" check --json 2>&1 || true)
  rm -rf "$tdir"

  if printf '%s' "$output" | grep -q '"status"' && printf '%s' "$output" | grep -q '"issues"'; then
    printf "✓ JSON output has expected fields\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ JSON output missing expected fields: %s\n" "$output"
    return 1
  fi
}

# Test: exit code 1 when issues found
test_exit_code_on_issues() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local tdir
  tdir="$(mktemp -d)"
  printf '# legacy-compose format\n' > "$tdir/docker-compose.yml"
  local exit_code=0
  (cd "$tdir" && bash "$SOURCE_FILE" check 2>&1) || exit_code=$?
  rm -rf "$tdir"

  if [[ "$exit_code" -ne 0 ]]; then
    printf "✓ Non-zero exit code on issues\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Expected non-zero exit code when issues found, got 0\n"
    return 1
  fi
}

# Test: exit code 0 when no issues (warnings alone don't fail)
test_exit_code_clean() {
  TESTS_RUN=$((TESTS_RUN + 1))
  local tdir
  tdir="$(mktemp -d)"
  # 3-field CS_N is a warning, not an issue — should exit 0
  printf 'CS_1=myservice:node:3000\n' > "$tdir/.env"
  local exit_code=0
  (cd "$tdir" && bash "$SOURCE_FILE" check 2>&1) || exit_code=$?
  rm -rf "$tdir"

  if [[ "$exit_code" -eq 0 ]]; then
    printf "✓ Exit code 0 when only warnings (no hard issues)\n"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    printf "✗ Expected exit code 0 for warnings-only, got %d\n" "$exit_code"
    return 1
  fi
}

# Run all tests
printf "Testing nself compat...\n\n"

setup

test_syntax
test_help
test_check_no_files
test_cs_three_field_warning
test_cs_four_field_ok
test_legacy_compose_issue
test_json_output
test_exit_code_on_issues
test_exit_code_clean

teardown

printf "\n"
printf "Tests passed: %d/%d\n" "$TESTS_PASSED" "$TESTS_RUN"

if [ "$TESTS_PASSED" -eq "$TESTS_RUN" ]; then
  printf "✅ All tests passed\n"
  exit 0
else
  printf "❌ Some tests failed\n"
  exit 1
fi
