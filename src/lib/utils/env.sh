#!/usr/bin/env bash

# Environment loading and management utilities

# Export colors for consistency
export COLOR_RESET='\033[0m'
export COLOR_BOLD='\033[1m'
export COLOR_RED='\033[0;31m'
export COLOR_GREEN='\033[0;32m'
export COLOR_YELLOW='\033[0;33m'
export COLOR_BLUE='\033[0;34m'
export COLOR_MAGENTA='\033[0;35m'
export COLOR_CYAN='\033[0;36m'
export COLOR_DIM='\033[2m'

# Load environment files with correct priority order
# Priority (lowest to highest - later files override earlier):
#   1. .env.dev     (team defaults, SHARED) - LOWEST
#   2. .env.staging/.env.prod (environment specific, SHARED)  
#   3. .env.local   (personal overrides, not shared)
#   4. .env         (LOCAL ONLY priority overrides)
#   5. .env.secrets (production ONLY secrets/keys) - HIGHEST
load_env_with_priority() {
  local silent="${1:-false}"
  local loaded=false
  
  # Determine current environment (default to dev)
  local current_env="${ENV:-dev}"
  
  # Load files in order from LOWEST to HIGHEST priority
  
  # 1. Team defaults (LOWEST priority)
  if [[ -f ".env.dev" ]]; then
    # log_debug "Loading .env.dev (team defaults)"
    set -a
    source ".env.dev" 2>/dev/null
    set +a
    loaded=true
  fi
  
  # 2. Environment-specific file (staging/prod)
  local env_file=""
  case "$current_env" in
    staging|stage)
      env_file=".env.staging"
      ;;
    prod|production)
      env_file=".env.prod"
      ;;
    dev|development)
      # Already loaded .env.dev above
      ;;
    *)
      # Support custom environments
      env_file=".env.${current_env}"
      ;;
  esac
  
  if [[ -n "$env_file" ]] && [[ -f "$env_file" ]]; then
    # log_debug "Loading $env_file (environment-specific)"
    set -a
    source "$env_file" 2>/dev/null
    set +a
    loaded=true
  fi
  
  # 3. Personal overrides
  if [[ -f ".env.local" ]]; then
    # log_debug "Loading .env.local (personal overrides)"
    set -a
    source ".env.local" 2>/dev/null
    set +a
    loaded=true
  fi
  
  # 4. Local priority overrides (.env file)
  if [[ -f ".env" ]]; then
    # log_debug "Loading .env (local priority overrides)"
    set -a
    source ".env" 2>/dev/null
    set +a
    loaded=true
  fi
  
  # 5. Secrets (HIGHEST priority - overwrites everything)
  if [[ -f ".env.secrets" ]]; then
    # log_debug "Loading .env.secrets (sensitive data - highest priority)"
    set -a
    source ".env.secrets" 2>/dev/null
    set +a
    loaded=true
  fi
  
  # if [[ "$loaded" == false ]]; then
  #   log_debug "No environment files found, using defaults only"
  # fi
  
  return 0
}

# Get environment variable with default
get_env_var() {
  local var_name="$1"
  local default_value="${2:-}"

  local value="${!var_name:-$default_value}"
  echo "$value"
}

# Set environment variable in file
set_env_var() {
  local var_name="$1"
  local value="$2"
  local file="${3:-.env.local}"

  # Check if variable already exists
  if grep -q "^${var_name}=" "$file" 2>/dev/null; then
    # Update existing variable
    if [[ "$OSTYPE" == "darwin"* ]]; then
      sed -i '' "s|^${var_name}=.*|${var_name}=${value}|" "$file"
    else
      sed -i "s|^${var_name}=.*|${var_name}=${value}|" "$file"
    fi
  else
    # Add new variable (ensure newline before if needed)
    # Check if file ends with newline
    if [[ -f "$file" ]] && [[ -s "$file" ]] && [[ $(tail -c 1 "$file" | wc -l) -eq 0 ]]; then
      echo "" >> "$file"
    fi
    echo "${var_name}=${value}" >> "$file"
  fi
}

# Check if environment variable is set
is_env_set() {
  local var_name="$1"
  [[ -n "${!var_name}" ]]
}

# Get environment type (dev/staging/prod)
get_environment() {
  local env="${ENV:-dev}"
  
  case "$env" in
    dev|development)
      echo "dev"
      ;;
    staging|stage)
      echo "staging"
      ;;
    prod|production)
      echo "prod"
      ;;
    *)
      echo "$env"
      ;;
  esac
}

# Check if running in production
is_production() {
  local env=$(get_environment)
  [[ "$env" == "prod" ]] || [[ "$env" == "production" ]]
}

# Check if running in development
is_development() {
  local env=$(get_environment)
  [[ "$env" == "dev" ]] || [[ "$env" == "development" ]]
}

# Export environment from file
export_env_from_file() {
  local file="$1"
  
  if [[ ! -f "$file" ]]; then
    return 1
  fi
  
  set -a
  source "$file"
  set +a
  
  return 0
}

# Validate required environment variables
validate_required_env() {
  local required_vars=("$@")
  local missing=()
  
  for var in "${required_vars[@]}"; do
    if ! is_env_set "$var"; then
      missing+=("$var")
    fi
  done
  
  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Missing required environment variables:"
    printf '  - %s\n' "${missing[@]}"
    return 1
  fi
  
  return 0
}

# Get all env vars with prefix
get_env_vars_with_prefix() {
  local prefix="$1"
  env | grep "^${prefix}" | cut -d= -f1
}

# Clear env vars with prefix
clear_env_vars_with_prefix() {
  local prefix="$1"
  
  for var in $(get_env_vars_with_prefix "$prefix"); do
    unset "$var"
  done
}

# Load environment for specific service
load_service_env() {
  local service="$1"
  local env_file=".env.${service}"
  
  if [[ -f "$env_file" ]]; then
    export_env_from_file "$env_file"
    return 0
  fi
  
  return 1
}

# Export functions
export -f load_env_with_priority
export -f get_env_var
export -f set_env_var
export -f is_env_set
export -f get_environment
export -f is_production
export -f is_development
export -f export_env_from_file
export -f validate_required_env
export -f get_env_vars_with_prefix
export -f clear_env_vars_with_prefix
export -f load_service_env