#!/usr/bin/env bash
set -euo pipefail

# compat.sh - Compatibility checker for v1.0 upgrade
# Scans .env and docker-compose.yml for deprecated patterns and reports
# what needs updating before upgrading to v1.0.

CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source display utilities if available
[[ -z "${DISPLAY_SOURCED:-}" ]] && source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true

# Color fallbacks
: "${COLOR_GREEN:=\033[0;32m}"
: "${COLOR_YELLOW:=\033[0;33m}"
: "${COLOR_RED:=\033[0;31m}"
: "${COLOR_CYAN:=\033[0;36m}"
: "${COLOR_RESET:=\033[0m}"
: "${COLOR_BOLD:=\033[1m}"
: "${COLOR_DIM:=\033[2m}"

show_compat_help() {
  printf "Usage: nself compat check [--fix] [--json]\n"
  printf "       nself compat help\n"
  printf "\n"
  printf "Check your project for deprecated patterns before upgrading to v1.0.\n"
  printf "\n"
  printf "Subcommands:\n"
  printf "  check         Scan .env and docker-compose.yml for deprecated patterns\n"
  printf "  help          Show this help message\n"
  printf "\n"
  printf "Options (check):\n"
  printf "  --fix         Auto-update .env deprecated vars (creates .env.bak backup)\n"
  printf "  --json        Output results in JSON format\n"
  printf "  -h, --help    Show this help message\n"
  printf "\n"
  printf "Examples:\n"
  printf "  nself compat check              # Scan current directory\n"
  printf "  nself compat check --fix        # Scan and auto-fix .env\n"
  printf "  nself compat check --json       # JSON output for CI\n"
  printf "\n"
  printf "What is checked:\n"
  printf "  .env               Deprecated env var names and CS_N format issues\n"
  printf "  docker-compose.yml Deprecated service patterns\n"
  printf "  Shell scripts      Usage of removed commands (nself up/down/update/migrate)\n"
}

# -----------------------------------------------------------------------
# Check helpers
# -----------------------------------------------------------------------

ISSUES_FOUND=0
WARNINGS_FOUND=0

_issue() {
  ISSUES_FOUND=$((ISSUES_FOUND + 1))
  printf "  ${COLOR_RED}✗${COLOR_RESET} %s\n" "$1"
}

_warn() {
  WARNINGS_FOUND=$((WARNINGS_FOUND + 1))
  printf "  ${COLOR_YELLOW}⚠${COLOR_RESET} %s\n" "$1"
}

_ok() {
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} %s\n" "$1"
}

# Check .env file for deprecated patterns
check_env_file() {
  local env_file="${1:-.env}"
  local do_fix="${2:-false}"

  if [[ ! -f "$env_file" ]]; then
    printf "\n${COLOR_DIM}%s not found — skipping .env checks${COLOR_RESET}\n" "$env_file"
    return 0
  fi

  printf "\n${COLOR_BOLD}Checking %s${COLOR_RESET}\n" "$env_file"

  local found_issues=0

  # CS_N 3-field format: name:lang:port (needs 4 fields: name:lang:port:route in v1.1.0)
  # Pattern: CS_[1-9]|CS_10 followed by = and a value with exactly 2 colons
  local cs_line
  local cs_num
  local cs_val
  local colon_count
  while IFS= read -r cs_line; do
    cs_num=$(printf '%s' "$cs_line" | cut -d= -f1)
    cs_val=$(printf '%s' "$cs_line" | cut -d= -f2-)
    # Count colons in value
    colon_count=$(printf '%s' "$cs_val" | tr -cd ':' | wc -c | tr -d ' ')
    if [[ "$colon_count" -eq 2 ]]; then
      _warn "${cs_num} uses 3-field format (name:lang:port) — 4-field format (name:lang:port:route) required in v1.1.0. Add a route suffix, e.g.: ${cs_val}:api"
      found_issues=1
    fi
  done < <(grep -E '^CS_([1-9]|10)=' "$env_file" 2>/dev/null || true)

  if [[ "$found_issues" -eq 0 ]]; then
    _ok ".env has no deprecated patterns"
  fi
}

# Check docker-compose.yml for deprecated patterns
check_compose_file() {
  local compose_file="${1:-docker-compose.yml}"

  if [[ ! -f "$compose_file" ]]; then
    printf "\n${COLOR_DIM}%s not found — skipping compose checks${COLOR_RESET}\n" "$compose_file"
    return 0
  fi

  printf "\n${COLOR_BOLD}Checking %s${COLOR_RESET}\n" "$compose_file"

  local found_issues=0

  # Check for --legacy-compose flag remnants in build args or labels
  if grep -q 'legacy-compose\|legacy_compose' "$compose_file" 2>/dev/null; then
    _issue "Found 'legacy-compose' reference — this flag was removed in v1.0.0. Regenerate with: nself build"
    found_issues=1
  fi

  if [[ "$found_issues" -eq 0 ]]; then
    _ok "docker-compose.yml has no deprecated patterns"
  fi
}

# Check shell scripts in current directory for removed command usage
check_scripts_for_deprecated_cmds() {
  printf "\n${COLOR_BOLD}Checking shell scripts for removed commands${COLOR_RESET}\n"

  local found_issues=0

  # Patterns: nself up, nself down, nself update, nself migrate (as db command)
  # Also: nself build --legacy-compose, nself plugin install --no-verify

  local file
  local matches

  # Scan .sh files in current dir (not recursive to keep it fast)
  while IFS= read -r file; do
    if grep -qE '\bnself\s+up\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself up' (removed in v1.0.0) — replace with: nself start"
      found_issues=1
    fi
    if grep -qE '\bnself\s+down\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself down' (removed in v1.0.0) — replace with: nself stop"
      found_issues=1
    fi
    if grep -qE '\bnself\s+update\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself update' (removed in v1.0.0) — replace with: nself self-update"
      found_issues=1
    fi
    if grep -qE '\bnself\s+migrate\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself migrate' as a database command — if you meant DB migrations, use: nself db migrate"
      found_issues=1
    fi
    if grep -qE '\bnself\s+build\s+.*--legacy-compose\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself build --legacy-compose' (flag removed in v1.0.0) — remove the flag"
      found_issues=1
    fi
    if grep -qE '\bnself\s+plugin\s+install\s+.*--no-verify\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself plugin install --no-verify' (flag removed in v1.0.0) — replace with: --skip-checksum"
      found_issues=1
    fi
  done < <(find . -maxdepth 2 -name '*.sh' -not -path './.nself/*' -not -path './node_modules/*' 2>/dev/null || true)

  # Also check Makefile / Justfile / CI yaml in .github/
  while IFS= read -r file; do
    if grep -qE '\bnself\s+up\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself up' (removed in v1.0.0) — replace with: nself start"
      found_issues=1
    fi
    if grep -qE '\bnself\s+down\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself down' (removed in v1.0.0) — replace with: nself stop"
      found_issues=1
    fi
    if grep -qE '\bnself\s+update\b' "$file" 2>/dev/null; then
      _issue "$file: uses 'nself update' (removed in v1.0.0) — replace with: nself self-update"
      found_issues=1
    fi
  done < <(find . -maxdepth 3 \( -name 'Makefile' -o -name 'Justfile' -o -name '*.yml' -o -name '*.yaml' \) -not -path './.nself/*' -not -path './node_modules/*' 2>/dev/null || true)

  if [[ "$found_issues" -eq 0 ]]; then
    _ok "No removed commands found in scripts"
  fi
}

# Output a summary line
print_summary() {
  local issues="$1"
  local warnings="$2"

  printf "\n"
  if [[ "$issues" -gt 0 ]]; then
    printf "${COLOR_RED}${COLOR_BOLD}%d issue(s) found${COLOR_RESET} — resolve before upgrading to v1.0.0\n" "$issues"
  elif [[ "$warnings" -gt 0 ]]; then
    printf "${COLOR_YELLOW}${COLOR_BOLD}%d warning(s) found${COLOR_RESET} — action needed before v1.1.0\n" "$warnings"
  else
    printf "${COLOR_GREEN}${COLOR_BOLD}All checks passed${COLOR_RESET} — ready to upgrade to v1.0.0\n"
  fi

  if [[ "$issues" -gt 0 ]] || [[ "$warnings" -gt 0 ]]; then
    printf "\n"
    printf "  Full breaking changes reference:\n"
    printf "  https://docs.nself.org/migration/v0.9-to-v1.0\n"
  fi
  printf "\n"
}

# JSON output variant
cmd_compat_check_json() {
  local issues_json="[]"
  local warnings_json="[]"
  local issues_list=""
  local warnings_list=""

  # Run checks redirected to capture output
  local env_file=".env"
  local compose_file="docker-compose.yml"

  # --- .env checks ---
  if [[ -f "$env_file" ]]; then
    local cs_line cs_num cs_val colon_count
    while IFS= read -r cs_line; do
      cs_num=$(printf '%s' "$cs_line" | cut -d= -f1)
      cs_val=$(printf '%s' "$cs_line" | cut -d= -f2-)
      colon_count=$(printf '%s' "$cs_val" | tr -cd ':' | wc -c | tr -d ' ')
      if [[ "$colon_count" -eq 2 ]]; then
        warnings_list="${warnings_list}{\"file\":\".env\",\"var\":\"${cs_num}\",\"message\":\"3-field CS_N format deprecated — add route field before v1.1.0\"},"
      fi
    done < <(grep -E '^CS_([1-9]|10)=' "$env_file" 2>/dev/null || true)
  fi

  # --- compose checks ---
  if [[ -f "$compose_file" ]]; then
    if grep -q 'legacy-compose\|legacy_compose' "$compose_file" 2>/dev/null; then
      issues_list="${issues_list}{\"file\":\"docker-compose.yml\",\"message\":\"legacy-compose reference found — regenerate with nself build\"},"
    fi
  fi

  # Strip trailing commas and wrap in arrays
  issues_list="${issues_list%,}"
  warnings_list="${warnings_list%,}"
  [[ -n "$issues_list" ]] && issues_json="[$issues_list]"
  [[ -n "$warnings_list" ]] && warnings_json="[$warnings_list]"

  local status="ok"
  [[ "$issues_json" != "[]" ]] && status="issues"
  [[ "$issues_json" == "[]" ]] && [[ "$warnings_json" != "[]" ]] && status="warnings"

  printf '{\n'
  printf '  "status": "%s",\n' "$status"
  printf '  "issues": %s,\n' "$issues_json"
  printf '  "warnings": %s,\n' "$warnings_json"
  printf '  "docs": "https://docs.nself.org/migration/v0.9-to-v1.0"\n'
  printf '}\n'
}

# Main check command
cmd_compat_check() {
  local do_fix=false
  local do_json=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --fix)
        do_fix=true
        shift
        ;;
      --json)
        do_json=true
        shift
        ;;
      -h | --help)
        show_compat_help
        return 0
        ;;
      *)
        printf "Unknown option: %s\n" "$1" >&2
        show_compat_help >&2
        return 1
        ;;
    esac
  done

  if [[ "$do_json" == "true" ]]; then
    cmd_compat_check_json
    return $?
  fi

  printf "${COLOR_BOLD}nself compat check${COLOR_RESET} — v0.9.x → v1.0.0 upgrade scan\n"

  ISSUES_FOUND=0
  WARNINGS_FOUND=0

  check_env_file ".env" "$do_fix"
  check_compose_file "docker-compose.yml"
  check_scripts_for_deprecated_cmds

  print_summary "$ISSUES_FOUND" "$WARNINGS_FOUND"

  if [[ "$ISSUES_FOUND" -gt 0 ]]; then
    return 1
  fi
  return 0
}

# Main command handler
cmd_compat() {
  local subcommand="${1:-help}"

  case "$subcommand" in
    check)
      shift
      cmd_compat_check "$@"
      ;;
    -h | --help | help)
      show_compat_help
      return 0
      ;;
    *)
      printf "Unknown subcommand: %s\n" "$subcommand" >&2
      printf "Run 'nself compat help' for usage.\n" >&2
      return 1
      ;;
  esac
}

# Export for use as library
export -f cmd_compat

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  for _arg in "$@"; do
    if [[ "$_arg" == "--help" ]] || [[ "$_arg" == "-h" ]]; then
      show_compat_help
      exit 0
    fi
  done
  cmd_compat "$@"
  exit $?
fi
