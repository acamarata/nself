-- DOWN
-- Rollback for 091b_p88_integration_probes.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP POLICY IF EXISTS np_claw_user_notif_policy ON np_claw_user_notifications;
ALTER TABLE IF EXISTS np_claw_user_notifications DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_claw_user_notif_user;
DROP TABLE IF EXISTS np_claw_user_notifications CASCADE;
DROP POLICY IF EXISTS np_claw_int_probes_user_policy ON np_claw_integration_probes;
ALTER TABLE IF EXISTS np_claw_integration_probes DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_claw_int_probes_user_tool_time;
DROP TABLE IF EXISTS np_claw_integration_probes CASCADE;
