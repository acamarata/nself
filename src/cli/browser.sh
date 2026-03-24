#!/usr/bin/env bash
# browser.sh - Browser plugin management for nself
# Manages headless browser automation via the nself-browser pro plugin.
#
# Commands:
#   nself browser status                                  Show browser service health
#   nself browser screenshot <url> [--output=<file>]      Capture a full-page screenshot (PNG)
#   nself browser scrape <url>                            Extract page text content
#   nself browser pdf <url> [--output=<file>]             Render page as PDF
#   nself browser execute <url> "<javascript>"            Execute JavaScript on a page
#   nself browser allowlist add <domain>                  Add domain to allowlist
#   nself browser allowlist remove <domain>               Remove domain from allowlist
#   nself browser allowlist list                          List allowlisted domains
#
# Usage: nself browser <subcommand> [options]

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
# Plugin check helper
# ============================================================================

_browser_check_plugin() {
  local plugin_dir="${NSELF_PLUGIN_DIR:-${NSELF_DATA_DIR:-/opt/nself}/plugins}"

  # Check via env var first (set when plugin is running)
  if [ -n "${PLUGIN_BROWSER_INTERNAL_URL:-}" ]; then
    return 0
  fi

  # Check for plugin.json on disk
  if [ -f "${plugin_dir}/browser/plugin.json" ]; then
    return 0
  fi

  cli_error "nself-browser plugin is not installed."
  printf "Install it: nself plugin install browser\n" >&2
  return 1
}

# ============================================================================
# Usage
# ============================================================================

browser_usage() {
  printf "nself browser — headless browser automation\n\n"
  printf "Usage: nself browser <subcommand> [options]\n\n"
  printf "Subcommands:\n"
  printf "  status                                 Show browser service health and version\n"
  printf "  screenshot <url> [--output=<file>]     Capture a full-page screenshot (PNG)\n"
  printf "  scrape <url>                           Extract and print page text content\n"
  printf "  pdf <url> [--output=<file>]            Render page as PDF\n"
  printf "  execute <url> \"<javascript>\"           Execute JavaScript on a page, print result\n"
  printf "  allowlist add <domain>                 Add a domain to the browser allowlist\n"
  printf "  allowlist remove <domain>              Remove a domain from the allowlist\n"
  printf "  allowlist list                         List all allowlisted domains\n\n"
  printf "Environment:\n"
  printf "  NSELF_BROWSER_URL           Browser plugin base URL (default: http://localhost:3715)\n"
  printf "  PLUGIN_BROWSER_INTERNAL_URL Override plugin URL\n"
  printf "  PLUGIN_INTERNAL_SECRET      Required for all commands\n\n"
  printf "Examples:\n"
  printf "  nself browser status\n"
  printf "  nself browser screenshot https://nself.org\n"
  printf "  nself browser screenshot https://nself.org --output=/tmp/nself.png\n"
  printf "  nself browser scrape https://nself.org/docs\n"
  printf "  nself browser pdf https://nself.org --output=/tmp/nself.pdf\n"
  printf "  nself browser execute https://nself.org \"document.title\"\n"
  printf "  nself browser allowlist add nself.org\n"
  printf "  nself browser allowlist list\n"
  printf "  nself browser allowlist remove nself.org\n"
}

# ============================================================================
# Top-level dispatcher
# ============================================================================

cmd_browser() {
  local subcommand="${1:-}"

  if [ -z "$subcommand" ]; then
    browser_usage
    exit 0
  fi

  shift

  case "$subcommand" in
    status)
      cmd_browser_status "$@"
      ;;
    screenshot)
      cmd_browser_screenshot "$@"
      ;;
    scrape)
      cmd_browser_scrape "$@"
      ;;
    pdf)
      cmd_browser_pdf "$@"
      ;;
    execute)
      cmd_browser_execute "$@"
      ;;
    allowlist)
      cmd_browser_allowlist "$@"
      ;;
    help | --help | -h)
      browser_usage
      exit 0
      ;;
    *)
      cli_error "Unknown subcommand: $subcommand"
      printf "\n"
      browser_usage
      exit 1
      ;;
  esac
}

# ============================================================================
# T-0861: status — show browser service health
# ============================================================================

cmd_browser_status() {
  _browser_check_plugin || return 1

  local browser_url="${PLUGIN_BROWSER_INTERNAL_URL:-${NSELF_BROWSER_URL:-http://localhost:3715}}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local response=""
  response=$(curl -s \
    -H "x-internal-token: ${internal_secret}" \
    "${browser_url}/browser/health" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from browser service at ${browser_url}. Is nself-browser running?"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    printf "\n\033[1mBrowser Service Status\033[0m\n"
    local status version engine
    status=$(printf '%s' "$response" | jq -r '.status // "unknown"' 2>/dev/null)
    version=$(printf '%s' "$response" | jq -r '.version // "unknown"' 2>/dev/null)
    engine=$(printf '%s' "$response" | jq -r '.engine // "chromium"' 2>/dev/null)

    local color="\033[0;32m"
    if [ "$status" != "ok" ] && [ "$status" != "healthy" ]; then
      color="\033[0;31m"
    fi

    printf "  Status:  ${color}%s\033[0m\n" "$status"
    printf "  Version: %s\n" "$version"
    printf "  Engine:  %s\n" "$engine"
    printf "\n"
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# T-0861: screenshot — capture a full-page screenshot
# ============================================================================

cmd_browser_screenshot() {
  _browser_check_plugin || return 1

  local url="${1:-}"
  local output_file=""
  local browser_url="${PLUGIN_BROWSER_INTERNAL_URL:-${NSELF_BROWSER_URL:-http://localhost:3715}}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$url" in
    --help | -h | "")
      printf "Usage: nself browser screenshot <url> [--output=<file>]\n\n"
      printf "  url       URL to capture\n"
      printf "  --output  Output PNG file path (default: /tmp/nself-screenshot-<timestamp>.png)\n\n"
      printf "Examples:\n"
      printf "  nself browser screenshot https://nself.org\n"
      printf "  nself browser screenshot https://nself.org --output=/tmp/nself.png\n"
      return 0
      ;;
  esac

  shift || true
  while [ $# -gt 0 ]; do
    case "$1" in
      --output=*) output_file="${1#--output=}"; shift ;;
      --output)   output_file="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  if [ -z "$output_file" ]; then
    local ts=""
    ts=$(date +%s)
    output_file="/tmp/nself-screenshot-${ts}.png"
  fi

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local escaped_url=""
  escaped_url=$(printf '%s' "$url" | sed 's/\\/\\\\/g; s/"/\\"/g')

  log_info "Capturing screenshot of ${url}..."

  local http_code=""
  http_code=$(curl -s -o "$output_file" -w "%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "x-internal-token: ${internal_secret}" \
    -d "{\"url\":\"${escaped_url}\"}" \
    "${browser_url}/browser/screenshot" 2>/dev/null)

  if [ "$http_code" != "200" ]; then
    local err_body=""
    err_body=$(cat "$output_file" 2>/dev/null || printf "")
    rm -f "$output_file" 2>/dev/null || true
    cli_error "Screenshot failed (HTTP ${http_code}): ${err_body}"
    return 1
  fi

  log_success "Screenshot saved to: ${output_file}"
}

# ============================================================================
# T-0861: scrape — extract page text content
# ============================================================================

cmd_browser_scrape() {
  _browser_check_plugin || return 1

  local url="${1:-}"
  local browser_url="${PLUGIN_BROWSER_INTERNAL_URL:-${NSELF_BROWSER_URL:-http://localhost:3715}}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$url" in
    --help | -h | "")
      printf "Usage: nself browser scrape <url>\n\n"
      printf "  url   URL to scrape\n\n"
      printf "Example: nself browser scrape https://nself.org/docs\n"
      return 0
      ;;
  esac

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local escaped_url=""
  escaped_url=$(printf '%s' "$url" | sed 's/\\/\\\\/g; s/"/\\"/g')

  log_info "Scraping ${url}..."

  local response=""
  response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "x-internal-token: ${internal_secret}" \
    -d "{\"url\":\"${escaped_url}\"}" \
    "${browser_url}/browser/scrape" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from browser service at ${browser_url}. Is nself-browser running?"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local text title
    text=$(printf '%s' "$response" | jq -r '.text // .content // empty' 2>/dev/null)
    title=$(printf '%s' "$response" | jq -r '.title // empty' 2>/dev/null)
    if [ -n "$text" ]; then
      if [ -n "$title" ]; then
        printf "\033[1m%s\033[0m\n\n" "$title"
      fi
      printf '%s\n' "$text"
    else
      printf '%s\n' "$response"
    fi
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# T-0861: pdf — render page as PDF
# ============================================================================

cmd_browser_pdf() {
  _browser_check_plugin || return 1

  local url="${1:-}"
  local output_file=""
  local browser_url="${PLUGIN_BROWSER_INTERNAL_URL:-${NSELF_BROWSER_URL:-http://localhost:3715}}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$url" in
    --help | -h | "")
      printf "Usage: nself browser pdf <url> [--output=<file>]\n\n"
      printf "  url       URL to render\n"
      printf "  --output  Output PDF file path (default: /tmp/nself-page-<timestamp>.pdf)\n\n"
      printf "Examples:\n"
      printf "  nself browser pdf https://nself.org\n"
      printf "  nself browser pdf https://nself.org/docs --output=/tmp/docs.pdf\n"
      return 0
      ;;
  esac

  shift || true
  while [ $# -gt 0 ]; do
    case "$1" in
      --output=*) output_file="${1#--output=}"; shift ;;
      --output)   output_file="$2"; shift 2 ;;
      *) shift ;;
    esac
  done

  if [ -z "$output_file" ]; then
    local ts=""
    ts=$(date +%s)
    output_file="/tmp/nself-page-${ts}.pdf"
  fi

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local escaped_url=""
  escaped_url=$(printf '%s' "$url" | sed 's/\\/\\\\/g; s/"/\\"/g')

  log_info "Rendering PDF for ${url}..."

  local http_code=""
  http_code=$(curl -s -o "$output_file" -w "%{http_code}" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "x-internal-token: ${internal_secret}" \
    -d "{\"url\":\"${escaped_url}\"}" \
    "${browser_url}/browser/pdf" 2>/dev/null)

  if [ "$http_code" != "200" ]; then
    local err_body=""
    err_body=$(cat "$output_file" 2>/dev/null || printf "")
    rm -f "$output_file" 2>/dev/null || true
    cli_error "PDF render failed (HTTP ${http_code}): ${err_body}"
    return 1
  fi

  log_success "PDF saved to: ${output_file}"
}

# ============================================================================
# T-0861: execute — run JavaScript on a page
# ============================================================================

cmd_browser_execute() {
  _browser_check_plugin || return 1

  local url="${1:-}"
  local javascript="${2:-}"
  local browser_url="${PLUGIN_BROWSER_INTERNAL_URL:-${NSELF_BROWSER_URL:-http://localhost:3715}}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  case "$url" in
    --help | -h | "")
      printf "Usage: nself browser execute <url> \"<javascript>\"\n\n"
      printf "  url         URL to load before executing script\n"
      printf "  javascript  JavaScript expression to evaluate\n\n"
      printf "Examples:\n"
      printf "  nself browser execute https://nself.org \"document.title\"\n"
      printf "  nself browser execute https://nself.org \"document.querySelectorAll('a').length\"\n"
      return 0
      ;;
  esac

  if [ -z "$javascript" ]; then
    cli_error "JavaScript expression required"
    printf "Usage: nself browser execute <url> \"<javascript>\"\n" >&2
    return 1
  fi

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  local escaped_url escaped_js
  escaped_url=$(printf '%s' "$url" | sed 's/\\/\\\\/g; s/"/\\"/g')
  escaped_js=$(printf '%s' "$javascript" | sed 's/\\/\\\\/g; s/"/\\"/g')

  local response=""
  response=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -H "x-internal-token: ${internal_secret}" \
    -d "{\"url\":\"${escaped_url}\",\"script\":\"${escaped_js}\"}" \
    "${browser_url}/browser/execute" 2>/dev/null)

  if [ -z "$response" ]; then
    cli_error "No response from browser service at ${browser_url}. Is nself-browser running?"
    return 1
  fi

  if command -v jq >/dev/null 2>&1; then
    local result=""
    result=$(printf '%s' "$response" | jq -r '.result // .value // empty' 2>/dev/null)
    if [ -n "$result" ]; then
      printf '%s\n' "$result"
    else
      printf '%s\n' "$response"
    fi
  else
    printf '%s\n' "$response"
  fi
}

# ============================================================================
# T-0861: allowlist — manage browser domain allowlist
# ============================================================================

cmd_browser_allowlist() {
  _browser_check_plugin || return 1

  local subcmd="${1:-list}"
  shift || true

  local browser_url="${PLUGIN_BROWSER_INTERNAL_URL:-${NSELF_BROWSER_URL:-http://localhost:3715}}"
  local internal_secret="${PLUGIN_INTERNAL_SECRET:-}"

  if [ -z "$internal_secret" ]; then
    cli_error "PLUGIN_INTERNAL_SECRET not set. Source your .env file first."
    return 1
  fi

  case "$subcmd" in

    list)
      local response=""
      response=$(curl -s \
        -H "x-internal-token: ${internal_secret}" \
        "${browser_url}/browser/allowlist" 2>/dev/null)

      if [ -z "$response" ]; then
        cli_error "No response from browser service at ${browser_url}. Is nself-browser running?"
        return 1
      fi

      if command -v jq >/dev/null 2>&1; then
        local count=""
        count=$(printf '%s' "$response" | jq '. | if type == "array" then length else (.domains // [] | length) end' 2>/dev/null || printf "0")
        if [ "${count:-0}" -eq 0 ]; then
          printf "No domains in allowlist. All domains allowed (no restrictions).\n"
          return 0
        fi
        printf "\n\033[1mAllowlisted Domains\033[0m\n"
        printf '%s' "$response" | jq -r '(if type == "array" then . else .domains // [] end) | .[]' 2>/dev/null \
        | while IFS='' read -r domain; do
            printf "  %s\n" "$domain"
          done
        printf "\n"
      else
        printf '%s\n' "$response"
      fi
      ;;

    add)
      local domain="${1:-}"
      if [ -z "$domain" ]; then
        printf "Usage: nself browser allowlist add <domain>\n" >&2
        return 1
      fi

      local escaped_domain=""
      escaped_domain=$(printf '%s' "$domain" | sed 's/\\/\\\\/g; s/"/\\"/g')

      local response=""
      response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "x-internal-token: ${internal_secret}" \
        -d "{\"domain\":\"${escaped_domain}\"}" \
        "${browser_url}/browser/allowlist" 2>/dev/null)

      if [ -z "$response" ]; then
        log_success "Domain '${domain}' added to allowlist."
        return 0
      fi

      if command -v jq >/dev/null 2>&1; then
        local ok=""
        ok=$(printf '%s' "$response" | jq -r '.ok // .success // .added // empty' 2>/dev/null)
        if [ -n "$ok" ] && [ "$ok" != "false" ]; then
          log_success "Domain '${domain}' added to allowlist."
        else
          printf '%s\n' "$response"
        fi
      else
        log_success "Domain '${domain}' added to allowlist."
      fi
      ;;

    remove)
      local domain="${1:-}"
      if [ -z "$domain" ]; then
        printf "Usage: nself browser allowlist remove <domain>\n" >&2
        return 1
      fi

      local escaped_domain=""
      escaped_domain=$(printf '%s' "$domain" | sed 's/\\/\\\\/g; s/"/\\"/g')

      local response=""
      response=$(curl -s -X DELETE \
        -H "x-internal-token: ${internal_secret}" \
        "${browser_url}/browser/allowlist/${escaped_domain}" 2>/dev/null)

      if command -v jq >/dev/null 2>&1; then
        local ok=""
        ok=$(printf '%s' "$response" | jq -r '.ok // .removed // empty' 2>/dev/null)
        if [ -n "$ok" ] && [ "$ok" != "false" ]; then
          log_success "Domain '${domain}' removed from allowlist."
        else
          printf '%s\n' "$response"
        fi
      else
        log_success "Domain '${domain}' removed from allowlist."
      fi
      ;;

    help | --help | -h)
      printf "Usage: nself browser allowlist <add|remove|list> [domain]\n\n"
      printf "Subcommands:\n"
      printf "  list              List all allowlisted domains\n"
      printf "  add <domain>      Add a domain to the allowlist\n"
      printf "  remove <domain>   Remove a domain from the allowlist\n\n"
      printf "Examples:\n"
      printf "  nself browser allowlist list\n"
      printf "  nself browser allowlist add nself.org\n"
      printf "  nself browser allowlist remove nself.org\n"
      ;;

    *)
      cli_error "Unknown allowlist action: $subcmd"
      printf "Actions: list, add, remove\n"
      exit 1
      ;;
  esac
}

# ============================================================================
# Entry point
# ============================================================================

cmd_browser "$@"
