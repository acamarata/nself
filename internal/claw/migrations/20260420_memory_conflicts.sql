-- P87 Sprint 01, Ticket 01.T03: Memory Conflicts Table
-- Spec §4.3

-- UP

CREATE TABLE IF NOT EXISTS np_claw_memory_conflicts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    memory_a_id     UUID NOT NULL REFERENCES np_claw_memories(id) ON DELETE CASCADE,
    memory_b_id     UUID NOT NULL REFERENCES np_claw_memories(id) ON DELETE CASCADE,
    similarity      REAL NOT NULL,
    reason          TEXT NOT NULL CHECK (reason IN ('opposite_predicate','contradictory_value','temporal_conflict')),
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','resolved','dismissed')),
    resolution      TEXT CHECK (resolution IN ('keep_a','keep_b','both_true','merge','neither')),
    resolved_by     UUID,
    resolved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_conflicts_pending
    ON np_claw_memory_conflicts(user_id)
    WHERE status = 'pending';

ALTER TABLE np_claw_memory_conflicts ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_memory_conflicts' AND policyname = 'memory_conflicts_user_isolation') THEN
        CREATE POLICY memory_conflicts_user_isolation ON np_claw_memory_conflicts
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- DOWN
-- DROP POLICY IF EXISTS memory_conflicts_user_isolation ON np_claw_memory_conflicts;
-- DROP TABLE IF EXISTS np_claw_memory_conflicts;
