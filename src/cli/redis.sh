#!/usr/bin/env bash
# redis.sh - Redis management CLI
# Part of nself v0.7.0 - Sprint 6

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$SCRIPT_DIR/../lib/redis/core.sh" ]] && source "$SCRIPT_DIR/../lib/redis/core.sh"

cmd_redis() {
  local subcommand="${1:-help}"
  shift || true

  case "$subcommand" in
    init)
      printf "Initializing Redis configuration...\n"
      redis_init && printf "✓ Redis configuration initialized\n"
      ;;

    add)
      local name=""
      local host="$REDIS_DEFAULT_HOST"
      local port="$REDIS_DEFAULT_PORT"
      local database="$REDIS_DEFAULT_DB"
      local password=""

      while [[ $# -gt 0 ]]; do
        case "$1" in
          --name) name="$2"; shift 2 ;;
          --host) host="$2"; shift 2 ;;
          --port) port="$2"; shift 2 ;;
          --db) database="$2"; shift 2 ;;
          --password) password="$2"; shift 2 ;;
          *) printf "Unknown option: %s\n" "$1" >&2; return 1 ;;
        esac
      done

      [[ -z "$name" ]] && { printf "ERROR: --name required\n" >&2; return 1; }

      printf "Adding Redis connection '%s'...\n" "$name"
      local conn_id=$(redis_connection_add "$name" "$host" "$port" "$database" "$password")
      printf "✓ Connection added (ID: %s)\n" "$conn_id"
      ;;

    list)
      printf "Redis connections:\n\n"
      redis_connection_list | jq -r '.[] | "\(.name) - \(.host):\(.port)/\(.database) [\(if .is_active then "active" else "inactive" end)]"' 2>/dev/null || printf "No connections configured\n"
      ;;

    get)
      local name="${1:-}"
      [[ -z "$name" ]] && { printf "ERROR: connection name required\n" >&2; return 1; }
      redis_connection_get "$name" | jq '.'
      ;;

    delete)
      local name="${1:-}"
      [[ -z "$name" ]] && { printf "ERROR: connection name required\n" >&2; return 1; }
      printf "Deleting connection '%s'...\n" "$name"
      redis_connection_delete "$name" && printf "✓ Connection deleted\n"
      ;;

    test)
      local name="${1:-}"
      [[ -z "$name" ]] && { printf "ERROR: connection name required\n" >&2; return 1; }
      printf "Testing connection '%s'...\n" "$name"
      redis_connection_test "$name" | jq '.'
      ;;

    health)
      local name="${1:-}"
      if [[ -n "$name" ]]; then
        redis_health_status "$name" | jq '.'
      else
        printf "Redis health status:\n\n"
        redis_health_status | jq -r '.[] | "\(.connection): \(.status) (\(.response_time_ms)ms) - \(.checked_at)"' 2>/dev/null || printf "No health checks available\n"
      fi
      ;;

    pool)
      local action="${1:-}"
      shift || true

      case "$action" in
        configure)
          local name=""
          local size="$REDIS_DEFAULT_POOL_SIZE"
          local min_idle="2"
          local max_idle="5"
          local timeout="300"

          while [[ $# -gt 0 ]]; do
            case "$1" in
              --name) name="$2"; shift 2 ;;
              --size) size="$2"; shift 2 ;;
              --min-idle) min_idle="$2"; shift 2 ;;
              --max-idle) max_idle="$2"; shift 2 ;;
              --timeout) timeout="$2"; shift 2 ;;
              *) printf "Unknown option: %s\n" "$1" >&2; return 1 ;;
            esac
          done

          [[ -z "$name" ]] && { printf "ERROR: --name required\n" >&2; return 1; }

          printf "Configuring connection pool for '%s'...\n" "$name"
          redis_pool_configure "$name" "$size" "$min_idle" "$max_idle" "$timeout"
          printf "✓ Pool configured\n"
          ;;

        get)
          local name="${1:-}"
          [[ -z "$name" ]] && { printf "ERROR: connection name required\n" >&2; return 1; }
          redis_pool_get "$name" | jq '.'
          ;;

        *)
          printf "ERROR: Unknown pool action: %s\n" "$action" >&2
          printf "Available: configure, get\n"
          return 1
          ;;
      esac
      ;;

    help|--help|-h)
      cat <<'HELP'
nself redis - Redis connection management

COMMANDS:
  init                           Initialize Redis configuration
  add --name NAME [options]      Add Redis connection
  list                           List all connections
  get NAME                       Get connection details
  delete NAME                    Delete connection
  test NAME                      Test connection
  health [NAME]                  Get health status (all or specific)
  pool configure --name NAME     Configure connection pool
  pool get NAME                  Get pool configuration

ADD OPTIONS:
  --name NAME        Connection name (required)
  --host HOST        Redis host (default: localhost)
  --port PORT        Redis port (default: 6379)
  --db DATABASE      Database number (default: 0)
  --password PASS    Redis password

POOL OPTIONS:
  --size SIZE        Pool size (default: 10)
  --min-idle N       Minimum idle connections (default: 2)
  --max-idle N       Maximum idle connections (default: 5)
  --timeout SEC      Idle timeout in seconds (default: 300)

EXAMPLES:
  nself redis init
  nself redis add --name main --host redis.local --port 6379
  nself redis add --name cache --host localhost --db 1
  nself redis test main
  nself redis health
  nself redis pool configure --name main --size 20
  nself redis list
HELP
      ;;

    *)
      printf "ERROR: Unknown command: %s\n" "$subcommand" >&2
      printf "Run 'nself redis help' for usage\n"
      return 1
      ;;
  esac
}

export -f cmd_redis
[[ "${BASH_SOURCE[0]}" == "${0}" ]] && cmd_redis "$@"
