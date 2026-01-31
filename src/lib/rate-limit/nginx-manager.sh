#!/usr/bin/env bash
# nginx-manager.sh - Nginx rate limit management (RATE-011)
# Part of nself - Rate limiting configuration for nginx
#
# Manages rate limit configuration, whitelist/blacklist, and nginx integration

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

if [[ -f "$SCRIPT_DIR/core.sh" ]]; then
  source "$SCRIPT_DIR/core.sh"
fi

# ============================================================================
# Rate Limit Configuration Management
# ============================================================================

# Set rate limit for a specific zone/endpoint
# Usage: nginx_rate_limit_set <zone> <rate>
nginx_rate_limit_set() {
  local zone="$1"
  local rate="$2"

  if [[ -z "$zone" ]] || [[ -z "$rate" ]]; then
    echo "ERROR: Zone and rate required" >&2
    echo "Usage: nginx_rate_limit_set <zone> <rate>" >&2
    echo "Zones: general, graphql_api, auth, uploads, static, webhooks, functions, user_api" >&2
    echo "Rate format: 10r/s (per second) or 100r/m (per minute)" >&2
    return 1
  fi

  # Validate zone
  case "$zone" in
    general|graphql_api|auth|uploads|static|webhooks|functions|user_api)
      ;;
    *)
      echo "ERROR: Invalid zone: $zone" >&2
      echo "Valid zones: general, graphql_api, auth, uploads, static, webhooks, functions, user_api" >&2
      return 1
      ;;
  esac

  # Validate rate format
  if ! echo "$rate" | grep -qE '^[0-9]+r/[sm]$'; then
    echo "ERROR: Invalid rate format: $rate" >&2
    echo "Format: 10r/s (per second) or 100r/m (per minute)" >&2
    return 1
  fi

  # Convert zone name to env var name (portable - Bash 3.2+ compatible)
  local zone_upper=$(echo "$zone" | tr '[:lower:]' '[:upper:]')
  local env_var="RATE_LIMIT_${zone_upper}_RATE"

  # Update .env file
  local env_file="${NSELF_ROOT}/.env"

  if [[ -f "$env_file" ]]; then
    # Check if variable exists
    if grep -q "^${env_var}=" "$env_file"; then
      # Update existing
      if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|^${env_var}=.*|${env_var}=${rate}|" "$env_file"
      else
        sed -i "s|^${env_var}=.*|${env_var}=${rate}|" "$env_file"
      fi
    else
      # Add new
      echo "${env_var}=${rate}" >> "$env_file"
    fi

    printf "✓ Set %s rate limit to %s\n" "$zone" "$rate"
    printf "  Run 'nself build' to apply changes\n"
    return 0
  else
    echo "ERROR: .env file not found at $env_file" >&2
    return 1
  fi
}

# List current rate limit configuration
# Usage: nginx_rate_limit_list
nginx_rate_limit_list() {
  local env_file="${NSELF_ROOT}/.env"

  if [[ ! -f "$env_file" ]]; then
    echo "ERROR: .env file not found" >&2
    return 1
  fi

  # Define zones with their defaults
  declare -a zones=(
    "RATE_LIMIT_GENERAL_RATE:10r/s:General API"
    "RATE_LIMIT_GRAPHQL_RATE:100r/m:GraphQL API"
    "RATE_LIMIT_AUTH_RATE:10r/m:Authentication"
    "RATE_LIMIT_UPLOAD_RATE:5r/m:File Uploads"
    "RATE_LIMIT_STATIC_RATE:1000r/m:Static Assets"
    "RATE_LIMIT_WEBHOOK_RATE:30r/m:Webhooks"
    "RATE_LIMIT_FUNCTIONS_RATE:50r/m:Functions"
    "RATE_LIMIT_USER_RATE:1000r/m:Per-User (Auth)"
  )

  printf "Current Rate Limit Configuration:\n\n"
  printf "%-30s %-15s %-20s\n" "ZONE" "RATE" "DESCRIPTION"
  printf "%-30s %-15s %-20s\n" "----" "----" "-----------"

  for zone_def in "${zones[@]}"; do
    IFS=':' read -r var_name default_rate description <<< "$zone_def"

    # Get value from .env or use default
    local rate
    if grep -q "^${var_name}=" "$env_file"; then
      rate=$(grep "^${var_name}=" "$env_file" | cut -d'=' -f2)
    else
      rate="${default_rate} (default)"
    fi

    printf "%-30s %-15s %-20s\n" "$var_name" "$rate" "$description"
  done

  printf "\nTo modify: nself auth rate-limit set <zone> <rate>\n"
  printf "Example: nself auth rate-limit set graphql_api 200r/m\n"

  return 0
}

# Get rate limit status from nginx
# Usage: nginx_rate_limit_status
nginx_rate_limit_status() {
  # Check if nginx container is running
  local nginx_container
  nginx_container=$(docker ps --filter 'name=nginx' --format '{{.Names}}' 2>/dev/null | head -1)

  if [[ -z "$nginx_container" ]]; then
    echo "ERROR: Nginx container not running" >&2
    echo "Start services with: nself start" >&2
    return 1
  fi

  printf "Rate Limiting Status:\n\n"

  # Check if rate limiting config exists
  if docker exec "$nginx_container" test -f /etc/nginx/includes/rate-limits.conf 2>/dev/null; then
    printf "✓ Rate limiting configuration: ACTIVE\n"

    # Extract zone configurations
    printf "\nConfigured Zones:\n"
    docker exec "$nginx_container" grep -E "limit_req_zone|limit_conn_zone" /etc/nginx/includes/rate-limits.conf 2>/dev/null \
      | grep -v "^#" \
      | sed 's/limit_req_zone/  Request:/' \
      | sed 's/limit_conn_zone/  Connection:/' \
      || echo "  No zones found"

  else
    printf "✗ Rate limiting configuration: NOT FOUND\n"
    printf "  Run 'nself build' to generate configuration\n"
  fi

  # Check for recent rate limit violations in logs
  printf "\nRecent Rate Limit Violations:\n"
  if docker exec "$nginx_container" test -f /var/log/nginx/error.log 2>/dev/null; then
    local violations
    violations=$(docker exec "$nginx_container" grep -c "limiting requests" /var/log/nginx/error.log 2>/dev/null || echo "0")
    printf "  Last 24 hours: %s violations\n" "$violations"

    # Show last 5 violations
    if [[ "$violations" -gt 0 ]]; then
      printf "\n  Last 5 violations:\n"
      docker exec "$nginx_container" grep "limiting requests" /var/log/nginx/error.log 2>/dev/null \
        | tail -5 \
        | sed 's/^/    /' \
        || true
    fi
  else
    printf "  No log file found\n"
  fi

  return 0
}

# Reset rate limits for an IP address
# Usage: nginx_rate_limit_reset_ip <ip_address>
nginx_rate_limit_reset_ip() {
  local ip="$1"

  if [[ -z "$ip" ]]; then
    echo "ERROR: IP address required" >&2
    return 1
  fi

  # Validate IP format
  if ! echo "$ip" | grep -qE '^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$'; then
    echo "ERROR: Invalid IP address format: $ip" >&2
    return 1
  fi

  # Reset in database
  local key="ip:${ip}"
  if rate_limit_reset "$key"; then
    printf "✓ Reset rate limits for IP: %s\n" "$ip"
    return 0
  else
    echo "ERROR: Failed to reset rate limits" >&2
    return 1
  fi
}

# ============================================================================
# Whitelist/Blacklist Management for Nginx
# ============================================================================

# Add IP to whitelist (bypass rate limits)
# Usage: nginx_whitelist_add <ip> [description]
nginx_whitelist_add() {
  local ip="$1"
  local description="${2:-Whitelisted IP}"

  if [[ -z "$ip" ]]; then
    echo "ERROR: IP address required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Escape values
  ip=$(echo "$ip" | sed "s/'/''/g")
  description=$(echo "$description" | sed "s/'/''/g")

  # Create whitelist table if it doesn't exist
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" <<EOSQL >/dev/null 2>&1
CREATE TABLE IF NOT EXISTS rate_limit.whitelist (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ip_address TEXT NOT NULL UNIQUE,
  description TEXT,
  enabled BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_whitelist_ip ON rate_limit.whitelist(ip_address);
CREATE INDEX IF NOT EXISTS idx_whitelist_enabled ON rate_limit.whitelist(enabled);
EOSQL

  # Insert IP
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "INSERT INTO rate_limit.whitelist (ip_address, description)
     VALUES ('$ip', '$description')
     ON CONFLICT (ip_address) DO UPDATE SET
       description = EXCLUDED.description,
       enabled = true,
       created_at = NOW();" \
    >/dev/null 2>&1

  if [[ $? -eq 0 ]]; then
    printf "✓ Added %s to whitelist\n" "$ip"
    printf "  Run 'nself build' to apply changes\n"
    return 0
  else
    echo "ERROR: Failed to add IP to whitelist" >&2
    return 1
  fi
}

# Remove IP from whitelist
# Usage: nginx_whitelist_remove <ip>
nginx_whitelist_remove() {
  local ip="$1"

  if [[ -z "$ip" ]]; then
    echo "ERROR: IP address required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Escape IP
  ip=$(echo "$ip" | sed "s/'/''/g")

  # Delete IP
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "DELETE FROM rate_limit.whitelist WHERE ip_address = '$ip';" \
    >/dev/null 2>&1

  if [[ $? -eq 0 ]]; then
    printf "✓ Removed %s from whitelist\n" "$ip"
    printf "  Run 'nself build' to apply changes\n"
    return 0
  else
    echo "ERROR: Failed to remove IP from whitelist" >&2
    return 1
  fi
}

# List whitelisted IPs
# Usage: nginx_whitelist_list
nginx_whitelist_list() {
  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Query whitelist
  local result
  result=$(docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -t -c \
    "SELECT json_agg(row_to_json(w))
     FROM (
       SELECT ip_address, description, enabled, created_at
       FROM rate_limit.whitelist
       ORDER BY created_at DESC
     ) w;" \
    2>/dev/null | xargs)

  if [[ -z "$result" ]] || [[ "$result" == "null" ]]; then
    echo "No whitelisted IPs"
    return 0
  fi

  echo "$result"
  return 0
}

# Add IP to blacklist (block entirely)
# Usage: nginx_blacklist_add <ip> [reason] [duration_seconds]
nginx_blacklist_add() {
  local ip="$1"
  local reason="${2:-Blocked for abuse}"
  local duration="${3:-}"

  if [[ -z "$ip" ]]; then
    echo "ERROR: IP address required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Escape values
  ip=$(echo "$ip" | sed "s/'/''/g")
  reason=$(echo "$reason" | sed "s/'/''/g")

  # Create blacklist table if it doesn't exist
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" <<EOSQL >/dev/null 2>&1
CREATE TABLE IF NOT EXISTS rate_limit.blacklist (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ip_address TEXT NOT NULL UNIQUE,
  reason TEXT,
  enabled BOOLEAN DEFAULT TRUE,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_blacklist_ip ON rate_limit.blacklist(ip_address);
CREATE INDEX IF NOT EXISTS idx_blacklist_enabled ON rate_limit.blacklist(enabled);
CREATE INDEX IF NOT EXISTS idx_blacklist_expires ON rate_limit.blacklist(expires_at);
EOSQL

  # Calculate expiry time if duration provided
  local expires_sql="NULL"
  if [[ -n "$duration" ]] && [[ "$duration" -gt 0 ]]; then
    expires_sql="NOW() + INTERVAL '$duration seconds'"
  fi

  # Insert IP
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "INSERT INTO rate_limit.blacklist (ip_address, reason, expires_at)
     VALUES ('$ip', '$reason', $expires_sql)
     ON CONFLICT (ip_address) DO UPDATE SET
       reason = EXCLUDED.reason,
       expires_at = EXCLUDED.expires_at,
       enabled = true,
       created_at = NOW();" \
    >/dev/null 2>&1

  if [[ $? -eq 0 ]]; then
    if [[ -n "$duration" ]] && [[ "$duration" -gt 0 ]]; then
      printf "✓ Blocked %s for %s seconds\n" "$ip" "$duration"
    else
      printf "✓ Blocked %s permanently\n" "$ip"
    fi
    printf "  Run 'nself build' to apply changes\n"
    return 0
  else
    echo "ERROR: Failed to add IP to blacklist" >&2
    return 1
  fi
}

# Remove IP from blacklist
# Usage: nginx_blacklist_remove <ip>
nginx_blacklist_remove() {
  local ip="$1"

  if [[ -z "$ip" ]]; then
    echo "ERROR: IP address required" >&2
    return 1
  fi

  # Get PostgreSQL container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)

  if [[ -z "$container" ]]; then
    echo "ERROR: PostgreSQL container not found" >&2
    return 1
  fi

  # Escape IP
  ip=$(echo "$ip" | sed "s/'/''/g")

  # Delete IP
  docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself_db}" -c \
    "DELETE FROM rate_limit.blacklist WHERE ip_address = '$ip';" \
    >/dev/null 2>&1

  if [[ $? -eq 0 ]]; then
    printf "✓ Unblocked %s\n" "$ip"
    printf "  Run 'nself build' to apply changes\n"
    return 0
  else
    echo "ERROR: Failed to remove IP from blacklist" >&2
    return 1
  fi
}

# ============================================================================
# Export functions
# ============================================================================

export -f nginx_rate_limit_set
export -f nginx_rate_limit_list
export -f nginx_rate_limit_status
export -f nginx_rate_limit_reset_ip
export -f nginx_whitelist_add
export -f nginx_whitelist_remove
export -f nginx_whitelist_list
export -f nginx_blacklist_add
export -f nginx_blacklist_remove
