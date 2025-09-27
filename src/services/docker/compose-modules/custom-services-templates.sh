#!/usr/bin/env bash
# custom-services-templates.sh - Generate Docker services from template-based CS_ variables

# Generate custom service from template-based CS_ variable
generate_template_based_service() {
  local index="$1"
  local service_name="$2"
  local template_type="$3"
  local service_port="$4"

  # Skip if service directory doesn't exist (template not copied)
  [[ ! -d "services/$service_name" ]] && return 0

  cat <<EOF

  # Custom Service ${index}: ${service_name}
  ${service_name}:
    build:
      context: ./services/${service_name}
      dockerfile: Dockerfile
    container_name: \${PROJECT_NAME}_${service_name}
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
EOF

  # Add ports if specified
  if [[ -n "$service_port" && "$service_port" != "0" ]]; then
    cat <<EOF
    ports:
      - "${service_port}:${service_port}"
EOF
  fi

  # Add environment variables
  cat <<EOF
    environment:
      - ENV=\${ENV:-dev}
      - NODE_ENV=\${ENV:-dev}
      - APP_ENV=\${ENV:-dev}
      - ENVIRONMENT=\${ENV:-dev}
      - PROJECT_NAME=\${PROJECT_NAME}
      - BASE_DOMAIN=\${BASE_DOMAIN:-localhost}
      - SERVICE_NAME=${service_name}
      - SERVICE_PORT=${service_port}
      - PORT=${service_port}
      - POSTGRES_HOST=postgres
      - POSTGRES_PORT=5432
      - POSTGRES_DB=\${POSTGRES_DB}
      - POSTGRES_USER=\${POSTGRES_USER}
      - POSTGRES_PASSWORD=\${POSTGRES_PASSWORD}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=\${REDIS_PASSWORD:-}
      - HASURA_GRAPHQL_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=\${HASURA_GRAPHQL_ADMIN_SECRET}
EOF

  # Add volumes for development (language-specific exclusions)
  cat <<EOF
    volumes:
      - ./services/${service_name}:/app
EOF

  # Add language-specific volume exclusions based on template type
  case "$template_type" in
    *js|*ts|node*|express*|nest*|fastify*|hono*|bullmq*|bun|deno)
      echo "      - /app/node_modules"
      ;;
    py*|fastapi|django*|flask|celery)
      echo "      - /app/.venv"
      echo "      - /app/__pycache__"
      ;;
    go|grpc|gin|echo|fiber)
      echo "      - /app/vendor"
      ;;
    java*|spring*|kotlin*|ktor)
      echo "      - /app/target"
      echo "      - /app/.gradle"
      ;;
    rust*|actix*)
      echo "      - /app/target"
      echo "      - /app/Cargo.lock"
      ;;
    php*|laravel)
      echo "      - /app/vendor"
      ;;
    ruby*|rails|sinatra)
      echo "      - /app/vendor"
      ;;
  esac

  # Add resource limits if specified (smart defaults)
  local memory_var="CS_${index}_MEMORY"
  local cpu_var="CS_${index}_CPU"
  local replicas_var="CS_${index}_REPLICAS"

  # Only add deploy section if any resource constraints are specified
  if [[ -n "${!memory_var:-}" ]] || [[ -n "${!cpu_var:-}" ]] || [[ -n "${!replicas_var:-}" ]]; then
    cat <<EOF
    deploy:
      resources:
        limits:
          memory: \${${memory_var}:-512M}
          cpus: '\${${cpu_var}:-0.5}'
      replicas: \${${replicas_var}:-1}
EOF
  fi

  # Add dependencies
  cat <<EOF
    depends_on:
      - postgres
      - redis
EOF

  [[ "${HASURA_ENABLED:-false}" == "true" ]] && echo "      - hasura"

  # Add healthcheck if port is exposed
  if [[ -n "$service_port" && "$service_port" != "0" ]]; then
    # Use appropriate health check based on template type
    case "$template_type" in
      *go*|*grpc*|*rust*)
        # These containers might not have curl, use a simple TCP check
        cat <<EOF
    healthcheck:
      test: ["CMD", "/bin/sh", "-c", "nc -z localhost ${service_port} || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
EOF
        ;;
      *)
        # Most other containers should have curl or wget
        cat <<EOF
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${service_port}/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
EOF
        ;;
    esac
  fi
}

# Generate all template-based custom services
generate_template_custom_services() {
  local services_found=false

  # Check for CS_ variables (format: service_name:template_type:port)
  for i in {1..20}; do
    local cs_var="CS_${i}"
    local cs_value="${!cs_var:-}"

    [[ -z "$cs_value" ]] && continue

    # Parse CS_ format: service_name:template_type:port
    IFS=':' read -r service_name template_type port <<< "$cs_value"

    # Skip if essential fields are missing
    [[ -z "$service_name" || -z "$template_type" ]] && continue

    if [[ "$services_found" == "false" ]]; then
      echo ""
      echo "  # ============================================"
      echo "  # Custom Services (from templates)"
      echo "  # ============================================"
      services_found=true
    fi

    generate_template_based_service "$i" "$service_name" "$template_type" "${port:-8000}"
  done
}

# Export functions
export -f generate_template_based_service
export -f generate_template_custom_services