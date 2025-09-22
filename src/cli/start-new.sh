#!/usr/bin/env bash
# start.sh - Modular, maintainable start command for nself
# Bash 3.2 compatible, cross-platform
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source core utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/progress.sh"
source "$SCRIPT_DIR/../lib/utils/hosts.sh"

# Source modular start components
source "$SCRIPT_DIR/../lib/start/pre-checks.sh"
source "$SCRIPT_DIR/../lib/start/docker-compose.sh"
source "$SCRIPT_DIR/../lib/start/port-manager.sh"

# Parse command line options
parse_options() {
  VERBOSE=false
  DETACHED=true
  AUTO_FIX=true
  SKIP_CHECKS=false
  FORCE_REBUILD=false

  while [ $# -gt 0 ]; do
    case "$1" in
      --verbose|-v)
        VERBOSE=true
        shift
        ;;
      --attach|-a)
        DETACHED=false
        shift
        ;;
      --no-auto-fix)
        AUTO_FIX=false
        shift
        ;;
      --skip-checks)
        SKIP_CHECKS=true
        shift
        ;;
      --rebuild)
        FORCE_REBUILD=true
        shift
        ;;
      --help|-h)
        show_help
        exit 0
        ;;
      *)
        log_error "Unknown option: $1"
        show_help
        exit 1
        ;;
    esac
  done
}

# Show help message
show_help() {
  cat << EOF
nself start - Start all services and infrastructure

Usage: nself start [OPTIONS]

Options:
  -v, --verbose       Show detailed output
  -a, --attach        Run in foreground (not detached)
  --no-auto-fix       Disable automatic port conflict resolution
  --skip-checks       Skip pre-flight checks
  --rebuild           Force rebuild of all containers
  -h, --help          Show this help message

Examples:
  nself start                    # Start all services
  nself start --verbose          # Start with detailed output
  nself start --attach           # Start in foreground
  nself start --rebuild          # Rebuild and start all services

EOF
}

# Main start function
main() {
  # Parse options
  parse_options "$@"

  # Show header
  display_header "nself start" "Start all services and infrastructure"

  # 1. Pre-flight checks
  if [ "$SKIP_CHECKS" = "false" ]; then
    if ! run_pre_checks "$VERBOSE"; then
      log_error "Pre-flight checks failed"
      exit 1
    fi
  fi

  # 2. Load environment
  local env_file=$(check_env_files)
  if [ -f "$env_file" ]; then
    set -a
    source "$env_file"
    set +a
    [ "$VERBOSE" = "true" ] && log_info "Loaded environment from $env_file"
  fi

  # 3. Check and update hosts file
  ensure_hosts_entries "${BASE_DOMAIN:-localhost}" "${PROJECT_NAME:-nself}"

  # 4. Handle port conflicts
  if [ "$AUTO_FIX" = "true" ]; then
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Checking for port conflicts..."

    local port_result=$(auto_resolve_ports "$env_file")
    if [ "$port_result" = "ports_updated" ]; then
      printf "\r${COLOR_YELLOW}⚡${COLOR_RESET} Resolved port conflicts         \n"
      # Reload environment with new ports
      set -a
      source "$env_file"
      set +a
    else
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} No port conflicts               \n"
    fi
  fi

  # 5. Check if already running
  local running_count=$(check_existing_services)
  if [ $running_count -gt 0 ]; then
    echo ""
    echo -e "${COLOR_GREEN}✓${COLOR_RESET} $running_count services are already running"
    echo ""
    echo -e "Use ${COLOR_BLUE}nself status${COLOR_RESET} to view service status"
    echo -e "Use ${COLOR_BLUE}nself restart${COLOR_RESET} to restart services"
    echo -e "Use ${COLOR_BLUE}nself stop${COLOR_RESET} to stop services"
    echo ""
    exit 0
  fi

  # 6. Create docker network
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Creating Docker network..."
  local network_name="${PROJECT_NAME:-nself}_network"
  if docker network create "$network_name" >/dev/null 2>&1; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Created Docker network          \n"
  else
    printf "\r${COLOR_DIM}○${COLOR_RESET} Docker network exists           \n"
  fi

  # 7. Build and start services
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Starting services..."

  # Determine build option
  local build_opt="true"
  if [ "$FORCE_REBUILD" = "true" ]; then
    build_opt="true"
  fi

  # Build compose command
  local compose_cmd=$(build_compose_command "$env_file" "${PROJECT_NAME:-nself}" "$DETACHED" "$build_opt")

  # Execute with progress
  if execute_compose_with_progress "$compose_cmd" "${PROJECT_NAME:-nself}" 600 "$VERBOSE"; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} All services started            \n"
  else
    printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to start services        \n"
    echo ""
    log_error "Check logs with: nself logs"
    exit 1
  fi

  # 8. Wait for services to be healthy
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Waiting for services to be healthy..."
  if check_services_health "${PROJECT_NAME:-nself}" 60; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} All services healthy            \n"
  else
    printf "\r${COLOR_YELLOW}⚠${COLOR_RESET}  Some services may be unhealthy \n"
  fi

  # 9. Show service URLs
  echo ""
  show_service_urls
  echo ""

  # 10. Show next steps
  echo -e "${COLOR_CYAN}➞ Next Steps${COLOR_RESET}"
  echo ""
  echo -e "  ${COLOR_BLUE}nself status${COLOR_RESET}  - View service status"
  echo -e "  ${COLOR_BLUE}nself logs${COLOR_RESET}    - View service logs"
  echo -e "  ${COLOR_BLUE}nself stop${COLOR_RESET}    - Stop all services"
  echo ""
}

# Show service URLs
show_service_urls() {
  local base_domain="${BASE_DOMAIN:-localhost}"
  local protocol="http"
  [ "${SSL_ENABLED:-false}" = "true" ] && protocol="https"

  echo -e "${COLOR_CYAN}➞ Service URLs${COLOR_RESET}"
  echo ""

  # Core services
  if [ "${NGINX_ENABLED:-true}" = "true" ]; then
    echo -e "  Application:    $protocol://$base_domain"
  fi

  if [ "${HASURA_ENABLED:-false}" = "true" ]; then
    echo -e "  GraphQL API:    $protocol://api.$base_domain"
    echo -e "   - Console:     $protocol://api.$base_domain/console"
  fi

  if [ "${AUTH_ENABLED:-false}" = "true" ]; then
    echo -e "  Auth:           $protocol://auth.$base_domain"
  fi

  if [ "${STORAGE_ENABLED:-false}" = "true" ]; then
    echo -e "  Storage:        $protocol://storage.$base_domain"
  fi

  # Admin interfaces
  if [ "${MAILPIT_ENABLED:-true}" = "true" ]; then
    echo -e "  Mail UI:        http://localhost:${MAILPIT_UI_PORT:-8025}"
  fi

  if [ "${MINIO_ENABLED:-false}" = "true" ]; then
    echo -e "  MinIO Console:  http://localhost:${MINIO_CONSOLE_PORT:-9001}"
  fi

  if [ "${GRAFANA_ENABLED:-false}" = "true" ]; then
    echo -e "  Grafana:        http://localhost:${GRAFANA_PORT:-3000}"
  fi
}

# Run main function
main "$@"