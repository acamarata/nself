#!/usr/bin/env bash
# role-manager.sh - Role management system (ROLE-001, ROLE-002)
# Part of nself v0.6.0 - Phase 1 Sprint 3
#
# Implements role-based access control (RBAC) with permissions

set -euo pipefail

# ============================================================================
# Role CRUD Operations
# ============================================================================

# Create a new role
# Usage: role_create <role_name> <description>
role_create() {
  local role_name="$1"
  local description="${2:-}"

  if [[ -z "$role_name" ]]; then
    echo "ERROR: Role name required" >&2
    return 1
  fi

  # Validate role name (alphanumeric, underscore, hyphen)
  if ! echo "$role_name" | grep -qE '^[a-zA-Z0-9_-]+$'; then
    echo "ERROR: Invalid role name. Use only letters, numbers, underscore, and hyphen" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Create roles table if it doesn't exist
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" <<'EOSQL' >/dev/null 2>&1
CREATE TABLE IF NOT EXISTS auth.roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT UNIQUE NOT NULL,
  description TEXT,
  is_default BOOLEAN DEFAULT FALSE,
  is_system BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_roles_name ON auth.roles(name);
CREATE INDEX IF NOT EXISTS idx_roles_is_default ON auth.roles(is_default);
EOSQL

  # Check if role already exists
  local existing_role
  existing_role=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT id FROM auth.roles WHERE name = '$role_name' LIMIT 1;" \
    2>/dev/null | xargs)

  if [[ -n "$existing_role" ]]; then
    echo "ERROR: Role '$role_name' already exists" >&2
    return 1
  fi

  # Escape description
  description=$(echo "$description" | sed "s/'/''/g")

  # Create role
  local role_id
  role_id=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "INSERT INTO auth.roles (name, description)
     VALUES ('$role_name', '$description')
     RETURNING id;" \
    2>/dev/null | xargs)

  if [[ -z "$role_id" ]]; then
    echo "ERROR: Failed to create role" >&2
    return 1
  fi

  echo "$role_id"
  return 0
}

# Get role by ID
# Usage: role_get_by_id <role_id>
role_get_by_id() {
  local role_id="$1"

  if [[ -z "$role_id" ]]; then
    echo "ERROR: Role ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Get role
  local role_json
  role_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT row_to_json(r) FROM (
       SELECT id, name, description, is_default, is_system, created_at, updated_at
       FROM auth.roles
       WHERE id = '$role_id'
     ) r;" \
    2>/dev/null | xargs)

  if [[ -z "$role_json" ]] || [[ "$role_json" == "null" ]]; then
    echo "ERROR: Role not found" >&2
    return 1
  fi

  echo "$role_json"
  return 0
}

# Get role by name
# Usage: role_get_by_name <role_name>
role_get_by_name() {
  local role_name="$1"

  if [[ -z "$role_name" ]]; then
    echo "ERROR: Role name required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Get role
  local role_json
  role_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT row_to_json(r) FROM (
       SELECT id, name, description, is_default, is_system, created_at, updated_at
       FROM auth.roles
       WHERE name = '$role_name'
     ) r;" \
    2>/dev/null | xargs)

  if [[ -z "$role_json" ]] || [[ "$role_json" == "null" ]]; then
    echo "ERROR: Role not found" >&2
    return 1
  fi

  echo "$role_json"
  return 0
}

# Update role
# Usage: role_update <role_id> <name> <description>
role_update() {
  local role_id="$1"
  local new_name="${2:-}"
  local new_description="${3:-}"

  if [[ -z "$role_id" ]]; then
    echo "ERROR: Role ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Check if role is system role
  local is_system
  is_system=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT is_system FROM auth.roles WHERE id = '$role_id' LIMIT 1;" \
    2>/dev/null | xargs)

  if [[ "$is_system" == "t" ]]; then
    echo "ERROR: Cannot modify system role" >&2
    return 1
  fi

  # Build update query
  local updates=()

  if [[ -n "$new_name" ]]; then
    # Validate name
    if ! echo "$new_name" | grep -qE '^[a-zA-Z0-9_-]+$'; then
      echo "ERROR: Invalid role name" >&2
      return 1
    fi
    updates+=("name = '$new_name'")
  fi

  if [[ -n "$new_description" ]]; then
    new_description=$(echo "$new_description" | sed "s/'/''/g")
    updates+=("description = '$new_description'")
  fi

  if [[ ${#updates[@]} -eq 0 ]]; then
    echo "ERROR: No fields to update" >&2
    return 1
  fi

  updates+=("updated_at = NOW()")

  # Join updates
  local update_clause=$(IFS=', '; echo "${updates[*]}")

  # Update role
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "UPDATE auth.roles SET $update_clause WHERE id = '$role_id';" \
    >/dev/null 2>&1

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Failed to update role" >&2
    return 1
  fi

  return 0
}

# Delete role
# Usage: role_delete <role_id>
role_delete() {
  local role_id="$1"

  if [[ -z "$role_id" ]]; then
    echo "ERROR: Role ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Check if role is system role
  local is_system
  is_system=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT is_system FROM auth.roles WHERE id = '$role_id' LIMIT 1;" \
    2>/dev/null | xargs)

  if [[ "$is_system" == "t" ]]; then
    echo "ERROR: Cannot delete system role" >&2
    return 1
  fi

  # Delete role
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "DELETE FROM auth.roles WHERE id = '$role_id';" \
    >/dev/null 2>&1

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Failed to delete role" >&2
    return 1
  fi

  return 0
}

# List all roles
# Usage: role_list [limit] [offset]
role_list() {
  local limit="${1:-50}"
  local offset="${2:-0}"

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Get roles
  local roles_json
  roles_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(r) FROM (
       SELECT id, name, description, is_default, is_system, created_at, updated_at
       FROM auth.roles
       ORDER BY is_system DESC, name ASC
       LIMIT $limit OFFSET $offset
     ) r;" \
    2>/dev/null | xargs)

  if [[ -z "$roles_json" ]] || [[ "$roles_json" == "null" ]]; then
    echo "[]"
    return 0
  fi

  echo "$roles_json"
  return 0
}

# ============================================================================
# Default Role Management
# ============================================================================

# Set role as default
# Usage: role_set_default <role_id>
role_set_default() {
  local role_id="$1"

  if [[ -z "$role_id" ]]; then
    echo "ERROR: Role ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Unset all default roles
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "UPDATE auth.roles SET is_default = FALSE;" \
    >/dev/null 2>&1

  # Set this role as default
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "UPDATE auth.roles SET is_default = TRUE WHERE id = '$role_id';" \
    >/dev/null 2>&1

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Failed to set default role" >&2
    return 1
  fi

  return 0
}

# Get default role
# Usage: role_get_default
role_get_default() {
  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Get default role
  local role_json
  role_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT row_to_json(r) FROM (
       SELECT id, name, description, is_default, is_system
       FROM auth.roles
       WHERE is_default = TRUE
       LIMIT 1
     ) r;" \
    2>/dev/null | xargs)

  if [[ -z "$role_json" ]] || [[ "$role_json" == "null" ]]; then
    echo "{}"
    return 0
  fi

  echo "$role_json"
  return 0
}

# ============================================================================
# User-Role Assignment
# ============================================================================

# Assign role to user
# Usage: role_assign_user <user_id> <role_id>
role_assign_user() {
  local user_id="$1"
  local role_id="$2"

  if [[ -z "$user_id" ]] || [[ -z "$role_id" ]]; then
    echo "ERROR: User ID and role ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Create user_roles table if it doesn't exist
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" <<'EOSQL' >/dev/null 2>&1
CREATE TABLE IF NOT EXISTS auth.user_roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
  role_id UUID REFERENCES auth.roles(id) ON DELETE CASCADE,
  assigned_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON auth.user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON auth.user_roles(role_id);
EOSQL

  # Assign role
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "INSERT INTO auth.user_roles (user_id, role_id)
     VALUES ('$user_id', '$role_id')
     ON CONFLICT (user_id, role_id) DO NOTHING;" \
    >/dev/null 2>&1

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Failed to assign role" >&2
    return 1
  fi

  return 0
}

# Revoke role from user
# Usage: role_revoke_user <user_id> <role_id>
role_revoke_user() {
  local user_id="$1"
  local role_id="$2"

  if [[ -z "$user_id" ]] || [[ -z "$role_id" ]]; then
    echo "ERROR: User ID and role ID required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Revoke role
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "DELETE FROM auth.user_roles
     WHERE user_id = '$user_id' AND role_id = '$role_id';" \
    >/dev/null 2>&1

  if [[ $? -ne 0 ]]; then
    echo "ERROR: Failed to revoke role" >&2
    return 1
  fi

  return 0
}

# Get user roles
# Usage: role_get_user_roles <user_id>
role_get_user_roles() {
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

  # Get user roles
  local roles_json
  roles_json=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(r) FROM (
       SELECT r.id, r.name, r.description, ur.assigned_at
       FROM auth.user_roles ur
       JOIN auth.roles r ON ur.role_id = r.id
       WHERE ur.user_id = '$user_id'
       ORDER BY r.name
     ) r;" \
    2>/dev/null | xargs)

  if [[ -z "$roles_json" ]] || [[ "$roles_json" == "null" ]]; then
    echo "[]"
    return 0
  fi

  echo "$roles_json"
  return 0
}

# ============================================================================
# Export functions
# ============================================================================

export -f role_create
export -f role_get_by_id
export -f role_get_by_name
export -f role_update
export -f role_delete
export -f role_list
export -f role_set_default
export -f role_get_default
export -f role_assign_user
export -f role_revoke_user
export -f role_get_user_roles
