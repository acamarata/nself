#!/usr/bin/env bash
# nginx-generator.sh - Generate nginx configs with runtime env var substitution
# Uses nginx envsubst or template variables for runtime configuration

# Determine SSL certificate directory name based on domain
# Returns: directory name for SSL certs (e.g., "localhost", "nself-org", "example-com")
get_ssl_cert_dir() {
  local domain="${BASE_DOMAIN:-localhost}"

  # For localhost or *.localhost domains, use localhost
  if [[ "$domain" == "localhost" ]] || [[ "$domain" == *".localhost" ]]; then
    echo "localhost"
    return
  fi

  # For nself.org domains, use nself-org
  if [[ "$domain" == "nself.org" ]] || [[ "$domain" == *".nself.org" ]]; then
    echo "nself-org"
    return
  fi

  # For other domains, convert dots to dashes (e.g., example.com -> example-com)
  echo "$domain" | tr '.' '-'
}

# Detect if running on Linux server (not Docker Desktop)
is_linux_server() {
  # Docker Desktop sets DOCKER_HOST or has special indicators
  # On Linux servers, host.docker.internal doesn't work
  if [[ "$(uname -s)" == "Linux" ]]; then
    # Check if Docker Desktop is installed (has desktop features)
    if ! docker info 2>/dev/null | grep -qi "docker desktop"; then
      return 0  # Is Linux server
    fi
  fi
  return 1  # Not Linux server (macOS, Windows, or Docker Desktop on Linux)
}

# Clean up old nginx site configs before regeneration
cleanup_nginx_sites() {
  # Remove all generated site configs - they will be regenerated
  rm -f nginx/sites/*.conf 2>/dev/null || true
}

# Generate main nginx configuration
generate_nginx_config() {
  local force="${1:-false}"

  # Clean up old configs first to prevent stale configs for disabled services
  cleanup_nginx_sites

  # Generate main nginx.conf
  generate_main_nginx_conf

  # Generate default server block
  generate_default_server

  # Generate service routes
  generate_service_routes

  # Generate frontend app routes (skip on Linux servers)
  if ! is_linux_server; then
    generate_frontend_routes
  fi

  # Generate custom service routes
  generate_custom_routes

  # Generate plugin webhook routes
  generate_plugin_routes
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
    server_names_hash_bucket_size 128;
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
  local base_domain="${BASE_DOMAIN:-localhost}"
  local ssl_dir
  ssl_dir=$(get_ssl_cert_dir)

  cat > nginx/conf.d/default.conf <<EOF
# Default server - redirect HTTP to HTTPS
server {
    listen 80 default_server;
    server_name _;

    # ACME challenge for Let's Encrypt
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        return 301 https://\$host\$request_uri;
    }
}

# Default HTTPS server
server {
    listen 443 ssl default_server;
    http2 on;
    server_name ${base_domain};

    # SSL certificates - path based on BASE_DOMAIN
    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files \$uri \$uri/ /index.html;
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
  local ssl_dir
  ssl_dir=$(get_ssl_cert_dir)

  # Hasura GraphQL API route
  if [[ "${HASURA_ENABLED:-false}" == "true" ]]; then
    local hasura_route="${HASURA_ROUTE:-api}"
    local base_domain="${BASE_DOMAIN:-localhost}"

    cat > nginx/sites/hasura.conf <<EOF
# Hasura GraphQL Engine
server {
    listen 443 ssl;
    http2 on;
    server_name ${hasura_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://hasura:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

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
    local auth_route="${AUTH_ROUTE:-auth}"
    local base_domain="${BASE_DOMAIN:-localhost}"

    cat > nginx/sites/auth.conf <<EOF
# Authentication Service
server {
    listen 443 ssl;
    http2 on;
    server_name ${auth_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://auth:4000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
  fi

  # Storage/MinIO route
  if [[ "${MINIO_ENABLED:-false}" == "true" ]]; then
    local storage_console_route="${STORAGE_CONSOLE_ROUTE:-storage-console}"
    local storage_route="${STORAGE_ROUTE:-storage}"
    local base_domain="${BASE_DOMAIN:-localhost}"
    cat > nginx/sites/storage.conf <<EOF
# MinIO Storage Console
server {
    listen 443 ssl;
    http2 on;
    server_name ${storage_console_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://minio:9001;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}

# MinIO S3 API
server {
    listen 443 ssl;
    http2 on;
    server_name ${storage_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    client_max_body_size 1000M;

    location / {
        proxy_pass http://minio:9000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
  fi

  # Optional services routes
  generate_optional_service_routes
}

# Generate optional service routes
generate_optional_service_routes() {
  local base_domain="${BASE_DOMAIN:-localhost}"
  local ssl_dir
  ssl_dir=$(get_ssl_cert_dir)

  # Admin dashboard
  if [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]]; then
    local admin_route="${ADMIN_ROUTE:-admin}"
    local admin_upstream="nself-admin:3021"

    # Admin-dev mode: route to local dev server instead of Docker container
    if [[ "${NSELF_ADMIN_DEV:-false}" == "true" ]]; then
      local dev_port="${NSELF_ADMIN_DEV_PORT:-3000}"
      # Use host.docker.internal for Docker Desktop, host-gateway for Linux
      if is_linux_server; then
        admin_upstream="172.17.0.1:${dev_port}"
      else
        admin_upstream="host.docker.internal:${dev_port}"
      fi
    fi

    cat > nginx/sites/admin.conf <<EOF
# nself Admin Dashboard${NSELF_ADMIN_DEV:+ (DEV MODE - routing to local server)}
server {
    listen 443 ssl;
    http2 on;
    server_name ${admin_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://${admin_upstream};
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

  # Search route (provider-agnostic: MeiliSearch or Typesense)
  local search_enabled="${SEARCH_ENABLED:-false}"
  local search_provider="${SEARCH_PROVIDER:-meilisearch}"

  # Legacy support for old MEILISEARCH_ENABLED variable
  if [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]]; then
    search_enabled="true"
    search_provider="meilisearch"
  fi

  # Legacy support for old TYPESENSE_ENABLED variable
  if [[ "${TYPESENSE_ENABLED:-false}" == "true" ]]; then
    search_enabled="true"
    search_provider="typesense"
  fi

  if [[ "$search_enabled" == "true" ]]; then
    local search_route="${SEARCH_ROUTE:-search}"
    local upstream_host=""
    local upstream_port=""
    local service_name=""

    case "$search_provider" in
      meilisearch)
        upstream_host="meilisearch"
        upstream_port="7700"
        service_name="MeiliSearch"
        ;;
      typesense)
        upstream_host="typesense"
        upstream_port="8108"
        service_name="Typesense"
        ;;
      *)
        upstream_host="meilisearch"
        upstream_port="7700"
        service_name="MeiliSearch"
        ;;
    esac

    cat > nginx/sites/search.conf <<EOF
# ${service_name} Search Engine
server {
    listen 443 ssl;
    http2 on;
    server_name ${search_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://${upstream_host}:${upstream_port};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # Typesense-specific headers (works for MeiliSearch too)
        proxy_set_header X-Typesense-API-KEY \$http_x_typesense_api_key;
    }
}
EOF
  fi

  # MailPit route (dev only - mail testing)
  if [[ "${MAILPIT_ENABLED:-false}" == "true" ]]; then
    local mail_route="${MAIL_ROUTE:-mail}"

    cat > nginx/sites/mailpit.conf <<EOF
# MailPit (Development Mail Testing)
server {
    listen 443 ssl;
    http2 on;
    server_name ${mail_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://mailpit:8025;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
  fi

  # MLflow route
  if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
    local mlflow_route="${MLFLOW_ROUTE:-mlflow}"

    cat > nginx/sites/mlflow.conf <<EOF
# MLflow
server {
    listen 443 ssl;
    http2 on;
    server_name ${mlflow_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://mlflow:5000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
  fi

  # Functions route
  if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
    local functions_route="${FUNCTIONS_ROUTE:-functions}"

    cat > nginx/sites/functions.conf <<EOF
# Serverless Functions
server {
    listen 443 ssl;
    http2 on;
    server_name ${functions_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://functions:3000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
  fi

  # Monitoring routes
  if [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
    # Grafana
    local grafana_route="${GRAFANA_ROUTE:-grafana}"

    cat > nginx/sites/grafana.conf <<EOF
server {
    listen 443 ssl;
    http2 on;
    server_name ${grafana_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://grafana:3000;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

    # Prometheus
    local prometheus_route="${PROMETHEUS_ROUTE:-prometheus}"

    cat > nginx/sites/prometheus.conf <<EOF
server {
    listen 443 ssl;
    http2 on;
    server_name ${prometheus_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://prometheus:9090;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

    # Alertmanager
    local alertmanager_route="${ALERTMANAGER_ROUTE:-alertmanager}"

    cat > nginx/sites/alertmanager.conf <<EOF
server {
    listen 443 ssl;
    http2 on;
    server_name ${alertmanager_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://alertmanager:9093;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF
  fi
}

# Generate frontend app routes
generate_frontend_routes() {
  local base_domain="${BASE_DOMAIN:-localhost}"
  local ssl_dir
  ssl_dir=$(get_ssl_cert_dir)

  for i in {1..10}; do
    local app_name_var="FRONTEND_APP_${i}_NAME"
    local app_name="${!app_name_var:-}"

    if [[ -n "$app_name" ]]; then
      # Get route and port from environment (substitute at build time)
      local app_route_var="FRONTEND_APP_${i}_ROUTE"
      local app_route="${!app_route_var:-${app_name}}"
      local app_port_var="FRONTEND_APP_${i}_PORT"
      local app_port="${!app_port_var:-$((3000 + i - 1))}"

      cat > "nginx/sites/frontend-${app_name}.conf" <<EOF
# Frontend Application: $app_name
server {
    listen 443 ssl;
    http2 on;
    server_name ${app_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        # Proxy to external frontend app running on host
        proxy_pass http://host.docker.internal:${app_port};
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
        local api_route_var="FRONTEND_APP_${i}_API_ROUTE"
        local api_route="${!api_route_var:-api.${app_name}}"
        local api_port_var="FRONTEND_APP_${i}_API_PORT"
        local api_port="${!api_port_var:-$((4000 + i))}"

        cat >> "nginx/sites/frontend-${app_name}.conf" <<EOF
server {
    listen 443 ssl;
    http2 on;
    server_name ${api_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        # Proxy to frontend's API endpoint
        proxy_pass http://host.docker.internal:${api_port};
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
  local base_domain="${BASE_DOMAIN:-localhost}"
  local ssl_dir
  ssl_dir=$(get_ssl_cert_dir)

  for i in {1..20}; do
    local cs_name_var="CS_${i}_NAME"
    local cs_name="${!cs_name_var:-}"

    if [[ -n "$cs_name" ]]; then
      # Check if service is public
      local cs_public_var="CS_${i}_PUBLIC"
      local cs_public="${!cs_public_var:-true}"

      if [[ "$cs_public" == "true" ]]; then
        # Get route and port from environment (substitute at build time)
        local cs_route_var="CS_${i}_ROUTE"
        local cs_route="${!cs_route_var:-${cs_name}}"
        local cs_port_var="CS_${i}_PORT"
        local cs_port="${!cs_port_var:-$((8000 + i))}"

        cat > "nginx/sites/custom-${cs_name}.conf" <<EOF
# Custom Service: $cs_name
server {
    listen 443 ssl;
    http2 on;
    server_name ${cs_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    location / {
        proxy_pass http://${cs_name}:${cs_port};
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
        proxy_pass http://${cs_name}:${cs_port}/health;
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

# Generate plugin webhook routes
# Routes webhooks to functions service or custom webhook handler
generate_plugin_routes() {
  local plugin_dir="${NSELF_PLUGIN_DIR:-$HOME/.nself/plugins}"
  local base_domain="${BASE_DOMAIN:-localhost}"
  local ssl_dir
  ssl_dir=$(get_ssl_cert_dir)

  # Check if any plugins are installed
  if [[ ! -d "$plugin_dir" ]]; then
    return 0
  fi

  # Get list of installed plugins (directories with plugin.json)
  local installed_plugins=""
  for plugin_path in "$plugin_dir"/*/plugin.json; do
    if [[ -f "$plugin_path" ]]; then
      local plugin_name
      plugin_name=$(dirname "$plugin_path")
      plugin_name=$(basename "$plugin_name")
      # Skip shared utilities directory
      if [[ "$plugin_name" != "_shared" ]]; then
        installed_plugins="${installed_plugins} ${plugin_name}"
      fi
    fi
  done

  # No plugins installed
  if [[ -z "${installed_plugins// /}" ]]; then
    return 0
  fi

  # Determine webhook handler upstream
  # Priority: WEBHOOK_HANDLER_HOST > functions service > hasura
  local webhook_upstream=""
  if [[ -n "${WEBHOOK_HANDLER_HOST:-}" ]]; then
    webhook_upstream="${WEBHOOK_HANDLER_HOST}:${WEBHOOK_HANDLER_PORT:-3000}"
  elif [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
    webhook_upstream="functions:3000"
  else
    # No webhook handler available - skip route generation
    return 0
  fi

  local webhooks_route="${WEBHOOKS_ROUTE:-webhooks}"

  cat > nginx/sites/webhooks.conf <<EOF
# Plugin Webhook Endpoints
# Installed plugins:${installed_plugins}
server {
    listen 443 ssl;
    http2 on;
    server_name ${webhooks_route}.${base_domain};

    ssl_certificate /etc/nginx/ssl/${ssl_dir}/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/${ssl_dir}/privkey.pem;

    # Rate limiting for webhooks
    limit_req zone=api burst=50 nodelay;

    # Common webhook settings
    client_max_body_size 10M;
EOF

  # Add location block for each installed plugin
  for plugin_name in $installed_plugins; do
    cat >> nginx/sites/webhooks.conf <<EOF

    # ${plugin_name} webhook endpoint
    location /${plugin_name} {
        proxy_pass http://${webhook_upstream}/webhooks/${plugin_name};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Webhook-Plugin ${plugin_name};
        proxy_set_header X-Original-URI \$request_uri;

        # Preserve raw body for signature verification
        proxy_set_header Content-Type \$content_type;
        proxy_pass_request_body on;

        # Timeouts for webhook processing
        proxy_connect_timeout 30s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
EOF
  done

  cat >> nginx/sites/webhooks.conf <<EOF

    # Health check
    location /health {
        access_log off;
        return 200 "webhooks healthy\n";
        add_header Content-Type text/plain;
    }
}
EOF
}

# Export all functions
export -f generate_nginx_config
export -f generate_main_nginx_conf
export -f generate_default_server
export -f generate_service_routes
export -f generate_optional_service_routes
export -f generate_frontend_routes
export -f generate_custom_routes
export -f generate_plugin_routes
export -f generate_database_init