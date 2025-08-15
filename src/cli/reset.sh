#!/usr/bin/env bash
# reset.sh - Reset project to clean state

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/header.sh"

# Command function
cmd_reset() {
    local force_reset=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force|-f)
                force_reset=true
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
    
    # Show standardized header
    show_command_header "nself reset" "Reset project to clean state"
    
    echo -e "${COLOR_YELLOW}⚠${COLOR_RESET}  This will:"
    echo "  • Stop and remove all containers"
    echo "  • Delete all Docker volumes"
    echo "  • Remove all generated files"
    echo "  • Backup env files with .old suffix"
    echo
    
    if [[ "$force_reset" != "true" ]]; then
        read -p "Are you sure you want to reset everything? [y/N]: " -n 1 -r
        echo
        
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Reset cancelled"
            return 1
        fi
    fi
    
    echo
    
    # Get project name first
    local project="${PROJECT_NAME:-myproject}"
    if [[ -f ".env.local" ]]; then
        set -a
        source .env.local 2>/dev/null || true
        set +a
        project="${PROJECT_NAME:-myproject}"
    fi
    
    # Stopping services
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Stopping services..."
    
    # First try docker-compose down if file exists
    if [[ -f "docker-compose.yml" ]]; then
        docker compose down -v >/dev/null 2>&1 || true
    fi
    
    # Then forcefully stop and remove ALL project containers
    local container_count=$(docker ps -aq --filter "name=${project}" | wc -l | tr -d ' ')
    if [[ $container_count -gt 0 ]]; then
        # Stop all project containers
        docker ps -q --filter "name=${project}" | xargs -r docker stop >/dev/null 2>&1 || true
        # Remove all project containers
        docker ps -aq --filter "name=${project}" | xargs -r docker rm -f >/dev/null 2>&1 || true
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Stopped and removed $container_count containers         \n"
    else
        printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No containers to stop                           \n"
    fi
    
    # Remove all volumes for this project
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Removing Docker volumes..."
    local volume_count=$(docker volume ls -q | grep "^${project}_" | wc -l | tr -d ' ')
    if [[ $volume_count -gt 0 ]]; then
        docker volume ls -q | grep "^${project}_" | xargs -r docker volume rm -f >/dev/null 2>&1 || true
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed $volume_count Docker volumes                     \n"
    else
        printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No volumes to remove                            \n"
    fi
    
    # Also remove the Docker network
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Removing Docker network..."
    if docker network ls | grep -q "${project}_network"; then
        docker network rm "${project}_network" >/dev/null 2>&1 || true
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed Docker network                          \n"
    else
        printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No network to remove                            \n"
    fi
    
    # Backup environment files and schema
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Backing up configuration files..."
    local backed_up=0
    [[ -f ".env" ]] && mv .env .env.old && ((backed_up++))
    [[ -f ".env.local" ]] && mv .env.local .env.local.old && ((backed_up++))
    [[ -f ".env.dev" ]] && mv .env.dev .env.dev.old && ((backed_up++))
    [[ -f ".env.prod" ]] && mv .env.prod .env.prod.old && ((backed_up++))
    [[ -f "schema.dbml" ]] && mv schema.dbml schema.dbml.old && ((backed_up++))
    
    if [[ $backed_up -gt 0 ]]; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Backed up $backed_up configuration files              \n"
    else
        printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No configuration files to backup                \n"
    fi
    
    # Remove ALL generated files and directories
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Removing generated files..."
    local items_to_remove=(
        # Docker files
        "docker-compose.yml"
        "docker-compose.yml.backup"
        "docker-compose.override.yml"
        ".dockerignore"
        
        # Environment files (except .old backups)
        ".env.example"
        ".env"
        ".env.dev"
        ".env.prod"
        ".env.prod-template"
        ".env.prod-secrets"
        
        # Service directories
        "nginx"
        "nginx.backup"
        "postgres"
        "hasura"
        "functions"
        "services"
        "config-server"
        "nestjs-run"
        "backend"
        "storage"
        "auth"
        "minio"
        "dashboard"
        
        # Certificates and binaries
        "certs"
        "bin"
        
        # Data directories
        "data"
        "volumes"
        ".volumes"
        "postgres-data"
        "minio-data"
        "redis-data"
        
        # Database files  
        "db"
        "init.sql"
        "schema.dbml"
        "seeds"
        "migrations"
        "metadata"
        "postgres/init"
        
        # Log files
        "logs"
        
        # Temporary files and scripts
        ".needs-rebuild"
        ".last-build-hash"
        ".port-overrides"
        "fix-healthchecks.sh"
        "fix-*.sh"
        
        # Other generated files
        "node_modules"
        "package-lock.json"
        "yarn.lock"
        "go.mod"
        "go.sum"
        "requirements.txt"
        "Pipfile"
        "Pipfile.lock"
    )
    
    local removed_count=0
    for item in "${items_to_remove[@]}"; do
        if [[ -e "$item" ]]; then
            rm -rf "$item"
            ((removed_count++))
        fi
    done
    
    # Remove files matching patterns
    for pattern in *.log *.pid *.lock .DS_Store; do
        for file in $pattern; do
            if [[ -f "$file" ]]; then
                rm -f "$file"
                ((removed_count++))
            fi
        done
    done
    
    # Remove any unity-* or project-prefixed directories (leftover from previous runs)
    for dir in ${project}-* unity-*; do
        if [[ -d "$dir" ]]; then
            rm -rf "$dir"
            ((removed_count++))
        fi
    done
    
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed $removed_count files and directories           \n"
    
    # Clean up Docker system (optional)
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning Docker system..."
    if docker system prune -f >/dev/null 2>&1; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Docker system cleaned                           \n"
    else
        printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} Docker cleanup skipped                          \n"
    fi
    
    echo
    log_success "Project reset complete!"
    echo
    
    echo -e "${COLOR_CYAN}➞ Next Steps${COLOR_RESET}"
    echo
    
    echo -e "${COLOR_BLUE}Start Fresh:${COLOR_RESET}"
    echo -e "  ${COLOR_BLUE}nself init${COLOR_RESET}     ${COLOR_DIM}# Create new configuration${COLOR_RESET}"
    echo -e "  ${COLOR_BLUE}nself build${COLOR_RESET}    ${COLOR_DIM}# Generate infrastructure${COLOR_RESET}"
    echo -e "  ${COLOR_BLUE}nself start${COLOR_RESET}       ${COLOR_DIM}# Start services${COLOR_RESET}"
    echo
    
    echo -e "${COLOR_BLUE}Restore Previous:${COLOR_RESET}"
    echo -e "  ${COLOR_BLUE}mv .env.local.old .env.local${COLOR_RESET}"
    echo -e "  ${COLOR_BLUE}nself build${COLOR_RESET}"
    echo -e "  ${COLOR_BLUE}nself start${COLOR_RESET}"
    echo
    
    log_info "Configuration backed up with .old suffix"
    
    return 0
}

# Show help
show_reset_help() {
    echo "Usage: nself reset [options]"
    echo
    echo "Reset the project to a clean state"
    echo
    echo "Options:"
    echo "  -f, --force    Skip confirmation prompt"
    echo "  -h, --help     Show this help message"
    echo
    echo "Actions:"
    echo "  • Stop and remove all containers"
    echo "  • Delete all Docker volumes"
    echo "  • Remove all generated files"
    echo "  • Backup env files with .old suffix"
    echo
    echo "Examples:"
    echo "  nself reset              # Interactive reset"
    echo "  nself reset --force      # Skip confirmation"
    echo
    echo "WARNING: This will remove all data and cannot be undone!"
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
