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
    
    # Check for inline comments and warn
    if grep -E '^\s*[A-Z_]+=[^#]*#' "$env_file" >/dev/null 2>&1; then
        log_warning "Inline comments detected in $env_file"
        log_info "Run 'nself validate-env --apply-fixes' to clean them"
    fi
    
    # Load environment variables
    set -a
    source "$env_file"
    set +a
    
    log_debug "Loaded environment from $env_file"
    return 0
}

# Load environment with proper priority
# Priority: .env > .env.local > .env.dev
load_env_with_priority() {
    local loaded=false
    
    # Check for .env first (highest priority - production)
    if [[ -f ".env" ]]; then
        log_debug "Loading .env (production mode)"
        set -a
        source ".env"
        set +a
        loaded=true
        # STOP HERE - ignore all other env files
        return 0
    fi
    
    # Check for .env.local next (development)
    if [[ -f ".env.local" ]]; then
        log_debug "Loading .env.local (development mode)"
        set -a
        source ".env.local"
        set +a
        loaded=true
        # STOP HERE - ignore .env.dev
        return 0
    fi
    
    # Check for .env.dev last (team defaults)
    if [[ -f ".env.dev" ]]; then
        log_debug "Loading .env.dev (team defaults)"
        set -a
        source ".env.dev"
        set +a
        loaded=true
        return 0
    fi
    
    if [[ "$loaded" == false ]]; then
        log_debug "No environment files found, using defaults only"
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
        echo "${var_name}=${value}" >> "$env_file"
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
    value="${value//\\/\\\\}"  # Escape backslashes
    value="${value//\"/\\\"}"  # Escape quotes
    value="${value//\$/\\\$}"  # Escape dollar signs
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
    sed -i.tmp 's/[[:space:]]*$//' "$env_file"  # Remove trailing whitespace
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
    
    cat > "$template_file" << 'EOF'
# NSELF Environment Configuration

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