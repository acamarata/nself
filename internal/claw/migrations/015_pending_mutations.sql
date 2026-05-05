-- T-2505: DB-backed pending mutation approval for admin NL→GraphQL pipeline.
--
-- Mutations detected by the admin query pipeline are cached here with state='pending'.
-- The owner approves or rejects via Telegram inline keyboard.
-- Expired entries are cleaned up by the cache_pending_mutation function.

CREATE TABLE IF NOT EXISTS np_claw_pending_mutations (
    hash        TEXT        PRIMARY KEY,
    session_id  UUID        NOT NULL,
    mutation    TEXT        NOT NULL,
    variables   JSONB,
    description TEXT        NOT NULL DEFAULT '',
    state       TEXT        NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    CONSTRAINT np_claw_pending_mutations_state_check
        CHECK (state IN ('pending', 'approved', 'rejected', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_np_claw_pending_mutations_state
    ON np_claw_pending_mutations(state, created_at) WHERE state = 'pending';
