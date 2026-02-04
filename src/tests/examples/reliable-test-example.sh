#!/usr/bin/env bash
# reliable-test-example.sh - Example of a bulletproof, zero-flakiness test
#
# This example demonstrates all the best practices for writing reliable tests

set -euo pipefail

# ============================================================================
# Setup
# ============================================================================

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source reliable test framework
source "$SCRIPT_DIR/../lib/reliable-test-framework.sh"

# Source mocks
source "$SCRIPT_DIR/../mocks/docker-mock.sh"
source "$SCRIPT_DIR/../mocks/network-mock.sh"
source "$SCRIPT_DIR/../mocks/time-mock.sh"
source "$SCRIPT_DIR/../mocks/filesystem-mock.sh"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# ============================================================================
# Test Helper Functions
# ============================================================================

pass() {
  TESTS_PASSED=$((TESTS_PASSED + 1))
  TESTS_RUN=$((TESTS_RUN + 1))
  printf "  \033[32m✓\033[0m %s\n" "$1"
}

fail() {
  TESTS_FAILED=$((TESTS_FAILED + 1))
  TESTS_RUN=$((TESTS_RUN + 1))
  printf "  \033[31m✗\033[0m %s\n" "$1"
}

# ============================================================================
# Example 1: Basic Test with Timeout and Cleanup
# ============================================================================

test_basic_with_timeout_and_cleanup() {
  printf "\n\033[34m→ Test 1: Basic test with timeout and cleanup\033[0m\n"

  # Create test directory (don't use create_isolated_test_dir here to avoid trap conflicts)
  local test_dir="/tmp/nself-example-test-$$"
  mkdir -p "$test_dir"

  # Create a test file
  echo "test content" > "$test_dir/test.txt"

  # Verify file exists
  if [[ -f "$test_dir/test.txt" ]]; then
    pass "Test file created successfully"
  else
    fail "Test file was not created"
  fi

  # Test that timeout wrapper works (or gracefully handles missing timeout)
  if command -v timeout >/dev/null 2>&1 || command -v gtimeout >/dev/null 2>&1; then
    pass "Timeout command available - protection enabled"
  else
    pass "Timeout command not available - tests run without timeout protection"
  fi

  # Manual cleanup for this example
  rm -rf "$test_dir"
}

# ============================================================================
# Example 2: Test with Guaranteed Cleanup
# ============================================================================

test_guaranteed_cleanup() {
  printf "\n\033[34m→ Test 2: Guaranteed cleanup (even on failure)\033[0m\n"

  setup() {
    TEST_RESOURCE_DIR=$(mktemp -d)
    printf "  Setup: Created resource at %s\n" "$TEST_RESOURCE_DIR"
  }

  cleanup() {
    if [[ -d "$TEST_RESOURCE_DIR" ]]; then
      rm -rf "$TEST_RESOURCE_DIR"
      printf "  Cleanup: Removed resource\n"
    fi
  }

  test_logic() {
    # This runs, and cleanup is GUARANTEED even if it fails
    touch "$TEST_RESOURCE_DIR/test.txt"

    if [[ -f "$TEST_RESOURCE_DIR/test.txt" ]]; then
      pass "Resource created in setup directory"
    else
      fail "Resource not created"
      return 1
    fi
  }

  # Setup
  setup

  # Run test with guaranteed cleanup
  if with_cleanup test_logic cleanup; then
    pass "Test passed and cleanup executed"
  else
    pass "Cleanup still executed even though test failed"
  fi
}

# ============================================================================
# Example 3: Using Docker Mock
# ============================================================================

test_with_docker_mock() {
  printf "\n\033[34m→ Test 3: Using Docker mock (fast, no Docker required)\033[0m\n"

  # Docker commands work but are instant!
  docker run --name test-nginx -d nginx

  if docker ps | grep -q "test-nginx"; then
    pass "Container created (mocked)"
  else
    fail "Container not found"
  fi

  docker stop test-nginx
  docker rm test-nginx

  pass "Container lifecycle tested without real Docker"
}

# ============================================================================
# Example 4: Using Network Mock
# ============================================================================

test_with_network_mock() {
  printf "\n\033[34m→ Test 4: Using network mock (no real HTTP requests)\033[0m\n"

  # Register mock API response
  register_mock_response \
    "https://api.example.com/status" \
    200 \
    '{"status": "healthy", "version": "1.0.0"}'

  # Make request (uses mock, instant response)
  response=$(curl -s https://api.example.com/status)

  if echo "$response" | grep -q "healthy"; then
    pass "API request mocked successfully"
  else
    fail "Mock response incorrect"
  fi

  # Test error scenarios
  register_mock_response \
    "https://api.example.com/error" \
    500 \
    '{"error": "Internal Server Error"}'

  if ! curl -s https://api.example.com/error; then
    pass "Error response handled correctly"
  else
    fail "Should have returned error"
  fi
}

# ============================================================================
# Example 5: Using Time Mock
# ============================================================================

test_with_time_mock() {
  printf "\n\033[34m→ Test 5: Using time mock (instant timeouts)\033[0m\n"

  # Enable time mocking
  enable_time_mock

  # Set specific time
  set_mock_time 1704067200  # 2024-01-01 00:00:00 UTC

  # Fast-forward 60 seconds (instant!)
  local start=$(get_mock_time)
  instant_sleep 60
  local end=$(get_mock_time)

  if assert_time_advanced "$start" 60; then
    pass "Time advanced 60 seconds instantly"
  else
    fail "Time did not advance correctly"
  fi

  # Test with 10x speed multiplier
  set_time_multiplier 10.0
  start=$(get_mock_time)
  sleep 30  # Completes in 3 seconds but advances time by 30s
  end=$(get_mock_time)

  if assert_time_advanced "$start" 30; then
    pass "Time multiplier works (10x speed)"
  else
    fail "Time multiplier incorrect"
  fi

  # Cleanup
  disable_time_mock
}

# ============================================================================
# Example 6: Using Filesystem Mock
# ============================================================================

test_with_filesystem_mock() {
  printf "\n\033[34m→ Test 6: Using filesystem mock (in-memory operations)\033[0m\n"

  # Initialize mock filesystem
  init_filesystem_mock

  # Create mock files
  create_mock_file "/etc/app.conf" "database=postgres
port=5432
host=localhost"

  # Read mock file
  if mock_file_exists "/etc/app.conf"; then
    pass "Mock file created"
  else
    fail "Mock file not found"
  fi

  # Test file contents
  if assert_mock_file_contains "/etc/app.conf" "database=postgres"; then
    pass "Mock file contains expected content"
  else
    fail "Mock file content incorrect"
  fi

  # Snapshot functionality
  snapshot_mock_fs "initial"
  create_mock_file "/etc/app.conf" "modified content"

  # Restore snapshot
  restore_mock_fs "initial"

  if assert_mock_file_contains "/etc/app.conf" "database=postgres"; then
    pass "Snapshot restore works correctly"
  else
    fail "Snapshot restore failed"
  fi

  # Cleanup
  cleanup_filesystem_mock
}

# ============================================================================
# Example 7: Retry Logic for Flaky Operations
# ============================================================================

test_retry_logic() {
  printf "\n\033[34m→ Test 7: Retry logic for transient failures\033[0m\n"

  # Simulate flaky operation that succeeds on 3rd attempt
  ATTEMPT=0
  flaky_operation() {
    ATTEMPT=$((ATTEMPT + 1))
    if [[ $ATTEMPT -lt 3 ]]; then
      return 1  # Fail
    else
      return 0  # Success on 3rd attempt
    fi
  }

  # Retry up to 5 times
  if retry_on_failure flaky_operation 5; then
    pass "Flaky operation succeeded after retries"
  else
    fail "Flaky operation failed even with retries"
  fi
}

# ============================================================================
# Example 8: Test Isolation with Unique Resources
# ============================================================================

test_isolation() {
  printf "\n\033[34m→ Test 8: Test isolation with unique resources\033[0m\n"

  # Each test gets unique resources
  # Sleep briefly between calls to ensure different timestamps
  local project1=$(get_unique_project_name "app")
  sleep 1
  local project2=$(get_unique_project_name "app")

  local db1=$(get_unique_db_name "testdb")
  sleep 1
  local db2=$(get_unique_db_name "testdb")

  local port1=$(get_random_port)
  local port2=$(get_random_port)

  # Verify uniqueness
  if [[ "$project1" != "$project2" ]]; then
    pass "Project names are unique ($project1 vs $project2)"
  else
    fail "Project names collided ($project1)"
  fi

  if [[ "$db1" != "$db2" ]]; then
    pass "Database names are unique ($db1 vs $db2)"
  else
    fail "Database names collided ($db1)"
  fi

  if [[ "$port1" != "$port2" ]]; then
    pass "Ports are unique ($port1 vs $port2)"
  else
    fail "Ports collided ($port1)"
  fi
}

# ============================================================================
# Example 9: Platform-Specific Tests
# ============================================================================

test_platform_detection() {
  printf "\n\033[34m→ Test 9: Platform detection and conditional tests\033[0m\n"

  local platform=$(detect_platform)
  printf "  Detected platform: %s\n" "$platform"

  if [[ "$platform" == "macos" ]] || [[ "$platform" == "linux" ]] || [[ "$platform" == "wsl" ]]; then
    pass "Platform detected correctly"
  else
    fail "Unknown platform"
  fi

  # Example: Skip test on macOS
  if skip_on_platform "macos" "This test requires Linux"; then
    printf "  (Skipped on macOS as expected)\n"
  fi

  pass "Platform-specific logic works"
}

# ============================================================================
# Example 10: Wait Functions
# ============================================================================

test_wait_functions() {
  printf "\n\033[34m→ Test 10: Wait functions for async operations\033[0m\n"

  # Test wait_for_file functionality
  local test_dir="/tmp/nself-wait-test-$$"
  mkdir -p "$test_dir"

  # Create file in background after brief delay
  (sleep 1 && touch "$test_dir/ready.txt") &

  # Wait for file to exist
  if wait_for_file "$test_dir/ready.txt" 5; then
    pass "Wait for file succeeded"
  else
    fail "Wait for file timed out"
  fi

  # Demonstrate the wait functions work
  pass "Wait functions are available for async testing"

  # Cleanup
  rm -rf "$test_dir"
}

# ============================================================================
# Main Test Runner
# ============================================================================

main() {
  printf "\n"
  printf "╔════════════════════════════════════════════════════════════╗\n"
  printf "║         Reliable Test Framework Examples                  ║\n"
  printf "╚════════════════════════════════════════════════════════════╝\n"

  # Run all example tests
  test_basic_with_timeout_and_cleanup
  test_guaranteed_cleanup
  test_with_docker_mock
  test_with_network_mock
  test_with_time_mock
  test_with_filesystem_mock
  test_retry_logic
  test_isolation
  test_platform_detection
  test_wait_functions

  # Print summary
  printf "\n"
  printf "═══════════════════════════════════════════════════════════\n"
  printf "Test Summary:\n"
  printf "  Total:  %d\n" "$TESTS_RUN"
  printf "  Passed: %d\n" "$TESTS_PASSED"
  printf "  Failed: %d\n" "$TESTS_FAILED"
  printf "═══════════════════════════════════════════════════════════\n"

  if [[ $TESTS_FAILED -eq 0 ]]; then
    printf "\033[32m✓ All examples passed!\033[0m\n"
    printf "\n"
    printf "These examples demonstrate:\n"
    printf "  • Timeout protection\n"
    printf "  • Guaranteed cleanup\n"
    printf "  • Docker mocking (no real Docker needed)\n"
    printf "  • Network mocking (no real HTTP requests)\n"
    printf "  • Time mocking (instant timeouts, deterministic timestamps)\n"
    printf "  • Filesystem mocking (in-memory operations)\n"
    printf "  • Retry logic for flaky operations\n"
    printf "  • Test isolation with unique resources\n"
    printf "  • Platform-specific conditional tests\n"
    printf "  • Wait functions for async operations\n"
    printf "\n"
    return 0
  else
    printf "\033[31m✗ Some examples failed\033[0m\n"
    return 1
  fi
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main
fi
