#!/usr/bin/env bash
# oauth.sh - DEPRECATED - Wrapper for 'nself auth oauth'
# This command has been consolidated into 'nself auth oauth'
# This wrapper provides backward compatibility

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Show deprecation warning
printf "\033[0;33m⚠\033[0m  WARNING: 'nself oauth' is deprecated. Use 'nself auth oauth' instead.\n" >&2
printf "   This compatibility wrapper will be removed in v1.0.0\n\n" >&2

# Delegate to new auth command
exec bash "$SCRIPT_DIR/auth.sh" oauth "$@"
