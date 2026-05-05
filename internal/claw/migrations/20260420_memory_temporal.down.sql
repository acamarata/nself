-- DOWN
-- Rollback for 20260420_memory_temporal.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_memory_valid_range;
DROP INDEX IF EXISTS idx_memory_valid_current;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS archived_at;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS source_type;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS confidence;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS superseded_by;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS valid_to;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS valid_from;
