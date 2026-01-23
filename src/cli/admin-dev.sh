#!/usr/bin/env bash
# admin-dev.sh - Quick toggle for nself-admin development mode
# Part of nself v0.4.7 - Infrastructure Everywhere
#
# Simple command for nself-admin contributors to toggle local dev mode
# Usage:
#   nself admin-dev on [path]   Enable dev mode with auto-rebuild
#   nself admin-dev off         Disable dev mode with auto-rebuild
#   nself admin-dev status      Show current dev mode status

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source shared utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true

# Ensure color variables are defined
COLOR_RESET=${COLOR_RESET:-$'\033[0m'}
COLOR_BLUE=${COLOR_BLUE:-$'\033[0;34m'}
COLOR_GREEN=${COLOR_GREEN:-$'\033[0;32m'}
COLOR_RED=${COLOR_RED:-$'\033[0;31m'}
COLOR_YELLOW=${COLOR_YELLOW:-$'\033[0;33m'}
COLOR_CYAN=${COLOR_CYAN:-$'\033[0;36m'}
COLOR_DIM=${COLOR_DIM:-$'\033[0;90m'}
COLOR_BOLD=${COLOR_BOLD:-$'\033[1m'}

# Simple logging functions if not already defined
if ! declare -f log_info >/dev/null 2>&1; then
  log_info() { printf "${COLOR_BLUE}[INFO]${COLOR_RESET} %s\n" "$1"; }
  log_success() { printf "${COLOR_GREEN}[OK]${COLOR_RESET} %s\n" "$1"; }
  log_error() { printf "${COLOR_RED}[ERROR]${COLOR_RESET} %s\n" "$1" >&2; }
  log_warning() { printf "${COLOR_YELLOW}[WARN]${COLOR_RESET} %s\n" "$1"; }
fi

show_help() {
  cat << 'EOF'
nself admin-dev - Quick toggle for nself-admin local development

USAGE:
  nself admin-dev on [port] [path]   Enable dev mode (auto-rebuilds)
  nself admin-dev off                Disable dev mode (auto-rebuilds)
  nself admin-dev status             Show current status

DESCRIPTION:
  Quickly switch between running nself-admin from Docker vs locally.
  This command automatically rebuilds nginx and restarts services.

EXAMPLES:
  nself admin-dev on                 # Enable on default port 3025
  nself admin-dev on 3000            # Enable on port 3000
  nself admin-dev on 3000 ~/Sites/nself-admin
  nself admin-dev off                # Switch back to Docker
  nself admin-dev status             # Check current mode

OPTIONS:
  -h, --help    Show this help message

NOTE:
  This is a convenience wrapper around 'nself service admin dev'
  with automatic rebuild and restart.
EOF
}

# Detect port from package.json if path is provided
detect_port_from_package() {
  local path="$1"
  local pkg_file="$path/package.json"

  if [[ -f "$pkg_file" ]]; then
    # Try to extract port from dev script or config
    local port
    port=$(grep -o '"port"[[:space:]]*:[[:space:]]*[0-9]*' "$pkg_file" 2>/dev/null | head -1 | grep -o '[0-9]*' || echo "")
    if [[ -n "$port" ]]; then
      echo "$port"
      return
    fi
  fi

  # Default port
  echo "3025"
}

# Get the active env file based on environment
get_active_env_file() {
  # Load environment to determine which file to modify
  load_env_with_priority 2>/dev/null || true

  local current_env="${ENV:-dev}"

  # For most cases, modify .env which has highest priority
  # But if we're explicitly in a different environment, use that file
  case "$current_env" in
    staging)
      if [[ -f ".env.staging" ]]; then
        echo ".env.staging"
        return
      fi
      ;;
    prod|production)
      if [[ -f ".env.prod" ]]; then
        echo ".env.prod"
        return
      fi
      ;;
  esac

  # Default to .env (highest priority)
  echo ".env"
}

cmd_admin_dev_on() {
  local port="${1:-3025}"
  local path="${2:-}"

  # If only one arg and it looks like a path, swap
  if [[ -n "$port" ]] && [[ -d "$port" ]]; then
    path="$port"
    port=$(detect_port_from_package "$path")
  fi

  # Validate port is numeric
  if ! [[ "$port" =~ ^[0-9]+$ ]]; then
    log_error "Invalid port: $port"
    return 1
  fi

  local env_file
  env_file=$(get_active_env_file)

  if [[ ! -f "$env_file" ]]; then
    log_error "No environment file found. Run 'nself init' first."
    return 1
  fi

  printf "\n${COLOR_CYAN}Enabling Admin Development Mode${COLOR_RESET}\n"
  printf "================================\n\n"

  log_info "Updating $env_file..."

  # Remove existing NSELF_ADMIN_DEV settings
  local temp_file
  temp_file=$(mktemp)
  grep -v "^NSELF_ADMIN_DEV" "$env_file" > "$temp_file" 2>/dev/null || true
  mv "$temp_file" "$env_file"

  # Add new settings
  {
    printf "\n# Admin Development Mode (local dev server)\n"
    printf "NSELF_ADMIN_DEV=true\n"
    printf "NSELF_ADMIN_DEV_PORT=%s\n" "$port"
    [[ -n "$path" ]] && printf "NSELF_ADMIN_DEV_PATH=%s\n" "$path"
  } >> "$env_file"

  log_success "Dev mode enabled (port: $port)"

  # Auto-rebuild and restart
  printf "\n"
  log_info "Rebuilding nginx configuration..."

  if bash "$SCRIPT_DIR/build.sh" --quiet 2>/dev/null; then
    log_success "Build complete"
  else
    # Try without --quiet flag
    bash "$SCRIPT_DIR/build.sh" 2>/dev/null || log_warning "Build may have had issues"
  fi

  log_info "Restarting nginx..."

  # Get project name for container
  load_env_with_priority 2>/dev/null || true
  local project_name="${PROJECT_NAME:-nself}"

  # Restart nginx container
  docker restart "${project_name}_nginx" 2>/dev/null || \
    docker restart "nginx" 2>/dev/null || \
    log_warning "Could not restart nginx (may need manual restart)"

  log_success "Nginx restarted"

  # Show next steps
  printf "\n${COLOR_GREEN}Admin dev mode is now ON${COLOR_RESET}\n\n"

  printf "Start your local admin server:\n"
  if [[ -n "$path" ]]; then
    printf "  ${COLOR_CYAN}cd %s && PORT=%s pnpm dev${COLOR_RESET}\n" "$path" "$port"
  else
    printf "  ${COLOR_CYAN}PORT=%s pnpm dev${COLOR_RESET}\n" "$port"
  fi

  local base_domain="${BASE_DOMAIN:-local.nself.org}"
  printf "\nAdmin UI: ${COLOR_BLUE}https://admin.%s${COLOR_RESET}\n" "$base_domain"
  printf "\nTo disable: ${COLOR_DIM}nself admin-dev off${COLOR_RESET}\n\n"
}

cmd_admin_dev_off() {
  local env_file
  env_file=$(get_active_env_file)

  if [[ ! -f "$env_file" ]]; then
    log_error "No environment file found"
    return 1
  fi

  printf "\n${COLOR_CYAN}Disabling Admin Development Mode${COLOR_RESET}\n"
  printf "=================================\n\n"

  log_info "Updating $env_file..."

  # Remove dev mode settings
  local temp_file
  temp_file=$(mktemp)
  grep -v "^NSELF_ADMIN_DEV" "$env_file" > "$temp_file" 2>/dev/null || true
  grep -v "^# Admin Development Mode" "$temp_file" > "$env_file" 2>/dev/null || mv "$temp_file" "$env_file"
  rm -f "$temp_file" 2>/dev/null || true

  log_success "Dev mode disabled"

  # Auto-rebuild and restart
  printf "\n"
  log_info "Rebuilding nginx configuration..."

  if bash "$SCRIPT_DIR/build.sh" --quiet 2>/dev/null; then
    log_success "Build complete"
  else
    bash "$SCRIPT_DIR/build.sh" 2>/dev/null || log_warning "Build may have had issues"
  fi

  log_info "Restarting services..."

  # Get project name
  load_env_with_priority 2>/dev/null || true
  local project_name="${PROJECT_NAME:-nself}"

  # Restart nginx and admin container
  docker restart "${project_name}_nginx" 2>/dev/null || \
    docker restart "nginx" 2>/dev/null || true

  # Try to start admin container if it exists
  docker start "${project_name}_nself-admin" 2>/dev/null || \
    docker start "nself-admin" 2>/dev/null || true

  log_success "Services restarted"

  # Show result
  printf "\n${COLOR_GREEN}Admin dev mode is now OFF${COLOR_RESET}\n\n"

  local base_domain="${BASE_DOMAIN:-local.nself.org}"
  printf "Admin UI now uses Docker container\n"
  printf "URL: ${COLOR_BLUE}https://admin.%s${COLOR_RESET}\n" "$base_domain"
  printf "\nTo re-enable: ${COLOR_DIM}nself admin-dev on${COLOR_RESET}\n\n"
}

cmd_admin_dev_status() {
  printf "\n${COLOR_CYAN}Admin Development Mode Status${COLOR_RESET}\n"
  printf "==============================\n\n"

  # Load environment with proper cascading
  load_env_with_priority 2>/dev/null || true

  local dev_enabled="${NSELF_ADMIN_DEV:-false}"
  local dev_port="${NSELF_ADMIN_DEV_PORT:-3025}"
  local dev_path="${NSELF_ADMIN_DEV_PATH:-}"
  local base_domain="${BASE_DOMAIN:-local.nself.org}"

  if [[ "$dev_enabled" == "true" ]]; then
    printf "Status: ${COLOR_GREEN}ON${COLOR_RESET} (local development)\n"
    printf "Port:   ${COLOR_BLUE}%s${COLOR_RESET}\n" "$dev_port"
    [[ -n "$dev_path" ]] && printf "Path:   ${COLOR_BLUE}%s${COLOR_RESET}\n" "$dev_path"
    printf "URL:    ${COLOR_BLUE}https://admin.%s${COLOR_RESET}\n" "$base_domain"
    printf "\n${COLOR_DIM}Nginx routes admin.* to localhost:%s${COLOR_RESET}\n" "$dev_port"
    printf "\nTo disable: ${COLOR_DIM}nself admin-dev off${COLOR_RESET}\n"
  else
    printf "Status: ${COLOR_DIM}OFF${COLOR_RESET} (using Docker container)\n"
    printf "URL:    ${COLOR_BLUE}https://admin.%s${COLOR_RESET}\n" "$base_domain"
    printf "\nTo enable: ${COLOR_DIM}nself admin-dev on [port] [path]${COLOR_RESET}\n"
  fi

  printf "\n"
}

# Main entry point
main() {
  local action="${1:-status}"
  shift || true

  case "$action" in
    on|enable)
      cmd_admin_dev_on "$@"
      ;;
    off|disable)
      cmd_admin_dev_off "$@"
      ;;
    status|"")
      cmd_admin_dev_status "$@"
      ;;
    -h|--help|help)
      show_help
      ;;
    *)
      log_error "Unknown action: $action"
      printf "\nUsage: nself admin-dev [on|off|status]\n"
      printf "       nself admin-dev --help\n\n"
      return 1
      ;;
  esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
