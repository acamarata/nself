#!/usr/bin/env bash
# k8s.sh - DEPRECATED: Use 'nself infra k8s' instead

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show deprecation warning
printf "\033[0;33m⚠\033[0m  The 'nself k8s' command is deprecated.\n"
printf "   Please use: \033[1mnself infra k8s\033[0m\n\n"

# Delegate to new command
exec "${SCRIPT_DIR}/infra.sh" k8s "$@"
