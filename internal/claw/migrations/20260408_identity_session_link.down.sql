-- DOWN
-- Rollback for 20260408_identity_session_link.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_sessions_identity_id;
ALTER TABLE IF EXISTS np_claw_sessions DROP COLUMN IF EXISTS identity_id;
