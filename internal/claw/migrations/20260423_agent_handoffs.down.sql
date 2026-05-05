-- Rollback: drop agent handoff table (Y02 — P95)
DROP INDEX IF EXISTS idx_np_agent_handoffs_to_agent;
DROP INDEX IF EXISTS idx_np_agent_handoffs_status;
DROP TABLE IF EXISTS np_agent_handoffs;
