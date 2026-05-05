-- P87 Sprint 08: Voice calls table for streaming STT sessions.
CREATE TABLE IF NOT EXISTS np_claw_voice_calls (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id         TEXT NOT NULL,
    session_id      TEXT,
    mode            TEXT NOT NULL DEFAULT 'tap_to_talk',
    provider        TEXT NOT NULL DEFAULT 'deepgram',
    language        TEXT NOT NULL DEFAULT 'en',
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    transcript      TEXT DEFAULT '',
    message_id      TEXT,
    cost_cents      INTEGER NOT NULL DEFAULT 0,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_voice_calls_user ON np_claw_voice_calls(user_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_voice_calls_session ON np_claw_voice_calls(session_id);
CREATE INDEX IF NOT EXISTS idx_np_claw_voice_calls_started ON np_claw_voice_calls(started_at);
