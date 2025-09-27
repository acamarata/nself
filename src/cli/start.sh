#!/usr/bin/env bash
# start.sh - Professional start command with clean progress indicators
# Matches the style of nself build command

set -uo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source only essential utilities
source "$LIB_DIR/utils/display.sh"
source "$LIB_DIR/utils/env.sh"

# Parse arguments first
VERBOSE=false
DEBUG=false
SHOW_HELP=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    -d|--debug)
      DEBUG=true
      VERBOSE=true  # Debug implies verbose
      shift
      ;;
    -h|--help)
      SHOW_HELP=true
      shift
      ;;
    *)
      shift
      ;;
  esac
done

# Show help if requested
if [[ "$SHOW_HELP" == "true" ]]; then
  echo "Usage: nself start [OPTIONS]"
  echo ""
  echo "Start all services defined in docker-compose.yml"
  echo ""
  echo "Options:"
  echo "  -v, --verbose   Show detailed Docker output"
  echo "  -d, --debug     Show debug information and detailed output"
  echo "  -h, --help      Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself start           # Start with clean output"
  echo "  nself start -v        # Start with verbose output"
  echo "  nself start --debug   # Start with full debug output"
  exit 0
fi

# Progress tracking functions
declare -a PROGRESS_STEPS=()
declare -a PROGRESS_STATUS=()
CURRENT_STEP=0

add_progress() {
  PROGRESS_STEPS+=("$1")
  PROGRESS_STATUS+=("pending")
}

update_progress() {
  local step=$1
  local status=$2
  PROGRESS_STATUS[$step]=$status

  if [[ "$VERBOSE" == "false" ]]; then
    # Clear line and show updated status
    local message="${PROGRESS_STEPS[$step]}"
    if [[ "$status" == "running" ]]; then
      printf "\r${COLOR_BLUE}⠋${COLOR_RESET} %s..." "$message"
    elif [[ "$status" == "done" ]]; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} %-40s\n" "$message"
    elif [[ "$status" == "error" ]]; then
      printf "\r${COLOR_RED}✗${COLOR_RESET} %-40s\n" "$message"
    fi
  fi
}

# Start services function
start_services() {
  # 1. Detect environment and project
  local env="${ENV:-dev}"
  if [[ -f ".env" ]]; then
    env=$(grep "^ENV=" .env 2>/dev/null | cut -d= -f2 || echo "dev")
  fi

  local project_name="${PROJECT_NAME:-}"
  if [[ -z "$project_name" ]] && [[ -f ".env" ]]; then
    project_name=$(grep "^PROJECT_NAME=" .env 2>/dev/null | cut -d= -f2)
  fi
  if [[ -z "$project_name" ]]; then
    project_name=$(basename "$PWD")
  fi

  # 2. Show header (like build command)
  show_command_header "nself start" "Start all project services"

  # 3. Setup progress steps
  add_progress "Validating prerequisites"
  add_progress "Cleaning previous state"
  add_progress "Creating network"
  add_progress "Creating volumes"
  add_progress "Creating containers"
  add_progress "Starting core services"
  add_progress "Starting optional services"
  add_progress "Starting monitoring"
  add_progress "Starting custom services"
  add_progress "Verifying health checks"

  # 4. Validate prerequisites
  update_progress 0 "running"

  if [[ ! -f "docker-compose.yml" ]]; then
    update_progress 0 "error"
    printf "\n${COLOR_RED}Error: docker-compose.yml not found${COLOR_RESET}\n"
    printf "Run '${COLOR_BLUE}nself build${COLOR_RESET}' first to generate configuration\n\n"
    return 1
  fi

  if ! command -v docker >/dev/null 2>&1; then
    update_progress 0 "error"
    printf "\n${COLOR_RED}Error: Docker is not installed or not running${COLOR_RESET}\n\n"
    return 1
  fi

  update_progress 0 "done"

  # 5. Clean up any conflicting containers FIRST (before env merge)
  update_progress 1 "running"

  # Also check for containers with the actual project name from .env
  local actual_project_name="${PROJECT_NAME:-$project_name}"
  if [[ -f ".env" ]]; then
    actual_project_name=$(grep "^PROJECT_NAME=" .env 2>/dev/null | cut -d= -f2 || echo "$project_name")
  fi

  # Clean up containers with both potential naming patterns
  local existing_containers=$(docker ps -aq --filter "name=${actual_project_name}_" 2>/dev/null)
  if [[ -n "$existing_containers" ]]; then
    docker rm -f $existing_containers >/dev/null 2>&1 || true
  fi

  # Also clean up with directory-based name if different
  if [[ "$project_name" != "$actual_project_name" ]]; then
    existing_containers=$(docker ps -aq --filter "name=${project_name}_" 2>/dev/null)
    if [[ -n "$existing_containers" ]]; then
      docker rm -f $existing_containers >/dev/null 2>&1 || true
    fi
  fi

  # Clean up any existing network to avoid conflicts
  docker network rm "${actual_project_name}_network" >/dev/null 2>&1 || true
  docker network rm "${project_name}_default" >/dev/null 2>&1 || true

  update_progress 1 "done"

  # 6. Source env-merger if available
  if [[ -f "$LIB_DIR/utils/env-merger.sh" ]]; then
    source "$LIB_DIR/utils/env-merger.sh"
  fi

  # 7. Generate merged runtime environment
  local target_env="${ENV:-dev}"
  if command -v merge_environments >/dev/null 2>&1; then
    if [[ "$VERBOSE" == "false" ]]; then
      merge_environments "$target_env" ".env.runtime" > /dev/null 2>&1
    else
      printf "Merging environment configuration...\n"
      merge_environments "$target_env" ".env.runtime"
    fi
  fi

  # 8. Determine env file and update project name from runtime
  local env_file=".env"
  if [[ -f ".env.runtime" ]]; then
    env_file=".env.runtime"
    # Update project_name from runtime file
    project_name=$(grep "^PROJECT_NAME=" .env.runtime 2>/dev/null | cut -d= -f2 || echo "$project_name")
  fi

  # 9. Start services with progress tracking
  local compose_cmd="docker compose"
  local start_output=$(mktemp)
  local error_output=$(mktemp)

  # Build the docker compose command
  local compose_args=(
    "--project-name" "$project_name"
    "--env-file" "$env_file"
    "up" "-d"
    "--remove-orphans"
  )

  if [[ "$DEBUG" == "true" ]]; then
    echo ""
    echo "DEBUG: Project name: $project_name"
    echo "DEBUG: Environment: $env"
    echo "DEBUG: Env file: $env_file"
    echo "DEBUG: Command: $compose_cmd ${compose_args[*]}"
    echo ""
  fi

  # Execute docker compose
  if [[ "$VERBOSE" == "true" ]]; then
    # Verbose mode - show Docker output directly
    $compose_cmd "${compose_args[@]}" 2>&1 | tee "$start_output"
    local exit_code=${PIPESTATUS[0]}
  else
    # Clean mode - capture output and show progress
    $compose_cmd "${compose_args[@]}" > "$start_output" 2> "$error_output" &
    local compose_pid=$!

    # Spinner characters for animation
    local spinner=('⠋' '⠙' '⠹' '⠸' '⠼' '⠴' '⠦' '⠧' '⠇' '⠏')
    local spin_index=0

    # Track progress based on docker output
    local network_done=false
    local volumes_done=false
    local containers_created=false
    local services_starting=false
    local monitoring_started=false
    local custom_started=false

    # Count total expected services (for progress tracking)
    local total_services=25  # From demo config
    local images_pulled=0
    local containers_started=0
    local current_action="Preparing Docker environment"
    local last_line=""

    while ps -p $compose_pid > /dev/null 2>&1; do
      # Update spinner
      spin_index=$(( (spin_index + 1) % 10 ))

      # Get the last non-empty line from output to see what's happening
      last_line=$(tail -n 5 "$start_output" 2>/dev/null | grep -v "^$" | tail -n 1 || echo "")

      # Check what's happening based on output patterns
      if echo "$last_line" | grep -q "Building\|Step\|RUN\|COPY\|FROM"; then
        # Building custom images
        current_action="Building custom Docker images"
        printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s..." "${spinner[$spin_index]}" "$current_action"

      elif echo "$last_line" | grep -q "Pulling\|Pull complete\|Already exists\|Downloading\|Extracting\|Waiting"; then
        # Count unique images being pulled
        images_pulled=$(grep -cE "(Pulling |Pull complete|Already exists|Downloaded newer)" "$start_output" 2>/dev/null || echo "0")
        current_action="Downloading Docker images"
        printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d images)" "${spinner[$spin_index]}" "$current_action" "$images_pulled"

      elif [[ "$network_done" == "false" ]] && grep -q "Network.*Created" "$start_output" 2>/dev/null; then
        update_progress 2 "done"
        network_done=true
        current_action="Creating network"

      elif [[ "$volumes_done" == "false" ]] && grep -q "Volume.*Created" "$start_output" 2>/dev/null; then
        update_progress 3 "done"
        volumes_done=true
        current_action="Creating volumes"

      elif echo "$last_line" | grep -q "Container.*Creating\|Container.*Created"; then
        # Count containers being created
        local created_count=$(grep -c "Container.*Created" "$start_output" 2>/dev/null || echo "0")
        local creating_count=$(grep -c "Container.*Creating" "$start_output" 2>/dev/null || echo "0")
        current_action="Creating containers"
        if [[ "$containers_created" == "false" ]]; then
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d/%d)" "${spinner[$spin_index]}" "$current_action" "$created_count" "$total_services"
        fi
        if [[ "$created_count" -ge "$total_services" ]] && [[ "$containers_created" == "false" ]]; then
          update_progress 4 "done"
          containers_created=true
        fi

      elif echo "$last_line" | grep -q "Container.*Starting\|Container.*Started\|Container.*Running"; then
        # Count containers being started
        containers_started=$(grep -c "Container.*Started" "$start_output" 2>/dev/null || echo "0")
        current_action="Starting containers"
        printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d/%d)" "${spinner[$spin_index]}" "$current_action" "$containers_started" "$total_services"

        # Update specific service categories as they start
        if [[ "$services_starting" == "false" ]] && grep -q "Container ${project_name}_postgres.*Started" "$start_output" 2>/dev/null; then
          update_progress 5 "done"
          services_starting=true
        fi

        if [[ "$services_starting" == "true" ]] && grep -q "Container ${project_name}_minio.*Started" "$start_output" 2>/dev/null; then
          update_progress 6 "done"
        fi

        if [[ "$monitoring_started" == "false" ]] && grep -q "Container ${project_name}_prometheus.*Started" "$start_output" 2>/dev/null; then
          update_progress 7 "done"
          monitoring_started=true
        fi

        if [[ "$custom_started" == "false" ]] && grep -q "Container ${project_name}_express_api.*Started" "$start_output" 2>/dev/null; then
          update_progress 8 "done"
          custom_started=true
        fi
      else
        # Default spinner while waiting
        printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s..." "${spinner[$spin_index]}" "$current_action"
      fi

      sleep 0.1  # Faster updates for smoother animation
    done

    wait $compose_pid
    local exit_code=$?

    # Clear the spinner line
    printf "\r%-60s\r" " "
  fi

  # 10. Check results
  if [[ $exit_code -eq 0 ]]; then
    # Mark any remaining steps as done
    for i in 2 3 4 5 6 7 8; do
      if [[ "${PROGRESS_STATUS[$i]}" == "pending" ]]; then
        update_progress $i "done"
      fi
    done

    # Verify health checks
    update_progress 9 "running"
    sleep 2  # Give services a moment to start health checks
    update_progress 9 "done"

    # Count running containers and health status
    local running_count=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')
    local healthy_count=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Status}}" 2>/dev/null | grep -c "healthy" || echo "0")
    local total_with_health=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Status}}" 2>/dev/null | grep -cE "(healthy|unhealthy|starting)" || echo "0")

    # Count service types
    local core_count=4
    local optional_count=$(grep -c "_ENABLED=true" "$env_file" 2>/dev/null || echo "0")
    local monitoring_count=0
    if grep -q "MONITORING_ENABLED=true" "$env_file" 2>/dev/null; then
      monitoring_count=10
    fi
    local custom_count=$(grep -c "^CS_[0-9]=" "$env_file" 2>/dev/null || echo "0")

    # Final summary (like build command)
    printf "\n"
    printf "${COLOR_GREEN}✓${COLOR_RESET} ${COLOR_BOLD}All services started successfully${COLOR_RESET}\n"
    printf "${COLOR_GREEN}✓${COLOR_RESET} Project: ${COLOR_BOLD}%s${COLOR_RESET} (%s) / BD: %s\n" "$project_name" "$env" "${BASE_DOMAIN:-localhost}"
    printf "${COLOR_GREEN}✓${COLOR_RESET} Services (%d): %d core, %d optional, %d monitoring, %d custom\n" \
      "$running_count" "$core_count" "$optional_count" "$monitoring_count" "$custom_count"

    if [[ $total_with_health -gt 0 ]]; then
      printf "${COLOR_GREEN}✓${COLOR_RESET} Health: %d/%d checks passing\n" "$healthy_count" "$total_with_health"
    fi

    printf "\n\n${COLOR_BOLD}Next steps:${COLOR_RESET}\n\n"
    printf "1. ${COLOR_BLUE}nself status${COLOR_RESET} - Check service health\n"
    printf "   View detailed status of all running services\n\n"
    printf "2. ${COLOR_BLUE}nself urls${COLOR_RESET} - View service URLs\n"
    printf "   Access your application and service dashboards\n\n"
    printf "3. ${COLOR_BLUE}nself logs${COLOR_RESET} - View service logs\n"
    printf "   Monitor real-time logs from all services\n\n"
    printf "For more help, use: ${COLOR_DIM}nself help${COLOR_RESET} or ${COLOR_DIM}nself help start${COLOR_RESET}\n\n"

  else
    # Error occurred - mark remaining steps as error
    for i in "${!PROGRESS_STATUS[@]}"; do
      if [[ "${PROGRESS_STATUS[$i]}" == "pending" || "${PROGRESS_STATUS[$i]}" == "running" ]]; then
        update_progress $i "error"
      fi
    done

    printf "\n${COLOR_RED}✗ Failed to start services${COLOR_RESET}\n\n"

    # Show error details
    if [[ -s "$error_output" ]]; then
      printf "${COLOR_RED}Error details:${COLOR_RESET}\n"
      # Show meaningful errors only
      grep -E "(ERROR|Error|error|failed|Failed|dependency|unhealthy)" "$error_output" 2>/dev/null | head -5 || true

      # Check specifically for postgres issues
      if grep -q "demo-app_postgres.*unhealthy\|demo-app_postgres.*Error" "$error_output" 2>/dev/null; then
        printf "\n${COLOR_YELLOW}PostgreSQL startup issue detected${COLOR_RESET}\n"
        printf "Check logs with: ${COLOR_DIM}docker logs demo-app_postgres${COLOR_RESET}\n"
      fi
    fi

    # In verbose mode, show full output
    if [[ "$VERBOSE" == "true" ]] && [[ -s "$start_output" ]]; then
      printf "\n${COLOR_DIM}Full output:${COLOR_RESET}\n"
      cat "$start_output"
    fi

    printf "\n💡 ${COLOR_DIM}Tip: Run with --verbose for detailed output${COLOR_RESET}\n\n"

    rm -f "$start_output" "$error_output"
    return 1
  fi

  # Clean up temp files
  rm -f "$start_output" "$error_output"
  return 0
}

# Run start
start_services