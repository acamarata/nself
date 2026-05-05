-- P87 Sprint 01, Ticket 01.T01: KG Edges + Predicates Tables
-- Spec §4.1 Typed Relationship Edges

-- UP

-- Predicate vocabulary
CREATE TABLE IF NOT EXISTS np_claw_kg_predicates (
    predicate       TEXT PRIMARY KEY,
    inverse_of      TEXT REFERENCES np_claw_kg_predicates(predicate),
    category        TEXT NOT NULL CHECK (category IN ('social','spatial','temporal','org','causal','possession','reference')),
    is_symmetric    BOOLEAN NOT NULL DEFAULT false,
    transitive      BOOLEAN NOT NULL DEFAULT false,
    description     TEXT
);

-- Knowledge-graph edges
CREATE TABLE IF NOT EXISTS np_claw_kg_edges (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL,
    subject_id          UUID NOT NULL REFERENCES np_claw_kg_entities(id) ON DELETE CASCADE,
    predicate           TEXT NOT NULL,
    object_id           UUID REFERENCES np_claw_kg_entities(id) ON DELETE CASCADE,
    object_literal      TEXT,
    confidence          REAL NOT NULL DEFAULT 0.7 CHECK (confidence >= 0 AND confidence <= 1),
    source_type         TEXT NOT NULL DEFAULT 'stated' CHECK (source_type IN ('stated','inferred','assumed','user_confirmed')),
    source_memory_id    UUID REFERENCES np_claw_memories(id) ON DELETE SET NULL,
    source_message_id   UUID REFERENCES np_claw_messages(id) ON DELETE SET NULL,
    valid_from          TIMESTAMPTZ NOT NULL DEFAULT now(),
    valid_to            TIMESTAMPTZ,
    superseded_by       UUID REFERENCES np_claw_kg_edges(id) ON DELETE SET NULL,
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_kg_edges_object CHECK (object_id IS NOT NULL OR object_literal IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_kg_edges_subject   ON np_claw_kg_edges(user_id, subject_id, predicate);
CREATE INDEX IF NOT EXISTS idx_kg_edges_object    ON np_claw_kg_edges(user_id, object_id, predicate);
CREATE INDEX IF NOT EXISTS idx_kg_edges_predicate ON np_claw_kg_edges(user_id, predicate);
CREATE INDEX IF NOT EXISTS idx_kg_edges_valid     ON np_claw_kg_edges(user_id) WHERE valid_to IS NULL;

-- RLS
ALTER TABLE np_claw_kg_edges ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_kg_edges' AND policyname = 'kg_edges_user_isolation') THEN
        CREATE POLICY kg_edges_user_isolation ON np_claw_kg_edges
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

ALTER TABLE np_claw_kg_predicates ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_kg_predicates' AND policyname = 'kg_predicates_read_all') THEN
        CREATE POLICY kg_predicates_read_all ON np_claw_kg_predicates FOR SELECT USING (true);
    END IF;
END $$;

-- Seed 20 predicates
INSERT INTO np_claw_kg_predicates (predicate, inverse_of, category, is_symmetric, transitive, description)
VALUES
    ('works_at',       NULL,           'org',        false, false, 'Subject is employed at object'),
    ('lives_in',       NULL,           'spatial',    false, false, 'Subject resides in object location'),
    ('friend_of',      NULL,           'social',     true,  false, 'Subjects are friends'),
    ('married_to',     NULL,           'social',     true,  false, 'Subjects are married'),
    ('parent_of',      'child_of',     'social',     false, false, 'Subject is parent of object'),
    ('child_of',       'parent_of',    'social',     false, false, 'Subject is child of object'),
    ('authored',       NULL,           'reference',  false, false, 'Subject authored the object'),
    ('mentioned_in',   NULL,           'reference',  false, false, 'Subject was mentioned in object'),
    ('depends_on',     NULL,           'causal',     false, true,  'Subject depends on object'),
    ('before',         'after',        'temporal',   false, false, 'Subject occurred before object'),
    ('after',          'before',       'temporal',   false, false, 'Subject occurred after object'),
    ('owns',           NULL,           'possession', false, false, 'Subject owns object'),
    ('uses',           NULL,           'possession', false, false, 'Subject uses object'),
    ('part_of',        NULL,           'org',        false, true,  'Subject is part of object'),
    ('member_of',      NULL,           'org',        false, false, 'Subject is member of object'),
    ('located_in',     NULL,           'spatial',    false, true,  'Subject is located within object'),
    ('related_to',     NULL,           'reference',  true,  false, 'Subjects are related'),
    ('blocks',         NULL,           'causal',     false, false, 'Subject blocks object'),
    ('funds',          NULL,           'possession', false, false, 'Subject funds object'),
    ('supersedes',     NULL,           'temporal',   false, false, 'Subject supersedes object')
ON CONFLICT (predicate) DO NOTHING;

-- Set inverse_of for symmetric pairs that reference each other
UPDATE np_claw_kg_predicates SET inverse_of = 'after'     WHERE predicate = 'before'    AND inverse_of IS NULL;
UPDATE np_claw_kg_predicates SET inverse_of = 'before'    WHERE predicate = 'after'     AND inverse_of IS NULL;
UPDATE np_claw_kg_predicates SET inverse_of = 'child_of'  WHERE predicate = 'parent_of' AND inverse_of IS NULL;
UPDATE np_claw_kg_predicates SET inverse_of = 'parent_of' WHERE predicate = 'child_of'  AND inverse_of IS NULL;

-- DOWN
-- DROP POLICY IF EXISTS kg_edges_user_isolation ON np_claw_kg_edges;
-- DROP POLICY IF EXISTS kg_predicates_read_all ON np_claw_kg_predicates;
-- DROP TABLE IF EXISTS np_claw_kg_edges;
-- DROP TABLE IF EXISTS np_claw_kg_predicates;
