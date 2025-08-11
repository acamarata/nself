#!/usr/bin/env bash
# init.sh - Initialize a new project

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/validation.sh"

# Command function
cmd_init() {
    local project_name="${1:-myproject}"
    local domain="${2:-localhost}"
    
    # Safety check: Don't run in nself repository
    if [[ -f "bin/nself" ]] && [[ -d "src/lib" ]] && [[ -d "docs" ]]; then
        log_error "Cannot initialize in the nself repository!"
        echo ""
        log_info "Please create a separate project directory:"
        log_info "  mkdir ~/myproject && cd ~/myproject"
        log_info "  nself init"
        return 1
    fi
    
    show_header "NSELF PROJECT INITIALIZATION"
    
    # Check if already initialized
    if [[ -f ".env.local" ]]; then
        log_warning "Project already initialized (.env.local exists)"
        if ! confirm_with_timeout "Reinitialize project?" 10 "n"; then
            return 1
        fi
        cp .env.local .env.local.backup
        log_info "Backed up existing configuration to .env.local.backup"
    fi
    
    log_info "Initializing project: $project_name"
    log_info "Domain: $domain"
    
    # Create environment file
    create_env_file "$project_name" "$domain"
    
    # Create directory structure
    create_project_structure
    
    # Generate secure secrets
    generate_secrets
    
    log_success "Project initialized successfully!"
    echo
    echo "Next steps:"
    echo "  1. Review and customize .env.local"
    echo "  2. Run: nself build"
    echo "  3. Run: nself up"
    
    return 0
}

# Create environment file
create_env_file() {
    local project_name="$1"
    local domain="$2"
    
    log_info "Creating environment configuration..."
    
    cat > .env.local << EOF
# NSELF Project Configuration
# Generated: $(date)

# Project Settings
PROJECT_NAME=$project_name
BASE_DOMAIN=$domain

# Database Configuration
POSTGRES_PASSWORD=$(generate_password)
POSTGRES_DB=$project_name
POSTGRES_USER=postgres
POSTGRES_PORT=5432

# Hasura Configuration
HASURA_GRAPHQL_ADMIN_SECRET=$(generate_password)
HASURA_GRAPHQL_DATABASE_URL=postgres://postgres:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}
HASURA_GRAPHQL_JWT_SECRET='{"type":"HS256","key":"$(generate_jwt_key)"}'
HASURA_GRAPHQL_UNAUTHORIZED_ROLE=anonymous
HASURA_PORT=8080

# Authentication
JWT_SECRET=$(generate_jwt_key)
COOKIE_SECRET=$(generate_password)
AUTH_PORT=4000

# Storage Configuration
S3_ACCESS_KEY=$(generate_password 16)
S3_SECRET_KEY=$(generate_password 32)
S3_BUCKET=$project_name
S3_REGION=us-east-1
S3_ENDPOINT=http://minio:9000
MINIO_PORT=9000

# Email Configuration
EMAIL_PROVIDER=development
SMTP_HOST=mailhog
SMTP_PORT=1025
SMTP_FROM=noreply@$domain

# Redis Configuration (optional)
REDIS_ENABLED=false
REDIS_PASSWORD=$(generate_password 16)
REDIS_PORT=6379

# SSL Configuration
SSL_ENABLED=false
SSL_EMAIL=admin@$domain

# Service Configuration
ENABLE_NESTJS=false
ENABLE_BULLMQ=false
ENABLE_GO=false
ENABLE_PYTHON=false

# Development Settings
DEBUG=false
VERBOSE=false
EOF
    
    log_success "Created .env.local"
}

# Create project structure
create_project_structure() {
    log_info "Creating project structure..."
    
    local dirs=(
        "hasura/migrations"
        "hasura/metadata"
        "hasura/seeds"
        "nginx/conf.d"
        "nginx/ssl"
        "postgres/init"
        "services/nest"
        "services/go"
        "services/py"
        "services/bullmq"
        "logs"
        "data"
        "backups"
    )
    
    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
    done
    
    # Create default nginx config
    if [[ ! -f "nginx/nginx.conf" ]]; then
        create_nginx_config
    fi
    
    # Create default schema
    if [[ ! -f "schema.dbml" ]]; then
        create_default_schema
    fi
    
    log_success "Created project structure"
}

# Generate secure password
generate_password() {
    local length="${1:-32}"
    
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex "$length" | cut -c1-"$length"
    else
        # Fallback to urandom
        tr -dc 'a-zA-Z0-9' < /dev/urandom | head -c "$length"
    fi
}

# Generate JWT key
generate_jwt_key() {
    # Generate 32-character key for HS256
    generate_password 32
}

# Generate all secrets
generate_secrets() {
    log_info "Generating secure secrets..."
    
    # Update weak passwords if they exist
    if grep -q "changeme\|password123" .env.local 2>/dev/null; then
        log_warning "Weak passwords detected, regenerating..."
        
        # Replace weak passwords with secure ones
        sed -i.bak \
            -e "s/POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$(generate_password)/" \
            -e "s/HASURA_GRAPHQL_ADMIN_SECRET=.*/HASURA_GRAPHQL_ADMIN_SECRET=$(generate_password)/" \
            -e "s/JWT_SECRET=.*/JWT_SECRET=$(generate_jwt_key)/" \
            -e "s/COOKIE_SECRET=.*/COOKIE_SECRET=$(generate_password)/" \
            .env.local
        
        log_success "Replaced weak passwords with secure ones"
    fi
}

# Create default nginx config
create_nginx_config() {
    cat > nginx/nginx.conf << 'EOF'
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    
    access_log /var/log/nginx/access.log main;
    
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    
    include /etc/nginx/conf.d/*.conf;
}
EOF
    
    log_info "Created nginx configuration"
}

# Create default database schema
create_default_schema() {
    cat > schema.dbml << 'EOF'
// NSELF Database Schema
// Documentation: https://dbml.dbdiagram.io/docs

Project nself {
  database_type: 'PostgreSQL'
  note: 'Database schema for NSELF project'
}

Table users {
  id uuid [pk, default: `gen_random_uuid()`]
  email varchar [unique, not null]
  username varchar [unique]
  password_hash varchar
  created_at timestamp [default: `now()`]
  updated_at timestamp [default: `now()`]
  
  indexes {
    email
    username
  }
}

Table profiles {
  id uuid [pk, default: `gen_random_uuid()`]
  user_id uuid [ref: - users.id, not null]
  full_name varchar
  avatar_url varchar
  bio text
  created_at timestamp [default: `now()`]
  updated_at timestamp [default: `now()`]
}

Table sessions {
  id uuid [pk, default: `gen_random_uuid()`]
  user_id uuid [ref: > users.id, not null]
  token varchar [unique, not null]
  expires_at timestamp [not null]
  created_at timestamp [default: `now()`]
  
  indexes {
    token
    user_id
  }
}
EOF
    
    log_info "Created default database schema"
}

# Export main function
export -f cmd_init