-- P87 Sprint 36 G36-03: Audit log schema upgrade
-- Adds missing columns, immutability trigger, and indexes.
-- The base table np_claw_audit_log was created in p85.

-- Add new columns (IF NOT EXISTS via DO block for idempotency)
DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN actor_type TEXT NOT NULL DEFAULT 'user';
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN actor_id TEXT NOT NULL DEFAULT '';
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN impersonator TEXT;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN session_id UUID;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN request_id UUID;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN resource_type TEXT;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN resource_id TEXT;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN before_state JSONB;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN after_state JSONB;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN meta JSONB NOT NULL DEFAULT '{}';
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN success BOOLEAN NOT NULL DEFAULT true;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN error TEXT;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN duration_ms INT;
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

DO $$ BEGIN
    ALTER TABLE np_claw_audit_log ADD COLUMN cost_usd NUMERIC(10,6);
EXCEPTION WHEN duplicate_column THEN NULL; END $$;

-- Backfill actor_id from actor_user_id for existing rows
UPDATE np_claw_audit_log SET actor_id = actor_user_id WHERE actor_id = '' AND actor_user_id IS NOT NULL;

-- Additional indexes
CREATE INDEX IF NOT EXISTS idx_audit_log_resource ON np_claw_audit_log (resource_type, resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_meta ON np_claw_audit_log USING GIN (meta);

-- Immutability trigger: audit log is append-only
CREATE OR REPLACE FUNCTION np_claw_audit_log_immutable()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only: UPDATE and DELETE are forbidden';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_audit_log_immutable ON np_claw_audit_log;
CREATE TRIGGER trg_audit_log_immutable
    BEFORE UPDATE OR DELETE ON np_claw_audit_log
    FOR EACH ROW
    EXECUTE FUNCTION np_claw_audit_log_immutable();
