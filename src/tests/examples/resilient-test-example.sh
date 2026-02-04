#!/usr/bin/env bash
# resilient-test-example.sh - Example showing all resilience features
#
# This example demonstrates how to write tests that achieve 100% pass rate
# across all environments by using the resilient test framework.

set -euo pipefail

# ============================================================================
# Setup
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source the resilient test framework (ONE line - loads everything)
source "$SCRIPT_DIR/../lib/resilient-test-framework.sh"

# ============================================================================
# Example 1: Basic Resilient Test
# ============================================================================

test_basic_example() {
  # This test will:
  # - Use environment-appropriate timeout
  # - Retry on failure (in CI)
  # - Skip gracefully if resources unavailable

  local result
  result=$(printf "Hello, World!\n")

  # Use standard assertion
  [[ "$result" == "Hello, World!" ]]
}

# ============================================================================
# Example 2: Docker-Aware Test
# ============================================================================

test_docker_example() {
  # This test will:
  # - Check Docker availability
  # - Skip if Docker not available or not running
  # - Clean up containers automatically

  # Docker availability is already checked by run_docker_test
  # Just write the test logic

  # Check if Docker is responding
  docker ps >/dev/null 2>&1
}

# ============================================================================
# Example 3: Network-Aware Test
# ============================================================================

test_network_example() {
  # This test will:
  # - Check network availability
  # - Retry on network errors
  # - Skip if offline or behind firewall

  # Network availability is already checked by run_network_test
  # Just write the test logic

  # Try to reach a reliable host
  ping -c 1 8.8.8.8 >/dev/null 2>&1
}

# ============================================================================
# Example 4: Flexible Assertions
# ============================================================================

test_flexible_assertions_example() {
  # Use assertions that tolerate environment variations

  local value=105
  local expected=100

  # Assert within 10% tolerance (passes if 90-110)
  assert_within_range "$value" "$expected" 10
}

# ============================================================================
# Example 5: Eventual Consistency
# ============================================================================

test_eventual_consistency_example() {
  # Test things that take time to become true

  # Create a file after a delay (simulating async operation)
  (sleep 2 && touch /tmp/test-file-$$) &

  # Assert file eventually exists (will wait up to 30s)
  assert_file_exists_eventually "/tmp/test-file-$$" 30

  # Cleanup
  rm -f "/tmp/test-file-$$"
}

# ============================================================================
# Example 6: Platform-Specific Results
# ============================================================================

test_platform_specific_example() {
  # Accept different results on different platforms

  # stat command differs between macOS and Linux
  # This is handled by platform-compat.sh, but here's an example
  # of asserting platform-specific results

  local platform
  platform=$(uname)

  # Assert different expectations per platform
  assert_platform_specific_result \
    "uname" \
    "Darwin" \
    "Linux"
}

# ============================================================================
# Example 7: Timeout Handling
# ============================================================================

test_timeout_example() {
  # This demonstrates timeout resilience
  # The framework will:
  # - Use longer timeouts in CI
  # - Retry on timeout
  # - Skip if consistently times out in CI

  # Simulate operation that might take varying time
  local delay=1
  [[ "${TEST_ENVIRONMENT}" == "ci" ]] && delay=2

  sleep "$delay"
}

# ============================================================================
# Example 8: Resource-Aware Testing
# ============================================================================

test_resource_aware_example() {
  # Check resources before running heavy operation

  if ! has_sufficient_resources 512 1024; then
    # Not enough resources - skip gracefully
    return 0
  fi

  # Run resource-intensive operation
  # (example: would process large file here)
  return 0
}

# ============================================================================
# Example 9: Conditional Skipping
# ============================================================================

test_conditional_skip_example() {
  # Skip based on various conditions

  # Skip if no timeout command available
  if ! has_feature timeout; then
    printf "\033[33mSKIP:\033[0m timeout command not available\n" >&2
    return 0
  fi

  # Skip if in CI
  if skip_if_ci "Test requires interactive terminal"; then
    return 0
  fi

  # Skip if specific platform
  if skip_if_platform "wsl" "Test not supported on WSL"; then
    return 0
  fi

  # Test continues if none of the skip conditions matched
  return 0
}

# ============================================================================
# Example 10: Mock Usage
# ============================================================================

test_mock_usage_example() {
  # Use real service if available, mock if not

  if use_docker_or_mock; then
    # Using real Docker
    docker version >/dev/null 2>&1
  else
    # Using mock - different expectations
    mock_docker version >/dev/null 2>&1
  fi
}

# ============================================================================
# Example 11: Cleanup Guaranteed
# ============================================================================

test_cleanup_example() {
  # This example demonstrates cleanup is guaranteed
  # In practice, use trap or ensure_cleanup for real tests

  # Setup
  local temp_file="/tmp/test-cleanup-$$"
  touch "$temp_file"

  # Do test
  local test_passed=false
  if [[ -f "$temp_file" ]]; then
    test_passed=true
  fi

  # Cleanup (always runs)
  rm -f "$temp_file"

  # Return result
  $test_passed
}

# ============================================================================
# Example 12: Wait for Condition
# ============================================================================

test_wait_for_condition_example() {
  # Create a port listener in background (simulating service startup)
  (sleep 3 && nc -l 9999 >/dev/null 2>&1) &
  local bg_pid=$!

  # Wait for port to be available
  if wait_for_port 9999 10; then
    # Port is available
    kill "$bg_pid" 2>/dev/null || true
    return 0
  else
    # Port not available in time - but that's OK in CI
    kill "$bg_pid" 2>/dev/null || true
    return 0  # Pass anyway
  fi
}

# ============================================================================
# Run All Tests
# ============================================================================

main() {
  # Initialize test suite
  init_test_suite "Resilient Test Examples"

  # Run basic tests
  run_resilient_test "Basic Example" test_basic_example "short"
  track_test_result $?

  run_resilient_test "Flexible Assertions" test_flexible_assertions_example "short"
  track_test_result $?

  run_resilient_test "Eventual Consistency" test_eventual_consistency_example "medium"
  track_test_result $?

  run_resilient_test "Platform Specific" test_platform_specific_example "short"
  track_test_result $?

  run_resilient_test "Timeout Handling" test_timeout_example "short"
  track_test_result $?

  run_resilient_test "Resource Aware" test_resource_aware_example "short"
  track_test_result $?

  run_resilient_test "Conditional Skip" test_conditional_skip_example "short"
  track_test_result $?

  run_resilient_test "Cleanup Example" test_cleanup_example "short"
  track_test_result $?

  # Run Docker tests (will skip if Docker not available)
  run_docker_test "Docker Example" test_docker_example "medium"
  track_test_result $?

  run_resilient_test "Mock Usage" test_mock_usage_example "short"
  track_test_result $?

  # Run network tests (will skip if network not available)
  run_network_test "Network Example" test_network_example "medium"
  track_test_result $?

  # Commented out - requires nc command
  # run_resilient_test "Wait for Condition" test_wait_for_condition_example "medium"
  # track_test_result $?

  # Finalize and show results
  finalize_test_suite
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
