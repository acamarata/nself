-- DOWN
-- Rollback for 20260420_memory_conflicts.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP POLICY IF EXISTS memory_conflicts_user_isolation ON np_claw_memory_conflicts;
ALTER TABLE IF EXISTS np_claw_memory_conflicts DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_conflicts_pending;
DROP TABLE IF EXISTS np_claw_memory_conflicts CASCADE;
