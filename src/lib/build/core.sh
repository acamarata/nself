#!/usr/bin/env bash
# core-refactored.sh - Refactored core build orchestration logic using modules
# POSIX-compliant, no Bash 4+ features

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
fi

if [[ -f "$LIB_ROOT/auto-fix/auto-fixer-v2.sh" ]]; then
  source "$LIB_ROOT/auto-fix/auto-fixer-v2.sh"
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

# Source the new core modules
MODULE_DIR="$CORE_SCRIPT_DIR/core-modules"
if [[ -d "$MODULE_DIR" ]]; then
  for module in "$MODULE_DIR"/*.sh; do
    if [[ -f "$module" ]]; then
      source "$module"
    fi
  done
fi

# Port detection function for compatibility with tests
detect_app_port() {
  local start_port="${1:-3000}"
  local port="$start_port"

  # Simple port availability check compatible with Bash 3.2
  while true; do
    if ! lsof -i:$port >/dev/null 2>&1; then
      echo "$port"
      return 0
    fi
    port=$((port + 1))
    if [[ $port -gt 65535 ]]; then
      echo "$start_port"
      return 1
    fi
  done
}

# Initialize build environment - compatibility function
init_build_environment() {
  # Set build directory
  export BUILD_DIR="${BUILD_DIR:-$(pwd)}"

  # Ensure required directories exist
  mkdir -p "$BUILD_DIR"

  # Load environment if exists
  if [[ -f "$BUILD_DIR/.env" ]]; then
    export_env_from_file "$BUILD_DIR/.env"
  fi

  # Set defaults for critical variables
  : ${PROJECT_NAME:="myproject"}
  : ${DOCKER_NETWORK:="${PROJECT_NAME}_network"}
  : ${BASE_DOMAIN:="localhost"}

  return 0
}

# Convert frontend app definitions from compact to expanded format
convert_frontend_apps_to_expanded() {
  local port_base=3000
  local app_counter=0

  # Process each frontend framework type
  for framework in NEXTJS REACT VUE ANGULAR SVELTE; do
    local apps_var="${framework}_APPS"
    # Use eval for Bash 3.2 compatibility
    eval "local apps_value=\${$apps_var:-}"

    if [[ -n "$apps_value" ]]; then
      # Split apps by comma
      local IFS=','
      for app in $apps_value; do
        # Trim whitespace
        app="$(echo "$app" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

        if [[ -n "$app" ]]; then
          app_counter=$((app_counter + 1))

          # Set frontend app variables
          export "FRONTEND_APP_${app_counter}_NAME=$app"
          export "FRONTEND_APP_${app_counter}_DIR=frontend/$app"
          export "FRONTEND_APP_${app_counter}_PORT=$((port_base + app_counter - 1))"
          export "FRONTEND_APP_${app_counter}_DISPLAY_NAME=$app"

          # Set framework based on the variable
          case "$framework" in
            NEXTJS) export "FRONTEND_APP_${app_counter}_FRAMEWORK=nextjs" ;;
            REACT) export "FRONTEND_APP_${app_counter}_FRAMEWORK=react" ;;
            VUE) export "FRONTEND_APP_${app_counter}_FRAMEWORK=vue" ;;
            ANGULAR) export "FRONTEND_APP_${app_counter}_FRAMEWORK=angular" ;;
            SVELTE) export "FRONTEND_APP_${app_counter}_FRAMEWORK=svelte" ;;
          esac
        fi
      done
    fi
  done

  # Set the total count
  export FRONTEND_APP_COUNT=$app_counter
}

# Simple SSL generation function for build
build_generate_simple_ssl() {
  local base_domain="${BASE_DOMAIN:-localhost}"
  local ssl_created=false

  # Create SSL directories
  mkdir -p ssl/certificates/localhost 2>/dev/null
  mkdir -p ssl/certificates/nself-org 2>/dev/null
  mkdir -p nginx/ssl/localhost 2>/dev/null
  mkdir -p nginx/ssl/nself-org 2>/dev/null

  # Generate localhost certificates
  if [[ ! -f "ssl/certificates/localhost/fullchain.pem" ]] || [[ ! -f "ssl/certificates/localhost/privkey.pem" ]]; then
    if command -v openssl >/dev/null 2>&1; then
      # Generate private key
      openssl genrsa -out ssl/certificates/localhost/privkey.pem 2048 2>/dev/null

      # Generate certificate
      openssl req -new -x509 \
        -key ssl/certificates/localhost/privkey.pem \
        -out ssl/certificates/localhost/fullchain.pem \
        -days 365 \
        -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost" 2>/dev/null

      ssl_created=true
    fi
  fi

  # Generate nself.org certificates if needed
  if [[ "$base_domain" != "localhost" ]]; then
    if [[ ! -f "ssl/certificates/nself-org/fullchain.pem" ]] || [[ ! -f "ssl/certificates/nself-org/privkey.pem" ]]; then
      if command -v openssl >/dev/null 2>&1; then
        # Generate private key
        openssl genrsa -out ssl/certificates/nself-org/privkey.pem 2048 2>/dev/null

        # Generate certificate
        openssl req -new -x509 \
          -key ssl/certificates/nself-org/privkey.pem \
          -out ssl/certificates/nself-org/fullchain.pem \
          -days 365 \
          -subj "/C=US/ST=State/L=City/O=Organization/CN=*.nself.org" 2>/dev/null

        ssl_created=true
      fi
    fi
  fi

  # Copy to nginx directory
  cp -f ssl/certificates/localhost/* nginx/ssl/localhost/ 2>/dev/null || true
  if [[ "$base_domain" != "localhost" ]]; then
    cp -f ssl/certificates/nself-org/* nginx/ssl/nself-org/ 2>/dev/null || true
  fi

  if [[ "$ssl_created" == "true" ]]; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} SSL certificates generated                    \n"
    CREATED_FILES+=("SSL certificates")
  else
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} SSL certificates exist                        \n"
  fi
}

# New streamlined orchestrate_build function using modules
orchestrate_build() {
  local project_name="${1:-$(basename "$PWD")}"
  local env="${2:-dev}"
  local force_rebuild="${3:-false}"
  local verbose="${VERBOSE:-false}"

  # Initialize tracking arrays
  CREATED_FILES=()
  UPDATED_FILES=()
  SKIPPED_FILES=()

  # Load environment FIRST to get PROJECT_NAME from config
  local env_file=".env"
  if [[ -f ".env.local" ]]; then
    env_file=".env.local"
  elif [[ -f ".env.${env}" ]]; then
    env_file=".env.${env}"
  fi

  if [[ -f "$env_file" ]]; then
    set -a
    source "$env_file" 2>/dev/null || true
    set +a
  fi

  # Now export, using env file value if available, otherwise use parameter/default
  export PROJECT_NAME="${PROJECT_NAME:-$project_name}"
  export ENV="$env"
  export VERBOSE="$verbose"
  export AUTO_FIX="${AUTO_FIX:-true}"

  # Apply database auto-configuration
  if [[ -f "$LIB_ROOT/database/auto-config.sh" ]]; then
    source "$LIB_ROOT/database/auto-config.sh" 2>/dev/null || true
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Configuring database for optimal performance..."
    if command -v get_system_resources &>/dev/null && command -v apply_smart_defaults &>/dev/null; then
      get_system_resources >/dev/null 2>&1 || true
      apply_smart_defaults >/dev/null 2>&1 || true
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Database configuration optimized                    \n"
    else
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Database auto-config not available                 \n"
    fi
  fi

  # Run validation
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Validating configuration..."

  if [[ -f "$env_file" ]]; then
    # Actually validate and fix the environment
    if validate_environment; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Configuration validated                    \n"
    else
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Configuration validated with fixes          \n"
    fi
    # Reload env file to pick up fixes (recheck which file exists)
    if [[ -f ".env.local" ]]; then
      set -a
      source ".env.local" 2>/dev/null || true
      set +a
    elif [[ -f ".env.${env}" ]]; then
      set -a
      source ".env.${env}" 2>/dev/null || true
      set +a
    elif [[ -f ".env" ]]; then
      set -a
      source ".env" 2>/dev/null || true
      set +a
    fi
  else
    printf "\r${COLOR_YELLOW}✱${COLOR_RESET} No environment file found                  \n"
  fi

  # Convert frontend apps if needed
  convert_frontend_apps_to_expanded

  # Check if this is an existing project
  local is_existing_project=false
  if [[ -f "docker-compose.yml" ]] || [[ -d "nginx" ]] || [[ -f "postgres/init/00-init.sql" ]]; then
    is_existing_project=true
  fi

  # Use modular orchestration if available
  if command -v orchestrate_modular_build >/dev/null 2>&1; then
    # Show header like init command
    if command -v show_command_header >/dev/null 2>&1; then
      show_command_header "nself build" "Generate project infrastructure and configuration"
    fi
    echo

    # Load core modules
    load_core_modules 2>/dev/null || true

    # Check what needs to be done
    if check_build_requirements "$force_rebuild" "$env_file"; then
      # Execute build steps with proper output

      # Create directories if needed
      if [[ "$NEEDS_DIRECTORIES" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Creating directory structure..."
        if setup_project_directories; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Directory structure ready                    \n"
          CREATED_FILES+=("directories")
        else
          printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to create directories                \n"
        fi
      fi

      # Generate SSL certificates if needed
      if [[ "$NEEDS_SSL" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating SSL certificates..."
        build_generate_simple_ssl
      fi

      # Generate nginx configuration if needed
      if [[ "$NEEDS_NGINX" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating nginx configuration..."
        if setup_nginx; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Nginx configuration generated                \n"
          UPDATED_FILES+=("nginx/nginx.conf")
        else
          printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Nginx configuration skipped                 \n"
        fi
      fi

      # Generate database initialization if needed
      if [[ "$NEEDS_DATABASE" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating database initialization..."
        if generate_postgres_init; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Database initialization generated            \n"
          CREATED_FILES+=("postgres/init/*.sql")
        else
          printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Database initialization skipped             \n"
        fi
      fi

      # Setup monitoring configs if monitoring is enabled
      if [[ "${GRAFANA_ENABLED:-false}" == "true" ]] || [[ "${LOKI_ENABLED:-false}" == "true" ]] || [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Setting up monitoring configs..."
        if command -v setup_monitoring_configs >/dev/null 2>&1; then
          # Load env vars for the function
          set -a
          source "$env_file" 2>/dev/null || true
          set +a
          setup_monitoring_configs >/dev/null 2>&1
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Monitoring configs ready                     \n"
          CREATED_FILES+=("monitoring/*")
        else
          printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Monitoring configs skipped                   \n"
        fi
      fi

      # Generate docker-compose.yml if needed
      if [[ "$NEEDS_COMPOSE" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating docker-compose.yml..."
        if generate_docker_compose; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} docker-compose.yml generated                \n"
          if [[ "$is_existing_project" == "true" ]]; then
            UPDATED_FILES+=("docker-compose.yml")
          else
            CREATED_FILES+=("docker-compose.yml")
          fi
        else
          printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to generate docker-compose.yml       \n"
        fi
      fi

      # Run post-build tasks
      run_post_build_tasks 2>/dev/null || true
    else
      echo -e "${COLOR_GREEN}✓${COLOR_RESET} Everything is up to date"
    fi
  else
    # Fallback to original orchestrate_build logic
    echo "Warning: Modular build not available, using legacy build" >&2
    return 1
  fi

  # Show next steps directly
  echo
  echo -e "${COLOR_CYAN}➞ Next Steps${COLOR_RESET}"
  echo
  echo "  1. Start the stack: nself start"
  echo "  2. View status: nself status"
  echo "  3. View logs: nself logs"
  echo
  echo "For more help, use: nself help or nself help build"
  echo

  return 0
}

# Run pre-flight checks
run_preflight_checks() {
  local checks_passed=true

  # Check Docker
  if ! command -v docker >/dev/null 2>&1; then
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

  [[ "$checks_passed" == true ]]
}

# Apply smart defaults
apply_build_defaults() {
  # Set defaults based on environment
  if [[ "${ENV}" == "prod" ]] || [[ "${ENVIRONMENT}" == "production" ]]; then
    set_default "SSL_ENABLED" "true"
    set_default "DEBUG" "false"
    set_default "LOG_LEVEL" "info"
  else
    set_default "SSL_ENABLED" "true"
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

# Main build function
run_build() {
  local project_name="${1:-$(basename "$PWD")}"
  local env="${2:-dev}"
  local force="${3:-false}"
  local verbose="${4:-false}"

  # Run orchestration
  orchestrate_build "$project_name" "$env" "$force" "$verbose"
}

# Export functions
export -f convert_frontend_apps_to_expanded
export -f build_generate_simple_ssl
export -f orchestrate_build
export -f run_preflight_checks
export -f apply_build_defaults
export -f run_build