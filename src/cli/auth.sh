#!/usr/bin/env bash
# auth.sh - Authentication & Security Management CLI
# Part of nself - Consolidated authentication and security commands
#
# Commands (38 subcommands total):
#   Authentication:
#     nself auth login [--provider=<provider>] [--email=<email>] [--password=<password>]
#     nself auth logout [--all]
#     nself auth status
#
#   MFA (Multi-Factor Authentication):
#     nself auth mfa enable [--method=totp|sms|email] [--user=<id>]
#     nself auth mfa disable [--method=<method>] [--user=<id>]
#     nself auth mfa verify [--method=<method>] [--user=<id>] [--code=<code>]
#     nself auth mfa backup-codes [generate|list|status] [--user=<id>]
#
#   Roles & Permissions:
#     nself auth roles list
#     nself auth roles create [--name=<name>] [--description=<desc>]
#     nself auth roles assign [--user=<id>] [--role=<name>]
#     nself auth roles remove [--user=<id>] [--role=<name>]
#
#   Devices:
#     nself auth devices list <user_id>
#     nself auth devices register <device>
#     nself auth devices revoke <device>
#     nself auth devices trust <device>
#
#   OAuth:
#     nself auth oauth install
#     nself auth oauth enable <provider>
#     nself auth oauth disable <provider>
#     nself auth oauth config <provider> [--client-id=<id>] [--client-secret=<secret>]
#     nself auth oauth test <provider>
#     nself auth oauth list
#     nself auth oauth status
#
#   Security:
#     nself auth security scan [--deep]
#     nself auth security audit
#     nself auth security report
#
#   SSL Management:
#     nself auth ssl generate [domain]
#     nself auth ssl install <cert>
#     nself auth ssl renew [domain]
#     nself auth ssl info [domain]
#     nself auth ssl trust
#
#   Rate Limiting:
#     nself auth rate-limit config [options]
#     nself auth rate-limit status
#     nself auth rate-limit reset [ip]
#
#   Webhooks:
#     nself auth webhooks create <url> [events]
#     nself auth webhooks list
#     nself auth webhooks delete <id>
#     nself auth webhooks test <id>
#     nself auth webhooks logs <id>
#
# Usage: nself auth <subcommand> [options]

set -euo pipefail

# Get script directory for sourcing dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Save CLI directory before sourcing other modules (which may override SCRIPT_DIR)
CLI_DIR="$SCRIPT_DIR"

# Source dependencies
if [[ -z "${CLI_OUTPUT_SOURCED:-}" ]]; then
  source "$NSELF_ROOT/src/lib/utils/cli-output.sh" 2>/dev/null || true
fi

if [[ -z "${EXIT_SUCCESS:-}" ]]; then
  source "$NSELF_ROOT/src/lib/config/constants.sh" 2>/dev/null || true
fi

# Source auth manager
if [[ -f "$NSELF_ROOT/src/lib/auth/auth-manager.sh" ]]; then
  source "$NSELF_ROOT/src/lib/auth/auth-manager.sh"
fi

# ============================================================================
# Command Functions
# ============================================================================

# Show usage information
auth_usage() {
  cat <<EOF
Usage: nself auth <subcommand> [options]

Authentication and Security management for nself

AUTHENTICATION:
  login              Authenticate user
  logout             End session
  status             Show current auth status

MFA (MULTI-FACTOR AUTHENTICATION):
  mfa enable         Enable MFA for a user
  mfa disable        Disable MFA for a user
  mfa verify         Verify MFA code
  mfa backup-codes   Manage backup codes

ROLES & PERMISSIONS:
  roles list         List all roles
  roles create       Create a new role
  roles assign       Assign role to user
  roles remove       Remove role from user

DEVICES:
  devices list       List user's devices
  devices register   Register a new device
  devices revoke     Revoke device access
  devices trust      Trust a device

OAUTH:
  oauth install      Install OAuth handlers service
  oauth enable       Enable OAuth provider
  oauth disable      Disable OAuth provider
  oauth config       Configure OAuth credentials
  oauth test         Test OAuth provider
  oauth list         List OAuth providers
  oauth status       Show OAuth service status

SECURITY:
  security scan      Security vulnerability scan
  security audit     Security audit
  security report    Generate security report

SSL:
  ssl generate       Generate SSL certificate
  ssl install        Install SSL certificate
  ssl renew          Renew SSL certificate
  ssl info           Show certificate info
  ssl trust          Trust local certificates

RATE LIMITING:
  rate-limit config  Configure rate limits
  rate-limit status  Rate limit status
  rate-limit reset   Reset rate limits

WEBHOOKS:
  webhooks create    Create webhook
  webhooks list      List webhooks
  webhooks delete    Delete webhook
  webhooks test      Test webhook
  webhooks logs      Webhook logs

EXAMPLES:
  # Email/password login
  nself auth login --email=user@example.com --password=secret

  # Enable MFA
  nself auth mfa enable --method=totp --user=<user_id>

  # List roles
  nself auth roles list

  # Configure OAuth
  nself auth oauth config google --client-id=xxx --client-secret=yyy

  # Security scan
  nself auth security scan

  # Generate SSL certificate
  nself auth ssl generate

  # Trust local certificates
  nself auth ssl trust

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
    # Authentication
    login)
      cmd_auth_login "$@"
      ;;
    logout)
      cmd_auth_logout "$@"
      ;;
    status)
      cmd_auth_status "$@"
      ;;

    # MFA
    mfa)
      cmd_auth_mfa "$@"
      ;;

    # Roles
    roles)
      cmd_auth_roles "$@"
      ;;

    # Devices
    devices)
      cmd_auth_devices "$@"
      ;;

    # OAuth
    oauth)
      cmd_auth_oauth "$@"
      ;;

    # Security
    security)
      cmd_auth_security "$@"
      ;;

    # SSL
    ssl)
      cmd_auth_ssl "$@"
      ;;

    # Rate Limiting
    rate-limit)
      cmd_auth_rate_limit "$@"
      ;;

    # Webhooks
    webhooks)
      cmd_auth_webhooks "$@"
      ;;

    help|--help|-h)
      auth_usage
      exit 0
      ;;
    *)
      cli_error "Unknown subcommand: $subcommand"
      printf "\n"
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
        cli_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  # Route to appropriate auth method
  if [[ -n "$email" ]] && [[ -n "$password" ]]; then
    auth_login_email "$email" "$password"
  elif [[ -n "$provider" ]]; then
    cli_warning "OAuth login not yet implemented (OAUTH-003+)"
    auth_login_oauth "$provider"
  elif [[ -n "$phone" ]]; then
    cli_warning "Phone login not yet implemented (AUTH-006)"
    auth_login_phone "$phone"
  elif $anonymous; then
    cli_warning "Anonymous login not yet implemented (AUTH-007)"
    auth_login_anonymous
  elif [[ -n "$email" ]]; then
    cli_warning "Magic link not yet implemented (AUTH-005)"
    auth_login_magic_link "$email"
  else
    cli_error "Please provide login credentials"
    printf "Options: --email and --password, --provider, --phone, or --anonymous\n"
    exit 1
  fi
}

# ============================================================================
# Logout Command
# ============================================================================

cmd_auth_logout() {
  local logout_all=false

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
        cli_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  cli_info "Logout functionality coming in AUTH-003"
  cli_info "Logout all sessions: $logout_all"
  cli_warning "Not yet implemented - Sprint 1 in progress"
}

# ============================================================================
# Status Command
# ============================================================================

cmd_auth_status() {
  cli_info "Auth status functionality coming in AUTH-003"
  cli_warning "Not yet implemented - Sprint 1 in progress"
}

# ============================================================================
# MFA Commands
# ============================================================================

cmd_auth_mfa() {
  local action="${1:-}"

  if [[ -z "$action" ]]; then
    cli_error "MFA action required"
    printf "Actions: enable, disable, verify, backup-codes\n"
    exit 1
  fi

  shift

  # Delegate to original mfa implementation
  if [[ -f "$CLI_DIR/_deprecated/mfa.sh.backup" ]]; then
    bash "$CLI_DIR/_deprecated/mfa.sh.backup" "$action" "$@"
  else
    cli_error "MFA module not found"
    exit 1
  fi
}

# ============================================================================
# Roles Commands
# ============================================================================

cmd_auth_roles() {
  local action="${1:-list}"
  shift || true

  # Delegate to original roles implementation
  if [[ -f "$CLI_DIR/_deprecated/roles.sh.backup" ]]; then
    bash "$CLI_DIR/_deprecated/roles.sh.backup" "$action" "$@"
  else
    cli_error "Roles module not found"
    exit 1
  fi
}

# ============================================================================
# Devices Commands
# ============================================================================

cmd_auth_devices() {
  local action="${1:-list}"
  shift || true

  # Delegate to original devices implementation
  if [[ -f "$CLI_DIR/_deprecated/devices.sh.backup" ]]; then
    bash "$CLI_DIR/_deprecated/devices.sh.backup" "$action" "$@"
  else
    cli_error "Devices module not found"
    exit 1
  fi
}

# ============================================================================
# OAuth Commands
# ============================================================================

cmd_auth_oauth() {
  local action="${1:-}"

  if [[ -z "$action" ]]; then
    cli_error "OAuth action required"
    printf "Actions: install, enable, disable, config, test, list, status\n"
    exit 1
  fi

  shift

  # Delegate to original oauth implementation
  if [[ -f "$CLI_DIR/_deprecated/oauth.sh.backup" ]]; then
    bash "$CLI_DIR/_deprecated/oauth.sh.backup" "$action" "$@"
  else
    cli_error "OAuth module not found"
    exit 1
  fi
}

# ============================================================================
# Security Commands
# ============================================================================

cmd_auth_security() {
  local action="${1:-scan}"
  shift || true

  # Delegate to original security implementation
  if [[ -f "$CLI_DIR/_deprecated/security.sh.backup" ]]; then
    bash "$CLI_DIR/_deprecated/security.sh.backup" "$action" "$@"
  else
    cli_error "Security module not found"
    exit 1
  fi
}

# ============================================================================
# SSL Commands
# ============================================================================

cmd_auth_ssl() {
  local action="${1:-}"

  if [[ -z "$action" ]]; then
    cli_error "SSL action required"
    printf "Actions: generate, install, renew, info, trust\n"
    exit 1
  fi

  shift

  case "$action" in
    generate)
      # Delegate to original ssl implementation
      if [[ -f "$CLI_DIR/_deprecated/ssl.sh.backup" ]]; then
        bash "$CLI_DIR/_deprecated/ssl.sh.backup" bootstrap "$@"
      else
        cli_error "SSL module not found"
        exit 1
      fi
      ;;
    renew)
      if [[ -f "$CLI_DIR/_deprecated/ssl.sh.backup" ]]; then
        bash "$CLI_DIR/_deprecated/ssl.sh.backup" renew "$@"
      else
        cli_error "SSL module not found"
        exit 1
      fi
      ;;
    info|status)
      if [[ -f "$CLI_DIR/_deprecated/ssl.sh.backup" ]]; then
        bash "$CLI_DIR/_deprecated/ssl.sh.backup" status "$@"
      else
        cli_error "SSL module not found"
        exit 1
      fi
      ;;
    trust)
      # Delegate to original trust implementation
      if [[ -f "$CLI_DIR/_deprecated/trust.sh.backup" ]]; then
        bash "$CLI_DIR/_deprecated/trust.sh.backup" install "$@"
      else
        cli_error "Trust module not found"
        exit 1
      fi
      ;;
    install)
      cli_warning "SSL installation is handled automatically by 'nself build'"
      cli_info "To manually trust certificates, use: nself auth ssl trust"
      ;;
    *)
      cli_error "Unknown SSL action: $action"
      printf "Actions: generate, install, renew, info, trust\n"
      exit 1
      ;;
  esac
}

# ============================================================================
# Rate Limit Commands
# ============================================================================

cmd_auth_rate_limit() {
  local action="${1:-status}"
  shift || true

  # Delegate to original rate-limit implementation
  if [[ -f "$CLI_DIR/_deprecated/rate-limit.sh.backup" ]]; then
    bash "$CLI_DIR/_deprecated/rate-limit.sh.backup" "$action" "$@"
  else
    cli_error "Rate limiting module not found"
    exit 1
  fi
}

# ============================================================================
# Webhooks Commands
# ============================================================================

cmd_auth_webhooks() {
  local action="${1:-list}"
  shift || true

  # Delegate to original webhooks implementation
  if [[ -f "$CLI_DIR/_deprecated/webhooks.sh.backup" ]]; then
    bash "$CLI_DIR/_deprecated/webhooks.sh.backup" "$action" "$@"
  else
    cli_error "Webhooks module not found"
    exit 1
  fi
}

# ============================================================================
# Export command for main CLI dispatcher
# ============================================================================

# If executed directly (for testing)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_auth "$@"
fi
