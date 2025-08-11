#!/usr/bin/env bash
# down.sh - Stop all services

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Command function
cmd_down() {
    local remove_volumes=false
    local remove_images=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --volumes|-v)
                remove_volumes=true
                shift
                ;;
            --rmi|--remove-images)
                remove_images=true
                shift
                ;;
            --help|-h)
                show_down_help
                return 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_down_help
                return 1
                ;;
        esac
    done
    
    show_header "Stopping Services"
    
    # Check Docker
    ensure_docker_running || return 1
    
    # Stop services
    log_info "Stopping all services..."
    
    if [[ "$remove_volumes" == true ]]; then
        log_warning "Removing volumes (data will be lost)"
        compose down -v
    elif [[ "$remove_images" == true ]]; then
        log_info "Removing images"
        compose down --rmi all
    else
        compose down
    fi
    
    if [[ $? -eq 0 ]]; then
        log_success "All services stopped"
        
        # Clean up stopped containers
        cleanup_stopped_containers
        
        return 0
    else
        log_error "Failed to stop services"
        return 1
    fi
}

# Show help
show_down_help() {
    echo "Usage: nself down [options]"
    echo
    echo "Options:"
    echo "  --volumes, -v      Remove volumes (WARNING: deletes data)"
    echo "  --rmi              Remove images"
    echo "  --help, -h         Show this help"
    echo
    echo "Examples:"
    echo "  nself down              # Stop services, keep data"
    echo "  nself down --volumes    # Stop and remove all data"
}

# Export for use as library
export -f cmd_down

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    pre_command "down" || exit $?
    cmd_down "$@"
    exit_code=$?
    post_command "down" $exit_code
    exit $exit_code
fi