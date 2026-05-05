-- P87 Sprint 01, Ticket 01.T00: KG Entities Table
-- Semantic-graph typed entities referenced by kg_edges, touchpoints, prune_candidates.
-- Distinct from np_claw_knowledge_nodes (chunked knowledge) — this table holds
-- canonical entities (people, places, orgs, concepts) for the knowledge-graph edges.
-- Schema aligned with graphify.go (uses `name` column for lookup).

-- UP

CREATE TABLE IF NOT EXISTS np_claw_kg_entities (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    name        TEXT NOT NULL,
    entity_type TEXT NOT NULL DEFAULT 'entity' CHECK (entity_type IN ('person','place','org','concept','event','entity')),
    aliases     TEXT[] NOT NULL DEFAULT '{}',
    properties  JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_kg_entities_user_name UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_kg_entities_user_name ON np_claw_kg_entities(user_id, name);
CREATE INDEX IF NOT EXISTS idx_kg_entities_user_type ON np_claw_kg_entities(user_id, entity_type);

ALTER TABLE np_claw_kg_entities ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_kg_entities' AND policyname = 'kg_entities_user_isolation') THEN
        CREATE POLICY kg_entities_user_isolation ON np_claw_kg_entities
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- DOWN
-- DROP POLICY IF EXISTS kg_entities_user_isolation ON np_claw_kg_entities;
-- DROP TABLE IF EXISTS np_claw_kg_entities;
