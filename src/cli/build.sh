#!/usr/bin/env bash

# build.sh - Modular build system with robust error checking and auto-fixing
set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source environment utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Main build command function
cmd_build() {
    # Pre-command hook
    pre_command "build" || exit $?
    
    # Use the modular orchestrator
    if [[ -f "$SCRIPT_DIR/build/orchestrator.sh" ]]; then
        log_info "Using modular build system..."
        bash "$SCRIPT_DIR/build/orchestrator.sh" "$@"
        local exit_code=$?
    else
        log_error "Modular build system not found, falling back to legacy build"
        # Could fallback to the old build.sh logic here if needed
        exit 1
    fi
    
    # Post-command hook
    post_command "build" $exit_code
    exit $exit_code
}

# Export for use as library
export -f cmd_build

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_build "$@"
fi