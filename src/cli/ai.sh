#!/usr/bin/env bash
# ai.sh - AI plugin management for nself
# Manages accounts, authentication, usage, source-tier routing, and caller tokens.
#
# Commands:
#   nself ai connect      --provider <anthropic> [--label <name>] [--priority <n>]
#   nself ai auth login   --provider <anthropic|openai> [--label <name>] [--priority <n>]
#   nself ai auth refresh [<account_id>]
#   nself ai auth test
#   nself ai auth add     --provider <p> --key <k> [--label <l>] [--priority <n>]
#   nself ai auth list
#   nself ai auth remove  <account_id>
#   nself ai usage [--today|--week|--month]
#   nself ai stats [--today|--week|--month]
#   nself ai routing show
#   nself ai routing set  --class <task_class> --tier <local|free_gemini|api_key> --priority <n> [--disable|--enable]
#   nself ai tokens create <namespace> [--classes <c1,c2>] [--rpm <n>]
#   nself ai tokens list
#   nself ai tokens remove <namespace>
#   nself ai tokens test <token>
#
# Usage: nself ai <subcommand> [options]

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

ai_usage() {
  printf "nself ai — AI plugin account management\n\n"
  printf "Usage: nself ai <subcommand> [options]\n\n"
  printf "Subcommands:\n"
  printf "  status                                      Show AI service health, providers, queue depths\n"
  printf "  providers list                              List configured providers with status\n"
  printf "  providers set-key <provider> <key>          Store an API key for a provider\n"
  printf "  query \"<text>\" [--class=<class>]            Send a test query to the AI service\n"
  printf "  connect      --provider <anthropic>          OAuth2 PKCE login for subscription accounts\n"
  printf "  auth login   --provider <anthropic|openai>  Alias for connect\n"
  printf "  auth refresh [<account_id>]                 Force token refresh\n"
  printf "  auth test                                   Test all active AI accounts\n"
  printf "  auth add     --provider <p> --key <k>       Add an API key account\n"
  printf "  auth list                                   List all configured accounts\n"
  printf "  auth remove  <account_id>                   Deactivate an account\n"
  printf "  usage [--today|--week|--month]              Show AI usage log\n"
  printf "  stats [--today|--week|--month]              Show AI usage summary + savings\n"
  printf "  routing show                                Show source-tier routing config\n"
  printf "  routing set  --class <c> --tier <t> --priority <n> [--disable|--enable]\n"
  printf "                                              Update a routing entry\n"
  printf "  transcribe <audio-file> [--language <code>] Transcribe audio via Whisper\n"
  printf "  tokens create <namespace> [--classes <c1,c2>] [--rpm <n>]\n"
  printf "                                              Create a caller token\n"
  printf "  tokens list                                 List all caller tokens\n"
  printf "  tokens remove <namespace>                   Remove a caller token\n"
  printf "  tokens test <token>                         Test a caller token\n\n"
  printf "Environment:\n"
  printf "  NSELF_AI_URL            AI plugin base URL (default: http://localhost:3101)\n"
  printf "  PLUGIN_INTERNAL_SECRET  required for usage/stats/routing/tokens commands\n\n"
  printf "Examples:\n"
  printf "  nself ai status\n"
  printf "  nself ai providers list\n"
  printf "  nself ai providers set-key gemini AIzaSy...\n"
  printf "  nself ai query \"What is nself?\"\n"
  printf "  nself ai query \"Classify this text\" --class=Classify\n"
  printf "  nself ai auth login --provider anthropic\n"
  printf "  nself ai auth login --provider openai --label my-plus\n"
  printf "  nself ai auth add --provider anthropic --key sk-ant-xxx\n"
  printf "  nself ai auth list\n"
  printf "  nself ai auth test\n"
  printf "  nself ai usage --today\n"
  printf "  nself ai stats\n"
  printf "  nself ai routing show\n"
  printf "  nself ai routing set --class chat --tier local --priority 1\n"
  printf "  nself ai routing set --class code --tier free_gemini --priority 1 --enable\n"
  printf "  nself ai models list\n"
  printf "  nself ai models install --auto\n"
  printf "  nself ai models install --model phi4-mini\n"
  printf "  nself ai models status\n"
  printf "  nself ai models remove tinyllama\n"
  printf "  nself ai transcribe audio.ogg\n"
  printf "  nself ai transcribe audio.ogg --language en\n"
  printf "  nself ai tokens create nself-mux --classes Summarize,Faq --rpm 120\n"
  printf "  nself ai tokens list\n"
  printf "  nself ai tokens remove nself-mux\n"
  printf "  nself ai tokens test nself_at_xxxxx...\n"
}

# ============================================================================
# Top-level dispatcher
# ============================================================================

cmd_ai() {
  local subcommand="${1:-}"

  if [ -z "$subcommand" ]; then
    ai_usage
    exit 0
  fi

  shift

  case "$subcommand" in
    status)
      cmd_ai_status "$@"
      ;;
    providers)
      cmd_ai_providers "$@"
      ;;
    query)
      cmd_ai_query "$@"
      ;;
    connect)
      cmd_ai_connect "$@"
      ;;
    auth)
      cmd_ai_auth "$@"
      ;;
    usage)
      cmd_ai_usage "$@"
      ;;
    stats)
      cmd_ai_stats "$@"
      ;;
    routing)
      cmd_ai_routing "$@"
      ;;
    models)
      cmd_ai_models "$@"
      ;;
    transcribe)
      cmd_ai_transcribe "$@"
      ;;
    tokens)
      cmd_ai_tokens "$@"
      ;;
    help | --help | -h)
      ai_usage
      exit 0
      ;;
    *)
      cli_error "Unknown subcommand: $subcommand"
      printf "\n"
      ai_usage
      exit 1
      ;;
  esac
}

# ============================================================================
# T-0858: status — show AI service health, providers, queue depths, version
# ============================================================================

cmd_ai_status() {
  local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local response=""
  response=$(curl -s \
    -H "x-internal-token: ${internal_secret}" \
    "${ai_url}/ai/health" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from ai service at ${ai_url}. Is it running?"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    printf "\n\033[1mAI Service Status\033[0m\n"
    local status version queue_depth
    status=$(printf '%s' "$response" | jq -r '.status // "unknown"' 2>/dev/null)
    version=$(printf '%s' "$response" | jq -r '.version // "unknown"' 2>/dev/null)
    queue_depth=$(printf '%s' "$response" | jq -r '.queue_depth // 0' 2>/dev/null)

    local color="\033[0;32m"
    if [ "$status" != "ok" ] && [ "$status" != "healthy" ]; then
      color="\033[0;31m"
    fi

    printf "  Status:      ${color}%s\033[0m\n" "$status"
    printf "  Version:     %s\n" "$version"
    printf "  Queue depth: %s\n" "$queue_depth"

    # Print providers table if present
    local providers_json=""
    providers_json=$(printf '%s' "$response" | jq -r '.providers // empty' 2>/dev/null)
    if [ -n "$providers_json" ] && [ "$providers_json" != "null" ]; then
      printf "\n  %-20s %-12s %s\n" "PROVIDER" "STATUS" "OAUTH HEALTH"
      printf "  %-20s %-12s %s\n" "--------" "------" "------------"
      printf '%s' "$response" | jq -r '.providers[] | [.name, .status, (.oauth_healthy // "n/a" | tostring)] | @tsv' 2>/dev/null \
      | while IFS='	' read -r pname pstatus poauth; do
          local pcolor="\033[0;32m"
          if [ "$pstatus" != "ok" ] && [ "$pstatus" != "active" ]; then
            pcolor="\033[0;33m"
          fi
          printf "  %-20s ${pcolor}%-12s\033[0m %s\n" "$pname" "$pstatus" "$poauth"
        done
    fi
    printf "\n"
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# T-0858: providers — list providers or store API keys
# ============================================================================

cmd_ai_providers() {
  local subcmd="${1:-list}"
  shift || true

  local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$subcmd" in

    list)
      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local response=""
      response=$(curl -s \
        -H "x-internal-token: ${internal_secret}" \
        "${ai_url}/ai/providers" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      if command -v jq >/dev/null 2>&1; then
        printf "\n\033[1mConfigured AI Providers\033[0m\n"
        printf "%-20s %-12s %-15s %s\n" "PROVIDER" "STATUS" "TYPE" "MODELS"
        printf "%-20s %-12s %-15s %s\n" "--------" "------" "----" "------"
        printf '%s' "$response" | jq -r '.[] | [.name, .status, (.type // "api_key"), (.models // [] | join(",") | if . == "" then "(default)" else . end)] | @tsv' 2>/dev/null \
        | while IFS='	' read -r pname pstatus ptype pmodels; do
            local pcolor="\033[0;32m"
            if [ "$pstatus" != "ok" ] && [ "$pstatus" != "active" ] && [ "$pstatus" != "ready" ]; then
              pcolor="\033[0;33m"
            fi
            printf "%-20s ${pcolor}%-12s\033[0m %-15s %s\n" "$pname" "$pstatus" "$ptype" "$pmodels"
          done
        printf "\n"
      else
        printf '%s\n' "$response"
      fi
      ;;

    set-key)
      # Store an API key for a provider.
      # Usage: nself ai providers set-key <provider> <key>
      local provider="${1:-}"
      local api_key="${2:-}"

      if [ -z "$provider" ] || [ -z "$api_key" ]; then
        printf "Usage: nself ai providers set-key <provider> <key>\n" >&2
        printf "Example: nself ai providers set-key gemini AIzaSy...\n" >&2
        return 1
      fi

      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local body="{\"provider\":\"${provider}\",\"api_key\":\"${api_key}\"}"
      local response=""
      response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "x-internal-token: ${internal_secret}" \
        -d "$body" \
        "${ai_url}/ai/providers/key" 2>/dev/null)

      if [ -z "$response" ]; then
        log_success "API key stored for provider '${provider}'."
      else
        if command -v jq >/dev/null 2>&1; then
          local ok=""
          ok=$(printf '%s' "$response" | jq -r '.ok // .success // empty' 2>/dev/null)
          if [ -n "$ok" ] && [ "$ok" != "false" ]; then
            log_success "API key stored for provider '${provider}'."
          else
            printf '%s\n' "$response"
          fi
        else
          printf '%s\n' "$response"
        fi
      fi
      ;;

    help | --help | -h)
      printf "Usage: nself ai providers <list|set-key> [options]\n\n"
      printf "Subcommands:\n"
      printf "  list                        List all configured providers with status\n"
      printf "  set-key <provider> <key>    Store an API key for a provider\n\n"
      printf "Examples:\n"
      printf "  nself ai providers list\n"
      printf "  nself ai providers set-key gemini AIzaSy...\n"
      printf "  nself ai providers set-key openai sk-proj-...\n"
      ;;

    *)
      cli_error "Unknown providers action: $subcmd"
      printf "Actions: list, set-key\n"
      exit 1
      ;;
  esac
}

# ============================================================================
# T-0858: query — send a test query to the AI service
# ============================================================================

cmd_ai_query() {
  # Usage: nself ai query "<text>" [--class=<class>]
  local text="" task_class="Chat"

  # First positional arg is the text (unless it starts with --)
  case "${1:-}" in
    --help | -h | "")
      printf "Usage: nself ai query \"<text>\" [--class=<class>]\n\n"
      printf "  text         The message to send\n"
      printf "  --class      Task class (default: Chat). Examples: Chat, Code, Summarize, Classify\n\n"
      printf "Examples:\n"
      printf "  nself ai query \"What is nself?\"\n"
      printf "  nself ai query \"Classify this: I love it\" --class=Classify\n"
      return 0
      ;;
  esac

  if [ "${1:-}" != "" ] && [ "$(printf '%s' "${1:-}" | cut -c1-2)" != "--" ]; then
    text="$1"
    shift
  fi

  while [ $# -gt 0 ]; do
    case "$1" in
      --class=*) task_class="${1#--class=}"; shift ;;
      --class)   task_class="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  if [ -z "$text" ]; then
    cli_error "Query text required"
    printf "Usage: nself ai query \"<text>\" [--class=<class>]\n" >&2
    return 1
  fi

  local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  # Escape the text for JSON (basic escaping of quotes and backslashes)
  local escaped_text=""
  escaped_text=$(printf '%s' "$text" | sed 's/\\/\\\\/g; s/"/\\"/g')

  local body="{\"messages\":[{\"role\":\"user\",\"content\":\"${escaped_text}\"}],\"task_class\":\"${task_class}\"}"

  log_info "Sending query (class: ${task_class})..."

  local response=""
  response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "x-internal-token: ${internal_secret}" \
    -d "$body" \
    "${ai_url}/ai/complete" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from ai service at ${ai_url}. Is it running?"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local content provider model
    content=$(printf '%s' "$response" | jq -r '.choices[0].message.content // .content // .response // empty' 2>/dev/null)
    provider=$(printf '%s' "$response" | jq -r '.provider // empty' 2>/dev/null)
    model=$(printf '%s' "$response" | jq -r '.model // empty' 2>/dev/null)

    if [ -n "$content" ]; then
      printf "\n%s\n" "$content"
      if [ -n "$provider" ] || [ -n "$model" ]; then
        printf "\n\033[2m[%s/%s]\033[0m\n" "$provider" "$model"
      fi
    else
      printf '%s\n' "$response"
    fi
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# Auth subcommand dispatcher
# ============================================================================

cmd_ai_auth() {
  local subcmd="${1:-}"

  if [ -z "$subcmd" ]; then
    cli_error "Auth action required"
    printf "Actions: login, refresh, test, add, list, remove\n"
    exit 1
  fi

  shift

  case "$subcmd" in

    login)
      # OAuth2 PKCE login — delegates to cmd_ai_connect.
      # Usage: nself ai auth login --provider <anthropic|openai> [--label <name>] [--priority <n>]
      cmd_ai_connect "$@"
      ;;

    refresh)
      # Force token refresh for an account.
      # Usage: nself ai auth refresh [<account_id>]
      local account_id="${1:-}"
      local ai_url="${NSELF_AI_URL:-http://localhost:3710}"

      if [ -z "$account_id" ]; then
        curl -s -X POST "${ai_url}/credentials/oauth/refresh-all" 2>/dev/null
        printf "\033[0;32m[SUCCESS]\033[0m All OAuth tokens refreshed.\n"
      else
        curl -s -X POST "${ai_url}/credentials/oauth/refresh/${account_id}" 2>/dev/null
        printf "\033[0;32m[SUCCESS]\033[0m Token refreshed for account %s.\n" "$account_id"
      fi
      ;;

    test)
      # Test all active AI accounts.
      # Usage: nself ai auth test
      local ai_url="${NSELF_AI_URL:-http://localhost:3710}"
      local result=""
      result=$(curl -s "${ai_url}/credentials/test" 2>/dev/null)
      printf '%s\n' "$result"
      ;;

    add)
      # Add an API key account.
      # Usage: nself ai auth add --provider <p> --key <k> [--label <l>] [--priority <n>]
      local provider="" api_key="" label="" priority="1"
      while [ $# -gt 0 ]; do
        case "$1" in
          --provider) provider="$2"; shift 2 ;;
          --key)      api_key="$2";  shift 2 ;;
          --label)    label="$2";    shift 2 ;;
          --priority) priority="$2"; shift 2 ;;
          *)          shift ;;
        esac
      done

      if [ -z "$provider" ] || [ -z "$api_key" ]; then
        printf "Usage: nself ai auth add --provider <p> --key <k> [--label <l>] [--priority <n>]\n" >&2
        return 1
      fi

      local ai_url="${NSELF_AI_URL:-http://localhost:3710}"
      local body=""
      if [ -n "$label" ]; then
        body="{\"provider\":\"${provider}\",\"api_key\":\"${api_key}\",\"label\":\"${label}\",\"priority\":${priority}}"
      else
        body="{\"provider\":\"${provider}\",\"api_key\":\"${api_key}\",\"priority\":${priority}}"
      fi

      local resp=""
      resp=$(curl -s -X POST "${ai_url}/accounts" \
        -H "Content-Type: application/json" \
        -d "$body" 2>/dev/null)

      printf '%s\n' "$resp"
      ;;

    list)
      # List all configured accounts.
      # Usage: nself ai auth list
      local ai_url="${NSELF_AI_URL:-http://localhost:3710}"
      local result=""
      result=$(curl -s "${ai_url}/accounts" 2>/dev/null)
      printf '%s\n' "$result"
      ;;

    remove)
      # Deactivate an account.
      # Usage: nself ai auth remove <account_id>
      local account_id="${1:-}"
      if [ -z "$account_id" ]; then
        printf "Usage: nself ai auth remove <account_id>\n" >&2
        return 1
      fi

      local ai_url="${NSELF_AI_URL:-http://localhost:3710}"
      local result=""
      result=$(curl -s -X DELETE "${ai_url}/accounts/${account_id}" 2>/dev/null)
      printf '%s\n' "$result"
      ;;

    help | --help | -h)
      ai_usage
      exit 0
      ;;

    *)
      cli_error "Unknown auth action: $subcmd"
      printf "Actions: login, refresh, test, add, list, remove\n"
      exit 1
      ;;

  esac
}

# ============================================================================
# T-1225: connect — keychain bridge OAuth flow for subscription accounts
#
# NOTE: The PKCE browser flow (T-1220) is architecturally blocked because
# Anthropic's OAuth client (9d1c250a-...) has server-side redirect URI
# restrictions — http://localhost:PORT/callback is rejected by claude.ai
# when initiated externally. The flow only works from within Claude Code.
#
# Instead we use the keychain bridge (Option A): read the token that
# Claude Code already stored in the macOS keychain after its own login flow.
# ============================================================================

cmd_ai_connect() {
  # Usage: nself ai connect --provider <anthropic> [--label <name>] [--priority <n>] [--list]
  local provider="" label="" priority="1" do_list=false
  while [ $# -gt 0 ]; do
    case "$1" in
      --provider) provider="$2"; shift 2 ;;
      --label)    label="$2";    shift 2 ;;
      --priority) priority="$2"; shift 2 ;;
      --list)     do_list=true;  shift ;;
      --help | -h)
        printf "Usage: nself ai connect --provider <anthropic> [--label <name>] [--priority <n>]\n\n"
        printf "Reads the OAuth token from the Claude Code keychain entry and stores it in\n"
        printf "the AI plugin account pool. Requires the Claude Code CLI (claude) to be logged in.\n\n"
        printf "Options:\n"
        printf "  --provider <name>   Provider to connect. Currently only 'anthropic' is supported.\n"
        printf "  --label <name>      Friendly label for this account (default: anthropic-oauth-YYYYMMDD)\n"
        printf "  --priority <n>      Routing priority (default: 1; lower = higher priority)\n"
        printf "  --list              List currently connected OAuth accounts\n\n"
        printf "How it works:\n"
        printf "  1. You log into Claude Code CLI normally: claude /login\n"
        printf "  2. nself reads the resulting token from the macOS keychain\n"
        printf "  3. nself stores the token in the AI plugin (encrypted, with auto-refresh)\n\n"
        printf "Examples:\n"
        printf "  nself ai connect --provider anthropic\n"
        printf "  nself ai connect --provider anthropic --label my-claude-max --priority 1\n"
        printf "  nself ai connect --list\n"
        return 0
        ;;
      *) shift ;;
    esac
  done

  local ai_url="${NSELF_AI_URL:-http://localhost:3101}"

  if [ "$do_list" = "true" ]; then
    local list_resp
    list_resp=$(curl -s "${ai_url}/accounts" 2>/dev/null)
    printf '%s\n' "$list_resp" | grep -o '"id":"[^"]*"\|"provider":"[^"]*"\|"label":"[^"]*"\|"auth_type":"[^"]*"' | \
      awk -F'"' 'NR%4==0{printf "\n"} {printf "  %s: %s\n", $2, $4}' || \
      printf '%s\n' "$list_resp"
    return 0
  fi

  if [ -z "$provider" ]; then
    printf "Usage: nself ai connect --provider <anthropic> [--label <name>] [--priority <n>]\n" >&2
    printf "Run 'nself ai connect --help' for more information.\n" >&2
    return 1
  fi

  # Only anthropic is supported via keychain bridge
  case "$provider" in
    anthropic) ;;
    *)
      cli_error "Unsupported OAuth provider: ${provider}. Only 'anthropic' is supported."
      return 1
      ;;
  esac

  # Require macOS (keychain is macOS-specific)
  if [ "$(uname -s)" != "Darwin" ]; then
    cli_error "nself ai connect requires macOS (reads from macOS Keychain)."
    printf "\n  On Linux: set the token directly via environment variable:\n"
    printf "    ANTHROPIC_OAUTH_TOKEN_1=sk-ant-oat01-... nself start\n"
    printf "  Or store via REST API:\n"
    printf "    curl -X POST \${NSELF_AI_URL}/accounts/oauth -d '{...}'\n\n"
    return 1
  fi

  # Check for the security command
  if ! command -v security >/dev/null 2>&1; then
    cli_error "macOS 'security' command not found."
    return 1
  fi

  # Read the Claude Code credentials from the keychain
  local keychain_json
  keychain_json=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null || true)

  if [ -z "$keychain_json" ]; then
    cli_error "No Claude Code credentials found in keychain."
    printf "\n  To add credentials:\n"
    printf "  1. Install Claude Code CLI: pip install claude-code OR brew install claude-code\n"
    printf "  2. Log in: claude /login\n"
    printf "  3. Then retry: nself ai connect --provider anthropic\n\n"
    printf "  For accounts beyond the first, log out and back in:\n"
    printf "  1. claude /logout\n"
    printf "  2. claude /login   (log in as the next account)\n"
    printf "  3. nself ai connect --provider anthropic --label account2 --priority 2\n\n"
    return 1
  fi

  # Parse the JSON using python3 (required for Claude Code anyway)
  if ! command -v python3 >/dev/null 2>&1; then
    cli_error "python3 is required to parse the keychain entry."
    return 1
  fi

  local access_token refresh_token expires_at email
  access_token=$(printf '%s' "$keychain_json" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    # Try nested structure first (claudeAiOauth.accessToken), then flat
    oauth = d.get('claudeAiOauth') or d
    print(oauth.get('accessToken') or oauth.get('access_token') or '')
except Exception:
    print('')
" 2>/dev/null || true)

  refresh_token=$(printf '%s' "$keychain_json" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    oauth = d.get('claudeAiOauth') or d
    print(oauth.get('refreshToken') or oauth.get('refresh_token') or '')
except Exception:
    print('')
" 2>/dev/null || true)

  expires_at=$(printf '%s' "$keychain_json" | python3 -c "
import sys, json
from datetime import datetime, timedelta, timezone
try:
    d = json.load(sys.stdin)
    oauth = d.get('claudeAiOauth') or d
    exp = oauth.get('expiresAt') or oauth.get('expires_at')
    if exp:
        # Handle epoch millis or epoch seconds
        if isinstance(exp, (int, float)):
            ts = exp/1000.0 if exp > 1e10 else float(exp)
            print(datetime.fromtimestamp(ts, tz=timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ'))
        else:
            print(str(exp))
    else:
        # Default 10h from now (Claude Max tokens expire ~10h)
        print((datetime.now(tz=timezone.utc)+timedelta(hours=10)).strftime('%Y-%m-%dT%H:%M:%SZ'))
except Exception:
    from datetime import datetime, timedelta, timezone
    print((datetime.now(tz=timezone.utc)+timedelta(hours=10)).strftime('%Y-%m-%dT%H:%M:%SZ'))
" 2>/dev/null || true)

  email=$(printf '%s' "$keychain_json" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    oauth = d.get('claudeAiOauth') or d
    print(oauth.get('email') or oauth.get('userEmail') or '')
except Exception:
    print('')
" 2>/dev/null || true)

  if [ -z "$access_token" ]; then
    cli_error "Could not extract access token from keychain entry."
    printf "\n  The keychain entry may be in an unexpected format.\n"
    printf "  Try: security find-generic-password -s 'Claude Code-credentials' -w\n"
    printf "  and inspect the JSON structure.\n\n"
    return 1
  fi

  # Verify it looks like a Claude OAuth token
  case "$access_token" in
    sk-ant-oat01-*) ;;
    *)
      log_info "Warning: token prefix does not match expected Claude OAuth format (sk-ant-oat01-)"
      log_info "Proceeding anyway — the AI plugin will auto-detect auth type."
      ;;
  esac

  # Default label
  if [ -z "$label" ]; then
    if [ -n "$email" ]; then
      label="${provider}-$(printf '%s' "$email" | cut -d@ -f1)-$(date +%Y%m%d)"
    else
      label="${provider}-oauth-$(date +%Y%m%d)"
    fi
  fi

  log_info "Storing ${provider} OAuth token in AI plugin..."
  if [ -n "$email" ]; then
    log_info "Account: ${email}"
  fi

  # POST tokens to the AI plugin
  local body refresh_field
  if [ -n "$refresh_token" ]; then
    refresh_field=",\"refresh_token\":\"${refresh_token}\""
  else
    refresh_field=""
  fi
  local email_field=""
  if [ -n "$email" ]; then
    email_field=",\"email\":\"${email}\""
  fi

  body="{\"provider\":\"${provider}\",\"access_token\":\"${access_token}\"${refresh_field},\"expires_at\":\"${expires_at}\",\"label\":\"${label}\",\"priority\":${priority}${email_field}}"

  local store_resp stored_id
  store_resp=$(curl -s -X POST "${ai_url}/accounts/oauth" \
    -H "Content-Type: application/json" \
    -d "$body" 2>/dev/null)
  stored_id=$(printf '%s' "$store_resp" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

  if [ -z "$stored_id" ]; then
    cli_error "Failed to store OAuth token in AI plugin. Response: ${store_resp}"
    printf "\n  Make sure the AI plugin is running: nself service status ai\n\n"
    return 1
  fi

  if [ -n "$email" ]; then
    log_success "Connected ${email} as '${label}' (id: ${stored_id}, priority: ${priority})"
  else
    log_success "Connected! OAuth account stored as '${label}' (id: ${stored_id}, priority: ${priority})"
  fi

  if [ -n "$refresh_token" ]; then
    log_info "Refresh token stored — account will auto-refresh before expiry."
  else
    log_info "No refresh token found — you will need to reconnect when the token expires (~10h)."
  fi
}

# ============================================================================
# T-1023: usage — direct AI plugin usage log (same as nself claw usage)
# ============================================================================

cmd_ai_usage() {
  local period="all"
  local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  while [ $# -gt 0 ]; do
    case "$1" in
      --today)  period="today";  shift ;;
      --week)   period="week";   shift ;;
      --month)  period="month";  shift ;;
      *) shift ;;
    esac
  done

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local response=""
  response=$(curl -s \
    -H "x-internal-token: ${internal_secret}" \
    "${ai_url}/ai/usage?period=${period}" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from ai service at ${ai_url}. Is it running?"
    return 1
  fi

  printf '%s\n' "$response"
}

cmd_ai_stats() {
  local period="all"
  local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  while [ $# -gt 0 ]; do
    case "$1" in
      --today)  period="today";  shift ;;
      --week)   period="week";   shift ;;
      --month)  period="month";  shift ;;
      *) shift ;;
    esac
  done

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local response=""
  response=$(curl -s \
    -H "x-internal-token: ${internal_secret}" \
    "${ai_url}/ai/usage/summary?period=${period}" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from ai service at ${ai_url}. Is it running?"
    return 1
  fi

  printf '%s\n' "$response"
}

# ============================================================================
# T-1031: routing — view and update source-tier routing config
# ============================================================================

cmd_ai_routing() {
  local subcmd="${1:-show}"
  shift || true

  case "$subcmd" in
    show)
      # Show current routing config as a color-coded table.
      local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
      local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local response=""
      response=$(curl -s \
        -H "x-internal-token: ${internal_secret}" \
        "${ai_url}/ai/routing/config" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      # Pretty-print with jq if available, otherwise raw JSON
      if command -v jq >/dev/null 2>&1; then
        printf "\n\033[1mAI Source-Tier Routing Config\033[0m\n"
        printf "%-20s %-15s %-10s %-8s\n" "TASK CLASS" "TIER" "PRIORITY" "ENABLED"
        printf "%-20s %-15s %-10s %-8s\n" "----------" "----" "--------" "-------"
        printf '%s' "$response" | jq -r '.[] | [.task_class, .source_tier, (.priority|tostring), (.enabled|tostring)] | @tsv' 2>/dev/null \
        | while IFS='	' read -r task_class tier priority enabled; do
            local color=""
            case "$tier" in
              local)        color="\033[0;32m" ;;  # green
              free_gemini)  color="\033[0;33m" ;;  # yellow
              api_key)      color="\033[0;31m" ;;  # red
              *)            color="\033[0m"    ;;
            esac
            printf "%-20s ${color}%-15s\033[0m %-10s %-8s\n" "$task_class" "$tier" "$priority" "$enabled"
          done
        printf "\n\033[0;32mlocal\033[0m = free (Ollama)  \033[0;33mfree_gemini\033[0m = free (Gemini quota)  \033[0;31mapi_key\033[0m = paid\n\n"
      else
        printf '%s\n' "$response"
      fi
      ;;

    set)
      # Update a routing entry.
      # Usage: nself ai routing set --class <c> --tier <t> --priority <n> [--disable|--enable]
      local task_class="" source_tier="" priority="" enabled="true"
      while [ $# -gt 0 ]; do
        case "$1" in
          --class)    task_class="$2";  shift 2 ;;
          --tier)     source_tier="$2"; shift 2 ;;
          --priority) priority="$2";    shift 2 ;;
          --disable)  enabled="false";  shift ;;
          --enable)   enabled="true";   shift ;;
          *)          shift ;;
        esac
      done

      if [ -z "$task_class" ] || [ -z "$source_tier" ] || [ -z "$priority" ]; then
        printf "Usage: nself ai routing set --class <task_class> --tier <local|free_gemini|api_key> --priority <n> [--disable|--enable]\n" >&2
        return 1
      fi

      # Validate tier
      case "$source_tier" in
        local|free_gemini|api_key) ;;
        *)
          cli_error "Invalid tier '${source_tier}'. Must be: local, free_gemini, or api_key"
          return 1
          ;;
      esac

      local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
      local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local body="[{\"task_class\":\"${task_class}\",\"source_tier\":\"${source_tier}\",\"priority\":${priority},\"enabled\":${enabled}}]"

      local response=""
      response=$(curl -s -X PUT \
        -H "Content-Type: application/json" \
        -H "x-internal-token: ${internal_secret}" \
        -d "$body" \
        "${ai_url}/ai/routing/config" 2>/dev/null)

      if [ -z "$response" ]; then
        log_success "Routing updated: ${task_class} → ${source_tier} (priority ${priority}, enabled=${enabled})"
      else
        printf '%s\n' "$response"
      fi
      ;;

    help | --help | -h)
      printf "nself ai routing — view and update source-tier routing config\n\n"
      printf "Usage: nself ai routing <show|set> [options]\n\n"
      printf "Subcommands:\n"
      printf "  show                          Show routing config table\n"
      printf "  set --class <c> --tier <t> --priority <n> [--disable|--enable]\n"
      printf "                                Update a routing entry\n\n"
      printf "Tiers:\n"
      printf "  local        Ollama (free, local GPU/CPU)\n"
      printf "  free_gemini  Google Gemini free quota\n"
      printf "  api_key      Paid API (OpenAI / Anthropic)\n\n"
      printf "Examples:\n"
      printf "  nself ai routing show\n"
      printf "  nself ai routing set --class chat --tier local --priority 1\n"
      printf "  nself ai routing set --class code --tier free_gemini --priority 1 --enable\n"
      printf "  nself ai routing set --class reason --tier local --priority 3 --disable\n"
      ;;

    *)
      cli_error "Unknown routing action: $subcmd"
      printf "Actions: show, set\n"
      exit 1
      ;;
  esac
}

# ============================================================================
# T-1036: models — local model management
# ============================================================================

cmd_ai_models() {
  local subcmd="${1:-list}"
  shift || true

  local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$subcmd" in
    list)
      # Show model catalog with installed/recommended status.
      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local response=""
      response=$(curl -s \
        -H "x-internal-token: ${internal_secret}" \
        "${ai_url}/ai/models/local" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      if command -v jq >/dev/null 2>&1; then
        printf "\n\033[1mLocal Model Catalog\033[0m\n"
        printf "%-22s %-8s %-10s %-12s %-12s\n" "NAME" "PARAMS" "RAM REQ" "STATUS" "NOTE"
        printf "%-22s %-8s %-10s %-12s %-12s\n" "----" "------" "-------" "------" "----"
        printf '%s' "$response" | jq -r '.[] | [.name, ((.params_b|tostring)+"B"), ((.ram_gb_required|tostring)+"GB"), (if .installed then .status else "not installed" end), (if .recommended then "★ recommended" else "" end)] | @tsv' 2>/dev/null \
        | while IFS='	' read -r name params ram status note; do
            local color=""
            case "$status" in
              ready)       color="\033[0;32m" ;;
              downloading) color="\033[0;33m" ;;
              failed)      color="\033[0;31m" ;;
              *)           color="\033[0m"    ;;
            esac
            local note_color=""
            if [ -n "$note" ]; then
              note_color="\033[0;33m"
            fi
            printf "%-22s %-8s %-10s ${color}%-12s\033[0m ${note_color}%s\033[0m\n" "$name" "$params" "$ram" "$status" "$note"
          done
        printf "\n"
      else
        printf '%s\n' "$response"
      fi
      ;;

    install)
      # Install a local model (or auto-select best).
      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local model="" auto_select="false"
      while [ $# -gt 0 ]; do
        case "$1" in
          --model) model="$2"; shift 2 ;;
          --auto)  auto_select="true"; shift ;;
          *)       shift ;;
        esac
      done

      local body=""
      if [ "$auto_select" = "true" ]; then
        body='{"auto_select":true}'
      elif [ -n "$model" ]; then
        body="{\"model\":\"${model}\"}"
      else
        # Default: auto-select
        body='{"auto_select":true}'
      fi

      local response=""
      response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "x-internal-token: ${internal_secret}" \
        -d "$body" \
        "${ai_url}/ai/models/local/install" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      if command -v jq >/dev/null 2>&1; then
        local model_name="" eta=""
        model_name=$(printf '%s' "$response" | jq -r '.model // empty')
        eta=$(printf '%s' "$response" | jq -r '.estimated_minutes // empty')
        if [ -n "$model_name" ]; then
          log_info "Downloading ${model_name} (estimated: ${eta} min). Run 'nself ai models status' to check progress."
        else
          printf '%s\n' "$response"
        fi
      else
        printf '%s\n' "$response"
      fi
      ;;

    status)
      # Show installed models and their download status.
      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local response=""
      response=$(curl -s \
        -H "x-internal-token: ${internal_secret}" \
        "${ai_url}/ai/models/local" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      if command -v jq >/dev/null 2>&1; then
        printf "\n\033[1mInstalled Local Models\033[0m\n"
        printf "%-22s %-12s\n" "MODEL" "STATUS"
        printf "%-22s %-12s\n" "-----" "------"
        printf '%s' "$response" | jq -r '.[] | select(.installed == true) | [.name, .status] | @tsv' 2>/dev/null \
        | while IFS='	' read -r name status; do
            local color=""
            case "$status" in
              ready)       color="\033[0;32m" ;;
              downloading) color="\033[0;33m" ;;
              failed)      color="\033[0;31m" ;;
              *)           color="\033[0m"    ;;
            esac
            printf "%-22s ${color}%s\033[0m\n" "$name" "$status"
          done
        printf "\n"
      else
        printf '%s' "$response" | grep -v '"installed":false' 2>/dev/null || printf '%s\n' "$response"
      fi
      ;;

    remove)
      # Remove a local model.
      local model_name="${1:-}"
      if [ -z "$model_name" ]; then
        printf "Usage: nself ai models remove <model_name>\n" >&2
        return 1
      fi

      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local response=""
      response=$(curl -s -X DELETE \
        -H "x-internal-token: ${internal_secret}" \
        "${ai_url}/ai/models/local/remove/${model_name}" 2>/dev/null)

      if command -v jq >/dev/null 2>&1; then
        local removed=""
        removed=$(printf '%s' "$response" | jq -r '.removed // empty' 2>/dev/null)
        if [ -n "$removed" ]; then
          log_success "Model ${removed} removed."
        else
          printf '%s\n' "$response"
        fi
      else
        printf '%s\n' "$response"
      fi
      ;;

    help | --help | -h)
      printf "nself ai models — manage local AI models\n\n"
      printf "Usage: nself ai models <list|install|status|remove> [options]\n\n"
      printf "Subcommands:\n"
      printf "  list                     Show model catalog with installed status\n"
      printf "  install [--auto]         Install auto-selected best model for this VPS\n"
      printf "  install --model <name>   Install a specific model\n"
      printf "  status                   Show installed models and download progress\n"
      printf "  remove <model>           Remove an installed model\n\n"
      printf "Examples:\n"
      printf "  nself ai models list\n"
      printf "  nself ai models install --auto\n"
      printf "  nself ai models install --model mistral\n"
      printf "  nself ai models status\n"
      printf "  nself ai models remove tinyllama\n"
      ;;

    *)
      cli_error "Unknown models action: $subcmd"
      printf "Actions: list, install, status, remove\n"
      exit 1
      ;;
  esac
}

# ============================================================================
# transcribe subcommand — T-1118
# ============================================================================

cmd_ai_transcribe() {
  # Upload an audio file to POST /ai/transcribe and print the transcript.
  # Usage: nself ai transcribe <audio-file> [--language <code>]

  # Show help if first arg is --help/-h or no args given.
  case "${1:-}" in
    --help | -h | "")
      printf "Usage: nself ai transcribe <audio-file> [--language <code>]\n\n"
      printf "  audio-file   Path to OGG, WAV, MP3, or M4A file\n"
      printf "  --language   Language code (default: auto-detect). Examples: en, ar, fr\n\n"
      printf "Requires Whisper installed: nself ai models install --model openai/whisper\n"
      return 0
      ;;
  esac

  local audio_file="${1:-}"
  shift || true

  local language=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --language) language="$2"; shift 2 ;;
      -l)         language="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  if [ -z "$audio_file" ]; then
    cli_error "Audio file path required"
    printf "Usage: nself ai transcribe <audio-file> [--language <code>]\n" >&2
    exit 1
  fi

  if [ ! -f "$audio_file" ]; then
    cli_error "File not found: $audio_file"
    exit 1
  fi

  local ai_url="${NSELF_AI_URL:-http://localhost:3710}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET is required"
    exit 1
  fi

  log_info "Transcribing $(basename "$audio_file") ..."

  local curl_args=()
  curl_args+=(-s -X POST)
  curl_args+=(-H "x-internal-secret: $internal_secret")
  curl_args+=(-F "audio=@${audio_file}")
  if [ -n "$language" ]; then
    curl_args+=(-F "language=$language")
  fi
  curl_args+=("${ai_url}/ai/transcribe")

  local response=""
  response=$(curl "${curl_args[@]}" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from /ai/transcribe. Is nself-ai running?"
    exit 1
  fi

  local http_status=""
  if command -v jq >/dev/null 2>&1; then
    http_status=$(printf '%s' "$response" | jq -r '.status // empty' 2>/dev/null)
  fi

  # Handle 503 — Whisper not installed.
  if [ "$http_status" = "503" ] || printf '%s' "$response" | grep -q '"no_whisper"'; then
    local msg=""
    msg=$(printf '%s' "$response" | jq -r '.message // .error // "Whisper not installed"' 2>/dev/null \
      || printf "Whisper not installed")
    cli_error "$msg"
    printf "Install Whisper: nself ai models install --model openai/whisper\n" >&2
    exit 1
  fi

  # Print transcript to stdout.
  if command -v jq >/dev/null 2>&1; then
    local transcript duration
    transcript=$(printf '%s' "$response" | jq -r '.transcript // .text // empty' 2>/dev/null)
    duration=$(printf '%s' "$response" | jq -r '.duration_seconds // empty' 2>/dev/null)
    if [ -n "$transcript" ]; then
      printf '%s\n' "$transcript"
      if [ -n "$duration" ]; then
        printf "\n\033[2mDuration: %ss\033[0m\n" "$duration"
      fi
    else
      printf '%s\n' "$response"
    fi
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# T-0804: tokens — create/list/remove/test caller tokens for X-Ai-Token auth
# ============================================================================

cmd_ai_tokens() {
  local subcmd="${1:-}"

  if [ -z "$subcmd" ]; then
    printf "Usage: nself ai tokens <create|list|remove|test> [options]\n" >&2
    return 1
  fi

  shift

  local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$subcmd" in

    create)
      # Generate a new caller token for a namespace.
      # Usage: nself ai tokens create <namespace> [--classes <c1,c2>] [--rpm <n>]
      local namespace="${1:-}"
      if [ -z "$namespace" ]; then
        printf "Usage: nself ai tokens create <namespace> [--classes <c1,c2>] [--rpm <n>]\n" >&2
        return 1
      fi
      shift

      local allowed_classes="" rate_limit_rpm="60"
      while [ $# -gt 0 ]; do
        case "$1" in
          --classes) allowed_classes="$2"; shift 2 ;;
          --rpm)     rate_limit_rpm="$2";  shift 2 ;;
          *)         shift ;;
        esac
      done

      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      # Build JSON body. allowed_classes is a comma-separated string converted
      # to a JSON array without using bash arrays (Bash 3.2 compatible).
      local classes_json="[]"
      if [ -n "$allowed_classes" ]; then
        # Convert "Summarize,Faq,Translate" → ["Summarize","Faq","Translate"]
        classes_json="["
        local first_class=1
        local IFS_SAVE="$IFS"
        IFS=","
        for cls in $allowed_classes; do
          if [ "$first_class" = "1" ]; then
            classes_json="${classes_json}\"${cls}\""
            first_class=0
          else
            classes_json="${classes_json},\"${cls}\""
          fi
        done
        IFS="$IFS_SAVE"
        classes_json="${classes_json}]"
      fi

      local body="{\"namespace\":\"${namespace}\",\"allowed_classes\":${classes_json},\"rate_limit_rpm\":${rate_limit_rpm}}"

      local response=""
      response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "x-internal-token: ${internal_secret}" \
        -d "$body" \
        "${ai_url}/internal/tokens/create" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      if command -v jq >/dev/null 2>&1; then
        local token="" ns=""
        token=$(printf '%s' "$response" | jq -r '.token // empty' 2>/dev/null)
        ns=$(printf '%s' "$response" | jq -r '.namespace // empty' 2>/dev/null)
        if [ -n "$token" ]; then
          log_success "Caller token created for namespace '${ns}'"
          printf "\n  Token: %s\n\n" "$token"
          printf "  Store this token securely — it will not be shown again.\n"
          printf "  Use it in requests as: X-Ai-Token: %s\n\n" "$token"
        else
          printf '%s\n' "$response"
        fi
      else
        printf '%s\n' "$response"
      fi
      ;;

    list)
      # List all caller tokens (namespaces + metadata, not raw tokens).
      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local response=""
      response=$(curl -s \
        -H "x-internal-token: ${internal_secret}" \
        "${ai_url}/internal/tokens" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      if command -v jq >/dev/null 2>&1; then
        printf "\n\033[1mCaller Tokens\033[0m\n"
        printf "%-30s %-8s %-10s %s\n" "NAMESPACE" "RPM" "ENABLED" "ALLOWED CLASSES"
        printf "%-30s %-8s %-10s %s\n" "---------" "---" "-------" "---------------"
        printf '%s' "$response" | jq -r '.[] | [.namespace, (.rate_limit_rpm|tostring), (.is_enabled|tostring), (.allowed_classes | if length == 0 then "(all)" else join(",") end)] | @tsv' 2>/dev/null \
        | while IFS='	' read -r ns rpm enabled classes; do
            local color="\033[0;32m"
            if [ "$enabled" = "false" ]; then
              color="\033[0;31m"
            fi
            printf "%-30s %-8s ${color}%-10s\033[0m %s\n" "$ns" "$rpm" "$enabled" "$classes"
          done
        printf "\n"
      else
        printf '%s\n' "$response"
      fi
      ;;

    remove)
      # Remove a caller token by namespace.
      # Usage: nself ai tokens remove <namespace>
      local namespace="${1:-}"
      if [ -z "$namespace" ]; then
        printf "Usage: nself ai tokens remove <namespace>\n" >&2
        return 1
      fi

      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      local response=""
      response=$(curl -s -X DELETE \
        -H "x-internal-token: ${internal_secret}" \
        "${ai_url}/internal/tokens/${namespace}" 2>/dev/null)

      if command -v jq >/dev/null 2>&1; then
        local removed=""
        removed=$(printf '%s' "$response" | jq -r '.removed // empty' 2>/dev/null)
        if [ -n "$removed" ]; then
          log_success "Caller token removed for namespace '${removed}'."
        else
          printf '%s\n' "$response"
        fi
      else
        printf '%s\n' "$response"
      fi
      ;;

    rotate-key)
      # Re-encrypt all saved OAuth tokens with a new key and update .env
      # Usage: nself ai tokens rotate-key
      local ai_url="${NSELF_AI_URL:-http://localhost:3101}"
      local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

      if [ -z "$internal_secret" ]; then
        cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
        return 1
      fi

      log_info "Generating new AI plugin encryption key..."
      local new_key
      new_key=$(openssl rand -base64 32)
      
      log_info "Instructing nself-ai server to re-encrypt stored OAuth tokens..."
      local body="{\"new_key\":\"${new_key}\"}"
      
      local response=""
      response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "x-internal-token: ${internal_secret}" \
        -d "$body" \
        "${ai_url}/tokens/rotate-key" 2>/dev/null)
        
      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      local ok="" rotated_count="0"
      if command -v jq >/dev/null 2>&1; then
        ok=$(printf '%s' "$response" | jq -r 'if .status == "success" then "true" else "false" end' 2>/dev/null)
        rotated_count=$(printf '%s' "$response" | jq -r '.rotated_count // 0' 2>/dev/null)
      else
        if printf '%s' "$response" | grep -q '"status":"success"'; then
          ok="true"
        fi
      fi
      
      if [ "$ok" = "true" ]; then
        log_success "Tokens re-encrypted successfully ($rotated_count tokens rotated)."
        
        # Update .env
        local env_file="$NSELF_ROOT/.env"
        if [ -f "$env_file" ]; then
          log_info "Updating PLUGIN_AI_ENCRYPTION_KEY in .env..."
          if grep -q "^PLUGIN_AI_ENCRYPTION_KEY=" "$env_file"; then
            if [[ "$OSTYPE" == "darwin"* ]]; then
              sed -i '' "s|^PLUGIN_AI_ENCRYPTION_KEY=.*|PLUGIN_AI_ENCRYPTION_KEY=\"$new_key\"|g" "$env_file"
            else
              sed -i "s|^PLUGIN_AI_ENCRYPTION_KEY=.*|PLUGIN_AI_ENCRYPTION_KEY=\"$new_key\"|g" "$env_file"
            fi
          else
            echo "PLUGIN_AI_ENCRYPTION_KEY=\"$new_key\"" >> "$env_file"
          fi
          log_success "Key rotation complete! You MUST restart the AI service to apply the new key."
          printf "\n  nself plugin restart ai\n"
        else
          cli_error "Could not find .env file at $env_file. You must manually set:"
          printf "  PLUGIN_AI_ENCRYPTION_KEY=\"%s\"\n" "$new_key"
        fi
      else
        cli_error "Key rotation failed."
        printf '%s\n' "$response"
        return 1
      fi
      ;;

    test)
      # Test whether a caller token is valid and show its permissions.
      # Usage: nself ai tokens test <token>
      local token="${1:-}"
      if [ -z "$token" ]; then
        printf "Usage: nself ai tokens test <token>\n" >&2
        return 1
      fi

      local response=""
      response=$(curl -s \
        -H "x-ai-token: ${token}" \
        "${ai_url}/ai/complete" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"messages":[{"role":"user","content":"ping"}],"max_tokens":1,"task_class":"Classify"}' \
        2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from ai service at ${ai_url}. Is it running?"
        return 1
      fi

      # A 401 response means the token is invalid. Any other response means valid.
      if printf '%s' "$response" | grep -q '"missing_token"\|"invalid_token"\|"token_disabled"'; then
        local error_code=""
        if command -v jq >/dev/null 2>&1; then
          error_code=$(printf '%s' "$response" | jq -r '.error // empty' 2>/dev/null)
        fi
        cli_error "Token invalid or disabled (${error_code:-unknown})."
        return 1
      else
        log_success "Token is valid and accepted by the AI service."
        if command -v jq >/dev/null 2>&1; then
          local provider=""
          provider=$(printf '%s' "$response" | jq -r '.provider // empty' 2>/dev/null)
          if [ -n "$provider" ]; then
            printf "  Provider: %s\n" "$provider"
          fi
        fi
      fi
      ;;

    help | --help | -h)
      printf "nself ai tokens — manage caller tokens for X-Ai-Token authentication\n\n"
      printf "Usage: nself ai tokens <create|list|remove|test> [options]\n\n"
      printf "Subcommands:\n"
      printf "  create <namespace> [--classes <c1,c2>] [--rpm <n>]\n"
      printf "                         Create a new caller token for a namespace\n"
      printf "  list                   List all namespaces and token metadata\n"
      printf "  remove <namespace>     Remove the caller token for a namespace\n"
      printf "  test <token>           Verify a token is accepted by the AI service\n"
      printf "  rotate-key             Re-encrypt all stored OAuth tokens with a new key\n\n"
      printf "Options for create:\n"
      printf "  --classes <c1,c2>  Comma-separated allowed task classes (default: all)\n"
      printf "                     Classes: Classify, Summarize, Faq, Translate,\n"
      printf "                     Chat, Code, Search, Sensitive, Legal, Medical,\n"
      printf "                     LongContext, Embed\n"
      printf "  --rpm <n>          Rate limit in requests/min (default: 60)\n\n"
      printf "Examples:\n"
      printf "  nself ai tokens create nself-mux --classes Summarize,Faq --rpm 120\n"
      printf "  nself ai tokens create nself-claw\n"
      printf "  nself ai tokens list\n"
      printf "  nself ai tokens remove nself-mux\n"
      printf "  nself ai tokens test nself_at_xxxxx...\n"
      ;;

    *)
      cli_error "Unknown tokens action: $subcmd"
      printf "Actions: create, list, remove, test, rotate-key\n"
      exit 1
      ;;

  esac
}

# ============================================================================
# Entry point
# ============================================================================

cmd_ai "$@"
