#!/bin/bash

# Main dispatcher for auto-fix functionality
# Analyzes errors and delegates to specific handlers

# Source all autofix handlers
AUTOFIX_DIR="$(dirname "${BASH_SOURCE[0]}")"
for handler in "$AUTOFIX_DIR"/*.sh; do
  if [[ -f "$handler" ]] && [[ "$handler" != *"dispatcher.sh" ]]; then
    source "$handler"
  fi
done

# Initialize state tracking
init_autofix_state

analyze_and_fix_service() {
  local service_name="$1"
  local service_logs="$2"

  # Remove unity_ prefix for cleaner display
  local clean_name="${service_name#unity_}"

  # Get attempt count and last strategy
  local attempts=$(get_service_attempts "$service_name")
  local last_strategy=$(get_last_fix_strategy "$service_name")

  # Check if it's the same error repeating
  local same_error=false
  if is_same_error "$service_name" "$service_logs"; then
    same_error=true
  fi

  # Prevent infinite loops
  if [[ $attempts -gt 5 ]]; then
    log_error "Auto-fix attempted $attempts times for $clean_name, giving up"
    return 1
  fi

  log_info "Analyzing $clean_name (attempt $((attempts + 1)), last strategy: $last_strategy)"

  # Postgres connection issues
  if echo "$service_logs" | grep -q "postgres.*connection\|connection.*postgres\|port 543[0-9]"; then
    log_info "Detected Postgres connection issue"

    # Progressive fix strategies based on attempts
    local strategy=""
    if [[ $attempts -eq 0 ]]; then
      strategy="restart_postgres"
    elif [[ $attempts -eq 1 ]] && [[ "$same_error" == "true" ]]; then
      strategy="check_port_config"
    elif [[ $attempts -eq 2 ]] && [[ "$same_error" == "true" ]]; then
      strategy="recreate_network"
    elif [[ $attempts -eq 3 ]] && [[ "$same_error" == "true" ]]; then
      strategy="full_database_reset"
    else
      strategy="recreate_service"
    fi

    record_fix_attempt "$service_name" "$strategy"
    fix_postgres_connection "$service_name" "$service_logs" "$strategy"
    return $?
  fi

  # Config server health check issues
  if [[ "$clean_name" == "config-server" ]] || [[ "$clean_name" == "config_server" ]]; then
    log_info "Detected config-server health issue"

    # Progressive fix strategies
    local strategy=""

    # Check if files exist first
    if [[ ! -f "config-server/index.js" ]] && [[ $attempts -eq 0 ]]; then
      # Files don't exist, skip straight to regeneration
      strategy="regenerate_files"
    elif [[ $attempts -eq 0 ]]; then
      strategy="wait_for_health"
    elif [[ $attempts -eq 1 ]] && [[ "$same_error" == "true" ]]; then
      strategy="regenerate_files"
    elif [[ $attempts -eq 2 ]] && [[ "$same_error" == "true" ]]; then
      strategy="remove_health_check"
    else
      strategy="recreate_service"
    fi

    record_fix_attempt "$service_name" "$strategy"
    fix_config_server "$service_name" "$service_logs" "$strategy"
    return $?
  fi

  # Redis connection issues
  if echo "$service_logs" | grep -q "redis.*connection\|connection.*redis\|port 637[0-9]"; then
    log_info "Detected Redis connection issue"
    fix_redis_connection "$service_name" "$service_logs"
    return $?
  fi

  # Generic network/DNS issues
  if echo "$service_logs" | grep -q "Unknown or invalid host\|cannot resolve\|Name or service not known"; then
    log_info "Detected network/DNS issue"
    fix_network_issue "$service_name" "$service_logs"
    return $?
  fi

  # Permission issues
  if echo "$service_logs" | grep -q "Permission denied\|permission denied\|EACCES"; then
    log_info "Detected permission issue"
    fix_permission_issue "$service_name" "$service_logs"
    return $?
  fi

  # Missing file/directory issues
  if echo "$service_logs" | grep -q "No such file or directory\|cannot find\|not found"; then
    log_info "Detected missing file/directory issue"
    fix_missing_files "$service_name" "$service_logs"
    return $?
  fi

  # Default fallback: recreate the service
  log_info "No specific fix identified, attempting service recreation"
  docker compose stop "$service_name" >/dev/null 2>&1
  docker compose rm -f "$service_name" >/dev/null 2>&1

  # For certain services, regenerate files
  case "$clean_name" in
  config-server | config_server)
    if [[ ! -f "config-server/index.js" ]]; then
      log_info "Regenerating config-server files"
      if [[ -f "$AUTOFIX_DIR/../auto-fix/dockerfile-generator.sh" ]]; then
        source "$AUTOFIX_DIR/../auto-fix/dockerfile-generator.sh"
        generate_dockerfile_for_service "config-server" "config-server"
      fi
    fi
    ;;
  esac

  return 99 # Retry
}

# Additional generic fix functions that don't warrant separate files

fix_redis_connection() {
  local service_name="$1"
  local service_logs="$2"

  log_info "Checking Redis availability"

  # Ensure Redis is running
  docker compose up -d redis >/dev/null 2>&1

  # Wait for Redis to be ready
  local max_wait=15
  local waited=0
  while [[ $waited -lt $max_wait ]]; do
    if docker exec unity_redis redis-cli ping >/dev/null 2>&1; then
      log_success "Redis is ready"

      # Restart the dependent service
      docker compose stop "$service_name" >/dev/null 2>&1
      docker compose rm -f "$service_name" >/dev/null 2>&1
      return 99 # Retry
    fi
    sleep 1
    ((waited++))
  done

  log_error "Redis failed to start after ${max_wait} seconds"
  return 1
}

fix_network_issue() {
  local service_name="$1"
  local service_logs="$2"

  log_info "Fixing network/DNS issues"

  # Recreate the Docker network
  local network_name="unity_default"
  docker network inspect $network_name >/dev/null 2>&1 || {
    log_info "Recreating Docker network"
    docker network create $network_name >/dev/null 2>&1
  }

  # Restart the service
  docker compose stop "$service_name" >/dev/null 2>&1
  docker compose rm -f "$service_name" >/dev/null 2>&1
  return 99 # Retry
}

fix_permission_issue() {
  local service_name="$1"
  local service_logs="$2"

  log_info "Fixing permission issues"

  # Try to fix common permission issues
  if [[ -d "./data" ]]; then
    chmod -R 755 ./data 2>/dev/null
  fi

  # Restart the service
  docker compose stop "$service_name" >/dev/null 2>&1
  docker compose rm -f "$service_name" >/dev/null 2>&1
  return 99 # Retry
}

fix_missing_files() {
  local service_name="$1"
  local service_logs="$2"
  local clean_name="${service_name#unity_}"

  log_info "Fixing missing files for $clean_name"

  # Service-specific file generation
  case "$clean_name" in
  config-server | config_server)
    mkdir -p config-server
    if [[ -f "$AUTOFIX_DIR/../auto-fix/dockerfile-generator.sh" ]]; then
      source "$AUTOFIX_DIR/../auto-fix/dockerfile-generator.sh"
      generate_dockerfile_for_service "config-server" "config-server"
    fi
    ;;
  esac

  # Rebuild configuration
  nself build --force >/dev/null 2>&1
  return 99 # Retry
}
