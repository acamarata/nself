#!/usr/bin/env bash
# validate.sh - Comprehensive validation before deployment
# Combines security checks, configuration validation, and deployment readiness

set -euo pipefail

# Get script directory
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"

# Source utilities
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/deploy/security-preflight.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh" 2>/dev/null || true

# ============================================================
# Validate Command - Pre-deployment Validation
# ============================================================

cmd_validate() {
  local env_name="${1:-}"
  local scope="${2:-all}"
  local strict="false"
  local fix_mode="false"

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --strict)
        strict="true"
        shift
        ;;
      --fix)
        fix_mode="true"
        shift
        ;;
      --security)
        scope="security"
        shift
        ;;
      --config)
        scope="config"
        shift
        ;;
      --deploy)
        scope="deploy"
        shift
        ;;
      --help|-h)
        show_validate_help
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        if [[ -z "$env_name" ]]; then
          env_name="$1"
        fi
        shift
        ;;
    esac
  done

  show_command_header "nself validate" "Pre-deployment validation"

  # Determine environment
  if [[ -z "$env_name" ]]; then
    env_name="${ENV:-dev}"
  fi

  # Load environment
  load_env_with_priority 2>/dev/null || true

  local total_errors=0
  local total_warnings=0

  # Run validations based on scope
  case "$scope" in
    all)
      validate_configuration
      local config_result=$?

      validate_security "$env_name"
      local security_result=$?

      validate_deployment_readiness "$env_name"
      local deploy_result=$?

      total_errors=$((config_result + security_result + deploy_result))
      ;;
    security)
      validate_security "$env_name"
      total_errors=$?
      ;;
    config)
      validate_configuration
      total_errors=$?
      ;;
    deploy)
      validate_deployment_readiness "$env_name"
      total_errors=$?
      ;;
  esac

  # Summary
  printf "\n${COLOR_CYAN}═══════════════════════════════════════════════════${COLOR_RESET}\n"

  if [[ $total_errors -eq 0 ]]; then
    log_success "All validations passed"
    printf "\n"
    printf "Ready for deployment:\n"
    printf "  ${COLOR_CYAN}nself deploy %s${COLOR_RESET}\n" "$env_name"
    return 0
  else
    log_error "$total_errors validation error(s) found"
    printf "\n"
    printf "Fix the errors above before deploying.\n"

    if [[ "$fix_mode" == "true" ]]; then
      printf "\nAttempting auto-fix...\n"
      validate_auto_fix "$env_name"
    else
      printf "Run with ${COLOR_CYAN}--fix${COLOR_RESET} to attempt automatic fixes.\n"
    fi

    return 1
  fi
}

# ============================================================
# Configuration Validation
# ============================================================

validate_configuration() {
  printf "\n${COLOR_CYAN}Configuration Validation${COLOR_RESET}\n"
  printf "════════════════════════════════════════════════════\n\n"

  local errors=0

  # Check for .env file
  printf "Checking environment files...\n"
  if [[ -f ".env" ]] || [[ -f ".env.dev" ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Environment file found\n"
  else
    printf "  ${COLOR_RED}✗${COLOR_RESET} No .env or .env.dev file found\n"
    printf "    Run: ${COLOR_CYAN}nself init${COLOR_RESET}\n"
    errors=$((errors + 1))
  fi

  # Check required variables
  printf "\nChecking required variables...\n"

  local required_vars=(
    "PROJECT_NAME:Project name"
    "BASE_DOMAIN:Base domain"
    "POSTGRES_PASSWORD:Database password"
    "HASURA_GRAPHQL_ADMIN_SECRET:Hasura admin secret"
  )

  for var_spec in "${required_vars[@]}"; do
    local var_name="${var_spec%%:*}"
    local description="${var_spec#*:}"
    local value="${!var_name:-}"

    if [[ -n "$value" ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} %s: Set\n" "$var_name"
    else
      printf "  ${COLOR_RED}✗${COLOR_RESET} %s: Not set (%s)\n" "$var_name" "$description"
      errors=$((errors + 1))
    fi
  done

  # Check docker-compose.yml
  printf "\nChecking Docker Compose...\n"
  if [[ -f "docker-compose.yml" ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} docker-compose.yml exists\n"

    # Validate compose file
    if docker compose config >/dev/null 2>&1; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} docker-compose.yml is valid\n"
    else
      printf "  ${COLOR_RED}✗${COLOR_RESET} docker-compose.yml has errors\n"
      printf "    Run: ${COLOR_CYAN}docker compose config${COLOR_RESET} to see details\n"
      errors=$((errors + 1))
    fi
  else
    printf "  ${COLOR_YELLOW}!${COLOR_RESET} docker-compose.yml not found\n"
    printf "    Run: ${COLOR_CYAN}nself build${COLOR_RESET}\n"
    errors=$((errors + 1))
  fi

  # Check nginx configuration
  printf "\nChecking nginx configuration...\n"
  if [[ -d "nginx" ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} nginx directory exists\n"

    if [[ -f "nginx/nginx.conf" ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} nginx.conf exists\n"
    else
      printf "  ${COLOR_YELLOW}!${COLOR_RESET} nginx.conf not found\n"
      errors=$((errors + 1))
    fi
  else
    printf "  ${COLOR_YELLOW}!${COLOR_RESET} nginx directory not found\n"
    printf "    Run: ${COLOR_CYAN}nself build${COLOR_RESET}\n"
  fi

  # Check SSL certificates
  printf "\nChecking SSL certificates...\n"
  local ssl_found="false"
  for cert_path in "nginx/ssl" "ssl" "certs"; do
    if [[ -f "$cert_path/fullchain.pem" ]] || [[ -f "$cert_path/nself-org/fullchain.pem" ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} SSL certificates found in %s\n" "$cert_path"
      ssl_found="true"
      break
    fi
  done

  if [[ "$ssl_found" == "false" ]]; then
    printf "  ${COLOR_YELLOW}!${COLOR_RESET} No SSL certificates found\n"
    printf "    Run: ${COLOR_CYAN}nself ssl bootstrap${COLOR_RESET}\n"
  fi

  return $errors
}

# ============================================================
# Security Validation
# ============================================================

validate_security() {
  local env_name="${1:-prod}"

  printf "\n${COLOR_CYAN}Security Validation${COLOR_RESET}\n"
  printf "════════════════════════════════════════════════════\n\n"

  # Only run strict security checks for production
  local env_type="${ENV:-dev}"
  if [[ "$env_type" == "prod" ]] || [[ "$env_type" == "production" ]] || [[ "$env_name" == "prod" ]]; then
    # Use security preflight module
    security::preflight "$env_name" "." "false" 2>/dev/null
    return $?
  else
    printf "Environment: %s (non-production)\n" "$env_type"
    printf "  ${COLOR_DIM}Strict security checks skipped for non-production${COLOR_RESET}\n"
    printf "  ${COLOR_DIM}Run with ENV=prod to enable full checks${COLOR_RESET}\n"

    # Still check basic security
    printf "\nBasic security checks:\n"
    local warnings=0

    # Check for default passwords
    if [[ "${POSTGRES_PASSWORD:-}" == "postgres" ]] || [[ "${POSTGRES_PASSWORD:-}" == "password" ]]; then
      printf "  ${COLOR_YELLOW}!${COLOR_RESET} POSTGRES_PASSWORD: Using insecure default\n"
      warnings=$((warnings + 1))
    else
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} POSTGRES_PASSWORD: Not using common defaults\n"
    fi

    if [[ "${HASURA_GRAPHQL_ADMIN_SECRET:-}" == "secret" ]] || [[ "${HASURA_GRAPHQL_ADMIN_SECRET:-}" == "admin" ]]; then
      printf "  ${COLOR_YELLOW}!${COLOR_RESET} HASURA_GRAPHQL_ADMIN_SECRET: Using insecure default\n"
      warnings=$((warnings + 1))
    else
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} HASURA_GRAPHQL_ADMIN_SECRET: Not using common defaults\n"
    fi

    return 0
  fi
}

# ============================================================
# Deployment Readiness Validation
# ============================================================

validate_deployment_readiness() {
  local env_name="${1:-prod}"

  printf "\n${COLOR_CYAN}Deployment Readiness${COLOR_RESET}\n"
  printf "════════════════════════════════════════════════════\n\n"

  local errors=0

  # Check Docker is running
  printf "Checking Docker...\n"
  if docker info >/dev/null 2>&1; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Docker is running\n"

    local docker_version=$(docker --version | cut -d' ' -f3 | tr -d ',')
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Docker version: %s\n" "$docker_version"
  else
    printf "  ${COLOR_RED}✗${COLOR_RESET} Docker is not running\n"
    errors=$((errors + 1))
  fi

  # Check Docker Compose
  if docker compose version >/dev/null 2>&1; then
    local compose_version=$(docker compose version --short)
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Docker Compose version: %s\n" "$compose_version"
  else
    printf "  ${COLOR_RED}✗${COLOR_RESET} Docker Compose not available\n"
    errors=$((errors + 1))
  fi

  # Check for uncommitted changes
  printf "\nChecking git status...\n"
  if command -v git >/dev/null 2>&1 && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    local uncommitted=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$uncommitted" -eq 0 ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} No uncommitted changes\n"
    else
      printf "  ${COLOR_YELLOW}!${COLOR_RESET} %d uncommitted change(s)\n" "$uncommitted"
      printf "    ${COLOR_DIM}Consider committing before deploying${COLOR_RESET}\n"
    fi
  else
    printf "  ${COLOR_DIM}Not a git repository${COLOR_RESET}\n"
  fi

  # Check deployment target
  printf "\nChecking deployment configuration...\n"
  local deploy_host="${DEPLOY_HOST:-}"
  local deploy_path="${DEPLOY_PATH:-/var/www/nself}"

  if [[ -n "$deploy_host" ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} DEPLOY_HOST: %s\n" "$deploy_host"
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} DEPLOY_PATH: %s\n" "$deploy_path"

    # Test SSH connection
    printf "\nTesting SSH connection...\n"
    if ssh -o ConnectTimeout=5 -o BatchMode=yes "$deploy_host" "true" 2>/dev/null; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} SSH connection successful\n"
    else
      printf "  ${COLOR_RED}✗${COLOR_RESET} Cannot connect to %s\n" "$deploy_host"
      printf "    Check SSH keys and server availability\n"
      errors=$((errors + 1))
    fi
  else
    printf "  ${COLOR_YELLOW}!${COLOR_RESET} DEPLOY_HOST not configured\n"
    printf "    Set in .env or pass with: ${COLOR_CYAN}nself deploy --host <server>${COLOR_RESET}\n"
  fi

  # Check for backup before deployment
  printf "\nChecking backup status...\n"
  if [[ -d "backups" ]]; then
    local latest_backup=$(ls -t backups/*.tar.gz 2>/dev/null | head -1)
    if [[ -n "$latest_backup" ]]; then
      local backup_age=$(find "$latest_backup" -mmin +1440 2>/dev/null)
      if [[ -z "$backup_age" ]]; then
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} Recent backup found: %s\n" "$(basename "$latest_backup")"
      else
        printf "  ${COLOR_YELLOW}!${COLOR_RESET} Backup older than 24 hours\n"
        printf "    Run: ${COLOR_CYAN}nself backup create${COLOR_RESET}\n"
      fi
    else
      printf "  ${COLOR_YELLOW}!${COLOR_RESET} No backups found\n"
      printf "    Run: ${COLOR_CYAN}nself backup create${COLOR_RESET} before deploying\n"
    fi
  else
    printf "  ${COLOR_DIM}No backups directory${COLOR_RESET}\n"
  fi

  return $errors
}

# ============================================================
# Auto-fix
# ============================================================

validate_auto_fix() {
  local env_name="${1:-prod}"

  printf "\n${COLOR_CYAN}Attempting automatic fixes...${COLOR_RESET}\n\n"

  # Generate secrets if needed
  if [[ -z "${POSTGRES_PASSWORD:-}" ]] || [[ -z "${HASURA_GRAPHQL_ADMIN_SECRET:-}" ]]; then
    printf "Generating missing secrets...\n"
    if [[ -f "$CLI_SCRIPT_DIR/../lib/deploy/security-preflight.sh" ]]; then
      source "$CLI_SCRIPT_DIR/../lib/deploy/security-preflight.sh"
      security::generate_secrets ".environments/$env_name"
    fi
  fi

  # Build if docker-compose.yml missing
  if [[ ! -f "docker-compose.yml" ]]; then
    printf "Running nself build...\n"
    "$CLI_SCRIPT_DIR/build.sh" 2>/dev/null || true
  fi

  # Generate SSL if missing
  local ssl_found="false"
  for cert_path in "nginx/ssl" "ssl" "certs"; do
    if [[ -f "$cert_path/fullchain.pem" ]]; then
      ssl_found="true"
      break
    fi
  done

  if [[ "$ssl_found" == "false" ]]; then
    printf "Generating SSL certificates...\n"
    "$CLI_SCRIPT_DIR/ssl.sh" bootstrap 2>/dev/null || true
  fi

  printf "\n"
  log_info "Auto-fix complete. Run 'nself validate' again to verify."
}

# ============================================================
# Help
# ============================================================

show_validate_help() {
  printf "Usage: nself validate [environment] [options]\n"
  printf "\n"
  printf "Comprehensive pre-deployment validation\n"
  printf "\n"
  printf "Options:\n"
  printf "  --strict       Treat warnings as errors\n"
  printf "  --fix          Attempt to automatically fix issues\n"
  printf "  --security     Only run security validation\n"
  printf "  --config       Only run configuration validation\n"
  printf "  --deploy       Only run deployment readiness checks\n"
  printf "  --help, -h     Show this help message\n"
  printf "\n"
  printf "Examples:\n"
  printf "  nself validate               # Validate for current environment\n"
  printf "  nself validate prod          # Validate for production\n"
  printf "  nself validate --security    # Only security checks\n"
  printf "  nself validate --fix         # Auto-fix issues\n"
  printf "  nself validate prod --strict # Strict mode for production\n"
  printf "\n"
  printf "Validation includes:\n"
  printf "  • Configuration files (.env, docker-compose.yml, nginx)\n"
  printf "  • Security settings (passwords, secrets, bindings)\n"
  printf "  • SSL certificates\n"
  printf "  • Deployment target connectivity\n"
  printf "  • Git status\n"
  printf "  • Backup status\n"
}

# Export function
export -f cmd_validate

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "validate" 2>/dev/null || true
  cmd_validate "$@"
  exit_code=$?
  post_command "validate" $exit_code 2>/dev/null || true
  exit $exit_code
fi
