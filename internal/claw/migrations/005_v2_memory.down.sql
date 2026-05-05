-- DOWN
-- Rollback for 005_v2_memory.sql
-- Auto-generated inverse operations. Idempotent (IF EXISTS / IF NOT EXISTS).
-- Review manually — some statements may be marked as non-reversible.

DROP TABLE IF EXISTS np_claw_personas CASCADE;
DROP TABLE IF EXISTS cl_thread_cores CASCADE;
DROP INDEX IF EXISTS idx_cl_memory_blocks_embedding;
DROP INDEX IF EXISTS idx_cl_memory_blocks_thread;
DROP TABLE IF EXISTS cl_memory_blocks CASCADE;
DROP INDEX IF EXISTS idx_cl_messages_embedding;
DROP INDEX IF EXISTS idx_cl_messages_thread_id;
DROP TABLE IF EXISTS cl_messages CASCADE;
DROP INDEX IF EXISTS idx_cl_threads_user_namespace;
DROP TABLE IF EXISTS cl_threads CASCADE;
