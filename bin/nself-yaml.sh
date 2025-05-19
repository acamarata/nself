#!/bin/bash

# nself-yaml.sh - Helper script to generate docker-compose.yml in the project directory and handle seed and host configuration

set -e

# ----------------------------
# Variables
# ----------------------------
OUTPUT_FILE="$PWD/docker-compose.yml"  # Ensure docker-compose.yml is created in the project directory
PROJECT_NAME=${PROJECT_NAME:-nproj}

# Always-enabled services that require named volumes
ALWAYS_VOLUME_SERVICES=("postgres" "traefik" "minio")

# Optional services that require named volumes
OPTIONAL_VOLUME_SERVICES=("haraka" "prometheus" "grafana")

# Paths to scripts
NSELF_SEED_SCRIPT="$(dirname "$(readlink -f "$0")")/nself-seed.sh"
NSELF_HOST_SCRIPT="$(dirname "$(readlink -f "$0")")/nself-host.sh"

# ----------------------------
# Helper Functions
# ----------------------------

# Function to print informational messages
echo_info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

# Function to print error messages
echo_error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

# Function to check mutually exclusive services
check_mutually_exclusive() {
  MAILHOG_ENABLED=${MAILHOG_ENABLED:-false}
  HARAKA_ENABLED=${HARAKA_ENABLED:-false}

  if [ "$MAILHOG_ENABLED" = "true" ] && [ "$HARAKA_ENABLED" = "true" ]; then
    echo_error "Both MailHog and Haraka are enabled. Please enable only one of them."
    exit 1
  fi
}

# Function to convert string to uppercase (compatible with Bash 3.x)
to_uppercase() {
  echo "$1" | tr '[:lower:]' '[:upper:]'
}

# Function to write the header of docker-compose.yml without the version key
write_header() {
  cat <<'EOF' > "$OUTPUT_FILE"
services:
EOF
}

# Function to generate docker-compose.yml
generate_docker_compose() {
  write_header

  # Postgres Service
  #echo_info "Adding postgres service..."
  cat <<EOF >> "$OUTPUT_FILE"
  postgres:
    image: nhost/postgres:${POSTGRES_VERSION:-16.4-202401126-1}
    environment:
      POSTGRES_DB: ${DB_NAME:-postgres}
      POSTGRES_USER: ${DB_USER:-postgres}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-mydbpassword}
    ports:
      - "${DB_PORT:-5432}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - postgres_lib:/var/lib/postgresql
      - ./seeds:/docker-entrypoint-initdb.d
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-postgres}"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped
EOF

  # Hasura Service (Always Enabled)
  #echo_info "Adding hasura service..."
  cat <<EOF >> "$OUTPUT_FILE"

  hasura:
    image: nhost/graphql-engine:${HASURA_VERSION:-v2.44.0-ce}
    ports:
      - "1337:8080"
    depends_on:
      - postgres
    environment:
      HASURA_GRAPHQL_DATABASE_URL: postgres://${DB_USER:-postgres}:${DB_PASSWORD:-mydbpassword}@postgres:${DB_PORT:-5432}/${DB_NAME:-postgres}
      HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET:-nhost-admin-secret}
      HASURA_GRAPHQL_JWT_SECRET: '{"type":"HS256","key":"${HASURA_GRAPHQL_JWT_SECRET:-myjwtsecretthats32characterslong}"}'
      HASURA_GRAPHQL_UNAUTHORIZED_ROLE: ${HASURA_GRAPHQL_UNAUTHORIZED_ROLE:-public}
      HASURA_GRAPHQL_ENABLE_CONSOLE: ${HASURA_GRAPHQL_ENABLE_CONSOLE:-true}
      HASURA_GRAPHQL_LOG_LEVEL: ${HASURA_GRAPHQL_LOG_LEVEL:-info}
      HASURA_GRAPHQL_ENABLE_CORS: ${HASURA_GRAPHQL_ENABLE_CORS:-true}
      HASURA_GRAPHQL_CORS_DOMAIN: "${HASURA_GRAPHQL_CORS_DOMAIN:-*}"
      HASURA_GRAPHQL_ENABLE_RATE_LIMIT: ${HASURA_GRAPHQL_ENABLE_RATE_LIMIT:-true}
      HASURA_GRAPHQL_RATE_LIMIT_REQUESTS: ${HASURA_GRAPHQL_RATE_LIMIT_REQUESTS:-1000}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF

  # Auth Service (Always Enabled)
  #echo_info "Adding auth service..."
  cat <<EOF >> "$OUTPUT_FILE"

  auth:
    image: nhost/hasura-auth:${AUTH_VERSION:-0.36.1}
    depends_on:
      - hasura
      - postgres
    ports:
      - "${AUTH_HOST_PORT:-4000}:${AUTH_CONTAINER_PORT:-4000}"
    environment:
      HASURA_ENDPOINT: http://hasura:8080
      AUTH_ACCESS_TOKEN_EXPIRES_IN: ${AUTH_ACCESS_TOKEN_EXPIRES_IN:-900}
      AUTH_REFRESH_TOKEN_EXPIRES_IN: ${AUTH_REFRESH_TOKEN_EXPIRES_IN:-2592000}
      AUTH_MFA_ENABLED: ${AUTH_MFA_ENABLED:-false}
      AUTH_MFA_TOTP_ISSUER: ${AUTH_MFA_TOTP_ISSUER:-nproj}
      AUTH_PASSWORD_MIN_LENGTH: ${AUTH_PASSWORD_MIN_LENGTH:-8}
      AUTH_PASSWORD_REQUIRE_SPECIAL: ${AUTH_PASSWORD_REQUIRE_SPECIAL:-false}
      AUTH_EMAIL_VERIFICATION_REQUIRED: ${AUTH_EMAIL_VERIFICATION_REQUIRED:-true}
      HASURA_GRAPHQL_JWT_SECRET: '{"type":"HS256","key":"${HASURA_GRAPHQL_JWT_SECRET:-myjwtsecretthats32characterslong}"}'
EOF

  # Optional Auth Providers
  if [ -n "${AUTH_PROVIDER_GOOGLE_CLIENT_ID}" ]; then
    echo "      AUTH_PROVIDER_GOOGLE_CLIENT_ID: ${AUTH_PROVIDER_GOOGLE_CLIENT_ID}" >> "$OUTPUT_FILE"
  fi

  if [ -n "${AUTH_PROVIDER_GOOGLE_CLIENT_SECRET}" ]; then
    echo "      AUTH_PROVIDER_GOOGLE_CLIENT_SECRET: ${AUTH_PROVIDER_GOOGLE_CLIENT_SECRET}" >> "$OUTPUT_FILE"
  fi

  # Complete the auth service section
  echo "    healthcheck:" >> "$OUTPUT_FILE"
  echo "      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost:${AUTH_CONTAINER_PORT:-4000}/healthz\"]" >> "$OUTPUT_FILE"
  echo "      interval: 30s" >> "$OUTPUT_FILE"
  echo "      timeout: 10s" >> "$OUTPUT_FILE"
  echo "      retries: 5" >> "$OUTPUT_FILE"
  echo "    restart: unless-stopped" >> "$OUTPUT_FILE"

  # Storage Service (Always Enabled)
  #echo_info "Adding storage service..."
  cat <<EOF >> "$OUTPUT_FILE"

  storage:
    image: nhost/hasura-storage:${STORAGE_VERSION:-0.6.1}
    depends_on:
      - hasura
    ports:
      - "${STORAGE_PORT:-9000}:9000"
    environment:
      STORAGE_BUCKET_NAME: ${STORAGE_BUCKET_NAME:-nproj-bucket}
      STORAGE_ACCESS_KEY: ${STORAGE_ACCESS_KEY:-storage_access_key}
      STORAGE_SECRET_KEY: ${STORAGE_SECRET_KEY:-storage_secret_key}
      STORAGE_MAX_FILE_SIZE: ${STORAGE_MAX_FILE_SIZE:-10485760}
      STORAGE_PUBLIC_ACCESS: ${STORAGE_PUBLIC_ACCESS:-true}
      STORAGE_PORT: ${STORAGE_PORT:-9000}
    command: serve
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${STORAGE_PORT:-9000}/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF

  # Functions Service (Always Enabled)
  #echo_info "Adding functions service..."
  cat <<EOF >> "$OUTPUT_FILE"

  functions:
    image: nhost/functions:${FUNCTIONS_VERSION:-1.2.0}
    depends_on:
      - auth
      - postgres
    ports:
      - "${FUNCTIONS_HOST_PORT:-4001}:${FUNCTIONS_CONTAINER_PORT:-1337}"
    environment:
      FUNCTIONS_TIMEOUT: ${FUNCTIONS_TIMEOUT:-30}
      FUNCTIONS_MEMORY_LIMIT: ${FUNCTIONS_MEMORY_LIMIT:-128}
      FUNCTIONS_PORT: ${FUNCTIONS_CONTAINER_PORT:-1337}
      FUNCTIONS_API_ROUTE: ${FUNCTIONS_API_ROUTE:-/functions}
    volumes:
      - ./functions:/usr/src/app
    working_dir: /usr/src/app
    command: sh -c "npm install && npm start"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${FUNCTIONS_CONTAINER_PORT:-1337}/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF

  # Dashboard Service (Always Enabled)
  #echo_info "Adding dashboard service..."
  cat <<EOF >> "$OUTPUT_FILE"

  dashboard:
    image: nhost/dashboard:${DASHBOARD_VERSION:-2.8.0}
    ports:
      - "3030:3030"
    depends_on:
      - hasura
    environment:
      HASURA_ENDPOINT: http://hasura:8080
      NEXT_PUBLIC_NHOST_CONFIGSERVER_URL: ${NEXT_PUBLIC_NHOST_CONFIGSERVER_URL:-http://localhost:1337}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3030/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF

  # Traefik Service (Always Enabled)
  #echo_info "Adding traefik service..."
  cat <<EOF >> "$OUTPUT_FILE"

  traefik:
    image: traefik:${TRAEFIK_VERSION:-3.2.1}
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - traefik_data:/letsencrypt
      - .traefik/dynamic:/dynamic
      - .traefik/traefik.yml:/traefik.yml:ro
      - .traefik/certs:/certs  # Mounting certificates for development
    environment:
      TRAEFIK_DASHBOARD_ENABLED: ${TRAEFIK_DASHBOARD_ENABLED:-true}
      SSL_ENABLED: ${SSL_ENABLED:-true}
      SSL_DOMAIN: ${SSL_DOMAIN:-nproj.run}
      SSL_EMAIL: ${SSL_EMAIL:-admin@nproj.run}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:80/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF

  # Optional SSL Cert Paths
  if [ -n "${SSL_CERT_PATH}" ]; then
    echo "      SSL_CERT_PATH: ${SSL_CERT_PATH}" >> "$OUTPUT_FILE"
  fi

  if [ -n "${SSL_KEY_PATH}" ]; then
    echo "      SSL_KEY_PATH: ${SSL_KEY_PATH}" >> "$OUTPUT_FILE"
  fi

  # MailHog Service (Optional)
  MAILHOG_ENABLED=${MAILHOG_ENABLED:-false}
  if [ "$MAILHOG_ENABLED" = "true" ]; then
    #echo_info "Adding mailhog service..."
    cat <<EOF >> "$OUTPUT_FILE"

  mailhog:
    image: anatomicjc/mailhog:${MAILHOG_VERSION:-1.0.1}
    ports:
      - "${MAILHOG_PORT:-1025}:1025"
      - "8025:8025"
    environment:
      MAIL_FROM: ${MAIL_FROM:-noreply@nproj.run}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8025/api/v2/status"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF
  fi

  # Haraka Service (Optional)
  HARAKA_ENABLED=${HARAKA_ENABLED:-false}
  if [ "$HARAKA_ENABLED" = "true" ]; then
    #echo_info "Adding haraka service..."
    cat <<EOF >> "$OUTPUT_FILE"

  haraka:
    image: instrumentisto/haraka:${HARAKA_VERSION:-3.0.3}
    ports:
      - "25:25"
      - "587:587"
    environment:
      HARAKA_DB_HOST: ${DB_HOST:-postgres}
      HARAKA_DB_PORT: ${DB_PORT:-5432}
      HARAKA_DB_USER: ${DB_USER:-postgres}
      HARAKA_DB_PASS: ${DB_PASSWORD:-mydbpassword}
      HARAKA_HOSTNAME: ${HARAKA_HOSTNAME:-mail.nproj.run}
      HARAKA_DB_NAME: ${HARAKA_DB_NAME:-haraka}
    volumes:
      - haraka_data:/data
    depends_on:
      - postgres
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:25/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF
  fi

  # Redis Service (Optional)
  REDIS_ENABLED=${REDIS_ENABLED:-false}
  if [ "$REDIS_ENABLED" = "true" ]; then
    #echo_info "Adding redis service..."
    cat <<EOF >> "$OUTPUT_FILE"

  redis:
    image: redis:${REDIS_VERSION:-7.4.1-alpine3.20}
    ports:
      - "${REDIS_PORT:-6379}:6379"
    environment:
      REDIS_PASSWORD: ${REDIS_PASSWORD}
    command: redis-server --requirepass ${REDIS_PASSWORD}
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "${REDIS_PASSWORD}", "ping"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF
  fi

  # Meilisearch Service (Optional)
  MEILISEARCH_ENABLED=${MEILISEARCH_ENABLED:-false}
  if [ "$MEILISEARCH_ENABLED" = "true" ]; then
    #echo_info "Adding meilisearch service..."
    cat <<EOF >> "$OUTPUT_FILE"

  meilisearch:
    image: getmeili/meilisearch:${MEILISEARCH_VERSION:-v1.11.3}
    ports:
      - "${MEILISEARCH_PORT:-7700}:7700"
    environment:
      MEILISEARCH_MASTER_KEY: ${MEILISEARCH_MASTER_KEY}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${MEILISEARCH_PORT:-7700}/health"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF
  fi

  # Prometheus Service (Optional)
  PROMETHEUS_ENABLED=${PROMETHEUS_ENABLED:-false}
  if [ "$PROMETHEUS_ENABLED" = "true" ]; then
    #echo_info "Adding prometheus service..."
    cat <<EOF >> "$OUTPUT_FILE"

  prometheus:
    image: prom/prometheus:${PROMETHEUS_VERSION:-v3.0.1}
    ports:
      - "${PROMETHEUS_PORT:-9090}:9090"
    volumes:
      - prometheus_data:/prometheus
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.enable-lifecycle'
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${PROMETHEUS_PORT:-9090}/-/healthy"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF
  fi

  # Grafana Service (Optional)
  GRAFANA_ENABLED=${GRAFANA_ENABLED:-false}
  if [ "$GRAFANA_ENABLED" = "true" ]; then
    #echo_info "Adding grafana service..."
    cat <<EOF >> "$OUTPUT_FILE"

  grafana:
    image: grafana/grafana:${GRAFANA_VERSION:-11.3.1}
    ports:
      - "${GRAFANA_PORT:-3000}:3000"
    environment:
      GF_SECURITY_ADMIN_USER: ${GF_SECURITY_ADMIN_USER}
      GF_SECURITY_ADMIN_PASSWORD: ${GF_SECURITY_ADMIN_PASSWORD}
    volumes:
      - grafana_data:/var/lib/grafana
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:${GRAFANA_PORT:-3000}/ || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped
EOF
  fi

  # MinIO Service (Always Enabled)
  #echo_info "Adding MinIO service..."
  cat <<EOF >> "$OUTPUT_FILE"

  minio:
    image: minio/minio:${MINIO_VERSION:-latest}
    ports:
      - "${MINIO_API_PORT:-9001}:9001"
      - "${MINIO_CONSOLE_PORT:-9002}:9002"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-minioadmin123}
      MINIO_REGION_NAME: ${MINIO_REGION_NAME:-us-east-1}
    volumes:
      - minio_data:/data
    command: server --console-address ":${MINIO_CONSOLE_PORT:-9002}" ${MINIO_DATA_DIR:-/data}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${MINIO_API_PORT:-9001}/minio/health/live"]
      interval: 30s
      timeout: 10s
      retries: 5
    depends_on:
      - hasura
    restart: unless-stopped
EOF

  # Write volumes section with explicit names
  write_volumes

  echo_info "docker-compose.yml generated successfully in /"

  # Execute nself-seed.sh to handle seed creation
  execute_seed_script

  # Execute nself-host.sh to configure routing
  execute_host_script
}

# Function to write volumes section with explicit names
write_volumes() {
  echo "" >> "$OUTPUT_FILE"
  echo "volumes:" >> "$OUTPUT_FILE"

  # Define volumes for always-enabled services
  for volume in "${ALWAYS_VOLUME_SERVICES[@]}"; do
    echo "  ${volume}_data:" >> "$OUTPUT_FILE"
    echo "    name: ${PROJECT_NAME}_${volume}_data" >> "$OUTPUT_FILE"
  done

  # Define volumes for optional services if enabled
  for volume in "${OPTIONAL_VOLUME_SERVICES[@]}"; do
    ENABLED_VAR=$(to_uppercase "$volume")_ENABLED
    enabled=${!ENABLED_VAR:-false}
    if [ "$enabled" = "true" ]; then
      echo "  ${volume}_data:" >> "$OUTPUT_FILE"
      echo "    name: ${PROJECT_NAME}_${volume}_data" >> "$OUTPUT_FILE"
    fi
  done

  # Additional volumes for specific services
  if [ "$MEILISEARCH_ENABLED" = "true" ]; then
    echo "  meilisearch_data:" >> "$OUTPUT_FILE"
    echo "    name: ${PROJECT_NAME}_meilisearch_data" >> "$OUTPUT_FILE"
  fi

  if [ "$PROMETHEUS_ENABLED" = "true" ]; then
    echo "  prometheus_data:" >> "$OUTPUT_FILE"
    echo "    name: ${PROJECT_NAME}_prometheus_data" >> "$OUTPUT_FILE"
  fi

  if [ "$GRAFANA_ENABLED" = "true" ]; then
    echo "  grafana_data:" >> "$OUTPUT_FILE"
    echo "    name: ${PROJECT_NAME}_grafana_data" >> "$OUTPUT_FILE"
  fi

  # Explicit volume for postgres_lib
  echo "  postgres_lib:" >> "$OUTPUT_FILE"
  echo "    name: ${PROJECT_NAME}_postgres_lib" >> "$OUTPUT_FILE"
}

# Function to execute nself-seed.sh
execute_seed_script() {
  if [ -f "$NSELF_SEED_SCRIPT" ]; then
    echo_info "Executing nself-seed.sh to handle seed creation..."
    bash "$NSELF_SEED_SCRIPT"
  else
    echo_error "nself-seed.sh not found in $(dirname "$NSELF_SEED_SCRIPT")."
    exit 1
  fi
}

# Function to execute nself-host.sh
execute_host_script() {
  if [ -f "$NSELF_HOST_SCRIPT" ]; then
    #echo_info "Executing nself-host.sh to configure routing..."
    bash "$NSELF_HOST_SCRIPT"
  else
    echo_error "nself-host.sh not found in $(dirname "$NSELF_HOST_SCRIPT")."
    exit 1
  fi
}

# ----------------------------
# Main Execution
# ----------------------------

# Check mutually exclusive services
check_mutually_exclusive

# Generate docker-compose.yml
generate_docker_compose

echo ""
