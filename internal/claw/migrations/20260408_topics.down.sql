-- DOWN
-- Rollback for 20260408_topics.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP INDEX IF EXISTS idx_memories_global;
DROP INDEX IF EXISTS idx_memories_topic;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS is_global;
ALTER TABLE IF EXISTS np_claw_memories DROP COLUMN IF EXISTS topic_id;
DROP TABLE IF EXISTS np_claw_topic_prefs CASCADE;
DROP INDEX IF EXISTS idx_topic_knowledge_node;
DROP TABLE IF EXISTS np_claw_topic_knowledge CASCADE;
DROP INDEX IF EXISTS idx_transitions_user;
DROP INDEX IF EXISTS idx_transitions_thread;
DROP TABLE IF EXISTS np_claw_topic_transitions CASCADE;
DROP INDEX IF EXISTS idx_topic_rels_target;
DROP INDEX IF EXISTS idx_topic_rels_source;
DROP TABLE IF EXISTS np_claw_topic_relations CASCADE;
DROP INDEX IF EXISTS idx_msg_topics_primary;
DROP INDEX IF EXISTS idx_msg_topics_message;
DROP INDEX IF EXISTS idx_msg_topics_topic;
DROP TABLE IF EXISTS np_claw_message_topics CASCADE;
DROP TRIGGER IF EXISTS trg_topic_sync_parent ON np_claw_topics;
DROP FUNCTION IF EXISTS sync_topic_parent_from_path() CASCADE;
DROP INDEX IF EXISTS idx_topics_embedding;
DROP INDEX IF EXISTS idx_topics_pinned;
DROP INDEX IF EXISTS idx_topics_user_parent;
DROP INDEX IF EXISTS idx_topics_user_path;
DROP INDEX IF EXISTS idx_topics_user_active;
DROP TABLE IF EXISTS np_claw_topics CASCADE;
