-- DOWN
-- Rollback for 20260413_shared_sessions.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_shared_sessions_user;
DROP INDEX IF EXISTS idx_np_claw_shared_sessions_session;
DROP INDEX IF EXISTS idx_np_claw_shared_sessions_token;
DROP TABLE IF EXISTS np_claw_shared_sessions CASCADE;
