#!/usr/bin/env bash
# utility-services.sh - Generate utility service definitions
# This module handles Mailpit, Adminer, BullMQ Dashboard, and other utility services

# Generate Mailpit email testing service
generate_mailpit_service() {
  local enabled="${MAILPIT_ENABLED:-true}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Mailpit - Email Testing Tool
  mailpit:
    image: axllent/mailpit:${MAILPIT_VERSION:-latest}
    container_name: \${PROJECT_NAME}_mailpit
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    environment:
      MP_SMTP_AUTH_ACCEPT_ANY: 1
      MP_SMTP_AUTH_ALLOW_INSECURE: 1
      MP_UI_BIND_ADDR: 0.0.0.0:8025
      MP_SMTP_BIND_ADDR: 0.0.0.0:1025
    ports:
      - "\${MAILPIT_SMTP_PORT:-1025}:1025"
      - "\${MAILPIT_UI_PORT:-8025}:8025"
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "8025"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}



# Generate nself Admin dashboard
generate_nself_admin_service() {
  local enabled="${NSELF_ADMIN_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # nself Admin - Project Management Dashboard
  nself-admin:
    image: acamarata/nself-admin:${NSELF_ADMIN_VERSION:-latest}
    container_name: \${PROJECT_NAME}_nself_admin
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
      hasura:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://\${POSTGRES_USER}:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}
      HASURA_GRAPHQL_ENDPOINT: http://hasura:8080/v1/graphql
      HASURA_GRAPHQL_ADMIN_SECRET: \${HASURA_GRAPHQL_ADMIN_SECRET}
      PORT: 3100
    ports:
      - "\${NSELF_ADMIN_PORT:-3101}:3100"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3100/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate Functions service (serverless functions runtime)
generate_functions_service() {
  local enabled="${FUNCTIONS_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  # Check if we should use fallback functions service
  local use_fallback="${FUNCTIONS_USE_FALLBACK:-false}"

  # If fallback is enabled or ENV is demo, use the fallback service
  if [[ "$use_fallback" == "true" ]] || [[ "${ENV:-}" == "demo" ]] || [[ "${DEMO_CONTENT:-false}" == "true" ]]; then
    # Generate fallback functions service
    cat <<EOF

  # Functions - Serverless Functions Runtime (Fallback)
  functions:
    build:
      context: ./fallback-services
      dockerfile: Dockerfile.functions
    container_name: \${PROJECT_NAME}_functions
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
      hasura:
        condition: service_healthy
    environment:
      PORT: 3000
      NODE_ENV: \${ENV:-development}
      DATABASE_URL: postgres://\${POSTGRES_USER}:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}
      HASURA_GRAPHQL_ENDPOINT: http://hasura:8080/v1/graphql
      HASURA_GRAPHQL_ADMIN_SECRET: \${HASURA_GRAPHQL_ADMIN_SECRET}
    volumes:
      - ./functions:/opt/project
    ports:
      - "\${FUNCTIONS_PORT:-3008}:3000"
EOF
  else
    # Use original nhost/functions
    cat <<EOF

  # Functions - Serverless Functions Runtime
  functions:
    image: nhost/functions:\${FUNCTIONS_VERSION:-latest}
    container_name: \${PROJECT_NAME}_functions
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
      hasura:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://\${POSTGRES_USER}:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}
      HASURA_GRAPHQL_ENDPOINT: http://hasura:8080/v1/graphql
      HASURA_GRAPHQL_ADMIN_SECRET: \${HASURA_GRAPHQL_ADMIN_SECRET}
      NODE_ENV: \${ENV:-development}
      PORT: 3008
    volumes:
      - ./functions:/opt/project
    ports:
      - "\${FUNCTIONS_PORT:-3008}:3008"
    healthcheck:
      test: ["CMD-SHELL", "node -e 'require(\"http\").get(\"http://localhost:3000/healthz\", (r) => process.exit(r.statusCode === 200 ? 0 : 1)).on(\"error\", () => process.exit(1))' || curl -f http://localhost:3000/healthz || wget -q --spider http://localhost:3000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 40s
EOF
  fi
}




# Generate backup service
generate_backup_service() {
  local enabled="${BACKUP_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Backup Service - Automated Database Backups
  backup:
    image: postgres:${POSTGRES_VERSION:-16-alpine}
    container_name: \${PROJECT_NAME}_backup
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      PGHOST: postgres
      PGUSER: \${POSTGRES_USER:-postgres}
      PGPASSWORD: \${POSTGRES_PASSWORD}
      PGDATABASE: \${POSTGRES_DB:-\${PROJECT_NAME}}
      BACKUP_SCHEDULE: \${BACKUP_SCHEDULE:-0 2 * * *}
      BACKUP_RETENTION_DAYS: \${BACKUP_RETENTION_DAYS:-7}
    volumes:
      - ./backups:/backups
      - ./scripts/backup.sh:/usr/local/bin/backup.sh:ro
    entrypoint: >
      sh -c "
        apk add --no-cache dcron &&
        echo '\${BACKUP_SCHEDULE} /usr/local/bin/backup.sh' | crontab - &&
        crond -f -l 2
      "
EOF
}

# Generate MLflow service
generate_mlflow_service() {
  local enabled="${MLFLOW_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  # Ensure MLflow directory and Dockerfile exist
  mkdir -p mlflow
  cat > mlflow/Dockerfile <<'DOCKERFILE'
FROM ghcr.io/mlflow/mlflow:latest

# Install PostgreSQL adapter
RUN pip install --no-cache-dir psycopg2-binary

# Ensure artifacts directory exists
RUN mkdir -p /mlflow/artifacts

# Set working directory
WORKDIR /mlflow
DOCKERFILE

  cat <<EOF

  # MLflow - Machine Learning Lifecycle Platform
  mlflow:
    build:
      context: ./mlflow
      dockerfile: Dockerfile
    container_name: \${PROJECT_NAME}_mlflow
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      MLFLOW_BACKEND_STORE_URI: postgresql://\${POSTGRES_USER:-postgres}:\${POSTGRES_PASSWORD}@postgres:5432/mlflow
      MLFLOW_DEFAULT_ARTIFACT_ROOT: /mlflow/artifacts
    command: [
      "mlflow", "server",
      "--backend-store-uri", "postgresql://\${POSTGRES_USER:-postgres}:\${POSTGRES_PASSWORD}@postgres:5432/mlflow",
      "--default-artifact-root", "/mlflow/artifacts",
      "--host", "0.0.0.0",
      "--port", "\${MLFLOW_PORT:-5005}",
      "--serve-artifacts"
    ]
    volumes:
      - mlflow_data:/mlflow/artifacts
    ports:
      - "\${MLFLOW_PORT:-5005}:\${MLFLOW_PORT:-5005}"
    healthcheck:
      test: ["CMD-SHELL", "python -c 'import urllib.request; urllib.request.urlopen(\"http://localhost:\${MLFLOW_PORT:-5005}/health\")' || wget --spider -q http://localhost:\${MLFLOW_PORT:-5005}/health || curl -f http://localhost:\${MLFLOW_PORT:-5005}/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
EOF
}

# Generate search services (Meilisearch by default)
generate_search_services() {
  # Meilisearch
  if [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]]; then
    cat <<EOF

  # Meilisearch - Lightning Fast Search
  meilisearch:
    image: getmeili/meilisearch:\${MEILISEARCH_VERSION:-v1.5}
    container_name: \${PROJECT_NAME}_meilisearch
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    environment:
      MEILI_MASTER_KEY: \${MEILISEARCH_MASTER_KEY:-changeme}
      MEILI_ENV: \${MEILI_ENV:-development}
      MEILI_HTTP_ADDR: 0.0.0.0:7700
      MEILI_NO_ANALYTICS: true
    volumes:
      - meilisearch_data:/meili_data
    ports:
      - "\${MEILISEARCH_PORT:-7700}:7700"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:7700/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
  fi
}

# Main function to generate all utility services in display order
generate_utility_services() {
  generate_nself_admin_service
  generate_minio_service
  generate_redis_service
  generate_functions_service
  generate_mailpit_service
  generate_search_services
  generate_mlflow_service
}

# Export functions
export -f generate_mailpit_service
export -f generate_nself_admin_service
export -f generate_functions_service
export -f generate_mlflow_service
export -f generate_search_services
export -f generate_utility_services