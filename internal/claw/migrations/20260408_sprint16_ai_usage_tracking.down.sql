-- DOWN
-- Rollback for 20260408_sprint16_ai_usage_tracking.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS metadata;
DROP INDEX IF EXISTS idx_np_claw_ai_usage_session;
DROP INDEX IF EXISTS idx_np_claw_ai_usage_created;
DROP TABLE IF EXISTS np_claw_ai_usage CASCADE;
