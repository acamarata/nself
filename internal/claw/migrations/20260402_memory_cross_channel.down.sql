-- DOWN
-- Rollback for 20260402_memory_cross_channel.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_np_claw_memories_entity_channel;
DROP INDEX IF EXISTS idx_np_claw_memories_source_channel;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS source_channel;
