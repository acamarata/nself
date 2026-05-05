-- DOWN
-- Rollback for 015_pending_mutations.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_pending_mutations_state;
DROP TABLE IF EXISTS np_claw_pending_mutations CASCADE;
