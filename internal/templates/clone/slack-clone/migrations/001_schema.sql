-- slack-clone schema
-- Tables: np_workspaces, np_channels, np_messages, np_threads, np_reactions

CREATE TABLE IF NOT EXISTS np_workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    owner_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES np_workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    is_private BOOLEAN NOT NULL DEFAULT FALSE,
    topic TEXT,
    description TEXT,
    created_by UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, name)
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

CREATE TABLE IF NOT EXISTS np_threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_message_id UUID NOT NULL REFERENCES np_messages(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_reactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES np_messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    emoji TEXT NOT NULL,
    UNIQUE(message_id, user_id, emoji)
);

-- RLS
ALTER TABLE np_workspaces ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_threads ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_reactions ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_workspaces_owner ON np_workspaces USING (owner_id = auth.uid());
CREATE POLICY np_channels_select ON np_channels FOR SELECT USING (is_private = FALSE OR created_by = auth.uid());
CREATE POLICY np_messages_select ON np_messages FOR SELECT USING (TRUE);
CREATE POLICY np_messages_author ON np_messages USING (author_id = auth.uid());
CREATE POLICY np_threads_select ON np_threads FOR SELECT USING (TRUE);
CREATE POLICY np_threads_author ON np_threads USING (author_id = auth.uid());
CREATE POLICY np_reactions_select ON np_reactions FOR SELECT USING (TRUE);
CREATE POLICY np_reactions_user ON np_reactions USING (user_id = auth.uid());

-- Indexes
CREATE INDEX IF NOT EXISTS np_messages_channel_idx ON np_messages(channel_id, created_at DESC);
CREATE INDEX IF NOT EXISTS np_threads_parent_idx ON np_threads(parent_message_id);
CREATE INDEX IF NOT EXISTS np_reactions_message_idx ON np_reactions(message_id);
