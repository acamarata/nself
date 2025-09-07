#!/usr/bin/env bash

# compose.sh - Generate docker-compose.yml from environment configuration

set -e

# Error handler with more details
trap 'echo "Error at line $LINENO in compose-generate.sh: $BASH_COMMAND" >&2' ERR

# Enable debugging if DEBUG is set
[[ "${DEBUG:-}" == "true" ]] && set -x

# Get script directory (macOS compatible)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source display utilities first (for logging functions)
source "$SCRIPT_DIR/../../lib/utils/display.sh" || {
  echo "Error: Cannot load display.sh" >&2
  exit 1
}

# Source environment utilities for safe loading
source "$SCRIPT_DIR/../../lib/utils/env.sh" || {
  log_error "Cannot load env.sh"
  exit 1
}

# Load environment safely (without executing JSON values)
if [ -f ".env" ] || [ -f ".env.dev" ]; then
  load_env_with_priority || {
    log_error "Failed to load environment"
    exit 1
  }
else
  log_error "No environment file found (.env or .env.dev)."
  exit 1
fi

log_info "Generating docker-compose.yml..."

# Source smart defaults to handle JWT construction
if ! declare -f load_env_with_defaults >/dev/null 2>&1; then
  if [[ ! -f "$SCRIPT_DIR/../../lib/config/smart-defaults.sh" ]]; then
    log_error "Cannot find smart-defaults.sh"
    exit 1
  fi
  source "$SCRIPT_DIR/../../lib/config/smart-defaults.sh"
fi
if ! load_env_with_defaults; then
  log_error "Failed to load environment with defaults"
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
  [[ "${VERBOSE:-}" == "true" ]] && log_info "Backed up existing docker-compose.yml"
fi

# Start docker-compose.yml
cat >docker-compose.yml <<EOF
services:
  # Nginx Reverse Proxy (always needed for routing)
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
      - ./nginx/ssl/localhost:/etc/nginx/ssl/localhost:ro
      - ./nginx/ssl/nself-org:/etc/nginx/ssl/nself-org:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
    extra_hosts:
      - "host.docker.internal:host-gateway"
    depends_on:
EOF

# Add dependencies based on enabled services
if [[ "$HASURA_ENABLED" == "true" ]]; then
  echo "      - hasura" >>docker-compose.yml
fi
if [[ "$AUTH_ENABLED" == "true" ]]; then
  echo "      - auth" >>docker-compose.yml
fi
if [[ "$STORAGE_ENABLED" == "true" ]]; then
  echo "      - minio" >>docker-compose.yml
fi


# Note: unity-functions and unity-dashboard are added by compose-inline-append.sh

if [[ "$EMAIL_PROVIDER" == "mailhog" ]]; then
  echo "      - mailhog" >>docker-compose.yml
fi
# Note: Services are managed in a separate docker-compose.yml in services/ directory

cat >>docker-compose.yml <<EOF
    networks:
      - default
EOF

# PostgreSQL Database (conditionally added)
if [[ "$POSTGRES_ENABLED" == "true" ]]; then
  cat >>docker-compose.yml <<EOF

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
EOF
fi

# Hasura GraphQL Engine (conditionally added)
if [[ "$HASURA_ENABLED" == "true" ]]; then
  cat >>docker-compose.yml <<EOF

  # Hasura GraphQL Engine
  hasura:
    image: hasura/graphql-engine:${HASURA_VERSION}
    container_name: ${PROJECT_NAME}_hasura
    restart: unless-stopped
EOF

  # Only add postgres dependency if postgres is enabled
  if [[ "$POSTGRES_ENABLED" == "true" ]]; then
    cat >>docker-compose.yml <<EOF
    depends_on:
      postgres:
        condition: service_healthy
EOF
  fi

  cat >>docker-compose.yml <<EOF
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
EOF
fi

# Hasura Auth Service (conditionally added)
if [[ "$AUTH_ENABLED" == "true" ]]; then
  cat >>docker-compose.yml <<EOF

  # Hasura Auth Service
  auth:
    image: nhost/hasura-auth:${AUTH_VERSION}
    container_name: ${PROJECT_NAME}_auth
    restart: unless-stopped
EOF

  # Add dependencies based on what's enabled
  has_deps=false
  if [[ "$POSTGRES_ENABLED" == "true" ]] || [[ "$HASURA_ENABLED" == "true" ]]; then
    echo "    depends_on:" >>docker-compose.yml
    if [[ "$POSTGRES_ENABLED" == "true" ]]; then
      echo "      postgres:" >>docker-compose.yml
      echo "        condition: service_healthy" >>docker-compose.yml
    fi
    if [[ "$HASURA_ENABLED" == "true" ]]; then
      echo "      hasura:" >>docker-compose.yml
      echo "        condition: service_healthy" >>docker-compose.yml
    fi
  fi

  cat >>docker-compose.yml <<EOF
    environment:
      AUTH_HOST: 0.0.0.0
      AUTH_PORT: 4000
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
      test: ["CMD", "curl", "-f", "http://localhost:4000/version"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
fi

# MinIO Object Storage (conditionally added - tied to STORAGE_ENABLED)
if [[ "$STORAGE_ENABLED" == "true" ]]; then
  cat >>docker-compose.yml <<EOF

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
EOF

  # Storage Service (part of storage stack)
  cat >>docker-compose.yml <<EOF

  # Storage Service
  storage:
    image: nhost/hasura-storage:0.6.1
    container_name: ${PROJECT_NAME}_storage
    restart: unless-stopped
    command: serve
    ports:
      - "${STORAGE_PORT:-5001}:${STORAGE_PORT:-5001}"
EOF

  # Add storage dependencies based on what's enabled  
  storage_has_deps=false
  if [[ "$POSTGRES_ENABLED" == "true" ]] || [[ "$HASURA_ENABLED" == "true" ]]; then
    storage_has_deps=true
  fi
  
  if [[ "$storage_has_deps" == "true" ]]; then
    echo "    depends_on:" >>docker-compose.yml
    if [[ "$POSTGRES_ENABLED" == "true" ]]; then
      echo "      postgres:" >>docker-compose.yml
      echo "        condition: service_healthy" >>docker-compose.yml
    fi
    if [[ "$HASURA_ENABLED" == "true" ]]; then
      echo "      hasura:" >>docker-compose.yml
      echo "        condition: service_healthy" >>docker-compose.yml
    fi
    # MinIO is always present when storage is enabled
    echo "      minio:" >>docker-compose.yml
    echo "        condition: service_healthy" >>docker-compose.yml
  fi

  cat >>docker-compose.yml <<EOF
    environment:
      BIND: 0.0.0.0:${STORAGE_PORT:-5001}
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
fi  # End of STORAGE_ENABLED block

# Add optional services

# Functions service - Skipped since compose-inline-append.sh adds unity-functions
if false; then # Disabled - using unity-functions from compose-inline-append.sh instead
  cat >>docker-compose.yml <<EOF

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

# Config Server (needed for dashboard)

# Nhost Dashboard - Skipped, using unity-dashboard from compose-inline-append.sh
if false; then # Disabled
  cat >>docker-compose.yml <<EOF

  dashboard:
    image: nhost/dashboard:${DASHBOARD_VERSION}
    container_name: ${PROJECT_NAME}_dashboard
    restart: unless-stopped
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
  cat >>docker-compose.yml <<EOF

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

# MLflow service
if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
  # Set defaults
  MLFLOW_VERSION="${MLFLOW_VERSION:-2.9.2}"
  MLFLOW_PORT="${MLFLOW_PORT:-5000}"
  MLFLOW_DB_NAME="${MLFLOW_DB_NAME:-mlflow}"
  MLFLOW_ARTIFACTS_BUCKET="${MLFLOW_ARTIFACTS_BUCKET:-mlflow-artifacts}"
  MLFLOW_MEMORY_LIMIT="${MLFLOW_MEMORY_LIMIT:-2Gi}"
  MLFLOW_CPU_LIMIT="${MLFLOW_CPU_LIMIT:-1000m}"
  MLFLOW_WORKERS="${MLFLOW_WORKERS:-4}"
  
  cat >>docker-compose.yml <<EOF

  # MLflow - ML Experiment Tracking & Model Registry
  mlflow:
    image: ghcr.io/mlflow/mlflow:v${MLFLOW_VERSION}
    container_name: ${PROJECT_NAME}_mlflow
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_started
    networks:
      - default
    ports:
      - "${MLFLOW_PORT}:5000"
    environment:
      # Backend store (PostgreSQL)
      MLFLOW_BACKEND_STORE_URI: postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${MLFLOW_DB_NAME}
      # Artifact store (MinIO S3)
      MLFLOW_DEFAULT_ARTIFACT_ROOT: s3://${MLFLOW_ARTIFACTS_BUCKET}/
      AWS_ACCESS_KEY_ID: ${MINIO_ROOT_USER:-minioadmin}
      AWS_SECRET_ACCESS_KEY: ${MINIO_ROOT_PASSWORD:-minioadmin}
      MLFLOW_S3_ENDPOINT_URL: http://minio:9000
      MLFLOW_S3_IGNORE_TLS: "true"
      # Server configuration
      MLFLOW_SERVE_ARTIFACTS: ${MLFLOW_SERVE_ARTIFACTS:-true}
      MLFLOW_WORKERS: ${MLFLOW_WORKERS}
EOF

  # Add authentication if enabled
  if [[ "${MLFLOW_AUTH_ENABLED:-false}" == "true" ]]; then
    cat >>docker-compose.yml <<EOF
      MLFLOW_AUTH_CONFIG_PATH: /app/auth.ini
      MLFLOW_AUTH_ENABLED: "true"
      MLFLOW_ADMIN_USERNAME: ${MLFLOW_ADMIN_USERNAME:-admin}
      MLFLOW_ADMIN_PASSWORD: ${MLFLOW_ADMIN_PASSWORD}
EOF
  fi

  cat >>docker-compose.yml <<EOF
    volumes:
      - mlflow_data:/mlflow
    command: >
      sh -c "
        pip install psycopg2-binary boto3 &&
        mlflow db upgrade postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${MLFLOW_DB_NAME} &&
        mlflow server
          --backend-store-uri postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${MLFLOW_DB_NAME}
          --default-artifact-root s3://${MLFLOW_ARTIFACTS_BUCKET}/
          --host 0.0.0.0
          --port 5000
          --serve-artifacts
          --workers ${MLFLOW_WORKERS}
      "
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:5000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s
    deploy:
      resources:
        limits:
          memory: ${MLFLOW_MEMORY_LIMIT}
          cpus: "${MLFLOW_CPU_LIMIT}"
EOF
fi

# Search Services
if [[ "${SEARCH_ENABLED:-false}" == "true" ]]; then
  case "${SEARCH_ENGINE:-meilisearch}" in
    meilisearch)
      cat >>docker-compose.yml <<EOF

  # Meilisearch - Lightning-fast, open-source search engine
  meilisearch:
    image: getmeili/meilisearch:${MEILISEARCH_VERSION}
    container_name: ${PROJECT_NAME}_meilisearch
    restart: unless-stopped
    ports:
      - "${MEILISEARCH_PORT:-7700}:7700"
    environment:
      MEILI_MASTER_KEY: ${MEILISEARCH_MASTER_KEY}
      MEILI_ENV: ${MEILISEARCH_ENV:-development}
      MEILI_HTTP_ADDR: 0.0.0.0:7700
      MEILI_NO_ANALYTICS: true
      MEILI_LOG_LEVEL: info
    volumes:
      - meilisearch_data:/meili_data
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:7700/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
      ;;
      
    typesense)
      cat >>docker-compose.yml <<EOF

  # Typesense - Fast, typo-tolerant search engine
  typesense:
    image: typesense/typesense:${TYPESENSE_VERSION}
    container_name: ${PROJECT_NAME}_typesense
    restart: unless-stopped
    ports:
      - "${TYPESENSE_PORT:-8108}:8108"
    environment:
      TYPESENSE_DATA_DIR: /data
      TYPESENSE_API_KEY: ${TYPESENSE_API_KEY}
      TYPESENSE_ENABLE_CORS: true
    volumes:
      - typesense_data:/data
    networks:
      - default
    command: '--data-dir /data --api-key=${TYPESENSE_API_KEY} --enable-cors'
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8108/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
      ;;
      
    elasticsearch)
      cat >>docker-compose.yml <<EOF

  # Elasticsearch - Distributed RESTful search and analytics engine
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:${ELASTICSEARCH_VERSION}
    container_name: ${PROJECT_NAME}_elasticsearch
    restart: unless-stopped
    ports:
      - "${ELASTICSEARCH_PORT:-9200}:9200"
    environment:
      - discovery.type=single-node
      - "ES_JAVA_OPTS=-Xms512m -Xmx${ELASTICSEARCH_MEMORY:-1g}"
      - xpack.security.enabled=true
      - xpack.security.enrollment.enabled=false
      - ELASTIC_PASSWORD=${ELASTICSEARCH_PASSWORD}
      - bootstrap.memory_lock=true
    ulimits:
      memlock:
        soft: -1
        hard: -1
    volumes:
      - elasticsearch_data:/usr/share/elasticsearch/data
    networks:
      - default
    healthcheck:
      test: ["CMD-SHELL", "curl -s -u elastic:${ELASTICSEARCH_PASSWORD} http://localhost:9200/_cluster/health || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
      ;;
      
    opensearch)
      cat >>docker-compose.yml <<EOF

  # OpenSearch - Open-source fork of Elasticsearch
  opensearch:
    image: opensearchproject/opensearch:${OPENSEARCH_VERSION}
    container_name: ${PROJECT_NAME}_opensearch
    restart: unless-stopped
    ports:
      - "${OPENSEARCH_PORT:-9200}:9200"
      - "9600:9600"  # Performance analyzer
    environment:
      - discovery.type=single-node
      - "OPENSEARCH_JAVA_OPTS=-Xms512m -Xmx${OPENSEARCH_MEMORY:-1g}"
      - OPENSEARCH_INITIAL_ADMIN_PASSWORD=${OPENSEARCH_PASSWORD}
      - plugins.security.ssl.http.enabled=false
      - compatibility.override_main_response_version=true
    ulimits:
      memlock:
        soft: -1
        hard: -1
    volumes:
      - opensearch_data:/usr/share/opensearch/data
    networks:
      - default
    healthcheck:
      test: ["CMD-SHELL", "curl -s http://localhost:9200/_cluster/health || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 5
      
  # OpenSearch Dashboards (optional UI)
  opensearch-dashboards:
    image: opensearchproject/opensearch-dashboards:${OPENSEARCH_VERSION}
    container_name: ${PROJECT_NAME}_opensearch_dashboards
    restart: unless-stopped
    ports:
      - "5601:5601"
    environment:
      OPENSEARCH_HOSTS: '["http://opensearch:9200"]'
      DISABLE_SECURITY_DASHBOARDS_PLUGIN: "true"
    depends_on:
      - opensearch
    networks:
      - default
EOF
      ;;
      
    zinc)
      cat >>docker-compose.yml <<EOF

  # Zinc - Lightweight alternative to Elasticsearch
  zinc:
    image: public.ecr.aws/zinclabs/zinc:${ZINC_VERSION}
    container_name: ${PROJECT_NAME}_zinc
    restart: unless-stopped
    ports:
      - "${ZINC_PORT:-4080}:4080"
    environment:
      ZINC_FIRST_ADMIN_USER: ${ZINC_ADMIN_USER}
      ZINC_FIRST_ADMIN_PASSWORD: ${ZINC_ADMIN_PASSWORD}
      ZINC_DATA_PATH: /data
      GIN_MODE: release
    volumes:
      - zinc_data:/data
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
      ;;
      
    sonic)
      cat >>docker-compose.yml <<EOF

  # Sonic - Fast, lightweight & schema-less search backend
  sonic:
    image: valeriansaliou/sonic:${SONIC_VERSION}
    container_name: ${PROJECT_NAME}_sonic
    restart: unless-stopped
    ports:
      - "${SONIC_PORT:-1491}:1491"
    volumes:
      - sonic_data:/var/lib/sonic
      - ./sonic/config.cfg:/etc/sonic.cfg:ro
    networks:
      - default
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "1491"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
      # Create Sonic config if it doesn't exist
      if [[ ! -f "./sonic/config.cfg" ]]; then
        mkdir -p ./sonic
        cat >./sonic/config.cfg <<EOF
# Sonic Configuration
# Documentation: https://github.com/valeriansaliou/sonic

[server]
log_level = "info"

[channel]
inet = "0.0.0.0:1491"
tcp_timeout = 300
auth_password = "${SONIC_PASSWORD}"

[kv]
path = "/var/lib/sonic/kv"
retain_word_objects = 1000

[store]
path = "/var/lib/sonic/store"

[store.fst]
pool_capacity = 8
query_limit_default = 10
query_limit_maximum = 100

[store.kv]
pool_capacity = 8
database_path = "/var/lib/sonic/kv/data"
EOF
      fi
      ;;
      
    *)
      log_warning "Unknown search engine: ${SEARCH_ENGINE}. Skipping search service."
      ;;
  esac
fi

# Functions Service (Serverless functions runtime)
if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
  cat >>docker-compose.yml <<EOF

  # Functions Service (Node.js serverless runtime)
  functions:
    image: node:20-alpine
    container_name: ${PROJECT_NAME}_functions
    restart: unless-stopped
    command: ["node", "/app/server.js"]
    environment:
      NODE_ENV: ${ENV:-development}
      PORT: 4300
      PROJECT_NAME: ${PROJECT_NAME}
      HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET}
      HASURA_GRAPHQL_ENDPOINT: http://hasura:8080/v1/graphql
      DATABASE_URL: ${HASURA_GRAPHQL_DATABASE_URL}
      JWT_SECRET: ${AUTH_JWT_SECRET}
      FUNCTIONS_PATH: /app/functions
      MAX_EXECUTION_TIME: 30000
      MEMORY_LIMIT: 512
EOF

  # Add Redis if enabled for function queuing
  if [[ "${REDIS_ENABLED:-false}" == "true" ]]; then
    cat >>docker-compose.yml <<EOF
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_PASSWORD: ${REDIS_PASSWORD}
      ENABLE_QUEUE: true
EOF
  fi

  cat >>docker-compose.yml <<EOF
    volumes:
      - ./functions:/app/functions:ro
      - ./functions-runtime:/app:ro
    networks:
      - default
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:4300/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF

  # Generate the functions runtime if it doesn't exist
  if [[ ! -f "./functions-runtime/server.js" ]]; then
    mkdir -p ./functions-runtime
    cat >./functions-runtime/server.js <<'FUNCEOF'
const http = require('http');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const { promisify } = require('util');

const PORT = process.env.PORT || 4300;
const FUNCTIONS_PATH = process.env.FUNCTIONS_PATH || '/app/functions';
const MAX_EXECUTION_TIME = parseInt(process.env.MAX_EXECUTION_TIME || '30000');

// Function cache
const functionCache = new Map();

// Load function from file
async function loadFunction(name) {
  if (functionCache.has(name)) {
    return functionCache.get(name);
  }

  const functionPath = path.join(FUNCTIONS_PATH, `${name}.js`);
  
  try {
    const code = await fs.promises.readFile(functionPath, 'utf8');
    const wrappedCode = `
      (async function() {
        ${code}
        return handler;
      })()
    `;
    
    const context = {
      console,
      process: { env: process.env },
      require: require,
      Buffer,
      setTimeout,
      clearTimeout,
      Promise,
      JSON,
      Date,
      Math,
      Array,
      Object,
      String,
      Number,
      Boolean,
      RegExp,
      Error
    };
    
    const script = new vm.Script(wrappedCode);
    const handler = await script.runInNewContext(context, {
      timeout: 5000,
      displayErrors: true
    });
    
    functionCache.set(name, handler);
    return handler;
  } catch (error) {
    console.error(`Failed to load function ${name}:`, error);
    return null;
  }
}

// Request handler
const server = http.createServer(async (req, res) => {
  // CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
  
  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }
  
  // Health check
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'healthy', functions: Array.from(functionCache.keys()) }));
    return;
  }
  
  // Function execution
  if (req.url.startsWith('/function/')) {
    const functionName = req.url.slice(10).split('?')[0];
    
    try {
      const handler = await loadFunction(functionName);
      
      if (!handler) {
        res.writeHead(404, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ error: 'Function not found' }));
        return;
      }
      
      // Parse request body
      let body = '';
      req.on('data', chunk => { body += chunk; });
      
      await new Promise(resolve => req.on('end', resolve));
      
      const event = {
        body: body ? JSON.parse(body) : {},
        headers: req.headers,
        method: req.method,
        query: new URL(req.url, `http://${req.headers.host}`).searchParams,
        path: req.url
      };
      
      // Execute function with timeout
      const timeoutPromise = new Promise((_, reject) => 
        setTimeout(() => reject(new Error('Function timeout')), MAX_EXECUTION_TIME)
      );
      
      const result = await Promise.race([
        handler(event, { env: process.env }),
        timeoutPromise
      ]);
      
      // Send response
      const response = typeof result === 'object' ? result : { body: result };
      res.writeHead(response.statusCode || 200, {
        'Content-Type': 'application/json',
        ...response.headers
      });
      res.end(JSON.stringify(response.body || response));
      
    } catch (error) {
      console.error('Function execution error:', error);
      res.writeHead(500, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ error: error.message }));
    }
    return;
  }
  
  // List functions
  if (req.url === '/functions') {
    try {
      const files = await fs.promises.readdir(FUNCTIONS_PATH);
      const functions = files.filter(f => f.endsWith('.js')).map(f => f.slice(0, -3));
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ functions }));
    } catch (error) {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ functions: [] }));
    }
    return;
  }
  
  // Default response
  res.writeHead(200, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({
    service: 'nself-functions',
    version: '1.0.0',
    endpoints: [
      '/health',
      '/functions',
      '/function/{name}'
    ]
  }));
});

// Start server
server.listen(PORT, '0.0.0.0', () => {
  console.log(`Functions service running on port ${PORT}`);
  console.log(`Functions path: ${FUNCTIONS_PATH}`);
  console.log(`Max execution time: ${MAX_EXECUTION_TIME}ms`);
  
  // Watch for function changes in development
  if (process.env.NODE_ENV !== 'production') {
    try {
      fs.watch(FUNCTIONS_PATH, { recursive: true }, (eventType, filename) => {
        if (filename && filename.endsWith('.js')) {
          const functionName = filename.slice(0, -3);
          functionCache.delete(functionName);
          console.log(`Reloading function: ${functionName}`);
        }
      });
      console.log('Watching for function changes...');
    } catch (err) {
      console.log('Function directory not found, will be created when functions are added');
    }
  }
});

// Graceful shutdown
process.on('SIGTERM', () => {
  server.close(() => {
    console.log('Functions service stopped');
    process.exit(0);
  });
});
FUNCEOF

    cat >./functions-runtime/package.json <<'FUNCEOF'
{
  "name": "nself-functions",
  "version": "1.0.0",
  "description": "Serverless functions runtime for nself",
  "main": "server.js",
  "scripts": {
    "start": "node server.js",
    "dev": "NODE_ENV=development node server.js"
  },
  "engines": {
    "node": ">=18.0.0"
  }
}
FUNCEOF

    # Create example function
    mkdir -p ./functions
    cat >./functions/hello.js <<'FUNCEOF'
// Example serverless function
async function handler(event, context) {
  const name = event.body?.name || event.query?.get('name') || 'World';
  
  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'application/json'
    },
    body: {
      message: `Hello, ${name}!`,
      timestamp: new Date().toISOString(),
      environment: context.env.NODE_ENV
    }
  };
}
FUNCEOF

    cat >./functions/webhook.js <<'FUNCEOF'
// Webhook handler example
async function handler(event, context) {
  console.log('Webhook received:', event.body);
  
  // Process webhook data
  const { action, data } = event.body || {};
  
  switch (action) {
    case 'user.created':
      // Handle new user
      console.log('New user:', data);
      break;
      
    case 'payment.completed':
      // Handle payment
      console.log('Payment completed:', data);
      break;
      
    default:
      console.log('Unknown action:', action);
  }
  
  return {
    statusCode: 200,
    body: { received: true }
  };
}
FUNCEOF

    cat >./functions/README.md <<'FUNCEOF'
# nself Functions

Place your serverless functions in this directory. Each function should be a separate `.js` file that exports a `handler` function.

## Function Structure

```javascript
async function handler(event, context) {
  // Your function logic here
  
  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'application/json'
    },
    body: {
      // Your response data
    }
  };
}
```

## Event Object

- `event.body` - Parsed JSON body (for POST requests)
- `event.headers` - Request headers
- `event.method` - HTTP method
- `event.query` - URLSearchParams object
- `event.path` - Request path

## Context Object

- `context.env` - Environment variables

## Calling Functions

Functions are available at: `http://localhost:4300/function/{name}`

Example:
```bash
curl http://localhost:4300/function/hello?name=Alice
curl -X POST http://localhost:4300/function/webhook -d '{"action":"test"}'
```

## Available Functions

- `hello` - Simple greeting function
- `webhook` - Webhook handler example

Add your own functions by creating new `.js` files in this directory.
FUNCEOF
  fi
fi

# MLflow Service (Machine Learning experiment tracking)
if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
  cat >>docker-compose.yml <<EOF

  # MLflow - ML Experiment Tracking
  mlflow:
    image: ghcr.io/mlflow/mlflow:latest
    container_name: ${PROJECT_NAME}_mlflow
    restart: unless-stopped
    command: >
      sh -c "pip install psycopg2-binary boto3 && 
             mlflow server 
             --host 0.0.0.0 
             --port 5000 
             --backend-store-uri postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/mlflow
             --default-artifact-root s3://${PROJECT_NAME}-mlflow/artifacts
             --serve-artifacts"
    environment:
      MLFLOW_S3_ENDPOINT_URL: http://minio:9000
      AWS_ACCESS_KEY_ID: ${MINIO_ROOT_USER}
      AWS_SECRET_ACCESS_KEY: ${MINIO_ROOT_PASSWORD}
      MLFLOW_TRACKING_USERNAME: ${MLFLOW_USERNAME:-admin}
      MLFLOW_TRACKING_PASSWORD: ${MLFLOW_PASSWORD:-${ADMIN_PASSWORD}}
    ports:
      - "${MLFLOW_PORT:-5000}:5000"
    depends_on:
      - postgres
EOF

  # Add MinIO dependency if storage is enabled
  if [[ "${STORAGE_ENABLED:-true}" == "true" ]]; then
    cat >>docker-compose.yml <<EOF
      - minio
EOF
  fi

  cat >>docker-compose.yml <<EOF
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:5000/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF

  # Create MLflow database if it doesn't exist
  cat >>docker-compose.yml <<EOF

  # MLflow DB initialization
  mlflow-init:
    image: postgres:${POSTGRES_VERSION}
    container_name: ${PROJECT_NAME}_mlflow_init
    command: >
      sh -c "
        until pg_isready -h postgres -U ${POSTGRES_USER}; do
          echo 'Waiting for PostgreSQL...'
          sleep 2
        done
        PGPASSWORD=${POSTGRES_PASSWORD} psql -h postgres -U ${POSTGRES_USER} -tc \"SELECT 1 FROM pg_database WHERE datname = 'mlflow'\" | grep -q 1 || 
        PGPASSWORD=${POSTGRES_PASSWORD} psql -h postgres -U ${POSTGRES_USER} -c 'CREATE DATABASE mlflow;'
        echo 'MLflow database ready'
      "
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    depends_on:
      - postgres
    networks:
      - default
    restart: "no"
EOF
fi

# Email service for development
if [[ "$EMAIL_PROVIDER" == "mailhog" ]] || [[ "$EMAIL_PROVIDER" == "mailpit" ]]; then
  # Use MailPit (modern replacement for MailHog)
  cat >>docker-compose.yml <<EOF

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
    cat >>docker-compose.yml <<EOF

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
  cat >>docker-compose.yml <<EOF

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
    echo "      redis:" >>docker-compose.yml
    echo "        condition: service_healthy" >>docker-compose.yml
  fi

  cat >>docker-compose.yml <<EOF
    environment:
      NODE_ENV: ${ENVIRONMENT}
      PORT: ${NESTJS_RUN_PORT}
      DATABASE_URL: ${HASURA_GRAPHQL_DATABASE_URL}
      HASURA_ENDPOINT: http://hasura:8080/v1/graphql
      HASURA_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET}
EOF

  if [[ "$REDIS_ENABLED" == "true" ]]; then
    echo "      REDIS_HOST: redis" >>docker-compose.yml
    echo "      REDIS_PORT: 6379" >>docker-compose.yml
    echo "      REDIS_PASSWORD: ${REDIS_PASSWORD}" >>docker-compose.yml
  fi

  cat >>docker-compose.yml <<EOF
    volumes:
      - ./nestjs-run:/app:ro
    networks:
      - default
EOF
fi

# Advanced Monitoring with Alerts (Prometheus, AlertManager, Grafana)
if [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]] || [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
  # Prometheus for metrics collection
  cat >>docker-compose.yml <<EOF

  # Prometheus - Metrics Collection
  prometheus:
    image: prom/prometheus:latest
    container_name: ${PROJECT_NAME}_prometheus
    restart: unless-stopped
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.enable-lifecycle'
      - '--web.enable-admin-api'
    ports:
      - "${PROMETHEUS_PORT:-9090}:9090"
    volumes:
      - ./monitoring/prometheus:/etc/prometheus:ro
      - prometheus_data:/prometheus
    networks:
      - default
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:9090/-/healthy"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF

  # Create Prometheus configuration if it doesn't exist
  if [[ ! -f "./monitoring/prometheus/prometheus.yml" ]]; then
    mkdir -p ./monitoring/prometheus
    cat >./monitoring/prometheus/prometheus.yml <<'PROMEOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']

rule_files:
  - "alerts.yml"

scrape_configs:
  # Scrape Prometheus itself
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # Scrape PostgreSQL exporter
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  # Scrape Hasura
  - job_name: 'hasura'
    static_configs:
      - targets: ['hasura:8080']

  # Scrape Node exporter for host metrics
  - job_name: 'node'
    static_configs:
      - targets: ['node-exporter:9100']

  # Scrape cAdvisor for container metrics
  - job_name: 'cadvisor'
    static_configs:
      - targets: ['cadvisor:8080']
PROMEOF

    # Create alert rules
    cat >./monitoring/prometheus/alerts.yml <<'ALERTEOF'
groups:
  - name: nself_alerts
    interval: 30s
    rules:
      # Service Down Alerts
      - alert: PostgresDown
        expr: up{job="postgres"} == 0
        for: 1m
        labels:
          severity: critical
          service: postgres
        annotations:
          summary: "PostgreSQL is down"
          description: "PostgreSQL database has been down for more than 1 minute"

      - alert: HasuraDown
        expr: up{job="hasura"} == 0
        for: 1m
        labels:
          severity: critical
          service: hasura
        annotations:
          summary: "Hasura GraphQL is down"
          description: "Hasura GraphQL engine has been down for more than 1 minute"

      # Resource Usage Alerts
      - alert: HighCPUUsage
        expr: (100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)) > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage detected"
          description: "CPU usage is above 80% for more than 5 minutes on {{ $labels.instance }}"

      - alert: HighMemoryUsage
        expr: (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / node_memory_MemTotal_bytes * 100 > 85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is above 85% for more than 5 minutes"

      - alert: DiskSpaceRunningLow
        expr: node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"} * 100 < 20
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Disk space running low"
          description: "Less than 20% disk space remaining on root filesystem"

      # Database Alerts
      - alert: PostgresConnectionsHigh
        expr: pg_stat_database_numbackends / pg_settings_max_connections > 0.8
        for: 5m
        labels:
          severity: warning
          service: postgres
        annotations:
          summary: "PostgreSQL connection pool nearly exhausted"
          description: "PostgreSQL is using more than 80% of max connections"

      - alert: PostgresSlowQueries
        expr: rate(pg_stat_statements_mean_exec_time_seconds[5m]) > 1
        for: 5m
        labels:
          severity: warning
          service: postgres
        annotations:
          summary: "PostgreSQL slow queries detected"
          description: "Average query execution time is above 1 second"

      # Container Alerts
      - alert: ContainerRestarting
        expr: rate(container_restart_count[5m]) > 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Container restarting frequently"
          description: "Container {{ $labels.name }} has restarted {{ $value }} times in the last 5 minutes"

      - alert: ContainerHighCPU
        expr: rate(container_cpu_usage_seconds_total[5m]) * 100 > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Container high CPU usage"
          description: "Container {{ $labels.name }} CPU usage is above 80%"

      - alert: ContainerHighMemory
        expr: (container_memory_usage_bytes / container_spec_memory_limit_bytes) * 100 > 85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Container high memory usage"
          description: "Container {{ $labels.name }} memory usage is above 85% of limit"

      # API/Service Alerts
      - alert: HasuraHighLatency
        expr: histogram_quantile(0.95, rate(hasura_http_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: warning
          service: hasura
        annotations:
          summary: "Hasura API high latency"
          description: "95th percentile response time is above 2 seconds"

      - alert: HasuraErrorRate
        expr: rate(hasura_http_requests_total{status=~"5.."}[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
          service: hasura
        annotations:
          summary: "Hasura API error rate high"
          description: "Error rate is above 5% for Hasura API"
ALERTEOF
  fi

  # AlertManager for alert routing
  if [[ "${ALERTMANAGER_ENABLED:-true}" == "true" ]]; then
    cat >>docker-compose.yml <<EOF

  # AlertManager - Alert Routing & Notifications
  alertmanager:
    image: prom/alertmanager:latest
    container_name: ${PROJECT_NAME}_alertmanager
    restart: unless-stopped
    ports:
      - "${ALERTMANAGER_PORT:-9093}:9093"
    volumes:
      - ./monitoring/alertmanager:/etc/alertmanager:ro
      - alertmanager_data:/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
    networks:
      - default
EOF

    # Create AlertManager configuration if it doesn't exist
    if [[ ! -f "./monitoring/alertmanager/alertmanager.yml" ]]; then
      mkdir -p ./monitoring/alertmanager
      cat >./monitoring/alertmanager/alertmanager.yml <<'AMEOF'
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  
  routes:
    - match:
        severity: critical
      receiver: 'critical'
      continue: true
    
    - match:
        severity: warning
      receiver: 'warning'

receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://localhost:4300/function/webhook-alerts'
        send_resolved: true

  - name: 'critical'
    webhook_configs:
      - url: 'http://localhost:4300/function/webhook-critical'
        send_resolved: true
    # Add email config if SMTP is configured
    # email_configs:
    #   - to: '${ADMIN_EMAIL}'
    #     from: 'alerts@${BASE_DOMAIN}'
    #     smarthost: '${AUTH_SMTP_HOST}:${AUTH_SMTP_PORT}'
    #     auth_username: '${AUTH_SMTP_USER}'
    #     auth_password: '${AUTH_SMTP_PASS}'

  - name: 'warning'
    webhook_configs:
      - url: 'http://localhost:4300/function/webhook-warnings'
        send_resolved: true

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'service']
AMEOF
    fi
  fi

  # Node Exporter for host metrics
  cat >>docker-compose.yml <<EOF

  # Node Exporter - Host Metrics
  node-exporter:
    image: prom/node-exporter:latest
    container_name: ${PROJECT_NAME}_node_exporter
    restart: unless-stopped
    ports:
      - "${NODE_EXPORTER_PORT:-9100}:9100"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.sysfs=/host/sys'
      - '--path.rootfs=/rootfs'
      - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)'
    networks:
      - default
EOF

  # PostgreSQL Exporter
  if [[ "${POSTGRES_ENABLED:-true}" == "true" ]]; then
    cat >>docker-compose.yml <<EOF

  # PostgreSQL Exporter - Database Metrics
  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:latest
    container_name: ${PROJECT_NAME}_postgres_exporter
    restart: unless-stopped
    ports:
      - "${POSTGRES_EXPORTER_PORT:-9187}:9187"
    environment:
      DATA_SOURCE_NAME: "postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"
    depends_on:
      - postgres
    networks:
      - default
EOF
  fi

  # cAdvisor for container metrics
  cat >>docker-compose.yml <<EOF

  # cAdvisor - Container Metrics
  cadvisor:
    image: gcr.io/cadvisor/cadvisor:latest
    container_name: ${PROJECT_NAME}_cadvisor
    restart: unless-stopped
    ports:
      - "${CADVISOR_PORT:-8090}:8080"
    volumes:
      - /:/rootfs:ro
      - /var/run:/var/run:ro
      - /sys:/sys:ro
      - /var/lib/docker/:/var/lib/docker:ro
      - /dev/disk/:/dev/disk:ro
    devices:
      - /dev/kmsg
    privileged: true
    networks:
      - default
EOF
fi

# Grafana for visualization
if [[ "${GRAFANA_ENABLED:-false}" == "true" ]] || [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
  cat >>docker-compose.yml <<EOF

  # Grafana - Metrics Visualization
  grafana:
    image: grafana/grafana:latest
    container_name: ${PROJECT_NAME}_grafana
    restart: unless-stopped
    ports:
      - "${GRAFANA_PORT:-3300}:3000"
    environment:
      GF_SECURITY_ADMIN_USER: ${GRAFANA_ADMIN_USER:-admin}
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_ADMIN_PASSWORD:-${ADMIN_PASSWORD}}
      GF_INSTALL_PLUGINS: ${GRAFANA_PLUGINS:-}
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/provisioning:/etc/grafana/provisioning:ro
    depends_on:
      - prometheus
    networks:
      - default
EOF

  # Create Grafana provisioning if it doesn't exist
  if [[ ! -f "./monitoring/grafana/provisioning/datasources/prometheus.yml" ]]; then
    mkdir -p ./monitoring/grafana/provisioning/{datasources,dashboards}
    
    # Datasource configuration
    cat >./monitoring/grafana/provisioning/datasources/prometheus.yml <<'GRAFEOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
GRAFEOF

    # Dashboard provisioning
    cat >./monitoring/grafana/provisioning/dashboards/dashboard.yml <<'GRAFEOF'
apiVersion: 1

providers:
  - name: 'nself'
    orgId: 1
    folder: 'nself'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
GRAFEOF
  fi
fi

# Include additional services (admin, microservices, etc)
# This runs before volumes/networks to keep services together
# Export all environment variables for the subshell
set -a
[[ -f ".env" ]] && source .env 2>/dev/null || true
set +a
# Add timeout protection for the append script
if command -v gtimeout >/dev/null 2>&1; then
    gtimeout 5 bash "$SCRIPT_DIR/compose-inline-append.sh" 2>/dev/null || true
elif command -v timeout >/dev/null 2>&1; then
    timeout 5 bash "$SCRIPT_DIR/compose-inline-append.sh" 2>/dev/null || true
else
    # Run without timeout if not available
    bash "$SCRIPT_DIR/compose-inline-append.sh" 2>/dev/null || true
fi

# Add volumes section
cat >>docker-compose.yml <<EOF

volumes:
  postgres_data:
    name: ${PROJECT_NAME}_postgres_data
  minio_data:
    name: ${PROJECT_NAME}_minio_data
  nself-admin-data:
    name: ${PROJECT_NAME}_admin_data
EOF

if [[ "$REDIS_ENABLED" == "true" ]]; then
  echo "  redis_data:" >>docker-compose.yml
  echo "    name: ${PROJECT_NAME}_redis_data" >>docker-compose.yml
fi

if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
  echo "  mlflow_data:" >>docker-compose.yml
  echo "    name: ${PROJECT_NAME}_mlflow_data" >>docker-compose.yml
fi

if [[ "$EMAIL_PROVIDER" == "mailhog" ]] || [[ "$EMAIL_PROVIDER" == "mailpit" ]]; then
  echo "  mailpit_data:" >>docker-compose.yml
  echo "    name: ${PROJECT_NAME}_mailpit_data" >>docker-compose.yml
fi

# Add search service volumes
if [[ "${SEARCH_ENABLED:-false}" == "true" ]]; then
  case "${SEARCH_ENGINE:-meilisearch}" in
    meilisearch)
      echo "  meilisearch_data:" >>docker-compose.yml
      echo "    name: ${PROJECT_NAME}_meilisearch_data" >>docker-compose.yml
      ;;
    typesense)
      echo "  typesense_data:" >>docker-compose.yml
      echo "    name: ${PROJECT_NAME}_typesense_data" >>docker-compose.yml
      ;;
    elasticsearch)
      echo "  elasticsearch_data:" >>docker-compose.yml
      echo "    name: ${PROJECT_NAME}_elasticsearch_data" >>docker-compose.yml
      ;;
    opensearch)
      echo "  opensearch_data:" >>docker-compose.yml
      echo "    name: ${PROJECT_NAME}_opensearch_data" >>docker-compose.yml
      ;;
    zinc)
      echo "  zinc_data:" >>docker-compose.yml
      echo "    name: ${PROJECT_NAME}_zinc_data" >>docker-compose.yml
      ;;
    sonic)
      echo "  sonic_data:" >>docker-compose.yml
      echo "    name: ${PROJECT_NAME}_sonic_data" >>docker-compose.yml
      ;;
  esac
fi

# Add monitoring volumes if enabled
if [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]] || [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
  echo "  prometheus_data:" >>docker-compose.yml
  echo "    name: ${PROJECT_NAME}_prometheus_data" >>docker-compose.yml
  
  if [[ "${ALERTMANAGER_ENABLED:-true}" == "true" ]]; then
    echo "  alertmanager_data:" >>docker-compose.yml
    echo "    name: ${PROJECT_NAME}_alertmanager_data" >>docker-compose.yml
  fi
fi

if [[ "${GRAFANA_ENABLED:-false}" == "true" ]] || [[ "${MONITORING_ENABLED:-false}" == "true" ]]; then
  echo "  grafana_data:" >>docker-compose.yml
  echo "    name: ${PROJECT_NAME}_grafana_data" >>docker-compose.yml
fi

# Add networks section
cat >>docker-compose.yml <<EOF

networks:
  default:
    name: ${DOCKER_NETWORK}
    driver: bridge
EOF

# Note: Docker Compose validation temporarily disabled during development
log_success "docker-compose.yml generated successfully!"
log_info "Run 'docker compose config' to validate the generated file"
