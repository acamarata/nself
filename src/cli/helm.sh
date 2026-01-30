#!/usr/bin/env bash
# helm.sh - DEPRECATED: Use 'nself infra helm' instead

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show deprecation warning
printf "\033[0;33m⚠\033[0m  The 'nself helm' command is deprecated.\n"
printf "   Please use: \033[1mnself infra helm\033[0m\n\n"

# Delegate to new command
exec "${SCRIPT_DIR}/infra.sh" helm "$@"
