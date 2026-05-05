-- Migration: memory history (versioning) + memory links (cross-entity correlation)
-- T-7603: np_claw_memory_history — auto-versioned edit log
-- T-7604: np_claw_memory_links  — cosine-similarity cross-entity links
-- T-7601: consolidated_from column on np_claw_memories

-- T-7601: track which memories were merged into this one
ALTER TABLE np_claw_memories
  ADD COLUMN IF NOT EXISTS consolidated_from JSONB NOT NULL DEFAULT '[]';

-- T-7603: memory version history — records content before each update
CREATE TABLE IF NOT EXISTS np_claw_memory_history (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id      UUID        NOT NULL REFERENCES np_claw_memories(id) ON DELETE CASCADE,
    version        INTEGER     NOT NULL,
    content_before TEXT        NOT NULL,
    edited_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_memory_history_memory_id
    ON np_claw_memory_history (memory_id, version DESC);

-- T-7604: memory link table — cosine-similarity cross-entity correlations
CREATE TABLE IF NOT EXISTS np_claw_memory_links (
    id         UUID  PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_a   UUID  NOT NULL REFERENCES np_claw_memories(id) ON DELETE CASCADE,
    memory_b   UUID  NOT NULL REFERENCES np_claw_memories(id) ON DELETE CASCADE,
    similarity REAL  NOT NULL,
    link_type  TEXT  NOT NULL DEFAULT 'similar',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (memory_a, memory_b)
);

CREATE INDEX IF NOT EXISTS idx_np_claw_memory_links_a ON np_claw_memory_links (memory_a);
CREATE INDEX IF NOT EXISTS idx_np_claw_memory_links_b ON np_claw_memory_links (memory_b);

-- T-7602: add scoring columns to np_claw_memories for smart pruning
ALTER TABLE np_claw_memories
  ADD COLUMN IF NOT EXISTS access_count     INTEGER   NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS last_accessed_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS confidence       FLOAT     NOT NULL DEFAULT 1.0;
