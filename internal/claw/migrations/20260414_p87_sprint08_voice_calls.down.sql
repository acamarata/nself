-- DOWN
-- Rollback for 20260414_p87_sprint08_voice_calls.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_voice_calls_started;
DROP INDEX IF EXISTS idx_np_claw_voice_calls_session;
DROP INDEX IF EXISTS idx_np_claw_voice_calls_user;
DROP TABLE IF EXISTS np_claw_voice_calls CASCADE;
