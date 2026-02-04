#!/usr/bin/env bash
# docker-mock.sh - Mock Docker API for fast, deterministic testing
#
# Provides mock Docker commands that simulate container operations
# without requiring Docker to be installed or running.

set -euo pipefail

# ============================================================================
# Mock State Management
# ============================================================================

MOCK_DOCKER_STATE_DIR="${MOCK_DOCKER_STATE_DIR:-/tmp/mock-docker-$$}"
mkdir -p "$MOCK_DOCKER_STATE_DIR"

# Initialize mock state
init_docker_mock() {
  mkdir -p "$MOCK_DOCKER_STATE_DIR/containers"
  mkdir -p "$MOCK_DOCKER_STATE_DIR/images"
  mkdir -p "$MOCK_DOCKER_STATE_DIR/networks"
  mkdir -p "$MOCK_DOCKER_STATE_DIR/volumes"
}

# Cleanup mock state
cleanup_docker_mock() {
  [[ -d "$MOCK_DOCKER_STATE_DIR" ]] && rm -rf "$MOCK_DOCKER_STATE_DIR"
}

# ============================================================================
# Mock Docker Commands
# ============================================================================

# Mock docker ps
mock_docker_ps() {
  local format="${2:-}"
  local all=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -a|--all)
        all=true
        shift
        ;;
      --format)
        format="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done

  # List containers
  if [[ "$all" == true ]]; then
    find "$MOCK_DOCKER_STATE_DIR/containers" -type f 2>/dev/null || true
  else
    find "$MOCK_DOCKER_STATE_DIR/containers" -type f -name "*_running" 2>/dev/null || true
  fi | while read -r container_file; do
    local container_name
    container_name=$(basename "$container_file" | sed 's/_running$//')
    printf "%s\n" "$container_name"
  done
}

# Mock docker run
mock_docker_run() {
  local container_name=""
  local image=""
  local detach=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --name)
        container_name="$2"
        shift 2
        ;;
      -d|--detach)
        detach=true
        shift
        ;;
      -p|--publish)
        # Ignore port mappings
        shift 2
        ;;
      -e|--env)
        # Ignore environment variables
        shift 2
        ;;
      -v|--volume)
        # Ignore volumes
        shift 2
        ;;
      --network)
        # Ignore network
        shift 2
        ;;
      *)
        if [[ -z "$image" ]]; then
          image="$1"
        fi
        shift
        ;;
    esac
  done

  # Generate container ID if no name provided
  if [[ -z "$container_name" ]]; then
    container_name="mock_$(date +%s)_$$"
  fi

  # Create container state
  local container_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running"
  printf "image=%s\nstatus=running\nstarted=%s\n" "$image" "$(date +%s)" > "$container_file"

  # Return container ID
  printf "%s\n" "$container_name"
}

# Mock docker stop
mock_docker_stop() {
  local container_name="$1"
  local container_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running"

  if [[ -f "$container_file" ]]; then
    # Change state to stopped
    mv "$container_file" "$MOCK_DOCKER_STATE_DIR/containers/${container_name}_stopped"
    printf "%s\n" "$container_name"
  else
    printf "Error: No such container: %s\n" "$container_name" >&2
    return 1
  fi
}

# Mock docker start
mock_docker_start() {
  local container_name="$1"
  local container_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_stopped"

  if [[ -f "$container_file" ]]; then
    # Change state to running
    mv "$container_file" "$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running"
    printf "%s\n" "$container_name"
  else
    printf "Error: No such container: %s\n" "$container_name" >&2
    return 1
  fi
}

# Mock docker rm
mock_docker_rm() {
  local container_name="$1"
  local force=false

  if [[ "$1" == "-f" ]] || [[ "$1" == "--force" ]]; then
    force=true
    shift
    container_name="$1"
  fi

  local running_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running"
  local stopped_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_stopped"

  if [[ -f "$running_file" ]]; then
    if [[ "$force" == true ]]; then
      rm -f "$running_file"
      printf "%s\n" "$container_name"
    else
      printf "Error: Cannot remove running container: %s\n" "$container_name" >&2
      return 1
    fi
  elif [[ -f "$stopped_file" ]]; then
    rm -f "$stopped_file"
    printf "%s\n" "$container_name"
  else
    printf "Error: No such container: %s\n" "$container_name" >&2
    return 1
  fi
}

# Mock docker exec
mock_docker_exec() {
  local container_name=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -it|-i|-t)
        shift
        ;;
      *)
        if [[ -z "$container_name" ]]; then
          container_name="$1"
        fi
        shift
        ;;
    esac
  done

  local container_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running"

  if [[ -f "$container_file" ]]; then
    # Simulate successful exec
    printf "Mock exec in %s\n" "$container_name"
    return 0
  else
    printf "Error: No such container: %s\n" "$container_name" >&2
    return 1
  fi
}

# Mock docker logs
mock_docker_logs() {
  local container_name="$1"
  local container_file

  # Find container (running or stopped)
  if [[ -f "$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running" ]]; then
    container_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running"
  elif [[ -f "$MOCK_DOCKER_STATE_DIR/containers/${container_name}_stopped" ]]; then
    container_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_stopped"
  else
    printf "Error: No such container: %s\n" "$container_name" >&2
    return 1
  fi

  # Return mock logs
  printf "[%s] Container started\n" "$(date)"
  printf "[%s] Mock application running\n" "$(date)"
}

# Mock docker inspect
mock_docker_inspect() {
  local container_name="$1"
  local container_file

  # Find container
  if [[ -f "$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running" ]]; then
    container_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_running"
  elif [[ -f "$MOCK_DOCKER_STATE_DIR/containers/${container_name}_stopped" ]]; then
    container_file="$MOCK_DOCKER_STATE_DIR/containers/${container_name}_stopped"
  else
    printf "[]\n"
    return 0
  fi

  # Return mock JSON
  printf '[\n'
  printf '  {\n'
  printf '    "Id": "%s",\n' "$container_name"
  printf '    "Name": "/%s",\n' "$container_name"
  printf '    "State": {\n'
  if [[ "$container_file" == *"_running" ]]; then
    printf '      "Status": "running",\n'
    printf '      "Running": true\n'
  else
    printf '      "Status": "exited",\n'
    printf '      "Running": false\n'
  fi
  printf '    }\n'
  printf '  }\n'
  printf ']\n'
}

# Mock docker-compose wrapper
mock_docker_compose() {
  local subcommand="$1"
  shift

  case "$subcommand" in
    up)
      # Mock up command
      printf "Mock: Starting services...\n"
      return 0
      ;;
    down)
      # Mock down command
      printf "Mock: Stopping services...\n"
      return 0
      ;;
    ps)
      # Mock ps command
      mock_docker_ps "$@"
      ;;
    logs)
      # Mock logs command
      printf "Mock: Showing logs...\n"
      return 0
      ;;
    *)
      printf "Mock docker-compose %s\n" "$subcommand"
      return 0
      ;;
  esac
}

# ============================================================================
# Install Mock as 'docker' Command
# ============================================================================

# Create mock docker function
docker() {
  local subcommand="${1:-}"
  shift || true

  case "$subcommand" in
    ps)
      mock_docker_ps "$@"
      ;;
    run)
      mock_docker_run "$@"
      ;;
    stop)
      mock_docker_stop "$@"
      ;;
    start)
      mock_docker_start "$@"
      ;;
    rm)
      mock_docker_rm "$@"
      ;;
    exec)
      mock_docker_exec "$@"
      ;;
    logs)
      mock_docker_logs "$@"
      ;;
    inspect)
      mock_docker_inspect "$@"
      ;;
    *)
      printf "Mock docker %s\n" "$subcommand"
      return 0
      ;;
  esac
}

# Export mock function
export -f docker
export -f mock_docker_ps
export -f mock_docker_run
export -f mock_docker_stop
export -f mock_docker_start
export -f mock_docker_rm
export -f mock_docker_exec
export -f mock_docker_logs
export -f mock_docker_inspect
export -f mock_docker_compose
export -f init_docker_mock
export -f cleanup_docker_mock

# Auto-initialize on source
init_docker_mock
