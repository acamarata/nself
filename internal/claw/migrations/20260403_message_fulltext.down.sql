-- DOWN
-- Rollback for 20260403_message_fulltext.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_messages_content_tsv;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS content_tsv;
