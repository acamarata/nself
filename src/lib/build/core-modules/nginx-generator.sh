#!/usr/bin/env bash
# nginx-generator.sh - Generate nginx configs with runtime env var substitution
# Uses nginx envsubst or template variables for runtime configuration

# Generate main nginx configuration
generate_nginx_config() {
  local force="${1:-false}"

  # Generate main nginx.conf
  generate_main_nginx_conf

  # Generate default server block
  generate_default_server

  # Generate service routes
  generate_service_routes

  # Generate frontend app routes
  generate_frontend_routes

  # Generate custom service routes
  generate_custom_routes
}

# Generate main nginx.conf
generate_main_nginx_conf() {
  cat > nginx/nginx.conf <<'EOF'
user nginx;
worker_processes auto;
pid /var/run/nginx.pid;

events {
    worker_connections 2048;
    multi_accept on;
    use epoll;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Performance
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;
    server_tokens off;
    client_max_body_size 100M;

    # SSL Configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Gzip
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml text/javascript
               application/json application/javascript application/xml+rss
               application/rss+xml application/atom+xml image/svg+xml
               text/x-js text/x-cross-domain-policy application/x-font-ttf
               application/x-font-opentype application/vnd.ms-fontobject
               image/x-icon;

    # Logging
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';

    access_log /var/log/nginx/access.log main;
    error_log /var/log/nginx/error.log warn;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=general:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=api:10m rate=100r/s;

    # Include all configurations
    include /etc/nginx/conf.d/*.conf;
    include /etc/nginx/sites/*.conf;
}
EOF
}

# Generate default server block
generate_default_server() {
  cat > nginx/conf.d/default.conf <<'EOF'
# Default server - redirect HTTP to HTTPS
server {
    listen 80 default_server;
    server_name _;

    # ACME challenge for Let's Encrypt
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

# Default HTTPS server
server {
    listen 443 ssl http2 default_server;
    server_name ${BASE_DOMAIN:-localhost};

    # SSL certificates - runtime path based on environment
    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files $uri $uri/ /index.html;
    }

    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
EOF
}

# Generate core service routes (Hasura, Auth, etc.)
generate_service_routes() {
  # Hasura GraphQL API route
  if [[ "${HASURA_ENABLED:-false}" == "true" ]]; then
    cat > nginx/sites/hasura.conf <<'EOF'
# Hasura GraphQL Engine
server {
    listen 443 ssl http2;
    server_name ${HASURA_ROUTE:-api}.${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        proxy_pass http://hasura:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_read_timeout 86400;

        # Increase buffer sizes
        proxy_buffer_size 128k;
        proxy_buffers 4 256k;
        proxy_busy_buffers_size 256k;
    }
}
EOF
  fi

  # Auth service route
  if [[ "${AUTH_ENABLED:-false}" == "true" ]]; then
    cat > nginx/sites/auth.conf <<'EOF'
# Authentication Service
server {
    listen 443 ssl http2;
    server_name ${AUTH_ROUTE:-auth}.${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        proxy_pass http://auth:4000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF
  fi

  # Storage/MinIO route
  if [[ "${MINIO_ENABLED:-true}" == "true" ]]; then
    cat > nginx/sites/storage.conf <<'EOF'
# MinIO Storage Console
server {
    listen 443 ssl http2;
    server_name ${STORAGE_CONSOLE_ROUTE:-storage-console}.${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        proxy_pass http://minio:9001;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# MinIO S3 API
server {
    listen 443 ssl http2;
    server_name ${STORAGE_ROUTE:-storage}.${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    client_max_body_size 1000M;

    location / {
        proxy_pass http://minio:9000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF
  fi

  # Optional services routes
  generate_optional_service_routes
}

# Generate optional service routes
generate_optional_service_routes() {
  # Admin dashboard
  if [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]]; then
    cat > nginx/sites/admin.conf <<'EOF'
server {
    listen 443 ssl http2;
    server_name ${ADMIN_ROUTE:-admin}.${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        proxy_pass http://nself-admin:3100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF
  fi

  # Monitoring routes
  if [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
    # Grafana
    cat > nginx/sites/grafana.conf <<'EOF'
server {
    listen 443 ssl http2;
    server_name ${GRAFANA_ROUTE:-grafana}.${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        proxy_pass http://grafana:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

    # Prometheus
    cat > nginx/sites/prometheus.conf <<'EOF'
server {
    listen 443 ssl http2;
    server_name ${PROMETHEUS_ROUTE:-prometheus}.${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        proxy_pass http://prometheus:9090;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF
  fi
}

# Generate frontend app routes
generate_frontend_routes() {
  for i in {1..10}; do
    local app_name_var="FRONTEND_APP_${i}_NAME"
    local app_name="${!app_name_var:-}"

    if [[ -n "$app_name" ]]; then
      cat > "nginx/sites/frontend-${app_name}.conf" <<EOF
# Frontend Application: $app_name
server {
    listen 443 ssl http2;
    # Runtime route resolution based on environment
    server_name \${FRONTEND_APP_${i}_ROUTE:-${app_name}}.\${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        # Proxy to external frontend app running on host
        proxy_pass http://host.docker.internal:\${FRONTEND_APP_${i}_PORT:-$((3000 + i - 1))};
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # WebSocket support for hot reload
        proxy_read_timeout 86400;
    }
}

# API route for frontend's remote schema (if configured)
EOF

      # Check if remote schema is configured
      local schema_var="FRONTEND_APP_${i}_REMOTE_SCHEMA_NAME"
      if [[ -n "${!schema_var:-}" ]]; then
        cat >> "nginx/sites/frontend-${app_name}.conf" <<EOF
server {
    listen 443 ssl http2;
    # Remote schema API route
    server_name \${FRONTEND_APP_${i}_API_ROUTE:-api.${app_name}}.\${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        # Proxy to frontend's API endpoint
        proxy_pass http://host.docker.internal:\${FRONTEND_APP_${i}_API_PORT:-$((4000 + i))};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
      fi
    fi
  done
}

# Generate custom service routes
generate_custom_routes() {
  for i in {1..20}; do
    local cs_name_var="CS_${i}_NAME"
    local cs_name="${!cs_name_var:-}"

    if [[ -n "$cs_name" ]]; then
      # Check if service is public
      local cs_public_var="CS_${i}_PUBLIC"
      local cs_public="${!cs_public_var:-true}"

      if [[ "$cs_public" == "true" ]]; then
        cat > "nginx/sites/custom-${cs_name}.conf" <<EOF
# Custom Service: $cs_name
server {
    listen 443 ssl http2;
    # Runtime route - can be overridden per environment
    server_name \${CS_${i}_ROUTE:-${cs_name}}.\${BASE_DOMAIN:-localhost};

    ssl_certificate /etc/nginx/ssl/localhost/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/localhost/privkey.pem;

    location / {
        proxy_pass http://${cs_name}:\${CS_${i}_PORT:-$((8000 + i))};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # API timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://${cs_name}:\${CS_${i}_PORT:-$((8000 + i))}/health;
        access_log off;
    }
}
EOF
      fi
    fi
  done
}

# Generate database initialization
generate_database_init() {
  local force="${1:-false}"

  if [[ "$force" == "true" ]] || [[ ! -f "postgres/init/00-init.sql" ]]; then
    cat > postgres/init/00-init.sql <<'EOF'
-- Database initialization script
-- Uses runtime environment variables

-- Create database if not exists
SELECT 'CREATE DATABASE ${POSTGRES_DB:-myproject}'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${POSTGRES_DB:-myproject}');

-- Create user if not exists
DO
$$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_user WHERE usename = '${POSTGRES_USER:-postgres}') THEN
    CREATE USER ${POSTGRES_USER:-postgres} WITH PASSWORD '${POSTGRES_PASSWORD:-postgres}';
  END IF;
END
$$;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE ${POSTGRES_DB:-myproject} TO ${POSTGRES_USER:-postgres};

-- Create schema for Hasura
\c ${POSTGRES_DB:-myproject};
CREATE SCHEMA IF NOT EXISTS hdb_catalog;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS storage;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS citext;

-- Create tables for each frontend app with table prefix
EOF

    # Add frontend app table creation
    for i in {1..10}; do
      local app_name_var="FRONTEND_APP_${i}_NAME"
      local app_name="${!app_name_var:-}"
      local prefix_var="FRONTEND_APP_${i}_TABLE_PREFIX"
      local prefix="${!prefix_var:-}"

      if [[ -n "$app_name" ]] && [[ -n "$prefix" ]]; then
        cat >> postgres/init/00-init.sql <<EOF

-- Tables for frontend app: $app_name
CREATE SCHEMA IF NOT EXISTS ${prefix}schema;
EOF
      fi
    done
  fi
}

# Export all functions
export -f generate_nginx_config
export -f generate_main_nginx_conf
export -f generate_default_server
export -f generate_service_routes
export -f generate_optional_service_routes
export -f generate_frontend_routes
export -f generate_custom_routes
export -f generate_database_init