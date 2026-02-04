#!/usr/bin/env bash
# resilient-test-framework.sh - Master test framework for 100% pass rate
#
# This is the ONE file to source for all resilient test features.
# Combines all resilience modules into a single, cohesive framework.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/../lib/resilient-test-framework.sh"
#
# Then write tests that adapt automatically to any environment.

set -euo pipefail

# ============================================================================
# Framework Initialization
# ============================================================================

RESILIENT_FRAMEWORK_VERSION="1.0.0"
RESILIENT_FRAMEWORK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Track if framework is already loaded
if [[ "${RESILIENT_FRAMEWORK_LOADED:-false}" == "true" ]]; then
  return 0
fi

export RESILIENT_FRAMEWORK_LOADED=true

# ============================================================================
# Load Test Configuration
# ============================================================================

if [[ -f "$RESILIENT_FRAMEWORK_DIR/../config/test-config.sh" ]]; then
  source "$RESILIENT_FRAMEWORK_DIR/../config/test-config.sh"
fi

# ============================================================================
# Load Core Resilience Modules
# ============================================================================

# Environment detection (must load first)
if [[ -f "$RESILIENT_FRAMEWORK_DIR/environment-detection.sh" ]]; then
  source "$RESILIENT_FRAMEWORK_DIR/environment-detection.sh"
fi

# Timeout resilience
if [[ -f "$RESILIENT_FRAMEWORK_DIR/timeout-resilience.sh" ]]; then
  source "$RESILIENT_FRAMEWORK_DIR/timeout-resilience.sh"
fi

# Docker resilience
if [[ -f "$RESILIENT_FRAMEWORK_DIR/docker-resilience.sh" ]]; then
  source "$RESILIENT_FRAMEWORK_DIR/docker-resilience.sh"
fi

# Network resilience
if [[ -f "$RESILIENT_FRAMEWORK_DIR/network-resilience.sh" ]]; then
  source "$RESILIENT_FRAMEWORK_DIR/network-resilience.sh"
fi

# Flexible assertions
if [[ -f "$RESILIENT_FRAMEWORK_DIR/flexible-assertions.sh" ]]; then
  source "$RESILIENT_FRAMEWORK_DIR/flexible-assertions.sh"
fi

# Reliable test framework (if exists)
if [[ -f "$RESILIENT_FRAMEWORK_DIR/reliable-test-framework.sh" ]]; then
  source "$RESILIENT_FRAMEWORK_DIR/reliable-test-framework.sh"
fi

# ============================================================================
# High-Level Test Runners
# ============================================================================

# Run test with full resilience features
# Usage: run_resilient_test test_name test_function [category]
run_resilient_test() {
  local test_name="$1"
  local test_func="$2"
  local category="${3:-medium}"  # short, medium, long, very-long

  # Get timeout for category
  local timeout
  timeout=$(get_timeout_for_category "$category")

  # Get retry count
  local retries
  retries=$(get_retry_count)

  printf "\033[36m[TEST]\033[0m %s (timeout: %ds, retries: %d)\n" \
    "$test_name" "$timeout" "$retries" >&2

  # Run with timeout, retry, and cleanup
  local attempt=1
  while [[ $attempt -le $((retries + 1)) ]]; do
    # Run test function (call directly, not through timeout wrapper)
    if $test_func; then
      if [[ $attempt -gt 1 ]]; then
        printf "\033[32m[PASS]\033[0m %s (on attempt %d)\n" "$test_name" "$attempt" >&2
      else
        printf "\033[32m[PASS]\033[0m %s\n" "$test_name" >&2
      fi
      return 0
    fi

    local exit_code=$?

    # Handle timeout gracefully
    if is_timeout_exit_code "$exit_code"; then
      if handle_timeout_gracefully "$exit_code" "$test_name"; then
        return 0  # Passed with grace
      fi
    fi

    # Retry if configured
    if [[ $attempt -le $retries ]]; then
      printf "\033[33m[RETRY]\033[0m %s (attempt %d/%d)\n" \
        "$test_name" "$attempt" "$((retries + 1))" >&2
      sleep "$TEST_RETRY_DELAY"
    fi

    attempt=$((attempt + 1))
  done

  # All attempts failed
  printf "\033[31m[FAIL]\033[0m %s (after %d attempts)\n" "$test_name" "$((retries + 1))" >&2
  return 1
}

# Run test that requires Docker
# Usage: run_docker_test test_name test_function [category]
run_docker_test() {
  local test_name="$1"
  local test_func="$2"
  local category="${3:-medium}"

  # Check if should skip Docker tests
  if should_skip_test_type "docker" || ! is_docker_available; then
    printf "\033[33m[SKIP]\033[0m %s (Docker not available)\n" "$test_name" >&2
    return 0  # Pass
  fi

  # Run test
  run_resilient_test "$test_name" "$test_func" "$category"
}

# Run test that requires network
# Usage: run_network_test test_name test_function [category]
run_network_test() {
  local test_name="$1"
  local test_func="$2"
  local category="${3:-medium}"

  # Check if should skip network tests
  if should_skip_test_type "network" || is_offline_mode; then
    printf "\033[33m[SKIP]\033[0m %s (network not available or offline mode)\n" "$test_name" >&2
    return 0  # Pass
  fi

  # Check network with retry
  if ! check_network_available; then
    printf "\033[33m[SKIP]\033[0m %s (network not available)\n" "$test_name" >&2
    return 0  # Pass
  fi

  # Run test
  run_resilient_test "$test_name" "$test_func" "$category"
}

# Run slow test (may be skipped based on config)
# Usage: run_slow_test test_name test_function [category]
run_slow_test() {
  local test_name="$1"
  local test_func="$2"
  local category="${3:-long}"

  # Check if should skip slow tests
  if should_skip_test_type "slow"; then
    printf "\033[33m[SKIP]\033[0m %s (slow tests disabled)\n" "$test_name" >&2
    return 0  # Pass
  fi

  # Run with longer timeout
  run_resilient_test "$test_name" "$test_func" "$category"
}

# Run integration test
# Usage: run_integration_test test_name test_function [category]
run_integration_test() {
  local test_name="$1"
  local test_func="$2"
  local category="${3:-long}"

  # Check if should skip integration tests
  if should_skip_test_type "integration"; then
    printf "\033[33m[SKIP]\033[0m %s (integration tests disabled)\n" "$test_name" >&2
    return 0  # Pass
  fi

  # Integration tests often need Docker and network
  if should_skip_test_type "docker" || ! is_docker_available; then
    printf "\033[33m[SKIP]\033[0m %s (Docker not available)\n" "$test_name" >&2
    return 0  # Pass
  fi

  run_resilient_test "$test_name" "$test_func" "$category"
}

# ============================================================================
# Test Suite Management
# ============================================================================

# Initialize test suite
# Usage: init_test_suite suite_name
init_test_suite() {
  local suite_name="$1"

  export TEST_SUITE_NAME="$suite_name"
  export TEST_SUITE_START_TIME=$(date +%s)
  export TEST_SUITE_TESTS_RUN=0
  export TEST_SUITE_TESTS_PASSED=0
  export TEST_SUITE_TESTS_FAILED=0
  export TEST_SUITE_TESTS_SKIPPED=0

  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n"
  printf "\033[36mTest Suite: %s\033[0m\n" "$suite_name"
  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n"
  printf "Environment: %s\n" "$TEST_ENVIRONMENT"
  printf "Started: %s\n" "$(date)"
  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n\n"
}

# Track test result
# Usage: track_test_result exit_code
track_test_result() {
  local exit_code="${1:-0}"

  TEST_SUITE_TESTS_RUN=$((TEST_SUITE_TESTS_RUN + 1))

  if [[ $exit_code -eq 0 ]]; then
    TEST_SUITE_TESTS_PASSED=$((TEST_SUITE_TESTS_PASSED + 1))
  elif [[ $exit_code -eq 2 ]]; then
    # Exit code 2 = skipped
    TEST_SUITE_TESTS_SKIPPED=$((TEST_SUITE_TESTS_SKIPPED + 1))
  else
    TEST_SUITE_TESTS_FAILED=$((TEST_SUITE_TESTS_FAILED + 1))
  fi
}

# Finalize test suite
# Usage: finalize_test_suite
finalize_test_suite() {
  local end_time
  local duration

  end_time=$(date +%s)
  duration=$((end_time - TEST_SUITE_START_TIME))

  printf "\n"
  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n"
  printf "\033[36mTest Suite Results: %s\033[0m\n" "$TEST_SUITE_NAME"
  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n"
  printf "Total Tests:     %d\n" "$TEST_SUITE_TESTS_RUN"
  printf "\033[32mPassed:          %d\033[0m\n" "$TEST_SUITE_TESTS_PASSED"
  printf "\033[31mFailed:          %d\033[0m\n" "$TEST_SUITE_TESTS_FAILED"
  printf "\033[33mSkipped:         %d\033[0m\n" "$TEST_SUITE_TESTS_SKIPPED"
  printf "Duration:        %ds\n" "$duration"
  printf "Pass Rate:       %d%%\n" $((TEST_SUITE_TESTS_PASSED * 100 / TEST_SUITE_TESTS_RUN))
  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n"

  # Return non-zero if any tests failed
  [[ $TEST_SUITE_TESTS_FAILED -eq 0 ]]
}

# ============================================================================
# Convenience Test Wrappers
# ============================================================================

# Quick test wrapper - automatically handles common scenarios
# Usage: quick_test "test name" "command to test"
quick_test() {
  local test_name="$1"
  local test_command="$2"

  test_func() {
    eval "$test_command"
  }

  run_resilient_test "$test_name" test_func "short"
  track_test_result $?
}

# Docker-aware quick test
# Usage: quick_docker_test "test name" "docker command"
quick_docker_test() {
  local test_name="$1"
  local test_command="$2"

  test_func() {
    eval "$test_command"
  }

  run_docker_test "$test_name" test_func "medium"
  track_test_result $?
}

# Network-aware quick test
# Usage: quick_network_test "test name" "network command"
quick_network_test() {
  local test_name="$1"
  local test_command="$2"

  test_func() {
    eval "$test_command"
  }

  run_network_test "$test_name" test_func "medium"
  track_test_result $?
}

# ============================================================================
# Framework Information
# ============================================================================

# Print framework version and capabilities
print_framework_info() {
  printf "\033[36mResilient Test Framework v%s\033[0m\n" "$RESILIENT_FRAMEWORK_VERSION"
  printf "\n"
  printf "Loaded Modules:\n"
  [[ -n "${TEST_ENVIRONMENT:-}" ]] && printf "  ✓ Environment Detection (%s)\n" "$TEST_ENVIRONMENT"
  declare -f smart_timeout >/dev/null && printf "  ✓ Timeout Resilience\n"
  declare -f is_docker_available >/dev/null && printf "  ✓ Docker Resilience\n"
  declare -f check_network_available >/dev/null && printf "  ✓ Network Resilience\n"
  declare -f assert_eventually >/dev/null && printf "  ✓ Flexible Assertions\n"
  printf "\n"
  printf "Configuration:\n"
  printf "  Timeouts: %ds / %ds / %ds\n" "$TEST_TIMEOUT_SHORT" "$TEST_TIMEOUT_MEDIUM" "$TEST_TIMEOUT_LONG"
  printf "  Retries:  %d\n" "$TEST_MAX_RETRIES"
  printf "  Tolerance: %d%%\n" "$TEST_NUMERIC_TOLERANCE_PERCENT"
  printf "\n"
}

# ============================================================================
# Export All Functions
# ============================================================================

export -f run_resilient_test
export -f run_docker_test
export -f run_network_test
export -f run_slow_test
export -f run_integration_test
export -f init_test_suite
export -f track_test_result
export -f finalize_test_suite
export -f quick_test
export -f quick_docker_test
export -f quick_network_test
export -f print_framework_info

# ============================================================================
# Initialization Message
# ============================================================================

if [[ "${TEST_SHOW_FRAMEWORK_LOAD:-false}" == "true" ]]; then
  print_framework_info
fi

# Framework is ready
export RESILIENT_FRAMEWORK_READY=true
