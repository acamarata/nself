#!/usr/bin/env bash
# user-manager.sh - User CRUD operations (USER-001)
# Part of nself v0.6.0 - Phase 1 Sprint 2
#
# Implements comprehensive user management operations
# Create, Read, Update, Delete, List, Search

set -euo pipefail

# Source password utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$SCRIPT_DIR/password-utils.sh" ]]; then
  source "$SCRIPT_DIR/password-utils.sh"
fi

# ============================================================================
# User Creation
# ============================================================================

# Create a new user
# Usage: user_create <email> [password] [phone] [metadata_json]
# Returns: User ID
user_create() {
  local email="$1"
  local password="${2:-}"
  local phone="${3:-}"
  local metadata_json="${4:-{}}"

  # Validate email
  if [[ -z "$email" ]]; then
    echo "ERROR: Email required" >&2
    return 1
  fi

  if ! echo "$email" | grep -qE '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'; then
    echo "ERROR: Invalid email format" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Check if user already exists
  local existing_user
  existing_user=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT id FROM auth.users WHERE email = '$email' LIMIT 1;" \
    2>/dev/null | xargs)

  if [[ -n "$existing_user" ]]; then
    echo "ERROR: User with email '$email' already exists" >&2
    return 1
  fi

  # Hash password if provided
  local password_hash=""
  if [[ -n "$password" ]]; then
    password_hash=$(hash_password "$password")
    if [[ -z "$password_hash" ]]; then
      echo "ERROR: Failed to hash password" >&2
      return 1
    fi
  fi

  # Build INSERT query
  local phone_clause=""
  if [[ -n "$phone" ]]; then
    phone_clause=", phone = '$phone'"
  fi

  local password_clause=""
  if [[ -n "$password_hash" ]]; then
    password_clause=", password_hash = '$password_hash'"
  fi

  # Create user
  local user_id
  user_id=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "INSERT INTO auth.users (email${phone_clause:+, phone}${password_clause:+, password_hash}, created_at, last_sign_in_at)
     VALUES ('$email'${phone:+, '$phone'}${password_hash:+, '$password_hash'}, NOW(), NULL)
     RETURNING id;" \
    2>/dev/null | xargs)

  if [[ -z "$user_id" ]]; then
    echo "ERROR: Failed to create user" >&2
    return 1
  fi

  # Store metadata if provided
  if [[ "$metadata_json" != "{}" ]]; then
    docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
      "INSERT INTO auth.user_metadata (user_id, metadata)
       VALUES ('$user_id', '$metadata_json'::jsonb);" \
      >/dev/null 2>&1
  fi

  echo "$user_id"
  return 0
}

# ============================================================================
# User Retrieval
# ============================================================================

# Get user by ID
# Usage: user_get_by_id <user_id>
# Returns: JSON user object
user_get_by_id() {
  local user_id="$1"

  if [[ -z "$user_id" ]]; then
    echo "ERROR: User ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Get user data
  local user_json
  user_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT row_to_json(u) FROM (
       SELECT
         id,
         email,
         phone,
         mfa_enabled,
         email_verified,
         phone_verified,
         created_at,
         last_sign_in_at
       FROM auth.users
       WHERE id = '$user_id'
     ) u;" \
    2>/dev/null | xargs)

  if [[ -z "$user_json" ]] || [[ "$user_json" == "null" ]]; then
    echo "ERROR: User not found" >&2
    return 1
  fi

  echo "$user_json"
  return 0
}

# Get user by email
# Usage: user_get_by_email <email>
# Returns: JSON user object
user_get_by_email() {
  local email="$1"

  if [[ -z "$email" ]]; then
    echo "ERROR: Email required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Get user data
  local user_json
  user_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT row_to_json(u) FROM (
       SELECT
         id,
         email,
         phone,
         mfa_enabled,
         email_verified,
         phone_verified,
         created_at,
         last_sign_in_at
       FROM auth.users
       WHERE email = '$email'
     ) u;" \
    2>/dev/null | xargs)

  if [[ -z "$user_json" ]] || [[ "$user_json" == "null" ]]; then
    echo "ERROR: User not found" >&2
    return 1
  fi

  echo "$user_json"
  return 0
}

# ============================================================================
# User Update
# ============================================================================

# Update user
# Usage: user_update <user_id> [email] [phone] [password]
user_update() {
  local user_id="$1"
  local new_email="${2:-}"
  local new_phone="${3:-}"
  local new_password="${4:-}"

  if [[ -z "$user_id" ]]; then
    echo "ERROR: User ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Check if user exists
  local existing_user
  existing_user=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT id FROM auth.users WHERE id = '$user_id' LIMIT 1;" \
    2>/dev/null | xargs)

  if [[ -z "$existing_user" ]]; then
    echo "ERROR: User not found" >&2
    return 1
  fi

  # Build UPDATE query
  local updates=()

  if [[ -n "$new_email" ]]; then
    # Validate email format
    if ! echo "$new_email" | grep -qE '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'; then
      echo "ERROR: Invalid email format" >&2
      return 1
    fi
    updates+=("email = '$new_email'")
    updates+=("email_verified = FALSE")
  fi

  if [[ -n "$new_phone" ]]; then
    updates+=("phone = '$new_phone'")
    updates+=("phone_verified = FALSE")
  fi

  if [[ -n "$new_password" ]]; then
    local password_hash
    password_hash=$(hash_password "$new_password")
    if [[ -z "$password_hash" ]]; then
      echo "ERROR: Failed to hash password" >&2
      return 1
    fi
    updates+=("password_hash = '$password_hash'")
  fi

  if [[ ${#updates[@]} -eq 0 ]]; then
    echo "ERROR: No fields to update" >&2
    return 1
  fi

  # Join updates with commas
  local update_clause=$(IFS=', '; echo "${updates[*]}")

  # Execute update
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "UPDATE auth.users
     SET $update_clause
     WHERE id = '$user_id';" \
    >/dev/null 2>&1

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Failed to update user" >&2
    return 1
  fi

  echo "✓ User updated successfully" >&2
  return 0
}

# ============================================================================
# User Deletion
# ============================================================================

# Delete user (soft delete)
# Usage: user_delete <user_id> [hard_delete]
user_delete() {
  local user_id="$1"
  local hard_delete="${2:-false}"

  if [[ -z "$user_id" ]]; then
    echo "ERROR: User ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  if [[ "$hard_delete" == "true" ]]; then
    # Hard delete - permanently remove user and all related data
    docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
      "DELETE FROM auth.users WHERE id = '$user_id';" \
      >/dev/null 2>&1

    if [[ $? -ne 0 ]]; then
      echo "ERROR: Failed to delete user" >&2
      return 1
    fi

    echo "✓ User permanently deleted" >&2
  else
    # Soft delete - add deleted_at timestamp
    # First, create deleted_at column if it doesn't exist
    docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
      "ALTER TABLE auth.users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;" \
      >/dev/null 2>&1

    # Mark as deleted
    docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
      "UPDATE auth.users
       SET deleted_at = NOW()
       WHERE id = '$user_id';" \
      >/dev/null 2>&1

    if [[ $? -ne 0 ]]; then
      echo "ERROR: Failed to delete user" >&2
      return 1
    fi

    echo "✓ User marked as deleted (soft delete)" >&2
  fi

  return 0
}

# Restore deleted user
# Usage: user_restore <user_id>
user_restore() {
  local user_id="$1"

  if [[ -z "$user_id" ]]; then
    echo "ERROR: User ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Restore user
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "UPDATE auth.users
     SET deleted_at = NULL
     WHERE id = '$user_id';" \
    >/dev/null 2>&1

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Failed to restore user" >&2
    return 1
  fi

  echo "✓ User restored successfully" >&2
  return 0
}

# ============================================================================
# User Listing & Search
# ============================================================================

# List all users
# Usage: user_list [limit] [offset] [include_deleted]
# Returns: JSON array of users
user_list() {
  local limit="${1:-50}"
  local offset="${2:-0}"
  local include_deleted="${3:-false}"

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Build query
  local where_clause=""
  if [[ "$include_deleted" != "true" ]]; then
    where_clause="WHERE deleted_at IS NULL"
  fi

  # Get users
  local users_json
  users_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(u) FROM (
       SELECT
         id,
         email,
         phone,
         mfa_enabled,
         email_verified,
         phone_verified,
         created_at,
         last_sign_in_at,
         deleted_at
       FROM auth.users
       $where_clause
       ORDER BY created_at DESC
       LIMIT $limit OFFSET $offset
     ) u;" \
    2>/dev/null | xargs)

  if [[ -z "$users_json" ]] || [[ "$users_json" == "null" ]]; then
    echo "[]"
    return 0
  fi

  echo "$users_json"
  return 0
}

# Search users
# Usage: user_search <query> [limit]
# Returns: JSON array of users
user_search() {
  local query="$1"
  local limit="${2:-50}"

  if [[ -z "$query" ]]; then
    echo "ERROR: Search query required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Search users by email or phone
  local users_json
  users_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(u) FROM (
       SELECT
         id,
         email,
         phone,
         mfa_enabled,
         email_verified,
         phone_verified,
         created_at,
         last_sign_in_at
       FROM auth.users
       WHERE deleted_at IS NULL
         AND (email ILIKE '%$query%' OR phone ILIKE '%$query%')
       ORDER BY created_at DESC
       LIMIT $limit
     ) u;" \
    2>/dev/null | xargs)

  if [[ -z "$users_json" ]] || [[ "$users_json" == "null" ]]; then
    echo "[]"
    return 0
  fi

  echo "$users_json"
  return 0
}

# Count total users
# Usage: user_count [include_deleted]
# Returns: Integer count
user_count() {
  local include_deleted="${1:-false}"

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Build query
  local where_clause=""
  if [[ "$include_deleted" != "true" ]]; then
    where_clause="WHERE deleted_at IS NULL"
  fi

  # Get count
  local count
  count=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT COUNT(*) FROM auth.users $where_clause;" \
    2>/dev/null | xargs)

  echo "${count:-0}"
  return 0
}

# ============================================================================
# Export functions
# ============================================================================

export -f user_create
export -f user_get_by_id
export -f user_get_by_email
export -f user_update
export -f user_delete
export -f user_restore
export -f user_list
export -f user_search
export -f user_count
