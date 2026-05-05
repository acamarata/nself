-- DOWN
-- Rollback for 002_actions.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_claw_actions_expires;
DROP INDEX IF EXISTS idx_claw_actions_device;
DROP INDEX IF EXISTS idx_claw_actions_user_status;
DROP TABLE IF EXISTS np_claw_actions CASCADE;
