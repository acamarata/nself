-- DOWN
-- Rollback for 009_admin_context.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_admin_context_current;
DROP TABLE IF EXISTS np_claw_admin_context CASCADE;
