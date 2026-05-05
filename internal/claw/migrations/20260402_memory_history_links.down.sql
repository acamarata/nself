-- DOWN
-- Rollback for 20260402_memory_history_links.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS confidence;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS last_accessed_at;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS access_count;
DROP INDEX IF EXISTS idx_np_claw_memory_links_b;
DROP INDEX IF EXISTS idx_np_claw_memory_links_a;
DROP TABLE IF EXISTS np_claw_memory_links CASCADE;
DROP INDEX IF EXISTS idx_np_claw_memory_history_memory_id;
DROP TABLE IF EXISTS np_claw_memory_history CASCADE;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS consolidated_from;
