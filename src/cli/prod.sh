#!/usr/bin/env bash
# prod.sh - DEPRECATED: Use 'nself deploy production' instead

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show deprecation warning
printf "\033[0;33m⚠\033[0m  The 'nself prod' command is deprecated.\n"
printf "   Please use: \033[1mnself deploy production\033[0m\n\n"

# Delegate to new command
exec "${SCRIPT_DIR}/deploy.sh" production "$@"
