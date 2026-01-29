#!/usr/bin/env bash
# rate-limit.sh - Rate limiting CLI (RATE-010)
# Part of nself v0.6.0 - Phase 1 Sprint 5
#
# Complete CLI interface for rate limiting

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib/rate-limit"

if [[ -f "$LIB_DIR/core.sh" ]]; then
  source "$LIB_DIR/core.sh"
fi
if [[ -f "$LIB_DIR/strategies.sh" ]]; then
  source "$LIB_DIR/strategies.sh"
fi
if [[ -f "$LIB_DIR/ip-limiter.sh" ]]; then
  source "$LIB_DIR/ip-limiter.sh"
fi
if [[ -f "$LIB_DIR/user-limiter.sh" ]]; then
  source "$LIB_DIR/user-limiter.sh"
fi
if [[ -f "$LIB_DIR/endpoint-limiter.sh" ]]; then
  source "$LIB_DIR/endpoint-limiter.sh"
fi

# ============================================================================
# CLI Main
# ============================================================================

cmd_rate_limit() {
  local subcommand="${1:-help}"
  shift || true

  case "$subcommand" in
    init)
      cmd_rate_limit_init "$@"
      ;;
    check)
      cmd_rate_limit_check "$@"
      ;;
    stats)
      cmd_rate_limit_stats "$@"
      ;;
    reset)
      cmd_rate_limit_reset "$@"
      ;;
    whitelist)
      cmd_rate_limit_whitelist "$@"
      ;;
    block)
      cmd_rate_limit_block "$@"
      ;;
    rules)
      cmd_rate_limit_rules "$@"
      ;;
    quota)
      cmd_rate_limit_quota "$@"
      ;;
    cleanup)
      cmd_rate_limit_cleanup "$@"
      ;;
    help|--help|-h)
      cmd_rate_limit_help
      ;;
    *)
      echo "ERROR: Unknown command: $subcommand"
      echo "Run 'nself rate-limit help' for usage information"
      return 1
      ;;
  esac
}

# ============================================================================
# Subcommands
# ============================================================================

# Initialize rate limiter
cmd_rate_limit_init() {
  echo "Initializing rate limiter..."

  if ! rate_limit_init; then
    echo "ERROR: Failed to initialize rate limiter" >&2
    return 1
  fi

  printf "\n✓ Rate limiter initialized successfully\n"
  return 0
}

# Check rate limit
cmd_rate_limit_check() {
  local type="${1:-}"
  local identifier="${2:-}"
  local max="${3:-}"
  local window="${4:-}"

  if [[ -z "$type" ]] || [[ -z "$identifier" ]]; then
    echo "ERROR: Type and identifier required"
    echo "Usage: nself rate-limit check <ip|user|endpoint> <identifier> [max] [window]"
    return 1
  fi

  local tokens_remaining
  case "$type" in
    ip)
      tokens_remaining=$(ip_rate_limit_check "$identifier" "$max" "$window")
      ;;
    user)
      tokens_remaining=$(user_rate_limit_check "$identifier" "$max" "$window")
      ;;
    endpoint)
      tokens_remaining=$(endpoint_rate_limit_check "$identifier" "$max" "$window")
      ;;
    *)
      echo "ERROR: Unknown type: $type"
      return 1
      ;;
  esac

  local result=$?
  if [[ $result -eq 0 ]]; then
    printf "✓ Allowed (tokens remaining: %s)\n" "$tokens_remaining"
    return 0
  else
    printf "✗ Rate limited (tokens remaining: %s)\n" "$tokens_remaining"
    return 1
  fi
}

# Get rate limit stats
cmd_rate_limit_stats() {
  local type="${1:-}"
  local identifier="${2:-}"
  local hours="${3:-24}"

  if [[ -z "$type" ]] || [[ -z "$identifier" ]]; then
    echo "ERROR: Type and identifier required"
    echo "Usage: nself rate-limit stats <ip|user|endpoint> <identifier> [hours]"
    return 1
  fi

  local stats
  case "$type" in
    ip)
      local key="ip:${identifier}"
      stats=$(rate_limit_get_stats "$key" "$hours")
      ;;
    user)
      stats=$(user_get_usage "$identifier" "$hours")
      ;;
    endpoint)
      stats=$(endpoint_get_usage "$identifier" "$hours")
      ;;
    *)
      echo "ERROR: Unknown type: $type"
      return 1
      ;;
  esac

  echo "$stats" | jq '.'
  return 0
}

# Reset rate limit
cmd_rate_limit_reset() {
  local type="${1:-}"
  local identifier="${2:-}"

  if [[ -z "$type" ]] || [[ -z "$identifier" ]]; then
    echo "ERROR: Type and identifier required"
    echo "Usage: nself rate-limit reset <ip|user|endpoint> <identifier>"
    return 1
  fi

  case "$type" in
    ip)
      local key="ip:${identifier}"
      rate_limit_reset "$key"
      ;;
    user)
      user_rate_limit_reset "$identifier"
      ;;
    endpoint)
      local key="endpoint:${identifier}"
      rate_limit_reset "$key"
      ;;
    *)
      echo "ERROR: Unknown type: $type"
      return 1
      ;;
  esac

  printf "✓ Rate limit reset for %s: %s\n" "$type" "$identifier"
  return 0
}

# Manage whitelist
cmd_rate_limit_whitelist() {
  local action="${1:-list}"
  shift || true

  case "$action" in
    add)
      local ip="${1:-}"
      local description="${2:-}"

      if [[ -z "$ip" ]]; then
        echo "ERROR: IP address required"
        echo "Usage: nself rate-limit whitelist add <ip> [description]"
        return 1
      fi

      if ! ip_whitelist_add "$ip" "$description"; then
        echo "ERROR: Failed to add IP to whitelist"
        return 1
      fi

      printf "✓ Added %s to whitelist\n" "$ip"
      ;;

    remove)
      local ip="${1:-}"

      if [[ -z "$ip" ]]; then
        echo "ERROR: IP address required"
        echo "Usage: nself rate-limit whitelist remove <ip>"
        return 1
      fi

      if ! ip_whitelist_remove "$ip"; then
        echo "ERROR: Failed to remove IP from whitelist"
        return 1
      fi

      printf "✓ Removed %s from whitelist\n" "$ip"
      ;;

    list)
      local whitelist
      whitelist=$(ip_whitelist_list)

      if [[ "$whitelist" == "[]" ]]; then
        echo "No whitelisted IPs"
        return 0
      fi

      echo "$whitelist" | jq -r '["IP", "ENABLED", "DESCRIPTION", "CREATED"],
        (.[] | [.ip_address, .enabled, .description, .created_at]) | @tsv' | column -t
      ;;

    *)
      echo "ERROR: Unknown action: $action"
      echo "Available: add, remove, list"
      return 1
      ;;
  esac

  return 0
}

# Manage IP blocking
cmd_rate_limit_block() {
  local action="${1:-list}"
  shift || true

  case "$action" in
    add)
      local ip="${1:-}"
      local reason="${2:-Rate limit exceeded}"
      local duration="${3:-3600}"

      if [[ -z "$ip" ]]; then
        echo "ERROR: IP address required"
        echo "Usage: nself rate-limit block add <ip> [reason] [duration_seconds]"
        return 1
      fi

      if ! ip_block "$ip" "$reason" "$duration"; then
        echo "ERROR: Failed to block IP"
        return 1
      fi

      printf "✓ Blocked %s for %s seconds\n" "$ip" "$duration"
      ;;

    remove)
      local ip="${1:-}"

      if [[ -z "$ip" ]]; then
        echo "ERROR: IP address required"
        echo "Usage: nself rate-limit block remove <ip>"
        return 1
      fi

      if ! ip_unblock "$ip"; then
        echo "ERROR: Failed to unblock IP"
        return 1
      fi

      printf "✓ Unblocked %s\n" "$ip"
      ;;

    check)
      local ip="${1:-}"

      if [[ -z "$ip" ]]; then
        echo "ERROR: IP address required"
        return 1
      fi

      if ip_is_blocked "$ip"; then
        printf "%s is BLOCKED\n" "$ip"
        return 0
      else
        printf "%s is NOT blocked\n" "$ip"
        return 1
      fi
      ;;

    *)
      echo "ERROR: Unknown action: $action"
      echo "Available: add, remove, check"
      return 1
      ;;
  esac

  return 0
}

# Manage endpoint rules
cmd_rate_limit_rules() {
  local action="${1:-list}"
  shift || true

  case "$action" in
    create)
      local name="${1:-}"
      local pattern="${2:-}"
      local max="${3:-}"
      local window="${4:-}"
      local priority="${5:-100}"

      if [[ -z "$name" ]] || [[ -z "$pattern" ]] || [[ -z "$max" ]] || [[ -z "$window" ]]; then
        echo "ERROR: Name, pattern, max requests, and window required"
        echo "Usage: nself rate-limit rules create <name> <pattern> <max> <window> [priority]"
        return 1
      fi

      local rule_id
      rule_id=$(endpoint_rule_create "$name" "$pattern" "$max" "$window" "$priority")

      if [[ $? -ne 0 ]]; then
        echo "ERROR: Failed to create rule"
        return 1
      fi

      printf "✓ Created rule: %s (ID: %s)\n" "$name" "$rule_id"
      ;;

    list)
      local rules
      rules=$(endpoint_rule_list)

      if [[ "$rules" == "[]" ]]; then
        echo "No rules configured"
        return 0
      fi

      echo "$rules" | jq -r '["NAME", "PATTERN", "MAX", "WINDOW", "ENABLED", "PRIORITY"],
        (.[] | [.name, .pattern, .max_requests, .window_seconds, .enabled, .priority]) | @tsv' | column -t
      ;;

    enable)
      local name="${1:-}"

      if [[ -z "$name" ]]; then
        echo "ERROR: Rule name required"
        return 1
      fi

      endpoint_rule_set_enabled "$name" true
      printf "✓ Enabled rule: %s\n" "$name"
      ;;

    disable)
      local name="${1:-}"

      if [[ -z "$name" ]]; then
        echo "ERROR: Rule name required"
        return 1
      fi

      endpoint_rule_set_enabled "$name" false
      printf "✓ Disabled rule: %s\n" "$name"
      ;;

    delete)
      local name="${1:-}"

      if [[ -z "$name" ]]; then
        echo "ERROR: Rule name required"
        return 1
      fi

      endpoint_rule_delete "$name"
      printf "✓ Deleted rule: %s\n" "$name"
      ;;

    *)
      echo "ERROR: Unknown action: $action"
      echo "Available: create, list, enable, disable, delete"
      return 1
      ;;
  esac

  return 0
}

# Manage user quotas
cmd_rate_limit_quota() {
  local action="${1:-get}"
  shift || true

  case "$action" in
    set)
      local user_id="${1:-}"
      local max="${2:-}"
      local window="${3:-}"

      if [[ -z "$user_id" ]] || [[ -z "$max" ]] || [[ -z "$window" ]]; then
        echo "ERROR: User ID, max requests, and window required"
        echo "Usage: nself rate-limit quota set <user_id> <max> <window>"
        return 1
      fi

      user_quota_set "$user_id" "$max" "$window"
      printf "✓ Set quota for user %s: %s requests per %s seconds\n" "$user_id" "$max" "$window"
      ;;

    get)
      local user_id="${1:-}"

      if [[ -z "$user_id" ]]; then
        echo "ERROR: User ID required"
        echo "Usage: nself rate-limit quota get <user_id>"
        return 1
      fi

      local quota
      quota=$(user_quota_get "$user_id")

      echo "$quota" | jq '.'
      ;;

    *)
      echo "ERROR: Unknown action: $action"
      echo "Available: set, get"
      return 1
      ;;
  esac

  return 0
}

# Cleanup old logs
cmd_rate_limit_cleanup() {
  local days="${1:-7}"

  if ! rate_limit_cleanup "$days"; then
    echo "ERROR: Failed to cleanup logs"
    return 1
  fi

  printf "✓ Cleaned up logs older than %s days\n" "$days"
  return 0
}

# ============================================================================
# Help
# ============================================================================

cmd_rate_limit_help() {
  cat <<'EOF'
nself rate-limit - Rate limiting and throttling management

USAGE:
  nself rate-limit <command> [options]

COMMANDS:
  init                 Initialize rate limiter
  check                Check rate limit for IP/user/endpoint
  stats                Get rate limit statistics
  reset                Reset rate limit for IP/user/endpoint
  whitelist            Manage IP whitelist
  block                Manage IP blocklist
  rules                Manage endpoint rules
  quota                Manage user quotas
  cleanup              Clean up old rate limit logs

EXAMPLES:
  # Initialize
  nself rate-limit init

  # Check rate limits
  nself rate-limit check ip 192.168.1.1
  nself rate-limit check user <user-uuid>
  nself rate-limit check endpoint /api/users

  # Get statistics
  nself rate-limit stats ip 192.168.1.1
  nself rate-limit stats user <user-uuid> 24
  nself rate-limit stats endpoint /api/login

  # Reset rate limits
  nself rate-limit reset ip 192.168.1.1
  nself rate-limit reset user <user-uuid>

  # Whitelist management
  nself rate-limit whitelist add 10.0.0.1 "Internal server"
  nself rate-limit whitelist list
  nself rate-limit whitelist remove 10.0.0.1

  # IP blocking
  nself rate-limit block add 1.2.3.4 "Abuse detected" 7200
  nself rate-limit block check 1.2.3.4
  nself rate-limit block remove 1.2.3.4

  # Endpoint rules
  nself rate-limit rules create login-limit "^/api/login" 5 300
  nself rate-limit rules list
  nself rate-limit rules enable login-limit
  nself rate-limit rules disable login-limit
  nself rate-limit rules delete login-limit

  # User quotas
  nself rate-limit quota set <user-uuid> 1000 3600
  nself rate-limit quota get <user-uuid>

  # Cleanup
  nself rate-limit cleanup 7

For more information: https://docs.nself.org/rate-limiting
EOF
}

# ============================================================================
# Export
# ============================================================================

export -f cmd_rate_limit

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_rate_limit "$@"
fi
