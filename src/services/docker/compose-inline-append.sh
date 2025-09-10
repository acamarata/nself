#!/usr/bin/env bash

# services-compose-inline.sh - Append services to main docker-compose.yml

# This script is called from compose.sh to add services inline
# It appends directly to the existing docker-compose.yml

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# NestJS Services
if [[ "$NESTJS_ENABLED" == "true" ]]; then
  IFS=',' read -ra NEST_SERVICES <<<"$NESTJS_SERVICES"
  PORT_COUNTER=0

  for service in "${NEST_SERVICES[@]}"; do
    service=$(echo "$service" | xargs)
    SERVICE_PORT=$((NESTJS_PORT_START + PORT_COUNTER))

    cat >>docker-compose.yml <<EOF

  # NestJS Service: $service
  ${PROJECT_NAME}-nest-$service:
    image: ${PROJECT_NAME}/nest-${service}:latest
    build:
      context: ./services/nest/$service
      dockerfile: Dockerfile
      tags:
        - ${PROJECT_NAME}/nest-${service}:latest
        - ${PROJECT_NAME}/nest-${service}:${ENV:-dev}
    container_name: ${PROJECT_NAME}_nest_$service
    restart: unless-stopped
    environment:
      - NODE_ENV=${ENVIRONMENT:-development}
      - PORT=$SERVICE_PORT
      - DATABASE_URL=${HASURA_GRAPHQL_DATABASE_URL}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - CORS_ORIGINS=https://*.${BASE_DOMAIN},http://localhost:*
    ports:
      - "$SERVICE_PORT:$SERVICE_PORT"
    depends_on:
      - postgres
      - hasura
EOF

    if [[ "$REDIS_ENABLED" == "true" ]]; then
      echo "      - redis" >>docker-compose.yml
    fi

    cat >>docker-compose.yml <<EOF
    networks:
      - default
    volumes:
      - ./services/nest/$service/src:/app/src:ro
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:$SERVICE_PORT/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 20s
EOF

    PORT_COUNTER=$((PORT_COUNTER + 1))
  done
fi

# BullMQ Workers - Support both BULL_SERVICES and BULLMQ_WORKERS variables
if [[ -n "$BULL_SERVICES" ]] || [[ "$BULLMQ_ENABLED" == "true" ]]; then
  # Use BULL_SERVICES if set, otherwise use BULLMQ_WORKERS
  WORKER_LIST="${BULL_SERVICES:-$BULLMQ_WORKERS}"
  IFS=',' read -ra BULLMQ_WORKER_LIST <<<"$WORKER_LIST"

  for worker in "${BULLMQ_WORKER_LIST[@]}"; do
    worker=$(echo "$worker" | xargs)

    # Use standard bull directory
    BULL_PATH="./services/bull/$worker"

    cat >>docker-compose.yml <<EOF

  # BullMQ Worker: $worker
  ${PROJECT_NAME}-bull-$worker:
    image: ${PROJECT_NAME}/bull-${worker}:latest
    build:
      context: $BULL_PATH
      dockerfile: Dockerfile
      tags:
        - ${PROJECT_NAME}/bull-${worker}:latest
        - ${PROJECT_NAME}/bull-${worker}:${ENV:-dev}
    container_name: ${PROJECT_NAME}_bull_$worker
    restart: unless-stopped
    environment:
      - NODE_ENV=${ENVIRONMENT:-development}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - BULLMQ_DASHBOARD_ENABLED=${BULLMQ_DASHBOARD_ENABLED}
      - BULLMQ_DASHBOARD_PORT=${BULLMQ_DASHBOARD_PORT}
      - DATABASE_URL=${HASURA_GRAPHQL_DATABASE_URL}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET}
EOF

    if [[ "$BULLMQ_DASHBOARD_ENABLED" == "true" ]]; then
      cat >>docker-compose.yml <<EOF
    ports:
      - "${BULLMQ_DASHBOARD_PORT}:${BULLMQ_DASHBOARD_PORT}"
EOF
      # Increment port for next worker
      BULLMQ_DASHBOARD_PORT=$((BULLMQ_DASHBOARD_PORT + 1))
    fi

    cat >>docker-compose.yml <<EOF
    depends_on:
      - redis
      - postgres
      - hasura
    networks:
      - default
    volumes:
      - $BULL_PATH/src:/app/src:ro
    healthcheck:
      test: ["CMD", "sh", "-c", "ps aux | grep 'node' | grep -v grep || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
EOF
  done
fi

# GoLang Services
if [[ "$GOLANG_ENABLED" == "true" ]]; then
  IFS=',' read -ra GO_SERVICES <<<"$GOLANG_SERVICES"
  PORT_COUNTER=0

  for service in "${GO_SERVICES[@]}"; do
    service=$(echo "$service" | xargs)
    SERVICE_PORT=$((GOLANG_PORT_START + PORT_COUNTER))

    cat >>docker-compose.yml <<EOF

  # GoLang Service: $service
  ${PROJECT_NAME}-go-$service:
    image: ${PROJECT_NAME}/go-${service}:latest
    build:
      context: ./services/go/$service
      dockerfile: Dockerfile
      tags:
        - ${PROJECT_NAME}/go-${service}:latest
        - ${PROJECT_NAME}/go-${service}:${ENV:-dev}
    container_name: ${PROJECT_NAME}_go_$service
    restart: unless-stopped
    environment:
      - GO_ENV=${ENVIRONMENT:-development}
      - PORT=$SERVICE_PORT
      - DATABASE_URL=${HASURA_GRAPHQL_DATABASE_URL}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_URL=redis://redis:6379
    ports:
      - "$SERVICE_PORT:$SERVICE_PORT"
    depends_on:
      - postgres
      - hasura
EOF

    if [[ "$REDIS_ENABLED" == "true" ]]; then
      echo "      - redis" >>docker-compose.yml
    fi

    cat >>docker-compose.yml <<EOF
    networks:
      - default
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:$SERVICE_PORT/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 20s
EOF

    PORT_COUNTER=$((PORT_COUNTER + 1))
  done
fi

# Python Services
if [[ "$PYTHON_ENABLED" == "true" ]]; then
  IFS=',' read -ra PY_SERVICES <<<"$PYTHON_SERVICES"
  PORT_COUNTER=0

  for service in "${PY_SERVICES[@]}"; do
    service=$(echo "$service" | xargs)
    SERVICE_PORT=$((PYTHON_PORT_START + PORT_COUNTER))

    cat >>docker-compose.yml <<EOF

  # Python Service: $service
  ${PROJECT_NAME}-py-$service:
    image: ${PROJECT_NAME}/py-${service}:latest
    build:
      context: ./services/py/$service
      dockerfile: Dockerfile
      tags:
        - ${PROJECT_NAME}/py-${service}:latest
        - ${PROJECT_NAME}/py-${service}:${ENV:-dev}
    container_name: ${PROJECT_NAME}_py_$service
    restart: unless-stopped
    environment:
      - PYTHON_ENV=${ENVIRONMENT:-development}
      - PORT=$SERVICE_PORT
      - DATABASE_URL=${HASURA_GRAPHQL_DATABASE_URL}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_URL=redis://redis:6379
    ports:
      - "$SERVICE_PORT:$SERVICE_PORT"
    depends_on:
      - postgres
      - hasura
EOF

    if [[ "$REDIS_ENABLED" == "true" ]]; then
      echo "      - redis" >>docker-compose.yml
    fi

    cat >>docker-compose.yml <<EOF
    networks:
      - default
    volumes:
      - ./services/py/$service:/app:ro
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:$SERVICE_PORT/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 20s
EOF

    PORT_COUNTER=$((PORT_COUNTER + 1))
  done
fi
# Generate Functions service if enabled
if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
  FUNCTIONS_PORT="${FUNCTIONS_PORT:-4300}"
  cat >>docker-compose.yml <<EOF

  unity-functions:
    image: unity/functions:latest
    build:
      context: ./functions
      dockerfile: Dockerfile
      tags:
        - unity/functions:latest
        - unity/functions:dev
    container_name: unity_functions
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

  unity-dashboard:
    image: unity/dashboard:latest
    build:
      context: ./dashboard
      dockerfile: Dockerfile
      tags:
        - unity/dashboard:latest
        - unity/dashboard:dev
    container_name: unity_dashboard
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

# Handle CS_ custom services
# Format: CS_1=service_name:type:port:options
for var in $(env | grep "^CS_[0-9]*=" | sort); do
  # Extract the key and value
  CS_KEY="${var%%=*}"
  CS_VALUE="${var#*=}"
  
  # Parse the CS value format: name:type:port:options
  IFS=':' read -r SERVICE_NAME SERVICE_TYPE SERVICE_PORT SERVICE_OPTIONS <<< "$CS_VALUE"
  
  # Skip if service name is empty
  if [[ -z "$SERVICE_NAME" ]]; then
    continue
  fi
  
  # Default port if not specified
  if [[ -z "$SERVICE_PORT" ]]; then
    SERVICE_PORT=$((4000 + ${CS_KEY#CS_}))
  fi
  
  # Check if the service directory exists
  if [[ -d "./services/${SERVICE_NAME}" ]]; then
    cat >>docker-compose.yml <<EOF

  # Custom Service: ${SERVICE_NAME}
  ${SERVICE_NAME}:
    build:
      context: ./services/${SERVICE_NAME}
      dockerfile: Dockerfile
    container_name: ${PROJECT_NAME}_${SERVICE_NAME}
    restart: unless-stopped
    environment:
      - NODE_ENV=${ENVIRONMENT:-development}
      - PORT=${SERVICE_PORT}
      - DATABASE_URL=postgres://postgres:${POSTGRES_PASSWORD:-changeme}@postgres:${POSTGRES_PORT:-5432}/${POSTGRES_DB:-nhost}
      - HASURA_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=${HASURA_GRAPHQL_ADMIN_SECRET}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - BASE_DOMAIN=${BASE_DOMAIN}
    ports:
      - "${SERVICE_PORT}:${SERVICE_PORT}"
    depends_on:
      - postgres
      - hasura
EOF
    
    # Add Redis dependency if enabled
    if [[ "$REDIS_ENABLED" == "true" ]]; then
      echo "      - redis" >>docker-compose.yml
    fi
    
    # Complete the service definition
    cat >>docker-compose.yml <<EOF
    networks:
      - default
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${SERVICE_PORT}/health"]
      interval: 30s
      timeout: 10s
      retries: 3
EOF
  fi
done
