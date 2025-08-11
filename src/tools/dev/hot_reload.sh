#!/usr/bin/env bash

# hot-reload.sh - Detect and apply configuration changes without full rebuild

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities (only source functions we need to avoid full script execution)




# Fix paths for new structure
source "$SCRIPT_DIR/../../lib/utils/env.sh"
source "$SCRIPT_DIR/../../lib/utils/display.sh"
# State directory for tracking changes
STATE_DIR=".nself"
STATE_FILE="$STATE_DIR/build-state.json"
LAST_ENV_FILE="$STATE_DIR/last-build.env"

# Create state directory if it doesn't exist
mkdir -p "$STATE_DIR"

# Helper function to generate state checksum
generate_env_checksum() {
    local env_file="${1:-.env.local}"
    if [ -f "$env_file" ]; then
        sha256sum "$env_file" | cut -d' ' -f1
    else
        echo "none"
    fi
}

# Helper function to detect changes
detect_changes() {
    local changes=()
    
    # Check if this is first run
    if [ ! -f "$LAST_ENV_FILE" ]; then
        log_info "No previous build state found. Full build required."
        return 1
    fi
    
    # Load current and previous environments
    load_env_safe ".env.local"
    local current_checksum=$(generate_env_checksum ".env.local")
    
    # Compare checksums
    local last_checksum=$(generate_env_checksum "$LAST_ENV_FILE")
    
    if [ "$current_checksum" == "$last_checksum" ]; then
        # No configuration changes
        return 0
    fi
    
    log_info "Configuration changes detected. Analyzing..."
    
    # Detect specific changes
    local needs_nginx_reload=false
    local needs_container_restart=false
    local needs_full_rebuild=false
    local new_services=()
    local removed_services=()
    
    # Check for APP route changes
    for i in {0..20}; do
        # Check both naming conventions
        for prefix in "APP_${i}_ROUTE" "APP_ROUTE_${i}"; do
            local current_val=$(get_env_var "$prefix" ".env.local")
            local last_val=$(get_env_var "$prefix" "$LAST_ENV_FILE")
            
            if [ "$current_val" != "$last_val" ]; then
                if [ -n "$current_val" ] && [ -z "$last_val" ]; then
                    log_info "  + New app route: $prefix=$current_val"
                    needs_nginx_reload=true
                elif [ -z "$current_val" ] && [ -n "$last_val" ]; then
                    log_info "  - Removed app route: $prefix"
                    needs_nginx_reload=true
                elif [ -n "$current_val" ] && [ -n "$last_val" ]; then
                    log_info "  ~ Modified app route: $prefix"
                    needs_nginx_reload=true
                fi
            fi
        done
    done
    
    # Check for service changes
    for service_type in "GOLANG_SERVICES" "PYTHON_SERVICES" "NESTJS_SERVICES" "BULLMQ_WORKERS"; do
        local current_services=$(get_env_var "$service_type" ".env.local")
        local last_services=$(get_env_var "$service_type" "$LAST_ENV_FILE")
        
        if [ "$current_services" != "$last_services" ]; then
            log_info "  ~ Changes in $service_type"
            
            # Parse comma-separated lists
            IFS=',' read -ra current_arr <<< "$current_services"
            IFS=',' read -ra last_arr <<< "$last_services"
            
            # Find new services
            for service in "${current_arr[@]}"; do
                service=$(echo "$service" | xargs)
                if [[ ! " ${last_arr[@]} " =~ " ${service} " ]]; then
                    log_info "    + New service: $service"
                    new_services+=("$service_type:$service")
                fi
            done
            
            # Find removed services
            for service in "${last_arr[@]}"; do
                service=$(echo "$service" | xargs)
                if [[ ! " ${current_arr[@]} " =~ " ${service} " ]]; then
                    log_info "    - Removed service: $service"
                    removed_services+=("$service_type:$service")
                fi
            done
            
            needs_container_restart=true
        fi
    done
    
    # Check for PostgreSQL extension changes
    local current_ext=$(get_env_var "POSTGRES_EXTENSIONS" ".env.local")
    local last_ext=$(get_env_var "POSTGRES_EXTENSIONS" "$LAST_ENV_FILE")
    
    if [ "$current_ext" != "$last_ext" ]; then
        log_info "  ~ PostgreSQL extensions changed"
        needs_full_rebuild=true
    fi
    
    # Check for major version changes
    for var in "POSTGRES_VERSION" "HASURA_VERSION" "NGINX_VERSION"; do
        local current_ver=$(get_env_var "$var" ".env.local")
        local last_ver=$(get_env_var "$var" "$LAST_ENV_FILE")
        
        if [ "$current_ver" != "$last_ver" ] && [ -n "$current_ver" ] && [ -n "$last_ver" ]; then
            log_info "  ~ $var changed from $last_ver to $current_ver"
            needs_full_rebuild=true
        fi
    done
    
    # Return status based on changes
    if [ "$needs_full_rebuild" = true ]; then
        log_warning "Full rebuild required due to major changes."
        return 2
    elif [ "$needs_container_restart" = true ] || [ "$needs_nginx_reload" = true ]; then
        # Store change flags for apply_changes
        echo "$needs_nginx_reload" > "$STATE_DIR/.needs_nginx_reload"
        echo "$needs_container_restart" > "$STATE_DIR/.needs_container_restart"
        printf '%s\n' "${new_services[@]}" > "$STATE_DIR/.new_services"
        printf '%s\n' "${removed_services[@]}" > "$STATE_DIR/.removed_services"
        return 3
    else
        return 0
    fi
}

# Helper function to apply incremental changes
apply_changes() {
    local dry_run="${1:-false}"
    
    if [ "$dry_run" = true ]; then
        log_info "DRY RUN: Showing what would be changed..."
    fi
    
    # Read change flags
    local needs_nginx_reload=false
    local needs_container_restart=false
    
    if [ -f "$STATE_DIR/.needs_nginx_reload" ]; then
        needs_nginx_reload=$(cat "$STATE_DIR/.needs_nginx_reload")
    fi
    
    if [ -f "$STATE_DIR/.needs_container_restart" ]; then
        needs_container_restart=$(cat "$STATE_DIR/.needs_container_restart")
    fi
    
    # Apply nginx changes
    if [ "$needs_nginx_reload" = true ]; then
        log_info "Updating nginx configuration..."
        
        if [ "$dry_run" = false ]; then
            # Regenerate nginx configs for app routes
            load_env_safe ".env.local"
            
            # Process APP routes
            for i in {0..20}; do
                route_var="APP_${i}_ROUTE"
                route_value="${!route_var}"
                
                # Also check alternative naming
                if [[ -z "$route_value" ]]; then
                    route_var="APP_ROUTE_$i"
                    route_value="${!route_var}"
                fi
                
                if [[ -n "$route_value" ]]; then
                    # Expand variables and parse
                    route_value=$(eval echo "$route_value")
                    IFS=':' read -r port domain <<< "$route_value"
                    subdomain="${domain%%.*}"
                    
                    log_info "  Configuring route: $subdomain (localhost:$port -> $domain)"
                    
                    # Generate nginx config file
                    generate_nginx_app_config "$subdomain" "$port" "$domain"
                elif [ -f "nginx/conf.d/app-route-$i.conf" ]; then
                    # Remove config if route was deleted
                    if [ "$dry_run" = false ]; then
                        rm -f "nginx/conf.d/app-route-$i.conf"
                        log_info "  Removed config for app route $i"
                    else
                        log_info "  Would remove config for app route $i"
                    fi
                fi
            done
            
            # Reload nginx
            if [ "$dry_run" = false ]; then
                log_info "Reloading nginx..."
                docker exec ${PROJECT_NAME}_nginx nginx -s reload || {
                    log_warning "Failed to reload nginx. Container might not be running."
                }
            else
                log_info "Would reload nginx"
            fi
        fi
    fi
    
    # Handle new services
    if [ -f "$STATE_DIR/.new_services" ]; then
        while IFS= read -r service_entry; do
            if [ -n "$service_entry" ]; then
                IFS=':' read -r service_type service_name <<< "$service_entry"
                log_info "Adding new service: $service_name ($service_type)"
                
                if [ "$dry_run" = false ]; then
                    # Generate service structure
                    case "$service_type" in
                        GOLANG_SERVICES)
                            generate_go_service "$service_name"
                            ;;
                        PYTHON_SERVICES)
                            generate_python_service "$service_name"
                            ;;
                        NESTJS_SERVICES)
                            generate_nestjs_service "$service_name"
                            ;;
                        BULLMQ_WORKERS)
                            generate_bullmq_worker "$service_name"
                            ;;
                    esac
                    
                    # Rebuild docker-compose.yml
                    bash "$BIN_DIR/generators/compose.sh"
                    
                    # Start the new service
                    docker-compose up -d "$service_name" || {
                        log_warning "Failed to start service $service_name"
                    }
                else
                    log_info "  Would generate and start $service_name"
                fi
            fi
        done < "$STATE_DIR/.new_services"
    fi
    
    # Handle removed services
    if [ -f "$STATE_DIR/.removed_services" ]; then
        while IFS= read -r service_entry; do
            if [ -n "$service_entry" ]; then
                IFS=':' read -r service_type service_name <<< "$service_entry"
                log_info "Removing service: $service_name"
                
                if [ "$dry_run" = false ]; then
                    # Stop and remove container
                    docker-compose stop "$service_name" 2>/dev/null || true
                    docker-compose rm -f "$service_name" 2>/dev/null || true
                    
                    # Rebuild docker-compose.yml
                    bash "$BIN_DIR/generators/compose.sh"
                    
                    log_info "  Service $service_name stopped (code preserved)"
                else
                    log_info "  Would stop and remove $service_name container"
                fi
            fi
        done < "$STATE_DIR/.removed_services"
    fi
    
    # Clean up state files
    if [ "$dry_run" = false ]; then
        rm -f "$STATE_DIR/.needs_nginx_reload"
        rm -f "$STATE_DIR/.needs_container_restart"
        rm -f "$STATE_DIR/.new_services"
        rm -f "$STATE_DIR/.removed_services"
        
        # Save current state
        cp ".env.local" "$LAST_ENV_FILE"
    fi
    
    log_success "Changes applied successfully!"
}

# Helper function to generate nginx app config
generate_nginx_app_config() {
    local subdomain=$1
    local port=$2
    local domain=$3
    
    # Detect OS for proper host networking
    OS="$(uname -s)"
    if [[ "$OS" == "Linux" ]]; then
        PROXY_HOST="172.17.0.1"
    else
        PROXY_HOST="host.docker.internal"
    fi
    
    cat > "nginx/conf.d/${subdomain}.conf" << EOF
# Frontend App Route: $subdomain
# localhost:$port -> $domain

server {
    listen 80;
    server_name $domain;

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name $domain;

    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;

    location / {
        proxy_pass http://$PROXY_HOST:$port;
        
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Forwarded-Port \$server_port;
        
        # WebSocket support
        proxy_read_timeout 86400;
        
        # Timeouts
        proxy_connect_timeout 60;
        proxy_send_timeout 60;
        proxy_read_timeout 60;
    }
}
EOF
}

# Main hot reload logic
hot_reload() {
    local action="${1:-check}"
    
    case "$action" in
        check)
            detect_changes
            local status=$?
            
            if [ $status -eq 0 ]; then
                log_success "No changes to apply."
            elif [ $status -eq 2 ]; then
                log_warning "Full rebuild required. Run 'nself build' to apply changes."
            elif [ $status -eq 3 ]; then
                log_info "Incremental changes can be applied."
                log_info "Run 'nself up --apply-changes' to apply them."
            fi
            ;;
            
        apply)
            detect_changes
            local status=$?
            
            if [ $status -eq 0 ]; then
                log_success "No changes to apply."
            elif [ $status -eq 2 ]; then
                log_warning "Full rebuild required. Run 'nself build' to apply changes."
                exit 1
            elif [ $status -eq 3 ]; then
                apply_changes false
            fi
            ;;
            
        dry-run)
            detect_changes
            local status=$?
            
            if [ $status -eq 3 ]; then
                apply_changes true
            fi
            ;;
            
        save-state)
            # Save current state without applying changes
            cp ".env.local" "$LAST_ENV_FILE" 2>/dev/null || true
            log_success "Current state saved."
            ;;
            
        *)
            log_error "Unknown action: $action"
            log_info "Valid actions: check, apply, dry-run, save-state"
            exit 1
            ;;
    esac
}

# Export for use in other scripts
export -f hot_reload
export -f detect_changes
export -f apply_changes