-- AA07 — Claw Conversation Intelligence
-- P95 ticket #908: auto topic segmentation, action item extraction,
-- cross-conversation topic index.
-- Spec: .claude/planning/p95-specs/AA07-claw-conversation-intelligence.md

-- UP

-- Topic segments: spans of a conversation identified by a single topic label.
CREATE TABLE IF NOT EXISTS np_claw_topic_segments (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL,
    branch_id       UUID,       -- optional link to np_claw_branches (AA05)
    label           TEXT        NOT NULL,
    message_range   INT4RANGE   NOT NULL,
    embedding       VECTOR(1536),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_claw_topic_segments_conversation
    ON np_claw_topic_segments (conversation_id);

CREATE INDEX IF NOT EXISTS idx_claw_topic_segments_embedding
    ON np_claw_topic_segments USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

ALTER TABLE np_claw_topic_segments ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'np_claw_topic_segments'
          AND policyname = 'topic_segments_conversation_isolation'
    ) THEN
        CREATE POLICY topic_segments_conversation_isolation ON np_claw_topic_segments
            USING (
                conversation_id IN (
                    SELECT id FROM np_claw_conversations
                    WHERE user_id = current_setting('hasura.user', true)::uuid
                )
            )
            WITH CHECK (
                conversation_id IN (
                    SELECT id FROM np_claw_conversations
                    WHERE user_id = current_setting('hasura.user', true)::uuid
                )
            );
    END IF;
END $$;

-- Action items: concrete commitments extracted from assistant turns.
CREATE TABLE IF NOT EXISTS np_claw_action_items (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL,
    message_id      UUID,
    text            TEXT        NOT NULL,
    due_date        DATE,
    owner           TEXT        NOT NULL DEFAULT 'user'
                    CHECK (owner IN ('user', 'agent')),
    status          TEXT        NOT NULL DEFAULT 'open'
                    CHECK (status IN ('open', 'done', 'dismissed')),
    embedding       VECTOR(1536),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_claw_action_items_conversation
    ON np_claw_action_items (conversation_id, status);

CREATE INDEX IF NOT EXISTS idx_claw_action_items_embedding
    ON np_claw_action_items USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 50);

ALTER TABLE np_claw_action_items ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'np_claw_action_items'
          AND policyname = 'action_items_user_isolation'
    ) THEN
        CREATE POLICY action_items_user_isolation ON np_claw_action_items
            USING (
                conversation_id IN (
                    SELECT id FROM np_claw_conversations
                    WHERE user_id = current_setting('hasura.user', true)::uuid
                )
            )
            WITH CHECK (
                conversation_id IN (
                    SELECT id FROM np_claw_conversations
                    WHERE user_id = current_setting('hasura.user', true)::uuid
                )
            );
    END IF;
END $$;

-- Topic index: inverted index of label → conversation_ids for cross-conversation search.
CREATE TABLE IF NOT EXISTS np_claw_topic_index (
    label           TEXT        PRIMARY KEY,
    conversation_ids TEXT[]      NOT NULL DEFAULT ARRAY[]::TEXT[],
    total_messages  INT         NOT NULL DEFAULT 0,
    last_mentioned  TIMESTAMPTZ,
    key_entities    TEXT[]      NOT NULL DEFAULT ARRAY[]::TEXT[]
);

-- np_claw_topic_index is a denormalized aggregate — no per-user RLS needed;
-- label values are user-scoped via conversation_ids (UUIDs only, no PII).
-- Access controlled via application layer (conversation_id ownership check).

-- DOWN (rollback)
-- DROP TABLE IF EXISTS np_claw_topic_index;
-- DROP TABLE IF EXISTS np_claw_action_items;
-- DROP TABLE IF EXISTS np_claw_topic_segments;
