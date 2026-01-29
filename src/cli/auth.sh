#!/usr/bin/env bash
# auth.sh - Authentication management CLI
# Part of nself v0.6.0 - Phase 1 Sprint 1
#
# Commands:
#   nself auth login [--provider=<provider>] [--email=<email>] [--password=<password>]
#   nself auth signup [--provider=<provider>] [--email=<email>] [--password=<password>]
#   nself auth logout [--all]
#   nself auth status
#   nself auth providers list
#   nself auth providers add <provider> [--client-id=<id>] [--client-secret=<secret>]
#   nself auth providers remove <provider>
#   nself auth providers enable <provider>
#   nself auth providers disable <provider>
#   nself auth sessions list
#   nself auth sessions revoke <session-id>
#   nself auth config [--show|--set key=value]
#
# Usage: nself auth <subcommand> [options]

set -euo pipefail

# Get script directory for sourcing dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source dependencies (only if not already sourced)
if ! declare -f log_error >/dev/null 2>&1; then
  source "$NSELF_ROOT/src/lib/utils/display.sh" 2>/dev/null || true
fi

if ! declare -f detect_environment >/dev/null 2>&1; then
  source "$NSELF_ROOT/src/lib/utils/env-detection.sh" 2>/dev/null || true
fi

# Constants might already be set by main CLI
if [[ -z "${EXIT_SUCCESS:-}" ]]; then
  source "$NSELF_ROOT/src/lib/config/constants.sh" 2>/dev/null || true
fi

# Source auth manager
if [[ -f "$NSELF_ROOT/src/lib/auth/auth-manager.sh" ]]; then
  source "$NSELF_ROOT/src/lib/auth/auth-manager.sh"
else
  log_error "Auth manager not found. Installation may be corrupt."
  exit 1
fi

# Initialize auth service
if ! auth_init 2>/dev/null; then
  log_warning "Auth service initialization failed. Some features may not work."
  log_warning "Make sure PostgreSQL is running: nself start"
fi

# ============================================================================
# Command Functions
# ============================================================================

# Show usage information
auth_usage() {
  cat <<EOF
Usage: nself auth <subcommand> [options]

Authentication management for nself

SUBCOMMANDS:
  login              Authenticate user
  signup             Create new user account
  logout             End session
  status             Show current auth status
  providers          Manage authentication providers
  sessions           Manage user sessions
  config             View or modify auth configuration

LOGIN OPTIONS:
  --provider=<name>     OAuth provider (google, github, apple, etc.)
  --email=<email>       Email address
  --password=<pass>     Password (for email/password auth)
  --phone=<number>      Phone number (for SMS auth)
  --anonymous           Anonymous authentication

SIGNUP OPTIONS:
  --provider=<name>     OAuth provider (google, github, apple, etc.)
  --email=<email>       Email address
  --password=<pass>     Password (for email/password auth)
  --phone=<number>      Phone number (for SMS auth)

LOGOUT OPTIONS:
  --all                 Logout from all sessions

PROVIDER SUBCOMMANDS:
  providers list                List all available providers
  providers add <name>          Add new OAuth provider
  providers remove <name>       Remove OAuth provider
  providers enable <name>       Enable provider
  providers disable <name>      Disable provider

SESSION SUBCOMMANDS:
  sessions list                 List active sessions
  sessions revoke <id>          Revoke specific session

CONFIG SUBCOMMANDS:
  config                        Show current configuration
  config --set key=value        Set configuration value

EXAMPLES:
  # Email/password login
  nself auth login --email=user@example.com --password=secret

  # OAuth login with Google
  nself auth login --provider=google

  # Sign up with email
  nself auth signup --email=user@example.com --password=secret

  # List OAuth providers
  nself auth providers list

  # Add Google OAuth provider
  nself auth providers add google --client-id=xxx --client-secret=yyy

  # List active sessions
  nself auth sessions list

  # Logout from all devices
  nself auth logout --all

For more information, see: docs/cli/auth.md
EOF
}

# ============================================================================
# Main Auth Command Router
# ============================================================================

cmd_auth() {
  local subcommand="${1:-}"

  if [[ -z "$subcommand" ]]; then
    auth_usage
    exit 0
  fi

  shift

  case "$subcommand" in
    login)
      cmd_auth_login "$@"
      ;;
    signup)
      cmd_auth_signup "$@"
      ;;
    logout)
      cmd_auth_logout "$@"
      ;;
    status)
      cmd_auth_status "$@"
      ;;
    providers)
      cmd_auth_providers "$@"
      ;;
    sessions)
      cmd_auth_sessions "$@"
      ;;
    config)
      cmd_auth_config "$@"
      ;;
    help|--help|-h)
      auth_usage
      exit 0
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      echo ""
      auth_usage
      exit 1
      ;;
  esac
}

# ============================================================================
# Login Command
# ============================================================================

cmd_auth_login() {
  local provider=""
  local email=""
  local password=""
  local phone=""
  local anonymous=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --provider=*)
        provider="${1#*=}"
        shift
        ;;
      --email=*)
        email="${1#*=}"
        shift
        ;;
      --password=*)
        password="${1#*=}"
        shift
        ;;
      --phone=*)
        phone="${1#*=}"
        shift
        ;;
      --anonymous)
        anonymous=true
        shift
        ;;
      --help|-h)
        auth_usage
        exit 0
        ;;
      *)
        log_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  # Placeholder - will implement in AUTH-004, AUTH-005, AUTH-006, AUTH-007
  log_info "Login functionality coming in AUTH-004 through AUTH-007"
  log_info "Provider: ${provider:-none}"
  log_info "Email: ${email:-none}"
  log_info "Phone: ${phone:-none}"
  log_info "Anonymous: $anonymous"

  # TODO: Implement actual login logic
  log_warning "⚠️  Not yet implemented - Sprint 1 in progress"
}

# ============================================================================
# Signup Command
# ============================================================================

cmd_auth_signup() {
  local provider=""
  local email=""
  local password=""
  local phone=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --provider=*)
        provider="${1#*=}"
        shift
        ;;
      --email=*)
        email="${1#*=}"
        shift
        ;;
      --password=*)
        password="${1#*=}"
        shift
        ;;
      --phone=*)
        phone="${1#*=}"
        shift
        ;;
      --help|-h)
        auth_usage
        exit 0
        ;;
      *)
        log_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  # Placeholder - will implement in AUTH-004, AUTH-005, AUTH-006
  log_info "Signup functionality coming in AUTH-004 through AUTH-006"
  log_info "Provider: ${provider:-none}"
  log_info "Email: ${email:-none}"
  log_info "Phone: ${phone:-none}"

  # TODO: Implement actual signup logic
  log_warning "⚠️  Not yet implemented - Sprint 1 in progress"
}

# ============================================================================
# Logout Command
# ============================================================================

cmd_auth_logout() {
  local logout_all=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --all)
        logout_all=true
        shift
        ;;
      --help|-h)
        auth_usage
        exit 0
        ;;
      *)
        log_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  # Placeholder - will implement in AUTH-003
  log_info "Logout functionality coming in AUTH-003"
  log_info "Logout all sessions: $logout_all"

  # TODO: Implement actual logout logic
  log_warning "⚠️  Not yet implemented - Sprint 1 in progress"
}

# ============================================================================
# Status Command
# ============================================================================

cmd_auth_status() {
  # Placeholder - will implement in AUTH-003
  log_info "Auth status functionality coming in AUTH-003"

  # TODO: Implement actual status logic
  log_warning "⚠️  Not yet implemented - Sprint 1 in progress"
}

# ============================================================================
# Providers Command
# ============================================================================

cmd_auth_providers() {
  local action="${1:-list}"
  shift || true

  case "$action" in
    list)
      cmd_auth_providers_list "$@"
      ;;
    add)
      cmd_auth_providers_add "$@"
      ;;
    remove)
      cmd_auth_providers_remove "$@"
      ;;
    enable)
      cmd_auth_providers_enable "$@"
      ;;
    disable)
      cmd_auth_providers_disable "$@"
      ;;
    help|--help|-h)
      auth_usage
      exit 0
      ;;
    *)
      log_error "Unknown providers subcommand: $action"
      exit 1
      ;;
  esac
}

cmd_auth_providers_list() {
  # Call auth manager to list providers
  auth_list_providers
}

cmd_auth_providers_add() {
  local provider_name="${1:-}"

  if [[ -z "$provider_name" ]]; then
    log_error "Provider name required"
    exit 1
  fi

  shift

  # Parse provider options
  local client_id=""
  local client_secret=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --client-id=*)
        client_id="${1#*=}"
        shift
        ;;
      --client-secret=*)
        client_secret="${1#*=}"
        shift
        ;;
      *)
        log_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  # Build provider config JSON
  local config="{}"
  if [[ -n "$client_id" ]] && [[ -n "$client_secret" ]]; then
    config="{\"client_id\": \"$client_id\", \"client_secret\": \"$client_secret\"}"
  elif [[ -n "$client_id" ]]; then
    config="{\"client_id\": \"$client_id\"}"
  fi

  # Call auth manager to add provider
  auth_add_provider "$provider_name" "oauth" "$config"
}

cmd_auth_providers_remove() {
  local provider_name="${1:-}"

  if [[ -z "$provider_name" ]]; then
    log_error "Provider name required"
    exit 1
  fi

  # Call auth manager to remove provider
  auth_remove_provider "$provider_name"
}

cmd_auth_providers_enable() {
  local provider_name="${1:-}"

  if [[ -z "$provider_name" ]]; then
    log_error "Provider name required"
    exit 1
  fi

  # Call auth manager to enable provider
  auth_enable_provider "$provider_name"
}

cmd_auth_providers_disable() {
  local provider_name="${1:-}"

  if [[ -z "$provider_name" ]]; then
    log_error "Provider name required"
    exit 1
  fi

  # Call auth manager to disable provider
  auth_disable_provider "$provider_name"
}

# ============================================================================
# Sessions Command
# ============================================================================

cmd_auth_sessions() {
  local action="${1:-list}"
  shift || true

  case "$action" in
    list)
      cmd_auth_sessions_list "$@"
      ;;
    revoke)
      cmd_auth_sessions_revoke "$@"
      ;;
    help|--help|-h)
      auth_usage
      exit 0
      ;;
    *)
      log_error "Unknown sessions subcommand: $action"
      exit 1
      ;;
  esac
}

cmd_auth_sessions_list() {
  # Placeholder - will implement in AUTH-003
  log_info "Sessions list functionality coming in AUTH-003"

  # TODO: Implement actual sessions listing
  log_warning "⚠️  Not yet implemented - Sprint 1 in progress"
}

cmd_auth_sessions_revoke() {
  local session_id="${1:-}"

  if [[ -z "$session_id" ]]; then
    log_error "Session ID required"
    exit 1
  fi

  # Placeholder - will implement in AUTH-003
  log_info "Session revoke functionality coming in AUTH-003"
  log_info "Session ID: $session_id"

  # TODO: Implement actual session revoke logic
  log_warning "⚠️  Not yet implemented - Sprint 1 in progress"
}

# ============================================================================
# Config Command
# ============================================================================

cmd_auth_config() {
  local show=true
  local set_key=""
  local set_value=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --show)
        show=true
        shift
        ;;
      --set)
        show=false
        if [[ "$2" =~ ^(.+)=(.+)$ ]]; then
          set_key="${BASH_REMATCH[1]}"
          set_value="${BASH_REMATCH[2]}"
          shift 2
        else
          log_error "Invalid --set format. Use: --set key=value"
          exit 1
        fi
        ;;
      --help|-h)
        auth_usage
        exit 0
        ;;
      *)
        log_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  if $show; then
    # Show current config
    log_info "Auth config show functionality coming in AUTH-003"
    log_warning "⚠️  Not yet implemented - Sprint 1 in progress"
  else
    # Set config value
    log_info "Auth config set functionality coming in AUTH-003"
    log_info "Key: $set_key"
    log_info "Value: $set_value"
    log_warning "⚠️  Not yet implemented - Sprint 1 in progress"
  fi
}

# ============================================================================
# Export command for main CLI dispatcher
# ============================================================================

# If executed directly (for testing)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_auth "$@"
fi
