-- DOWN
-- Rollback for 006_agent_queue.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_agent_queue_expires;
DROP INDEX IF EXISTS idx_np_claw_agent_queue_namespace_status;
DROP TABLE IF EXISTS np_claw_agent_queue CASCADE;
