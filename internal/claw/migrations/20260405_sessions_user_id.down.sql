-- DOWN
-- Rollback for 20260405_sessions_user_id.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_sessions_user_id;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS user_id;
