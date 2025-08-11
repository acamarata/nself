#!/usr/bin/env bash
# trust.sh - Trust and install SSL certificates

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
# Helper functions




# Command function
cmd_trust() {
    local action="${1:-install}"
    
    if [[ "$action" == "--help" ]] || [[ "$action" == "-h" ]]; then
        show_trust_help
        return 0
    fi
    
    # Load environment
    if [[ -f ".env.local" ]]; then
        load_env_safe ".env.local"
    else
        log_error "No .env.local found. Run 'nself init' first"
        return 1
    fi
    
    local base_domain="${BASE_DOMAIN:-localhost}"
    
    case "$action" in
        install)
            install_certificates "$base_domain"
            ;;
        uninstall|remove)
            uninstall_certificates "$base_domain"
            ;;
        verify|check)
            verify_certificates "$base_domain"
            ;;
        *)
            log_error "Unknown action: $action"
            show_trust_help
            return 1
            ;;
    esac
}

# Install SSL certificates
install_certificates() {
    local domain="$1"
    
    log_info "Installing SSL certificates for $domain"
    
    # Check for mkcert
    local mkcert_path="$SCRIPT_DIR/../tools/ssl/mkcert"
    if [[ ! -f "$mkcert_path" ]]; then
        log_error "mkcert not found. Run 'nself init' first"
        return 1
    fi
    
    # Create nginx/ssl directory if it doesn't exist
    mkdir -p nginx/ssl
    
    # Install root CA
    log_info "Installing root certificate authority..."
    if "$mkcert_path" -install; then
        log_success "Root CA installed"
    else
        log_warning "Could not install root CA automatically"
        log_info "You may need to manually trust the certificate"
    fi
    
    # Generate certificates
    log_info "Generating certificates for $domain and subdomains..."
    cd nginx/ssl
    
    # Generate certificates for all subdomains
    "$mkcert_path" \
        "$domain" \
        "*.$domain" \
        "api.$domain" \
        "auth.$domain" \
        "hasura.$domain" \
        "mail.$domain" \
        "files.$domain" \
        "s3.$domain" \
        "localhost" \
        "127.0.0.1" \
        "::1"
    
    if [[ $? -eq 0 ]]; then
        # Rename certificate files to standard names
        mv "${domain}+10.pem" "cert.pem" 2>/dev/null || true
        mv "${domain}+10-key.pem" "key.pem" 2>/dev/null || true
        
        log_success "Certificates generated successfully"
        
        # Set proper permissions
        chmod 644 cert.pem
        chmod 600 key.pem
        
        log_info "Certificate files:"
        echo "  - nginx/ssl/cert.pem"
        echo "  - nginx/ssl/key.pem"
        
        # Trust on macOS if available
        if [[ "$OSTYPE" == "darwin"* ]]; then
            log_info "Trusting certificate in macOS Keychain..."
            if security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain cert.pem 2>/dev/null; then
                log_success "Certificate trusted in macOS Keychain"
            else
                log_info "Run with sudo to trust in system keychain:"
                echo "  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain nginx/ssl/cert.pem"
            fi
        fi
        
        cd ../..
        return 0
    else
        log_error "Failed to generate certificates"
        cd ../..
        return 1
    fi
}

# Uninstall SSL certificates
uninstall_certificates() {
    local domain="$1"
    
    log_info "Removing SSL certificates for $domain"
    
    # Remove certificate files
    if [[ -d "nginx/ssl" ]]; then
        rm -f nginx/ssl/cert.pem nginx/ssl/key.pem
        log_success "Certificate files removed"
    fi
    
    # Uninstall mkcert root CA
    local mkcert_path="$SCRIPT_DIR/../tools/ssl/mkcert"
    if [[ -f "$mkcert_path" ]]; then
        log_info "Removing root certificate authority..."
        "$mkcert_path" -uninstall
        log_success "Root CA removed"
    fi
    
    # Remove from macOS Keychain if on Mac
    if [[ "$OSTYPE" == "darwin"* ]]; then
        log_info "Removing from macOS Keychain..."
        security delete-certificate -c "$domain" 2>/dev/null || true
        security delete-certificate -c "mkcert*" 2>/dev/null || true
    fi
    
    log_success "Certificates uninstalled"
}

# Verify SSL certificates
verify_certificates() {
    local domain="$1"
    
    log_info "Verifying SSL certificates for $domain"
    
    # Check if certificate files exist
    if [[ ! -f "nginx/ssl/cert.pem" ]] || [[ ! -f "nginx/ssl/key.pem" ]]; then
        log_error "Certificate files not found"
        log_info "Run 'nself trust install' to generate certificates"
        return 1
    fi
    
    # Verify certificate
    log_info "Certificate details:"
    openssl x509 -in nginx/ssl/cert.pem -noout -text | grep -E "(Subject:|DNS:|Not After)"
    
    # Check if certificate is valid
    if openssl x509 -in nginx/ssl/cert.pem -noout -checkend 0; then
        log_success "Certificate is valid"
    else
        log_error "Certificate has expired"
        log_info "Run 'nself trust install' to regenerate"
        return 1
    fi
    
    # Check trust status on macOS
    if [[ "$OSTYPE" == "darwin"* ]]; then
        log_info "Checking macOS Keychain trust..."
        if security verify-cert -c nginx/ssl/cert.pem 2>/dev/null; then
            log_success "Certificate is trusted in macOS Keychain"
        else
            log_warning "Certificate is not trusted in macOS Keychain"
            log_info "Run 'sudo nself trust install' to trust it"
        fi
    fi
    
    # Test HTTPS connectivity if services are running
    if docker ps --format "{{.Names}}" | grep -q "nginx"; then
        log_info "Testing HTTPS connectivity..."
        if curl -k -s -o /dev/null -w "%{http_code}" "https://$domain" | grep -q "200\|301\|302"; then
            log_success "HTTPS is working"
        else
            log_warning "HTTPS connection failed (services may not be fully up)"
        fi
    else
        log_info "Services not running. Start with 'nself up' to test HTTPS"
    fi
    
    return 0
}

# Show help
show_trust_help() {
    echo "Usage: nself trust [action]"
    echo
    echo "Manage SSL certificates for local development"
    echo
    echo "Actions:"
    echo "  install    Install and trust SSL certificates (default)"
    echo "  uninstall  Remove SSL certificates and trust"
    echo "  verify     Check certificate status and validity"
    echo
    echo "Examples:"
    echo "  nself trust              # Install certificates"
    echo "  nself trust install      # Install certificates"
    echo "  nself trust verify       # Check certificate status"
    echo "  nself trust uninstall    # Remove certificates"
    echo
    echo "Note: On macOS, you may need to run with sudo to trust"
    echo "certificates in the system keychain."
}

# Export for use as library
export -f cmd_trust

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_trust "$@"
fi