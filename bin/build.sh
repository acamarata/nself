#!/bin/bash

# build.sh - Build project structure from environment configuration

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATES_DIR="$SCRIPT_DIR/templates"

# Source environment utilities for safe loading
source "$SCRIPT_DIR/env-utils.sh"

# Helper functions
echo_info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

echo_success() {
  echo -e "\033[1;32m[SUCCESS]\033[0m $1"
}

echo_error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

echo_warning() {
  echo -e "\033[1;33m[WARNING]\033[0m $1"
}

# Load environment safely (without executing JSON values)
if [ -f ".env.local" ]; then
  load_env_safe ".env.local"
  
  # Expand nested variables (e.g., HASURA_ROUTE=api.${BASE_DOMAIN})
  # This ensures variables like ${BASE_DOMAIN} inside other variables get expanded
  for var in HASURA_ROUTE AUTH_ROUTE STORAGE_ROUTE STORAGE_CONSOLE_ROUTE FUNCTIONS_ROUTE DASHBOARD_ROUTE MAILPIT_ROUTE MAIL_ROUTE FILES_ROUTE MAILHOG_ROUTE; do
    if [[ -n "${!var}" ]]; then
      expanded_value=$(eval echo "${!var}")
      export "$var=$expanded_value"
    fi
  done
else
  echo_error "No .env.local file found."
  exit 1
fi

# Input validation functions
validate_domain() {
  local domain=$1
  if [[ ! "$domain" =~ ^[a-zA-Z0-9][a-zA-Z0-9-]*(\.[a-zA-Z0-9][a-zA-Z0-9-]*)*$ ]]; then
    echo_error "Invalid domain format: $domain"
    return 1
  fi
  return 0
}

validate_port() {
  local port=$1
  if [[ ! "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
    echo_error "Invalid port number: $port"
    return 1
  fi
  return 0
}

validate_password() {
  local password=$1
  local name=$2
  
  # Check minimum length
  if [ ${#password} -lt 12 ]; then
    echo_error "$name must be at least 12 characters long"
    return 1
  fi
  
  # Check for common weak passwords
  local weak_passwords=("password" "123456" "admin" "secret" "default" "changeme" "password123")
  for weak in "${weak_passwords[@]}"; do
    if [[ "${password,,}" == *"$weak"* ]]; then
      echo_error "$name contains a common weak password pattern"
      return 1
    fi
  done
  
  return 0
}

# Validate required variables
if ! validate_domain "$BASE_DOMAIN"; then
  exit 1
fi

# Compose database URL from individual variables (with validation)
if [[ -z "$POSTGRES_USER" ]] || [[ -z "$POSTGRES_PASSWORD" ]] || [[ -z "$POSTGRES_HOST" ]] || [[ -z "$POSTGRES_PORT" ]] || [[ -z "$POSTGRES_DB" ]]; then
  echo_error "Missing required PostgreSQL configuration variables"
  exit 1
fi

HASURA_GRAPHQL_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"

# Environment-specific security warnings
# Support both ENV and ENVIRONMENT for backward compatibility
if [[ "$ENV" == "prod" ]] || [[ "$ENVIRONMENT" == "production" ]]; then
  echo_info "Building for PRODUCTION environment"
  
  # Validate passwords
  if ! validate_password "$POSTGRES_PASSWORD" "PostgreSQL password"; then
    exit 1
  fi
  
  if ! validate_password "$HASURA_GRAPHQL_ADMIN_SECRET" "Hasura admin secret"; then
    exit 1
  fi
  
  # Check other critical passwords
  if [[ -n "$MINIO_ROOT_PASSWORD" ]] && ! validate_password "$MINIO_ROOT_PASSWORD" "MinIO root password"; then
    exit 1
  fi
  
  # Check SSL configuration
  if [[ "$SSL_MODE" == "none" ]]; then
    echo_error "WARNING: SSL is disabled in production! This is insecure."
    exit 1
  fi
else
  echo_info "Building for DEVELOPMENT environment"
fi

# Create directory structure
echo_info "Creating directory structure..."

mkdir -p nginx/conf.d
mkdir -p nginx/ssl
mkdir -p postgres/init
mkdir -p hasura/metadata
mkdir -p hasura/migrations
mkdir -p functions/src
mkdir -p certs
mkdir -p config-server

# Generate Nginx configuration
echo_info "Generating Nginx configuration..."

cat > nginx/nginx.conf << 'EOF'
user nginx;
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

    # SSL Settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-AES256-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-DSS-AES128-GCM-SHA256:kEDH+AESGCM:ECDHE-RSA-AES128-SHA256:ECDHE-ECDSA-AES128-SHA256:ECDHE-RSA-AES128-SHA:ECDHE-ECDSA-AES128-SHA:ECDHE-RSA-AES256-SHA384:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA:ECDHE-ECDSA-AES256-SHA:DHE-RSA-AES128-SHA256:DHE-RSA-AES128-SHA:DHE-DSS-AES128-SHA256:DHE-RSA-AES256-SHA256:DHE-DSS-AES256-SHA:DHE-RSA-AES256-SHA:!aNULL:!eNULL:!EXPORT:!DES:!RC4:!3DES:!MD5:!PSK;
    ssl_prefer_server_ciphers on;

    # Gzip Settings
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml text/javascript application/json application/javascript application/xml+rss application/rss+xml application/atom+xml image/svg+xml;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    include /etc/nginx/conf.d/*.conf;
}
EOF

# Generate SSL certificates for local development
if [[ "$SSL_MODE" == "local" ]] && [[ "$BASE_DOMAIN" == *"nself.org"* ]]; then
  echo_info "Generating trusted SSL certificates..."
  
  # Check if mkcert is available
  MKCERT_PATH="$SCRIPT_DIR/mkcert"
  
  if [[ -f "$MKCERT_PATH" ]] && [[ -x "$MKCERT_PATH" ]]; then
    # Use mkcert for trusted certificates
    echo_info "Using mkcert for automatic SSL trust..."
    
    # Install root CA if not already installed
    "$MKCERT_PATH" -install 2>/dev/null || true
    
    # Generate certificates for all domains
    "$MKCERT_PATH" -cert-file nginx/ssl/nself.org.crt \
                   -key-file nginx/ssl/nself.org.key \
                   "*.nself.org" "*.local.nself.org" "local.nself.org" 2>/dev/null || {
      echo_warning "mkcert certificate generation failed, falling back to self-signed"
      # Fallback to self-signed
      openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
        -keyout nginx/ssl/nself.org.key \
        -out nginx/ssl/nself.org.crt \
        -subj "/C=US/ST=State/L=City/O=NSELF/CN=*.nself.org" \
        -addext "subjectAltName=DNS:*.nself.org,DNS:*.local.nself.org,DNS:local.nself.org"
    }
    
    echo_success "✅ SSL certificates generated and automatically trusted!"
    echo_info "   No browser warnings - certificates are already trusted."
  elif [ ! -f "$SCRIPT_DIR/certs/nself.org.crt" ]; then
    # Generate self-signed certificate as fallback
    echo_warning "mkcert not found, generating self-signed certificate..."
    echo_warning "Run 'nself trust' after build to trust the certificate"
    
    openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
      -keyout nginx/ssl/nself.org.key \
      -out nginx/ssl/nself.org.crt \
      -subj "/C=US/ST=State/L=City/O=NSELF/CN=*.nself.org" \
      -addext "subjectAltName=DNS:*.nself.org,DNS:*.local.nself.org,DNS:local.nself.org"
    
    echo_warning "⚠️  Self-signed certificate generated."
    echo_warning "   Run 'nself trust' to avoid browser security warnings."
  else
    # Copy pre-made certificates
    cp "$SCRIPT_DIR/certs/nself.org.crt" nginx/ssl/
    cp "$SCRIPT_DIR/certs/nself.org.key" nginx/ssl/
    echo_warning "Using existing certificates. Run 'nself trust' if you see browser warnings."
  fi
fi

# Generate service configurations
echo_info "Generating service configurations..."

# Hasura configuration
cat > nginx/conf.d/hasura.conf << EOF
upstream hasura {
    server hasura:8080;
}

server {
    listen 80;
    server_name ${HASURA_ROUTE};

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${HASURA_ROUTE};

    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;

    location / {
        proxy_pass http://hasura;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# Auth configuration
cat > nginx/conf.d/auth.conf << EOF
upstream auth {
    server auth:${AUTH_PORT:-4000};
}

server {
    listen 80;
    server_name ${AUTH_ROUTE};

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${AUTH_ROUTE};

    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;

    location / {
        proxy_pass http://auth;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# Storage API configuration (hasura-storage)
cat > nginx/conf.d/files.conf << EOF
upstream storage-api {
    server storage:5000;
}

server {
    listen 80;
    server_name files.${BASE_DOMAIN};
    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name files.${BASE_DOMAIN};
    
    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;
    
    client_max_body_size 1000m;
    
    location / {
        proxy_pass http://storage-api;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# S3 Storage configuration (MinIO)
cat > nginx/conf.d/s3.conf << EOF
upstream minio {
    server minio:9000;
}

upstream minio-console {
    server minio:9001;
}

server {
    listen 80;
    server_name ${STORAGE_ROUTE};

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${STORAGE_ROUTE};

    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;

    client_max_body_size 1000m;

    location / {
        proxy_pass http://minio;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}

server {
    listen 80;
    server_name ${STORAGE_CONSOLE_ROUTE};

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${STORAGE_CONSOLE_ROUTE};

    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;

    location / {
        proxy_pass http://minio-console;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# Functions configuration if enabled
if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
  cat > nginx/conf.d/functions.conf << EOF
upstream functions {
    server functions:3000;
}

server {
    listen 80;
    server_name ${FUNCTIONS_ROUTE};

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${FUNCTIONS_ROUTE};

    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;

    location / {
        proxy_pass http://functions;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
fi

# Dashboard configuration
if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
  cat > nginx/conf.d/dashboard.conf << EOF
upstream dashboard {
    server dashboard:3000;
}

server {
    listen 80;
    server_name ${DASHBOARD_ROUTE};

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${DASHBOARD_ROUTE};

    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;

    location / {
        proxy_pass http://dashboard;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
EOF
fi

# Email UI configuration for development
if [[ "$EMAIL_PROVIDER" == "mailhog" ]] || [[ "$EMAIL_PROVIDER" == "mailpit" ]]; then
  # Determine the service name and route
  if [[ "$EMAIL_PROVIDER" == "mailpit" ]]; then
    MAIL_SERVICE="mailpit"
    MAIL_ROUTE="${MAILPIT_ROUTE:-mail.${BASE_DOMAIN}}"
  else
    MAIL_SERVICE="mailhog"
    MAIL_ROUTE="${MAILHOG_ROUTE:-mailhog.${BASE_DOMAIN}}"
  fi
  
  cat > nginx/conf.d/mail.conf << EOF
upstream mail {
    server ${MAIL_SERVICE}:8025;
}

server {
    listen 80;
    server_name ${MAIL_ROUTE};

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name ${MAIL_ROUTE};

    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;

    location / {
        proxy_pass http://mail;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
fi

# Generate frontend app routes
echo_info "Configuring frontend app routes..."

# Process APP_ROUTE_* environment variables
for i in {0..20}; do
  route_var="APP_ROUTE_$i"
  route_value="${!route_var}"
  
  # Also check for alternative naming convention APP_N_ROUTE
  if [[ -z "$route_value" ]]; then
    route_var="APP_${i}_ROUTE"
    route_value="${!route_var}"
  fi
  
  if [[ -n "$route_value" ]]; then
    # Expand variables in route_value (e.g., ${BASE_DOMAIN})
    route_value=$(eval echo "$route_value")
    
    # Parse port:domain format
    IFS=':' read -r port domain <<< "$route_value"
    
    # Validate port and domain
    if ! validate_port "$port"; then
      echo_error "Skipping invalid app route $i"
      continue
    fi
    
    if ! validate_domain "$domain"; then
      echo_error "Skipping invalid app route $i"
      continue
    fi
    
    # Extract subdomain from domain
    subdomain="${domain%%.*}"
    
    echo_info "Adding route: $subdomain (localhost:$port -> $domain)"
    
    cat > nginx/conf.d/${subdomain}.conf << EOF
# Frontend App Route $i: $subdomain
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

EOF

    # Add environment-specific security headers
    if ([[ "$ENV" == "prod" ]] || [[ "$ENVIRONMENT" == "production" ]]) && [[ "$SECURITY_HEADERS_ENABLED" == "true" ]]; then
      cat >> nginx/conf.d/${subdomain}.conf << EOF
    # Production security headers
    add_header Strict-Transport-Security "max-age=${HSTS_MAX_AGE}; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Permissions-Policy "geolocation=(), microphone=(), camera=()" always;

EOF
    fi

    # Add rate limiting if enabled
    if [[ "$RATE_LIMIT_ENABLED" == "true" ]]; then
      cat >> nginx/conf.d/${subdomain}.conf << EOF
    # Rate limiting
    limit_req_zone \$binary_remote_addr zone=app${i}_limit:10m rate=${RATE_LIMIT_REQUESTS_PER_MINUTE}r/m;
    limit_req zone=app${i}_limit burst=${RATE_LIMIT_BURST} nodelay;

EOF
    fi

    # Detect OS for proper host networking
    OS="$(uname -s)"
    if [[ "$OS" == "Linux" ]]; then
      # On Linux, use the default gateway IP
      PROXY_HOST="172.17.0.1"
    else
      # On Mac/Windows with Docker Desktop
      PROXY_HOST="host.docker.internal"
    fi
    
    cat >> nginx/conf.d/${subdomain}.conf << EOF
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
        proxy_connect_timeout ${NGINX_PROXY_TIMEOUT};
        proxy_send_timeout ${NGINX_PROXY_TIMEOUT};
        proxy_read_timeout ${NGINX_PROXY_TIMEOUT};
    }
}
EOF
  fi
done

# Generate database initialization script
echo_info "Creating database initialization script..."

cat > postgres/init/00-init.sql << EOF
-- Initialize database
CREATE DATABASE IF NOT EXISTS ${POSTGRES_DB};

-- Enable extensions if specified
\c ${POSTGRES_DB};
EOF

# Add PostgreSQL extensions
if [[ -n "$POSTGRES_EXTENSIONS" ]]; then
  IFS=',' read -ra EXTENSIONS <<< "$POSTGRES_EXTENSIONS"
  for ext in "${EXTENSIONS[@]}"; do
    # Trim whitespace
    ext=$(echo "$ext" | xargs)
    
    # Handle special cases for extension names
    if [[ "$ext" == "uuid-ossp" ]]; then
      echo "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" >> postgres/init/00-init.sql
    elif [[ "$ext" == "timescaledb" ]]; then
      # TimescaleDB needs special handling
      echo "CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;" >> postgres/init/00-init.sql
    elif [[ "$ext" == "postgis" ]]; then
      echo "CREATE EXTENSION IF NOT EXISTS postgis CASCADE;" >> postgres/init/00-init.sql
    elif [[ "$ext" == "pgvector" ]]; then
      echo "CREATE EXTENSION IF NOT EXISTS vector;" >> postgres/init/00-init.sql
    else
      echo "CREATE EXTENSION IF NOT EXISTS $ext;" >> postgres/init/00-init.sql
    fi
  done
fi

# Create required schemas for services
cat >> postgres/init/00-init.sql << EOF

-- Create schemas for hasura-storage and hasura-auth
CREATE SCHEMA IF NOT EXISTS storage;
CREATE SCHEMA IF NOT EXISTS auth;
EOF

# Generate docker-compose.yml
echo_info "Generating docker-compose.yml..."

bash "$SCRIPT_DIR/compose.sh"

# Generate Hasura metadata
echo_info "Creating Hasura metadata..."

cat > hasura/metadata/version.yaml << EOF
version: 3
EOF

cat > hasura/metadata/databases.yaml << EOF
- name: default
  kind: postgres
  configuration:
    connection_info:
      database_url:
        from_env: HASURA_GRAPHQL_DATABASE_URL
      isolation_level: read-committed
      pool_settings:
        connection_lifetime: 600
        idle_timeout: 180
        max_connections: 50
        retries: 1
      use_prepared_statements: true
EOF

# Generate Hasura actions and events if services are enabled
if [[ "$SERVICES_ENABLED" == "true" ]]; then
  if [[ -f "$TEMPLATES_DIR/hasura/actions.yaml.template" ]]; then
    echo_info "Creating Hasura actions metadata..."
    cp "$TEMPLATES_DIR/hasura/actions.yaml.template" "hasura/metadata/actions.yaml"
  fi
  
  if [[ -f "$TEMPLATES_DIR/hasura/event_triggers.yaml.template" ]]; then
    echo_info "Creating Hasura event triggers metadata..."
    cp "$TEMPLATES_DIR/hasura/event_triggers.yaml.template" "hasura/metadata/event_triggers.yaml"
  fi
fi

# Create placeholder migration
mkdir -p hasura/migrations/default/1_init

# Use our schema template if services are enabled, otherwise create basic migration
if [[ "$SERVICES_ENABLED" == "true" ]] && [[ -f "$TEMPLATES_DIR/hasura/schema.sql.template" ]]; then
  echo_info "Using time-series schema for migration..."
  cp "$TEMPLATES_DIR/hasura/schema.sql.template" "hasura/migrations/default/1_init/up.sql"
else
  cat > hasura/migrations/default/1_init/up.sql << EOF
-- Initial migration
-- Add your database schema here

-- Basic users table for multi-app support
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT UNIQUE NOT NULL,
  display_name TEXT,
  avatar_url TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create trigger to update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS \$\$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
\$\$ LANGUAGE plpgsql;

CREATE TRIGGER users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Example: Additional tables would go here
-- CREATE TABLE app1_data (
--   id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--   user_id UUID REFERENCES users(id),
--   content JSONB,
--   created_at TIMESTAMPTZ DEFAULT NOW()
-- );
EOF
fi

echo "" > hasura/migrations/default/1_init/down.sql

# Create functions example if enabled
if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
  cat > functions/src/index.js << EOF
const express = require('express');
const app = express();

app.use(express.json());

app.get('/', (req, res) => {
  res.json({ message: 'Functions service is running!' });
});

app.post('/hello', (req, res) => {
  const { name = 'World' } = req.body;
  res.json({ message: \`Hello, \${name}!\` });
});

const port = process.env.PORT || 3000;
app.listen(port, () => {
  console.log(\`Functions service listening on port \${port}\`);
});
EOF

  cat > functions/package.json << EOF
{
  "name": "nself-functions",
  "version": "1.0.0",
  "description": "Serverless functions for nself",
  "main": "src/index.js",
  "scripts": {
    "start": "node src/index.js",
    "dev": "nodemon src/index.js"
  },
  "dependencies": {
    "express": "^4.18.2"
  },
  "devDependencies": {
    "nodemon": "^3.0.1"
  }
}
EOF

  cat > functions/Dockerfile << EOF
FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install --only=production

COPY . .

EXPOSE 3000

CMD ["npm", "start"]
EOF

  # Generate package-lock.json for functions
  echo_info "Generating package-lock.json for functions..."
  (cd functions && npm install --package-lock-only 2>/dev/null) || {
    # If npm install fails, create a basic package-lock.json
    cat > functions/package-lock.json << EOF
{
  "name": "nself-functions",
  "version": "1.0.0",
  "lockfileVersion": 2,
  "requires": true,
  "packages": {}
}
EOF
  }
fi

# Create config server for Dashboard if enabled
if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
  echo_info "Setting up config server for Dashboard..."
  
  # Copy config server files from templates
  cp "$SCRIPT_DIR/templates/config-server/package.json" config-server/package.json 2>/dev/null || {
    # If template doesn't exist, create it
    cat > config-server/package.json << 'EOF'
{
  "name": "nself-config-server",
  "version": "1.0.0",
  "description": "Mock config server for Nhost Dashboard local development",
  "main": "server.js",
  "scripts": {
    "start": "node server.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "cors": "^2.8.5"
  }
}
EOF
  }
  
  cp "$SCRIPT_DIR/templates/config-server/Dockerfile" config-server/Dockerfile 2>/dev/null || {
    cat > config-server/Dockerfile << 'EOF'
FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install --production

COPY . .

EXPOSE 4001

CMD ["npm", "start"]
EOF
  }
  
  cp "$SCRIPT_DIR/templates/config-server/server.js" config-server/server.js 2>/dev/null || {
    # Create inline if template missing
    echo_warning "Config server template not found, creating basic version..."
    cat > config-server/server.js << 'EOF'
const express = require('express');
const cors = require('cors');
const app = express();

app.use(cors());
app.use(express.json());

// Mock project configuration
const projectConfig = {
  project: {
    id: 'local-dev',
    name: process.env.PROJECT_NAME || 'nself-project',
    subdomain: 'local',
    region: 'local'
  },
  config: {
    hasura: {
      adminSecret: process.env.HASURA_GRAPHQL_ADMIN_SECRET,
      url: `https://api.${process.env.BASE_DOMAIN}`
    },
    auth: {
      url: `https://auth.${process.env.BASE_DOMAIN}`
    },
    storage: {
      url: `https://storage.${process.env.BASE_DOMAIN}`
    },
    functions: {
      url: `https://functions.${process.env.BASE_DOMAIN}`
    }
  }
};

app.get('/healthz', (req, res) => {
  res.json({ status: 'healthy' });
});

app.get('/v1/config', (req, res) => {
  res.json(projectConfig);
});

const PORT = process.env.PORT || 4001;
app.listen(PORT, '0.0.0.0', () => {
  console.log(`Config server running on port ${PORT}`);
});
EOF
  }
  
  # Create nginx config for config server
  cat > nginx/conf.d/config.conf << EOF
upstream config-server {
    server config-server:4001;
}

server {
    listen 80;
    server_name config.${BASE_DOMAIN};
    
    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

server {
    listen 443 ssl http2;
    server_name config.${BASE_DOMAIN};
    
    ssl_certificate /etc/nginx/ssl/nself.org.crt;
    ssl_certificate_key /etc/nginx/ssl/nself.org.key;
    
    location / {
        proxy_pass http://config-server;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
fi

# Note: NestJS services are now generated in services/ directory by services.sh
# The old microservices/ folder is deprecated

# Create services directory structure if enabled
if [[ "$SERVICES_ENABLED" == "true" ]]; then
  echo_info "Creating services directory structure..."
  bash "$SCRIPT_DIR/services.sh"
fi

# Create sample schema.dbml if configured and doesn't exist
if [ -n "$LOCAL_SCHEMA_FILE" ] && [ ! -f "$LOCAL_SCHEMA_FILE" ]; then
  echo_info "Creating sample database schema: $LOCAL_SCHEMA_FILE"
  bash "$SCRIPT_DIR/db.sh" sample > /dev/null 2>&1
elif [ ! -f "schema.dbml" ] && [ -z "$DBML_URL" ] && [ -z "$DBML_SCHEMA_URL" ] && [ -z "$DBDOCS_SCHEMA_URL" ]; then
  # Create a default schema.dbml if no schema configuration exists
  echo_info "Creating sample database schema: schema.dbml"
  bash "$SCRIPT_DIR/db.sh" sample > /dev/null 2>&1
fi

echo_success "Project structure created successfully!"

# SSL certificate reminder
if [[ "$SSL_MODE" == "local" ]]; then
  echo ""
  echo_warning "🔐 SSL Certificate Notice:"
  echo_info "   To avoid browser security warnings, trust the certificate:"
  echo_info "   Run: nself trust"
  echo_info "   This will install the certificate to your system trust store"
  echo ""
fi

# Display next steps
echo_info "Next steps:"
echo_info "1. Review and customize the generated files:"
echo_info "   - postgres/init/00-init.sql - Add your database schema"
echo_info "   - hasura/migrations/default/1_init/up.sql - Define your tables"
echo_info "   - nginx/conf.d/*.conf - Adjust routing if needed"

if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
  echo_info "   - functions/src/index.js - Add your serverless functions"
fi

if [[ "$NESTJS_ENABLED" == "true" ]]; then
  echo_info "   - microservices/weather-service/ - Implement your microservice"
fi

echo_info "2. Run 'nself up' to start the services"