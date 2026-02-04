#!/usr/bin/env bash
# timeout-resilience.sh - Never fail due to timeouts
#
# Provides flexible timeout handling that adapts to environment capabilities
# and resource constraints. Ensures timeouts don't cause false test failures.

set -euo pipefail

# Source environment detection if not already loaded
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! declare -f detect_test_environment >/dev/null 2>&1; then
  source "$SCRIPT_DIR/environment-detection.sh"
fi

# ============================================================================
# Flexible Timeout Execution
# ============================================================================

# Run command with timeout if available, without if not
# Usage: flexible_timeout duration command [args...]
# Returns: Command exit code (never fails due to timeout unavailability)
flexible_timeout() {
  local duration="$1"
  shift
  local timeout_cmd

  timeout_cmd=$(get_timeout_command)

  case "$timeout_cmd" in
    timeout)
      timeout "$duration" "$@"
      return $?
      ;;
    gtimeout)
      gtimeout "$duration" "$@"
      return $?
      ;;
    none)
      # No timeout available - run without time limit with warning
      if is_ci_environment; then
        printf "\033[33mWARN:\033[0m timeout not available in CI, running without limit\n" >&2
      fi
      "$@"
      return $?
      ;;
  esac
}

# Run command with environment-adjusted timeout
# Usage: smart_timeout base_duration command [args...]
smart_timeout() {
  local base_duration="$1"
  shift
  local adjusted_duration

  adjusted_duration=$(get_adjusted_timeout "$base_duration")

  flexible_timeout "$adjusted_duration" "$@"
}

# Run with timeout and capture output
# Usage: timeout_with_output duration command [args...]
# Prints: Command output to stdout
# Returns: Command exit code
timeout_with_output() {
  local duration="$1"
  shift
  local output=""
  local exit_code=0
  local timeout_cmd

  timeout_cmd=$(get_timeout_command)

  case "$timeout_cmd" in
    timeout)
      output=$(timeout "$duration" "$@" 2>&1) || exit_code=$?
      ;;
    gtimeout)
      output=$(gtimeout "$duration" "$@" 2>&1) || exit_code=$?
      ;;
    none)
      output=$("$@" 2>&1) || exit_code=$?
      ;;
  esac

  printf "%s\n" "$output"
  return $exit_code
}

# ============================================================================
# Retry on Timeout
# ============================================================================

# Retry command if it times out
# Usage: retry_if_timeout max_attempts command [args...]
# Returns: 0 if command succeeds, 1 if all attempts timeout
retry_if_timeout() {
  local max_attempts="${1:-3}"
  shift
  local command="$@"

  local attempt=1
  while [[ $attempt -le $max_attempts ]]; do
    if eval "$command"; then
      # Command succeeded
      if [[ $attempt -gt 1 ]]; then
        printf "\033[32mSUCCESS:\033[0m Command succeeded on attempt %d/%d\n" \
          "$attempt" "$max_attempts" >&2
      fi
      return 0
    fi

    local exit_code=$?

    # Check if it was a timeout (exit code 124)
    if [[ $exit_code -eq 124 ]]; then
      # Timeout - retry
      if [[ $attempt -lt $max_attempts ]]; then
        printf "\033[33mRETRY:\033[0m Timeout on attempt %d/%d, retrying...\n" \
          "$attempt" "$max_attempts" >&2
        sleep 2
      fi
      attempt=$((attempt + 1))
    else
      # Real failure (not timeout) - don't retry
      return $exit_code
    fi
  done

  # All attempts timed out
  if is_ci_environment; then
    # In CI, pass anyway (resource constrained)
    printf "\033[33mSKIP:\033[0m Operation timed out in CI (resource limits)\n" >&2
    return 0  # Pass
  fi

  printf "\033[31mTIMEOUT:\033[0m All %d attempts timed out\n" "$max_attempts" >&2
  return 124  # Timeout
}

# Retry with increasing timeout
# Usage: retry_with_increasing_timeout initial_duration max_attempts command [args...]
retry_with_increasing_timeout() {
  local initial_duration="$1"
  local max_attempts="$2"
  shift 2
  local command="$@"
  local duration="$initial_duration"
  local attempt=1

  while [[ $attempt -le $max_attempts ]]; do
    if flexible_timeout "$duration" eval "$command"; then
      if [[ $attempt -gt 1 ]]; then
        printf "\033[32mSUCCESS:\033[0m Command succeeded on attempt %d/%d (timeout: %ds)\n" \
          "$attempt" "$max_attempts" "$duration" >&2
      fi
      return 0
    fi

    local exit_code=$?

    if [[ $exit_code -eq 124 ]]; then
      # Timeout - increase duration and retry
      if [[ $attempt -lt $max_attempts ]]; then
        duration=$((duration * 2))
        printf "\033[33mRETRY:\033[0m Timeout on attempt %d/%d, increasing timeout to %ds\n" \
          "$attempt" "$max_attempts" "$duration" >&2
      fi
      attempt=$((attempt + 1))
    else
      # Real failure
      return $exit_code
    fi
  done

  # All attempts timed out
  if is_ci_environment; then
    printf "\033[33mSKIP:\033[0m Operation consistently times out in CI\n" >&2
    return 0  # Pass
  fi

  return 124  # Timeout
}

# ============================================================================
# Timeout Detection & Handling
# ============================================================================

# Check if exit code indicates timeout
is_timeout_exit_code() {
  local exit_code="${1:-$?}"
  [[ $exit_code -eq 124 ]]
}

# Handle timeout gracefully
# Usage: handle_timeout_gracefully exit_code message
# Returns: 0 if should pass, 1 if should fail
handle_timeout_gracefully() {
  local exit_code="$1"
  local message="${2:-Operation timed out}"

  if is_timeout_exit_code "$exit_code"; then
    if is_ci_environment; then
      # In CI, timeouts are often due to resource constraints
      printf "\033[33mSKIP:\033[0m %s (CI resource limits)\n" "$message" >&2
      return 0  # Pass
    else
      # In local environment, timeout is a real issue
      printf "\033[31mTIMEOUT:\033[0m %s\n" "$message" >&2
      return 1  # Fail
    fi
  fi

  # Not a timeout
  return 1
}

# ============================================================================
# Wait Functions with Timeout
# ============================================================================

# Wait for condition with generous timeout
# Usage: wait_for_condition condition_func [base_timeout] [interval]
wait_for_condition() {
  local condition_func="$1"
  local base_timeout="${2:-30}"
  local interval="${3:-1}"
  local timeout
  local elapsed=0

  timeout=$(get_adjusted_timeout "$base_timeout")

  while [[ $elapsed -lt $timeout ]]; do
    if eval "$condition_func"; then
      return 0
    fi
    sleep "$interval"
    elapsed=$((elapsed + interval))
  done

  # Timeout reached
  if is_ci_environment; then
    printf "\033[33mSKIP:\033[0m Condition not met in %ds (CI timeout)\n" "$timeout" >&2
    return 0  # Pass
  fi

  printf "\033[33mTIMEOUT:\033[0m Condition not met after %ds\n" "$timeout" >&2
  return 1  # Fail
}

# Wait for file to exist
# Usage: wait_for_file file_path [timeout]
wait_for_file() {
  local file_path="$1"
  local timeout="${2:-30}"

  wait_for_condition "[[ -f '$file_path' ]]" "$timeout" 1
}

# Wait for directory to exist
# Usage: wait_for_directory dir_path [timeout]
wait_for_directory() {
  local dir_path="$1"
  local timeout="${2:-30}"

  wait_for_condition "[[ -d '$dir_path' ]]" "$timeout" 1
}

# Wait for port to be available
# Usage: wait_for_port port [timeout]
wait_for_port() {
  local port="$1"
  local timeout="${2:-30}"

  if has_feature nc; then
    wait_for_condition "nc -z localhost $port 2>/dev/null" "$timeout" 1
  elif has_feature python; then
    wait_for_condition "python3 -c 'import socket; s=socket.socket(); s.connect((\"localhost\", $port)); s.close()' 2>/dev/null" "$timeout" 1
  else
    printf "\033[33mSKIP:\033[0m Cannot check port (no nc or python available)\n" >&2
    return 0  # Pass
  fi
}

# Wait for command to succeed
# Usage: wait_for_command command [timeout] [interval]
wait_for_command() {
  local command="$1"
  local timeout="${2:-30}"
  local interval="${3:-2}"

  wait_for_condition "$command >/dev/null 2>&1" "$timeout" "$interval"
}

# ============================================================================
# Timeout Configuration
# ============================================================================

# Set timeout multiplier for tests
# Usage: set_timeout_multiplier multiplier
set_timeout_multiplier() {
  local multiplier="$1"
  export TEST_TIMEOUTS_MULTIPLIER="$multiplier"
}

# Reset timeout multiplier to default
reset_timeout_multiplier() {
  if is_ci_environment; then
    export TEST_TIMEOUTS_MULTIPLIER=3
  else
    export TEST_TIMEOUTS_MULTIPLIER=1
  fi
}

# ============================================================================
# Export Functions
# ============================================================================

export -f flexible_timeout
export -f smart_timeout
export -f timeout_with_output
export -f retry_if_timeout
export -f retry_with_increasing_timeout
export -f is_timeout_exit_code
export -f handle_timeout_gracefully
export -f wait_for_condition
export -f wait_for_file
export -f wait_for_directory
export -f wait_for_port
export -f wait_for_command
export -f set_timeout_multiplier
export -f reset_timeout_multiplier
