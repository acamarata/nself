#!/usr/bin/env bash
# monitoring.sh - Rate limit monitoring (RATE-012)
# Part of nself - Monitoring, alerting, and dashboards for rate limiting
#
# Provides metrics, violation tracking, and alert integration

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -f "$SCRIPT_DIR/core.sh" ]]; then
  source "$SCRIPT_DIR/core.sh"
fi

# ============================================================================
# Rate Limit Violation Tracking
# ============================================================================

# Get recent rate limit violations
# Usage: rate_limit_violations [hours]
rate_limit_violations() {
  local hours="${1:-24}"

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' 2>/dev/null | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Query violations
  local violations
  violations=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(row_to_json(v))
     FROM (
       SELECT
         key,
         COUNT(*) as violation_count,
         MAX(requested_at) as last_violation,
         MIN(requested_at) as first_violation
       FROM rate_limit.log
       WHERE allowed = false
         AND requested_at >= NOW() - INTERVAL '$hours hours'
       GROUP BY key
       ORDER BY violation_count DESC
       LIMIT 100
     ) v;" \
    2>/dev/null | xargs)

  if [[ -z "$violations" ]] || [[ "$violations" == "null" ]]; then
    echo "[]"
    return 0
  fi

  echo "$violations"
  return 0
}

# Get top violators (IPs with most rate limit hits)
# Usage: rate_limit_top_violators [limit] [hours]
rate_limit_top_violators() {
  local limit="${1:-10}"
  local hours="${2:-24}"

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' 2>/dev/null | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Query top violators
  local violators
  violators=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(row_to_json(v))
     FROM (
       SELECT
         SPLIT_PART(key, ':', 2) as ip_address,
         COUNT(*) as total_requests,
         SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) as violations,
         ROUND(100.0 * SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) / COUNT(*), 2) as violation_rate,
         MAX(requested_at) as last_seen
       FROM rate_limit.log
       WHERE key LIKE 'ip:%'
         AND requested_at >= NOW() - INTERVAL '$hours hours'
       GROUP BY SPLIT_PART(key, ':', 2)
       HAVING SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) > 0
       ORDER BY violations DESC
       LIMIT $limit
     ) v;" \
    2>/dev/null | xargs)

  if [[ -z "$violators" ]] || [[ "$violators" == "null" ]]; then
    echo "[]"
    return 0
  fi

  echo "$violators"
  return 0
}

# Get rate limit metrics for Prometheus
# Usage: rate_limit_metrics
rate_limit_metrics() {
  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' 2>/dev/null | head -1)

  if [[ -z "$container" ]]; then
    echo "# ERROR: PostgreSQL container not found"
    return 1
  fi

  # Generate Prometheus-compatible metrics
  cat <<EOF
# HELP nself_rate_limit_requests_total Total number of requests tracked by rate limiter
# TYPE nself_rate_limit_requests_total counter
EOF

  # Total requests
  local total_requests
  total_requests=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT COUNT(*) FROM rate_limit.log WHERE requested_at >= NOW() - INTERVAL '1 hour';" \
    2>/dev/null | xargs)

  echo "nself_rate_limit_requests_total ${total_requests:-0}"

  # Violations
  cat <<EOF

# HELP nself_rate_limit_violations_total Total number of rate limit violations
# TYPE nself_rate_limit_violations_total counter
EOF

  local violations
  violations=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT COUNT(*) FROM rate_limit.log WHERE allowed = false AND requested_at >= NOW() - INTERVAL '1 hour';" \
    2>/dev/null | xargs)

  echo "nself_rate_limit_violations_total ${violations:-0}"

  # Violation rate
  cat <<EOF

# HELP nself_rate_limit_violation_rate Rate of violations (0-1)
# TYPE nself_rate_limit_violation_rate gauge
EOF

  local violation_rate
  if [[ "${total_requests:-0}" -gt 0 ]]; then
    violation_rate=$(echo "scale=4; ${violations:-0} / ${total_requests:-1}" | bc)
  else
    violation_rate="0"
  fi

  echo "nself_rate_limit_violation_rate ${violation_rate}"

  # Active buckets
  cat <<EOF

# HELP nself_rate_limit_active_buckets Number of active rate limit buckets
# TYPE nself_rate_limit_active_buckets gauge
EOF

  local active_buckets
  active_buckets=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT COUNT(*) FROM rate_limit.buckets;" \
    2>/dev/null | xargs)

  echo "nself_rate_limit_active_buckets ${active_buckets:-0}"

  return 0
}

# ============================================================================
# Alerting
# ============================================================================

# Check for suspicious patterns and generate alerts
# Usage: rate_limit_check_alerts
rate_limit_check_alerts() {
  local alerts=()

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' 2>/dev/null | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Check for IPs with high violation rates
  local high_violators
  high_violators=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT COUNT(*)
     FROM (
       SELECT
         SPLIT_PART(key, ':', 2) as ip,
         SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) as violations
       FROM rate_limit.log
       WHERE key LIKE 'ip:%'
         AND requested_at >= NOW() - INTERVAL '1 hour'
       GROUP BY SPLIT_PART(key, ':', 2)
       HAVING SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) > 100
     ) v;" \
    2>/dev/null | xargs)

  if [[ "${high_violators:-0}" -gt 0 ]]; then
    alerts+=("HIGH: ${high_violators} IPs with >100 violations in last hour")
  fi

  # Check for sudden spikes in violations
  local current_hour_violations
  current_hour_violations=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT COUNT(*) FROM rate_limit.log WHERE allowed = false AND requested_at >= NOW() - INTERVAL '1 hour';" \
    2>/dev/null | xargs)

  local previous_hour_violations
  previous_hour_violations=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT COUNT(*) FROM rate_limit.log WHERE allowed = false AND requested_at >= NOW() - INTERVAL '2 hours' AND requested_at < NOW() - INTERVAL '1 hour';" \
    2>/dev/null | xargs)

  # Alert if current hour is 5x previous hour
  if [[ "${previous_hour_violations:-0}" -gt 0 ]]; then
    local ratio=$((current_hour_violations / previous_hour_violations))
    if [[ "$ratio" -ge 5 ]]; then
      alerts+=("WARNING: 5x increase in violations (${current_hour_violations} vs ${previous_hour_violations})")
    fi
  fi

  # Check for potential DDoS (very high request rate)
  local recent_requests
  recent_requests=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT COUNT(*) FROM rate_limit.log WHERE requested_at >= NOW() - INTERVAL '5 minutes';" \
    2>/dev/null | xargs)

  if [[ "${recent_requests:-0}" -gt 10000 ]]; then
    alerts+=("CRITICAL: Potential DDoS - ${recent_requests} requests in last 5 minutes")
  fi

  # Output alerts
  if [[ ${#alerts[@]} -eq 0 ]]; then
    printf "No alerts\n"
    return 0
  else
    printf "ALERTS DETECTED:\n"
    for alert in "${alerts[@]}"; do
      printf "  • %s\n" "$alert"
    done
    return 1
  fi
}

# Generate Grafana dashboard JSON for rate limiting
# Usage: rate_limit_grafana_dashboard
rate_limit_grafana_dashboard() {
  cat <<'EOF'
{
  "dashboard": {
    "title": "nself Rate Limiting",
    "tags": ["nself", "security", "rate-limiting"],
    "timezone": "browser",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(nself_rate_limit_requests_total[5m])",
            "legendFormat": "Requests/sec"
          }
        ]
      },
      {
        "title": "Violation Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "nself_rate_limit_violation_rate",
            "legendFormat": "Violation Rate"
          }
        ]
      },
      {
        "title": "Top Violating IPs",
        "type": "table",
        "targets": [
          {
            "expr": "topk(10, nself_rate_limit_violations_by_ip)",
            "format": "table"
          }
        ]
      },
      {
        "title": "Active Rate Limit Buckets",
        "type": "stat",
        "targets": [
          {
            "expr": "nself_rate_limit_active_buckets"
          }
        ]
      }
    ]
  }
}
EOF
}

# ============================================================================
# Log Analysis
# ============================================================================

# Analyze nginx error logs for rate limit violations
# Usage: rate_limit_analyze_nginx_logs
rate_limit_analyze_nginx_logs() {
  local nginx_container
  nginx_container=$(docker ps --filter 'name=nginx' --format '{{.Names}}' 2>/dev/null | head -1)

  if [[ -z "$nginx_container" ]]; then
    echo "ERROR: Nginx container not running" >&2
    return 1
  fi

  printf "Analyzing nginx error logs for rate limiting...\n\n"

  # Extract rate limit violations
  printf "Rate Limit Violations:\n"
  docker exec "$nginx_container" grep "limiting requests" /var/log/nginx/error.log 2>/dev/null \
    | tail -20 \
    | sed 's/^/  /' \
    || echo "  No violations found"

  printf "\n"

  # Extract IPs being limited
  printf "Top Rate Limited IPs:\n"
  docker exec "$nginx_container" grep "limiting requests" /var/log/nginx/error.log 2>/dev/null \
    | grep -oE "client: [0-9.]+" \
    | cut -d' ' -f2 \
    | sort \
    | uniq -c \
    | sort -rn \
    | head -10 \
    | awk '{printf "  %5d  %s\n", $1, $2}' \
    || echo "  No IPs found"

  return 0
}

# ============================================================================
# Export functions
# ============================================================================

export -f rate_limit_violations
export -f rate_limit_top_violators
export -f rate_limit_metrics
export -f rate_limit_check_alerts
export -f rate_limit_grafana_dashboard
export -f rate_limit_analyze_nginx_logs
