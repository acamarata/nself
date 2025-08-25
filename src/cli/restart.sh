#!/usr/bin/env bash
# restart.sh - Smart restart for services

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
cmd_restart() {
  local smart_mode=true
  local force_all=false
  local services_to_restart=""
  local verbose=false

  # Parse options
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --all | -a)
      force_all=true
      smart_mode=false
      shift
      ;;
    --smart | -s)
      smart_mode=true
      shift
      ;;
    --verbose | -v)
      verbose=true
      shift
      ;;
    --help | -h)
      show_help
      return 0
      ;;
    *)
      # Assume it's a service name
      services_to_restart="$services_to_restart $1"
      shift
      ;;
    esac
  done

  # Load service health utilities
  if [[ -f "$SCRIPT_DIR/../lib/utils/service-health.sh" ]]; then
    source "$SCRIPT_DIR/../lib/utils/service-health.sh"
  fi

  # Check if docker-compose.yml exists
  if [[ ! -f "docker-compose.yml" ]]; then
    log_error "docker-compose.yml not found"
    log_info "Run 'nself build' first to generate infrastructure"
    return 1
  fi

  # Load environment
  if [[ -f ".env.local" ]]; then
    set -a
    load_env_with_priority
    set +a
  fi

  echo

  # If specific services requested, restart only those
  if [[ -n "$services_to_restart" ]]; then
    show_command_header "nself restart" "Restarting specific services"

    for service in $services_to_restart; do
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Restarting $service..."
      if docker compose restart "$service" >/dev/null 2>&1; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Restarted $service                     \n"
      else
        printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to restart $service             \n"
      fi
    done

    echo
    log_success "Service restart completed"
    return 0
  fi

  # Smart mode: Check what has changed
  if [[ "$smart_mode" == "true" ]] && [[ "$force_all" != "true" ]]; then
    # Check if any services are running
    local running_services=$(docker compose ps --services --filter "status=running" 2>/dev/null)

    if [[ -z "$running_services" ]]; then
      # No services running, just start them
      show_command_header "nself restart" "No services running, starting all"

      source "$SCRIPT_DIR/start.sh"
      cmd_start
      return $?
    fi

    # Check if configuration has changed
    if check_config_changed; then
      show_command_header "nself restart" "Smart restart - applying configuration changes"

      # Detect which services need restart
      echo -e "${COLOR_CYAN}➞ Analyzing Changes${COLOR_RESET}"
      echo

      # For now, check if docker-compose.yml or .env.local changed
      # In the future, we can be smarter about which specific services changed
      local compose_changed=false
      local env_changed=false

      # Get modification times
      local compose_modified=$(stat -f %m "docker-compose.yml" 2>/dev/null || stat -c %Y "docker-compose.yml" 2>/dev/null)
      local env_modified=0
      if [[ -f ".env.local" ]]; then
        env_modified=$(stat -f %m ".env.local" 2>/dev/null || stat -c %Y ".env.local" 2>/dev/null)
      fi

      # Get oldest container start time
      local project_name="${PROJECT_NAME:-unity}"
      local oldest_container_start=$(docker ps --format "{{.Names}}" 2>/dev/null | grep "^${project_name}_" | head -1 | xargs docker inspect --format='{{.State.StartedAt}}' 2>/dev/null)

      if [[ -n "$oldest_container_start" ]]; then
        local container_timestamp=$(date -d "$oldest_container_start" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%S" "${oldest_container_start%%.*}" +%s 2>/dev/null)

        if [[ $compose_modified -gt $container_timestamp ]]; then
          compose_changed=true
          echo -e "  ${COLOR_YELLOW}●${COLOR_RESET} docker-compose.yml changed"
        fi

        if [[ $env_modified -gt $container_timestamp ]]; then
          env_changed=true
          echo -e "  ${COLOR_YELLOW}●${COLOR_RESET} .env.local changed"
        fi
      fi

      echo

      # Rebuild if needed
      if [[ "$compose_changed" == "true" ]] || [[ "$env_changed" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Rebuilding configuration..."
        if nself build --force >/dev/null 2>&1; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Configuration rebuilt                     \n"
        else
          printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to rebuild configuration           \n"
          return 1
        fi
      fi

      # Smart restart with minimal downtime
      echo
      echo -e "${COLOR_CYAN}➞ Restarting Services${COLOR_RESET}"
      echo

      # Use docker compose up to handle changes intelligently
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Applying changes with minimal downtime..."

      local output_file=$(mktemp)
      if docker compose up -d --build --remove-orphans 2>&1 >"$output_file"; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Services updated successfully                        \n"

        # Show what was recreated
        local recreated=$(grep "Recreated\|Created" "$output_file" | sed 's/.*Container //' | sed 's/ .*//' | sort -u)
        if [[ -n "$recreated" ]]; then
          echo
          echo -e "${COLOR_CYAN}➞ Updated Services${COLOR_RESET}"
          echo
          for container in $recreated; do
            local service="${container#${project_name}_}"
            echo -e "  ${COLOR_GREEN}↻${COLOR_RESET} $service"
          done
        fi
      else
        printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to update services                           \n"
        if [[ "$verbose" == "true" ]]; then
          echo
          cat "$output_file"
        fi
        rm -f "$output_file"
        return 1
      fi

      rm -f "$output_file"

    else
      # No configuration changes, just restart running services
      show_command_header "nself restart" "Quick restart - no configuration changes"

      echo -e "${COLOR_CYAN}➞ Restarting Services${COLOR_RESET}"
      echo

      # Get all running services
      local services=$(docker compose ps --services --filter "status=running" 2>/dev/null)
      local service_count=$(echo "$services" | wc -l | tr -d ' ')

      printf "${COLOR_BLUE}⠋${COLOR_RESET} Restarting $service_count services..."

      if docker compose restart >/dev/null 2>&1; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Restarted $service_count services                         \n"
      else
        printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to restart services                              \n"
        return 1
      fi
    fi

  else
    # Force restart all (full down + up)
    show_command_header "nself restart" "Full restart - stopping and starting all services"

    # Stop services
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Stopping services..."
    if docker compose down >/dev/null 2>&1; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Services stopped                           \n"
    else
      printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to stop services                    \n"
    fi

    # Small delay
    sleep 2

    # Start services
    echo
    source "$SCRIPT_DIR/start.sh"
    cmd_start
    return $?
  fi

  # Wait for services to be healthy
  echo
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Waiting for services to be healthy..."

  sleep 3

  # Check health
  if check_all_services_healthy false; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} All services healthy                              \n"
    echo

    # Display service status
    display_running_services
    echo

    # Show service URLs
    show_service_urls
  else
    printf "\r${COLOR_YELLOW}⚠${COLOR_RESET}  Some services may still be starting              \n"
    echo
    log_info "Run 'nself status' to check service health"
  fi

  echo
  log_success "Restart completed successfully"
  return 0
}

# Show service URLs (same as in start.sh)
show_service_urls() {
  echo -e "${COLOR_CYAN}➞ Service URLs${COLOR_RESET}"
  echo

  # Load environment if available
  if [[ -f ".env.local" ]]; then
    set -a
    load_env_with_priority
    set +a
  fi

  # Get base domain or use default
  local base_domain="${BASE_DOMAIN:-localhost}"

  # Check which services are actually running (remove unity_ prefix)
  local running_services=$(docker ps --format "table {{.Names}}" | grep "^unity_" | sed 's/^unity_//' 2>/dev/null)

  # Track if any URLs were shown
  local urls_shown=false

  # Hasura GraphQL
  if echo "$running_services" | grep -q "hasura"; then
    local hasura_port=$(docker port unity_hasura 8080 2>/dev/null | cut -d: -f2)
    if [[ -n "$hasura_port" ]]; then
      echo -e "${COLOR_GREEN}✓${COLOR_RESET} GraphQL:    ${COLOR_BLUE}http://localhost:$hasura_port/console${COLOR_RESET}"
      urls_shown=true
    fi
  fi

  # PostgreSQL Database
  if echo "$running_services" | grep -q "postgres"; then
    local pg_port=$(docker port unity_postgres 5432 2>/dev/null | cut -d: -f2)
    if [[ -z "$pg_port" ]]; then
      pg_port="${POSTGRES_PORT:-5432}"
    fi
    echo -e "${COLOR_GREEN}✓${COLOR_RESET} Database:   ${COLOR_BLUE}postgresql://localhost:$pg_port/postgres${COLOR_RESET}"
    urls_shown=true
  fi

  # Redis
  if echo "$running_services" | grep -q "redis"; then
    local redis_port=$(docker port unity_redis 6379 2>/dev/null | cut -d: -f2)
    if [[ -z "$redis_port" ]]; then
      redis_port="${REDIS_PORT:-6379}"
    fi
    echo -e "${COLOR_GREEN}✓${COLOR_RESET} Redis:      ${COLOR_BLUE}redis://localhost:$redis_port${COLOR_RESET}"
    urls_shown=true
  fi

  # If no URLs shown, indicate no exposed services
  if [[ "$urls_shown" != "true" ]]; then
    echo -e "${COLOR_DIM}No services with exposed ports found${COLOR_RESET}"
  fi
}

# Show help
show_help() {
  echo "Usage: nself restart [OPTIONS] [SERVICES...]"
  echo ""
  echo "Smart restart for services with minimal downtime"
  echo ""
  echo "Options:"
  echo "  -s, --smart        Smart mode - only restart changed services (default)"
  echo "  -a, --all          Force full restart (stop + start)"
  echo "  -v, --verbose      Show detailed output"
  echo "  -h, --help         Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself restart                  # Smart restart (detects changes)"
  echo "  nself restart postgres          # Restart only postgres"
  echo "  nself restart postgres redis    # Restart specific services"
  echo "  nself restart --all             # Full restart all services"
  echo ""
  echo "Smart restart will:"
  echo "  • Detect configuration changes"
  echo "  • Rebuild if needed"
  echo "  • Apply changes with minimal downtime"
  echo "  • Only restart affected services"
}

# Export for use as library
export -f cmd_restart

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "restart" || exit $?
  cmd_restart "$@"
  exit_code=$?
  post_command "restart" $exit_code
  exit $exit_code
fi
