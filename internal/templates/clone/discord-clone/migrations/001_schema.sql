-- discord-clone schema
-- Tables: np_servers, np_channels, np_messages, np_members, np_roles

CREATE TABLE IF NOT EXISTS np_servers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    icon_url TEXT,
    invite_code TEXT UNIQUE DEFAULT substr(md5(random()::text), 1, 8),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES np_servers(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT DEFAULT '#99AAB5',
    permissions JSONB DEFAULT '{}',
    position INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS np_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES np_servers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role_id UUID REFERENCES np_roles(id) ON DELETE SET NULL,
    nickname TEXT,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(server_id, user_id)
);

CREATE TABLE IF NOT EXISTS np_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id UUID NOT NULL REFERENCES np_servers(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'text' CHECK (type IN ('text','voice','announcement')),
    position INTEGER NOT NULL DEFAULT 0,
    topic TEXT,
    is_private BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES np_channels(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    attachments JSONB DEFAULT '[]',
    is_edited BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RLS
ALTER TABLE np_servers ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_roles ENABLE ROW LEVEL SECURITY;

-- Members can read servers they belong to
CREATE POLICY np_servers_member ON np_servers FOR SELECT USING (
    id IN (SELECT server_id FROM np_members WHERE user_id = auth.uid())
    OR owner_id = auth.uid()
);
CREATE POLICY np_servers_owner ON np_servers USING (owner_id = auth.uid());

-- Channels visible to server members
CREATE POLICY np_channels_member ON np_channels FOR SELECT USING (
    server_id IN (SELECT server_id FROM np_members WHERE user_id = auth.uid())
);

-- Messages visible to channel members
CREATE POLICY np_messages_member ON np_messages FOR SELECT USING (
    channel_id IN (
        SELECT c.id FROM np_channels c
        JOIN np_members m ON m.server_id = c.server_id
        WHERE m.user_id = auth.uid()
    )
);
CREATE POLICY np_messages_author ON np_messages USING (author_id = auth.uid());

-- Indexes
CREATE INDEX IF NOT EXISTS np_messages_channel_idx ON np_messages(channel_id, created_at DESC);
CREATE INDEX IF NOT EXISTS np_members_server_idx ON np_members(server_id);
CREATE INDEX IF NOT EXISTS np_members_user_idx ON np_members(user_id);
