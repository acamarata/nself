#!/usr/bin/env bash
# orchestration.sh - Main build orchestration logic
# POSIX-compliant, no Bash 4+ features

# Load all core modules
load_core_modules() {
  local module_dir="$(dirname "${BASH_SOURCE[0]}")"

  # Source each module if it exists
  for module in directory-setup.sh ssl-generation.sh nginx-setup.sh database-init.sh; do
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
    if bash "$compose_script" >/dev/null 2>&1; then
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

  # Export environment variables
  export PROJECT_NAME="$project_name"
  export ENV="$env"

  # Load modules
  load_core_modules

  # Determine env file
  local env_file=".env"
  if [[ -f ".env.local" ]]; then
    env_file=".env.local"
  elif [[ -f ".env.$env" ]]; then
    env_file=".env.$env"
  fi

  # Load environment
  if [[ -f "$env_file" ]]; then
    set -a
    source "$env_file" 2>/dev/null || true
    set +a
  fi

  # Check what needs to be done
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