#!/usr/bin/env bash
# docker.sh - Centralized Docker utilities

# Source display utilities
UTILS_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$UTILS_DIR/display.sh" 2>/dev/null || true

# Ensure Docker is running
ensure_docker_running() {
  if ! docker info >/dev/null 2>&1; then
    log_error "Docker is not running"
    log_info "Please start Docker Desktop or run: sudo systemctl start docker"
    return 1
  fi
  return 0
}

# Docker Compose wrapper - enforces v2 and consistent options
compose() {
  local env_file="${COMPOSE_ENV_FILE:-.env.local}"
  local project="${PROJECT_NAME:-nself}"
  local compose_files=""
  
  # Always use main docker-compose.yml
  compose_files="-f docker-compose.yml"
  
  # Add custom services if exists
  if [[ -f "docker-compose.custom.yml" ]]; then
    compose_files="$compose_files -f docker-compose.custom.yml"
  fi
  
  # Add override if exists
  if [[ -f "docker-compose.override.yml" ]]; then
    compose_files="$compose_files -f docker-compose.override.yml"
  fi

  if [[ -f "$env_file" ]]; then
    docker compose $compose_files --project-name "$project" --env-file "$env_file" "$@"
  else
    docker compose $compose_files --project-name "$project" "$@"
  fi
}

# Get project name
project_name() {
  echo "${PROJECT_NAME:-nself}"
}

# Get container name for a service
container_name() {
  local service="$1"
  echo "$(project_name)_${service}_1"
}

# Check if service is running
is_service_running() {
  local service="$1"
  compose ps --services --filter "status=running" 2>/dev/null | grep -qx "$service"
}

# Get service health status
service_health() {
  local service="$1"
  local container=$(container_name "$service")

  local status=$(docker inspect "$container" --format='{{.State.Health.Status}}' 2>/dev/null)

  if [[ -z "$status" ]]; then
    # No health check defined, check if running
    if docker inspect "$container" --format='{{.State.Status}}' 2>/dev/null | grep -q "running"; then
      echo "running"
    else
      echo "stopped"
    fi
  else
    echo "$status"
  fi
}

# Wait for service to be healthy
wait_service_healthy() {
  local service="$1"
  local timeout="${2:-60}"
  local start_time=$(date +%s)

  log_info "Waiting for $service to be healthy..."

  while [[ $(($(date +%s) - start_time)) -lt $timeout ]]; do
    local health=$(service_health "$service")

    case "$health" in
    healthy | running)
      log_success "$service is healthy"
      return 0
      ;;
    starting)
      sleep 2
      ;;
    unhealthy | stopped)
      log_error "$service is $health"
      return 1
      ;;
    esac
  done

  log_warning "Timeout waiting for $service to be healthy"
  return 1
}

# Kill containers using a port (only our project)
kill_port_if_ours() {
  local port="$1"
  local project=$(project_name)

  # Find containers from our project using this port
  local containers=$(docker ps --filter "publish=$port" --format '{{.Names}}' | grep -E "^${project}_" || true)

  if [[ -n "$containers" ]]; then
    log_info "Stopping our containers using port $port"
    echo "$containers" | xargs -r docker stop >/dev/null 2>&1 || true
    echo "$containers" | xargs -r docker rm >/dev/null 2>&1 || true
    return 0
  fi

  return 1
}

# Check if port is available
is_port_available() {
  local port="$1"

  # Check with multiple methods for compatibility
  if command -v lsof >/dev/null 2>&1; then
    ! lsof -i ":$port" >/dev/null 2>&1
  elif command -v netstat >/dev/null 2>&1; then
    ! netstat -tln 2>/dev/null | grep -q ":$port "
  elif command -v ss >/dev/null 2>&1; then
    ! ss -tln 2>/dev/null | grep -q ":$port "
  else
    # Fallback: try to bind to the port
    ! nc -z localhost "$port" 2>/dev/null
  fi
}

# Get all ports used by our project
ports_in_use_for_project() {
  local project=$(project_name)
  docker ps --filter "name=${project}_" --format '{{.Ports}}' |
    grep -oE '[0-9]+' | sort -u
}

# Clean up stopped containers from our project
cleanup_stopped_containers() {
  local project=$(project_name)
  local containers=$(docker ps -a --filter "name=${project}_" --filter "status=exited" --format '{{.Names}}')

  if [[ -n "$containers" ]]; then
    log_info "Cleaning up stopped containers"
    echo "$containers" | xargs -r docker rm >/dev/null 2>&1 || true
  fi
}

# Get Docker Compose config
get_compose_config() {
  compose config 2>/dev/null
}

# Validate Docker Compose file
validate_compose_file() {
  if compose config -q 2>/dev/null; then
    return 0
  else
    log_error "Docker Compose configuration is invalid"
    compose config 2>&1 | head -20
    return 1
  fi
}

# Export all functions
export -f ensure_docker_running compose project_name container_name
export -f is_service_running service_health wait_service_healthy
export -f kill_port_if_ours is_port_available ports_in_use_for_project
export -f cleanup_stopped_containers get_compose_config validate_compose_file
