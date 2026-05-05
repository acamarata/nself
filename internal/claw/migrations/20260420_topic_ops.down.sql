-- DOWN
-- Rollback for 20260420_topic_ops.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP POLICY IF EXISTS topic_ops_user_isolation ON np_claw_topic_ops_log;
ALTER TABLE IF EXISTS np_claw_topic_ops_log DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS np_claw_topic_ops_log CASCADE;
ALTER TABLE IF EXISTS np_claw_topics DROP COLUMN IF EXISTS split_from;
ALTER TABLE IF EXISTS np_claw_topics DROP COLUMN IF EXISTS merged_into;
ALTER TABLE IF EXISTS np_claw_topics DROP COLUMN IF EXISTS last_active_at;
ALTER TABLE IF EXISTS np_claw_topics DROP COLUMN IF EXISTS archived_at;
