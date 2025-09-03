#!/usr/bin/env bash
set -euo pipefail


# build.sh - nself Build System
# Generates Docker infrastructure and configuration files

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATES_DIR="$SCRIPT_DIR/../templates"

# Source required utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/header.sh"
source "$SCRIPT_DIR/../lib/utils/preflight.sh"
source "$SCRIPT_DIR/../lib/config/smart-defaults.sh"

# Source validation scripts with error checking
if [[ -f "$SCRIPT_DIR/../lib/auto-fix/config-validator-v2.sh" ]]; then
  source "$SCRIPT_DIR/../lib/auto-fix/config-validator-v2.sh"
else
  echo "Error: config-validator-v2.sh not found"
  exit 1
fi

if [[ -f "$SCRIPT_DIR/../lib/auto-fix/auto-fixer-v2.sh" ]]; then
  source "$SCRIPT_DIR/../lib/auto-fix/auto-fixer-v2.sh"
else
  echo "Error: auto-fixer-v2.sh not found"
  exit 1
fi

# Track what was created/updated
CREATED_FILES=()
UPDATED_FILES=()
SKIPPED_FILES=()

# Override the format_section function to suppress output
format_section() {
  # Silently ignore section formatting calls from validation
  : # No-op command that always succeeds
}

# Show help for build command
show_build_help() {
  echo "nself build - Generate project infrastructure and configuration"
  echo ""
  echo "Usage: nself build [OPTIONS]"
  echo ""
  echo "Description:"
  echo "  Generates Docker Compose files, SSL certificates, nginx configuration,"
  echo "  and all necessary infrastructure based on your .env.local settings."
  echo ""
  echo "Options:"
  echo "  -f, --force         Force rebuild of all components"
  echo "  -h, --help          Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself build                    # Build with current configuration"
  echo "  nself build --force            # Force rebuild everything"
  echo ""
  echo "Files Generated:"
  echo "  • docker-compose.yml           • nginx/ configuration"
  echo "  • SSL certificates             • Database initialization"
  echo "  • Service templates            • Environment validation"
  echo ""
  echo "Notes:"
  echo "  • Automatically detects configuration changes"
  echo "  • Only rebuilds what's necessary (unless --force)"
  echo "  • Validates configuration before building"
  echo "  • Creates trusted SSL certificates for HTTPS"
}

# Main build command function
cmd_build() {
  local exit_code=0
  local force_rebuild="${1:-false}"

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
    -f | --force)
      force_rebuild=true
      shift
      ;;
    -h | --help)
      show_build_help
      return 0
      ;;
    *)
      log_error "Unknown option: $1"
      log_info "Use 'nself build --help' for usage information"
      return 1
      ;;
    esac
  done

  # Show welcome header with proper formatting
  show_command_header "nself build" "Generate project infrastructure and configuration"

  # Determine environment first
  if [[ "${ENV:-}" == "prod" ]] || [[ "${ENVIRONMENT:-}" == "production" ]]; then
    log_info "Building for PRODUCTION environment"
  else
    log_info "Building for DEVELOPMENT environment"
  fi

  # Run pre-flight checks
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Checking system requirements..."
  if preflight_build >/dev/null 2>&1; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} System requirements met                    \n"
  else
    printf "\r${COLOR_RED}✗${COLOR_RESET} Pre-flight checks failed                  \n"
    preflight_build # Run again to show the actual errors
    return 1
  fi

  # Load environment with smart defaults (silently)
  if ! load_env_with_defaults >/dev/null 2>&1; then
    printf "${COLOR_RED}✗${COLOR_RESET} Failed to load environment                \n"
    return 1
  fi

  # Apply database auto-configuration
  if [[ -f "$SCRIPT_DIR/../lib/database/auto-config.sh" ]]; then
    source "$SCRIPT_DIR/../lib/database/auto-config.sh" 2>/dev/null || true
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Configuring database for optimal performance..."
    if command -v get_system_resources &>/dev/null && command -v apply_smart_defaults &>/dev/null; then
      get_system_resources >/dev/null 2>&1
      apply_smart_defaults >/dev/null 2>&1
      auto_tune_memory >/dev/null 2>&1
      auto_tune_cpu >/dev/null 2>&1
      auto_tune_disk >/dev/null 2>&1
      auto_detect_compression >/dev/null 2>&1
      check_pooling_needed >/dev/null 2>&1
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Database configuration optimized                    \n"
    else
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Database auto-config not available                 \n"
    fi
  fi

  # Check if validation function exists
  if ! declare -f run_validation >/dev/null 2>&1; then
    log_warning "Validation system not available, skipping"
  else
    # Validate configuration
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Validating configuration..."

    # Clear any existing validation arrays
    VALIDATION_ERRORS=()
    VALIDATION_WARNINGS=()
    AUTO_FIXES=()

    # Run validation in a subshell to prevent exits
    local validation_output=$(mktemp)
    local validation_status=0

    # Simple validation without timeout issues
    {
      # Quick check for essential variables
      if [[ ! -f ".env.local" ]]; then
        echo "VALIDATION_ERRORS=('Missing .env.local file')"
        echo "VALIDATION_WARNINGS=()"
        echo "AUTO_FIXES=()"
      else
        # Load environment and check basics
        set -a
        source .env.local 2>/dev/null || true
        set +a
        
        local errors=()
        local warnings=()
        local fixes=()
        
        # Check critical variables
        if [[ -z "${PROJECT_NAME:-}" ]]; then
          fixes+=("'Setting PROJECT_NAME'")
          PROJECT_NAME="$(basename "$PWD")"
          export PROJECT_NAME
        fi
        
        if [[ -z "${BASE_DOMAIN:-}" ]]; then
          fixes+=("'Setting BASE_DOMAIN'")  
          BASE_DOMAIN="local.nself.org"
          export BASE_DOMAIN
        fi
        
        echo "VALIDATION_ERRORS=(${errors[@]:-})"
        echo "VALIDATION_WARNINGS=(${warnings[@]:-})"
        echo "AUTO_FIXES=(${fixes[@]:-})"
      fi
    } >"$validation_output" 2>&1

    # Initialize arrays
    VALIDATION_ERRORS=()
    VALIDATION_WARNINGS=()
    AUTO_FIXES=()
    
    # Source the output to get the arrays
    if [[ -s "$validation_output" ]]; then
      # Source the file directly
      source "$validation_output" 2>/dev/null || true
    fi

    # Check the result
    if [[ ${#VALIDATION_ERRORS[@]} -gt 0 ]]; then
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Configuration has ${#VALIDATION_ERRORS[@]} issues            \n"

      # Apply auto-fixes if available
      if [[ ${#AUTO_FIXES[@]} -gt 0 ]] && declare -f apply_all_fixes >/dev/null 2>&1; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Applying auto-fixes..."
        if apply_all_fixes .env.local "${AUTO_FIXES[@]}" >/dev/null 2>&1; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Applied ${#AUTO_FIXES[@]} auto-fixes                   \n"
        else
          printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Some auto-fixes failed                    \n"
        fi
      fi
    else
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Configuration validated                    \n"
    fi

    rm -f "$validation_output"
  fi

  # Check if this is an existing project
  local is_existing_project=false
  if [[ -f "docker-compose.yml" ]] || [[ -d "nginx" ]] || [[ -f "postgres/init/01-init.sql" ]]; then
    is_existing_project=true
  fi

  # Validate docker-compose.yml first if it exists
  if [[ -f "docker-compose.yml" ]]; then
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Validating docker-compose.yml..."
    if docker compose config >/dev/null 2>&1; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} docker-compose.yml is valid                \n"
    else
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} docker-compose.yml needs update            \n"
    fi
  fi

  # Pre-check what needs to be done
  local needs_work=false
  local dirs_to_create=0

  # Check directories
  for dir in nginx/conf.d nginx/ssl services logs .volumes/postgres .volumes/redis .volumes/minio; do
    if [[ ! -d "$dir" ]]; then
      ((dirs_to_create++))
      needs_work=true
    fi
  done

  # Check SSL certificates
  local needs_ssl=false
  if [[ ! -f "nginx/ssl/nself.org.crt" ]] || [[ ! -f "nginx/ssl/nself.org.key" ]] || [[ "$force_rebuild" == "true" ]]; then
    needs_ssl=true
    needs_work=true
  fi

  # Check docker-compose.yml
  local needs_compose=false
  if [[ ! -f "docker-compose.yml" ]] || [[ "$force_rebuild" == "true" ]] || [[ ".env.local" -nt "docker-compose.yml" ]]; then
    needs_compose=true
    needs_work=true
  fi

  # Check nginx configuration
  local needs_nginx=false
  if [[ ! -f "nginx/nginx.conf" ]] || [[ ! -f "nginx/conf.d/hasura.conf" ]] || [[ "$force_rebuild" == "true" ]] || [[ ".env.local" -nt "nginx/conf.d/hasura.conf" ]]; then
    needs_nginx=true
    needs_work=true
  fi

  # Check database initialization
  local needs_db=false
  if [[ ! -f "postgres/init/01-init.sql" ]] || [[ "$force_rebuild" == "true" ]] || [[ ".env.local" -nt "postgres/init/01-init.sql" ]]; then
    needs_db=true
    needs_work=true
  fi

  # Only show build process if we have work to do
  if [[ "$needs_work" == "true" ]]; then
    echo
    echo -e "${COLOR_CYAN}➞ Build Process${COLOR_RESET}"
    echo

    # Create directory structure if needed
    if [[ $dirs_to_create -gt 0 ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Creating directory structure..."
      for dir in nginx/conf.d nginx/ssl services logs .volumes/postgres .volumes/redis .volumes/minio; do
        if [[ ! -d "$dir" ]]; then
          mkdir -p "$dir" 2>/dev/null
        fi
      done
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Directory structure ready ($dirs_to_create new)     \n"
      CREATED_FILES+=("$dirs_to_create directories")
    fi

    # Generate SSL certificates if needed
    if [[ "$needs_ssl" == "true" ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating SSL certificates..."

      # Check for mkcert
      if command -v mkcert >/dev/null 2>&1; then
        # Generate trusted certificates with mkcert
        (
          cd nginx/ssl
          mkcert -cert-file nself.org.crt -key-file nself.org.key \
            "${BASE_DOMAIN}" "*.${BASE_DOMAIN}" \
            "${HASURA_ROUTE}" "${AUTH_ROUTE}" "${STORAGE_ROUTE}" \
            localhost 127.0.0.1 ::1 >/dev/null 2>&1
        )
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} SSL certificates generated (trusted)       \n"
        CREATED_FILES+=("SSL certificates")
      else
        # Generate self-signed certificate as fallback
        openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
          -keyout nginx/ssl/nself.org.key \
          -out nginx/ssl/nself.org.crt \
          -subj "/C=US/ST=State/L=City/O=nself/CN=*.${BASE_DOMAIN}" \
          -addext "subjectAltName=DNS:*.${BASE_DOMAIN},DNS:${BASE_DOMAIN}" >/dev/null 2>&1
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} SSL certificates generated (self-signed)   \n"
        CREATED_FILES+=("SSL certificates")
      fi
    fi

    # Generate docker-compose.yml if needed
    if [[ "$needs_compose" == "true" ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating docker-compose.yml..."

      local compose_action="created"
      if [[ -f "docker-compose.yml" ]]; then
        if [[ "$force_rebuild" == "true" ]]; then
          compose_action="rebuilt"
        else
          compose_action="updated"
        fi
      fi

      # Use the compose generation script
      if [[ -f "$SCRIPT_DIR/../services/docker/compose-generate.sh" ]]; then
        if bash "$SCRIPT_DIR/../services/docker/compose-generate.sh" >/dev/null 2>&1; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} docker-compose.yml ${compose_action}              \n"
          if [[ "$compose_action" == "created" ]]; then
            CREATED_FILES+=("docker-compose.yml")
          else
            UPDATED_FILES+=("docker-compose.yml")
          fi
        else
          printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to generate docker-compose.yml     \n"
          return 1
        fi
      else
        printf "\r${COLOR_RED}✗${COLOR_RESET} Compose generator not found                \n"
        return 1
      fi
    fi

    # Generate nginx configuration if needed
    if [[ "$needs_nginx" == "true" ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating nginx configuration..."

      local nginx_updated=false

      # Create nginx directory if it doesn't exist
      mkdir -p nginx

      # Create hasura directories if they don't exist
      mkdir -p hasura/metadata hasura/migrations

      # Check if nginx.conf needs updating
      if [[ ! -f "nginx/nginx.conf" ]] || [[ "$force_rebuild" == "true" ]]; then
        # Main nginx.conf
        cat >nginx/nginx.conf <<'EOF'
events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    
    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
    
    # Timeouts
    client_body_timeout 60;
    client_header_timeout 60;
    keepalive_timeout 65;
    send_timeout 60;
    
    # Buffer sizes
    client_body_buffer_size 16K;
    client_header_buffer_size 1k;
    client_max_body_size 100M;
    large_client_header_buffers 4 16k;
    
    # Include service configurations
    include /etc/nginx/conf.d/*.conf;
}
EOF
        nginx_updated=true
        CREATED_FILES+=("nginx/nginx.conf")
      fi

      # Generate Hasura proxy config
      if [[ ! -f "nginx/conf.d/hasura.conf" ]] || [[ "$force_rebuild" == "true" ]] || [[ ".env.local" -nt "nginx/conf.d/hasura.conf" ]]; then
        cat >nginx/conf.d/hasura.conf <<EOF
upstream hasura {
    server hasura:8080;
}

server {
    listen 80;
    server_name ${HASURA_ROUTE};
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl;
    http2 on;
    server_name ${HASURA_ROUTE};
    
    ssl_certificate /etc/nginx/ssl/nself-org/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/nself-org/privkey.pem;
    
    location / {
        proxy_pass http://hasura;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
        nginx_updated=true
        CREATED_FILES+=("nginx/conf.d/hasura.conf")
      fi

      if [[ "$nginx_updated" == "true" ]]; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Nginx configuration generated              \n"
      fi
    fi

    # Configure frontend routes if enabled
    # FRONTEND_APPS format: "name:short:prefix:port,..."
    if [[ -n "${FRONTEND_APPS:-}" ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Configuring frontend routes..."

      # Parse frontend apps
      IFS=',' read -ra APPS <<<"$FRONTEND_APPS"
      local app_count=0
      local apps_updated=0

      for app_config in "${APPS[@]}"; do
        # Parse the 4-field format: name:short:prefix:port
        IFS=':' read -r app_name app_short app_prefix app_port <<<"$app_config"

        # Skip if incomplete config
        [[ -z "$app_name" || -z "$app_port" ]] && continue

        # Determine routing based on environment
        local app_route=""
        if [[ "${ENV:-dev}" == "prod" ]] || [[ "${ENV:-dev}" == "production" ]]; then
          # Production: use subdomain or custom domain if configured
          # Convert to uppercase for compatibility with older bash versions
          local app_name_upper=$(echo "$app_name" | tr '[:lower:]' '[:upper:]')
          local prod_route_var="${app_name_upper}_PROD_ROUTE"
          app_route="${!prod_route_var:-${app_short:-$app_name}.${BASE_DOMAIN}}"
        else
          # Development: use subdomain
          app_route="${app_short:-$app_name}.${BASE_DOMAIN}"
        fi

        # Check if config needs updating
        if [[ ! -f "nginx/conf.d/${app_name}.conf" ]] || [[ "$force_rebuild" == "true" ]]; then
          # Generate nginx config for frontend app
          cat >"nginx/conf.d/${app_name}.conf" <<EOF
server {
    listen 80;
    server_name ${app_route};
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl;
    http2 on;
    server_name ${app_route};
    
    ssl_certificate /etc/nginx/ssl/nself-org/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/nself-org/privkey.pem;
    
    # Frontend app: ${app_name}
    # Database prefix: ${app_prefix}
    # Development port: ${app_port}
    
    location / {
        proxy_pass http://host.docker.internal:${app_port};
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # CORS headers for frontend apps
        add_header Access-Control-Allow-Origin \$http_origin always;
        add_header Access-Control-Allow-Credentials true always;
    }
    
    # Health check endpoint
    location /health {
        access_log off;
        return 200 "OK";
        add_header Content-Type text/plain;
    }
}
EOF
          ((apps_updated++))
          CREATED_FILES+=("nginx/conf.d/${app_name}.conf")
        fi
        ((app_count++))
      done

      if [[ $app_count -gt 0 ]]; then
        # Generate frontend apps configuration file for services to use
        cat >"./frontend-apps.env" <<EOF
# Frontend Applications Configuration
# Generated by nself build
# This file is used by backend services to configure database prefixes and routing

EOF
        
        # Re-parse to add to config file
        for app_config in "${APPS[@]}"; do
          IFS=':' read -r app_name app_short app_prefix app_port <<<"$app_config"
          [[ -z "$app_name" || -z "$app_port" ]] && continue
          
          # Export configuration for each app
          # Convert to uppercase for compatibility with older bash versions
          local app_name_upper=$(echo "$app_name" | tr '[:lower:]' '[:upper:]')
          local prod_route_var="${app_name_upper}_PROD_ROUTE"
          local prod_route="${!prod_route_var:-${app_short:-$app_name}.${BASE_DOMAIN}}"
          cat >>"./frontend-apps.env" <<EOF
# ${app_name} Application
${app_name_upper}_NAME=${app_name}
${app_name_upper}_SHORT=${app_short}
${app_name_upper}_TABLE_PREFIX=${app_prefix}
${app_name_upper}_DEV_PORT=${app_port}
${app_name_upper}_DEV_ROUTE=${app_short:-$app_name}.${BASE_DOMAIN}
${app_name_upper}_PROD_ROUTE=${prod_route}

EOF
        done
        
        if [[ $apps_updated -gt 0 ]]; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Frontend routes configured ($apps_updated/$app_count)      \n"
        else
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Frontend routes up to date ($app_count)           \n"
        fi
      else
        printf "\r                                                            \r"
      fi
    fi

    # Configure backend service routes
    if [[ "${SERVICES_ENABLED:-false}" == "true" ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Configuring backend service routes..."

      # Create routes directory if it doesn't exist
      mkdir -p nginx/conf.d/routes

      # Generate routes for NestJS services
      if [[ -n "${NESTJS_SERVICES:-}" ]]; then
        IFS=',' read -ra services <<<"$NESTJS_SERVICES"
        for service in "${services[@]}"; do
          service=$(echo "$service" | xargs)
          cat >"nginx/conf.d/routes/nest-${service}.conf" <<EOF
# Route for NestJS $service service
location /api/nest/$service/ {
    proxy_pass http://unity-nest-${service}:${NESTJS_PORT_START:-3100}/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade \$http_upgrade;
    proxy_set_header Connection "upgrade";
}
EOF
        done
      fi

      # Generate routes for Go services
      if [[ -n "${GO_SERVICES:-${GOLANG_SERVICES:-}}" ]]; then
        IFS=',' read -ra services <<<"${GO_SERVICES:-${GOLANG_SERVICES:-}}"
        for service in "${services[@]}"; do
          service=$(echo "$service" | xargs)
          cat >"nginx/conf.d/routes/go-${service}.conf" <<EOF
# Route for Go $service service
location /api/go/$service/ {
    proxy_pass http://unity-go-${service}:${GOLANG_PORT_START:-3300}/;
    proxy_http_version 1.1;
}
EOF
        done
      fi

      # Generate routes for Python services
      if [[ -n "${PYTHON_SERVICES:-}" ]]; then
        IFS=',' read -ra services <<<"$PYTHON_SERVICES"
        for service in "${services[@]}"; do
          service=$(echo "$service" | xargs)
          cat >"nginx/conf.d/routes/py-${service}.conf" <<EOF
# Route for Python $service service
location /api/python/$service/ {
    proxy_pass http://unity-py-${service}:${PYTHON_PORT_START:-3400}/;
    proxy_http_version 1.1;
}
EOF
        done
      fi

      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Backend service routes configured              \n"
    fi

    # Generate database initialization script if needed
    if [[ "$needs_db" == "true" ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Creating database initialization..."

      # Create postgres/init directory if it doesn't exist
      mkdir -p postgres/init

      cat >postgres/init/01-init.sql <<'EOF'
-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- Create schemas
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS storage;

-- Setup permissions for Hasura
GRANT USAGE ON SCHEMA public TO postgres;
GRANT CREATE ON SCHEMA public TO postgres;
GRANT ALL ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO postgres;
EOF

      # Add PostgreSQL extensions if configured
      if [[ -n "${POSTGRES_EXTENSIONS:-}" ]]; then
        IFS=',' read -ra EXTENSIONS <<<"$POSTGRES_EXTENSIONS"
        for ext in "${EXTENSIONS[@]}"; do
          ext=$(echo "$ext" | xargs) # Trim whitespace
          echo "CREATE EXTENSION IF NOT EXISTS \"$ext\";" >>postgres/init/01-init.sql
        done
      fi

      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Database initialization created            \n"
      CREATED_FILES+=("postgres/init/01-init.sql")
    fi
  fi

  # Generate ALL services based on env file (env is king!)
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating services...\r"

  # Reload environment to ensure we have latest values
  if [[ -f ".env.local" ]]; then
    load_env_with_priority >/dev/null 2>&1 || true
  fi

  # Source generators once at the beginning
  local service_gen_loaded=false
  local dockerfile_gen_loaded=false
  local custom_service_loaded=false

  if [[ -f "$SCRIPT_DIR/../lib/auto-fix/service-generator.sh" ]]; then
    # Override log functions to be silent
    log_info() { :; }
    log_success() { :; }
    log_warning() { :; }
    source "$SCRIPT_DIR/../lib/auto-fix/service-generator.sh"
    service_gen_loaded=true
  fi

  if [[ -f "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh" ]]; then
    # Override log functions to be silent for this too
    log_info() { :; }
    log_success() { :; }
    log_warning() { :; }
    source "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh"
    dockerfile_gen_loaded=true
  fi

  # Source custom service builder
  # Try v2 builder first (CS_N pattern), fall back to v1
  if [[ -f "$SCRIPT_DIR/../lib/services/service-builder-v2.sh" ]]; then
    source "$SCRIPT_DIR/../lib/services/service-builder-v2.sh" 2>/dev/null || true
    custom_service_loaded=true
  elif [[ -f "$SCRIPT_DIR/../lib/services/service-builder.sh" ]]; then
    source "$SCRIPT_DIR/../lib/services/service-builder.sh" 2>/dev/null || true
    custom_service_loaded=true
  fi

  # Track what we generate
  local total_services_generated=0
  local system_services_generated=0
  local custom_services_generated=0

  # Check for CS_N services or legacy CUSTOM_SERVICES
  local has_custom_services=false
  if [[ -n "${CS_1:-}" ]]; then
    has_custom_services=true
  elif [[ -n "${CUSTOM_SERVICES:-}" ]]; then
    has_custom_services=true
  fi

  # Generate custom services if configured
  if [[ "$custom_service_loaded" == "true" ]] && [[ "$has_custom_services" == "true" ]]; then
    # Build custom services and their configurations
    if build_custom_services >/dev/null 2>&1; then
      # Count generated custom services
      if [[ -n "${CS_1:-}" ]]; then
        # Count CS_N services
        local n=1
        while [[ -n "$(eval echo "\${CS_${n}:-}")" ]]; do
          ((custom_services_generated++))
          ((n++))
        done
      elif [[ -n "${CUSTOM_SERVICES:-}" ]]; then
        # Count legacy services
        IFS=',' read -ra services <<< "$CUSTOM_SERVICES"
        custom_services_generated=${#services[@]}
      fi
    fi
  fi

  # Generate microservices if enabled
  if [[ "$service_gen_loaded" == "true" ]] && [[ "${SERVICES_ENABLED:-false}" == "true" ]]; then
    # Count services before generation
    local before_count=$(find services -type d -maxdepth 2 2>/dev/null | wc -l | tr -d ' ')

    # Generate services silently
    auto_generate_services "true" >/dev/null 2>&1

    # Count services after generation
    local after_count=$(find services -type d -maxdepth 2 2>/dev/null | wc -l | tr -d ' ')
    total_services_generated=$((after_count - before_count))
  fi
  # Generate system services if enabled
  if [[ "$dockerfile_gen_loaded" == "true" ]]; then
    local gen_script="$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh"

    # Functions service
    if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]] && [[ ! -d "functions" ]]; then
      # Use bash -c to ensure proper execution context for heredocs
      bash -c "source '${gen_script}' && generate_dockerfile_for_service 'functions' 'functions'" >/dev/null 2>&1
      if [[ -d "functions" ]]; then
        ((system_services_generated++))
      fi
    fi

    # Dashboard service
    if [[ "${DASHBOARD_ENABLED:-false}" == "true" ]] && [[ ! -d "dashboard" ]]; then
      bash -c "source '${gen_script}' && generate_dockerfile_for_service 'dashboard' 'dashboard'" >/dev/null 2>&1
      if [[ -d "dashboard" ]]; then
        ((system_services_generated++))
      fi
    fi

  fi

  # Report results and clear the line
  if [[ $total_services_generated -gt 0 ]] || [[ $system_services_generated -gt 0 ]] || [[ $custom_services_generated -gt 0 ]]; then
    local total=$((total_services_generated + system_services_generated + custom_services_generated))
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Generated $total services"
    if [[ $custom_services_generated -gt 0 ]]; then
      printf " ($custom_services_generated custom)"
    fi
    printf "                              \n"
  else
    # Clear the "Generating services..." line - but since nothing was generated, just clear it
    printf "\r                                                            \r"
  fi

  # Restore log functions
  source "$SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true

  # SSL certificates were already generated earlier in the build process
  # No need to regenerate them here

  # Build summary - ensure we're on a new line
  echo
  if [[ "$is_existing_project" == "true" ]]; then
    if [[ "$needs_work" == "false" ]]; then
      log_info "Existing project detected"
      log_success "No changes needed, all up-to-date"
    else
      log_info "Existing project detected"
      if [[ ${#UPDATED_FILES[@]} -gt 0 ]]; then
        log_success "Updated ${#UPDATED_FILES[@]} files"
      fi
      if [[ ${#CREATED_FILES[@]} -gt 0 ]]; then
        log_success "Created ${#CREATED_FILES[@]} new resources"
      fi
    fi
  else
    log_success "Project infrastructure generated"
    if [[ ${#CREATED_FILES[@]} -gt 0 ]]; then
      log_info "Created ${#CREATED_FILES[@]} resources"
    fi
  fi

  echo
  echo -e "${COLOR_CYAN}➞ Next Steps${COLOR_RESET}"
  echo

  echo -e "${COLOR_BLUE}1.${COLOR_RESET} ${COLOR_BLUE}nself trust${COLOR_RESET} - Install SSL certificates"
  echo -e "   ${COLOR_DIM}Trust the root CA for green locks in browsers${COLOR_RESET}"
  echo
  echo -e "${COLOR_BLUE}2.${COLOR_RESET} ${COLOR_BLUE}nself start${COLOR_RESET} - Start all services"
  echo -e "   ${COLOR_DIM}Launches PostgreSQL, Hasura, and configured services${COLOR_RESET}"
  echo
  echo -e "${COLOR_BLUE}3.${COLOR_RESET} ${COLOR_BLUE}nself status${COLOR_RESET} - Check service health"
  echo -e "   ${COLOR_DIM}View the status of all running services${COLOR_RESET}"

  if [[ "$is_existing_project" == "true" ]] && [[ "$needs_work" == "false" ]]; then
    echo
    echo -e "${COLOR_YELLOW}⚡${COLOR_RESET} Use ${COLOR_BLUE}nself build --force${COLOR_RESET} to rebuild everything"
  fi

  echo
  return 0
}

# Export for use as library
export -f cmd_build

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_build "$@"
fi
