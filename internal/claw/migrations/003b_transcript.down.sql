-- DOWN
-- Rollback for 003b_transcript.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_messages_content_fts;
DROP INDEX IF EXISTS idx_np_claw_messages_user;
ALTER TABLE IF EXISTS np_claw_messages DROP CONSTRAINT IF EXISTS np_claw_messages_role_check;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS tool_call_id;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS tool_calls;
ALTER TABLE IF EXISTS np_claw_messages DROP COLUMN IF EXISTS user_id;
