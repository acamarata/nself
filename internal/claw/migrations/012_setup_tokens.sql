-- T-1511: Ensure setup_tokens table exists (may be missing if auth::migrate() errored silently)
CREATE SCHEMA IF NOT EXISTS np_claw;

CREATE TABLE IF NOT EXISTS np_claw.setup_tokens (
    token       TEXT        PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 minutes',
    used_at     TIMESTAMPTZ
);
