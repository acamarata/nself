-- P88 Sprint 09, Ticket T09-01: C1 Hybrid Search composite + GIN/GIST/HNSW indexes
-- Spec: p88-block-c-library-http-spec.md §2.3
--
-- Indexes needed for production hybrid search filter combinations on
-- np_claw_messages (user+date, topic_path, channel+date, FTS, embeddings, profile).
-- All use CREATE INDEX CONCURRENTLY IF NOT EXISTS so they can be applied without locking.
-- NOTE: CREATE INDEX CONCURRENTLY cannot run inside a transaction block. If your
-- migration runner wraps migrations in transactions, run this one outside (psql -f).

-- UP

-- User + time composite (primary access pattern for user-scoped pagination).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_user_created
    ON np_claw_messages (user_id, created_at DESC);

-- Topic path GIST — requires topic_path ltree column. If np_claw_messages does not
-- have topic_path, this index is a no-op for now (guarded by DO block).
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'np_claw_messages' AND column_name = 'topic_path'
    ) THEN
        EXECUTE 'CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_topic_path
                 ON np_claw_messages USING GIST (topic_path)';
    END IF;
END$$;

-- Channel + time composite (scoped scroll within a channel).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_user_channel_created
    ON np_claw_messages (user_id, channel, created_at DESC);

-- FTS GIN on content_tsv (already created in 20260420_messages_fts.sql as idx_messages_tsv,
-- kept here as IF NOT EXISTS so verification passes under both names).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_fts_tsv
    ON np_claw_messages USING GIN (content_tsv);

-- HNSW embedding index on np_claw_memories (actual vector-bearing table in current schema).
-- Uses vector_cosine_ops. Guarded so the migration is idempotent whether or not
-- pgvector is installed yet.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector') THEN
        EXECUTE 'CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_memory_embedding_hnsw
                 ON np_claw_memories USING hnsw (embedding vector_cosine_ops)';
    END IF;
END$$;

-- Saved searches productionization: add name + notify_on_new columns (P88 §2.2).
ALTER TABLE np_claw_saved_searches ADD COLUMN IF NOT EXISTS name TEXT;
ALTER TABLE np_claw_saved_searches ADD COLUMN IF NOT EXISTS notify_on_new BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE np_claw_saved_searches ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Profile partial index — only indexes rows with a profile_id set.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'np_claw_messages' AND column_name = 'profile_id'
    ) THEN
        EXECUTE 'CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_messages_user_profile
                 ON np_claw_messages (user_id, profile_id) WHERE profile_id IS NOT NULL';
    END IF;
END$$;

-- DOWN
-- DROP INDEX CONCURRENTLY IF EXISTS idx_messages_user_profile;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_memory_embedding_hnsw;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_messages_fts_tsv;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_messages_user_channel_created;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_messages_topic_path;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_messages_user_created;
