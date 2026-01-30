#!/usr/bin/env bash
# docker-compose.sh - Docker Compose generation for build

# Source platform compatibility functions
source "$(dirname "${BASH_SOURCE[0]}")/../utils/platform-compat.sh" 2>/dev/null || true

# Generate docker-compose.yml
generate_docker_compose() {
  # Determine the correct path to compose-generate.sh
  local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local compose_script="${script_dir}/../../services/docker/compose-generate.sh"

  # Fallback if first path doesn't work
  if [[ ! -f "$compose_script" ]]; then
    compose_script="${LIB_DIR}/../../services/docker/compose-generate.sh"
  fi

  if [[ -f "$compose_script" ]]; then
    bash "$compose_script"
    return $?
  else
    echo "Error: compose-generate.sh not found" >&2
    echo "  Tried: ${script_dir}/../../services/docker/compose-generate.sh" >&2
    echo "  Tried: ${LIB_DIR}/../../services/docker/compose-generate.sh" >&2
    return 1
  fi
}

# Add nginx service
add_nginx_service() {
  local file="$1"

  cat >>"$file" <<EOF

  nginx:
    image: nginx:alpine
    container_name: \${PROJECT_NAME}_nginx
    restart: unless-stopped
    ports:
      - "\${NGINX_PORT:-80}:80"
      - "\${NGINX_SSL_PORT:-443}:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./nginx/sites:/etc/nginx/sites:ro
      - ./ssl/certificates:/etc/nginx/ssl:ro
      - nginx_cache:/var/cache/nginx
    networks:
      - nself_network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF

  # Add dependencies if other services exist
  local deps=()
  [[ "${HASURA_ENABLED:-false}" == "true" ]] && deps+=("hasura")
  [[ "${AUTH_ENABLED:-false}" == "true" ]] && deps+=("auth")
  [[ "${MINIO_ENABLED:-false}" == "true" ]] && deps+=("minio")

  if [[ ${#deps[@]} -gt 0 ]]; then
    echo "    depends_on:" >>"$file"
    for dep in "${deps[@]}"; do
      echo "      - $dep" >>"$file"
    done
  fi
}

# Add PostgreSQL service
add_postgres_service() {
  local file="$1"

  cat >>"$file" <<EOF

  postgres:
    image: postgres:\${POSTGRES_VERSION:-15}-alpine
    container_name: \${PROJECT_NAME}_postgres
    restart: unless-stopped
    ports:
      - "\${POSTGRES_PORT:-5432}:5432"
    environment:
      POSTGRES_DB: \${POSTGRES_DB:-\${PROJECT_NAME}}
      POSTGRES_USER: \${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD:-postgres}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-db.sql:/docker-entrypoint-initdb.d/init.sql:ro
    networks:
      - nself_network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U \${POSTGRES_USER:-postgres}"]
      interval: 10s
      timeout: 5s
      retries: 5
EOF
}

# Add Redis service
# SECURITY: Redis is configured with localhost-only binding and password auth
add_redis_service() {
  local file="$1"

  # Build Redis command with security options
  local redis_cmd="redis-server --appendonly yes --protected-mode yes"
  if [[ -n "${REDIS_PASSWORD:-}" ]]; then
    redis_cmd="${redis_cmd} --requirepass \${REDIS_PASSWORD}"
  fi

  cat >>"$file" <<EOF

  # Redis Cache - SECURITY: Bound to localhost only
  redis:
    image: redis:\${REDIS_VERSION:-7}-alpine
    container_name: \${PROJECT_NAME}_redis
    restart: unless-stopped
    command: ${redis_cmd}
    ports:
      # SECURITY: Bind to localhost only - prevents external access
      - "127.0.0.1:\${REDIS_PORT:-6379}:6379"
    volumes:
      - redis_data:/data
    networks:
      - nself_network
EOF

  # Use appropriate healthcheck based on password
  if [[ -n "${REDIS_PASSWORD:-}" ]]; then
    cat >>"$file" <<EOF
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "\${REDIS_PASSWORD}", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
EOF
  else
    cat >>"$file" <<EOF
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
EOF
  fi
}

# Add Hasura service
add_hasura_service() {
  local file="$1"

  cat >>"$file" <<EOF

  hasura:
    image: hasura/graphql-engine:\${HASURA_VERSION:-latest}
    container_name: \${PROJECT_NAME}_hasura
    restart: unless-stopped
    ports:
      - "\${HASURA_PORT:-8080}:8080"
    environment:
      HASURA_GRAPHQL_DATABASE_URL: postgres://\${POSTGRES_USER:-postgres}:\${POSTGRES_PASSWORD:-postgres}@postgres:5432/\${POSTGRES_DB:-\${PROJECT_NAME}}
      HASURA_GRAPHQL_ENABLE_CONSOLE: "\${HASURA_CONSOLE:-true}"
      HASURA_GRAPHQL_DEV_MODE: "\${HASURA_DEV_MODE:-true}"
      HASURA_GRAPHQL_ADMIN_SECRET: \${HASURA_ADMIN_SECRET:-myadminsecret}
      HASURA_GRAPHQL_JWT_SECRET: \${HASURA_JWT_SECRET:-}
      HASURA_GRAPHQL_UNAUTHORIZED_ROLE: \${HASURA_UNAUTHORIZED_ROLE:-anonymous}
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - nself_network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF
}

# Add Auth service
add_auth_service() {
  local file="$1"

  cat >>"$file" <<EOF

  auth:
    image: nhost/hasura-auth:\${AUTH_VERSION:-latest}
    container_name: \${PROJECT_NAME}_auth
    restart: unless-stopped
    ports:
      - "\${AUTH_PORT:-4000}:4000"
    environment:
      DATABASE_URL: postgres://\${POSTGRES_USER:-postgres}:\${POSTGRES_PASSWORD:-postgres}@postgres:5432/\${POSTGRES_DB:-\${PROJECT_NAME}}
      HASURA_GRAPHQL_ADMIN_SECRET: \${HASURA_ADMIN_SECRET:-myadminsecret}
      HASURA_GRAPHQL_URL: http://hasura:8080/v1/graphql
      JWT_SECRET: \${AUTH_JWT_SECRET:-\${HASURA_JWT_SECRET:-}}
      AUTH_HOST: \${AUTH_HOST:-0.0.0.0}
      AUTH_PORT: 4000
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - nself_network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:4000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF
}

# Add Storage service
add_storage_service() {
  local file="$1"

  cat >>"$file" <<EOF

  storage:
    image: nhost/hasura-storage:\${STORAGE_VERSION:-latest}
    container_name: \${PROJECT_NAME}_storage
    restart: unless-stopped
    ports:
      - "\${STORAGE_PORT:-5000}:5000"
    environment:
      DATABASE_URL: postgres://\${POSTGRES_USER:-postgres}:\${POSTGRES_PASSWORD:-postgres}@postgres:5432/\${POSTGRES_DB:-\${PROJECT_NAME}}
      HASURA_METADATA: "1"
      HASURA_GRAPHQL_URL: http://hasura:8080/v1
      HASURA_GRAPHQL_ADMIN_SECRET: \${HASURA_ADMIN_SECRET:-myadminsecret}
      S3_ENDPOINT: \${S3_ENDPOINT:-}
      S3_ACCESS_KEY: \${S3_ACCESS_KEY:-}
      S3_SECRET_KEY: \${S3_SECRET_KEY:-}
      S3_BUCKET: \${S3_BUCKET:-\${PROJECT_NAME}-storage}
      S3_REGION: \${S3_REGION:-us-east-1}
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - storage_data:/data
    networks:
      - nself_network
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:5000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF
}

# Add custom services
add_custom_services() {
  local file="$1"

  # Check for custom service definitions
  if [[ -d "services" ]]; then
    for service_file in services/*.yml services/*.yaml; do
      if [[ -f "$service_file" ]]; then
        echo "" >>"$file"
        cat "$service_file" >>"$file"
      fi
    done
  fi
}

# Add volumes section
add_volumes_section() {
  local file="$1"

  echo "" >>"$file"
  echo "volumes:" >>"$file"

  [[ "${POSTGRES_ENABLED:-true}" == "true" ]] && echo "  postgres_data:" >>"$file"
  [[ "${REDIS_ENABLED:-false}" == "true" ]] && echo "  redis_data:" >>"$file"
  [[ "${MINIO_ENABLED:-false}" == "true" ]] && echo "  minio_data:" >>"$file"
  [[ "${NGINX_ENABLED:-true}" == "true" ]] && echo "  nginx_cache:" >>"$file"
}

# Add networks section
add_networks_section() {
  local file="$1"

  cat >>"$file" <<EOF

networks:
  nself_network:
    driver: bridge
    name: \${PROJECT_NAME}_network
EOF
}

# Validate docker-compose.yml
validate_docker_compose() {
  local compose_file="${1:-docker-compose.yml}"

  if [[ ! -f "$compose_file" ]]; then
    show_error "docker-compose.yml not found"
    return 1
  fi

  # Try to validate the compose file
  if command_exists docker-compose; then
    if docker-compose -f "$compose_file" config >/dev/null 2>&1; then
      return 0
    else
      show_error "docker-compose.yml validation failed"
      return 1
    fi
  elif docker compose version >/dev/null 2>&1; then
    if docker compose -f "$compose_file" config >/dev/null 2>&1; then
      return 0
    else
      show_error "docker-compose.yml validation failed"
      return 1
    fi
  fi

  return 0
}

# Export functions
export -f generate_docker_compose
export -f add_nginx_service
export -f add_postgres_service
export -f add_redis_service
export -f add_hasura_service
export -f add_auth_service
export -f add_storage_service
export -f add_custom_services
export -f add_volumes_section
export -f add_networks_section
export -f validate_docker_compose
