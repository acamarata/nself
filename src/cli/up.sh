#!/usr/bin/env bash
# up.sh - Start services with streamlined error handling

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/utils/progress.sh"
source "$SCRIPT_DIR/../lib/errors/base.sh"
source "$SCRIPT_DIR/../lib/errors/quick-check.sh"
source "$SCRIPT_DIR/../lib/errors/handlers/ports.sh"
source "$SCRIPT_DIR/../lib/errors/handlers/build.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Command function
cmd_up() {
    local verbose=false
    local skip_checks=false
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose|-v)
                verbose=true
                shift
                ;;
            --skip-checks)
                skip_checks=true
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
    
    show_header "Starting Services"
    
    # Quick essential checks only (fast)
    if [[ "$skip_checks" != "true" ]]; then
        log_info "Running quick checks..."
        
        if ! run_essential_checks; then
            log_error "Essential checks failed"
            return 1
        fi
    fi
    
    # Check if we need to build
    if [[ ! -f "docker-compose.yml" ]]; then
        log_info "No docker-compose.yml found"
        log_info "Running 'nself build' first..."
        
        if ! bash "$SCRIPT_DIR/build.sh"; then
            log_error "Build failed"
            return 1
        fi
    fi
    
    # Start services
    log_info "Starting containers..."
    
    local output
    local result
    
    if [[ "$verbose" == "true" ]]; then
        compose up -d --build
        result=$?
    else
        output=$(compose up -d --build 2>&1)
        result=$?
        
        # Show progress for long operations
        if echo "$output" | grep -q "Building"; then
            echo "$output" | grep "Building\|=>"
        fi
    fi
    
    if [[ $result -eq 0 ]]; then
        log_success "All containers started successfully!"
        show_service_urls
    else
        log_error "Failed to start services"
        
        # Analyze the specific error
        if [[ "$verbose" != "true" ]]; then
            analyze_startup_error "$output"
        fi
        
        return 1
    fi
}

# Analyze startup errors and offer solutions
analyze_startup_error() {
    local output="$1"
    
    # Port conflicts
    if echo "$output" | grep -q "port is already allocated"; then
        log_error "Port conflict detected"
        
        local port=$(echo "$output" | grep -oE "port [0-9]+" | grep -oE "[0-9]+" | head -1)
        local process=$(get_port_process $port)
        
        log_info "Port $port is used by: $process"
        offer_port_solutions
        
    # Build errors
    elif echo "$output" | grep -q "failed to solve\|exit code:"; then
        analyze_build_failure "$output"
        # If the build error was go module-related and we attempted a fix,
        # offer to rebuild without cache to avoid stale layers.
        if echo "$output" | grep -q "missing go.sum entry"; then
            echo ""
            read -p "Rebuild images without cache now? [Y/n]: " -n 1 -r
            echo ""
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                log_info "Rebuilding images without cache..."
                compose build --no-cache || {
                    log_error "No-cache rebuild failed"
                    return 1
                }
                log_info "Starting containers..."
                compose up -d || {
                    log_error "Startup failed after rebuild"
                    return 1
                }
                log_success "Containers started successfully after rebuild"
                return 0
            fi
        fi
        
    # Docker not running
    elif echo "$output" | grep -q "Cannot connect to the Docker daemon"; then
        log_error "Docker is not running"
        quick_docker_check
        
    # Network issues
    elif echo "$output" | grep -q "network.*not found"; then
        log_error "Docker network issue"
        log_info "Try: docker network prune && nself up"
        
    # Generic error - show output
    else
        echo "$output" | tail -20
        echo ""
        log_info "For more details, run: nself up --verbose"
    fi
}

# Show service URLs
show_service_urls() {
    if [[ ! -f ".env.local" ]]; then
        return
    fi
    
    source .env.local
    
    echo ""
    log_info "Services available at:"
    
    # Check which services are actually running
    local running_services=$(docker compose ps --services --filter "status=running" 2>/dev/null)
    
    if echo "$running_services" | grep -q "hasura"; then
        log_info "  GraphQL:    https://gql.$BASE_DOMAIN"
    fi
    
    if echo "$running_services" | grep -q "auth"; then
        log_info "  Auth:       https://auth.$BASE_DOMAIN"
    fi
    
    if echo "$running_services" | grep -q "minio"; then
        log_info "  Storage:    https://files.$BASE_DOMAIN"
    fi
    
    if echo "$running_services" | grep -q "mailpit"; then
        log_info "  Mail UI:    https://mail.$BASE_DOMAIN"
    fi
    
    echo ""
    log_info "Commands:"
    log_info "  nself status  - Check service health"
    log_info "  nself logs    - View logs"
    log_info "  nself down    - Stop services"
}

# Show help
show_help() {
    echo "Usage: nself up [options]"
    echo ""
    echo "Start all services"
    echo ""
    echo "Options:"
    echo "  --verbose, -v    Show detailed output"
    echo "  --skip-checks    Skip pre-flight checks"
    echo "  --help, -h       Show this help"
    echo ""
    echo "Quick checks performed:"
    echo "  • Docker daemon status"
    echo "  • Environment file (.env.local)"
    echo "  • Port availability (80, 443, 5432, 8080)"
    echo ""
    echo "If errors occur, you'll be offered solutions"
}

# Export for use as library
export -f cmd_up

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    pre_command "up" || exit $?
    cmd_up "$@"
    exit_code=$?
    post_command "up" $exit_code
    exit $exit_code
fi