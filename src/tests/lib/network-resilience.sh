#!/usr/bin/env bash
# network-resilience.sh - Handle network unavailability gracefully
#
# Provides network-aware testing that gracefully handles offline environments,
# firewalls, and intermittent connectivity issues.

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! declare -f detect_test_environment >/dev/null 2>&1; then
  source "$SCRIPT_DIR/environment-detection.sh"
fi

# ============================================================================
# Network Detection
# ============================================================================

# Check if network is available with retries
# Returns: 0 if available, 1 if not
check_network_available() {
  local max_attempts="${1:-3}"
  local timeout="${2:-2}"
  local attempt=1

  while [[ $attempt -le $max_attempts ]]; do
    # Try multiple reliable hosts
    if ping -c 1 -W "$timeout" 8.8.8.8 >/dev/null 2>&1; then
      return 0  # Network available
    fi

    if ping -c 1 -W "$timeout" 1.1.1.1 >/dev/null 2>&1; then
      return 0  # Network available
    fi

    # Try DNS resolution as fallback
    if command -v host >/dev/null 2>&1; then
      if host google.com >/dev/null 2>&1; then
        return 0  # Network available
      fi
    fi

    attempt=$((attempt + 1))
    [[ $attempt -le $max_attempts ]] && sleep 1
  done

  return 1  # Network not available
}

# Check if specific host is reachable
# Usage: is_host_reachable hostname [timeout]
is_host_reachable() {
  local hostname="$1"
  local timeout="${2:-2}"

  # Try ping first
  if ping -c 1 -W "$timeout" "$hostname" >/dev/null 2>&1; then
    return 0
  fi

  # Try DNS resolution
  if command -v host >/dev/null 2>&1; then
    if host "$hostname" >/dev/null 2>&1; then
      return 0
    fi
  fi

  # Try curl if available
  if command -v curl >/dev/null 2>&1; then
    if curl -s -m "$timeout" -o /dev/null "http://$hostname" 2>/dev/null; then
      return 0
    fi
  fi

  return 1
}

# Check if URL is accessible
# Usage: is_url_accessible url [timeout]
is_url_accessible() {
  local url="$1"
  local timeout="${2:-5}"

  if command -v curl >/dev/null 2>&1; then
    if curl -s -m "$timeout" -o /dev/null -f "$url" 2>/dev/null; then
      return 0
    fi
  elif command -v wget >/dev/null 2>&1; then
    if wget -q -T "$timeout" -O /dev/null "$url" 2>/dev/null; then
      return 0
    fi
  fi

  return 1
}

# ============================================================================
# Network Test Execution
# ============================================================================

# Run test with network or skip gracefully
# Usage: test_with_network test_function
test_with_network() {
  local test_func="$1"

  if check_network_available; then
    $test_func
    return $?
  else
    printf "\033[33mSKIP:\033[0m %s (network not available)\n" "$test_func" >&2
    return 0  # Pass
  fi
}

# Require network or skip test
# Usage: require_network [message]
require_network() {
  local message="${1:-Network is required}"

  if ! check_network_available; then
    printf "\033[33mSKIP:\033[0m %s\n" "$message" >&2
    return 1  # Not available
  fi

  return 0  # Available
}

# Run test with specific host or skip
# Usage: test_with_host hostname test_function
test_with_host() {
  local hostname="$1"
  local test_func="$2"

  if is_host_reachable "$hostname"; then
    $test_func
    return $?
  else
    printf "\033[33mSKIP:\033[0m %s (host %s not reachable)\n" "$test_func" "$hostname" >&2
    return 0  # Pass
  fi
}

# ============================================================================
# External Service Mocking
# ============================================================================

# Mock external API service
# Usage: mock_external_api service_name
mock_external_api() {
  local service="$1"

  case "$service" in
    stripe)
      mock_stripe_api
      ;;
    github)
      mock_github_api
      ;;
    sendgrid)
      mock_sendgrid_api
      ;;
    twilio)
      mock_twilio_api
      ;;
    *)
      mock_generic_api "$service"
      ;;
  esac
}

# Mock Stripe API
mock_stripe_api() {
  export STRIPE_API_URL="http://localhost:12111/mock/stripe"
  export STRIPE_API_MOCK=true
  printf "\033[33mINFO:\033[0m Using Stripe API mock\n" >&2
}

# Mock GitHub API
mock_github_api() {
  export GITHUB_API_URL="http://localhost:12112/mock/github"
  export GITHUB_API_MOCK=true
  printf "\033[33mINFO:\033[0m Using GitHub API mock\n" >&2
}

# Mock SendGrid API
mock_sendgrid_api() {
  export SENDGRID_API_URL="http://localhost:12113/mock/sendgrid"
  export SENDGRID_API_MOCK=true
  printf "\033[33mINFO:\033[0m Using SendGrid API mock\n" >&2
}

# Mock Twilio API
mock_twilio_api() {
  export TWILIO_API_URL="http://localhost:12114/mock/twilio"
  export TWILIO_API_MOCK=true
  printf "\033[33mINFO:\033[0m Using Twilio API mock\n" >&2
}

# Mock generic API
mock_generic_api() {
  local service="$1"
  local port=$((12100 + RANDOM % 1000))

  export "${service}_API_URL=http://localhost:${port}/mock/${service}"
  export "${service}_API_MOCK=true"
  printf "\033[33mINFO:\033[0m Using %s API mock on port %d\n" "$service" "$port" >&2
}

# ============================================================================
# Network Retry Logic
# ============================================================================

# Retry network operation with exponential backoff
# Usage: retry_network_operation command [max_attempts]
retry_network_operation() {
  local command="$1"
  local max_attempts="${2:-3}"
  local initial_delay=1
  local attempt=1
  local delay=$initial_delay

  while [[ $attempt -le $max_attempts ]]; do
    if eval "$command"; then
      return 0  # Success
    fi

    local exit_code=$?

    # Check if it's a network error
    if is_network_error "$exit_code"; then
      if [[ $attempt -lt $max_attempts ]]; then
        printf "\033[33mRETRY:\033[0m Network error on attempt %d/%d, retrying in %ds...\n" \
          "$attempt" "$max_attempts" "$delay" >&2
        sleep "$delay"
        delay=$((delay * 2))  # Exponential backoff
      fi
      attempt=$((attempt + 1))
    else
      # Not a network error - don't retry
      return $exit_code
    fi
  done

  # All attempts failed
  printf "\033[33mSKIP:\033[0m Network operation failed after %d attempts\n" "$max_attempts" >&2
  return 0  # Pass (don't fail test due to network issues)
}

# Check if exit code indicates network error
is_network_error() {
  local exit_code="$1"

  # Common network error exit codes
  case "$exit_code" in
    6|7|28|35|51|52|56|60)  # curl error codes
      return 0  # Is network error
      ;;
    *)
      return 1  # Not network error
      ;;
  esac
}

# ============================================================================
# HTTP Request Helpers
# ============================================================================

# Make HTTP request with fallback and retry
# Usage: safe_http_request url [method] [data]
safe_http_request() {
  local url="$1"
  local method="${2:-GET}"
  local data="${3:-}"

  # Check network first
  if ! check_network_available 1 1; then
    printf "\033[33mSKIP:\033[0m HTTP request skipped (no network)\n" >&2
    return 1
  fi

  # Try curl
  if command -v curl >/dev/null 2>&1; then
    if [[ -n "$data" ]]; then
      curl -s -X "$method" -d "$data" "$url" 2>/dev/null
    else
      curl -s -X "$method" "$url" 2>/dev/null
    fi
    return $?
  fi

  # Try wget
  if command -v wget >/dev/null 2>&1; then
    wget -q -O - "$url" 2>/dev/null
    return $?
  fi

  # No HTTP client available
  printf "\033[33mSKIP:\033[0m No HTTP client available (curl/wget)\n" >&2
  return 1
}

# Download file with retry
# Usage: safe_download url output_file [max_attempts]
safe_download() {
  local url="$1"
  local output="$2"
  local max_attempts="${3:-3}"
  local attempt=1

  while [[ $attempt -le $max_attempts ]]; do
    # Try curl
    if command -v curl >/dev/null 2>&1; then
      if curl -s -L -o "$output" "$url" 2>/dev/null; then
        return 0  # Success
      fi
    # Try wget
    elif command -v wget >/dev/null 2>&1; then
      if wget -q -O "$output" "$url" 2>/dev/null; then
        return 0  # Success
      fi
    else
      printf "\033[33mSKIP:\033[0m No download tool available\n" >&2
      return 1
    fi

    # Failed - retry
    if [[ $attempt -lt $max_attempts ]]; then
      printf "\033[33mRETRY:\033[0m Download failed on attempt %d/%d, retrying...\n" \
        "$attempt" "$max_attempts" >&2
      sleep 2
    fi

    attempt=$((attempt + 1))
  done

  # All attempts failed
  printf "\033[33mSKIP:\033[0m Download failed after %d attempts\n" "$max_attempts" >&2
  return 1
}

# ============================================================================
# Offline Mode Support
# ============================================================================

# Enable offline test mode
enable_offline_mode() {
  export TEST_OFFLINE_MODE=true
  printf "\033[33mINFO:\033[0m Offline mode enabled - network tests will be skipped\n" >&2
}

# Disable offline test mode
disable_offline_mode() {
  export TEST_OFFLINE_MODE=false
}

# Check if offline mode is enabled
is_offline_mode() {
  [[ "${TEST_OFFLINE_MODE:-false}" == "true" ]]
}

# Skip test if offline mode is enabled
skip_if_offline() {
  local message="${1:-Test requires network}"

  if is_offline_mode; then
    printf "\033[33mSKIP:\033[0m %s (offline mode)\n" "$message" >&2
    return 0  # Should skip
  fi

  return 1  # Should not skip
}

# ============================================================================
# Export Functions
# ============================================================================

export -f check_network_available
export -f is_host_reachable
export -f is_url_accessible
export -f test_with_network
export -f require_network
export -f test_with_host
export -f mock_external_api
export -f mock_stripe_api
export -f mock_github_api
export -f mock_sendgrid_api
export -f mock_twilio_api
export -f mock_generic_api
export -f retry_network_operation
export -f is_network_error
export -f safe_http_request
export -f safe_download
export -f enable_offline_mode
export -f disable_offline_mode
export -f is_offline_mode
export -f skip_if_offline
