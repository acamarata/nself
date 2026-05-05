-- DOWN
-- Rollback for 20260412_agents_teams.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_team_members_agent;
DROP TABLE IF EXISTS np_claw_team_members CASCADE;
DROP TABLE IF EXISTS np_claw_teams CASCADE;
DROP INDEX IF EXISTS idx_np_claw_agents_enabled;
DROP INDEX IF EXISTS idx_np_claw_agents_user;
DROP TABLE IF EXISTS np_claw_agents CASCADE;
