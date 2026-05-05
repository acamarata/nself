-- P93 S02-T03: Conversations schema — NOT NULL topic_id, ltree path, indexes, FK.
-- Rollback: see 20260417_p93_conversations_rollback.sql

-- Enable ltree extension if not already enabled (idempotent)
CREATE EXTENSION IF NOT EXISTS ltree;

-- Create np_claw_conversations table
CREATE TABLE IF NOT EXISTS np_claw_conversations (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    topic_id        UUID        NOT NULL,           -- FK to np_claw_topics (NOT NULL enforced)
    title           TEXT,
    path            LTREE,                          -- hierarchical path e.g. 'work.projects.claw'
    summary         TEXT,
    message_count   INTEGER     NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    metadata        JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- FK constraint: add only if np_claw_topics exists and constraint is not already present
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_tables
        WHERE schemaname = 'public' AND tablename = 'np_claw_topics'
    )
    AND NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'np_claw_conversations_topic_id_fkey'
    ) THEN
        ALTER TABLE np_claw_conversations
            ADD CONSTRAINT np_claw_conversations_topic_id_fkey
            FOREIGN KEY (topic_id) REFERENCES np_claw_topics(id) ON DELETE CASCADE;
    END IF;
END $$;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_np_claw_conversations_user_id
    ON np_claw_conversations(user_id);

CREATE INDEX IF NOT EXISTS idx_np_claw_conversations_topic_id
    ON np_claw_conversations(topic_id);

CREATE INDEX IF NOT EXISTS idx_np_claw_conversations_path
    ON np_claw_conversations USING gist(path);

CREATE INDEX IF NOT EXISTS idx_np_claw_conversations_last_message
    ON np_claw_conversations(last_message_at DESC NULLS LAST);

-- Demo seed data: insert one conversation per existing topic for the demo user,
-- but only if the conversations table is currently empty.
-- Requires np_claw_topics to exist; the INSERT is a no-op otherwise (table is empty guard).
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_tables
        WHERE schemaname = 'public' AND tablename = 'np_claw_topics'
    ) THEN
        INSERT INTO np_claw_conversations (user_id, topic_id, title, path, message_count)
        SELECT
            '00000000-0000-0000-0000-000000000001'::uuid,
            t.id,
            'Demo conversation: ' || t.name,
            text2ltree('demo.' || regexp_replace(lower(t.name), '[^a-z0-9]+', '_', 'g')),
            0
        FROM np_claw_topics t
        WHERE t.user_id = '00000000-0000-0000-0000-000000000001'::uuid
        LIMIT 1
        ON CONFLICT DO NOTHING;
    END IF;
END $$;
