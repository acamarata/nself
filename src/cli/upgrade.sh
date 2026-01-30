#!/usr/bin/env bash
# upgrade.sh - DEPRECATED: Use 'nself deploy upgrade' instead

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show deprecation warning
printf "\033[0;33m⚠\033[0m  The 'nself upgrade' command is deprecated.\n"
printf "   Please use: \033[1mnself deploy upgrade\033[0m\n\n"

# Delegate to new command
exec "${SCRIPT_DIR}/deploy.sh" upgrade "$@"
