#!/usr/bin/env bash
set -euo pipefail

# admin.sh - Admin UI management commands

# Determine root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source required utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"

# Setup minimal admin-only environment in blank directory
admin_minimal_setup() {
  show_command_header "nself admin" "Setting up minimal admin environment"
  
  # Create minimal .env for admin-only mode
  if [[ ! -f ".env" ]]; then
    log_info "Creating minimal admin configuration..."
    
    cat > .env << 'EOF'
# Minimal nself configuration for admin UI only
PROJECT_NAME=admin-setup
BASE_DOMAIN=localhost
ENV=dev

# Admin UI Configuration
NSELF_ADMIN_ENABLED=true
NSELF_ADMIN_PORT=3100
NSELF_ADMIN_AUTH_PROVIDER=basic
NSELF_ADMIN_ROUTE=localhost:3100

# SSL Configuration (disabled for minimal setup)
SSL_ENABLED=false

# Database (minimal for admin operations)
POSTGRES_ENABLED=true
POSTGRES_PORT=5432
POSTGRES_DB=nself_admin
POSTGRES_USER=nself
POSTGRES_PASSWORD=nself_admin_temp

# Disable other services for minimal setup
HASURA_ENABLED=false
AUTH_ENABLED=false
STORAGE_ENABLED=false
REDIS_ENABLED=false
NESTJS_ENABLED=false
FUNCTIONS_ENABLED=false
EOF
    
    log_success "Created minimal .env configuration"
  else
    log_info "Found existing .env, enabling admin UI..."
    
    # Just enable admin in existing config
    if grep -q "^NSELF_ADMIN_ENABLED=" .env; then
      sed -i.bak 's/^NSELF_ADMIN_ENABLED=.*/NSELF_ADMIN_ENABLED=true/' .env
    else
      echo "NSELF_ADMIN_ENABLED=true" >> .env
    fi
  fi
  
  # Pull the admin image
  log_info "Pulling nself-admin Docker image..."
  docker pull acamarata/nself-admin:latest
  
  # Set admin password
  log_info "Setting up admin access..."
  
  # Generate a temporary password if none set
  if ! grep -q "^ADMIN_PASSWORD_HASH=" .env 2>/dev/null; then
    local temp_password="admin123"
    
    # Generate password hash with salt
    local password_hash
    if command -v python3 >/dev/null 2>&1; then
      password_hash=$(python3 -c "
import hashlib, os, base64
salt = os.urandom(32)
pwd_hash = hashlib.pbkdf2_hmac('sha256', '$temp_password'.encode('utf-8'), salt, 100000)
combined = salt + pwd_hash
print(base64.b64encode(combined).decode('ascii'))
")
    elif command -v openssl >/dev/null 2>&1; then
      local salt=$(openssl rand -hex 16)
      password_hash="${salt}:$(echo -n "${salt}${temp_password}" | openssl dgst -sha256 -binary | base64)"
    else
      password_hash=$(echo -n "$temp_password" | sha256sum | cut -d' ' -f1)
    fi
    
    echo "ADMIN_PASSWORD_HASH=$password_hash" >> .env
    
    log_warning "Temporary password set: $temp_password"
    log_info "Change this password after first login!"
  fi
  
  # Generate secret key
  if ! grep -q "^ADMIN_SECRET_KEY=" .env 2>/dev/null; then
    local secret_key=$(openssl rand -hex 32 2>/dev/null || date +%s | sha256sum | head -c 64)
    echo "ADMIN_SECRET_KEY=$secret_key" >> .env
  fi
  
  # Start minimal admin environment
  log_info "Starting minimal admin environment..."
  
  # Create minimal docker-compose for admin only
  cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: nself_admin
      POSTGRES_USER: nself
      POSTGRES_PASSWORD: nself_admin_temp
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nself"]
      interval: 5s
      timeout: 5s
      retries: 5

  # Admin UI service
  nself-admin:
    image: acamarata/nself-admin:latest
    container_name: ${PROJECT_NAME}_admin
    ports:
      - "3100:3021"
    environment:
      - PROJECT_DIR=/workspace
      - ADMIN_SECRET_KEY=${ADMIN_SECRET_KEY}
      - ADMIN_PASSWORD_HASH=${ADMIN_PASSWORD_HASH}
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./:/workspace:rw
      - nself-admin-data:/app/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3021/health"]
      interval: 30s
      timeout: 10s
      retries: 5

volumes:
  postgres_data:
  nself-admin-data:
EOF
  
  log_success "Created minimal docker-compose.yml"
  
  # Start services
  log_info "Starting admin UI..."
  
  if command -v docker-compose >/dev/null 2>&1; then
    docker-compose up -d
  elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    docker compose up -d
  else
    log_error "Docker Compose not found!"
    log_info "Please install Docker Compose to continue"
    return 1
  fi
  
  # Wait for services to be ready
  log_info "Waiting for services to start..."
  sleep 5
  
  # Show access information
  echo ""
  log_success "Admin UI is now running!"
  echo ""
  echo "📋 Access Information:"
  echo "  URL:      http://localhost:3100"
  echo "  Username: admin"
  echo "  Password: admin123"
  echo ""
  log_warning "Change the default password after first login!"
  echo ""
  echo "📝 Next Steps:"
  echo "  1. Open http://localhost:3100 in your browser"
  echo "  2. Login with the credentials above"
  echo "  3. Use the admin UI to configure your full nself project"
  echo "  4. The admin UI will guide you through project setup"
  echo ""
  
  # Try to open browser
  if command -v open >/dev/null 2>&1; then
    log_info "Opening browser..."
    open "http://localhost:3100" 2>/dev/null &
  elif command -v xdg-open >/dev/null 2>&1; then
    log_info "Opening browser..."
    xdg-open "http://localhost:3100" 2>/dev/null &
  fi
}

# Show help for admin command
show_admin_help() {
  echo "nself admin - Admin UI management"
  echo ""
  echo "Usage: nself admin [subcommand] [OPTIONS]"
  echo ""
  echo "Quick Start:"
  echo "  nself admin                    # Setup minimal admin UI in any directory"
  echo ""
  echo "Subcommands:"
  echo "  enable     Enable admin web interface"
  echo "  disable    Disable admin web interface"
  echo "  status     Show admin UI status"
  echo "  password   Set admin password"
  echo "  reset      Reset admin to defaults"
  echo "  logs       View admin logs"
  echo "  open       Open admin in browser"
  echo ""
  echo "Description:"
  echo "  Run 'nself admin' in any blank directory to instantly spin up the"
  echo "  admin UI with minimal configuration. The web interface will guide"
  echo "  you through setting up a full nself project and can restart"
  echo "  services as needed."
  echo ""
  echo "Examples:"
  echo "  nself admin                    # Instant admin UI setup"
  echo "  nself admin enable             # Enable admin UI in existing project"
  echo "  nself admin password           # Set admin password"
  echo "  nself admin open               # Open in browser"
  echo "  nself admin logs --follow      # View live logs"
}

# Enable admin UI
admin_enable() {
  show_command_header "nself admin enable" "Enable admin web interface"
  
  # Check if .env exists
  if [[ ! -f ".env" ]]; then
    log_error "No .env file found. Run 'nself init' first."
    return 1
  fi
  
  # Load environment
  load_env_with_priority 2>/dev/null || true
  
  # Set admin enabled
  log_info "Enabling admin UI..."
  
  # Update .env
  if grep -q "^NSELF_ADMIN_ENABLED=" .env 2>/dev/null; then
    sed -i.bak 's/^NSELF_ADMIN_ENABLED=.*/NSELF_ADMIN_ENABLED=true/' .env
  else
    echo "NSELF_ADMIN_ENABLED=true" >> .env
  fi
  
  # Set default values if not present
  if ! grep -q "^NSELF_ADMIN_PORT=" .env 2>/dev/null; then
    # Ensure we're on a new line before appending
    echo "" >> .env
    echo "NSELF_ADMIN_PORT=3100" >> .env
  fi
  
  if ! grep -q "^NSELF_ADMIN_AUTH_PROVIDER=" .env 2>/dev/null; then
    echo "NSELF_ADMIN_AUTH_PROVIDER=basic" >> .env
  fi
  
  if ! grep -q "^NSELF_ADMIN_ROUTE=" .env 2>/dev/null; then
    echo "NSELF_ADMIN_ROUTE=admin.\${BASE_DOMAIN}" >> .env
  fi
  
  # Ensure admin password hash and secret key are set
  if ! grep -q "^ADMIN_PASSWORD_HASH=" .env 2>/dev/null; then
    log_warning "No admin password set. Generating temporary password..."
    local temp_password="admin$(date +%s | sha256sum | head -c 8)"
    
    # Generate password hash
    local password_hash
    if command -v python3 >/dev/null 2>&1; then
      password_hash=$(python3 -c "
import hashlib, os, base64
salt = os.urandom(32)
pwd_hash = hashlib.pbkdf2_hmac('sha256', '$temp_password'.encode('utf-8'), salt, 100000)
combined = salt + pwd_hash
print(base64.b64encode(combined).decode('ascii'))
")
    else
      password_hash=$(echo -n "$temp_password" | sha256sum | cut -d' ' -f1)
    fi
    
    echo "ADMIN_PASSWORD_HASH=$password_hash" >> .env
    log_info "Temporary admin password: $temp_password"
    log_info "Change this after first login with 'nself admin password'"
  fi
  
  if ! grep -q "^ADMIN_SECRET_KEY=" .env 2>/dev/null; then
    local secret_key=$(openssl rand -hex 32 2>/dev/null || date +%s | sha256sum | head -c 64)
    echo "ADMIN_SECRET_KEY=$secret_key" >> .env
  fi
  
  log_success "Admin UI enabled"
  log_info "Run 'nself build' to generate configuration"
  log_info "Then 'nself start' to launch the admin UI"
  
  # Show access URL
  local protocol="http"
  local ssl_mode="${SSL_MODE:-none}"
  if [[ "$ssl_mode" == "local" ]] || [[ "$ssl_mode" == "letsencrypt" ]]; then
    protocol="https"
  fi
  
  local admin_route="${NSELF_ADMIN_ROUTE:-admin.${BASE_DOMAIN}}"
  admin_route=$(echo "$admin_route" | sed "s/\${BASE_DOMAIN}/$BASE_DOMAIN/g")
  
  echo ""
  log_info "Admin UI will be available at:"
  echo "  ${protocol}://${admin_route}"
  
  if [[ -z "${ADMIN_PASSWORD_HASH:-}" ]]; then
    echo ""
    log_warning "No admin password set!"
    log_info "Run 'nself admin password' to set one"
  fi
}

# Disable admin UI
admin_disable() {
  show_command_header "nself admin disable" "Disable admin web interface"
  
  # Check if .env exists
  if [[ ! -f ".env" ]]; then
    log_error "No .env file found. Run 'nself init' first."
    return 1
  fi
  
  # Load environment
  load_env_with_priority 2>/dev/null || true
  
  log_info "Disabling admin UI..."
  
  # Update .env
  if grep -q "^NSELF_ADMIN_ENABLED=" .env 2>/dev/null; then
    sed -i.bak 's/^NSELF_ADMIN_ENABLED=.*/NSELF_ADMIN_ENABLED=false/' .env
  else
    echo "NSELF_ADMIN_ENABLED=false" >> .env
  fi
  
  # Get project name for container
  local project_name="${PROJECT_NAME:-myproject}"
  local container_name="${project_name}_admin"
  
  # Stop admin container if running
  if docker ps --format "{{.Names}}" | grep -q "$container_name"; then
    log_info "Stopping admin container..."
    docker stop "$container_name" >/dev/null 2>&1
    docker rm "$container_name" >/dev/null 2>&1
  fi
  
  log_success "Admin UI disabled"
  log_info "Run 'nself build' to update configuration"
}

# Show admin status
admin_status() {
  show_command_header "nself admin status" "Admin UI status"
  
  # Check if .env exists
  if [[ ! -f ".env" ]]; then
    log_error "No .env file found. Run 'nself init' first."
    return 1
  fi
  
  # Load environment
  load_env_with_priority 2>/dev/null || true
  
  local admin_enabled="${NSELF_ADMIN_ENABLED:-false}"
  local admin_port="${NSELF_ADMIN_PORT:-3022}"
  local admin_username="${ADMIN_USERNAME:-admin}"
  local admin_route="${NSELF_ADMIN_ROUTE:-admin.${BASE_DOMAIN}}"
  admin_route=$(echo "$admin_route" | sed "s/\${BASE_DOMAIN}/$BASE_DOMAIN/g")
  
  echo "Configuration:"
  echo "  Enabled:  $admin_enabled"
  echo "  Port:     $admin_port"
  echo "  Username: $admin_username"
  echo "  Route:    $admin_route"
  
  if [[ -n "${ADMIN_PASSWORD_HASH:-}" ]]; then
    echo "  Password: [SET]"
  else
    echo "  Password: [NOT SET]"
  fi
  
  if [[ "${ADMIN_2FA_ENABLED:-false}" == "true" ]]; then
    echo "  2FA:      Enabled"
  else
    echo "  2FA:      Disabled"
  fi
  
  echo ""
  echo "Container Status:"
  
  # Get project name for container
  local project_name="${PROJECT_NAME:-myproject}"
  local container_name="${project_name}_admin"
  
  if docker ps --format "{{.Names}}" | grep -q "$container_name"; then
    log_success "Admin container is running"
    
    # Show container info
    local container_info=$(docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1)
    echo "  $container_info"
    
    # Show access URL
    local protocol="http"
    local ssl_mode="${SSL_MODE:-none}"
    if [[ "$ssl_mode" == "local" ]] || [[ "$ssl_mode" == "letsencrypt" ]]; then
      protocol="https"
    fi
    
    echo ""
    log_info "Access URL: ${protocol}://${admin_route}"
  else
    if [[ "$admin_enabled" == "true" ]]; then
      log_warning "Admin container is not running"
      log_info "Run 'nself start' to launch it"
    else
      log_info "Admin container is not running (disabled)"
    fi
  fi
}

# Set admin password
admin_password() {
  show_command_header "nself admin password" "Set admin password"
  
  local password="${1:-}"
  
  # Prompt for password if not provided
  if [[ -z "$password" ]]; then
    echo -n "Enter admin password: "
    read -s password
    echo
    echo -n "Confirm password: "
    local confirm
    read -s confirm
    echo
    
    if [[ "$password" != "$confirm" ]]; then
      log_error "Passwords do not match"
      return 1
    fi
  fi
  
  # Validate password
  if [[ ${#password} -lt 8 ]]; then
    log_error "Password must be at least 8 characters"
    return 1
  fi
  
  # Generate password hash with salt using Python (more secure than plain SHA256)
  local password_hash
  if command -v python3 >/dev/null 2>&1; then
    password_hash=$(python3 -c "
import hashlib, os, base64
salt = os.urandom(32)
pwd_hash = hashlib.pbkdf2_hmac('sha256', '$password'.encode('utf-8'), salt, 100000)
combined = salt + pwd_hash
print(base64.b64encode(combined).decode('ascii'))
")
  elif command -v openssl >/dev/null 2>&1; then
    # Fallback to OpenSSL with salt
    local salt=$(openssl rand -hex 16)
    password_hash="${salt}:$(echo -n "${salt}${password}" | openssl dgst -sha256 -binary | base64)"
  else
    # Last resort: SHA256 with warning
    log_warning "Using weak password hashing (no Python or OpenSSL available)"
    password_hash=$(echo -n "$password" | sha256sum | cut -d' ' -f1)
  fi
  
  # Update .env - escape special characters in hash
  if grep -q "^ADMIN_PASSWORD_HASH=" .env 2>/dev/null; then
    # Use a different delimiter to avoid issues with special characters
    # First, create a temp file with the new value
    grep -v "^ADMIN_PASSWORD_HASH=" .env > .env.tmp
    echo "ADMIN_PASSWORD_HASH=$password_hash" >> .env.tmp
    mv .env.tmp .env
  else
    echo "ADMIN_PASSWORD_HASH=$password_hash" >> .env
  fi
  
  # Generate secret key if not present
  if ! grep -q "^ADMIN_SECRET_KEY=" .env 2>/dev/null; then
    local secret_key=$(openssl rand -hex 32)
    echo "ADMIN_SECRET_KEY=$secret_key" >> .env
  fi
  
  log_success "Admin password set successfully"
  
  # Get project name for container
  local project_name="${PROJECT_NAME:-myproject}"
  local container_name="${project_name}_admin"
  
  # Restart admin if running
  if docker ps --format "{{.Names}}" | grep -q "$container_name"; then
    log_info "Restarting admin container..."
    docker restart "$container_name" >/dev/null 2>&1
  fi
}

# Reset admin configuration
admin_reset() {
  show_command_header "nself admin reset" "Reset admin to defaults"
  
  log_warning "This will reset all admin settings to defaults"
  echo -n "Continue? (y/N): "
  local confirm
  read confirm
  
  if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
    log_info "Reset cancelled"
    return 0
  fi
  
  log_info "Resetting admin configuration..."
  
  # Remove admin settings from .env
  sed -i.bak '/^ADMIN_/d' .env
  
  # Set defaults
  echo "NSELF_ADMIN_ENABLED=false" >> .env
  echo "NSELF_ADMIN_PORT=3100" >> .env
  echo "NSELF_ADMIN_AUTH_PROVIDER=basic" >> .env
  echo "NSELF_ADMIN_ROUTE=admin.\${BASE_DOMAIN}" >> .env
  
  # Get project name for container
  local project_name="${PROJECT_NAME:-myproject}"
  local container_name="${project_name}_admin"
  
  # Stop admin container if running
  if docker ps --format "{{.Names}}" | grep -q "$container_name"; then
    log_info "Stopping admin container..."
    docker stop "$container_name" >/dev/null 2>&1
    docker rm "$container_name" >/dev/null 2>&1
  fi
  
  log_success "Admin configuration reset to defaults"
}

# View admin logs
admin_logs() {
  show_command_header "nself admin logs" "View admin logs"
  
  local follow=false
  local tail_lines=50
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
    --follow | -f)
      follow=true
      shift
      ;;
    --tail | -n)
      tail_lines="$2"
      shift 2
      ;;
    *)
      shift
      ;;
    esac
  done
  
  # Load environment to get project name
  load_env_with_priority 2>/dev/null || true
  
  # Get project name for container
  local project_name="${PROJECT_NAME:-myproject}"
  local container_name="${project_name}_admin"
  
  # Check if admin container exists
  if ! docker ps -a --format "{{.Names}}" | grep -q "$container_name"; then
    log_error "Admin container not found ($container_name)"
    log_info "Run 'nself admin enable' and 'nself start' first"
    return 1
  fi
  
  # Show logs
  if [[ "$follow" == "true" ]]; then
    docker logs -f --tail "$tail_lines" "$container_name"
  else
    docker logs --tail "$tail_lines" "$container_name"
  fi
}

# Open admin in browser
admin_open() {
  show_command_header "nself admin open" "Open admin in browser"
  
  # Check if .env exists
  if [[ ! -f ".env" ]]; then
    log_error "No .env file found. Run 'nself init' first."
    return 1
  fi
  
  # Load environment
  load_env_with_priority 2>/dev/null || true
  
  # Get project name for container
  local project_name="${PROJECT_NAME:-myproject}"
  local container_name="${project_name}_admin"
  
  # Check if admin is running
  if ! docker ps --format "{{.Names}}" | grep -q "$container_name"; then
    log_error "Admin container is not running"
    log_info "Run 'nself admin enable' and 'nself start' first"
    return 1
  fi
  
  # Determine URL - use localhost for simplicity
  local admin_port="${NSELF_ADMIN_PORT:-3100}"
  local url="http://localhost:${admin_port}"
  
  log_info "Opening admin UI at: $url"
  
  # Open in browser (cross-platform)
  if command -v open >/dev/null 2>&1; then
    open "$url" 2>/dev/null &
  elif command -v xdg-open >/dev/null 2>&1; then
    xdg-open "$url" 2>/dev/null &
  elif command -v start >/dev/null 2>&1; then
    start "$url" 2>/dev/null &
  else
    log_warning "Could not open browser automatically"
    log_info "Please navigate to: $url"
  fi
}

# Main command function
cmd_admin() {
  local subcommand="${1:-}"
  shift || true
  
  case "$subcommand" in
  enable)
    admin_enable "$@"
    ;;
  disable)
    admin_disable "$@"
    ;;
  status)
    admin_status "$@"
    ;;
  password)
    admin_password "$@"
    ;;
  reset)
    admin_reset "$@"
    ;;
  logs)
    admin_logs "$@"
    ;;
  open)
    admin_open "$@"
    ;;
  -h | --help | help)
    show_admin_help
    ;;
  "")
    # If no subcommand and no .env, setup minimal admin
    if [[ ! -f ".env" ]]; then
      admin_minimal_setup
    else
      # If .env exists, show status
      admin_status
    fi
    ;;
  *)
    log_error "Unknown subcommand: $subcommand"
    echo ""
    show_admin_help
    return 1
    ;;
  esac
}

# Export for use as library
export -f cmd_admin

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_admin "$@"
fi