#!/usr/bin/env bash
# reset.sh - Reset project to clean state

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/utils/progress.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Command function
cmd_reset() {
    local force=false
    local keep_env=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force|-f)
                force=true
                shift
                ;;
            --keep-env)
                keep_env=true
                shift
                ;;
            --help|-h)
                show_reset_help
                return 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_reset_help
                return 1
                ;;
        esac
    done
    
    show_header "Reset Project"
    
    log_warning "This will remove all containers, volumes, and generated files!"
    
    if [[ "$force" != true ]]; then
        if ! confirm_with_timeout "Are you sure you want to reset?" 10 "n"; then
            log_info "Reset cancelled"
            return 0
        fi
    fi
    
    # Stop and remove all containers and volumes
    log_info "Stopping all services..."
    source "$SCRIPT_DIR/down.sh"
    cmd_down --volumes
    
    # Remove generated files
    log_info "Removing generated files..."
    rm -f docker-compose.yml
    rm -f docker compose.override.yml
    rm -rf backend/docker-compose.yml
    
    # Clean Docker resources
    log_info "Cleaning Docker resources..."
    docker system prune -f >/dev/null 2>&1
    
    # Remove data directories
    log_info "Removing data directories..."
    rm -rf data/ volumes/ postgres-data/ minio-data/ redis-data/
    rm -rf hasura/migrations/* hasura/metadata/*
    
    # Optionally keep environment file
    if [[ "$keep_env" != true ]]; then
        if [[ -f ".env.local" ]]; then
            mv .env.local .env.local.backup
            log_info "Environment file backed up to .env.local.backup"
        fi
    else
        log_info "Keeping environment file"
    fi
    
    # Clean logs
    rm -rf logs/*.log
    
    log_success "Project reset complete"
    echo
    echo "Next steps:"
    if [[ "$keep_env" != true ]]; then
        echo "  1. Run: nself init"
    fi
    echo "  2. Run: nself build"
    echo "  3. Run: nself up"
    
    return 0
}

# Show help
show_reset_help() {
    echo "Usage: nself reset [options]"
    echo
    echo "Reset the project to a clean state"
    echo
    echo "Options:"
    echo "  --force, -f    Skip confirmation prompt"
    echo "  --keep-env     Keep .env.local file"
    echo "  --help, -h     Show this help"
    echo
    echo "WARNING: This will remove all data!"
}

# Export for use as library
export -f cmd_reset

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    pre_command "reset" || exit $?
    cmd_reset "$@"
    exit_code=$?
    post_command "reset" $exit_code
    exit $exit_code
fi
