#!/usr/bin/env bash
# Mock infrastructure for reliable, fast testing
# This provides mocks for external dependencies to enable testing without Docker, network, etc.

set -euo pipefail

# ============================================================================
# Mock Docker API
# ============================================================================

mock_docker() {
  local operation="${1:-}"
  shift || true

  case "$operation" in
    ps)
      # Mock container list
      printf "CONTAINER ID   IMAGE                    COMMAND                  CREATED          STATUS                    PORTS                    NAMES\n"
      printf "abc123def456   postgres:15              \"docker-entrypoint.s…\"   10 minutes ago   Up 10 minutes (healthy)   5432/tcp                 nself_postgres\n"
      printf "def456ghi789   hasura/graphql-engine    \"graphql-engine serv…\"   10 minutes ago   Up 10 minutes (healthy)   8080/tcp                 nself_hasura\n"
      printf "ghi789jkl012   nhost/auth               \"/bin/sh -c 'npm sta…\"   10 minutes ago   Up 10 minutes (healthy)   4000/tcp                 nself_auth\n"
      ;;

    inspect)
      # Mock container inspection
      local container_id="${1:-abc123def456}"
      printf '{\n'
      printf '  "Id": "%s",\n' "$container_id"
      printf '  "State": {\n'
      printf '    "Running": true,\n'
      printf '    "Status": "running",\n'
      printf '    "Health": {\n'
      printf '      "Status": "healthy"\n'
      printf '    }\n'
      printf '  },\n'
      printf '  "Name": "/nself_test_container"\n'
      printf '}\n'
      ;;

    logs)
      # Mock container logs
      local container_id="${1:-abc123def456}"
      printf "Mock log entry 1 for %s\n" "$container_id"
      printf "Mock log entry 2 for %s\n" "$container_id"
      printf "Mock log entry 3 for %s\n" "$container_id"
      ;;

    exec)
      # Mock container exec
      shift || true  # Skip container ID
      local command="$*"
      printf "Mock exec output for: %s\n" "$command"
      ;;

    run)
      # Mock container run
      printf "abc123def456\n"
      ;;

    rm|stop|kill)
      # Mock container operations (no output)
      return 0
      ;;

    version)
      # Mock Docker version
      printf "Docker version 24.0.0, build abc123\n"
      ;;

    info)
      # Mock Docker info
      printf "Containers: 3\n"
      printf "Running: 3\n"
      printf "Paused: 0\n"
      printf "Stopped: 0\n"
      ;;

    *)
      # Unknown operation - just succeed
      return 0
      ;;
  esac
}

# ============================================================================
# Mock Network Calls
# ============================================================================

mock_curl() {
  local url="${1:-}"
  local method="${MOCK_HTTP_METHOD:-GET}"
  local response="${MOCK_HTTP_RESPONSE:-{\"status\":\"ok\"}}"
  local status_code="${MOCK_HTTP_STATUS:-200}"

  # Simulate network delay if requested
  if [[ -n "${MOCK_NETWORK_DELAY:-}" ]]; then
    sleep "${MOCK_NETWORK_DELAY}"
  fi

  # Simulate network failure if requested
  if [[ "${MOCK_NETWORK_FAIL:-false}" == "true" ]]; then
    printf "curl: (7) Failed to connect to %s\n" "$url" >&2
    return 7
  fi

  # Output response
  printf "%s\n" "$response"

  # Return based on status code
  if [[ "$status_code" -ge 200 ]] && [[ "$status_code" -lt 300 ]]; then
    return 0
  else
    return 1
  fi
}

# ============================================================================
# Controllable Time for Timeout Tests
# ============================================================================

# Global mock time (can be overridden)
MOCK_TIME="${MOCK_TIME:-}"

mock_date() {
  local format="${1:-}"

  # If mock time is set, use it
  if [[ -n "$MOCK_TIME" ]]; then
    case "$format" in
      +%s|%s)
        printf "%s\n" "$MOCK_TIME"
        ;;
      +%Y-%m-%d)
        # Convert timestamp to date (basic implementation)
        if command -v python3 >/dev/null 2>&1; then
          python3 -c "import datetime; print(datetime.datetime.fromtimestamp($MOCK_TIME).strftime('%Y-%m-%d'))"
        else
          # Fallback to real date
          command date "$format"
        fi
        ;;
      *)
        # For other formats, use real date
        command date "$format"
        ;;
    esac
  else
    # Use real date
    command date "$format"
  fi
}

# Advance mock time by N seconds
advance_mock_time() {
  local seconds="${1:-1}"

  if [[ -z "$MOCK_TIME" ]]; then
    MOCK_TIME=$(date +%s)
  fi

  MOCK_TIME=$((MOCK_TIME + seconds))
  export MOCK_TIME
}

# ============================================================================
# Deterministic Random Data
# ============================================================================

MOCK_RANDOM_SEED="${MOCK_RANDOM_SEED:-12345}"
MOCK_RANDOM_COUNTER=0

mock_random() {
  local max="${1:-32767}"

  # Simple deterministic pseudo-random based on seed + counter
  local value=$(( (MOCK_RANDOM_SEED + MOCK_RANDOM_COUNTER) % max ))
  MOCK_RANDOM_COUNTER=$((MOCK_RANDOM_COUNTER + 1))

  printf "%d\n" "$value"
}

# Reset random generator
reset_mock_random() {
  MOCK_RANDOM_COUNTER=0
}

# ============================================================================
# Fast tmpfs-backed File Operations
# ============================================================================

create_test_tmpfs() {
  local prefix="${1:-nself-test}"
  local test_dir

  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS - use regular temp dir (no tmpfs)
    test_dir=$(mktemp -d -t "${prefix}")
  else
    # Linux - use tmpfs (/dev/shm) for faster I/O
    if [[ -d "/dev/shm" ]]; then
      test_dir="/dev/shm/${prefix}-$$-${RANDOM}"
      mkdir -p "$test_dir"
    else
      # Fallback to regular temp
      test_dir=$(mktemp -d)
    fi
  fi

  printf "%s\n" "$test_dir"
}

# ============================================================================
# Mock PostgreSQL (psql)
# ============================================================================

mock_psql() {
  local query="${MOCK_PSQL_QUERY:-}"
  local response="${MOCK_PSQL_RESPONSE:-}"

  # If response is set, use it
  if [[ -n "$response" ]]; then
    printf "%s\n" "$response"
    return 0
  fi

  # Otherwise, generate mock response based on query
  if [[ "$*" == *"SELECT"* ]] || [[ "$query" == *"SELECT"* ]]; then
    printf " id | name  | email\n"
    printf "----+-------+------------------\n"
    printf "  1 | user1 | user1@example.com\n"
    printf "  2 | user2 | user2@example.com\n"
    printf "(2 rows)\n"
  elif [[ "$*" == *"INSERT"* ]] || [[ "$query" == *"INSERT"* ]]; then
    printf "INSERT 0 1\n"
  elif [[ "$*" == *"UPDATE"* ]] || [[ "$query" == *"UPDATE"* ]]; then
    printf "UPDATE 1\n"
  elif [[ "$*" == *"DELETE"* ]] || [[ "$query" == *"DELETE"* ]]; then
    printf "DELETE 1\n"
  else
    # Generic success
    printf "OK\n"
  fi
}

# ============================================================================
# Mock Redis CLI
# ============================================================================

mock_redis_cli() {
  local operation="${1:-}"
  shift || true

  case "$operation" in
    GET)
      local key="$1"
      printf "\"mock-value-for-%s\"\n" "$key"
      ;;
    SET)
      printf "OK\n"
      ;;
    DEL)
      printf "(integer) 1\n"
      ;;
    PING)
      printf "PONG\n"
      ;;
    INFO)
      printf "# Server\n"
      printf "redis_version:7.0.0\n"
      printf "redis_mode:standalone\n"
      ;;
    *)
      printf "OK\n"
      ;;
  esac
}

# ============================================================================
# Mock Git Operations
# ============================================================================

mock_git() {
  local operation="${1:-}"
  shift || true

  case "$operation" in
    status)
      printf "On branch main\n"
      printf "Your branch is up to date with 'origin/main'.\n"
      printf "\n"
      printf "nothing to commit, working tree clean\n"
      ;;
    log)
      printf "commit abc123def456 (HEAD -> main, origin/main)\n"
      printf "Author: Test User <test@example.com>\n"
      printf "Date:   Mon Jan 1 12:00:00 2024 +0000\n"
      printf "\n"
      printf "    Mock commit message\n"
      ;;
    rev-parse)
      printf "abc123def456\n"
      ;;
    branch)
      printf "* main\n"
      ;;
    *)
      return 0
      ;;
  esac
}

# ============================================================================
# Environment Control
# ============================================================================

# Enable all mocks
enable_all_mocks() {
  export -f mock_docker
  export -f mock_curl
  export -f mock_date
  export -f mock_random
  export -f mock_psql
  export -f mock_redis_cli
  export -f mock_git

  # Create aliases (only works in interactive shells)
  # For non-interactive, tests should call mock functions directly
  alias docker=mock_docker 2>/dev/null || true
  alias curl=mock_curl 2>/dev/null || true
  alias date=mock_date 2>/dev/null || true
  alias psql=mock_psql 2>/dev/null || true
  alias redis-cli=mock_redis_cli 2>/dev/null || true
  alias git=mock_git 2>/dev/null || true
}

# Disable all mocks
disable_all_mocks() {
  unalias docker 2>/dev/null || true
  unalias curl 2>/dev/null || true
  unalias date 2>/dev/null || true
  unalias psql 2>/dev/null || true
  unalias redis-cli 2>/dev/null || true
  unalias git 2>/dev/null || true
}

# Check if real Docker is available
has_real_docker() {
  command -v docker >/dev/null 2>&1 && docker ps >/dev/null 2>&1
}

# Check if real PostgreSQL is available
has_real_postgres() {
  command -v psql >/dev/null 2>&1
}

# Check if real Redis is available
has_real_redis() {
  command -v redis-cli >/dev/null 2>&1 && redis-cli ping >/dev/null 2>&1
}

# ============================================================================
# Export Functions
# ============================================================================

export -f mock_docker
export -f mock_curl
export -f mock_date
export -f advance_mock_time
export -f mock_random
export -f reset_mock_random
export -f create_test_tmpfs
export -f mock_psql
export -f mock_redis_cli
export -f mock_git
export -f enable_all_mocks
export -f disable_all_mocks
export -f has_real_docker
export -f has_real_postgres
export -f has_real_redis

# ============================================================================
# Usage Example
# ============================================================================

# Source this file in your tests:
#   source "$(dirname "${BASH_SOURCE[0]}")/../mocks/mock-infrastructure.sh"
#
# Then use mocks:
#   if ! has_real_docker; then
#     docker() { mock_docker "$@"; }
#   fi
#
#   result=$(docker ps)
#   # Works with real or mocked Docker!
