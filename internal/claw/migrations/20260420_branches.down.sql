-- DOWN
-- Rollback for 20260420_branches.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP POLICY IF EXISTS branches_user_isolation ON np_claw_branches;
ALTER TABLE IF EXISTS np_claw_branches DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_branches_session;
DROP TABLE IF EXISTS np_claw_branches CASCADE;
