#!/usr/bin/env bash

# services-compose-inline.sh - Append services to main docker-compose.yml

# This script is called from compose.sh to add services inline
# It appends directly to the existing docker-compose.yml

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"



# Generate Functions service if enabled
if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
  FUNCTIONS_PORT="${FUNCTIONS_PORT:-4300}"
  cat >>docker-compose.yml <<EOF

  ${PROJECT_NAME:-nself}-functions:
    image: ${PROJECT_NAME:-nself}/functions:latest
    build:
      context: ./functions
      dockerfile: Dockerfile
      tags:
        - ${PROJECT_NAME:-nself}/functions:latest
        - ${PROJECT_NAME:-nself}/functions:dev
    container_name: ${PROJECT_NAME:-nself}_functions
    restart: unless-stopped
    environment:
      - NODE_ENV=${ENVIRONMENT:-development}
      - FUNCTIONS_PORT=$FUNCTIONS_PORT
      - DATABASE_URL=${DATABASE_URL:-postgres://postgres:${POSTGRES_PASSWORD:-changeme}@postgres:5432/nhost}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_ADMIN_SECRET:-changeme}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
    ports:
      - "$FUNCTIONS_PORT:$FUNCTIONS_PORT"
    depends_on:
      - postgres
      - hasura
      - redis
    networks:
      - default
    volumes:
      - ./functions:/app/functions:ro
      - ./functions/package.json:/app/package.json:ro
      - ./functions/index.js:/app/index.js:ro
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:$FUNCTIONS_PORT/health"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF
fi

# Generate Dashboard service if enabled
if [[ "${DASHBOARD_ENABLED:-false}" == "true" ]]; then
  DASHBOARD_PORT="${DASHBOARD_PORT:-4500}"
  cat >>docker-compose.yml <<EOF

  ${PROJECT_NAME:-nself}-dashboard:
    image: ${PROJECT_NAME:-nself}/dashboard:latest
    build:
      context: ./dashboard
      dockerfile: Dockerfile
      tags:
        - ${PROJECT_NAME:-nself}/dashboard:latest
        - ${PROJECT_NAME:-nself}/dashboard:dev
    container_name: ${PROJECT_NAME:-nself}_dashboard
    restart: unless-stopped
    environment:
      - NODE_ENV=${ENVIRONMENT:-development}
      - VITE_API_URL=${VITE_API_URL:-http://localhost/api}
      - VITE_HASURA_URL=${VITE_HASURA_URL:-http://localhost:8080/v1/graphql}
    ports:
      - "$DASHBOARD_PORT:80"
    depends_on:
      - nginx
    networks:
      - default
    # Dashboard is served from nginx, no source volume needed
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:80/"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF
fi

# nself-admin service
if [[ "$NSELF_ADMIN_ENABLED" == "true" ]]; then
  cat >>docker-compose.yml <<EOF

  # nself Admin UI
  # Note: Requires Docker socket access for container management
  nself-admin:
    image: acamarata/nself-admin:latest
    container_name: \${PROJECT_NAME}_admin
    restart: unless-stopped
    ports:
      - "\${NSELF_ADMIN_PORT:-3100}:3021"
    environment:
      - PROJECT_DIR=/workspace
      - PROJECT_NAME=\${PROJECT_NAME}
      - BASE_DOMAIN=\${BASE_DOMAIN}
      - ENV=\${ENV}
      - ADMIN_SECRET_KEY=\${ADMIN_SECRET_KEY}
      - ADMIN_PASSWORD_HASH=\${ADMIN_PASSWORD_HASH}
      - DATABASE_URL=\${HASURA_GRAPHQL_DATABASE_URL}
      - HASURA_ENDPOINT=http://hasura:8080
      - HASURA_ADMIN_SECRET=\${HASURA_GRAPHQL_ADMIN_SECRET}
      - DOCKER_HOST=unix:///var/run/docker.sock
    volumes:
      - ./:/workspace:rw
      - nself-admin-data:/app/data
      - /var/run/docker.sock:/var/run/docker.sock:ro
    depends_on:
      - postgres
      - hasura
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3021/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
fi

# Note: CS_N custom services are handled by service-builder.sh
# which generates docker-compose.custom.yml
