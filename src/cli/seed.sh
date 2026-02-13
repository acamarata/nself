#!/usr/bin/env bash
# seed.sh - Database seeding (load demo/test data)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source utilities
source "$LIB_DIR/utils/display.sh"
source "$LIB_DIR/utils/env.sh"

# Seed directory
SEEDS_DIR="${SEEDS_DIR:-db/seeds}"

# Colors
COLOR_RESET='\033[0m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_BLUE='\033[0;34m'
COLOR_DIM='\033[2m'

# Show help
show_help() {
  cat << 'EOF'
Usage: nself seed [seed-name] [options]

Load demo/test data from SQL seed files.

Commands:
  nself seed               Run all seeds in order
  nself seed <name>        Run a specific seed file
  nself seed --list        List available seed files
  nself seed --all         Run all seeds (same as no arguments)

Options:
  -h, --help              Show this help message
  -v, --verbose           Show detailed output
  --list                  List available seed files

Examples:
  nself seed                    # Run all seeds
  nself seed users              # Run db/seeds/users.sql
  nself seed 01_demo_users      # Run db/seeds/01_demo_users.sql
  nself seed --list             # List all seed files

Seed File Format:
  Seeds are stored in db/seeds/
  Files are executed in alphabetical order (use prefixes: 01_, 02_, etc.)

  Example seed file (db/seeds/01_demo_users.sql):
  -- Seed: Demo Users
  -- Description: Create demo user accounts

  INSERT INTO auth.users (id, email, display_name)
  VALUES
    ('user-1', 'demo@example.com', 'Demo User'),
    ('user-2', 'admin@example.com', 'Admin User')
  ON CONFLICT (email) DO NOTHING;

  Note: Use ON CONFLICT to make seeds idempotent (safe to re-run)

EOF
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

# List all seed files
list_seeds() {
  if [[ ! -d "$SEEDS_DIR" ]]; then
    printf "${COLOR_DIM}No seeds directory found: $SEEDS_DIR${COLOR_RESET}\n"
    return 0
  fi

  local seeds=$(find "$SEEDS_DIR" -name "*.sql" -type f | sort)

  if [[ -z "$seeds" ]]; then
    printf "${COLOR_DIM}No seed files found in $SEEDS_DIR${COLOR_RESET}\n"
    return 0
  fi

  printf "${COLOR_BLUE}Available Seed Files:${COLOR_RESET}\n\n"

  while IFS= read -r file; do
    local basename=$(basename "$file")
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} $basename\n"
  done <<< "$seeds"

  local count=$(echo "$seeds" | wc -l | tr -d ' ')
  printf "\n${COLOR_DIM}Total: $count seed file(s)${COLOR_RESET}\n"
}

# Run a single seed file
run_seed_file() {
  local file="$1"
  local basename=$(basename "$file")

  printf "  ${COLOR_BLUE}⠿${COLOR_RESET} Running: $basename\n"

  local connection=$(get_db_connection) || return 1
  local container=$(echo "$connection" | cut -d: -f1)
  local user=$(echo "$connection" | cut -d: -f2)
  local db=$(echo "$connection" | cut -d: -f3)

  if docker exec -i "$container" psql -U "$user" -d "$db" < "$file" >/dev/null 2>&1; then
    printf "    ${COLOR_GREEN}✓${COLOR_RESET} Completed: $basename\n"
    return 0
  else
    printf "    ${COLOR_RED}✗${COLOR_RESET} Failed: $basename\n" >&2
    return 1
  fi
}

# Run all seed files
run_all_seeds() {
  if [[ ! -d "$SEEDS_DIR" ]]; then
    printf "${COLOR_YELLOW}⚠${COLOR_RESET} Seeds directory not found: $SEEDS_DIR\n"
    printf "  Create ${COLOR_BLUE}$SEEDS_DIR${COLOR_RESET} and add SQL seed files\n"
    return 0
  fi

  local seeds=$(find "$SEEDS_DIR" -name "*.sql" -type f | sort)

  if [[ -z "$seeds" ]]; then
    printf "${COLOR_YELLOW}⚠${COLOR_RESET} No seed files found in $SEEDS_DIR\n"
    return 0
  fi

  local count=$(echo "$seeds" | wc -l | tr -d ' ')
  printf "${COLOR_BLUE}⠿${COLOR_RESET} Running $count seed file(s)...\n\n"

  local success=0
  local failed=0

  while IFS= read -r file; do
    if run_seed_file "$file"; then
      ((success++))
    else
      ((failed++))
      break
    fi
  done <<< "$seeds"

  printf "\n"

  if [[ $failed -eq 0 ]]; then
    printf "${COLOR_GREEN}✓${COLOR_RESET} Successfully ran $success seed file(s)\n"
    return 0
  else
    printf "${COLOR_RED}✗${COLOR_RESET} Failed to run seeds ($success succeeded, $failed failed)\n" >&2
    return 1
  fi
}

# Run a specific seed
run_specific_seed() {
  local name="$1"

  if [[ ! -d "$SEEDS_DIR" ]]; then
    printf "${COLOR_RED}✗${COLOR_RESET} Seeds directory not found: $SEEDS_DIR\n" >&2
    return 1
  fi

  # Try exact filename first
  local file="$SEEDS_DIR/${name}.sql"

  if [[ ! -f "$file" ]]; then
    # Try without .sql extension
    file="$SEEDS_DIR/${name}"
  fi

  if [[ ! -f "$file" ]]; then
    # Try finding by pattern
    local matches=$(find "$SEEDS_DIR" -name "*${name}*.sql" -type f | head -n 1)
    if [[ -n "$matches" ]]; then
      file="$matches"
    else
      printf "${COLOR_RED}✗${COLOR_RESET} Seed file not found: $name\n" >&2
      printf "  Run ${COLOR_BLUE}nself seed --list${COLOR_RESET} to see available seeds\n" >&2
      return 1
    fi
  fi

  printf "${COLOR_BLUE}⠿${COLOR_RESET} Running seed: $(basename "$file")\n\n"

  if run_seed_file "$file"; then
    printf "\n${COLOR_GREEN}✓${COLOR_RESET} Seed completed successfully\n"
    return 0
  else
    printf "\n${COLOR_RED}✗${COLOR_RESET} Seed failed\n" >&2
    return 1
  fi
}

# Main command dispatcher
main() {
  local command="${1:-}"

  case "$command" in
    --list)
      list_seeds
      ;;
    --all)
      run_all_seeds
      ;;
    -h|--help|help)
      show_help
      ;;
    "")
      run_all_seeds
      ;;
    *)
      run_specific_seed "$command"
      ;;
  esac
}

main "$@"
