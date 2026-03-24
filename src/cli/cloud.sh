#!/usr/bin/env bash
# cloud.sh - nSelf Cloud hosting management
# Manages servers on nSelf's managed cloud hosting (cloud.nself.org)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source shared utilities
source "${SCRIPT_DIR}/../lib/utils/display.sh" 2>/dev/null || true

# Fallback logging if display.sh failed to load
if ! type log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m✓\033[0m %s\n" "$1"; }
fi
if ! type log_warning >/dev/null 2>&1; then
  log_warning() { printf "\033[0;33m!\033[0m %s\n" "$1"; }
fi
if ! type log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m✗\033[0m %s\n" "$1" >&2; }
fi
if ! type log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34mℹ\033[0m %s\n" "$1"; }
fi

# ============================================================================
# CONSTANTS
# ============================================================================

NSELF_CLOUD_API_URL="${NSELF_CLOUD_API_URL:-https://cloud.nself.org/api}"
NSELF_CLOUD_TOKEN_FILE="$HOME/.nself/cloud-token"

# ============================================================================
# HELPERS
# ============================================================================

# Resolve the cloud API token from env or token file
_cloud_get_token() {
  if [[ -n "${NSELF_CLOUD_TOKEN:-}" ]]; then
    printf '%s' "$NSELF_CLOUD_TOKEN"
    return 0
  fi
  if [[ -f "$NSELF_CLOUD_TOKEN_FILE" ]]; then
    local token
    token="$(cat "$NSELF_CLOUD_TOKEN_FILE")"
    if [[ -n "$token" ]]; then
      printf '%s' "$token"
      return 0
    fi
  fi
  return 1
}

# Verify authentication is configured
_cloud_require_auth() {
  if ! _cloud_get_token >/dev/null 2>&1; then
    log_error "Not authenticated with nSelf Cloud"
    printf "\n"
    printf "  Set your cloud token using one of:\n"
    printf "    export NSELF_CLOUD_TOKEN=<your-token>\n"
    printf "    printf '<your-token>' > %s\n" "$NSELF_CLOUD_TOKEN_FILE"
    printf "\n"
    printf "  Get your token at: https://cloud.nself.org/settings/tokens\n"
    return 1
  fi
}

# Verify jq is available
_cloud_require_jq() {
  if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required for cloud commands but was not found"
    printf "  Install: brew install jq (macOS) or apt install jq (Linux)\n"
    return 1
  fi
}

# Verify curl is available
_cloud_require_curl() {
  if ! command -v curl >/dev/null 2>&1; then
    log_error "curl is required for cloud commands but was not found"
    return 1
  fi
}

# Make an authenticated API request
# Usage: _cloud_api GET /cloud/servers
#        _cloud_api POST /cloud/servers/abc/upgrade '{"plan":"cx31"}'
_cloud_api() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local token
  token="$(_cloud_get_token)"

  local url="${NSELF_CLOUD_API_URL}${path}"
  local curl_args="-s -w \n%{http_code}"
  local response

  if [[ -n "$body" ]]; then
    response=$(curl $curl_args -X "$method" \
      -H "Authorization: Bearer ${token}" \
      -H "Content-Type: application/json" \
      -d "$body" \
      "$url" 2>/dev/null) || true
  else
    response=$(curl $curl_args -X "$method" \
      -H "Authorization: Bearer ${token}" \
      -H "Content-Type: application/json" \
      "$url" 2>/dev/null) || true
  fi

  # Extract HTTP status code (last line) and body (everything before)
  local http_code
  local resp_body
  http_code=$(printf '%s' "$response" | tail -n 1)
  resp_body=$(printf '%s' "$response" | sed '$d')

  # Handle HTTP errors
  case "$http_code" in
    2[0-9][0-9])
      # Success
      printf '%s' "$resp_body"
      return 0
      ;;
    401)
      log_error "Authentication failed (401). Check your cloud token."
      return 1
      ;;
    403)
      log_error "Permission denied (403). Your token lacks access to this resource."
      return 1
      ;;
    404)
      log_error "Resource not found (404)."
      return 1
      ;;
    "")
      log_error "Could not reach nSelf Cloud API at ${NSELF_CLOUD_API_URL}"
      printf "  Check your network connection or NSELF_CLOUD_API_URL setting.\n"
      return 1
      ;;
    *)
      local err_msg
      err_msg=$(printf '%s' "$resp_body" | jq -r '.message // .error // empty' 2>/dev/null || true)
      if [[ -n "$err_msg" ]]; then
        log_error "API error ($http_code): $err_msg"
      else
        log_error "API request failed with HTTP $http_code"
      fi
      return 1
      ;;
  esac
}

# ============================================================================
# HELP
# ============================================================================

show_cloud_help() {
  cat <<'EOF'
nself cloud - nSelf Cloud Hosting Management

USAGE:
  nself cloud <subcommand> [options]

SUBCOMMANDS:
  status                        List all servers with status, IP, plan, and health
  upgrade <server-id> <plan>    Upgrade a server to a new plan
  destroy <server-id>           Permanently destroy a server and all its data

AUTHENTICATION:
  Set NSELF_CLOUD_TOKEN env var or write your token to ~/.nself/cloud-token.
  Get your token at: https://cloud.nself.org/settings/tokens

OPTIONS:
  --confirm       Required for upgrade and destroy operations
  -h, --help      Show this help message

EXAMPLES:
  nself cloud status
  nself cloud upgrade srv_abc123 cx31 --confirm
  nself cloud destroy srv_abc123 --confirm

LEGACY PROVIDER COMMANDS:
  Provider infrastructure commands have moved to 'nself infra provider'.
  Run 'nself infra provider --help' for details.
EOF
}

# ============================================================================
# SUBCOMMANDS
# ============================================================================

# nself cloud status — list user's servers
cmd_cloud_status() {
  _cloud_require_auth || return 1
  _cloud_require_jq || return 1
  _cloud_require_curl || return 1

  log_info "Fetching servers from nSelf Cloud..."
  printf "\n"

  local response
  if ! response=$(_cloud_api GET /cloud/servers); then
    return 1
  fi

  # Check if response is valid JSON with servers
  local count
  count=$(printf '%s' "$response" | jq -r '.servers | length' 2>/dev/null || printf '0')

  if [[ "$count" = "0" ]]; then
    log_info "No servers found. Create one at https://cloud.nself.org"
    return 0
  fi

  # Print table header
  printf "%-14s  %-16s  %-10s  %-10s  %-8s\n" \
    "SERVER ID" "IP ADDRESS" "PLAN" "STATUS" "HEALTH"
  printf "%-14s  %-16s  %-10s  %-10s  %-8s\n" \
    "─────────────" "────────────────" "──────────" "──────────" "────────"

  # Print each server row
  local i=0
  while [ "$i" -lt "$count" ]; do
    local sid sip splan sstatus shealth
    sid=$(printf '%s' "$response" | jq -r ".servers[$i].id // \"—\"")
    sip=$(printf '%s' "$response" | jq -r ".servers[$i].ip // \"—\"")
    splan=$(printf '%s' "$response" | jq -r ".servers[$i].plan // \"—\"")
    sstatus=$(printf '%s' "$response" | jq -r ".servers[$i].status // \"—\"")
    shealth=$(printf '%s' "$response" | jq -r ".servers[$i].health // \"—\"")

    # Color-code status
    local status_display
    case "$sstatus" in
      running)
        status_display=$(printf "\033[0;32m%s\033[0m" "$sstatus")
        ;;
      stopped|shutoff)
        status_display=$(printf "\033[0;31m%s\033[0m" "$sstatus")
        ;;
      provisioning|rebuilding|migrating)
        status_display=$(printf "\033[0;33m%s\033[0m" "$sstatus")
        ;;
      *)
        status_display="$sstatus"
        ;;
    esac

    # Color-code health
    local health_display
    case "$shealth" in
      healthy)
        health_display=$(printf "\033[0;32m%s\033[0m" "$shealth")
        ;;
      degraded)
        health_display=$(printf "\033[0;33m%s\033[0m" "$shealth")
        ;;
      unhealthy|critical)
        health_display=$(printf "\033[0;31m%s\033[0m" "$shealth")
        ;;
      *)
        health_display="$shealth"
        ;;
    esac

    printf "%-14s  %-16s  %-10s  %-10b  %-8b\n" \
      "$sid" "$sip" "$splan" "$status_display" "$health_display"

    i=$((i + 1))
  done

  printf "\n"
  log_info "$count server(s) listed"
}

# nself cloud upgrade <server-id> <plan> --confirm
cmd_cloud_upgrade() {
  _cloud_require_auth || return 1
  _cloud_require_jq || return 1
  _cloud_require_curl || return 1

  local server_id=""
  local plan=""
  local confirmed=false

  # Parse arguments
  while [ $# -gt 0 ]; do
    case "$1" in
      --confirm)
        confirmed=true
        shift
        ;;
      -h|--help)
        printf "Usage: nself cloud upgrade <server-id> <plan> --confirm\n\n"
        printf "Upgrade a cloud server to a new plan.\n\n"
        printf "Arguments:\n"
        printf "  server-id    The server to upgrade (e.g., srv_abc123)\n"
        printf "  plan         The target plan (e.g., cx31, cx41)\n\n"
        printf "Options:\n"
        printf "  --confirm    Required. Confirms the upgrade operation.\n"
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        if [[ -z "$server_id" ]]; then
          server_id="$1"
        elif [[ -z "$plan" ]]; then
          plan="$1"
        else
          log_error "Unexpected argument: $1"
          return 1
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$server_id" ]]; then
    log_error "Server ID required"
    printf "  Usage: nself cloud upgrade <server-id> <plan> --confirm\n"
    return 1
  fi

  if [[ -z "$plan" ]]; then
    log_error "Target plan required"
    printf "  Usage: nself cloud upgrade <server-id> <plan> --confirm\n"
    return 1
  fi

  if [[ "$confirmed" != "true" ]]; then
    log_error "Upgrade requires the --confirm flag"
    printf "\n"
    printf "  This will change the plan for server %s to %s.\n" "$server_id" "$plan"
    printf "  Re-run with --confirm to proceed:\n"
    printf "    nself cloud upgrade %s %s --confirm\n" "$server_id" "$plan"
    return 1
  fi

  log_info "Upgrading server $server_id to plan $plan..."

  local body
  body=$(printf '{"plan":"%s"}' "$plan")

  local response
  if ! response=$(_cloud_api POST "/cloud/servers/${server_id}/upgrade" "$body"); then
    return 1
  fi

  # Extract confirmation details from response
  local new_plan old_plan cost_change
  new_plan=$(printf '%s' "$response" | jq -r '.plan // .new_plan // "'"$plan"'"')
  old_plan=$(printf '%s' "$response" | jq -r '.old_plan // "unknown"')
  cost_change=$(printf '%s' "$response" | jq -r '.cost_change // .estimated_cost_change // "see dashboard"')

  log_success "Server $server_id upgrade initiated"
  printf "\n"
  printf "  Previous plan:     %s\n" "$old_plan"
  printf "  New plan:          %s\n" "$new_plan"
  printf "  Est. cost change:  %s\n" "$cost_change"
  printf "\n"
  log_info "Monitor progress: nself cloud status"
}

# nself cloud destroy <server-id> --confirm
cmd_cloud_destroy() {
  _cloud_require_auth || return 1
  _cloud_require_jq || return 1
  _cloud_require_curl || return 1

  local server_id=""
  local confirmed=false

  # Parse arguments
  while [ $# -gt 0 ]; do
    case "$1" in
      --confirm)
        confirmed=true
        shift
        ;;
      -h|--help)
        printf "Usage: nself cloud destroy <server-id> --confirm\n\n"
        printf "Permanently destroy a cloud server and all its data.\n\n"
        printf "Arguments:\n"
        printf "  server-id    The server to destroy (e.g., srv_abc123)\n\n"
        printf "Options:\n"
        printf "  --confirm    Required. You must also type the server ID to confirm.\n"
        printf "\n"
        printf "WARNING: This action is irreversible. All data will be permanently lost.\n"
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        if [[ -z "$server_id" ]]; then
          server_id="$1"
        else
          log_error "Unexpected argument: $1"
          return 1
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$server_id" ]]; then
    log_error "Server ID required"
    printf "  Usage: nself cloud destroy <server-id> --confirm\n"
    return 1
  fi

  if [[ "$confirmed" != "true" ]]; then
    log_error "Destroy requires the --confirm flag"
    printf "\n"
    printf "  This will permanently destroy server %s and ALL its data.\n" "$server_id"
    printf "  Re-run with --confirm to proceed:\n"
    printf "    nself cloud destroy %s --confirm\n" "$server_id"
    return 1
  fi

  # Double confirmation: user must type the server ID
  log_warning "You are about to PERMANENTLY DESTROY server: $server_id"
  log_warning "All data on this server will be irreversibly lost."
  printf "\n"
  printf "  Type the server ID to confirm: "
  local confirm_input
  read -r confirm_input

  if [[ "$confirm_input" != "$server_id" ]]; then
    log_error "Server ID does not match. Destroy aborted."
    return 1
  fi

  log_info "Destroying server $server_id..."

  if ! _cloud_api DELETE "/cloud/servers/${server_id}" >/dev/null; then
    return 1
  fi

  log_success "Server $server_id has been destroyed"
  printf "\n"
  log_warning "All data has been permanently deleted."
  log_info "Verify with: nself cloud status"
}

# ============================================================================
# LEGACY DEPRECATION BRIDGE
# ============================================================================

_cloud_legacy_delegate() {
  log_warning "DEPRECATION: 'nself cloud $1' has moved to 'nself infra provider'"
  printf "             Please use 'nself infra provider %s' instead.\n" "$*"
  printf "             https://docs.nself.org/migration/cloud-to-infra\n"
  printf "\n"
  if [[ -f "${SCRIPT_DIR}/infra.sh" ]]; then
    bash "${SCRIPT_DIR}/infra.sh" provider "$@"
  else
    log_error "infra.sh not found for legacy delegation"
    return 1
  fi
}

# ============================================================================
# MAIN ENTRY POINT
# ============================================================================

main() {
  local subcommand="${1:-}"
  shift || true

  case "$subcommand" in
    ""|help|-h|--help)
      show_cloud_help
      ;;
    status)
      cmd_cloud_status "$@"
      ;;
    upgrade)
      cmd_cloud_upgrade "$@"
      ;;
    destroy)
      cmd_cloud_destroy "$@"
      ;;
    # Legacy commands delegate to infra provider
    provider|server|cost|deploy)
      _cloud_legacy_delegate "$subcommand" "$@"
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      printf "\nRun 'nself cloud --help' for available subcommands.\n"
      return 1
      ;;
  esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
