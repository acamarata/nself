-- nself-claw: action dispatch schema (T-0967)
-- Idempotent — IF NOT EXISTS throughout

CREATE TABLE IF NOT EXISTS np_claw_actions (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID,
    device_id       UUID         NOT NULL,
    user_id         TEXT         NOT NULL,
    action_type     TEXT         NOT NULL,
    params          JSONB        NOT NULL DEFAULT '{}',
    status          TEXT         NOT NULL DEFAULT 'pending',
    result          JSONB,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    dispatched_at   TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ  NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

CREATE INDEX IF NOT EXISTS idx_claw_actions_user_status
    ON np_claw_actions(user_id, status);

CREATE INDEX IF NOT EXISTS idx_claw_actions_device
    ON np_claw_actions(device_id);

CREATE INDEX IF NOT EXISTS idx_claw_actions_expires
    ON np_claw_actions(expires_at) WHERE status IN ('pending', 'dispatched');
