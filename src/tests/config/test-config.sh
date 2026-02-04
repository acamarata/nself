#!/usr/bin/env bash
# test-config.sh - Global test configuration for maximum resilience
#
# Centralized configuration that adapts to environment to maximize test
# pass rates while maintaining test integrity.

set -euo pipefail

# ============================================================================
# Environment Detection
# ============================================================================

# Auto-detect environment if not already set
if [[ -z "${TEST_ENVIRONMENT:-}" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  if [[ -f "$SCRIPT_DIR/../lib/environment-detection.sh" ]]; then
    source "$SCRIPT_DIR/../lib/environment-detection.sh"
    detect_test_environment >/dev/null
  else
    # Fallback detection
    if [[ -n "${CI:-}" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]]; then
      export TEST_ENVIRONMENT="ci"
    elif [[ "$(uname)" == "Darwin" ]]; then
      export TEST_ENVIRONMENT="macos"
    elif [[ "$(uname)" == "Linux" ]]; then
      export TEST_ENVIRONMENT="linux"
    else
      export TEST_ENVIRONMENT="unknown"
    fi
  fi
fi

# ============================================================================
# Timeout Configuration
# ============================================================================

# Base timeouts (seconds) - adjusted based on environment
if [[ "${TEST_ENVIRONMENT}" == "ci" ]]; then
  # CI environments need longer timeouts due to resource sharing
  export TEST_TIMEOUT_SHORT="${TEST_TIMEOUT_SHORT:-60}"
  export TEST_TIMEOUT_MEDIUM="${TEST_TIMEOUT_MEDIUM:-180}"
  export TEST_TIMEOUT_LONG="${TEST_TIMEOUT_LONG:-300}"
  export TEST_TIMEOUT_VERY_LONG="${TEST_TIMEOUT_VERY_LONG:-600}"
else
  # Local development - shorter timeouts
  export TEST_TIMEOUT_SHORT="${TEST_TIMEOUT_SHORT:-10}"
  export TEST_TIMEOUT_MEDIUM="${TEST_TIMEOUT_MEDIUM:-30}"
  export TEST_TIMEOUT_LONG="${TEST_TIMEOUT_LONG:-60}"
  export TEST_TIMEOUT_VERY_LONG="${TEST_TIMEOUT_VERY_LONG:-120}"
fi

# Timeout multiplier (can be overridden)
export TEST_TIMEOUTS_MULTIPLIER="${TEST_TIMEOUTS_MULTIPLIER:-1}"

# ============================================================================
# Retry Configuration
# ============================================================================

# Retry counts - higher in CI
if [[ "${TEST_ENVIRONMENT}" == "ci" ]]; then
  export TEST_MAX_RETRIES="${TEST_MAX_RETRIES:-3}"
  export TEST_RETRY_DELAY="${TEST_RETRY_DELAY:-2}"
  export TEST_RETRY_BACKOFF="${TEST_RETRY_BACKOFF:-exponential}"
else
  export TEST_MAX_RETRIES="${TEST_MAX_RETRIES:-1}"
  export TEST_RETRY_DELAY="${TEST_RETRY_DELAY:-1}"
  export TEST_RETRY_BACKOFF="${TEST_RETRY_BACKOFF:-none}"
fi

# Network-specific retries
export TEST_NETWORK_MAX_RETRIES="${TEST_NETWORK_MAX_RETRIES:-3}"
export TEST_NETWORK_RETRY_DELAY="${TEST_NETWORK_RETRY_DELAY:-2}"

# ============================================================================
# Resource Thresholds
# ============================================================================

# Minimum resource requirements (MB)
export TEST_MIN_MEMORY_MB="${TEST_MIN_MEMORY_MB:-256}"
export TEST_MIN_DISK_MB="${TEST_MIN_DISK_MB:-512}"

# Recommended resources (MB)
export TEST_RECOMMENDED_MEMORY_MB="${TEST_RECOMMENDED_MEMORY_MB:-1024}"
export TEST_RECOMMENDED_DISK_MB="${TEST_RECOMMENDED_DISK_MB:-2048}"

# Docker resource requirements (MB)
export TEST_DOCKER_MIN_MEMORY_MB="${TEST_DOCKER_MIN_MEMORY_MB:-512}"
export TEST_DOCKER_RECOMMENDED_MEMORY_MB="${TEST_DOCKER_RECOMMENDED_MEMORY_MB:-2048}"

# ============================================================================
# Tolerance Configuration
# ============================================================================

# Numeric tolerance (percentage)
export TEST_NUMERIC_TOLERANCE_PERCENT="${TEST_NUMERIC_TOLERANCE_PERCENT:-10}"

# Timing tolerance (percentage) - very lenient for timing
if [[ "${TEST_ENVIRONMENT}" == "ci" ]]; then
  export TEST_TIMING_TOLERANCE_PERCENT="${TEST_TIMING_TOLERANCE_PERCENT:-100}"  # 100% tolerance in CI
else
  export TEST_TIMING_TOLERANCE_PERCENT="${TEST_TIMING_TOLERANCE_PERCENT:-50}"
fi

# File size tolerance (percentage)
export TEST_FILESIZE_TOLERANCE_PERCENT="${TEST_FILESIZE_TOLERANCE_PERCENT:-20}"

# ============================================================================
# Skip Configuration
# ============================================================================

# Skip behavior (auto, always, never)
export TEST_SKIP_NETWORK_TESTS="${TEST_SKIP_NETWORK_TESTS:-auto}"
export TEST_SKIP_DOCKER_TESTS="${TEST_SKIP_DOCKER_TESTS:-auto}"
export TEST_SKIP_SLOW_TESTS="${TEST_SKIP_SLOW_TESTS:-auto}"
export TEST_SKIP_INTEGRATION_TESTS="${TEST_SKIP_INTEGRATION_TESTS:-auto}"

# In CI, auto-skip based on availability
if [[ "${TEST_ENVIRONMENT}" == "ci" ]]; then
  # CI usually has Docker but may have flaky network
  export TEST_SKIP_NETWORK_TESTS="${TEST_SKIP_NETWORK_TESTS:-auto}"
  export TEST_SKIP_DOCKER_TESTS="${TEST_SKIP_DOCKER_TESTS:-never}"
fi

# ============================================================================
# Verbosity & Output
# ============================================================================

# Log levels (debug, info, warn, error)
export TEST_LOG_LEVEL="${TEST_LOG_LEVEL:-info}"

# Show progress indicators
export TEST_SHOW_PROGRESS="${TEST_SHOW_PROGRESS:-true}"

# Show timing information
export TEST_SHOW_TIMING="${TEST_SHOW_TIMING:-true}"

# Color output
export TEST_COLOR_OUTPUT="${TEST_COLOR_OUTPUT:-auto}"

# ============================================================================
# Isolation Configuration
# ============================================================================

# Test isolation mode (full, partial, none)
export TEST_ISOLATION_MODE="${TEST_ISOLATION_MODE:-full}"

# Cleanup behavior (always, on-success, on-failure, never)
export TEST_CLEANUP_MODE="${TEST_CLEANUP_MODE:-always}"

# Temp directory prefix
export TEST_TEMP_PREFIX="${TEST_TEMP_PREFIX:-nself-test}"

# ============================================================================
# Parallel Execution
# ============================================================================

# Maximum parallel tests (0 = auto-detect)
if [[ "${TEST_ENVIRONMENT}" == "ci" ]]; then
  export TEST_MAX_PARALLEL="${TEST_MAX_PARALLEL:-2}"  # Conservative in CI
else
  export TEST_MAX_PARALLEL="${TEST_MAX_PARALLEL:-4}"
fi

# Parallel test timeout multiplier
export TEST_PARALLEL_TIMEOUT_MULTIPLIER="${TEST_PARALLEL_TIMEOUT_MULTIPLIER:-2}"

# ============================================================================
# Failure Handling
# ============================================================================

# Fail fast (stop on first failure)
export TEST_FAIL_FAST="${TEST_FAIL_FAST:-false}"

# Continue on expected failures
export TEST_CONTINUE_ON_EXPECTED_FAILURE="${TEST_CONTINUE_ON_EXPECTED_FAILURE:-true}"

# Retry failed tests
export TEST_RETRY_FAILED="${TEST_RETRY_FAILED:-true}"

# ============================================================================
# Mock Configuration
# ============================================================================

# Use mocks when services unavailable
export TEST_USE_MOCKS="${TEST_USE_MOCKS:-auto}"

# Mock external services
export TEST_MOCK_EXTERNAL_APIS="${TEST_MOCK_EXTERNAL_APIS:-auto}"

# Mock Docker when unavailable
export TEST_MOCK_DOCKER="${TEST_MOCK_DOCKER:-auto}"

# ============================================================================
# Coverage Configuration
# ============================================================================

# Track coverage
export TEST_TRACK_COVERAGE="${TEST_TRACK_COVERAGE:-false}"

# Minimum coverage threshold (percentage)
export TEST_MIN_COVERAGE_PERCENT="${TEST_MIN_COVERAGE_PERCENT:-70}"

# ============================================================================
# Performance Configuration
# ============================================================================

# Enable performance tracking
export TEST_TRACK_PERFORMANCE="${TEST_TRACK_PERFORMANCE:-false}"

# Performance baseline directory
export TEST_PERFORMANCE_BASELINE="${TEST_PERFORMANCE_BASELINE:-./test-baselines}"

# Performance variance tolerance (percentage)
export TEST_PERFORMANCE_TOLERANCE="${TEST_PERFORMANCE_TOLERANCE:-20}"

# ============================================================================
# Helper Functions
# ============================================================================

# Get timeout for category
get_timeout_for_category() {
  local category="$1"

  case "$category" in
    short)
      printf "%d\n" "$TEST_TIMEOUT_SHORT"
      ;;
    medium)
      printf "%d\n" "$TEST_TIMEOUT_MEDIUM"
      ;;
    long)
      printf "%d\n" "$TEST_TIMEOUT_LONG"
      ;;
    very-long)
      printf "%d\n" "$TEST_TIMEOUT_VERY_LONG"
      ;;
    *)
      printf "%d\n" "$TEST_TIMEOUT_MEDIUM"
      ;;
  esac
}

# Should skip test based on configuration
should_skip_test_type() {
  local test_type="$1"
  local skip_config

  case "$test_type" in
    network)
      skip_config="$TEST_SKIP_NETWORK_TESTS"
      ;;
    docker)
      skip_config="$TEST_SKIP_DOCKER_TESTS"
      ;;
    slow)
      skip_config="$TEST_SKIP_SLOW_TESTS"
      ;;
    integration)
      skip_config="$TEST_SKIP_INTEGRATION_TESTS"
      ;;
    *)
      return 1  # Don't skip
      ;;
  esac

  case "$skip_config" in
    always)
      return 0  # Skip
      ;;
    never)
      return 1  # Don't skip
      ;;
    auto)
      # Auto-detect based on availability
      # This will be handled by the specific test frameworks
      return 1  # Don't skip (let framework decide)
      ;;
    *)
      return 1  # Don't skip by default
      ;;
  esac
}

# Print test configuration summary
print_test_config() {
  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n"
  printf "\033[36mTest Configuration\033[0m\n"
  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n"
  printf "Environment:     %s\n" "$TEST_ENVIRONMENT"
  printf "Timeout Short:   %ds\n" "$TEST_TIMEOUT_SHORT"
  printf "Timeout Medium:  %ds\n" "$TEST_TIMEOUT_MEDIUM"
  printf "Timeout Long:    %ds\n" "$TEST_TIMEOUT_LONG"
  printf "Max Retries:     %d\n" "$TEST_MAX_RETRIES"
  printf "Tolerance:       %d%%\n" "$TEST_NUMERIC_TOLERANCE_PERCENT"
  printf "Parallel Tests:  %d\n" "$TEST_MAX_PARALLEL"
  printf "Isolation Mode:  %s\n" "$TEST_ISOLATION_MODE"
  printf "Cleanup Mode:    %s\n" "$TEST_CLEANUP_MODE"
  printf "\033[36m=%.0s\033[0m" {1..80}
  printf "\n"
}

# ============================================================================
# Export Functions
# ============================================================================

export -f get_timeout_for_category
export -f should_skip_test_type
export -f print_test_config

# ============================================================================
# Auto-Configuration Messages
# ============================================================================

if [[ "${TEST_SHOW_CONFIG:-false}" == "true" ]]; then
  print_test_config
fi
