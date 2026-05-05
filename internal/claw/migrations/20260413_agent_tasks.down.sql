-- DOWN
-- Rollback for 20260413_agent_tasks.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_claw_agent_tasks_status;
DROP INDEX IF EXISTS idx_claw_agent_tasks_agent_id;
DROP TABLE IF EXISTS np_claw_agent_tasks CASCADE;
