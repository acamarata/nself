#!/usr/bin/env bash

# compose.sh - Generate docker-compose.yml from environment configuration

set -e

# Get script directory
SCRIPT_DIR="$(dirname "$(readlink -f "$0" 2>/dev/null || realpath "$0" 2>/dev/null || echo "$0")")"

# Source environment utilities for safe loading
source "$SCRIPT_DIR/../../lib/utils/env.sh"

# Load environment safely (without executing JSON values)
if [ -f ".env.local" ]; then
  load_env_safe ".env.local"
else
  echo "Error: No .env.local file found."
  exit 1
fi

# Compose database URLs from individual variables
export HASURA_GRAPHQL_DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"
S3_ENDPOINT="http://minio:${MINIO_PORT}"

# Ensure DOCKER_NETWORK is expanded for Docker Compose
DOCKER_NETWORK="${PROJECT_NAME}_network"
export DOCKER_NETWORK

# Set environment-specific defaults
# Support both ENV and ENVIRONMENT for backward compatibility
if [[ "$ENV" == "prod" ]] || [[ "$ENVIRONMENT" == "production" ]]; then
  # Production defaults
  HASURA_GRAPHQL_ENABLE_CONSOLE=${HASURA_GRAPHQL_ENABLE_CONSOLE:-false}
  HASURA_GRAPHQL_DEV_MODE=${HASURA_GRAPHQL_DEV_MODE:-false}
  HASURA_GRAPHQL_CORS_DOMAIN=${HASURA_GRAPHQL_CORS_DOMAIN:-"https://${BASE_DOMAIN}"}
  SECURITY_HEADERS_ENABLED=${SECURITY_HEADERS_ENABLED:-true}
  RATE_LIMIT_ENABLED=${RATE_LIMIT_ENABLED:-true}
  LOG_LEVEL=${LOG_LEVEL:-warn}
else
  # Development defaults
  HASURA_GRAPHQL_ENABLE_CONSOLE=${HASURA_GRAPHQL_ENABLE_CONSOLE:-true}
  HASURA_GRAPHQL_DEV_MODE=${HASURA_GRAPHQL_DEV_MODE:-true}
  HASURA_GRAPHQL_CORS_DOMAIN=${HASURA_GRAPHQL_CORS_DOMAIN:-"*"}
  SECURITY_HEADERS_ENABLED=${SECURITY_HEADERS_ENABLED:-false}
  LOG_LEVEL=${LOG_LEVEL:-info}
fi

# Backup existing docker-compose.yml if it exists
if [ -f "docker-compose.yml" ]; then
  cp docker-compose.yml docker-compose.yml.backup
  echo "Backed up existing docker-compose.yml to docker-compose.yml.backup"
fi

# Start docker-compose.yml
cat > docker-compose.yml << EOF
services:
  # Nginx Reverse Proxy
  nginx:
    image: nginx:${NGINX_VERSION}
    container_name: ${PROJECT_NAME}_nginx
    restart: unless-stopped
    ports:
      - "${NGINX_HTTP_PORT}:80"
      - "${NGINX_HTTPS_PORT}:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    extra_hosts:
      - "host.docker.internal:host-gateway"
    depends_on:
      - hasura
      - auth
      - minio
EOF

if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
  echo "      - functions" >> docker-compose.yml
fi

if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
  echo "      - dashboard" >> docker-compose.yml
  echo "      - config-server" >> docker-compose.yml
fi

if [[ "$EMAIL_PROVIDER" == "mailhog" ]]; then
  echo "      - mailhog" >> docker-compose.yml
fi
# Note: Services are managed in a separate docker-compose.yml in services/ directory

cat >> docker-compose.yml << EOF
    networks:
      - default

  # PostgreSQL Database
  postgres:
    image: postgres:${POSTGRES_VERSION}
    container_name: ${PROJECT_NAME}_postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./postgres/init:/docker-entrypoint-initdb.d:ro
    ports:
      - "${POSTGRES_PORT}:5432"
    networks:
      - default
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Hasura GraphQL Engine
  hasura:
    image: hasura/graphql-engine:${HASURA_VERSION}
    container_name: ${PROJECT_NAME}_hasura
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      HASURA_GRAPHQL_DATABASE_URL: ${HASURA_GRAPHQL_DATABASE_URL}
      HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET}
      HASURA_GRAPHQL_JWT_SECRET: |
        ${HASURA_GRAPHQL_JWT_SECRET}
      HASURA_GRAPHQL_ENABLE_CONSOLE: ${HASURA_GRAPHQL_ENABLE_CONSOLE}
      HASURA_GRAPHQL_DEV_MODE: ${HASURA_GRAPHQL_DEV_MODE}
      HASURA_GRAPHQL_ENABLED_LOG_TYPES: "startup,http-log,webhook-log,websocket-log"
      HASURA_GRAPHQL_ENABLE_TELEMETRY: ${HASURA_GRAPHQL_ENABLE_TELEMETRY}
      HASURA_GRAPHQL_CORS_DOMAIN: "${HASURA_GRAPHQL_CORS_DOMAIN}"
      HASURA_GRAPHQL_UNAUTHORIZED_ROLE: public
    volumes:
      - ./hasura/metadata:/hasura-metadata
      - ./hasura/migrations:/hasura-migrations
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5

  # Hasura Auth Service
  auth:
    image: nhost/hasura-auth:${AUTH_VERSION}
    container_name: ${PROJECT_NAME}_auth
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      hasura:
        condition: service_healthy
    environment:
      AUTH_HOST: 0.0.0.0
      AUTH_PORT: ${AUTH_PORT}
      HASURA_GRAPHQL_DATABASE_URL: ${HASURA_GRAPHQL_DATABASE_URL}
      HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET}
      HASURA_GRAPHQL_JWT_SECRET: |
        ${HASURA_GRAPHQL_JWT_SECRET}
      HASURA_GRAPHQL_GRAPHQL_URL: http://hasura:8080/v1/graphql
      HASURA_GRAPHQL_ENDPOINT: http://hasura:8080/v1/graphql
      AUTH_CLIENT_URL: ${AUTH_CLIENT_URL}
      AUTH_SERVER_URL: https://${AUTH_ROUTE}
      AUTH_SMTP_HOST: ${AUTH_SMTP_HOST:-${EMAIL_PROVIDER:-mailpit}}
      AUTH_SMTP_PORT: ${AUTH_SMTP_PORT:-1025}
      AUTH_SMTP_USER: "${AUTH_SMTP_USER}"
      AUTH_SMTP_PASS: "${AUTH_SMTP_PASS}"
      AUTH_SMTP_SECURE: ${AUTH_SMTP_SECURE}
      AUTH_SMTP_SENDER: ${AUTH_SMTP_SENDER}
      AUTH_JWT_REFRESH_TOKEN_EXPIRES_IN: ${AUTH_JWT_REFRESH_TOKEN_EXPIRES_IN}
      AUTH_JWT_ACCESS_TOKEN_EXPIRES_IN: ${AUTH_JWT_ACCESS_TOKEN_EXPIRES_IN}
      AUTH_WEBAUTHN_ENABLED: ${AUTH_WEBAUTHN_ENABLED}
      AUTH_WEBAUTHN_RP_NAME: ${PROJECT_NAME}
      AUTH_WEBAUTHN_RP_ORIGINS: https://${AUTH_ROUTE}
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${AUTH_PORT}/version"]
      interval: 30s
      timeout: 10s
      retries: 5

  # MinIO Object Storage
  minio:
    image: minio/minio:${MINIO_VERSION}
    container_name: ${PROJECT_NAME}_minio
    restart: unless-stopped
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio_data:/data
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  # Storage Service
  storage:
    image: nhost/hasura-storage:0.6.1
    container_name: ${PROJECT_NAME}_storage
    restart: unless-stopped
    command: serve
    ports:
      - "5000:5000"
    depends_on:
      postgres:
        condition: service_healthy
      hasura:
        condition: service_healthy
      minio:
        condition: service_healthy
    environment:
      BIND: 0.0.0.0:5000
      HASURA_METADATA: 1
      HASURA_ENDPOINT: http://hasura:8080/v1
      HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET}
      S3_ACCESS_KEY: ${S3_ACCESS_KEY}
      S3_SECRET_KEY: ${S3_SECRET_KEY}
      S3_ENDPOINT: http://minio:9000
      S3_BUCKET: ${S3_BUCKET}
      S3_REGION: ${S3_REGION}
      POSTGRES_MIGRATIONS: 1
      POSTGRES_MIGRATIONS_SOURCE: postgres://postgres:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
    networks:
      - default
EOF

# Add optional services

# Functions service
if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
  cat >> docker-compose.yml << EOF

  # Functions Service
  functions:
    build: ./functions
    container_name: ${PROJECT_NAME}_functions
    restart: unless-stopped
    environment:
      PORT: 3000
      HASURA_GRAPHQL_ENDPOINT: http://hasura:8080/v1/graphql
      HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET}
    volumes:
      - ./functions/src:/app/src:ro
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
fi

# Dashboard service
if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
  # First add config server
  cat >> docker-compose.yml << EOF

  # Config Server for Dashboard
  config-server:
    build:
      context: ./config-server
      dockerfile: Dockerfile
    container_name: ${PROJECT_NAME}_config-server
    restart: unless-stopped
    environment:
      - PORT=4001
      - PROJECT_NAME=${PROJECT_NAME}
      - BASE_DOMAIN=${BASE_DOMAIN}
      - HASURA_GRAPHQL_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET}
      - HASURA_VERSION=${HASURA_VERSION}
      - AUTH_VERSION=${AUTH_VERSION}
      - STORAGE_VERSION=${STORAGE_VERSION}
      - POSTGRES_VERSION=${POSTGRES_VERSION}
      - POSTGRES_DB=${POSTGRES_DB}
      - POSTGRES_HOST=postgres
      - POSTGRES_PORT=5432
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - AUTH_CLIENT_URL=${AUTH_CLIENT_URL}
      - AUTH_JWT_ACCESS_TOKEN_EXPIRES_IN=${AUTH_JWT_ACCESS_TOKEN_EXPIRES_IN}
      - AUTH_JWT_REFRESH_TOKEN_EXPIRES_IN=${AUTH_JWT_REFRESH_TOKEN_EXPIRES_IN}
      - AUTH_SMTP_HOST=${AUTH_SMTP_HOST}
      - AUTH_SMTP_PORT=${AUTH_SMTP_PORT}
      - AUTH_SMTP_SENDER=${AUTH_SMTP_SENDER}
      - JWT_KEY=${JWT_KEY}
    networks:
      - default
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:4001/healthz"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Nhost Dashboard
  dashboard:
    image: nhost/dashboard:${DASHBOARD_VERSION}
    container_name: ${PROJECT_NAME}_dashboard
    restart: unless-stopped
    depends_on:
      config-server:
        condition: service_healthy
    environment:
      - NEXT_PUBLIC_NHOST_PLATFORM=false
      - NEXT_PUBLIC_ENV=dev
      - NEXT_PUBLIC_NHOST_HASURA_URL=https://${HASURA_ROUTE}
      - NEXT_PUBLIC_NHOST_AUTH_URL=https://${AUTH_ROUTE}
      - NEXT_PUBLIC_NHOST_STORAGE_URL=https://${STORAGE_ROUTE}
      - NEXT_PUBLIC_NHOST_FUNCTIONS_URL=https://${FUNCTIONS_ROUTE}
      - NEXT_PUBLIC_NHOST_HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET}
      - NEXT_PUBLIC_NHOST_CONFIGSERVER_URL=https://config.${BASE_DOMAIN}
      - NEXT_PUBLIC_NHOST_REGION=local
      - NEXT_PUBLIC_NHOST_SUBDOMAIN=local
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
fi

# Redis service
if [[ "$REDIS_ENABLED" == "true" ]]; then
  cat >> docker-compose.yml << EOF

  # Redis Cache
  redis:
    image: redis:${REDIS_VERSION}
    container_name: ${PROJECT_NAME}_redis
    restart: unless-stopped
    command: ${REDIS_PASSWORD:+redis-server --requirepass ${REDIS_PASSWORD}}${REDIS_PASSWORD:-redis-server}
    volumes:
      - redis_data:/data
    ports:
      - "${REDIS_PORT}:6379"
    networks:
      - default
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
fi

# Email service for development
if [[ "$EMAIL_PROVIDER" == "mailhog" ]] || [[ "$EMAIL_PROVIDER" == "mailpit" ]]; then
  # Use MailPit (modern replacement for MailHog)
  cat >> docker-compose.yml << EOF

  # MailPit (Development Email - Modern replacement for MailHog)
  mailpit:
    image: axllent/mailpit:latest
    container_name: ${PROJECT_NAME}_mailpit
    restart: unless-stopped
    ports:
      - "${MAILPIT_SMTP_PORT:-1025}:1025"  # SMTP
      - "${MAILPIT_UI_PORT:-8025}:8025"    # Web UI
    environment:
      MP_SMTP_AUTH_ACCEPT_ANY: 1
      MP_SMTP_AUTH_ALLOW_INSECURE: 1
      MP_UI_AUTH_FILE: ""  # No auth for development
      MP_MAX_MESSAGES: 5000
      MP_DATABASE: /data/mailpit.db
    volumes:
      - mailpit_data:/data
    networks:
      - default
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8025/api/v1/info"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
  
  # Create alias for backward compatibility
  if [[ "$EMAIL_PROVIDER" == "mailhog" ]]; then
    cat >> docker-compose.yml << EOF

  # Alias for backward compatibility
  mailhog:
    extends:
      service: mailpit
    container_name: ${PROJECT_NAME}_mailhog
EOF
  fi
fi

# NestJS Run Service (Constantly Running Microservices)
if [[ "$NESTJS_RUN_ENABLED" == "true" ]]; then
  cat >> docker-compose.yml << EOF

  # NestJS Run Service
  nestjs-run:
    build: ./nestjs-run
    container_name: ${PROJECT_NAME}_nestjs_run
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      hasura:
        condition: service_healthy
EOF

  if [[ "$REDIS_ENABLED" == "true" ]]; then
    echo "      redis:" >> docker-compose.yml
    echo "        condition: service_healthy" >> docker-compose.yml
  fi

  cat >> docker-compose.yml << EOF
    environment:
      NODE_ENV: ${ENVIRONMENT}
      PORT: ${NESTJS_RUN_PORT}
      DATABASE_URL: ${HASURA_GRAPHQL_DATABASE_URL}
      HASURA_ENDPOINT: http://hasura:8080/v1/graphql
      HASURA_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET}
EOF

  if [[ "$REDIS_ENABLED" == "true" ]]; then
    echo "      REDIS_HOST: redis" >> docker-compose.yml
    echo "      REDIS_PORT: 6379" >> docker-compose.yml
    echo "      REDIS_PASSWORD: ${REDIS_PASSWORD}" >> docker-compose.yml
  fi

  cat >> docker-compose.yml << EOF
    volumes:
      - ./nestjs-run:/app:ro
    networks:
      - default
EOF
fi

# Add backend services if enabled
if [[ "$SERVICES_ENABLED" == "true" ]]; then
  echo "" >> docker-compose.yml
  echo "  # ============================================" >> docker-compose.yml
  echo "  # Backend Services (NestJS, BullMQ, Go, Python)" >> docker-compose.yml
  echo "  # ============================================" >> docker-compose.yml
  
  # Include the services directly in main compose
  bash "$SCRIPT_DIR/compose-inline-append.sh"
fi

# Add volumes section
cat >> docker-compose.yml << EOF

volumes:
  postgres_data:
    name: ${PROJECT_NAME}_postgres_data
  minio_data:
    name: ${PROJECT_NAME}_minio_data
EOF

if [[ "$REDIS_ENABLED" == "true" ]]; then
  echo "  redis_data:" >> docker-compose.yml
  echo "    name: ${PROJECT_NAME}_redis_data" >> docker-compose.yml
fi

if [[ "$EMAIL_PROVIDER" == "mailhog" ]] || [[ "$EMAIL_PROVIDER" == "mailpit" ]]; then
  echo "  mailpit_data:" >> docker-compose.yml
  echo "    name: ${PROJECT_NAME}_mailpit_data" >> docker-compose.yml
fi

# Add networks section
cat >> docker-compose.yml << EOF

networks:
  default:
    name: ${DOCKER_NETWORK}
    driver: bridge
EOF

# Note: Docker Compose validation temporarily disabled during development
echo "docker-compose.yml generated successfully!"
echo "Note: Run 'docker compose config' to validate the generated file"