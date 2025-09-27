#!/usr/bin/env bash
# orchestration.sh - Main build orchestration logic
# POSIX-compliant, no Bash 4+ features

# Load all core modules
load_core_modules() {
  local module_dir="$(dirname "${BASH_SOURCE[0]}")"
  local lib_dir="$(dirname "$module_dir")"

  # Source service detection first
  if [[ -f "$lib_dir/service-detection.sh" ]]; then
    source "$lib_dir/service-detection.sh"
  fi

  # Source each module if it exists
  for module in directory-setup.sh ssl-generation.sh nginx-setup.sh database-init.sh mlflow-setup.sh; do
    if [[ -f "$module_dir/$module" ]]; then
      source "$module_dir/$module"
    fi
  done
}

# Check what needs to be built
check_build_requirements() {
  local force_rebuild="${1:-false}"
  local env_file="${2:-.env}"

  # Initialize requirement flags
  export NEEDS_DIRECTORIES=false
  export NEEDS_SSL=false
  export NEEDS_NGINX=false
  export NEEDS_DATABASE=false
  export NEEDS_COMPOSE=false
  export NEEDS_SERVICES=false

  # Check directory structure
  if command -v check_directory_structure >/dev/null 2>&1; then
    local missing_dirs=$(check_directory_structure)
    if [[ $missing_dirs -gt 0 ]]; then
      NEEDS_DIRECTORIES=true
    fi
  fi

  # Check SSL certificates
  if [[ ! -f "ssl/certificates/localhost/fullchain.pem" ]] || [[ "$force_rebuild" == "true" ]]; then
    NEEDS_SSL=true
  fi

  # Check nginx configuration
  if [[ ! -f "nginx/nginx.conf" ]] || [[ ! -f "nginx/conf.d/default.conf" ]] || [[ "$force_rebuild" == "true" ]]; then
    NEEDS_NGINX=true
  elif [[ -f "$env_file" ]] && [[ "$env_file" -nt "nginx/nginx.conf" ]]; then
    NEEDS_NGINX=true
  fi

  # Check database initialization
  if [[ "${POSTGRES_ENABLED:-false}" == "true" ]]; then
    if [[ ! -f "postgres/init/00-init.sql" ]] || [[ "$force_rebuild" == "true" ]]; then
      NEEDS_DATABASE=true
    elif [[ -f "$env_file" ]] && [[ "$env_file" -nt "postgres/init/00-init.sql" ]]; then
      NEEDS_DATABASE=true
    fi
  fi

  # Check docker-compose.yml
  if [[ ! -f "docker-compose.yml" ]] || [[ "$force_rebuild" == "true" ]]; then
    NEEDS_COMPOSE=true
  elif [[ -f "$env_file" ]] && [[ "$env_file" -nt "docker-compose.yml" ]]; then
    NEEDS_COMPOSE=true
  fi

  # Check monitoring configuration if enabled
  if [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
    export NEEDS_MONITORING=false
    if [[ ! -d "monitoring" ]] || [[ ! -f "monitoring/prometheus/prometheus.yml" ]]; then
      NEEDS_MONITORING=true
    elif [[ -f "$env_file" ]] && [[ "$env_file" -nt "monitoring/prometheus/prometheus.yml" ]]; then
      NEEDS_MONITORING=true
    fi
  fi

  # Check custom services from templates
  if [[ -n "${CUSTOM_SERVICES:-}" ]]; then
    export NEEDS_CUSTOM_SERVICES=false
    for service in $CUSTOM_SERVICES; do
      if [[ ! -d "services/$service" ]]; then
        NEEDS_CUSTOM_SERVICES=true
        break
      fi
    done
  fi

  # Check if fallback services are needed
  if [[ "${ENV:-}" == "demo" ]] || [[ "${DEMO_CONTENT:-false}" == "true" ]]; then
    if [[ ! -d "fallback-services" ]] || [[ "$force_rebuild" == "true" ]]; then
      export NEEDS_FALLBACKS=true
    fi
  fi

  # Check if any work is needed
  if [[ "$NEEDS_DIRECTORIES" == "true" ]] || \
     [[ "$NEEDS_SSL" == "true" ]] || \
     [[ "$NEEDS_NGINX" == "true" ]] || \
     [[ "$NEEDS_DATABASE" == "true" ]] || \
     [[ "$NEEDS_COMPOSE" == "true" ]]; then
    return 0
  else
    return 1
  fi
}

# Execute build steps
execute_build_steps() {
  local force_rebuild="${1:-false}"

  # Track what was done
  local steps_completed=0
  local steps_failed=0

  # Create directories if needed
  if [[ "$NEEDS_DIRECTORIES" == "true" ]]; then
    if command -v setup_project_directories >/dev/null 2>&1; then
      if setup_project_directories; then
        steps_completed=$((steps_completed + 1))
      else
        steps_failed=$((steps_failed + 1))
        echo "Failed to create directory structure" >&2
      fi
    fi
  fi

  # Generate SSL certificates if needed
  if [[ "$NEEDS_SSL" == "true" ]]; then
    if command -v setup_ssl_certificates >/dev/null 2>&1; then
      if setup_ssl_certificates "$force_rebuild"; then
        steps_completed=$((steps_completed + 1))
      else
        steps_failed=$((steps_failed + 1))
        echo "Failed to generate SSL certificates" >&2
      fi
    fi
  fi

  # Generate nginx configuration if needed
  if [[ "$NEEDS_NGINX" == "true" ]]; then
    if command -v setup_nginx >/dev/null 2>&1; then
      if setup_nginx; then
        steps_completed=$((steps_completed + 1))
      else
        steps_failed=$((steps_failed + 1))
        echo "Failed to generate nginx configuration" >&2
      fi
    fi
  fi

  # Generate database initialization if needed
  if [[ "$NEEDS_DATABASE" == "true" ]]; then
    if command -v generate_postgres_init >/dev/null 2>&1; then
      if generate_postgres_init; then
        steps_completed=$((steps_completed + 1))
      else
        steps_failed=$((steps_failed + 1))
        echo "Failed to generate database initialization" >&2
      fi
    fi
  fi

  # Generate custom services from templates
  if [[ -n "${CUSTOM_SERVICES:-}" ]]; then
    if command -v generate_custom_services >/dev/null 2>&1; then
      if generate_custom_services; then
        steps_completed=$((steps_completed + 1))
      else
        steps_failed=$((steps_failed + 1))
        echo "Failed to generate custom services" >&2
      fi
    fi
  fi

  # Generate fallback services for demo/test environments
  if [[ "${NEEDS_FALLBACKS:-false}" == "true" ]] || [[ "${ENV:-}" == "demo" ]] || [[ "${DEMO_CONTENT:-false}" == "true" ]]; then
    if command -v generate_fallback_services >/dev/null 2>&1; then
      if generate_fallback_services "$PWD"; then
        steps_completed=$((steps_completed + 1))
      else
        echo "Warning: Failed to generate fallback services" >&2
      fi
    fi
  fi

  # Generate MLflow Dockerfile if needed
  if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
    if command -v generate_mlflow_dockerfile >/dev/null 2>&1; then
      generate_mlflow_dockerfile "$PWD/mlflow"
    fi
  fi

  # Setup monitoring configs if needed
  if command -v setup_monitoring_configs >/dev/null 2>&1; then
    setup_monitoring_configs
  fi

  # Generate docker-compose.yml if needed
  if [[ "$NEEDS_COMPOSE" == "true" ]]; then
    if generate_docker_compose; then
      steps_completed=$((steps_completed + 1))
    else
      steps_failed=$((steps_failed + 1))
      echo "Failed to generate docker-compose.yml" >&2
    fi
  fi

  # Generate nginx configuration with new generator
  if [[ -f "$(dirname "${BASH_SOURCE[0]}")/nginx-generator.sh" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/nginx-generator.sh"
    generate_nginx_config "$force_rebuild"
  fi

  # Return status
  if [[ $steps_failed -gt 0 ]]; then
    return 1
  fi

  return 0
}

# Generate docker-compose.yml
generate_docker_compose() {
  local compose_script="${LIB_ROOT:-/usr/local/lib/nself}/../services/docker/compose-generate.sh"

  if [[ -f "$compose_script" ]]; then
    # Run the compose generation with clean output (disable tracing)
    if (set +x; bash "$compose_script") >/dev/null 2>&1; then
      # Apply health check fixes if available
      if [[ -f "${LIB_ROOT:-/usr/local/lib/nself}/auto-fix/healthcheck-fix.sh" ]]; then
        source "${LIB_ROOT:-/usr/local/lib/nself}/auto-fix/healthcheck-fix.sh"
        fix_healthchecks "docker-compose.yml" >/dev/null 2>&1
      fi
      return 0
    fi
  fi

  return 1
}

# Run post-build tasks
run_post_build_tasks() {
  # Apply auto-fixes
  if [[ -f "${LIB_ROOT:-/usr/local/lib/nself}/auto-fix/core.sh" ]]; then
    source "${LIB_ROOT:-/usr/local/lib/nself}/auto-fix/core.sh"
    if command -v apply_all_auto_fixes >/dev/null 2>&1; then
      apply_all_auto_fixes >/dev/null 2>&1
    fi
  fi

  # Validate docker-compose.yml
  if [[ -f "docker-compose.yml" ]]; then
    docker compose config >/dev/null 2>&1 || true
  fi

  # Set proper permissions
  if command -v set_directory_permissions >/dev/null 2>&1; then
    set_directory_permissions
  fi

  return 0
}

# Main orchestration function
orchestrate_modular_build() {
  local project_name="${1:-$(basename "$PWD")}"
  local env="${2:-dev}"
  local force="${3:-false}"

  # Use new orchestrator if available
  if [[ -f "$(dirname "${BASH_SOURCE[0]}")/build-orchestrator.sh" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/build-orchestrator.sh"
    orchestrate_build "$project_name" "$env" "$force"
    return $?
  fi

  # Export environment variables with smart defaults
  # Project name - use ternary: explicit > parameter > directory
  export PROJECT_NAME="${PROJECT_NAME:-${project_name:-$(basename "$PWD")}}"

  # Environment - use ternary: NSELF_ENV > ENV > parameter > default
  export ENV="${NSELF_ENV:-${ENV:-${env:-dev}}}"

  # Detect all enabled services using smart defaults
  if command -v detect_enabled_services >/dev/null 2>&1; then
    detect_enabled_services
    detect_custom_services
    detect_frontend_apps
  fi

  # Load modules
  load_core_modules

  # IMPORTANT: Load environment cascade for service detection
  # While build outputs are environment-agnostic (use ternary patterns),
  # we need to know which services are configured
  if command -v cascade_env_vars >/dev/null 2>&1; then
    cascade_env_vars "$ENV"
  else
    # Fallback: load env files in cascade order
    if [[ -f ".env.dev" ]]; then
      set -a
      source ".env.dev" 2>/dev/null || true
      set +a
    fi
    if [[ -f ".env" ]]; then
      set -a
      source ".env" 2>/dev/null || true
      set +a
    fi
  fi

  # Set smart defaults for common variables (can be overridden by explicitly set env vars)
  export BASE_DOMAIN="${BASE_DOMAIN:-localhost}"
  export POSTGRES_PORT="${POSTGRES_PORT:-5432}"
  export HASURA_PORT="${HASURA_PORT:-8080}"
  export AUTH_PORT="${AUTH_PORT:-4000}"
  export STORAGE_PORT="${STORAGE_PORT:-5000}"
  export REDIS_PORT="${REDIS_PORT:-6379}"
  export MINIO_PORT="${MINIO_PORT:-9000}"
  export MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"

  # Check what needs to be done (use .env for timestamp check only)
  local env_file=".env"
  if ! check_build_requirements "$force" "$env_file"; then
    echo "Everything is up to date"
    return 0
  fi

  # Execute build steps
  if ! execute_build_steps "$force"; then
    echo "Build failed" >&2
    return 1
  fi

  # Run post-build tasks
  run_post_build_tasks

  echo "Build completed successfully"
  return 0
}

# Export functions
export -f load_core_modules
export -f check_build_requirements
export -f execute_build_steps
export -f generate_docker_compose
export -f run_post_build_tasks
export -f orchestrate_modular_build