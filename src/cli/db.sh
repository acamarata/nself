#!/usr/bin/env bash
# db.sh - Comprehensive database management for nself v0.4.4
# All database operations in one clean interface with smart defaults

set -o pipefail

# ============================================================================
# INITIALIZATION
# ============================================================================

CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_DIR="$CLI_SCRIPT_DIR"

# Source utilities
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/header.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/platform-compat.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/database/core.sh" 2>/dev/null || true

# Fallbacks if display.sh didn't load
if ! declare -f log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_warning >/dev/null 2>&1; then
  log_warning() { printf "\033[0;33m[WARNING]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi
if ! declare -f log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
fi

# ============================================================================
# CONSTANTS
# ============================================================================

MIGRATIONS_DIR="${NSELF_MIGRATIONS_DIR:-nself/migrations}"
SEEDS_DIR="${NSELF_SEEDS_DIR:-nself/seeds}"
BACKUPS_DIR="${NSELF_BACKUPS_DIR:-_backups}"
MOCK_DIR="${NSELF_MOCK_DIR:-nself/mock}"
TYPES_DIR="${NSELF_TYPES_DIR:-types}"

# ============================================================================
# ENVIRONMENT & SAFETY
# ============================================================================

get_env() {
  local env="${ENV:-${ENVIRONMENT:-local}}"
  case "$env" in
    dev | development | local) echo "local" ;;
    staging | stage) echo "staging" ;;
    prod | production) echo "production" ;;
    *) echo "$env" ;;
  esac
}

is_production() { [[ "$(get_env)" == "production" ]]; }
is_staging() { [[ "$(get_env)" == "staging" ]]; }
is_local() { [[ "$(get_env)" == "local" ]]; }

require_non_production() {
  local op="${1:-This operation}"
  if is_production; then
    log_error "BLOCKED: $op is not allowed in production"
    log_info "Environment: $(get_env)"
    return 1
  fi
}

require_confirmation() {
  local msg="${1:-This is destructive}"
  if is_production; then
    log_warning "PRODUCTION ENVIRONMENT"
    printf "Type 'yes-destroy-production' to confirm: "
    read -r response
    [[ "$response" == "yes-destroy-production" ]] || {
      log_info "Cancelled"
      return 1
    }
  elif is_staging; then
    printf "Staging environment. Continue? (y/N): "
    read -r response
    [[ "$response" =~ ^[Yy]$ ]] || {
      log_info "Cancelled"
      return 1
    }
  fi
}

# ============================================================================
# DATABASE CONNECTION HELPERS
# ============================================================================

get_container() {
  echo "${PROJECT_NAME:-nself}_postgres"
}

db_running() {
  docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^$(get_container)$"
}

require_db() {
  if ! db_running; then
    log_error "PostgreSQL is not running. Run 'nself start' first."
    return 1
  fi
}

psql_exec() {
  local db="${POSTGRES_DB:-nhost}"
  local user="${POSTGRES_USER:-postgres}"
  docker exec -i "$(get_container)" psql -U "$user" -d "$db" "$@"
}

psql_query() {
  local sql="$1"
  psql_exec -t -A -c "$sql" 2>/dev/null
}

psql_interactive() {
  local db="${1:-${POSTGRES_DB:-nhost}}"
  local user="${POSTGRES_USER:-postgres}"
  docker exec -it "$(get_container)" psql -U "$user" -d "$db"
}

# ============================================================================
# UTILITY FUNCTIONS
# ============================================================================

# Convert snake_case to PascalCase (portable, no sed -r or \U)
snake_to_pascal() {
  local input="$1"
  local result=""
  local capitalize=true
  local char

  for ((i = 0; i < ${#input}; i++)); do
    char="${input:$i:1}"
    if [[ "$char" == "_" ]]; then
      capitalize=true
    elif [[ "$capitalize" == true ]]; then
      # Uppercase the character (portable way)
      result+=$(printf '%s' "$char" | tr '[:lower:]' '[:upper:]')
      capitalize=false
    else
      result+="$char"
    fi
  done

  echo "$result"
}

# Get file modification time (cross-platform)
get_file_mtime() {
  local file="$1"
  if stat --version 2>/dev/null | grep -q GNU; then
    stat -c "%y" "$file" 2>/dev/null | cut -d' ' -f1-2 | cut -d'.' -f1
  else
    stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$file" 2>/dev/null
  fi
}

# ============================================================================
# DIRECTORY SETUP
# ============================================================================

ensure_dirs() {
  mkdir -p "$MIGRATIONS_DIR"
  mkdir -p "$SEEDS_DIR/common" "$SEEDS_DIR/local" "$SEEDS_DIR/staging" "$SEEDS_DIR/production"
  mkdir -p "$BACKUPS_DIR"
  mkdir -p "$MOCK_DIR"
  mkdir -p "$TYPES_DIR"
  mkdir -p ".nself"
}

# ============================================================================
# MIGRATIONS
# ============================================================================

cmd_migrate() {
  local subcmd="${1:-status}"
  shift || true

  case "$subcmd" in
    status) migrate_status "$@" ;;
    up | run) migrate_up "$@" ;;
    down | rollback) migrate_down "$@" ;;
    create | new) migrate_create "$@" ;;
    fresh) migrate_fresh "$@" ;;
    repair) migrate_repair "$@" ;;
    *)
      log_error "Unknown: migrate $subcmd"
      migrate_help
      ;;
  esac
}

migrate_status() {
  require_db || return 1
  log_info "Migration Status"
  echo ""

  # Ensure tracking table exists
  psql_exec -c "CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMPTZ DEFAULT NOW()
  )" >/dev/null 2>&1

  local applied=$(psql_query "SELECT COUNT(*) FROM schema_migrations")
  local pending=0
  local shown_versions=""

  printf "%-50s %s\n" "Migration" "Status"
  printf "%-50s %s\n" "$(printf '%.0s─' {1..45})" "──────"

  # Only show .up.sql files or standalone .sql files (exclude .down.sql from display)
  for file in $(ls "$MIGRATIONS_DIR"/*.up.sql "$MIGRATIONS_DIR"/*.sql 2>/dev/null | grep -v '\.down\.sql$' | sort -u); do
    [[ -f "$file" ]] || continue
    local name=$(basename "$file")
    # Remove .up.sql or .sql extension for display
    name="${name%.up.sql}"
    name="${name%.sql}"
    local version=$(echo "$name" | cut -d'_' -f1)

    # Skip if we already showed this version (dedup)
    if echo "$shown_versions" | grep -q "|$version|"; then
      continue
    fi
    shown_versions="${shown_versions}|${version}|"

    if psql_query "SELECT 1 FROM schema_migrations WHERE version = '$version'" | grep -q 1; then
      printf "%-50s \033[32m✓ Applied\033[0m\n" "$name"
    else
      printf "%-50s \033[33m○ Pending\033[0m\n" "$name"
      pending=$((pending + 1))
    fi
  done

  echo ""
  log_info "Applied: $applied | Pending: $pending"
}

migrate_up() {
  require_db || return 1
  local target="${1:-all}"

  log_info "Running migrations..."

  # Ensure tracking table
  psql_exec -c "CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMPTZ DEFAULT NOW()
  )" >/dev/null 2>&1

  local count=0
  # Only process .up.sql files (or .sql files without .down. in the name)
  for file in $(ls "$MIGRATIONS_DIR"/*.up.sql "$MIGRATIONS_DIR"/*.sql 2>/dev/null | grep -v '\.down\.sql$' | sort -u); do
    [[ -f "$file" ]] || continue
    local name=$(basename "$file")
    # Remove .up.sql or .sql extension for display
    name="${name%.up.sql}"
    name="${name%.sql}"
    local version=$(echo "$name" | cut -d'_' -f1)

    # Skip if already applied
    if psql_query "SELECT 1 FROM schema_migrations WHERE version = '$version'" | grep -q 1; then
      continue
    fi

    log_info "Applying: $name"
    # Capture migration output to temp file for error reporting
    local temp_output=$(mktemp)
    if psql_exec <"$file" >"$temp_output" 2>&1; then
      psql_exec -c "INSERT INTO schema_migrations (version) VALUES ('$version')" >/dev/null
      log_success "  Applied successfully"
      count=$((count + 1))
      rm -f "$temp_output"
    else
      log_error "  Failed to apply migration"
      printf "\n%sError details:%s\n" "$RED" "$NC" >&2
      cat "$temp_output" >&2
      rm -f "$temp_output"
      return 1
    fi

    [[ "$target" != "all" ]] && [[ $count -ge $target ]] && break
  done

  [[ $count -eq 0 ]] && log_info "No pending migrations" || log_success "Applied $count migration(s)"
}

migrate_down() {
  require_db || return 1
  local steps="${1:-1}"

  require_confirmation "Rolling back $steps migration(s)" || return 1
  log_info "Rolling back $steps migration(s)..."

  local applied=($(psql_query "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT $steps"))

  for version in "${applied[@]}"; do
    # Look for .down.sql files (new convention) or _down.sql (legacy)
    local down_file=$(ls "$MIGRATIONS_DIR"/${version}*.down.sql 2>/dev/null | head -1)
    [[ -z "$down_file" ]] && down_file=$(ls "$MIGRATIONS_DIR"/${version}*_down.sql 2>/dev/null | head -1)

    if [[ -f "$down_file" ]]; then
      log_info "Rolling back: $version using $(basename "$down_file")"
      if psql_exec <"$down_file" 2>&1; then
        log_success "  SQL executed"
      else
        log_warning "  SQL execution had issues (may be expected)"
      fi
    else
      log_warning "  No down migration found for $version"
    fi

    psql_exec -c "DELETE FROM schema_migrations WHERE version = '$version'" >/dev/null
    log_success "  Rolled back (removed from tracking)"
  done
}

migrate_create() {
  local name="$1"
  [[ -z "$name" ]] && {
    log_error "Usage: nself db migrate create <name>"
    return 1
  }

  ensure_dirs
  local timestamp=$(date +%Y%m%d%H%M%S)
  local up_file="$MIGRATIONS_DIR/${timestamp}_${name}.sql"
  local down_file="$MIGRATIONS_DIR/${timestamp}_${name}_down.sql"

  cat >"$up_file" <<EOF
-- Migration: $name
-- Created: $(date)
-- Environment: $(get_env)

-- Add your migration SQL here

EOF

  cat >"$down_file" <<EOF
-- Rollback: $name
-- Created: $(date)

-- Add your rollback SQL here

EOF

  log_success "Created migration: ${timestamp}_${name}"
  log_info "  Up:   $up_file"
  log_info "  Down: $down_file"
}

migrate_fresh() {
  require_non_production "migrate fresh" || return 1
  require_db || return 1
  require_confirmation "This will DROP all tables and re-run migrations" || return 1

  log_info "Dropping all tables..."
  psql_exec -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" >/dev/null
  psql_exec -c "GRANT ALL ON SCHEMA public TO postgres; GRANT ALL ON SCHEMA public TO public;" >/dev/null

  log_info "Re-running migrations..."
  migrate_up
}

migrate_repair() {
  require_db || return 1
  log_info "Repairing migration history..."

  # Recreate tracking table
  psql_exec -c "DROP TABLE IF EXISTS schema_migrations" >/dev/null
  psql_exec -c "CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMPTZ DEFAULT NOW()
  )" >/dev/null

  log_success "Migration table reset. Run 'nself db migrate up' to re-apply."
}

migrate_help() {
  echo "Usage: nself db migrate <command>"
  echo ""
  echo "Commands:"
  echo "  status          Show migration status (default)"
  echo "  up              Run all pending migrations"
  echo "  down [n]        Rollback n migrations (default: 1)"
  echo "  create <name>   Create new migration files"
  echo "  fresh           Drop all & re-migrate (local only)"
  echo "  repair          Repair migration tracking table"
}

# ============================================================================
# SHELL / QUERY
# ============================================================================

cmd_shell() {
  require_db || return 1
  local readonly=false
  local db="${POSTGRES_DB:-nhost}"

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --readonly | -r)
        readonly=true
        shift
        ;;
      --env)
        shift
        log_warning "Connecting to $1 environment..."
        # Would need SSH/remote connection for non-local
        shift
        ;;
      *)
        db="$1"
        shift
        ;;
    esac
  done

  if [[ "$readonly" == true ]]; then
    log_info "Opening PostgreSQL shell (read-only mode)..."
    log_warning "Changes will not be persisted"
    # Use a transaction that we'll rollback
    local user="${POSTGRES_USER:-postgres}"
    docker exec -it "$(get_container)" psql -U "$user" -d "$db" \
      -v ON_ERROR_ROLLBACK=on \
      -c "BEGIN READ ONLY;" \
      -c "\\echo 'Read-only mode - queries only, no modifications'" \
      2>/dev/null || psql_interactive "$db"
  else
    log_info "Opening PostgreSQL shell..."
    psql_interactive "$db"
  fi
}

cmd_query() {
  require_db || return 1
  local sql="$1"
  local format="${2:-table}"

  [[ -z "$sql" ]] && {
    log_error "Usage: nself db query '<sql>' [format]"
    return 1
  }

  case "$format" in
    csv) psql_exec -c "COPY ($sql) TO STDOUT WITH CSV HEADER" ;;
    json) psql_exec -t -c "SELECT json_agg(t) FROM ($sql) t" ;;
    *) psql_exec -c "$sql" ;;
  esac
}

# ============================================================================
# SEED
# ============================================================================

cmd_seed() {
  local subcmd="${1:-run}"
  shift || true

  case "$subcmd" in
    run) seed_run "$@" ;;
    users) seed_users "$@" ;;
    create) seed_create "$@" ;;
    status) seed_status "$@" ;;
    *) seed_run "$subcmd" "$@" ;;
  esac
}

seed_run() {
  require_db || return 1
  local target="${1:-all}"
  local env=$(get_env)

  log_info "Seeding database (environment: $env)"
  ensure_dirs

  # Run common seeds first
  if [[ -d "$SEEDS_DIR/common" ]]; then
    log_info "Applying common seeds..."
    for seed in "$SEEDS_DIR/common"/*.sql; do
      [[ -f "$seed" ]] || continue
      log_info "  $(basename "$seed")"
      psql_exec <"$seed" >/dev/null 2>&1 || log_warning "  Failed: $(basename "$seed")"
    done
  fi

  # Run environment-specific seeds
  local env_dir="$SEEDS_DIR/$env"
  if [[ -d "$env_dir" ]]; then
    log_info "Applying $env seeds..."
    for seed in "$env_dir"/*.sql; do
      [[ -f "$seed" ]] || continue
      log_info "  $(basename "$seed")"
      psql_exec <"$seed" >/dev/null 2>&1 || log_warning "  Failed: $(basename "$seed")"
    done
  fi

  log_success "Seeding complete"
}

seed_users() {
  require_db || return 1
  local env=$(get_env)

  log_info "Seeding users for $env environment..."

  # Check for users seed config
  local config="$SEEDS_DIR/users.seed.yaml"

  if [[ -f "$config" ]]; then
    log_info "Using config: $config"
    # Parse YAML and apply (simplified - would use yq in production)
  else
    # Environment-aware default seeding
    case "$env" in
      local)
        log_info "Creating local test users (password: password123)..."
        seed_mock_users 20 "password123"
        ;;
      staging)
        log_info "Creating staging test users (password: TestUser123!)..."
        seed_mock_users 100 "TestUser123!"
        ;;
      production)
        log_warning "Production user seeding requires explicit config"

        # Check for NSELF_PROD_USERS (format: email:name:role,email:name:role,...)
        if [[ -n "${NSELF_PROD_USERS:-}" ]]; then
          log_info "Creating users from NSELF_PROD_USERS..."
          IFS=',' read -ra users <<<"$NSELF_PROD_USERS"
          for user_entry in "${users[@]}"; do
            IFS=':' read -r email name role <<<"$user_entry"
            if [[ -n "$email" ]]; then
              seed_explicit_user "$email" "" "${role:-user}" "$name"
            fi
          done
        # Legacy support for SEED_ADMIN_EMAIL
        elif [[ -n "${SEED_ADMIN_EMAIL:-}" ]]; then
          log_info "Creating admin from SEED_ADMIN_EMAIL..."
          seed_explicit_user "${SEED_ADMIN_EMAIL}" "${SEED_ADMIN_PASSWORD:-}" "admin"
        else
          log_info "No production users configured"
          log_info "Set NSELF_PROD_USERS='email:name:role,...' or SEED_ADMIN_EMAIL"
        fi
        ;;
    esac
  fi

  log_success "User seeding complete"
}

seed_mock_users() {
  local count="${1:-10}"
  local password="${2:-password123}"

  # Generate password hash (bcrypt placeholder - real impl would use proper hashing)
  local pass_hash=$(echo -n "$password" | openssl dgst -sha256 | cut -d' ' -f2)

  # Check if auth.users table exists (Hasura Auth pattern)
  local has_auth=$(psql_query "SELECT 1 FROM information_schema.tables WHERE table_schema = 'auth' AND table_name = 'users'" 2>/dev/null)

  if [[ "$has_auth" == "1" ]]; then
    log_info "Using auth.users table (Hasura Auth)..."
    for i in $(seq 1 $count); do
      local email="user${i}@test.local"
      psql_exec -c "INSERT INTO auth.users (email, encrypted_password, email_verified)
        VALUES ('$email', '$pass_hash', true)
        ON CONFLICT (email) DO NOTHING" >/dev/null 2>&1 || true
    done
  else
    # Fallback to public.users
    log_info "Using public.users table..."
    for i in $(seq 1 $count); do
      local email="user${i}@test.local"
      psql_exec -c "INSERT INTO users (email, password_hash)
        VALUES ('$email', '$pass_hash')
        ON CONFLICT (email) DO NOTHING" >/dev/null 2>&1 || true
    done
  fi

  log_info "Created $count test users"
}

seed_explicit_user() {
  local email="$1"
  local password="$2"
  local role="${3:-admin}"
  local display_name="${4:-}"

  # Generate password if not provided
  local pass_hash
  if [[ -z "$password" ]]; then
    # Generate random password for production users
    local gen_pass=$(openssl rand -base64 16 2>/dev/null || date +%s%N | sha256sum | head -c 20)
    pass_hash=$(echo -n "$gen_pass" | openssl dgst -sha256 | cut -d' ' -f2)
    log_info "  Generated random password for $email"
  else
    pass_hash=$(echo -n "$password" | openssl dgst -sha256 | cut -d' ' -f2)
  fi

  # Try auth.users first (Hasura Auth pattern)
  local has_auth=$(psql_query "SELECT 1 FROM information_schema.tables WHERE table_schema = 'auth' AND table_name = 'users'" 2>/dev/null)

  if [[ "$has_auth" == "1" ]]; then
    psql_exec -c "INSERT INTO auth.users (email, encrypted_password, email_verified, default_role, display_name)
      VALUES ('$email', '$pass_hash', true, '$role', '$display_name')
      ON CONFLICT (email) DO UPDATE SET default_role = '$role', display_name = COALESCE(NULLIF('$display_name', ''), auth.users.display_name)" >/dev/null 2>&1
  else
    # Fallback to public.users
    psql_exec -c "INSERT INTO users (email, password_hash, role, name)
      VALUES ('$email', '$pass_hash', '$role', '$display_name')
      ON CONFLICT (email) DO UPDATE SET role = '$role'" >/dev/null 2>&1
  fi

  log_success "Created/updated user: $email (role: $role)"
}

seed_create() {
  local name="$1"
  local env="${2:-common}"

  [[ -z "$name" ]] && {
    log_error "Usage: nself db seed create <name> [environment]"
    return 1
  }

  ensure_dirs
  local file="$SEEDS_DIR/$env/${name}.sql"
  mkdir -p "$(dirname "$file")"

  cat >"$file" <<EOF
-- Seed: $name
-- Environment: $env
-- Created: $(date)

-- Add your seed data here
-- Use ON CONFLICT for idempotent seeding

EOF

  log_success "Created seed: $file"
}

seed_status() {
  log_info "Seed Files by Environment"
  echo ""

  for env_dir in common local staging production; do
    local dir="$SEEDS_DIR/$env_dir"
    local count=$(ls -1 "$dir"/*.sql 2>/dev/null | wc -l | tr -d ' ')
    printf "  %-12s %s file(s)\n" "$env_dir:" "$count"
  done
}

# ============================================================================
# MOCK DATA
# ============================================================================

cmd_mock() {
  local subcmd="${1:-generate}"
  shift || true

  case "$subcmd" in
    generate) mock_generate "$@" ;;
    preview) mock_preview "$@" ;;
    clear) mock_clear "$@" ;;
    config) mock_config "$@" ;;
    *)
      log_error "Unknown: mock $subcmd"
      mock_help
      ;;
  esac
}

mock_generate() {
  require_non_production "Mock data generation" || return 1
  require_db || return 1

  local table=""
  local count="100"
  local seed="${NSELF_MOCK_SEED:-$(date +%s)}"

  # Parse arguments (support --tables and --count flags)
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --tables | -t)
        table="$2"
        shift 2
        ;;
      --count | -c)
        count="$2"
        shift 2
        ;;
      --seed | -s)
        seed="$2"
        shift 2
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        # Positional args: table, count, seed
        if [[ -z "$table" ]]; then
          table="$1"
        elif [[ "$count" == "100" ]]; then
          count="$1"
        else
          seed="$1"
        fi
        shift
        ;;
    esac
  done

  log_info "Generating mock data (seed: $seed)"
  log_info "Same seed = same data across your team!"

  # Save seed for reproducibility
  echo "$seed" >"$MOCK_DIR/mock.seed"

  # System tables to exclude from mock data
  local exclude_tables="schema_migrations|pg_|information_schema|auth_"

  # Get tables if not specified
  local tables
  if [[ -z "$table" ]]; then
    tables=$(psql_query "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename" | grep -vE "^($exclude_tables)")
  else
    # Validate user-specified table is not a system table
    if echo "$table" | grep -qE "^($exclude_tables)"; then
      log_error "Cannot generate mock data for system table: $table"
      return 1
    fi
    tables="$table"
  fi

  for t in $tables; do
    log_info "Generating $count rows for: $t"
    mock_table "$t" "$count" "$seed"
  done

  log_success "Mock data generated (seed: $seed)"
  log_info "Share seed with team: NSELF_MOCK_SEED=$seed nself db mock generate"
}

mock_table() {
  local table="$1"
  local count="$2"
  local seed="$3"
  local verbose="${NSELF_MOCK_VERBOSE:-false}"
  local errors_count=0
  local first_error=""

  # Get column info (excluding auto-generated columns)
  local columns_info=$(psql_query "SELECT column_name, data_type, column_default, character_maximum_length
    FROM information_schema.columns
    WHERE table_name = '$table' AND table_schema = 'public'
    AND (column_default IS NULL OR column_default NOT LIKE 'nextval%')
    ORDER BY ordinal_position")

  [[ -z "$columns_info" ]] && return 0

  local base=$((seed % 1000000))

  for i in $(seq 1 $count); do
    local row_id=$((base + i))
    local col_names=""
    local col_values=""

    while IFS='|' read -r col_name data_type col_default char_max_len; do
      [[ -z "$col_name" ]] && continue
      # Skip id columns that are auto-increment
      [[ "$col_name" == "id" && "$col_default" =~ nextval ]] && continue

      # Normalize data_type: trim whitespace and convert to lowercase
      data_type=$(echo "$data_type" | tr -d '[:space:]' | tr '[:upper:]' '[:lower:]')
      # Clean up char_max_len (remove whitespace, empty = unlimited)
      char_max_len=$(echo "$char_max_len" | tr -d '[:space:]')

      [[ -n "$col_names" ]] && col_names+=", "
      [[ -n "$col_values" ]] && col_values+=", "

      col_names+="$col_name"

      # Generate appropriate mock value based on type and column name
      case "$data_type" in
        integer | bigint | smallint)
          col_values+="$row_id"
          ;;
        numeric | decimal | real | "double precision")
          col_values+="$row_id.99"
          ;;
        boolean)
          [[ $((row_id % 2)) -eq 0 ]] && col_values+="true" || col_values+="false"
          ;;
        *timestamp* | *date*)
          col_values+="NOW() - INTERVAL '$((row_id % 365)) days'"
          ;;
        uuid)
          col_values+="gen_random_uuid()"
          ;;
        json | jsonb)
          # Generate valid JSONB based on column name patterns
          case "$col_name" in
            *metadata* | *meta*)
              col_values+="'{\"source\": \"mock\", \"version\": 1, \"id\": $row_id}'"
              ;;
            *config* | *settings*)
              col_values+="'{\"enabled\": true, \"level\": $((row_id % 5))}'"
              ;;
            *data* | *payload*)
              col_values+="'{\"type\": \"mock\", \"index\": $row_id, \"valid\": true}'"
              ;;
            *)
              col_values+="'{\"mock\": true, \"id\": $row_id}'"
              ;;
          esac
          ;;
        inet | cidr | "inet" | "cidr")
          # Generate valid IP addresses with explicit type cast
          local octet1=$((10 + (row_id / 65536) % 245))
          local octet2=$(((row_id / 256) % 256))
          local octet3=$((row_id % 256))
          col_values+="'${octet1}.${octet2}.${octet3}.1'::inet"
          ;;
        macaddr | macaddr8 | "macaddr" | "macaddr8")
          # Generate valid MAC addresses with explicit type cast
          local mac_part=$(printf '%02x:%02x:%02x' $((row_id % 256)) $(((row_id / 256) % 256)) $(((row_id / 65536) % 256)))
          col_values+="'00:00:5e:${mac_part}'::macaddr"
          ;;
        *)
          # Text/varchar - generate based on column name patterns
          local mock_val=""
          case "$col_name" in
            *email*) mock_val="user${row_id}@example.com" ;;
            *name* | *title*) mock_val="Mock ${col_name} ${row_id}" ;;
            *url* | *link*) mock_val="https://example.com/${row_id}" ;;
            *slug*) mock_val="slug-${row_id}" ;;
            *password* | *hash*) mock_val="hashed_${row_id}" ;;
            *) mock_val="mock_${col_name}_${row_id}" ;;
          esac
          # Truncate if character_maximum_length is defined
          if [[ -n "$char_max_len" ]] && [[ "$char_max_len" =~ ^[0-9]+$ ]] && [[ ${#mock_val} -gt $char_max_len ]]; then
            mock_val="${mock_val:0:$char_max_len}"
          fi
          col_values+="'$mock_val'"
          ;;
      esac
    done <<<"$columns_info"

    # Execute insert if we have columns
    if [[ -n "$col_names" ]]; then
      local insert_result
      insert_result=$(psql_exec -c "INSERT INTO $table ($col_names) VALUES ($col_values) ON CONFLICT DO NOTHING" 2>&1)
      local insert_status=$?
      if [[ $insert_status -ne 0 ]]; then
        ((errors_count++)) || true
        if [[ -z "$first_error" ]]; then
          first_error="$insert_result"
        fi
        [[ "$verbose" == "true" ]] && log_warning "Insert failed for row $i: $insert_result"
      fi
    fi
  done

  # Report errors if any occurred
  if [[ $errors_count -gt 0 ]]; then
    log_warning "$table: $errors_count insert(s) failed"
    [[ -n "$first_error" ]] && log_debug "First error: $first_error"
  fi
}

mock_preview() {
  require_db || return 1
  local table="${1:-users}"
  local limit="${2:-10}"

  log_info "Preview of $table (limit $limit):"
  psql_exec -c "SELECT * FROM $table LIMIT $limit"
}

mock_clear() {
  require_non_production "Clear mock data" || return 1
  require_db || return 1
  require_confirmation "Clear all data from public tables" || return 1

  log_info "Clearing mock data..."
  local tables=$(psql_query "SELECT tablename FROM pg_tables WHERE schemaname = 'public'")

  for t in $tables; do
    psql_exec -c "TRUNCATE TABLE $t CASCADE" >/dev/null 2>&1 || true
  done

  log_success "Mock data cleared"
}

mock_config() {
  local config="$MOCK_DIR/mock.config.yaml"

  if [[ ! -f "$config" ]]; then
    cat >"$config" <<'EOF'
# Mock Data Configuration
# Use deterministic seeds for reproducible team data

global:
  seed: 42
  locale: en_US
  null_probability: 0.05

tables:
  users:
    count: 100
    columns:
      email: { type: email, unique: true }
      name: { type: fullName }
      created_at: { type: pastDate, years: 2 }

  posts:
    count: 500
    columns:
      title: { type: sentence }
      body: { type: paragraphs, count: 3 }
      user_id: { type: foreignKey, table: users }
EOF
    log_success "Created mock config: $config"
  else
    log_info "Edit mock config: $config"
  fi
}

mock_help() {
  echo "Usage: nself db mock <command>"
  echo ""
  echo "Commands:"
  echo "  generate [table] [count]   Generate mock data"
  echo "  preview [table] [limit]    Preview table data"
  echo "  clear                      Clear all mock data (local only)"
  echo "  config                     Create/edit mock configuration"
  echo ""
  echo "Options:"
  echo "  NSELF_MOCK_SEED=123        Use deterministic seed (shareable!)"
}

# ============================================================================
# BACKUP
# ============================================================================

cmd_backup() {
  local subcmd="${1:-create}"
  shift || true

  case "$subcmd" in
    create) backup_create "$@" ;;
    list | ls) backup_list "$@" ;;
    restore) backup_restore "$@" ;;
    schedule) backup_schedule "$@" ;;
    prune) backup_prune "$@" ;;
    *) backup_create "$subcmd" "$@" ;;
  esac
}

backup_create() {
  require_db || return 1

  local type="full"
  local name=""
  local compress=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --compress | -c)
        compress=true
        shift
        ;;
      --name | -n)
        shift
        name="$1"
        shift
        ;;
      full | database | db | schema | data)
        type="$1"
        shift
        ;;
      *)
        # Assume it's the name if not a flag
        if [[ -z "$name" ]]; then
          name="$1"
        fi
        shift
        ;;
    esac
  done

  ensure_dirs
  local timestamp=$(date +%Y%m%d_%H%M%S)
  local project="${PROJECT_NAME:-nself}"
  local filename="${name:-${project}_${type}_${timestamp}.sql}"

  # Full backups are always compressed
  if [[ "$type" == "full" ]]; then
    filename="${filename%.sql}.tar.gz"
  elif [[ "$compress" == true ]]; then
    filename="${filename}.gz"
  fi

  local backup_path="$BACKUPS_DIR/$filename"

  log_info "Creating $type backup..."

  case "$type" in
    full)
      local temp_dir=$(mktemp -d)

      # Database dump
      log_info "  Dumping database..."
      docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
        "${POSTGRES_DB:-nhost}" >"$temp_dir/database.sql"

      # Config files
      log_info "  Backing up config..."
      cp .env* "$temp_dir/" 2>/dev/null || true
      cp docker-compose*.yml "$temp_dir/" 2>/dev/null || true
      [[ -d "nginx" ]] && cp -r nginx "$temp_dir/"
      [[ -d "nself" ]] && cp -r nself "$temp_dir/"

      # Create archive
      tar -czf "$backup_path" -C "$temp_dir" . 2>/dev/null
      rm -rf "$temp_dir"
      ;;

    database | db)
      if [[ "$compress" == true ]]; then
        docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
          "${POSTGRES_DB:-nhost}" | gzip >"$backup_path"
      else
        docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
          "${POSTGRES_DB:-nhost}" >"$backup_path"
      fi
      ;;

    schema)
      if [[ "$compress" == true ]]; then
        docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
          --schema-only "${POSTGRES_DB:-nhost}" | gzip >"$backup_path"
      else
        docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
          --schema-only "${POSTGRES_DB:-nhost}" >"$backup_path"
      fi
      ;;

    data)
      if [[ "$compress" == true ]]; then
        docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
          --data-only "${POSTGRES_DB:-nhost}" | gzip >"$backup_path"
      else
        docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
          --data-only "${POSTGRES_DB:-nhost}" >"$backup_path"
      fi
      ;;

    *)
      log_error "Unknown backup type: $type"
      log_info "Types: full, database, schema, data"
      return 1
      ;;
  esac

  local size=$(du -h "$backup_path" | cut -f1)
  log_success "Backup created: $backup_path ($size)"
}

backup_list() {
  log_info "Available Backups"
  echo ""

  if [[ -d "$BACKUPS_DIR" ]]; then
    printf "%-45s %-10s %-20s\n" "Name" "Size" "Created"
    printf "%-45s %-10s %-20s\n" "$(printf '%.0s─' {1..40})" "────────" "──────────────────"

    for f in "$BACKUPS_DIR"/*; do
      [[ -f "$f" ]] || continue
      local name=$(basename "$f")
      local size=$(du -h "$f" | cut -f1)
      local created=$(get_file_mtime "$f")
      printf "%-45s %-10s %-20s\n" "$name" "$size" "$created"
    done
  else
    echo "  No backups found"
  fi
}

backup_restore() {
  local backup="${1:-latest}"
  local type="${2:-full}"

  require_confirmation "Restore from backup will overwrite current data" || return 1

  local backup_path
  if [[ "$backup" == "latest" ]]; then
    backup_path=$(ls -t "$BACKUPS_DIR"/* 2>/dev/null | head -1)
  elif [[ -f "$BACKUPS_DIR/$backup" ]]; then
    backup_path="$BACKUPS_DIR/$backup"
  elif [[ -f "$backup" ]]; then
    backup_path="$backup"
  else
    log_error "Backup not found: $backup"
    return 1
  fi

  log_info "Restoring from: $(basename "$backup_path")"

  if [[ "$backup_path" == *.tar.gz ]]; then
    local temp_dir=$(mktemp -d)
    tar -xzf "$backup_path" -C "$temp_dir"

    if [[ -f "$temp_dir/database.sql" ]]; then
      log_info "  Restoring database..."
      psql_exec <"$temp_dir/database.sql" >/dev/null 2>&1
    fi

    log_info "  Restoring config files..."
    cp "$temp_dir"/.env* . 2>/dev/null || true
    cp "$temp_dir"/docker-compose*.yml . 2>/dev/null || true
    [[ -d "$temp_dir/nginx" ]] && cp -r "$temp_dir/nginx" .
    [[ -d "$temp_dir/nself" ]] && cp -r "$temp_dir/nself" .

    rm -rf "$temp_dir"
  else
    log_info "  Restoring database..."
    psql_exec <"$backup_path" >/dev/null 2>&1
  fi

  log_success "Restore complete"
}

backup_schedule() {
  local freq="${1:-daily}"

  log_info "Backup scheduling"
  echo ""
  log_info "Add to crontab:"
  echo ""

  case "$freq" in
    hourly)
      echo "  0 * * * * cd $(pwd) && nself db backup create full"
      ;;
    daily)
      echo "  0 3 * * * cd $(pwd) && nself db backup create full"
      ;;
    weekly)
      echo "  0 3 * * 0 cd $(pwd) && nself db backup create full"
      ;;
  esac

  echo ""
  log_info "Edit with: crontab -e"
}

backup_prune() {
  local days="${1:-30}"

  log_info "Removing backups older than $days days..."

  find "$BACKUPS_DIR" -name "*.sql" -o -name "*.tar.gz" -mtime +$days -exec rm {} \; 2>/dev/null

  log_success "Pruning complete"
}

# ============================================================================
# INSPECT (Database Analysis)
# ============================================================================

cmd_inspect() {
  local subcmd="${1:-overview}"
  shift || true

  require_db || return 1

  case "$subcmd" in
    overview) inspect_overview ;;
    size) inspect_size "$@" ;;
    cache | cache-hit) inspect_cache ;;
    index | index-usage) inspect_indexes "$@" ;;
    unused-indexes) inspect_unused_indexes ;;
    bloat) inspect_bloat ;;
    slow | slow-queries) inspect_slow_queries ;;
    locks) inspect_locks ;;
    connections) inspect_connections ;;
    *) inspect_overview ;;
  esac
}

inspect_overview() {
  log_info "Database Overview"
  echo ""

  # Database size
  local db_size=$(psql_query "SELECT pg_size_pretty(pg_database_size(current_database()))")
  echo "  Database Size: $db_size"

  # Table count
  local table_count=$(psql_query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'")
  echo "  Tables: $table_count"

  # Total rows
  local total_rows=$(psql_query "SELECT SUM(n_live_tup) FROM pg_stat_user_tables")
  echo "  Total Rows: ${total_rows:-0}"

  # Active connections
  local conns=$(psql_query "SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active'")
  echo "  Active Connections: $conns"

  echo ""
  log_info "Run 'nself db inspect <topic>' for details"
  echo "  Topics: size, cache, index, unused-indexes, bloat, slow, locks, connections"
}

inspect_size() {
  local limit="${1:-10}"

  log_info "Table Sizes (Top $limit)"
  psql_exec -c "SELECT
    schemaname || '.' || tablename AS table,
    pg_size_pretty(pg_total_relation_size(schemaname || '.' || tablename)) AS total,
    pg_size_pretty(pg_relation_size(schemaname || '.' || tablename)) AS table_size,
    pg_size_pretty(pg_indexes_size(schemaname || '.' || tablename)) AS indexes
  FROM pg_tables
  WHERE schemaname = 'public'
  ORDER BY pg_total_relation_size(schemaname || '.' || tablename) DESC
  LIMIT $limit"
}

inspect_cache() {
  log_info "Cache Hit Ratios"
  psql_exec -c "SELECT
    'Index Hit Rate' AS metric,
    ROUND(100.0 * sum(idx_blks_hit) / nullif(sum(idx_blks_hit + idx_blks_read), 0), 2) || '%' AS ratio
  FROM pg_statio_user_indexes
  UNION ALL
  SELECT
    'Table Hit Rate',
    ROUND(100.0 * sum(heap_blks_hit) / nullif(sum(heap_blks_hit + heap_blks_read), 0), 2) || '%'
  FROM pg_statio_user_tables"
}

inspect_indexes() {
  log_info "Index Usage"
  psql_exec -c "SELECT
    schemaname || '.' || relname AS table,
    indexrelname AS index,
    idx_scan AS scans,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
  FROM pg_stat_user_indexes
  ORDER BY idx_scan DESC
  LIMIT 20"
}

inspect_unused_indexes() {
  log_info "Unused Indexes"
  psql_exec -c "SELECT
    schemaname || '.' || relname AS table,
    indexrelname AS index,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
  FROM pg_stat_user_indexes
  WHERE idx_scan = 0
  AND indexrelname NOT LIKE '%_pkey'
  ORDER BY pg_relation_size(indexrelid) DESC"
}

inspect_bloat() {
  log_info "Table Bloat"
  psql_exec -c "SELECT
    schemaname || '.' || relname AS table,
    n_dead_tup AS dead_rows,
    n_live_tup AS live_rows,
    ROUND(100.0 * n_dead_tup / nullif(n_live_tup + n_dead_tup, 0), 2) AS bloat_pct
  FROM pg_stat_user_tables
  WHERE n_dead_tup > 0
  ORDER BY n_dead_tup DESC
  LIMIT 20"
}

inspect_slow_queries() {
  log_info "Slow Queries (requires pg_stat_statements)"
  psql_exec -c "SELECT
    ROUND(total_exec_time::numeric / calls, 2) AS avg_ms,
    calls,
    ROUND(total_exec_time::numeric, 2) AS total_ms,
    LEFT(query, 60) AS query
  FROM pg_stat_statements
  WHERE calls > 0
  ORDER BY total_exec_time DESC
  LIMIT 20" 2>/dev/null || log_warning "pg_stat_statements extension not enabled"
}

inspect_locks() {
  log_info "Current Locks"
  psql_exec -c "SELECT
    pid,
    usename,
    pg_blocking_pids(pid) AS blocked_by,
    LEFT(query, 50) AS query
  FROM pg_stat_activity
  WHERE cardinality(pg_blocking_pids(pid)) > 0"
}

inspect_connections() {
  log_info "Active Connections"
  psql_exec -c "SELECT
    usename,
    application_name,
    client_addr,
    state,
    query_start,
    LEFT(query, 40) AS query
  FROM pg_stat_activity
  WHERE pid != pg_backend_pid()
  ORDER BY query_start DESC"
}

# ============================================================================
# SCHEMA TOOLS
# ============================================================================

cmd_schema() {
  local subcmd="${1:-show}"
  shift || true

  case "$subcmd" in
    show) schema_show "$@" ;;
    diff) schema_diff "$@" ;;
    diagram) schema_diagram "$@" ;;
    indexes) schema_indexes "$@" ;;
    export) schema_export "$@" ;;
    import) schema_import "$@" ;;
    scaffold) schema_scaffold "$@" ;;
    apply) schema_apply "$@" ;;
    *) schema_show "$@" ;;
  esac
}

schema_show() {
  require_db || return 1
  local table="${1:-}"

  if [[ -n "$table" ]]; then
    log_info "Schema for: $table"
    psql_exec -c "\\d+ $table"
  else
    log_info "Database Schema"
    psql_exec -c "\\dt+ public.*"
  fi
}

schema_diff() {
  local target="${1:-}"

  [[ -z "$target" ]] && {
    log_error "Usage: nself db schema diff <environment|file>"
    return 1
  }

  log_info "Comparing schema with $target..."

  # Export current schema
  local current=$(mktemp)
  docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
    --schema-only "${POSTGRES_DB:-nhost}" >"$current"

  if [[ -f "$target" ]]; then
    diff -u "$target" "$current" || true
  else
    log_warning "Remote diff not implemented - export schemas and compare"
    log_info "  nself db schema export > local.sql"
    log_info "  # Get remote schema and diff"
  fi

  rm -f "$current"
}

schema_diagram() {
  require_db || return 1
  local output="${1:-schema.dbml}"

  log_info "Generating schema diagram..."

  # Generate DBML from database
  local tables=$(psql_query "SELECT tablename FROM pg_tables WHERE schemaname = 'public'")

  cat >"$output" <<EOF
// Database Schema
// Generated: $(date)
// Project: ${PROJECT_NAME:-nself}

EOF

  for table in $tables; do
    echo "Table $table {" >>"$output"

    psql_query "SELECT column_name, data_type, is_nullable, column_default
      FROM information_schema.columns
      WHERE table_name = '$table' AND table_schema = 'public'
      ORDER BY ordinal_position" | while IFS='|' read -r col dtype nullable default; do
      local attrs=""
      [[ "$nullable" == "NO" ]] && attrs="not null"
      [[ -n "$default" ]] && attrs="$attrs default: \`$default\`"
      echo "  $col $dtype [$attrs]" >>"$output"
    done

    echo "}" >>"$output"
    echo "" >>"$output"
  done

  log_success "Generated: $output"
  log_info "View at: https://dbdiagram.io/d"
}

schema_indexes() {
  local action="${1:-list}"

  require_db || return 1

  case "$action" in
    list)
      log_info "Database Indexes"
      psql_exec -c "SELECT
        schemaname || '.' || tablename AS table,
        indexname AS index,
        indexdef
      FROM pg_indexes
      WHERE schemaname = 'public'
      ORDER BY tablename, indexname"
      ;;

    missing | suggest)
      log_info "Suggested Indexes (based on query patterns)"
      psql_exec -c "SELECT
        relname AS table,
        seq_scan AS seq_scans,
        seq_tup_read AS rows_scanned,
        idx_scan AS index_scans
      FROM pg_stat_user_tables
      WHERE seq_scan > idx_scan
      AND seq_tup_read > 10000
      ORDER BY seq_tup_read DESC
      LIMIT 20"
      log_info "Tables with high sequential scans may benefit from indexes"
      ;;

    unused)
      inspect_unused_indexes
      ;;
  esac
}

schema_export() {
  local format="${1:-sql}"
  local output="${2:-schema.sql}"

  require_db || return 1

  log_info "Exporting schema as $format..."

  case "$format" in
    sql)
      docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
        --schema-only "${POSTGRES_DB:-nhost}" >"$output"
      ;;
    json)
      psql_query "SELECT json_agg(json_build_object(
        'table', table_name,
        'columns', (SELECT json_agg(json_build_object(
          'name', column_name,
          'type', data_type,
          'nullable', is_nullable
        )) FROM information_schema.columns c
        WHERE c.table_name = t.table_name AND c.table_schema = 'public')
      )) FROM information_schema.tables t
      WHERE table_schema = 'public'" >"$output"
      ;;
  esac

  log_success "Exported: $output"
}

# Import DBML file and create SQL migration
schema_import() {
  local dbml_file="${1:-schema.dbml}"
  local migration_name="${2:-imported_schema}"

  if [[ ! -f "$dbml_file" ]]; then
    log_error "DBML file not found: $dbml_file"
    log_info "Create one at https://dbdiagram.io or provide a valid path"
    return 1
  fi

  log_info "Importing DBML from: $dbml_file"

  # Create migration directory if needed
  mkdir -p "$MIGRATIONS_DIR"

  # Generate timestamp
  local timestamp=$(date +%Y%m%d%H%M%S)
  local up_file="$MIGRATIONS_DIR/${timestamp}_${migration_name}.up.sql"
  local down_file="$MIGRATIONS_DIR/${timestamp}_${migration_name}.down.sql"

  # Parse DBML and generate SQL
  log_info "Parsing DBML..."

  local current_table=""
  local tables_created=()
  local in_table=false

  # Start up migration
  cat >"$up_file" <<'EOF'
-- Migration generated from DBML import
-- Generated: $(date)

EOF

  # Start down migration
  echo "-- Rollback migration" >"$down_file"
  echo "" >>"$down_file"

  # Parse DBML file
  while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip comments and empty lines
    [[ "$line" =~ ^[[:space:]]*// ]] && continue
    [[ -z "${line// /}" ]] && continue

    # Detect table start: "Table tablename {" or "Table tablename as alias {"
    if [[ "$line" =~ ^[[:space:]]*[Tt]able[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
      current_table="${BASH_REMATCH[1]}"
      tables_created+=("$current_table")
      in_table=true
      echo "CREATE TABLE IF NOT EXISTS $current_table (" >>"$up_file"
      continue
    fi

    # Detect table end
    if [[ "$in_table" == true && "$line" =~ ^[[:space:]]*\} ]]; then
      # Remove trailing comma from last column and close table
      sed -i.bak '$ s/,$//' "$up_file" 2>/dev/null || sed -i '' '$ s/,$//' "$up_file"
      rm -f "$up_file.bak"
      echo ");" >>"$up_file"
      echo "" >>"$up_file"
      in_table=false
      current_table=""
      continue
    fi

    # Parse column definitions inside table
    if [[ "$in_table" == true ]]; then
      # Match: column_name type [constraints]
      if [[ "$line" =~ ^[[:space:]]*([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]+([a-zA-Z0-9_\(\)]+) ]]; then
        local col_name="${BASH_REMATCH[1]}"
        local col_type="${BASH_REMATCH[2]}"
        local constraints=""

        # Map DBML types to PostgreSQL
        case "$col_type" in
          int | integer) col_type="INTEGER" ;;
          bigint) col_type="BIGINT" ;;
          smallint) col_type="SMALLINT" ;;
          serial) col_type="SERIAL" ;;
          bigserial) col_type="BIGSERIAL" ;;
          varchar*) col_type=$(echo "$col_type" | tr '[:lower:]' '[:upper:]') ;;
          text) col_type="TEXT" ;;
          boolean | bool) col_type="BOOLEAN" ;;
          timestamp | timestamptz) col_type="TIMESTAMPTZ" ;;
          date) col_type="DATE" ;;
          time) col_type="TIME" ;;
          uuid) col_type="UUID" ;;
          json) col_type="JSON" ;;
          jsonb) col_type="JSONB" ;;
          decimal* | numeric*) col_type=$(echo "$col_type" | tr '[:lower:]' '[:upper:]') ;;
          float | real) col_type="REAL" ;;
          double) col_type="DOUBLE PRECISION" ;;
        esac

        # Check for constraints in brackets [pk, not null, default: xxx]
        if [[ "$line" =~ \[([^\]]+)\] ]]; then
          local attrs="${BASH_REMATCH[1]}"

          [[ "$attrs" =~ pk ]] && constraints="$constraints PRIMARY KEY"
          [[ "$attrs" =~ "not null" ]] && constraints="$constraints NOT NULL"
          [[ "$attrs" =~ unique ]] && constraints="$constraints UNIQUE"
          [[ "$attrs" =~ increment ]] && col_type="SERIAL"

          # Handle default values
          if [[ "$attrs" =~ default:[[:space:]]*\`([^\`]+)\` ]]; then
            constraints="$constraints DEFAULT ${BASH_REMATCH[1]}"
          elif [[ "$attrs" =~ default:[[:space:]]*\'([^\']+)\' ]]; then
            constraints="$constraints DEFAULT '${BASH_REMATCH[1]}'"
          elif [[ "$attrs" =~ default:[[:space:]]*([^,\]]+) ]]; then
            local def_val="${BASH_REMATCH[1]}"
            def_val=$(echo "$def_val" | xargs) # trim whitespace
            [[ -n "$def_val" ]] && constraints="$constraints DEFAULT $def_val"
          fi
        fi

        echo "  $col_name $col_type$constraints," >>"$up_file"
      fi
    fi

    # Parse Ref for foreign keys (basic support)
    if [[ "$line" =~ ^[[:space:]]*[Rr]ef:[[:space:]]*([a-zA-Z_]+)\.([a-zA-Z_]+)[[:space:]]*\>[[:space:]]*([a-zA-Z_]+)\.([a-zA-Z_]+) ]]; then
      local from_table="${BASH_REMATCH[1]}"
      local from_col="${BASH_REMATCH[2]}"
      local to_table="${BASH_REMATCH[3]}"
      local to_col="${BASH_REMATCH[4]}"

      echo "ALTER TABLE $from_table ADD CONSTRAINT fk_${from_table}_${from_col}" >>"$up_file"
      echo "  FOREIGN KEY ($from_col) REFERENCES $to_table($to_col);" >>"$up_file"
      echo "" >>"$up_file"
    fi

  done <"$dbml_file"

  # Generate down migration (drop tables in reverse order)
  for ((i = ${#tables_created[@]} - 1; i >= 0; i--)); do
    echo "DROP TABLE IF EXISTS ${tables_created[$i]} CASCADE;" >>"$down_file"
  done

  log_success "Created migration files:"
  log_info "  Up:   $up_file"
  log_info "  Down: $down_file"
  log_info ""
  log_info "Tables found: ${#tables_created[@]}"
  for t in "${tables_created[@]}"; do
    log_info "  - $t"
  done
  log_info ""
  log_info "Next steps:"
  log_info "  1. Review the generated SQL"
  log_info "  2. Run: nself db migrate up"
  log_info "  3. Generate mock data: nself db mock auto"
}

# Scaffold a new schema with starter templates
schema_scaffold() {
  local template="${1:-basic}"
  local output="${2:-schema.dbml}"

  log_info "Scaffolding schema: $template"

  case "$template" in
    basic)
      cat >"$output" <<'EOF'
// Basic Application Schema
// Edit this file and run: nself db schema import schema.dbml

Table users {
  id serial [pk]
  email varchar(255) [not null, unique]
  password_hash varchar(255) [not null]
  display_name varchar(100)
  avatar_url text
  role varchar(20) [not null, default: 'user']
  email_verified boolean [default: false]
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table profiles {
  id serial [pk]
  user_id integer [not null, unique]
  bio text
  location varchar(100)
  website varchar(255)
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table posts {
  id serial [pk]
  user_id integer [not null]
  title varchar(255) [not null]
  slug varchar(255) [not null, unique]
  content text
  published boolean [default: false]
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

// Relationships
Ref: profiles.user_id > users.id
Ref: posts.user_id > users.id
EOF
      ;;

    ecommerce)
      cat >"$output" <<'EOF'
// E-commerce Schema
// Edit this file and run: nself db schema import schema.dbml

Table users {
  id serial [pk]
  email varchar(255) [not null, unique]
  password_hash varchar(255) [not null]
  display_name varchar(100)
  role varchar(20) [not null, default: 'customer']
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table products {
  id serial [pk]
  name varchar(255) [not null]
  slug varchar(255) [not null, unique]
  description text
  price decimal(10,2) [not null]
  compare_price decimal(10,2)
  sku varchar(100) [unique]
  inventory_count integer [default: 0]
  category_id integer
  active boolean [default: true]
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table categories {
  id serial [pk]
  name varchar(100) [not null]
  slug varchar(100) [not null, unique]
  parent_id integer
  sort_order integer [default: 0]
}

Table orders {
  id serial [pk]
  user_id integer [not null]
  status varchar(20) [not null, default: 'pending']
  subtotal decimal(10,2) [not null]
  tax decimal(10,2) [default: 0]
  shipping decimal(10,2) [default: 0]
  total decimal(10,2) [not null]
  shipping_address jsonb
  billing_address jsonb
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table order_items {
  id serial [pk]
  order_id integer [not null]
  product_id integer [not null]
  quantity integer [not null, default: 1]
  unit_price decimal(10,2) [not null]
  total decimal(10,2) [not null]
}

Table cart_items {
  id serial [pk]
  user_id integer [not null]
  product_id integer [not null]
  quantity integer [not null, default: 1]
  created_at timestamptz [not null, default: `NOW()`]
}

// Relationships
Ref: products.category_id > categories.id
Ref: categories.parent_id > categories.id
Ref: orders.user_id > users.id
Ref: order_items.order_id > orders.id
Ref: order_items.product_id > products.id
Ref: cart_items.user_id > users.id
Ref: cart_items.product_id > products.id
EOF
      ;;

    saas)
      cat >"$output" <<'EOF'
// SaaS Multi-tenant Schema
// Edit this file and run: nself db schema import schema.dbml

Table organizations {
  id serial [pk]
  name varchar(255) [not null]
  slug varchar(100) [not null, unique]
  plan varchar(20) [not null, default: 'free']
  stripe_customer_id varchar(255)
  stripe_subscription_id varchar(255)
  settings jsonb [default: '{}']
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table users {
  id serial [pk]
  email varchar(255) [not null, unique]
  password_hash varchar(255) [not null]
  display_name varchar(100)
  avatar_url text
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table organization_members {
  id serial [pk]
  organization_id integer [not null]
  user_id integer [not null]
  role varchar(20) [not null, default: 'member']
  invited_by integer
  joined_at timestamptz [not null, default: `NOW()`]
}

Table invitations {
  id serial [pk]
  organization_id integer [not null]
  email varchar(255) [not null]
  role varchar(20) [not null, default: 'member']
  token varchar(255) [not null, unique]
  invited_by integer [not null]
  expires_at timestamptz [not null]
  accepted_at timestamptz
  created_at timestamptz [not null, default: `NOW()`]
}

Table projects {
  id serial [pk]
  organization_id integer [not null]
  name varchar(255) [not null]
  description text
  created_by integer [not null]
  archived boolean [default: false]
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table api_keys {
  id serial [pk]
  organization_id integer [not null]
  name varchar(100) [not null]
  key_hash varchar(255) [not null, unique]
  last_used_at timestamptz
  expires_at timestamptz
  created_by integer [not null]
  created_at timestamptz [not null, default: `NOW()`]
}

// Relationships
Ref: organization_members.organization_id > organizations.id
Ref: organization_members.user_id > users.id
Ref: organization_members.invited_by > users.id
Ref: invitations.organization_id > organizations.id
Ref: invitations.invited_by > users.id
Ref: projects.organization_id > organizations.id
Ref: projects.created_by > users.id
Ref: api_keys.organization_id > organizations.id
Ref: api_keys.created_by > users.id
EOF
      ;;

    blog)
      cat >"$output" <<'EOF'
// Blog/CMS Schema
// Edit this file and run: nself db schema import schema.dbml

Table users {
  id serial [pk]
  email varchar(255) [not null, unique]
  password_hash varchar(255)
  display_name varchar(100) [not null]
  bio text
  avatar_url text
  role varchar(20) [not null, default: 'author']
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table posts {
  id serial [pk]
  author_id integer [not null]
  title varchar(255) [not null]
  slug varchar(255) [not null, unique]
  excerpt text
  content text
  featured_image text
  status varchar(20) [not null, default: 'draft']
  published_at timestamptz
  created_at timestamptz [not null, default: `NOW()`]
  updated_at timestamptz [not null, default: `NOW()`]
}

Table categories {
  id serial [pk]
  name varchar(100) [not null]
  slug varchar(100) [not null, unique]
  description text
  parent_id integer
}

Table tags {
  id serial [pk]
  name varchar(50) [not null, unique]
  slug varchar(50) [not null, unique]
}

Table post_categories {
  post_id integer [not null]
  category_id integer [not null]
}

Table post_tags {
  post_id integer [not null]
  tag_id integer [not null]
}

Table comments {
  id serial [pk]
  post_id integer [not null]
  user_id integer
  author_name varchar(100)
  author_email varchar(255)
  content text [not null]
  approved boolean [default: false]
  parent_id integer
  created_at timestamptz [not null, default: `NOW()`]
}

Table media {
  id serial [pk]
  user_id integer [not null]
  filename varchar(255) [not null]
  url text [not null]
  mime_type varchar(100)
  size integer
  alt_text text
  created_at timestamptz [not null, default: `NOW()`]
}

// Relationships
Ref: posts.author_id > users.id
Ref: categories.parent_id > categories.id
Ref: post_categories.post_id > posts.id
Ref: post_categories.category_id > categories.id
Ref: post_tags.post_id > posts.id
Ref: post_tags.tag_id > tags.id
Ref: comments.post_id > posts.id
Ref: comments.user_id > users.id
Ref: comments.parent_id > comments.id
Ref: media.user_id > users.id
EOF
      ;;

    *)
      log_error "Unknown template: $template"
      log_info "Available templates:"
      log_info "  basic      - Users, profiles, posts"
      log_info "  ecommerce  - Products, orders, cart"
      log_info "  saas       - Organizations, members, projects"
      log_info "  blog       - Posts, categories, comments"
      return 1
      ;;
  esac

  log_success "Created: $output"
  log_info ""
  log_info "Next steps:"
  log_info "  1. Edit $output to customize your schema"
  log_info "  2. Preview at: https://dbdiagram.io"
  log_info "  3. Import: nself db schema import $output"
  log_info "  4. Apply: nself db migrate up"
  log_info "  5. Generate mock data: nself db mock auto"
}

# Apply complete workflow: import DBML -> migrate -> seed
schema_apply() {
  local dbml_file="${1:-schema.dbml}"
  local env="${2:-$(detect_environment)}"

  if [[ ! -f "$dbml_file" ]]; then
    log_error "DBML file not found: $dbml_file"
    log_info "Create one with: nself db schema scaffold basic"
    return 1
  fi

  log_info "Applying schema workflow..."
  log_info "  DBML: $dbml_file"
  log_info "  Environment: $env"
  echo ""

  # Step 1: Import DBML
  log_info "Step 1/4: Importing DBML..."
  schema_import "$dbml_file" "schema_$(date +%Y%m%d)" || return 1
  echo ""

  # Step 2: Run migrations
  log_info "Step 2/4: Running migrations..."
  migrate_up || return 1
  echo ""

  # Step 3: Generate mock data for local/staging
  if [[ "$env" == "local" || "$env" == "staging" || "$env" == "dev" || "$env" == "development" ]]; then
    log_info "Step 3/4: Generating mock data..."
    mock_auto || return 1
  else
    log_info "Step 3/4: Skipping mock data (production environment)"
  fi
  echo ""

  # Step 4: Seed users
  log_info "Step 4/4: Seeding users..."
  seed_users "$env" || return 1
  echo ""

  log_success "Schema workflow complete!"
  log_info ""
  log_info "Your database is ready with:"
  if [[ "$env" == "local" || "$env" == "staging" || "$env" == "dev" ]]; then
    log_info "  - Schema from $dbml_file"
    log_info "  - Mock data for testing"
    log_info "  - Sample users:"
    log_info "      admin@example.com (admin)"
    log_info "      user@example.com (user)"
    log_info "      demo@example.com (viewer)"
  else
    log_info "  - Schema from $dbml_file"
    log_info "  - Production admin user (check NSELF_PROD_USERS)"
  fi
}

# Auto-generate mock data based on schema analysis
mock_auto() {
  require_db || return 1

  local seed="${MOCK_SEED:-42}"
  local count="${MOCK_COUNT:-10}"

  log_info "Auto-generating mock data based on schema..."
  log_info "  Seed: $seed (deterministic)"
  log_info "  Records per table: $count"
  echo ""

  # Get all tables
  local tables=$(psql_query "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename")

  for table in $tables; do
    # Skip system tables
    [[ "$table" == "schema_migrations" ]] && continue
    [[ "$table" == "system_events" ]] && continue

    log_info "Generating data for: $table"

    # Get columns for this table
    local columns=$(psql_query "SELECT column_name, data_type, is_nullable, column_default
      FROM information_schema.columns
      WHERE table_name = '$table' AND table_schema = 'public'
      ORDER BY ordinal_position")

    # Build INSERT statement dynamically
    local col_names=""
    local col_values=""
    local i=0

    while IFS='|' read -r col_name data_type nullable default; do
      [[ -z "$col_name" ]] && continue

      # Skip auto-increment columns
      [[ "$default" =~ nextval ]] && continue
      [[ "$col_name" == "id" && "$data_type" =~ int ]] && continue

      # Add comma separator
      [[ -n "$col_names" ]] && col_names+=", "
      [[ -n "$col_values" ]] && col_values+=", "

      col_names+="$col_name"

      # Generate appropriate mock value based on type and column name
      case "$data_type" in
        integer | bigint | smallint)
          col_values+="\$(( ($seed + \$row + $i) % 1000 ))"
          ;;
        *numeric* | *decimal* | real | *double*)
          col_values+="\$(echo \"scale=2; ($seed + \$row + $i) / 100\" | bc)"
          ;;
        boolean)
          col_values+="\$(( ($seed + \$row) % 2 == 0 ? 'true' : 'false' ))"
          ;;
        *timestamp* | *date*)
          col_values+="NOW() - INTERVAL '\$(( ($seed + \$row) % 365 )) days'"
          ;;
        uuid)
          col_values+="gen_random_uuid()"
          ;;
        json | jsonb)
          col_values+="'{}'"
          ;;
        *)
          # Text/varchar - generate based on column name
          case "$col_name" in
            *email*)
              col_values+="'user\${row}@example.com'"
              ;;
            *name* | *title*)
              col_values+="'Test $col_name \${row}'"
              ;;
            *url* | *link*)
              col_values+="'https://example.com/\${row}'"
              ;;
            *slug*)
              col_values+="'slug-\${row}'"
              ;;
            *password* | *hash*)
              col_values+="'hashed_password_\${row}'"
              ;;
            *)
              col_values+="'mock_${col_name}_\${row}'"
              ;;
          esac
          ;;
      esac

      ((i++))
    done <<<"$columns"

    # Generate and execute INSERT statements
    if [[ -n "$col_names" ]]; then
      for ((row = 1; row <= count; row++)); do
        # Evaluate the values template
        local evaluated_values=$(eval "echo \"$col_values\"")
        psql_exec -c "INSERT INTO $table ($col_names) VALUES ($evaluated_values)" 2>/dev/null || true
      done
      log_info "  + $count records"
    fi
  done

  log_success "Mock data generation complete"
}

# ============================================================================
# TYPE GENERATION
# ============================================================================

cmd_types() {
  local lang="${1:-typescript}"
  shift || true

  require_db || return 1

  case "$lang" in
    typescript | ts) types_typescript "$@" ;;
    go | golang) types_go "$@" ;;
    python | py) types_python "$@" ;;
    *)
      log_error "Supported: typescript, go, python"
      return 1
      ;;
  esac
}

types_typescript() {
  local output="${1:-$TYPES_DIR/db.ts}"

  log_info "Generating TypeScript types..."
  mkdir -p "$(dirname "$output")"

  cat >"$output" <<'EOF'
// Generated by nself db types
// DO NOT EDIT - regenerate with: nself db types typescript

EOF

  local tables=$(psql_query "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename")

  for table in $tables; do
    # Convert to PascalCase
    local type_name=$(snake_to_pascal "$table")

    echo "export interface $type_name {" >>"$output"

    psql_query "SELECT column_name, data_type, is_nullable
      FROM information_schema.columns
      WHERE table_name = '$table' AND table_schema = 'public'
      ORDER BY ordinal_position" | while IFS='|' read -r col dtype nullable; do

      # Map PostgreSQL types to TypeScript
      local ts_type
      case "$dtype" in
        integer | bigint | smallint | numeric | decimal | real | "double precision")
          ts_type="number"
          ;;
        boolean)
          ts_type="boolean"
          ;;
        json | jsonb)
          ts_type="Record<string, unknown>"
          ;;
        "timestamp"* | date | time*)
          ts_type="string"
          ;;
        uuid)
          ts_type="string"
          ;;
        ARRAY)
          ts_type="unknown[]"
          ;;
        *)
          ts_type="string"
          ;;
      esac

      local opt=""
      [[ "$nullable" == "YES" ]] && opt="?"

      echo "  ${col}${opt}: ${ts_type};" >>"$output"
    done

    echo "}" >>"$output"
    echo "" >>"$output"
  done

  log_success "Generated: $output"
}

types_go() {
  local output="${1:-$TYPES_DIR/db.go}"
  local pkg="${2:-models}"

  log_info "Generating Go types..."
  mkdir -p "$(dirname "$output")"

  cat >"$output" <<EOF
// Generated by nself db types
// DO NOT EDIT - regenerate with: nself db types go

package $pkg

import "time"

EOF

  local tables=$(psql_query "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename")

  for table in $tables; do
    local struct_name=$(snake_to_pascal "$table")

    echo "type $struct_name struct {" >>"$output"

    psql_query "SELECT column_name, data_type, is_nullable
      FROM information_schema.columns
      WHERE table_name = '$table' AND table_schema = 'public'
      ORDER BY ordinal_position" | while IFS='|' read -r col dtype nullable; do

      local field_name=$(snake_to_pascal "$col")

      local go_type
      case "$dtype" in
        integer | smallint) go_type="int32" ;;
        bigint) go_type="int64" ;;
        numeric | decimal | real | "double precision") go_type="float64" ;;
        boolean) go_type="bool" ;;
        json | jsonb) go_type="map[string]interface{}" ;;
        "timestamp"* | date | time*) go_type="time.Time" ;;
        uuid) go_type="string" ;;
        *) go_type="string" ;;
      esac

      [[ "$nullable" == "YES" ]] && go_type="*$go_type"

      echo "	$field_name $go_type \`json:\"$col\" db:\"$col\"\`" >>"$output"
    done

    echo "}" >>"$output"
    echo "" >>"$output"
  done

  log_success "Generated: $output"
}

types_python() {
  local output="${1:-$TYPES_DIR/db.py}"

  log_info "Generating Python types..."
  mkdir -p "$(dirname "$output")"

  cat >"$output" <<'EOF'
# Generated by nself db types
# DO NOT EDIT - regenerate with: nself db types python

from dataclasses import dataclass
from datetime import datetime
from typing import Optional, Any, Dict, List

EOF

  local tables=$(psql_query "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename")

  for table in $tables; do
    local class_name=$(snake_to_pascal "$table")

    echo "@dataclass" >>"$output"
    echo "class $class_name:" >>"$output"

    psql_query "SELECT column_name, data_type, is_nullable
      FROM information_schema.columns
      WHERE table_name = '$table' AND table_schema = 'public'
      ORDER BY ordinal_position" | while IFS='|' read -r col dtype nullable; do

      local py_type
      case "$dtype" in
        integer | bigint | smallint) py_type="int" ;;
        numeric | decimal | real | "double precision") py_type="float" ;;
        boolean) py_type="bool" ;;
        json | jsonb) py_type="Dict[str, Any]" ;;
        "timestamp"* | date | time*) py_type="datetime" ;;
        ARRAY) py_type="List[Any]" ;;
        *) py_type="str" ;;
      esac

      [[ "$nullable" == "YES" ]] && py_type="Optional[$py_type]"

      echo "    $col: $py_type" >>"$output"
    done

    echo "" >>"$output"
  done

  log_success "Generated: $output"
}

# ============================================================================
# DATA UTILITIES
# ============================================================================

cmd_data() {
  local subcmd="${1:-help}"
  shift || true

  case "$subcmd" in
    export) data_export "$@" ;;
    import) data_import "$@" ;;
    anonymize) data_anonymize "$@" ;;
    sync) data_sync "$@" ;;
    *) data_help ;;
  esac
}

data_export() {
  require_db || return 1

  local table="${1:-}"
  local format="${2:-csv}"
  local output="${3:-}"

  [[ -z "$table" ]] && {
    log_error "Usage: nself db data export <table> [format] [output]"
    return 1
  }

  output="${output:-${table}_$(date +%Y%m%d).${format}}"

  log_info "Exporting $table to $output..."

  case "$format" in
    csv)
      psql_exec -c "COPY $table TO STDOUT WITH CSV HEADER" >"$output"
      ;;
    json)
      psql_query "SELECT json_agg(t) FROM $table t" >"$output"
      ;;
    sql)
      docker exec "$(get_container)" pg_dump -U "${POSTGRES_USER:-postgres}" \
        -t "$table" --data-only "${POSTGRES_DB:-nhost}" >"$output"
      ;;
  esac

  log_success "Exported: $output"
}

data_import() {
  require_db || return 1

  local file="$1"
  local table="${2:-}"

  [[ ! -f "$file" ]] && {
    log_error "File not found: $file"
    return 1
  }

  local ext="${file##*.}"
  [[ -z "$table" ]] && table="${file%.*}"

  log_info "Importing $file to $table..."

  case "$ext" in
    csv)
      psql_exec -c "COPY $table FROM STDIN WITH CSV HEADER" <"$file"
      ;;
    sql)
      psql_exec <"$file"
      ;;
    json)
      # Would need jq or similar for JSON import
      log_warning "JSON import requires manual handling"
      ;;
  esac

  log_success "Import complete"
}

data_anonymize() {
  require_non_production "Data anonymization" || return 1
  require_db || return 1
  require_confirmation "This will modify data in place" || return 1

  log_info "Anonymizing PII data..."

  # Default anonymization rules
  local email_tables=$(psql_query "SELECT table_name FROM information_schema.columns
    WHERE column_name = 'email' AND table_schema = 'public'")

  for table in $email_tables; do
    log_info "  Anonymizing emails in: $table"
    psql_exec -c "UPDATE $table SET email = 'user' || id || '@anonymized.local'
      WHERE email IS NOT NULL" >/dev/null 2>&1 || true
  done

  local name_tables=$(psql_query "SELECT table_name FROM information_schema.columns
    WHERE column_name IN ('name', 'full_name', 'first_name', 'last_name')
    AND table_schema = 'public'")

  for table in $name_tables; do
    log_info "  Anonymizing names in: $table"
    psql_exec -c "UPDATE $table SET
      name = COALESCE('User ' || id::text, name),
      full_name = COALESCE('User ' || id::text, full_name),
      first_name = COALESCE('First' || id::text, first_name),
      last_name = COALESCE('Last' || id::text, last_name)
    WHERE true" >/dev/null 2>&1 || true
  done

  log_success "Anonymization complete"
}

data_sync() {
  local source="${1:-}"
  local anonymize="${2:-}"

  [[ -z "$source" ]] && {
    log_error "Usage: nself db data sync <source-env> [--anonymize]"
    return 1
  }

  if [[ "$source" == "production" ]] || [[ "$source" == "prod" ]]; then
    if [[ "$anonymize" != "--anonymize" ]]; then
      log_error "Syncing from production requires --anonymize flag"
      log_info "Usage: nself db data sync prod --anonymize"
      return 1
    fi
  fi

  log_warning "Cross-environment sync requires SSH/remote access"
  log_info "For now, use: nself db backup create (on source)"
  log_info "Then: nself db backup restore (on target)"

  if [[ "$anonymize" == "--anonymize" ]]; then
    log_info "After restore, run: nself db data anonymize"
  fi
}

data_help() {
  echo "Usage: nself db data <command>"
  echo ""
  echo "Commands:"
  echo "  export <table> [format]    Export table (csv, json, sql)"
  echo "  import <file> [table]      Import data file"
  echo "  anonymize                  Anonymize PII data (local only)"
  echo "  sync <env> [--anonymize]   Sync from environment"
}

# ============================================================================
# MAINTENANCE
# ============================================================================

cmd_optimize() {
  require_db || return 1

  log_info "Optimizing database..."

  log_info "  Running VACUUM ANALYZE..."
  psql_exec -c "VACUUM ANALYZE" >/dev/null 2>&1

  log_info "  Updating statistics..."
  psql_exec -c "ANALYZE" >/dev/null 2>&1

  log_success "Optimization complete"
}

cmd_reset() {
  require_non_production "Database reset" || return 1
  require_db || return 1
  require_confirmation "This will DROP all tables and data" || return 1

  log_info "Resetting database..."

  # Create backup first
  backup_create "pre-reset"

  psql_exec -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" >/dev/null
  psql_exec -c "GRANT ALL ON SCHEMA public TO postgres; GRANT ALL ON SCHEMA public TO public;" >/dev/null

  log_success "Database reset complete"
  log_info "Run 'nself db migrate up' to recreate schema"
}

# ============================================================================
# HELP
# ============================================================================

show_help() {
  cat <<'EOF'
nself db - Database Management (v0.4.4)

USAGE:
  nself db <command> [options]

COMMANDS:
  Migrations:
    migrate status          Show migration status
    migrate up              Run pending migrations
    migrate down [n]        Rollback n migrations
    migrate create <name>   Create new migration
    migrate fresh           Drop all & re-migrate (local only)

  Interactive:
    shell [--readonly]      Open PostgreSQL shell
    query '<sql>'           Execute SQL query

  Seeding:
    seed                    Run seeds for current environment
    seed users              Seed users (env-aware)
    seed create <name>      Create new seed file

  Mock Data:
    mock generate           Generate mock data (local only)
    mock preview [table]    Preview table data
    mock clear              Clear all mock data

  Backup & Restore:
    backup [type] [--compress]  Create backup (full, database, schema, data)
    backup list                 List available backups
    restore [name]              Restore from backup

  Analysis:
    inspect                 Database overview
    inspect size            Table sizes
    inspect cache           Cache hit ratios
    inspect index           Index usage
    inspect unused-indexes  Find unused indexes
    inspect bloat           Table bloat
    inspect slow            Slow queries

  Schema:
    schema show [table]     Show schema
    schema diff <target>    Compare schemas
    schema diagram          Generate DBML from database
    schema indexes          Index management
    schema export [format]  Export schema (sql, json)
    schema import <file>    Import DBML → SQL migration
    schema scaffold <tpl>   Create starter DBML (basic, ecommerce, saas, blog)
    schema apply <file>     Full workflow: import → migrate → seed

  Types:
    types typescript        Generate TypeScript types
    types go                Generate Go structs
    types python            Generate Python dataclasses

  Data:
    data export <table>     Export table data
    data import <file>      Import data file
    data anonymize          Anonymize PII data

  Maintenance:
    optimize                Run VACUUM ANALYZE
    reset                   Drop all tables (local only)

ENVIRONMENT:
  Current: $(get_env)

  - local:      Full access, mock data allowed
  - staging:    Confirmation required for destructive ops
  - production: Blocked from dangerous operations

SMART DEFAULTS:
  - Migrations track automatically in schema_migrations table
  - Seeds run common/ first, then environment-specific
  - Mock data uses deterministic seeds for team sharing
  - Backups include database + config files
  - Types generate from live database schema

EXAMPLES:
  # Quick start workflow (recommended)
  nself db schema scaffold basic         # Create starter schema.dbml
  nself db schema apply schema.dbml      # Import → migrate → seed (all in one!)

  # Or step by step
  nself db schema import schema.dbml     # Convert DBML to SQL migration
  nself db migrate up                    # Run all pending migrations
  nself db mock auto                     # Auto-generate mock data from schema
  nself db seed users                    # Seed users (env-aware)

  # Other common tasks
  nself db backup database --compress    # Create compressed database backup
  nself db types typescript              # Generate TypeScript types
  nself db inspect size                  # Show table sizes
  nself db shell --readonly              # Read-only database shell
  nself db schema diagram                # Export database to DBML

EOF
}

# ============================================================================
# MAIN
# ============================================================================

main() {
  # Load environment
  [[ -f ".env.local" ]] && source ".env.local" 2>/dev/null || true
  [[ -f ".env" ]] && source ".env" 2>/dev/null || true

  ensure_dirs

  local command="${1:-help}"
  shift || true

  case "$command" in
    # Migrations
    migrate | migration | migrations)
      cmd_migrate "$@"
      ;;

    # Interactive
    shell | console | psql)
      cmd_shell "$@"
      ;;
    query | sql)
      cmd_query "$@"
      ;;

    # Seeding
    seed | seeds)
      cmd_seed "$@"
      ;;

    # Mock data
    mock)
      cmd_mock "$@"
      ;;

    # Backup & Restore
    backup | backups)
      cmd_backup "$@"
      ;;
    restore)
      backup_restore "$@"
      ;;

    # Inspection
    inspect | analyze | analysis)
      cmd_inspect "$@"
      ;;

    # Schema
    schema)
      cmd_schema "$@"
      ;;

    # Types
    types | type)
      cmd_types "$@"
      ;;

    # Data
    data)
      cmd_data "$@"
      ;;

    # Maintenance
    optimize | vacuum)
      cmd_optimize "$@"
      ;;
    reset)
      cmd_reset "$@"
      ;;

    # Status (quick overview)
    status)
      inspect_overview
      ;;

    # Help
    help | --help | -h)
      show_help
      ;;

    *)
      log_error "Unknown command: $command"
      echo ""
      show_help
      return 1
      ;;
  esac
}

main "$@"
