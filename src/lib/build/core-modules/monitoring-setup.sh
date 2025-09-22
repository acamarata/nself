#!/usr/bin/env bash
# monitoring-setup.sh - Set up monitoring configuration files during build
# Bash 3.2 compatible, cross-platform

# Setup monitoring configuration files
setup_monitoring_configs() {
  local monitoring_enabled="${GRAFANA_ENABLED:-false}"
  local loki_enabled="${LOKI_ENABLED:-false}"
  local prometheus_enabled="${PROMETHEUS_ENABLED:-false}"

  # Skip if no monitoring services are enabled
  if [[ "$monitoring_enabled" != "true" ]] && [[ "$loki_enabled" != "true" ]] && [[ "$prometheus_enabled" != "true" ]]; then
    return 0
  fi

  echo "Setting up monitoring configurations..."

  # Create Loki config if enabled
  if [[ "$loki_enabled" == "true" ]] || [[ "$monitoring_enabled" == "true" ]]; then
    if [[ ! -f "monitoring/loki/local-config.yaml" ]]; then
      mkdir -p monitoring/loki
      cat > monitoring/loki/local-config.yaml <<'EOF'
auth_enabled: false

server:
  http_listen_port: 3100
  grpc_listen_port: 9096

common:
  path_prefix: /loki
  storage:
    filesystem:
      chunks_directory: /loki/chunks
      rules_directory: /loki/rules
  replication_factor: 1
  ring:
    instance_addr: 127.0.0.1
    kvstore:
      store: inmemory

schema_config:
  configs:
    - from: 2020-10-24
      store: tsdb
      object_store: filesystem
      schema: v13
      index:
        prefix: index_
        period: 24h

ruler:
  alertmanager_url: http://localhost:9093

limits_config:
  allow_structured_metadata: false
EOF
    fi
  fi

  # Create Prometheus config if enabled
  if [[ "$prometheus_enabled" == "true" ]] || [[ "$monitoring_enabled" == "true" ]]; then
    if [[ ! -f "monitoring/prometheus/prometheus.yml" ]]; then
      mkdir -p monitoring/prometheus
      cat > monitoring/prometheus/prometheus.yml <<'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node'
    static_configs:
      - targets: ['host.docker.internal:9100']

  - job_name: 'docker'
    static_configs:
      - targets: ['host.docker.internal:9323']

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:5432']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis:6379']

  - job_name: 'hasura'
    static_configs:
      - targets: ['hasura:8080']
EOF
    fi
  fi

  # Create Promtail config if enabled
  if [[ "$loki_enabled" == "true" ]] || [[ "$monitoring_enabled" == "true" ]]; then
    if [[ ! -f "monitoring/promtail/config.yml" ]]; then
      mkdir -p monitoring/promtail
      cat > monitoring/promtail/config.yml <<'EOF'
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: containers
    static_configs:
      - targets:
          - localhost
        labels:
          job: containerlogs
          __path__: /var/lib/docker/containers/*/*log
    pipeline_stages:
      - json:
          expressions:
            output: log
            stream: stream
            attrs:
      - json:
          expressions:
            tag:
          source: attrs
      - regex:
          expression: (?P<container_name>(?:[^|]*))\|(?P<image_name>(?:[^|]*))
          source: tag
      - timestamp:
          format: RFC3339Nano
          source: time
      - labels:
          stream:
          container_name:
          image_name:
      - output:
          source: output
EOF
    fi
  fi

  # Create Grafana provisioning if enabled
  if [[ "$monitoring_enabled" == "true" ]]; then
    if [[ ! -d "monitoring/grafana/provisioning" ]]; then
      mkdir -p monitoring/grafana/provisioning/datasources
      mkdir -p monitoring/grafana/provisioning/dashboards

      # Create datasources config
      cat > monitoring/grafana/provisioning/datasources/prometheus.yml <<'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    jsonData:
      httpHeaderName1: 'X-Scope-OrgID'
    secureJsonData:
      httpHeaderValue1: '1'
EOF

      # Create dashboards config
      cat > monitoring/grafana/provisioning/dashboards/dashboards.yml <<'EOF'
apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    options:
      path: /etc/grafana/provisioning/dashboards
EOF
    fi
  fi

  # Create Tempo config if enabled
  if [[ "${TEMPO_ENABLED:-false}" == "true" ]]; then
    if [[ ! -f "monitoring/tempo/tempo.yml" ]]; then
      mkdir -p monitoring/tempo
      cat > monitoring/tempo/tempo.yml <<'EOF'
server:
  http_listen_port: 3200

distributor:
  receivers:
    otlp:
      protocols:
        grpc:

storage:
  trace:
    backend: local
    local:
      path: /tmp/tempo/blocks

metrics_generator:
  registry:
    external_labels:
      source: tempo
  storage:
    path: /tmp/tempo/generator/wal
EOF
    fi
  fi

  return 0
}

# Export function
export -f setup_monitoring_configs