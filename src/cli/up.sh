#!/usr/bin/env bash
# up.sh - Alias for start.sh (backward compatibility)

# Get script directory
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"

# Source and call start command
source "$SCRIPT_DIR/start.sh"

# Call the start function with all arguments
cmd_start "$@"
