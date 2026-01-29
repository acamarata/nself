#!/usr/bin/env bash
# api.sh - Admin API endpoints
# Part of nself v0.7.0

set -euo pipefail

admin_stats_overview() {
  local container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  
  local stats=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT row_to_json(s) FROM (
       SELECT
         (SELECT COUNT(*) FROM auth.users WHERE deleted_at IS NULL) AS total_users,
         (SELECT COUNT(*) FROM auth.sessions WHERE expires_at > NOW()) AS active_sessions,
         (SELECT COUNT(*) FROM auth.roles WHERE is_system = FALSE) AS custom_roles,
         (SELECT COUNT(*) FROM secrets.vault WHERE is_active = TRUE) AS total_secrets,
         (SELECT COUNT(*) FROM webhooks.endpoints WHERE enabled = TRUE) AS active_webhooks,
         (SELECT COUNT(*) FROM rate_limit.log WHERE requested_at >= NOW() - INTERVAL '1 hour') AS requests_last_hour
     ) s;" 2>/dev/null | xargs)
  
  echo "$stats" | jq '.'
}

admin_users_list() {
  local limit="${1:-50}"
  local offset="${2:-0}"
  local container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  
  local users=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(u) FROM (
       SELECT id, email, created_at, last_sign_in_at, email_verified,
              (SELECT json_agg(r.name) FROM auth.user_roles ur
               JOIN auth.roles r ON ur.role_id = r.id WHERE ur.user_id = users.id) AS roles
       FROM auth.users WHERE deleted_at IS NULL
       ORDER BY created_at DESC LIMIT $limit OFFSET $offset
     ) u;" 2>/dev/null | xargs)
  
  [[ -z "$users" || "$users" == "null" ]] && echo "[]" || echo "$users" | jq '.'
}

admin_activity_recent() {
  local hours="${1:-24}"
  local container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  
  local activity=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(a) FROM (
       SELECT event_type, action, result, created_at
       FROM audit.events
       WHERE created_at >= NOW() - INTERVAL '$hours hours'
       ORDER BY created_at DESC LIMIT 100
     ) a;" 2>/dev/null | xargs)
  
  [[ -z "$activity" || "$activity" == "null" ]] && echo "[]" || echo "$activity" | jq '.'
}

admin_security_events() {
  local container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  
  local events=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(e) FROM (
       SELECT 'rate_limit' AS type, key, COUNT(*) AS count
       FROM rate_limit.log
       WHERE allowed = FALSE AND requested_at >= NOW() - INTERVAL '1 hour'
       GROUP BY key
       ORDER BY count DESC LIMIT 10
     ) e;" 2>/dev/null | xargs)
  
  [[ -z "$events" || "$events" == "null" ]] && echo "[]" || echo "$events" | jq '.'
}

export -f admin_stats_overview admin_users_list admin_activity_recent admin_security_events
