-- DOWN
-- Rollback for 20260420_briefings.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP POLICY IF EXISTS briefings_user_isolation ON np_claw_briefings;
ALTER TABLE IF EXISTS np_claw_briefings DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS np_claw_briefings CASCADE;
-- (derived from DO block — review manually)
DROP POLICY IF EXISTS briefing_config_user_isolation ON np_claw_briefing_config;
ALTER TABLE IF EXISTS np_claw_briefing_config DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS np_claw_briefing_config CASCADE;
