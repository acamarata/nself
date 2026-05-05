-- Sprint 07 / N4: Shared conversations.
-- A read-only public token pointing at a session; anyone with the URL can
-- read the messages until the share expires.

CREATE TABLE IF NOT EXISTS np_claw_shared_sessions (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id   UUID         NOT NULL REFERENCES np_claw_sessions(id) ON DELETE CASCADE,
    user_id      TEXT         NOT NULL,
    token        TEXT         NOT NULL UNIQUE,
    ttl_seconds  INTEGER      NOT NULL DEFAULT 86400,
    expires_at   TIMESTAMPTZ  NOT NULL,
    view_count   INTEGER      NOT NULL DEFAULT 0,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_shared_sessions_token
    ON np_claw_shared_sessions(token);
CREATE INDEX IF NOT EXISTS idx_np_claw_shared_sessions_session
    ON np_claw_shared_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_shared_sessions_user
    ON np_claw_shared_sessions(user_id);
