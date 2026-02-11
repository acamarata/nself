#!/usr/bin/env bash
# harden.sh - Security hardening automation
set -euo pipefail

CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$CLI_SCRIPT_DIR/../lib/utils/cli-output.sh" 2>/dev/null || true

# Source hardening library
if [[ -f "$CLI_SCRIPT_DIR/../lib/security/hardening.sh" ]]; then
  source "$CLI_SCRIPT_DIR/../lib/security/hardening.sh"
fi

cmd_harden() {
  local subcommand="${1:-}"

  if [[ -z "$subcommand" ]]; then
    harden_interactive
    return $?
  fi

  shift || true

  case "$subcommand" in
    all)
      harden_all
      ;;
    secrets)
      harden_secrets
      ;;
    cors)
      harden_cors
      ;;
    help | --help | -h)
      harden_usage
      ;;
    *)
      cli_error "Unknown subcommand: $subcommand"
      harden_usage
      exit 1
      ;;
  esac
}

harden_usage() {
  cat <<EOF
Usage: nself harden [command]

Security hardening automation

COMMANDS:
  all              Apply all hardening fixes
  secrets          Rotate weak secrets
  cors             Fix CORS configuration

  help             Show this help

EXAMPLES:
  nself harden                 # Interactive wizard
  nself harden all             # Apply all fixes
  nself harden secrets         # Rotate weak secrets only

For more information: https://docs.nself.org/security/hardening
EOF
}

# If executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_harden "$@"
fi

export -f cmd_harden
