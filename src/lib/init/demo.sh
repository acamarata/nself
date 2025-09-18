#!/usr/bin/env bash
# demo.sh - Demo setup functionality for nself init --demo
#
# Creates a complete demo environment with all services enabled,
# custom backend services, frontend apps, and remote schemas

# Source required utilities
DEMO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source display utilities for consistent styling
if [[ -f "$DEMO_DIR/../utils/display.sh" ]]; then
  source "$DEMO_DIR/../utils/display.sh"
else
  # Fallback definitions if display.sh not available
  show_command_header() {
    echo ""
    echo -e "\033[1;34m$1\033[0m"
    echo "$2"
    echo ""
  }
  safe_echo() {
    echo "$@"
  }
  log_success() {
    echo -e "\033[32m✓\033[0m $1"
  }
  log_info() {
    echo -e "\033[34mℹ\033[0m $1"
  }
  log_warning() {
    echo -e "\033[33m⚠\033[0m $1"
  }
  log_error() {
    echo -e "\033[31m✗\033[0m $1"
  }
fi

source "$DEMO_DIR/../utils/env.sh" 2>/dev/null || true
source "$DEMO_DIR/gitignore.sh" 2>/dev/null || true

# Setup complete demo environment
# Inputs: $1 - script directory
# Outputs: Creates demo configuration files
# Returns: 0 on success, error code on failure
setup_demo() {
  local script_dir="${1:-$DEMO_DIR}"
  # Find the templates directory relative to the init module
  local templates_dir="$(cd "$DEMO_DIR" && cd ../../templates/demo && pwd)"
  local current_dir="$(pwd)"

  # Show standard header with demo subtitle
  show_command_header "nself init --demo" "Create a complete demo application"
  echo ""  # Add blank line after header

  echo "This will create a complete demo configuration for:"
  echo ""
  echo "  ✅ All core services (PostgreSQL, Hasura, Auth, Nginx)"
  echo "  ✅ All optional services enabled"
  echo "  ✅ 2 Custom backend services"
  echo "  ✅ 2 Frontend applications"
  echo "  ✅ Remote schemas configured for GraphQL federation"
  echo "  ✅ Demo data and seed content for database"
  echo ""

  # Check templates exist
  if [[ ! -f "$templates_dir/.env.demo" ]]; then
    log_error "Demo templates not found at $templates_dir"
    log_info "Please ensure nself is properly installed"
    return 1
  fi

  # Copy demo environment file
  cp "$templates_dir/.env.demo" .env.dev

  # Create local .env for overrides
  cat > .env << 'EOF'
# Local Configuration Overrides for Demo
# Add any personal overrides here

# Uncomment to change the default domain
# BASE_DOMAIN=localhost

# Uncomment to change default ports
# POSTGRES_PORT=5433
# REDIS_PORT=6380
EOF

  # Create .env.example from demo
  cp "$templates_dir/.env.demo" .env.example

  # Ensure gitignore
  if [[ -f "$script_dir/gitignore.sh" ]]; then
    source "$script_dir/gitignore.sh"
    ensure_gitignore >/dev/null 2>&1
  fi

  echo "Next steps:"
  echo ""
  safe_echo "${COLOR_BLUE:-}1.${COLOR_RESET:-} Edit .env to customize (optional)"
  safe_echo "   ${COLOR_DIM:-}All services are pre-configured for demo${COLOR_RESET:-}"
  echo ""
  safe_echo "${COLOR_BLUE:-}2.${COLOR_RESET:-} nself build - Generate project files"
  safe_echo "   ${COLOR_DIM:-}Creates Docker configs and services${COLOR_RESET:-}"
  echo ""
  safe_echo "${COLOR_BLUE:-}3.${COLOR_RESET:-} nself start - Start your backend"
  safe_echo "   ${COLOR_DIM:-}Launches all demo services${COLOR_RESET:-}"
  echo ""

  # Add help line at bottom
  echo "For more help, use: nself help or nself help init"
  echo ""

  return 0
}

# Export the function
export -f setup_demo