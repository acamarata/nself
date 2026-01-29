#!/usr/bin/env bash
# secrets.sh - Production secrets management
# Generate, validate, and rotate secrets for nself deployments

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
# Secrets Command - Production Secrets Management
# ============================================================

cmd_secrets() {
  local subcommand="${1:-help}"
  shift || true

  case "$subcommand" in
    generate)
      secrets_generate "$@"
      ;;
    validate)
      secrets_validate "$@"
      ;;
    rotate)
      secrets_rotate "$@"
      ;;
    show)
      secrets_show "$@"
      ;;
    help|--help|-h)
      show_secrets_help
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      show_secrets_help
      return 1
      ;;
  esac
}

# ============================================================
# Generate Secrets
# ============================================================

secrets_generate() {
  local env_name=""
  local force="false"
  local output=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --env|-e)
        env_name="$2"
        shift 2
        ;;
      --force|-f)
        force="true"
        shift
        ;;
      --output|-o)
        output="$2"
        shift 2
        ;;
      *)
        if [[ -z "$env_name" ]]; then
          env_name="$1"
        fi
        shift
        ;;
    esac
  done

  show_command_header "nself secrets generate" "Generate secure production secrets"

  # Determine environment directory
  local env_dir=""
  if [[ -n "$env_name" ]]; then
    env_dir=".environments/$env_name"
  else
    env_dir="."
    env_name="local"
  fi

  # Check if secrets file already exists
  local secrets_file="$env_dir/.env.secrets"
  if [[ -n "$output" ]]; then
    secrets_file="$output"
  fi

  if [[ -f "$secrets_file" ]] && [[ "$force" != "true" ]]; then
    log_warning "Secrets file already exists: $secrets_file"
    printf "\n"
    printf "Options:\n"
    printf "  1. Use --force to overwrite existing secrets\n"
    printf "  2. Use 'nself secrets rotate' to update specific secrets\n"
    printf "\n"
    printf "Overwrite existing secrets? [y/N]: "
    read -r confirm
    confirm=$(printf "%s" "$confirm" | tr '[:upper:]' '[:lower:]')
    if [[ "$confirm" != "y" ]]; then
      log_info "Cancelled"
      return 0
    fi
  fi

  # Create directory if needed
  mkdir -p "$(dirname "$secrets_file")"

  printf "\nGenerating cryptographically secure secrets...\n\n"

  # Generate all secrets
  local postgres_pw=$(generate_password 44)
  local hasura_secret=$(generate_secret 64)
  local jwt_secret=$(generate_secret 64)
  local auth_secret=$(generate_secret 64)
  local redis_pw=$(generate_password 44)
  local minio_pw=$(generate_password 44)
  local meili_key=$(generate_secret 44)
  local grafana_pw=$(generate_password 32)
  local encryption_key=$(generate_secret 32)

  # Write secrets file
  cat > "$secrets_file" << EOF
# ============================================================
# nself Production Secrets
# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
# Environment: $env_name
# ============================================================
#
# IMPORTANT SECURITY NOTES:
#   - NEVER commit this file to version control
#   - Set permissions: chmod 600 $secrets_file
#   - Store securely (password manager, vault, etc.)
#   - Rotate regularly (at least every 90 days)
#
# ============================================================

# Database
POSTGRES_PASSWORD=$postgres_pw

# Hasura GraphQL Engine
HASURA_GRAPHQL_ADMIN_SECRET=$hasura_secret

# Authentication Service
AUTH_JWT_SECRET=$jwt_secret
AUTH_SECRET_KEY=$auth_secret

# Redis (if REDIS_ENABLED=true)
REDIS_PASSWORD=$redis_pw

# MinIO Storage (if MINIO_ENABLED=true)
MINIO_ROOT_PASSWORD=$minio_pw

# MeiliSearch (if MEILISEARCH_ENABLED=true)
MEILISEARCH_MASTER_KEY=$meili_key

# Grafana (if MONITORING_ENABLED=true)
GRAFANA_ADMIN_PASSWORD=$grafana_pw

# Encryption key for sensitive data
ENCRYPTION_KEY=$encryption_key
EOF

  # Set secure permissions
  chmod 600 "$secrets_file"

  # Show what was generated
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} POSTGRES_PASSWORD           (%d chars)\n" "${#postgres_pw}"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} HASURA_GRAPHQL_ADMIN_SECRET (%d chars)\n" "${#hasura_secret}"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} AUTH_JWT_SECRET             (%d chars)\n" "${#jwt_secret}"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} AUTH_SECRET_KEY             (%d chars)\n" "${#auth_secret}"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} REDIS_PASSWORD              (%d chars)\n" "${#redis_pw}"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} MINIO_ROOT_PASSWORD         (%d chars)\n" "${#minio_pw}"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} MEILISEARCH_MASTER_KEY      (%d chars)\n" "${#meili_key}"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} GRAFANA_ADMIN_PASSWORD      (%d chars)\n" "${#grafana_pw}"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} ENCRYPTION_KEY              (%d chars)\n" "${#encryption_key}"

  printf "\n"
  log_success "Secrets saved to: $secrets_file"
  printf "  Permissions: 600 (owner read/write only)\n"

  printf "\n"
  printf "Next steps:\n"
  printf "  1. Review secrets:       ${COLOR_CYAN}cat %s${COLOR_RESET}\n" "$secrets_file"
  printf "  2. Validate deployment:  ${COLOR_CYAN}nself validate %s${COLOR_RESET}\n" "$env_name"
  printf "  3. Deploy:               ${COLOR_CYAN}nself deploy %s${COLOR_RESET}\n" "$env_name"
  printf "\n"

  # Add to .gitignore if not present
  add_to_gitignore ".env.secrets"
  add_to_gitignore ".environments/*/.env.secrets"
}

# ============================================================
# Validate Secrets
# ============================================================

secrets_validate() {
  local env_name="${1:-}"

  show_command_header "nself secrets validate" "Validate secrets configuration"

  local env_dir="."
  if [[ -n "$env_name" ]]; then
    env_dir=".environments/$env_name"
  fi

  # Load environment
  if [[ -f "$env_dir/.env" ]]; then
    set -a
    source "$env_dir/.env" 2>/dev/null || true
    set +a
  fi

  if [[ -f "$env_dir/.env.secrets" ]]; then
    set -a
    source "$env_dir/.env.secrets" 2>/dev/null || true
    set +a
  fi

  # Use security preflight for validation
  if command -v security::preflight >/dev/null 2>&1; then
    security::preflight "$env_name" "$env_dir" "false"
    return $?
  else
    # Manual validation
    local errors=0

    printf "Checking secrets...\n\n"

    # Check required secrets
    local secrets=(
      "POSTGRES_PASSWORD"
      "HASURA_GRAPHQL_ADMIN_SECRET"
      "AUTH_JWT_SECRET"
    )

    for secret in "${secrets[@]}"; do
      local value="${!secret:-}"
      if [[ -z "$value" ]]; then
        printf "  ${COLOR_RED}✗${COLOR_RESET} %s: Not set\n" "$secret"
        errors=$((errors + 1))
      elif [[ ${#value} -lt 16 ]]; then
        printf "  ${COLOR_YELLOW}!${COLOR_RESET} %s: Weak (%d chars)\n" "$secret" "${#value}"
      else
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} %s: OK (%d chars)\n" "$secret" "${#value}"
      fi
    done

    if [[ $errors -eq 0 ]]; then
      printf "\n"
      log_success "All secrets validated"
      return 0
    else
      printf "\n"
      log_error "$errors secret(s) missing or invalid"
      return 1
    fi
  fi
}

# ============================================================
# Rotate Secrets
# ============================================================

secrets_rotate() {
  local env_name=""
  local secret_name=""
  local all="false"

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --env|-e)
        env_name="$2"
        shift 2
        ;;
      --all)
        all="true"
        shift
        ;;
      *)
        secret_name="$1"
        shift
        ;;
    esac
  done

  show_command_header "nself secrets rotate" "Rotate production secrets"

  if [[ "$all" == "true" ]]; then
    log_warning "Rotating ALL secrets will require restarting all services"
    printf "\n"
    printf "This will:\n"
    printf "  1. Generate new secrets\n"
    printf "  2. Update .env.secrets file\n"
    printf "  3. Require deployment to apply changes\n"
    printf "\n"
    printf "Continue? [y/N]: "
    read -r confirm
    confirm=$(printf "%s" "$confirm" | tr '[:upper:]' '[:lower:]')
    if [[ "$confirm" != "y" ]]; then
      log_info "Cancelled"
      return 0
    fi

    secrets_generate --env "$env_name" --force
    return $?
  fi

  if [[ -z "$secret_name" ]]; then
    printf "Specify a secret to rotate:\n"
    printf "  nself secrets rotate POSTGRES_PASSWORD --env prod\n"
    printf "  nself secrets rotate --all --env prod\n"
    return 1
  fi

  # Rotate single secret
  local env_dir="."
  if [[ -n "$env_name" ]]; then
    env_dir=".environments/$env_name"
  fi

  local secrets_file="$env_dir/.env.secrets"
  if [[ ! -f "$secrets_file" ]]; then
    log_error "Secrets file not found: $secrets_file"
    return 1
  fi

  # Generate new value
  local new_value=""
  case "$secret_name" in
    *PASSWORD*)
      new_value=$(generate_password 44)
      ;;
    *)
      new_value=$(generate_secret 64)
      ;;
  esac

  # Update secret in file
  if grep -q "^${secret_name}=" "$secrets_file" 2>/dev/null; then
    # Use platform-safe sed
    if [[ "$OSTYPE" == "darwin"* ]]; then
      sed -i '' "s|^${secret_name}=.*|${secret_name}=${new_value}|" "$secrets_file"
    else
      sed -i "s|^${secret_name}=.*|${secret_name}=${new_value}|" "$secrets_file"
    fi
    log_success "Rotated $secret_name"
  else
    # Add new secret
    printf "%s=%s\n" "$secret_name" "$new_value" >> "$secrets_file"
    log_success "Added $secret_name"
  fi

  printf "\n"
  printf "Next steps:\n"
  printf "  1. Deploy to apply changes: ${COLOR_CYAN}nself deploy %s${COLOR_RESET}\n" "${env_name:-prod}"
  printf "  2. Restart affected services if needed\n"
}

# ============================================================
# Show Secrets (masked)
# ============================================================

secrets_show() {
  local env_name="${1:-}"
  local unmask="false"

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --unmask)
        unmask="true"
        shift
        ;;
      *)
        if [[ -z "$env_name" ]]; then
          env_name="$1"
        fi
        shift
        ;;
    esac
  done

  show_command_header "nself secrets show" "Display configured secrets"

  local env_dir="."
  if [[ -n "$env_name" ]]; then
    env_dir=".environments/$env_name"
  fi

  local secrets_file="$env_dir/.env.secrets"
  if [[ ! -f "$secrets_file" ]]; then
    log_error "Secrets file not found: $secrets_file"
    printf "\n"
    printf "Generate secrets with: ${COLOR_CYAN}nself secrets generate --env %s${COLOR_RESET}\n" "${env_name:-prod}"
    return 1
  fi

  printf "Secrets file: %s\n\n" "$secrets_file"

  # Read and display secrets
  while IFS= read -r line; do
    # Skip comments and empty lines
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "$line" ]] && continue

    local var_name="${line%%=*}"
    local var_value="${line#*=}"

    if [[ "$unmask" == "true" ]]; then
      printf "  %s=%s\n" "$var_name" "$var_value"
    else
      # Mask the value
      local length=${#var_value}
      local masked="${var_value:0:4}$( printf '%*s' $((length - 8)) | tr ' ' '*' )${var_value: -4}"
      printf "  %s=%s (%d chars)\n" "$var_name" "$masked" "$length"
    fi
  done < "$secrets_file"

  printf "\n"
  if [[ "$unmask" != "true" ]]; then
    printf "${COLOR_DIM}Use --unmask to show actual values${COLOR_RESET}\n"
  fi
}

# ============================================================
# Helper Functions
# ============================================================

generate_password() {
  local length="${1:-32}"
  openssl rand -base64 "$((length * 2))" | tr -d '/+=' | head -c "$length"
}

generate_secret() {
  local length="${1:-64}"
  openssl rand -base64 "$((length * 2))" | tr -d '/+=' | head -c "$length"
}

add_to_gitignore() {
  local pattern="$1"
  local gitignore=".gitignore"

  if [[ -f "$gitignore" ]]; then
    if ! grep -q "^${pattern}$" "$gitignore" 2>/dev/null; then
      printf "\n# Secrets (auto-added by nself)\n%s\n" "$pattern" >> "$gitignore"
    fi
  fi
}

# ============================================================
# Help
# ============================================================

show_secrets_help() {
  printf "Usage: nself secrets <command> [options]\n"
  printf "\n"
  printf "Production secrets management\n"
  printf "\n"
  printf "Commands:\n"
  printf "  generate     Generate secure production secrets\n"
  printf "  validate     Validate secrets configuration\n"
  printf "  rotate       Rotate secrets (single or all)\n"
  printf "  show         Display configured secrets (masked)\n"
  printf "\n"
  printf "Options:\n"
  printf "  --env, -e    Environment name (default: current directory)\n"
  printf "  --force, -f  Overwrite existing secrets\n"
  printf "  --output, -o Output file path\n"
  printf "  --all        Rotate all secrets\n"
  printf "  --unmask     Show actual secret values\n"
  printf "\n"
  printf "Examples:\n"
  printf "  nself secrets generate --env prod     # Generate for production\n"
  printf "  nself secrets validate --env staging  # Validate staging secrets\n"
  printf "  nself secrets rotate POSTGRES_PASSWORD --env prod\n"
  printf "  nself secrets rotate --all --env prod # Rotate all secrets\n"
  printf "  nself secrets show --env prod         # Show masked secrets\n"
  printf "  nself secrets show --env prod --unmask # Show actual values\n"
  printf "\n"
  printf "Security best practices:\n"
  printf "  • Generate secrets at least 32 characters long\n"
  printf "  • Never commit .env.secrets to version control\n"
  printf "  • Rotate secrets regularly (every 90 days)\n"
  printf "  • Use different secrets for each environment\n"
  printf "  • Store secrets in a password manager or vault\n"
}

# Export function
export -f cmd_secrets

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "secrets" 2>/dev/null || true
  cmd_secrets "$@"
  exit_code=$?
  post_command "secrets" $exit_code 2>/dev/null || true
  exit $exit_code
fi
