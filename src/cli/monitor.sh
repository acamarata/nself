#!/usr/bin/env bash
# monitor.sh - Real-time monitoring dashboard

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/monitoring/dashboard.sh"

# Command function
cmd_monitor() {
  local mode="${1:-dashboard}"
  
  # Check for help
  if [[ "$mode" == "--help" ]] || [[ "$mode" == "-h" ]]; then
    show_monitor_help
    return 0
  fi
  
  case "$mode" in
    dashboard)
      # Full dashboard mode
      run_monitoring_dashboard
      ;;
    services)
      # Monitor services only
      monitor_services_live
      ;;
    resources)
      # Monitor resources only
      monitor_resources_live
      ;;
    logs)
      # Monitor logs in real-time
      monitor_logs_live
      ;;
    alerts)
      # Monitor alerts
      monitor_alerts_live
      ;;
    *)
      log_error "Unknown monitor mode: $mode"
      show_monitor_help
      return 1
      ;;
  esac
}

# Show help
show_monitor_help() {
  echo "nself monitor - Real-time monitoring dashboard"
  echo ""
  echo "Usage: nself monitor [mode] [options]"
  echo ""
  echo "Modes:"
  echo "  dashboard    Full monitoring dashboard (default)"
  echo "  services     Monitor service health"
  echo "  resources    Monitor resource usage"
  echo "  logs         Monitor logs in real-time"
  echo "  alerts       Monitor active alerts"
  echo ""
  echo "Options:"
  echo "  --interval <sec>   Refresh interval (default: 2)"
  echo "  --no-color         Disable colors"
  echo "  -h, --help         Show this help message"
  echo ""
  echo "Keyboard Controls:"
  echo "  q/Q         Quit"
  echo "  r/R         Refresh immediately"
  echo "  s           Switch to services view"
  echo "  c           Switch to resources (CPU/memory) view"
  echo "  l           Switch to logs view"
  echo "  a           Switch to alerts view"
  echo "  ↑/↓         Scroll (in logs view)"
  echo "  Space       Pause/resume auto-refresh"
  echo ""
  echo "Examples:"
  echo "  nself monitor"
  echo "  nself monitor services"
  echo "  nself monitor resources --interval 5"
}

# Run full monitoring dashboard
run_monitoring_dashboard() {
  # Clear screen and hide cursor
  clear
  tput civis 2>/dev/null || true
  
  # Trap to restore cursor on exit
  trap 'tput cnorm 2>/dev/null || true; clear' EXIT INT TERM
  
  local refresh_interval="${MONITOR_INTERVAL:-2}"
  local paused=false
  local view="dashboard"
  
  while true; do
    # Clear screen for refresh
    clear
    
    # Show header
    show_dashboard_header "$view" "$paused"
    
    # Show appropriate view
    case "$view" in
      dashboard)
        show_dashboard_view
        ;;
      services)
        show_services_view
        ;;
      resources)
        show_resources_view
        ;;
      logs)
        show_logs_view
        ;;
      alerts)
        show_alerts_view
        ;;
    esac
    
    # Show footer with controls
    show_dashboard_footer
    
    # Handle input with timeout
    if read -t "$refresh_interval" -n 1 key; then
      case "$key" in
        q|Q)
          break
          ;;
        r|R)
          continue
          ;;
        s)
          view="services"
          ;;
        c)
          view="resources"
          ;;
        l)
          view="logs"
          ;;
        a)
          view="alerts"
          ;;
        " ")
          paused=$([[ "$paused" == "true" ]] && echo "false" || echo "true")
          ;;
      esac
    fi
    
    # Skip refresh if paused
    if [[ "$paused" == "true" ]]; then
      sleep 0.1
      continue
    fi
  done
}

# Show dashboard header
show_dashboard_header() {
  local view="$1"
  local paused="$2"
  
  local status_indicator="[0;32m●[0m"
  if [[ "$paused" == "true" ]]; then
    status_indicator="[0;33m⏸[0m"
  fi
  
  echo -e "[0;36m╔══════════════════════════════════════════════════════════════════════════════╗[0m"
  echo -e "[0;36m║[0m  [1mnself monitor[0m - $view view  $status_indicator $(date '+%Y-%m-%d %H:%M:%S')                    [0;36m║[0m"
  echo -e "[0;36m╚══════════════════════════════════════════════════════════════════════════════╝[0m"
  echo
}

# Show dashboard view
show_dashboard_view() {
  # Services summary
  echo -e "[0;36m▶ Services[0m"
  local services=$(docker compose ps --format "table {{.Service}}\t{{.Status}}" 2>/dev/null | tail -n +2)
  local running=$(echo "$services" | grep -c "Up" || echo "0")
  local total=$(echo "$services" | wc -l | xargs)
  echo "  Status: $running/$total services running"
  
  # Quick service list
  echo "$services" | head -5 | while IFS=$'\t' read -r service status; do
    local indicator="[0;31m✗[0m"
    if [[ "$status" == *"Up"* ]]; then
      indicator="[0;32m✓[0m"
    fi
    printf "  $indicator %-20s %s\n" "$service" "$status"
  done
  
  echo
  
  # Resources summary
  echo -e "[0;36m▶ Resources[0m"
  local cpu_usage=$(docker stats --no-stream --format "{{.CPUPerc}}" 2>/dev/null | sed 's/%//' | awk '{sum+=$1} END {printf "%.1f", sum}')
  local mem_usage=$(docker stats --no-stream --format "{{.MemUsage}}" 2>/dev/null | head -1)
  echo "  CPU Total: ${cpu_usage}%"
  echo "  Memory: $mem_usage"
  
  echo
  
  # Top consumers
  echo -e "[0;36m▶ Top Consumers[0m"
  docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemPerc}}" 2>/dev/null | head -6 | tail -n +2 | while IFS=$'\t' read -r name cpu mem; do
    printf "  %-30s CPU: %6s  MEM: %6s\n" "$name" "$cpu" "$mem"
  done
  
  echo
  
  # Recent alerts
  echo -e "[0;36m▶ Recent Alerts[0m"
  if [[ -f "/tmp/nself-alerts.log" ]]; then
    tail -3 /tmp/nself-alerts.log | while read -r line; do
      echo "  $line"
    done
  else
    echo "  No recent alerts"
  fi
}

# Show services view
show_services_view() {
  echo -e "[0;36m▶ Service Health[0m"
  echo
  
  docker compose ps --format "table {{.Service}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null | while IFS=$'\t' read -r service status ports; do
    if [[ "$service" == "Service" ]]; then
      printf "  %-20s %-30s %s\n" "SERVICE" "STATUS" "PORTS"
      printf "  %-20s %-30s %s\n" "-------" "------" "-----"
    else
      local indicator="[0;31m✗[0m"
      local health=""
      if [[ "$status" == *"Up"* ]]; then
        indicator="[0;32m✓[0m"
        if [[ "$status" == *"healthy"* ]]; then
          health="[0;32m[healthy][0m"
        elif [[ "$status" == *"unhealthy"* ]]; then
          health="[0;31m[unhealthy][0m"
          indicator="[0;33m⚠[0m"
        fi
      fi
      printf "  $indicator %-18s %-30s %s\n" "$service" "$status $health" "$ports"
    fi
  done
  
  echo
  echo -e "[0;36m▶ Container Restart Count[0m"
  docker compose ps --format "{{.Service}}" 2>/dev/null | while read -r service; do
    local restarts=$(docker inspect "$(docker compose ps -q "$service" 2>/dev/null)" --format='{{.RestartCount}}' 2>/dev/null || echo "0")
    if [[ "$restarts" -gt 0 ]]; then
      printf "  %-20s %d restarts\n" "$service" "$restarts"
    fi
  done
}

# Show resources view
show_resources_view() {
  echo -e "[0;36m▶ Resource Usage[0m"
  echo
  
  docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}" 2>/dev/null
  
  echo
  echo -e "[0;36m▶ System Resources[0m"
  
  # CPU info
  if [[ "$(uname)" == "Darwin" ]]; then
    local cpu_usage=$(top -l 1 | grep "CPU usage" | awk '{print $3}' | sed 's/%//')
    echo "  System CPU: ${cpu_usage}%"
  else
    local cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | sed 's/%us,//')
    echo "  System CPU: ${cpu_usage}%"
  fi
  
  # Memory info
  if [[ "$(uname)" == "Darwin" ]]; then
    local mem_info=$(top -l 1 | grep "PhysMem")
    echo "  System Memory: $mem_info"
  else
    local mem_info=$(free -h | grep "^Mem" | awk '{printf "%s / %s (%s used)", $3, $2, $3}')
    echo "  System Memory: $mem_info"
  fi
  
  # Disk info
  local disk_info=$(df -h . | tail -1 | awk '{printf "%s / %s (%s used)", $3, $2, $5}')
  echo "  Disk Usage: $disk_info"
}

# Show logs view
show_logs_view() {
  echo -e "[0;36m▶ Recent Logs[0m"
  echo
  
  # Show last 20 lines from all services
  docker compose logs --tail=20 --timestamps 2>/dev/null | tail -20
}

# Show alerts view
show_alerts_view() {
  echo -e "[0;36m▶ Active Alerts[0m"
  echo
  
  # Check for unhealthy services
  local unhealthy=$(docker compose ps --format "{{.Service}}\t{{.Status}}" 2>/dev/null | grep -E "unhealthy|Exit|Restarting" || true)
  if [[ -n "$unhealthy" ]]; then
    echo -e "  [0;31m⚠ Unhealthy Services:[0m"
    echo "$unhealthy" | while IFS=$'\t' read -r service status; do
      echo "    - $service: $status"
    done
    echo
  fi
  
  # Check disk space
  local disk_usage=$(df -h . | tail -1 | awk '{print $5}' | sed 's/%//')
  if [[ "$disk_usage" -gt 80 ]]; then
    echo -e "  [0;33m⚠ High Disk Usage: ${disk_usage}%[0m"
    echo
  fi
  
  # Check memory
  if [[ "$(uname)" != "Darwin" ]]; then
    local mem_usage=$(free | grep "^Mem" | awk '{printf "%.0f", $3/$2 * 100}')
    if [[ "$mem_usage" -gt 80 ]]; then
      echo -e "  [0;33m⚠ High Memory Usage: ${mem_usage}%[0m"
      echo
    fi
  fi
  
  # Show alert log
  if [[ -f "/tmp/nself-alerts.log" ]]; then
    echo -e "  [0;36mRecent Alert History:[0m"
    tail -10 /tmp/nself-alerts.log | while read -r line; do
      echo "    $line"
    done
  else
    echo "  No alerts logged"
  fi
}

# Show dashboard footer
show_dashboard_footer() {
  echo
  echo -e "[0;36m────────────────────────────────────────────────────────────────────────────────[0m"
  echo -e "Controls: [q]uit | [r]efresh | [s]ervices | [c]pu/resources | [l]ogs | [a]lerts | [space] pause"
}

# Monitor services live (standalone)
monitor_services_live() {
  while true; do
    clear
    show_command_header "nself monitor services" "Live service monitoring"
    show_services_view
    sleep "${MONITOR_INTERVAL:-2}"
  done
}

# Monitor resources live (standalone)
monitor_resources_live() {
  while true; do
    clear
    show_command_header "nself monitor resources" "Live resource monitoring"
    show_resources_view
    sleep "${MONITOR_INTERVAL:-2}"
  done
}

# Monitor logs live (standalone)
monitor_logs_live() {
  show_command_header "nself monitor logs" "Live log streaming"
  docker compose logs -f --tail=50
}

# Monitor alerts live (standalone)
monitor_alerts_live() {
  while true; do
    clear
    show_command_header "nself monitor alerts" "Live alert monitoring"
    show_alerts_view
    sleep "${MONITOR_INTERVAL:-5}"
  done
}

# Export for use as library
export -f cmd_monitor

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_monitor "$@"
  exit $?
fi