-- P87 Sprint 03, Ticket 03.T03: Saved Searches Table
-- Spec §5.13

-- UP

CREATE TABLE IF NOT EXISTS np_claw_saved_searches (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT NOT NULL,
    name         TEXT,
    query        TEXT NOT NULL,
    filters      JSONB,
    notify_on_new BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_saved_searches_user ON np_claw_saved_searches(user_id, created_at DESC);

-- DOWN
-- DROP TABLE IF EXISTS np_claw_saved_searches;
