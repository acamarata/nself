#!/usr/bin/env bash
# cron.sh - cron job scheduler management for nself
# Manages scheduled jobs for the nself-cron pro plugin.
#
# Commands:
#   nself cron list [--json]                                              List all cron jobs
#   nself cron add --name <name> --expr <cron> --action <type> --target <url|cmd>  Add job
#   nself cron remove <id>                                                Remove a job
#   nself cron trigger <id>                                               Manually trigger a job now
#   nself cron logs [<id>] [--limit N] [--json]                          Show run history
#   nself cron enable <id>                                                Enable a paused job
#   nself cron disable <id>                                               Pause a job
#   nself cron status                                                     Show plugin status + scheduler state
#
# Usage: nself cron <subcommand> [options]

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

cron_usage() {
  printf "nself cron — cron job scheduler management\n\n"
  printf "Usage: nself cron <subcommand> [options]\n\n"
  printf "Subcommands:\n"
  printf "  list [--json]                                                 List all cron jobs\n"
  printf "  add --name <name> --expr <cron>                              Add a new cron job\n"
  printf "      --action <type> --target <url|cmd>\n"
  printf "  remove <id>                                                   Remove a job\n"
  printf "  trigger <id>                                                  Manually trigger a job now\n"
  printf "  logs [<id>] [--limit N] [--json]                             Show run history\n"
  printf "  enable <id>                                                   Enable a paused job\n"
  printf "  disable <id>                                                  Pause a job\n"
  printf "  status                                                        Show plugin status + scheduler state\n\n"
  printf "Environment:\n"
  printf "  CRON_PORT               cron plugin port (default: 3713)\n"
  printf "  PLUGIN_INTERNAL_SECRET  required for all commands\n\n"
  printf "Examples:\n"
  printf "  nself cron list\n"
  printf "  nself cron add --name cleanup --expr '0 2 * * *' --action http --target https://api.example.com/cleanup\n"
  printf "  nself cron add --name backup --expr '30 3 * * 0' --action command --target '/scripts/backup.sh'\n"
  printf "  nself cron trigger abc123\n"
  printf "  nself cron logs abc123 --limit 10\n"
  printf "  nself cron disable abc123\n"
  printf "  nself cron enable abc123\n"
  printf "  nself cron remove abc123\n"
  printf "  nself cron status\n"
}

# ============================================================================
# Plugin connectivity check
# ============================================================================

_cron_base_url() {
  printf "http://127.0.0.1:%s" "${CRON_PORT:-3713}"
}

_cron_check_running() {
  local base_url
  base_url="$(_cron_base_url)"
  if ! curl -s --max-time 2 "${base_url}/health" >/dev/null 2>&1; then
    cli_error "Cron plugin not running. Install: nself plugin install cron"
    return 1
  fi
}

# ============================================================================
# list subcommand
# ============================================================================

cmd_list() {
  local json_output="false"

  while [ $# -gt 0 ]; do
    case "$1" in
      --json) json_output="true"; shift ;;
      *) shift ;;
    esac
  done

  _cron_check_running || return 1

  local base_url
  base_url="$(_cron_base_url)"

  local resp=""
  resp=$(curl -s "${base_url}/jobs" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from cron plugin at ${base_url}"
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
      log_info "No cron jobs configured."
      printf "Add one with: nself cron add --name <name> --expr '<cron>' --action <type> --target <url|cmd>\n"
      return 0
    fi

    printf "\033[34m%-36s %-20s %-18s %-12s %s\033[0m\n" "ID" "Name" "Schedule" "Status" "Last Run"
    printf "%-36s %-20s %-18s %-12s %s\n" "------------------------------------" "--------------------" "------------------" "------------" "--------"

    printf '%s' "$resp" | jq -r '.[] | [.id // "?", .name // "?", .cron_expression // .schedule // "?", (if .enabled // true then "enabled" else "disabled" end), .last_run // .last_run_at // "-"] | @tsv' 2>/dev/null \
      | while IFS=$(printf '\t') read -r job_id job_name schedule job_status last_run; do
          if [ "$job_status" = "enabled" ]; then
            printf "\033[32m%-36s %-20s %-18s %-12s %s\033[0m\n" "$job_id" "$job_name" "$schedule" "$job_status" "$last_run"
          else
            printf "\033[33m%-36s %-20s %-18s %-12s %s\033[0m\n" "$job_id" "$job_name" "$schedule" "$job_status" "$last_run"
          fi
        done
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# add subcommand
# ============================================================================

cmd_add() {
  local name="" expr="" action_type="" action_target=""

  while [ $# -gt 0 ]; do
    case "$1" in
      --name)   name="$2";          shift 2 ;;
      --expr)   expr="$2";          shift 2 ;;
      --action) action_type="$2";   shift 2 ;;
      --target) action_target="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  if [ -z "$name" ]; then
    cli_error "--name is required"
    printf "Usage: nself cron add --name <name> --expr <cron> --action <type> --target <url|cmd>\n" >&2
    return 1
  fi

  if [ -z "$expr" ]; then
    cli_error "--expr (cron expression) is required"
    printf "Usage: nself cron add --name <name> --expr <cron> --action <type> --target <url|cmd>\n" >&2
    return 1
  fi

  if [ -z "$action_type" ]; then
    cli_error "--action is required (e.g. http, command)"
    printf "Usage: nself cron add --name <name> --expr <cron> --action <type> --target <url|cmd>\n" >&2
    return 1
  fi

  if [ -z "$action_target" ]; then
    cli_error "--target is required"
    printf "Usage: nself cron add --name <name> --expr <cron> --action <type> --target <url|cmd>\n" >&2
    return 1
  fi

  _cron_check_running || return 1

  local base_url
  base_url="$(_cron_base_url)"

  local safe_name="" safe_expr="" safe_action_type="" safe_action_target=""
  safe_name=$(printf '%s' "$name" | sed 's/\\/\\\\/g; s/"/\\"/g')
  safe_expr=$(printf '%s' "$expr" | sed 's/\\/\\\\/g; s/"/\\"/g')
  safe_action_type=$(printf '%s' "$action_type" | sed 's/\\/\\\\/g; s/"/\\"/g')
  safe_action_target=$(printf '%s' "$action_target" | sed 's/\\/\\\\/g; s/"/\\"/g')

  local payload="{\"name\":\"${safe_name}\",\"cron_expression\":\"${safe_expr}\",\"action_type\":\"${safe_action_type}\",\"action_target\":\"${safe_action_target}\"}"

  local resp=""
  resp=$(curl -s -X POST "${base_url}/jobs" \
    -H "Content-Type: application/json" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" \
    -d "$payload" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from cron plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local job_id=""
    job_id=$(printf '%s' "$resp" | jq -r '.id // ""' 2>/dev/null || true)
    local err_out=""
    err_out=$(printf '%s' "$resp" | jq -r '.error // ""' 2>/dev/null || true)
    if [ -n "$err_out" ]; then
      cli_error "Failed to add job: ${err_out}"
      return 1
    fi
    log_success "Cron job added: ${name} (ID: ${job_id})"
    printf "  Schedule: %s\n" "$expr"
    printf "  Action:   %s -> %s\n" "$action_type" "$action_target"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# remove subcommand
# ============================================================================

cmd_remove() {
  local job_id="${1:-}"

  if [ -z "$job_id" ]; then
    cli_error "Job ID required"
    printf "Usage: nself cron remove <id>\n" >&2
    return 1
  fi

  _cron_check_running || return 1

  local base_url
  base_url="$(_cron_base_url)"

  local safe_id=""
  safe_id=$(printf '%s' "$job_id" | sed 's/ /%20/g')

  local resp=""
  resp=$(curl -s -X DELETE "${base_url}/jobs/${safe_id}" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from cron plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local err_out=""
    err_out=$(printf '%s' "$resp" | jq -r '.error // ""' 2>/dev/null || true)
    if [ -n "$err_out" ]; then
      cli_error "Failed to remove job: ${err_out}"
      return 1
    fi
    log_success "Cron job removed: ${job_id}"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# trigger subcommand
# ============================================================================

cmd_trigger() {
  local job_id="${1:-}"

  if [ -z "$job_id" ]; then
    cli_error "Job ID required"
    printf "Usage: nself cron trigger <id>\n" >&2
    return 1
  fi

  _cron_check_running || return 1

  local base_url
  base_url="$(_cron_base_url)"

  local safe_id=""
  safe_id=$(printf '%s' "$job_id" | sed 's/ /%20/g')

  log_info "Triggering job: ${job_id}"

  local resp=""
  resp=$(curl -s -X POST "${base_url}/jobs/${safe_id}/trigger" \
    -H "Content-Type: application/json" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from cron plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local trigger_status=""
    trigger_status=$(printf '%s' "$resp" | jq -r '.status // .error // "unknown"' 2>/dev/null || true)
    local run_id=""
    run_id=$(printf '%s' "$resp" | jq -r '.run_id // ""' 2>/dev/null || true)
    if [ "$trigger_status" = "triggered" ] || [ "$trigger_status" = "ok" ] || [ "$trigger_status" = "running" ]; then
      log_success "Job triggered: ${job_id}"
      if [ -n "$run_id" ]; then
        printf "  Run ID: %s\n" "$run_id"
      fi
    else
      cli_error "Trigger failed: ${trigger_status}"
      return 1
    fi
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# logs subcommand
# ============================================================================

cmd_logs() {
  local job_id=""
  local limit="50"
  local json_output="false"

  # First positional arg that doesn't start with -- is the job ID
  if [ $# -gt 0 ] && [ "${1#--}" = "$1" ]; then
    job_id="$1"
    shift
  fi

  while [ $# -gt 0 ]; do
    case "$1" in
      --limit) limit="$2"; shift 2 ;;
      --json)  json_output="true"; shift ;;
      *) shift ;;
    esac
  done

  _cron_check_running || return 1

  local base_url
  base_url="$(_cron_base_url)"

  local endpoint=""
  if [ -n "$job_id" ]; then
    local safe_id=""
    safe_id=$(printf '%s' "$job_id" | sed 's/ /%20/g')
    endpoint="${base_url}/jobs/${safe_id}/history?limit=${limit}"
  else
    endpoint="${base_url}/history?limit=${limit}"
  fi

  local resp=""
  resp=$(curl -s "${endpoint}" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from cron plugin at ${base_url}"
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
      log_info "No run history found."
      return 0
    fi

    printf "\033[34m%-36s %-20s %-10s %-24s %s\033[0m\n" "Run ID" "Job Name" "Status" "Started" "Duration"
    printf "%-36s %-20s %-10s %-24s %s\n" "------------------------------------" "--------------------" "----------" "------------------------" "--------"

    printf '%s' "$resp" | jq -r '.[] | [.run_id // .id // "?", .job_name // .name // "?", .status // "?", .started_at // .created_at // "-", (.duration_ms // "" | tostring)] | @tsv' 2>/dev/null \
      | while IFS=$(printf '\t') read -r run_id job_name run_status started_at duration; do
          if [ "$run_status" = "success" ] || [ "$run_status" = "ok" ]; then
            local dur_label=""
            if [ -n "$duration" ] && [ "$duration" != "null" ] && [ "$duration" != "" ]; then
              dur_label="${duration}ms"
            else
              dur_label="-"
            fi
            printf "\033[32m%-36s %-20s %-10s %-24s %s\033[0m\n" "$run_id" "$job_name" "$run_status" "$started_at" "$dur_label"
          else
            local dur_label2=""
            if [ -n "$duration" ] && [ "$duration" != "null" ] && [ "$duration" != "" ]; then
              dur_label2="${duration}ms"
            else
              dur_label2="-"
            fi
            printf "\033[31m%-36s %-20s %-10s %-24s %s\033[0m\n" "$run_id" "$job_name" "$run_status" "$started_at" "$dur_label2"
          fi
        done
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# enable subcommand
# ============================================================================

cmd_enable() {
  local job_id="${1:-}"

  if [ -z "$job_id" ]; then
    cli_error "Job ID required"
    printf "Usage: nself cron enable <id>\n" >&2
    return 1
  fi

  _cron_check_running || return 1

  local base_url
  base_url="$(_cron_base_url)"

  local safe_id=""
  safe_id=$(printf '%s' "$job_id" | sed 's/ /%20/g')

  local resp=""
  resp=$(curl -s -X PATCH "${base_url}/jobs/${safe_id}" \
    -H "Content-Type: application/json" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" \
    -d '{"enabled":true}' 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from cron plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local err_out=""
    err_out=$(printf '%s' "$resp" | jq -r '.error // ""' 2>/dev/null || true)
    if [ -n "$err_out" ]; then
      cli_error "Failed to enable job: ${err_out}"
      return 1
    fi
    log_success "Cron job enabled: ${job_id}"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# disable subcommand
# ============================================================================

cmd_disable() {
  local job_id="${1:-}"

  if [ -z "$job_id" ]; then
    cli_error "Job ID required"
    printf "Usage: nself cron disable <id>\n" >&2
    return 1
  fi

  _cron_check_running || return 1

  local base_url
  base_url="$(_cron_base_url)"

  local safe_id=""
  safe_id=$(printf '%s' "$job_id" | sed 's/ /%20/g')

  local resp=""
  resp=$(curl -s -X PATCH "${base_url}/jobs/${safe_id}" \
    -H "Content-Type: application/json" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" \
    -d '{"enabled":false}' 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from cron plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local err_out=""
    err_out=$(printf '%s' "$resp" | jq -r '.error // ""' 2>/dev/null || true)
    if [ -n "$err_out" ]; then
      cli_error "Failed to disable job: ${err_out}"
      return 1
    fi
    log_success "Cron job disabled: ${job_id}"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# status subcommand
# ============================================================================

cmd_status() {
  _cron_check_running || return 1

  local base_url
  base_url="$(_cron_base_url)"

  local resp=""
  resp=$(curl -s "${base_url}/health" \
    -H "X-Internal-Token: ${PLUGIN_INTERNAL_SECRET:-}" 2>/dev/null)

  if [ -z "$resp" ]; then
    cli_error "No response from cron plugin at ${base_url}"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local plugin_status=""
    plugin_status=$(printf '%s' "$resp" | jq -r '.status // "unknown"' 2>/dev/null || true)
    local version=""
    version=$(printf '%s' "$resp" | jq -r '.version // ""' 2>/dev/null || true)
    local scheduler_state=""
    scheduler_state=$(printf '%s' "$resp" | jq -r '.scheduler // .scheduler_state // ""' 2>/dev/null || true)
    local job_count=""
    job_count=$(printf '%s' "$resp" | jq -r '.jobs // .job_count // ""' 2>/dev/null || true)

    printf "Cron plugin\n"
    printf "  Status:    %s\n" "$plugin_status"
    if [ -n "$version" ]; then
      printf "  Version:   %s\n" "$version"
    fi
    if [ -n "$scheduler_state" ]; then
      printf "  Scheduler: %s\n" "$scheduler_state"
    fi
    if [ -n "$job_count" ]; then
      printf "  Jobs:      %s\n" "$job_count"
    fi
    printf "  Endpoint:  %s\n" "$base_url"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# Top-level dispatcher
# ============================================================================

cmd_cron() {
  local subcommand="${1:-}"

  if [ -z "$subcommand" ]; then
    cron_usage
    exit 0
  fi

  case "$subcommand" in
    --help | -h)
      cron_usage
      exit 0
      ;;
    *) ;;
  esac

  shift

  case "$subcommand" in
    list)
      cmd_list "$@"
      ;;
    add)
      cmd_add "$@"
      ;;
    remove)
      cmd_remove "$@"
      ;;
    trigger)
      cmd_trigger "$@"
      ;;
    logs)
      cmd_logs "$@"
      ;;
    enable)
      cmd_enable "$@"
      ;;
    disable)
      cmd_disable "$@"
      ;;
    status)
      cmd_status "$@"
      ;;
    help | --help | -h)
      cron_usage
      exit 0
      ;;
    *)
      cli_error "Unknown subcommand: $subcommand"
      printf "\n"
      cron_usage
      exit 1
      ;;
  esac
}

# Export for use as library
export -f cmd_cron

# Execute if run directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
  if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ] || [ -z "${1:-}" ]; then
    cmd_cron "$@"
    exit 0
  fi
  cmd_cron "$@"
  exit $?
fi
