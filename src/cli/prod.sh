#!/usr/bin/env bash
set -euo pipefail

# prod.sh - Production environment management and hardening
# Provides subcommands for production configuration, security, and monitoring

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
LIB_DIR="$SCRIPT_DIR/../lib"

source "$LIB_DIR/utils/display.sh"
source "$LIB_DIR/utils/env.sh"
source "$LIB_DIR/utils/platform-compat.sh" 2>/dev/null || true

# Source security modules
source "$LIB_DIR/security/checklist.sh" 2>/dev/null || true
source "$LIB_DIR/security/secrets.sh" 2>/dev/null || true
source "$LIB_DIR/security/ssl-letsencrypt.sh" 2>/dev/null || true
source "$LIB_DIR/security/firewall.sh" 2>/dev/null || true

# Main command function
cmd_prod() {
  local subcommand="${1:-status}"
  shift 2>/dev/null || true

  case "$subcommand" in
    init)
      prod_init "$@"
      ;;
    check|audit)
      prod_check "$@"
      ;;
    secrets)
      prod_secrets "$@"
      ;;
    ssl)
      prod_ssl "$@"
      ;;
    firewall)
      prod_firewall "$@"
      ;;
    harden)
      prod_harden "$@"
      ;;
    status)
      prod_status "$@"
      ;;
    --help|-h|help)
      show_prod_help
      ;;
    *)
      # Backward compatibility: treat as domain for init
      if [[ "$subcommand" =~ \. ]]; then
        prod_init "$subcommand" "$@"
      else
        log_error "Unknown subcommand: $subcommand"
        printf "\n"
        show_prod_help
        return 1
      fi
      ;;
  esac
}

# Initialize production configuration
prod_init() {
  local domain="${1:-}"
  local email="${2:-}"

  # Parse additional arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
      --email)
        email="$2"
        shift 2
        ;;
      --force|-f)
        local force="true"
        shift
        ;;
      *)
        if [[ -z "$domain" ]]; then
          domain="$1"
        elif [[ -z "$email" ]]; then
          email="$1"
        fi
        shift
        ;;
    esac
  done

  show_command_header "nself prod init" "Initialize production configuration"

  if [[ -z "$domain" ]]; then
    log_error "Domain is required"
    printf "Usage: nself prod init <domain> [--email <email>]\n"
    return 1
  fi

  log_info "Configuring production for domain: $domain"

  # Load current environment
  if [[ -f ".env" ]] || [[ -f ".env.dev" ]]; then
    load_env_with_priority
  else
    log_error "No .env or .env.dev found. Run 'nself init' first"
    return 1
  fi

  # Determine which env file to modify
  local env_file=".env"
  [[ ! -f ".env" ]] && [[ -f ".env.dev" ]] && env_file=".env.dev"

  log_info "Updating configuration..."

  # Set production domain and environment
  set_env_var "BASE_DOMAIN" "$domain" "$env_file"
  set_env_var "ENV" "production" "$env_file"
  set_env_var "NODE_ENV" "production" "$env_file"
  set_env_var "GO_ENV" "production" "$env_file"
  set_env_var "DEBUG" "false" "$env_file"
  set_env_var "LOG_LEVEL" "warning" "$env_file"

  # Enable SSL
  set_env_var "SSL_ENABLED" "true" "$env_file"
  set_env_var "SSL_PROVIDER" "letsencrypt" "$env_file"

  if [[ -n "$email" ]]; then
    set_env_var "SSL_EMAIL" "$email" "$env_file"
  fi

  # Disable Hasura dev features
  set_env_var "HASURA_GRAPHQL_DEV_MODE" "false" "$env_file"
  set_env_var "HASURA_GRAPHQL_ENABLE_CONSOLE" "false" "$env_file"

  # Create production compose override
  create_prod_compose

  log_success "Production configuration initialized"
  printf "\n"
  printf "Next steps:\n"
  printf "  1. Generate secrets:  nself prod secrets generate\n"
  printf "  2. Run security check: nself prod check\n"
  printf "  3. Configure SSL:     nself prod ssl request %s\n" "$domain"
  printf "  4. Configure firewall: nself prod firewall\n"
  printf "  5. Build and deploy:  nself build && nself deploy prod\n"
  printf "\n"

  return 0
}

# Run security audit
prod_check() {
  local verbose="${1:-false}"

  [[ "$1" == "--verbose" || "$1" == "-v" ]] && verbose="true"

  show_command_header "nself prod check" "Production security audit"

  if command -v security::audit >/dev/null 2>&1; then
    security::audit "" "$verbose"
    return $?
  else
    # Fallback basic checks
    log_info "Running basic security checks..."

    local errors=0

    # Check DEBUG mode
    local debug_mode
    debug_mode=$(grep "^DEBUG=" .env 2>/dev/null | cut -d'=' -f2)
    if [[ "$debug_mode" == "true" ]]; then
      printf "  ${COLOR_RED}✗${COLOR_RESET} DEBUG mode is enabled\n"
      errors=$((errors + 1))
    else
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} DEBUG mode is disabled\n"
    fi

    # Check SSL
    local ssl_enabled
    ssl_enabled=$(grep "^SSL_ENABLED=" .env 2>/dev/null | cut -d'=' -f2)
    if [[ "$ssl_enabled" != "true" ]]; then
      printf "  ${COLOR_RED}✗${COLOR_RESET} SSL is not enabled\n"
      errors=$((errors + 1))
    else
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} SSL is enabled\n"
    fi

    # Check for default passwords
    if grep -qE "(password|changeme|secret123)" .env 2>/dev/null; then
      printf "  ${COLOR_RED}✗${COLOR_RESET} Weak/default passwords detected\n"
      errors=$((errors + 1))
    else
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} No obvious weak passwords\n"
    fi

    printf "\n"
    if [[ $errors -gt 0 ]]; then
      log_error "Security check failed with $errors issue(s)"
      return 1
    else
      log_success "Basic security check passed"
      return 0
    fi
  fi
}

# Manage secrets
prod_secrets() {
  local action="${1:-help}"
  shift 2>/dev/null || true

  show_command_header "nself prod secrets" "Secrets management"

  case "$action" in
    generate)
      local force=""
      [[ "$1" == "--force" || "$1" == "-f" ]] && force="true"

      if command -v secrets::generate_all >/dev/null 2>&1; then
        secrets::generate_all ".env.secrets" "$force"
      else
        # Fallback
        log_info "Generating production secrets..."

        if [[ -f ".env.secrets" ]] && [[ "$force" != "true" ]]; then
          log_error "Secrets file already exists. Use --force to overwrite."
          return 1
        fi

        cat > ".env.secrets" <<EOF
# Production Secrets - NEVER COMMIT TO VERSION CONTROL
# Generated by nself on $(date +%Y-%m-%d)

POSTGRES_PASSWORD=$(openssl rand -hex 16)
HASURA_GRAPHQL_ADMIN_SECRET=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
COOKIE_SECRET=$(openssl rand -hex 16)
MINIO_ROOT_PASSWORD=$(openssl rand -hex 16)
REDIS_PASSWORD=$(openssl rand -hex 16)
EOF
        chmod 600 ".env.secrets"
        log_success "Generated secrets in .env.secrets"
      fi
      ;;

    rotate)
      local secret_name="${1:-}"
      if command -v secrets::rotate >/dev/null 2>&1; then
        secrets::rotate "$secret_name"
      else
        log_error "Secret rotation not available"
        return 1
      fi
      ;;

    validate|check)
      if command -v secrets::validate >/dev/null 2>&1; then
        secrets::validate ".env.secrets"
      else
        if [[ -f ".env.secrets" ]]; then
          log_success "Secrets file exists"
          local perms
          perms=$(safe_stat_perms ".env.secrets" 2>/dev/null || stat -f "%OLp" ".env.secrets" 2>/dev/null || echo "unknown")
          printf "  Permissions: %s\n" "$perms"
        else
          log_error "No secrets file found"
          return 1
        fi
      fi
      ;;

    show)
      local unmask=""
      [[ "$1" == "--unmask" ]] && unmask="true"
      if command -v secrets::show >/dev/null 2>&1; then
        secrets::show ".env.secrets" "$unmask"
      else
        log_error "Secrets display not available"
        return 1
      fi
      ;;

    help|--help|-h|*)
      printf "Usage: nself prod secrets <action>\n\n"
      printf "Actions:\n"
      printf "  generate [--force]   Generate all production secrets\n"
      printf "  validate             Validate secrets file\n"
      printf "  rotate <name>        Rotate a specific secret\n"
      printf "  show [--unmask]      Show secrets (masked by default)\n"
      printf "\n"
      printf "Examples:\n"
      printf "  nself prod secrets generate\n"
      printf "  nself prod secrets validate\n"
      printf "  nself prod secrets rotate POSTGRES_PASSWORD\n"
      ;;
  esac
}

# Manage SSL certificates
prod_ssl() {
  local action="${1:-status}"
  shift 2>/dev/null || true

  show_command_header "nself prod ssl" "SSL certificate management"

  case "$action" in
    status)
      if command -v ssl::status >/dev/null 2>&1; then
        ssl::status "./ssl/cert.pem"
      else
        if [[ -f "ssl/cert.pem" ]]; then
          log_success "SSL certificate exists"
          if command -v openssl >/dev/null 2>&1; then
            local expiry
            expiry=$(openssl x509 -enddate -noout -in "ssl/cert.pem" 2>/dev/null | cut -d'=' -f2)
            printf "  Expires: %s\n" "${expiry:-unknown}"
          fi
        else
          log_warning "No SSL certificate found"
        fi
      fi
      ;;

    request)
      local domain="${1:-}"
      local email="${2:-}"
      local staging=""

      # Parse arguments
      while [[ $# -gt 0 ]]; do
        case $1 in
          --staging) staging="true"; shift ;;
          --email) email="$2"; shift 2 ;;
          *)
            if [[ -z "$domain" ]]; then domain="$1"; fi
            shift
            ;;
        esac
      done

      if [[ -z "$domain" ]]; then
        domain=$(grep "^BASE_DOMAIN=" .env 2>/dev/null | cut -d'=' -f2)
      fi

      if [[ -z "$email" ]]; then
        email=$(grep "^SSL_EMAIL=" .env 2>/dev/null | cut -d'=' -f2)
      fi

      if [[ -z "$domain" ]]; then
        log_error "Domain is required"
        printf "Usage: nself prod ssl request <domain> [--email <email>]\n"
        return 1
      fi

      if [[ -z "$email" ]]; then
        log_error "Email is required for Let's Encrypt"
        printf "Use: nself prod ssl request %s --email admin@%s\n" "$domain" "$domain"
        return 1
      fi

      if command -v ssl::request_cert >/dev/null 2>&1; then
        ssl::request_cert "$domain" "$email" "webroot" "$staging"
      else
        log_info "To request an SSL certificate, run:"
        printf "  certbot certonly --webroot -w ./certbot-webroot -d %s --email %s\n" "$domain" "$email"
      fi
      ;;

    renew)
      local force=""
      [[ "$1" == "--force" ]] && force="true"

      if command -v ssl::renew >/dev/null 2>&1; then
        ssl::renew "$force"
      else
        log_info "To renew certificates, run:"
        printf "  certbot renew\n"
      fi
      ;;

    self-signed)
      local domain="${1:-localhost}"
      if command -v ssl::generate_self_signed >/dev/null 2>&1; then
        ssl::generate_self_signed "$domain"
      else
        log_info "Generating self-signed certificate..."
        mkdir -p ssl
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
          -keyout ssl/key.pem -out ssl/cert.pem \
          -subj "/CN=$domain" 2>/dev/null
        log_success "Generated self-signed certificate"
      fi
      ;;

    verify)
      if command -v ssl::verify_chain >/dev/null 2>&1; then
        ssl::verify_chain
      else
        log_error "SSL verification not available"
        return 1
      fi
      ;;

    help|--help|-h|*)
      printf "Usage: nself prod ssl <action>\n\n"
      printf "Actions:\n"
      printf "  status               Check current SSL certificate status\n"
      printf "  request <domain>     Request Let's Encrypt certificate\n"
      printf "  renew [--force]      Renew SSL certificates\n"
      printf "  self-signed [domain] Generate self-signed certificate\n"
      printf "  verify               Verify certificate chain\n"
      printf "\n"
      printf "Examples:\n"
      printf "  nself prod ssl status\n"
      printf "  nself prod ssl request example.com --email admin@example.com\n"
      printf "  nself prod ssl renew\n"
      ;;
  esac
}

# Manage firewall
prod_firewall() {
  local action="${1:-status}"
  shift 2>/dev/null || true

  show_command_header "nself prod firewall" "Firewall management"

  case "$action" in
    status)
      if command -v firewall::status >/dev/null 2>&1; then
        firewall::status
      else
        log_warning "Firewall status check requires security module"
        # Try basic detection
        if command -v ufw >/dev/null 2>&1; then
          printf "Firewall: UFW\n"
          ufw status 2>/dev/null || printf "Status: Unknown\n"
        elif command -v firewall-cmd >/dev/null 2>&1; then
          printf "Firewall: firewalld\n"
          firewall-cmd --state 2>/dev/null || printf "Status: Unknown\n"
        else
          log_warning "No firewall detected"
        fi
      fi
      ;;

    configure)
      local dry_run=""
      [[ "$1" == "--dry-run" ]] && dry_run="true"

      if command -v firewall::configure >/dev/null 2>&1; then
        firewall::configure "$dry_run"
      else
        log_error "Firewall configuration not available"
        printf "\nRecommended manual configuration:\n"
        printf "  sudo ufw default deny incoming\n"
        printf "  sudo ufw allow 22/tcp\n"
        printf "  sudo ufw allow 80/tcp\n"
        printf "  sudo ufw allow 443/tcp\n"
        printf "  sudo ufw enable\n"
        return 1
      fi
      ;;

    allow)
      local port="${1:-}"
      local protocol="${2:-tcp}"

      if [[ -z "$port" ]]; then
        log_error "Port is required"
        return 1
      fi

      if command -v firewall::allow_port >/dev/null 2>&1; then
        firewall::allow_port "$port" "$protocol"
      else
        log_info "To allow port $port:"
        printf "  sudo ufw allow %s/%s\n" "$port" "$protocol"
      fi
      ;;

    block)
      local port="${1:-}"

      if [[ -z "$port" ]]; then
        log_error "Port is required"
        return 1
      fi

      if command -v firewall::block_port >/dev/null 2>&1; then
        firewall::block_port "$port"
      else
        log_info "To block port $port:"
        printf "  sudo ufw deny %s\n" "$port"
      fi
      ;;

    recommendations)
      if command -v firewall::recommendations >/dev/null 2>&1; then
        firewall::recommendations
      else
        printf "Firewall Recommendations:\n\n"
        printf "1. Enable firewall and default deny incoming\n"
        printf "2. Only allow ports 22, 80, 443\n"
        printf "3. Use SSH key authentication\n"
        printf "4. Consider fail2ban for brute-force protection\n"
      fi
      ;;

    help|--help|-h|*)
      printf "Usage: nself prod firewall <action>\n\n"
      printf "Actions:\n"
      printf "  status               Check firewall status\n"
      printf "  configure [--dry-run] Configure recommended rules\n"
      printf "  allow <port>         Allow a port\n"
      printf "  block <port>         Block a port\n"
      printf "  recommendations      Show security recommendations\n"
      printf "\n"
      printf "Examples:\n"
      printf "  nself prod firewall status\n"
      printf "  nself prod firewall configure --dry-run\n"
      printf "  nself prod firewall allow 8080\n"
      ;;
  esac
}

# Apply all hardening measures
prod_harden() {
  local dry_run=""
  local skip_firewall=""

  while [[ $# -gt 0 ]]; do
    case $1 in
      --dry-run) dry_run="true"; shift ;;
      --skip-firewall) skip_firewall="true"; shift ;;
      *) shift ;;
    esac
  done

  show_command_header "nself prod harden" "Apply production hardening"

  if [[ "$dry_run" == "true" ]]; then
    log_warning "DRY RUN - No changes will be made"
    printf "\n"
  fi

  local steps_completed=0
  local steps_failed=0

  # Step 1: Generate secrets
  printf "${COLOR_CYAN}Step 1: Secrets${COLOR_RESET}\n"
  if [[ -f ".env.secrets" ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Secrets file exists\n"
    steps_completed=$((steps_completed + 1))
  else
    if [[ "$dry_run" != "true" ]]; then
      if prod_secrets generate >/dev/null 2>&1; then
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} Generated secrets\n"
        steps_completed=$((steps_completed + 1))
      else
        printf "  ${COLOR_RED}✗${COLOR_RESET} Failed to generate secrets\n"
        steps_failed=$((steps_failed + 1))
      fi
    else
      printf "  Would generate: .env.secrets\n"
    fi
  fi

  # Step 2: Environment settings
  printf "\n${COLOR_CYAN}Step 2: Environment Settings${COLOR_RESET}\n"
  local env_file=".env"
  [[ ! -f ".env" ]] && env_file=".env.dev"

  if [[ -f "$env_file" ]]; then
    local current_debug
    current_debug=$(grep "^DEBUG=" "$env_file" 2>/dev/null | cut -d'=' -f2)

    if [[ "$current_debug" == "false" ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} DEBUG is disabled\n"
    else
      if [[ "$dry_run" != "true" ]]; then
        set_env_var "DEBUG" "false" "$env_file"
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} Disabled DEBUG mode\n"
      else
        printf "  Would set: DEBUG=false\n"
      fi
    fi

    if [[ "$dry_run" != "true" ]]; then
      set_env_var "HASURA_GRAPHQL_DEV_MODE" "false" "$env_file"
      set_env_var "HASURA_GRAPHQL_ENABLE_CONSOLE" "false" "$env_file"
      set_env_var "LOG_LEVEL" "warning" "$env_file"
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} Applied production settings\n"
    else
      printf "  Would set: HASURA_GRAPHQL_DEV_MODE=false\n"
      printf "  Would set: HASURA_GRAPHQL_ENABLE_CONSOLE=false\n"
      printf "  Would set: LOG_LEVEL=warning\n"
    fi
    steps_completed=$((steps_completed + 1))
  else
    printf "  ${COLOR_RED}✗${COLOR_RESET} No .env file found\n"
    steps_failed=$((steps_failed + 1))
  fi

  # Step 3: SSL
  printf "\n${COLOR_CYAN}Step 3: SSL/TLS${COLOR_RESET}\n"
  if [[ -f "ssl/cert.pem" ]] && [[ -f "ssl/key.pem" ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} SSL certificates present\n"

    # Check key permissions
    local key_perms
    key_perms=$(safe_stat_perms "ssl/key.pem" 2>/dev/null || echo "unknown")
    if [[ "$key_perms" == "600" ]] || [[ "$key_perms" == "400" ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} SSL key permissions are secure\n"
    else
      if [[ "$dry_run" != "true" ]]; then
        chmod 600 ssl/key.pem
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} Fixed SSL key permissions\n"
      else
        printf "  Would fix: ssl/key.pem permissions to 600\n"
      fi
    fi
    steps_completed=$((steps_completed + 1))
  else
    printf "  ${COLOR_YELLOW}⚠${COLOR_RESET} SSL certificates not found\n"
    printf "    Generate with: nself prod ssl self-signed\n"
    printf "    Or request:    nself prod ssl request <domain>\n"
  fi

  # Step 4: Firewall
  if [[ "$skip_firewall" != "true" ]]; then
    printf "\n${COLOR_CYAN}Step 4: Firewall${COLOR_RESET}\n"
    if command -v firewall::detect >/dev/null 2>&1; then
      local fw_type
      fw_type=$(firewall::detect)
      if [[ "$fw_type" != "none" ]]; then
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} Firewall detected: %s\n" "$fw_type"
        if [[ "$dry_run" != "true" ]]; then
          printf "    Configure with: nself prod firewall configure\n"
        fi
      else
        printf "  ${COLOR_YELLOW}⚠${COLOR_RESET} No firewall detected\n"
      fi
    else
      printf "  ${COLOR_YELLOW}⚠${COLOR_RESET} Firewall detection not available\n"
    fi
  fi

  # Step 5: File permissions
  printf "\n${COLOR_CYAN}Step 5: File Permissions${COLOR_RESET}\n"
  local files_fixed=0

  for file in .env .env.secrets .env.local; do
    if [[ -f "$file" ]]; then
      local perms
      perms=$(safe_stat_perms "$file" 2>/dev/null || echo "unknown")
      if [[ "$perms" != "600" ]] && [[ "$perms" != "640" ]]; then
        if [[ "$dry_run" != "true" ]]; then
          chmod 600 "$file"
          files_fixed=$((files_fixed + 1))
        else
          printf "  Would fix: %s permissions to 600\n" "$file"
        fi
      fi
    fi
  done

  if [[ $files_fixed -gt 0 ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Fixed permissions on %d file(s)\n" "$files_fixed"
  else
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} File permissions are secure\n"
  fi
  steps_completed=$((steps_completed + 1))

  # Summary
  printf "\n"
  printf "${COLOR_CYAN}═══════════════════════════════════════${COLOR_RESET}\n"
  printf "${COLOR_CYAN}         Hardening Summary${COLOR_RESET}\n"
  printf "${COLOR_CYAN}═══════════════════════════════════════${COLOR_RESET}\n"
  printf "  Steps completed: %d\n" "$steps_completed"
  printf "  Steps failed:    %d\n" "$steps_failed"
  printf "\n"

  if [[ $steps_failed -eq 0 ]]; then
    log_success "Production hardening complete"
    printf "\nRun security audit: nself prod check\n"
  else
    log_warning "Hardening completed with issues"
    printf "\nReview and fix issues, then run: nself prod check\n"
  fi

  return $steps_failed
}

# Show production status
prod_status() {
  show_command_header "nself prod status" "Production environment status"

  # Check environment
  printf "${COLOR_CYAN}Environment${COLOR_RESET}\n"
  local env_setting
  env_setting=$(grep "^ENV=" .env 2>/dev/null | cut -d'=' -f2 || echo "not set")
  printf "  Environment: %s\n" "$env_setting"

  local domain
  domain=$(grep "^BASE_DOMAIN=" .env 2>/dev/null | cut -d'=' -f2 || echo "not set")
  printf "  Domain:      %s\n" "$domain"

  local debug
  debug=$(grep "^DEBUG=" .env 2>/dev/null | cut -d'=' -f2 || echo "not set")
  if [[ "$debug" == "true" ]]; then
    printf "  Debug:       ${COLOR_RED}%s${COLOR_RESET}\n" "$debug"
  else
    printf "  Debug:       ${COLOR_GREEN}%s${COLOR_RESET}\n" "$debug"
  fi

  # Check secrets
  printf "\n${COLOR_CYAN}Secrets${COLOR_RESET}\n"
  if [[ -f ".env.secrets" ]]; then
    local perms
    perms=$(safe_stat_perms ".env.secrets" 2>/dev/null || echo "?")
    printf "  Secrets file: ${COLOR_GREEN}exists${COLOR_RESET} (perms: %s)\n" "$perms"
  else
    printf "  Secrets file: ${COLOR_YELLOW}missing${COLOR_RESET}\n"
  fi

  # Check SSL
  printf "\n${COLOR_CYAN}SSL/TLS${COLOR_RESET}\n"
  local ssl_enabled
  ssl_enabled=$(grep "^SSL_ENABLED=" .env 2>/dev/null | cut -d'=' -f2 || echo "false")
  printf "  Enabled: %s\n" "$ssl_enabled"

  if [[ -f "ssl/cert.pem" ]]; then
    printf "  Certificate: ${COLOR_GREEN}present${COLOR_RESET}\n"
    if command -v openssl >/dev/null 2>&1; then
      local expiry
      expiry=$(openssl x509 -enddate -noout -in "ssl/cert.pem" 2>/dev/null | cut -d'=' -f2)
      printf "  Expires: %s\n" "${expiry:-unknown}"
    fi
  else
    printf "  Certificate: ${COLOR_YELLOW}missing${COLOR_RESET}\n"
  fi

  # Check compose files
  printf "\n${COLOR_CYAN}Docker Compose${COLOR_RESET}\n"
  if [[ -f "docker-compose.yml" ]]; then
    printf "  docker-compose.yml: ${COLOR_GREEN}present${COLOR_RESET}\n"
  else
    printf "  docker-compose.yml: ${COLOR_YELLOW}missing${COLOR_RESET}\n"
  fi

  if [[ -f "docker-compose.prod.yml" ]]; then
    printf "  docker-compose.prod.yml: ${COLOR_GREEN}present${COLOR_RESET}\n"
  else
    printf "  docker-compose.prod.yml: ${COLOR_YELLOW}missing${COLOR_RESET}\n"
  fi

  printf "\n"
  printf "Run ${COLOR_CYAN}nself prod check${COLOR_RESET} for full security audit\n"
}

# Create production compose override
create_prod_compose() {
  cat > docker-compose.prod.yml <<'EOF'
# Production Docker Compose Override
# Generated by nself prod init

services:
  nginx:
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./ssl:/etc/nginx/ssl:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./certbot-webroot:/var/www/certbot:ro
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  postgres:
    restart: always
    volumes:
      - postgres_data:/var/lib/postgresql/data
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  hasura:
    restart: always
    environment:
      HASURA_GRAPHQL_DEV_MODE: "false"
      HASURA_GRAPHQL_ENABLE_CONSOLE: "false"
      HASURA_GRAPHQL_LOG_LEVEL: "warn"
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  auth:
    restart: always
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  minio:
    restart: always
    volumes:
      - minio_data:/data
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  redis:
    restart: always
    volumes:
      - redis_data:/data
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  postgres_data:
  minio_data:
  redis_data:
EOF

  log_info "Created docker-compose.prod.yml"
}

# Show help
show_prod_help() {
  printf "Usage: nself prod <subcommand> [options]\n"
  printf "\n"
  printf "Production environment management and hardening\n"
  printf "\n"
  printf "Subcommands:\n"
  printf "  status               Show production status (default)\n"
  printf "  init <domain>        Initialize production configuration\n"
  printf "  check                Run security audit\n"
  printf "  secrets              Manage secrets (generate, rotate, validate)\n"
  printf "  ssl                  Manage SSL certificates\n"
  printf "  firewall             Configure firewall\n"
  printf "  harden               Apply all hardening measures\n"
  printf "\n"
  printf "Examples:\n"
  printf "  nself prod status\n"
  printf "  nself prod init example.com --email admin@example.com\n"
  printf "  nself prod secrets generate\n"
  printf "  nself prod ssl request example.com\n"
  printf "  nself prod firewall configure --dry-run\n"
  printf "  nself prod harden\n"
  printf "\n"
  printf "Subcommand help:\n"
  printf "  nself prod secrets --help\n"
  printf "  nself prod ssl --help\n"
  printf "  nself prod firewall --help\n"
}

# Export for use as library
export -f cmd_prod

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_prod "$@"
fi
