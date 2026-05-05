-- T-0601: np_claw_attachments table
-- T-0602: np_claw_media_embeddings table
-- T-0603: np_claw_media_events table

-- Ensure pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- ==========================================================================
-- np_claw_attachments
-- ==========================================================================
CREATE TABLE IF NOT EXISTS np_claw_attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    topic_id        UUID,
    message_id      UUID,
    kind            TEXT NOT NULL CHECK (kind IN ('image','pdf','audio','video','doc','screenshot')),
    mime_type       TEXT NOT NULL,
    original_name   TEXT NOT NULL,
    size_bytes      BIGINT NOT NULL,
    sha256          TEXT NOT NULL,
    storage_key     TEXT NOT NULL,
    variants        JSONB NOT NULL DEFAULT '{}',
    width           INT,
    height          INT,
    duration_ms     INT,
    page_count      INT,
    status          TEXT NOT NULL DEFAULT 'uploading'
                    CHECK (status IN ('uploading','processing','ready','error','quarantined')),
    error           TEXT,
    provider_used   TEXT,
    cost_cents      INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    ready_at        TIMESTAMPTZ,
    UNIQUE (user_id, sha256)
);

CREATE INDEX IF NOT EXISTS idx_claw_attachments_user_created
    ON np_claw_attachments (user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_claw_attachments_status_partial
    ON np_claw_attachments (status)
    WHERE status IN ('uploading', 'processing');

-- ==========================================================================
-- np_claw_media_embeddings
-- ==========================================================================
CREATE TABLE IF NOT EXISTS np_claw_media_embeddings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attachment_id   UUID NOT NULL REFERENCES np_claw_attachments(id) ON DELETE CASCADE,
    chunk_index     INT NOT NULL,
    chunk_kind      TEXT NOT NULL CHECK (chunk_kind IN ('description','ocr','page','frame','segment')),
    chunk_text      TEXT NOT NULL,
    embedding       vector(1536),
    meta            JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (attachment_id, chunk_index, chunk_kind)
);

CREATE INDEX IF NOT EXISTS idx_claw_media_embeddings_vector
    ON np_claw_media_embeddings
    USING hnsw (embedding vector_cosine_ops);

-- ==========================================================================
-- np_claw_media_events
-- ==========================================================================
CREATE TABLE IF NOT EXISTS np_claw_media_events (
    id              BIGSERIAL PRIMARY KEY,
    attachment_id   UUID REFERENCES np_claw_attachments(id) ON DELETE SET NULL,
    user_id         TEXT NOT NULL,
    event           TEXT NOT NULL CHECK (event IN ('uploaded','processed','failed','viewed','deleted')),
    provider        TEXT,
    bytes           BIGINT,
    duration_ms     INT,
    cost_cents      INT,
    meta            JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_claw_media_events_user_created
    ON np_claw_media_events (user_id, created_at DESC);
