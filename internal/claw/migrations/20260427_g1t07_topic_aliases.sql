-- G1-T07: MAR (Memory-Augmented Retrieval) — topic aliases + embeddings.
-- Spec: P97 G1-T07
--
-- Aliases widen topic match by storing alternate names + per-alias embeddings
-- so the retrieval pipeline can resolve "auto loan" -> "Cars" topic, etc.

-- UP

CREATE TABLE IF NOT EXISTS np_claw_topic_aliases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id    UUID NOT NULL REFERENCES np_claw_topics(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    alias       TEXT NOT NULL,
    embedding   vector(1536),
    confidence  REAL NOT NULL DEFAULT 1.0 CHECK (confidence >= 0 AND confidence <= 1),
    source      TEXT NOT NULL DEFAULT 'user' CHECK (source IN ('user','agent','system','imported')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (topic_id, alias)
);

CREATE INDEX IF NOT EXISTS idx_np_claw_topic_aliases_topic_id
    ON np_claw_topic_aliases (topic_id);

CREATE INDEX IF NOT EXISTS idx_np_claw_topic_aliases_user_id
    ON np_claw_topic_aliases (user_id);

-- HNSW index on embedding for fast cosine search. Skipped when pgvector lacks
-- HNSW support; falls back to ivfflat which is still fast enough at this scale.
DO $$ BEGIN
    BEGIN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_np_claw_topic_aliases_embedding_hnsw
            ON np_claw_topic_aliases USING hnsw (embedding vector_cosine_ops)';
    EXCEPTION WHEN OTHERS THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_np_claw_topic_aliases_embedding_ivf
            ON np_claw_topic_aliases USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100)';
    END;
END $$;

ALTER TABLE np_claw_topic_aliases ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies
        WHERE tablename = 'np_claw_topic_aliases' AND policyname = 'topic_aliases_user_isolation'
    ) THEN
        CREATE POLICY topic_aliases_user_isolation ON np_claw_topic_aliases
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;
