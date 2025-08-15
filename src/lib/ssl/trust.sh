#!/usr/bin/env bash
# trust.sh - OS trust store management for SSL certificates

set -euo pipefail

# Get the directory where this script is located
TRUST_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$TRUST_LIB_DIR/../.." && pwd)"
NSELF_BIN_DIR="${HOME}/.nself/bin"

# Source utilities
source "$TRUST_LIB_DIR/../utils/display.sh" 2>/dev/null || true
source "$TRUST_LIB_DIR/ssl.sh" 2>/dev/null || true

# Install root CA to system trust stores
trust::install_root_ca() {
    local mkcert_cmd
    
    if ! mkcert_cmd="$(ssl::get_mkcert 2>/dev/null)"; then
        log_error "mkcert not available. Run 'nself ssl bootstrap' first."
        return 1
    fi
    
    log_info "Installing root CA to system trust stores..."
    
    # Run mkcert -install
    if $mkcert_cmd -install; then
        log_success "Root CA installed successfully"
        
        # Platform-specific success messages
        local os="$(uname -s)"
        case "$os" in
            Darwin)
                log_info "✓ Added to macOS Keychain"
                log_info "✓ Added to Firefox (if installed)"
                ;;
            Linux)
                log_info "✓ Added to system certificate store"
                log_info "✓ Added to Firefox/Chrome NSS database (if available)"
                ;;
        esac
        
        return 0
    else
        log_error "Failed to install root CA"
        return 1
    fi
}

# Install PFX certificate on Windows
trust::install_pfx_windows() {
    local pfx_file="$1"
    
    if [[ ! -f "$pfx_file" ]]; then
        log_error "PFX file not found: $pfx_file"
        return 1
    fi
    
    log_info "Installing certificate to Windows certificate store..."
    
    # Use certutil to import the certificate
    if command -v certutil &> /dev/null; then
        if certutil -user -importpfx "$pfx_file" NoRoot 2>/dev/null; then
            log_success "Certificate imported to Windows store"
            return 0
        fi
    fi
    
    # Fallback to PowerShell if available
    if command -v powershell &> /dev/null; then
        local ps_script="Import-PfxCertificate -FilePath '$pfx_file' -CertStoreLocation Cert:\CurrentUser\My"
        if powershell -Command "$ps_script" 2>/dev/null; then
            log_success "Certificate imported to Windows store"
            return 0
        fi
    fi
    
    log_warning "Could not automatically import certificate"
    log_info "Please manually import: $pfx_file"
    log_info "Double-click the file and follow the import wizard"
    return 1
}

# Uninstall root CA from system
trust::uninstall_root_ca() {
    local mkcert_cmd
    
    if ! mkcert_cmd="$(ssl::get_mkcert 2>/dev/null)"; then
        log_warning "mkcert not available"
        return 1
    fi
    
    log_info "Uninstalling root CA from system trust stores..."
    
    if $mkcert_cmd -uninstall; then
        log_success "Root CA uninstalled successfully"
        return 0
    else
        log_error "Failed to uninstall root CA"
        return 1
    fi
}

# Check trust status
trust::status() {
    local mkcert_cmd
    
    echo "Trust Store Status:"
    echo "==================="
    echo
    
    # Check mkcert root CA
    if mkcert_cmd="$(ssl::get_mkcert 2>/dev/null)"; then
        echo "mkcert Root CA:"
        
        local ca_root="$($mkcert_cmd -CAROOT 2>/dev/null)"
        if [[ -n "$ca_root" ]]; then
            echo "  Location: $ca_root"
            
            if [[ -f "$ca_root/rootCA.pem" ]]; then
                echo "  ✓ Root CA certificate exists"
                
                # Check if installed
                if $mkcert_cmd -install -check 2>/dev/null; then
                    echo "  ✓ Installed in system trust store"
                else
                    echo "  ✗ Not installed in system (run 'nself trust')"
                fi
                
                # Show certificate details
                local ca_subject=$(openssl x509 -in "$ca_root/rootCA.pem" -noout -subject 2>/dev/null | sed 's/subject=//')
                local ca_expiry=$(openssl x509 -in "$ca_root/rootCA.pem" -noout -enddate 2>/dev/null | cut -d= -f2)
                echo "  Subject: $ca_subject"
                echo "  Expires: $ca_expiry"
            else
                echo "  ✗ Root CA certificate not found"
            fi
        else
            echo "  ✗ mkcert not initialized"
        fi
    else
        echo "  ✗ mkcert not installed"
    fi
    echo
    
    # Platform-specific trust store info
    local os="$(uname -s)"
    case "$os" in
        Darwin)
            echo "Platform: macOS"
            echo "  Trust stores: System Keychain, Firefox NSS"
            # Check if certificate is in keychain
            if security find-certificate -c "mkcert" &>/dev/null; then
                echo "  ✓ mkcert certificate found in Keychain"
            fi
            ;;
        Linux)
            echo "Platform: Linux"
            echo "  Trust stores: /usr/local/share/ca-certificates, Firefox/Chrome NSS"
            # Check common certificate locations
            if [[ -f "/usr/local/share/ca-certificates/mkcert-rootCA.crt" ]] || \
               [[ -f "/etc/pki/ca-trust/source/anchors/mkcert-rootCA.pem" ]]; then
                echo "  ✓ mkcert certificate found in system store"
            fi
            ;;
    esac
    echo
    
    # Check for PFX files
    local pfx_file="$NSELF_ROOT/templates/certs/nself-org/wildcard.pfx"
    if [[ -f "$pfx_file" ]]; then
        echo "Windows Certificate (PFX):"
        echo "  ✓ PFX bundle available at: $pfx_file"
        echo "  Import manually on Windows for *.local.nself.org"
    fi
}

# Main trust installation function
trust::install() {
    log_info "Setting up certificate trust..."
    
    # Install mkcert root CA
    if ! trust::install_root_ca; then
        return 1
    fi
    
    # Check for Windows and PFX
    if [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        local pfx_file="$NSELF_ROOT/templates/certs/nself-org/wildcard.pfx"
        if [[ -f "$pfx_file" ]]; then
            log_info "Found PFX certificate for Windows"
            trust::install_pfx_windows "$pfx_file"
        fi
    fi
    
    echo
    log_success "Certificate trust setup complete!"
    log_info "Your browser will now trust locally-generated certificates"
    
    # Show what domains are trusted
    echo
    echo "Trusted domains:"
    echo "  • localhost, *.localhost"
    echo "  • 127.0.0.1, ::1"
    
    if [[ -f "$NSELF_ROOT/templates/certs/nself-org/fullchain.pem" ]]; then
        local issuer=$(openssl x509 -in "$NSELF_ROOT/templates/certs/nself-org/fullchain.pem" -noout -issuer 2>/dev/null | sed 's/issuer=//')
        if [[ "$issuer" != *"Let's Encrypt"* ]]; then
            echo "  • local.nself.org, *.local.nself.org"
        fi
    fi
    
    echo
    log_info "You may need to restart your browser for changes to take effect"
    
    return 0
}

# Export functions
export -f trust::install_root_ca
export -f trust::install_pfx_windows
export -f trust::uninstall_root_ca
export -f trust::status
export -f trust::install