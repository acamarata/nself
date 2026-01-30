#!/usr/bin/env bash
# redis.sh - DEPRECATED: Use 'nself service redis' instead

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show deprecation warning
printf "\033[0;33m⚠\033[0m  The 'nself redis' command is deprecated.\n"
printf "   Please use: \033[1mnself service redis\033[0m\n\n"

# Delegate to new command
exec "${SCRIPT_DIR}/service.sh" redis "$@"
