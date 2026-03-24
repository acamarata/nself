#!/usr/bin/env bash
# email.sh - DEPRECATED: Use 'nself service email' instead

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

set -euo pipefail

# Intercept --help before delegating
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
  printf "DEPRECATION NOTICE\n\n"
  printf "  'nself email' is deprecated and will be removed in v1.0.0.\n"
  printf "  Please use: nself service email\n\n"
  exit 0
fi

# Show deprecation warning and delegate
printf "\033[0;33m⚠\033[0m  'nself email' is deprecated. Use: \033[1mnself service email\033[0m\n\n" >&2
exec "${SCRIPT_DIR}/service.sh" email "$@"
