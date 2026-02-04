#!/usr/bin/env bash
# environment-detection.sh - Auto-detect and adapt to test environments
#
# Provides intelligent environment detection and test adaptation to maximize
# test pass rates across all platforms and CI environments.

set -euo pipefail

# ============================================================================
# Environment Detection
# ============================================================================

# Detect test environment and set global variable
# Sets: TEST_ENVIRONMENT (ci, macos, linux, wsl, unknown)
detect_test_environment() {
  local env="unknown"

  # Detect CI (highest priority)
  if [[ -n "${CI:-}" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]] || [[ -n "${GITLAB_CI:-}" ]] || [[ -n "${CIRCLECI:-}" ]]; then
    env="ci"
  # Detect macOS
  elif [[ "$(uname)" == "Darwin" ]]; then
    env="macos"
  # Detect WSL
  elif [[ -f "/proc/version" ]] && grep -qi microsoft /proc/version 2>/dev/null; then
    env="wsl"
  # Detect Linux
  elif [[ "$(uname)" == "Linux" ]]; then
    env="linux"
  fi

  export TEST_ENVIRONMENT="$env"

  # Set CI-specific defaults
  if [[ "$env" == "ci" ]]; then
    export TEST_CI_MODE="true"
    export TEST_TIMEOUTS_MULTIPLIER="${TEST_TIMEOUTS_MULTIPLIER:-3}"
    export TEST_RETRY_COUNT="${TEST_RETRY_COUNT:-3}"
    export TEST_SKIP_SLOW="${TEST_SKIP_SLOW:-false}"
  else
    export TEST_CI_MODE="false"
    export TEST_TIMEOUTS_MULTIPLIER="${TEST_TIMEOUTS_MULTIPLIER:-1}"
    export TEST_RETRY_COUNT="${TEST_RETRY_COUNT:-1}"
  fi

  printf "%s\n" "$env"
}

# Check if running in CI
is_ci_environment() {
  [[ "${TEST_ENVIRONMENT:-}" == "ci" ]] || [[ -n "${CI:-}" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]]
}

# Check if running on macOS
is_macos_environment() {
  [[ "${TEST_ENVIRONMENT:-}" == "macos" ]] || [[ "$(uname)" == "Darwin" ]]
}

# Check if running on Linux
is_linux_environment() {
  [[ "${TEST_ENVIRONMENT:-}" == "linux" ]] || [[ "$(uname)" == "Linux" ]]
}

# Check if running in WSL
is_wsl_environment() {
  [[ "${TEST_ENVIRONMENT:-}" == "wsl" ]] || ([[ -f "/proc/version" ]] && grep -qi microsoft /proc/version 2>/dev/null)
}

# ============================================================================
# Test Skipping Logic
# ============================================================================

# Skip test if environment doesn't match
# Usage: should_skip_test test_name required_env
# Returns: 0 if should skip, 1 if should run
should_skip_test() {
  local test_name="$1"
  local required_env="$2"

  # Auto-detect if not already detected
  [[ -z "${TEST_ENVIRONMENT:-}" ]] && detect_test_environment >/dev/null

  if [[ "$TEST_ENVIRONMENT" != "$required_env" ]] && [[ "$required_env" != "any" ]]; then
    printf "\033[33mSKIP:\033[0m %s (requires %s, current: %s)\n" \
      "$test_name" "$required_env" "$TEST_ENVIRONMENT" >&2
    return 0  # Should skip
  fi

  return 1  # Should not skip
}

# Skip test on specific platform
# Usage: skip_if_platform platform_name message
skip_if_platform() {
  local platform="$1"
  local message="${2:-Test not supported on $platform}"

  [[ -z "${TEST_ENVIRONMENT:-}" ]] && detect_test_environment >/dev/null

  if [[ "$TEST_ENVIRONMENT" == "$platform" ]]; then
    printf "\033[33mSKIP:\033[0m %s\n" "$message" >&2
    return 0  # Should skip
  fi

  return 1  # Should not skip
}

# Skip test if in CI
# Usage: skip_if_ci message
skip_if_ci() {
  local message="${1:-Test not supported in CI}"

  if is_ci_environment; then
    printf "\033[33mSKIP:\033[0m %s\n" "$message" >&2
    return 0  # Should skip
  fi

  return 1  # Should not skip
}

# ============================================================================
# Feature Detection
# ============================================================================

# Check if feature/command is available
# Usage: has_feature feature_name
# Returns: 0 if available, 1 if not
has_feature() {
  local feature="$1"

  case "$feature" in
    timeout)
      command -v timeout >/dev/null 2>&1 || command -v gtimeout >/dev/null 2>&1
      ;;
    docker)
      command -v docker >/dev/null 2>&1 && docker ps >/dev/null 2>&1
      ;;
    docker-compose)
      command -v docker-compose >/dev/null 2>&1 || (command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1)
      ;;
    network)
      ping -c 1 -W 2 8.8.8.8 >/dev/null 2>&1
      ;;
    nc|netcat)
      command -v nc >/dev/null 2>&1
      ;;
    python)
      command -v python3 >/dev/null 2>&1 || command -v python >/dev/null 2>&1
      ;;
    jq)
      command -v jq >/dev/null 2>&1
      ;;
    curl)
      command -v curl >/dev/null 2>&1
      ;;
    wget)
      command -v wget >/dev/null 2>&1
      ;;
    git)
      command -v git >/dev/null 2>&1
      ;;
    *)
      command -v "$feature" >/dev/null 2>&1
      ;;
  esac
}

# Run test only if feature available
# Usage: test_if_available feature test_function
test_if_available() {
  local feature="$1"
  local test_func="$2"

  if has_feature "$feature"; then
    $test_func
    return $?
  else
    printf "\033[33mSKIP:\033[0m Feature '%s' not available\n" "$feature" >&2
    return 0  # Pass (don't fail)
  fi
}

# Require feature or skip test
# Usage: require_feature feature_name message
require_feature() {
  local feature="$1"
  local message="${2:-Feature $feature is required}"

  if ! has_feature "$feature"; then
    printf "\033[33mSKIP:\033[0m %s\n" "$message" >&2
    return 1  # Feature not available
  fi

  return 0  # Feature available
}

# ============================================================================
# Resource Detection
# ============================================================================

# Get available memory in MB
get_available_memory_mb() {
  local mem_mb=0

  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS
    mem_mb=$(sysctl -n hw.memsize 2>/dev/null | awk '{print int($1/1024/1024)}')
  elif [[ -f "/proc/meminfo" ]]; then
    # Linux
    mem_mb=$(grep MemAvailable /proc/meminfo 2>/dev/null | awk '{print int($2/1024)}')
    # Fallback to MemFree if MemAvailable not present
    if [[ -z "$mem_mb" ]] || [[ "$mem_mb" -eq 0 ]]; then
      mem_mb=$(grep MemFree /proc/meminfo 2>/dev/null | awk '{print int($2/1024)}')
    fi
  fi

  # Default to 1GB if can't detect
  [[ -z "$mem_mb" ]] || [[ "$mem_mb" -eq 0 ]] && mem_mb=1024

  printf "%d\n" "$mem_mb"
}

# Get available disk space in MB
get_available_disk_mb() {
  local disk_mb=0
  local test_dir="${1:-/tmp}"

  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS
    disk_mb=$(df -m "$test_dir" 2>/dev/null | tail -1 | awk '{print int($4)}')
  else
    # Linux
    disk_mb=$(df -m "$test_dir" 2>/dev/null | tail -1 | awk '{print int($4)}')
  fi

  # Default to 10GB if can't detect
  [[ -z "$disk_mb" ]] || [[ "$disk_mb" -eq 0 ]] && disk_mb=10240

  printf "%d\n" "$disk_mb"
}

# Check if sufficient resources available
# Usage: has_sufficient_resources min_memory_mb min_disk_mb
has_sufficient_resources() {
  local min_memory="${1:-512}"
  local min_disk="${2:-1024}"

  local available_memory
  local available_disk

  available_memory=$(get_available_memory_mb)
  available_disk=$(get_available_disk_mb)

  if [[ $available_memory -lt $min_memory ]]; then
    printf "\033[33mWARN:\033[0m Low memory: %d MB available (need %d MB)\n" \
      "$available_memory" "$min_memory" >&2
    return 1
  fi

  if [[ $available_disk -lt $min_disk ]]; then
    printf "\033[33mWARN:\033[0m Low disk: %d MB available (need %d MB)\n" \
      "$available_disk" "$min_disk" >&2
    return 1
  fi

  return 0
}

# ============================================================================
# Platform-Specific Behavior
# ============================================================================

# Get timeout command for platform
get_timeout_command() {
  if command -v timeout >/dev/null 2>&1; then
    printf "timeout\n"
  elif command -v gtimeout >/dev/null 2>&1; then
    printf "gtimeout\n"
  else
    printf "none\n"
  fi
}

# Get platform-specific temp directory
get_platform_temp_dir() {
  local prefix="${1:-nself-test}"

  # Try TMPDIR first
  if [[ -n "${TMPDIR:-}" ]] && [[ -d "$TMPDIR" ]] && [[ -w "$TMPDIR" ]]; then
    mktemp -d "$TMPDIR/${prefix}.XXXXXX" 2>/dev/null && return 0
  fi

  # Try /tmp
  if [[ -d "/tmp" ]] && [[ -w "/tmp" ]]; then
    mktemp -d "/tmp/${prefix}.XXXXXX" 2>/dev/null && return 0
  fi

  # Try /var/tmp
  if [[ -d "/var/tmp" ]] && [[ -w "/var/tmp" ]]; then
    mktemp -d "/var/tmp/${prefix}.XXXXXX" 2>/dev/null && return 0
  fi

  # Fallback - create in current directory
  local temp_dir="./${prefix}.$$"
  mkdir -p "$temp_dir" && printf "%s\n" "$temp_dir"
}

# ============================================================================
# Test Environment Configuration
# ============================================================================

# Get timeout multiplier for current environment
get_timeout_multiplier() {
  if is_ci_environment; then
    printf "%d\n" "${TEST_TIMEOUTS_MULTIPLIER:-3}"
  else
    printf "1\n"
  fi
}

# Get adjusted timeout for environment
# Usage: get_adjusted_timeout base_timeout
get_adjusted_timeout() {
  local base_timeout="$1"
  local multiplier

  multiplier=$(get_timeout_multiplier)
  printf "%d\n" $((base_timeout * multiplier))
}

# Get retry count for environment
get_retry_count() {
  printf "%d\n" "${TEST_RETRY_COUNT:-1}"
}

# ============================================================================
# Export Functions
# ============================================================================

export -f detect_test_environment
export -f is_ci_environment
export -f is_macos_environment
export -f is_linux_environment
export -f is_wsl_environment
export -f should_skip_test
export -f skip_if_platform
export -f skip_if_ci
export -f has_feature
export -f test_if_available
export -f require_feature
export -f get_available_memory_mb
export -f get_available_disk_mb
export -f has_sufficient_resources
export -f get_timeout_command
export -f get_platform_temp_dir
export -f get_timeout_multiplier
export -f get_adjusted_timeout
export -f get_retry_count

# Auto-detect environment on source
detect_test_environment >/dev/null
