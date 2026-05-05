-- DOWN
-- Rollback for 20260420_messages_fts.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_messages_channel;
DROP INDEX IF EXISTS idx_messages_trgm;
DROP INDEX IF EXISTS idx_messages_tsv;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS parent_branch_id;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS branch_id;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS subtopic;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS topic_id;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS channel;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS content_tsv;
