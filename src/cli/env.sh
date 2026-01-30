#!/usr/bin/env bash
# env.sh - DEPRECATED: Redirects to nself config env
# This file maintained for backward compatibility only
# Use: nself config env instead

set -euo pipefail

# Get script directory
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source the consolidated config command
source "$CLI_SCRIPT_DIR/config.sh"

# Show deprecation warning
if [[ -t 1 ]]; then
  printf "\033[0;33m⚠\033[0m \033[2mDeprecation Notice:\033[0m 'nself env' is deprecated\n" >&2
  printf "  \033[2mUse:\033[0m \033[0;36mnself config env\033[0m instead\n\n" >&2
fi

# Redirect to config env
cmd_config "env" "$@"
