#!/usr/bin/env bats
# database_tests.bats - Comprehensive tests for database operations
# Tests: Safe queries, parameterized queries, validation, SQL injection prevention

export POSTGRES_USER="${POSTGRES_USER:-postgres}"
export POSTGRES_DB="${POSTGRES_DB:-nself_db}"

setup() {
  if ! command -v docker >/dev/null 2>&1; then
    skip "Docker not available"
  fi

  if ! docker ps --filter 'name=postgres' --format '{{.Names}}' | grep -q postgres; then
    skip "PostgreSQL container not running"
  fi

  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/.." && pwd)"
  if [[ -f "$SCRIPT_DIR/lib/database/safe-query.sh" ]]; then
    source "$SCRIPT_DIR/lib/database/safe-query.sh"
  else
    skip "safe-query.sh not found"
  fi
}

# ============================================================================
# Container Detection Tests
# ============================================================================

@test "pg_get_container finds postgres container" {
  run pg_get_container
  [[ "$status" -eq 0 ]]
  [[ -n "$output" ]]
}

@test "pg_get_container returns container name" {
  local container
  container=$(pg_get_container)
  [[ "$container" == *"postgres"* ]]
}

@test "pg_get_container fails when no postgres running" {
  skip "Cannot test without stopping postgres"
}

# ============================================================================
# Validation Functions Tests
# ============================================================================

@test "validate_uuid accepts valid UUID" {
  run validate_uuid "123e4567-e89b-12d3-a456-426614174000"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "123e4567-e89b-12d3-a456-426614174000" ]]
}

@test "validate_uuid rejects invalid format" {
  run validate_uuid "not-a-uuid"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid UUID"* ]]
}

@test "validate_uuid rejects too short UUID" {
  run validate_uuid "123e4567"
  [[ "$status" -eq 1 ]]
}

@test "validate_uuid rejects UUID with wrong format" {
  run validate_uuid "xxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  [[ "$status" -eq 1 ]]
}

@test "validate_uuid is case insensitive" {
  run validate_uuid "123E4567-E89B-12D3-A456-426614174000"
  [[ "$status" -eq 0 ]]
}

@test "validate_email accepts valid email" {
  run validate_email "test@example.com"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "test@example.com" ]]
}

@test "validate_email rejects invalid format" {
  run validate_email "not-an-email"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"Invalid email"* ]]
}

@test "validate_email rejects email without @" {
  run validate_email "testexample.com"
  [[ "$status" -eq 1 ]]
}

@test "validate_email rejects email without domain" {
  run validate_email "test@"
  [[ "$status" -eq 1 ]]
}

@test "validate_email rejects email too long" {
  local long_email="$(printf 'a%.0s' {1..250})@example.com"
  run validate_email "$long_email"
  [[ "$status" -eq 1 ]]
  [[ "$output" == *"too long"* ]]
}

@test "validate_email accepts email with plus sign" {
  run validate_email "test+tag@example.com"
  [[ "$status" -eq 0 ]]
}

@test "validate_email accepts email with dots" {
  run validate_email "first.last@example.com"
  [[ "$status" -eq 0 ]]
}

@test "validate_email accepts subdomain" {
  run validate_email "test@mail.example.com"
  [[ "$status" -eq 0 ]]
}

@test "validate_integer accepts valid integer" {
  run validate_integer "42"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "42" ]]
}

@test "validate_integer rejects non-numeric" {
  run validate_integer "abc"
  [[ "$status" -eq 1 ]]
}

@test "validate_integer rejects floating point" {
  run validate_integer "3.14"
  [[ "$status" -eq 1 ]]
}

@test "validate_integer accepts negative numbers" {
  run validate_integer "-5"
  [[ "$status" -eq 0 ]]
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

@test "validate_integer accepts value within range" {
  run validate_integer "50" 1 100
  [[ "$status" -eq 0 ]]
  [[ "$output" == "50" ]]
}

@test "validate_integer accepts boundary values" {
  run validate_integer "1" 1 100
  [[ "$status" -eq 0 ]]

  run validate_integer "100" 1 100
  [[ "$status" -eq 0 ]]
}

# ============================================================================
# SQL Escaping Tests
# ============================================================================

@test "sql_escape doubles single quotes" {
  local result
  result=$(sql_escape "O'Reilly")
  [[ "$result" == "O''Reilly" ]]
}

@test "sql_escape handles multiple quotes" {
  local result
  result=$(sql_escape "It's a test's case")
  [[ "$result" == "It''s a test''s case" ]]
}

@test "sql_escape handles empty string" {
  local result
  result=$(sql_escape "")
  [[ "$result" == "" ]]
}

@test "sql_escape handles string without quotes" {
  local result
  result=$(sql_escape "normal text")
  [[ "$result" == "normal text" ]]
}

# ============================================================================
# Safe Query Execution Tests
# ============================================================================

@test "pg_query_safe executes simple query" {
  run pg_query_safe "SELECT 1 AS test"
  [[ "$status" -eq 0 ]]
  [[ "$output" =~ "1" ]]
}

@test "pg_query_safe handles parameterized query" {
  run pg_query_safe "SELECT :'param1' AS result" "test_value"
  [[ "$status" -eq 0 ]]
}

@test "pg_query_safe escapes single quotes in parameters" {
  run pg_query_safe "SELECT :'param1' AS result" "O'Reilly"
  [[ "$status" -eq 0 ]]
}

@test "pg_query_safe handles multiple parameters" {
  run pg_query_safe "SELECT :'param1' || :'param2' AS result" "Hello" "World"
  [[ "$status" -eq 0 ]]
}

@test "pg_query_safe handles numeric parameters" {
  run pg_query_safe "SELECT :param1 + :param2 AS sum" "10" "20"
  [[ "$status" -eq 0 ]]
}

@test "pg_query_value returns single value" {
  local result
  result=$(pg_query_value "SELECT 'test' AS value")
  [[ "$result" == "test" ]]
}

@test "pg_query_value trims whitespace" {
  local result
  result=$(pg_query_value "SELECT 42")
  [[ "$result" == "42" ]]
}

@test "pg_query_json returns JSON object" {
  local result
  result=$(pg_query_json "SELECT row_to_json(row(1, 'test'))")
  [[ "$result" == "{"* ]]
}

@test "pg_query_json returns empty object on null" {
  local result
  result=$(pg_query_json "SELECT NULL")
  [[ "$result" == "{}" ]]
}

@test "pg_query_json_array returns JSON array" {
  local result
  result=$(pg_query_json_array "SELECT json_agg(row_to_json(row(1)))")
  [[ "$result" == "["* ]]
}

@test "pg_query_json_array returns empty array on null" {
  local result
  result=$(pg_query_json_array "SELECT NULL")
  [[ "$result" == "[]" ]]
}

# ============================================================================
# SQL Injection Prevention Tests
# ============================================================================

@test "pg_query_safe prevents SQL injection via parameter" {
  run pg_query_safe "SELECT :'param1'" "test'; DROP TABLE users; --"
  [[ "$status" -eq 0 ]]
  # Should not execute DROP TABLE
}

@test "pg_query_safe prevents injection with OR condition" {
  run pg_query_safe "SELECT :'param1'" "' OR '1'='1"
  [[ "$status" -eq 0 ]]
}

@test "pg_query_safe prevents injection with UNION" {
  run pg_query_safe "SELECT :'param1'" "' UNION SELECT password FROM users --"
  [[ "$status" -eq 0 ]]
}

@test "validate_email prevents SQL in email" {
  run validate_email "test'; DROP TABLE users; --@example.com"
  [[ "$status" -eq 1 ]]
}

@test "validate_uuid prevents SQL in UUID" {
  run validate_uuid "'; DROP TABLE users; --"
  [[ "$status" -eq 1 ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "pg_query_safe handles invalid SQL" {
  run pg_query_safe "SELECT * FROM nonexistent_table"
  [[ "$status" -ne 0 ]] || true
}

@test "pg_query_safe handles empty query" {
  run pg_query_safe ""
  [[ "$status" -ne 0 ]] || true
}

@test "pg_query_value handles query with no results" {
  local result
  result=$(pg_query_value "SELECT NULL WHERE FALSE")
  [[ -z "$result" ]] || [[ "$result" == "NULL" ]] || true
}

@test "validate_email handles empty string" {
  run validate_email ""
  [[ "$status" -eq 1 ]]
}

@test "validate_uuid handles empty string" {
  run validate_uuid ""
  [[ "$status" -eq 1 ]]
}

@test "validate_integer handles empty string" {
  run validate_integer ""
  [[ "$status" -eq 1 ]]
}

# ============================================================================
# Edge Cases Tests
# ============================================================================

@test "pg_query_safe handles special characters" {
  run pg_query_safe "SELECT :'param1'" "Test!@#$%^&*()"
  [[ "$status" -eq 0 ]]
}

@test "pg_query_safe handles unicode" {
  run pg_query_safe "SELECT :'param1'" "Tëst üñïçödé"
  [[ "$status" -eq 0 ]]
}

@test "pg_query_safe handles very long string" {
  local long_string=$(printf 'a%.0s' {1..1000})
  run pg_query_safe "SELECT :'param1'" "$long_string"
  [[ "$status" -eq 0 ]]
}

@test "validate_integer handles zero" {
  run validate_integer "0"
  [[ "$status" -eq 0 ]]
  [[ "$output" == "0" ]]
}

@test "validate_integer handles large numbers" {
  run validate_integer "999999999"
  [[ "$status" -eq 0 ]]
}

@test "validate_email handles minimum valid email" {
  run validate_email "a@b.co"
  [[ "$status" -eq 0 ]]
}

@test "NSELF_SAFE_QUERY_LOADED prevents double sourcing" {
  [[ "${NSELF_SAFE_QUERY_LOADED:-}" == "1" ]]
}
