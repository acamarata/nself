#!/usr/bin/env bash
# resources.sh - Auto-calculate Docker resource limits
# Part of nself v0.9.8 - Production Features

set -euo pipefail

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../utils/cli-output.sh" 2>/dev/null || true

# Resource allocation weights by service type
# Scale from 1-10 where 10 = highest resource priority
# Bash 3.2 compatible: use functions instead of associative arrays
get_service_weight() {
  case "$1" in
    postgres) echo "10" ;;
    hasura) echo "8" ;;
    minio|meilisearch|mlflow) echo "7" ;;
    redis|prometheus|loki|tempo|functions) echo "6" ;;
    auth|grafana|custom) echo "5" ;;
    nginx) echo "4" ;;
    *) echo "5" ;; # default
  esac
}

# Minimum memory requirements (in MB)
get_min_memory() {
  case "$1" in
    postgres|meilisearch|prometheus|mlflow) echo "512" ;;
    hasura|minio|grafana|loki|tempo|functions) echo "256" ;;
    redis|auth|custom) echo "128" ;;
    nginx) echo "64" ;;
    *) echo "128" ;; # default
  esac
}

# Minimum CPU requirements (in millicores)
get_min_cpu() {
  case "$1" in
    postgres) echo "500" ;;
    hasura|minio|meilisearch|prometheus|loki|tempo|functions|mlflow) echo "250" ;;
    redis|auth|nginx|grafana|custom) echo "100" ;;
    *) echo "100" ;; # default
  esac
}

# Detect system resources
detect_system_resources() {
  local total_memory_mb=0
  local total_cpu_cores=0

  # Detect OS and get resources accordingly
  if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    total_memory_mb=$(sysctl -n hw.memsize 2>/dev/null | awk '{print int($1/1024/1024)}' || echo "8192")
    total_cpu_cores=$(sysctl -n hw.ncpu 2>/dev/null || echo "4")
  elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    total_memory_mb=$(free -m | awk '/^Mem:/{print $2}' || echo "8192")
    total_cpu_cores=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo || echo "4")
  else
    # Default fallback
    total_memory_mb=8192
    total_cpu_cores=4
  fi

  # Account for system overhead (reserve 20%)
  local available_memory=$((total_memory_mb * 80 / 100))
  local available_cpu=$((total_cpu_cores * 80 / 100))

  printf '{"total_memory_mb": %d, "total_cpu_cores": %d, "available_memory_mb": %d, "available_cpu_cores": %d}\n' \
    "$total_memory_mb" "$total_cpu_cores" "$available_memory" "$available_cpu"
}

# Get list of enabled services
get_enabled_services() {
  local services=()

  # Always required services
  services+=("postgres" "hasura" "auth" "nginx")

  # Optional services
  [[ "${REDIS_ENABLED:-false}" == "true" ]] && services+=("redis")
  [[ "${MINIO_ENABLED:-false}" == "true" ]] && services+=("minio")
  [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]] && services+=("meilisearch")
  [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]] && services+=("functions")
  [[ "${MLFLOW_ENABLED:-false}" == "true" ]] && services+=("mlflow")

  # Monitoring services
  if [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
    services+=("prometheus" "grafana" "loki" "tempo" "alertmanager" "cadvisor" "node_exporter" "postgres_exporter")
    [[ "${REDIS_ENABLED:-false}" == "true" ]] && services+=("redis_exporter")
  fi

  # Custom services (CS_1 through CS_10)
  for i in {1..10}; do
    if [[ -n "${!CS_${i}:-}" ]]; then
      services+=("custom")
    fi
  done

  printf '%s\n' "${services[@]}"
}

# Calculate resource allocation
calculate_resources() {
  local service="$1"
  local total_memory_mb="$2"
  local total_cpu_cores="$3"
  local service_count="$4"

  # Get service weight using Bash 3.2 compatible function
  local weight
  weight=$(get_service_weight "$service")

  # Check for manual override (Bash 3.2 compatible uppercase conversion)
  local service_upper
  service_upper=$(echo "$service" | tr '[:lower:]' '[:upper:]')
  local mem_var="${service_upper}_MEMORY_MB"
  local cpu_var="${service_upper}_CPU_MILLICORES"

  if [[ -n "${!mem_var:-}" ]]; then
    local memory_mb="${!mem_var}"
  else
    # Calculate based on weight and available resources
    local memory_mb=$((total_memory_mb * weight / (service_count * 5)))

    # Enforce minimum using Bash 3.2 compatible function
    local min_mem
    min_mem=$(get_min_memory "$service")
    [[ $memory_mb -lt $min_mem ]] && memory_mb=$min_mem
  fi

  if [[ -n "${!cpu_var:-}" ]]; then
    local cpu_millicores="${!cpu_var}"
  else
    # Calculate CPU (in millicores)
    local cpu_millicores=$((total_cpu_cores * 1000 * weight / (service_count * 5)))

    # Enforce minimum using Bash 3.2 compatible function
    local min_cpu
    min_cpu=$(get_min_cpu "$service")
    [[ $cpu_millicores -lt $min_cpu ]] && cpu_millicores=$min_cpu
  fi

  # Calculate reservation (50% of limit)
  local mem_reservation=$((memory_mb / 2))
  local cpu_reservation=$((cpu_millicores / 2))

  printf '{"service": "%s", "memory_limit_mb": %d, "memory_reservation_mb": %d, "cpu_limit_millicores": %d, "cpu_reservation_millicores": %d}\n' \
    "$service" "$memory_mb" "$mem_reservation" "$cpu_millicores" "$cpu_reservation"
}

# Generate resource limits for docker-compose
generate_compose_resources() {
  local service="$1"
  local limits="$2"

  local mem_limit=$(printf '%s\n' "$limits" | grep -o '"memory_limit_mb": *[0-9]*' | sed 's/"memory_limit_mb": *//')
  local mem_reservation=$(printf '%s\n' "$limits" | grep -o '"memory_reservation_mb": *[0-9]*' | sed 's/"memory_reservation_mb": *//')
  local cpu_limit=$(printf '%s\n' "$limits" | grep -o '"cpu_limit_millicores": *[0-9]*' | sed 's/"cpu_limit_millicores": *//')
  local cpu_reservation=$(printf '%s\n' "$limits" | grep -o '"cpu_reservation_millicores": *[0-9]*' | sed 's/"cpu_reservation_millicores": *//')

  # Convert millicores to Docker CPU format (1000 millicores = 1.0 CPU)
  local cpu_limit_docker=$(awk "BEGIN {printf \"%.2f\", $cpu_limit/1000}")
  local cpu_reservation_docker=$(awk "BEGIN {printf \"%.2f\", $cpu_reservation/1000}")

  cat <<EOF
    deploy:
      resources:
        limits:
          cpus: '${cpu_limit_docker}'
          memory: ${mem_limit}M
        reservations:
          cpus: '${cpu_reservation_docker}'
          memory: ${mem_reservation}M
EOF
}

# Main function to calculate all service resources
calculate_all_resources() {
  local output_format="${1:-text}"

  # Detect system resources
  local sys_resources=$(detect_system_resources)
  local total_memory=$(printf '%s\n' "$sys_resources" | grep -o '"available_memory_mb": *[0-9]*' | sed 's/"available_memory_mb": *//')
  local total_cpu=$(printf '%s\n' "$sys_resources" | grep -o '"available_cpu_cores": *[0-9]*' | sed 's/"available_cpu_cores": *//')

  # Get enabled services
  local services=($(get_enabled_services))
  local service_count=${#services[@]}

  if [[ "$output_format" == "json" ]]; then
    printf '{"system": %s, "services": [' "$sys_resources"

    local first=true
    for service in "${services[@]}"; do
      [[ "$first" != "true" ]] && printf ","
      first=false
      calculate_resources "$service" "$total_memory" "$total_cpu" "$service_count"
    done

    printf ']}\n'
  else
    # Text output
    printf "\n"
    printf "System Resources:\n"
    printf "  Total Memory: %d MB\n" "$total_memory"
    printf "  Total CPU: %d cores\n" "$total_cpu"
    printf "  Services: %d\n" "$service_count"
    printf "\n"
    printf "Resource Allocation:\n"
    printf "  %-20s %-12s %-12s %-12s %-12s\n" "Service" "Mem Limit" "Mem Reserve" "CPU Limit" "CPU Reserve"
    printf "  %-20s %-12s %-12s %-12s %-12s\n" "-------" "---------" "-----------" "---------" "-----------"

    for service in "${services[@]}"; do
      local limits=$(calculate_resources "$service" "$total_memory" "$total_cpu" "$service_count")
      local mem_limit=$(printf '%s\n' "$limits" | grep -o '"memory_limit_mb": *[0-9]*' | sed 's/"memory_limit_mb": *//')
      local mem_reservation=$(printf '%s\n' "$limits" | grep -o '"memory_reservation_mb": *[0-9]*' | sed 's/"memory_reservation_mb": *//')
      local cpu_limit=$(printf '%s\n' "$limits" | grep -o '"cpu_limit_millicores": *[0-9]*' | sed 's/"cpu_limit_millicores": *//')
      local cpu_reservation=$(printf '%s\n' "$limits" | grep -o '"cpu_reservation_millicores": *[0-9]*' | sed 's/"cpu_reservation_millicores": *//')

      printf "  %-20s %-12s %-12s %-12s %-12s\n" \
        "$service" \
        "${mem_limit}MB" \
        "${mem_reservation}MB" \
        "${cpu_limit}m" \
        "${cpu_reservation}m"
    done
    printf "\n"
  fi
}

# Check if resources meet minimum requirements
check_minimum_requirements() {
  local sys_resources=$(detect_system_resources)
  local total_memory=$(printf '%s\n' "$sys_resources" | grep -o '"available_memory_mb": *[0-9]*' | sed 's/"available_memory_mb": *//')
  local total_cpu=$(printf '%s\n' "$sys_resources" | grep -o '"available_cpu_cores": *[0-9]*' | sed 's/"available_cpu_cores": *//')

  # Minimum requirements for basic nself setup
  local min_memory_required=2048  # 2GB
  local min_cpu_required=2        # 2 cores

  local warnings=()

  if [[ $total_memory -lt $min_memory_required ]]; then
    warnings+=("WARNING: Available memory (${total_memory}MB) is below recommended minimum (${min_memory_required}MB)")
  fi

  if [[ $total_cpu -lt $min_cpu_required ]]; then
    warnings+=("WARNING: Available CPU (${total_cpu} cores) is below recommended minimum (${min_cpu_required} cores)")
  fi

  # Get service count
  local services=($(get_enabled_services))
  local service_count=${#services[@]}

  # Calculate total minimum memory needed
  local total_min_memory=0
  for service in "${services[@]}"; do
    local min_mem
    min_mem=$(get_min_memory "$service")
    total_min_memory=$((total_min_memory + min_mem))
  done

  if [[ $total_memory -lt $total_min_memory ]]; then
    warnings+=("ERROR: Insufficient memory! Need at least ${total_min_memory}MB for ${service_count} services, but only ${total_memory}MB available")
  fi

  if [[ ${#warnings[@]} -gt 0 ]]; then
    for warning in "${warnings[@]}"; do
      printf '%s\n' "$warning" >&2
    done
    [[ ${#warnings[@]} -gt 0 ]] && return 1
  fi

  return 0
}

# Export functions
export -f detect_system_resources get_enabled_services calculate_resources
export -f generate_compose_resources calculate_all_resources check_minimum_requirements

# If run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  case "${1:-check}" in
    check)
      check_minimum_requirements
      ;;
    calculate)
      calculate_all_resources "${2:-text}"
      ;;
    detect)
      detect_system_resources
      ;;
    *)
      printf "Usage: %s {check|calculate|detect}\n" "$0"
      exit 1
      ;;
  esac
fi
