#!/usr/bin/env bash
# mux.sh - mux plugin management for nself
# Manages tokens and pipeline configuration for the nself-mux pro plugin.
#
# Commands:
#   nself mux tokens import --file <path.json>  Bulk-import delivery auth tokens
#   nself mux tokens list                        List stored token names
#
# Usage: nself mux <subcommand> [options]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source display helpers
source "$NSELF_ROOT/src/lib/utils/cli-output.sh" 2>/dev/null || true
source "$NSELF_ROOT/src/lib/utils/display.sh" 2>/dev/null || true

# Fallbacks if display helpers didn't load
if ! declare -f cli_error >/dev/null 2>&1; then
  cli_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi
if ! declare -f log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi

# ============================================================================
# Usage
# ============================================================================

mux_usage() {
  printf "nself mux — mux pipeline plugin management\n\n"
  printf "Usage: nself mux <subcommand> [options]\n\n"
  printf "Subcommands:\n"
  printf "  status                       Show mux service health and push status\n"
  printf "  reload                       Hot-reload YAML rules without restarting\n"
  printf "  rules [--format json]        List configured routing rules\n"
  printf "  shadow on|off                Enable or disable shadow (dry-run) mode\n"
  printf "  test [options]               Dry-run rules against a test message\n"
  printf "  logs [-f] [-n N]             Tail mux container logs\n"
  printf "  tokens import --file <path>  Bulk-import delivery auth tokens from JSON\n"
  printf "  tokens list                  List stored token names (no values shown)\n"
  printf "  push-status                  Show per-account Pub/Sub push health\n"
  printf "  dlq list                     List all DLQ entries\n"
  printf "  dlq retry <id>               Retry a single DLQ entry immediately\n"
  printf "  dlq clear                    Delete all permanently-failed DLQ entries\n\n"
  printf "Environment:\n"
  printf "  NSELF_MUX_URL       mux plugin base URL (default: http://localhost:3711)\n"
  printf "  NSELF_PROJECT_DIR   project root for shadow env var updates\n\n"
  printf "Examples:\n"
  printf "  nself mux status\n"
  printf "  nself mux reload\n"
  printf "  nself mux rules\n"
  printf "  nself mux shadow on\n"
  printf "  nself mux test --from boss@example.com --subject 'Invoice #42'\n"
  printf "  nself mux logs -f\n"
  printf "  nself mux tokens import --file ./tokens.json\n"
  printf "  nself mux tokens list\n"
  printf "  nself mux push-status\n\n"
  printf "Token file format (tokens.json):\n"
  printf '  [{"name":"my-webhook","token":"Bearer abc123","description":"Main webhook"}]\n'
}

# ============================================================================
# Top-level dispatcher
# ============================================================================

cmd_mux() {
  local subcommand="${1:-}"

  if [ -z "$subcommand" ]; then
    mux_usage
    exit 0
  fi

  shift

  case "$subcommand" in
    status)
      cmd_mux_status "$@"
      ;;
    reload)
      cmd_mux_reload "$@"
      ;;
    rules)
      cmd_mux_rules "$@"
      ;;
    shadow)
      cmd_mux_shadow "$@"
      ;;
    test)
      cmd_mux_test "$@"
      ;;
    logs)
      cmd_mux_logs "$@"
      ;;
    tokens)
      cmd_mux_tokens "$@"
      ;;
    push-status)
      cmd_mux_push_status "$@"
      ;;
    dlq)
      cmd_mux_dlq "$@"
      ;;
    help | --help | -h)
      mux_usage
      exit 0
      ;;
    *)
      cli_error "Unknown subcommand: $subcommand"
      printf "\n"
      mux_usage
      exit 1
      ;;
  esac
}

# ============================================================================
# status subcommand — show mux health and push status
# ============================================================================

cmd_mux_status() {
  local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"

  local health=""
  health=$(curl -sf --max-time 5 "${mux_url}/health" 2>/dev/null)
  if [ $? -ne 0 ] || [ -z "$health" ]; then
    printf "\033[31m[DOWN]\033[0m Cannot reach mux at %s\n" "$mux_url" >&2
    exit 1
  fi

  # Extract version + uptime from JSON (no jq required)
  local version uptime_str
  version=$(printf '%s' "$health" | grep -o '"version":"[^"]*"' | cut -d'"' -f4)
  uptime_str=$(printf '%s' "$health" | grep -o '"uptime_secs":[0-9]*' | cut -d':' -f2)

  printf "\033[32m[OK]\033[0m nself-mux is running\n"
  if [ -n "$version" ]; then
    printf "  Version:  %s\n" "$version"
  fi
  if [ -n "$uptime_str" ]; then
    printf "  Uptime:   %ss\n" "$uptime_str"
  fi
  printf "  URL:      %s\n" "$mux_url"
  printf "\n"

  # Also show push status
  cmd_mux_push_status "$@"
}

# ============================================================================
# reload subcommand — hot-reload YAML rules
# ============================================================================

cmd_mux_reload() {
  local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"

  log_info "Sending reload signal to mux..."

  local resp=""
  resp=$(curl -sf --max-time 10 -X POST "${mux_url}/mux/reload" 2>/dev/null)
  if [ $? -ne 0 ] || [ -z "$resp" ]; then
    cli_error "No response from mux at ${mux_url}. Is it running?"
    exit 1
  fi

  local status=""
  status=$(printf '%s' "$resp" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
  if [ "$status" = "reloading" ]; then
    log_success "Rules reloaded."
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# rules subcommand — list routing rules
# ============================================================================

cmd_mux_rules() {
  local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"
  local format="table"

  while [ $# -gt 0 ]; do
    case "$1" in
      --format | -f) format="${2:-table}"; shift 2 ;;
      --json)        format="json"; shift ;;
      --help | -h)   mux_usage; exit 0 ;;
      *)             shift ;;
    esac
  done

  local resp=""
  resp=$(curl -sf --max-time 10 "${mux_url}/mux/rules" 2>/dev/null)
  if [ $? -ne 0 ] || [ -z "$resp" ]; then
    cli_error "No response from mux at ${mux_url}. Is it running?"
    exit 1
  fi

  if [ "$format" = "json" ]; then
    if command -v jq >/dev/null 2>&1; then
      printf '%s' "$resp" | jq .
    else
      printf '%s\n' "$resp"
    fi
    return 0
  fi

  # Table format using jq if available
  if command -v jq >/dev/null 2>&1; then
    local count
    count=$(printf '%s' "$resp" | jq '.rules | length')
    printf "\033[34m%-5s %-30s %-10s %-30s %-10s\033[0m\n" "PRI" "Name" "Enabled" "From Pattern" "Action"
    printf "%-5s %-30s %-10s %-30s %-10s\n" "---" "----" "-------" "------------" "------"
    printf '%s' "$resp" | jq -r '.rules[] | [
      (.priority | tostring),
      .name,
      (if .enabled then "yes" else "no" end),
      (.conditions.from // "(any)"),
      (.action | keys[0])
    ] | @tsv' | while IFS=$(printf '\t') read -r pri name enabled from_pat action; do
      local color=""
      if [ "$enabled" = "yes" ]; then
        color="\033[32m"
      else
        color="\033[33m"
      fi
      printf "${color}%-5s %-30s %-10s %-30s %-10s\033[0m\n" \
        "$pri" "$name" "$enabled" "$from_pat" "$action"
    done
    printf "\n%s rule(s) total.\n" "$count"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# shadow subcommand — toggle shadow (dry-run) mode
# ============================================================================

cmd_mux_shadow() {
  local mode="${1:-}"

  case "$mode" in
    on|off) ;;
    *)
      cli_error "Usage: nself mux shadow on|off"
      exit 1
      ;;
  esac

  # Find .env file in project dir (check NSELF_PROJECT_DIR, then PWD)
  local env_file=""
  if [ -n "${NSELF_PROJECT_DIR:-}" ] && [ -f "${NSELF_PROJECT_DIR}/.env" ]; then
    env_file="${NSELF_PROJECT_DIR}/.env"
  elif [ -f "${PWD}/.env" ]; then
    env_file="${PWD}/.env"
  fi

  local new_val="false"
  if [ "$mode" = "on" ]; then
    new_val="true"
  fi

  if [ -n "$env_file" ]; then
    # Update or append PLUGIN_MUX_SHADOW_MODE in .env
    if grep -q '^PLUGIN_MUX_SHADOW_MODE=' "$env_file" 2>/dev/null; then
      # Use a temp file to avoid platform-specific sed -i issues
      local tmp_file
      tmp_file="${env_file}.mux_tmp"
      while IFS= read -r line; do
        case "$line" in
          PLUGIN_MUX_SHADOW_MODE=*) printf 'PLUGIN_MUX_SHADOW_MODE=%s\n' "$new_val" ;;
          *) printf '%s\n' "$line" ;;
        esac
      done < "$env_file" > "$tmp_file"
      mv "$tmp_file" "$env_file"
    else
      printf 'PLUGIN_MUX_SHADOW_MODE=%s\n' "$new_val" >> "$env_file"
    fi
    log_success "Shadow mode set to: $mode (updated $env_file)"
    printf "  Run 'nself build && nself restart' to apply, or:\n"
  else
    log_warning "No .env file found. Set NSELF_PROJECT_DIR or run from your project root."
    printf "  Manually add to your .env:\n"
    printf "    PLUGIN_MUX_SHADOW_MODE=%s\n" "$new_val"
    printf "  Then rebuild: nself build && nself restart\n"
  fi

  # Attempt live reload via HTTP (best-effort — may not pick up env var change)
  local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"
  if curl -sf --max-time 3 -X POST "${mux_url}/mux/reload" >/dev/null 2>&1; then
    log_info "Live reload triggered. Note: env var changes require a full restart."
  fi
}

# ============================================================================
# test subcommand — dry-run rules against a synthetic message
# ============================================================================

cmd_mux_test() {
  local from_addr="" subject="" body="" has_attachment="false"
  local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"

  while [ $# -gt 0 ]; do
    case "$1" in
      --from)           from_addr="$2"; shift 2 ;;
      --subject | -s)   subject="$2"; shift 2 ;;
      --body | -b)      body="$2"; shift 2 ;;
      --attachment)     has_attachment="true"; shift ;;
      --help | -h)
        printf "Usage: nself mux test [options]\n\n"
        printf "Options:\n"
        printf "  --from <address>    Sender email address\n"
        printf "  --subject <text>    Message subject\n"
        printf "  --body <text>       Message body snippet\n"
        printf "  --attachment        Mark as having an attachment\n\n"
        printf "Example:\n"
        printf "  nself mux test --from ceo@example.com --subject 'Q4 numbers'\n"
        exit 0
        ;;
      *) shift ;;
    esac
  done

  local payload
  payload=$(printf '{"from":"%s","subject":"%s","body":"%s","has_attachment":%s}' \
    "$from_addr" "$subject" "$body" "$has_attachment")

  log_info "Running dry-run against rules..."

  local resp=""
  resp=$(curl -sf --max-time 10 \
    -X POST "${mux_url}/mux/test" \
    -H "Content-Type: application/json" \
    -d "$payload" 2>/dev/null)

  if [ $? -ne 0 ] || [ -z "$resp" ]; then
    cli_error "No response from mux at ${mux_url}. Is it running?"
    exit 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local checked matched
    checked=$(printf '%s' "$resp" | jq '.checked')
    matched=$(printf '%s' "$resp" | jq '.matched')
    printf "\n  Checked: %s rule(s)\n" "$checked"
    printf "  Matched: %s rule(s)\n\n" "$matched"
    if [ "$matched" != "0" ]; then
      printf '%s' "$resp" | jq -r '.rules[] | "  \(.priority | tostring | @text) | \(.name) → \(.action | keys[0])"'
    else
      printf "  (no rules matched)\n"
    fi
    printf "\n"
  else
    printf '%s\n' "$resp"
  fi
}

# ============================================================================
# logs subcommand — tail mux container logs
# ============================================================================

cmd_mux_logs() {
  local follow=false
  local tail_n=100

  while [ $# -gt 0 ]; do
    case "$1" in
      -f | --follow)  follow=true; shift ;;
      -n)             tail_n="$2"; shift 2 ;;
      --help | -h)
        printf "Usage: nself mux logs [-f] [-n N]\n\n"
        printf "  -f, --follow   Follow log output\n"
        printf "  -n N           Show last N lines (default: 100)\n"
        exit 0
        ;;
      *) shift ;;
    esac
  done

  # Find the mux container (matches any container with "mux" in its name)
  local container=""
  if command -v docker >/dev/null 2>&1; then
    container=$(docker ps --filter "name=mux" --format "{{.Names}}" 2>/dev/null | head -1)
  fi

  if [ -z "$container" ]; then
    cli_error "No running mux container found. Is the mux plugin running?"
    printf "  Try: nself start\n"
    exit 1
  fi

  log_info "Streaming logs from container: $container"

  if [ "$follow" = "true" ]; then
    docker logs --tail "$tail_n" -f "$container"
  else
    docker logs --tail "$tail_n" "$container"
  fi
}

# ============================================================================
# Tokens subcommand dispatcher
# ============================================================================

cmd_mux_tokens() {
  local subcmd="${1:-}"

  if [ -z "$subcmd" ]; then
    cli_error "Tokens action required"
    printf "Actions: import, list\n"
    exit 1
  fi

  shift

  case "$subcmd" in

    import)
      # Bulk-import delivery auth tokens from a JSON file.
      # Usage: nself mux tokens import --file <path>
      # File format: [{"name": string, "token": string, "description"?: string}, ...]
      local file_path=""
      while [ $# -gt 0 ]; do
        case "$1" in
          --file | -f) file_path="$2"; shift 2 ;;
          *) shift ;;
        esac
      done

      if [ -z "$file_path" ]; then
        printf "Usage: nself mux tokens import --file <path.json>\n" >&2
        return 1
      fi

      if [ ! -f "$file_path" ]; then
        cli_error "File not found: $file_path"
        return 1
      fi

      # Read file content
      local tokens_json=""
      tokens_json=$(cat "$file_path")

      if [ -z "$tokens_json" ]; then
        cli_error "Token file is empty: $file_path"
        return 1
      fi

      local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"

      log_info "Importing tokens from: $file_path"

      local payload="{\"tokens\":${tokens_json}}"
      local resp=""
      resp=$(curl -s -X POST "${mux_url}/tokens/import" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null)

      if [ -z "$resp" ]; then
        cli_error "No response from mux service at ${mux_url}. Is it running?"
        return 1
      fi

      # Extract imported/skipped counts
      local imported="" skipped=""
      imported=$(printf '%s' "$resp" | grep -o '"imported":[0-9]*' | cut -d':' -f2 || true)
      skipped=$(printf '%s' "$resp" | grep -o '"skipped":[0-9]*' | cut -d':' -f2 || true)

      if [ -n "$imported" ]; then
        log_success "Import complete: ${imported} imported, ${skipped:-0} skipped"
      else
        # Show raw response if unexpected
        printf '%s\n' "$resp"
      fi
      ;;

    list)
      # List stored token names (values are never shown).
      # Usage: nself mux tokens list
      local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"
      local result=""
      result=$(curl -s "${mux_url}/tokens" 2>/dev/null)
      printf '%s\n' "$result"
      ;;

    help | --help | -h)
      mux_usage
      exit 0
      ;;

    *)
      cli_error "Unknown tokens action: $subcmd"
      printf "Actions: import, list\n"
      exit 1
      ;;

  esac
}

# ============================================================================
# push-status subcommand
# ============================================================================

cmd_mux_push_status() {
  # Display per-account Pub/Sub push health from the mux health endpoint.
  # Usage: nself mux push-status
  local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"
  local response
  response=$(curl -sf "${mux_url}/mux/health/push-status" 2>/dev/null)
  if [ $? -ne 0 ] || [ -z "$response" ]; then
    printf "\033[31mError: could not reach mux at %s\033[0m\n" "$mux_url" >&2
    exit 1
  fi

  printf "\033[34m%-30s %-25s %s\033[0m\n" "Account" "Last Push" "Status"
  printf "%-30s %-25s %s\n" "-------" "---------" "------"

  # Parse JSON with jq if available, else print raw response
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$response" | jq -r '.[] | [.account_email, (.last_push_at // "never"), (if .is_stalled then "STALLED" else "OK" end)] | @tsv' | \
    while IFS=$(printf '\t') read -r email last_push status; do
      if [ "$status" = "STALLED" ]; then
        printf "\033[31m%-30s %-25s %s\033[0m\n" "$email" "$last_push" "$status"
      else
        printf "\033[32m%-30s %-25s %s\033[0m\n" "$email" "$last_push" "$status"
      fi
    done
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# dlq subcommand dispatcher — list, retry, clear DLQ entries
# ============================================================================

cmd_mux_dlq() {
  local subcmd="${1:-}"
  local mux_url="${NSELF_MUX_URL:-http://localhost:3711}"

  if [ -z "$subcmd" ]; then
    printf "Usage: nself mux dlq <list|retry <id>|clear>\n"
    exit 1
  fi

  shift

  case "$subcmd" in

    list)
      local resp=""
      resp=$(curl -sf --max-time 10 "${mux_url}/mux/dlq" 2>/dev/null)
      if [ $? -ne 0 ] || [ -z "$resp" ]; then
        cli_error "No response from mux at ${mux_url}. Is it running?"
        exit 1
      fi
      if command -v jq >/dev/null 2>&1; then
        local count
        count=$(printf '%s' "$resp" | jq 'length')
        if [ "$count" = "0" ]; then
          printf "DLQ is empty.\n"
        else
          printf "\033[34m%-38s %-10s %-30s\033[0m\n" "ID" "Attempts" "Last Error"
          printf "%-38s %-10s %-30s\n" "---" "--------" "----------"
          printf '%s' "$resp" | jq -r '.[] | [.id, (.attempt_count | tostring), .last_error] | @tsv' | \
          while IFS=$(printf '\t') read -r id attempts last_err; do
            printf "%-38s %-10s %-30s\n" "$id" "$attempts" "$last_err"
          done
          printf "\n%s entry/entries in DLQ.\n" "$count"
        fi
      else
        printf '%s\n' "$resp"
      fi
      ;;

    retry)
      local entry_id="${1:-}"
      if [ -z "$entry_id" ]; then
        cli_error "Usage: nself mux dlq retry <id>"
        exit 1
      fi
      local resp=""
      resp=$(curl -sf --max-time 10 -X POST "${mux_url}/mux/dlq/${entry_id}/retry" 2>/dev/null)
      if [ $? -ne 0 ] || [ -z "$resp" ]; then
        cli_error "No response from mux at ${mux_url}. Is it running?"
        exit 1
      fi
      local retried=""
      retried=$(printf '%s' "$resp" | grep -o '"retried":true' || true)
      if [ -n "$retried" ]; then
        log_success "DLQ entry ${entry_id} queued for retry."
      else
        printf '%s\n' "$resp"
      fi
      ;;

    clear)
      # Delete all permanently-failed entries
      local resp=""
      resp=$(curl -sf --max-time 10 \
        -X POST "${mux_url}/mux/dlq/clear-failed" \
        -H "Content-Type: application/json" 2>/dev/null)
      if [ -z "$resp" ]; then
        # Endpoint may not exist yet — fall back to listing and noting the limitation
        cli_error "clear-failed endpoint not available. Use 'nself mux dlq list' and retry individual entries."
        exit 1
      fi
      log_success "Permanently-failed DLQ entries cleared."
      ;;

    help | --help | -h)
      printf "Usage: nself mux dlq <list|retry <id>|clear>\n\n"
      printf "  list         Show all DLQ entries\n"
      printf "  retry <id>   Retry a specific DLQ entry immediately\n"
      printf "  clear        Delete all permanently-failed entries\n"
      exit 0
      ;;

    *)
      cli_error "Unknown dlq action: $subcmd"
      printf "Actions: list, retry <id>, clear\n"
      exit 1
      ;;

  esac
}

# ============================================================================
# Entry point
# ============================================================================

cmd_mux "$@"
