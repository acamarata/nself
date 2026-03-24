#!/usr/bin/env bash
# health-watchdog.sh - System health watchdog with Telegram alerting
# Part of nself SysOps - Bash 3.2 compatible

set -euo pipefail

# Get the directory where this script is located
_WATCHDOG_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_WATCHDOG_LIB_ROOT="$(dirname "$_WATCHDOG_DIR")"

# Source dependencies
source "$_WATCHDOG_LIB_ROOT/utils/display.sh" 2>/dev/null || true
source "$_WATCHDOG_LIB_ROOT/utils/platform-compat.sh" 2>/dev/null || true

# Configuration with defaults
WATCHDOG_CPU_THRESHOLD="${WATCHDOG_CPU_THRESHOLD:-80}"
WATCHDOG_DISK_THRESHOLD="${WATCHDOG_DISK_THRESHOLD:-85}"
WATCHDOG_MEMORY_THRESHOLD="${WATCHDOG_MEMORY_THRESHOLD:-90}"
WATCHDOG_CHECK_INTERVAL="${WATCHDOG_CHECK_INTERVAL:-60}"
WATCHDOG_COOLDOWN="${WATCHDOG_COOLDOWN:-300}"
WATCHDOG_STATE_FILE="${WATCHDOG_STATE_FILE:-/tmp/nself-watchdog-state}"
WATCHDOG_LOG="${WATCHDOG_LOG:-./logs/watchdog.log}"

# Telegram configuration
TELEGRAM_BOT_TOKEN="${TELEGRAM_BOT_TOKEN:-}"
TELEGRAM_CHAT_ID="${TELEGRAM_CHAT_ID:-}"

# Initialize watchdog state
watchdog_init() {
  mkdir -p "$(dirname "$WATCHDOG_LOG")"
  touch "$WATCHDOG_LOG"
  if [[ ! -f "$WATCHDOG_STATE_FILE" ]]; then
    printf "" > "$WATCHDOG_STATE_FILE"
  fi
}

# Log a watchdog event
watchdog_log() {
  local level="$1"
  local message="$2"
  printf "[%s] [%s] %s\n" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "$message" >> "$WATCHDOG_LOG"
}

# Check if an alert should be sent (cooldown-based dedup)
watchdog_should_alert() {
  local key="$1"
  local cooldown="${WATCHDOG_COOLDOWN:-300}"
  local now
  now=$(date +%s)

  local last_alert=""
  if [[ -f "$WATCHDOG_STATE_FILE" ]]; then
    last_alert=$(grep "^${key}=" "$WATCHDOG_STATE_FILE" 2>/dev/null | tail -1 | cut -d= -f2)
  fi

  if [[ -z "$last_alert" ]]; then
    return 0
  fi

  local elapsed=$((now - last_alert))
  if [[ "$elapsed" -gt "$cooldown" ]]; then
    return 0
  fi
  return 1
}

# Update alert state
watchdog_update_state() {
  local key="$1"
  local now
  now=$(date +%s)

  if [[ -f "$WATCHDOG_STATE_FILE" ]] && grep -q "^${key}=" "$WATCHDOG_STATE_FILE" 2>/dev/null; then
    # Update existing entry using a temp file (Bash 3.2 safe)
    local tmpfile
    tmpfile=$(mktemp)
    while IFS= read -r line; do
      case "$line" in
        "${key}="*) printf '%s=%s\n' "$key" "$now" ;;
        *) printf '%s\n' "$line" ;;
      esac
    done < "$WATCHDOG_STATE_FILE" > "$tmpfile"
    mv "$tmpfile" "$WATCHDOG_STATE_FILE"
  else
    printf '%s=%s\n' "$key" "$now" >> "$WATCHDOG_STATE_FILE"
  fi
}

# Send Telegram alert
watchdog_telegram_alert() {
  local message="$1"

  if [[ -z "$TELEGRAM_BOT_TOKEN" ]] || [[ -z "$TELEGRAM_CHAT_ID" ]]; then
    watchdog_log "WARNING" "Telegram not configured, alert not sent: $message"
    return 1
  fi

  if ! command -v curl >/dev/null 2>&1; then
    watchdog_log "ERROR" "curl not found, cannot send Telegram alert"
    return 1
  fi

  local hostname
  hostname=$(hostname 2>/dev/null || printf "unknown")

  local full_message
  full_message=$(printf "[nself] %s\nHost: %s\nTime: %s" "$message" "$hostname" "$(date -u +%Y-%m-%dT%H:%M:%SZ)")

  local url="https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage"

  if curl -s -X POST "$url" \
    -d "chat_id=${TELEGRAM_CHAT_ID}" \
    -d "text=${full_message}" \
    -d "parse_mode=HTML" \
    >/dev/null 2>&1; then
    watchdog_log "INFO" "Telegram alert sent: $message"
  else
    watchdog_log "ERROR" "Failed to send Telegram alert: $message"
    return 1
  fi
}

# Send alert through configured channels
watchdog_alert() {
  local key="$1"
  local message="$2"

  if ! watchdog_should_alert "$key"; then
    return 0
  fi

  watchdog_log "ALERT" "$message"
  watchdog_telegram_alert "$message" || true
  watchdog_update_state "$key"
}

# Check CPU usage
watchdog_check_cpu() {
  local threshold="${WATCHDOG_CPU_THRESHOLD:-80}"
  local cpu_usage=""

  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS: use top
    cpu_usage=$(top -l 1 -n 0 2>/dev/null | grep "CPU usage" | awk '{print int($3)}')
  else
    # Linux: use /proc/stat or top
    if [[ -f /proc/stat ]]; then
      local cpu_line1 cpu_line2
      cpu_line1=$(head -1 /proc/stat)
      sleep 1
      cpu_line2=$(head -1 /proc/stat)

      # Parse idle values
      local idle1 total1 idle2 total2
      idle1=$(printf '%s' "$cpu_line1" | awk '{print $5}')
      total1=$(printf '%s' "$cpu_line1" | awk '{sum=0; for(i=2;i<=NF;i++) sum+=$i; print sum}')
      idle2=$(printf '%s' "$cpu_line2" | awk '{print $5}')
      total2=$(printf '%s' "$cpu_line2" | awk '{sum=0; for(i=2;i<=NF;i++) sum+=$i; print sum}')

      local idle_diff=$((idle2 - idle1))
      local total_diff=$((total2 - total1))
      if [[ "$total_diff" -gt 0 ]]; then
        cpu_usage=$(( (total_diff - idle_diff) * 100 / total_diff ))
      else
        cpu_usage=0
      fi
    else
      cpu_usage=$(top -bn1 2>/dev/null | grep "Cpu(s)" | awk '{print int(100 - $8)}')
    fi
  fi

  if [[ -z "$cpu_usage" ]]; then
    return 0
  fi

  if [[ "$cpu_usage" -gt "$threshold" ]]; then
    watchdog_alert "cpu_high" "CPU usage is ${cpu_usage}% (threshold: ${threshold}%)"
    return 1
  fi
  return 0
}

# Check disk usage
watchdog_check_disk() {
  local threshold="${WATCHDOG_DISK_THRESHOLD:-85}"

  # Check root partition and Docker data directory
  local partitions="/ /var/lib/docker"
  for mount_point in $partitions; do
    if ! df "$mount_point" >/dev/null 2>&1; then
      continue
    fi

    local usage
    usage=$(df "$mount_point" 2>/dev/null | tail -1 | awk '{gsub(/%/,"",$5); print $5}')
    if [[ -z "$usage" ]]; then
      continue
    fi

    if [[ "$usage" -gt "$threshold" ]]; then
      watchdog_alert "disk_high_${mount_point}" "Disk usage on $mount_point is ${usage}% (threshold: ${threshold}%)"
    fi
  done
}

# Check memory usage
watchdog_check_memory() {
  local threshold="${WATCHDOG_MEMORY_THRESHOLD:-90}"
  local mem_usage=""

  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS: approximate from vm_stat
    local page_size
    page_size=$(sysctl -n hw.pagesize 2>/dev/null || printf "4096")
    local pages_free
    pages_free=$(vm_stat 2>/dev/null | grep "Pages free" | awk '{print $3}' | tr -d '.')
    local pages_active
    pages_active=$(vm_stat 2>/dev/null | grep "Pages active" | awk '{print $3}' | tr -d '.')
    local pages_inactive
    pages_inactive=$(vm_stat 2>/dev/null | grep "Pages inactive" | awk '{print $3}' | tr -d '.')
    local pages_wired
    pages_wired=$(vm_stat 2>/dev/null | grep "Pages wired" | awk '{print $4}' | tr -d '.')

    if [[ -n "$pages_free" ]] && [[ -n "$pages_active" ]]; then
      local total=$((pages_free + pages_active + pages_inactive + pages_wired))
      local used=$((pages_active + pages_wired))
      if [[ "$total" -gt 0 ]]; then
        mem_usage=$((used * 100 / total))
      fi
    fi
  else
    # Linux: use /proc/meminfo
    if [[ -f /proc/meminfo ]]; then
      local total_kb available_kb
      total_kb=$(grep '^MemTotal:' /proc/meminfo | awk '{print $2}')
      available_kb=$(grep '^MemAvailable:' /proc/meminfo | awk '{print $2}')
      if [[ -n "$total_kb" ]] && [[ -n "$available_kb" ]] && [[ "$total_kb" -gt 0 ]]; then
        local used_kb=$((total_kb - available_kb))
        mem_usage=$((used_kb * 100 / total_kb))
      fi
    fi
  fi

  if [[ -z "$mem_usage" ]]; then
    return 0
  fi

  if [[ "$mem_usage" -gt "$threshold" ]]; then
    watchdog_alert "memory_high" "Memory usage is ${mem_usage}% (threshold: ${threshold}%)"
    return 1
  fi
  return 0
}

# Check Docker container health
watchdog_check_containers() {
  if ! command -v docker >/dev/null 2>&1; then
    return 0
  fi

  local project_name="${PROJECT_NAME:-}"
  local filter=""
  if [[ -n "$project_name" ]]; then
    filter="--filter name=${project_name}_"
  fi

  # Get all containers that should be running (restart: unless-stopped)
  local down_containers=""
  while IFS= read -r line; do
    if [[ -z "$line" ]]; then
      continue
    fi
    local name status
    name=$(printf '%s' "$line" | awk '{print $1}')
    status=$(printf '%s' "$line" | awk '{print $2}')

    if [[ "$status" != "running" ]]; then
      if [[ -n "$down_containers" ]]; then
        down_containers="${down_containers}, ${name}"
      else
        down_containers="$name"
      fi
    fi
  done < <(docker ps -a $filter --format "{{.Names}} {{.State}}" 2>/dev/null)

  if [[ -n "$down_containers" ]]; then
    watchdog_alert "container_down" "Container(s) not running: $down_containers"
    return 1
  fi
  return 0
}

# Run all health checks once
watchdog_check_all() {
  watchdog_init

  local issues=0
  watchdog_check_cpu || issues=$((issues + 1))
  watchdog_check_disk || issues=$((issues + 1))
  watchdog_check_memory || issues=$((issues + 1))
  watchdog_check_containers || issues=$((issues + 1))

  if [[ "$issues" -eq 0 ]]; then
    printf "\033[32m%s\033[0m All health checks passed\n" "SUCCESS"
  else
    printf "\033[33m%s\033[0m %d issue(s) detected\n" "WARNING" "$issues"
  fi

  return "$issues"
}

# Run watchdog in continuous loop mode
watchdog_run() {
  local interval="${WATCHDOG_CHECK_INTERVAL:-60}"

  if [[ -z "$TELEGRAM_BOT_TOKEN" ]] || [[ -z "$TELEGRAM_CHAT_ID" ]]; then
    printf "\033[33m%s\033[0m TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID not set.\n" "WARNING"
    printf "  Alerts will only be logged to %s\n" "$WATCHDOG_LOG"
  fi

  printf "\033[34m%s\033[0m Starting health watchdog (interval: %ss)\n" "INFO" "$interval"
  printf "  CPU threshold:    %s%%\n" "$WATCHDOG_CPU_THRESHOLD"
  printf "  Disk threshold:   %s%%\n" "$WATCHDOG_DISK_THRESHOLD"
  printf "  Memory threshold: %s%%\n" "$WATCHDOG_MEMORY_THRESHOLD"
  printf "  Alert cooldown:   %ss\n" "$WATCHDOG_COOLDOWN"
  printf "\n"

  watchdog_init

  while true; do
    watchdog_check_cpu || true
    watchdog_check_disk || true
    watchdog_check_memory || true
    watchdog_check_containers || true
    sleep "$interval"
  done
}

# Main dispatch
health_watchdog() {
  local action="${1:-check}"
  shift 2>/dev/null || true

  case "$action" in
    check|--check)
      watchdog_check_all
      ;;
    run|--run|daemon)
      watchdog_run
      ;;
    status|--status)
      if [[ -f "$WATCHDOG_STATE_FILE" ]]; then
        printf "\033[34m%s\033[0m Watchdog state:\n" "INFO"
        cat "$WATCHDOG_STATE_FILE"
      else
        printf "\033[34m%s\033[0m No watchdog state (never run)\n" "INFO"
      fi
      ;;
    --help|help|-h)
      printf "nself health watchdog - System health monitoring\n\n"
      printf "USAGE:\n"
      printf "  health_watchdog check    Run all checks once\n"
      printf "  health_watchdog run      Run continuous monitoring daemon\n"
      printf "  health_watchdog status   Show alert state\n"
      printf "\nENV VARS:\n"
      printf "  TELEGRAM_BOT_TOKEN      Telegram bot token for alerts\n"
      printf "  TELEGRAM_CHAT_ID        Telegram chat ID for alerts\n"
      printf "  WATCHDOG_CPU_THRESHOLD  CPU alert threshold (default: 80)\n"
      printf "  WATCHDOG_DISK_THRESHOLD Disk alert threshold (default: 85)\n"
      printf "  WATCHDOG_MEMORY_THRESHOLD Memory alert threshold (default: 90)\n"
      printf "  WATCHDOG_CHECK_INTERVAL Check interval in seconds (default: 60)\n"
      ;;
    *)
      printf "\033[33m%s\033[0m Unknown action: %s (use --help)\n" "WARNING" "$action" >&2
      return 1
      ;;
  esac
}

# Export functions
export -f health_watchdog 2>/dev/null || true
export -f watchdog_check_all 2>/dev/null || true
export -f watchdog_telegram_alert 2>/dev/null || true
