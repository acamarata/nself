#!/usr/bin/env bash
# prod.sh - Configure for production deployment

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"

# Command function
cmd_prod() {
    local domain=""
    local email=""
    local level="basic"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_prod_help
                return 0
                ;;
            --level)
                level="$2"
                shift 2
                ;;
            --email)
                email="$2"
                shift 2
                ;;
            *)
                if [[ -z "$domain" ]]; then
                    domain="$1"
                elif [[ -z "$email" ]]; then
                    email="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Show command header
    show_command_header "nself prod" "Configure for production deployment"
    
    # If no domain provided, use default based on level
    if [[ -z "$domain" ]]; then
        case "$level" in
            basic)
                domain="example.com"
                ;;
            standard)
                domain="yourcompany.com"
                ;;
            enterprise)
                domain="enterprise.com"
                ;;
            *)
                domain="localhost"
                ;;
        esac
    fi
    
    show_header "Production Configuration"
    
    log_info "Configuring for production deployment"
    log_info "Domain: $domain"
    
    # Load current environment
    if [[ -f ".env.local" ]]; then
        source .env.local
    else
        log_error "No .env.local found. Run 'nself init' first"
        return 1
    fi
    
    # Update configuration for production
    log_info "Updating configuration..."
    
    # Set production domain
    set_env_var "BASE_DOMAIN" "$domain" ".env.local"
    
    # Enable SSL
    set_env_var "SSL_ENABLED" "true" ".env.local"
    
    # Set SSL email if provided
    if [[ -n "$email" ]]; then
        set_env_var "SSL_EMAIL" "$email" ".env.local"
    fi
    
    # Set production mode
    set_env_var "NODE_ENV" "production" ".env.local"
    set_env_var "GO_ENV" "production" ".env.local"
    set_env_var "DEBUG" "false" ".env.local"
    set_env_var "VERBOSE" "false" ".env.local"
    
    # Generate strong secrets if still using defaults
    if grep -q "changeme\|password123" .env.local 2>/dev/null; then
        log_warning "Weak passwords detected, generating secure ones..."
        
        # Generate secure passwords
        local new_postgres_pass=$(openssl rand -hex 32)
        local new_hasura_secret=$(openssl rand -hex 32)
        local new_jwt_secret=$(openssl rand -hex 32)
        
        set_env_var "POSTGRES_PASSWORD" "$new_postgres_pass" ".env.local"
        set_env_var "HASURA_GRAPHQL_ADMIN_SECRET" "$new_hasura_secret" ".env.local"
        set_env_var "JWT_SECRET" "$new_jwt_secret" ".env.local"
        set_env_var "COOKIE_SECRET" "$(openssl rand -hex 32)" ".env.local"
        
        log_success "Generated secure secrets"
    fi
    
    # Create production docker compose override
    create_prod_compose
    
    log_success "Production configuration complete"
    echo
    echo "Next steps:"
    echo "  1. Review .env.local for production settings"
    echo "  2. Set up SSL certificates:"
    echo "     - For Let's Encrypt: Ensure ports 80/443 are accessible"
    echo "     - For custom certs: Place in nginx/ssl/"
    echo "  3. Run: nself build"
    echo "  4. Run: nself start"
    echo
    log_warning "Remember to:"
    echo "  - Configure DNS to point to your server"
    echo "  - Open firewall ports 80 and 443"
    echo "  - Set up monitoring and backups"
    
    return 0
}

# Create production compose override
create_prod_compose() {
    cat > docker-compose.prod.yml << 'EOF'
services:
  nginx:
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
  
  postgres:
    restart: always
    volumes:
      - postgres_data:/var/lib/postgresql/data
  
  hasura:
    restart: always
    environment:
      HASURA_GRAPHQL_DEV_MODE: "false"
      HASURA_GRAPHQL_ENABLE_CONSOLE: "false"
  
  minio:
    restart: always
    volumes:
      - minio_data:/data

volumes:
  postgres_data:
  minio_data:
EOF
    
    log_info "Created docker-compose.prod.yml"
}

# Show help
show_prod_help() {
    echo "Usage: nself prod <domain> [email]"
    echo
    echo "Configure project for production deployment"
    echo
    echo "Arguments:"
    echo "  domain    Production domain (e.g., example.com)"
    echo "  email     SSL certificate email (optional)"
    echo
    echo "Examples:"
    echo "  nself prod example.com"
    echo "  nself prod example.com admin@example.com"
    echo
    echo "This will:"
    echo "  - Set production domain"
    echo "  - Enable SSL"
    echo "  - Generate secure passwords"
    echo "  - Create production compose file"
}

# Export for use as library
export -f cmd_prod

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_prod "$@"
fi