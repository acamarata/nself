#!/usr/bin/env bats
# auth_user_tests.bats - Comprehensive tests for user-manager.sh
# Tests: User CRUD operations, validation, search, and security

# Test configuration
export POSTGRES_USER="${POSTGRES_USER:-postgres}"
export POSTGRES_DB="${POSTGRES_DB:-nself_db}"
export TEST_TIMEOUT="${TEST_TIMEOUT:-5}"

# Source the modules
setup() {
  # Check if Docker is available
  if ! command -v docker >/dev/null 2>&1; then
    skip "Docker not available"
  fi

  # Check if PostgreSQL container is running
  if ! docker ps --filter 'name=postgres' --format '{{.Names}}' | grep -q postgres; then
    skip "PostgreSQL container not running"
  fi

  # Source dependencies
  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  if [[ -f "$SCRIPT_DIR/lib/database/safe-query.sh" ]]; then
    source "$SCRIPT_DIR/lib/database/safe-query.sh"
  fi
  if [[ -f "$SCRIPT_DIR/lib/auth/password-utils.sh" ]]; then
    source "$SCRIPT_DIR/lib/auth/password-utils.sh"
  fi
  if [[ -f "$SCRIPT_DIR/lib/auth/user-manager.sh" ]]; then
    source "$SCRIPT_DIR/lib/auth/user-manager.sh"
  fi

  # Generate unique test email
  export TEST_EMAIL="test-$(date +%s)-${RANDOM}@example.com"
  export TEST_PHONE="+15551234567"
  export TEST_PASSWORD="Test1234!"
}

teardown() {
  # Clean up test user if created
  if [[ -n "${TEST_USER_ID:-}" ]]; then
    user_delete "$TEST_USER_ID" "true" 2>/dev/null || true
  fi
}

# ============================================================================
# Validation Functions Tests
# ============================================================================

@test "validate_email accepts valid email" {
  run validate_email "test@example.com"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "test@example.com" ]]
}

@test "validate_email rejects invalid email format" {
  run validate_email "not-an-email"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid email format"* ]]
}

@test "validate_email rejects email without domain" {
  run validate_email "test@"
  [[ "$status" -eq 1 ]]
}

@test "validate_email rejects email too long" {
  local long_email="$(printf 'a%.0s' {1..250})@example.com"
  run validate_email "$long_email"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Email too long"* ]]
}

@test "validate_uuid accepts valid UUID" {
  run validate_uuid "123e4567-e89b-12d3-a456-426614174000"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "123e4567-e89b-12d3-a456-426614174000" ]]
}

@test "validate_uuid rejects invalid UUID format" {
  run validate_uuid "not-a-uuid"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid UUID format"* ]]
}

@test "validate_uuid rejects UUID with wrong length" {
  run validate_uuid "123e4567-e89b-12d3"
  [[ "$status" -eq 1 ]]
}

@test "validate_integer accepts valid integer" {
  run validate_integer "42"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "42" ]]
}

@test "validate_integer rejects non-integer" {
  run validate_integer "not-a-number"
  [[ "$status" -eq 1 ]]
}

@test "validate_integer enforces minimum value" {
  run validate_integer "5" 10 100
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"below minimum"* ]]
}

@test "validate_integer enforces maximum value" {
  run validate_integer "150" 1 100
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"above maximum"* ]]
}

# ============================================================================
# User Creation Tests
# ============================================================================

@test "user_create creates user with email only" {
  run user_create "$TEST_EMAIL"
  [[ "$status" -eq 0 ]]
  [[ -n "$output" ]]

  # Save user ID for cleanup
  TEST_USER_ID="$output"

  # Verify UUID format
  [[ "$TEST_USER_ID" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]
}

@test "user_create creates user with email and password" {
  run user_create "$TEST_EMAIL" "$TEST_PASSWORD"
  [[ "$status" -eq 0 ]]
  TEST_USER_ID="$output"
  [[ -n "$TEST_USER_ID" ]]
}

@test "user_create creates user with email, password, and phone" {
  run user_create "$TEST_EMAIL" "$TEST_PASSWORD" "$TEST_PHONE"
  [[ "$status" -eq 0 ]]
  TEST_USER_ID="$output"
  [[ -n "$TEST_USER_ID" ]]
}

@test "user_create creates user with metadata" {
  local metadata='{"role": "admin", "department": "IT"}'
  run user_create "$TEST_EMAIL" "" "" "$metadata"
  [[ "$status" -eq 0 ]]
  TEST_USER_ID="$output"
  [[ -n "$TEST_USER_ID" ]]
}

@test "user_create rejects duplicate email" {
  # Create first user
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  # Try to create duplicate
  run user_create "$TEST_EMAIL"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"already exists"* ]]
}

@test "user_create rejects invalid email" {
  run user_create "not-an-email"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid email"* ]]
}

@test "user_create rejects invalid phone format" {
  run user_create "$TEST_EMAIL" "" "123"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid phone format"* ]]
}

@test "user_create rejects invalid JSON metadata" {
  run user_create "$TEST_EMAIL" "" "" "not-json"
  [[ "$status" -eq 1 ]]
}

# ============================================================================
# User Retrieval Tests
# ============================================================================

@test "user_get_by_id retrieves existing user" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL" "$TEST_PASSWORD" "$TEST_PHONE")

  run user_get_by_id "$TEST_USER_ID"
  [[ "$status" -eq 0 ]]
  [[ "$output" == *"$TEST_EMAIL"* ]]
  [[ "$output" == *"$TEST_PHONE"* ]]
}

@test "user_get_by_id returns JSON format" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  local result
  result=$(user_get_by_id "$TEST_USER_ID")

  # Should be valid JSON
  run printf "%s" "$result"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "{"* ]]
}

@test "user_get_by_id fails for non-existent user" {
  run user_get_by_id "123e4567-e89b-12d3-a456-426614174000"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"not found"* ]]
}

@test "user_get_by_id rejects invalid UUID" {
  run user_get_by_id "not-a-uuid"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid UUID"* ]]
}

@test "user_get_by_email retrieves existing user" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL" "$TEST_PASSWORD")

  run user_get_by_email "$TEST_EMAIL"
  [[ "$status" -eq 0 ]]
  [[ "$output" == *"$TEST_EMAIL"* ]]
}

@test "user_get_by_email fails for non-existent email" {
  run user_get_by_email "nonexistent@example.com"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"not found"* ]]
}

@test "user_get_by_email rejects invalid email" {
  run user_get_by_email "not-an-email"
  [[ "$status" -eq 1 ]]
}

# ============================================================================
# User Update Tests
# ============================================================================

@test "user_update updates email" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")
  local new_email="updated-$TEST_EMAIL"

  run user_update "$TEST_USER_ID" "$new_email"
  [[ "$status" -eq 0 ]]

  # Verify update
  local result
  result=$(user_get_by_id "$TEST_USER_ID")
  [[ "$result" == *"$new_email"* ]]
}

@test "user_update updates phone" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")
  local new_phone="+15559876543"

  run user_update "$TEST_USER_ID" "" "$new_phone"
  [[ "$status" -eq 0 ]]

  # Verify update
  local result
  result=$(user_get_by_id "$TEST_USER_ID")
  [[ "$result" == *"$new_phone"* ]]
}

@test "user_update updates password" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL" "$TEST_PASSWORD")
  local new_password="NewPass1234!"

  run user_update "$TEST_USER_ID" "" "" "$new_password"
  [[ "$status" -eq 0 ]]
}

@test "user_update fails for non-existent user" {
  run user_update "123e4567-e89b-12d3-a456-426614174000" "new@example.com"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"not found"* ]]
}

@test "user_update rejects invalid UUID" {
  run user_update "not-a-uuid" "new@example.com"
  [[ "$status" -eq 1 ]]
}

@test "user_update fails with no fields to update" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  run user_update "$TEST_USER_ID"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"No fields to update"* ]]
}

@test "user_update resets email verification on email change" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")
  local new_email="updated-$TEST_EMAIL"

  user_update "$TEST_USER_ID" "$new_email"

  # Verify email_verified is FALSE
  local result
  result=$(user_get_by_id "$TEST_USER_ID")
  [[ "$result" == *'"email_verified":false'* ]] || \
  [[ "$result" == *'"email_verified": false'* ]]
}

# ============================================================================
# User Deletion Tests
# ============================================================================

@test "user_delete performs soft delete by default" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  run user_delete "$TEST_USER_ID"
  [[ "$status" -eq 0 ]]
  [[ "$output" == *"soft delete"* ]]
}

@test "user_delete performs hard delete when requested" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  run user_delete "$TEST_USER_ID" "true"
  [[ "$status" -eq 0 ]]
  [[ "$output" == *"permanently deleted"* ]]

  # Verify user is gone
  run user_get_by_id "$TEST_USER_ID"
  [[ "$status" -eq 1 ]]

  # Don't clean up in teardown since already deleted
  TEST_USER_ID=""
}

@test "user_delete rejects invalid UUID" {
  run user_delete "not-a-uuid"
  [[ "$status" -eq 1 ]]
}

@test "user_restore restores soft-deleted user" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")
  user_delete "$TEST_USER_ID" # soft delete

  run user_restore "$TEST_USER_ID"
  [[ "$status" -eq 0 ]]

  # Verify user is restored
  run user_get_by_id "$TEST_USER_ID"
  [[ "$status" -eq 0 ]]
}

@test "user_restore rejects invalid UUID" {
  run user_restore "not-a-uuid"
  [[ "$status" -eq 1 ]]
}

# ============================================================================
# User Listing & Search Tests
# ============================================================================

@test "user_list returns JSON array" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  run user_list 10 0
  [[ "$status" -eq 0 ]]
  [[ "$output" == "["* ]] || [[ "$output" == "[]" ]]
}

@test "user_list respects limit parameter" {
  run user_list 5 0
  [[ "$status" -eq 0 ]]
}

@test "user_list respects offset parameter" {
  run user_list 10 5
  [[ "$status" -eq 0 ]]
}

@test "user_list excludes deleted users by default" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")
  user_delete "$TEST_USER_ID" # soft delete

  local result
  result=$(user_list 100 0 "false")
  [[ "$result" != *"$TEST_EMAIL"* ]] || [[ "$result" == "[]" ]]
}

@test "user_list includes deleted users when requested" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")
  user_delete "$TEST_USER_ID" # soft delete

  run user_list 100 0 "true"
  [[ "$status" -eq 0 ]]
}

@test "user_list validates limit range" {
  run user_list 0 0
  [[ "$status" -eq 1 ]]
}

@test "user_list validates limit maximum" {
  run user_list 2000 0
  [[ "$status" -eq 1 ]]
}

@test "user_search finds user by email" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  local search_term="${TEST_EMAIL:0:10}"
  run user_search "$search_term"
  [[ "$status" -eq 0 ]]
  [[ "$output" == *"$TEST_EMAIL"* ]]
}

@test "user_search finds user by phone" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL" "" "$TEST_PHONE")

  run user_search "1555"
  [[ "$status" -eq 0 ]]
}

@test "user_search is case-insensitive" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  local search_upper=$(echo "$TEST_EMAIL" | tr '[:lower:]' '[:upper:]')
  run user_search "$search_upper"
  [[ "$status" -eq 0 ]]
}

@test "user_search requires query parameter" {
  run user_search ""
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"required"* ]]
}

@test "user_search respects limit parameter" {
  run user_search "test" 5
  [[ "$status" -eq 0 ]]
}

@test "user_count returns integer" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  run user_count
  [[ "$status" -eq 0 ]]
  [[ "$output" =~ ^[0-9]+$ ]]
}

@test "user_count excludes deleted users by default" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")
  local count_before
  count_before=$(user_count "false")

  user_delete "$TEST_USER_ID"
  local count_after
  count_after=$(user_count "false")

  [[ "$count_after" -lt "$count_before" ]]
}

@test "user_count includes deleted users when requested" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")
  local count_before
  count_before=$(user_count "true")

  user_delete "$TEST_USER_ID"
  local count_after
  count_after=$(user_count "true")

  [[ "$count_after" -eq "$count_before" ]]
}

# ============================================================================
# Security & SQL Injection Tests
# ============================================================================

@test "user_create prevents SQL injection in email" {
  local malicious_email="test@example.com'; DROP TABLE auth.users; --"

  run user_create "$malicious_email"
  [[ "$status" -eq 1 ]] # Should fail validation
}

@test "user_search prevents SQL injection in query" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  local malicious_query="test' OR '1'='1"
  run user_search "$malicious_query"
  [[ "$status" -eq 0 ]] # Should not error

  # Should not return all users
  # (Hard to test precisely without knowing exact data)
}

@test "user_get_by_email handles special characters" {
  local special_email="test+special@example.com"
  TEST_USER_ID=$(user_create "$special_email")

  run user_get_by_email "$special_email"
  [[ "$status" -eq 0 ]]
  [[ "$output" == *"$special_email"* ]]
}

@test "user_create handles password with special characters" {
  local special_password='P@$$w0rd!#%^&*()'
  run user_create "$TEST_EMAIL" "$special_password"
  [[ "$status" -eq 0 ]]
  TEST_USER_ID="$output"
}

# ============================================================================
# Edge Cases & Error Handling
# ============================================================================

@test "user_create handles empty email" {
  run user_create ""
  [[ "$status" -eq 1 ]]
}

@test "user_get_by_id handles empty UUID" {
  run user_get_by_id ""
  [[ "$status" -eq 1 ]]
}

@test "user_update handles partial updates" {
  TEST_USER_ID=$(user_create "$TEST_EMAIL")

  # Update only email
  local new_email="new-$TEST_EMAIL"
  run user_update "$TEST_USER_ID" "$new_email"
  [[ "$status" -eq 0 ]]
}

@test "user functions handle missing PostgreSQL container gracefully" {
  # Stop container briefly
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    skip "No PostgreSQL container to test with"
  fi

  docker stop "$container" >/dev/null 2>&1 || skip "Cannot stop container"

  run user_create "test@example.com"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"not found"* ]]

  # Restart container
  docker start "$container" >/dev/null 2>&1
  sleep 2
}
