-- DOWN
-- Rollback for 001_init.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP TABLE IF EXISTS np_claw_tools CASCADE;
DROP INDEX IF EXISTS idx_np_claw_sessions_user_id;
DROP TABLE IF EXISTS np_claw_sessions CASCADE;
DROP INDEX IF EXISTS idx_np_claw_memory_user_id;
DROP TABLE IF EXISTS np_claw_memory CASCADE;
