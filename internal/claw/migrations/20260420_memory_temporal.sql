-- P87 Sprint 01, Ticket 01.T02: Temporal Columns on np_claw_memories
-- Spec §4.2

-- UP

ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS valid_from      TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS valid_to        TIMESTAMPTZ;
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS superseded_by   UUID REFERENCES np_claw_memories(id) ON DELETE SET NULL;
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS confidence      REAL NOT NULL DEFAULT 0.7 CHECK (confidence >= 0 AND confidence <= 1);
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS source_type     TEXT NOT NULL DEFAULT 'stated';
ALTER TABLE np_claw_memories ADD COLUMN IF NOT EXISTS archived_at     TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_memory_valid_current
    ON np_claw_memories(user_id, topic_id)
    WHERE valid_to IS NULL AND archived_at IS NULL;

-- btree_gist extension needed for GIST index on tstzrange
CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE INDEX IF NOT EXISTS idx_memory_valid_range
    ON np_claw_memories USING GIST (tstzrange(valid_from, valid_to));

-- DOWN
-- DROP INDEX IF EXISTS idx_memory_valid_range;
-- DROP INDEX IF EXISTS idx_memory_valid_current;
-- ALTER TABLE np_claw_memories DROP COLUMN IF EXISTS archived_at;
-- ALTER TABLE np_claw_memories DROP COLUMN IF EXISTS source_type;
-- ALTER TABLE np_claw_memories DROP COLUMN IF EXISTS confidence;
-- ALTER TABLE np_claw_memories DROP COLUMN IF EXISTS superseded_by;
-- ALTER TABLE np_claw_memories DROP COLUMN IF EXISTS valid_to;
-- ALTER TABLE np_claw_memories DROP COLUMN IF EXISTS valid_from;
