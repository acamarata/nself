#!/usr/bin/env bash
# auto-fix.sh - Automatic fixes applied during start
# Bash 3.2 compatible, cross-platform

# Source platform compatibility utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../utils/platform-compat.sh" 2>/dev/null || {
  # Fallback definition if not found
  safe_sed_inline() {
    local file="$1"
    shift
    if [[ "$OSTYPE" == "darwin"* ]]; then
      sed -i '' "$@" "$file"
    else
      sed -i "$@" "$file"
    fi
  }
}

# Auto-fix common issues before starting services
apply_start_auto_fixes() {
  local project_name="${1:-nself}"
  local env_file="${2:-.env}"
  local verbose="${3:-false}"

  # Fix 1: MLFlow port 5000 conflict on macOS
  if [[ "$(uname)" == "Darwin" ]]; then
    # Check if port 5000 is in use (likely Control Center)
    if lsof -i :5000 >/dev/null 2>&1; then
      # Check if MLFLOW_PORT is still set to 5000
      if grep -q "MLFLOW_PORT=5000" "$env_file" 2>/dev/null; then
        [ "$verbose" = "true" ] && echo "Auto-fixing: MLFlow port conflict (5000 -> 5005)"

        # Update env file
        safe_sed_inline "$env_file" 's/MLFLOW_PORT=5000/MLFLOW_PORT=5005/g'

        # Also update docker-compose if it exists
        if [[ -f "docker-compose.yml" ]]; then
          safe_sed_inline "docker-compose.yml" 's/\${MLFLOW_PORT:-5000}/\${MLFLOW_PORT:-5005}/g'
          safe_sed_inline "docker-compose.yml" 's/localhost:5000/localhost:5005/g'
        fi
      fi
    fi
  fi

  # Fix 2: Create monitoring config files if missing
  if [[ "${GRAFANA_ENABLED:-false}" == "true" ]] || [[ "${LOKI_ENABLED:-false}" == "true" ]]; then
    if [[ ! -f "monitoring/loki/local-config.yaml" ]]; then
      [ "$verbose" = "true" ] && echo "Auto-fixing: Creating missing monitoring configs"

      # Source monitoring setup if available
      if command -v setup_monitoring_configs >/dev/null 2>&1; then
        setup_monitoring_configs >/dev/null 2>&1
      fi
    fi
  fi

  # Fix 3: Ensure DATABASE_URL is set if missing
  if ! grep -q "^DATABASE_URL=" "$env_file" 2>/dev/null; then
    local db_name="${POSTGRES_DB:-${PROJECT_NAME}_db}"
    # Sanitize database name (replace hyphens with underscores)
    local safe_db_name=$(echo "$db_name" | tr '-' '_')
    local db_user="${POSTGRES_USER:-postgres}"
    local db_pass="${POSTGRES_PASSWORD:-postgres}"
    local db_host="${POSTGRES_HOST:-postgres}"
    local db_port="${POSTGRES_PORT:-5432}"

    [ "$verbose" = "true" ] && echo "Auto-fixing: Adding DATABASE_URL to env file"
    echo "DATABASE_URL=postgres://${db_user}:${db_pass}@${db_host}:${db_port}/${safe_db_name}" >> "$env_file"
  fi

  # Fix 4: Fix JWT configuration for auth service
  if [[ "${AUTH_ENABLED:-false}" == "true" ]]; then
    # Ensure AUTH_JWT_SECRET is properly set
    if ! grep -q "^AUTH_JWT_SECRET=" "$env_file" 2>/dev/null; then
      local jwt_secret="${HASURA_JWT_KEY:-$(openssl rand -hex 32 2>/dev/null || echo 'demo-jwt-secret-key-minimum-32-characters-long')}"
      [ "$verbose" = "true" ] && echo "Auto-fixing: Setting AUTH_JWT_SECRET"
      echo "AUTH_JWT_SECRET=${jwt_secret}" >> "$env_file"
    fi
  fi

  return 0
}

# Export function
export -f apply_start_auto_fixes