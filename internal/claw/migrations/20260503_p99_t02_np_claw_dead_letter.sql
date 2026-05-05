-- P99 T02 (PLUG-02): Dead Letter Queue for phase2 parse failures.
-- Messages that fail JSON parsing after all in-process retries are persisted
-- here so they can be inspected, replayed, or resolved by an admin.

CREATE TABLE IF NOT EXISTS np_claw_dead_letter (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    message_id      TEXT NOT NULL,
    thread_id       TEXT,
    payload         JSONB NOT NULL,          -- original message content JSON
    error_detail    TEXT NOT NULL,           -- last parse error string
    reason          TEXT NOT NULL DEFAULT 'parse_error', -- error classification
    attempt_count   INT  NOT NULL DEFAULT 1,
    next_retry_at   TIMESTAMPTZ,             -- NULL means exhausted (>= 5 attempts)
    resolved_at     TIMESTAMPTZ,             -- NULL means still pending
    resolved_by     TEXT,                    -- admin user or system that resolved
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for the background retry runner: find pending rows ready to retry.
CREATE INDEX IF NOT EXISTS np_claw_dead_letter_retry_idx
    ON np_claw_dead_letter (next_retry_at)
    WHERE resolved_at IS NULL AND next_retry_at IS NOT NULL;

-- Index for admin list endpoint: filter by user and pending state.
CREATE INDEX IF NOT EXISTS np_claw_dead_letter_user_pending_idx
    ON np_claw_dead_letter (user_id, created_at DESC)
    WHERE resolved_at IS NULL;

-- Keep updated_at current automatically.
CREATE OR REPLACE FUNCTION np_claw_dead_letter_set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS np_claw_dead_letter_updated_at ON np_claw_dead_letter;
CREATE TRIGGER np_claw_dead_letter_updated_at
    BEFORE UPDATE ON np_claw_dead_letter
    FOR EACH ROW EXECUTE FUNCTION np_claw_dead_letter_set_updated_at();
