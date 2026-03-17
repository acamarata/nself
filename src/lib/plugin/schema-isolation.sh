#!/usr/bin/env bash
# schema-isolation.sh — Plugin PG schema + role provisioning
# Bash 3.2 compatible. No echo -e, no ${var,,}, no declare -A, no mapfile.
#
# Called by plugin.sh after plugin files are installed, before nself build.
# For each plugin, ensures:
#   1. Postgres schema  np_{name}  exists
#   2. DB role          np_{name}_role  exists
#   3. USAGE on schema  granted to role
#   4. search_path      set on role: np_{name},public
#   5. PLUGIN_SCHEMA    exported as env var for container use

set -o pipefail

# Source platform compatibility helpers if not already loaded
if ! declare -f safe_sed_inline >/dev/null 2>&1; then
  _SCHEMA_ISO_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  # shellcheck source=../utils/platform-compat.sh
  source "$_SCHEMA_ISO_SCRIPT_DIR/../utils/platform-compat.sh" 2>/dev/null || true
fi

# ---------------------------------------------------------------------------
# _plugin_schema_db_exec <sql>
# Execute SQL against the running Postgres container, or direct psql.
# Uses the same pattern as plugin_db_exec in core.sh.
# Returns 0 on success, non-zero on failure.
# ---------------------------------------------------------------------------
_plugin_schema_db_exec() {
  local sql="$1"
  local project_name="${PROJECT_NAME:-nself}"
  local db_container="${project_name}_postgres"
  local db_user="${POSTGRES_USER:-postgres}"
  local db_name="${POSTGRES_DB:-${project_name}}"

  if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${db_container}$"; then
    docker exec -i "$db_container" psql -U "$db_user" -d "$db_name" \
      -v ON_ERROR_STOP=1 -c "$sql" >/dev/null 2>&1
    return $?
  fi

  if command -v psql >/dev/null 2>&1; then
    local db_host="${POSTGRES_HOST:-localhost}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_pass="${POSTGRES_PASSWORD:-}"
    local uri
    uri="postgresql://${db_user}:${db_pass}@${db_host}:${db_port}/${db_name}"
    psql "$uri" -v ON_ERROR_STOP=1 -c "$sql" >/dev/null 2>&1
    return $?
  fi

  # No postgres accessible — non-fatal, skip schema creation
  return 2
}

# ---------------------------------------------------------------------------
# _plugin_schema_db_exec_quiet <sql>
# Same as above but suppresses ON_ERROR_STOP so idempotent IF NOT EXISTS
# statements don't fail when the object already exists.
# ---------------------------------------------------------------------------
_plugin_schema_db_exec_quiet() {
  local sql="$1"
  local project_name="${PROJECT_NAME:-nself}"
  local db_container="${project_name}_postgres"
  local db_user="${POSTGRES_USER:-postgres}"
  local db_name="${POSTGRES_DB:-${project_name}}"

  if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${db_container}$"; then
    docker exec -i "$db_container" psql -U "$db_user" -d "$db_name" \
      -c "$sql" >/dev/null 2>&1 || true
    return 0
  fi

  if command -v psql >/dev/null 2>&1; then
    local db_host="${POSTGRES_HOST:-localhost}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_pass="${POSTGRES_PASSWORD:-}"
    local uri
    uri="postgresql://${db_user}:${db_pass}@${db_host}:${db_port}/${db_name}"
    psql "$uri" -c "$sql" >/dev/null 2>&1 || true
    return 0
  fi

  return 2
}

# ---------------------------------------------------------------------------
# _plugin_schema_postgres_running
# Returns 0 if the Postgres container or a reachable psql is available.
# ---------------------------------------------------------------------------
_plugin_schema_postgres_running() {
  local project_name="${PROJECT_NAME:-nself}"
  local db_container="${project_name}_postgres"

  if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${db_container}$"; then
    return 0
  fi

  if command -v psql >/dev/null 2>&1; then
    local db_host="${POSTGRES_HOST:-localhost}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_user="${POSTGRES_USER:-postgres}"
    local db_pass="${POSTGRES_PASSWORD:-}"
    local db_name="${POSTGRES_DB:-${project_name}}"
    local uri
    uri="postgresql://${db_user}:${db_pass}@${db_host}:${db_port}/${db_name}"
    psql "$uri" -c "SELECT 1" >/dev/null 2>&1
    return $?
  fi

  return 1
}

# ---------------------------------------------------------------------------
# create_plugin_schema <plugin_name>
#
# Idempotent. Creates np_{name} schema and np_{name}_role, then:
#   - GRANT USAGE ON SCHEMA np_{name} TO np_{name}_role
#   - ALTER ROLE np_{name}_role SET search_path = np_{name},public
#   - Exports PLUGIN_SCHEMA=np_{name} into the environment
#
# Skips silently when Postgres is not accessible (install continues).
# ---------------------------------------------------------------------------
create_plugin_schema() {
  local plugin_name="$1"

  if [ -z "$plugin_name" ]; then
    printf '\033[0;31m[ERROR]\033[0m create_plugin_schema: plugin name required\n' >&2
    return 1
  fi

  # Build schema and role names — lowercase only, replace hyphens with underscores
  local schema_name
  schema_name="np_$(printf '%s' "$plugin_name" | tr '[:upper:]' '[:lower:]' | tr '-' '_')"
  local role_name="${schema_name}_role"

  # Check Postgres availability
  if ! _plugin_schema_postgres_running; then
    printf '\033[0;33m[WARNING]\033[0m Plugin schema setup skipped: Postgres not running (will be applied on next start)\n' >&2
    export PLUGIN_SCHEMA="$schema_name"
    return 0
  fi

  printf '\033[0;34m[INFO]\033[0m Setting up schema isolation for plugin: %s\n' "$plugin_name"
  printf '\033[0;34m[INFO]\033[0m  schema: %s\n' "$schema_name"
  printf '\033[0;34m[INFO]\033[0m  role:   %s\n' "$role_name"

  # 1. Create role if not exists
  _plugin_schema_db_exec_quiet "DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = '${role_name}') THEN
      CREATE ROLE ${role_name} NOLOGIN;
    END IF;
  END \$\$;"

  # 2. Create schema if not exists (owned by superuser, accessible to role)
  _plugin_schema_db_exec_quiet "CREATE SCHEMA IF NOT EXISTS ${schema_name};"

  # 3. Grant USAGE on schema to role
  _plugin_schema_db_exec_quiet "GRANT USAGE ON SCHEMA ${schema_name} TO ${role_name};"

  # 4. Allow role to create objects in its schema
  _plugin_schema_db_exec_quiet "GRANT CREATE ON SCHEMA ${schema_name} TO ${role_name};"

  # 5. Set default search_path for the role
  _plugin_schema_db_exec_quiet "ALTER ROLE ${role_name} SET search_path = ${schema_name}, public;"

  # 6. Ensure np_common schema exists with schema_versions tracking table
  _plugin_schema_db_exec_quiet "CREATE SCHEMA IF NOT EXISTS np_common;"
  _plugin_schema_db_exec_quiet "CREATE TABLE IF NOT EXISTS np_common.schema_versions (
    plugin TEXT NOT NULL,
    version INT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plugin, version)
  );"

  printf '\033[0;32m[SUCCESS]\033[0m Plugin schema isolation ready: %s\n' "$schema_name"

  # 7. Export PLUGIN_SCHEMA for use by the plugin container
  export PLUGIN_SCHEMA="$schema_name"

  return 0
}

# ---------------------------------------------------------------------------
# check_plugin_schema_isolation <plugin_name>
# Used by nself doctor to warn if np_{name}_* tables still exist in public.
# Returns 0 if no public tables found (clean), 1 if tables need migration.
# Prints a warning when tables are found.
# ---------------------------------------------------------------------------
check_plugin_schema_isolation() {
  local plugin_name="$1"

  if [ -z "$plugin_name" ]; then
    return 0
  fi

  local schema_prefix
  schema_prefix="np_$(printf '%s' "$plugin_name" | tr '[:upper:]' '[:lower:]' | tr '-' '_')_"

  # Check if Postgres is available — if not, skip silently
  if ! _plugin_schema_postgres_running; then
    return 0
  fi

  local project_name="${PROJECT_NAME:-nself}"
  local db_container="${project_name}_postgres"
  local db_user="${POSTGRES_USER:-postgres}"
  local db_name="${POSTGRES_DB:-${project_name}}"

  local count=0
  local sql
  sql="SELECT COUNT(*) FROM information_schema.tables
       WHERE table_schema = 'public'
       AND table_name LIKE '${schema_prefix}%';"

  if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${db_container}$"; then
    count=$(docker exec -i "$db_container" psql -U "$db_user" -d "$db_name" -t -c "$sql" 2>/dev/null | tr -d ' \n' || printf '0')
  elif command -v psql >/dev/null 2>&1; then
    local db_host="${POSTGRES_HOST:-localhost}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_pass="${POSTGRES_PASSWORD:-}"
    local uri
    uri="postgresql://${db_user}:${db_pass}@${db_host}:${db_port}/${db_name}"
    count=$(psql "$uri" -t -c "$sql" 2>/dev/null | tr -d ' \n' || printf '0')
  fi

  if [ "${count:-0}" -gt 0 ] 2>/dev/null; then
    printf '\033[0;33m[WARNING]\033[0m Plugin "%s" has %s table(s) in public schema (schema isolation incomplete)\n' \
      "$plugin_name" "$count" >&2
    printf '           Run: nself plugin migrate-schema %s\n' "$plugin_name" >&2
    return 1
  fi

  return 0
}

# Export functions
export -f create_plugin_schema 2>/dev/null || true
export -f check_plugin_schema_isolation 2>/dev/null || true
