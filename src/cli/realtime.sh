#!/usr/bin/env bash
#
# nself realtime - Real-time communication management
#
# Manages WebSocket server, channels, presence, broadcast, and database subscriptions
#

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source libraries
source "$SCRIPT_DIR/../lib/utils/output.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"

# Source config if available (optional for testing)
[[ -f "$SCRIPT_DIR/../lib/config/env.sh" ]] && source "$SCRIPT_DIR/../lib/config/env.sh" || true

# Source realtime modules
source "$SCRIPT_DIR/../lib/realtime/channels.sh"
source "$SCRIPT_DIR/../lib/realtime/presence.sh"
source "$SCRIPT_DIR/../lib/realtime/broadcast.sh"
source "$SCRIPT_DIR/../lib/realtime/subscriptions.sh"

# ============================================================================
# Usage
# ============================================================================

usage() {
    cat << EOF
Usage: nself realtime <command> [options]

Real-time communication management (Supabase/Nhost compatible)

COMMANDS:
  # System Management
  init                           Initialize real-time system
  status                         Show real-time system status
  logs [--follow]                Show real-time logs
  cleanup                        Clean up stale connections and old messages

  # Database Subscriptions (CDC)
  subscribe <table> [events]     Subscribe to table changes (INSERT,UPDATE,DELETE)
  unsubscribe <table>            Unsubscribe from table changes
  listen <table> [seconds]       Listen to table changes in real-time
  subscriptions                  List all active subscriptions

  # Channel Management
  channel create <name> [type]   Create a channel (type: public|private|presence)
  channel list [type]            List channels (filter: all|public|private|presence)
  channel get <id>               Get channel details
  channel delete <id>            Delete a channel
  channel members <id>           List channel members
  channel join <channel> <user>  Add user to channel
  channel leave <channel> <user> Remove user from channel

  # Broadcast Messages
  broadcast <channel> <event> <payload>  Send message to channel
  messages <channel> [limit]             Get recent messages
  replay <channel> <timestamp>           Replay messages since timestamp
  events <channel> [hours]               List event types

  # Presence Tracking
  presence track <user> <channel> [status]  Track user presence (online|away|offline)
  presence get <user> [channel]             Get user presence
  presence online [channel]                 List online users
  presence count [channel]                  Count online users
  presence offline <user> [channel]         Set user offline
  presence stats                            Get presence statistics

  # Connection Management
  connections [--json]           Show active WebSocket connections
  stats                          Show detailed real-time statistics

OPTIONS:
  -h, --help                     Show this help message
  --json                         Output in JSON format
  --format <fmt>                 Output format (table, json, csv)

EXAMPLES:
  # Initialize real-time system
  nself realtime init

  # Subscribe to database table changes
  nself realtime subscribe public.users INSERT,UPDATE,DELETE
  nself realtime listen public.users

  # Create a channel
  nself realtime channel create "general" public
  nself realtime channel create "support" private

  # Broadcast message
  nself realtime broadcast general user.joined '{"user_id": "123", "name": "John"}'

  # Track presence
  nself realtime presence track user-123 general online
  nself realtime presence online general

  # Monitor system
  nself realtime connections
  nself realtime stats
  nself realtime logs --follow

For more information:
  https://docs.nself.org/realtime

EOF
}

# ============================================================================
# System Commands
# ============================================================================

cmd_init() {
    info "Initializing real-time system..."
    printf "\n"

    # Check if PostgreSQL is running
    if ! docker_container_running "postgres"; then
        error "PostgreSQL is not running"
        printf "\nStart PostgreSQL with: nself start\n\n"
        return 1
    fi

    # Create migration file if it doesn't exist
    local migration_dir="$ROOT_DIR/postgres/migrations"
    mkdir -p "$migration_dir"

    local migration_file="$migration_dir/012_create_realtime_system.sql"

    if [[ ! -f "$migration_file" ]]; then
        info "Creating realtime migration..."
        cat > "$migration_file" << 'EOSQL'
-- Real-time System Migration
-- Creates schema, tables, and functions for real-time features

-- Create schema
CREATE SCHEMA IF NOT EXISTS realtime;

-- Channels table
CREATE TABLE IF NOT EXISTS realtime.channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'public' CHECK (type IN ('public', 'private', 'presence')),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_channels_type ON realtime.channels(type);
CREATE INDEX IF NOT EXISTS idx_channels_slug ON realtime.channels(slug);

-- Channel members
CREATE TABLE IF NOT EXISTS realtime.channel_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES realtime.channels(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    role TEXT DEFAULT 'member' CHECK (role IN ('member', 'moderator', 'admin')),
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(channel_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_channel_members_channel ON realtime.channel_members(channel_id);
CREATE INDEX IF NOT EXISTS idx_channel_members_user ON realtime.channel_members(user_id);

-- Messages (broadcast)
CREATE TABLE IF NOT EXISTS realtime.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES realtime.channels(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    sender_id TEXT,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_channel ON realtime.messages(channel_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_event ON realtime.messages(event_type);
CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON realtime.messages(sent_at);

-- Presence tracking
CREATE TABLE IF NOT EXISTS realtime.presence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    channel_id UUID REFERENCES realtime.channels(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'online' CHECK (status IN ('online', 'away', 'offline')),
    metadata JSONB DEFAULT '{}'::jsonb,
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, COALESCE(channel_id, '00000000-0000-0000-0000-000000000000'::uuid))
);

CREATE INDEX IF NOT EXISTS idx_presence_user ON realtime.presence(user_id);
CREATE INDEX IF NOT EXISTS idx_presence_channel ON realtime.presence(channel_id);
CREATE INDEX IF NOT EXISTS idx_presence_status ON realtime.presence(status);

-- Subscriptions (table CDC)
CREATE TABLE IF NOT EXISTS realtime.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_name TEXT NOT NULL UNIQUE,
    events TEXT[] NOT NULL DEFAULT ARRAY['INSERT', 'UPDATE', 'DELETE'],
    filter TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Connections tracking
CREATE TABLE IF NOT EXISTS realtime.connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT,
    connection_id TEXT NOT NULL UNIQUE,
    status TEXT DEFAULT 'connected' CHECK (status IN ('connected', 'disconnected')),
    connected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_connections_user ON realtime.connections(user_id);
CREATE INDEX IF NOT EXISTS idx_connections_status ON realtime.connections(status);

-- Cleanup function for stale connections
CREATE OR REPLACE FUNCTION realtime.cleanup_stale_connections(timeout_seconds INT DEFAULT 300)
RETURNS INT AS $$
DECLARE
    cleaned INT;
BEGIN
    UPDATE realtime.connections
    SET status = 'disconnected'
    WHERE status = 'connected'
      AND EXTRACT(EPOCH FROM (NOW() - last_seen_at)) > timeout_seconds;

    GET DIAGNOSTICS cleaned = ROW_COUNT;
    RETURN cleaned;
END;
$$ LANGUAGE plpgsql;

-- Cleanup function for old broadcasts
CREATE OR REPLACE FUNCTION realtime.cleanup_expired_broadcasts(retention_hours INT DEFAULT 24)
RETURNS INT AS $$
DECLARE
    cleaned INT;
BEGIN
    DELETE FROM realtime.messages
    WHERE sent_at < NOW() - (retention_hours || ' hours')::INTERVAL;

    GET DIAGNOSTICS cleaned = ROW_COUNT;
    RETURN cleaned;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT USAGE ON SCHEMA realtime TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA realtime TO PUBLIC;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA realtime TO PUBLIC;

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Real-time system initialized successfully';
END $$;
EOSQL
        success "Migration file created"
    fi

    # Run migration
    info "Running migration..."
    if docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" < "$migration_file" >/dev/null 2>&1; then
        success "Real-time system initialized successfully"
    else
        error "Failed to initialize real-time system"
        return 1
    fi

    printf "\n"
    success "Real-time system is ready!"
    printf "\nNext steps:\n"
    printf "  1. Subscribe to table changes: nself realtime subscribe public.users\n"
    printf "  2. Create a channel: nself realtime channel create general\n"
    printf "  3. Track presence: nself realtime presence track <user-id> <channel>\n"
    printf "  4. Broadcast message: nself realtime broadcast <channel> <event> <payload>\n"
    printf "\n"
}

cmd_status() {
    info "Real-time System Status"
    printf "\n"

    # Check if realtime schema exists
    local schema_exists
    schema_exists=$(docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c \
        "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'realtime');" | tr -d ' \n')

    if [[ "$schema_exists" != "t" ]]; then
        error "Real-time system not initialized"
        printf "\nRun: nself realtime init\n\n"
        return 1
    fi

    success "Real-time system is initialized"
    printf "\n"

    # Get statistics
    docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
        SELECT
            (SELECT COUNT(*) FROM realtime.channels) as channels,
            (SELECT COUNT(*) FROM realtime.messages WHERE sent_at > NOW() - INTERVAL '24 hours') as messages_24h,
            (SELECT COUNT(*) FROM realtime.presence WHERE status != 'offline') as online_users,
            (SELECT COUNT(*) FROM realtime.subscriptions) as table_subscriptions;
        "
}

cmd_logs() {
    local follow=false

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --follow|-f)
                follow=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done

    info "Real-time logs"
    printf "\n"

    # Show recent messages
    docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "
        SELECT
            m.sent_at,
            c.slug as channel,
            m.event_type,
            m.payload
        FROM realtime.messages m
        JOIN realtime.channels c ON m.channel_id = c.id
        ORDER BY m.sent_at DESC
        LIMIT 50;
        "

    if [[ "$follow" == "true" ]]; then
        info "Following logs (Ctrl+C to stop)..."
        # This would need a proper implementation with LISTEN
        warn "Follow mode not yet implemented"
    fi
}

cmd_cleanup() {
    info "Cleaning up real-time system..."
    printf "\n"

    local sql="
    SELECT
        realtime.cleanup_stale_connections() as stale_connections,
        realtime.cleanup_expired_broadcasts() as expired_broadcasts;
    "

    docker exec -i "$(docker_get_container_name postgres)" \
        psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$sql"

    success "Cleanup complete"
}

# ============================================================================
# Command Router
# ============================================================================

main() {
    if [[ $# -eq 0 ]]; then
        usage
        exit 1
    fi

    local command="$1"
    shift

    case "$command" in
        # System commands
        init)
            cmd_init "$@"
            ;;
        status)
            cmd_status "$@"
            ;;
        logs)
            cmd_logs "$@"
            ;;
        cleanup)
            cmd_cleanup "$@"
            ;;

        # Database subscriptions
        subscribe)
            subscribe_table "$@"
            ;;
        unsubscribe)
            unsubscribe_table "$@"
            ;;
        listen)
            listen_table "$@"
            ;;
        subscriptions)
            list_subscriptions "$@"
            ;;

        # Channel commands
        channel)
            local subcmd="${1:-}"
            shift || true
            case "$subcmd" in
                create)
                    channel_create "$@"
                    ;;
                list)
                    channel_list "table" "$@"
                    ;;
                get)
                    channel_get "$@"
                    ;;
                delete)
                    channel_delete "$@"
                    ;;
                members)
                    channel_list_members "$@"
                    ;;
                join)
                    channel_add_member "$@"
                    ;;
                leave)
                    channel_remove_member "$@"
                    ;;
                *)
                    error "Unknown channel subcommand: $subcmd"
                    usage
                    exit 1
                    ;;
            esac
            ;;

        # Broadcast commands
        broadcast)
            broadcast_send "$@"
            ;;
        messages)
            broadcast_get_messages "$@"
            ;;
        replay)
            broadcast_replay "$@"
            ;;
        events)
            broadcast_list_events "$@"
            ;;

        # Presence commands
        presence)
            local subcmd="${1:-}"
            shift || true
            case "$subcmd" in
                track)
                    presence_track "$@"
                    ;;
                get)
                    presence_get "$@"
                    ;;
                online)
                    presence_list_online "$@"
                    ;;
                count)
                    presence_count_online "$@"
                    ;;
                offline)
                    presence_set_offline "$@"
                    ;;
                stats)
                    presence_stats "$@"
                    ;;
                cleanup)
                    presence_cleanup "$@"
                    ;;
                *)
                    error "Unknown presence subcommand: $subcmd"
                    usage
                    exit 1
                    ;;
            esac
            ;;

        # Connection management
        connections)
            local sql="
            SELECT
                c.user_id,
                c.connection_id,
                c.status,
                c.connected_at,
                c.last_seen_at,
                EXTRACT(EPOCH FROM (NOW() - c.last_seen_at))::int as seconds_since_seen
            FROM realtime.connections c
            WHERE c.status = 'connected'
            ORDER BY c.connected_at DESC;
            "
            docker exec -i "$(docker_get_container_name postgres)" \
                psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "$sql"
            ;;

        stats)
            cmd_status
            printf "\n"
            info "Subscription Statistics"
            subscription_stats
            printf "\n"
            info "Presence Statistics"
            presence_stats
            printf "\n"
            info "Broadcast Statistics"
            broadcast_stats
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

# Run main
main "$@"
