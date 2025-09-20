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
      test: ["CMD", "wget", "--spider", "-q", "http://127.0.0.1:$FUNCTIONS_PORT/health"]
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
      - ADMIN_SECRET_KEY=\${ADMIN_SECRET_KEY:-admin-secret-key-change-me}
      - ADMIN_PASSWORD_HASH=\${ADMIN_PASSWORD_HASH:-\$\$2b\$\$10\$\$EpRnTzVlqHNP0.fUbXUwSOyuiXe/QLSUG6xNekdHgTGmrpHEfIoxm}
      - DATABASE_URL=postgres://\${POSTGRES_USER:-postgres}:\${POSTGRES_PASSWORD:-changeme}@postgres:5432/\${POSTGRES_DB:-nhost}
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
      test: ["CMD", "wget", "--spider", "-q", "http://127.0.0.1:3021/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
fi

# Generate custom backend services
# NestJS Services
if [[ -n "${NESTJS_SERVICES:-}" ]]; then
  IFS=',' read -ra SERVICES <<< "$NESTJS_SERVICES"
  for service in "${SERVICES[@]}"; do
    service=$(echo "$service" | xargs)  # Trim whitespace
    if [[ -d "services/nest/$service" ]]; then
      cat >>docker-compose.yml <<EOF

  nest-${service}:
    build:
      context: ./services/nest/${service}
      dockerfile: Dockerfile
    container_name: ${PROJECT_NAME:-nself}_nest_${service}
    restart: unless-stopped
    environment:
      - NODE_ENV=${ENV:-development}
      - DATABASE_URL=postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-changeme}@postgres:5432/${POSTGRES_DB:-nhost}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET:-changeme}
    depends_on:
      - postgres
      - hasura
    networks:
      - default
    volumes:
      - ./services/nest/${service}:/app:ro
EOF
    fi
  done
fi

# Python Services
if [[ -n "${PYTHON_SERVICES:-}" ]]; then
  IFS=',' read -ra SERVICES <<< "$PYTHON_SERVICES"
  for service in "${SERVICES[@]}"; do
    service=$(echo "$service" | xargs)
    if [[ -d "services/py/$service" ]]; then
      cat >>docker-compose.yml <<EOF

  py-${service}:
    build:
      context: ./services/py/${service}
      dockerfile: Dockerfile
    container_name: ${PROJECT_NAME:-nself}_py_${service}
    restart: unless-stopped
    environment:
      - ENVIRONMENT=${ENV:-development}
      - DATABASE_URL=postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-changeme}@postgres:5432/${POSTGRES_DB:-nhost}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET:-changeme}
    depends_on:
      - postgres
      - hasura
    networks:
      - default
    volumes:
      - ./services/py/${service}:/app:ro
EOF
    fi
  done
fi

# Go Services
if [[ -n "${GOLANG_SERVICES:-}" ]]; then
  IFS=',' read -ra SERVICES <<< "$GOLANG_SERVICES"
  for service in "${SERVICES[@]}"; do
    service=$(echo "$service" | xargs)
    if [[ -d "services/go/$service" ]]; then
      cat >>docker-compose.yml <<EOF

  go-${service}:
    build:
      context: ./services/go/${service}
      dockerfile: Dockerfile
    container_name: ${PROJECT_NAME:-nself}_go_${service}
    restart: unless-stopped
    environment:
      - ENVIRONMENT=${ENV:-development}
      - DATABASE_URL=postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-changeme}@postgres:5432/${POSTGRES_DB:-nhost}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET:-changeme}
    depends_on:
      - postgres
      - hasura
    networks:
      - default
    volumes:
      - ./services/go/${service}:/app:ro
EOF
    fi
  done
fi

# Rust Services
if [[ -n "${RUST_SERVICES:-}" ]]; then
  IFS=',' read -ra SERVICES <<< "$RUST_SERVICES"
  for service in "${SERVICES[@]}"; do
    service=$(echo "$service" | xargs)
    if [[ -d "services/rust/$service" ]]; then
      cat >>docker-compose.yml <<EOF

  rust-${service}:
    build:
      context: ./services/rust/${service}
      dockerfile: Dockerfile
    container_name: ${PROJECT_NAME:-nself}_rust_${service}
    restart: unless-stopped
    environment:
      - ENVIRONMENT=${ENV:-development}
      - DATABASE_URL=postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-changeme}@postgres:5432/${POSTGRES_DB:-nhost}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET:-changeme}
    depends_on:
      - postgres
      - hasura
    networks:
      - default
EOF
    fi
  done
fi

# Java Services
if [[ -n "${JAVA_SERVICES:-}" ]]; then
  IFS=',' read -ra SERVICES <<< "$JAVA_SERVICES"
  for service in "${SERVICES[@]}"; do
    service=$(echo "$service" | xargs)
    if [[ -d "services/java/$service" ]]; then
      cat >>docker-compose.yml <<EOF

  java-${service}:
    build:
      context: ./services/java/${service}
      dockerfile: Dockerfile
    container_name: ${PROJECT_NAME:-nself}_java_${service}
    restart: unless-stopped
    environment:
      - ENVIRONMENT=${ENV:-development}
      - DATABASE_URL=postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-changeme}@postgres:5432/${POSTGRES_DB:-nhost}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET:-changeme}
    depends_on:
      - postgres
      - hasura
    networks:
      - default
EOF
    fi
  done
fi

# Frontend apps are external - nginx routes are generated separately
