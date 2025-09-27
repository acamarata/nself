#!/usr/bin/env bash
# start-simple.sh - Simplified, reliable start command
# Environment-aware but execution-simple

# Don't use set -e as we want to handle errors gracefully
set -uo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source only essential utilities
source "$LIB_DIR/utils/display.sh"
source "$LIB_DIR/utils/env.sh"

# Simple start function
start_services() {
  local verbose="${1:-false}"

  # 1. Show header
  show_command_header "nself start" "Starting all services"

  # 2. Detect environment (simple)
  local env="${ENV:-dev}"
  if [[ -f ".env" ]]; then
    env=$(grep "^ENV=" .env 2>/dev/null | cut -d= -f2 || echo "dev")
  fi
  printf "Environment: ${COLOR_BLUE}%s${COLOR_RESET}\n\n" "$env"

  # 3. Get project name from .env or use directory name
  local project_name="${PROJECT_NAME:-}"
  if [[ -z "$project_name" ]] && [[ -f ".env" ]]; then
    project_name=$(grep "^PROJECT_NAME=" .env 2>/dev/null | cut -d= -f2)
  fi
  if [[ -z "$project_name" ]]; then
    project_name=$(basename "$PWD")
  fi

  # 4. Ensure we have required files
  if [[ ! -f "docker-compose.yml" ]]; then
    log_error "docker-compose.yml not found. Run 'nself build' first."
    return 1
  fi

  # 5. Determine which env file to use (build created .env.runtime for the current environment)
  local env_file=".env"
  if [[ -f ".env.runtime" ]]; then
    env_file=".env.runtime"
  fi

  # 6. Clean up any conflicting containers (from previous runs with different project names)
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning up any conflicting containers...\n"
  local existing_containers=$(docker ps -aq --filter "name=${project_name}_" 2>/dev/null)
  if [[ -n "$existing_containers" ]]; then
    docker rm -f $existing_containers >/dev/null 2>&1 || true
  fi
  printf "\r${COLOR_GREEN}✓${COLOR_RESET} Ready to start services                    \n"

  # 7. Start services with docker compose
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Starting Docker services...\n"

  local compose_cmd="docker compose"
  if ! command -v docker >/dev/null 2>&1; then
    log_error "Docker is not installed or not running"
    return 1
  fi

  # Use --no-build to avoid hanging, and remove orphans
  local start_output=$(mktemp)
  if $compose_cmd \
    --project-name "$project_name" \
    --env-file "$env_file" \
    up -d \
    --no-build \
    --remove-orphans \
    2>&1 | tee "$start_output"; then

    # Count started services
    local started_count=$(grep -c "Started" "$start_output" 2>/dev/null || echo "0")
    local created_count=$(grep -c "Created" "$start_output" 2>/dev/null || echo "0")
    local running_count=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')

    printf "\n${COLOR_GREEN}✓${COLOR_RESET} Services starting: %d running, %d created\n" "$running_count" "$created_count"

    # Give services a moment to stabilize
    sleep 3

    # Show final status
    printf "\n${COLOR_BLUE}Service Status:${COLOR_RESET}\n"
    docker ps --filter "label=com.docker.compose.project=$project_name" \
      --format "table {{.Names}}\t{{.Status}}" 2>/dev/null | head -20

    printf "\n${COLOR_GREEN}✓${COLOR_RESET} All services started\n"
    printf "  Use ${COLOR_BLUE}nself status${COLOR_RESET} to check health\n"
    printf "  Use ${COLOR_BLUE}nself urls${COLOR_RESET} to see service URLs\n"
    printf "  Use ${COLOR_BLUE}nself stop${COLOR_RESET} to stop services\n\n"

  else
    log_error "Failed to start services"
    cat "$start_output"
    rm -f "$start_output"
    return 1
  fi

  rm -f "$start_output"
  return 0
}

# Parse arguments
VERBOSE=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    -h|--help)
      echo "Usage: nself start [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  -v, --verbose   Show detailed output"
      echo "  -h, --help      Show this help message"
      exit 0
      ;;
    *)
      shift
      ;;
  esac
done

# Run start
start_services "$VERBOSE"