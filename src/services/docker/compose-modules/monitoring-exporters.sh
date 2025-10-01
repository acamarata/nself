#!/usr/bin/env bash
# monitoring-exporters.sh - Generate monitoring exporter services

# Generate Tempo service
generate_tempo_service() {
  [[ "${TEMPO_ENABLED:-false}" != "true" ]] && return 0

  cat <<EOF

  # Tempo - Distributed Tracing
  tempo:
    image: grafana/tempo:${TEMPO_VERSION:-latest}
    container_name: \${PROJECT_NAME}_tempo
    restart: unless-stopped
    command: [ "-config.file=/etc/tempo.yaml" ]
    volumes:
      - ./monitoring/tempo/tempo.yml:/etc/tempo.yaml:ro
      - tempo_data:/var/tempo
    ports:
      - "\${TEMPO_PORT:-3200}:3200"
      - "14268:14268"  # Jaeger ingest
    networks:
      - \${DOCKER_NETWORK}
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:3200/ready"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate Alertmanager service
generate_alertmanager_service() {
  [[ "${ALERTMANAGER_ENABLED:-false}" != "true" ]] && return 0

  cat <<EOF

  # Alertmanager - Alert Routing
  alertmanager:
    image: prom/alertmanager:${ALERTMANAGER_VERSION:-latest}
    container_name: \${PROJECT_NAME}_alertmanager
    restart: unless-stopped
    volumes:
      - ./monitoring/alertmanager:/etc/alertmanager:ro
      - alertmanager_data:/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
    ports:
      - "\${ALERTMANAGER_PORT:-9093}:9093"
    networks:
      - \${DOCKER_NETWORK}
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:9093/-/healthy"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate cAdvisor service
generate_cadvisor_service() {
  [[ "${CADVISOR_ENABLED:-false}" != "true" ]] && return 0

  # Detect OS for proper volume mounts
  local volumes=""
  if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS doesn't have /cgroup, use Docker-only mounts
    volumes="      - /var/run:/var/run:ro
      - /var/lib/docker:/var/lib/docker:ro"
  else
    # Linux has full cgroup support
    volumes="      - /:/rootfs:ro
      - /var/run:/var/run:ro
      - /sys:/sys:ro
      - /var/lib/docker:/var/lib/docker:ro
      - /cgroup:/cgroup:ro"
  fi

  cat <<EOF

  # cAdvisor - Container Metrics
  cadvisor:
    image: gcr.io/cadvisor/cadvisor:${CADVISOR_VERSION:-latest}
    container_name: \${PROJECT_NAME}_cadvisor
    restart: unless-stopped
    privileged: true
    volumes:
$volumes
    ports:
      - "\${CADVISOR_PORT:-8082}:8080"
    networks:
      - \${DOCKER_NETWORK}
    command:
      - '--housekeeping_interval=10s'
      - '--docker_only=true'
EOF

  # Add devices section only for Linux
  if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "    devices:"
    echo "      - /dev/kmsg:/dev/kmsg"
  fi
}

# Generate Node Exporter service
generate_node_exporter_service() {
  [[ "${NODE_EXPORTER_ENABLED:-false}" != "true" ]] && return 0

  cat <<EOF

  # Node Exporter - Host Metrics
  node-exporter:
    image: prom/node-exporter:${NODE_EXPORTER_VERSION:-latest}
    container_name: \${PROJECT_NAME}_node_exporter
    restart: unless-stopped
    pid: host
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.rootfs=/rootfs'
      - '--path.sysfs=/host/sys'
      - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)($$|/)'
    ports:
      - "\${NODE_EXPORTER_PORT:-9100}:9100"
    networks:
      - \${DOCKER_NETWORK}
EOF
}

# Generate Postgres Exporter service
generate_postgres_exporter_service() {
  [[ "${POSTGRES_EXPORTER_ENABLED:-false}" != "true" ]] && return 0
  # Note: PostgreSQL is always required in nself, so no need to check POSTGRES_ENABLED

  cat <<EOF

  # Postgres Exporter - Database Metrics
  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:${POSTGRES_EXPORTER_VERSION:-latest}
    container_name: \${PROJECT_NAME}_postgres_exporter
    restart: unless-stopped
    environment:
      DATA_SOURCE_NAME: "postgresql://\${POSTGRES_USER}:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}?sslmode=disable"
    ports:
      - "\${POSTGRES_EXPORTER_PORT:-9187}:9187"
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      - postgres
EOF
}

# Generate Redis Exporter service
generate_redis_exporter_service() {
  # Redis exporter requires both monitoring AND Redis to be enabled
  [[ "${REDIS_EXPORTER_ENABLED:-false}" != "true" ]] && return 0
  [[ "${REDIS_ENABLED:-false}" != "true" ]] && return 0

  local redis_addr="redis://redis:6379"
  [[ -n "${REDIS_PASSWORD}" ]] && redis_addr="redis://:\${REDIS_PASSWORD}@redis:6379"

  cat <<EOF

  # Redis Exporter - Redis Metrics
  redis-exporter:
    image: oliver006/redis_exporter:${REDIS_EXPORTER_VERSION:-latest}
    container_name: \${PROJECT_NAME}_redis_exporter
    restart: unless-stopped
    environment:
      REDIS_ADDR: "${redis_addr}"
    ports:
      - "\${REDIS_EXPORTER_PORT:-9121}:9121"
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      - redis
EOF
}

# Generate all monitoring exporters
generate_monitoring_exporters() {
  local has_exporters=false

  # Check if any exporters are enabled
  [[ "${TEMPO_ENABLED:-false}" == "true" ]] && has_exporters=true
  [[ "${ALERTMANAGER_ENABLED:-false}" == "true" ]] && has_exporters=true
  [[ "${CADVISOR_ENABLED:-false}" == "true" ]] && has_exporters=true
  [[ "${NODE_EXPORTER_ENABLED:-false}" == "true" ]] && has_exporters=true
  [[ "${POSTGRES_EXPORTER_ENABLED:-false}" == "true" ]] && has_exporters=true
  [[ "${REDIS_EXPORTER_ENABLED:-false}" == "true" ]] && has_exporters=true

  [[ "$has_exporters" != "true" ]] && return 0

  echo ""
  echo "  # ============================================"
  echo "  # Monitoring Exporters & Tracing"
  echo "  # ============================================"

  generate_tempo_service
  generate_alertmanager_service
  generate_cadvisor_service
  generate_node_exporter_service
  generate_postgres_exporter_service
  generate_redis_exporter_service
}

# Export functions
export -f generate_tempo_service
export -f generate_alertmanager_service
export -f generate_cadvisor_service
export -f generate_node_exporter_service
export -f generate_postgres_exporter_service
export -f generate_redis_exporter_service
export -f generate_monitoring_exporters