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
  show_command_header "nself admin" "Launching nself admin UI"
  echo ""  # Add blank line after header

  # Check if directory has existing nself project
  if [[ -f ".env" ]] || [[ -f ".env.dev" ]] || [[ -f "docker-compose.yml" ]]; then
    log_warning "Existing project files detected"
    log_info "For a clean admin setup, run in an empty directory"
    echo ""
  fi
  
  # Pull the admin image (always get latest, suppress output unless there's an update)
  local pull_output
  pull_output=$(docker pull acamarata/nself-admin:latest 2>&1)
  if echo "$pull_output" | grep -q "Downloaded newer image"; then
    log_success "Updated nself-admin image"
  elif echo "$pull_output" | grep -q "Image is up to date"; then
    log_success "nself-admin image ready"
  else
    # Only show full output if there's an error
    if ! echo "$pull_output" | grep -q "up to date\|Downloaded"; then
      echo "$pull_output"
    fi
  fi
  
  # Find an available port starting from 3021
  local admin_port=3021
  local max_port=3030
  local port_found=false

  while [ $admin_port -le $max_port ]; do
    if ! lsof -Pi :$admin_port -t >/dev/null 2>&1; then
      port_found=true
      break
    fi
    ((admin_port++))
  done

  if [ "$port_found" = false ]; then
    log_error "No available ports found between 3021-3030"
    log_info "Please free up a port or specify a custom port"
    return 1
  fi

  # Show which port we're using if not default
  if [ $admin_port -ne 3021 ]; then
    log_info "Using port $admin_port (3021 was in use)"
  fi

  # Get current directory for mounting
  local current_dir="$(pwd)"

  # Stop any existing nself-admin container
  docker stop nself-admin 2>/dev/null || true
  docker rm nself-admin 2>/dev/null || true

  # Run the admin container directly without creating any files
  log_info "Starting nself-admin container..."

  # Start the container with docker run
  local container_id
  container_id=$(docker run -d \
    --name nself-admin \
    -p "$admin_port:3021" \
    -v "$current_dir:/workspace:rw" \
    -e "NSELF_PROJECT_PATH=/workspace" \
    -e "PROJECT_PATH=/workspace" \
    --restart unless-stopped \
    acamarata/nself-admin:latest 2>&1)

  if [[ $? -ne 0 ]]; then
    log_error "Failed to start nself-admin container"
    echo "$container_id"
    return 1
  fi

  # Wait for container to be ready
  local max_attempts=30
  local attempt=0

  while [ $attempt -lt $max_attempts ]; do
    if docker ps | grep -q "nself-admin.*Up"; then
      log_success "nself-admin is running"
      break
    fi
    sleep 1
    ((attempt++))
  done

  if [ $attempt -eq $max_attempts ]; then
    log_error "nself-admin failed to start"
    docker logs nself-admin 2>&1 | tail -10
    return 1
  fi

  # Show access information
  echo ""
  log_success "Admin UI is now running!"
  echo ""
  echo "📋 Access Information:"
  echo "  URL:      http://localhost:$admin_port"
  echo ""
  echo "📝 Next Steps:"
  echo "  1. Open http://localhost:$admin_port in your browser"
  echo "  2. Set up your admin credentials on first login"
  echo "  3. Use the admin UI to configure your full nself project"
  echo ""
  
  # Try to open browser (silently)
  if command -v open >/dev/null 2>&1; then
    open "http://localhost:$admin_port" 2>/dev/null &
  elif command -v xdg-open >/dev/null 2>&1; then
    xdg-open "http://localhost:$admin_port" 2>/dev/null &
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
    echo "NSELF_ADMIN_PORT=3021" >> .env
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
  local admin_port="${NSELF_ADMIN_PORT:-3021}"
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
  echo "NSELF_ADMIN_PORT=3021" >> .env
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
  local admin_port="${NSELF_ADMIN_PORT:-3021}"
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