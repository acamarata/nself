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

# Source comprehensive config validator
if [[ -f "$CORE_SCRIPT_DIR/config-validator.sh" ]]; then
  source "$CORE_SCRIPT_DIR/config-validator.sh"
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

          # Set frontend app variables (no DIR - frontends are external)
          export "FRONTEND_APP_${app_counter}_NAME=$app"
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

  # IMPORTANT: Build loads env to detect WHAT to provision
  # But outputs use runtime vars for HOW they're configured
  # This makes the build output work with any environment

  # Load environment files for service detection
  # This tells us WHAT services to build, not HOW to configure them
  load_env_for_detection() {
    local env="${ENV:-dev}"

    # Load files in cascade order for proper detection
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

  # Load env for detection
  load_env_for_detection

  # Validate and sanitize environment variables (including PROJECT_NAME)
  if command -v validate_environment >/dev/null 2>&1; then
    validate_environment || true
  fi

  # Detect environment from loaded vars
  if command -v detect_environment >/dev/null 2>&1; then
    env="$(detect_environment)"
  else
    env="${ENV:-$env}"
  fi

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
      # Redis exporter only if Redis is also enabled
      if [[ "${REDIS_ENABLED:-false}" == "true" ]]; then
        export REDIS_EXPORTER_ENABLED="true"
      fi
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

  # Run service detection
  detect_all_services

  # Export with smart defaults - these can be overridden by env vars if explicitly set
  export PROJECT_NAME="${PROJECT_NAME:-$project_name}"
  export ENV="$env"
  export VERBOSE="$verbose"
  export AUTO_FIX="${AUTO_FIX:-true}"

  # Set smart defaults for all common ports and domains
  export BASE_DOMAIN="${BASE_DOMAIN:-localhost}"
  export POSTGRES_PORT="${POSTGRES_PORT:-5432}"
  export HASURA_PORT="${HASURA_PORT:-8080}"
  export AUTH_PORT="${AUTH_PORT:-4000}"
  export STORAGE_PORT="${STORAGE_PORT:-5000}"
  export REDIS_PORT="${REDIS_PORT:-6379}"
  export MINIO_PORT="${MINIO_PORT:-9000}"
  export MINIO_CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"

  # Show header FIRST (before any validation output)
  if command -v show_command_header >/dev/null 2>&1; then
    show_command_header "nself build" "Generate project infrastructure and configuration"
  fi

  # Apply database auto-configuration
  if [[ -f "$LIB_ROOT/database/auto-config.sh" ]]; then
    source "$LIB_ROOT/database/auto-config.sh" 2>/dev/null || true
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Configuring database for optimal performance..."
    if command -v get_system_resources >/dev/null 2>&1 && command -v apply_smart_defaults >/dev/null 2>&1; then
      get_system_resources >/dev/null 2>&1 || true
      apply_smart_defaults >/dev/null 2>&1 || true
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Database configuration optimized                    \n"
    else
      printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Database auto-config not available                 \n"
    fi
  fi

  # Run validation using smart defaults only
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Validating build requirements..."

  # Check if at least one env file exists for reference
  local has_config=false
  local env_file=".env"  # Default to .env for compatibility checks
  for file in .env .env.dev .env.local .env.staging .env.prod; do
    if [[ -f "$file" ]]; then
      has_config=true
      env_file="$file"  # Use first found for timestamp checks
      break
    fi
  done

  if [[ "$has_config" == "true" ]]; then
    # Run validation with smart defaults
    if command -v validate_build_config >/dev/null 2>&1; then
      # Validation will use smart defaults, not loaded env
      local validation_output=$(validate_build_config 2>&1)
      local validation_result=$?

      if [[ $validation_result -eq 0 ]]; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Build requirements validated                \n"
      else
        printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Build proceeding with defaults              \n"
      fi
    else
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Build requirements checked                  \n"
    fi
  else
    printf "\r${COLOR_YELLOW}✱${COLOR_RESET} No environment file found                  \n"
  fi

  # Convert frontend apps if needed
  convert_frontend_apps_to_expanded

  # Check for route conflicts (skip if urls script not available or errors)
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Checking for route conflicts..."

  # Use nself urls command to check conflicts
  local conflict_check_output
  local urls_script="$LIB_ROOT/../cli/urls.sh"
  if [[ -f "$urls_script" ]]; then
    # Try to check conflicts, but don't fail build if script has issues
    conflict_check_output=$(cd "$PWD" && "$urls_script" --check-conflicts 2>&1 || true)
    local conflict_result=$?

    if [[ $conflict_result -eq 0 ]] || [[ -z "$conflict_check_output" ]]; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Route validation complete                      \n"
    elif [[ "$conflict_check_output" == *"conflict"* ]]; then
      printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} Potential route conflicts detected             \n"
      echo "$conflict_check_output" | head -5

      # If AUTO_FIX is enabled, try to fix conflicts
      if [[ "${AUTO_FIX:-true}" == "true" ]]; then
        echo
        echo "${COLOR_YELLOW}Attempting to auto-fix route conflicts...${COLOR_RESET}"

        # Extract suggested fixes and apply them
        while IFS= read -r line; do
          if [[ "$line" =~ CS_([0-9]+)_ROUTE=([a-z-]+) ]]; then
            local cs_num="${BASH_REMATCH[1]}"
            local new_route="${BASH_REMATCH[2]}"

            # Add the route fix to the env file
            echo "CS_${cs_num}_ROUTE=${new_route}" >> "$env_file"
            echo "  Applied: CS_${cs_num}_ROUTE=${new_route}"
          fi
        done <<< "$conflict_check_output"

        # Reload environment with fixes
        set -a
        source "$env_file" 2>/dev/null || true
        set +a

        echo "${COLOR_GREEN}✓ Route conflicts auto-fixed${COLOR_RESET}"
        echo
      else
        echo "${COLOR_YELLOW}Please fix the conflicts manually before continuing.${COLOR_RESET}"
        return 1
      fi
    fi
  else
    printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} Route conflict detection not available         \n"
  fi

  # Check if this is an existing project
  local is_existing_project=false
  if [[ -f "docker-compose.yml" ]] || [[ -d "nginx" ]] || [[ -f "postgres/init/00-init.sql" ]]; then
    is_existing_project=true
  fi

  # Track what will be built/updated
  local BUILD_ACTIONS=()
  local SKIP_ACTIONS=()

  # Debug output for troubleshooting (only if --debug flag explicitly passed to build command)
  # Note: Uses BUILD_DEBUG (set by --debug flag) not DEBUG (application debug flag)
  if [[ "${BUILD_DEBUG:-false}" == "true" ]]; then
    echo "Debug: Checking for orchestrate_modular_build function..." >&2
    if command -v orchestrate_modular_build >/dev/null 2>&1; then
      echo "Debug: Found orchestrate_modular_build" >&2
    else
      echo "Debug: orchestrate_modular_build not found, will use fallback" >&2
    fi
  fi

  # Use modular orchestration if available
  if command -v orchestrate_modular_build >/dev/null 2>&1; then
    # Load core modules
    if [[ "${BUILD_DEBUG:-false}" == "true" ]]; then
      echo "Debug: Loading core modules..." >&2
    fi
    load_core_modules 2>/dev/null || true

    # Check what needs to be done
    # Special case: If this is a completely fresh project, force build everything
    local needs_initial_build=false
    if [[ ! -f "docker-compose.yml" ]] && [[ ! -d "nginx" ]] && [[ ! -d "postgres" ]]; then
      needs_initial_build=true
      if [[ "${BUILD_DEBUG:-false}" == "true" ]]; then
        echo "Debug: Fresh project detected, forcing initial build" >&2
      fi
      # Force all flags to true for initial build
      export NEEDS_DIRECTORIES=true
      export NEEDS_SSL=true
      export NEEDS_NGINX=true
      export NEEDS_DATABASE=true
      export NEEDS_COMPOSE=true
    fi

    if check_build_requirements "$force_rebuild" "$env_file" || [[ "$needs_initial_build" == "true" ]]; then
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

        # Create nginx directories
        mkdir -p nginx/{sites,conf.d,includes,routes} 2>/dev/null || true

        # Use nginx-generator if available, otherwise fallback to setup_nginx
        if command -v generate_nginx_config >/dev/null 2>&1; then
          generate_nginx_config "$force_rebuild"
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Nginx configuration generated                \n"
          UPDATED_FILES+=("nginx/nginx.conf")
          UPDATED_FILES+=("nginx/sites/*.conf")
        elif command -v setup_nginx >/dev/null 2>&1; then
          setup_nginx
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

      # Generate custom services from templates
      if [[ -n "${CUSTOM_SERVICES:-}" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating custom services..."
        if generate_custom_services; then
          printf "\r${COLOR_GREEN}✓${COLOR_RESET} Custom services generated                    \n"
          CREATED_FILES+=("services/*")
        else
          printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Custom services generation skipped          \n"
        fi
      fi

      # Generate fallback services for auth and functions (for demo/problematic scenarios)
      if [[ "${ENV:-}" == "demo" ]] || [[ "${DEMO_CONTENT:-false}" == "true" ]] || [[ "${GENERATE_FALLBACKS:-false}" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Generating fallback services..."

        # Source the fallback services module
        if [[ -f "$CORE_SCRIPT_DIR/fallback-services.sh" ]]; then
          source "$CORE_SCRIPT_DIR/fallback-services.sh"
          if command -v generate_fallback_services >/dev/null 2>&1; then
            if generate_fallback_services "$PWD"; then
              printf "\r${COLOR_GREEN}✓${COLOR_RESET} Fallback services generated                  \n"
              CREATED_FILES+=("fallback-services/*")
            else
              printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Fallback services generation failed         \n"
            fi
          fi
        else
          printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Fallback services module not found          \n"
        fi
      fi

      # Setup monitoring configs if monitoring is enabled
      if [[ "${GRAFANA_ENABLED:-false}" == "true" ]] || [[ "${LOKI_ENABLED:-false}" == "true" ]] || [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Setting up monitoring configs..."

        local monitoring_result=0
        if command -v setup_monitoring_configs >/dev/null 2>&1; then
          # Load env vars for the function
          set -a
          source "$env_file" 2>/dev/null || true
          set +a

          # Run with timeout to prevent hanging
          if timeout 5 bash -c "$(declare -f setup_monitoring_configs); setup_monitoring_configs" >/dev/null 2>&1; then
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} Monitoring configs ready                     \n"
            CREATED_FILES+=("monitoring/*")
          else
            printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Monitoring setup incomplete                  \n"
          fi
        else
          printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Monitoring module not available             \n"
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

      # Skip runtime variables documentation generation
      # Documentation should be in the wiki/docs, not generated in project

      # Run post-build tasks
      run_post_build_tasks 2>/dev/null || true
    else
      echo -e "${COLOR_GREEN}✓${COLOR_RESET} Everything is up to date"
    fi
  else
    # Fallback to direct orchestrate_build if modular version not available
    # This can happen if modules weren't sourced properly
    if [[ "${BUILD_DEBUG:-false}" == "true" ]]; then
      echo "Info: Using fallback build orchestration" >&2
    fi

    # Check if orchestrate_build function exists
    if command -v orchestrate_build >/dev/null 2>&1; then
      if [[ "${BUILD_DEBUG:-false}" == "true" ]]; then
        echo "Debug: Calling orchestrate_build with project_name=$project_name env=$env" >&2
      fi
      orchestrate_build "$project_name" "$env" "$force_rebuild" "$verbose"
      return $?
    else
      # Critical error - no build orchestration available
      echo "Error: Build orchestration functions not found!" >&2
      echo "This may indicate an installation problem." >&2
      echo "" >&2
      echo "Try running with debug mode for more information:" >&2
      echo "  nself build --debug" >&2
      echo "" >&2
      echo "Or reinstall nself:" >&2
      echo "  curl -sSL https://raw.githubusercontent.com/nself-project/nself/main/install.sh | bash" >&2
      return 1
    fi
  fi

  # Show build summary (dynamic and concise)
  show_build_summary() {
    echo ""

    # Determine build context
    local is_first_build=false
    local has_changes=false
    local changes_made=()

    # Check if this is first build by looking for key infrastructure files
    if [[ ! -f "docker-compose.yml" ]] || [[ ! -d "nginx" ]] || [[ ! -d "postgres" ]]; then
      is_first_build=true
    elif [[ -d ".volumes/backups/compose" ]]; then
      # Has backups, so this is an update
      is_first_build=false

      # Detect what changed (these would be set by various build modules)
      if [[ "${DOCKER_COMPOSE_CHANGED:-false}" == "true" ]]; then
        has_changes=true
        changes_made+=("Docker Compose")
      fi
      if [[ "${NGINX_CONFIG_CHANGED:-false}" == "true" ]]; then
        has_changes=true
        changes_made+=("Nginx routes")
      fi
      if [[ "${SSL_REGENERATED:-false}" == "true" ]]; then
        has_changes=true
        changes_made+=("SSL certificates")
      fi
      if [[ "${CUSTOM_SERVICES_GENERATED:-false}" == "true" ]]; then
        has_changes=true
        changes_made+=("Custom services")
      fi
    fi

    # Count services
    local required_count=4  # Always 4 required services
    local optional_count=0
    local monitoring_count=0
    local custom_count="${CUSTOM_SERVICE_COUNT:-0}"
    local frontend_count="${FRONTEND_APP_COUNT:-0}"

    # Count optional services
    [[ "${REDIS_ENABLED:-false}" == "true" ]] && optional_count=$((optional_count + 1))
    [[ "${MINIO_ENABLED:-false}" == "true" ]] && optional_count=$((optional_count + 1))
    [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]] && optional_count=$((optional_count + 1))
    [[ "${MAILPIT_ENABLED:-true}" == "true" ]] && optional_count=$((optional_count + 1))
    [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]] && optional_count=$((optional_count + 1))
    [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]] && optional_count=$((optional_count + 1))
    [[ "${MLFLOW_ENABLED:-false}" == "true" ]] && optional_count=$((optional_count + 1))

    # Count monitoring services
    if [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
      monitoring_count=10  # Full monitoring bundle
    fi

    local total_containers=$((required_count + optional_count + monitoring_count + custom_count))

    # Mode/Build status FIRST
    if [[ "$is_first_build" == "true" ]]; then
      printf "✓ Mode: First time build, generated from scratch\n"
    elif [[ "${force_rebuild:-false}" == "true" ]]; then
      printf "✓ Mode: Force rebuild - all configurations regenerated\n"
    elif [[ "$has_changes" == "true" ]] && [[ ${#changes_made[@]} -gt 0 ]]; then
      printf "✓ Mode: Changes detected - "
      # List changes inline
      local first=true
      for change in "${changes_made[@]}"; do
        if [[ "$first" == "true" ]]; then
          printf "%s" "$change"
          first=false
        else
          printf ", %s" "$change"
        fi
      done
      printf " updated\n"
    else
      printf "✓ Mode: No changes - everything up to date\n"
    fi

    # Project info with BD: for base domain
    printf "✓ Project: \033[0;34m%s\033[0m (%s) / BD: \033[0;33m%s\033[0m\n" "${PROJECT_NAME}" "${ENV}" "${BASE_DOMAIN}"

    # Services with total count in blue
    printf "✓ Services (\033[0;34m%d\033[0m): %d core, %d optional, %d monitoring, %d custom\n" \
      "$total_containers" "$required_count" "$optional_count" "$monitoring_count" "$custom_count"

    # Show file stats if changes were made
    if [[ ${#CREATED_FILES[@]} -gt 0 ]] || [[ ${#UPDATED_FILES[@]} -gt 0 ]]; then
      local file_summary="✓ Files: "
      [[ ${#CREATED_FILES[@]} -gt 0 ]] && file_summary+="${#CREATED_FILES[@]} created"
      [[ ${#CREATED_FILES[@]} -gt 0 ]] && [[ ${#UPDATED_FILES[@]} -gt 0 ]] && file_summary+=", "
      [[ ${#UPDATED_FILES[@]} -gt 0 ]] && file_summary+="${#UPDATED_FILES[@]} updated"
      echo "$file_summary"
    fi

    echo ""
  }

  # Show build summary
  show_build_summary

  # Show next steps with improved formatting
  echo ""
  echo "Next steps:"
  echo ""

  # Define dim color for descriptions
  local COLOR_DIM=""
  if [[ -t 1 ]]; then
    COLOR_DIM='\033[2m'
  fi

  echo -e "${COLOR_BLUE:-}1.${COLOR_RESET:-} nself start - Launch your services"
  echo -e "   ${COLOR_DIM}Starts all configured Docker containers${COLOR_RESET:-}"
  echo ""
  echo -e "${COLOR_BLUE:-}2.${COLOR_RESET:-} nself status - Check service health"
  echo -e "   ${COLOR_DIM}View running containers and their status${COLOR_RESET:-}"
  echo ""
  echo -e "${COLOR_BLUE:-}3.${COLOR_RESET:-} nself logs - View service logs"
  echo -e "   ${COLOR_DIM}Monitor real-time logs from all services${COLOR_RESET:-}"
  echo ""
  echo "For more help, use: nself help or nself help build"
  echo ""

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
  set_default "REDIS_ENABLED" "false"
  set_default "MINIO_ENABLED" "false"

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