#!/usr/bin/env bash
# clean.sh - Clean up Docker resources for the project

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/config/smart-defaults.sh"

# Load environment with smart defaults
load_env_with_defaults >/dev/null 2>&1

# Command function
cmd_clean() {
    local all=false
    local images=false
    local volumes=false
    local networks=false
    local containers=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --all|-a)
                all=true
                shift
                ;;
            --images|-i)
                images=true
                shift
                ;;
            --volumes|-v)
                volumes=true
                shift
                ;;
            --networks|-n)
                networks=true
                shift
                ;;
            --containers|-c)
                containers=true
                shift
                ;;
            --help|-h)
                show_help
                return 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                return 1
                ;;
        esac
    done
    
    # If no specific options, default to cleaning images
    if [[ "$all" == "false" ]] && [[ "$images" == "false" ]] && [[ "$volumes" == "false" ]] && [[ "$networks" == "false" ]] && [[ "$containers" == "false" ]]; then
        images=true
    fi
    
    # Show header
    echo
    echo -e "${COLOR_BLUE}╔══════════════════════════════════════════════════════════╗${COLOR_RESET}"
    echo -e "${COLOR_BLUE}║${COLOR_RESET}  ${BOLD}nself clean${RESET}                                             ${COLOR_BLUE}║${COLOR_RESET}"
    echo -e "${COLOR_BLUE}║${COLOR_RESET}  ${COLOR_DIM}Cleaning up Docker resources${COLOR_RESET}                           ${COLOR_BLUE}║${COLOR_RESET}"
    echo -e "${COLOR_BLUE}╚══════════════════════════════════════════════════════════╝${COLOR_RESET}"
    echo
    
    local project_name="${PROJECT_NAME:-myproject}"
    
    # Clean containers
    if [[ "$all" == "true" ]] || [[ "$containers" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning stopped containers..."
        
        # Get stopped containers for this project
        local stopped=$(docker ps -a --filter "name=${project_name}" --filter "status=exited" -q 2>/dev/null)
        
        if [[ -n "$stopped" ]]; then
            local count=$(echo "$stopped" | wc -l | tr -d ' ')
            docker rm $stopped >/dev/null 2>&1
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed $count stopped containers                    \n"
        else
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} No stopped containers to clean                       \n"
        fi
    fi
    
    # Clean images
    if [[ "$all" == "true" ]] || [[ "$images" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning project images..."
        
        # Get all images for this project (including dangling)
        # Handle both naming patterns: project_name/* and project_name-*
        # Also match the actual pattern we see: project_name-project_name-*
        local project_images=$(docker images | grep -E "^${project_name}" | awk '{print $3}' | sort -u)
        local dangling_images=$(docker images -f "dangling=true" -q 2>/dev/null | sort -u)
        
        local all_images=$(echo -e "$project_images\n$dangling_images" | sort -u | grep -v '^$')
        
        if [[ -n "$all_images" ]]; then
            local count=$(echo "$all_images" | wc -l | tr -d ' ')
            
            # Remove images (force to handle any that might be in use)
            echo "$all_images" | xargs docker rmi -f >/dev/null 2>&1
            
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed $count project/dangling images                \n"
        else
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} No images to clean                                   \n"
        fi
        
        # Also prune build cache
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning build cache..."
        docker builder prune -f >/dev/null 2>&1
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Cleaned build cache                                  \n"
    fi
    
    # Clean volumes
    if [[ "$all" == "true" ]] || [[ "$volumes" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning project volumes..."
        
        # Get volumes for this project
        local project_volumes=$(docker volume ls --filter "name=${project_name}" -q 2>/dev/null)
        
        if [[ -n "$project_volumes" ]]; then
            local count=$(echo "$project_volumes" | wc -l | tr -d ' ')
            
            echo
            log_warning "This will delete all data in project volumes!"
            read -p "Are you sure? (y/N): " confirm
            
            if [[ "$confirm" == "y" ]] || [[ "$confirm" == "Y" ]]; then
                echo "$project_volumes" | xargs docker volume rm >/dev/null 2>&1
                printf "${COLOR_GREEN}✓${COLOR_RESET} Removed $count project volumes                        \n"
            else
                printf "${COLOR_YELLOW}✱${COLOR_RESET} Skipped volume cleanup                               \n"
            fi
        else
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} No volumes to clean                                  \n"
        fi
    fi
    
    # Clean networks
    if [[ "$all" == "true" ]] || [[ "$networks" == "true" ]]; then
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning project networks..."
        
        # Get networks for this project
        local project_networks=$(docker network ls --filter "name=${project_name}" -q 2>/dev/null)
        
        if [[ -n "$project_networks" ]]; then
            local count=$(echo "$project_networks" | wc -l | tr -d ' ')
            echo "$project_networks" | xargs docker network rm >/dev/null 2>&1
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed $count project networks                       \n"
        else
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} No networks to clean                                 \n"
        fi
    fi
    
    # System-wide prune (if --all)
    if [[ "$all" == "true" ]]; then
        echo
        log_info "Running system-wide Docker cleanup..."
        docker system prune -f >/dev/null 2>&1
        log_success "System cleanup complete"
    fi
    
    echo
    log_success "Cleanup complete!"
    
    # Show disk space reclaimed
    echo
    printf "${COLOR_CYAN}➞ Disk Space${COLOR_RESET}\n"
    docker system df --format "table {{.Type}}\t{{.Size}}\t{{.Reclaimable}}" 2>/dev/null | head -5
}

# Show help
show_help() {
    echo "Usage: nself clean [OPTIONS]"
    echo ""
    echo "Clean up Docker resources for the project"
    echo ""
    echo "Options:"
    echo "  -i, --images       Clean project images and build cache (default)"
    echo "  -c, --containers   Remove stopped containers"
    echo "  -v, --volumes      Remove project volumes (WARNING: deletes data)"
    echo "  -n, --networks     Remove project networks"
    echo "  -a, --all          Clean everything (containers, images, volumes, networks)"
    echo "  -h, --help         Show this help message"
    echo ""
    echo "Examples:"
    echo "  nself clean           # Clean images and build cache"
    echo "  nself clean -c        # Remove stopped containers"
    echo "  nself clean -a        # Clean everything"
    echo "  nself clean -i -c     # Clean images and containers"
}

# Export for use as library
export -f cmd_clean

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_clean "$@"
    exit $?
fi