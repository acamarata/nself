-- G2-T04: persistent FCM/push retry queue
--
-- On FCM/APNs/ntfy timeout the dispatcher writes a row here. A cron worker
-- drains due rows every 30s, retries the push, and marks success/failure.

CREATE TABLE IF NOT EXISTS np_claw_push_retry_queue (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    push_token      TEXT NOT NULL,
    platform        TEXT NOT NULL,
    title           TEXT NOT NULL,
    body            TEXT NOT NULL,
    attempts        INT NOT NULL DEFAULT 0,
    max_attempts    INT NOT NULL DEFAULT 5,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error      TEXT,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending','running','done','dead')),
    locked_until    TIMESTAMPTZ,
    locked_by       TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_claw_push_retry_due
    ON np_claw_push_retry_queue (status, next_attempt_at)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_claw_push_retry_user
    ON np_claw_push_retry_queue (user_id);

COMMENT ON TABLE np_claw_push_retry_queue IS
    'G2-T04: persistent retry queue for FCM/APNs/ntfy push notifications that timed out.';
