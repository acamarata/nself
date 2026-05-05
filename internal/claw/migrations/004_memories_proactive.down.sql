-- DOWN
-- Rollback for 004_memories_proactive.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP TABLE IF EXISTS np_claw_user_preferences CASCADE;
DROP TABLE IF EXISTS np_claw_proactive_jobs CASCADE;
DROP INDEX IF EXISTS idx_np_claw_memories_confidence;
DROP INDEX IF EXISTS idx_np_claw_memories_entity_type;
DROP INDEX IF EXISTS idx_np_claw_memories_entity_id;
DROP TABLE IF EXISTS np_claw_memories CASCADE;
