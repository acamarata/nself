-- DOWN
-- Rollback for 20260317_memories_v2_create.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS memories_subject_idx;
DROP INDEX IF EXISTS memories_type_idx;
DROP INDEX IF EXISTS memories_embedding_hnsw;
DROP INDEX IF EXISTS idx_np_claw_memories_confidence;
DROP INDEX IF EXISTS idx_np_claw_memories_entity_channel;
DROP INDEX IF EXISTS idx_np_claw_memories_source_channel;
DROP INDEX IF EXISTS idx_np_claw_memories_entity_type;
DROP INDEX IF EXISTS idx_np_claw_memories_project_id;
DROP INDEX IF EXISTS idx_np_claw_memories_entity_id;
DROP TABLE IF EXISTS np_claw_memories CASCADE;
