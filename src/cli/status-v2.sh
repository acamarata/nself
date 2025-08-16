#!/usr/bin/env bash

# status-v2.sh - Enhanced service status with clean table output

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Service definitions with URLs
declare -A SERVICE_URLS=(
  ["nginx"]="https://localhost"
  ["postgres"]="postgresql://localhost:5432"
  ["hasura"]="http://localhost:8080/console"
  ["auth"]="http://localhost:3000"
  ["storage"]="http://localhost:3001"
  ["redis"]="redis://localhost:6379"
  ["mailhog"]="http://localhost:8025"
)

# Get container status
get_container_info() {
  local container_name="$1"
  local project="${PROJECT_NAME:-nself}"
  
  # Try different naming patterns
  local patterns=(
    "${project}-${container_name}-1"
    "${project}_${container_name}_1"
    "${container_name}"
  )
  
  for pattern in "${patterns[@]}"; do
    if docker ps -a --format "{{.Names}}" | grep -q "^${pattern}$"; then
      local status=$(docker ps -a --filter "name=${pattern}" --format "{{.Status}}")
      local state=$(docker ps -a --filter "name=${pattern}" --format "{{.State}}")
      local ports=$(docker ps -a --filter "name=${pattern}" --format "{{.Ports}}" | sed 's/,/, /g')
      
      echo "${state}|${status}|${ports}"
      return 0
    fi
  done
  
  echo "not_found||"
  return 1
}

# Format uptime from status string
format_uptime() {
  local status="$1"
  
  if [[ "$status" =~ Up[[:space:]]+(About[[:space:]]+)?([0-9]+)[[:space:]]+(second|minute|hour|day) ]]; then
    echo "${BASH_REMATCH[2]} ${BASH_REMATCH[3]}s"
  elif [[ "$status" =~ Up[[:space:]]+(.*) ]]; then
    echo "${BASH_REMATCH[1]}"
  elif [[ "$status" =~ Exited ]]; then
    echo "Stopped"
  else
    echo "-"
  fi
}

# Get service health
get_health_status() {
  local container_name="$1"
  local state="$2"
  
  if [[ "$state" == "running" ]]; then
    # Check if container is healthy
    local project="${PROJECT_NAME:-nself}"
    local health=$(docker inspect "${project}-${container_name}-1" 2>/dev/null | grep -o '"Status":"[^"]*"' | cut -d'"' -f4 | head -1)
    
    case "$health" in
      healthy) echo "${COLOR_GREEN}✓ Healthy${COLOR_RESET}" ;;
      unhealthy) echo "${COLOR_RED}✗ Unhealthy${COLOR_RESET}" ;;
      starting) echo "${COLOR_YELLOW}⟳ Starting${COLOR_RESET}" ;;
      *) echo "${COLOR_GREEN}● Running${COLOR_RESET}" ;;
    esac
  elif [[ "$state" == "exited" ]]; then
    echo "${COLOR_RED}○ Stopped${COLOR_RESET}"
  elif [[ "$state" == "not_found" ]]; then
    echo "${COLOR_DIM}─ Not deployed${COLOR_RESET}"
  else
    echo "${COLOR_YELLOW}? Unknown${COLOR_RESET}"
  fi
}

# Display service table
display_service_table() {
  # Header
  printf "\n"
  printf "%-15s %-15s %-12s %-30s %-10s\n" "Service" "Status" "Uptime" "URL/Endpoint" "Ports"
  printf "%-15s %-15s %-12s %-30s %-10s\n" "───────" "──────" "──────" "────────────" "─────"
  
  local services=(nginx postgres hasura auth storage redis mailhog)
  local all_running=true
  local any_running=false
  
  for service in "${services[@]}"; do
    IFS='|' read -r state status ports <<< "$(get_container_info "$service")"
    
    if [[ "$state" == "running" ]]; then
      any_running=true
    else
      all_running=false
    fi
    
    local health=$(get_health_status "$service" "$state")
    local uptime=$(format_uptime "$status")
    local url="${SERVICE_URLS[$service]:-}"
    
    # Format ports for display
    if [[ -n "$ports" ]]; then
      ports=$(echo "$ports" | sed 's/0.0.0.0://g' | sed 's/->/ /g' | cut -d' ' -f1 | head -c 10)
    else
      ports="-"
    fi
    
    printf "%-15s %-25s %-12s %-30s %-10s\n" \
      "$service" "$health" "$uptime" "$url" "$ports"
  done
  
  echo ""
}

# Display resource usage
display_resource_usage() {
  show_section "Resource Usage"
  
  # Get Docker stats
  local stats=$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null | grep -E "nself|postgres|hasura|nginx|redis" || true)
  
  if [[ -n "$stats" ]]; then
    echo "$stats" | while IFS=$'\t' read -r container cpu mem; do
      if [[ "$container" != "CONTAINER" ]]; then
        printf "  %-20s CPU: %-8s Memory: %s\n" "$container" "$cpu" "$mem"
      fi
    done
  else
    echo "  No containers running"
  fi
  
  echo ""
}

# Display quick actions
display_quick_actions() {
  show_section "Quick Actions"
  
  echo "  ${COLOR_BLUE}nself start${COLOR_RESET}    - Start all services"
  echo "  ${COLOR_BLUE}nself stop${COLOR_RESET}     - Stop all services"
  echo "  ${COLOR_BLUE}nself restart${COLOR_RESET}  - Restart all services"
  echo "  ${COLOR_BLUE}nself logs${COLOR_RESET}     - View service logs"
  echo "  ${COLOR_BLUE}nself doctor${COLOR_RESET}   - Run system diagnostics"
  echo ""
}

# Check critical services
check_critical_services() {
  local critical=(postgres nginx)
  local all_critical_running=true
  
  for service in "${critical[@]}"; do
    IFS='|' read -r state status ports <<< "$(get_container_info "$service")"
    if [[ "$state" != "running" ]]; then
      all_critical_running=false
      break
    fi
  done
  
  if [[ "$all_critical_running" == "true" ]]; then
    return 0
  else
    return 1
  fi
}

# Main command function
cmd_status() {
  local verbose="${1:-}"
  
  if [[ "$verbose" == "-h" ]] || [[ "$verbose" == "--help" ]]; then
    echo "Usage: nself status [OPTIONS]"
    echo ""
    echo "Display status of all nself services"
    echo ""
    echo "Options:"
    echo "  -v, --verbose    Show detailed resource usage"
    echo "  -h, --help       Show this help message"
    echo ""
    echo "Examples:"
    echo "  nself status           # Show service status"
    echo "  nself status -v        # Show status with resource usage"
    return 0
  fi
  
  show_command_header "nself status" "Service health and status overview"
  
  # Main status table
  display_service_table
  
  # Show resource usage if verbose
  if [[ "$verbose" == "-v" ]] || [[ "$verbose" == "--verbose" ]]; then
    display_resource_usage
  fi
  
  # Quick actions
  display_quick_actions
  
  # Overall status summary
  if check_critical_services; then
    log_success "Core services are operational"
  else
    log_warning "Some services are not running. Run 'nself start' to start services."
  fi
  
  echo ""
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "status" || exit $?
  cmd_status "$@"
  exit_code=$?
  post_command "status" $exit_code
  exit $exit_code
fi