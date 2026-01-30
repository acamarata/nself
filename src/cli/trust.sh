#!/usr/bin/env bash
# trust.sh - DEPRECATED - Wrapper for 'nself auth ssl trust'
# This command has been consolidated into 'nself auth ssl trust'
# This wrapper provides backward compatibility

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show deprecation warning
printf "\033[0;33m⚠\033[0m  WARNING: 'nself trust' is deprecated. Use 'nself auth ssl trust' instead.\n" >&2
printf "   This compatibility wrapper will be removed in v1.0.0\n\n" >&2

# Delegate to new auth command
exec bash "$SCRIPT_DIR/auth.sh" ssl trust "$@"
