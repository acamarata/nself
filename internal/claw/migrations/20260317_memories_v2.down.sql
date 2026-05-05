-- DOWN
-- Rollback for 20260317_memories_v2.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- update np_claw_memories
  set memory_type = 'fact',
      so... cannot be auto-reversed; verify manually if rollback needed.
DROP INDEX IF EXISTS memories_subject_idx;
DROP INDEX IF EXISTS memories_type_idx;
DROP INDEX IF EXISTS memories_embedding_hnsw;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS blind_index;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS content_embedding;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS last_used_at;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS use_count;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS expires_at;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS superseded_by;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS source_conversation_id;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS source_type;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS subject;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS memory_type;
