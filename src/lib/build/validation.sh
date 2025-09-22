#!/usr/bin/env bash
# validation.sh - Environment validation for build

# Source auto-fix utilities
VALIDATION_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$VALIDATION_DIR/../auto-fix/env-quotes-fix.sh" ]]; then
  source "$VALIDATION_DIR/../auto-fix/env-quotes-fix.sh"
fi

# Source display utilities for show_warning, show_error, show_info
if [[ -f "$VALIDATION_DIR/../utils/display.sh" ]]; then
  source "$VALIDATION_DIR/../utils/display.sh"
fi

# Validate environment configuration
validate_environment() {
  local validation_passed=true
  local errors=()
  local warnings=()
  local fixes=()

  # Fix unquoted values with spaces first (before loading)
  if command -v auto_fix_env_quotes >/dev/null 2>&1; then
    auto_fix_env_quotes
  fi

  # Check required variables
  if [[ -z "${PROJECT_NAME:-}" ]]; then
    PROJECT_NAME="$(basename "$PWD")"
    fixes+=("Set PROJECT_NAME to '$PROJECT_NAME'")
    export PROJECT_NAME
  fi

  if [[ -z "${BASE_DOMAIN:-}" ]]; then
    BASE_DOMAIN="localhost"
    fixes+=("Set BASE_DOMAIN to 'localhost'")
    export BASE_DOMAIN
  fi

  # Validate PROJECT_NAME format
  if [[ ! "$PROJECT_NAME" =~ ^[a-z0-9][a-z0-9-]*[a-z0-9]$ ]]; then
    local fixed_name=$(echo "$PROJECT_NAME" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g' | sed 's/^-//;s/-$//' | sed 's/--*/-/g')
    if [[ "$fixed_name" != "$PROJECT_NAME" ]]; then
      PROJECT_NAME="$fixed_name"
      fixes+=("Fixed PROJECT_NAME to '$PROJECT_NAME'")
      export PROJECT_NAME
    fi
  fi

  # Validate domain format
  if [[ "$BASE_DOMAIN" == *" "* ]]; then
    BASE_DOMAIN="${BASE_DOMAIN// /}"
    fixes+=("Removed spaces from BASE_DOMAIN")
    export BASE_DOMAIN
  fi

  # Check for port conflicts
  check_port_conflicts

  # Validate boolean values
  validate_boolean_vars

  # Apply fixes to .env if needed
  if [[ ${#fixes[@]} -gt 0 ]]; then
    apply_validation_fixes
  fi

  # Show validation results
  if [[ ${#errors[@]} -gt 0 ]]; then
    for error in "${errors[@]}"; do
      show_error "$error"
    done
    validation_passed=false
  fi

  if [[ ${#warnings[@]} -gt 0 ]]; then
    for warning in "${warnings[@]}"; do
      show_warning "$warning"
    done
  fi

  if [[ ${#fixes[@]} -gt 0 ]]; then
    show_info "Applied ${#fixes[@]} automatic fixes"
  fi

  [[ "$validation_passed" == true ]]
}

# Check for port conflicts
check_port_conflicts() {
  local ports_to_check=(
    "NGINX_PORT:80"
    "NGINX_SSL_PORT:443"
    "POSTGRES_PORT:5432"
    "HASURA_PORT:8080"
    "AUTH_PORT:4000"
    "STORAGE_PORT:5000"
    "REDIS_PORT:6379"
  )

  for port_var in "${ports_to_check[@]}"; do
    local var_name="${port_var%:*}"
    local default_port="${port_var#*:}"
    # Use eval for Bash 3.2 compatibility
    eval "local port=\${$var_name:-$default_port}"

    # Check if port is in use
    if command -v lsof >/dev/null 2>&1; then
      if lsof -Pi :"$port" -t >/dev/null 2>&1; then
        warnings+=("Port $port (${var_name}) is already in use")
      fi
    fi
  done
}

# Validate boolean variables
validate_boolean_vars() {
  local bool_vars=(
    "SSL_ENABLED"
    "NGINX_ENABLED"
    "POSTGRES_ENABLED"
    "HASURA_ENABLED"
    "AUTH_ENABLED"
    "STORAGE_ENABLED"
    "REDIS_ENABLED"
    "FUNCTIONS_ENABLED"
    "NESTJS_ENABLED"
    "NSELF_ADMIN_ENABLED"
  )

  for var in "${bool_vars[@]}"; do
    # Use eval for Bash 3.2 compatibility
    eval "local value=\${$var:-}"
    if [[ -n "$value" ]] && [[ "$value" != "true" ]] && [[ "$value" != "false" ]]; then
      # Convert common boolean representations
      local value_lower=$(echo "$value" | tr '[:upper:]' '[:lower:]')
      case "$value_lower" in
        1|yes|y|on|enabled)
          eval "$var=true"
          fixes+=("Fixed $var to 'true'")
          ;;
        0|no|n|off|disabled)
          eval "$var=false"
          fixes+=("Fixed $var to 'false'")
          ;;
        *)
          eval "$var=false"
          warnings+=("Invalid boolean value for $var: '$value' (set to 'false')")
          ;;
      esac
    fi
  done
}

# Apply validation fixes to .env files
apply_validation_fixes() {
  local env_file=".env"

  # Determine which env file to update
  if [[ -f ".env.local" ]]; then
    env_file=".env.local"
  elif [[ -f ".env.${ENV:-dev}" ]]; then
    env_file=".env.${ENV:-dev}"
  fi

  # Backup the file to _backup/timestamp structure
  local timestamp="$(date +%Y%m%d_%H%M%S)"
  local backup_dir="_backup/${timestamp}"
  mkdir -p "$backup_dir"
  cp "$env_file" "${backup_dir}/$(basename "$env_file")" 2>/dev/null || true

  # Apply fixes (carefully)
  local temp_file=$(mktemp)

  # Read the original file and apply fixes
  while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip empty lines and comments
    if [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*# ]]; then
      echo "$line" >> "$temp_file"
      continue
    fi

    # Parse key=value pairs
    if [[ "$line" =~ ^([A-Z_]+)=(.*)$ ]]; then
      local key="${BASH_REMATCH[1]}"
      local value="${BASH_REMATCH[2]}"

      # Check if we have a new value for this key
      # Use eval for Bash 3.2 compatibility
      eval "local new_value=\${$key:-}"
      if [[ -n "$new_value" ]]; then
        echo "${key}=${new_value}" >> "$temp_file"
      else
        echo "$line" >> "$temp_file"
      fi
    else
      echo "$line" >> "$temp_file"
    fi
  done < "$env_file"

  # Add any new variables that weren't in the file
  for fix in "${fixes[@]}"; do
    if [[ "$fix" =~ Set[[:space:]]([A-Z_]+)[[:space:]]to ]]; then
      local key="${BASH_REMATCH[1]}"
      if ! grep -q "^${key}=" "$temp_file"; then
        eval "local new_val=\${$key:-}"
        echo "${key}=${new_val}" >> "$temp_file"
      fi
    fi
  done

  # Move temp file to original
  mv "$temp_file" "$env_file"
}

# Validate service dependencies
validate_service_dependencies() {
  # Use eval for Bash 3.2 compatibility with indirect variable references
  # Check Hasura dependencies
  if [[ "${HASURA_ENABLED:-false}" == "true" ]]; then
    if [[ "${POSTGRES_ENABLED:-false}" != "true" ]]; then
      POSTGRES_ENABLED=true
      fixes+=("Enabled PostgreSQL (required by Hasura)")
    fi
  fi

  # Check Auth dependencies
  if [[ "${AUTH_ENABLED:-false}" == "true" ]]; then
    if [[ "${POSTGRES_ENABLED:-false}" != "true" ]]; then
      POSTGRES_ENABLED=true
      fixes+=("Enabled PostgreSQL (required by Auth)")
    fi
  fi

  # Check Storage dependencies
  if [[ "${STORAGE_ENABLED:-false}" == "true" ]]; then
    if [[ "${POSTGRES_ENABLED:-false}" != "true" ]]; then
      POSTGRES_ENABLED=true
      fixes+=("Enabled PostgreSQL (required by Storage)")
    fi
  fi
}

# Export functions
export -f validate_environment
export -f check_port_conflicts
export -f validate_boolean_vars
export -f apply_validation_fixes
export -f validate_service_dependencies