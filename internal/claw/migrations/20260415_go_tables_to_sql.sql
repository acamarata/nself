-- Migration: Move Go-created tables to SQL migrations
-- Previously created at runtime by Go code in:
--   internal/auth/auth.go (np_claw.users, passkeys, sessions, setup_tokens, push_subscriptions, webauthn_state)
--   internal/knowledge/graph.go (np_claw_knowledge_nodes, np_claw_knowledge_edges)
--   internal/features/knowledge_memory.go (np_claw_memory_rooms)
-- All statements are idempotent (IF NOT EXISTS).

-- Schema
CREATE SCHEMA IF NOT EXISTS np_claw;

-- ============================================================================
-- np_claw.users
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_claw.users (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name        TEXT        NOT NULL,
    email               TEXT,
    password_hash       TEXT,
    e2e_enabled         BOOLEAN     NOT NULL DEFAULT false,
    e2e_salt            TEXT,
    e2e_key_hint        TEXT,
    recovery_key_hash   TEXT,
    timezone            TEXT        NOT NULL DEFAULT 'UTC',
    role                TEXT        NOT NULL DEFAULT 'owner'
                        CHECK (role IN ('owner', 'member', 'viewer')),
    invited_by          UUID        REFERENCES np_claw.users(id),
    invite_accepted_at  TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ
);

-- ============================================================================
-- np_claw.passkeys
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_claw.passkeys (
    id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID    NOT NULL REFERENCES np_claw.users(id) ON DELETE CASCADE,
    credential_id   TEXT    NOT NULL UNIQUE,
    public_key      BYTEA   NOT NULL,
    sign_count      BIGINT  NOT NULL DEFAULT 0,
    device_name     TEXT,
    aaguid          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ
);

-- ============================================================================
-- np_claw.sessions
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_claw.sessions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES np_claw.users(id) ON DELETE CASCADE,
    token_hash      TEXT        NOT NULL UNIQUE,
    device_info     JSONB       NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_np_claw_sessions_user ON np_claw.sessions(user_id) WHERE revoked_at IS NULL;

-- ============================================================================
-- np_claw.setup_tokens
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_claw.setup_tokens (
    token       TEXT        PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 minutes',
    used_at     TIMESTAMPTZ
);

-- ============================================================================
-- np_claw.push_subscriptions
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_claw.push_subscriptions (
    id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID    NOT NULL REFERENCES np_claw.users(id) ON DELETE CASCADE,
    endpoint        TEXT    NOT NULL UNIQUE,
    keys            JSONB   NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ
);

-- ============================================================================
-- np_claw.webauthn_state
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_claw.webauthn_state (
    session_id  TEXT        PRIMARY KEY,
    state_data  BYTEA       NOT NULL,
    user_id     UUID        REFERENCES np_claw.users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '5 minutes'
);
CREATE INDEX IF NOT EXISTS idx_np_claw_webauthn_state_expires ON np_claw.webauthn_state(expires_at);

-- ============================================================================
-- np_claw_knowledge_nodes (knowledge graph)
-- ============================================================================
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS np_claw_knowledge_nodes (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL,
    label         TEXT        NOT NULL,
    node_type     TEXT        NOT NULL DEFAULT 'concept',
    content       TEXT        NOT NULL DEFAULT '',
    metadata      JSONB       NOT NULL DEFAULT '{}',
    embedding     vector(1536),
    mention_count INT         NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_claw_knowledge_edges (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID        NOT NULL,
    source_node_id UUID        NOT NULL REFERENCES np_claw_knowledge_nodes(id) ON DELETE CASCADE,
    target_node_id UUID        NOT NULL REFERENCES np_claw_knowledge_nodes(id) ON DELETE CASCADE,
    relation       TEXT        NOT NULL DEFAULT 'related_to',
    weight         FLOAT       NOT NULL DEFAULT 1.0,
    metadata       JSONB       NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- np_claw_memory_rooms
-- ============================================================================
CREATE TABLE IF NOT EXISTS np_claw_memory_rooms (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    color       TEXT        NOT NULL DEFAULT '#6366F1',
    icon        TEXT        NOT NULL DEFAULT 'folder',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_memory_rooms_user ON np_claw_memory_rooms (user_id);

-- Add nullable room_id FK to existing memories table (best-effort).
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS room_id UUID REFERENCES np_claw_memory_rooms(id) ON DELETE SET NULL;
