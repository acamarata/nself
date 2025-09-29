#!/usr/bin/env bash
# build-orchestrator.sh - Main build orchestration with proper env handling
# Loads env to detect WHAT to build, but outputs use runtime vars for HOW

# Main orchestration function
orchestrate_build() {
  local project_name="${1:-$(basename "$PWD")}"
  local env="${2:-dev}"
  local force="${3:-false}"
  local verbose="${4:-false}"

  # Export basic settings
  export PROJECT_NAME="${PROJECT_NAME:-$project_name}"
  export ENV="${ENV:-$env}"
  export VERBOSE="$verbose"

  # Load environment files to detect what services to build
  # This tells us WHAT to provision, not HOW to configure it
  load_env_for_detection

  # Validate and fix environment variables (including PROJECT_NAME)
  if command -v validate_environment >/dev/null 2>&1; then
    validate_environment || true
  fi

  # Detect all services and apps
  detect_all_services

  # Now build everything with runtime variables
  build_all_components "$force"

  return 0
}

# Load environment files ONLY for service detection
load_env_for_detection() {
  local env="${ENV:-dev}"

  # Load files in cascade order for proper detection
  # .env.dev -> .env.[env] -> .env
  if [[ -f ".env.dev" ]]; then
    set -a
    source ".env.dev" 2>/dev/null || true
    set +a
  fi

  # Load environment-specific file
  case "$env" in
    staging)
      if [[ -f ".env.staging" ]]; then
        set -a
        source ".env.staging" 2>/dev/null || true
        set +a
      fi
      ;;
    prod|production)
      if [[ -f ".env.prod" ]]; then
        set -a
        source ".env.prod" 2>/dev/null || true
        set +a
      fi
      ;;
  esac

  # Load local overrides last
  if [[ -f ".env" ]]; then
    set -a
    source ".env" 2>/dev/null || true
    set +a
  fi
}

# Detect all services that need to be built
detect_all_services() {
  # Core services detection
  export POSTGRES_ENABLED="${POSTGRES_ENABLED:-true}"
  export HASURA_ENABLED="${HASURA_ENABLED:-true}"
  export AUTH_ENABLED="${AUTH_ENABLED:-true}"
  export NGINX_ENABLED="${NGINX_ENABLED:-true}"

  # Optional services
  export NSELF_ADMIN_ENABLED="${NSELF_ADMIN_ENABLED:-false}"
  export MINIO_ENABLED="${MINIO_ENABLED:-${STORAGE_ENABLED:-false}}"
  export REDIS_ENABLED="${REDIS_ENABLED:-false}"
  export MEILISEARCH_ENABLED="${MEILISEARCH_ENABLED:-false}"
  export MAILPIT_ENABLED="${MAILPIT_ENABLED:-false}"
  export MLFLOW_ENABLED="${MLFLOW_ENABLED:-false}"
  export FUNCTIONS_ENABLED="${FUNCTIONS_ENABLED:-false}"

  # Monitoring bundle
  if [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
    export PROMETHEUS_ENABLED="true"
    export GRAFANA_ENABLED="true"
    export LOKI_ENABLED="true"
    export PROMTAIL_ENABLED="true"
    export TEMPO_ENABLED="true"
    export ALERTMANAGER_ENABLED="true"
    export CADVISOR_ENABLED="true"
    export NODE_EXPORTER_ENABLED="true"
    export POSTGRES_EXPORTER_ENABLED="true"
    export REDIS_EXPORTER_ENABLED="true"
  fi

  # Detect custom services (CS_N)
  detect_custom_services

  # Detect frontend apps
  detect_frontend_apps
}

# Detect custom services
detect_custom_services() {
  export CUSTOM_SERVICES=""
  export CUSTOM_SERVICE_COUNT=0

  for i in {1..20}; do
    local cs_var="CS_${i}"
    local cs_value="${!cs_var:-}"

    if [[ -n "$cs_value" ]]; then
      CUSTOM_SERVICE_COUNT=$((CUSTOM_SERVICE_COUNT + 1))

      # Parse service definition
      IFS=':' read -r name template port <<< "$cs_value"

      # Export service details for build
      export "CS_${i}_NAME=$name"
      export "CS_${i}_TEMPLATE=$template"
      export "CS_${i}_PORT=${port:-$((8000 + i))}"

      # Add to list
      CUSTOM_SERVICES="$CUSTOM_SERVICES $name"
    fi
  done
}

# Detect frontend applications
detect_frontend_apps() {
  export FRONTEND_APPS=""
  export FRONTEND_APP_COUNT=0

  for i in {1..10}; do
    # Support both NAME and SYSTEM_NAME
    local app_name_var="FRONTEND_APP_${i}_NAME"
    local app_system_var="FRONTEND_APP_${i}_SYSTEM_NAME"
    local app_name="${!app_name_var:-${!app_system_var:-}}"

    if [[ -n "$app_name" ]]; then
      FRONTEND_APP_COUNT=$((FRONTEND_APP_COUNT + 1))

      # Export app details for build
      export "FRONTEND_APP_${i}_NAME=$app_name"
      local port_var="FRONTEND_APP_${i}_PORT"
      export "FRONTEND_APP_${i}_PORT=${!port_var:-$((3000 + i - 1))}"

      # Check for remote schema configuration
      local schema_var="FRONTEND_APP_${i}_REMOTE_SCHEMA_NAME"
      if [[ -n "${!schema_var:-}" ]]; then
        export "$schema_var=${!schema_var}"
      fi

      # Add to list
      FRONTEND_APPS="$FRONTEND_APPS $app_name"
    fi
  done
}

# Build all components
build_all_components() {
  local force="${1:-false}"

  echo "Building components for:"
  echo "  • Core services: 4"
  echo "  • Optional services: $(count_enabled_optional)"
  echo "  • Custom services: $CUSTOM_SERVICE_COUNT"
  echo "  • Frontend apps: $FRONTEND_APP_COUNT"
  [[ "$MONITORING_ENABLED" == "true" ]] && echo "  • Monitoring: 10 services"
  echo ""

  # Create directory structure
  setup_directories

  # Generate SSL certificates
  generate_ssl_certificates "$force"

  # Copy custom service templates
  copy_custom_service_templates "$force"

  # Generate nginx configuration with runtime vars
  if [[ -f "$(dirname "${BASH_SOURCE[0]}")/nginx-generator.sh" ]]; then
    source "$(dirname "${BASH_SOURCE[0]}")/nginx-generator.sh"
  fi
  generate_nginx_config "$force"

  # Generate docker-compose with runtime vars
  if command -v generate_docker_compose >/dev/null 2>&1; then
    generate_docker_compose
  else
    # Fallback to compose-generate script
    local compose_script="${NSELF_ROOT:-/Users/admin/Sites/nself}/src/services/docker/compose-generate.sh"
    if [[ -f "$compose_script" ]]; then
      bash "$compose_script"
    fi
  fi

  # Generate database initialization
  if command -v generate_database_init >/dev/null 2>&1; then
    generate_database_init "$force"
  else
    # Basic database init
    mkdir -p postgres/init
    cat > postgres/init/00-init.sql <<'EOF'
-- Database initialization
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
EOF
  fi

  return 0
}

# Setup directory structure
setup_directories() {
  mkdir -p nginx/{conf.d,sites,includes,routes} 2>/dev/null || true
  mkdir -p ssl/certificates 2>/dev/null || true
  mkdir -p postgres/init 2>/dev/null || true
  mkdir -p services 2>/dev/null || true
  mkdir -p monitoring/{prometheus,grafana,loki,alertmanager,tempo} 2>/dev/null || true
  mkdir -p .volumes/{postgres,redis,minio,grafana,prometheus} 2>/dev/null || true
}

# Generate SSL certificates
generate_ssl_certificates() {
  local force="${1:-false}"

  if [[ "$force" == "true" ]] || [[ ! -f "ssl/cert.pem" ]]; then
    # Generate self-signed cert for local development
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
      -keyout ssl/key.pem \
      -out ssl/cert.pem \
      -subj "/C=US/ST=State/L=City/O=Dev/CN=localhost" \
      2>/dev/null || true

    # Also create versioned certs for different domains
    cp ssl/cert.pem ssl/certificates/localhost/fullchain.pem 2>/dev/null || true
    cp ssl/key.pem ssl/certificates/localhost/privkey.pem 2>/dev/null || true
  fi
}

# Copy custom service templates
copy_custom_service_templates() {
  local force="${1:-false}"
  local nself_root="${NSELF_ROOT:-/Users/admin/Sites/nself}"

  for i in {1..20}; do
    local cs_name_var="CS_${i}_NAME"
    local cs_template_var="CS_${i}_TEMPLATE"
    local cs_port_var="CS_${i}_PORT"

    local name="${!cs_name_var:-}"
    local template="${!cs_template_var:-}"
    local port="${!cs_port_var:-}"

    if [[ -n "$name" ]] && [[ -n "$template" ]]; then
      local service_dir="services/$name"

      # Skip if already exists and not forcing
      if [[ -d "$service_dir" ]] && [[ "$force" != "true" ]]; then
        continue
      fi

      # Find and copy template
      for lang in js python go rust; do
        local template_dir="$nself_root/src/templates/services/$lang/$template"
        if [[ -d "$template_dir" ]]; then
          echo "  → Copying template '$template' to services/$name"
          mkdir -p "$service_dir"
          cp -r "$template_dir"/* "$service_dir/" 2>/dev/null || true

          # Replace placeholders
          find "$service_dir" -type f \( -name "*.js" -o -name "*.ts" -o -name "*.py" \
            -o -name "*.go" -o -name "*.json" -o -name "*.yml" -o -name "Dockerfile*" \) \
            -exec sed -i.bak \
              -e "s/{{SERVICE_NAME}}/$name/g" \
              -e "s/{{SERVICE_PORT}}/$port/g" \
              -e "s/{{PROJECT_NAME}}/\${PROJECT_NAME}/g" \
              {} \; 2>/dev/null || true

          # Remove .template extensions
          find "$service_dir" -name "*.template" -exec bash -c 'mv "$1" "${1%.template}"' _ {} \;

          # Cleanup backup files
          find "$service_dir" -name "*.bak" -delete 2>/dev/null || true

          break
        fi
      done
    fi
  done
}

# Count enabled optional services
count_enabled_optional() {
  local count=0
  [[ "$NSELF_ADMIN_ENABLED" == "true" ]] && count=$((count + 1))
  [[ "$MINIO_ENABLED" == "true" ]] && count=$((count + 1))
  [[ "$REDIS_ENABLED" == "true" ]] && count=$((count + 1))
  [[ "$MEILISEARCH_ENABLED" == "true" ]] && count=$((count + 1))
  [[ "$MAILPIT_ENABLED" == "true" ]] && count=$((count + 1))
  [[ "$MLFLOW_ENABLED" == "true" ]] && count=$((count + 1))
  [[ "$FUNCTIONS_ENABLED" == "true" ]] && count=$((count + 1))
  echo $count
}

# Export functions
export -f orchestrate_build
export -f load_env_for_detection
export -f detect_all_services
export -f detect_custom_services
export -f detect_frontend_apps
export -f build_all_components
export -f setup_directories
export -f generate_ssl_certificates
export -f copy_custom_service_templates
export -f count_enabled_optional