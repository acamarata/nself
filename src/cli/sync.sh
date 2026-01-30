#!/usr/bin/env bash
# sync.sh - DEPRECATED: Use 'nself deploy sync' instead

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show deprecation warning
printf "\033[0;33m⚠\033[0m  The 'nself sync' command is deprecated.\n"
printf "   Please use: \033[1mnself deploy sync\033[0m\n\n"

# Delegate to new command
exec "${SCRIPT_DIR}/deploy.sh" sync "$@"
