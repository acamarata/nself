#!/usr/bin/env bash
# env.sh - Environment loading and management utilities

# Source display utilities
UTILS_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$UTILS_DIR/display.sh" 2>/dev/null || true

# Load environment file safely
load_env_safe() {
  local env_file="${1:-.env.local}"

  if [[ ! -f "$env_file" ]]; then
    log_error "Environment file not found: $env_file"
    return 1
  fi

  # Silently handle inline comments by loading through a cleaned temp file
  local temp_env=$(mktemp)

  # Remove inline comments while preserving the rest of the line
  # This regex keeps everything before the # that's not inside quotes
  sed 's/^\([^#]*[^[:space:]#]\)[[:space:]]*#.*$/\1/' "$env_file" >"$temp_env"

  # Load environment variables from cleaned file
  set -a
  source "$temp_env"
  set +a

  rm -f "$temp_env"

  # log_debug is not always available, so use conditional
  if declare -f log_debug >/dev/null 2>&1; then
    log_debug "Loaded environment from $env_file"
  fi
  return 0
}

# Load environment with proper priority
# New precedence system:
# 1. .env.secrets (always loaded if exists - for secrets only)
# 2. .env (if exists, skips all others except secrets)
# 3. .env.local (personal overrides)
# 4. .env.{ENV} (environment-specific: dev/staging/prod)
# 5. .env.dev (team defaults - baseline)
load_env_with_priority() {
  local loaded=false
  local current_env="${ENV:-dev}"
  
  # Always load secrets first if they exist (regardless of other files)
  if [[ -f ".env.secrets" ]]; then
    # Check file permissions for security
    local perms=$(stat -f "%OLp" ".env.secrets" 2>/dev/null || stat -c "%a" ".env.secrets" 2>/dev/null)
    if [[ -n "$perms" ]] && [[ "$perms" != "600" ]] && [[ "$perms" != "400" ]]; then
      if declare -f log_warning >/dev/null 2>&1; then
        log_warning ".env.secrets has permissions $perms (should be 600 or 400)"
        log_warning "Fix with: chmod 600 .env.secrets"
      fi
    fi
    
    # Use conditional logging
    if declare -f log_debug >/dev/null 2>&1; then
      log_debug "Loading .env.secrets (sensitive data)"
    fi
    set -a
    source ".env.secrets" 2>/dev/null
    set +a
  fi
  
  # Check for .env override (highest priority - usually production)
  if [[ -f ".env" ]]; then
    if declare -f log_debug >/dev/null 2>&1; then
      log_debug "Loading .env (override mode - ignoring other env files)"
    fi
    set -a
    source ".env"
    set +a
    return 0  # Stop here - .env overrides everything
  fi
  
  # Load in reverse order of precedence (so higher priority overrides)
  # Start with team defaults
  if [[ -f ".env.dev" ]]; then
    if declare -f log_debug >/dev/null 2>&1; then
      log_debug "Loading .env.dev (team defaults)"
    fi
    set -a
    source ".env.dev" 2>/dev/null
    set +a
    loaded=true
  fi
  
  # Load environment-specific file based on ENV variable
  local env_file=""
  case "$current_env" in
    dev|development)
      env_file=".env.dev"  # Already loaded above
      ;;
    staging|stage)
      env_file=".env.staging"
      ;;
    prod|production)
      env_file=".env.prod"
      ;;
    *)
      env_file=".env.${current_env}"  # Support custom environments
      ;;
  esac
  
  if [[ -n "$env_file" ]] && [[ -f "$env_file" ]] && [[ "$env_file" != ".env.dev" ]]; then
    if declare -f log_debug >/dev/null 2>&1; then
      log_debug "Loading $env_file (environment-specific)"
    fi
    set -a
    source "$env_file" 2>/dev/null
    set +a
    loaded=true
  fi
  
  # Load personal overrides last (highest priority after .env)
  if [[ -f ".env.local" ]]; then
    if declare -f log_debug >/dev/null 2>&1; then
      log_debug "Loading .env.local (personal overrides)"
    fi
    set -a
    source ".env.local" 2>/dev/null
    set +a
    loaded=true
  fi
  
  if [[ "$loaded" == false ]]; then
    if declare -f log_debug >/dev/null 2>&1; then
      log_debug "No environment files found, using defaults only"
    fi
  fi
  
  return 0
}

# Get environment variable with default
get_env_var() {
  local var_name="$1"
  local default_value="${2:-}"

  local value="${!var_name:-$default_value}"
  echo "$value"
}

# Set environment variable in file
set_env_var() {
  local var_name="$1"
  local value="$2"
  local env_file="${3:-.env.local}"

  if grep -q "^${var_name}=" "$env_file" 2>/dev/null; then
    # Update existing variable
    sed -i.bak "s|^${var_name}=.*|${var_name}=${value}|" "$env_file"
  else
    # Add new variable
    echo "${var_name}=${value}" >>"$env_file"
  fi
}

# Expand variables safely
expand_vars_safe() {
  local template="$1"
  local expanded

  # Use envsubst if available
  if command -v envsubst >/dev/null 2>&1; then
    expanded=$(echo "$template" | envsubst)
  else
    # Fallback to simple expansion
    expanded="$template"
    while [[ "$expanded" =~ \$\{([^}]+)\} ]]; do
      local var="${BASH_REMATCH[1]}"
      local val="${!var:-}"
      expanded="${expanded//\${$var\}/$val}"
    done
  fi

  echo "$expanded"
}

# Escape value for config files
escape_for_config() {
  local value="$1"
  # Escape special characters for safe inclusion in config files
  value="${value//\\/\\\\}" # Escape backslashes
  value="${value//\"/\\\"}" # Escape quotes
  value="${value//\$/\\\$}" # Escape dollar signs
  echo "$value"
}

# Clean environment file (remove inline comments)
clean_env_file() {
  local env_file="${1:-.env.local}"
  local backup_file="${2:-${env_file}.backup}"

  if [[ ! -f "$env_file" ]]; then
    log_error "Environment file not found: $env_file"
    return 1
  fi

  # Create backup
  cp "$env_file" "$backup_file"
  log_info "Backup created: $backup_file"

  # Remove inline comments
  sed -i.tmp 's/\([^#]*\)#.*/\1/' "$env_file"
  sed -i.tmp 's/[[:space:]]*$//' "$env_file" # Remove trailing whitespace
  rm -f "${env_file}.tmp"

  log_success "Cleaned inline comments from $env_file"
  return 0
}

# Validate required environment variables
validate_required_env() {
  local -a required_vars=("$@")
  local missing=()

  for var in "${required_vars[@]}"; do
    if [[ -z "${!var:-}" ]]; then
      missing+=("$var")
    fi
  done

  if [[ ${#missing[@]} -gt 0 ]]; then
    log_error "Missing required environment variables:"
    for var in "${missing[@]}"; do
      echo "  - $var"
    done
    return 1
  fi

  return 0
}

# Generate environment template
generate_env_template() {
  local template_file="${1:-.env.example}"

  cat >"$template_file" <<'EOF'
# nself Environment Configuration

# Project Settings
PROJECT_NAME=myproject
BASE_DOMAIN=localhost

# Database Configuration
POSTGRES_PASSWORD=changeme
POSTGRES_DB=myproject

# Hasura Configuration
HASURA_GRAPHQL_ADMIN_SECRET=changeme
HASURA_GRAPHQL_JWT_SECRET='{"type":"HS256","key":"changeme-32-character-secret-key"}'

# Authentication
JWT_SECRET=changeme-32-character-secret-key
COOKIE_SECRET=changeme-32-character-secret-key

# Storage Configuration
S3_ACCESS_KEY=minioaccesskey
S3_SECRET_KEY=miniosecretkey
S3_BUCKET=myproject

# Email Configuration (optional)
EMAIL_PROVIDER=development
SMTP_HOST=
SMTP_PORT=
SMTP_USER=
SMTP_PASS=
SMTP_FROM=

# Redis Configuration (optional)
REDIS_ENABLED=false
REDIS_PASSWORD=

# SSL Configuration
SSL_ENABLED=false
EOF

  log_success "Generated environment template: $template_file"
}

# Export all functions
export -f load_env_safe get_env_var set_env_var expand_vars_safe
export -f escape_for_config clean_env_file validate_required_env
export -f generate_env_template
