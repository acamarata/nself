#!/usr/bin/env bash
# up.sh - Start all services (simplified)

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/utils/progress.sh"
source "$SCRIPT_DIR/../lib/auto-fix/core.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Command function
cmd_up() {
    local dry_run=false
    local verbose=false
    local rebuild=false
    local no_health_check=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                dry_run=true
                shift
                ;;
            --verbose|-v)
                verbose=true
                export VERBOSE=true
                shift
                ;;
            --rebuild)
                rebuild=true
                shift
                ;;
            --no-health-check)
                no_health_check=true
                shift
                ;;
            --help|-h)
                show_up_help
                return 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_up_help
                return 1
                ;;
        esac
    done
    
    show_header "Starting Services"
    
    # Check Docker
    ensure_docker_running || return 1
    
    # Build if needed
    if [[ "$rebuild" == true ]] || [[ ! -f "docker-compose.yml" ]]; then
        log_info "Building project structure..."
        if [[ -f "$SCRIPT_DIR/build.sh" ]]; then
            source "$SCRIPT_DIR/build.sh"
            cmd_build || return 1
        fi
    fi
    
    # Validate compose file
    if ! validate_compose_file; then
        log_error "Docker Compose configuration is invalid"
        
        # Try auto-fix
        if [[ "$AUTO_FIX_ENABLED" == "true" ]]; then
            attempt_auto_fix "docker_build" "compose validation"
        fi
        return 1
    fi
    
    # Dry run - just show what would happen
    if [[ "$dry_run" == true ]]; then
        log_info "Dry run mode - showing configuration:"
        compose config
        return 0
    fi
    
    # Start services
    log_info "Starting services..."
    if ! start_services "$verbose"; then
        # Handle failure with auto-fix
        handle_startup_failure
        return 1
    fi
    
    # Health checks
    if [[ "$no_health_check" != true ]]; then
        check_services_health
    fi
    
    # Show success
    show_success_info
    
    return 0
}

# Show help
show_up_help() {
    echo "Usage: nself up [options]"
    echo
    echo "Options:"
    echo "  --dry-run          Show what would be done without starting"
    echo "  --verbose, -v      Enable verbose output"
    echo "  --rebuild          Force rebuild before starting"
    echo "  --no-health-check  Skip health checks"
    echo "  --help, -h         Show this help"
    echo
    echo "Examples:"
    echo "  nself up                    # Start all services"
    echo "  nself up --verbose          # Start with detailed output"
    echo "  nself up --rebuild          # Rebuild and start"
}

# Start services
start_services() {
    local verbose="$1"
    
    if [[ "$verbose" == true ]]; then
        # Verbose mode - show output
        compose up -d
    else
        # Normal mode - show progress
        {
            compose up -d >/dev/null 2>&1
        } &
        local pid=$!
        
        show_spinner "$pid" "Starting containers..."
        wait "$pid"
        local result=$?
        
        if [[ $result -eq 0 ]]; then
            stop_spinner "success" "Containers started"
        else
            stop_spinner "error" "Failed to start containers"
            return 1
        fi
    fi
    
    return 0
}

# Handle startup failure
handle_startup_failure() {
    log_error "Failed to start services"
    
    # Get last error from docker
    local error_log=$(compose logs --tail 50 2>&1)
    
    # Classify error
    local error_type=""
    
    if echo "$error_log" | grep -q "bind.*address already in use"; then
        error_type="port_conflict_self"
    elif echo "$error_log" | grep -q "Cannot connect to the Docker daemon"; then
        error_type="docker_not_running"
    elif echo "$error_log" | grep -q "no such file.*requirements.txt\|package.json\|go.mod"; then
        error_type="dependency_missing"
    fi
    
    # Try auto-fix if enabled
    if [[ "$AUTO_FIX_ENABLED" == "true" ]] && [[ -n "$error_type" ]]; then
        log_info "Attempting to auto-fix: $error_type"
        
        if attempt_auto_fix "$error_type" "$error_log"; then
            log_info "Retrying service startup..."
            start_services false
            return $?
        fi
    fi
    
    # Show error details
    echo
    log_error "Check the logs for details:"
    echo "  nself logs --tail 50"
    
    return 1
}

# Check services health
check_services_health() {
    log_info "Checking service health..."
    
    local services=(postgres hasura)
    local all_healthy=true
    
    for service in "${services[@]}"; do
        if wait_service_healthy "$service" 30; then
            log_success "$service is healthy"
        else
            log_warning "$service may not be fully healthy"
            all_healthy=false
        fi
    done
    
    if [[ "$all_healthy" == true ]]; then
        log_success "All core services are healthy"
    else
        log_warning "Some services may need more time to start"
    fi
}

# Show success information
show_success_info() {
    echo
    draw_box "Services started successfully!" "success"
    
    # Show running services
    echo
    log_info "Running services:"
    compose ps --services --filter "status=running" | while read -r service; do
        echo "  ${ICON_SUCCESS} $service"
    done
    
    # Show URLs if available
    if [[ -n "${BASE_DOMAIN:-}" ]]; then
        echo
        show_section "Service URLs"
        echo "  ${ICON_ARROW} GraphQL:    ${COLOR_CYAN}http://localhost:8080${COLOR_RESET}"
        echo "  ${ICON_ARROW} Storage:    ${COLOR_CYAN}http://localhost:9000${COLOR_RESET}"
        
        if [[ "$BASE_DOMAIN" != "localhost" ]]; then
            echo
            echo "  With domain configuration:"
            echo "  ${ICON_ARROW} GraphQL:    ${COLOR_CYAN}https://api.${BASE_DOMAIN}${COLOR_RESET}"
            echo "  ${ICON_ARROW} Storage:    ${COLOR_CYAN}https://storage.${BASE_DOMAIN}${COLOR_RESET}"
        fi
    fi
    
    echo
    log_info "View logs: nself logs"
    log_info "Check status: nself status"
}

# Export main function
export -f cmd_up

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    pre_command "up" || exit $?
    cmd_up "$@"
    exit_code=$?
    post_command "up" $exit_code
    exit $exit_code
fi