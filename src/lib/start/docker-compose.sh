#!/usr/bin/env bash
# docker-compose.sh - Docker Compose operations for nself start
# Bash 3.2 compatible, cross-platform

# Get the appropriate docker compose command
get_compose_command() {
  # Prefer docker compose v2
  if docker compose version >/dev/null 2>&1; then
    echo "docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    echo "docker-compose"
  else
    echo "docker compose"  # Assume v2
  fi
}

# Build docker compose command with all flags
build_compose_command() {
  local env_file="${1:-.env}"
  local project="${2:-nself}"
  local detached="${3:-true}"
  local build="${4:-true}"

  local compose_cmd=$(get_compose_command)
  local cmd="$compose_cmd --project-name \"$project\" --env-file \"$env_file\""

  if [ "$build" = "true" ]; then
    cmd="$cmd up --build"
  else
    cmd="$cmd up"
  fi

  if [ "$detached" = "true" ]; then
    cmd="$cmd -d"
  fi

  echo "$cmd"
}

# Execute docker compose with progress monitoring
execute_compose_with_progress() {
  local compose_cmd="$1"
  local project="${2:-nself}"
  local timeout="${3:-600}"  # 10 minutes default
  local verbose="${4:-false}"

  local output_file=$(mktemp)
  local services_to_start=0
  local compose_bin=$(get_compose_command)

  # Get expected service count
  services_to_start=$($compose_bin --project-name "$project" config --services 2>/dev/null | wc -l | tr -d ' ')

  if [ "$verbose" = "true" ]; then
    # Show full output
    printf "\n"
    eval "$compose_cmd" 2>&1 | tee "$output_file"
    local result=${PIPESTATUS[0]}
  else
    # Run in background with progress monitoring
    (eval "$compose_cmd" 2>&1) >"$output_file" &
    local compose_pid=$!

    # Monitor progress
    monitor_compose_progress "$compose_pid" "$output_file" "$project" "$services_to_start" "$timeout"
    local result=$?
  fi

  rm -f "$output_file"
  return $result
}

# Monitor docker compose progress
monitor_compose_progress() {
  local compose_pid="$1"
  local output_file="$2"
  local project="$3"
  local expected_services="${4:-0}"
  local timeout="${5:-600}"

  local spin_chars="⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
  local i=0
  local elapsed=0
  local last_message=""

  while kill -0 $compose_pid 2>/dev/null; do
    if [ $elapsed -ge $timeout ]; then
      printf "\r${COLOR_RED}✗${COLOR_RESET} Timeout after ${timeout}s                  \n"
      kill $compose_pid 2>/dev/null
      return 1
    fi

    # Get current running count
    local current_count=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')

    # Parse output for status
    local status_msg=""
    if [ -f "$output_file" ] && [ $elapsed -gt 1 ]; then
      # Check for pulling
      if tail -20 "$output_file" 2>/dev/null | grep -q "Pulling"; then
        local image=$(tail -20 "$output_file" | grep "Pulling" | tail -1 | awk '{print $1}')
        status_msg=" - Pulling $image"
      # Check for building
      elif tail -20 "$output_file" 2>/dev/null | grep -q "Building"; then
        local service=$(tail -20 "$output_file" | grep "Building" | tail -1 | awk '{print $2}')
        status_msg=" - Building $service"
      # Check for creating
      elif tail -20 "$output_file" 2>/dev/null | grep -q "Creating"; then
        local container=$(tail -20 "$output_file" | grep "Creating" | tail -1 | awk '{print $2}')
        status_msg=" - Creating $container"
      fi
    fi

    # Display progress
    local char="${spin_chars:$((i % ${#spin_chars})):1}"

    if [ -n "$status_msg" ]; then
      printf "\r${COLOR_BLUE}%s${COLOR_RESET} Starting services...%s          " "$char" "$status_msg"
    elif [ $expected_services -gt 0 ] && [ $current_count -gt 0 ]; then
      printf "\r${COLOR_BLUE}%s${COLOR_RESET} Starting services... (%d/%d)          " "$char" "$current_count" "$expected_services"
    else
      printf "\r${COLOR_BLUE}%s${COLOR_RESET} Initializing services...          " "$char"
    fi

    i=$((i + 1))
    elapsed=$((elapsed + 1))
    sleep 1
  done

  wait $compose_pid
  local compose_exit_code=$?

  # Check if essential services are running regardless of compose exit code
  local running_count=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')

  # If we have running services, consider it successful even if compose had warnings/errors
  if [ "$running_count" -gt 0 ]; then
    return 0
  else
    return $compose_exit_code
  fi
}

# Check if all services are healthy
check_services_health() {
  local project="${1:-nself}"
  local max_wait="${2:-60}"
  local elapsed=0
  local spin_chars="⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
  local i=0

  while [ $elapsed -lt $max_wait ]; do
    # Get service status counts
    local total=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Names}}" | wc -l | tr -d ' ')
    local healthy=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Status}}" | grep -c "(healthy)" 2>/dev/null || echo "0")
    local unhealthy=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Status}}" | grep -c "(unhealthy)" 2>/dev/null || echo "0")
    local starting=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Status}}" | grep -c "starting" 2>/dev/null || echo "0")

    # Clean up counts
    healthy=$(echo "$healthy" | tr -d ' \n\r')
    unhealthy=$(echo "$unhealthy" | tr -d ' \n\r')
    starting=$(echo "$starting" | tr -d ' \n\r')

    # Show progress
    local char="${spin_chars:$((i % ${#spin_chars})):1}"
    printf "\r${COLOR_BLUE}%s${COLOR_RESET} Waiting for services... (healthy: %d/%d)          " "$char" "$healthy" "$total"

    # If all are healthy or none are unhealthy and none are starting, we're good
    if [ "$unhealthy" -eq 0 ] && [ "$starting" -eq 0 ]; then
      return 0
    fi

    sleep 2
    elapsed=$((elapsed + 2))
    i=$((i + 1))
  done

  return 1
}

# Stop services
stop_services() {
  local project="${1:-nself}"
  local compose_cmd=$(get_compose_command)

  $compose_cmd --project-name "$project" down
}

# Restart specific service
restart_service() {
  local service="$1"
  local project="${2:-nself}"
  local compose_cmd=$(get_compose_command)

  $compose_cmd --project-name "$project" restart "$service"
}