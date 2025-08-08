#!/bin/bash

# nself.sh - Main CLI tool for managing self-hosted Nhost stack

set -e

# ----------------------------
# Resolve Script Directory
# ----------------------------
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do
  DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
SCRIPT_DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"
TEMPLATES_DIR="$SCRIPT_DIR/templates"

# ----------------------------
# Variables
# ----------------------------
VERSION_FILE="$SCRIPT_DIR/VERSION"
REPO_RAW_URL="https://raw.githubusercontent.com/acamarata/nself/main"
LOCAL_VERSION=""
LATEST_VERSION=""

# ----------------------------
# Helper Functions
# ----------------------------

# Function to print colored messages
echo_info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

echo_success() {
  echo -e "\033[1;32m[SUCCESS]\033[0m $1"
}

echo_warning() {
  echo -e "\033[1;33m[WARNING]\033[0m $1"
}

echo_error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

# Progress spinner
show_spinner() {
  local pid=$1
  local message=$2
  local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  local i=0
  
  printf "\033[1;36m%s\033[0m" "$message"
  
  while kill -0 "$pid" 2>/dev/null; do
    i=$(( (i+1) %10 ))
    printf "\r\033[1;36m%s %s\033[0m" "$message" "${spin:$i:1}"
    sleep 0.1
  done
  
  wait "$pid"
  local result=$?
  
  if [ $result -eq 0 ]; then
    printf "\r\033[1;32m✓\033[0m %s\n" "$message"
  else
    printf "\r\033[1;31m✗\033[0m %s\n" "$message"
  fi
  
  return $result
}

# Clean line output
echo_clean() {
  printf "\r\033[K%s\n" "$1"
}

# Function to check if a command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Function to read local version
read_local_version() {
  if [ -f "$VERSION_FILE" ]; then
    LOCAL_VERSION=$(tr -d '[:space:]' < "$VERSION_FILE")
  else
    echo_error "VERSION file not found."
    exit 1
  fi
}

# Function to fetch latest version from GitHub
fetch_latest_version() {
  LATEST_VERSION=$(curl -fsSL "$REPO_RAW_URL/bin/VERSION" 2>/dev/null || echo "")
  LATEST_VERSION=$(echo "$LATEST_VERSION" | tr -d '[:space:]')
}

# Function to compare versions
is_newer_version() {
  local_ver=${LOCAL_VERSION#v}
  latest_ver=${LATEST_VERSION#v}

  IFS='.' read -r -a local_parts <<< "$local_ver"
  IFS='.' read -r -a latest_parts <<< "$latest_ver"

  for i in 0 1 2; do
    local_part=${local_parts[i]:-0}
    latest_part=${latest_parts[i]:-0}
    if (( 10#$latest_part > 10#$local_part )); then
      return 0
    elif (( 10#$latest_part < 10#$local_part )); then
      return 1
    fi
  done

  return 1
}

# Function to check for updates
check_for_updates() {
  read_local_version
  fetch_latest_version

  if [ -n "$LATEST_VERSION" ] && is_newer_version; then
    echo_info "New version available: $LATEST_VERSION (current: $LOCAL_VERSION)"
    echo_info "Run 'nself update' to upgrade"
  fi
}

# Function to check for pending database migrations
check_for_migrations() {
  # Only check if we have migration directories
  if [ ! -d "hasura/migrations/default" ]; then
    return 0
  fi
  
  # Count migration directories
  local migration_count
  migration_count=$(find hasura/migrations/default -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l)
  
  if [ "$migration_count" -eq 0 ]; then
    return 0
  fi
  
  # Try to check if database has tables (basic check for applied migrations)
  local table_count=0
  if docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres" 2>/dev/null; then
    # Wait a moment for postgres to be ready
    sleep 2
    table_count=$(docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -d "${POSTGRES_DB:-nhost}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'" 2>/dev/null || echo "0")
    table_count=$(echo "$table_count" | tr -d ' ')
  fi
  
  # If we have migrations but very few tables, migrations are likely pending
  if [ "$migration_count" -gt 0 ] && [ "$table_count" -lt 5 ]; then
    echo ""
    echo_warning "═══════════════════════════════════════════════════════════════"
    echo_warning "  DATABASE MIGRATIONS PENDING"
    echo_warning "  Found $migration_count migration(s) that may need to be applied"
    echo_warning "  Run 'nself db update' to bring your database up to date"
    echo_warning "═══════════════════════════════════════════════════════════════"
    echo ""
  fi
}

# Function to load environment variables
load_env() {
  # Check for .env override (production)
  if [ -f ".env" ] && [ -f ".env.local" ]; then
    echo_info "Found both .env and .env.local - .env will override .env.local settings"
    ENV_FILE=".env"
  elif [ -f ".env.local" ]; then
    ENV_FILE=".env.local"
  elif [ -f ".env" ]; then
    ENV_FILE=".env"
  else
    echo_error "No .env.local or .env file found."
    echo_info "Run 'nself init' to initialize a new project."
    exit 1
  fi

  # Load .env.local first if both exist
  if [ -f ".env.local" ] && [ "$ENV_FILE" = ".env" ]; then
    set -o allexport
    source ".env.local"
    set +o allexport
  fi
  
  # Load primary env file
  set -o allexport
  source "$ENV_FILE"
  set +o allexport

  export PROJECT_NAME=${PROJECT_NAME:-myproject}
  export ENVIRONMENT=${ENVIRONMENT:-development}
  export BASE_DOMAIN=${BASE_DOMAIN:-local.nself.org}
  
  # Production safety checks
  if [[ "$ENVIRONMENT" == "production" ]]; then
    echo_warning "Running in PRODUCTION mode"
    
    # Check for insecure defaults and weak passwords
    if [[ "$POSTGRES_PASSWORD" == "secretpassword" ]] || [[ "${#POSTGRES_PASSWORD}" -lt 12 ]]; then
      echo_error "Weak PostgreSQL password detected in production!"
      echo_error "Please set a secure POSTGRES_PASSWORD (min 12 chars) in your .env file"
      exit 1
    fi
    
    if [[ "$HASURA_GRAPHQL_ADMIN_SECRET" == "hasura-admin-secret" ]] || [[ "${#HASURA_GRAPHQL_ADMIN_SECRET}" -lt 12 ]]; then
      echo_error "Weak Hasura admin secret detected in production!"
      echo_error "Please set a secure HASURA_GRAPHQL_ADMIN_SECRET (min 12 chars) in your .env file"
      exit 1
    fi
    
    # Check JWT secret
    if [[ "$HASURA_GRAPHQL_JWT_SECRET" == *"CHANGE-THIS"* ]]; then
      echo_error "Default JWT secret detected in production!"
      echo_error "Please generate a secure JWT secret with: openssl rand -hex 32"
      exit 1
    fi
    
    if [[ "$SSL_MODE" == "none" ]] || [[ "$SSL_MODE" == "local" && "$BASE_DOMAIN" != *"nself.org" ]]; then
      echo_error "Insecure SSL configuration for production!"
      echo_error "Please use 'letsencrypt' or 'custom' SSL mode"
      exit 1
    fi
  fi
}

# ----------------------------
# Command Functions
# ----------------------------

# Initialize project
cmd_init() {
  if [ -f ".env.local" ]; then
    echo_error ".env.local already exists in this directory."
    exit 1
  fi

  if [ ! -f "$TEMPLATES_DIR/.env.example" ]; then
    echo_error "Template .env.example not found."
    exit 1
  fi

  echo_info "Initializing nself project..."
  
  # Copy env template
  cp "$TEMPLATES_DIR/.env.example" ".env.local"
  
  echo_success "Project initialized!"
  echo_info "Please edit .env.local to configure your project"
  echo_info "Then run 'nself build' to generate the project structure"
}

# Build project structure
cmd_build() {
  if [ ! -f ".env.local" ]; then
    echo_error "No .env.local file found. Run 'nself init' first."
    exit 1
  fi

  load_env
  
  echo ""
  echo_info "🔨 Building project: $PROJECT_NAME"
  echo ""

  # Run build script in background
  (bash "$SCRIPT_DIR/build.sh" > /tmp/nself_build.log 2>&1) &
  local build_pid=$!
  
  show_spinner $build_pid "  Generating Docker configuration"
  
  # Check if build was successful
  if [ $? -eq 0 ]; then
    echo_success ""
    echo_success "  ✅ Project structure created!"
    echo ""
    echo_info "  Generated files:"
    echo_info "  • docker-compose.yml"
    echo_info "  • nginx/ (proxy configuration)"
    echo_info "  • hasura/ (GraphQL engine)"
    echo_info "  • postgres/ (database init)"
    if [ "$SERVICES_ENABLED" == "true" ]; then
      echo_info "  • microservices/ (backend services)"
    fi
    echo ""
    echo_info "  Next: Run 'nself up' to start services"
  else
    echo_error "Build failed. Check /tmp/nself_build.log for details."
    exit 1
  fi
}

# Start services
cmd_up() {
  if [ ! -f ".env.local" ]; then
    echo_error "No .env.local file found. Run 'nself init' first."
    exit 1
  fi

  load_env
  
  # nself up only applies existing migrations, no DBML sync here
  # Use 'nself dbsync' to sync schema and generate migrations

  if [ ! -f "docker-compose.yml" ]; then
    echo_info "docker-compose.yml not found. Running build first..."
    cmd_build
  fi

  echo ""
  echo_info "🚀 Starting services..."
  echo ""
  
  # Start all services in background
  (docker compose up -d > /tmp/nself_up.log 2>&1) &
  show_spinner $! "  Starting Docker containers"
  
  if [ $? -eq 0 ]; then
    # Give services a moment to stabilize
    sleep 2
    bash "$SCRIPT_DIR/success.sh"
    
    # Show additional service info if enabled
    if [[ "$SERVICES_ENABLED" == "true" ]]; then
      echo_info "Backend services started:"
      [[ "$NESTJS_ENABLED" == "true" ]] && echo_info "  - NestJS services on ports starting at $NESTJS_PORT_START"
      [[ "$BULLMQ_ENABLED" == "true" ]] && echo_info "  - BullMQ dashboard: http://localhost:${BULLMQ_DASHBOARD_PORT:-3200}"
      [[ "$GOLANG_ENABLED" == "true" ]] && echo_info "  - GoLang services on ports starting at $GOLANG_PORT_START"
      [[ "$PYTHON_ENABLED" == "true" ]] && echo_info "  - Python services on ports starting at $PYTHON_PORT_START"
    fi
    
    # Check for pending migrations after services are up
    check_for_migrations
  else
    echo_error "Failed to start services."
    exit 1
  fi
}

# Stop services
cmd_down() {
  if [ ! -f ".env.local" ]; then
    echo_error "No .env.local file found."
    exit 1
  fi

  load_env

  echo ""
  echo_info "🛑 Stopping services..."
  echo ""
  
  # Stop all services
  (docker compose down > /tmp/nself_down.log 2>&1) &
  show_spinner $! "  Stopping Docker containers"
  
  if [ $? -eq 0 ]; then
    echo ""
    echo_success "  ✅ All services stopped"
    echo ""
  else
    echo_error "Failed to stop services. Check /tmp/nself_down.log for details."
    exit 1
  fi
}

# Reset project
cmd_reset() {
  if [ ! -f ".env.local" ] && [ ! -f ".env" ]; then
    echo_error "No project to reset."
    exit 1
  fi

  echo ""
  echo_warning "════════════════════════════════════════════════════"
  echo_warning "  ⚠️  COMPLETE PROJECT RESET"
  echo_warning "════════════════════════════════════════════════════"
  echo_warning ""
  echo_warning "  This will DELETE:"
  echo_warning "  • All Docker containers for project: ${PROJECT_NAME:-myproject}"
  echo_warning "  • All Docker volumes and stored data"
  echo_warning "  • All Docker networks"
  echo_warning "  • All generated configuration files"
  echo_warning "  • Your current .env.local (backed up to .env.old)"
  echo_warning ""
  echo_warning "════════════════════════════════════════════════════"
  echo ""
  read -p "Type 'RESET' to confirm total deletion: " -r
  echo
  
  if [[ ! $REPLY == "RESET" ]]; then
    echo_info "Reset cancelled."
    exit 0
  fi

  # Load environment if available
  if [ -f ".env.local" ] || [ -f ".env" ]; then
    load_env
  fi
  
  local project="${PROJECT_NAME:-myproject}"
  
  echo ""
  echo_info "Starting complete reset for project: $project"
  echo ""
  
  # Stop and remove all Docker resources for this project
  echo_info "🐳 Cleaning Docker resources..."
  
  # Stop containers
  (docker compose down -v 2>/dev/null || true) &
  show_spinner $! "  Stopping containers"
  
  # Remove project-specific containers
  local containers=$(docker ps -a --filter "name=${project}" --format "{{.Names}}" 2>/dev/null)
  if [ -n "$containers" ]; then
    (echo "$containers" | xargs -r docker rm -f 2>/dev/null || true) &
    show_spinner $! "  Removing containers"
  fi
  
  # Remove project-specific volumes
  local volumes=$(docker volume ls --filter "name=${project}" --format "{{.Name}}" 2>/dev/null)
  if [ -n "$volumes" ]; then
    (echo "$volumes" | xargs -r docker volume rm -f 2>/dev/null || true) &
    show_spinner $! "  Removing volumes"
  fi
  
  # Remove project-specific networks
  local networks=$(docker network ls --filter "name=${project}" --format "{{.Name}}" 2>/dev/null)
  if [ -n "$networks" ]; then
    (echo "$networks" | xargs -r docker network rm 2>/dev/null || true) &
    show_spinner $! "  Removing networks"
  fi
  
  echo ""
  echo_info "📁 Cleaning project files..."
  
  # Remove generated files and directories
  (
    rm -rf docker-compose.yml 2>/dev/null
    rm -rf nginx/ 2>/dev/null
    rm -rf hasura/ 2>/dev/null
    rm -rf postgres/ 2>/dev/null
    rm -rf functions/ 2>/dev/null
    rm -rf microservices/ 2>/dev/null
    rm -rf certs/ 2>/dev/null
    rm -rf seeds/ 2>/dev/null
    rm -rf bin/dbsyncs/ 2>/dev/null
    rm -rf .nself/ 2>/dev/null
    rm -f schema.dbml 2>/dev/null
    rm -f schema.dbml.backup 2>/dev/null
    rm -f dbml-error.log 2>/dev/null
  ) &
  show_spinner $! "  Removing generated files"
  
  # Backup and replace .env.local
  if [ -f ".env.local" ]; then
    (mv .env.local .env.old 2>/dev/null) &
    show_spinner $! "  Backing up .env.local to .env.old"
  fi
  
  if [ -f ".env" ]; then
    (mv .env .env.old 2>/dev/null) &
    show_spinner $! "  Backing up .env to .env.old"
  fi
  
  # Create fresh .env.local from template
  if [ -f "$TEMPLATES_DIR/.env.example" ]; then
    (cp "$TEMPLATES_DIR/.env.example" .env.local) &
    show_spinner $! "  Creating fresh .env.local from template"
  fi
  
  echo ""
  echo_success "════════════════════════════════════════════════════"
  echo_success "  ✅ Project Reset Complete!"
  echo_success "════════════════════════════════════════════════════"
  echo ""
  echo_info "  Next steps:"
  echo_info "  1. Edit .env.local to configure your project"
  echo_info "  2. Run 'nself build' to generate structure"
  echo_info "  3. Run 'nself up' to start services"
  echo ""
  echo_info "  Your previous configuration was saved to: .env.old"
  echo ""
}

# Create production environment file
cmd_prod() {
  if [ ! -f ".env.local" ]; then
    echo_error "No .env.local file found. Run 'nself init' first."
    exit 1
  fi

  echo_info "Creating production .env file from .env.local..."
  
  # Copy .env.local to .env
  cp .env.local .env
  
  # Generate secure secrets
  echo_info "🔐 Generating cryptographically secure random passwords, hashes, and secrets..."
  echo_info "These are uniquely generated for your deployment - not pre-set values!"
  
  # Function to generate random password
  generate_password() {
    openssl rand -hex 32 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1
  }
  
  # Generate all required secrets
  POSTGRES_PASSWORD_GENERATED=$(generate_password)
  HASURA_SECRET_GENERATED=$(generate_password)
  MINIO_PASSWORD_GENERATED=$(generate_password)
  S3_SECRET_GENERATED=$(generate_password)
  JWT_KEY_GENERATED=$(generate_password)
  REDIS_PASSWORD_GENERATED=$(openssl rand -hex 16 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 16 | head -n 1)
  
  # Create a minimal production override template
  cat > .env.prod-template << EOF
# ============================================
# nself Production Environment Configuration
# ============================================
# This file contains ONLY the values you need to override for production
# The system will use .env.local as the base and apply these overrides
# 
# IMPORTANT: Delete any lines you don't need to change!
# Only keep the specific overrides required for your production environment
# ============================================

# ============================================
# CORE SETTINGS (REQUIRED)
# ============================================

# Set to production mode (enables security checks)
ENVIRONMENT=production

# Your production domain (without https://)
# Example: api.mycompany.com or backend.myapp.io
BASE_DOMAIN=yourdomain.com

# ============================================
# SECURITY & PASSWORDS (REQUIRED - MUST CHANGE ALL)
# ============================================

# PostgreSQL Database Password
# GENERATED SECURE PASSWORD - Keep this safe!
POSTGRES_PASSWORD=${POSTGRES_PASSWORD_GENERATED}

# Hasura Admin Secret (access to GraphQL admin APIs)
# GENERATED SECURE SECRET - Keep this safe!
HASURA_GRAPHQL_ADMIN_SECRET=${HASURA_SECRET_GENERATED}

# MinIO/S3 Storage Credentials
# GENERATED SECURE PASSWORDS - Keep these safe!
MINIO_ROOT_PASSWORD=${MINIO_PASSWORD_GENERATED}
S3_SECRET_KEY=${S3_SECRET_GENERATED}

# JWT Secret for Authentication
# GENERATED SECURE KEY - Keep this safe!
HASURA_GRAPHQL_JWT_SECRET='{"type":"HS256","key":"${JWT_KEY_GENERATED}"}'

# Redis Password (if using Redis)
# GENERATED SECURE PASSWORD - Uncomment if using Redis
# REDIS_PASSWORD=${REDIS_PASSWORD_GENERATED}

# ============================================
# SSL/TLS CONFIGURATION (REQUIRED)
# ============================================

# SSL Mode Options:
# - letsencrypt: Free automatic SSL certificates
# - custom: Use your own certificates
SSL_MODE=letsencrypt

# Email for Let's Encrypt notifications
LETSENCRYPT_EMAIL=admin@yourdomain.com

# Set to false for production (true uses Let's Encrypt staging server)
LETSENCRYPT_STAGING=false

# For custom SSL, uncomment and set paths:
# SSL_CERT_PATH=/path/to/cert.pem
# SSL_KEY_PATH=/path/to/key.pem

# ============================================
# EMAIL CONFIGURATION (REQUIRED FOR AUTH)
# ============================================

# SMTP Settings - Examples for common providers:

# SendGrid:
AUTH_SMTP_HOST=smtp.sendgrid.net
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=apikey
AUTH_SMTP_PASS=YOUR_SENDGRID_API_KEY
AUTH_SMTP_SECURE=true

# # AWS SES:
# AUTH_SMTP_HOST=email-smtp.us-east-1.amazonaws.com
# AUTH_SMTP_PORT=587
# AUTH_SMTP_USER=YOUR_AWS_SMTP_USERNAME
# AUTH_SMTP_PASS=YOUR_AWS_SMTP_PASSWORD
# AUTH_SMTP_SECURE=true

# # Mailgun:
# AUTH_SMTP_HOST=smtp.mailgun.org
# AUTH_SMTP_PORT=587
# AUTH_SMTP_USER=postmaster@mg.yourdomain.com
# AUTH_SMTP_PASS=YOUR_MAILGUN_PASSWORD
# AUTH_SMTP_SECURE=true

# From address for auth emails
AUTH_SMTP_SENDER=noreply@yourdomain.com

# ============================================
# APPLICATION URLS (REQUIRED)
# ============================================

# URL where your frontend application is hosted
# This is used for auth redirects and CORS
AUTH_CLIENT_URL=https://app.yourdomain.com

# GraphQL API CORS domain (your frontend domain)
HASURA_GRAPHQL_CORS_DOMAIN=https://yourdomain.com

# ============================================
# PRODUCTION OPTIMIZATIONS (RECOMMENDED)
# ============================================

# Disable Hasura console in production
HASURA_GRAPHQL_ENABLE_CONSOLE=false

# Disable development mode
HASURA_GRAPHQL_DEV_MODE=false

# Set appropriate log level (debug, info, warn, error)
LOG_LEVEL=warn

# Enable rate limiting
RATE_LIMIT_ENABLED=true

# ============================================
# SERVICE URLS (OPTIONAL - ONLY IF CHANGING DEFAULTS)
# ============================================

# Only uncomment if you need different subdomains than defaults:
# HASURA_ROUTE=api.yourdomain.com
# AUTH_ROUTE=auth.yourdomain.com
# STORAGE_ROUTE=storage.yourdomain.com
# STORAGE_CONSOLE_ROUTE=storage-console.yourdomain.com

# ============================================
# FRONTEND APP ROUTING (OPTIONAL)
# ============================================

# Route your frontend apps through Nginx
# Format: LOCAL_PORT:subdomain.yourdomain.com
# APP_1_ROUTE=3000:app.yourdomain.com
# APP_2_ROUTE=3001:admin.yourdomain.com

# ============================================
# BACKUP CONFIGURATION (OPTIONAL BUT RECOMMENDED)
# ============================================

# S3 Backup Settings
# BACKUP_S3_ENDPOINT=s3.amazonaws.com
# BACKUP_S3_BUCKET=my-nself-backups
# BACKUP_S3_ACCESS_KEY=YOUR_AWS_ACCESS_KEY
# BACKUP_S3_SECRET_KEY=YOUR_AWS_SECRET_KEY
# BACKUP_S3_REGION=us-east-1

# ============================================
# MONITORING (OPTIONAL)
# ============================================

# Sentry Error Tracking
# SENTRY_DSN=https://YOUR_SENTRY_DSN@sentry.io/PROJECT_ID

# DataDog APM
# DD_AGENT_HOST=datadog-agent
# DD_TRACE_ENABLED=true

# ============================================
# NOTES
# ============================================
# 1. Generate all passwords with: openssl rand -hex 32
# 2. Never commit this file to version control
# 3. Keep a secure backup of this configuration
# 4. Test in staging before production deployment
# 5. Monitor logs after deployment: docker compose logs -f
# ============================================
EOF

  # Also create a separate file with just the generated credentials
  cat > .env.prod-secrets << EOF
# ============================================
# GENERATED PRODUCTION SECRETS
# Created: $(date)
# SAVE THIS FILE IN A SECURE LOCATION!
# ============================================

# Database
POSTGRES_PASSWORD=${POSTGRES_PASSWORD_GENERATED}

# Hasura
HASURA_GRAPHQL_ADMIN_SECRET=${HASURA_SECRET_GENERATED}
HASURA_GRAPHQL_JWT_SECRET='{"type":"HS256","key":"${JWT_KEY_GENERATED}"}'

# Storage
MINIO_ROOT_PASSWORD=${MINIO_PASSWORD_GENERATED}
S3_SECRET_KEY=${S3_SECRET_GENERATED}

# Redis (if used)
REDIS_PASSWORD=${REDIS_PASSWORD_GENERATED}

# ============================================
# Copy these values to your password manager!
# ============================================
EOF

  echo_success "Created .env file from .env.local"
  echo_success "Created .env.prod-template with production configuration"
  echo_success "Generated secure secrets in .env.prod-secrets"
  echo ""
  echo_info "🔐 Secure passwords have been generated for you!"
  echo_info "The generated secrets are saved in:"
  echo_info "  • .env.prod-template (ready to use)"
  echo_info "  • .env.prod-secrets (backup copy)"
  echo ""
  echo_info "Next steps for production deployment:"
  echo_info "1. Review and edit .env.prod-template"
  echo_info "2. Update your domain and email settings"
  echo_info "3. Copy .env.prod-template to .env"
  echo_info "4. Run 'nself up' to deploy"
  echo ""
  echo_warning "IMPORTANT: Save .env.prod-secrets in a secure location!"
  echo_warning "Delete it after copying to your password manager."
}

# Restart services
cmd_restart() {
  echo_info "Restarting services..."
  
  # Down then up
  cmd_down
  echo ""
  cmd_up
}

# Update nself
cmd_update() {
  echo_info "Checking for updates..."
  
  read_local_version
  fetch_latest_version

  if [ -z "$LATEST_VERSION" ]; then
    echo_error "Failed to fetch latest version."
    exit 1
  fi

  if is_newer_version; then
    echo_info "Updating from $LOCAL_VERSION to $LATEST_VERSION..."
    
    # Download update script and run it
    if curl -fsSL "$REPO_RAW_URL/install.sh" | bash; then
      echo_success "Update complete!"
    else
      echo_error "Update failed."
      exit 1
    fi
  else
    echo_info "Already running the latest version ($LOCAL_VERSION)."
  fi
}

# Display help
cmd_help() {
  cat << EOF

$(echo_info "nself cli v$(cat "$VERSION_FILE" 2>/dev/null || echo "0.1.0") - The Complete Nhost Self-hosted Stack")

$(echo_info "Usage:") nself [command] [options]

$(echo_info "Commands:")
  init        Initialize a new project with .env.local
  build       Build project structure from .env.local  
  up          Start all services (core + backend services)
  down        Stop all services
  restart     Restart all services (down + up)
  prod        Create production .env from .env.local
  reset       Reset project (delete all data)
  update      Update nself to latest version
  version     Show current version
  help        Display this help message
  
  db          Database tools (run 'nself db' for details)

$(echo_info "Options:")
  -h, --help     Display help
  -v, --version  Show version

$(echo_info "Quick Start:")
  $ nself init                    # Create new project
  $ nano .env.local              # Configure your project
  $ nself build                  # Generate Docker configs
  $ nself up                     # Start everything!

$(echo_info "Production Deployment:")
  $ nself prod                   # Generate secure passwords & config
  $ nano .env.prod-template      # Edit domain and email
  $ cp .env.prod-template .env   # Activate production config
  $ nself up                     # Deploy to production

$(echo_info "Service URLs (local.nself.org):")
  • GraphQL API:     https://api.local.nself.org
  • Authentication:  https://auth.local.nself.org  
  • Storage:         https://storage.local.nself.org
  • Dashboard:       https://dashboard.local.nself.org
  
$(echo_info "Optional Services:")
  • Redis:           In-memory cache and queues
  • NestJS:          Microservices for Hasura actions
  • BullMQ:          Background job processing
  • GoLang:          High-performance services
  • Python:          ML/AI and data analysis

$(echo_info "Documentation:") https://nself.com
$(echo_info "GitHub:") https://github.com/acamarata/nself

EOF
}

# Display version
cmd_version() {
  read_local_version
  echo "nself version $LOCAL_VERSION"
}

# ----------------------------
# Main Logic
# ----------------------------

# Check for version flag first
case "$1" in
  -v|--version|version)
    cmd_version
    exit 0
    ;;
  -h|--help|help)
    cmd_help
    exit 0
    ;;
esac

# Process commands
COMMAND="$1"

# Check dependencies (skip for update)
if [ "$COMMAND" != "update" ]; then
  if ! command_exists docker; then
    echo_error "Docker is not installed. Please install Docker first."
    exit 1
  fi

  if ! command_exists docker compose && ! command_exists docker-compose; then
    echo_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
  fi
fi

case "$COMMAND" in
  init)
    check_for_updates
    cmd_init
    ;;
  build)
    check_for_updates
    cmd_build
    ;;
  up)
    check_for_updates
    cmd_up
    ;;
  down)
    cmd_down
    ;;
  restart)
    cmd_restart
    ;;
  reset)
    cmd_reset
    ;;
  prod)
    cmd_prod
    ;;
  update)
    cmd_update
    ;;
  db)
    # Database tools - migrations, seeding, and schema management
    shift
    "$SCRIPT_DIR/db.sh" "$@"
    ;;
  "")
    cmd_help
    ;;
  *)
    echo_error "Unknown command: $COMMAND"
    cmd_help
    exit 1
    ;;
esac