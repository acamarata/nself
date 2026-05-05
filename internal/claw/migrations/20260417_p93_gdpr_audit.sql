-- P93 S02-T08: GDPR audit trail and export job tables.

-- Append-only audit trail (user-scoped, never deleted).
-- Distinct from np_claw_audit_log (admin-facing). This table is user-scoped,
-- exported on GDPR request, and excluded from erasure by compliance rule.
CREATE TABLE IF NOT EXISTS np_claw_audit_trail (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL,
    actor_id       UUID,           -- NULL = system action
    action         TEXT NOT NULL,  -- dot-namespaced e.g. "memory.create"
    resource_type  TEXT,
    resource_id    TEXT,
    metadata       JSONB NOT NULL DEFAULT '{}',
    ip_address     INET,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_np_claw_audit_trail_user
    ON np_claw_audit_trail(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_np_claw_audit_trail_action
    ON np_claw_audit_trail(action);

-- GDPR export jobs (async processing; goroutine spawned per request).
-- status values: pending → processing → complete | failed
-- export_url stores a data: URI for the JSON archive (production: signed MinIO/S3 URL).
-- error_msg column name matches gdpr_handlers.go SQL references.
CREATE TABLE IF NOT EXISTS np_claw_gdpr_export_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','processing','complete','failed')),
    export_url      TEXT,          -- populated when status = complete
    error_msg       TEXT,          -- populated when status = failed
    row_count       BIGINT,
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ    -- export file expiry (7 days after completion)
);

CREATE INDEX IF NOT EXISTS idx_np_claw_gdpr_export_jobs_user
    ON np_claw_gdpr_export_jobs(user_id, requested_at DESC);

-- GDPR delete requests (tombstone tracking).
-- One active request per user; ON CONFLICT (user_id) in gdpr_handlers.go
-- relies on the unique constraint below to upsert cleanly.
-- status values: pending → processing → complete | failed
CREATE TABLE IF NOT EXISTS np_claw_gdpr_delete_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    UNIQUE (user_id),
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','processing','complete','failed')),
    tables_cleared  TEXT[],        -- list of tables that were cleared
    error_message   TEXT,
    requested_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_np_claw_gdpr_delete_requests_user
    ON np_claw_gdpr_delete_requests(user_id);

-- Rollback:
-- DROP TABLE IF EXISTS np_claw_gdpr_delete_requests, np_claw_gdpr_export_jobs, np_claw_audit_trail;
