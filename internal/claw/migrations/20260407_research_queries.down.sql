-- DOWN
-- Rollback for 20260407_research_queries.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_claw_research_created;
DROP INDEX IF EXISTS idx_claw_research_user_id;
DROP TABLE IF EXISTS np_claw_research_results CASCADE;
