-- P87 Sprint 01, Ticket 01.T08: Touchpoints Table
-- Spec §4.8

-- UP

CREATE TABLE IF NOT EXISTS np_claw_touchpoints (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL,
    entity_id               UUID NOT NULL REFERENCES np_claw_kg_entities(id) ON DELETE CASCADE,
    last_interaction_at     TIMESTAMPTZ NOT NULL,
    cadence_days            INT NOT NULL DEFAULT 60,
    snooze_until            TIMESTAMPTZ,
    muted                   BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT uq_touchpoints_user_entity UNIQUE (user_id, entity_id)
);

ALTER TABLE np_claw_touchpoints ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_touchpoints' AND policyname = 'touchpoints_user_isolation') THEN
        CREATE POLICY touchpoints_user_isolation ON np_claw_touchpoints
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- DOWN
-- DROP POLICY IF EXISTS touchpoints_user_isolation ON np_claw_touchpoints;
-- DROP TABLE IF EXISTS np_claw_touchpoints;
