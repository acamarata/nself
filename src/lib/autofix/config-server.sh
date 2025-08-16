#!/bin/bash

fix_config_server() {
  local service_name="$1"
  local service_logs="$2"
  local strategy="${3:-wait_for_health}"

  log_info "Applying strategy: $strategy"

  case "$strategy" in
  wait_for_health)
    # First attempt: Check if service is actually working
    if echo "$service_logs" | grep -q "Config server listening on port"; then
      log_info "Config server is running, checking actual health"

      # Try to access the health endpoint
      local config_port=$(docker port unity_config-server 4001 2>/dev/null | cut -d: -f2)
      if [[ -z "$config_port" ]]; then
        config_port=4001
      fi

      # Test the health endpoint
      if curl -s -f "http://localhost:$config_port/healthz" >/dev/null 2>&1; then
        log_success "Config server health endpoint is responding"

        # Wait a bit more for Docker to recognize it
        sleep 5

        local health_status=$(docker inspect unity_config-server --format='{{.State.Health.Status}}' 2>/dev/null)
        if [[ "$health_status" == "healthy" ]]; then
          log_success "Config server is now healthy"
          return 0
        else
          log_warning "Health endpoint works but Docker still reports unhealthy"
          # Continue anyway since the service is actually working
          return 0
        fi
      else
        log_warning "Health endpoint not responding, files may be missing"
        # Try regenerating files
        return 99
      fi
    fi
    return 99 # Try next strategy
    ;;

  regenerate_files)
    # Second attempt: Regenerate config-server files
    log_info "Regenerating config-server files"

    # Stop and remove the container
    docker compose stop config-server >/dev/null 2>&1
    docker compose rm -f config-server >/dev/null 2>&1

    # Create config-server directory if it doesn't exist
    mkdir -p config-server

    # Force regeneration of files
    log_info "Generating config server files"

    # Find the dockerfile generator
    local generator_paths=(
      "$SCRIPT_DIR/../auto-fix/dockerfile-generator.sh"
      "/Users/admin/Sites/nself/src/lib/auto-fix/dockerfile-generator.sh"
    )

    local generator_found=false
    for generator in "${generator_paths[@]}"; do
      if [[ -f "$generator" ]]; then
        source "$generator"
        generate_config_server "config-server"
        generator_found=true
        break
      fi
    done

    if [[ "$generator_found" != "true" ]]; then
      log_error "Could not find dockerfile generator"
      return 1
    fi

    # Verify files were created
    if [[ -f "config-server/index.js" ]] && [[ -f "config-server/Dockerfile" ]]; then
      log_success "Config server files created"

      # Rebuild
      nself build --force >/dev/null 2>&1

      # Start the service
      docker compose up -d config-server >/dev/null 2>&1
      sleep 5
    else
      log_error "Failed to create config server files"
      return 1
    fi

    return 99 # Retry
    ;;

  remove_health_check)
    # Third attempt: Continue without health check
    log_info "Removing health check requirement for config-server"

    # The service is actually running, just mark it as ok
    if echo "$service_logs" | grep -q "Config server listening on port"; then
      log_warning "Config server is running but health check failing, continuing anyway"

      # Try to update docker-compose to remove health check
      # This would need to be done in the compose file generation

      # For now, just recreate without waiting for health
      docker compose stop config-server >/dev/null 2>&1
      docker compose rm -f config-server >/dev/null 2>&1
      docker compose up -d config-server >/dev/null 2>&1

      # Don't wait for health, just check if running
      sleep 3
      if docker ps | grep -q unity_config-server; then
        log_success "Config server is running (ignoring health check)"
        return 0
      fi
    fi
    return 99 # Try next strategy
    ;;

  recreate_service)
    # Fallback: Just recreate
    log_info "Recreating config-server"
    docker compose stop config-server >/dev/null 2>&1
    docker compose rm -f config-server >/dev/null 2>&1

    # Ensure files exist
    if [[ ! -f "config-server/index.js" ]]; then
      mkdir -p config-server
      if [[ -f "$SCRIPT_DIR/../auto-fix/dockerfile-generator.sh" ]]; then
        source "$SCRIPT_DIR/../auto-fix/dockerfile-generator.sh"
        generate_dockerfile_for_service "config-server" "config-server"
      fi
    fi

    return 99 # Retry
    ;;

  *)
    log_error "Unknown strategy: $strategy"
    return 1
    ;;
  esac
}
