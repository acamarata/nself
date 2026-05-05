-- P88 Sprint 12 T12-01: Audit log schema upgrade for production HTTP surface.
-- Adds actor_name column and BRIN index for time-range queries on massive tables.

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN actor_name TEXT;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

-- BRIN index for time-range scans (billions of rows, append-only table)
CREATE INDEX IF NOT EXISTS idx_audit_created_brin
    ON np_claw_audit_log USING BRIN (created_at);

-- Composite indexes for filter queries
CREATE INDEX IF NOT EXISTS idx_audit_actor_created
    ON np_claw_audit_log (actor_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_action_created
    ON np_claw_audit_log (action, created_at DESC);

-- Retention config table
CREATE TABLE IF NOT EXISTS np_claw_audit_retention_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    default_days INT NOT NULL DEFAULT 2555, -- ~7 years
    by_resource_type JSONB NOT NULL DEFAULT '{}',
    locked_by_compliance BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by TEXT
);

-- Seed default retention config if empty
INSERT INTO np_claw_audit_retention_config (default_days)
SELECT 2555
WHERE NOT EXISTS (SELECT 1 FROM np_claw_audit_retention_config);

-- Rate limit tracking for export
CREATE TABLE IF NOT EXISTS np_claw_audit_export_rate (
    user_id TEXT NOT NULL,
    exported_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_audit_export_rate_user
    ON np_claw_audit_export_rate (user_id, exported_at DESC);
