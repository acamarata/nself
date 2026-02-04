#!/usr/bin/env bash
# alert-rules.sh - Default Prometheus alert rules generator
# Part of nself v0.9.8 - Production Features

set -euo pipefail

# Generate Prometheus alert rules
generate_prometheus_alerts() {
  cat <<'EOF'
groups:
  - name: nself_infrastructure
    interval: 30s
    rules:
      # =============================================================================
      # HIGH PRIORITY ALERTS
      # =============================================================================

      - alert: ServiceDown
        expr: up == 0
        for: 1m
        labels:
          severity: critical
          category: availability
        annotations:
          summary: "Service {{ $labels.job }} is down"
          description: "{{ $labels.job }} has been down for more than 1 minute."

      - alert: HighCPUUsage
        expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 5m
        labels:
          severity: warning
          category: performance
        annotations:
          summary: "High CPU usage on {{ $labels.instance }}"
          description: "CPU usage is above 80% (current: {{ $value }}%)"

      - alert: HighMemoryUsage
        expr: (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 90
        for: 5m
        labels:
          severity: critical
          category: performance
        annotations:
          summary: "High memory usage on {{ $labels.instance }}"
          description: "Memory usage is above 90% (current: {{ $value }}%)"

      - alert: LowDiskSpace
        expr: (node_filesystem_avail_bytes{fstype!~"tmpfs|fuse.lxcfs"} / node_filesystem_size_bytes{fstype!~"tmpfs|fuse.lxcfs"}) * 100 < 10
        for: 5m
        labels:
          severity: critical
          category: storage
        annotations:
          summary: "Low disk space on {{ $labels.instance }}"
          description: "Disk {{ $labels.mountpoint }} has less than 10% free space (current: {{ $value }}%)"

      # =============================================================================
      # DATABASE ALERTS
      # =============================================================================

      - alert: PostgreSQLDown
        expr: pg_up == 0
        for: 1m
        labels:
          severity: critical
          category: database
        annotations:
          summary: "PostgreSQL is down"
          description: "PostgreSQL database is not responding"

      - alert: PostgreSQLTooManyConnections
        expr: sum(pg_stat_activity_count) > pg_settings_max_connections * 0.8
        for: 5m
        labels:
          severity: warning
          category: database
        annotations:
          summary: "PostgreSQL has too many connections"
          description: "PostgreSQL is using {{ $value }}% of max connections"

      - alert: PostgreSQLSlowQueries
        expr: rate(pg_stat_statements_mean_exec_time_seconds[5m]) > 1
        for: 5m
        labels:
          severity: warning
          category: database
        annotations:
          summary: "PostgreSQL has slow queries"
          description: "Average query execution time is {{ $value }}s"

      - alert: PostgreSQLDeadlocks
        expr: rate(pg_stat_database_deadlocks[5m]) > 0
        for: 1m
        labels:
          severity: warning
          category: database
        annotations:
          summary: "PostgreSQL deadlocks detected"
          description: "{{ $value }} deadlocks per second"

      # =============================================================================
      # REDIS ALERTS (if enabled)
      # =============================================================================

      - alert: RedisDown
        expr: redis_up == 0
        for: 1m
        labels:
          severity: critical
          category: cache
        annotations:
          summary: "Redis is down"
          description: "Redis cache is not responding"

      - alert: RedisHighMemoryUsage
        expr: (redis_memory_used_bytes / redis_memory_max_bytes) * 100 > 90
        for: 5m
        labels:
          severity: warning
          category: cache
        annotations:
          summary: "Redis memory usage is high"
          description: "Redis is using {{ $value }}% of max memory"

      - alert: RedisTooManyConnections
        expr: redis_connected_clients > 1000
        for: 5m
        labels:
          severity: warning
          category: cache
        annotations:
          summary: "Redis has too many connections"
          description: "Redis has {{ $value }} connected clients"

      # =============================================================================
      # CONTAINER ALERTS
      # =============================================================================

      - alert: ContainerHighCPU
        expr: rate(container_cpu_usage_seconds_total[5m]) * 100 > 80
        for: 5m
        labels:
          severity: warning
          category: containers
        annotations:
          summary: "Container {{ $labels.name }} has high CPU usage"
          description: "CPU usage is {{ $value }}%"

      - alert: ContainerHighMemory
        expr: (container_memory_usage_bytes / container_spec_memory_limit_bytes) * 100 > 90
        for: 5m
        labels:
          severity: warning
          category: containers
        annotations:
          summary: "Container {{ $labels.name }} has high memory usage"
          description: "Memory usage is {{ $value }}%"

      - alert: ContainerRestarting
        expr: rate(container_restart_count[5m]) > 0
        for: 1m
        labels:
          severity: warning
          category: containers
        annotations:
          summary: "Container {{ $labels.name }} is restarting"
          description: "Container has restarted {{ $value }} times"

      # =============================================================================
      # SSL CERTIFICATE ALERTS
      # =============================================================================

      - alert: SSLCertificateExpiringSoon
        expr: (ssl_certificate_expiry_seconds - time()) / 86400 < 30
        for: 1h
        labels:
          severity: warning
          category: security
        annotations:
          summary: "SSL certificate expiring soon"
          description: "SSL certificate for {{ $labels.domain }} expires in {{ $value }} days"

      - alert: SSLCertificateExpired
        expr: ssl_certificate_expiry_seconds - time() < 0
        for: 1m
        labels:
          severity: critical
          category: security
        annotations:
          summary: "SSL certificate has expired"
          description: "SSL certificate for {{ $labels.domain }} has expired"

      # =============================================================================
      # APPLICATION ALERTS
      # =============================================================================

      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
          category: application
        annotations:
          summary: "High error rate on {{ $labels.service }}"
          description: "Error rate is {{ $value }}%"

      - alert: SlowResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
          category: application
        annotations:
          summary: "Slow response time on {{ $labels.service }}"
          description: "95th percentile response time is {{ $value }}s"

      # =============================================================================
      # HASURA ALERTS
      # =============================================================================

      - alert: HasuraHighLatency
        expr: histogram_quantile(0.95, rate(hasura_graphql_requests_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
          category: graphql
        annotations:
          summary: "Hasura has high latency"
          description: "95th percentile latency is {{ $value }}s"

      - alert: HasuraErrorRate
        expr: rate(hasura_graphql_requests_total{status="error"}[5m]) / rate(hasura_graphql_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
          category: graphql
        annotations:
          summary: "Hasura has high error rate"
          description: "Error rate is {{ $value }}%"

      # =============================================================================
      # BACKUP ALERTS
      # =============================================================================

      - alert: BackupFailed
        expr: time() - backup_last_success_timestamp_seconds > 86400
        for: 1h
        labels:
          severity: warning
          category: backup
        annotations:
          summary: "Backup has not succeeded recently"
          description: "Last successful backup was {{ $value }} seconds ago"

      - alert: NoRecentBackup
        expr: time() - backup_last_success_timestamp_seconds > 172800
        for: 1h
        labels:
          severity: critical
          category: backup
        annotations:
          summary: "No recent backup found"
          description: "No successful backup in the last 48 hours"

EOF
}

# Generate Alertmanager configuration
generate_alertmanager_config() {
  local slack_webhook="${SLACK_WEBHOOK_URL:-}"
  local email_to="${ALERT_EMAIL_TO:-}"
  local email_from="${ALERT_EMAIL_FROM:-nself@localhost}"
  local smtp_host="${SMTP_HOST:-localhost}"
  local smtp_port="${SMTP_PORT:-587}"

  cat <<EOF
global:
  resolve_timeout: 5m
  slack_api_url: '${slack_webhook}'
  smtp_smarthost: '${smtp_host}:${smtp_port}'
  smtp_from: '${email_from}'
  smtp_auth_username: '${SMTP_USERNAME:-}'
  smtp_auth_password: '${SMTP_PASSWORD:-}'

# Templates
templates:
  - '/etc/alertmanager/templates/*.tmpl'

# Route tree
route:
  receiver: 'default'
  group_by: ['alertname', 'severity', 'category']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

  # Child routes
  routes:
    - match:
        severity: critical
      receiver: 'critical-alerts'
      group_wait: 5s
      repeat_interval: 1h

    - match:
        severity: warning
      receiver: 'warning-alerts'
      repeat_interval: 6h

    - match:
        category: database
      receiver: 'database-alerts'

# Receivers
receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://localhost:5001/alerts'

EOF

  # Add Slack receiver if webhook is configured
  if [[ -n "$slack_webhook" ]]; then
    cat <<EOF
  - name: 'critical-alerts'
    slack_configs:
      - channel: '#alerts-critical'
        title: 'Critical Alert: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
        send_resolved: true

  - name: 'warning-alerts'
    slack_configs:
      - channel: '#alerts-warning'
        title: 'Warning: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
        send_resolved: true

  - name: 'database-alerts'
    slack_configs:
      - channel: '#alerts-database'
        title: 'Database Alert: {{ .GroupLabels.alertname }}'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
        send_resolved: true

EOF
  fi

  # Add email receiver if configured
  if [[ -n "$email_to" ]]; then
    cat <<EOF
  # Email notifications
  - name: 'email-alerts'
    email_configs:
      - to: '${email_to}'
        headers:
          Subject: '[nself Alert] {{ .GroupLabels.alertname }}'
        html: |
          <h2>Alert: {{ .GroupLabels.alertname }}</h2>
          <p><strong>Severity:</strong> {{ .GroupLabels.severity }}</p>
          <p><strong>Category:</strong> {{ .GroupLabels.category }}</p>
          {{ range .Alerts }}
          <p>{{ .Annotations.description }}</p>
          {{ end }}

EOF
  fi

  # Inhibition rules
  cat <<'EOF'

# Inhibition rules
inhibit_rules:
  # Mute warnings if critical alert is firing
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'instance']

  # Mute individual service alerts if entire node is down
  - source_match:
      alertname: 'ServiceDown'
    target_match_re:
      alertname: '.*'
    equal: ['instance']
EOF
}

# Generate alert templates for Slack/Email
generate_alert_templates() {
  cat <<'EOF'
{{ define "slack.default.title" }}
[{{ .Status | toUpper }}{{ if eq .Status "firing" }}:{{ .Alerts.Firing | len }}{{ end }}] {{ .GroupLabels.alertname }}
{{ end }}

{{ define "slack.default.text" }}
{{ range .Alerts }}
*Alert:* {{ .Labels.alertname }}
*Severity:* {{ .Labels.severity }}
*Category:* {{ .Labels.category }}
*Description:* {{ .Annotations.description }}
*Instance:* {{ .Labels.instance }}
{{ end }}
{{ end }}

{{ define "email.default.subject" }}
[nself Alert] {{ .GroupLabels.alertname }} - {{ .GroupLabels.severity }}
{{ end }}

{{ define "email.default.html" }}
<!DOCTYPE html>
<html>
<head>
  <style>
    body { font-family: Arial, sans-serif; }
    .alert { padding: 15px; margin: 10px 0; border-radius: 5px; }
    .critical { background-color: #ffebee; border-left: 4px solid #f44336; }
    .warning { background-color: #fff3e0; border-left: 4px solid #ff9800; }
    .label { font-weight: bold; }
  </style>
</head>
<body>
  <h2>nself Alert Notification</h2>
  {{ range .Alerts }}
  <div class="alert {{ .Labels.severity }}">
    <p><span class="label">Alert:</span> {{ .Labels.alertname }}</p>
    <p><span class="label">Severity:</span> {{ .Labels.severity }}</p>
    <p><span class="label">Category:</span> {{ .Labels.category }}</p>
    <p><span class="label">Description:</span> {{ .Annotations.description }}</p>
    <p><span class="label">Instance:</span> {{ .Labels.instance }}</p>
    <p><span class="label">Time:</span> {{ .StartsAt }}</p>
  </div>
  {{ end }}
</body>
</html>
{{ end }}
EOF
}

# Write alert rules to file
write_alert_rules() {
  local output_dir="${1:-./monitoring/prometheus}"
  mkdir -p "$output_dir"

  generate_prometheus_alerts >"$output_dir/alert-rules.yml"
  echo "Alert rules written to $output_dir/alert-rules.yml"
}

# Write alertmanager config to file
write_alertmanager_config() {
  local output_dir="${1:-./monitoring/alertmanager}"
  mkdir -p "$output_dir"
  mkdir -p "$output_dir/templates"

  generate_alertmanager_config >"$output_dir/alertmanager.yml"
  generate_alert_templates >"$output_dir/templates/default.tmpl"

  echo "Alertmanager config written to $output_dir/alertmanager.yml"
  echo "Alert templates written to $output_dir/templates/default.tmpl"
}

# Export functions
export -f generate_prometheus_alerts generate_alertmanager_config generate_alert_templates
export -f write_alert_rules write_alertmanager_config

# If run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  case "${1:-all}" in
    rules)
      write_alert_rules "${2:-./monitoring/prometheus}"
      ;;
    alertmanager)
      write_alertmanager_config "${2:-./monitoring/alertmanager}"
      ;;
    all)
      write_alert_rules "./monitoring/prometheus"
      write_alertmanager_config "./monitoring/alertmanager"
      ;;
    *)
      echo "Usage: $0 {rules|alertmanager|all} [output_dir]"
      exit 1
      ;;
  esac
fi
