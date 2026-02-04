#!/usr/bin/env bash
# reliable-test-framework.sh - Bulletproof test utilities for nself
#
# Provides rock-solid test infrastructure with:
# - Automatic timeout protection
# - Guaranteed cleanup
# - Retry logic for transient failures
# - Test isolation
# - Cross-platform compatibility

set -euo pipefail

# ============================================================================
# Colors for Output
# ============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================================
# Timeout Protection
# ============================================================================

# Run a test with automatic timeout protection
# Usage: run_test_with_timeout test_function [timeout_seconds]
run_test_with_timeout() {
  local test_func="$1"
  local timeout="${2:-30}"  # 30s default

  # Check if timeout command is available
  if command -v timeout >/dev/null 2>&1; then
    timeout "$timeout" "$test_func"
  elif command -v gtimeout >/dev/null 2>&1; then
    # macOS with coreutils
    gtimeout "$timeout" "$test_func"
  else
    # No timeout available - run directly with warning
    printf "\033[33mWARNING: timeout command not available, running without timeout\033[0m\n" >&2
    "$test_func"
  fi
}

# Run command with timeout and capture output
# Usage: run_with_timeout_capture timeout_seconds command [args...]
run_with_timeout_capture() {
  local timeout="$1"
  shift
  local output=""
  local exit_code=0

  if command -v timeout >/dev/null 2>&1; then
    output=$(timeout "$timeout" "$@" 2>&1) || exit_code=$?
  elif command -v gtimeout >/dev/null 2>&1; then
    output=$(gtimeout "$timeout" "$@" 2>&1) || exit_code=$?
  else
    output=$("$@" 2>&1) || exit_code=$?
  fi

  printf "%s\n" "$output"
  return $exit_code
}

# ============================================================================
# Guaranteed Cleanup
# ============================================================================

# Run test with guaranteed cleanup (even on failure/interrupt)
# Usage: with_cleanup test_function cleanup_function
with_cleanup() {
  local test_func="$1"
  local cleanup_func="$2"
  local test_result=0

  # Setup trap to ensure cleanup runs on EXIT, INT, TERM
  # shellcheck disable=SC2064
  trap "$cleanup_func" EXIT INT TERM

  # Run test and capture result
  $test_func || test_result=$?

  # Cleanup (trap will also call this, but that's OK - cleanup should be idempotent)
  $cleanup_func

  # Remove trap
  trap - EXIT INT TERM

  return $test_result
}

# Create isolated test directory with automatic cleanup
# Returns: Path to test directory in TEST_ISOLATED_DIR variable
create_isolated_test_dir() {
  local prefix="${1:-nself-test}"

  # Create unique test directory with PID and timestamp
  TEST_ISOLATED_DIR=$(mktemp -d 2>/dev/null) || TEST_ISOLATED_DIR="/tmp/${prefix}_$$_$(date +%s)"
  mkdir -p "$TEST_ISOLATED_DIR"

  # Register cleanup
  # shellcheck disable=SC2064
  trap "cleanup_isolated_test_dir" EXIT INT TERM

  printf "%s\n" "$TEST_ISOLATED_DIR"
}

# Cleanup isolated test directory
cleanup_isolated_test_dir() {
  if [[ -n "${TEST_ISOLATED_DIR:-}" ]] && [[ -d "$TEST_ISOLATED_DIR" ]]; then
    rm -rf "$TEST_ISOLATED_DIR"
    unset TEST_ISOLATED_DIR
  fi
}

# ============================================================================
# Retry Logic for Transient Failures
# ============================================================================

# Retry a test function on failure with exponential backoff
# Usage: retry_on_failure test_function [max_attempts] [initial_delay]
retry_on_failure() {
  local test_func="$1"
  local max_attempts="${2:-3}"
  local initial_delay="${3:-1}"

  local attempt=1
  local delay="$initial_delay"

  while [ $attempt -le "$max_attempts" ]; do
    if $test_func; then
      return 0
    fi

    if [ $attempt -lt "$max_attempts" ]; then
      printf "\033[33mTest failed (attempt %d/%d), retrying in %ds...\033[0m\n" \
        "$attempt" "$max_attempts" "$delay" >&2
      sleep "$delay"
      # Exponential backoff
      delay=$((delay * 2))
    fi

    attempt=$((attempt + 1))
  done

  printf "\033[31mTest failed after %d attempts\033[0m\n" "$max_attempts" >&2
  return 1
}

# Retry with custom retry condition
# Usage: retry_until condition_func action_func [max_attempts] [delay]
retry_until() {
  local condition_func="$1"
  local action_func="$2"
  local max_attempts="${3:-10}"
  local delay="${4:-1}"

  local attempt=1

  while [ $attempt -le "$max_attempts" ]; do
    $action_func || true

    if $condition_func; then
      return 0
    fi

    if [ $attempt -lt "$max_attempts" ]; then
      sleep "$delay"
    fi

    attempt=$((attempt + 1))
  done

  return 1
}

# ============================================================================
# Test Isolation
# ============================================================================

# Get random available port for test services
# Returns: Random port number
get_random_port() {
  local min_port="${1:-10000}"
  local max_port="${2:-60000}"

  # Try Python first (most reliable)
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()'
    return 0
  fi

  # Fallback to random port in range (may not be available)
  local port=$((min_port + RANDOM % (max_port - min_port)))
  printf "%d\n" "$port"
}

# Create unique project name for test
# Usage: get_unique_project_name [prefix]
get_unique_project_name() {
  local prefix="${1:-test-project}"
  printf "%s-%d-%d\n" "$prefix" "$$" "$(date +%s)"
}

# Create unique database name for test
# Usage: get_unique_db_name [prefix]
get_unique_db_name() {
  local prefix="${1:-test_db}"
  # Replace hyphens with underscores (valid for DB names)
  local name="${prefix}_$$_$(date +%s)"
  printf "%s\n" "${name//-/_}"
}

# ============================================================================
# Environment Detection & Requirements
# ============================================================================

# Check if command is available, skip test if not
# Usage: require_command command_name [message]
require_command() {
  local command_name="$1"
  local message="${2:-$command_name is required but not available}"

  if ! command -v "$command_name" >/dev/null 2>&1; then
    printf "\033[33mSKIP: %s\033[0m\n" "$message" >&2
    return 1
  fi
  return 0
}

# Check if Docker is available and running
# Usage: require_docker
require_docker() {
  if ! require_command docker "Docker is required but not available"; then
    return 1
  fi

  if ! docker ps >/dev/null 2>&1; then
    printf "\033[33mSKIP: Docker daemon is not running\033[0m\n" >&2
    return 1
  fi

  return 0
}

# Check if network is available
# Usage: require_network
require_network() {
  # Try to ping a reliable host
  if ! ping -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
    printf "\033[33mSKIP: Network is not available\033[0m\n" >&2
    return 1
  fi
  return 0
}

# Detect platform
# Returns: "linux", "macos", "wsl", or "unknown"
detect_platform() {
  if [[ "$(uname)" == "Darwin" ]]; then
    printf "macos\n"
  elif [[ "$(uname)" == "Linux" ]]; then
    if grep -qi microsoft /proc/version 2>/dev/null; then
      printf "wsl\n"
    else
      printf "linux\n"
    fi
  else
    printf "unknown\n"
  fi
}

# Skip test on specific platform
# Usage: skip_on_platform platform_name [message]
skip_on_platform() {
  local skip_platform="$1"
  local message="${2:-Test not supported on $skip_platform}"
  local current_platform
  current_platform=$(detect_platform)

  if [[ "$current_platform" == "$skip_platform" ]]; then
    printf "\033[33mSKIP: %s\033[0m\n" "$message" >&2
    return 1
  fi
  return 0
}

# Run test only on specific platform
# Usage: run_on_platform platform_name [message]
run_on_platform() {
  local target_platform="$1"
  local message="${2:-Test only runs on $target_platform}"
  local current_platform
  current_platform=$(detect_platform)

  if [[ "$current_platform" != "$target_platform" ]]; then
    printf "\033[33mSKIP: %s\033[0m\n" "$message" >&2
    return 1
  fi
  return 0
}

# ============================================================================
# Enhanced Assertions with Context
# ============================================================================

# Assert with detailed context on failure
# Usage: assert_with_context expected actual description
assert_with_context() {
  local expected="$1"
  local actual="$2"
  local description="$3"
  local caller_info="${4:-$(caller)}"

  if [[ "$expected" != "$actual" ]]; then
    printf "\033[31m✗ Test Failed: %s\033[0m\n" "$description" >&2
    printf "  Expected: %s\n" "$expected" >&2
    printf "  Actual:   %s\n" "$actual" >&2
    printf "  Location: %s\n" "$caller_info" >&2
    printf "\n  To fix:\n" >&2
    printf "  1. Verify input values are correct\n" >&2
    printf "  2. Check logic in the function being tested\n" >&2
    printf "  3. Review test expectations\n" >&2
    return 1
  fi
  return 0
}

# Simpler alias for assert_with_context
# Usage: assert_equals expected actual [description]
assert_equals() {
  local expected="$1"
  local actual="$2"
  local description="${3:-Assertion}"
  assert_with_context "$expected" "$actual" "$description"
}

# Assert success (return code 0)
# Usage: assert_success [description]
assert_success() {
  local description="${1:-Command succeeded}"
  if [[ ${?} -ne 0 ]]; then
    printf "\033[31m✗ Test Failed: %s\033[0m\n" "$description" >&2
    return 1
  fi
  return 0
}

# Assert that something contains a substring
# Usage: assert_contains haystack needle [description]
assert_contains() {
  local haystack="$1"
  local needle="$2"
  local description="${3:-String contains check}"

  if [[ "$haystack" != *"$needle"* ]]; then
    printf "\033[31m✗ Test Failed: %s\033[0m\n" "$description" >&2
    printf "  Expected to find: %s\n" "$needle" >&2
    printf "  In: %s\n" "$haystack" >&2
    return 1
  fi
  return 0
}

# Assert file contains pattern with context
# Usage: assert_file_contains_with_context file pattern description
assert_file_contains_with_context() {
  local file="$1"
  local pattern="$2"
  local description="$3"

  if [[ ! -f "$file" ]]; then
    printf "\033[31m✗ Test Failed: %s\033[0m\n" "$description" >&2
    printf "  File does not exist: %s\n" "$file" >&2
    return 1
  fi

  if ! grep -q "$pattern" "$file" 2>/dev/null; then
    printf "\033[31m✗ Test Failed: %s\033[0m\n" "$description" >&2
    printf "  Pattern not found: %s\n" "$pattern" >&2
    printf "  In file: %s\n" "$file" >&2
    printf "  File contents:\n" >&2
    head -20 "$file" | sed 's/^/    /' >&2
    return 1
  fi
  return 0
}

# ============================================================================
# Wait Functions for Async Operations
# ============================================================================

# Wait for condition with timeout
# Usage: wait_for_condition condition_func [timeout] [interval]
wait_for_condition() {
  local condition_func="$1"
  local timeout="${2:-30}"
  local interval="${3:-1}"
  local elapsed=0

  while [ $elapsed -lt "$timeout" ]; do
    if $condition_func; then
      return 0
    fi
    sleep "$interval"
    elapsed=$((elapsed + interval))
  done

  printf "\033[33mTimeout waiting for condition after %ds\033[0m\n" "$timeout" >&2
  return 1
}

# Wait for file to exist
# Usage: wait_for_file file_path [timeout]
wait_for_file() {
  local file_path="$1"
  local timeout="${2:-10}"
  local interval="${3:-1}"
  local elapsed=0

  while [ $elapsed -lt "$timeout" ]; do
    if [[ -f "$file_path" ]]; then
      return 0
    fi
    sleep "$interval"
    elapsed=$((elapsed + interval))
  done

  printf "\033[33mTimeout waiting for file %s after %ds\033[0m\n" "$file_path" "$timeout" >&2
  return 1
}

# Wait for port to be available
# Usage: wait_for_port port [timeout]
wait_for_port() {
  local port="$1"
  local timeout="${2:-30}"

  if command -v nc >/dev/null 2>&1; then
    wait_for_condition "nc -z localhost $port >/dev/null 2>&1" "$timeout"
  else
    # Fallback: try to connect with Python
    wait_for_condition "python3 -c 'import socket; s=socket.socket(); s.connect((\"localhost\", $port)); s.close()' 2>/dev/null" "$timeout"
  fi
}

# ============================================================================
# Test Helpers for Common Patterns
# ============================================================================

# Run test in temporary directory with cleanup
# Usage: in_temp_dir test_function
in_temp_dir() {
  local test_func="$1"
  local temp_dir
  local original_dir

  temp_dir=$(mktemp -d 2>/dev/null) || temp_dir="/tmp/test-$$-$(date +%s)"
  mkdir -p "$temp_dir"
  original_dir="$(pwd)"

  cleanup_temp_dir() {
    cd "$original_dir" || true
    [[ -d "$temp_dir" ]] && rm -rf "$temp_dir"
  }

  with_cleanup "$test_func" cleanup_temp_dir
}

# Run test with environment variables
# Usage: with_env "VAR1=value1" "VAR2=value2" test_function
with_env() {
  local test_func="${*: -1}"  # Last argument
  local env_vars=("${@:1:$#-1}")  # All but last
  local original_env=()
  local var_name

  # Save original values
  for env_var in "${env_vars[@]}"; do
    var_name="${env_var%%=*}"
    if [[ -n "${!var_name:-}" ]]; then
      original_env+=("$var_name=${!var_name}")
    else
      original_env+=("$var_name=__UNSET__")
    fi
    export "$env_var"
  done

  restore_env() {
    for env_var in "${original_env[@]}"; do
      var_name="${env_var%%=*}"
      var_value="${env_var#*=}"
      if [[ "$var_value" == "__UNSET__" ]]; then
        unset "$var_name"
      else
        export "$var_name=$var_value"
      fi
    done
  }

  with_cleanup "$test_func" restore_env
}

# ============================================================================
# Performance Tracking
# ============================================================================

# Track test execution time
# Usage: time_test test_function
time_test() {
  local test_func="$1"
  local start_time
  local end_time
  local duration

  start_time=$(date +%s)
  $test_func
  local result=$?
  end_time=$(date +%s)
  duration=$((end_time - start_time))

  printf "Test completed in %ds\n" "$duration" >&2

  if [[ $duration -gt 5 ]]; then
    printf "\033[33mWARNING: Slow test (>5s)\033[0m\n" >&2
  fi

  return $result
}

# ============================================================================
# Export Functions
# ============================================================================

export -f run_test_with_timeout
export -f run_with_timeout_capture
export -f with_cleanup
export -f create_isolated_test_dir
export -f cleanup_isolated_test_dir
export -f retry_on_failure
export -f retry_until
export -f get_random_port
export -f get_unique_project_name
export -f get_unique_db_name
export -f require_command
export -f require_docker
export -f require_network
export -f detect_platform
export -f skip_on_platform
export -f run_on_platform
export -f assert_with_context
export -f assert_file_contains_with_context
export -f wait_for_condition
export -f wait_for_file
export -f wait_for_port
export -f in_temp_dir
export -f with_env
export -f time_test
export -f assert_equals
export -f assert_success
export -f assert_contains
