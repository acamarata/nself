-- Phase 81: Conversation Intelligence — Sprint 01: Topic Schema
-- All tables for the topic-based conversation organization system.

-- Extensions (idempotent)
CREATE EXTENSION IF NOT EXISTS ltree;
CREATE EXTENSION IF NOT EXISTS vector;

-- 2.1 Topics
CREATE TABLE IF NOT EXISTS np_claw_topics (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    parent_id       UUID        REFERENCES np_claw_topics(id) ON DELETE SET NULL,
    path            LTREE       NOT NULL,
    name            TEXT        NOT NULL,
    slug            TEXT        NOT NULL,
    description     TEXT,
    icon            TEXT,
    color           TEXT,
    sort_order      INT         NOT NULL DEFAULT 0,

    -- Classification
    embedding       vector(1536),
    keywords        TEXT[]      NOT NULL DEFAULT '{}',

    -- Lifecycle
    auto_detected   BOOLEAN     NOT NULL DEFAULT true,
    is_pinned       BOOLEAN     NOT NULL DEFAULT false,
    is_archived     BOOLEAN     NOT NULL DEFAULT false,
    message_count   INT         NOT NULL DEFAULT 0,
    memory_count    INT         NOT NULL DEFAULT 0,

    -- Timestamps
    last_active_at  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, path)
);

CREATE INDEX IF NOT EXISTS idx_topics_user_active ON np_claw_topics(user_id, is_archived, last_active_at DESC);
CREATE INDEX IF NOT EXISTS idx_topics_user_path ON np_claw_topics USING GIST (path gist_ltree_ops);
CREATE INDEX IF NOT EXISTS idx_topics_user_parent ON np_claw_topics(user_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_topics_pinned ON np_claw_topics(user_id) WHERE is_pinned = true;

-- HNSW index on centroid embedding (skip silently if vector unavailable)
CREATE INDEX IF NOT EXISTS idx_topics_embedding ON np_claw_topics
    USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

-- Trigger: keep parent_id in sync with ltree path
CREATE OR REPLACE FUNCTION sync_topic_parent_from_path() RETURNS TRIGGER AS $$
BEGIN
    IF nlevel(NEW.path) > 1 THEN
        SELECT id INTO NEW.parent_id
        FROM np_claw_topics
        WHERE user_id = NEW.user_id
          AND path = subltree(NEW.path, 0, nlevel(NEW.path) - 1);
    ELSE
        NEW.parent_id := NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_topic_sync_parent ON np_claw_topics;
CREATE TRIGGER trg_topic_sync_parent
    BEFORE INSERT OR UPDATE OF path ON np_claw_topics
    FOR EACH ROW EXECUTE FUNCTION sync_topic_parent_from_path();

-- 2.2 Message-Topic Association
CREATE TABLE IF NOT EXISTS np_claw_message_topics (
    message_id      UUID        NOT NULL REFERENCES np_claw_messages(id) ON DELETE CASCADE,
    topic_id        UUID        NOT NULL REFERENCES np_claw_topics(id) ON DELETE CASCADE,
    is_primary      BOOLEAN     NOT NULL DEFAULT false,
    confidence      REAL        NOT NULL DEFAULT 0.8,
    classified_by   TEXT        NOT NULL DEFAULT 'embedding',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (message_id, topic_id)
);

CREATE INDEX IF NOT EXISTS idx_msg_topics_topic ON np_claw_message_topics(topic_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_msg_topics_message ON np_claw_message_topics(message_id);
CREATE INDEX IF NOT EXISTS idx_msg_topics_primary ON np_claw_message_topics(topic_id) WHERE is_primary = true;

-- 2.4 Topic Relations
CREATE TABLE IF NOT EXISTS np_claw_topic_relations (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    source_topic_id UUID        NOT NULL REFERENCES np_claw_topics(id) ON DELETE CASCADE,
    target_topic_id UUID        NOT NULL REFERENCES np_claw_topics(id) ON DELETE CASCADE,
    relation_type   TEXT        NOT NULL CHECK (relation_type IN (
        'related', 'merged_from', 'split_from', 'depends_on', 'bridges'
    )),
    weight          REAL        NOT NULL DEFAULT 1.0,
    auto_detected   BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (source_topic_id, target_topic_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_topic_rels_source ON np_claw_topic_relations(source_topic_id);
CREATE INDEX IF NOT EXISTS idx_topic_rels_target ON np_claw_topic_relations(target_topic_id);

-- 2.5 Topic Transitions
CREATE TABLE IF NOT EXISTS np_claw_topic_transitions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL,
    thread_id       UUID        NOT NULL,
    from_topic_id   UUID        REFERENCES np_claw_topics(id) ON DELETE SET NULL,
    to_topic_id     UUID        NOT NULL REFERENCES np_claw_topics(id) ON DELETE CASCADE,
    message_id      UUID        NOT NULL,
    transition_type TEXT        NOT NULL DEFAULT 'drift',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transitions_thread ON np_claw_topic_transitions(thread_id, created_at);
CREATE INDEX IF NOT EXISTS idx_transitions_user ON np_claw_topic_transitions(user_id, created_at DESC);

-- 2.6 Topic-Knowledge links
CREATE TABLE IF NOT EXISTS np_claw_topic_knowledge (
    topic_id        UUID        NOT NULL REFERENCES np_claw_topics(id) ON DELETE CASCADE,
    node_id         UUID        NOT NULL REFERENCES np_claw_knowledge_nodes(id) ON DELETE CASCADE,
    relevance       REAL        NOT NULL DEFAULT 1.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (topic_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_topic_knowledge_node ON np_claw_topic_knowledge(node_id);

-- 2.7 Topic Preferences
CREATE TABLE IF NOT EXISTS np_claw_topic_prefs (
    user_id         UUID        NOT NULL,
    topic_id        UUID        NOT NULL REFERENCES np_claw_topics(id) ON DELETE CASCADE,
    notify          BOOLEAN     NOT NULL DEFAULT true,
    auto_summarize  BOOLEAN     NOT NULL DEFAULT true,
    custom_prompt   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, topic_id)
);

-- 2.3 Topic-scoped memories (ALTER existing table)
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS topic_id UUID REFERENCES np_claw_topics(id) ON DELETE SET NULL;
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS is_global BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_memories_topic ON np_claw_memories(topic_id) WHERE topic_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_memories_global ON np_claw_memories(entity_id) WHERE is_global = true;

-- Seed "General" topic for every existing user
-- Use np_claw.users (UUID) instead of np_claw_memories.entity_id (TEXT) to avoid type mismatch
INSERT INTO np_claw_topics (user_id, path, name, slug, is_pinned, auto_detected)
SELECT DISTINCT id, 'general'::ltree, 'General', 'general', true, false
FROM np_claw.users
ON CONFLICT (user_id, path) DO NOTHING;
