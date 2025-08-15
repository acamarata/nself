#!/usr/bin/env bash
# ssl.sh - SSL certificate management commands

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/ssl/ssl.sh"
source "$SCRIPT_DIR/../lib/ssl/trust.sh"

# Command function
cmd_ssl() {
    local subcommand="${1:-bootstrap}"
    shift || true
    
    case "$subcommand" in
        bootstrap)
            ssl_bootstrap "$@"
            ;;
        renew)
            ssl_renew "$@"
            ;;
        status)
            ssl_status "$@"
            ;;
        help|--help|-h)
            show_ssl_help
            ;;
        *)
            log_error "Unknown subcommand: $subcommand"
            show_ssl_help
            return 1
            ;;
    esac
}

# Bootstrap SSL certificates
ssl_bootstrap() {
    show_command_header "nself ssl bootstrap" "Generate SSL certificates for development"
    
    # Load environment variables if .env.local exists
    if [[ -f ".env.local" ]]; then
        set -a
        source .env.local
        set +a
    fi
    
    # Ensure tools are available
    if ! ssl::ensure_tools; then
        return 1
    fi
    
    local success_count=0
    local total_count=0
    
    # Try public wildcard first if DNS provider is configured
    if [[ -n "${DNS_PROVIDER:-}" ]]; then
        ((total_count++))
        log_info "Attempting to issue public wildcard certificate..."
        if ssl::issue_public_wildcard; then
            ((success_count++))
            log_success "Public wildcard certificate issued for *.local.nself.org"
        else
            log_warning "Failed to issue public wildcard, falling back to internal certificate"
            if ssl::issue_internal_nself_org; then
                ((success_count++))
                log_success "Internal certificate generated for *.local.nself.org"
            fi
        fi
    else
        # No DNS provider, use internal certificate
        ((total_count++))
        log_info "No DNS provider configured, generating internal certificate..."
        if ssl::issue_internal_nself_org; then
            ((success_count++))
            log_success "Internal certificate generated for *.local.nself.org"
        fi
    fi
    
    # Generate localhost certificates if fallback is enabled
    if [[ "${SSL_FALLBACK_LOCALHOST:-true}" == "true" ]]; then
        ((total_count++))
        log_info "Generating localhost certificates..."
        if ssl::issue_localhost_bundle; then
            ((success_count++))
            log_success "Localhost certificates generated"
        fi
    fi
    
    # Copy certificates to project if we're in a project directory
    if [[ -f "docker-compose.yml" ]]; then
        ssl::copy_into_project "."
        ssl::render_nginx_snippets "."
    fi
    
    echo
    if [[ $success_count -eq $total_count ]]; then
        log_success "SSL bootstrap completed successfully ($success_count/$total_count)"
        
        # Check if root CA needs to be installed
        local mkcert_cmd
        if mkcert_cmd="$(ssl::get_mkcert 2>/dev/null)"; then
            if ! $mkcert_cmd -install -check 2>/dev/null; then
                echo
                log_warning "Root CA not installed in system trust store"
                log_info "Run 'nself trust' to install it and remove browser warnings"
            fi
        fi
    else
        log_warning "SSL bootstrap completed with issues ($success_count/$total_count successful)"
    fi
    
    return 0
}

# Renew SSL certificates
ssl_renew() {
    show_command_header "nself ssl renew" "Renew SSL certificates"
    
    # Load environment variables if .env.local exists
    if [[ -f ".env.local" ]]; then
        set -a
        source .env.local
        set +a
    fi
    
    # Only renew public certificates
    if [[ -n "${DNS_PROVIDER:-}" ]]; then
        log_info "Renewing public wildcard certificate..."
        if ssl::issue_public_wildcard; then
            log_success "Certificate renewed successfully"
            
            # Copy to project if in project directory
            if [[ -f "docker-compose.yml" ]]; then
                ssl::copy_into_project "."
                log_info "Updated certificates in project"
                log_info "Restart nginx to apply: docker compose restart nginx"
            fi
            
            return 0
        else
            log_error "Failed to renew certificate"
            return 1
        fi
    else
        log_info "No public certificate to renew (DNS provider not configured)"
        log_info "Internal certificates don't expire frequently and can be regenerated with 'nself ssl bootstrap'"
        return 0
    fi
}

# Show SSL status
ssl_status() {
    show_command_header "nself ssl status" "SSL certificate status and configuration"
    
    ssl::status
    echo
    trust::status
    
    return 0
}

# Show help
show_ssl_help() {
    echo "Usage: nself ssl <subcommand> [options]"
    echo ""
    echo "Manage SSL certificates for development"
    echo ""
    echo "Subcommands:"
    echo "  bootstrap    Generate SSL certificates (default)"
    echo "  renew        Renew public wildcard certificate"
    echo "  status       Show certificate status"
    echo "  help         Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  DNS_PROVIDER           DNS provider for Let's Encrypt (cloudflare, route53, digitalocean)"
    echo "  DNS_API_TOKEN          API token for DNS provider"
    echo "  SSL_PUBLIC_WILDCARD    Try to issue public wildcard (default: true)"
    echo "  SSL_FALLBACK_LOCALHOST Generate localhost certificates (default: true)"
    echo "  SSL_NSELF_ORG_DOMAIN   Domain for wildcard (default: local.nself.org)"
    echo ""
    echo "Examples:"
    echo "  nself ssl bootstrap    # Generate all certificates"
    echo "  nself ssl renew        # Renew public wildcard if configured"
    echo "  nself ssl status       # Show certificate details"
    echo ""
    echo "After bootstrap, run 'nself trust' to install root CA in your system"
}

# Export for use as library
export -f cmd_ssl

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_ssl "$@"
    exit $?
fi