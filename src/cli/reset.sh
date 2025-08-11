#!/usr/bin/env bash
# reset.sh - Reset project to clean state

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"

# Command function
cmd_reset() {
    show_header "NSELF PROJECT RESET"
    
    log_warning "This will:"
    echo "  • Stop and remove all containers"
    echo "  • Delete all Docker volumes"
    echo "  • Remove all generated files"
    echo "  • Backup env files with .old suffix"
    echo ""
    
    read -p "Are you sure you want to reset everything? [y/N]: " -n 1 -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Reset cancelled"
        return 1
    fi
    
    
    echo ""
    log_info "Stopping services..."
    
    # Stop all containers
    if [[ -f "docker-compose.yml" ]]; then
        docker compose down -v 2>/dev/null || true
        log_success "Stopped and removed containers"
    fi
    
    # Remove all volumes for this project
    if [[ -f ".env.local" ]]; then
        set -a
        source .env.local 2>/dev/null || true
        set +a
        local project="${PROJECT_NAME:-myproject}"
        
        log_info "Removing volumes for project: $project"
        docker volume ls -q | grep "^${project}_" | xargs -r docker volume rm 2>/dev/null || true
        log_success "Removed Docker volumes"
    fi
    
    # Backup environment files
    log_info "Backing up environment files..."
    [[ -f ".env" ]] && mv .env .env.old && log_success "Backed up .env → .env.old"
    [[ -f ".env.local" ]] && mv .env.local .env.local.old && log_success "Backed up .env.local → .env.local.old"
    [[ -f ".env.dev" ]] && mv .env.dev .env.dev.old && log_success "Backed up .env.dev → .env.dev.old"
    [[ -f ".env.prod" ]] && mv .env.prod .env.prod.old && log_success "Backed up .env.prod → .env.prod.old"
    
    # Remove generated files and directories
    log_info "Removing generated files..."
    local items_to_remove=(
        "docker-compose.yml"
        "docker-compose.yml.backup"
        ".env.example"
        "nginx"
        "postgres"
        "hasura"
        "functions"
        "services"
        "config-server"
        "nestjs-run"
        "schema.dbml"
        "seeds"
    )
    
    for item in "${items_to_remove[@]}"; do
        if [[ -e "$item" ]]; then
            rm -rf "$item"
            log_success "Removed $item"
        fi
    done
    
    # Clean up Docker system (optional)
    log_info "Cleaning Docker system..."
    docker system prune -f 2>/dev/null || true
    
    echo ""
    log_success "Project reset complete!"
    echo ""
    echo "📝 Your old configuration is saved with .old suffix"
    echo ""
    echo "🚀 To start fresh:"
    echo "   nself init    (creates new env files)"
    echo "   nself build   (generates project)"
    echo "   nself up      (starts services)"
    echo ""
    echo "♻️  To restore previous configuration:"
    echo "   mv .env.local.old .env.local"
    echo "   nself build"
    echo "   nself up"
    
    return 0
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
