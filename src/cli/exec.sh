#!/usr/bin/env bash

# exec.sh - Execute commands in containers

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/docker.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/header.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Show help
show_exec_help() {
  echo "nself exec - Execute commands in containers"
  echo ""
  echo "Usage: nself exec [options] <service> [command]"
  echo ""
  echo "Options:"
  echo "  -it            Interactive terminal (default for shell)"
  echo "  -T             Disable pseudo-TTY allocation"
  echo "  -u <user>      Run as specific user"
  echo "  -w <dir>       Working directory inside container"
  echo "  -e <VAR=val>   Set environment variable"
  echo "  --no-tty       Disable TTY (same as -T)"
  echo "  -h, --help     Show this help message"
  echo ""
  echo "Services:"
  echo "  postgres       PostgreSQL database"
  echo "  hasura         Hasura GraphQL engine"
  echo "  auth           nHost authentication service"
  echo "  nginx          Web server"
  echo "  redis          Redis cache (if enabled)"
  echo "  minio          S3-compatible storage (if enabled)"
  echo "  functions      Serverless functions (if enabled)"
  echo "  <service>      Any other service in docker-compose.yml"
  echo ""
  echo "Examples:"
  echo "  nself exec postgres psql -U postgres"
  echo "  nself exec hasura hasura-cli console"
  echo "  nself exec redis redis-cli"
  echo "  nself exec -it nginx /bin/bash"
  echo "  nself exec -u root nginx ls -la /etc/nginx"
  echo ""
  echo "Common commands:"
  echo "  postgres:"
  echo "    psql -U postgres                    # PostgreSQL shell"
  echo "    pg_dump -U postgres dbname          # Dump database"
  echo "    pg_restore -U postgres -d dbname    # Restore database"
  echo ""
  echo "  redis:"
  echo "    redis-cli                            # Redis shell"
  echo "    redis-cli FLUSHALL                   # Clear all data"
  echo "    redis-cli INFO                       # Server info"
  echo ""
  echo "  hasura:"
  echo "    hasura-cli console                   # Open console"
  echo "    hasura-cli migrate status            # Migration status"
  echo "    hasura-cli metadata export           # Export metadata"
}

# Execute command in container
cmd_exec() {
  local service=""
  local docker_opts=""
  local interactive=false
  local no_tty=false
  local user=""
  local workdir=""
  local env_vars=()
  local command=()
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help)
        show_exec_help
        return 0
        ;;
      -it)
        interactive=true
        shift
        ;;
      -i)
        interactive=true
        shift
        ;;
      -t)
        # Just -t without -i
        shift
        ;;
      -T|--no-tty)
        no_tty=true
        shift
        ;;
      -u)
        user="$2"
        shift 2
        ;;
      -w)
        workdir="$2"
        shift 2
        ;;
      -e)
        env_vars+=("-e" "$2")
        shift 2
        ;;
      *)
        if [[ -z "$service" ]]; then
          service="$1"
        else
          command+=("$1")
        fi
        shift
        ;;
    esac
  done
  
  # Check if service specified
  if [[ -z "$service" ]]; then
    log_error "No service specified"
    echo ""
    echo "Usage: nself exec <service> [command]"
    echo "Run 'nself exec --help' for more information"
    return 1
  fi
  
  # Check if docker-compose.yml exists
  if [[ ! -f "docker-compose.yml" ]]; then
    log_error "No docker-compose.yml found. Run 'nself build' first."
    return 1
  fi
  
  # Load environment
  load_env_with_priority
  
  # Check if service exists
  if ! grep -q "^  ${service}:" docker-compose.yml 2>/dev/null; then
    log_error "Service '$service' not found in docker-compose.yml"
    echo ""
    echo "Available services:"
    grep "^  [a-z]" docker-compose.yml | sed 's/://' | sed 's/^/  /' | sort
    return 1
  fi
  
  # Check if service is running
  if ! compose ps --status running --services 2>/dev/null | grep -q "^${service}$"; then
    log_warning "Service '$service' is not running"
    echo ""
    echo "Start the service with: nself start"
    return 1
  fi
  
  # Build docker exec options as array for safety
  local docker_opts_array=()

  if [[ "$interactive" == "true" ]] && [[ "$no_tty" != "true" ]]; then
    docker_opts_array+=("-it")
  elif [[ "$interactive" == "true" ]]; then
    docker_opts_array+=("-i")
  fi
  # Don't auto-add -t for non-interactive commands - let docker handle it
  
  # Add user option
  if [[ -n "$user" ]]; then
    docker_opts_array+=("-u" "$user")
  fi
  
  # Add workdir option
  if [[ -n "$workdir" ]]; then
    docker_opts_array+=("-w" "$workdir")
  fi
  
  # Default commands for services
  if [[ ${#command[@]} -eq 0 ]]; then
    case "$service" in
      postgres)
        command=("psql" "-U" "postgres")
        if [[ "$interactive" != "true" ]]; then
          docker_opts_array+=("-it")
        fi
        ;;
      redis)
        command=("redis-cli")
        if [[ "$interactive" != "true" ]]; then
          docker_opts_array+=("-it")
        fi
        ;;
      nginx)
        command=("/bin/bash")
        if [[ "$interactive" != "true" ]]; then
          docker_opts_array+=("-it")
        fi
        ;;
      hasura|auth|minio)
        command=("/bin/sh")
        if [[ "$interactive" != "true" ]]; then
          docker_opts_array+=("-it")
        fi
        ;;
      *)
        # Try bash first, then sh
        command=("/bin/bash")
        if [[ "$interactive" != "true" ]]; then
          docker_opts_array+=("-it")
        fi
        ;;
    esac
  fi
  
  # Get container name - try different methods
  local container_name
  
  # Try compose ps with simpler format first
  container_name=$(compose ps --format "table {{.Name}}" "$service" 2>/dev/null | tail -n +2 | head -1)
  
  # If that fails, try with JSON parsing
  if [[ -z "$container_name" ]]; then
    if command -v jq >/dev/null 2>&1; then
      # Use jq if available
      container_name=$(compose ps --format json "$service" 2>/dev/null | jq -r '.Name' 2>/dev/null | head -1)
    else
      # Fallback to grep
      container_name=$(compose ps --format json "$service" 2>/dev/null | grep -o '"Name":"[^"]*"' | cut -d'"' -f4 | head -1)
    fi
  fi
  
  # Final fallback - use container prefix
  if [[ -z "$container_name" ]]; then
    local project_name="${PROJECT_NAME:-$(basename "$(pwd)")}"
    container_name="${project_name}-${service}-1"
  fi
  
  # Check if container actually exists
  if ! docker ps --format "{{.Names}}" | grep -q "^${container_name}$"; then
    log_error "Container '$container_name' for service '$service' is not running"
    return 1
  fi
  
  # Show what we're doing
  show_command_header "nself exec" "Execute in container"
  echo ""
  printf "${COLOR_CYAN}➞ Container:${COLOR_RESET} %s\n" "$container_name"
  printf "${COLOR_CYAN}➞ Command:${COLOR_RESET} %s\n" "${command[*]}"
  echo ""
  
  # Execute the command safely with array expansion
  docker exec "${docker_opts_array[@]}" "${env_vars[@]}" "$container_name" "${command[@]}"
}

# Export for use as library
export -f cmd_exec

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "exec" || exit $?
  cmd_exec "$@"
  exit_code=$?
  post_command "exec" $exit_code
  exit $exit_code
fi