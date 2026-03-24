#!/usr/bin/env bash
# postgres-maintenance.sh - PostgreSQL maintenance operations
# Part of nself SysOps - Bash 3.2 compatible

set -euo pipefail

# Get the directory where this script is located
_PG_MAINT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_PG_MAINT_LIB_ROOT="$(dirname "$_PG_MAINT_DIR")"

# Source dependencies
source "$_PG_MAINT_LIB_ROOT/utils/display.sh" 2>/dev/null || true
source "$_PG_MAINT_LIB_ROOT/utils/platform-compat.sh" 2>/dev/null || true

# Log directory
MAINTENANCE_LOG_DIR="${MAINTENANCE_LOG_DIR:-./logs/maintenance}"

# Run VACUUM ANALYZE on all nSelf-managed databases
postgres_maintenance_vacuum() {
  local db_user="${POSTGRES_USER:-postgres}"
  local verbose="${1:-false}"

  # Find the postgres container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  if [[ -z "$container" ]]; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "PostgreSQL container not found or not running" >&2
    return 1
  fi

  # Set up logging
  mkdir -p "$MAINTENANCE_LOG_DIR"
  local log_file="${MAINTENANCE_LOG_DIR}/vacuum_$(date +%Y%m%d_%H%M%S).log"

  printf "\033[34m%s\033[0m Running VACUUM ANALYZE on all databases...\n" "INFO"

  # Get list of all non-template databases
  local databases
  databases=$(docker exec "$container" psql -U "$db_user" -d postgres -t -A -c \
    "SELECT datname FROM pg_database WHERE datistemplate = false AND datname != 'postgres';" 2>/dev/null)

  if [[ -z "$databases" ]]; then
    printf "\033[33m%s\033[0m No user databases found\n" "WARNING"
    return 0
  fi

  local success_count=0
  local fail_count=0
  local start_time
  start_time=$(date +%s)

  printf "Databases found: %s\n" "$(printf '%s' "$databases" | tr '\n' ', ')" | tee -a "$log_file"
  printf "\n"

  # Run VACUUM ANALYZE on each database
  while IFS= read -r db_name; do
    if [[ -z "$db_name" ]]; then
      continue
    fi

    printf "  Vacuuming: %s ... " "$db_name" | tee -a "$log_file"

    local vacuum_cmd="VACUUM (ANALYZE"
    if [[ "$verbose" == "true" ]]; then
      vacuum_cmd="${vacuum_cmd}, VERBOSE"
    fi
    vacuum_cmd="${vacuum_cmd});"

    if docker exec "$container" psql -U "$db_user" -d "$db_name" -c "$vacuum_cmd" >> "$log_file" 2>&1; then
      printf "\033[32mdone\033[0m\n"
      success_count=$((success_count + 1))
    else
      printf "\033[31mfailed\033[0m\n"
      fail_count=$((fail_count + 1))
    fi
  done <<EOF
$databases
EOF

  local end_time
  end_time=$(date +%s)
  local duration=$((end_time - start_time))

  printf "\n"
  printf "\033[34m%s\033[0m Maintenance complete in %ds: %d succeeded, %d failed\n" \
    "INFO" "$duration" "$success_count" "$fail_count"
  printf "\033[34m%s\033[0m Log: %s\n" "INFO" "$log_file"

  if [[ "$fail_count" -gt 0 ]]; then
    return 1
  fi
  return 0
}

# Run REINDEX on all databases
postgres_maintenance_reindex() {
  local db_user="${POSTGRES_USER:-postgres}"

  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  if [[ -z "$container" ]]; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "PostgreSQL container not found or not running" >&2
    return 1
  fi

  mkdir -p "$MAINTENANCE_LOG_DIR"
  local log_file="${MAINTENANCE_LOG_DIR}/reindex_$(date +%Y%m%d_%H%M%S).log"

  printf "\033[34m%s\033[0m Running REINDEX on all databases...\n" "INFO"

  local databases
  databases=$(docker exec "$container" psql -U "$db_user" -d postgres -t -A -c \
    "SELECT datname FROM pg_database WHERE datistemplate = false AND datname != 'postgres';" 2>/dev/null)

  while IFS= read -r db_name; do
    if [[ -z "$db_name" ]]; then
      continue
    fi
    printf "  Reindexing: %s ... " "$db_name" | tee -a "$log_file"
    if docker exec "$container" psql -U "$db_user" -d "$db_name" -c "REINDEX DATABASE \"$db_name\";" >> "$log_file" 2>&1; then
      printf "\033[32mdone\033[0m\n"
    else
      printf "\033[31mfailed\033[0m\n"
    fi
  done <<EOF
$databases
EOF

  printf "\033[34m%s\033[0m Log: %s\n" "INFO" "$log_file"
}

# Show database statistics (bloat, dead tuples, table sizes)
postgres_maintenance_stats() {
  local db_name="${1:-${POSTGRES_DB:-nself_db}}"
  local db_user="${POSTGRES_USER:-postgres}"

  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  if [[ -z "$container" ]]; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "PostgreSQL container not found or not running" >&2
    return 1
  fi

  printf "\033[34m%s\033[0m Database statistics for '%s':\n\n" "INFO" "$db_name"

  # Table sizes and dead tuples
  docker exec "$container" psql -U "$db_user" -d "$db_name" -c "
    SELECT
      schemaname || '.' || relname AS table_name,
      pg_size_pretty(pg_total_relation_size(relid)) AS total_size,
      n_live_tup AS live_rows,
      n_dead_tup AS dead_rows,
      CASE WHEN n_live_tup > 0
        THEN round(100.0 * n_dead_tup / n_live_tup, 1)
        ELSE 0
      END AS dead_pct,
      last_vacuum::date AS last_vacuum,
      last_analyze::date AS last_analyze
    FROM pg_stat_user_tables
    ORDER BY pg_total_relation_size(relid) DESC
    LIMIT 20;
  " 2>/dev/null || {
    printf "\033[31m%s\033[0m Failed to query database statistics\n" "ERROR" >&2
    return 1
  }
}

# Main dispatch for nself maintenance postgres subcommand
postgres_maintenance() {
  local action="${1:-status}"
  shift 2>/dev/null || true

  case "$action" in
    --run|run|vacuum)
      postgres_maintenance_vacuum "$@"
      ;;
    --reindex|reindex)
      postgres_maintenance_reindex "$@"
      ;;
    --stats|stats)
      postgres_maintenance_stats "$@"
      ;;
    --help|help|-h)
      printf "nself maintenance postgres - PostgreSQL maintenance\n\n"
      printf "USAGE:\n"
      printf "  nself maintenance postgres --run [--verbose]  Run VACUUM ANALYZE\n"
      printf "  nself maintenance postgres --reindex          Reindex all databases\n"
      printf "  nself maintenance postgres --stats [db]       Show table statistics\n"
      ;;
    *)
      printf "\033[33m%s\033[0m Unknown action: %s (use --help)\n" "WARNING" "$action" >&2
      return 1
      ;;
  esac
}

# Export functions
export -f postgres_maintenance 2>/dev/null || true
export -f postgres_maintenance_vacuum 2>/dev/null || true
export -f postgres_maintenance_reindex 2>/dev/null || true
export -f postgres_maintenance_stats 2>/dev/null || true
