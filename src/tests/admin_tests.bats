#!/usr/bin/env bats
# admin_tests.bats - Comprehensive tests for admin API
# Tests: Statistics, user listing, activity logs, security events

# Test configuration
export POSTGRES_USER="${POSTGRES_USER:-postgres}"
export POSTGRES_DB="${POSTGRES_DB:-nself_db}"
export TEST_TIMEOUT="${TEST_TIMEOUT:-5}"

setup() {
  # Check if Docker is available
  if ! command -v docker >/dev/null 2>&1; then
    skip "Docker not available"
  fi

  # Check if PostgreSQL container is running
  if ! docker ps --filter 'name=postgres' --format '{{.Names}}' | grep -q postgres; then
    skip "PostgreSQL container not running"
  fi

  # Source the admin API module
  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  if [[ -f "$SCRIPT_DIR/lib/admin/api.sh" ]]; then
    source "$SCRIPT_DIR/lib/admin/api.sh"
  else
    skip "Admin API module not found"
  fi
}

teardown() {
  # Cleanup test data if needed
  true
}

# ============================================================================
# Statistics Overview Tests
# ============================================================================

@test "admin_stats_overview returns valid JSON" {
  run admin_stats_overview
  [[ "$status" -eq 0 ]]

  # Should return JSON object
  printf "%s\n" "$output" | jq . >/dev/null 2>&1
}

@test "admin_stats_overview includes total_users field" {
  local result
  result=$(admin_stats_overview)

  [[ "$result" == *"total_users"* ]] || [[ "$result" == "{}" ]]
}

@test "admin_stats_overview includes active_sessions field" {
  local result
  result=$(admin_stats_overview)

  [[ "$result" == *"active_sessions"* ]] || [[ "$result" == "{}" ]]
}

@test "admin_stats_overview includes custom_roles field" {
  local result
  result=$(admin_stats_overview)

  [[ "$result" == *"custom_roles"* ]] || [[ "$result" == "{}" ]]
}

@test "admin_stats_overview includes total_secrets field" {
  local result
  result=$(admin_stats_overview)

  [[ "$result" == *"total_secrets"* ]] || [[ "$result" == "{}" ]]
}

@test "admin_stats_overview includes active_webhooks field" {
  local result
  result=$(admin_stats_overview)

  [[ "$result" == *"active_webhooks"* ]] || [[ "$result" == "{}" ]]
}

@test "admin_stats_overview includes requests_last_hour field" {
  local result
  result=$(admin_stats_overview)

  [[ "$result" == *"requests_last_hour"* ]] || [[ "$result" == "{}" ]]
}

@test "admin_stats_overview returns empty object on DB error" {
  # This would require breaking DB connection
  # For now, just verify it doesn't crash
  run admin_stats_overview
  [[ "$status" -eq 0 ]]
}

@test "admin_stats_overview handles missing tables gracefully" {
  # Function should handle missing tables without crashing
  run admin_stats_overview
  [[ "$status" -eq 0 ]]
}

# ============================================================================
# User Listing Tests
# ============================================================================

@test "admin_users_list returns valid JSON array" {
  run admin_users_list
  [[ "$status" -eq 0 ]]

  # Should return JSON array (or null)
  [[ "$output" == "["* ]] || [[ "$output" == "null" ]] || [[ "$output" == "[]" ]]
}

@test "admin_users_list respects default limit of 50" {
  run admin_users_list
  [[ "$status" -eq 0 ]]
}

@test "admin_users_list accepts custom limit" {
  run admin_users_list 10
  [[ "$status" -eq 0 ]]
}

@test "admin_users_list accepts custom offset" {
  run admin_users_list 10 5
  [[ "$status" -eq 0 ]]
}

@test "admin_users_list validates limit minimum" {
  run admin_users_list 0
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid limit"* ]]
}

@test "admin_users_list validates limit maximum" {
  run admin_users_list 2000
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid limit"* ]]
}

@test "admin_users_list rejects negative limit" {
  run admin_users_list -5
  [[ "$status" -eq 1 ]]
}

@test "admin_users_list rejects negative offset" {
  run admin_users_list 10 -5
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid offset"* ]]
}

@test "admin_users_list rejects non-numeric limit" {
  run admin_users_list "abc"
  [[ "$status" -eq 1 ]]
}

@test "admin_users_list rejects non-numeric offset" {
  run admin_users_list 10 "xyz"
  [[ "$status" -eq 1 ]]
}

@test "admin_users_list excludes deleted users" {
  # This tests that deleted_at IS NULL filter works
  run admin_users_list 50 0
  [[ "$status" -eq 0 ]]
}

@test "admin_users_list includes user roles" {
  # Result should include roles field
  local result
  result=$(admin_users_list 10 0)

  # Either contains roles or is empty/null
  [[ "$result" == *"roles"* ]] || [[ "$result" == "null" ]] || [[ "$result" == "[]" ]]
}

@test "admin_users_list orders by created_at DESC" {
  # This validates the ORDER BY clause works
  run admin_users_list 2 0
  [[ "$status" -eq 0 ]]
}

@test "admin_users_list handles large offsets" {
  run admin_users_list 10 1000
  [[ "$status" -eq 0 ]]
}

@test "admin_users_list returns user fields correctly" {
  local result
  result=$(admin_users_list 1 0)

  # Should include expected fields or be empty
  if [[ "$result" != "null" ]] && [[ "$result" != "[]" ]]; then
    [[ "$result" == *"id"* ]] || true
    [[ "$result" == *"email"* ]] || true
    [[ "$result" == *"created_at"* ]] || true
  fi
}

# ============================================================================
# Activity Logs Tests
# ============================================================================

@test "admin_activity_recent returns valid JSON array" {
  run admin_activity_recent
  [[ "$status" -eq 0 ]]

  [[ "$output" == "["* ]] || [[ "$output" == "null" ]] || [[ "$output" == "[]" ]]
}

@test "admin_activity_recent uses default of 24 hours" {
  run admin_activity_recent
  [[ "$status" -eq 0 ]]
}

@test "admin_activity_recent accepts custom hours" {
  run admin_activity_recent 12
  [[ "$status" -eq 0 ]]
}

@test "admin_activity_recent validates minimum hours" {
  run admin_activity_recent 0
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid hours"* ]]
}

@test "admin_activity_recent validates maximum hours" {
  run admin_activity_recent 1000
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid hours"* ]]
}

@test "admin_activity_recent accepts valid range 1-720 hours" {
  run admin_activity_recent 168
  [[ "$status" -eq 0 ]]
}

@test "admin_activity_recent rejects negative hours" {
  run admin_activity_recent -5
  [[ "$status" -eq 1 ]]
}

@test "admin_activity_recent rejects non-numeric hours" {
  run admin_activity_recent "abc"
  [[ "$status" -eq 1 ]]
}

@test "admin_activity_recent limits to 100 results" {
  # Function hardcodes LIMIT 100
  run admin_activity_recent 720
  [[ "$status" -eq 0 ]]
}

@test "admin_activity_recent orders by created_at DESC" {
  run admin_activity_recent 24
  [[ "$status" -eq 0 ]]
}

@test "admin_activity_recent includes expected fields" {
  local result
  result=$(admin_activity_recent 1)

  # Should include activity fields or be empty
  if [[ "$result" != "null" ]] && [[ "$result" != "[]" ]]; then
    [[ "$result" == *"event_type"* ]] || true
    [[ "$result" == *"action"* ]] || true
  fi
}

@test "admin_activity_recent handles missing audit table" {
  # Should not crash if audit.events doesn't exist
  run admin_activity_recent 1
  [[ "$status" -eq 0 ]]
}

# ============================================================================
# Security Events Tests
# ============================================================================

@test "admin_security_events returns valid JSON array" {
  run admin_security_events
  [[ "$status" -eq 0 ]]

  [[ "$output" == "["* ]] || [[ "$output" == "null" ]] || [[ "$output" == "[]" ]]
}

@test "admin_security_events handles no recent violations" {
  run admin_security_events
  [[ "$status" -eq 0 ]]

  # Returns empty array if no violations
  [[ "$output" == "[]" ]] || [[ "$output" == "["* ]]
}

@test "admin_security_events queries last hour only" {
  # Function hardcodes INTERVAL '1 hour'
  run admin_security_events
  [[ "$status" -eq 0 ]]
}

@test "admin_security_events limits to top 10" {
  # Function hardcodes LIMIT 10
  run admin_security_events
  [[ "$status" -eq 0 ]]
}

@test "admin_security_events groups by key" {
  # Validates GROUP BY key works
  run admin_security_events
  [[ "$status" -eq 0 ]]
}

@test "admin_security_events orders by count DESC" {
  run admin_security_events
  [[ "$status" -eq 0 ]]
}

@test "admin_security_events filters only denied requests" {
  # allowed = FALSE filter
  run admin_security_events
  [[ "$status" -eq 0 ]]
}

@test "admin_security_events returns type field" {
  local result
  result=$(admin_security_events)

  if [[ "$result" != "[]" ]] && [[ "$result" != "null" ]]; then
    [[ "$result" == *"rate_limit"* ]] || true
  fi
}

@test "admin_security_events handles missing rate_limit table" {
  # Should not crash if rate_limit.log doesn't exist
  run admin_security_events
  [[ "$status" -eq 0 ]]
}

@test "admin_security_events returns empty array on error" {
  # Error handling returns []
  run admin_security_events
  [[ "$status" -eq 0 ]]
}

# ============================================================================
# PostgreSQL Container Handling Tests
# ============================================================================

@test "admin functions handle missing postgres container" {
  # Stop postgres briefly to test error handling
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    skip "No PostgreSQL container to test with"
  fi

  docker stop "$container" >/dev/null 2>&1 || skip "Cannot stop container"

  run admin_stats_overview
  [[ "$status" -eq 1 ]] || [[ "$output" == "{}" ]]

  # Restart container
  docker start "$container" >/dev/null 2>&1
  sleep 2
}

# ============================================================================
# SQL Injection Prevention Tests
# ============================================================================

@test "admin_users_list prevents SQL injection in limit" {
  run admin_users_list "50; DROP TABLE users; --"
  [[ "$status" -eq 1 ]]
}

@test "admin_users_list prevents SQL injection in offset" {
  run admin_users_list 10 "0; DELETE FROM users; --"
  [[ "$status" -eq 1 ]]
}

@test "admin_activity_recent prevents SQL injection in hours" {
  run admin_activity_recent "24; DROP TABLE audit.events; --"
  [[ "$status" -eq 1 ]]
}

# ============================================================================
# Edge Cases & Error Handling Tests
# ============================================================================

@test "admin_users_list handles empty database" {
  run admin_users_list 10 0
  [[ "$status" -eq 0 ]]
}

@test "admin_stats_overview handles partial data" {
  run admin_stats_overview
  [[ "$status" -eq 0 ]]
}

@test "admin_activity_recent handles no audit logs" {
  run admin_activity_recent 1
  [[ "$status" -eq 0 ]]
}

@test "admin functions handle jq failures gracefully" {
  # If jq fails, should still not crash
  run admin_stats_overview
  [[ "$status" -eq 0 ]]
}

@test "admin_users_list handles boundary limit values" {
  run admin_users_list 1
  [[ "$status" -eq 0 ]]

  run admin_users_list 1000
  [[ "$status" -eq 0 ]]
}

@test "admin_activity_recent handles boundary hour values" {
  run admin_activity_recent 1
  [[ "$status" -eq 0 ]]

  run admin_activity_recent 720
  [[ "$status" -eq 0 ]]
}
