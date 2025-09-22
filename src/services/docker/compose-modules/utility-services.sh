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

# Generate Adminer database management tool
generate_adminer_service() {
  local enabled="${ADMINER_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Adminer - Database Management
  adminer:
    image: adminer:${ADMINER_VERSION:-latest}
    container_name: \${PROJECT_NAME}_adminer
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      ADMINER_DEFAULT_SERVER: postgres
      ADMINER_DESIGN: \${ADMINER_DESIGN:-pepa-linha}
    ports:
      - "\${ADMINER_PORT:-8090}:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate BullMQ Dashboard for queue monitoring
generate_bullmq_dashboard() {
  local enabled="${BULLMQ_DASHBOARD_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # BullMQ Dashboard - Queue Monitoring
  bullmq-dashboard:
    image: taskforcesh/bullmq-dashboard:${BULLMQ_DASHBOARD_VERSION:-latest}
    container_name: \${PROJECT_NAME}_bullmq_dashboard
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      redis:
        condition: service_healthy
    environment:
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_PASSWORD: \${REDIS_PASSWORD:-}
    ports:
      - "\${BULLMQ_DASHBOARD_PORT:-3010}:3000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate pgAdmin for PostgreSQL management
generate_pgadmin_service() {
  local enabled="${PGADMIN_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # pgAdmin - PostgreSQL Management
  pgadmin:
    image: dpage/pgadmin4:${PGADMIN_VERSION:-latest}
    container_name: \${PROJECT_NAME}_pgadmin
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      PGADMIN_DEFAULT_EMAIL: \${PGADMIN_DEFAULT_EMAIL:-admin@admin.com}
      PGADMIN_DEFAULT_PASSWORD: \${PGADMIN_DEFAULT_PASSWORD:-admin}
      PGADMIN_CONFIG_SERVER_MODE: "False"
      PGADMIN_CONFIG_MASTER_PASSWORD_REQUIRED: "False"
    volumes:
      - pgadmin_data:/var/lib/pgadmin
    ports:
      - "\${PGADMIN_PORT:-5050}:80"
    healthcheck:
      test: ["CMD", "wget", "-O", "-", "http://localhost:80/misc/ping"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate Swagger UI for API documentation
generate_swagger_ui() {
  local enabled="${SWAGGER_UI_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Swagger UI - API Documentation
  swagger-ui:
    image: swaggerapi/swagger-ui:${SWAGGER_UI_VERSION:-latest}
    container_name: \${PROJECT_NAME}_swagger_ui
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    environment:
      SWAGGER_JSON: \${SWAGGER_JSON:-/swagger/swagger.json}
      BASE_URL: \${SWAGGER_BASE_URL:-/swagger}
      DEEP_LINKING: "true"
      PERSIST_AUTHORIZATION: "true"
    volumes:
      - ./swagger:/swagger:ro
    ports:
      - "\${SWAGGER_UI_PORT:-8091}:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate Portainer for Docker management
generate_portainer_service() {
  local enabled="${PORTAINER_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Portainer - Docker Management
  portainer:
    image: portainer/portainer-ce:${PORTAINER_VERSION:-latest}
    container_name: \${PROJECT_NAME}_portainer
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    security_opt:
      - no-new-privileges:true
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - portainer_data:/data
    ports:
      - "\${PORTAINER_PORT:-9000}:9000"
      - "\${PORTAINER_EDGE_PORT:-8000}:8000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
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

# Main function to generate all utility services
generate_utility_services() {
  generate_mailpit_service
  generate_adminer_service
  generate_bullmq_dashboard
  generate_pgadmin_service
  generate_swagger_ui
  generate_portainer_service
  generate_backup_service
}

# Export functions
export -f generate_mailpit_service
export -f generate_adminer_service
export -f generate_bullmq_dashboard
export -f generate_pgadmin_service
export -f generate_swagger_ui
export -f generate_portainer_service
export -f generate_backup_service
export -f generate_utility_services