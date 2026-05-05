-- DOWN
-- Rollback for m_p88_c1_indexes.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

-- (derived from DO block — review manually)
DROP INDEX IF EXISTS idx_messages_user_profile;
ALTER TABLE IF EXISTS np_claw_saved_searches DROP COLUMN IF EXISTS updated_at;
ALTER TABLE IF EXISTS np_claw_saved_searches DROP COLUMN IF EXISTS notify_on_new;
ALTER TABLE IF EXISTS np_claw_saved_searches DROP COLUMN IF EXISTS name;
-- (derived from DO block — review manually)
DROP INDEX IF EXISTS idx_memory_embedding_hnsw;
DROP INDEX IF EXISTS idx_messages_fts_tsv;
DROP INDEX IF EXISTS idx_messages_user_channel_created;
-- (derived from DO block — review manually)
DROP INDEX IF EXISTS idx_messages_topic_path;
DROP INDEX IF EXISTS idx_messages_user_created;
