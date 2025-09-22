#!/usr/bin/env bash
# core-services.sh - Generate core service definitions for docker-compose
# This module handles PostgreSQL, Hasura, Auth, MinIO, and Redis services

# Sanitize database name (replace hyphens with underscores for PostgreSQL compatibility)
sanitize_db_name() {
  echo "$1" | tr '-' '_'
}

# Generate PostgreSQL service configuration
generate_postgres_service() {
  local enabled="${POSTGRES_ENABLED:-true}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # PostgreSQL Database
  postgres:
    image: postgres:${POSTGRES_VERSION:-16-alpine}
    container_name: \${PROJECT_NAME}_postgres
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    environment:
      POSTGRES_USER: \${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD}
      POSTGRES_DB: \${POSTGRES_DB:-\${PROJECT_NAME}}
      POSTGRES_HOST_AUTH_METHOD: trust
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres/init:/docker-entrypoint-initdb.d:ro
    ports:
      - "\${POSTGRES_PORT:-5432}:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U \${POSTGRES_USER:-postgres}"]
      interval: 10s
      timeout: 5s
      retries: 5
EOF
}

# Generate Hasura GraphQL Engine service
generate_hasura_service() {
  local enabled="${HASURA_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Hasura GraphQL Engine
  hasura:
    image: hasura/graphql-engine:${HASURA_VERSION:-v2.36.0}
    container_name: \${PROJECT_NAME}_hasura
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      HASURA_GRAPHQL_DATABASE_URL: \${DATABASE_URL}
      HASURA_GRAPHQL_ADMIN_SECRET: \${HASURA_GRAPHQL_ADMIN_SECRET}
      HASURA_GRAPHQL_ENABLE_CONSOLE: "true"
      HASURA_GRAPHQL_DEV_MODE: \${HASURA_DEV_MODE:-false}
      HASURA_GRAPHQL_ENABLE_TELEMETRY: "false"
      HASURA_GRAPHQL_CORS_DOMAIN: "*"
      HASURA_GRAPHQL_LOG_LEVEL: \${HASURA_LOG_LEVEL:-info}
EOF

  # Add auth configuration based on auth mode
  if [[ "${AUTH_ENABLED:-false}" == "true" ]]; then
    # Use webhook auth mode
    cat <<EOF
      HASURA_GRAPHQL_AUTH_HOOK: http://auth:4000/webhook
      HASURA_GRAPHQL_AUTH_HOOK_MODE: GET
EOF
  else
    # No auth - just use admin secret
    cat <<EOF
      HASURA_GRAPHQL_UNAUTHORIZED_ROLE: public
EOF
  fi

  cat <<EOF
    ports:
      - "\${HASURA_PORT:-8080}:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate Auth service configuration
generate_auth_service() {
  local enabled="${AUTH_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  # Determine which auth image to use
  local auth_image="${AUTH_IMAGE:-nhost/hasura-auth:latest}"

  cat <<EOF

  # Hasura Auth Service
  auth:
    image: ${auth_image}
    container_name: \${PROJECT_NAME}_auth
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
EOF

  # Note: Don't add hasura dependency to avoid circular dependency
  # Hasura depends on auth webhook, auth doesn't need hasura

  cat <<EOF
    environment:
      AUTH_HOST: "0.0.0.0"
      AUTH_PORT: "4000"
      AUTH_LOG_LEVEL: \${AUTH_LOG_LEVEL:-info}
      DATABASE_URL: \${DATABASE_URL}
      AUTH_SERVER_URL: \${AUTH_SERVER_URL:-http://localhost:4000}
      AUTH_CLIENT_URL: \${AUTH_CLIENT_URL:-http://localhost:3000}
      AUTH_JWT_SECRET: \${AUTH_JWT_SECRET}
      AUTH_ACCESS_TOKEN_EXPIRES_IN: \${AUTH_ACCESS_TOKEN_EXPIRES_IN:-900}
      AUTH_REFRESH_TOKEN_EXPIRES_IN: \${AUTH_REFRESH_TOKEN_EXPIRES_IN:-2592000}
      AUTH_SMTP_HOST: \${AUTH_SMTP_HOST:-mailpit}
      AUTH_SMTP_PORT: \${AUTH_SMTP_PORT:-1025}
      AUTH_SMTP_USER: \${AUTH_SMTP_USER}
      AUTH_SMTP_PASS: \${AUTH_SMTP_PASS}
      AUTH_SMTP_SECURE: \${AUTH_SMTP_SECURE:-false}
      AUTH_SMTP_SENDER: \${AUTH_SMTP_SENDER:-noreply@\${BASE_DOMAIN:-localhost}}
      AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED: \${AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED:-false}
EOF

  # Add OAuth providers if configured
  if [[ -n "${AUTH_PROVIDER_GOOGLE_CLIENT_ID:-}" ]]; then
    cat <<EOF
      AUTH_PROVIDER_GOOGLE_ENABLED: "true"
      AUTH_PROVIDER_GOOGLE_CLIENT_ID: \${AUTH_PROVIDER_GOOGLE_CLIENT_ID}
      AUTH_PROVIDER_GOOGLE_CLIENT_SECRET: \${AUTH_PROVIDER_GOOGLE_CLIENT_SECRET}
EOF
  fi

  if [[ -n "${AUTH_PROVIDER_GITHUB_CLIENT_ID:-}" ]]; then
    cat <<EOF
      AUTH_PROVIDER_GITHUB_ENABLED: "true"
      AUTH_PROVIDER_GITHUB_CLIENT_ID: \${AUTH_PROVIDER_GITHUB_CLIENT_ID}
      AUTH_PROVIDER_GITHUB_CLIENT_SECRET: \${AUTH_PROVIDER_GITHUB_CLIENT_SECRET}
EOF
  fi

  cat <<EOF
    ports:
      - "\${AUTH_PORT:-4000}:4000"
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:4000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate MinIO service configuration
generate_minio_service() {
  local enabled="${MINIO_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # MinIO Object Storage
  minio:
    image: minio/minio:${MINIO_VERSION:-latest}
    container_name: \${PROJECT_NAME}_minio
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    environment:
      MINIO_ROOT_USER: \${MINIO_ROOT_USER:-minioadmin}
      MINIO_ROOT_PASSWORD: \${MINIO_ROOT_PASSWORD:-minioadmin}
      MINIO_DEFAULT_BUCKETS: \${MINIO_DEFAULT_BUCKETS:-uploads}
      MINIO_REGION: \${MINIO_REGION:-us-east-1}
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data
    ports:
      - "\${MINIO_PORT:-9000}:9000"
      - "\${MINIO_CONSOLE_PORT:-9001}:9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3
EOF

  # Add bucket initialization if configured
  if [[ -n "${MINIO_DEFAULT_BUCKETS:-}" ]]; then
    cat <<EOF

  # MinIO Client for bucket initialization
  minio-client:
    image: minio/mc:latest
    container_name: \${PROJECT_NAME}_minio_client
    restart: "no"
    profiles: ["init"]
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      /usr/bin/mc config host add myminio http://minio:9000 \${MINIO_ROOT_USER:-minioadmin} \${MINIO_ROOT_PASSWORD:-minioadmin};
      /usr/bin/mc mb -p myminio/\${MINIO_DEFAULT_BUCKETS:-uploads};
      /usr/bin/mc anonymous set download myminio/\${MINIO_DEFAULT_BUCKETS:-uploads};
      exit 0;
      "
EOF
  fi
}

# Generate Redis service configuration
generate_redis_service() {
  local enabled="${REDIS_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Redis Cache
  redis:
    image: redis:${REDIS_VERSION:-7-alpine}
    container_name: \${PROJECT_NAME}_redis
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "\${REDIS_PORT:-6379}:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
EOF
}

# Export functions
export -f generate_postgres_service
export -f generate_hasura_service
export -f generate_auth_service
export -f generate_minio_service
export -f generate_redis_service