-- P87 Sprint 01, Ticket 01.T11: Prune Candidates Table
-- Spec §4.11

-- UP

CREATE TABLE IF NOT EXISTS np_claw_prune_candidates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL,
    memory_id   UUID REFERENCES np_claw_memories(id) ON DELETE CASCADE,
    edge_id     UUID REFERENCES np_claw_kg_edges(id) ON DELETE CASCADE,
    reason      TEXT NOT NULL CHECK (reason IN ('low_confidence','superseded','stale','contradicted','user_rejected')),
    score       REAL NOT NULL,
    status      TEXT NOT NULL DEFAULT 'pending',
    decided_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_prune_exactly_one CHECK (((memory_id IS NOT NULL)::int + (edge_id IS NOT NULL)::int) = 1)
);

ALTER TABLE np_claw_prune_candidates ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_prune_candidates' AND policyname = 'prune_candidates_user_isolation') THEN
        CREATE POLICY prune_candidates_user_isolation ON np_claw_prune_candidates
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- DOWN
-- DROP POLICY IF EXISTS prune_candidates_user_isolation ON np_claw_prune_candidates;
-- DROP TABLE IF EXISTS np_claw_prune_candidates;
