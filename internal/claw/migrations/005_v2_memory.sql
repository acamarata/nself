-- nself-claw v2 memory architecture (T-0810)
-- Persistent threads, messages, memory blocks, thread cores, and personas.
-- All statements use IF NOT EXISTS for idempotency.

-- ─────────────────────────────────────────────
-- 1. Threads
-- ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS cl_threads (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    namespace       VARCHAR(100) NOT NULL DEFAULT 'default',
    persona_name    VARCHAR(100) NOT NULL DEFAULT 'CamClaw',
    title           VARCHAR(500),
    parent_id       UUID        REFERENCES cl_threads(id),
    is_archived     BOOLEAN     NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cl_threads_user_namespace
    ON cl_threads(user_id, namespace);

-- ─────────────────────────────────────────────
-- 2. Messages
-- ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS cl_messages (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id       UUID        NOT NULL REFERENCES cl_threads(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content         TEXT        NOT NULL,
    embedding       vector(1536),
    token_count     INTEGER     NOT NULL DEFAULT 0,
    is_compressed   BOOLEAN     NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cl_messages_thread_id
    ON cl_messages(thread_id);

CREATE INDEX IF NOT EXISTS idx_cl_messages_embedding
    ON cl_messages USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ─────────────────────────────────────────────
-- 3. Memory blocks (compressed summaries with embeddings)
-- ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS cl_memory_blocks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id       UUID        NOT NULL REFERENCES cl_threads(id) ON DELETE CASCADE,
    content         TEXT        NOT NULL,
    embedding       vector(1536),
    source_msg_ids  UUID[]      NOT NULL DEFAULT '{}',
    token_count     INTEGER     NOT NULL DEFAULT 0,
    relevance_score DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cl_memory_blocks_thread
    ON cl_memory_blocks(thread_id);

CREATE INDEX IF NOT EXISTS idx_cl_memory_blocks_embedding
    ON cl_memory_blocks USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- ─────────────────────────────────────────────
-- 4. Thread cores (rolling summary per thread)
-- ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS cl_thread_cores (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id       UUID        UNIQUE NOT NULL REFERENCES cl_threads(id) ON DELETE CASCADE,
    summary         TEXT        NOT NULL DEFAULT '',
    embedding       vector(1536),
    token_count     INTEGER     NOT NULL DEFAULT 0,
    message_count   INTEGER     NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────
-- 5. Personas
-- ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS np_claw_personas (
    id                          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                        VARCHAR(100) UNIQUE NOT NULL,
    namespace                   VARCHAR(100) NOT NULL DEFAULT 'default',
    system_prompt               TEXT        NOT NULL,
    memory_mode                 VARCHAR(50) NOT NULL DEFAULT 'full'
                                    CHECK (memory_mode IN ('full', 'session_only', 'none')),
    audience_modes              JSONB       NOT NULL DEFAULT '{}',
    human_escalation_threshold  DOUBLE PRECISION NOT NULL DEFAULT 0.9,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
