-- nself-claw: initial schema (T-0205)
-- Idempotent — IF NOT EXISTS throughout

CREATE TABLE IF NOT EXISTS np_claw_memory (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    embedding   vector(1536),               -- pgvector for semantic search (requires pgvector)
    tags        TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, key)
);

CREATE INDEX IF NOT EXISTS idx_np_claw_memory_user_id
    ON np_claw_memory(user_id);

CREATE TABLE IF NOT EXISTS np_claw_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL,
    name        TEXT,
    context     JSONB NOT NULL DEFAULT '[]', -- message history
    tool_calls  JSONB NOT NULL DEFAULT '[]', -- tool call log
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_sessions_user_id
    ON np_claw_sessions(user_id);

CREATE TABLE IF NOT EXISTS np_claw_tools (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT,
    schema      JSONB NOT NULL,             -- OpenAI function schema
    endpoint    TEXT NOT NULL,              -- URL to call
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    requires_auth BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
