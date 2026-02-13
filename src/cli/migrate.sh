#!/usr/bin/env bash
# migrate.sh - DEPRECATED: Use 'nself db migrate' instead
# This wrapper redirects to the proper db subcommand

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for deprecation warning
COLOR_RESET='\033[0m'
COLOR_YELLOW='\033[0;33m'
COLOR_BLUE='\033[0;34m'
COLOR_DIM='\033[2m'

# Show help
show_help() {
  cat << 'EOF'
Usage: nself migrate <command> [options]

Database migration management - apply schema changes without restarting containers.

Commands:
  create <name>    Create a new migration file
  apply            Apply all pending migrations
  rollback         Rollback the last applied migration
  status           Show migration status
  list             List all migrations

Options:
  -h, --help       Show this help message
  -v, --verbose    Show detailed output

Examples:
  nself migrate create add_user_roles
  nself migrate apply
  nself migrate rollback
  nself migrate status

Migration File Format:
  Migrations are stored in db/migrations/
  Format: YYYYMMDD_HHMMSS_description.sql

  Example migration file:
  -- Migration: add_user_roles
  -- Created: 2026-02-13

  -- UP: Apply migration
  CREATE TABLE user_roles (
    user_id UUID REFERENCES auth.users(id),
    role VARCHAR(50) NOT NULL
  );

  -- DOWN: Rollback migration (optional)
  DROP TABLE user_roles;

EOF
}

# Ensure migrations directory exists
ensure_migrations_dir() {
  if [[ ! -d "$MIGRATIONS_DIR" ]]; then
    mkdir -p "$MIGRATIONS_DIR"
    printf "${COLOR_GREEN}✓${COLOR_RESET} Created migrations directory: $MIGRATIONS_DIR\n"
  fi
}

# Get database connection info
get_db_connection() {
  # Load environment
  if [[ -f ".env.runtime" ]]; then
    set -a
    source .env.runtime 2>/dev/null || true
    set +a
  elif [[ -f ".env" ]]; then
    set -a
    source .env 2>/dev/null || true
    set +a
  fi

  local db_name="${POSTGRES_DB:-postgres}"
  local db_user="${POSTGRES_USER:-postgres}"
  local project_name="${PROJECT_NAME:-$(basename "$PWD")}"

  # Determine postgres container name
  local container_name="${project_name}_postgres"

  # Check if container exists
  if ! docker ps --filter "name=$container_name" --format "{{.Names}}" | grep -q "$container_name"; then
    printf "${COLOR_RED}✗${COLOR_RESET} PostgreSQL container not running: $container_name\n" >&2
    printf "  Run ${COLOR_BLUE}nself start${COLOR_RESET} first\n" >&2
    return 1
  fi

  echo "$container_name:$db_user:$db_name"
}

# Execute SQL in database
exec_sql() {
  local sql="$1"
  local connection=$(get_db_connection) || return 1
  local container=$(echo "$connection" | cut -d: -f1)
  local user=$(echo "$connection" | cut -d: -f2)
  local db=$(echo "$connection" | cut -d: -f3)

  docker exec -i "$container" psql -U "$user" -d "$db" -c "$sql" 2>&1
}

# Ensure migrations table exists
ensure_migrations_table() {
  local sql="CREATE TABLE IF NOT EXISTS $MIGRATIONS_TABLE (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    applied_at TIMESTAMPTZ DEFAULT NOW()
  );"

  exec_sql "$sql" >/dev/null 2>&1 || {
    printf "${COLOR_RED}✗${COLOR_RESET} Failed to create migrations table\n" >&2
    return 1
  }
}

# Create new migration
cmd_create() {
  local name="$1"

  if [[ -z "$name" ]]; then
    printf "${COLOR_RED}✗${COLOR_RESET} Migration name required\n" >&2
    printf "  Usage: nself migrate create <name>\n" >&2
    return 1
  fi

  ensure_migrations_dir

  # Generate timestamp
  local timestamp=$(date +%Y%m%d_%H%M%S)
  local filename="${MIGRATIONS_DIR}/${timestamp}_${name}.sql"

  # Create migration file with template
  cat > "$filename" << 'EOFMIG'
-- Migration: MIGRATION_NAME
-- Created: CREATION_DATE

-- UP: Apply migration
-- Add your SQL here


-- DOWN: Rollback migration (optional)
-- Add rollback SQL here

EOFMIG

  # Replace placeholders
  sed -i.bak "s/MIGRATION_NAME/$name/g" "$filename"
  sed -i.bak "s/CREATION_DATE/$(date +%Y-%m-%d)/g" "$filename"
  rm -f "$filename.bak"

  printf "${COLOR_GREEN}✓${COLOR_RESET} Created migration: ${COLOR_BLUE}$filename${COLOR_RESET}\n"
  printf "  Edit the file to add your SQL statements\n"
}

# Get list of applied migrations
get_applied_migrations() {
  ensure_migrations_table || return 1

  exec_sql "SELECT name FROM $MIGRATIONS_TABLE ORDER BY id;" 2>/dev/null | \
    grep -v "^-" | grep -v "^ *$" | grep -v "^name$" | grep -v "rows)" | grep -v "^(" || true
}

# Get list of pending migrations
get_pending_migrations() {
  if [[ ! -d "$MIGRATIONS_DIR" ]]; then
    return 0
  fi

  local applied=$(get_applied_migrations | tr '\n' '|' | sed 's/|$//' | sed 's/^ *//')

  find "$MIGRATIONS_DIR" -name "*.sql" -type f | sort | while read -r file; do
    local basename=$(basename "$file" .sql)
    if [[ -z "$applied" ]] || ! echo "$basename" | grep -qE "^($applied)$"; then
      echo "$basename"
    fi
  done
}

# Apply pending migrations
cmd_apply() {
  ensure_migrations_table || return 1

  local pending=$(get_pending_migrations)

  if [[ -z "$pending" ]]; then
    printf "${COLOR_GREEN}✓${COLOR_RESET} No pending migrations\n"
    return 0
  fi

  local count=$(echo "$pending" | wc -l | tr -d ' ')
  printf "${COLOR_BLUE}⠿${COLOR_RESET} Applying $count migration(s)...\n\n"

  local applied=0
  local failed=0
  local connection=$(get_db_connection) || return 1
  local container=$(echo "$connection" | cut -d: -f1)
  local user=$(echo "$connection" | cut -d: -f2)
  local db=$(echo "$connection" | cut -d: -f3)

  while IFS= read -r migration; do
    local file="${MIGRATIONS_DIR}/${migration}.sql"

    printf "  ${COLOR_BLUE}⠿${COLOR_RESET} Applying: $migration\n"

    # Extract UP section (everything before -- DOWN:)
    local up_sql=$(sed -n '/-- UP:/,/-- DOWN:/p' "$file" | grep -v "^-- DOWN:" | grep -v "^-- UP:")

    if [[ -z "$up_sql" ]]; then
      # No UP section marker, use entire file
      up_sql=$(cat "$file")
    fi

    # Apply migration
    if echo "$up_sql" | docker exec -i "$container" psql -U "$user" -d "$db" >/dev/null 2>&1; then

      # Record in migrations table
      exec_sql "INSERT INTO $MIGRATIONS_TABLE (name) VALUES ('$migration');" >/dev/null 2>&1

      printf "    ${COLOR_GREEN}✓${COLOR_RESET} Applied: $migration\n"
      ((applied++))
    else
      printf "    ${COLOR_RED}✗${COLOR_RESET} Failed: $migration\n" >&2
      ((failed++))
      break
    fi
  done <<< "$pending"

  printf "\n"

  if [[ $failed -eq 0 ]]; then
    printf "${COLOR_GREEN}✓${COLOR_RESET} Successfully applied $applied migration(s)\n"
    return 0
  else
    printf "${COLOR_RED}✗${COLOR_RESET} Failed to apply migrations ($applied applied, $failed failed)\n" >&2
    return 1
  fi
}

# Rollback last migration
cmd_rollback() {
  ensure_migrations_table || return 1

  # Get last applied migration
  local last=$(exec_sql "SELECT name FROM $MIGRATIONS_TABLE ORDER BY id DESC LIMIT 1;" 2>/dev/null | \
    grep -v "^-" | grep -v "^ *$" | grep -v "^name$" | grep -v "rows)" | grep -v "^(" | head -n 1 | sed 's/^ *//')

  if [[ -z "$last" ]]; then
    printf "${COLOR_YELLOW}⚠${COLOR_RESET} No migrations to rollback\n"
    return 0
  fi

  local file="${MIGRATIONS_DIR}/${last}.sql"

  if [[ ! -f "$file" ]]; then
    printf "${COLOR_RED}✗${COLOR_RESET} Migration file not found: $file\n" >&2
    return 1
  fi

  printf "${COLOR_YELLOW}⚠${COLOR_RESET} Rolling back: $last\n"

  # Extract DOWN section
  local down_sql=$(sed -n '/-- DOWN:/,$p' "$file" | grep -v "^-- DOWN:")

  if [[ -z "$down_sql" ]]; then
    printf "${COLOR_RED}✗${COLOR_RESET} No rollback SQL found (-- DOWN: section missing)\n" >&2
    return 1
  fi

  local connection=$(get_db_connection) || return 1
  local container=$(echo "$connection" | cut -d: -f1)
  local user=$(echo "$connection" | cut -d: -f2)
  local db=$(echo "$connection" | cut -d: -f3)

  # Apply rollback
  if echo "$down_sql" | docker exec -i "$container" psql -U "$user" -d "$db" >/dev/null 2>&1; then

    # Remove from migrations table
    exec_sql "DELETE FROM $MIGRATIONS_TABLE WHERE name = '$last';" >/dev/null 2>&1

    printf "${COLOR_GREEN}✓${COLOR_RESET} Rolled back: $last\n"
    return 0
  else
    printf "${COLOR_RED}✗${COLOR_RESET} Failed to rollback migration\n" >&2
    return 1
  fi
}

# Show migration status
cmd_status() {
  ensure_migrations_table || return 1

  printf "${COLOR_BLUE}Migration Status:${COLOR_RESET}\n\n"

  # Applied migrations
  local applied=$(get_applied_migrations | sed 's/^ *//')

  if [[ -n "$applied" ]]; then
    printf "${COLOR_GREEN}Applied Migrations:${COLOR_RESET}\n"
    while IFS= read -r migration; do
      [[ -z "$migration" ]] && continue
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} $migration\n"
    done <<< "$applied"
    printf "\n"
  else
    printf "${COLOR_DIM}No applied migrations${COLOR_RESET}\n\n"
  fi

  # Pending migrations
  local pending=$(get_pending_migrations)

  if [[ -n "$pending" ]]; then
    printf "${COLOR_YELLOW}Pending Migrations:${COLOR_RESET}\n"
    while IFS= read -r migration; do
      [[ -z "$migration" ]] && continue
      printf "  ${COLOR_YELLOW}○${COLOR_RESET} $migration\n"
    done <<< "$pending"
    printf "\n"
  else
    printf "${COLOR_DIM}No pending migrations${COLOR_RESET}\n\n"
  fi

  # Summary
  local applied_count=$(echo "$applied" | grep -c . || echo "0")
  local pending_count=$(echo "$pending" | grep -c . || echo "0")

  printf "${COLOR_DIM}Total:${COLOR_RESET} $applied_count applied, $pending_count pending\n"
}

# List all migrations
cmd_list() {
  ensure_migrations_dir

  if [[ ! -d "$MIGRATIONS_DIR" ]] || [[ -z "$(ls -A "$MIGRATIONS_DIR"/*.sql 2>/dev/null)" ]]; then
    printf "${COLOR_DIM}No migrations found in $MIGRATIONS_DIR${COLOR_RESET}\n"
    return 0
  fi

  printf "${COLOR_BLUE}Available Migrations:${COLOR_RESET}\n\n"

  find "$MIGRATIONS_DIR" -name "*.sql" -type f | sort | while read -r file; do
    local basename=$(basename "$file" .sql")
    printf "  $basename\n"
  done
}

# Main command dispatcher
main() {
  local command="${1:-}"

  case "$command" in
    create)
      shift
      cmd_create "$@"
      ;;
    apply)
      cmd_apply
      ;;
    rollback)
      cmd_rollback
      ;;
    status)
      cmd_status
      ;;
    list)
      cmd_list
      ;;
    -h|--help|help|"")
      show_help
      ;;
    *)
      printf "${COLOR_RED}✗${COLOR_RESET} Unknown command: $command\n" >&2
      printf "  Run ${COLOR_BLUE}nself migrate --help${COLOR_RESET} for usage\n" >&2
      return 1
      ;;
  esac
}

main "$@"
