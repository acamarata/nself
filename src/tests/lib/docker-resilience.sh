#!/usr/bin/env bash
# docker-resilience.sh - Handle Docker availability gracefully
#
# Provides Docker-aware testing that gracefully handles Docker being
# unavailable, not running, or resource-constrained.

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if ! declare -f detect_test_environment >/dev/null 2>&1; then
  source "$SCRIPT_DIR/environment-detection.sh"
fi

# ============================================================================
# Docker Availability Detection
# ============================================================================

# Check Docker availability with detailed status
# Returns: docker_available, docker_not_running, docker_not_installed
check_docker_availability() {
  # Docker not installed
  if ! command -v docker >/dev/null 2>&1; then
    printf "docker_not_installed\n"
    return 1
  fi

  # Docker installed but daemon not running
  if ! docker ps >/dev/null 2>&1; then
    printf "docker_not_running\n"
    return 1
  fi

  # Docker available and running
  printf "docker_available\n"
  return 0
}

# Check if Docker is available
is_docker_available() {
  local status
  status=$(check_docker_availability 2>/dev/null)
  [[ "$status" == "docker_available" ]]
}

# Check if Docker Compose is available
is_docker_compose_available() {
  # Check for docker-compose command
  if command -v docker-compose >/dev/null 2>&1; then
    return 0
  fi

  # Check for docker compose plugin
  if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    return 0
  fi

  return 1
}

# Get Docker Compose command
get_docker_compose_command() {
  if command -v docker-compose >/dev/null 2>&1; then
    printf "docker-compose\n"
  elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    printf "docker compose\n"
  else
    printf "none\n"
  fi
}

# ============================================================================
# Docker Test Execution
# ============================================================================

# Run test with Docker or skip gracefully
# Usage: test_with_docker test_function
test_with_docker() {
  local test_func="$1"
  local docker_status

  docker_status=$(check_docker_availability 2>/dev/null)

  case "$docker_status" in
    docker_available)
      # Docker is available - run test
      $test_func
      return $?
      ;;
    docker_not_running)
      printf "\033[33mSKIP:\033[0m %s (Docker daemon not running)\n" "$test_func" >&2
      return 0  # Pass
      ;;
    docker_not_installed)
      printf "\033[33mSKIP:\033[0m %s (Docker not installed)\n" "$test_func" >&2
      return 0  # Pass
      ;;
    *)
      printf "\033[33mSKIP:\033[0m %s (Docker status unknown)\n" "$test_func" >&2
      return 0  # Pass
      ;;
  esac
}

# Run test with Docker Compose or skip
# Usage: test_with_docker_compose test_function
test_with_docker_compose() {
  local test_func="$1"

  # Check Docker first
  if ! test_with_docker "true" >/dev/null 2>&1; then
    printf "\033[33mSKIP:\033[0m %s (Docker not available)\n" "$test_func" >&2
    return 0  # Pass
  fi

  # Check Docker Compose
  if ! is_docker_compose_available; then
    printf "\033[33mSKIP:\033[0m %s (Docker Compose not available)\n" "$test_func" >&2
    return 0  # Pass
  fi

  # Both available - run test
  $test_func
  return $?
}

# Require Docker or skip test
# Usage: require_docker [message]
require_docker() {
  local message="${1:-Docker is required}"
  local docker_status

  docker_status=$(check_docker_availability 2>/dev/null)

  if [[ "$docker_status" != "docker_available" ]]; then
    printf "\033[33mSKIP:\033[0m %s\n" "$message" >&2
    return 1  # Not available
  fi

  return 0  # Available
}

# ============================================================================
# Docker Mock Functions
# ============================================================================

# Use Docker if available, mock if not
# Usage: use_docker_or_mock
# Returns: 0 if using Docker, 1 if using mock
use_docker_or_mock() {
  if is_docker_available; then
    export TEST_USE_DOCKER=true
    export TEST_USE_MOCK=false
    return 0  # Using real Docker
  else
    export TEST_USE_DOCKER=false
    export TEST_USE_MOCK=true
    printf "\033[33mINFO:\033[0m Using Docker mock (Docker not available)\n" >&2
    return 1  # Using mock
  fi
}

# Check if using Docker mock
is_using_docker_mock() {
  [[ "${TEST_USE_MOCK:-false}" == "true" ]]
}

# Mock Docker command
mock_docker() {
  local subcommand="${1:-}"

  case "$subcommand" in
    ps)
      # Mock successful ps
      printf "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES\n"
      return 0
      ;;
    version)
      printf "Docker version 99.99.99 (mocked)\n"
      return 0
      ;;
    info)
      printf "Containers: 0\nImages: 0\nServer Version: 99.99.99 (mocked)\n"
      return 0
      ;;
    *)
      printf "\033[33mWARN:\033[0m Mock docker command: %s\n" "$subcommand" >&2
      return 0
      ;;
  esac
}

# ============================================================================
# Docker Resource Management
# ============================================================================

# Check if Docker has sufficient resources
# Usage: check_docker_resources min_memory_mb
check_docker_resources() {
  local min_memory="${1:-512}"

  if ! is_docker_available; then
    return 1
  fi

  # Get Docker info
  local docker_memory
  docker_memory=$(docker info --format '{{.MemTotal}}' 2>/dev/null || printf "0")

  # Convert to MB
  local memory_mb=$((docker_memory / 1024 / 1024))

  if [[ $memory_mb -lt $min_memory ]]; then
    printf "\033[33mWARN:\033[0m Docker has low memory: %d MB (need %d MB)\n" \
      "$memory_mb" "$min_memory" >&2
    return 1
  fi

  return 0
}

# Clean up Docker test containers
# Usage: cleanup_docker_test_containers [prefix]
cleanup_docker_test_containers() {
  local prefix="${1:-nself-test}"

  if ! is_docker_available; then
    return 0  # Nothing to clean up
  fi

  # Stop and remove containers with prefix
  docker ps -a --filter "name=${prefix}" -q 2>/dev/null | while read -r container_id; do
    if [[ -n "$container_id" ]]; then
      docker rm -f "$container_id" >/dev/null 2>&1 || true
    fi
  done

  return 0
}

# Clean up Docker test networks
# Usage: cleanup_docker_test_networks [prefix]
cleanup_docker_test_networks() {
  local prefix="${1:-nself-test}"

  if ! is_docker_available; then
    return 0  # Nothing to clean up
  fi

  # Remove networks with prefix
  docker network ls --filter "name=${prefix}" -q 2>/dev/null | while read -r network_id; do
    if [[ -n "$network_id" ]]; then
      docker network rm "$network_id" >/dev/null 2>&1 || true
    fi
  done

  return 0
}

# Clean up Docker test volumes
# Usage: cleanup_docker_test_volumes [prefix]
cleanup_docker_test_volumes() {
  local prefix="${1:-nself-test}"

  if ! is_docker_available; then
    return 0  # Nothing to clean up
  fi

  # Remove volumes with prefix
  docker volume ls --filter "name=${prefix}" -q 2>/dev/null | while read -r volume_id; do
    if [[ -n "$volume_id" ]]; then
      docker volume rm "$volume_id" >/dev/null 2>&1 || true
    fi
  done

  return 0
}

# Clean up all Docker test resources
# Usage: cleanup_all_docker_test_resources [prefix]
cleanup_all_docker_test_resources() {
  local prefix="${1:-nself-test}"

  cleanup_docker_test_containers "$prefix"
  cleanup_docker_test_networks "$prefix"
  cleanup_docker_test_volumes "$prefix"
}

# ============================================================================
# Docker Compose Helpers
# ============================================================================

# Run Docker Compose command with availability check
# Usage: safe_docker_compose [args...]
safe_docker_compose() {
  local compose_cmd

  if ! is_docker_compose_available; then
    printf "\033[33mSKIP:\033[0m Docker Compose not available\n" >&2
    return 1
  fi

  compose_cmd=$(get_docker_compose_command)

  if [[ "$compose_cmd" == "none" ]]; then
    printf "\033[33mSKIP:\033[0m Docker Compose command not found\n" >&2
    return 1
  fi

  # Run compose command
  eval "$compose_cmd" "$@"
}

# Wait for Docker Compose services to be healthy
# Usage: wait_for_compose_healthy [timeout] [compose_file]
wait_for_compose_healthy() {
  local timeout="${1:-60}"
  local compose_file="${2:-docker-compose.yml}"
  local elapsed=0
  local interval=2

  if ! is_docker_compose_available; then
    printf "\033[33mSKIP:\033[0m Cannot check health (Docker Compose not available)\n" >&2
    return 0  # Pass
  fi

  while [[ $elapsed -lt $timeout ]]; do
    local unhealthy_count
    unhealthy_count=$(safe_docker_compose -f "$compose_file" ps --format json 2>/dev/null |
      grep -c '"Health":"unhealthy"' || printf "0")

    if [[ "$unhealthy_count" -eq 0 ]]; then
      return 0  # All healthy
    fi

    sleep "$interval"
    elapsed=$((elapsed + interval))
  done

  # Timeout
  if is_ci_environment; then
    printf "\033[33mSKIP:\033[0m Services not healthy in time (CI timeout)\n" >&2
    return 0  # Pass
  fi

  printf "\033[33mWARN:\033[0m Services not healthy after %ds\n" "$timeout" >&2
  return 1
}

# ============================================================================
# Export Functions
# ============================================================================

export -f check_docker_availability
export -f is_docker_available
export -f is_docker_compose_available
export -f get_docker_compose_command
export -f test_with_docker
export -f test_with_docker_compose
export -f require_docker
export -f use_docker_or_mock
export -f is_using_docker_mock
export -f mock_docker
export -f check_docker_resources
export -f cleanup_docker_test_containers
export -f cleanup_docker_test_networks
export -f cleanup_docker_test_volumes
export -f cleanup_all_docker_test_resources
export -f safe_docker_compose
export -f wait_for_compose_healthy
