package tenant

// MigrationAuditLog returns the SQL to create nself_ops.audit_log.
// The table is INSERT-only (no UPDATE or DELETE permissions).
func MigrationAuditLog() string {
	return `-- nself_ops.audit_log: immutable audit trail for tenant-scoped operations
CREATE SCHEMA IF NOT EXISTS nself_ops;

CREATE TABLE IF NOT EXISTS nself_ops.audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    user_id     UUID,
    action      TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    ip          INET,
    user_agent  TEXT,
    payload_sha256 TEXT,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_tenant_id ON nself_ops.audit_log (tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON nself_ops.audit_log (created_at);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON nself_ops.audit_log (action);

-- INSERT-only: revoke UPDATE and DELETE from all roles
REVOKE UPDATE, DELETE ON nself_ops.audit_log FROM PUBLIC;

COMMENT ON TABLE nself_ops.audit_log IS 'Immutable audit trail. INSERT-only, no UPDATE or DELETE.';`
}

// MigrationUsageDaily returns the SQL to create nself_ops.usage_daily.
func MigrationUsageDaily() string {
	return `-- nself_ops.usage_daily: per-tenant daily usage metrics
CREATE SCHEMA IF NOT EXISTS nself_ops;

CREATE TABLE IF NOT EXISTS nself_ops.usage_daily (
    tenant_id UUID NOT NULL,
    day       DATE NOT NULL,
    metric    TEXT NOT NULL,
    value     DOUBLE PRECISION NOT NULL DEFAULT 0,
    PRIMARY KEY (tenant_id, day, metric)
);

CREATE INDEX IF NOT EXISTS idx_usage_daily_day ON nself_ops.usage_daily (day);

COMMENT ON TABLE nself_ops.usage_daily IS 'Per-tenant daily usage metrics for billing metering.';`
}

// MigrationStripeOutbox returns the SQL to create nself_ops.stripe_outbox.
func MigrationStripeOutbox() string {
	return `-- nself_ops.stripe_outbox: retry queue for Stripe API calls
CREATE SCHEMA IF NOT EXISTS nself_ops;

CREATE TABLE IF NOT EXISTS nself_ops.stripe_outbox (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    attempts     INT NOT NULL DEFAULT 0,
    last_error   TEXT,
    processed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_stripe_outbox_pending ON nself_ops.stripe_outbox (created_at)
    WHERE processed_at IS NULL;

COMMENT ON TABLE nself_ops.stripe_outbox IS 'Transactional outbox for Stripe usage record pushes. Retried every 5 minutes.';`
}

// MigrationTenantsTable returns the SQL to create the core tenants table.
func MigrationTenantsTable() string {
	return `-- public.tenants: core tenant registry
CREATE TABLE IF NOT EXISTS public.tenants (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                TEXT NOT NULL UNIQUE,
    plan                TEXT NOT NULL DEFAULT 'basic',
    status              TEXT NOT NULL DEFAULT 'active',
    stripe_customer_id  TEXT,
    stripe_subscription_id TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    suspended_at        TIMESTAMPTZ,
    suspend_reason      TEXT,
    destroyed_at        TIMESTAMPTZ,
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'suspended', 'destroyed')),
    CONSTRAINT tenants_plan_check CHECK (plan IN ('basic', 'pro', 'elite', 'business', 'business-plus', 'enterprise'))
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON public.tenants (slug);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON public.tenants (status);`
}
