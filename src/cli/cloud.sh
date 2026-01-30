#!/usr/bin/env bash
# cloud.sh - DEPRECATED: Use 'nself provider' instead
# This is a compatibility wrapper that will be removed in v1.0.0
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source shared utilities
source "${SCRIPT_DIR}/../lib/utils/display.sh" 2>/dev/null || true

# Fallback logging if display.sh failed to load
if ! declare -f log_warning >/dev/null 2>&1; then
  log_warning() { printf "\033[0;33m!\033[0m %s\n" "$1"; }
fi
if ! declare -f log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m✗\033[0m %s\n" "$1" >&2; }
fi

# Show deprecation warning
show_deprecation_warning() {
  log_warning "DEPRECATION: 'nself cloud' is deprecated and will be removed in v1.0.0"
  printf "             Please use 'nself provider' instead.\n"
  printf "             https://docs.nself.org/migration/cloud-to-provider\n"
  echo ""
}

show_cloud_help() {
  show_deprecation_warning
  cat << 'EOF'
nself cloud - DEPRECATED (use 'nself provider' instead)

MIGRATION GUIDE:
  Old Command                           →  New Command
  ────────────────────────────────────────────────────────────────────
  nself cloud provider list             →  nself provider info list
  nself cloud provider init <name>      →  nself provider info init <name>
  nself cloud provider validate         →  nself provider info validate
  nself cloud provider info <name>      →  nself provider info show <name>

  nself cloud server create             →  nself provider server create
  nself cloud server list               →  nself provider server list
  nself cloud server ssh <name>         →  nself provider server ssh <name>

  nself cloud cost estimate             →  nself provider cost estimate
  nself cloud deploy quick <server>     →  nself provider deploy quick <server>

For full documentation, run: nself provider --help
EOF
}

# Main entry point - delegates to provider.sh
main() {
  # Check if user is asking for help
  if [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "help" ]] || [[ $# -eq 0 ]]; then
    show_cloud_help
    return 0
  fi

  # Show deprecation warning for actual command usage
  show_deprecation_warning

  # Delegate all commands to provider.sh
  bash "${SCRIPT_DIR}/provider.sh" "$@"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
