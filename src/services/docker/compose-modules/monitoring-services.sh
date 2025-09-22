#!/usr/bin/env bash
# monitoring-services.sh - Generate monitoring and search service definitions
# This module handles MLflow, search engines, logging, and monitoring services

# Generate MLflow service
generate_mlflow_service() {
  local enabled="${MLFLOW_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # MLflow - Machine Learning Lifecycle Platform
  mlflow:
    image: python:3.9-slim
    container_name: \${PROJECT_NAME}_mlflow
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      MLFLOW_BACKEND_STORE_URI: postgresql://\${POSTGRES_USER:-postgres}:\${POSTGRES_PASSWORD}@postgres:5432/mlflow
      MLFLOW_DEFAULT_ARTIFACT_ROOT: \${MLFLOW_ARTIFACT_ROOT:-/mlflow/artifacts}
      MLFLOW_HOST: 0.0.0.0
      MLFLOW_PORT: \${MLFLOW_PORT:-5005}
    command: >
      sh -c "
        pip install --no-cache-dir mlflow psycopg2-binary &&
        mlflow db upgrade \$\${MLFLOW_BACKEND_STORE_URI} &&
        mlflow server
          --backend-store-uri \$\${MLFLOW_BACKEND_STORE_URI}
          --default-artifact-root \$\${MLFLOW_DEFAULT_ARTIFACT_ROOT}
          --host \$\${MLFLOW_HOST}
          --port \$\${MLFLOW_PORT}
          --serve-artifacts
      "
    volumes:
      - mlflow_data:/mlflow/artifacts
    ports:
      - "\${MLFLOW_PORT:-5005}:\${MLFLOW_PORT:-5005}"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:\${MLFLOW_PORT:-5005}/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
EOF
}

# Generate search services (Meilisearch, Typesense, or Sonic)
generate_search_services() {
  # Meilisearch
  if [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]]; then
    cat <<EOF

  # Meilisearch - Lightning Fast Search
  meilisearch:
    image: getmeili/meilisearch:${MEILISEARCH_VERSION:-v1.5}
    container_name: \${PROJECT_NAME}_meilisearch
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    environment:
      MEILI_MASTER_KEY: \${MEILISEARCH_MASTER_KEY:-masterKey}
      MEILI_NO_ANALYTICS: \${MEILISEARCH_NO_ANALYTICS:-true}
      MEILI_ENV: \${MEILISEARCH_ENV:-development}
      MEILI_LOG_LEVEL: \${MEILISEARCH_LOG_LEVEL:-INFO}
      MEILI_DB_PATH: /meili_data
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

  # Typesense
  if [[ "${TYPESENSE_ENABLED:-false}" == "true" ]]; then
    cat <<EOF

  # Typesense - Fast Search Engine
  typesense:
    image: typesense/typesense:${TYPESENSE_VERSION:-0.25.1}
    container_name: \${PROJECT_NAME}_typesense
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    environment:
      TYPESENSE_DATA_DIR: /data
      TYPESENSE_API_KEY: \${TYPESENSE_API_KEY:-xyz}
      TYPESENSE_ENABLE_CORS: \${TYPESENSE_ENABLE_CORS:-true}
    command: "--data-dir /data --api-key=\${TYPESENSE_API_KEY:-xyz} --enable-cors"
    volumes:
      - typesense_data:/data
    ports:
      - "\${TYPESENSE_PORT:-8108}:8108"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8108/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
  fi

  # Sonic
  if [[ "${SONIC_ENABLED:-false}" == "true" ]]; then
    cat <<EOF

  # Sonic - Fast, lightweight search backend
  sonic:
    image: valeriansaliou/sonic:${SONIC_VERSION:-v1.4.8}
    container_name: \${PROJECT_NAME}_sonic
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    volumes:
      - sonic_data:/var/lib/sonic/store
      - ./sonic/config.cfg:/etc/sonic.cfg:ro
    ports:
      - "\${SONIC_PORT:-1491}:1491"
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "1491"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
  fi
}

# Generate Grafana monitoring service
generate_grafana_service() {
  local enabled="${GRAFANA_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Grafana - Monitoring Dashboard
  grafana:
    image: grafana/grafana:${GRAFANA_VERSION:-latest}
    container_name: \${PROJECT_NAME}_grafana
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    environment:
      GF_SECURITY_ADMIN_USER: \${GRAFANA_ADMIN_USER:-admin}
      GF_SECURITY_ADMIN_PASSWORD: \${GRAFANA_ADMIN_PASSWORD:-admin}
      GF_INSTALL_PLUGINS: \${GRAFANA_PLUGINS:-}
      GF_SERVER_ROOT_URL: \${GRAFANA_ROOT_URL:-http://localhost:3000}
      GF_ANALYTICS_REPORTING_ENABLED: "false"
      GF_ANALYTICS_CHECK_FOR_UPDATES: "false"
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana/dashboards:/etc/grafana/provisioning/dashboards:ro
      - ./monitoring/grafana/datasources:/etc/grafana/provisioning/datasources:ro
    ports:
      - "\${GRAFANA_PORT:-3000}:3000"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/api/health"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate Prometheus service
generate_prometheus_service() {
  local enabled="${PROMETHEUS_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Prometheus - Metrics Collection
  prometheus:
    image: prom/prometheus:${PROMETHEUS_VERSION:-latest}
    container_name: \${PROJECT_NAME}_prometheus
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
      - '--web.enable-lifecycle'
    volumes:
      - prometheus_data:/prometheus
      - ./monitoring/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
    ports:
      - "\${PROMETHEUS_PORT:-9090}:9090"
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:9090/-/healthy"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate Loki logging service
generate_loki_service() {
  local enabled="${LOKI_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Loki - Log Aggregation
  loki:
    image: grafana/loki:${LOKI_VERSION:-2.9.0}
    container_name: \${PROJECT_NAME}_loki
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    command: -config.file=/etc/loki/local-config.yaml
    volumes:
      - loki_data:/loki
      - ./monitoring/loki/local-config.yaml:/etc/loki/local-config.yaml:ro
    ports:
      - "\${LOKI_PORT:-3100}:3100"
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:3100/ready"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Generate Promtail log collector
generate_promtail_service() {
  local enabled="${PROMTAIL_ENABLED:-false}"
  [[ "$enabled" != "true" ]] && return 0

  cat <<EOF

  # Promtail - Log Collector for Loki
  promtail:
    image: grafana/promtail:${PROMTAIL_VERSION:-2.9.0}
    container_name: \${PROJECT_NAME}_promtail
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
    depends_on:
      - loki
    command: -config.file=/etc/promtail/config.yml
    volumes:
      - /var/log:/var/log:ro
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - ./monitoring/promtail/config.yml:/etc/promtail/config.yml:ro
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:9080/ready"]
      interval: 30s
      timeout: 10s
      retries: 5
EOF
}

# Main function to generate all monitoring services
generate_monitoring_services() {
  # Generate individual services
  generate_mlflow_service
  generate_search_services
  generate_grafana_service
  generate_prometheus_service
  generate_loki_service
  generate_promtail_service
}

# Export functions
export -f generate_mlflow_service
export -f generate_search_services
export -f generate_grafana_service
export -f generate_prometheus_service
export -f generate_loki_service
export -f generate_promtail_service
export -f generate_monitoring_services