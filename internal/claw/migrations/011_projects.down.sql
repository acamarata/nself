-- DOWN
-- Rollback for 011_projects.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_sessions DROP CONSTRAINT IF EXISTS IF;
DROP INDEX IF EXISTS idx_np_claw_projects_active_name;
DROP INDEX IF EXISTS idx_np_claw_projects_archived;
DROP TABLE IF EXISTS np_claw_projects CASCADE;
