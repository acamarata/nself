-- P87 Sprint 01, Ticket 01.T07: Briefing Config + Briefings Tables
-- Spec §4.7

-- UP

CREATE TABLE IF NOT EXISTS np_claw_briefing_config (
    user_id     UUID PRIMARY KEY,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    local_time  TIME NOT NULL DEFAULT '07:00',
    timezone    TEXT NOT NULL DEFAULT 'UTC',
    channels    TEXT[] NOT NULL DEFAULT ARRAY['push'],
    sections    TEXT[] NOT NULL DEFAULT ARRAY['urgent','follow_ups','yesterday_decisions','calendar','overnight_email'],
    min_items   INT NOT NULL DEFAULT 3,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE np_claw_briefing_config ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_briefing_config' AND policyname = 'briefing_config_user_isolation') THEN
        CREATE POLICY briefing_config_user_isolation ON np_claw_briefing_config
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS np_claw_briefings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    for_date        DATE NOT NULL,
    body_md         TEXT NOT NULL,
    body_json       JSONB NOT NULL,
    token_cost      INT,
    delivered_at    TIMESTAMPTZ,
    delivery_status TEXT NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_briefings_user_date UNIQUE (user_id, for_date)
);

ALTER TABLE np_claw_briefings ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_claw_briefings' AND policyname = 'briefings_user_isolation') THEN
        CREATE POLICY briefings_user_isolation ON np_claw_briefings
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- DOWN
-- DROP POLICY IF EXISTS briefings_user_isolation ON np_claw_briefings;
-- DROP TABLE IF EXISTS np_claw_briefings;
-- DROP POLICY IF EXISTS briefing_config_user_isolation ON np_claw_briefing_config;
-- DROP TABLE IF EXISTS np_claw_briefing_config;
