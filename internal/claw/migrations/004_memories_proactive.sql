-- T-1185: np_claw_memories + np_claw_proactive_jobs + np_claw_user_preferences
-- All statements use IF NOT EXISTS / ON CONFLICT DO NOTHING for idempotency.

-- ─────────────────────────────────────────────
-- 1. Persistent user/entity memories
-- ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS np_claw_memories (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id        TEXT NOT NULL,
    entity_type      TEXT NOT NULL DEFAULT 'user',  -- user|service|project
    content          TEXT NOT NULL,
    confidence       REAL NOT NULL DEFAULT 1.0,
    times_reinforced INTEGER NOT NULL DEFAULT 1,
    source           TEXT NOT NULL DEFAULT 'extracted',  -- extracted|explicit
    context_hash     TEXT,  -- SHA-256 of source message (for dedup)
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_memories_entity_id
    ON np_claw_memories(entity_id);

CREATE INDEX IF NOT EXISTS idx_np_claw_memories_entity_type
    ON np_claw_memories(entity_type);

CREATE INDEX IF NOT EXISTS idx_np_claw_memories_confidence
    ON np_claw_memories(confidence DESC);

-- ─────────────────────────────────────────────
-- 2. Proactive background jobs
-- ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS np_claw_proactive_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type        TEXT NOT NULL,  -- morning_digest|health_report|ssl_check|disk_check|anomaly_detect
    enabled         BOOLEAN NOT NULL DEFAULT true,
    cron_expression TEXT NOT NULL,  -- e.g. "0 7 * * *" for 7am daily
    next_run_at     TIMESTAMPTZ,
    last_run_at     TIMESTAMPTZ,
    last_result     JSONB,
    failure_count   INTEGER NOT NULL DEFAULT 0,
    config          JSONB NOT NULL DEFAULT '{}',  -- job-specific config (thresholds, targets)
    quiet_hours_start INTEGER NOT NULL DEFAULT 22,  -- 10pm
    quiet_hours_end   INTEGER NOT NULL DEFAULT 7,   -- 7am
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────
-- 3. Per-user proactive feature preferences
-- ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS np_claw_user_preferences (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          TEXT NOT NULL UNIQUE,
    digest_enabled   BOOLEAN NOT NULL DEFAULT true,
    digest_time_hour INTEGER NOT NULL DEFAULT 7,
    health_enabled   BOOLEAN NOT NULL DEFAULT true,
    alerts_enabled   BOOLEAN NOT NULL DEFAULT true,
    quiet_hours_start INTEGER NOT NULL DEFAULT 22,
    quiet_hours_end   INTEGER NOT NULL DEFAULT 7,
    push_token       TEXT,
    tg_chat_id       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ─────────────────────────────────────────────
-- 4. Seed default proactive jobs
-- ─────────────────────────────────────────────

INSERT INTO np_claw_proactive_jobs (job_type, cron_expression, next_run_at, config)
VALUES
    ('morning_digest',  '0 7 * * *',    NOW() + INTERVAL '1 hour',    '{"ai_tier": "free"}'),
    ('health_report',   '0 9 * * 1',    NOW() + INTERVAL '1 day',     '{"include_ssl": true, "include_versions": true}'),
    ('ssl_check',       '0 8 * * *',    NOW() + INTERVAL '1 hour',    '{"warning_days": 30, "critical_days": 7}'),
    ('disk_check',      '*/30 * * * *', NOW() + INTERVAL '30 minutes','{"warning_pct": 80, "critical_pct": 90}'),
    ('anomaly_detect',  '*/5 * * * *',  NOW() + INTERVAL '5 minutes', '{"baseline_days": 7, "threshold_multiplier": 2.0}')
ON CONFLICT DO NOTHING;
