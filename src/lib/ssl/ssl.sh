#!/usr/bin/env bash
# ssl.sh - Core SSL certificate management functions

set -euo pipefail

# Get the directory where this script is located
SSL_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$SSL_LIB_DIR/../.." && pwd)"
TEMPLATES_DIR="$NSELF_ROOT/templates"
CERTS_DIR="$TEMPLATES_DIR/certs"
NSELF_BIN_DIR="${HOME}/.nself/bin"

# Source utilities
source "$SSL_LIB_DIR/../utils/display.sh" 2>/dev/null || true

# Default configuration
SSL_NSELF_ORG_DOMAIN="${SSL_NSELF_ORG_DOMAIN:-local.nself.org}"
SSL_LOCALHOST_NAMES="${SSL_LOCALHOST_NAMES:-localhost,*.localhost,127.0.0.1,::1}"
SSL_PUBLIC_WILDCARD="${SSL_PUBLIC_WILDCARD:-true}"
SSL_FALLBACK_LOCALHOST="${SSL_FALLBACK_LOCALHOST:-true}"

# Ensure required tools are available
ssl::ensure_tools() {
    local missing_tools=()
    
    # Check for docker
    if ! command -v docker &> /dev/null; then
        missing_tools+=("docker")
    fi
    
    # Check for openssl
    if ! command -v openssl &> /dev/null; then
        missing_tools+=("openssl")
    fi
    
    # If localhost fallback is enabled, ensure mkcert
    if [[ "$SSL_FALLBACK_LOCALHOST" == "true" ]] || [[ -z "${DNS_PROVIDER:-}" ]]; then
        if ! command -v mkcert &> /dev/null && [[ ! -f "$NSELF_BIN_DIR/mkcert" ]]; then
            log_info "Installing mkcert for local certificate generation..."
            ssl::install_mkcert
        fi
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Please install the missing tools and try again"
        return 1
    fi
    
    return 0
}

# Install mkcert binary
ssl::install_mkcert() {
    mkdir -p "$NSELF_BIN_DIR"
    
    local os arch mkcert_url
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"
    
    # Map architecture names
    case "$arch" in
        x86_64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) log_error "Unsupported architecture: $arch"; return 1 ;;
    esac
    
    # Construct download URL
    local mkcert_version="v1.4.4"
    case "$os" in
        darwin)
            mkcert_url="https://github.com/FiloSottile/mkcert/releases/download/${mkcert_version}/mkcert-${mkcert_version}-${os}-${arch}"
            ;;
        linux)
            mkcert_url="https://github.com/FiloSottile/mkcert/releases/download/${mkcert_version}/mkcert-${mkcert_version}-${os}-${arch}"
            ;;
        *)
            log_error "Unsupported OS: $os"
            return 1
            ;;
    esac
    
    log_info "Downloading mkcert from $mkcert_url..."
    if curl -L -o "$NSELF_BIN_DIR/mkcert" "$mkcert_url"; then
        chmod +x "$NSELF_BIN_DIR/mkcert"
        export PATH="$NSELF_BIN_DIR:$PATH"
        log_success "mkcert installed successfully"
    else
        log_error "Failed to download mkcert"
        return 1
    fi
}

# Get mkcert command (either from PATH or our bin dir)
ssl::get_mkcert() {
    if command -v mkcert &> /dev/null; then
        echo "mkcert"
    elif [[ -f "$NSELF_BIN_DIR/mkcert" ]]; then
        echo "$NSELF_BIN_DIR/mkcert"
    else
        return 1
    fi
}

# Issue public wildcard certificate via Let's Encrypt
ssl::issue_public_wildcard() {
    local domain="${SSL_NSELF_ORG_DOMAIN}"
    local provider="${DNS_PROVIDER:-}"
    local output_dir="$CERTS_DIR/nself-org"
    
    if [[ -z "$provider" ]]; then
        log_warning "No DNS provider configured, skipping public wildcard"
        return 1
    fi
    
    log_info "Issuing Let's Encrypt wildcard certificate for *.$domain..."
    
    # Prepare acme.sh working directory
    local acme_dir="$HOME/.nself/acme"
    mkdir -p "$acme_dir"
    
    # Build docker command based on provider
    local docker_env_args=()
    case "$provider" in
        cloudflare)
            docker_env_args+=("-e" "CF_Token=${DNS_API_TOKEN:-}")
            docker_env_args+=("-e" "CF_Email=${CF_EMAIL:-}")
            ;;
        route53)
            docker_env_args+=("-e" "AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID:-}")
            docker_env_args+=("-e" "AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY:-}")
            ;;
        digitalocean)
            docker_env_args+=("-e" "DO_API_KEY=${DO_API_KEY:-}")
            ;;
        *)
            log_error "Unsupported DNS provider: $provider"
            return 1
            ;;
    esac
    
    # Run acme.sh in docker
    if docker run --rm \
        "${docker_env_args[@]}" \
        -v "$acme_dir:/acme.sh" \
        neilpang/acme.sh:latest \
        --issue --dns "dns_$provider" \
        -d "*.$domain" \
        -d "$domain" \
        --keylength ec-256 \
        --server letsencrypt; then
        
        # Copy certificates to our template directory
        mkdir -p "$output_dir"
        
        local cert_dir="$acme_dir/${domain}_ecc"
        if [[ -d "$cert_dir" ]]; then
            cp "$cert_dir/fullchain.cer" "$output_dir/fullchain.pem"
            cp "$cert_dir/${domain}.key" "$output_dir/privkey.pem"
            cp "$cert_dir/${domain}.cer" "$output_dir/cert.pem"
            
            # Create PFX for Windows
            openssl pkcs12 -export \
                -out "$output_dir/wildcard.pfx" \
                -inkey "$output_dir/privkey.pem" \
                -in "$output_dir/fullchain.pem" \
                -passout pass: 2>/dev/null || true
            
            log_success "Public wildcard certificate issued successfully"
            return 0
        fi
    fi
    
    log_error "Failed to issue public wildcard certificate"
    return 1
}

# Issue internal certificate for *.local.nself.org using mkcert
ssl::issue_internal_nself_org() {
    local domain="${SSL_NSELF_ORG_DOMAIN}"
    local output_dir="$CERTS_DIR/nself-org"
    local mkcert_cmd
    
    if ! mkcert_cmd="$(ssl::get_mkcert)"; then
        log_error "mkcert not available"
        return 1
    fi
    
    log_info "Generating internal certificate for *.$domain..."
    
    mkdir -p "$output_dir"
    
    # Generate certificate
    if $mkcert_cmd \
        -cert-file "$output_dir/cert.pem" \
        -key-file "$output_dir/privkey.pem" \
        "$domain" "*.$domain"; then
        
        # mkcert doesn't create a fullchain, so we'll use cert as fullchain
        cp "$output_dir/cert.pem" "$output_dir/fullchain.pem"
        
        # Create PFX for Windows
        openssl pkcs12 -export \
            -out "$output_dir/wildcard.pfx" \
            -inkey "$output_dir/privkey.pem" \
            -in "$output_dir/cert.pem" \
            -passout pass: 2>/dev/null || true
        
        log_success "Internal wildcard certificate generated successfully"
        log_warning "Run 'nself trust' to install the root CA in your system"
        return 0
    fi
    
    log_error "Failed to generate internal certificate"
    return 1
}

# Issue localhost certificate bundle using mkcert
ssl::issue_localhost_bundle() {
    local output_dir="$CERTS_DIR/localhost"
    local mkcert_cmd
    
    if ! mkcert_cmd="$(ssl::get_mkcert)"; then
        log_error "mkcert not available"
        return 1
    fi
    
    log_info "Generating localhost certificate bundle..."
    
    # Install mkcert root CA if not already done
    $mkcert_cmd -install 2>/dev/null || true
    
    mkdir -p "$output_dir"
    
    # Parse comma-separated names
    local names=()
    IFS=',' read -ra names <<< "$SSL_LOCALHOST_NAMES"
    
    # Generate certificate
    if $mkcert_cmd \
        -cert-file "$output_dir/cert.pem" \
        -key-file "$output_dir/privkey.pem" \
        "${names[@]}"; then
        
        # mkcert doesn't create a fullchain, so we'll use cert as fullchain
        cp "$output_dir/cert.pem" "$output_dir/fullchain.pem"
        
        log_success "Localhost certificate bundle generated successfully"
        return 0
    fi
    
    log_error "Failed to generate localhost certificate"
    return 1
}

# Copy certificates into project
ssl::copy_into_project() {
    local project_dir="${1:-.}"
    local nginx_ssl_dir="$project_dir/nginx/ssl"
    
    log_info "Copying certificates to project..."
    
    # Create target directories
    mkdir -p "$nginx_ssl_dir"/{localhost,nself-org}
    
    # Copy localhost certificates if they exist
    if [[ -f "$CERTS_DIR/localhost/fullchain.pem" ]]; then
        cp "$CERTS_DIR/localhost/fullchain.pem" "$nginx_ssl_dir/localhost/"
        cp "$CERTS_DIR/localhost/privkey.pem" "$nginx_ssl_dir/localhost/"
        chmod 600 "$nginx_ssl_dir/localhost/privkey.pem"
        log_success "Copied localhost certificates"
    fi
    
    # Copy nself.org certificates if they exist
    if [[ -f "$CERTS_DIR/nself-org/fullchain.pem" ]]; then
        cp "$CERTS_DIR/nself-org/fullchain.pem" "$nginx_ssl_dir/nself-org/"
        cp "$CERTS_DIR/nself-org/privkey.pem" "$nginx_ssl_dir/nself-org/"
        chmod 600 "$nginx_ssl_dir/nself-org/privkey.pem"
        
        # Copy PFX if it exists
        if [[ -f "$CERTS_DIR/nself-org/wildcard.pfx" ]]; then
            cp "$CERTS_DIR/nself-org/wildcard.pfx" "$nginx_ssl_dir/nself-org/"
        fi
        
        log_success "Copied *.local.nself.org certificates"
    fi
}

# Render nginx SSL configuration snippets
ssl::render_nginx_snippets() {
    local project_dir="${1:-.}"
    local nginx_conf_dir="$project_dir/nginx/conf.d"
    
    log_info "Generating nginx SSL configurations..."
    
    mkdir -p "$nginx_conf_dir"
    
    # Generate SSL configuration for *.local.nself.org
    if [[ -f "$project_dir/nginx/ssl/nself-org/fullchain.pem" ]]; then
        cat > "$nginx_conf_dir/ssl-local-nself-org.conf" << 'EOF'
# Wildcard SSL for *.local.nself.org
server {
    listen 443 ssl;
    http2 on;
    server_name ~^(?<subdomain>.+)\.local\.nself\.org$;
    
    ssl_certificate     /etc/nginx/ssl/nself-org/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/nself-org/privkey.pem;
    
    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    
    # SSL session caching
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:10m;
    ssl_session_tickets off;
    
    # Disable HSTS in development to avoid pinning
    # add_header Strict-Transport-Security "max-age=0";
    
    # Route to appropriate backend based on subdomain
    location / {
        # Default proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Return 404 for unmapped subdomains
        return 404;
    }
}

# HTTP to HTTPS redirect for *.local.nself.org
server {
    listen 80;
    server_name ~^(?<subdomain>.+)\.local\.nself\.org$;
    return 301 https://$host$request_uri;
}
EOF
        log_success "Generated SSL configuration for *.local.nself.org"
    fi
    
    # Generate SSL configuration for localhost
    if [[ -f "$project_dir/nginx/ssl/localhost/fullchain.pem" ]]; then
        cat > "$nginx_conf_dir/ssl-localhost.conf" << 'EOF'
# SSL for localhost and *.localhost
server {
    listen 443 ssl;
    http2 on;
    server_name localhost *.localhost;
    
    ssl_certificate     /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;
    
    # Modern SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    
    # SSL session caching
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:10m;
    ssl_session_tickets off;
    
    # Default route to main application
    location / {
        proxy_pass http://hasura:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# HTTP to HTTPS redirect for localhost
server {
    listen 80;
    server_name localhost *.localhost;
    return 301 https://$host$request_uri;
}
EOF
        log_success "Generated SSL configuration for localhost"
    fi
    
    # Create routes directory for subdomain-specific configs
    mkdir -p "$nginx_conf_dir/routes"
    
    # Create default route files for common subdomains
    local subdomains=("api" "auth" "storage" "functions" "dashboard")
    for subdomain in "${subdomains[@]}"; do
        ssl::create_route_config "$nginx_conf_dir/routes" "$subdomain"
    done
}

# Create route configuration for a subdomain
ssl::create_route_config() {
    local routes_dir="$1"
    local subdomain="$2"
    
    case "$subdomain" in
        api)
            cat > "$routes_dir/api.conf" << 'EOF'
proxy_pass http://hasura:8080;
EOF
            ;;
        auth)
            cat > "$routes_dir/auth.conf" << 'EOF'
proxy_pass http://auth:4000;
EOF
            ;;
        storage)
            cat > "$routes_dir/storage.conf" << 'EOF'
proxy_pass http://storage:5001;
client_max_body_size 1000m;
EOF
            ;;
        functions)
            cat > "$routes_dir/functions.conf" << 'EOF'
proxy_pass http://unity-functions:4300;
EOF
            ;;
        dashboard)
            cat > "$routes_dir/dashboard.conf" << 'EOF'
proxy_pass http://unity-dashboard:80;
EOF
            ;;
    esac
}

# Check certificate status
ssl::status() {
    echo "SSL Certificate Status:"
    echo "======================="
    echo
    
    # Check localhost certificates
    echo "Localhost Certificates:"
    if [[ -f "$CERTS_DIR/localhost/fullchain.pem" ]]; then
        local expiry=$(openssl x509 -in "$CERTS_DIR/localhost/fullchain.pem" -noout -enddate 2>/dev/null | cut -d= -f2)
        echo "  ✓ Certificate exists (expires: $expiry)"
    else
        echo "  ✗ No certificate found"
    fi
    echo
    
    # Check nself.org certificates
    echo "*.local.nself.org Certificates:"
    if [[ -f "$CERTS_DIR/nself-org/fullchain.pem" ]]; then
        local expiry=$(openssl x509 -in "$CERTS_DIR/nself-org/fullchain.pem" -noout -enddate 2>/dev/null | cut -d= -f2)
        local issuer=$(openssl x509 -in "$CERTS_DIR/nself-org/fullchain.pem" -noout -issuer 2>/dev/null | sed 's/issuer=//')
        echo "  ✓ Certificate exists"
        echo "    Issuer: $issuer"
        echo "    Expires: $expiry"
        
        if [[ "$issuer" == *"Let's Encrypt"* ]]; then
            echo "    Type: Public (Let's Encrypt)"
        else
            echo "    Type: Internal (mkcert)"
        fi
    else
        echo "  ✗ No certificate found"
    fi
    echo
    
    # Check mkcert installation
    echo "mkcert Status:"
    if mkcert_cmd="$(ssl::get_mkcert 2>/dev/null)"; then
        echo "  ✓ mkcert installed at: $mkcert_cmd"
        if $mkcert_cmd -install -check 2>/dev/null; then
            echo "  ✓ Root CA is installed in system"
        else
            echo "  ✗ Root CA not installed (run 'nself trust')"
        fi
    else
        echo "  ✗ mkcert not installed"
    fi
}

# Export functions
export -f ssl::ensure_tools
export -f ssl::install_mkcert
export -f ssl::get_mkcert
export -f ssl::issue_public_wildcard
export -f ssl::issue_internal_nself_org
export -f ssl::issue_localhost_bundle
export -f ssl::copy_into_project
export -f ssl::render_nginx_snippets
export -f ssl::create_route_config
export -f ssl::status