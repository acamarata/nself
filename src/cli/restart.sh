#!/usr/bin/env bash
# restart.sh - Restart all services

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Command function
cmd_restart() {
    local options="$@"
    
    show_header "Restarting Services"
    
    # Stop services
    log_info "Stopping services..."
    source "$SCRIPT_DIR/down.sh"
    cmd_down || return 1
    
    # Small delay
    sleep 2
    
    # Start services
    log_info "Starting services..."
    source "$SCRIPT_DIR/up.sh"
    cmd_up $options || return 1
    
    log_success "Services restarted successfully"
    return 0
}

# Export for use as library
export -f cmd_restart

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    pre_command "restart" || exit $?
    cmd_restart "$@"
    exit_code=$?
    post_command "restart" $exit_code
    exit $exit_code
fi
