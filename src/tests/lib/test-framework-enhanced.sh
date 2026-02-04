#!/usr/bin/env bash
# Enhanced test framework with reliability features
# Extends the base test framework with timeout, retry, cleanup, and more

set -euo pipefail

# ============================================================================
# Source Base Framework
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source existing test framework if it exists
if [[ -f "$SCRIPT_DIR/../test_framework.sh" ]]; then
  source "$SCRIPT_DIR/../test_framework.sh"
fi

# Source mock infrastructure
if [[ -f "$SCRIPT_DIR/../mocks/mock-infrastructure.sh" ]]; then
  source "$SCRIPT_DIR/../mocks/mock-infrastructure.sh"
fi

# ============================================================================
# Enhanced Test Execution
# ============================================================================

# Run test with timeout protection
run_test_with_timeout() {
  local test_func="$1"
  local timeout="${2:-30}"  # Default 30 seconds

  # Check if timeout command is available
  if command -v timeout >/dev/null 2>&1; then
    timeout "$timeout" bash -c "$test_func"
    local result=$?

    if [[ $result -eq 124 ]]; then
      printf "\033[31mTIMEOUT:\033[0m %s exceeded %ss\n" "$test_func" "$timeout" >&2
      return 1
    fi

    return $result
  elif command -v gtimeout >/dev/null 2>&1; then
    # macOS with coreutils
    gtimeout "$timeout" bash -c "$test_func"
    local result=$?

    if [[ $result -eq 124 ]]; then
      printf "\033[31mTIMEOUT:\033[0m %s exceeded %ss\n" "$test_func" "$timeout" >&2
      return 1
    fi

    return $result
  else
    # No timeout command - run directly but warn
    printf "\033[33mWARN:\033[0m timeout command not available, running without timeout\n" >&2
    bash -c "$test_func"
    return $?
  fi
}

# ============================================================================
# Retry Logic for Flaky Operations
# ============================================================================

# Retry a test function multiple times
retry_test() {
  local test_func="$1"
  local max_attempts="${2:-3}"
  local delay="${3:-1}"
  local attempt=1

  while [[ $attempt -le $max_attempts ]]; do
    if $test_func; then
      # Test passed
      if [[ $attempt -gt 1 ]]; then
        printf "\033[33mINFO:\033[0m %s passed on attempt %d/%d\n" "$test_func" "$attempt" "$max_attempts"
      fi
      return 0
    fi

    # Test failed
    if [[ $attempt -lt $max_attempts ]]; then
      printf "\033[33mRETRY:\033[0m %s failed on attempt %d/%d, retrying in %ds...\n" \
        "$test_func" "$attempt" "$max_attempts" "$delay" >&2
      sleep "$delay"
    fi

    attempt=$((attempt + 1))
  done

  # All attempts failed
  printf "\033[31mFAILED:\033[0m %s after %d attempts\n" "$test_func" "$max_attempts" >&2
  return 1
}

# Retry with exponential backoff
retry_with_backoff() {
  local test_func="$1"
  local max_attempts="${2:-3}"
  local initial_delay="${3:-1}"
  local attempt=1
  local delay=$initial_delay

  while [[ $attempt -le $max_attempts ]]; do
    if $test_func; then
      if [[ $attempt -gt 1 ]]; then
        printf "\033[33mINFO:\033[0m %s passed on attempt %d/%d\n" "$test_func" "$attempt" "$max_attempts"
      fi
      return 0
    fi

    if [[ $attempt -lt $max_attempts ]]; then
      printf "\033[33mRETRY:\033[0m %s failed on attempt %d/%d, retrying in %ds...\n" \
        "$test_func" "$attempt" "$max_attempts" "$delay" >&2
      sleep "$delay"
      delay=$((delay * 2))  # Exponential backoff
    fi

    attempt=$((attempt + 1))
  done

  printf "\033[31mFAILED:\033[0m %s after %d attempts\n" "$test_func" "$max_attempts" >&2
  return 1
}

# ============================================================================
# Cleanup Management
# ============================================================================

# Global cleanup functions array
declare -a CLEANUP_FUNCTIONS=()

# Register cleanup function
ensure_cleanup() {
  local cleanup_func="$1"

  # Add to cleanup array
  CLEANUP_FUNCTIONS+=("$cleanup_func")

  # Register trap to run all cleanups
  trap run_all_cleanups EXIT INT TERM
}

# Run all registered cleanup functions
run_all_cleanups() {
  local exit_code=$?

  # Run cleanups in reverse order (LIFO)
  for ((i=${#CLEANUP_FUNCTIONS[@]}-1; i>=0; i--)); do
    local cleanup="${CLEANUP_FUNCTIONS[$i]}"
    if [[ -n "$cleanup" ]]; then
      eval "$cleanup" 2>/dev/null || true
    fi
  done

  # Clear array
  CLEANUP_FUNCTIONS=()

  return $exit_code
}

# Clear all cleanup functions
clear_cleanups() {
  CLEANUP_FUNCTIONS=()
}

# ============================================================================
# Test Control Flow
# ============================================================================

# Fail fast on critical errors
fail_fast() {
  local error_msg="$1"
  local exit_code="${2:-1}"

  printf "\033[31mCRITICAL ERROR:\033[0m %s\n" "$error_msg" >&2

  # Run cleanups
  run_all_cleanups

  exit "$exit_code"
}

# Skip test gracefully
skip_test() {
  local reason="$1"

  printf "\033[33mSKIP:\033[0m %s\n" "$reason"
  return 0
}

# Mark test as expected to fail
expect_failure() {
  local test_func="$1"
  local expected_error="${2:-}"

  if $test_func 2>&1; then
    printf "\033[31mUNEXPECTED PASS:\033[0m %s was expected to fail\n" "$test_func" >&2
    return 1
  else
    printf "\033[32mEXPECTED FAILURE:\033[0m %s failed as expected\n" "$test_func"
    return 0
  fi
}

# ============================================================================
# Environment Detection
# ============================================================================

# Check if running in CI
is_ci() {
  [[ -n "${CI:-}" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]] || [[ -n "${GITLAB_CI:-}" ]] || [[ -n "${CIRCLECI:-}" ]]
}

# Check if running on macOS
is_macos() {
  [[ "$(uname)" == "Darwin" ]]
}

# Check if running on Linux
is_linux() {
  [[ "$(uname)" == "Linux" ]]
}

# Check if running in WSL
is_wsl() {
  [[ -f "/proc/version" ]] && grep -qi microsoft /proc/version
}

# Get current platform
get_platform() {
  if is_macos; then
    printf "macos\n"
  elif is_wsl; then
    printf "wsl\n"
  elif is_linux; then
    printf "linux\n"
  else
    printf "unknown\n"
  fi
}

# ============================================================================
# Assertion Helpers
# ============================================================================

# Assert command succeeds
assert_success() {
  local exit_code="${1:-$?}"

  if [[ $exit_code -ne 0 ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m Expected success (0), got %d\n" "$exit_code" >&2
    return 1
  fi
  return 0
}

# Assert command fails
assert_failure() {
  local exit_code="${1:-$?}"

  if [[ $exit_code -eq 0 ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m Expected failure (non-zero), got 0\n" >&2
    return 1
  fi
  return 0
}

# Assert equals
assert_equals() {
  local expected="$1"
  local actual="$2"

  if [[ "$expected" != "$actual" ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m\n" >&2
    printf "  Expected: %s\n" "$expected" >&2
    printf "  Actual:   %s\n" "$actual" >&2
    return 1
  fi
  return 0
}

# Assert not equals
assert_not_equals() {
  local not_expected="$1"
  local actual="$2"

  if [[ "$not_expected" == "$actual" ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m\n" >&2
    printf "  Not expected: %s\n" "$not_expected" >&2
    printf "  Actual:       %s\n" "$actual" >&2
    return 1
  fi
  return 0
}

# Assert contains
assert_contains() {
  local haystack="$1"
  local needle="$2"

  if [[ "$haystack" != *"$needle"* ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m\n" >&2
    printf "  String:   %s\n" "$haystack" >&2
    printf "  Expected to contain: %s\n" "$needle" >&2
    return 1
  fi
  return 0
}

# Assert not contains
assert_not_contains() {
  local haystack="$1"
  local needle="$2"

  if [[ "$haystack" == *"$needle"* ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m\n" >&2
    printf "  String:   %s\n" "$haystack" >&2
    printf "  Expected NOT to contain: %s\n" "$needle" >&2
    return 1
  fi
  return 0
}

# Assert file exists
assert_file_exists() {
  local file="$1"

  if [[ ! -f "$file" ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m File does not exist: %s\n" "$file" >&2
    return 1
  fi
  return 0
}

# Assert file not exists
assert_file_not_exists() {
  local file="$1"

  if [[ -f "$file" ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m File exists but should not: %s\n" "$file" >&2
    return 1
  fi
  return 0
}

# Assert directory exists
assert_dir_exists() {
  local dir="$1"

  if [[ ! -d "$dir" ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m Directory does not exist: %s\n" "$dir" >&2
    return 1
  fi
  return 0
}

# Assert directory not exists
assert_dir_not_exists() {
  local dir="$1"

  if [[ -d "$dir" ]]; then
    printf "\033[31mASSERTION FAILED:\033[0m Directory exists but should not: %s\n" "$dir" >&2
    return 1
  fi
  return 0
}

# ============================================================================
# Test Isolation
# ============================================================================

# Create isolated test environment
create_test_env() {
  local test_name="${1:-test}"
  local test_dir

  # Create temporary directory
  test_dir=$(create_test_tmpfs "nself-${test_name}")

  # Register cleanup
  ensure_cleanup "rm -rf '$test_dir'"

  # Export for test to use
  export TEST_DIR="$test_dir"
  export TEST_ENV_FILE="$test_dir/.env"

  # Create basic structure
  mkdir -p "$test_dir"/{docker,nginx,ssl,services,monitoring}

  printf "%s\n" "$test_dir"
}

# Run test in isolation
run_isolated_test() {
  local test_func="$1"
  local test_name="${2:-${test_func}}"

  # Create isolated environment
  local test_dir
  test_dir=$(create_test_env "$test_name")

  # Change to test directory
  pushd "$test_dir" >/dev/null || return 1

  # Run test
  local result=0
  $test_func || result=$?

  # Return to original directory
  popd >/dev/null || true

  # Cleanup happens automatically via ensure_cleanup

  return $result
}

# ============================================================================
# Performance Measurement
# ============================================================================

# Measure test execution time
measure_test_time() {
  local test_func="$1"
  local start_time
  local end_time
  local duration

  start_time=$(date +%s)
  $test_func
  local result=$?
  end_time=$(date +%s)

  duration=$((end_time - start_time))

  printf "\033[36mTIME:\033[0m %s took %ds\n" "$test_func" "$duration"

  return $result
}

# Benchmark test (run multiple times and average)
benchmark_test() {
  local test_func="$1"
  local iterations="${2:-10}"
  local total_time=0
  local i

  printf "Benchmarking %s (%d iterations)...\n" "$test_func" "$iterations"

  for ((i=1; i<=iterations; i++)); do
    local start_time
    local end_time

    start_time=$(date +%s)
    $test_func >/dev/null 2>&1
    end_time=$(date +%s)

    local duration=$((end_time - start_time))
    total_time=$((total_time + duration))

    printf "  Iteration %d: %ds\n" "$i" "$duration"
  done

  local avg_time=$((total_time / iterations))
  printf "\033[36mAVERAGE:\033[0m %s took %ds (over %d iterations)\n" \
    "$test_func" "$avg_time" "$iterations"
}

# ============================================================================
# Test Reporting
# ============================================================================

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Run test and track result
run_and_track_test() {
  local test_name="$1"
  local test_func="$2"

  TESTS_RUN=$((TESTS_RUN + 1))

  printf "\n\033[36m[%d/%d]\033[0m Running: %s\n" \
    "$TESTS_RUN" "${TOTAL_TESTS:-0}" "$test_name"

  if $test_func; then
    TESTS_PASSED=$((TESTS_PASSED + 1))
    printf "\033[32m✓ PASS:\033[0m %s\n" "$test_name"
    return 0
  else
    TESTS_FAILED=$((TESTS_FAILED + 1))
    printf "\033[31m✗ FAIL:\033[0m %s\n" "$test_name"
    return 1
  fi
}

# Print test summary
print_test_summary() {
  printf "\n"
  printf "=%.0s" {1..80}
  printf "\n"
  printf "TEST SUMMARY\n"
  printf "=%.0s" {1..80}
  printf "\n"
  printf "Total:   %d\n" "$TESTS_RUN"
  printf "\033[32mPassed:  %d\033[0m\n" "$TESTS_PASSED"
  printf "\033[31mFailed:  %d\033[0m\n" "$TESTS_FAILED"
  printf "\033[33mSkipped: %d\033[0m\n" "$TESTS_SKIPPED"
  printf "=%.0s" {1..80}
  printf "\n"

  # Return non-zero if any tests failed
  [[ $TESTS_FAILED -eq 0 ]]
}

# ============================================================================
# Export Enhanced Functions
# ============================================================================

export -f run_test_with_timeout
export -f retry_test
export -f retry_with_backoff
export -f ensure_cleanup
export -f run_all_cleanups
export -f clear_cleanups
export -f fail_fast
export -f skip_test
export -f expect_failure
export -f is_ci
export -f is_macos
export -f is_linux
export -f is_wsl
export -f get_platform
export -f assert_success
export -f assert_failure
export -f assert_equals
export -f assert_not_equals
export -f assert_contains
export -f assert_not_contains
export -f assert_file_exists
export -f assert_file_not_exists
export -f assert_dir_exists
export -f assert_dir_not_exists
export -f create_test_env
export -f run_isolated_test
export -f measure_test_time
export -f benchmark_test
export -f run_and_track_test
export -f print_test_summary

# ============================================================================
# Usage Example
# ============================================================================

# Source this file in your tests:
#   source "$(dirname "${BASH_SOURCE[0]}")/../lib/test-framework-enhanced.sh"
#
# Then use enhanced features:
#
#   test_my_feature() {
#     local result
#     result=$(my_command)
#     assert_equals "expected" "$result"
#   }
#
#   # Run with timeout
#   run_test_with_timeout test_my_feature 30
#
#   # Run with retry
#   retry_test test_my_feature 3
#
#   # Run in isolation
#   run_isolated_test test_my_feature
#
#   # Cleanup automatically
#   ensure_cleanup "rm -rf /tmp/test-data"
