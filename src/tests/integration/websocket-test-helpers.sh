#!/usr/bin/env bash
# websocket-test-helpers.sh - Helper functions for WebSocket testing
# Part of nself v0.8.0 - Sprint 16: Real-Time Collaboration
#
# NOTE: These helpers are for future use when WebSocket server is implemented.
# Current real-time tests focus on database layer validation.
#
# Usage:
#   source websocket-test-helpers.sh
#   ws_test_connection "ws://localhost:8080/realtime"
#   ws_send_message "$ws_pid" '{"type":"subscribe","channel":"test"}'

set -euo pipefail

# ============================================================================
# WebSocket Connection Management
# ============================================================================

# Test if wscat is available
ws_check_wscat() {
  if command -v wscat >/dev/null 2>&1; then
    return 0
  elif command -v npm >/dev/null 2>&1; then
    printf "wscat not found. Install with: npm install -g wscat\n" >&2
    return 1
  else
    printf "wscat not available and npm not found\n" >&2
    return 1
  fi
}

# Establish WebSocket connection using wscat
# Args: $1 = WebSocket URL, $2 = Auth token (optional)
# Returns: PID of wscat process
ws_connect() {
  local url="$1"
  local token="${2:-}"
  local temp_dir
  temp_dir=$(mktemp -d)
  local log_file="$temp_dir/ws_output.log"

  if ! ws_check_wscat; then
    return 1
  fi

  # Start wscat in background
  if [[ -n "$token" ]]; then
    wscat -c "$url" -H "Authorization: Bearer $token" > "$log_file" 2>&1 &
  else
    wscat -c "$url" > "$log_file" 2>&1 &
  fi

  local pid=$!

  # Wait for connection
  sleep 1

  # Check if process is still running
  if ! kill -0 "$pid" 2>/dev/null; then
    printf "WebSocket connection failed\n" >&2
    return 1
  fi

  # Store log file path for later retrieval
  printf "%s:%s" "$pid" "$log_file"
}

# Send message over WebSocket
# Args: $1 = connection info (from ws_connect), $2 = message JSON
ws_send() {
  local conn_info="$1"
  local message="$2"
  local pid="${conn_info%%:*}"

  if ! kill -0 "$pid" 2>/dev/null; then
    printf "WebSocket connection not alive\n" >&2
    return 1
  fi

  # Send message to wscat stdin
  printf "%s\n" "$message" > "/proc/$pid/fd/0" 2>/dev/null || {
    printf "Failed to send message\n" >&2
    return 1
  }
}

# Read messages from WebSocket
# Args: $1 = connection info, $2 = timeout seconds (optional)
ws_read() {
  local conn_info="$1"
  local timeout="${2:-5}"
  local log_file="${conn_info#*:}"

  if [[ ! -f "$log_file" ]]; then
    printf "Log file not found\n" >&2
    return 1
  fi

  # Wait for messages with timeout
  local elapsed=0
  while [[ $elapsed -lt $timeout ]]; do
    if [[ -s "$log_file" ]]; then
      cat "$log_file"
      return 0
    fi
    sleep 1
    elapsed=$((elapsed + 1))
  done

  printf "Timeout waiting for messages\n" >&2
  return 1
}

# Close WebSocket connection
# Args: $1 = connection info (from ws_connect)
ws_disconnect() {
  local conn_info="$1"
  local pid="${conn_info%%:*}"
  local log_file="${conn_info#*:}"

  # Kill wscat process
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null
  fi

  # Clean up log file
  if [[ -f "$log_file" ]]; then
    rm -f "$log_file"
    rmdir "$(dirname "$log_file")" 2>/dev/null || true
  fi
}

# ============================================================================
# WebSocket Testing with curl
# ============================================================================

# Alternative: Test WebSocket upgrade using curl
# Args: $1 = URL, $2 = Auth token (optional)
ws_test_upgrade() {
  local url="$1"
  local token="${2:-}"
  local key
  key=$(printf "%s" "test-key-$(date +%s)" | base64)

  local headers=(
    -H "Connection: Upgrade"
    -H "Upgrade: websocket"
    -H "Sec-WebSocket-Key: $key"
    -H "Sec-WebSocket-Version: 13"
  )

  if [[ -n "$token" ]]; then
    headers+=(-H "Authorization: Bearer $token")
  fi

  # Test upgrade request
  local response
  response=$(curl -i -s "${headers[@]}" "$url")

  # Check for 101 Switching Protocols
  if printf "%s" "$response" | grep -q "101 Switching Protocols"; then
    return 0
  else
    printf "WebSocket upgrade failed:\n%s\n" "$response" >&2
    return 1
  fi
}

# ============================================================================
# PostgreSQL NOTIFY/LISTEN Testing
# ============================================================================

# Listen for PostgreSQL notifications
# Args: $1 = channel name, $2 = timeout seconds
pg_listen() {
  local channel="$1"
  local timeout="${2:-10}"
  local db_cmd

  # Get database connection
  db_cmd=$(get_db_connection)

  if [[ -z "$db_cmd" ]]; then
    printf "Cannot connect to database\n" >&2
    return 1
  fi

  # Start listening (this blocks)
  if command -v timeout >/dev/null 2>&1; then
    timeout "$timeout" $db_cmd "LISTEN $channel; SELECT pg_sleep($timeout);" 2>/dev/null || true
  else
    # Fallback without timeout
    $db_cmd "LISTEN $channel; SELECT pg_sleep($timeout);" 2>/dev/null || true
  fi
}

# Send PostgreSQL notification
# Args: $1 = channel name, $2 = payload
pg_notify() {
  local channel="$1"
  local payload="$2"
  local db_cmd

  db_cmd=$(get_db_connection)

  if [[ -z "$db_cmd" ]]; then
    printf "Cannot connect to database\n" >&2
    return 1
  fi

  $db_cmd "SELECT pg_notify('$channel', '$payload');" >/dev/null 2>&1
}

# ============================================================================
# Integration Test Helpers
# ============================================================================

# Test full real-time flow: connect → subscribe → send → receive
# Args: $1 = WebSocket URL, $2 = channel name, $3 = message
ws_integration_test() {
  local url="$1"
  local channel="$2"
  local message="$3"

  # Connect
  printf "Connecting to %s...\n" "$url"
  local conn
  conn=$(ws_connect "$url")
  if [[ -z "$conn" ]]; then
    printf "Connection failed\n" >&2
    return 1
  fi

  # Subscribe to channel
  printf "Subscribing to channel: %s\n" "$channel"
  local subscribe_msg
  subscribe_msg=$(printf '{"type":"subscribe","channel":"%s"}' "$channel")
  ws_send "$conn" "$subscribe_msg"

  sleep 1

  # Send message
  printf "Sending message: %s\n" "$message"
  local send_msg
  send_msg=$(printf '{"type":"message","channel":"%s","content":"%s"}' "$channel" "$message")
  ws_send "$conn" "$send_msg"

  # Read response
  printf "Reading response...\n"
  local response
  response=$(ws_read "$conn" 5)

  # Disconnect
  ws_disconnect "$conn"

  # Check response
  if printf "%s" "$response" | grep -q "$message"; then
    printf "✓ Integration test passed\n"
    return 0
  else
    printf "✗ Integration test failed\n" >&2
    return 1
  fi
}

# ============================================================================
# Load Testing Helpers
# ============================================================================

# Create multiple WebSocket connections for load testing
# Args: $1 = URL, $2 = number of connections
ws_load_test() {
  local url="$1"
  local count="${2:-10}"
  local pids=()

  printf "Creating %d WebSocket connections...\n" "$count"

  for i in $(seq 1 "$count"); do
    local conn
    conn=$(ws_connect "$url")
    if [[ -n "$conn" ]]; then
      pids+=("$conn")
    else
      printf "Failed to create connection %d\n" "$i" >&2
    fi
  done

  printf "Created %d connections\n" "${#pids[@]}"

  # Wait for user input to close
  printf "Press Enter to close all connections...\n"
  read -r

  # Close all connections
  for pid in "${pids[@]}"; do
    ws_disconnect "$pid"
  done

  printf "All connections closed\n"
}

# ============================================================================
# Example Usage
# ============================================================================

# Example: Test WebSocket connection
example_connection_test() {
  local url="ws://localhost:8080/realtime"

  if ws_test_upgrade "$url"; then
    printf "✓ WebSocket server is available\n"
  else
    printf "✗ WebSocket server is not available\n"
    return 1
  fi
}

# Example: Test NOTIFY/LISTEN
example_notify_listen_test() {
  local channel="test_channel"

  # Start listener in background
  printf "Starting listener...\n"
  pg_listen "$channel" 10 &
  local listener_pid=$!

  # Wait for listener to start
  sleep 2

  # Send notification
  printf "Sending notification...\n"
  pg_notify "$channel" '{"event":"test"}'

  # Wait for listener
  wait "$listener_pid"

  printf "✓ NOTIFY/LISTEN test complete\n"
}

# Example: Full integration test
example_full_test() {
  local url="ws://localhost:8080/realtime"
  local channel="test-channel"
  local message="Hello, WebSocket!"

  ws_integration_test "$url" "$channel" "$message"
}

# ============================================================================
# Helper for database connection (imported from test-realtime.sh)
# ============================================================================

get_db_connection() {
  local db_host="${POSTGRES_HOST:-localhost}"
  local db_port="${POSTGRES_PORT:-5432}"
  local db_user="${POSTGRES_USER:-postgres}"
  local db_name="${POSTGRES_DB:-nself}"

  local pg_container
  pg_container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -n "$pg_container" ]]; then
    printf "docker exec %s psql -U %s -d %s -t -A -c" "$pg_container" "$db_user" "$db_name"
  else
    printf "psql -h %s -p %s -U %s -d %s -t -A -c" "$db_host" "$db_port" "$db_user" "$db_name"
  fi
}

# ============================================================================
# Main - Run examples if executed directly
# ============================================================================

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  printf "\n=== WebSocket Test Helpers - Examples ===\n\n"

  printf "Example 1: Test WebSocket upgrade\n"
  example_connection_test || printf "Skipping (WebSocket server not running)\n"

  printf "\nExample 2: Test PostgreSQL NOTIFY/LISTEN\n"
  example_notify_listen_test || printf "Skipping (PostgreSQL not available)\n"

  printf "\nExample 3: Full integration test\n"
  printf "(Requires WebSocket server running)\n"
  # Uncomment to run:
  # example_full_test

  printf "\n=== Examples Complete ===\n\n"
  printf "To use these helpers in your tests:\n"
  printf "  source websocket-test-helpers.sh\n"
  printf "  ws_connect \"ws://localhost:8080/realtime\"\n"
fi
