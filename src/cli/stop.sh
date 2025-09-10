#!/usr/bin/env bash
set -euo pipefail

# stop.sh - Stop all services with enhanced feedback

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Load environment with smart defaults
if [[ -f "$SCRIPT_DIR/../lib/config/smart-defaults.sh" ]]; then
  source "$SCRIPT_DIR/../lib/config/smart-defaults.sh"
  load_env_with_defaults >/dev/null 2>&1
fi

# Command function
cmd_stop() {
  local remove_volumes=false
  local remove_images=false
  local remove_orphans=false
  local verbose=false
  local services_to_stop=""

  # Parse options
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --volumes | -v)
      remove_volumes=true
      shift
      ;;
    --rmi | --remove-images)
      remove_images=true
      shift
      ;;
    --remove-orphans)
      remove_orphans=true
      shift
      ;;
    --verbose)
      verbose=true
      shift
      ;;
    --help | -h)
      show_down_help
      return 0
      ;;
    -*)
      log_error "Unknown option: $1"
      show_down_help
      return 1
      ;;
    *)
      # Assume it's a service name
      services_to_stop="$services_to_stop $1"
      shift
      ;;
    esac
  done

  # Load service health utilities if available
  if [[ -f "$SCRIPT_DIR/../lib/utils/service-health.sh" ]]; then
    source "$SCRIPT_DIR/../lib/utils/service-health.sh"
  fi

  # Check if docker-compose.yml exists
  if [[ ! -f "docker-compose.yml" ]]; then
    log_error "docker-compose.yml not found"
    log_info "No services to stop"
    return 0
  fi

  # Load environment
  if [[ -f ".env" ]] || [[ -f ".env.dev" ]]; then
    set -a
    load_env_with_priority
    set +a
  fi

  # Get project name
  local project_name="${PROJECT_NAME:-unity}"

  # If specific services requested, stop only those
  if [[ -n "$services_to_stop" ]]; then
    show_command_header "nself stop" "Stop services and containers"
    echo -e "${COLOR_CYAN}→${COLOR_RESET} Stopping specific services"
    echo

    for service in $services_to_stop; do
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Stopping $service..."
      if docker compose stop "$service" >/dev/null 2>&1; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Stopped $service                       \n"

        # Remove the container if requested
        if [[ "$remove_volumes" == true ]] || [[ "$remove_images" == true ]]; then
          docker compose rm -f "$service" >/dev/null 2>&1
        fi
      else
        printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to stop $service                \n"
      fi
    done

    echo
    log_success "Service shutdown completed"
    return 0
  fi

  # Show header first (no extra echo before it)
  show_command_header "nself stop" "Stop services and containers"
  
  # Stop health monitoring daemon if running
  if [[ -f "$SCRIPT_DIR/../lib/auto-fix/health-check-daemon.sh" ]]; then
    source "$SCRIPT_DIR/../lib/auto-fix/health-check-daemon.sh"
    if is_daemon_running 2>/dev/null; then
      stop_health_daemon >/dev/null 2>&1
      log_info "Stopped health monitoring daemon"
    fi
  fi

  # Check what's currently running using docker ps directly
  local running_containers=$(docker ps --filter "name=^${project_name}_" --format "{{.Names}}" 2>/dev/null)
  local running_count=0

  if [[ -n "$running_containers" ]]; then
    running_count=$(echo "$running_containers" | wc -l | tr -d ' ')
  fi

  if [[ $running_count -eq 0 ]]; then
    echo -e "${COLOR_GREEN}✓${COLOR_RESET} No services are currently running"
    echo

    # Check for stopped containers
    local stopped_containers=$(docker ps -a --filter "name=^${project_name}_" --format "{{.Names}}" 2>/dev/null)

    if [[ -n "$stopped_containers" ]]; then
      local stopped_count=$(echo "$stopped_containers" | wc -l | tr -d ' ')
      echo -e "${COLOR_YELLOW}⚠${COLOR_RESET}  Found $stopped_count stopped containers"
      echo

      if [[ "$remove_volumes" == true ]] || [[ "$remove_images" == true ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning up..."
        compose down $(if [[ "$remove_volumes" == true ]]; then echo "-v"; fi) $(if [[ "$remove_images" == true ]]; then echo "--rmi all"; fi) >/dev/null 2>&1
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Cleanup completed                      \n"
      else
        echo -e "   Run ${COLOR_BLUE}nself stop --volumes${COLOR_RESET} to remove all data"
        echo -e "   Run ${COLOR_BLUE}docker system prune${COLOR_RESET} to clean up Docker"
      fi
    fi

    echo
    return 0
  fi

  # Show operation type
  if [[ "$remove_volumes" == true ]]; then
    echo -e "${COLOR_CYAN}→${COLOR_RESET} Stopping services and removing data ($running_count running)"
  elif [[ "$remove_images" == true ]]; then
    echo -e "${COLOR_CYAN}→${COLOR_RESET} Stopping services and removing images ($running_count running)"
  else
    echo -e "${COLOR_CYAN}→${COLOR_RESET} Stopping all services ($running_count running)"
  fi
  echo

  # Ensure Docker is running
  if ! docker info >/dev/null 2>&1; then
    log_error "Docker is not running"
    return 1
  fi

  # Build the compose down command arguments
  local compose_args="down"

  if [[ "$remove_volumes" == true ]]; then
    compose_args="$compose_args -v"
  fi

  if [[ "$remove_images" == true ]]; then
    compose_args="$compose_args --rmi all"
  fi

  if [[ "$remove_orphans" == true ]]; then
    compose_args="$compose_args --remove-orphans"
  fi

  # Execute the shutdown
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Shutting down services..."

  local output_file=$(mktemp)

  if [[ "$verbose" == true ]]; then
    # Show full output in verbose mode
    printf "\n"
    compose $compose_args 2>&1 | tee "$output_file"
    result=${PIPESTATUS[0]}
  else
    # Run silently with spinner
    (compose $compose_args 2>&1) >"$output_file" &
    local compose_pid=$!

    # Show spinner while waiting
    local spin_chars="⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
    local i=0
    while kill -0 $compose_pid 2>/dev/null; do
      local char="${spin_chars:$((i % ${#spin_chars})):1}"
      printf "\r${COLOR_BLUE}%s${COLOR_RESET} Shutting down services..." "$char"
      ((i++))
      sleep 0.1
    done
    wait $compose_pid
    result=$?
  fi

  if [[ $result -eq 0 ]]; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} All services stopped                              \n"

    # Show additional cleanup info
    if [[ "$remove_volumes" == true ]]; then
      echo
      # Count what was removed
      local removed_volumes=$(grep -c "Volume.*removed" "$output_file" 2>/dev/null | tr -d '\n' || echo "0")
      local removed_networks=$(grep -c "Network.*removed" "$output_file" 2>/dev/null | tr -d '\n' || echo "0")

      if [[ $removed_volumes -gt 0 ]]; then
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} Removed $removed_volumes volumes"
      fi

      if [[ $removed_networks -gt 0 ]]; then
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} Removed $removed_networks networks"
      fi

      echo -e "${COLOR_YELLOW}⚠${COLOR_RESET}  All persistent data has been removed"
    fi

    if [[ "$remove_images" == true ]]; then
      local removed_images=$(grep -c "Image.*deleted" "$output_file" 2>/dev/null || echo "0")
      if [[ $removed_images -gt 0 ]]; then
        echo
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} Removed $removed_images images"
      fi
    fi

    # Clean up orphaned containers if any
    local orphaned=$(docker ps -a --filter "name=^${project_name}_" --format "{{.Names}}" 2>/dev/null)
    if [[ -n "$orphaned" ]]; then
      echo
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning up orphaned containers..."
      docker rm -f $orphaned >/dev/null 2>&1
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Orphaned containers removed                    \n"
    fi

    echo
    log_success "All services stopped"

    # Show next steps
    echo
    echo "nself start | nself status | nself clean"
  else
    printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to stop services                           \n"

    # Show error details
    if [[ "$verbose" != true ]]; then
      echo
      local error_lines=$(grep -E "error|ERROR|failed|Failed" "$output_file" | head -5)
      if [[ -n "$error_lines" ]]; then
        echo "$error_lines"
        echo
      fi
      log_info "Run with --verbose for more details"
    fi

    rm -f "$output_file"
    return 1
  fi

  rm -f "$output_file"
  return 0
}

# Show help
show_down_help() {
  echo "Usage: nself stop [OPTIONS] [SERVICES...]"
  echo ""
  echo "Stop services and optionally remove containers, volumes, and images"
  echo ""
  echo "Options:"
  echo "  -v, --volumes        Remove volumes (WARNING: deletes all data)"
  echo "  --rmi                Remove images"
  echo "  --remove-orphans     Remove containers for services not in compose file"
  echo "  --verbose            Show detailed output"
  echo "  -h, --help           Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself stop                    # Stop all services, keep data"
  echo "  nself stop postgres           # Stop only postgres"
  echo "  nself stop --volumes          # Stop and remove all data"
  echo "  nself stop --rmi              # Stop and remove images"
  echo "  nself stop --volumes --rmi    # Full cleanup"
  echo ""
  echo "Note: Data in volumes is preserved unless --volumes is specified"
}

# Export for use as library
export -f cmd_stop

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "stop" || exit $?
  cmd_stop "$@"
  exit_code=$?
  post_command "stop" $exit_code
  exit $exit_code
fi
