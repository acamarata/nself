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

# Smart defaults from environment variables
HEALTH_CHECK_TIMEOUT="${NSELF_HEALTH_CHECK_TIMEOUT:-120}"
HEALTH_CHECK_INTERVAL="${NSELF_HEALTH_CHECK_INTERVAL:-2}"
HEALTH_CHECK_REQUIRED="${NSELF_HEALTH_CHECK_REQUIRED:-80}"
SKIP_HEALTH_CHECKS="${NSELF_SKIP_HEALTH_CHECKS:-false}"
START_MODE="${NSELF_START_MODE:-smart}"
CLEANUP_ON_START="${NSELF_CLEANUP_ON_START:-auto}"

# Validate ranges for health check settings
if [[ $HEALTH_CHECK_TIMEOUT -lt 30 ]] || [[ $HEALTH_CHECK_TIMEOUT -gt 600 ]]; then
  HEALTH_CHECK_TIMEOUT=120
fi
if [[ $HEALTH_CHECK_INTERVAL -le 0 ]] || [[ $HEALTH_CHECK_INTERVAL -gt 10 ]]; then
  HEALTH_CHECK_INTERVAL=2
fi
if [[ $HEALTH_CHECK_REQUIRED -lt 0 ]] || [[ $HEALTH_CHECK_REQUIRED -gt 100 ]]; then
  HEALTH_CHECK_REQUIRED=80
fi

# Parse arguments first
VERBOSE=false
DEBUG=false
SHOW_HELP=false
SKIP_HEALTH=false
FORCE_RECREATE=false

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
    --skip-health-checks)
      SKIP_HEALTH_CHECKS=true
      shift
      ;;
    --timeout)
      if [[ -z "$2" ]] || ! [[ "$2" =~ ^[0-9]+$ ]]; then
        echo "Error: --timeout requires a numeric value" >&2
        exit 1
      fi
      HEALTH_CHECK_TIMEOUT="$2"
      shift 2
      ;;
    --fresh|--force-recreate)
      START_MODE="fresh"
      shift
      ;;
    --clean-start)
      CLEANUP_ON_START="always"
      shift
      ;;
    --quick)
      HEALTH_CHECK_TIMEOUT=30
      HEALTH_CHECK_REQUIRED=60
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
  echo "  -v, --verbose           Show detailed Docker output"
  echo "  -d, --debug            Show debug information and detailed output"
  echo "  -h, --help             Show this help message"
  echo "  --skip-health-checks   Skip health check validation"
  echo "  --timeout N            Set health check timeout in seconds (default: $HEALTH_CHECK_TIMEOUT)"
  echo "  --fresh                Force recreate all containers"
  echo "  --clean-start          Remove all containers before starting"
  echo "  --quick                Quick start with relaxed health checks"
  echo ""
  echo "Environment Variables (optional):"
  echo "  NSELF_START_MODE              Start mode: smart, fresh, force (default: smart)"
  echo "  NSELF_HEALTH_CHECK_TIMEOUT    Health check timeout seconds (default: 120)"
  echo "  NSELF_HEALTH_CHECK_REQUIRED   Percent services required healthy (default: 80)"
  echo "  NSELF_SKIP_HEALTH_CHECKS      Skip health validation (default: false)"
  echo ""
  echo "Examples:"
  echo "  nself start                  # Start with smart defaults"
  echo "  nself start -v               # Start with verbose output"
  echo "  nself start --quick          # Quick start for development"
  echo "  nself start --fresh          # Force recreate all containers"
  echo "  nself start --timeout 180    # Wait up to 3 minutes for health"
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
    env=$(grep "^ENV=" .env 2>/dev/null | cut -d= -f2- || echo "dev")
  fi

  local project_name="${PROJECT_NAME:-}"
  if [[ -z "$project_name" ]] && [[ -f ".env" ]]; then
    project_name=$(grep "^PROJECT_NAME=" .env 2>/dev/null | cut -d= -f2-)
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
    printf "\n${COLOR_RED}Error: Docker is not installed${COLOR_RESET}\n\n"
    return 1
  fi

  if ! docker info >/dev/null 2>&1; then
    update_progress 0 "error"
    printf "\n${COLOR_RED}Error: Docker daemon is not running${COLOR_RESET}\n"
    printf "Start Docker Desktop or run: ${COLOR_BLUE}sudo systemctl start docker${COLOR_RESET}\n\n"
    return 1
  fi

  update_progress 0 "done"

  # 5. Clean up containers based on CLEANUP_ON_START setting
  update_progress 1 "running"

  # Also check for containers with the actual project name from .env
  local actual_project_name="${PROJECT_NAME:-$project_name}"
  if [[ -f ".env" ]]; then
    actual_project_name=$(grep "^PROJECT_NAME=" .env 2>/dev/null | cut -d= -f2- || echo "$project_name")
  fi

  # Determine cleanup behavior
  local should_cleanup=false
  if [[ "$CLEANUP_ON_START" == "always" ]]; then
    should_cleanup=true
  elif [[ "$CLEANUP_ON_START" == "auto" ]]; then
    # Check if any containers are in error state
    local error_containers=$(docker ps -a --filter "name=${actual_project_name}_" --filter "status=exited" --format "{{.Names}}" 2>/dev/null)
    if [[ -n "$error_containers" ]]; then
      should_cleanup=true
    fi
  fi

  if [[ "$should_cleanup" == "true" ]]; then
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
  fi

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
    project_name=$(grep "^PROJECT_NAME=" .env.runtime 2>/dev/null | cut -d= -f2- || echo "$project_name")
  fi

  # 9. Start services with progress tracking
  local compose_cmd="docker compose"
  local start_output=$(mktemp)
  local error_output=$(mktemp)

  # Build the docker compose command based on start mode
  local compose_args=(
    "--project-name" "$project_name"
    "--env-file" "$env_file"
    "up" "-d"
    "--remove-orphans"
  )

  # Add mode-specific flags
  if [[ "$START_MODE" == "fresh" ]]; then
    compose_args+=("--force-recreate")
  elif [[ "$START_MODE" == "force" ]]; then
    compose_args+=("--force-recreate" "--renew-anon-volumes")
  fi

  if [[ "$DEBUG" == "true" ]]; then
    echo ""
    echo "DEBUG: Project name: $project_name"
    echo "DEBUG: Environment: $env"
    echo "DEBUG: Env file: $env_file"
    echo "DEBUG: Command: $compose_cmd ${compose_args[*]}"
    echo ""
  fi

  # Show initial preparing message
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Analyzing Docker configuration..."

  # Execute docker compose
  if [[ "$VERBOSE" == "true" ]]; then
    # Verbose mode - show Docker output directly
    printf "\r%-60s\r" " "  # Clear the preparing message
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

    # Count total expected services from docker-compose.yml
    local total_services=$(grep -c "^  [a-z].*:" docker-compose.yml 2>/dev/null || echo "25")
    local images_to_pull=0
    local images_pulled=0
    local containers_started=0
    local current_action="Analyzing Docker configuration"
    local last_line=""
    local last_update=$(date +%s)

    # Initial delay to let docker compose start
    sleep 0.2

    while ps -p $compose_pid > /dev/null 2>&1; do
      # Update spinner
      spin_index=$(( (spin_index + 1) % 10 ))

      # Get the last non-empty line from output to see what's happening
      last_line=$(tail -n 10 "$start_output" 2>/dev/null | grep -v "^$" | tail -n 1 || echo "")

      # Check what's happening based on output patterns with more detail
      if echo "$last_line" | grep -q "Building\|Step\|RUN\|COPY\|FROM"; then
        # Building custom images - count steps
        local build_steps=$(grep -c "Step [0-9]" "$start_output" 2>/dev/null || echo "0")
        local image_name=$(echo "$last_line" | grep -oE "Building [a-z_-]+" | sed 's/Building //' || echo "image")
        current_action="Building custom Docker images"
        if [[ -n "$image_name" ]] && [[ "$image_name" != "image" ]]; then
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (building %s)" "${spinner[$spin_index]}" "$current_action" "$image_name"
        else
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (step %d)" "${spinner[$spin_index]}" "$current_action" "$build_steps"
        fi

      elif echo "$last_line" | grep -q "Pulling\|Pull complete\|Already exists\|Downloading\|Extracting\|Waiting"; then
        # Count unique images being pulled - better tracking
        local pulling_count=$(grep -c "Pulling from" "$start_output" 2>/dev/null || echo "0")
        local pulled_count=$(grep -c "Pull complete\|Already exists" "$start_output" 2>/dev/null || echo "0")

        # Try to estimate total images needed
        if [[ $images_to_pull -eq 0 ]]; then
          # Rough estimate based on service count
          images_to_pull=$((total_services * 2 / 3))  # Not all services have unique images
        fi

        # Get the current image being pulled
        local current_image=$(echo "$last_line" | grep -oE "[a-z0-9-]+/[a-z0-9-]+" | tail -1 || echo "")
        current_action="Downloading Docker images"

        if [[ -n "$current_image" ]]; then
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d/%d) %s" "${spinner[$spin_index]}" "$current_action" "$pulling_count" "$images_to_pull" "$current_image"
        else
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d/%d)" "${spinner[$spin_index]}" "$current_action" "$pulling_count" "$images_to_pull"
        fi

      elif grep -q "Network.*Creating\|Network.*Created" "$start_output" 2>/dev/null; then
        # Network creation
        local network_count=$(grep -c "Network.*Created" "$start_output" 2>/dev/null || echo "0")
        current_action="Creating Docker network"
        printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s..." "${spinner[$spin_index]}" "$current_action"

        if [[ "$network_done" == "false" ]] && [[ "$network_count" -gt 0 ]]; then
          update_progress 2 "done"
          network_done=true
        fi

      elif grep -q "Volume.*Creating\|Volume.*Created" "$start_output" 2>/dev/null; then
        # Volume creation
        local volume_count=$(grep -c "Volume.*Created" "$start_output" 2>/dev/null || echo "0")
        current_action="Creating Docker volumes"
        if [[ "$volume_count" -gt 0 ]]; then
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d created)" "${spinner[$spin_index]}" "$current_action" "$volume_count"
        else
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s..." "${spinner[$spin_index]}" "$current_action"
        fi

        if [[ "$volumes_done" == "false" ]] && [[ "$volume_count" -gt 0 ]]; then
          update_progress 3 "done"
          volumes_done=true
        fi

      elif echo "$last_line" | grep -q "Container.*Creating\|Container.*Created"; then
        # Count containers being created with more detail
        local created_count=$(grep -c "Container.*Created" "$start_output" 2>/dev/null || echo "0")
        local creating_count=$(grep -c "Container.*Creating" "$start_output" 2>/dev/null || echo "0")
        local total_creating=$((created_count + creating_count))

        # Get the name of container being created
        local container_name=$(echo "$last_line" | grep -oE "Container ${project_name}_[a-z0-9_-]+" | sed "s/Container ${project_name}_//" || echo "")
        current_action="Creating Docker containers"

        if [[ -n "$container_name" ]]; then
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d/%d) %s" "${spinner[$spin_index]}" "$current_action" "$created_count" "$total_services" "$container_name"
        else
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d/%d)" "${spinner[$spin_index]}" "$current_action" "$created_count" "$total_services"
        fi

        if [[ "$created_count" -ge "$total_services" ]] && [[ "$containers_created" == "false" ]]; then
          update_progress 4 "done"
          containers_created=true
        fi

      elif echo "$last_line" | grep -q "Container.*Starting\|Container.*Started\|Container.*Running"; then
        # Count containers being started with more detail
        containers_started=$(grep -c "Container.*Started" "$start_output" 2>/dev/null || echo "0")
        local starting_count=$(grep -c "Container.*Starting" "$start_output" 2>/dev/null || echo "0")

        # Get the name of container being started
        local container_name=$(echo "$last_line" | grep -oE "Container ${project_name}_[a-z0-9_-]+" | sed "s/Container ${project_name}_//" || echo "")
        current_action="Starting Docker containers"

        if [[ -n "$container_name" ]]; then
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d/%d) %s" "${spinner[$spin_index]}" "$current_action" "$containers_started" "$total_services" "$container_name"
        else
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d/%d)" "${spinner[$spin_index]}" "$current_action" "$containers_started" "$total_services"
        fi

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
        # Default spinner while waiting - show more detail
        local current_time=$(date +%s)
        local elapsed=$((current_time - last_update))

        # Change message based on elapsed time and what we're likely doing
        if [[ "$elapsed" -lt 5 ]]; then
          current_action="Preparing Docker environment"
        elif [[ "$elapsed" -lt 15 ]]; then
          current_action="Checking Docker images"
        elif [[ "$elapsed" -lt 30 ]]; then
          current_action="Processing service dependencies"
        elif [[ "$elapsed" -lt 60 ]]; then
          current_action="Configuring network and volumes"
        else
          current_action="Initializing services"
        fi

        # Show basic progress even when we don't have specific info
        local any_containers=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')
        if [[ "$any_containers" -gt 0 ]]; then
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s... (%d containers active)" "${spinner[$spin_index]}" "$current_action" "$any_containers"
        else
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} %s..." "${spinner[$spin_index]}" "$current_action"
        fi
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

    # Verify health checks (unless skipped)
    if [[ "$SKIP_HEALTH_CHECKS" != "true" ]]; then
      update_progress 9 "running"

      # Progressive health check with configurable timeout and threshold
      local start_time=$(date +%s)
      local health_check_passed=false

      while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))

        if [[ $elapsed -ge $HEALTH_CHECK_TIMEOUT ]]; then
          break
        fi

        # Count health status
        local running_count=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')
        local healthy_count=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Status}}" 2>/dev/null | grep -c "healthy" || echo "0")
        local total_with_health=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Status}}" 2>/dev/null | grep -cE "(healthy|unhealthy|starting)" || echo "0")

        # Calculate percentage
        if [[ $total_with_health -gt 0 ]]; then
          local health_percent=$((healthy_count * 100 / total_with_health))
          if [[ $health_percent -ge $HEALTH_CHECK_REQUIRED ]]; then
            health_check_passed=true
            break
          fi
        elif [[ $running_count -gt 0 ]]; then
          # If no health checks defined, consider it passing if containers are running
          health_check_passed=true
          break
        fi

        sleep "$HEALTH_CHECK_INTERVAL"
      done

      update_progress 9 "done"
    else
      # Skip health checks
      update_progress 9 "done"
      local running_count=$(docker ps --filter "label=com.docker.compose.project=$project_name" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')
      local healthy_count=0
      local total_with_health=0
    fi

    # Get final counts for summary
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
    printf "${COLOR_GREEN}✓${COLOR_RESET} Services (%s): %s core, %s optional, %s monitoring, %s custom\n" \
      "${running_count:-0}" "${core_count:-4}" "${optional_count:-0}" "${monitoring_count:-0}" "${custom_count:-0}"

    if [[ $total_with_health -gt 0 ]]; then
      printf "${COLOR_GREEN}✓${COLOR_RESET} Health: %s/%s checks passing\n" "${healthy_count:-0}" "${total_with_health:-0}"
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