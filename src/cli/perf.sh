#!/usr/bin/env bash

# perf.sh - Performance profiling and analysis
# v0.4.6 - Part of the Scaling & Performance release

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/header.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Color fallbacks
: "${COLOR_GREEN:=\033[0;32m}"
: "${COLOR_YELLOW:=\033[0;33m}"
: "${COLOR_RED:=\033[0;31m}"
: "${COLOR_CYAN:=\033[0;36m}"
: "${COLOR_RESET:=\033[0m}"
: "${COLOR_DIM:=\033[2m}"
: "${COLOR_BOLD:=\033[1m}"

# Output format
OUTPUT_FORMAT="table"

# Show help
show_perf_help() {
  cat << 'EOF'
nself perf - Performance profiling and analysis

Usage: nself perf <subcommand> [options]

Subcommands:
  profile [service]     Full system or service-specific profile
  analyze               Analyze current performance
  slow-queries          Detailed slow query analysis
  report                Generate performance report
  dashboard             Real-time terminal dashboard
  suggest               Get optimization suggestions

Options:
  --duration <seconds>  Profile duration (default: 60)
  --format <format>     Output format: table, json (default: table)
  --output <file>       Save report to file
  -h, --help            Show this help message

Examples:
  nself perf profile                 # Full system profile
  nself perf profile postgres        # Profile specific service
  nself perf analyze --slow-queries  # Analyze with slow query focus
  nself perf slow-queries            # Detailed slow query analysis
  nself perf report --format json    # Generate JSON report
  nself perf dashboard               # Real-time dashboard
  nself perf suggest                 # Get optimization tips
EOF
}

# Profile system or service
cmd_profile() {
  local service="${1:-}"
  local duration="${PROFILE_DURATION:-60}"

  if [[ -n "$service" ]]; then
    show_command_header "nself perf" "Profiling service: $service"
  else
    show_command_header "nself perf" "Full system profile (${duration}s)"
  fi
  echo ""

  # Check if docker-compose.yml exists
  if [[ ! -f "docker-compose.yml" ]]; then
    log_error "No docker-compose.yml found. Run 'nself build' first."
    return 1
  fi

  load_env_with_priority
  local project_name="${PROJECT_NAME:-nself}"

  if [[ -n "$service" ]]; then
    # Profile specific service
    local container_name="${project_name}_${service}"

    if ! docker ps --format "{{.Names}}" | grep -q "^${container_name}"; then
      log_error "Service '$service' is not running"
      return 1
    fi

    printf "${COLOR_CYAN}➞ Service Profile: %s${COLOR_RESET}\n" "$service"
    echo ""

    # Get container stats
    printf "%-15s %-10s %-15s %-15s %-15s\n" "METRIC" "CURRENT" "AVG" "MAX" "STATUS"
    echo "─────────────────────────────────────────────────────────────────────"

    # Collect stats over duration
    local cpu_total=0
    local mem_total=0
    local cpu_max=0
    local mem_max=0
    local samples=0

    log_info "Collecting samples for ${duration}s..."

    local end_time=$((SECONDS + duration))
    while [[ $SECONDS -lt $end_time ]]; do
      local stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemPerc}}" "$container_name" 2>/dev/null)
      if [[ -n "$stats" ]]; then
        local cpu=$(echo "$stats" | cut -f1 | tr -d '%')
        local mem=$(echo "$stats" | cut -f2 | tr -d '%')

        cpu_total=$(awk "BEGIN {print $cpu_total + $cpu}")
        mem_total=$(awk "BEGIN {print $mem_total + $mem}")

        if (( $(echo "$cpu > $cpu_max" | bc -l 2>/dev/null || echo 0) )); then
          cpu_max=$cpu
        fi
        if (( $(echo "$mem > $mem_max" | bc -l 2>/dev/null || echo 0) )); then
          mem_max=$mem
        fi

        samples=$((samples + 1))
      fi
      sleep 2
    done

    if [[ $samples -gt 0 ]]; then
      local cpu_avg=$(awk "BEGIN {printf \"%.1f\", $cpu_total / $samples}")
      local mem_avg=$(awk "BEGIN {printf \"%.1f\", $mem_total / $samples}")

      # Current stats
      local current=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemPerc}}" "$container_name" 2>/dev/null)
      local cpu_cur=$(echo "$current" | cut -f1)
      local mem_cur=$(echo "$current" | cut -f2)

      # Determine status
      local cpu_status="✓ Normal"
      local mem_status="✓ Normal"

      if (( $(echo "$cpu_avg > 70" | bc -l 2>/dev/null || echo 0) )); then
        cpu_status="⚠ High"
      fi
      if (( $(echo "$mem_avg > 80" | bc -l 2>/dev/null || echo 0) )); then
        mem_status="⚠ High"
      fi

      printf "%-15s %-10s %-15s %-15s %-15s\n" "CPU Usage" "$cpu_cur" "${cpu_avg}%" "${cpu_max}%" "$cpu_status"
      printf "%-15s %-10s %-15s %-15s %-15s\n" "Memory Usage" "$mem_cur" "${mem_avg}%" "${mem_max}%" "$mem_status"
    fi

  else
    # Full system profile
    printf "${COLOR_CYAN}➞ System Profile Summary${COLOR_RESET}\n"
    echo ""

    # Get all running containers
    local containers=($(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null))

    printf "%-20s %-10s %-15s %-10s\n" "SERVICE" "CPU%" "MEMORY" "STATUS"
    echo "─────────────────────────────────────────────────────────"

    for container in "${containers[@]}"; do
      local service_name=${container#${project_name}_}
      local stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}" "$container" 2>/dev/null)

      if [[ -n "$stats" ]]; then
        local cpu=$(echo "$stats" | cut -f1)
        local mem=$(echo "$stats" | cut -f2)
        local status="✓"

        # Check health
        local health=$(docker inspect "$container" --format='{{.State.Health.Status}}' 2>/dev/null || echo "none")
        if [[ "$health" == "unhealthy" ]]; then
          status="✗"
        fi

        printf "%-20s %-10s %-15s %-10s\n" "$service_name" "$cpu" "$mem" "$status"
      fi
    done
  fi

  echo ""
  log_info "Use 'nself perf suggest' for optimization recommendations"
}

# Analyze performance
cmd_analyze() {
  local show_slow_queries=false
  local show_memory=false
  local show_cpu=false

  # Parse analyze flags
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --slow-queries) show_slow_queries=true; shift ;;
      --memory) show_memory=true; shift ;;
      --cpu) show_cpu=true; shift ;;
      *) shift ;;
    esac
  done

  show_command_header "nself perf" "Performance analysis"
  echo ""

  load_env_with_priority
  local project_name="${PROJECT_NAME:-nself}"

  # Default: show all
  if [[ "$show_slow_queries" == "false" ]] && [[ "$show_memory" == "false" ]] && [[ "$show_cpu" == "false" ]]; then
    show_slow_queries=true
    show_memory=true
    show_cpu=true
  fi

  # CPU Analysis
  if [[ "$show_cpu" == "true" ]]; then
    printf "${COLOR_CYAN}➞ CPU Analysis${COLOR_RESET}\n"
    echo ""

    local high_cpu=()
    local containers=($(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null))

    for container in "${containers[@]}"; do
      local cpu=$(docker stats --no-stream --format "{{.CPUPerc}}" "$container" 2>/dev/null | tr -d '%')
      if [[ -n "$cpu" ]] && (( $(echo "$cpu > 50" | bc -l 2>/dev/null || echo 0) )); then
        high_cpu+=("${container#${project_name}_}:${cpu}%")
      fi
    done

    if [[ ${#high_cpu[@]} -gt 0 ]]; then
      log_warning "High CPU usage detected:"
      for item in "${high_cpu[@]}"; do
        echo "  - $item"
      done
    else
      log_success "CPU usage is normal across all services"
    fi
    echo ""
  fi

  # Memory Analysis
  if [[ "$show_memory" == "true" ]]; then
    printf "${COLOR_CYAN}➞ Memory Analysis${COLOR_RESET}\n"
    echo ""

    local high_mem=()
    local containers=($(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null))

    for container in "${containers[@]}"; do
      local mem=$(docker stats --no-stream --format "{{.MemPerc}}" "$container" 2>/dev/null | tr -d '%')
      if [[ -n "$mem" ]] && (( $(echo "$mem > 70" | bc -l 2>/dev/null || echo 0) )); then
        high_mem+=("${container#${project_name}_}:${mem}%")
      fi
    done

    if [[ ${#high_mem[@]} -gt 0 ]]; then
      log_warning "High memory usage detected:"
      for item in "${high_mem[@]}"; do
        echo "  - $item"
      done
    else
      log_success "Memory usage is normal across all services"
    fi
    echo ""
  fi

  # Slow Query Analysis
  if [[ "$show_slow_queries" == "true" ]]; then
    cmd_slow_queries
  fi
}

# Slow query analysis
cmd_slow_queries() {
  printf "${COLOR_CYAN}➞ Slow Query Analysis${COLOR_RESET}\n"
  echo ""

  load_env_with_priority
  local project_name="${PROJECT_NAME:-nself}"
  local container="${project_name}_postgres"

  if ! docker ps --format "{{.Names}}" | grep -q "^${container}"; then
    log_warning "PostgreSQL is not running"
    return 0
  fi

  local db_user="${POSTGRES_USER:-postgres}"
  local db_name="${POSTGRES_DB:-nhost}"
  local threshold="${SLOW_QUERY_THRESHOLD:-100}"  # milliseconds

  # Check if pg_stat_statements is available
  local has_pg_stat=$(docker exec "$container" psql -U "$db_user" -d "$db_name" -t -c \
    "SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements';" 2>/dev/null | xargs)

  if [[ "$has_pg_stat" == "1" ]]; then
    log_info "Analyzing queries slower than ${threshold}ms..."
    echo ""

    # Get slow queries
    local slow_queries=$(docker exec "$container" psql -U "$db_user" -d "$db_name" -t -c "
      SELECT
        round(mean_exec_time::numeric, 2) as avg_ms,
        calls,
        left(query, 60) as query
      FROM pg_stat_statements
      WHERE mean_exec_time > $threshold
      ORDER BY mean_exec_time DESC
      LIMIT 10;
    " 2>/dev/null)

    if [[ -n "$slow_queries" && "$slow_queries" != *"0 rows"* ]]; then
      printf "%-12s %-10s %-60s\n" "AVG (ms)" "CALLS" "QUERY"
      echo "────────────────────────────────────────────────────────────────────────────────"
      echo "$slow_queries" | while read -r line; do
        if [[ -n "$line" ]]; then
          echo "$line"
        fi
      done
    else
      log_success "No slow queries detected (threshold: ${threshold}ms)"
    fi
  else
    log_warning "pg_stat_statements extension not installed"
    log_info "Enable with: CREATE EXTENSION pg_stat_statements;"

    # Fallback: Check pg_stat_activity for long-running queries
    echo ""
    log_info "Checking for long-running queries..."

    local long_running=$(docker exec "$container" psql -U "$db_user" -d "$db_name" -t -c "
      SELECT
        pid,
        round(extract(epoch from (now() - query_start))::numeric, 1) as duration_sec,
        left(query, 50) as query
      FROM pg_stat_activity
      WHERE state = 'active'
        AND query NOT LIKE '%pg_stat_activity%'
        AND query_start < now() - interval '5 seconds'
      ORDER BY query_start
      LIMIT 5;
    " 2>/dev/null)

    if [[ -n "$long_running" ]]; then
      echo "$long_running"
    else
      log_success "No long-running queries found"
    fi
  fi

  echo ""
}

# Generate performance report
cmd_report() {
  local output_file="${REPORT_OUTPUT:-}"
  local format="${OUTPUT_FORMAT:-table}"

  show_command_header "nself perf" "Generating performance report"
  echo ""

  load_env_with_priority
  local project_name="${PROJECT_NAME:-nself}"
  local timestamp=$(date +%Y%m%d_%H%M%S)

  if [[ "$format" == "json" ]]; then
    local report="{"
    report+="\"timestamp\": \"$(date -Iseconds)\","
    report+="\"project\": \"$project_name\","
    report+="\"services\": ["

    local first=true
    local containers=($(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null))

    for container in "${containers[@]}"; do
      local service_name=${container#${project_name}_}
      local stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemPerc}}\t{{.MemUsage}}" "$container" 2>/dev/null)
      local cpu=$(echo "$stats" | cut -f1 | tr -d '%')
      local mem=$(echo "$stats" | cut -f2 | tr -d '%')
      local mem_usage=$(echo "$stats" | cut -f3)

      [[ "$first" == "true" ]] || report+=","
      first=false

      report+="{\"name\": \"$service_name\", \"cpu\": $cpu, \"memory\": $mem, \"memory_usage\": \"$mem_usage\"}"
    done

    report+="]}"

    if [[ -n "$output_file" ]]; then
      echo "$report" > "$output_file"
      log_success "Report saved to: $output_file"
    else
      echo "$report"
    fi
  else
    # Table format
    printf "${COLOR_CYAN}Performance Report - %s${COLOR_RESET}\n" "$(date)"
    echo "════════════════════════════════════════════════════════════════"
    echo ""

    printf "Project: %s\n" "$project_name"
    printf "Environment: %s\n" "${ENV:-local}"
    echo ""

    printf "${COLOR_BOLD}Service Metrics${COLOR_RESET}\n"
    echo "────────────────────────────────────────────────────────────────"
    printf "%-20s %-10s %-10s %-20s\n" "SERVICE" "CPU%" "MEM%" "MEMORY USAGE"
    echo "────────────────────────────────────────────────────────────────"

    local containers=($(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null))
    for container in "${containers[@]}"; do
      local service_name=${container#${project_name}_}
      local stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemPerc}}\t{{.MemUsage}}" "$container" 2>/dev/null)
      local cpu=$(echo "$stats" | cut -f1)
      local mem=$(echo "$stats" | cut -f2)
      local mem_usage=$(echo "$stats" | cut -f3)

      printf "%-20s %-10s %-10s %-20s\n" "$service_name" "$cpu" "$mem" "$mem_usage"
    done

    echo ""

    if [[ -n "$output_file" ]]; then
      # Save to file (redirect entire output)
      log_success "Report displayed above"
    fi
  fi
}

# Real-time dashboard
cmd_dashboard() {
  show_command_header "nself perf" "Real-time performance dashboard"

  load_env_with_priority
  local project_name="${PROJECT_NAME:-nself}"

  log_info "Starting real-time dashboard (Ctrl+C to exit)..."
  echo ""

  trap 'echo ""; log_info "Dashboard stopped"; exit 0' INT

  while true; do
    clear
    printf "${COLOR_BOLD}nself Performance Dashboard${COLOR_RESET} - %s\n" "$(date '+%H:%M:%S')"
    echo "════════════════════════════════════════════════════════════════════"
    echo ""

    # Get stats
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" \
      $(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null | xargs) 2>/dev/null | \
      sed "s/${project_name}_//g"

    echo ""
    echo "Press Ctrl+C to exit"

    sleep 2
  done
}

# Optimization suggestions
cmd_suggest() {
  show_command_header "nself perf" "Optimization suggestions"
  echo ""

  load_env_with_priority
  local project_name="${PROJECT_NAME:-nself}"
  local suggestions=()

  # Check container stats
  local containers=($(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null))

  for container in "${containers[@]}"; do
    local service_name=${container#${project_name}_}
    local stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemPerc}}" "$container" 2>/dev/null)
    local cpu=$(echo "$stats" | cut -f1 | tr -d '%')
    local mem=$(echo "$stats" | cut -f2 | tr -d '%')

    if [[ -n "$cpu" ]] && (( $(echo "$cpu > 70" | bc -l 2>/dev/null || echo 0) )); then
      suggestions+=("High CPU on $service_name (${cpu}%): Consider scaling horizontally with 'nself scale out $service_name'")
    fi

    if [[ -n "$mem" ]] && (( $(echo "$mem > 80" | bc -l 2>/dev/null || echo 0) )); then
      suggestions+=("High memory on $service_name (${mem}%): Increase memory limit with 'nself scale $service_name --memory 4G'")
    fi
  done

  # Check PostgreSQL specific
  local pg_container="${project_name}_postgres"
  if docker ps --format "{{.Names}}" | grep -q "^${pg_container}"; then
    local db_user="${POSTGRES_USER:-postgres}"
    local db_name="${POSTGRES_DB:-nhost}"

    # Check connection count
    local conn_count=$(docker exec "$pg_container" psql -U "$db_user" -d "$db_name" -t -c \
      "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | xargs)

    if [[ -n "$conn_count" ]] && [[ "$conn_count" -gt 50 ]]; then
      suggestions+=("High database connections ($conn_count): Enable connection pooling with 'nself scale pooler enable'")
    fi

    # Check for missing indexes
    local missing_indexes=$(docker exec "$pg_container" psql -U "$db_user" -d "$db_name" -t -c "
      SELECT count(*) FROM pg_stat_user_tables
      WHERE seq_scan > idx_scan AND n_live_tup > 10000;
    " 2>/dev/null | xargs)

    if [[ -n "$missing_indexes" ]] && [[ "$missing_indexes" -gt 0 ]]; then
      suggestions+=("$missing_indexes tables have sequential scans > index scans: Run 'nself db inspect index-advisor'")
    fi
  fi

  # Check Redis if enabled
  if [[ "${REDIS_ENABLED:-false}" == "true" ]]; then
    local redis_container="${project_name}_redis"
    if docker ps --format "{{.Names}}" | grep -q "^${redis_container}"; then
      local redis_mem=$(docker exec "$redis_container" redis-cli INFO memory 2>/dev/null | grep used_memory_human | cut -d: -f2 | tr -d '\r')
      if [[ -n "$redis_mem" ]]; then
        suggestions+=("Redis memory usage: $redis_mem - Consider memory limits if growing")
      fi
    fi
  fi

  # Output suggestions
  if [[ ${#suggestions[@]} -gt 0 ]]; then
    printf "${COLOR_CYAN}➞ Optimization Recommendations${COLOR_RESET}\n"
    echo ""
    local i=1
    for suggestion in "${suggestions[@]}"; do
      printf "${COLOR_YELLOW}%d.${COLOR_RESET} %s\n" "$i" "$suggestion"
      i=$((i + 1))
    done
  else
    log_success "No immediate optimizations needed"
    log_info "System performance looks healthy"
  fi

  echo ""
  log_info "For detailed analysis, run 'nself perf analyze'"
}

# Main command handler
cmd_perf() {
  local subcommand="${1:-}"
  shift 2>/dev/null || true

  # Parse global options
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --duration)
        PROFILE_DURATION="$2"
        shift 2
        ;;
      --format)
        OUTPUT_FORMAT="$2"
        shift 2
        ;;
      --output)
        REPORT_OUTPUT="$2"
        shift 2
        ;;
      --json)
        OUTPUT_FORMAT="json"
        shift
        ;;
      -h|--help)
        show_perf_help
        return 0
        ;;
      *)
        break
        ;;
    esac
  done

  case "$subcommand" in
    profile)
      cmd_profile "$@"
      ;;
    analyze)
      cmd_analyze "$@"
      ;;
    slow-queries)
      cmd_slow_queries
      ;;
    report)
      cmd_report
      ;;
    dashboard)
      cmd_dashboard
      ;;
    suggest)
      cmd_suggest
      ;;
    -h|--help|"")
      show_perf_help
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      show_perf_help
      return 1
      ;;
  esac
}

# Export for use as library
export -f cmd_perf

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "perf" || exit $?
  cmd_perf "$@"
  exit_code=$?
  post_command "perf" $exit_code
  exit $exit_code
fi
