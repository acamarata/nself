-- T-7363: Full-text search across conversations
-- Add tsvector column to np_claw_messages with GIN index.

ALTER TABLE np_claw_messages
  ADD COLUMN IF NOT EXISTS content_tsv tsvector
    GENERATED ALWAYS AS (to_tsvector('english', content)) STORED;

CREATE INDEX IF NOT EXISTS idx_np_claw_messages_content_tsv
  ON np_claw_messages USING GIN (content_tsv);
