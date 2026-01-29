-- Migration 012: Real-Time Collaboration System
-- WebSocket infrastructure, channels, presence, and messaging

BEGIN;

-- ============================================================================
-- SCHEMA: realtime
-- Real-time communication and collaboration
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS realtime;

-- WebSocket connections (active connections)
CREATE TABLE realtime.connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE,

    -- Connection details
    connection_id TEXT UNIQUE NOT NULL, -- WebSocket connection ID
    socket_id TEXT NOT NULL, -- Socket.io socket ID or similar

    -- Client info
    client_ip TEXT,
    user_agent TEXT,

    -- Status
    status TEXT NOT NULL DEFAULT 'connected' CHECK (status IN ('connected', 'disconnected', 'idle')),

    -- Timestamps
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ
);

CREATE INDEX idx_connections_user ON realtime.connections(user_id);
CREATE INDEX idx_connections_tenant ON realtime.connections(tenant_id);
CREATE INDEX idx_connections_status ON realtime.connections(status);
CREATE INDEX idx_connections_connection_id ON realtime.connections(connection_id);

-- Channels (rooms for group communication)
CREATE TABLE realtime.channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE,

    -- Channel details
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,

    -- Channel type
    type TEXT NOT NULL DEFAULT 'public' CHECK (type IN ('public', 'private', 'presence', 'direct')),

    -- Settings
    max_members INTEGER DEFAULT 100,
    is_persistent BOOLEAN DEFAULT true, -- Store message history

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,

    UNIQUE (tenant_id, slug)
);

CREATE INDEX idx_channels_tenant ON realtime.channels(tenant_id);
CREATE INDEX idx_channels_slug ON realtime.channels(slug);
CREATE INDEX idx_channels_type ON realtime.channels(type);

-- Channel members (users subscribed to channels)
CREATE TABLE realtime.channel_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES realtime.channels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,

    -- Role in channel
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),

    -- Subscription
    subscribed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_read_at TIMESTAMPTZ,

    -- Permissions
    can_send BOOLEAN DEFAULT true,
    can_invite BOOLEAN DEFAULT false,

    UNIQUE (channel_id, user_id)
);

CREATE INDEX idx_channel_members_channel ON realtime.channel_members(channel_id);
CREATE INDEX idx_channel_members_user ON realtime.channel_members(user_id);

-- Messages (channel messages)
CREATE TABLE realtime.messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES realtime.channels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,

    -- Message content
    content TEXT NOT NULL,
    message_type TEXT NOT NULL DEFAULT 'text' CHECK (message_type IN ('text', 'system', 'file', 'event')),

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Delivery
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    edited_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_messages_channel ON realtime.messages(channel_id);
CREATE INDEX idx_messages_user ON realtime.messages(user_id);
CREATE INDEX idx_messages_sent_at ON realtime.messages(sent_at);

-- Presence (who's online, where)
CREATE TABLE realtime.presence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    channel_id UUID REFERENCES realtime.channels(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE,

    -- Presence info
    status TEXT NOT NULL DEFAULT 'online' CHECK (status IN ('online', 'away', 'busy', 'offline')),

    -- Location context
    page_url TEXT,
    resource_type TEXT, -- 'document', 'project', 'canvas', etc.
    resource_id UUID,

    -- Cursor/selection (for collaborative editing)
    cursor_position JSONB,
    selection JSONB,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Timestamps
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, channel_id)
);

CREATE INDEX idx_presence_user ON realtime.presence(user_id);
CREATE INDEX idx_presence_channel ON realtime.presence(channel_id);
CREATE INDEX idx_presence_tenant ON realtime.presence(tenant_id);
CREATE INDEX idx_presence_status ON realtime.presence(status);

-- Broadcasts (ephemeral messages that don't persist)
CREATE TABLE realtime.broadcasts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES realtime.channels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,

    -- Event
    event_type TEXT NOT NULL, -- 'typing', 'cursor_move', 'selection_change', etc.
    payload JSONB NOT NULL,

    -- Expiry
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '5 minutes'
);

CREATE INDEX idx_broadcasts_channel ON realtime.broadcasts(channel_id);
CREATE INDEX idx_broadcasts_expires ON realtime.broadcasts(expires_at);

-- Subscriptions (user's channel subscriptions with filters)
CREATE TABLE realtime.subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    channel_id UUID NOT NULL REFERENCES realtime.channels(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES realtime.connections(id) ON DELETE CASCADE,

    -- Filters (for selective message delivery)
    filters JSONB DEFAULT '{}'::jsonb,

    subscribed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (connection_id, channel_id)
);

CREATE INDEX idx_subscriptions_user ON realtime.subscriptions(user_id);
CREATE INDEX idx_subscriptions_channel ON realtime.subscriptions(channel_id);
CREATE INDEX idx_subscriptions_connection ON realtime.subscriptions(connection_id);

-- ============================================================================
-- FUNCTIONS: Real-Time Operations
-- ============================================================================

-- Function: Record WebSocket connection
CREATE OR REPLACE FUNCTION realtime.connect(
    p_user_id UUID,
    p_tenant_id UUID,
    p_connection_id TEXT,
    p_socket_id TEXT,
    p_client_ip TEXT DEFAULT NULL,
    p_user_agent TEXT DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_id UUID;
BEGIN
    INSERT INTO realtime.connections (
        user_id, tenant_id, connection_id, socket_id,
        client_ip, user_agent, status
    )
    VALUES (
        p_user_id, p_tenant_id, p_connection_id, p_socket_id,
        p_client_ip, p_user_agent, 'connected'
    )
    RETURNING id INTO v_id;

    RETURN v_id;
END;
$$ LANGUAGE plpgsql;

-- Function: Disconnect WebSocket
CREATE OR REPLACE FUNCTION realtime.disconnect(p_connection_id TEXT)
RETURNS VOID AS $$
BEGIN
    UPDATE realtime.connections
    SET status = 'disconnected',
        disconnected_at = NOW()
    WHERE connection_id = p_connection_id;

    -- Remove presence
    DELETE FROM realtime.presence
    WHERE user_id IN (
        SELECT user_id FROM realtime.connections
        WHERE connection_id = p_connection_id
    );

    -- Remove subscriptions
    DELETE FROM realtime.subscriptions
    WHERE connection_id IN (
        SELECT id FROM realtime.connections
        WHERE connection_id = p_connection_id
    );
END;
$$ LANGUAGE plpgsql;

-- Function: Update presence
CREATE OR REPLACE FUNCTION realtime.update_presence(
    p_user_id UUID,
    p_channel_id UUID DEFAULT NULL,
    p_status TEXT DEFAULT 'online',
    p_metadata JSONB DEFAULT '{}'::jsonb
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO realtime.presence (
        user_id, channel_id, status, metadata
    )
    VALUES (
        p_user_id, p_channel_id, p_status, p_metadata
    )
    ON CONFLICT (user_id, channel_id)
    DO UPDATE SET
        status = p_status,
        metadata = p_metadata,
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Function: Send message to channel
CREATE OR REPLACE FUNCTION realtime.send_message(
    p_channel_id UUID,
    p_user_id UUID,
    p_content TEXT,
    p_message_type TEXT DEFAULT 'text',
    p_metadata JSONB DEFAULT '{}'::jsonb
)
RETURNS UUID AS $$
DECLARE
    v_message_id UUID;
    v_channel_type TEXT;
    v_is_member BOOLEAN;
BEGIN
    -- Check if channel exists and get type
    SELECT type INTO v_channel_type
    FROM realtime.channels
    WHERE id = p_channel_id;

    IF v_channel_type IS NULL THEN
        RAISE EXCEPTION 'Channel not found';
    END IF;

    -- Check if user is member (for private channels)
    IF v_channel_type IN ('private', 'presence') THEN
        SELECT EXISTS (
            SELECT 1 FROM realtime.channel_members
            WHERE channel_id = p_channel_id
            AND user_id = p_user_id
            AND can_send = true
        ) INTO v_is_member;

        IF NOT v_is_member THEN
            RAISE EXCEPTION 'User not authorized to send to this channel';
        END IF;
    END IF;

    -- Insert message
    INSERT INTO realtime.messages (
        channel_id, user_id, content, message_type, metadata
    )
    VALUES (
        p_channel_id, p_user_id, p_content, p_message_type, p_metadata
    )
    RETURNING id INTO v_message_id;

    -- Trigger notification (would be handled by WebSocket server)
    PERFORM pg_notify(
        'channel_' || p_channel_id::text,
        json_build_object(
            'type', 'new_message',
            'message_id', v_message_id,
            'user_id', p_user_id
        )::text
    );

    RETURN v_message_id;
END;
$$ LANGUAGE plpgsql;

-- Function: Get online users in channel
CREATE OR REPLACE FUNCTION realtime.get_online_users(p_channel_id UUID)
RETURNS TABLE (
    user_id UUID,
    status TEXT,
    cursor_position JSONB,
    updated_at TIMESTAMPTZ
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        p.user_id,
        p.status,
        p.cursor_position,
        p.updated_at
    FROM realtime.presence p
    WHERE p.channel_id = p_channel_id
    AND p.status != 'offline'
    ORDER BY p.updated_at DESC;
END;
$$ LANGUAGE plpgsql;

-- Function: Broadcast ephemeral event
CREATE OR REPLACE FUNCTION realtime.broadcast(
    p_channel_id UUID,
    p_user_id UUID,
    p_event_type TEXT,
    p_payload JSONB
)
RETURNS UUID AS $$
DECLARE
    v_broadcast_id UUID;
BEGIN
    INSERT INTO realtime.broadcasts (
        channel_id, user_id, event_type, payload
    )
    VALUES (
        p_channel_id, p_user_id, p_event_type, p_payload
    )
    RETURNING id INTO v_broadcast_id;

    -- Notify via PostgreSQL NOTIFY
    PERFORM pg_notify(
        'channel_' || p_channel_id::text,
        json_build_object(
            'type', 'broadcast',
            'event_type', p_event_type,
            'user_id', p_user_id,
            'payload', p_payload
        )::text
    );

    RETURN v_broadcast_id;
END;
$$ LANGUAGE plpgsql;

-- Function: Cleanup expired broadcasts
CREATE OR REPLACE FUNCTION realtime.cleanup_expired_broadcasts()
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    DELETE FROM realtime.broadcasts
    WHERE expires_at < NOW();

    GET DIAGNOSTICS v_deleted = ROW_COUNT;

    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- Function: Cleanup old connections
CREATE OR REPLACE FUNCTION realtime.cleanup_stale_connections()
RETURNS INTEGER AS $$
DECLARE
    v_deleted INTEGER;
BEGIN
    -- Mark connections as disconnected if not seen in 5 minutes
    UPDATE realtime.connections
    SET status = 'disconnected',
        disconnected_at = NOW()
    WHERE status = 'connected'
    AND last_seen_at < NOW() - INTERVAL '5 minutes';

    -- Delete connections older than 24 hours
    DELETE FROM realtime.connections
    WHERE status = 'disconnected'
    AND disconnected_at < NOW() - INTERVAL '24 hours';

    GET DIAGNOSTICS v_deleted = ROW_COUNT;

    RETURN v_deleted;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Trigger: Update last_seen_at on connection activity
CREATE OR REPLACE FUNCTION realtime.update_last_seen()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE realtime.connections
    SET last_seen_at = NOW()
    WHERE user_id = NEW.user_id
    AND status = 'connected';

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_last_seen_on_message
    AFTER INSERT ON realtime.messages
    FOR EACH ROW
    EXECUTE FUNCTION realtime.update_last_seen();

-- ============================================================================
-- VIEWS
-- ============================================================================

-- View: Active connections
CREATE OR REPLACE VIEW realtime.active_connections AS
SELECT
    c.id,
    c.user_id,
    c.tenant_id,
    c.connection_id,
    c.connected_at,
    c.last_seen_at,
    COUNT(DISTINCT s.channel_id) as subscribed_channels
FROM realtime.connections c
LEFT JOIN realtime.subscriptions s ON c.id = s.connection_id
WHERE c.status = 'connected'
GROUP BY c.id;

-- View: Channel activity
CREATE OR REPLACE VIEW realtime.channel_activity AS
SELECT
    c.id,
    c.slug,
    c.name,
    COUNT(DISTINCT cm.user_id) as total_members,
    COUNT(DISTINCT p.user_id) as online_members,
    COUNT(m.id) as message_count_24h
FROM realtime.channels c
LEFT JOIN realtime.channel_members cm ON c.id = cm.channel_id
LEFT JOIN realtime.presence p ON c.id = p.channel_id AND p.status != 'offline'
LEFT JOIN realtime.messages m ON c.id = m.channel_id AND m.sent_at > NOW() - INTERVAL '24 hours'
GROUP BY c.id;

-- ============================================================================
-- ROW LEVEL SECURITY
-- ============================================================================

ALTER TABLE realtime.connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE realtime.channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE realtime.channel_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE realtime.messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE realtime.presence ENABLE ROW LEVEL SECURITY;

-- RLS: Users can see their own connections
CREATE POLICY connections_own ON realtime.connections
    FOR ALL
    USING (user_id = tenants.current_user_id());

-- RLS: Users can see channels they're members of
CREATE POLICY channels_member ON realtime.channels
    FOR SELECT
    USING (
        type = 'public'
        OR
        id IN (
            SELECT channel_id FROM realtime.channel_members
            WHERE user_id = tenants.current_user_id()
        )
    );

-- RLS: Users can see messages in their channels
CREATE POLICY messages_channel_member ON realtime.messages
    FOR SELECT
    USING (
        channel_id IN (
            SELECT channel_id FROM realtime.channel_members
            WHERE user_id = tenants.current_user_id()
        )
        OR
        channel_id IN (
            SELECT id FROM realtime.channels WHERE type = 'public'
        )
    );

-- RLS: Users can send messages to channels they're in
CREATE POLICY messages_send ON realtime.messages
    FOR INSERT
    WITH CHECK (
        user_id = tenants.current_user_id()
        AND
        channel_id IN (
            SELECT channel_id FROM realtime.channel_members
            WHERE user_id = tenants.current_user_id()
            AND can_send = true
        )
    );

-- ============================================================================
-- GRANTS
-- ============================================================================

GRANT USAGE ON SCHEMA realtime TO hasura;
GRANT SELECT ON ALL TABLES IN SCHEMA realtime TO hasura;
GRANT INSERT, UPDATE, DELETE ON realtime.connections TO hasura;
GRANT INSERT, UPDATE, DELETE ON realtime.messages TO hasura;
GRANT INSERT, UPDATE, DELETE ON realtime.presence TO hasura;
GRANT INSERT, UPDATE, DELETE ON realtime.broadcasts TO hasura;
GRANT INSERT, UPDATE, DELETE ON realtime.subscriptions TO hasura;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA realtime TO hasura;

COMMIT;
