-- DOWN
-- Rollback for 20260420_touchpoints.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP POLICY IF EXISTS touchpoints_user_isolation ON np_claw_touchpoints;
ALTER TABLE IF EXISTS np_claw_touchpoints DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS np_claw_touchpoints CASCADE;
