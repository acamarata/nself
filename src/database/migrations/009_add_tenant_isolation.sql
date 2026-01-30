-- Migration 009: Add Tenant Isolation to Existing Tables
-- Adds tenant_id columns and RLS policies to core tables

BEGIN;

-- ============================================================================
-- Add tenant_id to auth.users
-- ============================================================================

-- Add tenant_id column to users
ALTER TABLE IF EXISTS auth.users
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

-- Create index on tenant_id
CREATE INDEX IF NOT EXISTS idx_users_tenant ON auth.users(tenant_id);

-- Set default tenant for existing users
-- (assigns to 'default' tenant created during tenant_init)
UPDATE auth.users
SET tenant_id = (SELECT id FROM tenants.tenants WHERE slug = 'default' LIMIT 1)
WHERE tenant_id IS NULL;

-- Enable RLS on auth.users
ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users can only see users in their tenant
CREATE POLICY user_tenant_isolation ON auth.users
    FOR SELECT
    USING (
        tenant_id = tenants.current_tenant_id()
        OR
        tenants.is_tenant_member(tenant_id, tenants.current_user_id())
    );

-- RLS Policy: Users can only update own record or admins can update tenant users
CREATE POLICY user_update ON auth.users
    FOR UPDATE
    USING (
        id = tenants.current_user_id()
        OR
        (tenant_id = tenants.current_tenant_id()
         AND tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin'))
    );

-- ============================================================================
-- Add tenant_id to auth.sessions
-- ============================================================================

ALTER TABLE IF EXISTS auth.sessions
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_sessions_tenant ON auth.sessions(tenant_id);

-- Set tenant_id for existing sessions based on user's tenant
UPDATE auth.sessions s
SET tenant_id = u.tenant_id
FROM auth.users u
WHERE s.user_id = u.id AND s.tenant_id IS NULL;

-- Enable RLS on sessions
ALTER TABLE auth.sessions ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Sessions are isolated by tenant
CREATE POLICY session_tenant_isolation ON auth.sessions
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
    );

-- ============================================================================
-- Add tenant_id to auth.refresh_tokens
-- ============================================================================

ALTER TABLE IF EXISTS auth.refresh_tokens
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_tenant ON auth.refresh_tokens(tenant_id);

-- Set tenant_id based on user
UPDATE auth.refresh_tokens rt
SET tenant_id = u.tenant_id
FROM auth.users u
WHERE rt.user_id = u.id AND rt.tenant_id IS NULL;

-- Enable RLS
ALTER TABLE auth.refresh_tokens ENABLE ROW LEVEL SECURITY;

CREATE POLICY refresh_token_tenant_isolation ON auth.refresh_tokens
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
    );

-- ============================================================================
-- Add tenant_id to auth.mfa_factors
-- ============================================================================

ALTER TABLE IF EXISTS auth.mfa_factors
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_mfa_factors_tenant ON auth.mfa_factors(tenant_id);

UPDATE auth.mfa_factors m
SET tenant_id = u.tenant_id
FROM auth.users u
WHERE m.user_id = u.id AND m.tenant_id IS NULL;

ALTER TABLE auth.mfa_factors ENABLE ROW LEVEL SECURITY;

CREATE POLICY mfa_tenant_isolation ON auth.mfa_factors
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
    );

-- ============================================================================
-- Add tenant_id to auth.api_keys
-- ============================================================================

ALTER TABLE IF EXISTS auth.api_keys
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON auth.api_keys(tenant_id);

UPDATE auth.api_keys ak
SET tenant_id = u.tenant_id
FROM auth.users u
WHERE ak.user_id = u.id AND ak.tenant_id IS NULL;

ALTER TABLE auth.api_keys ENABLE ROW LEVEL SECURITY;

CREATE POLICY api_key_tenant_isolation ON auth.api_keys
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
    );

-- ============================================================================
-- Observability tables - tenant isolation
-- ============================================================================

-- Metrics
ALTER TABLE IF EXISTS metrics.metrics
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_metrics_tenant ON metrics.metrics(tenant_id);

ALTER TABLE metrics.metrics ENABLE ROW LEVEL SECURITY;

CREATE POLICY metrics_tenant_isolation ON metrics.metrics
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
        OR
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- Logs
ALTER TABLE IF EXISTS logs.log_entries
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_logs_tenant ON logs.log_entries(tenant_id);

ALTER TABLE logs.log_entries ENABLE ROW LEVEL SECURITY;

CREATE POLICY logs_tenant_isolation ON logs.log_entries
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
        OR
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- Traces
ALTER TABLE IF EXISTS tracing.traces
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_traces_tenant ON tracing.traces(tenant_id);

ALTER TABLE tracing.traces ENABLE ROW LEVEL SECURITY;

CREATE POLICY traces_tenant_isolation ON tracing.traces
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
        OR
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- ============================================================================
-- Backup tables - tenant isolation
-- ============================================================================

ALTER TABLE IF EXISTS backups.schedules
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_backup_schedules_tenant ON backups.schedules(tenant_id);

ALTER TABLE backups.schedules ENABLE ROW LEVEL SECURITY;

CREATE POLICY backup_schedules_tenant_isolation ON backups.schedules
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
        OR
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

ALTER TABLE IF EXISTS backups.history
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_backup_history_tenant ON backups.history(tenant_id);

ALTER TABLE backups.history ENABLE ROW LEVEL SECURITY;

CREATE POLICY backup_history_tenant_isolation ON backups.history
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
        OR
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- ============================================================================
-- Compliance tables - tenant isolation
-- ============================================================================

ALTER TABLE IF EXISTS compliance.data_requests
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_data_requests_tenant ON compliance.data_requests(tenant_id);

ALTER TABLE compliance.data_requests ENABLE ROW LEVEL SECURITY;

CREATE POLICY data_requests_tenant_isolation ON compliance.data_requests
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
        OR
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- ============================================================================
-- Redis configuration - tenant isolation
-- ============================================================================

ALTER TABLE IF EXISTS redis_config.connections
ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants.tenants(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_redis_connections_tenant ON redis_config.connections(tenant_id);

ALTER TABLE redis_config.connections ENABLE ROW LEVEL SECURITY;

CREATE POLICY redis_connections_tenant_isolation ON redis_config.connections
    FOR ALL
    USING (
        tenant_id = tenants.current_tenant_id()
        OR
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- ============================================================================
-- Helper Functions for Tenant-Aware Operations
-- ============================================================================

-- Function: Get tenant's database quota
CREATE OR REPLACE FUNCTION tenants.get_tenant_database_size(p_tenant_id UUID)
RETURNS BIGINT AS $$
DECLARE
    v_size BIGINT;
BEGIN
    -- Calculate size of tenant's data across all tables
    -- This is a simplified version - in production, would be more comprehensive
    SELECT
        COALESCE(SUM(pg_total_relation_size(quote_ident(schemaname) || '.' || quote_ident(tablename))), 0)
    INTO v_size
    FROM pg_tables
    WHERE schemaname LIKE 'tenant_%';

    RETURN v_size;
END;
$$ LANGUAGE plpgsql;

-- Function: Check if tenant is within storage quota
CREATE OR REPLACE FUNCTION tenants.check_storage_quota(p_tenant_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_size BIGINT;
    v_max_size BIGINT;
BEGIN
    -- Get current size
    v_current_size := tenants.get_tenant_database_size(p_tenant_id);

    -- Get max allowed size
    SELECT max_storage_gb * 1073741824 INTO v_max_size
    FROM tenants.tenants
    WHERE id = p_tenant_id;

    RETURN v_current_size < v_max_size;
END;
$$ LANGUAGE plpgsql;

-- Function: Get tenant's API request count (this month)
CREATE OR REPLACE FUNCTION tenants.get_tenant_api_requests(p_tenant_id UUID)
RETURNS INTEGER AS $$
DECLARE
    v_count INTEGER;
BEGIN
    -- Count API requests this month from metrics
    SELECT COUNT(*) INTO v_count
    FROM metrics.metrics
    WHERE tenant_id = p_tenant_id
    AND metric_name LIKE 'api.request%'
    AND timestamp >= DATE_TRUNC('month', NOW());

    RETURN v_count;
END;
$$ LANGUAGE plpgsql;

-- Function: Check if tenant is within API quota
CREATE OR REPLACE FUNCTION tenants.check_api_quota(p_tenant_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_current_count INTEGER;
    v_max_count INTEGER;
BEGIN
    v_current_count := tenants.get_tenant_api_requests(p_tenant_id);

    SELECT max_api_requests_per_month INTO v_max_count
    FROM tenants.tenants
    WHERE id = p_tenant_id;

    RETURN v_current_count < v_max_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Tenant-Aware Views
-- ============================================================================

-- View: Current tenant's users
CREATE OR REPLACE VIEW tenants.my_tenant_users AS
SELECT
    u.*
FROM auth.users u
WHERE u.tenant_id = tenants.current_tenant_id();

-- View: Current tenant's usage statistics
CREATE OR REPLACE VIEW tenants.my_tenant_usage AS
SELECT
    t.id as tenant_id,
    t.name as tenant_name,
    t.plan_id,
    t.max_users,
    COUNT(DISTINCT u.id) as current_users,
    t.max_storage_gb,
    tenants.get_tenant_database_size(t.id) as current_storage_bytes,
    t.max_api_requests_per_month,
    tenants.get_tenant_api_requests(t.id) as current_api_requests
FROM tenants.tenants t
LEFT JOIN auth.users u ON u.tenant_id = t.id
WHERE t.id = tenants.current_tenant_id()
GROUP BY t.id;

-- ============================================================================
-- Triggers for Tenant Quota Enforcement
-- ============================================================================

-- Trigger: Prevent user creation if tenant at user limit
CREATE OR REPLACE FUNCTION tenants.check_user_limit()
RETURNS TRIGGER AS $$
DECLARE
    v_tenant_id UUID;
    v_max_users INTEGER;
    v_current_users INTEGER;
BEGIN
    v_tenant_id := NEW.tenant_id;

    -- Get tenant limits
    SELECT max_users INTO v_max_users
    FROM tenants.tenants
    WHERE id = v_tenant_id;

    -- Count current users
    SELECT COUNT(*) INTO v_current_users
    FROM auth.users
    WHERE tenant_id = v_tenant_id;

    -- Check limit
    IF v_current_users >= v_max_users THEN
        RAISE EXCEPTION 'Tenant has reached maximum user limit (%)', v_max_users;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER enforce_user_limit
    BEFORE INSERT ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION tenants.check_user_limit();

-- ============================================================================
-- Grants
-- ============================================================================

GRANT EXECUTE ON FUNCTION tenants.get_tenant_database_size(UUID) TO hasura;
GRANT EXECUTE ON FUNCTION tenants.check_storage_quota(UUID) TO hasura;
GRANT EXECUTE ON FUNCTION tenants.get_tenant_api_requests(UUID) TO hasura;
GRANT EXECUTE ON FUNCTION tenants.check_api_quota(UUID) TO hasura;

GRANT SELECT ON tenants.my_tenant_users TO hasura;
GRANT SELECT ON tenants.my_tenant_usage TO hasura;

COMMIT;
