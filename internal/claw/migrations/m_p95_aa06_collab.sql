-- P95 AA06: Claw Collaboration Multi-User
-- Feature flag: claw_collab (Y17) — gated behind flag; safe to migrate ahead of flag ON.

BEGIN;

-- Shared conversation membership.
-- Tracks which users have access to each conversation and at what role level.
CREATE TABLE IF NOT EXISTS np_claw_shared_conversations (
    conversation_id uuid    NOT NULL REFERENCES np_claw_conversations(id) ON DELETE CASCADE,
    user_id         uuid    NOT NULL REFERENCES np_users(id) ON DELETE CASCADE,
    role            text    NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    joined_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_np_claw_shared_conversations_user
    ON np_claw_shared_conversations (user_id);

-- Team Knowledge Base.
-- Stores curated org-wide facts that agents inject into shared conversation context.
-- Requires pgvector extension (already enabled by the ai plugin migration).
CREATE TABLE IF NOT EXISTS np_claw_team_kb (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    title      text        NOT NULL,
    body       text        NOT NULL,
    embedding  vector(1536),          -- populated by ai plugin; nullable during backfill
    tags       text[]      NOT NULL DEFAULT '{}',
    created_by uuid        NOT NULL REFERENCES np_users(id),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_team_kb_embedding
    ON np_claw_team_kb USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 50);

CREATE INDEX IF NOT EXISTS idx_np_claw_team_kb_updated_at
    ON np_claw_team_kb (updated_at DESC);

-- Mention records.
-- Created when a message contains an @-mention of an org member.
CREATE TABLE IF NOT EXISTS np_claw_mentions (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id      uuid        NOT NULL REFERENCES np_claw_messages(id) ON DELETE CASCADE,
    mentioned_user  uuid        NOT NULL REFERENCES np_users(id) ON DELETE CASCADE,
    seen            boolean     NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_mentions_user_unseen
    ON np_claw_mentions (mentioned_user, seen)
    WHERE seen = false;

-- Feature flag row (OFF by default, per Y17 spec).
-- The flags.Store reads this table at startup and caches in memory.
INSERT INTO np_feature_flags (flag, enabled, updated_at)
VALUES ('claw_collab', false, now())
ON CONFLICT (flag) DO NOTHING;

COMMIT;
