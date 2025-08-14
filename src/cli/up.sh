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
source "$SCRIPT_DIR/../lib/auto-fix/config-validator-v2.sh"
source "$SCRIPT_DIR/../lib/auto-fix/auto-fixer-v2.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"
source "$SCRIPT_DIR/../lib/config/smart-defaults.sh"

# Load environment with smart defaults
load_env_with_defaults >/dev/null 2>&1

# Command function
cmd_up() {
    local verbose=false
    local skip_checks=false
    local detached=true
    local retry_count="${UP_RETRY_COUNT:-0}"
    # Default to true for ALWAYS_AUTOFIX unless explicitly set to false
    local auto_fix="${ALWAYS_AUTOFIX:-true}"
    if [[ "$auto_fix" == "false" ]]; then
        auto_fix="false"
    else
        # Any value other than "false" means true (including empty/unset)
        auto_fix="true"
    fi
    local max_retries=30
    
    # Check for ALWAYS_AUTOFIX mode
    if [[ "$auto_fix" == "true" ]]; then
        max_retries=30  # Allow more retries in auto-fix mode
    else
        max_retries=5   # Fewer retries in interactive mode
    fi
    
    # Prevent infinite retry loops
    if [[ $retry_count -ge $max_retries ]]; then
        if [[ "$auto_fix" == "true" ]]; then
            printf "\r${COLOR_RED}✗${COLOR_RESET} Auto-fix exceeded $max_retries attempts                     \n"
        else
            log_error "Too many retries. Please check your configuration."
        fi
        return 1
    fi
    
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
            --attach|-a)
                detached=false
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
    
    # Show header with proper nself formatting (skip on retry or in auto-fix mode)
    if [[ $retry_count -eq 0 ]] && [[ "$auto_fix" != "true" ]]; then
        echo
        echo -e "${COLOR_BLUE}╔══════════════════════════════════════════════════════════╗${COLOR_RESET}"
        echo -e "${COLOR_BLUE}║${COLOR_RESET}  ${BOLD}nself up${RESET}                                                ${COLOR_BLUE}║${COLOR_RESET}"
        echo -e "${COLOR_BLUE}║${COLOR_RESET}  ${COLOR_DIM}Starting all services and infrastructure${COLOR_RESET}                ${COLOR_BLUE}║${COLOR_RESET}"
        echo -e "${COLOR_BLUE}╚══════════════════════════════════════════════════════════╝${COLOR_RESET}"
        echo
    elif [[ $retry_count -eq 0 ]] && [[ "$auto_fix" == "true" ]]; then
        # Concise output for auto-fix mode
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Starting all services..."
    fi
    
    # Run comprehensive pre-flight checks (skip if retrying after auto-fix)
    if [[ $retry_count -eq 0 ]]; then
        if ! source "$SCRIPT_DIR/../lib/utils/preflight.sh" 2>/dev/null; then
            log_error "Failed to load pre-flight checks"
            return 1
        fi
        
        if [[ "$auto_fix" != "true" ]]; then
            printf "${COLOR_BLUE}⠋${COLOR_RESET} Running pre-flight checks..."
            if preflight_up >/dev/null 2>&1; then
                printf "\r${COLOR_GREEN}✓${COLOR_RESET} Pre-flight checks passed                   \n"
            else
                printf "\r${COLOR_RED}✗${COLOR_RESET} Pre-flight checks failed                   \n"
                preflight_up  # Run again to show the actual errors
                return 1
            fi
        else
            # Silent pre-flight checks in auto-fix mode
            if ! preflight_up >/dev/null 2>&1; then
                # Only show failure if we can't continue
                printf "\r${COLOR_RED}✗${COLOR_RESET} System requirements not met                \n"
                return 1
            fi
        fi
    fi
    
    # Comprehensive upfront port checking (skip on retry)
    if [[ "$skip_checks" != "true" ]] && [[ $retry_count -eq 0 ]]; then
        # Source port scanner
        if [[ -f "$SCRIPT_DIR/../lib/utils/port-scanner.sh" ]]; then
            source "$SCRIPT_DIR/../lib/utils/port-scanner.sh"
            
            printf "${COLOR_BLUE}⠋${COLOR_RESET} Checking port availability..."
            
            # Pre-check all ports from docker-compose
            local port_conflicts=$(precheck_all_ports "docker-compose.yml")
            
            if [[ -n "$port_conflicts" ]]; then
                printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Port conflicts detected                    \n"
                echo
                
                # Parse and display conflicts
                local has_conflicts=false
                for conflict in $port_conflicts; do
                    IFS=':' read -r port info <<< "$conflict"
                    IFS='|' read -r pid process path <<< "$info"
                    
                    if [[ "$port" != "" ]]; then
                        has_conflicts=true
                        log_error "Port $port is in use"
                        if [[ "$process" != "unknown" ]]; then
                            log_info "Used by: $path (PID: $pid)"
                        fi
                    fi
                done
                
                if [[ "$has_conflicts" == "true" ]]; then
                    local choice=1  # Default to auto-fix
                    
                    if [[ "$auto_fix" != "true" ]]; then
                        echo
                        log_info "Port conflict options:"
                        echo "  1) Auto-fix: Change to alternative ports"
                        echo "  2) Stop conflicting services"  
                        echo "  3) Continue anyway (may fail)"
                        echo "  4) Cancel"
                        echo
                        
                        read -p "Select option (1-4): " choice
                    fi
                    
                    case "$choice" in
                        1)
                            if [[ "$auto_fix" == "true" ]]; then
                                # Concise output in auto-fix mode
                                local port_changes=""
                                for conflict in $port_conflicts; do
                                    IFS=':' read -r port info <<< "$conflict"
                                    if [[ -n "$port" ]]; then
                                        local new_port=$(suggest_alternative_port "$port")
                                        if [[ -n "$new_port" ]]; then
                                            fix_port_in_env "" "$port" "$new_port"
                                            port_changes="${port_changes}Port $port→$new_port, "
                                        fi
                                    fi
                                done
                                
                                if [[ -n "$port_changes" ]]; then
                                    printf "\r${COLOR_YELLOW}⚡${COLOR_RESET} ${port_changes%, }              \n"
                                    printf "${COLOR_BLUE}⠋${COLOR_RESET} Rebuilding configuration..."
                                    if nself build --force >/dev/null 2>&1; then
                                        printf "\r${COLOR_BLUE}⠋${COLOR_RESET} Starting all services...                   "
                                    else
                                        printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to rebuild                         \n"
                                        return 1
                                    fi
                                fi
                            else
                                log_info "Auto-fixing port conflicts..."
                                
                                # Auto-fix each conflict
                                for conflict in $port_conflicts; do
                                    IFS=':' read -r port info <<< "$conflict"
                                    if [[ -n "$port" ]]; then
                                        local new_port=$(suggest_alternative_port "$port")
                                        if [[ -n "$new_port" ]]; then
                                            log_info "Changing port $port to $new_port"
                                            fix_port_in_env "" "$port" "$new_port"
                                        fi
                                    fi
                                done
                                
                                log_success "Ports updated in .env.local"
                                log_info "Rebuilding configuration..."
                                
                                # Rebuild with new ports
                                if nself build --force >/dev/null 2>&1; then
                                    log_success "Configuration rebuilt with new ports"
                                else
                                    log_error "Failed to rebuild configuration"
                                    return 1
                                fi
                            fi
                            ;;
                        2)
                            log_info "Stop the conflicting services manually, then run 'nself up' again"
                            return 1
                            ;;
                        3)
                            log_warning "Continuing despite port conflicts..."
                            ;;
                        4)
                            log_info "Cancelled"
                            return 1
                            ;;
                        *)
                            log_error "Invalid option"
                            return 1
                            ;;
                    esac
                fi
            else
                printf "\r${COLOR_GREEN}✓${COLOR_RESET} Ports available                            \n"
            fi
        else
            # Fallback to old check
            printf "${COLOR_BLUE}⠋${COLOR_RESET} Checking port availability..."
            if ! run_essential_checks false >/dev/null 2>&1; then
                printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Port conflicts detected                    \n"
                echo
                run_essential_checks true
                if [[ $? -ne 0 ]]; then
                    echo
                    log_error "Port checks failed"
                    return 1
                fi
            else
                printf "\r${COLOR_GREEN}✓${COLOR_RESET} Ports available                            \n"
            fi
        fi
    fi
    
    # Skip validation on retry (we're continuing from where we left off)
    if [[ $retry_count -eq 0 ]]; then
        # Validate docker-compose.yml exists
        if [[ ! -f "docker-compose.yml" ]]; then
            printf "${COLOR_RED}✗${COLOR_RESET} docker-compose.yml not found               \n"
            echo
            log_info "Run 'nself build' first to generate infrastructure"
            return 1
        fi
        
        # Load environment for validation
        if [[ -f ".env.local" ]]; then
            set -a
            source .env.local
            set +a
        fi
        
        # Validate docker-compose.yml
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Validating configuration..."
        if compose config >/dev/null 2>&1; then
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} Configuration valid                        \n"
        else
            printf "\r${COLOR_RED}✗${COLOR_RESET} Invalid docker-compose.yml                \n"
            echo
            # Show the actual validation error
            local validation_error=$(compose config 2>&1 | grep -v "\.go:[0-9]" | head -5)
            if [[ -n "$validation_error" ]]; then
                echo "$validation_error"
                echo
            fi
            log_info "Run 'nself build' to regenerate configuration"
            return 1
        fi
    else
        # On retry, still need to load environment
        if [[ -f ".env.local" ]]; then
            set -a
            source .env.local
            set +a
        fi
    fi
    
    if [[ $retry_count -eq 0 ]]; then
        echo
        echo -e "${COLOR_CYAN}➞ Starting Services${COLOR_RESET}"
        echo
    fi
    
    # Start services
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Starting Docker containers..."
    
    local output_file=$(mktemp)
    local result
    
    # Ensure environment is loaded for compose wrapper
    if [[ -f ".env.local" ]]; then
        set -a
        source .env.local
        set +a
    fi
    
    # Build the compose command
    local compose_cmd="compose up"
    if [[ "$detached" == "true" ]]; then
        compose_cmd="$compose_cmd -d"
    fi
    compose_cmd="$compose_cmd --build"
    
    if [[ "$verbose" == "true" ]]; then
        # Show full output in verbose mode
        printf "\n"
        eval "$compose_cmd" 2>&1 | tee "$output_file"
        result=${PIPESTATUS[0]}
    else
        # Run silently with spinner
        (eval "$compose_cmd" 2>&1) > "$output_file" &
        local compose_pid=$!
        
        # Show spinner while waiting
        local spin_chars="⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
        local i=0
        while kill -0 $compose_pid 2>/dev/null; do
            local char="${spin_chars:$((i % ${#spin_chars})):1}"
            printf "\r${COLOR_BLUE}%s${COLOR_RESET} Starting Docker containers...          " "$char"
            ((i++))
            sleep 0.1
        done
        wait $compose_pid
        result=$?
    fi
    
    if [[ $result -eq 0 ]]; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Docker containers started                  \n"
        
        # Verify services are actually running
        printf "${COLOR_BLUE}⠋${COLOR_RESET} Verifying services..."
        sleep 2  # Give services a moment to fully start
        
        local running_services=$(docker compose ps --services --filter "status=running" 2>/dev/null | wc -l | tr -d ' ')
        local total_services=$(docker compose ps --services 2>/dev/null | wc -l | tr -d ' ')
        
        if [[ $running_services -eq $total_services ]] && [[ $total_services -gt 0 ]]; then
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} All services running ($running_services/$total_services)              \n"
            
            echo
            log_success "Services started successfully!"
            show_service_urls
            
            echo
            echo -e "${COLOR_CYAN}➞ Next Steps${COLOR_RESET}"
            echo
            echo -e "${COLOR_BLUE}1.${COLOR_RESET} ${COLOR_BLUE}nself status${COLOR_RESET} - Check service health"
            echo -e "   ${COLOR_DIM}View detailed status of all services${COLOR_RESET}"
            echo
            echo -e "${COLOR_BLUE}2.${COLOR_RESET} ${COLOR_BLUE}nself logs${COLOR_RESET} - View service logs"
            echo -e "   ${COLOR_DIM}Monitor real-time logs from services${COLOR_RESET}"
            echo
            echo -e "${COLOR_BLUE}3.${COLOR_RESET} ${COLOR_BLUE}nself urls${COLOR_RESET} - View service endpoints"
            echo -e "   ${COLOR_DIM}Display all available service URLs${COLOR_RESET}"
        else
            printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Some services failed to start ($running_services/$total_services)    \n"
            echo
            analyze_startup_error "$(cat "$output_file")"
            rm -f "$output_file"
            return 1
        fi
    else
        printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to start services                   \n"
        echo
        
        # Save output for debugging if DEBUG is set
        if [[ "${DEBUG:-}" == "true" ]]; then
            log_info "Debug: Output saved to /tmp/nself-up-error.log"
            cp "$output_file" /tmp/nself-up-error.log
        fi
        
        # Analyze the specific error
        analyze_startup_error "$(cat "$output_file")"
        local analyze_result=$?
        rm -f "$output_file"
        
        # Check if auto-fix requested a retry
        if [[ $analyze_result -eq 99 ]]; then
            # Retry the up command after auto-fix with incremented counter
            UP_RETRY_COUNT=$((retry_count + 1)) cmd_up "$@"
            return $?
        fi
        
        return 1
    fi
    
    rm -f "$output_file"
}

# Analyze startup errors and offer solutions
analyze_startup_error() {
    local output="$1"
    
    # Container name conflicts
    if echo "$output" | grep -q "The container name.*is already in use"; then
        local container=$(echo "$output" | grep -oE 'container name "[^"]+"' | grep -oE '"[^"]+"' | tr -d '"' | head -1)
        if [[ -n "$container" ]]; then
            log_error "Container name conflict: $container"
            echo
            log_info "Another project is using the same container names"
            echo
            log_info "Solutions:"
            echo "  1. Stop the other project: ${COLOR_BLUE}docker stop $container${COLOR_RESET}"
            echo "  2. Remove old containers: ${COLOR_BLUE}docker rm $container${COLOR_RESET}"  
            echo "  3. Change PROJECT_NAME in .env.local"
            echo "  4. Use a different project: ${COLOR_BLUE}nself down && nself init --project new-name${COLOR_RESET}"
        else
            log_error "Container name conflict detected"
            echo "$output" | grep "container name" | head -3
        fi
        
    # Port conflicts
    elif echo "$output" | grep -q "port is already allocated\|bind: address already in use"; then
        # Extract port from various Docker error formats
        local port=$(echo "$output" | grep -oE "0\.0\.0\.0:[0-9]+" | grep -oE "[0-9]+$" | head -1)
        if [[ -z "$port" ]]; then
            port=$(echo "$output" | grep -oE "bind: address already in use.*:[0-9]+" | grep -oE "[0-9]+$" | head -1)
        fi
        if [[ -z "$port" ]]; then
            port=$(echo "$output" | grep -oE "port [0-9]+" | grep -oE "[0-9]+" | head -1)
        fi
        
        if [[ -n "$port" ]] && [[ "$port" != "0" ]]; then
            log_error "Port $port is already in use"
            
            # Try to identify the process
            if command -v lsof >/dev/null 2>&1; then
                local process_pid=$(lsof -i :$port -sTCP:LISTEN -t 2>/dev/null | head -1)
                if [[ -n "$process_pid" ]]; then
                    local process_name=$(ps -p $process_pid -o comm= 2>/dev/null)
                    local full_path=$(ps -p $process_pid -o command= 2>/dev/null | cut -d' ' -f1)
                    echo
                    log_info "Used by: $full_path (PID: $process_pid)"
                fi
            fi
            
            # Source port scanner for auto-fix capabilities
            if [[ -f "$SCRIPT_DIR/../lib/utils/port-scanner.sh" ]]; then
                source "$SCRIPT_DIR/../lib/utils/port-scanner.sh"
                
                echo
                log_info "Port conflict options:"
                echo "  1) Auto-fix: Change to alternative port"
                echo "  2) Stop conflicting service"
                echo "  3) Cancel"
                echo
                
                read -p "Select option (1-3): " choice
                
                case "$choice" in
                    1)
                        local new_port=$(suggest_alternative_port "$port")
                        if [[ -n "$new_port" ]]; then
                            log_info "Changing port $port to $new_port in .env.local"
                            fix_port_in_env "" "$port" "$new_port"
                            log_success "Port updated"
                            echo
                            log_info "Rebuilding and retrying..."
                            
                            # Rebuild and retry
                            if nself build --force >/dev/null 2>&1; then
                                sleep 2
                                return 99  # Retry
                            fi
                        else
                            log_error "Could not find alternative port"
                        fi
                        ;;
                    2)
                        if [[ -n "$process_pid" ]]; then
                            log_info "Run: ${COLOR_BLUE}kill $process_pid${COLOR_RESET}"
                        fi
                        log_info "Then run 'nself up' again"
                        ;;
                    3)
                        log_info "Cancelled"
                        ;;
                    *)
                        log_error "Invalid option"
                        ;;
                esac
            else
                echo
                log_info "Solutions:"
                echo "  1. Stop the conflicting service"
                echo "  2. Change the port in .env.local"
                echo "  3. Run: ${COLOR_BLUE}nself down && nself up${COLOR_RESET}"
            fi
        else
            log_error "Port conflict detected"
            echo "$output" | grep -E "port|bind" | head -5
        fi
        
    # Network conflicts
    elif echo "$output" | grep -q "a network with name.*exists but was not created"; then
        local network=$(echo "$output" | grep -oE 'network with name [^ ]+' | cut -d' ' -f4)
        log_error "Network conflict: $network"
        echo
        log_info "Another project is using the same network name"
        echo
        log_info "Solutions:"
        echo "  1. Remove the old network: ${COLOR_BLUE}docker network rm $network${COLOR_RESET}"
        echo "  2. Change PROJECT_NAME in .env.local"
        echo "  3. Run: ${COLOR_BLUE}nself build --force && nself up${COLOR_RESET}"
        
    # Volume conflicts
    elif echo "$output" | grep -q "volume.*already exists but was created for project"; then
        local volume=$(echo "$output" | grep -oE 'volume "[^"]+"' | grep -oE '"[^"]+"' | tr -d '"' | head -1)
        log_error "Volume conflict: $volume"
        echo
        log_info "Another project is using the same volume names"
        echo
        log_info "Solutions:"
        echo "  1. Remove old volumes: ${COLOR_BLUE}docker volume rm $volume${COLOR_RESET}"
        echo "  2. Change PROJECT_NAME in .env.local"
        echo "  3. Use different volumes for this project"
        
    # Build context errors (missing directories)
    elif echo "$output" | grep -q "unable to prepare context:\|path.*not found"; then
        log_error "Build context error - missing service directory"
        
        # Extract the missing path
        local missing_path=$(echo "$output" | grep -oE 'path "[^"]+"' | grep -oE '"[^"]+"' | tr -d '"' | head -1)
        if [[ -z "$missing_path" ]]; then
            missing_path=$(echo "$output" | grep "not found" | head -1 | sed 's/.*path //' | sed 's/ not found.*//')
        fi
        
        if [[ -n "$missing_path" ]]; then
            # Make path relative if it's absolute and in current project
            local relative_path="${missing_path#$(pwd)/}"
            echo "Missing: $relative_path"
            
            # Check if this is a service directory (handle various naming)
            # Match services/(type)/(name) pattern - we support any service type in env
            if [[ "$relative_path" =~ ^services/([^/]+)/([^/]+) ]]; then
                echo
                log_info "Auto-fixing: Generating missing service..."
                
                # Source service generator
                if [[ -f "$SCRIPT_DIR/../lib/auto-fix/service-generator.sh" ]]; then
                    source "$SCRIPT_DIR/../lib/auto-fix/service-generator.sh"
                    
                    if check_missing_service "$relative_path"; then
                        log_success "Service generated successfully"
                        echo
                        log_info "Continuing startup with newly generated service..."
                        echo
                        # Clean up temporary file
                        rm -f "$output_file"
                        
                        # Give filesystem and Docker time to recognize new files
                        sleep 2
                        
                        # Return special code to indicate retry needed
                        return 99  # Special code to indicate retry needed
                    fi
                fi
            fi
        fi
        
        echo
        log_info "Solutions:"
        echo "  1. Create the missing directory: ${COLOR_BLUE}mkdir -p ${relative_path}${COLOR_RESET}"
        echo "  2. Check your docker-compose.yml for incorrect paths"
        echo "  3. Rebuild configuration: ${COLOR_BLUE}nself build --force${COLOR_RESET}"
        
    # Build errors  
    elif echo "$output" | grep -q "failed to solve\|exit code:\|missing go.sum entry\|failed to read dockerfile"; then
        log_error "Build error detected"
        
        # Check for missing Dockerfile
        if echo "$output" | grep -q "failed to read dockerfile.*no such file"; then
            # Extract the service name from the error
            local service_name=$(echo "$output" | grep -oE "target [^:]+:" | sed 's/target //; s/://' | head -1)
            if [[ -n "$service_name" ]]; then
                if [[ "${ALWAYS_AUTOFIX:-false}" == "true" ]]; then
                    # Concise output in auto-fix mode
                    printf "\r${COLOR_YELLOW}⚡${COLOR_RESET} Generating $service_name Dockerfile...              \n"
                    
                    # Source the dockerfile generator
                    if [[ -f "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh" ]]; then
                        source "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh"
                        
                        # Generate the appropriate service files
                        if generate_dockerfile_for_service "$service_name" >/dev/null 2>&1; then
                            printf "${COLOR_BLUE}⠋${COLOR_RESET} Continuing startup..."
                            sleep 2  # Give Docker time to recognize new files
                            return 99  # Retry
                        else
                            printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to generate Dockerfile              \n"
                        fi
                    fi
                else
                    log_info "Service '$service_name' is missing its Dockerfile"
                    echo
                    log_info "Auto-fixing: Generating $service_name service..."
                    
                    # Source the dockerfile generator
                    if [[ -f "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh" ]]; then
                        source "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh"
                        
                        # Generate the appropriate service files
                        if generate_dockerfile_for_service "$service_name"; then
                            log_success "Service $service_name generated successfully"
                            echo
                            log_info "Continuing startup..."
                            echo
                            sleep 2  # Give Docker time to recognize new files
                            return 99  # Retry
                        else
                            log_error "Failed to generate service $service_name"
                        fi
                    else
                        log_error "Dockerfile generator not found"
                    fi
                fi
            fi
        fi
        
        if echo "$output" | grep -q "missing go.sum entry\|go mod\|go.sum"; then
            log_info "Go module issue detected"
            echo
            log_info "Auto-fixing: Regenerating Go services..."
            
            # Find and regenerate all Go services
            if [[ -d "services/go" ]]; then
                for service_dir in services/go/*/; do
                    if [[ -d "$service_dir" ]]; then
                        service_name=$(basename "$service_dir")
                        log_info "Fixing Go service: $service_name"
                        
                        # Add go.sum if missing
                        if [[ ! -f "$service_dir/go.sum" ]]; then
                            cat > "$service_dir/go.sum" << 'EOF'
github.com/gorilla/mux v1.8.0 h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=
github.com/gorilla/mux v1.8.0/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
EOF
                        fi
                    fi
                done
                log_success "Go services fixed"
                echo
                log_info "Continuing startup..."
                echo
                sleep 2
                return 99  # Retry
            else
                echo
                log_info "Try: ${COLOR_BLUE}nself build --force && nself up${COLOR_RESET}"
            fi
        else
            # Show relevant build error lines
            echo "$output" | grep -E "ERROR|failed|exit code" | head -10
            echo
            log_info "Try: ${COLOR_BLUE}docker compose build --no-cache${COLOR_RESET}"
        fi
        
    # Docker not running
    elif echo "$output" | grep -q "Cannot connect to the Docker daemon\|docker daemon is not running"; then
        log_error "Docker is not running"
        echo
        log_info "Please start Docker Desktop or run:"
        echo "  ${COLOR_BLUE}sudo systemctl start docker${COLOR_RESET}"
        
    # Network issues
    elif echo "$output" | grep -q "network.*not found\|Error response from daemon.*network"; then
        log_error "Docker network issue"
        
        local network=$(echo "$output" | grep -oE "network [^ ]+" | cut -d' ' -f2 | head -1)
        if [[ -n "$network" ]]; then
            log_info "Network '$network' not found"
        fi
        
        echo
        log_info "Try: ${COLOR_BLUE}docker network prune && nself build && nself up${COLOR_RESET}"
        
    # Permission issues
    elif echo "$output" | grep -q "permission denied\|Permission denied"; then
        log_error "Permission denied"
        echo "$output" | grep -i "permission" | head -3
        echo
        log_info "Try: ${COLOR_BLUE}sudo nself up${COLOR_RESET}"
        
    # Unhealthy service dependency
    elif echo "$output" | grep -q "dependency failed to start.*is unhealthy"; then
        local unhealthy_service=$(echo "$output" | grep -oE "container [^ ]+ is unhealthy" | sed 's/container //; s/ is unhealthy//' | head -1)
        
        if [[ -n "$unhealthy_service" ]]; then
            log_error "Service is unhealthy: $unhealthy_service"
            echo
            
            # Get logs from the unhealthy service
            local service_logs=$(docker logs "$unhealthy_service" 2>&1 | tail -10)
            if [[ -n "$service_logs" ]]; then
                log_info "Recent logs from $unhealthy_service:"
                echo "$service_logs" | head -5
                echo
            fi
            
            # Check if it's a missing dependency or configuration issue
            if echo "$service_logs" | grep -q "Cannot connect\|connection refused\|ECONNREFUSED"; then
                log_info "Service has connection issues. Checking dependencies..."
            fi
            
            echo
            log_info "Options:"
            echo "  1) Auto-fix: Recreate the service"
            echo "  2) View full logs"
            echo "  3) Continue anyway"
            echo "  4) Cancel"
            echo
            
            read -p "Select option (1-4): " choice
            
            case "$choice" in
                1)
                    log_info "Recreating $unhealthy_service..."
                    docker stop "$unhealthy_service" >/dev/null 2>&1
                    docker rm "$unhealthy_service" >/dev/null 2>&1
                    
                    # Check if config-server needs regeneration
                    if [[ "$unhealthy_service" == *"config-server"* ]] && [[ ! -f "config-server/index.js" ]]; then
                        log_info "Regenerating config-server files..."
                        if [[ -f "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh" ]]; then
                            source "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh"
                            generate_dockerfile_for_service "config-server" "config-server"
                        fi
                    fi
                    
                    log_success "Service recreated"
                    echo
                    log_info "Retrying startup..."
                    sleep 2
                    return 99  # Retry
                    ;;
                2)
                    docker logs "$unhealthy_service" 2>&1 | tail -50
                    echo
                    log_info "Run 'nself up' again when ready"
                    ;;
                3)
                    log_warning "Continuing with unhealthy service..."
                    ;;
                4)
                    log_info "Cancelled"
                    ;;
                *)
                    log_error "Invalid option"
                    ;;
            esac
        else
            log_error "Service dependency is unhealthy"
            echo "$output" | grep "unhealthy" | head -3
        fi
        
    # Generic error - show relevant output
    elif true; then
        # First check for common Docker Compose issues
        if echo "$output" | grep -q "no configuration file provided"; then
            log_error "docker-compose.yml not found"
            echo
            log_info "Run: ${COLOR_BLUE}nself build${COLOR_RESET}"
        elif echo "$output" | grep -q "yaml: "; then
            log_error "Invalid docker-compose.yml format"
            local yaml_error=$(echo "$output" | grep "yaml: " | head -1)
            echo "$yaml_error"
            echo
            log_info "Run: ${COLOR_BLUE}nself build --force${COLOR_RESET}"
        elif echo "$output" | grep -q "pull access denied\|no matching manifest"; then
            log_error "Docker image pull failed"
            local image_error=$(echo "$output" | grep -E "pull access denied|no matching manifest" | head -1)
            echo "$image_error"
            echo
            log_info "Check your Docker Hub access or image names"
        else
            # Generic error - try to find meaningful lines
            log_error "Service startup failed"
            
            # Look for actual error messages, not stack traces
            local error_lines=$(echo "$output" | grep -v "\.go:[0-9]" | grep -v "runtime\." | grep -v "github.com/" | grep -v "^[[:space:]]*$" | grep -E "error|ERROR|failed|Failed|unable|Unable" | head -5)
            
            if [[ -n "$error_lines" ]]; then
                echo "$error_lines"
            else
                # If no clear error, show the docker compose command output
                local compose_output=$(echo "$output" | grep -E "Container.*Creating|Container.*Error|Error response from daemon" | head -5)
                if [[ -n "$compose_output" ]]; then
                    echo "$compose_output"
                else
                    # Last resort - show any non-stack-trace output
                    echo "$output" | grep -v "\.go:[0-9]" | grep -v "^[[:space:]]*$" | head -5
                fi
            fi
        fi
    fi
    
    echo
    log_info "For more details, run: ${COLOR_BLUE}nself up --verbose${COLOR_RESET}"
}

# Show service URLs
show_service_urls() {
    if [[ ! -f ".env.local" ]]; then
        return
    fi
    
    source .env.local
    
    echo -e "${COLOR_CYAN}➞ Service URLs${COLOR_RESET}"
    echo
    
    # Check which services are actually running
    local running_services=$(docker compose ps --services --filter "status=running" 2>/dev/null)
    
    if echo "$running_services" | grep -q "hasura"; then
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} GraphQL:    ${COLOR_BLUE}https://${HASURA_ROUTE:-gql.$BASE_DOMAIN}${COLOR_RESET}"
    fi
    
    if echo "$running_services" | grep -q "auth"; then
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} Auth:       ${COLOR_BLUE}https://${AUTH_ROUTE:-auth.$BASE_DOMAIN}${COLOR_RESET}"
    fi
    
    if echo "$running_services" | grep -q "minio"; then
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} Storage:    ${COLOR_BLUE}https://${STORAGE_ROUTE:-files.$BASE_DOMAIN}${COLOR_RESET}"
    fi
    
    if echo "$running_services" | grep -q "mailpit"; then
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} Mail UI:    ${COLOR_BLUE}https://mail.$BASE_DOMAIN${COLOR_RESET}"
    fi
    
    if echo "$running_services" | grep -q "postgres"; then
        local pg_port="${POSTGRES_PORT:-5432}"
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} Database:   ${COLOR_BLUE}postgresql://localhost:$pg_port${COLOR_RESET}"
    fi
}

# Show help
show_help() {
    echo "Usage: nself up [OPTIONS]"
    echo ""
    echo "Start all services defined in docker-compose.yml"
    echo ""
    echo "Options:"
    echo "  -v, --verbose      Show detailed output"
    echo "  -a, --attach       Run in foreground (attached mode)"
    echo "  --skip-checks      Skip port availability checks"
    echo "  -h, --help         Show this help message"
    echo ""
    echo "Examples:"
    echo "  nself up           # Start services in background"
    echo "  nself up -v        # Start with verbose output"
    echo "  nself up -a        # Start in foreground mode"
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