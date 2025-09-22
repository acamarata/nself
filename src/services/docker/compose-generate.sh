#!/usr/bin/env bash
# compose-generate.sh - Generate docker-compose.yml configuration
# Refactored modular version using separate service modules
set -euo pipefail

# Error handler with more details
trap 'echo "Error on line $LINENO: $BASH_COMMAND" >&2' ERR

# Enable debugging if DEBUG is set
[[ "${DEBUG:-}" == "true" ]] && set -x

# Get script directory (macOS compatible)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source display utilities first (for logging functions)
if [[ -f "$SCRIPT_DIR/../../lib/utils/display.sh" ]]; then
  source "$SCRIPT_DIR/../../lib/utils/display.sh"
fi

# Source environment utilities for safe loading
if [[ -f "$SCRIPT_DIR/../../lib/utils/env.sh" ]]; then
  source "$SCRIPT_DIR/../../lib/utils/env.sh"
fi

# Load environment safely (prioritize .env.dev over .env)
env_file=""
if [[ -f .env.dev ]]; then
  env_file=".env.dev"
elif [[ -f .env.local ]]; then
  env_file=".env.local"
elif [[ -f .env ]]; then
  env_file=".env"
fi

if [[ -n "$env_file" ]]; then
  set -a
  while IFS='=' read -r key value; do
    # Skip comments and empty lines
    [[ "$key" =~ ^#.*$ || -z "$key" ]] && continue
    # Skip if value contains JSON-like structures or command substitutions
    if [[ ! "$value" =~ [\{\}\$\`] ]]; then
      export "$key=$value"
    fi
  done < "$env_file"
  set +a
else
  echo "Warning: No .env file found" >&2
fi

# Source smart defaults to handle JWT construction
if [[ -f "$SCRIPT_DIR/../../services/auth/smart-defaults.sh" ]]; then
  source "$SCRIPT_DIR/../../services/auth/smart-defaults.sh"
  apply_smart_defaults
fi

# Source auth config for multi-app support
if [[ -f "$SCRIPT_DIR/../../services/auth/multi-app.sh" ]]; then
  source "$SCRIPT_DIR/../../services/auth/multi-app.sh"
fi

# Source all service modules
MODULES_DIR="$SCRIPT_DIR/compose-modules"
if [[ -d "$MODULES_DIR" ]]; then
  for module in "$MODULES_DIR"/*.sh; do
    [[ -f "$module" ]] && source "$module"
  done
else
  echo "Error: compose-modules directory not found" >&2
  exit 1
fi

# Compose database URLs from individual variables
# CRITICAL: Always use port 5432 for internal container-to-container communication
# The POSTGRES_PORT variable is for external host access only
construct_database_urls() {
  local db_user="${POSTGRES_USER:-postgres}"
  local db_pass="${POSTGRES_PASSWORD}"
  local db_host="postgres"  # Container name for internal networking
  local db_port="5432"      # Always use 5432 internally
  local db_name="${POSTGRES_DB:-${PROJECT_NAME}}"

  # URL encode password to handle special characters
  local encoded_pass=$(url_encode "$db_pass")

  # Construct database URLs
  export DATABASE_URL="postgresql://${db_user}:${encoded_pass}@${db_host}:${db_port}/${db_name}"
  export POSTGRES_URL="postgresql://${db_user}:${encoded_pass}@${db_host}:${db_port}/${db_name}"

  # Auth-specific database URL (can point to a different database)
  local auth_db="${AUTH_DATABASE_NAME:-${db_name}}"
  export AUTH_DATABASE_URL="postgresql://${db_user}:${encoded_pass}@${db_host}:${db_port}/${auth_db}"

  # Storage database URL (for Hasura Storage)
  local storage_db="${STORAGE_DATABASE_NAME:-${db_name}}"
  export STORAGE_DATABASE_URL="postgresql://${db_user}:${encoded_pass}@${db_host}:${db_port}/${storage_db}"
}

# URL encode function for password handling
url_encode() {
  local string="${1}"
  local strlen=${#string}
  local encoded=""
  local pos c o

  for (( pos=0 ; pos<strlen ; pos++ )); do
    c=${string:$pos:1}
    case "$c" in
      [-_.~a-zA-Z0-9] ) o="${c}" ;;
      * ) printf -v o '%%%02x' "'$c" ;;
    esac
    encoded+="${o}"
  done
  echo "${encoded}"
}

# Ensure DOCKER_NETWORK is expanded for Docker Compose
DOCKER_NETWORK="${PROJECT_NAME}_network"
export DOCKER_NETWORK

# Set environment-specific defaults
# Support both ENV and ENVIRONMENT for backward compatibility
ENVIRONMENT="${ENV:-${ENVIRONMENT:-development}}"
export NODE_ENV="${NODE_ENV:-$ENVIRONMENT}"

# Set defaults based on environment
case "$ENVIRONMENT" in
  production|prod)
    export LOG_LEVEL="${LOG_LEVEL:-warn}"
    export DEBUG="${DEBUG:-false}"
    ;;
  staging|stage)
    export LOG_LEVEL="${LOG_LEVEL:-info}"
    export DEBUG="${DEBUG:-false}"
    ;;
  *)
    export LOG_LEVEL="${LOG_LEVEL:-debug}"
    export DEBUG="${DEBUG:-false}"
    ;;
esac

# Backup existing docker-compose.yml only if it will be changed
backup_existing_compose() {
  if [[ -f docker-compose.yml ]]; then
    local backup_dir=".volumes/backups/compose"
    mkdir -p "$backup_dir"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    cp docker-compose.yml "$backup_dir/docker-compose.yml.$timestamp"

    # Keep only last 10 backups
    ls -t "$backup_dir"/docker-compose.yml.* 2>/dev/null | tail -n +11 | xargs rm -f 2>/dev/null || true
  fi
}

# Generate the complete docker-compose.yml
generate_docker_compose() {
  # Set default DOCKER_NETWORK if not set
  : ${DOCKER_NETWORK:="${PROJECT_NAME}_network"}

  # Construct database URLs
  construct_database_urls

  # Backup existing file
  backup_existing_compose

  # Start generating the compose file
  cat > docker-compose.yml <<EOF
# Generated by nself build - DO NOT EDIT MANUALLY
# Project: ${PROJECT_NAME}
# Date: $(date '+%Y-%m-%d %H:%M:%S')
# Version: $(cat "$SCRIPT_DIR/../VERSION" 2>/dev/null || echo "unknown")

version: '3.8'

networks:
  ${DOCKER_NETWORK}:
    name: ${DOCKER_NETWORK}
    driver: bridge

volumes:
  postgres_data:
    driver: local
EOF

  # Add conditional volumes based on enabled services
  [[ "${REDIS_ENABLED:-false}" == "true" ]] && echo "  redis_data:" >> docker-compose.yml
  [[ "${MINIO_ENABLED:-false}" == "true" ]] && echo "  minio_data:" >> docker-compose.yml
  [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]] && echo "  meilisearch_data:" >> docker-compose.yml
  [[ "${TYPESENSE_ENABLED:-false}" == "true" ]] && echo "  typesense_data:" >> docker-compose.yml
  [[ "${SONIC_ENABLED:-false}" == "true" ]] && echo "  sonic_data:" >> docker-compose.yml
  [[ "${MLFLOW_ENABLED:-false}" == "true" ]] && echo "  mlflow_data:" >> docker-compose.yml
  [[ "${GRAFANA_ENABLED:-false}" == "true" ]] && echo "  grafana_data:" >> docker-compose.yml
  [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]] && echo "  prometheus_data:" >> docker-compose.yml
  [[ "${LOKI_ENABLED:-false}" == "true" ]] && echo "  loki_data:" >> docker-compose.yml
  [[ "${PGADMIN_ENABLED:-false}" == "true" ]] && echo "  pgadmin_data:" >> docker-compose.yml
  [[ "${PORTAINER_ENABLED:-false}" == "true" ]] && echo "  portainer_data:" >> docker-compose.yml

  # Start services section
  echo "" >> docker-compose.yml
  echo "services:" >> docker-compose.yml

  # Generate core services
  echo "  # ============================================" >> docker-compose.yml
  echo "  # Core Services" >> docker-compose.yml
  echo "  # ============================================" >> docker-compose.yml

  generate_postgres_service >> docker-compose.yml
  generate_hasura_service >> docker-compose.yml
  generate_auth_service >> docker-compose.yml
  generate_minio_service >> docker-compose.yml
  generate_redis_service >> docker-compose.yml

  # Generate utility services
  if [[ "${MAILPIT_ENABLED:-true}" == "true" ]] || \
     [[ "${ADMINER_ENABLED:-false}" == "true" ]] || \
     [[ "${BULLMQ_DASHBOARD_ENABLED:-false}" == "true" ]] || \
     [[ "${PGADMIN_ENABLED:-false}" == "true" ]] || \
     [[ "${SWAGGER_UI_ENABLED:-false}" == "true" ]] || \
     [[ "${PORTAINER_ENABLED:-false}" == "true" ]] || \
     [[ "${BACKUP_ENABLED:-false}" == "true" ]]; then
    echo "" >> docker-compose.yml
    echo "  # ============================================" >> docker-compose.yml
    echo "  # Utility Services" >> docker-compose.yml
    echo "  # ============================================" >> docker-compose.yml
    generate_utility_services >> docker-compose.yml
  fi

  # Generate monitoring services
  if [[ "${MLFLOW_ENABLED:-false}" == "true" ]] || \
     [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]] || \
     [[ "${TYPESENSE_ENABLED:-false}" == "true" ]] || \
     [[ "${SONIC_ENABLED:-false}" == "true" ]] || \
     [[ "${GRAFANA_ENABLED:-false}" == "true" ]] || \
     [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]] || \
     [[ "${LOKI_ENABLED:-false}" == "true" ]] || \
     [[ "${PROMTAIL_ENABLED:-false}" == "true" ]]; then
    echo "" >> docker-compose.yml
    echo "  # ============================================" >> docker-compose.yml
    echo "  # Monitoring & Search Services" >> docker-compose.yml
    echo "  # ============================================" >> docker-compose.yml
    generate_monitoring_services >> docker-compose.yml
  fi

  # Generate frontend apps
  generate_frontend_apps >> docker-compose.yml

  # Generate custom services
  generate_custom_services >> docker-compose.yml

  echo "" >> docker-compose.yml
  echo "# End of generated docker-compose.yml" >> docker-compose.yml
}

# Main execution
main() {
  # Sanitize and set defaults for critical variables
  if [[ -z "$PROJECT_NAME" ]] || [[ "$PROJECT_NAME" =~ [[:space:]] ]]; then
    PROJECT_NAME=$(echo "${PROJECT_NAME:-myproject}" | tr -d ' ' | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9_-]//g')
    [[ -z "$PROJECT_NAME" ]] && PROJECT_NAME="myproject"
  fi

  echo "Generating docker-compose.yml for project: ${PROJECT_NAME}"

  # Generate the compose file
  if ! generate_docker_compose; then
    echo "Error: Failed to generate docker-compose.yml" >&2
    return 1
  fi

  echo "✓ docker-compose.yml generated successfully"

  # Validate the generated file (skip if docker not available)
  if command -v docker >/dev/null 2>&1; then
    if timeout 5 docker compose config >/dev/null 2>&1; then
      echo "✓ docker-compose.yml validation passed"
    else
      echo "⚠ docker-compose.yml validation warnings - please review the configuration" >&2
      # Don't exit on validation warnings
    fi
  else
    echo "⚠ Docker not available - skipping validation"
  fi
}

# Run main function
main "$@"