#!/usr/bin/env bash
# notify.sh - notification channel management for nself
# Manages notification channels and send dispatch for the nself-notify pro plugin.
#
# Commands:
#   nself notify send <channel> <message>                                Send a notification
#   nself notify send --channel <name> --msg <text>                      Alternative form
#   nself notify channels list                                           List configured channels
#   nself notify channels add telegram --token <tok> --chat-id <id>     Add Telegram channel
#   nself notify channels add webhook --name <name> --url <url>         Add webhook channel
#   nself notify channels add slack --name <name> --webhook-url <url>   Add Slack channel
#   nself notify channels add email --name <name> --to <addr>           Add email channel
#   nself notify channels test <channel>                                 Test send to channel
#   nself notify channels remove <channel>                               Remove channel
#   nself notify log [--limit N] [--channel <name>] [--json]            Show recent dispatch log
#   nself notify status                                                  Show plugin status
#
# Usage: nself notify <subcommand> [options]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source display helpers
source "$NSELF_ROOT/src/lib/utils/cli-output.sh" 2>/dev/null || true
source "$NSELF_ROOT/src/lib/utils/display.sh" 2>/dev/null || true

# Fallbacks if display helpers didn't load
if ! type cli_error >/dev/null 2>&1; then
  cli_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi
if ! type log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
fi
if ! type log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
fi
if ! type log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi

# ============================================================================
# Usage
# ============================================================================

notify_usage() {
  printf "nself notify — notification channel management\n\n"
  printf "Usage: nself notify <subcommand> [options]\n\n"
  printf "Subcommands:\n"
  printf "  send <channel> <message>                                     Send a notification\n"
  printf "  send --channel <name> --msg <text>                           Alternative form\n"
  printf "  channels list                                                List configured channels\n"
  printf "  channels add telegram --token <tok> --chat-id <id>          Add Telegram channel\n"
  printf "  channels add webhook --name <name> --url <url>              Add webhook channel\n"
  printf "  channels add slack --name <name> --webhook-url <url>        Add Slack channel\n"
  printf "  channels add email --name <name> --to <addr>                Add email channel\n"
  printf "  channels test <channel>                                      Test send to channel\n"
  printf "  channels remove <channel>                                    Remove channel\n"
  printf "  log [--limit N] [--channel <name>] [--json]                 Show recent dispatch log\n"
  printf "  status                                                        Show plugin status\n\n"
  printf "Environment:\n"
  printf "  NOTIFY_PORT             notify plugin port (default: 3712)\n"
  printf "  PLUGIN_INTERNAL_SECRET  required for all commands\n\n"
  printf "Examples:\n"
  printf "  nself notify send alerts 'Deploy complete'\n"
  printf "  nself notify send --channel alerts --msg 'Deploy complete'\n"
  printf "  nself notify channels list\n"
  printf "  nself notify channels add telegram --token <tok> --chat-id <id>\n"
  printf "  nself notify channels add slack --name ops --webhook-url <url>\n"
  printf "  nself notify channels test alerts\n"
  printf "  nself notify log --limit 20\n"
  printf "  nself notify status\n"
}

# ============================================================================
# Plugin connectivity check
# ============================================================================

_notify_base_url() {
  printf "http://127.0.0.1:%s" "${NOTIFY_PORT:-3712}"
}

_notify_check_running() {
  local base_url
  base_url="$(_notify_base_url)"
  if ! curl -s --max-time 2 "${base_url}/health" >/dev/null 2>&1; then
    cli_error "Notify plugin not running. Install: nself plugin install notify"
    return 1
  fi
}

# ============================================================================
# send subcommand
# ============================================================================

cmd_send() {
  local channel="" message=""

  # Support both positional and flag-based forms
  if [ "${1:-}" = "--channel" ] || [ "${1:-}" = "--msg" ]; then
    # Flag form: --channel <name> --msg <text>
    while [ $# -gt 0 ]; do
      case "$1" in
        --channel) channel="$2"; shift 2 ;;
        --msg)     message="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
  else
    # Positional form: send <channel> <message>
    channel="${1:-}"
    message="${2:-}"
  fi

  if [ -z "$channel" ]; then
    cli_error "Channel name required"
    printf "Usage: nself notify send <channel> <message>\n" >&2
    printf "   or: nself notify send --channel <name> --msg <text>\n" >&2
    return 1
  fi

  if [ -z "$message" ]; then
    cli_error "Message required"
    printf "Usage: nself notify send <channel> <message>\n" >&2
    return 1
  fi

  _notify_check_running || return 1

  local base_url
  base_url="$(_notify_base_url)"

  local safe_channel="" safe_message=""
  safe_channel=$(printf '%s' "$channel" | sed 's/\\/\\\\/g; s/"/\\"/g')
  safe_message=$(printf '%s' "$message" | sed 's/\\/\\\\/g; s/"/\\"/g')

  local payload="{\"channel\":\"${safe_channel}\",\"message\":\"${safe_message}\"}"

  local resp=""
  resp=$(curl -s -X POST "${base_url}/notify" \
    -H "Content-Type: application/json" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" \
    -d "$payload" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from notify plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local status=""
    status=$(printf '%s' "$resp" | jq -r '.status // .error // "unknown"' 2>/dev/null || true)
    if [ "$status" = "sent" ] || [ "$status" = "ok" ] || [ "$status" = "success" ]; then
      log_success "Notification sent to channel: ${channel}"
    else
      cli_error "Send failed: ${status}"
      return 1
    fi
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# channels subcommand dispatcher
# ============================================================================

cmd_channels() {
  local subcmd="${1:-}"

  if [ -z "$subcmd" ]; then
    cli_error "channels action required"
    printf "Actions: list, add, test, remove\n"
    return 1
  fi

  shift

  case "$subcmd" in
    list)    cmd_channels_list   "$@" ;;
    add)     cmd_channels_add    "$@" ;;
    test)    cmd_channels_test   "$@" ;;
    remove)  cmd_channels_remove "$@" ;;
    help | --help | -h)
      notify_usage
      exit 0
      ;;
    *)
      cli_error "Unknown channels action: $subcmd"
      printf "Actions: list, add, test, remove\n"
      return 1
      ;;
  esac
}

cmd_channels_list() {
  _notify_check_running || return 1

  local base_url
  base_url="$(_notify_base_url)"

  local resp=""
  resp=$(curl -s "${base_url}/channels" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from notify plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local count=""
    count=$(printf '%s' "$resp" | jq 'if type == "array" then length else 0 end' 2>/dev/null || printf "0")
    if [ "$count" = "0" ]; then
      log_info "No channels configured."
      printf "Add one with: nself notify channels add <type> [options]\n"
      return 0
    fi

    printf "\033[34m%-20s %-12s %s\033[0m\n" "Name" "Type" "Status"
    printf "%-20s %-12s %s\n" "--------------------" "------------" "-------"

    printf '%s' "$resp" | jq -r '.[] | [.name // "?", .type // "?", (if .enabled // true then "enabled" else "disabled" end)] | @tsv' 2>/dev/null \
      | while IFS=$(printf '\t') read -r ch_name ch_type ch_status; do
          if [ "$ch_status" = "enabled" ]; then
            printf "\033[32m%-20s %-12s %s\033[0m\n" "$ch_name" "$ch_type" "$ch_status"
          else
            printf "\033[33m%-20s %-12s %s\033[0m\n" "$ch_name" "$ch_type" "$ch_status"
          fi
        done
  else
    printf '%s\n' "$resp"
  fi
}

cmd_channels_add() {
  local ch_type="${1:-}"

  if [ -z "$ch_type" ]; then
    cli_error "Channel type required"
    printf "Types: telegram, webhook, slack, email\n"
    return 1
  fi

  shift

  local name="" token="" chat_id="" url="" webhook_url="" to_addr=""

  while [ $# -gt 0 ]; do
    case "$1" in
      --token)       token="$2";       shift 2 ;;
      --chat-id)     chat_id="$2";     shift 2 ;;
      --name)        name="$2";        shift 2 ;;
      --url)         url="$2";         shift 2 ;;
      --webhook-url) webhook_url="$2"; shift 2 ;;
      --to)          to_addr="$2";     shift 2 ;;
      *) shift ;;
    esac
  done

  _notify_check_running || return 1

  local base_url
  base_url="$(_notify_base_url)"
  local payload=""

  case "$ch_type" in
    telegram)
      if [ -z "$token" ] || [ -z "$chat_id" ]; then
        cli_error "Telegram channel requires --token and --chat-id"
        printf "Usage: nself notify channels add telegram --token <tok> --chat-id <id>\n" >&2
        return 1
      fi
      local safe_token="" safe_chat_id=""
      safe_token=$(printf '%s' "$token" | sed 's/\\/\\\\/g; s/"/\\"/g')
      safe_chat_id=$(printf '%s' "$chat_id" | sed 's/\\/\\\\/g; s/"/\\"/g')
      payload="{\"type\":\"telegram\",\"token\":\"${safe_token}\",\"chat_id\":\"${safe_chat_id}\"}"
      ;;
    webhook)
      if [ -z "$name" ] || [ -z "$url" ]; then
        cli_error "Webhook channel requires --name and --url"
        printf "Usage: nself notify channels add webhook --name <name> --url <url>\n" >&2
        return 1
      fi
      local safe_name_wh="" safe_url=""
      safe_name_wh=$(printf '%s' "$name" | sed 's/\\/\\\\/g; s/"/\\"/g')
      safe_url=$(printf '%s' "$url" | sed 's/\\/\\\\/g; s/"/\\"/g')
      payload="{\"type\":\"webhook\",\"name\":\"${safe_name_wh}\",\"url\":\"${safe_url}\"}"
      ;;
    slack)
      if [ -z "$name" ] || [ -z "$webhook_url" ]; then
        cli_error "Slack channel requires --name and --webhook-url"
        printf "Usage: nself notify channels add slack --name <name> --webhook-url <url>\n" >&2
        return 1
      fi
      local safe_name_sl="" safe_wh_url=""
      safe_name_sl=$(printf '%s' "$name" | sed 's/\\/\\\\/g; s/"/\\"/g')
      safe_wh_url=$(printf '%s' "$webhook_url" | sed 's/\\/\\\\/g; s/"/\\"/g')
      payload="{\"type\":\"slack\",\"name\":\"${safe_name_sl}\",\"webhook_url\":\"${safe_wh_url}\"}"
      ;;
    email)
      if [ -z "$name" ] || [ -z "$to_addr" ]; then
        cli_error "Email channel requires --name and --to"
        printf "Usage: nself notify channels add email --name <name> --to <addr>\n" >&2
        return 1
      fi
      local safe_name_em="" safe_to=""
      safe_name_em=$(printf '%s' "$name" | sed 's/\\/\\\\/g; s/"/\\"/g')
      safe_to=$(printf '%s' "$to_addr" | sed 's/\\/\\\\/g; s/"/\\"/g')
      payload="{\"type\":\"email\",\"name\":\"${safe_name_em}\",\"to\":\"${safe_to}\"}"
      ;;
    *)
      cli_error "Unknown channel type: ${ch_type}"
      printf "Types: telegram, webhook, slack, email\n"
      return 1
      ;;
  esac

  local resp=""
  resp=$(curl -s -X POST "${base_url}/channels" \
    -H "Content-Type: application/json" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" \
    -d "$payload" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from notify plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local ch_name_out=""
    ch_name_out=$(printf '%s' "$resp" | jq -r '.name // .id // "channel"' 2>/dev/null || true)
    local err_out=""
    err_out=$(printf '%s' "$resp" | jq -r '.error // ""' 2>/dev/null || true)
    if [ -n "$err_out" ]; then
      cli_error "Failed to add channel: ${err_out}"
      return 1
    fi
    log_success "Channel added: ${ch_name_out}"
  else
    printf '%s\n' "$resp"
  fi
}

cmd_channels_test() {
  local channel="${1:-}"

  if [ -z "$channel" ]; then
    cli_error "Channel name required"
    printf "Usage: nself notify channels test <channel>\n" >&2
    return 1
  fi

  _notify_check_running || return 1

  local base_url
  base_url="$(_notify_base_url)"

  local safe_channel=""
  safe_channel=$(printf '%s' "$channel" | sed 's/ /%20/g')

  log_info "Sending test notification to channel: ${channel}"

  local resp=""
  resp=$(curl -s -X POST "${base_url}/channels/${safe_channel}/test" \
    -H "Content-Type: application/json" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from notify plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local status=""
    status=$(printf '%s' "$resp" | jq -r '.status // .error // "unknown"' 2>/dev/null || true)
    if [ "$status" = "sent" ] || [ "$status" = "ok" ] || [ "$status" = "success" ]; then
      log_success "Test notification sent to: ${channel}"
    else
      cli_error "Test failed: ${status}"
      return 1
    fi
  else
    printf '%s\n' "$resp"
  fi
}

cmd_channels_remove() {
  local channel="${1:-}"

  if [ -z "$channel" ]; then
    cli_error "Channel name required"
    printf "Usage: nself notify channels remove <channel>\n" >&2
    return 1
  fi

  _notify_check_running || return 1

  local base_url
  base_url="$(_notify_base_url)"

  local safe_channel=""
  safe_channel=$(printf '%s' "$channel" | sed 's/ /%20/g')

  local resp=""
  resp=$(curl -s -X DELETE "${base_url}/channels/${safe_channel}" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from notify plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local err_out=""
    err_out=$(printf '%s' "$resp" | jq -r '.error // ""' 2>/dev/null || true)
    if [ -n "$err_out" ]; then
      cli_error "Failed to remove channel: ${err_out}"
      return 1
    fi
    log_success "Channel removed: ${channel}"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# log subcommand
# ============================================================================

cmd_log() {
  local limit="50"
  local channel=""
  local json_output="false"

  while [ $# -gt 0 ]; do
    case "$1" in
      --limit)   limit="$2";   shift 2 ;;
      --channel) channel="$2"; shift 2 ;;
      --json)    json_output="true"; shift ;;
      *) shift ;;
    esac
  done

  _notify_check_running || return 1

  local base_url
  base_url="$(_notify_base_url)"

  local query="limit=${limit}"
  if [ -n "$channel" ]; then
    local safe_ch=""
    safe_ch=$(printf '%s' "$channel" | sed 's/ /%20/g')
    query="${query}&channel=${safe_ch}"
  fi

  local resp=""
  resp=$(curl -s "${base_url}/log?${query}" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from notify plugin at ${base_url}"
    return 1
  fi

  if [ "$json_output" = "true" ]; then
    printf '%s\n' "$resp"
    return 0
  fi

  if command -v jq >/dev/null 2>&1; then
    local count=""
    count=$(printf '%s' "$resp" | jq 'if type == "array" then length else 0 end' 2>/dev/null || printf "0")
    if [ "$count" = "0" ]; then
      log_info "No dispatch log entries found."
      return 0
    fi

    printf "\033[34m%-20s %-15s %-10s %s\033[0m\n" "Time" "Channel" "Status" "Message"
    printf "%-20s %-15s %-10s %s\n" "--------------------" "---------------" "----------" "-------"

    printf '%s' "$resp" | jq -r '.[] | [.created_at // .timestamp // "?", .channel // "?", .status // "?", (.message // "" | gsub("\n"; " ") | .[0:60])] | @tsv' 2>/dev/null \
      | while IFS=$(printf '\t') read -r ts ch st msg; do
          if [ "$st" = "sent" ] || [ "$st" = "ok" ] || [ "$st" = "success" ]; then
            printf "\033[32m%-20s %-15s %-10s %s\033[0m\n" "$ts" "$ch" "$st" "$msg"
          else
            printf "\033[31m%-20s %-15s %-10s %s\033[0m\n" "$ts" "$ch" "$st" "$msg"
          fi
        done
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# status subcommand
# ============================================================================

cmd_status() {
  _notify_check_running || return 1

  local base_url
  base_url="$(_notify_base_url)"

  local resp=""
  resp=$(curl -s "${base_url}/health" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from notify plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local plugin_status=""
    plugin_status=$(printf '%s' "$resp" | jq -r '.status // "unknown"' 2>/dev/null || true)
    local version=""
    version=$(printf '%s' "$resp" | jq -r '.version // ""' 2>/dev/null || true)
    local channels_count=""
    channels_count=$(printf '%s' "$resp" | jq -r '.channels // .channel_count // ""' 2>/dev/null || true)

    printf "Notify plugin\n"
    printf "  Status:   %s\n" "$plugin_status"
    if [ -n "$version" ]; then
      printf "  Version:  %s\n" "$version"
    fi
    if [ -n "$channels_count" ]; then
      printf "  Channels: %s\n" "$channels_count"
    fi
    printf "  Endpoint: %s\n" "$base_url"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# Top-level dispatcher
# ============================================================================

cmd_notify() {
  local subcommand="${1:-}"

  if [ -z "$subcommand" ]; then
    notify_usage
    exit 0
  fi

  case "$subcommand" in
    --help | -h)
      notify_usage
      exit 0
      ;;
    *) ;;
  esac

  shift

  case "$subcommand" in
    send)
      cmd_send "$@"
      ;;
    channels)
      cmd_channels "$@"
      ;;
    log)
      cmd_log "$@"
      ;;
    status)
      cmd_status "$@"
      ;;
    help | --help | -h)
      notify_usage
      exit 0
      ;;
    *)
      cli_error "Unknown subcommand: $subcommand"
      printf "\n"
      notify_usage
      exit 1
      ;;
  esac
}

# Export for use as library
export -f cmd_notify

# Execute if run directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ] || [ -z "${1:-}" ]; then
    cmd_notify "$@"
    exit 0
  fi
  cmd_notify "$@"
  exit $?
fi
