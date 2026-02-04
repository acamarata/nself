#!/usr/bin/env bash
# network-mock.sh - Mock network operations for testing
#
# Provides mock HTTP requests, network delays, and timeout simulations
# without requiring actual network connectivity.

set -euo pipefail

# ============================================================================
# Mock HTTP State
# ============================================================================

MOCK_HTTP_STATE_DIR="${MOCK_HTTP_STATE_DIR:-/tmp/mock-http-$$}"
mkdir -p "$MOCK_HTTP_STATE_DIR/responses"
mkdir -p "$MOCK_HTTP_STATE_DIR/requests"

# Initialize mock HTTP
init_network_mock() {
  mkdir -p "$MOCK_HTTP_STATE_DIR/responses"
  mkdir -p "$MOCK_HTTP_STATE_DIR/requests"
}

# Cleanup mock HTTP
cleanup_network_mock() {
  [[ -d "$MOCK_HTTP_STATE_DIR" ]] && rm -rf "$MOCK_HTTP_STATE_DIR"
}

# ============================================================================
# Mock Response Configuration
# ============================================================================

# Register a mock HTTP response
# Usage: register_mock_response url status_code response_body
register_mock_response() {
  local url="$1"
  local status_code="$2"
  local response_body="$3"

  # Create safe filename from URL
  local safe_filename
  safe_filename=$(printf "%s" "$url" | sed 's|[^a-zA-Z0-9]|_|g')

  # Store response
  local response_file="$MOCK_HTTP_STATE_DIR/responses/$safe_filename"
  printf "status=%s\n" "$status_code" > "$response_file"
  printf "body=%s\n" "$response_body" >> "$response_file"
}

# Register mock response from file
# Usage: register_mock_response_from_file url status_code file_path
register_mock_response_from_file() {
  local url="$1"
  local status_code="$2"
  local file_path="$3"

  local response_body
  response_body=$(cat "$file_path")
  register_mock_response "$url" "$status_code" "$response_body"
}

# ============================================================================
# Mock curl
# ============================================================================

mock_curl() {
  local url=""
  local method="GET"
  local output_file=""
  local silent=false
  local show_headers=false
  local write_out=""
  local data=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -X|--request)
        method="$2"
        shift 2
        ;;
      -o|--output)
        output_file="$2"
        shift 2
        ;;
      -s|--silent)
        silent=true
        shift
        ;;
      -i|--include)
        show_headers=true
        shift
        ;;
      -w|--write-out)
        write_out="$2"
        shift 2
        ;;
      -d|--data)
        data="$2"
        method="POST"
        shift 2
        ;;
      -H|--header)
        # Ignore headers for now
        shift 2
        ;;
      http*|https*)
        url="$1"
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  # Log request
  local request_log="$MOCK_HTTP_STATE_DIR/requests/$(date +%s)_$$"
  printf "method=%s\nurl=%s\ndata=%s\n" "$method" "$url" "$data" > "$request_log"

  # Find mock response
  local safe_filename
  safe_filename=$(printf "%s" "$url" | sed 's|[^a-zA-Z0-9]|_|g')
  local response_file="$MOCK_HTTP_STATE_DIR/responses/$safe_filename"

  if [[ -f "$response_file" ]]; then
    local status_code
    local response_body

    status_code=$(grep "^status=" "$response_file" | cut -d= -f2)
    response_body=$(grep "^body=" "$response_file" | cut -d= -f2-)

    # Show headers if requested
    if [[ "$show_headers" == true ]]; then
      printf "HTTP/1.1 %s OK\n" "$status_code"
      printf "Content-Type: application/json\n"
      printf "Content-Length: %d\n" "${#response_body}"
      printf "\n"
    fi

    # Output response
    if [[ -n "$output_file" ]]; then
      printf "%s\n" "$response_body" > "$output_file"
    else
      printf "%s\n" "$response_body"
    fi

    # Handle write-out (for status code, etc.)
    if [[ -n "$write_out" ]]; then
      # Simple substitution for common write-out variables
      printf "%s\n" "$write_out" | sed "s/%{http_code}/$status_code/g"
    fi

    # Return success for 2xx status codes
    if [[ "$status_code" =~ ^2 ]]; then
      return 0
    else
      return 22  # curl error code for HTTP error
    fi
  else
    # No mock response registered
    if [[ "$silent" != true ]]; then
      printf "curl: (6) Could not resolve host: %s\n" "$url" >&2
    fi
    return 6
  fi
}

# ============================================================================
# Mock wget
# ============================================================================

mock_wget() {
  local url=""
  local output_file=""
  local quiet=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -O|--output-document)
        output_file="$2"
        shift 2
        ;;
      -q|--quiet)
        quiet=true
        shift
        ;;
      http*|https*)
        url="$1"
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  # Find mock response
  local safe_filename
  safe_filename=$(printf "%s" "$url" | sed 's|[^a-zA-Z0-9]|_|g')
  local response_file="$MOCK_HTTP_STATE_DIR/responses/$safe_filename"

  if [[ -f "$response_file" ]]; then
    local status_code
    local response_body

    status_code=$(grep "^status=" "$response_file" | cut -d= -f2)
    response_body=$(grep "^body=" "$response_file" | cut -d= -f2-)

    if [[ "$quiet" != true ]]; then
      printf "Saving to: '%s'\n" "${output_file:--}"
      printf "%s - %s saved\n" "$(date)" "${output_file:--}"
    fi

    # Output response
    if [[ -n "$output_file" ]] && [[ "$output_file" != "-" ]]; then
      printf "%s\n" "$response_body" > "$output_file"
    else
      printf "%s\n" "$response_body"
    fi

    return 0
  else
    printf "wget: unable to resolve host address '%s'\n" "$url" >&2
    return 4
  fi
}

# ============================================================================
# Network Simulation
# ============================================================================

# Simulate network latency
# Usage: simulate_network_delay milliseconds
simulate_network_delay() {
  local delay_ms="$1"
  local delay_sec=$(awk "BEGIN {print $delay_ms/1000}")
  sleep "$delay_sec"
}

# Simulate network timeout
# Usage: simulate_network_timeout (always returns timeout error)
simulate_network_timeout() {
  printf "curl: (28) Operation timed out\n" >&2
  return 28
}

# Simulate connection refused
# Usage: simulate_connection_refused
simulate_connection_refused() {
  printf "curl: (7) Failed to connect to localhost port 8080: Connection refused\n" >&2
  return 7
}

# ============================================================================
# Test Helpers
# ============================================================================

# Get all requests made to a URL
# Usage: get_requests_to_url url
get_requests_to_url() {
  local url="$1"
  local count=0

  find "$MOCK_HTTP_STATE_DIR/requests" -type f 2>/dev/null | while read -r request_file; do
    if grep -q "url=$url" "$request_file" 2>/dev/null; then
      count=$((count + 1))
      cat "$request_file"
      printf "\n---\n"
    fi
  done
}

# Assert request was made
# Usage: assert_request_made url [method]
assert_request_made() {
  local url="$1"
  local method="${2:-GET}"

  local found=false
  find "$MOCK_HTTP_STATE_DIR/requests" -type f 2>/dev/null | while read -r request_file; do
    if grep -q "url=$url" "$request_file" 2>/dev/null && \
       grep -q "method=$method" "$request_file" 2>/dev/null; then
      found=true
    fi
  done

  if [[ "$found" == true ]]; then
    return 0
  else
    printf "Assertion failed: No %s request made to %s\n" "$method" "$url" >&2
    return 1
  fi
}

# Clear all mock requests
clear_mock_requests() {
  rm -rf "$MOCK_HTTP_STATE_DIR/requests"/*
}

# ============================================================================
# Install Mocks
# ============================================================================

# Override curl
curl() {
  mock_curl "$@"
}

# Override wget
wget() {
  mock_wget "$@"
}

# Export functions
export -f curl
export -f wget
export -f mock_curl
export -f mock_wget
export -f register_mock_response
export -f register_mock_response_from_file
export -f simulate_network_delay
export -f simulate_network_timeout
export -f simulate_connection_refused
export -f get_requests_to_url
export -f assert_request_made
export -f clear_mock_requests
export -f init_network_mock
export -f cleanup_network_mock

# Auto-initialize
init_network_mock
