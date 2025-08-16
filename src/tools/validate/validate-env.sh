#!/usr/bin/env bash

# validate-env.sh - Comprehensive environment file validation

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source display utilities
source "$SCRIPT_DIR/../../lib/utils/display.sh"

# Track validation errors
VALIDATION_ERRORS=()
VALIDATION_WARNINGS=()

# Add an error to the list
add_error() {
  VALIDATION_ERRORS+=("$1")
}

# Add a warning to the list
add_warning() {
  VALIDATION_WARNINGS+=("$1")
}

# Validate env file syntax
validate_env_syntax() {
  local env_file="${1:-.env.local}"
  local line_num=0
  local has_inline_comments=false

  while IFS= read -r line || [ -n "$line" ]; do
    line_num=$((line_num + 1))

    # Skip empty lines and comments
    if [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*# ]]; then
      continue
    fi

    # Check for valid variable assignment
    if [[ ! "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*= ]]; then
      add_error "Line $line_num: Invalid syntax - '$line'"
      add_error "  Variables must start with a letter or underscore"
      continue
    fi

    # Extract variable name and value
    var_name="${line%%=*}"
    var_value="${line#*=}"

    # Check for inline comments (the most common issue)
    if [[ "$var_value" =~ ^[^\"\']*#.*$ ]] && [[ ! "$var_value" =~ ^\{.*\}$ ]]; then
      has_inline_comments=true
      # Just count them, we'll offer to fix at the end
      continue
    fi

    # Check for unquoted values with spaces (but not if it's an inline comment)
    if [[ ! "$var_value" =~ "#" ]] && [[ "$var_value" =~ [[:space:]] ]] && [[ ! "$var_value" =~ ^[\"\'] ]] && [[ ! "$var_value" =~ ^\{.*\}$ ]]; then
      # Not quoted and contains spaces (and not JSON or inline comment)
      add_error "Line $line_num: Unquoted value with spaces in $var_name"
      add_error "  Use quotes: $var_name=\"$var_value\""
    fi

    # Check for missing quotes around URLs with special characters
    if [[ "$var_name" =~ (URL|URI|ENDPOINT|HOST) ]] && [[ "$var_value" =~ [\#\&\?] ]] && [[ ! "$var_value" =~ ^[\"\'] ]]; then
      # Only warn if it's not an inline comment issue
      if [[ ! "$var_value" =~ "#" ]]; then
        add_warning "Line $line_num: URL with special characters should be quoted in $var_name"
      fi
    fi

    # Check for trailing spaces
    if [[ "$line" =~ [[:space:]]$ ]]; then
      add_warning "Line $line_num: Trailing whitespace detected"
    fi

    # Check for Windows line endings
    if [[ "$line" =~ $'\r' ]]; then
      add_error "Line $line_num: Windows line endings (\\r\\n) detected"
      add_error "  Run: dos2unix $env_file"
    fi
  done <"$env_file"

  # If we found inline comments, offer to fix them
  if [ "$has_inline_comments" = true ]; then
    echo ""
    log_warning "⚠️  Inline comments detected in $env_file"
    log_warning "Docker Compose cannot parse inline comments correctly."
    echo ""
    log_info "Would you like to automatically fix this? (Y/n)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY]|)$ ]]; then
      log_info "Creating backup: ${env_file}.backup"
      cp "$env_file" "${env_file}.backup"

      log_info "Removing inline comments..."
      # Use the clean_env_file function to fix the file
      source "$SCRIPT_DIR/../../lib/utils/env.sh"
      clean_env_file "$env_file" "$env_file.tmp"
      mv "$env_file.tmp" "$env_file"

      log_success "✅ Inline comments removed. Original saved as ${env_file}.backup"
      log_info "Please review the changes and run 'nself start' again."
      return 2 # Special return code to indicate file was fixed
    else
      add_error "Inline comments must be removed for Docker Compose compatibility"
      add_error "  Run: nself fix-env"
      add_error "  Or manually move comments to their own lines"
    fi
  fi
}

# Validate required variables
validate_required_vars() {
  local env_file="${1:-.env.local}"

  # Load environment safely
  source "$SCRIPT_DIR/../../lib/utils/env.sh"
  load_env_safe "$env_file"

  # Required variables
  local required_vars=(
    "PROJECT_NAME"
    "BASE_DOMAIN"
    "POSTGRES_DB"
    "POSTGRES_USER"
    "POSTGRES_PASSWORD"
    "HASURA_GRAPHQL_ADMIN_SECRET"
  )

  for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
      add_error "Required variable $var is not set"
    fi
  done

  # Check for default/insecure values
  if [[ "$ENV" == "prod" ]] || [[ "$ENVIRONMENT" == "production" ]]; then
    # In production, these are errors
    if [[ "$POSTGRES_PASSWORD" == "secretpassword" ]]; then
      add_error "Default PostgreSQL password detected in production!"
      add_error "  Generate secure password: openssl rand -hex 32"
    fi

    if [[ "$HASURA_GRAPHQL_ADMIN_SECRET" == "hasura-admin-secret" ]]; then
      add_error "Default Hasura admin secret detected in production!"
      add_error "  Generate secure secret: openssl rand -hex 32"
    fi

    if [[ "${#POSTGRES_PASSWORD}" -lt 12 ]]; then
      add_error "PostgreSQL password too short for production (min 12 chars)"
    fi

    # Check for missing required production values
    if [[ -z "$LETSENCRYPT_EMAIL" ]] && [[ "$SSL_MODE" == "letsencrypt" ]]; then
      add_error "LETSENCRYPT_EMAIL is required for Let's Encrypt SSL"
    fi
  else
    # In development, these are just warnings
    if [[ "$POSTGRES_PASSWORD" == "secretpassword" ]]; then
      add_warning "Using default PostgreSQL password (OK for development)"
    fi

    if [[ "$HASURA_GRAPHQL_ADMIN_SECRET" == "hasura-admin-secret" ]]; then
      add_warning "Using default Hasura admin secret (OK for development)"
    fi
  fi

  # Check MinIO credentials
  if [[ "$MINIO_ROOT_PASSWORD" == "minioadmin" ]]; then
    if [[ "$ENV" == "prod" ]] || [[ "$ENVIRONMENT" == "production" ]]; then
      add_error "Default MinIO password detected in production!"
      add_error "  Set a secure MINIO_ROOT_PASSWORD"
    else
      add_warning "Using default MinIO password (OK for development)"
    fi
  fi

  # Check S3 credentials
  if [[ "$S3_SECRET_KEY" == "storage-secret-key" ]]; then
    if [[ "$ENV" == "prod" ]] || [[ "$ENVIRONMENT" == "production" ]]; then
      add_error "Default S3 secret key detected in production!"
      add_error "  Generate secure key: openssl rand -hex 32"
    else
      add_warning "Using default S3 secret key (OK for development)"
    fi
  fi
}

# Validate port availability
validate_ports() {
  local env_file="${1:-.env.local}"

  # Load environment
  source "$SCRIPT_DIR/../../lib/utils/env.sh"
  load_env_safe "$env_file"

  # List of ports to check
  local ports=(
    "${POSTGRES_PORT:-5432}:PostgreSQL"
    "${HASURA_PORT:-8080}:Hasura"
    "${AUTH_PORT:-4000}:Auth"
    "${MINIO_PORT:-9000}:MinIO"
    "${NGINX_HTTP_PORT:-80}:Nginx HTTP"
    "${NGINX_HTTPS_PORT:-443}:Nginx HTTPS"
  )

  # Add service ports if enabled
  if [[ "$REDIS_ENABLED" == "true" ]]; then
    ports+=("${REDIS_PORT:-6379}:Redis")
  fi

  if [[ "$NESTJS_ENABLED" == "true" ]]; then
    IFS=',' read -ra NEST_SERVICES <<<"${NESTJS_SERVICES:-}"
    local port_counter=0
    for service in "${NEST_SERVICES[@]}"; do
      service=$(echo "$service" | xargs)
      if [ -n "$service" ]; then
        local service_port=$((${NESTJS_PORT_START:-4100} + port_counter))
        ports+=("$service_port:NestJS-$service")
        port_counter=$((port_counter + 1))
      fi
    done
  fi

  # Check each port
  for port_info in "${ports[@]}"; do
    local port="${port_info%%:*}"
    local service="${port_info#*:}"

    # Check if port is in use
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
      local process_info=$(lsof -Pi :$port -sTCP:LISTEN 2>/dev/null | tail -1 | awk '{print $1}')

      # Check if it's a Docker container from our project
      local conflicting_container=$(docker ps --filter "publish=$port" --format "{{.Names}}" 2>/dev/null | grep "${PROJECT_NAME}" | head -1)

      if [ -n "$conflicting_container" ]; then
        # It's our own container - this can be auto-fixed during startup
        add_warning "Port $port ($service) used by old container: $conflicting_container (will auto-fix)"
      else
        # External process using the port - let startup handle this with user choices
        add_warning "Port $port ($service) is busy (used by: $process_info)"
        add_warning "  nself start will offer options to resolve this automatically"
      fi
    fi
  done
}

# Validate Docker resources
validate_docker_resources() {
  # Check if Docker is running
  if ! docker info >/dev/null 2>&1; then
    add_error "Docker is not running or not accessible"
    add_error "  Start Docker Desktop or Docker daemon"
    return 1
  fi

  # Check Docker Compose version
  if ! docker compose version >/dev/null 2>&1; then
    add_error "Docker Compose v2 is not available"
    add_error "  Update Docker Desktop or install docker-compose-plugin"
    return 1
  fi

  # Check available disk space
  local available_space=$(df -k . | awk 'NR==2 {print $4}')
  local required_space=$((5 * 1024 * 1024)) # 5GB in KB

  if [ "$available_space" -lt "$required_space" ]; then
    add_warning "Low disk space: $((available_space / 1024 / 1024))GB available"
    add_warning "  Recommended: At least 5GB free space"
  fi

  # Check Docker disk usage
  local docker_disk=$(docker system df --format "table {{.Size}}" | tail -1 2>/dev/null || echo "0")
  if [[ "$docker_disk" =~ ([0-9]+)GB ]] && [ "${BASH_REMATCH[1]}" -gt 20 ]; then
    add_warning "Docker using ${BASH_REMATCH[1]}GB of disk space"
    add_warning "  Run: docker system prune -a"
  fi
}

# Validate service configurations
validate_service_configs() {
  local env_file="${1:-.env.local}"

  # Load environment
  source "$SCRIPT_DIR/../../lib/utils/env.sh"
  load_env_safe "$env_file"

  # Check NestJS services
  if [[ "$NESTJS_ENABLED" == "true" ]]; then
    if [ -z "$NESTJS_SERVICES" ]; then
      add_error "NESTJS_ENABLED=true but NESTJS_SERVICES is empty"
      add_error "  Example: NESTJS_SERVICES=\"api,worker,admin\""
    else
      # Validate service names
      IFS=',' read -ra services <<<"$NESTJS_SERVICES"
      for service in "${services[@]}"; do
        service=$(echo "$service" | xargs)
        if [[ ! "$service" =~ ^[a-z][a-z0-9-]*$ ]]; then
          add_error "Invalid NestJS service name: '$service'"
          add_error "  Use lowercase letters, numbers, and hyphens only"
        fi
      done
    fi
  fi

  # Check email configuration
  if [[ "$AUTH_SMTP_HOST" == "" ]] && [[ "$EMAIL_PROVIDER" != "mailhog" ]] && [[ "$EMAIL_PROVIDER" != "mailpit" ]]; then
    add_warning "No email configuration found"
    add_warning "  Auth service email features will not work"
    add_warning "  Set EMAIL_PROVIDER=mailhog for development"
  fi

  # Check SSL configuration for production
  if [[ "$ENV" == "prod" ]] && [[ "$SSL_MODE" == "none" ]]; then
    add_error "SSL is disabled in production!"
    add_error "  Set SSL_MODE=letsencrypt or SSL_MODE=custom"
  fi

  # Check JWT secret configuration
  if [[ "$HASURA_GRAPHQL_JWT_SECRET" == *"CHANGE-THIS"* ]]; then
    if [[ "$ENV" == "prod" ]] || [[ "$ENVIRONMENT" == "production" ]]; then
      add_error "Default JWT secret detected in production!"
      add_error "  Generate secure key: openssl rand -hex 32"
    else
      add_warning "Using default JWT secret (OK for development)"
      add_warning "  For production, generate key: openssl rand -hex 32"
    fi
  fi

  # Validate JSON in JWT secret
  if [[ -n "$HASURA_GRAPHQL_JWT_SECRET" ]] && [[ "$HASURA_GRAPHQL_JWT_SECRET" =~ ^\{ ]]; then
    if ! echo "$HASURA_GRAPHQL_JWT_SECRET" | python3 -m json.tool >/dev/null 2>&1; then
      add_error "Invalid JSON in HASURA_GRAPHQL_JWT_SECRET"
      add_error "  Check for missing quotes or commas"
    fi
  fi
}

# Validate domain and routing configuration
validate_routing() {
  local env_file="${1:-.env.local}"

  # Load environment
  source "$SCRIPT_DIR/../../lib/utils/env.sh"
  load_env_safe "$env_file"

  # Check BASE_DOMAIN format
  if [[ "$BASE_DOMAIN" =~ ^https?:// ]]; then
    add_error "BASE_DOMAIN should not include protocol"
    add_error "  Use: BASE_DOMAIN=\"example.com\" not \"https://example.com\""
  fi

  if [[ "$BASE_DOMAIN" =~ /$ ]]; then
    add_error "BASE_DOMAIN should not end with /"
    add_error "  Use: BASE_DOMAIN=\"example.com\" not \"example.com/\""
  fi

  # Check if subdomains are properly configured
  local expected_routes=(
    "api.${BASE_DOMAIN}:Hasura GraphQL"
    "auth.${BASE_DOMAIN}:Authentication"
    "storage.${BASE_DOMAIN}:File Storage"
  )

  if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
    expected_routes+=("dashboard.${BASE_DOMAIN}:Admin Dashboard")
  fi

  if [[ "$EMAIL_PROVIDER" == "mailhog" ]] || [[ "$EMAIL_PROVIDER" == "mailpit" ]]; then
    expected_routes+=("mailhog.${BASE_DOMAIN}:Email Testing")
  fi

  echo ""
  log_info "Expected service routes:"
  for route in "${expected_routes[@]}"; do
    echo "  • ${route%%:*} → ${route#*:}"
  done

  # Check custom app routing
  if [[ -n "$APP_DOMAINS" ]]; then
    echo ""
    log_info "Custom app domains configured:"
    IFS=',' read -ra domains <<<"$APP_DOMAINS"
    for domain in "${domains[@]}"; do
      domain=$(echo "$domain" | xargs)
      if [[ ! "$domain" =~ ^[a-z0-9.-]+$ ]]; then
        add_error "Invalid domain format: $domain"
        add_error "  Use lowercase letters, numbers, dots, and hyphens only"
      else
        echo "  • $domain"
      fi
    done
  fi
}

# Main validation function
validate_env() {
  local env_file="${1:-.env.local}"

  if [ ! -f "$env_file" ]; then
    log_error "Environment file not found: $env_file"
    log_info "Run 'nself init' to create one"
    return 1
  fi

  log_info "Validating environment configuration..."
  echo ""

  # Run all validations
  validate_env_syntax "$env_file"
  validate_required_vars "$env_file"
  validate_ports
  validate_docker_resources
  validate_service_configs "$env_file"
  validate_routing "$env_file"

  # Display results
  echo ""
  if [ ${#VALIDATION_ERRORS[@]} -gt 0 ]; then
    log_error "❌ Validation failed with ${#VALIDATION_ERRORS[@]} error(s):"
    echo ""
    for error in "${VALIDATION_ERRORS[@]}"; do
      log_error "$error"
    done
    echo ""
  fi

  if [ ${#VALIDATION_WARNINGS[@]} -gt 0 ]; then
    log_warning "⚠️  ${#VALIDATION_WARNINGS[@]} warning(s):"
    echo ""
    for warning in "${VALIDATION_WARNINGS[@]}"; do
      log_warning "$warning"
    done
    echo ""
  fi

  if [ ${#VALIDATION_ERRORS[@]} -eq 0 ]; then
    if [ ${#VALIDATION_WARNINGS[@]} -eq 0 ]; then
      log_success "✅ Environment validation passed!"
    else
      log_success "✅ Environment validation passed with warnings"
    fi
    return 0
  else
    log_error "Fix the errors above before running 'nself start'"
    return 1
  fi
}

# Export for use in other scripts
export -f validate_env
export -f validate_env_syntax
export -f validate_required_vars
export -f validate_ports
export -f validate_docker_resources
export -f validate_service_configs
export -f validate_routing

# Run validation if called directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  # Parse arguments
  env_file=".env.local"
  apply_fixes=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
    --apply-fixes)
      apply_fixes=true
      shift
      ;;
    -*)
      log_error "Unknown option: $1"
      echo "Usage: nself validate-env [env-file] [--apply-fixes]"
      exit 1
      ;;
    *)
      env_file="$1"
      shift
      ;;
    esac
  done

  # Run validation
  validate_env "$env_file"
  result=$?

  # If there were errors and apply-fixes was requested, try to fix them
  if [[ $result -ne 0 ]] && [[ "$apply_fixes" == "true" ]]; then
    log_info "Attempting to apply fixes..."
    # TODO: Implement auto-fixes for common issues
    log_warning "Auto-fix not yet implemented for all issues"
  fi

  exit $result
fi
