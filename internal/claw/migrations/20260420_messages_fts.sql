-- P87 Sprint 01, Ticket 01.T05: FTS + Channel + Branch Columns on np_claw_messages
-- Spec §4.5

-- UP

CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS content_tsv       tsvector GENERATED ALWAYS AS (to_tsvector('simple', coalesce(content, ''))) STORED;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS channel           TEXT NOT NULL DEFAULT 'chat';
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS topic_id          UUID REFERENCES np_claw_topics(id) ON DELETE SET NULL;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS subtopic          TEXT;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS branch_id         UUID;
ALTER TABLE np_claw_messages ADD COLUMN IF NOT EXISTS parent_branch_id  UUID;

CREATE INDEX IF NOT EXISTS idx_messages_tsv     ON np_claw_messages USING GIN(content_tsv);
CREATE INDEX IF NOT EXISTS idx_messages_trgm    ON np_claw_messages USING GIN(content gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_messages_channel ON np_claw_messages(user_id, channel, created_at DESC);

-- DOWN
-- DROP INDEX IF EXISTS idx_messages_channel;
-- DROP INDEX IF EXISTS idx_messages_trgm;
-- DROP INDEX IF EXISTS idx_messages_tsv;
-- ALTER TABLE np_claw_messages DROP COLUMN IF EXISTS parent_branch_id;
-- ALTER TABLE np_claw_messages DROP COLUMN IF EXISTS branch_id;
-- ALTER TABLE np_claw_messages DROP COLUMN IF EXISTS subtopic;
-- ALTER TABLE np_claw_messages DROP COLUMN IF EXISTS topic_id;
-- ALTER TABLE np_claw_messages DROP COLUMN IF EXISTS channel;
-- ALTER TABLE np_claw_messages DROP COLUMN IF EXISTS content_tsv;
