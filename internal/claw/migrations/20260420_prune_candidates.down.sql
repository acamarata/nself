-- DOWN
-- Rollback for 20260420_prune_candidates.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP POLICY IF EXISTS prune_candidates_user_isolation ON np_claw_prune_candidates;
ALTER TABLE IF EXISTS np_claw_prune_candidates DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS np_claw_prune_candidates CASCADE;
