-- DOWN
-- Rollback for 008_playbooks.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_playbooks_name;
DROP TABLE IF EXISTS np_claw_playbooks CASCADE;
