-- P88 Sprint 35, H49-11: Budget test columns + notify prefs table

-- UP

-- Test budget columns on np_ai_budget
ALTER TABLE np_ai_budget ADD COLUMN IF NOT EXISTS is_test BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE np_ai_budget ADD COLUMN IF NOT EXISTS test_expires_at TIMESTAMPTZ;

-- Auto-expire test budgets via index (for cleanup cron)
CREATE INDEX IF NOT EXISTS idx_ai_budget_test_expiry
    ON np_ai_budget (test_expires_at) WHERE is_test = true;

-- H49-10: Notification preferences table
CREATE TABLE IF NOT EXISTS np_ai_notify_pref (
    user_id           UUID PRIMARY KEY,
    z_threshold       NUMERIC(4,2) NOT NULL DEFAULT 3.0,
    min_usd_floor     NUMERIC(8,2) NOT NULL DEFAULT 1.00,
    channels          JSONB NOT NULL DEFAULT '{"email":{"low":true,"medium":true,"high":true},"push":{"low":false,"medium":true,"high":true},"telegram":{"low":false,"medium":false,"high":true},"in_app":{"low":true,"medium":true,"high":true}}'::jsonb,
    muted_purposes    TEXT[] NOT NULL DEFAULT '{}',
    quiet_hours_start TIME NOT NULL DEFAULT '22:00',
    quiet_hours_end   TIME NOT NULL DEFAULT '07:00',
    tz                TEXT NOT NULL DEFAULT 'UTC',
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE np_ai_notify_pref ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'np_ai_notify_pref' AND policyname = 'notify_pref_user_isolation') THEN
        CREATE POLICY notify_pref_user_isolation ON np_ai_notify_pref
            USING (user_id = current_setting('hasura.user', true)::uuid)
            WITH CHECK (user_id = current_setting('hasura.user', true)::uuid);
    END IF;
END $$;

-- H49-11: Cost override audit table
CREATE TABLE IF NOT EXISTS np_ai_cost_override_audit (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    budget_id   UUID NOT NULL,
    user_id     UUID NOT NULL,
    token       TEXT NOT NULL,
    used_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cost_usd    NUMERIC(10,4)
);

-- Notify jobs table (if not already created by notify plugin)
CREATE TABLE IF NOT EXISTS np_notify_jobs (
    id                  BIGSERIAL PRIMARY KEY,
    job_type            TEXT NOT NULL,
    user_id             UUID NOT NULL,
    payload             JSONB NOT NULL,
    bypass_quiet_hours  BOOLEAN NOT NULL DEFAULT false,
    status              TEXT NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_notify_jobs_pending ON np_notify_jobs (status, created_at) WHERE status = 'pending';

-- H49-18: Add percentile columns to task class stats
ALTER TABLE np_ai_task_class_stats ADD COLUMN IF NOT EXISTS percentile_25_cost NUMERIC(10,6) DEFAULT 0;
ALTER TABLE np_ai_task_class_stats ADD COLUMN IF NOT EXISTS percentile_50_cost NUMERIC(10,6) DEFAULT 0;
ALTER TABLE np_ai_task_class_stats ADD COLUMN IF NOT EXISTS percentile_75_cost NUMERIC(10,6) DEFAULT 0;

-- DOWN
ALTER TABLE np_ai_budget DROP COLUMN IF EXISTS is_test;
ALTER TABLE np_ai_budget DROP COLUMN IF EXISTS test_expires_at;
DROP TABLE IF EXISTS np_ai_notify_pref;
DROP TABLE IF EXISTS np_ai_cost_override_audit;
-- np_notify_jobs intentionally not dropped (owned by notify plugin)
