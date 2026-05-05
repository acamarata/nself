-- P88 Sprint 16: Integration probes table for daily health dashboard.
-- Reversible: DROP TABLE np_claw_integration_probes;

CREATE TABLE IF NOT EXISTS np_claw_integration_probes (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        TEXT        NOT NULL,
    tool_name      TEXT        NOT NULL,
    status         TEXT        NOT NULL CHECK (status IN ('pass', 'fail', 'skipped')),
    latency_ms     BIGINT      NOT NULL DEFAULT 0,
    error          TEXT,
    payload_sample JSONB,
    probed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claw_int_probes_user_tool_time
    ON np_claw_integration_probes (user_id, tool_name, probed_at DESC);

-- RLS: users can only see their own probe results
ALTER TABLE np_claw_integration_probes ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_claw_int_probes_user_policy ON np_claw_integration_probes
    FOR ALL
    USING (user_id = current_setting('hasura.user', true))
    WITH CHECK (user_id = current_setting('hasura.user', true));

-- User notifications table (used for regression banners and other alerts)
CREATE TABLE IF NOT EXISTS np_claw_user_notifications (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT        NOT NULL,
    type       TEXT        NOT NULL,
    title      TEXT        NOT NULL,
    body       TEXT,
    metadata   JSONB,
    read       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claw_user_notif_user
    ON np_claw_user_notifications (user_id, created_at DESC);

ALTER TABLE np_claw_user_notifications ENABLE ROW LEVEL SECURITY;

CREATE POLICY np_claw_user_notif_policy ON np_claw_user_notifications
    FOR ALL
    USING (user_id = current_setting('hasura.user', true))
    WITH CHECK (user_id = current_setting('hasura.user', true));
