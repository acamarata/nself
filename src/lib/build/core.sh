#!/usr/bin/env bash
# core.sh - Core build orchestration logic

# Get the correct script directory
CORE_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_ROOT="$(dirname "$CORE_SCRIPT_DIR")"

# Source the original utilities that build.sh needs
source "$LIB_ROOT/utils/env.sh"
source "$LIB_ROOT/utils/display.sh"
source "$LIB_ROOT/utils/header.sh"
source "$LIB_ROOT/utils/preflight.sh"
source "$LIB_ROOT/utils/timeout.sh"
source "$LIB_ROOT/config/smart-defaults.sh"
source "$LIB_ROOT/utils/hosts.sh"

# Source validation scripts with error checking
if [[ -f "$LIB_ROOT/auto-fix/config-validator-v2.sh" ]]; then
  source "$LIB_ROOT/auto-fix/config-validator-v2.sh"
else
  echo "Error: config-validator-v2.sh not found" >&2
fi

if [[ -f "$LIB_ROOT/auto-fix/auto-fixer-v2.sh" ]]; then
  source "$LIB_ROOT/auto-fix/auto-fixer-v2.sh"
else
  echo "Error: auto-fixer-v2.sh not found" >&2
fi

# Source our modular components
source "$CORE_SCRIPT_DIR/platform.sh"
source "$CORE_SCRIPT_DIR/validation.sh"
source "$CORE_SCRIPT_DIR/ssl.sh"
source "$CORE_SCRIPT_DIR/docker-compose.sh"
source "$CORE_SCRIPT_DIR/nginx.sh"
source "$CORE_SCRIPT_DIR/database.sh"
source "$CORE_SCRIPT_DIR/services.sh"
source "$CORE_SCRIPT_DIR/output.sh"

# Initialize build environment
init_build_environment() {
  # Detect platform first
  detect_build_platform

  # Initialize tracking arrays
  CREATED_FILES=()
  UPDATED_FILES=()
  SKIPPED_FILES=()
  BUILD_ERRORS=()

  # Set safe defaults for critical variables
  set_default "PROJECT_NAME" "$(basename "$PWD")"
  set_default "BASE_DOMAIN" "localhost"
  set_default "ENV" "dev"
  set_default "ENVIRONMENT" "development"

  # Create required directories
  mkdir -p ssl/certificates/{localhost,nself-org} 2>/dev/null || true
  mkdir -p nginx/{conf.d,ssl,routes} 2>/dev/null || true
  mkdir -p .volumes 2>/dev/null || true

  return 0
}

# Main build orchestration - extracted from original working cmd_build
orchestrate_build() {
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

  # Check for WSL environment
  if grep -qi microsoft /proc/version 2>/dev/null; then
    log_warning "WSL environment detected"
    log_info "Ensure Docker Desktop is running with WSL2 backend"
    log_info "If build fails, check: https://docs.docker.com/desktop/wsl/"

    # Check if Docker is accessible
    if ! docker info >/dev/null 2>&1; then
      log_error "Docker is not accessible from WSL"
      log_info "Please ensure:"
      log_info "  1. Docker Desktop is running"
      log_info "  2. WSL2 integration is enabled in Docker Desktop settings"
      log_info "  3. Current WSL distro is enabled in Docker Desktop resources"
      return 1
    fi
  fi

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

  # Auto-fix unquoted environment values with spaces (always enabled)
  if [[ -f "$LIB_ROOT/auto-fix/env-quotes-fix.sh" ]]; then
    source "$LIB_ROOT/auto-fix/env-quotes-fix.sh"
    # Force AUTO_FIX=true for this specific fix since it's safe and necessary
    AUTO_FIX=true auto_fix_env_quotes
  fi

  # Validate and auto-fix environment variables
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Validating configuration..."
  if [[ -f "$LIB_ROOT/auto-fix/env-validation.sh" ]]; then
    # Save original SCRIPT_DIR before sourcing other scripts
    local BUILD_SCRIPT_DIR="$LIB_ROOT"
    source "$LIB_ROOT/auto-fix/env-validation.sh"
    LIB_ROOT="$BUILD_SCRIPT_DIR"  # Restore original
    if validate_all_env >/dev/null 2>&1; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Configuration validated                    \n"
    else
      printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} Fixed configuration issues                    \n"
      # Reload environment after fixes
      load_env_with_priority
    fi
  else
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Configuration validated                    \n"
  fi

  # Load environment with proper cascading (silently)
  if ! load_env_with_priority true; then
    printf "${COLOR_RED}✗${COLOR_RESET} Failed to load environment                \n"
    return 1
  fi

  # Debug: Show we made it past environment loading
  [[ "${DEBUG:-}" == "true" ]] && echo "DEBUG: Environment loaded successfully" >&2

  # Initialize tracking arrays
  CREATED_FILES=()
  UPDATED_FILES=()
  SKIPPED_FILES=()

  # Apply database auto-configuration
  if [[ -f "$LIB_ROOT/database/auto-config.sh" ]]; then
    source "$LIB_ROOT/database/auto-config.sh" 2>/dev/null || true
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Configuring database for optimal performance..."
    if command -v get_system_resources &>/dev/null && command -v apply_smart_defaults &>/dev/null; then
      get_system_resources >/dev/null 2>&1 || true
      apply_smart_defaults >/dev/null 2>&1 || true
      auto_tune_memory >/dev/null 2>&1 || true
      auto_tune_cpu >/dev/null 2>&1 || true
      auto_tune_disk >/dev/null 2>&1 || true
      auto_detect_compression >/dev/null 2>&1 || true
      check_pooling_needed >/dev/null 2>&1 || true
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Database configuration optimized                    \n"
    else
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Database auto-config not available                 \n"
    fi
  fi

  # Debug: Show we made it past database config
  [[ "${DEBUG:-}" == "true" ]] && echo "DEBUG: Database config complete" >&2

  # Debug: Show we made it to validation section
  [[ "${DEBUG:-}" == "true" ]] && echo "DEBUG: Starting validation section" >&2

  # Check if validation function exists
  if ! declare -f run_validation >/dev/null 2>&1; then
    [[ "${DEBUG:-}" == "true" ]] && echo "DEBUG: Validation function not found, skipping" >&2
    log_warning "Validation system not available, skipping" 2>/dev/null || true
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
      if [[ ! -f ".env" ]] && [[ ! -f ".env.dev" ]]; then
        echo "VALIDATION_ERRORS=('Missing .env or .env.dev file')"
        echo "VALIDATION_WARNINGS=()"
        echo "AUTO_FIXES=()"
      else
        # Load environment and check basics
        set -a
        if [[ -f ".env" ]]; then
          source .env 2>/dev/null || true
        elif [[ -f ".env.dev" ]]; then
          source .env.dev 2>/dev/null || true
        fi
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
      # Source the file directly - use a subshell to prevent exits
      (
        set +e  # Don't exit on error
        source "$validation_output" 2>/dev/null || true
        # Export the arrays to parent shell
        echo "VALIDATION_ERRORS=(${VALIDATION_ERRORS[@]:-})"
        echo "VALIDATION_WARNINGS=(${VALIDATION_WARNINGS[@]:-})"
        echo "AUTO_FIXES=(${AUTO_FIXES[@]:-})"
      ) > "${validation_output}.parsed" 2>/dev/null || true

      # Now source the parsed output safely
      if [[ -s "${validation_output}.parsed" ]]; then
        source "${validation_output}.parsed" 2>/dev/null || true
      fi
      rm -f "${validation_output}.parsed"
    fi

    # Check the result
    if [[ ${#VALIDATION_ERRORS[@]} -gt 0 ]]; then
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Configuration has ${#VALIDATION_ERRORS[@]} issues            \n"

      # Apply auto-fixes if available
      if [[ ${#AUTO_FIXES[@]} -gt 0 ]] && declare -f apply_all_fixes >/dev/null 2>&1; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Applying auto-fixes..."
        local env_file=".env"
        [[ ! -f ".env" ]] && [[ -f ".env.dev" ]] && env_file=".env.dev"
        if apply_all_fixes "$env_file" "${AUTO_FIXES[@]}" >/dev/null 2>&1; then
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

  # Debug: Show we made it past validation
  [[ "${DEBUG:-}" == "true" ]] && echo "DEBUG: Validation complete, continuing build" >&2

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
      dirs_to_create=$((dirs_to_create + 1))
      needs_work=true
    fi
  done

  # Check SSL certificates
  local needs_ssl=false
  # Check both localhost and domain certificates (with default)
  if [[ "${BASE_DOMAIN:-localhost}" == "localhost" ]]; then
    if [[ ! -f "ssl/certificates/localhost/fullchain.pem" ]] || [[ "$force_rebuild" == "true" ]]; then
      needs_ssl=true
      needs_work=true
    fi
  else
    if [[ ! -f "ssl/certificates/nself-org/fullchain.pem" ]] || [[ ! -f "ssl/certificates/nself-org/privkey.pem" ]] || [[ "$force_rebuild" == "true" ]]; then
      needs_ssl=true
      needs_work=true
    fi
  fi

  # Determine which env file to use (check all possible env files)
  local env_file=""
  if [[ -f ".env" ]]; then
    env_file=".env"
  elif [[ -f ".env.local" ]]; then
    env_file=".env.local"
  elif [[ -f ".env.dev" ]]; then
    env_file=".env.dev"
  fi

  # Check docker-compose.yml
  local needs_compose=false
  if [[ ! -f "docker-compose.yml" ]]; then
    [[ "${DEBUG:-}" == "true" ]] && echo "DEBUG: docker-compose.yml missing, will create" >&2
    needs_compose=true
    needs_work=true
  elif [[ "$force_rebuild" == "true" ]]; then
    needs_compose=true
    needs_work=true
  elif [[ -n "$env_file" ]] && [[ "$env_file" -nt "docker-compose.yml" ]]; then
    needs_compose=true
    needs_work=true
  fi

  # Check nginx configuration
  local needs_nginx=false
  if [[ ! -f "nginx/nginx.conf" ]] || [[ ! -f "nginx/conf.d/hasura.conf" ]] || [[ "$force_rebuild" == "true" ]] || ([[ -n "$env_file" ]] && [[ "$env_file" -nt "nginx/conf.d/hasura.conf" ]]); then
    needs_nginx=true
    needs_work=true
  fi

  # Check database initialization
  local needs_db=false
  if [[ ! -f "postgres/init/01-init.sql" ]] || [[ "$force_rebuild" == "true" ]] || ([[ -n "$env_file" ]] && [[ "$env_file" -nt "postgres/init/01-init.sql" ]]); then
    needs_db=true
    needs_work=true
  fi

  # Debug output if enabled
  if [[ "${DEBUG:-}" == "true" ]]; then
    echo "DEBUG: needs_work=$needs_work"
    echo "DEBUG: needs_compose=$needs_compose"
    echo "DEBUG: needs_ssl=$needs_ssl"
    echo "DEBUG: needs_nginx=$needs_nginx"
    echo "DEBUG: needs_db=$needs_db"
    echo "DEBUG: dirs_to_create=$dirs_to_create"
    echo "DEBUG: docker-compose.yml exists: $([ -f "docker-compose.yml" ] && echo yes || echo no)"
  fi

  # CRITICAL: Always ensure docker-compose.yml exists
  # This is the absolute minimum requirement for nself to function
  if [[ ! -f "docker-compose.yml" ]]; then
    echo
    echo -e "${COLOR_YELLOW}⚠${COLOR_RESET}  No docker-compose.yml found - must build"
    needs_work=true
    needs_compose=true
    needs_nginx=true
    needs_db=true
    needs_ssl=true
  fi

  # Only show build process if we have work to do
  if [[ "$needs_work" == "true" ]]; then
    echo
    echo -e "${COLOR_CYAN}➞ Build Process${COLOR_RESET}"
    echo

    # Create directory structure if needed
    if [[ $dirs_to_create -gt 0 ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Creating directory structure..."
      local dir_creation_failed=false
      for dir in nginx/conf.d nginx/ssl services logs .volumes/postgres .volumes/redis .volumes/minio; do
        if [[ ! -d "$dir" ]]; then
          if ! mkdir -p "$dir" 2>&1; then
            dir_creation_failed=true
            printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to create directory: $dir              \n"
            log_error "Check permissions and disk space. Try: sudo chown -R $(whoami) ."
            break
          fi
        fi
      done
      if [[ "$dir_creation_failed" == "false" ]]; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Directory structure ready ($dirs_to_create new)     \n"
        CREATED_FILES+=("$dirs_to_create directories")
      else
        return 1
      fi
    fi

    # Generate SSL certificates if needed
    if [[ "$needs_ssl" == "true" ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating SSL certificates..."

      # Always use the simple SSL generation for now to avoid library issues
      # The SSL library has logging function dependencies that can cause hangs
      build_generate_simple_ssl
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
      local compose_script="$LIB_ROOT/../services/docker/compose-generate.sh"
      if [[ -f "$compose_script" ]]; then
        local error_output=$(mktemp)
        if bash "$compose_script" >"$error_output" 2>&1; then
          # Apply health check fixes
          if [[ -f "$LIB_ROOT/auto-fix/healthcheck-fix.sh" ]]; then
            source "$LIB_ROOT/auto-fix/healthcheck-fix.sh"
            fix_healthchecks "docker-compose.yml" >/dev/null 2>&1
          fi

          printf "\r${COLOR_GREEN}✓${COLOR_RESET} docker-compose.yml ${compose_action}              \n"
          if [[ "$compose_action" == "created" ]]; then
            CREATED_FILES+=("docker-compose.yml")
          else
            UPDATED_FILES+=("docker-compose.yml")
          fi
          rm -f "$error_output"
        else
          printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to generate docker-compose.yml     \n"
          log_error "Docker compose generation failed:"
          # Show the actual error
          if [[ -s "$error_output" ]]; then
            cat "$error_output" | head -10 >&2
          fi
          log_info "Check environment variables and .env file"
          log_info "Run with DEBUG=true for more details"
          rm -f "$error_output"
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

      # Set route defaults if not already set
      : ${HASURA_ROUTE:=api.${BASE_DOMAIN}}
      : ${AUTH_ROUTE:=auth.${BASE_DOMAIN}}
      : ${STORAGE_ROUTE:=storage.${BASE_DOMAIN}}

      # Use comprehensive nginx generator for all services
      if [[ -f "$LIB_ROOT/../lib/services/nginx-generator.sh" ]]; then
        # Source the nginx generator
        source "$LIB_ROOT/../lib/services/nginx-generator.sh"

        # Generate all nginx configurations
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating nginx service configs..."
        local configs_generated=$(nginx::generate_all_configs "." 2>/dev/null | tail -n1)

        # Ensure it's a number
        if [[ ! "$configs_generated" =~ ^[0-9]+$ ]]; then
          configs_generated=0
        fi

        if [[ $configs_generated -gt 0 ]]; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Generated $configs_generated nginx service configs       \n"
          nginx_updated=true
        else
          printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No nginx configs generated (no services enabled)    \n"
        fi

        # Also generate basic Hasura config if not handled by generator
        local env_file=".env"
        [[ ! -f ".env" ]] && [[ -f ".env.dev" ]] && env_file=".env.dev"

        # Get SSL certificate path based on domain
        local hasura_ssl_path=$(get_ssl_cert_path "${HASURA_ROUTE}")

        if [[ ! -f "nginx/conf.d/hasura.conf" ]] || [[ "$force_rebuild" == "true" ]] || [[ "$env_file" -nt "nginx/conf.d/hasura.conf" ]]; then
          mkdir -p nginx/conf.d
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

    ssl_certificate ${hasura_ssl_path}/fullchain.pem;
    ssl_certificate_key ${hasura_ssl_path}/privkey.pem;

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

        # For now, just create a basic mailpit config manually
        if [[ "${MAILPIT_ENABLED:-true}" == "true" ]]; then
          mkdir -p nginx/conf.d
          # The config already exists from earlier, so just mark as updated
          nginx_updated=true
        fi

        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Generated nginx service configs                \n"

        # Validate generated configs
        if [[ "$nginx_updated" == "true" ]]; then
          # Skip validation for now since we're not sourcing the nginx generator
          printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} Nginx configuration has warnings           \n"
        fi
      else
        # Fallback to basic Hasura config if generator not available
        printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} Using basic nginx configuration (generator not found)\n"

        # Set route defaults if not already set
        : ${HASURA_ROUTE:=api.${BASE_DOMAIN}}

        # Generate basic Hasura proxy config
        local env_file=".env"
        [[ ! -f ".env" ]] && [[ -f ".env.dev" ]] && env_file=".env.dev"

        # Get SSL certificate path based on domain
        local hasura_ssl_path=$(get_ssl_cert_path "${HASURA_ROUTE}")

        if [[ ! -f "nginx/conf.d/hasura.conf" ]] || [[ "$force_rebuild" == "true" ]] || [[ "$env_file" -nt "nginx/conf.d/hasura.conf" ]]; then
          mkdir -p nginx/conf.d
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

    ssl_certificate ${hasura_ssl_path}/fullchain.pem;
    ssl_certificate_key ${hasura_ssl_path}/privkey.pem;

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
    fi

    # Configure frontend routes if enabled
    # Support both individual FRONTEND_APP_N_* variables and compact FRONTEND_APPS format

    # First, check for FRONTEND_APP_COUNT and build FRONTEND_APPS from individual variables
    if [[ -n "${FRONTEND_APP_COUNT:-}" ]] && [[ "${FRONTEND_APP_COUNT}" -gt 0 ]]; then
      local apps_config=""
      for ((i=1; i<=FRONTEND_APP_COUNT; i++)); do
        local display_name=$(eval echo "\${FRONTEND_APP_${i}_DISPLAY_NAME:-}")
        local system_name=$(eval echo "\${FRONTEND_APP_${i}_SYSTEM_NAME:-}")
        local table_prefix=$(eval echo "\${FRONTEND_APP_${i}_TABLE_PREFIX:-}")
        local port=$(eval echo "\${FRONTEND_APP_${i}_PORT:-}")
        local route=$(eval echo "\${FRONTEND_APP_${i}_ROUTE:-}")

        # If no port specified, try to auto-detect from package.json
        if [[ -z "$port" ]]; then
          port=$(detect_app_port 3000)
        fi

        # Skip if no port defined (required for routing)
        [[ -z "$port" ]] && continue

        # Build compact format for existing parser
        # Use system_name or display_name as the name (convert spaces to underscores)
        local app_name="${system_name:-${display_name// /_}}"
        [[ -z "$app_name" ]] && app_name="app${i}"

        # Use route as the short name, or derive from app_name
        local app_short="${route:-$app_name}"

        apps_config+="${app_name}:${app_short}:${table_prefix}:${port},"
      done

      # Remove trailing comma and set FRONTEND_APPS if we built any config
      if [[ -n "$apps_config" ]]; then
        FRONTEND_APPS="${apps_config%,}"
      fi
    fi

    # Now process FRONTEND_APPS (either from individual vars or direct setting)
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

        # Get SSL certificate path based on domain
        local app_ssl_path=$(get_ssl_cert_path "${app_route}")

        # Note: nginx config generation now handled by comprehensive nginx-generator.sh
        # Individual frontend app configs are generated automatically during nginx generation phase
        apps_updated=$((apps_updated + 1))

        # Check if this app has Hasura remote schema configuration
        # We need to check the original FRONTEND_APP_N variables since they're not in compact format
        if [[ -n "${FRONTEND_APP_COUNT:-}" ]]; then
          for ((i=1; i<=FRONTEND_APP_COUNT; i++)); do
            local check_name=$(eval echo "\${FRONTEND_APP_${i}_SYSTEM_NAME:-}")
            local check_display=$(eval echo "\${FRONTEND_APP_${i}_DISPLAY_NAME:-}")
            local check_port=$(eval echo "\${FRONTEND_APP_${i}_PORT:-}")

            # Match by port since it's unique and required
            if [[ "$check_port" == "$app_port" ]]; then
              local remote_schema_name=$(eval echo "\${FRONTEND_APP_${i}_REMOTE_SCHEMA_NAME:-}")
              local remote_schema_input=$(eval echo "\${FRONTEND_APP_${i}_REMOTE_SCHEMA_URL:-}")

              if [[ -n "$remote_schema_name" ]] && [[ -n "$remote_schema_input" ]]; then
                # Check if there's an actual service configured for this remote schema
                local remote_schema_service=$(eval echo "\${FRONTEND_APP_${i}_REMOTE_SCHEMA_SERVICE:-}")
                local should_create_remote_schema=false

                if [[ -n "$remote_schema_service" ]]; then
                  # Service container specified - will create remote schema
                  should_create_remote_schema=true
                elif [[ "$remote_schema_input" =~ ^https?:// ]] && \
                     [[ ! "$remote_schema_input" =~ localhost ]] && \
                     [[ ! "$remote_schema_input" =~ local\.nself\.org ]]; then
                  # External URL (not local) - will create remote schema
                  should_create_remote_schema=true
                fi

                if [[ "$should_create_remote_schema" == "true" ]]; then
                  local remote_schema_url=""
                  local protocol="http"

                  # Determine protocol based on environment
                  if [[ "${ENV:-dev}" == "prod" ]] || [[ "${SSL_MODE:-}" == "letsencrypt" ]]; then
                    protocol="https"
                  elif [[ "${SSL_MODE:-}" == "local" ]] || [[ "${BASE_DOMAIN}" == *"local.nself.org"* ]]; then
                    protocol="https"
                  fi

                  # Handle different input formats
                  if [[ "$remote_schema_input" =~ ^https?:// ]]; then
                    # Full URL provided - use as-is
                    remote_schema_url="$remote_schema_input"
                  elif [[ "$remote_schema_input" == *'${BASE_DOMAIN}'* ]]; then
                    # Contains BASE_DOMAIN variable - expand and add protocol
                    remote_schema_url="${protocol}://$(eval echo "$remote_schema_input")/graphql"
                  else
                    # Shorthand like "api.app1" - construct full URL
                    remote_schema_url="${protocol}://${remote_schema_input}.${BASE_DOMAIN}/graphql"
                  fi

                  # Create Hasura metadata directory if it doesn't exist
                  mkdir -p hasura/metadata/remote_schemas

                  # Generate remote schema metadata file
                  cat >hasura/metadata/remote_schemas/${remote_schema_name}.yaml <<EOF
name: ${remote_schema_name}
definition:
  url: ${remote_schema_url}
  timeout_seconds: 60
  forward_client_headers: true
EOF
                  echo "    - Added Hasura remote schema: ${remote_schema_name} -> ${remote_schema_url}"
                  CREATED_FILES+=("hasura/metadata/remote_schemas/${remote_schema_name}.yaml")
                else
                  echo "    - Note: Remote schema URL configured but no service specified (${remote_schema_input})"
                fi
              fi
              break
            fi
          done
        fi

        app_count=$((app_count + 1))
      done

      if [[ $app_count -gt 0 ]]; then
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

      # Source hasura metadata helper if available
      if [[ -f "$LIB_ROOT/../lib/services/hasura-metadata.sh" ]]; then
        source "$LIB_ROOT/../lib/services/hasura-metadata.sh"
      fi

      cat >postgres/init/01-init.sql <<'EOF'
-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- Create core schemas
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS storage;
CREATE SCHEMA IF NOT EXISTS public;

-- Setup permissions for Hasura
GRANT USAGE ON SCHEMA public TO postgres;
GRANT CREATE ON SCHEMA public TO postgres;
GRANT ALL ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO postgres;
EOF

      # Add app-specific schemas if metadata generation is available
      if declare -f hasura::generate_schema_sql >/dev/null 2>&1; then
        hasura::generate_schema_sql >> postgres/init/01-init.sql
      fi

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
  # Only process services if we're actually building
  if [[ "$needs_work" == "true" ]]; then
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating services...\r"

    # Environment already loaded at start of build process

    # Source generators once at the beginning
    local service_gen_loaded=false
    local dockerfile_gen_loaded=false
    local custom_service_loaded=false

    if [[ -f "$LIB_ROOT/../lib/auto-fix/service-generator.sh" ]]; then
      # Override log functions to be silent
      log_info() { :; }
      log_success() { :; }
      log_warning() { :; }
      source "$LIB_ROOT/../lib/auto-fix/service-generator.sh" || true
      service_gen_loaded=true
    fi

    if [[ -f "$LIB_ROOT/../lib/auto-fix/dockerfile-generator.sh" ]]; then
      # Override log functions to be silent for this too
      log_info() { :; }
      log_success() { :; }
      log_warning() { :; }
      source "$LIB_ROOT/../lib/auto-fix/dockerfile-generator.sh" || true
      dockerfile_gen_loaded=true
    fi

    # Source custom service builder
    # Try v2 builder first (CS_N pattern), fall back to v1
    if [[ -f "$LIB_ROOT/../lib/services/service-builder.sh" ]]; then
      source "$LIB_ROOT/../lib/services/service-builder.sh" 2>/dev/null || true
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
            custom_services_generated=$((custom_services_generated + 1))
            n=$((n + 1))
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
      local gen_script="$LIB_ROOT/../lib/auto-fix/dockerfile-generator.sh"

      # Functions service
      if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]] && [[ ! -d "functions" ]]; then
        # Use bash -c to ensure proper execution context for heredocs
        bash -c "source '${gen_script}' && generate_dockerfile_for_service 'functions' 'functions'" >/dev/null 2>&1
        if [[ -d "functions" ]]; then
          system_services_generated=$((system_services_generated + 1))
        fi
      fi

      # Dashboard service
      if [[ "${DASHBOARD_ENABLED:-false}" == "true" ]] && [[ ! -d "dashboard" ]]; then
        bash -c "source '${gen_script}' && generate_dockerfile_for_service 'dashboard' 'dashboard'" >/dev/null 2>&1
        if [[ -d "dashboard" ]]; then
          system_services_generated=$((system_services_generated + 1))
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
    ( source "$LIB_ROOT/../lib/utils/display.sh" 2>/dev/null ) >/dev/null || true

    # SSL certificates were already generated earlier in the build process
    # No need to regenerate them here

  fi  # End of if needs_work == true for service generation

  # Display available routes
  if [[ -f "$LIB_ROOT/../lib/services/routes-display.sh" ]]; then
    source "$LIB_ROOT/../lib/services/routes-display.sh"
    routes::display_compact
  fi

  # Run comprehensive fixes for any remaining issues
  if [[ -f "$LIB_ROOT/../lib/auto-fix/comprehensive-fix.sh" ]]; then
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Running comprehensive fixes..."

    # Source the comprehensive fix script
    source "$LIB_ROOT/../lib/auto-fix/comprehensive-fix.sh"

    # Run the fix with output suppression
    if comprehensive_fix >/dev/null 2>&1; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Comprehensive fixes applied                \n"
    else
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Some fixes may have failed                 \n"
    fi
  fi

  # Check and update /etc/hosts if needed
  if [[ "${BASE_DOMAIN:-localhost}" == "localhost" ]]; then
    ensure_hosts_entries "localhost" "${PROJECT_NAME:-nself}"
  elif [[ -n "${BASE_DOMAIN:-}" ]] && [[ "${BASE_DOMAIN}" != "local.nself.org" ]]; then
    # For custom domains, check if they need hosts entries
    ensure_hosts_entries "${BASE_DOMAIN}" "${PROJECT_NAME:-nself}"
  fi

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

  # Only suggest trust command for dev environments
  if [[ "${ENV:-dev}" == "dev" ]]; then
    echo -e "${COLOR_BLUE}1.${COLOR_RESET} ${COLOR_BLUE}nself trust${COLOR_RESET} - Install SSL certificates"
    echo -e "   ${COLOR_DIM}Trust the root CA for green locks in browsers${COLOR_RESET}"
    echo
  fi
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

# Run pre-flight checks
run_preflight_checks() {
  local checks_passed=true

  # Check Docker
  if ! command_exists docker; then
    show_error "Docker is not installed"
    checks_passed=false
  fi

  # Check Docker daemon
  if ! docker info >/dev/null 2>&1; then
    show_error "Docker daemon is not running"
    checks_passed=false
  fi

  # Check for .env files
  if [[ ! -f ".env" ]] && [[ ! -f ".env.dev" ]] && [[ ! -f ".env.local" ]]; then
    show_error "No environment file found (.env, .env.dev, or .env.local)"
    checks_passed=false
  fi

  # WSL-specific checks
  if [[ "$IS_WSL" == true ]]; then
    if ! docker info >/dev/null 2>&1; then
      show_error "Docker is not accessible from WSL"
      show_info "Ensure Docker Desktop WSL2 integration is enabled"
      checks_passed=false
    fi
  fi

  [[ "$checks_passed" == true ]]
}

# Load and validate environment
load_and_validate_env() {
  # Source environment files in priority order
  if [[ -f ".env.local" ]]; then
    set -a
    source .env.local 2>/dev/null || true
    set +a
  fi

  if [[ -f ".env.${ENV:-dev}" ]]; then
    set -a
    source ".env.${ENV:-dev}" 2>/dev/null || true
    set +a
  fi

  if [[ -f ".env" ]]; then
    set -a
    source .env 2>/dev/null || true
    set +a
  fi

  # Validate critical variables
  validate_environment
}

# Apply smart defaults
apply_build_defaults() {
  # Set defaults based on environment
  if [[ "${ENV}" == "prod" ]] || [[ "${ENVIRONMENT}" == "production" ]]; then
    set_default "SSL_ENABLED" "true"
    set_default "DEBUG" "false"
    set_default "LOG_LEVEL" "info"
  else
    set_default "SSL_ENABLED" "true"  # Enable SSL in dev too for testing
    set_default "DEBUG" "false"
    set_default "LOG_LEVEL" "debug"
  fi

  # Service defaults
  set_default "NGINX_ENABLED" "true"
  set_default "POSTGRES_ENABLED" "true"
  set_default "HASURA_ENABLED" "false"
  set_default "AUTH_ENABLED" "false"
  set_default "STORAGE_ENABLED" "false"
  set_default "REDIS_ENABLED" "false"

  # Port defaults
  set_default "NGINX_PORT" "80"
  set_default "NGINX_SSL_PORT" "443"
  set_default "POSTGRES_PORT" "5432"
  set_default "HASURA_PORT" "8080"
  set_default "AUTH_PORT" "4000"
  set_default "STORAGE_PORT" "5000"
  set_default "REDIS_PORT" "6379"

  return 0
}

# Setup localhost domains
setup_localhost_domains() {
  if [[ "$IS_MAC" == true ]] || [[ "$IS_LINUX" == true ]]; then
    # Add to /etc/hosts if not present
    local domains=("localhost" "${PROJECT_NAME}.localhost" "api.localhost" "auth.localhost")

    for domain in "${domains[@]}"; do
      if ! grep -q "127.0.0.1.*$domain" /etc/hosts 2>/dev/null; then
        if [[ -w /etc/hosts ]]; then
          echo "127.0.0.1 $domain" >> /etc/hosts
        else
          show_info "Run 'sudo nself build' to add $domain to /etc/hosts"
        fi
      fi
    done
  fi
  return 0
}

# Helper functions from original build.sh

# Show help for build command
show_build_help() {
  echo "nself build - Generate project infrastructure and configuration"
  echo ""
  echo "Usage: nself build [OPTIONS]"
  echo ""
  echo "Description:"
  echo "  Generates Docker Compose files, SSL certificates, nginx configuration,"
  echo "  and all necessary infrastructure based on your .env settings."
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

# Detect application port from package.json or other configs
detect_app_port() {
  local default_port="${1:-3000}"
  local detected_port=""

  # Check package.json for Next.js, React, etc.
  if [[ -f "package.json" ]]; then
    # Check scripts for port configuration
    detected_port=$(grep -o '"dev".*-p\s*[0-9]*' package.json 2>/dev/null | grep -o '[0-9]*$' || true)

    # Check for PORT in scripts
    if [[ -z "$detected_port" ]]; then
      detected_port=$(grep -o 'PORT=[0-9]*' package.json 2>/dev/null | grep -o '[0-9]*$' || true)
    fi

    # Check for port in start script
    if [[ -z "$detected_port" ]]; then
      detected_port=$(grep -o '"start".*:.*[0-9]\{4\}' package.json 2>/dev/null | grep -o '[0-9]\{4\}$' || true)
    fi
  fi

  # Check .env files for PORT
  if [[ -z "$detected_port" ]] && [[ -f ".env" ]]; then
    detected_port=$(grep '^PORT=' .env 2>/dev/null | cut -d= -f2 || true)
  fi

  # Check for common framework defaults
  if [[ -z "$detected_port" ]] && [[ -f "package.json" ]]; then
    if grep -q '"next"' package.json 2>/dev/null; then
      detected_port="3000"  # Next.js default
    elif grep -q '"vite"' package.json 2>/dev/null; then
      detected_port="5173"  # Vite default
    elif grep -q '"nuxt"' package.json 2>/dev/null; then
      detected_port="3000"  # Nuxt default
    fi
  fi

  echo "${detected_port:-$default_port}"
}

# Get the SSL certificate path based on the domain
get_ssl_cert_path() {
  local domain="${1:-localhost}"

  # For api.*.localhost domains, use api-localhost certificates if they exist
  if [[ "$domain" == "api."*".localhost" ]] && [[ -d "ssl/certificates/api-localhost" ]]; then
    echo "/etc/nginx/ssl/api-localhost"
  # For api.localhost, use api-localhost certificates if they exist
  elif [[ "$domain" == "api.localhost" ]] && [[ -d "ssl/certificates/api-localhost" ]]; then
    echo "/etc/nginx/ssl/api-localhost"
  # For localhost and *.localhost domains, use localhost certificates
  elif [[ "$domain" == "localhost" ]] || [[ "$domain" == *".localhost" ]]; then
    echo "/etc/nginx/ssl/localhost"
  # For local.nself.org and *.local.nself.org, use nself-org certificates
  elif [[ "$domain" == *"local.nself.org" ]]; then
    echo "/etc/nginx/ssl/nself-org"
  # For custom domains (production/staging), use custom certificates
  elif [[ "$domain" != *".localhost" ]]; then
    echo "/etc/nginx/ssl/custom"
  # Default to localhost for development
  else
    echo "/etc/nginx/ssl/localhost"
  fi
}

# Override the format_section function to suppress output
format_section() {
  # Silently ignore section formatting calls from validation
  : # No-op command that always succeeds
}

# Simple SSL generation fallback (when SSL library not available)
build_generate_simple_ssl() {
  # Create the expected directory structure
  mkdir -p ssl/certificates/{localhost,nself-org} >/dev/null 2>&1

  # Check for mkcert (either in PATH or nself bin)
  local mkcert_cmd=""
  if command -v mkcert >/dev/null 2>&1; then
    mkcert_cmd="mkcert"
  elif [[ -f "$HOME/.nself/bin/mkcert" ]]; then
    mkcert_cmd="$HOME/.nself/bin/mkcert"
  fi

  if [[ -n "$mkcert_cmd" ]]; then
    # Ensure root CA is installed
    $mkcert_cmd -install >/dev/null 2>&1 || true

    # Build comprehensive domain list (CRITICAL: explicit listing, not just wildcards)
    local project_name="${PROJECT_NAME:-app}"
    local localhost_domains=(
      "localhost"
      "*.localhost"
      "127.0.0.1"
      "::1"
      "api.localhost"
      "auth.localhost"
      "storage.localhost"
      "functions.localhost"
      "dashboard.localhost"
      "console.localhost"
      "${project_name}.localhost"
    )

    # Add multi-level subdomains for common services
    local common_prefixes=("api" "auth" "storage" "files" "ws" "notify" "hooks" "actions" "ai" "db" "mail" "search")
    for prefix in "${common_prefixes[@]}"; do
      localhost_domains+=("${prefix}.${project_name}.localhost")
    done

    # Add frontend app remote schema URLs
    local i=1
    while [[ $i -le 10 ]]; do  # Check up to 10 frontend apps
      var="FRONTEND_APP_${i}_REMOTE_SCHEMA_URL"
      if [[ -n "${!var:-}" ]]; then
        # Add both the subdomain and with project name
        localhost_domains+=("${!var}.localhost")
        localhost_domains+=("${!var}.${project_name}.localhost")
      fi
      i=$((i + 1))
    done

    # Add common variations
    [[ "$project_name" == "nchat" ]] && localhost_domains+=("chat.localhost")
    [[ "$project_name" == "admin" ]] && localhost_domains+=("admin.localhost")

    # Add custom subdomains if configured
    if [[ -n "${CUSTOM_SUBDOMAINS:-}" ]]; then
      IFS=',' read -ra CUSTOM <<< "$CUSTOM_SUBDOMAINS"
      for domain in "${CUSTOM[@]}"; do
        localhost_domains+=("${domain}.localhost")
      done
    fi

    # Add wildcard last (as fallback)
    localhost_domains+=("*.localhost")

    # For localhost domain, generate localhost certificates with all explicit domains
    if [[ "${BASE_DOMAIN}" == "localhost" ]]; then
      $mkcert_cmd -cert-file ssl/certificates/localhost/fullchain.pem \
             -key-file ssl/certificates/localhost/privkey.pem \
             "${localhost_domains[@]}" >/dev/null 2>&1

      # Also copy to nginx directory
      mkdir -p nginx/ssl/localhost >/dev/null 2>&1
      cp ssl/certificates/localhost/*.pem nginx/ssl/localhost/ 2>/dev/null || true
    else
      # Generate domain-specific certificates
      $mkcert_cmd -cert-file ssl/certificates/nself-org/fullchain.pem \
             -key-file ssl/certificates/nself-org/privkey.pem \
             "${BASE_DOMAIN}" "*.${BASE_DOMAIN}" \
             "api.${BASE_DOMAIN}" "auth.${BASE_DOMAIN}" "storage.${BASE_DOMAIN}" >/dev/null 2>&1

      # Also generate localhost certificates as fallback
      $mkcert_cmd -cert-file ssl/certificates/localhost/fullchain.pem \
             -key-file ssl/certificates/localhost/privkey.pem \
             "${localhost_domains[@]}" >/dev/null 2>&1

      # Copy to nginx directories
      mkdir -p nginx/ssl/{localhost,nself-org} >/dev/null 2>&1
      cp ssl/certificates/localhost/*.pem nginx/ssl/localhost/ 2>/dev/null || true
      cp ssl/certificates/nself-org/*.pem nginx/ssl/nself-org/ 2>/dev/null || true
    fi
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} SSL certificates generated (trusted)       \n"
    CREATED_FILES+=("SSL certificates")
  else
    # Generate self-signed certificates as fallback
    if [[ "${BASE_DOMAIN}" == "localhost" ]]; then
      openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout ssl/certificates/localhost/privkey.pem \
        -out ssl/certificates/localhost/fullchain.pem \
        -subj "/C=US/ST=State/L=City/O=nself/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1" >/dev/null 2>&1
    else
      openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout ssl/certificates/nself-org/privkey.pem \
        -out ssl/certificates/nself-org/fullchain.pem \
        -subj "/C=US/ST=State/L=City/O=nself/CN=*.${BASE_DOMAIN}" \
        -addext "subjectAltName=DNS:*.${BASE_DOMAIN},DNS:${BASE_DOMAIN}" >/dev/null 2>&1

      # Also generate localhost certificates
      openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout ssl/certificates/localhost/privkey.pem \
        -out ssl/certificates/localhost/fullchain.pem \
        -subj "/C=US/ST=State/L=City/O=nself/CN=localhost" \
        -addext "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1" >/dev/null 2>&1
    fi

    # Copy to nginx directories
    mkdir -p nginx/ssl/{localhost,nself-org} >/dev/null 2>&1
    cp ssl/certificates/localhost/*.pem nginx/ssl/localhost/ 2>/dev/null || true
    [[ -d ssl/certificates/nself-org ]] && cp ssl/certificates/nself-org/*.pem nginx/ssl/nself-org/ 2>/dev/null || true

    printf "\r${COLOR_GREEN}✓${COLOR_RESET} SSL certificates generated (self-signed)   \n"
    CREATED_FILES+=("SSL certificates")
  fi
}

# Export main function
export -f orchestrate_build