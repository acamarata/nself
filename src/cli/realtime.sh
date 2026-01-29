#!/usr/bin/env bash
#
# nself realtime - Real-time communication management
#
# Manages WebSocket server, channels, and real-time features
#

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source libraries
source "$SCRIPT_DIR/../lib/utils/output.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/config/env.sh"

# ============================================================================
# Usage
# ============================================================================

usage() {
    cat << EOF
Usage: nself realtime <command> [options]

Real-time communication management

COMMANDS:
  init                    Initialize real-time system
  start                   Start WebSocket server
  stop                    Stop WebSocket server
  restart                 Restart WebSocket server
  status                  Show real-time server status
  logs                    Show WebSocket server logs

  # Channel management
  channel create <name>   Create a channel
  channel list            List all channels
  channel delete <id>     Delete a channel

  # Monitoring
  connections             Show active connections
  stats                   Show real-time statistics
  cleanup                 Clean up stale connections and broadcasts

OPTIONS:
  -h, --help              Show this help message
  --json                  Output in JSON format
  --port <port>           WebSocket server port (default: 3100)

EXAMPLES:
  # Initialize real-time system
  nself realtime init

  # Start WebSocket server
  nself realtime start

  # Create channel
  nself realtime channel create "general"

  # Monitor connections
  nself realtime connections
  nself realtime stats

EOF
}

# ============================================================================
# Main Command Router
# ============================================================================

main() {
    if [[ $# -eq 0 ]]; then
        usage
        exit 1
    fi

    local command="$1"
    shift

    case "$command" in
        init)
            realtime_init "$@"
            ;;
        start)
            realtime_start "$@"
            ;;
        stop)
            realtime_stop "$@"
            ;;
        restart)
            realtime_restart "$@"
            ;;
        status)
            realtime_status "$@"
            ;;
        logs)
            realtime_logs "$@"
            ;;
        channel)
            realtime_channel_cmd "$@"
            ;;
        connections)
            realtime_connections "$@"
            ;;
        stats)
            realtime_stats "$@"
            ;;
        cleanup)
            realtime_cleanup "$@"
            ;;
        -h|--help|help)
            usage
            exit 0
            ;;
        *)
            error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

# ============================================================================
# Commands
# ============================================================================

realtime_init() {
    info "Initializing real-time system..."

    if ! docker_container_running "postgres"; then
        error "PostgreSQL is not running. Start it with: nself start"
        return 1
    fi

    local migration_file="$ROOT_DIR/postgres/migrations/012_create_realtime_system.sql"

    if [[ ! -f "$migration_file" ]]; then
        error "Migration file not found: $migration_file"
        return 1
    fi

    if docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" < "$migration_file" >/dev/null 2>&1; then
        success "Real-time system initialized"
    else
        error "Failed to initialize real-time system"
        return 1
    fi

    # Create WebSocket server service if it doesn't exist
    local ws_service_dir="$ROOT_DIR/services/websocket-server"
    if [[ ! -d "$ws_service_dir" ]]; then
        info "Creating WebSocket server service..."
        mkdir -p "$ws_service_dir"

        # Copy template files
        cp "$ROOT_DIR/src/templates/services/websocket-server/package.json.template" "$ws_service_dir/package.json"
        cp "$ROOT_DIR/src/templates/services/websocket-server/server.js.template" "$ws_service_dir/server.js"
        cp "$ROOT_DIR/src/templates/services/websocket-server/Dockerfile.template" "$ws_service_dir/Dockerfile"

        # Replace placeholders
        sed -i.bak 's/{{SERVICE_NAME}}/websocket-server/g' "$ws_service_dir"/* 2>/dev/null || true
        sed -i.bak 's/{{SERVICE_NAME_UPPER}}/WEBSOCKET_SERVER/g' "$ws_service_dir"/* 2>/dev/null || true
        sed -i.bak 's/{{PORT}}/3100/g' "$ws_service_dir"/* 2>/dev/null || true
        rm -f "$ws_service_dir"/*.bak

        success "WebSocket server service created"
    fi

    success "Real-time initialization complete"
}

realtime_start() {
    info "Starting WebSocket server..."

    if docker_container_running "websocket-server"; then
        warn "WebSocket server is already running"
        return 0
    fi

    # Start via docker-compose (assumes it's in the compose file)
    docker-compose up -d websocket-server

    success "WebSocket server started"
}

realtime_stop() {
    info "Stopping WebSocket server..."

    docker-compose stop websocket-server

    success "WebSocket server stopped"
}

realtime_restart() {
    realtime_stop
    sleep 2
    realtime_start
}

realtime_status() {
    if docker_container_running "websocket-server"; then
        success "WebSocket server is running"

        # Get connection count
        local connections
        connections=$(docker exec -i "$(docker_get_container_name postgres)" \
            psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c \
            "SELECT COUNT(*) FROM realtime.connections WHERE status = 'connected'" | tr -d ' \n')

        printf "  Active connections: %s\n" "$connections"
    else
        warn "WebSocket server is not running"
    fi
}

realtime_logs() {
    docker-compose logs -f websocket-server
}

realtime_connections() {
    info "Active WebSocket connections"
    printf "\n"

    local sql="
    SELECT
        c.user_id,
        c.connected_at,
        c.last_seen_at,
        COUNT(s.channel_id) as subscribed_channels
    FROM realtime.connections c
    LEFT JOIN realtime.subscriptions s ON c.id = s.connection_id
    WHERE c.status = 'connected'
    GROUP BY c.id
    ORDER BY c.connected_at DESC;
    "

    docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$sql"
}

realtime_stats() {
    info "Real-time Statistics"
    printf "\n"

    local sql="
    SELECT
        (SELECT COUNT(*) FROM realtime.connections WHERE status = 'connected') as active_connections,
        (SELECT COUNT(*) FROM realtime.channels) as total_channels,
        (SELECT COUNT(*) FROM realtime.messages WHERE sent_at > NOW() - INTERVAL '24 hours') as messages_24h,
        (SELECT COUNT(DISTINCT user_id) FROM realtime.presence WHERE status != 'offline') as online_users;
    "

    docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$sql"
}

realtime_cleanup() {
    info "Cleaning up stale connections and broadcasts..."

    local sql="
    SELECT
        realtime.cleanup_stale_connections() as stale_connections,
        realtime.cleanup_expired_broadcasts() as expired_broadcasts;
    "

    docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$sql"

    success "Cleanup complete"
}

realtime_channel_cmd() {
    if [[ $# -eq 0 ]]; then
        error "channel command requires a subcommand"
        usage
        exit 1
    fi

    local subcmd="$1"
    shift

    case "$subcmd" in
        create)
            realtime_channel_create "$@"
            ;;
        list)
            realtime_channel_list "$@"
            ;;
        delete)
            realtime_channel_delete "$@"
            ;;
        *)
            error "Unknown channel subcommand: $subcmd"
            exit 1
            ;;
    esac
}

realtime_channel_create() {
    local name="$1"
    local type="${2:-public}"

    if [[ -z "$name" ]]; then
        error "Channel name required"
        return 1
    fi

    local slug
    slug=$(printf "%s" "$name" | tr '[:upper:]' '[:lower:]' | tr ' ' '-' | sed 's/[^a-z0-9-]//g')

    info "Creating channel: $name"

    local sql="
    INSERT INTO realtime.channels (name, slug, type)
    VALUES ('$name', '$slug', '$type')
    RETURNING id;
    "

    local channel_id
    channel_id=$(docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "$sql" | tr -d ' \n')

    success "Channel created: $slug (ID: $channel_id)"
}

realtime_channel_list() {
    local sql="
    SELECT
        c.slug,
        c.name,
        c.type,
        COUNT(DISTINCT cm.user_id) as members,
        COUNT(DISTINCT p.user_id) as online,
        c.created_at
    FROM realtime.channels c
    LEFT JOIN realtime.channel_members cm ON c.id = cm.channel_id
    LEFT JOIN realtime.presence p ON c.id = p.channel_id AND p.status != 'offline'
    GROUP BY c.id
    ORDER BY c.created_at DESC;
    "

    docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$sql"
}

realtime_channel_delete() {
    local channel_id="$1"

    if [[ -z "$channel_id" ]]; then
        error "Channel ID required"
        return 1
    fi

    info "Deleting channel: $channel_id"

    local sql="DELETE FROM realtime.channels WHERE id = '$channel_id' OR slug = '$channel_id';"

    docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$sql" >/dev/null 2>&1

    success "Channel deleted"
}

# Run main
main "$@"
