#!/usr/bin/env bash
# down.sh - Alias for stop.sh (backward compatibility)

# Get script directory
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"

# Source and call stop command
source "$SCRIPT_DIR/stop.sh"

# Call the stop function with all arguments
cmd_stop "$@"