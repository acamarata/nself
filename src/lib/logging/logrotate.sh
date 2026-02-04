#!/usr/bin/env bash
# logrotate.sh - Log rotation configuration generator
# Part of nself v0.9.8 - Production Features

set -euo pipefail

# Generate logrotate configuration
generate_logrotate_config() {
  local service="$1"
  local log_path="$2"
  local retention_days="${3:-7}"
  local compress="${4:-true}"
  local max_size="${5:-100M}"

  cat <<EOF
# Logrotate configuration for $service
$log_path {
    daily
    rotate $retention_days
    maxsize $max_size
    missingok
    notifempty
    $([ "$compress" = "true" ] && echo "compress" || echo "nocompress")
    delaycompress
    sharedscripts
    postrotate
        # Send SIGHUP to service to reopen log files
        docker kill --signal=HUP \${PROJECT_NAME}_$service 2>/dev/null || true
    endscript
}
EOF
}

# Generate master logrotate configuration for all services
generate_all_logrotate_configs() {
  local output_dir="${1:-.nself/logrotate.d}"
  mkdir -p "$output_dir"

  # Load environment
  source "$(dirname "${BASH_SOURCE[0]}")/../utils/env.sh" 2>/dev/null || true
  load_env_with_priority 2>/dev/null || true

  local project_name="${PROJECT_NAME:-nself}"
  local retention_days="${LOG_RETENTION_DAYS:-7}"

  # Nginx logs
  generate_logrotate_config "nginx" "./logs/nginx/*.log" "$retention_days" "true" "100M" \
    >"$output_dir/nginx"

  # PostgreSQL logs
  generate_logrotate_config "postgres" "./logs/postgres/*.log" "$retention_days" "true" "100M" \
    >"$output_dir/postgres"

  # Hasura logs
  generate_logrotate_config "hasura" "./logs/hasura/*.log" "$retention_days" "true" "50M" \
    >"$output_dir/hasura"

  # Auth service logs
  generate_logrotate_config "auth" "./logs/auth/*.log" "$retention_days" "true" "50M" \
    >"$output_dir/auth"

  # Redis logs (if enabled)
  if [[ "${REDIS_ENABLED:-false}" == "true" ]]; then
    generate_logrotate_config "redis" "./logs/redis/*.log" "$retention_days" "true" "50M" \
      >"$output_dir/redis"
  fi

  # MinIO logs (if enabled)
  if [[ "${MINIO_ENABLED:-false}" == "true" ]]; then
    generate_logrotate_config "minio" "./logs/minio/*.log" "$retention_days" "true" "100M" \
      >"$output_dir/minio"
  fi

  # Monitoring service logs
  if [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
    generate_logrotate_config "prometheus" "./logs/prometheus/*.log" "$retention_days" "true" "100M" \
      >"$output_dir/prometheus"

    generate_logrotate_config "grafana" "./logs/grafana/*.log" "$retention_days" "true" "50M" \
      >"$output_dir/grafana"

    generate_logrotate_config "loki" "./logs/loki/*.log" "$retention_days" "true" "100M" \
      >"$output_dir/loki"
  fi

  # Custom services (CS_1 through CS_10)
  for i in {1..10}; do
    local cs_var="CS_$i"
    if [[ -n "${!cs_var:-}" ]]; then
      local service_name=$(echo "${!cs_var}" | cut -d':' -f1)
      generate_logrotate_config "$service_name" "./logs/${service_name}/*.log" "$retention_days" "true" "50M" \
        >"$output_dir/${service_name}"
    fi
  done

  printf "Generated logrotate configurations in %s/\n" "$output_dir"
}

# Install logrotate configurations system-wide (requires sudo)
install_logrotate_configs() {
  local source_dir="${1:-.nself/logrotate.d}"
  local target_dir="/etc/logrotate.d"

  if [[ ! -d "$source_dir" ]]; then
    echo "Error: Source directory $source_dir not found" >&2
    echo "Run 'generate_all_logrotate_configs' first" >&2
    return 1
  fi

  if [[ $EUID -ne 0 ]]; then
    echo "Error: This function requires sudo privileges" >&2
    echo "Run: sudo bash -c 'source $0 && install_logrotate_configs'" >&2
    return 1
  fi

  # Copy configurations
  for config in "$source_dir"/*; do
    if [[ -f "$config" ]]; then
      local basename=$(basename "$config")
      cp "$config" "$target_dir/nself-$basename"
      chmod 644 "$target_dir/nself-$basename"
      echo "Installed: $target_dir/nself-$basename"
    fi
  done

  echo "Logrotate configurations installed successfully"
  echo "Test with: logrotate -d /etc/logrotate.d/nself-*"
}

# Generate Docker Compose logging configuration
generate_docker_compose_logging() {
  local service="$1"
  local max_size="${2:-10m}"
  local max_file="${3:-3}"

  cat <<EOF
    logging:
      driver: json-file
      options:
        max-size: "$max_size"
        max-file: "$max_file"
        labels: "service=$service"
        tag: "{{.Name}}/{{.ID}}"
EOF
}

# Setup log directory structure
setup_log_directories() {
  local base_dir="${1:-./logs}"

  # Create log directories for all services
  local services=(
    "nginx"
    "postgres"
    "hasura"
    "auth"
  )

  # Add optional services
  [[ "${REDIS_ENABLED:-false}" == "true" ]] && services+=("redis")
  [[ "${MINIO_ENABLED:-false}" == "true" ]] && services+=("minio")
  [[ "${MONITORING_ENABLED:-false}" == "true" ]] && services+=("prometheus" "grafana" "loki" "tempo")

  # Custom services
  for i in {1..10}; do
    local cs_var="CS_$i"
    if [[ -n "${!cs_var:-}" ]]; then
      local service_name=$(echo "${!cs_var}" | cut -d':' -f1)
      services+=("$service_name")
    fi
  done

  # Create directories
  for service in "${services[@]}"; do
    mkdir -p "$base_dir/$service"
    touch "$base_dir/$service/.gitkeep"
  done

  # Create .gitignore for log directory
  cat >"$base_dir/.gitignore" <<'EOF'
# Ignore all log files
*.log
*.log.*
*.gz

# Keep directory structure
!.gitkeep
EOF

  printf "Log directories created in %s/\n" "$base_dir"
}

# Cleanup old logs manually
cleanup_old_logs() {
  local base_dir="${1:-./logs}"
  local days="${2:-7}"

  echo "Cleaning logs older than $days days from $base_dir..."

  local count=0
  while IFS= read -r -d '' file; do
    rm -f "$file"
    count=$((count + 1))
  done < <(find "$base_dir" -type f -name "*.log*" -mtime +${days} -print0 2>/dev/null)

  echo "Removed $count old log file(s)"
}

# Check log disk usage
check_log_disk_usage() {
  local base_dir="${1:-./logs}"

  if [[ ! -d "$base_dir" ]]; then
    echo "Log directory not found: $base_dir"
    return 1
  fi

  echo "Log Disk Usage:"
  echo ""

  # Total usage
  local total=$(du -sh "$base_dir" 2>/dev/null | cut -f1)
  echo "Total: $total"
  echo ""

  # Per-service usage
  echo "By service:"
  du -sh "$base_dir"/* 2>/dev/null | sort -hr | while read -r size path; do
    local service=$(basename "$path")
    printf "  %-20s %s\n" "$service" "$size"
  done
}

# Export functions
export -f generate_logrotate_config generate_all_logrotate_configs install_logrotate_configs
export -f generate_docker_compose_logging setup_log_directories cleanup_old_logs check_log_disk_usage

# If run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  case "${1:-help}" in
    generate)
      generate_all_logrotate_configs "${2:-.nself/logrotate.d}"
      ;;
    install)
      install_logrotate_configs "${2:-.nself/logrotate.d}"
      ;;
    setup)
      setup_log_directories "${2:-./logs}"
      ;;
    cleanup)
      cleanup_old_logs "${2:-./logs}" "${3:-7}"
      ;;
    usage)
      check_log_disk_usage "${2:-./logs}"
      ;;
    *)
      echo "Usage: $0 {generate|install|setup|cleanup|usage} [options]"
      echo ""
      echo "Commands:"
      echo "  generate [dir]       Generate logrotate configs"
      echo "  install [dir]        Install configs to /etc/logrotate.d (requires sudo)"
      echo "  setup [dir]          Create log directory structure"
      echo "  cleanup [dir] [days] Remove logs older than N days"
      echo "  usage [dir]          Show disk usage by service"
      exit 1
      ;;
  esac
fi
