#!/usr/bin/env bash
#
# nself org - Organization management (DEPRECATED)
#
# WARNING: This command is deprecated. Use 'nself tenant org' instead.
#
# This file now acts as a wrapper that redirects to 'nself tenant org'
# with a deprecation warning.
#

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source cli-output for formatting
source "$SCRIPT_DIR/../lib/utils/cli-output.sh"

# Show deprecation warning
show_deprecation_warning() {
  printf "\n"
  cli_warning "DEPRECATION NOTICE"
  printf "\n"
  printf "  The 'nself org' command is deprecated.\n"
  printf "  Please use 'nself tenant org' instead.\n"
  printf "\n"
  cli_info "Redirecting to 'nself tenant org'..."
  printf "\n"
  sleep 1
}

# Main function - redirect to tenant org
main() {
  show_deprecation_warning

  # Delegate to tenant org command (tenant.sh is in same directory)
  exec bash "$(dirname "${BASH_SOURCE[0]}")/tenant.sh" org "$@"
}

# Run main
main "$@"
