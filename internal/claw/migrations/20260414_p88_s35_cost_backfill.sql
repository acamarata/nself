-- P88 Sprint 35, H49-04: Cost backfill migration
-- Resolves pricing for historical usage rows, recomputes rollups from scratch.
-- Run via `nself migrate up` in staging first; prod after 48h review.

-- UP

-- 1. Cursor + cron log tables (idempotent)
CREATE TABLE IF NOT EXISTS np_cost_rollup_cursor (
    job_name TEXT PRIMARY KEY,
    last_processed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS np_cron_run_log (
    id BIGSERIAL PRIMARY KEY,
    job_name TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    rows_processed BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'running'
);

CREATE INDEX IF NOT EXISTS idx_cron_run_log_job_name ON np_cron_run_log (job_name, finished_at DESC);

-- 2. Reconciliation log
CREATE TABLE IF NOT EXISTS np_ai_cost_reconciliation_log (
    id BIGSERIAL PRIMARY KEY,
    usage_log_id UUID NOT NULL,
    reason TEXT NOT NULL,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Backfill pricing for rows with cost_usd = 0 and no pricing_id
WITH resolved AS (
    SELECT u.id AS usage_id,
           p.id AS pricing_id,
           (COALESCE(u.input_tokens, 0) * p.input_per_mtok_usd / 1000000.0) +
           (COALESCE(u.output_tokens, 0) * p.output_per_mtok_usd / 1000000.0) AS computed_cost
    FROM np_ai_usage_log u
    JOIN np_ai_provider_pricing p
      ON p.provider = u.provider
     AND p.model = u.model
     AND u.created_at >= p.effective_date
     AND u.created_at < COALESCE(p.deprecated_at, 'infinity'::timestamptz)
    WHERE u.cost_usd = 0 AND u.pricing_id IS NULL
)
UPDATE np_ai_usage_log u
SET cost_usd = r.computed_cost,
    pricing_id = r.pricing_id
FROM resolved r
WHERE u.id = r.usage_id;

-- 4. Log unresolvable rows
INSERT INTO np_ai_cost_reconciliation_log (usage_log_id, reason)
SELECT id, 'pricing_missing'
FROM np_ai_usage_log
WHERE cost_usd = 0 AND pricing_id IS NULL
ON CONFLICT DO NOTHING;

-- 5. Clear rollup tables (reproducible from source)
TRUNCATE np_ai_user_daily_cost;
TRUNCATE np_ai_tenant_daily_cost;

-- 6. Reset rollup cursors so Go rollup rebuilds from the beginning
DELETE FROM np_cost_rollup_cursor WHERE job_name IN ('ai_cost_rollup_user', 'ai_cost_rollup_tenant');

-- Note: After this migration, run the Go rollup crons to rebuild daily cost tables.
-- The rollup will process from the earliest usage_log entry forward in 1-day chunks.

-- DOWN
-- Rollback is not destructive — rollup tables are reproducible.
-- To undo: re-run rollup from scratch.
