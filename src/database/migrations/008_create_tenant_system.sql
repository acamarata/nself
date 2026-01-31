-- Migration 008: Multi-Tenancy Foundation
-- Creates core tenant management tables and Row-Level Security (RLS) policies

BEGIN;

-- ============================================================================
-- SCHEMA: tenants
-- Core tenant management
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS tenants;

-- Tenants table
CREATE TABLE tenants.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),

    -- Tenant configuration
    plan_id TEXT DEFAULT 'free',
    max_users INTEGER DEFAULT 5,
    max_storage_gb INTEGER DEFAULT 1,
    max_api_requests_per_month INTEGER DEFAULT 10000,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    suspended_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    -- Owner
    owner_user_id UUID NOT NULL,

    -- Settings
    settings JSONB DEFAULT '{}'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Indexes
CREATE INDEX idx_tenants_slug ON tenants.tenants(slug);
CREATE INDEX idx_tenants_status ON tenants.tenants(status);
CREATE INDEX idx_tenants_owner ON tenants.tenants(owner_user_id);

-- Tenant schemas (tracks PostgreSQL schemas created for each tenant)
CREATE TABLE tenants.tenant_schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants.tenants(id) ON DELETE CASCADE,
    schema_name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (tenant_id, schema_name)
);

CREATE INDEX idx_tenant_schemas_tenant ON tenants.tenant_schemas(tenant_id);

-- Tenant domains (custom domains per tenant)
CREATE TABLE tenants.tenant_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants.tenants(id) ON DELETE CASCADE,
    domain TEXT UNIQUE NOT NULL,
    is_primary BOOLEAN DEFAULT false,
    is_verified BOOLEAN DEFAULT false,
    verification_token TEXT,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_domains_tenant ON tenants.tenant_domains(tenant_id);
CREATE INDEX idx_tenant_domains_domain ON tenants.tenant_domains(domain);
-- Partial unique index: only one primary domain per tenant
CREATE UNIQUE INDEX idx_tenant_domains_primary ON tenants.tenant_domains(tenant_id, is_primary) WHERE is_primary = true;

-- Tenant members (users belonging to tenants)
CREATE TABLE tenants.tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'guest')),

    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    invited_by UUID,

    UNIQUE (tenant_id, user_id)
);

CREATE INDEX idx_tenant_members_tenant ON tenants.tenant_members(tenant_id);
CREATE INDEX idx_tenant_members_user ON tenants.tenant_members(user_id);

-- Tenant settings (key-value settings per tenant)
CREATE TABLE tenants.tenant_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants.tenants(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (tenant_id, key)
);

CREATE INDEX idx_tenant_settings_tenant ON tenants.tenant_settings(tenant_id);

-- ============================================================================
-- FUNCTIONS: Tenant Management
-- ============================================================================

-- Function: Get current tenant ID from session
CREATE OR REPLACE FUNCTION tenants.current_tenant_id()
RETURNS UUID AS $$
BEGIN
    -- Get tenant_id from Hasura session variable
    RETURN current_setting('hasura.user.x-hasura-tenant-id', true)::uuid;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Get current user ID from session
CREATE OR REPLACE FUNCTION tenants.current_user_id()
RETURNS UUID AS $$
BEGIN
    -- Get user_id from Hasura session variable
    RETURN current_setting('hasura.user.x-hasura-user-id', true)::uuid;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Check if user is member of tenant
CREATE OR REPLACE FUNCTION tenants.is_tenant_member(p_tenant_id UUID, p_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM tenants.tenant_members
        WHERE tenant_id = p_tenant_id
        AND user_id = p_user_id
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Get user's tenant role
CREATE OR REPLACE FUNCTION tenants.get_user_tenant_role(p_tenant_id UUID, p_user_id UUID)
RETURNS TEXT AS $$
DECLARE
    v_role TEXT;
BEGIN
    SELECT role INTO v_role
    FROM tenants.tenant_members
    WHERE tenant_id = p_tenant_id
    AND user_id = p_user_id;

    RETURN v_role;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Create tenant schema
CREATE OR REPLACE FUNCTION tenants.create_tenant_schema(p_tenant_id UUID)
RETURNS TEXT AS $$
DECLARE
    v_schema_name TEXT;
BEGIN
    -- Generate schema name
    v_schema_name := 'tenant_' || replace(p_tenant_id::text, '-', '_');

    -- Create schema
    EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', v_schema_name);

    -- Record schema creation
    INSERT INTO tenants.tenant_schemas (tenant_id, schema_name)
    VALUES (p_tenant_id, v_schema_name)
    ON CONFLICT (tenant_id, schema_name) DO NOTHING;

    RETURN v_schema_name;
END;
$$ LANGUAGE plpgsql;

-- Function: Drop tenant schema
CREATE OR REPLACE FUNCTION tenants.drop_tenant_schema(p_tenant_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_schema_name TEXT;
BEGIN
    -- Get schema name
    SELECT schema_name INTO v_schema_name
    FROM tenants.tenant_schemas
    WHERE tenant_id = p_tenant_id;

    IF v_schema_name IS NULL THEN
        RETURN false;
    END IF;

    -- Drop schema
    EXECUTE format('DROP SCHEMA IF EXISTS %I CASCADE', v_schema_name);

    -- Remove record
    DELETE FROM tenants.tenant_schemas
    WHERE tenant_id = p_tenant_id;

    RETURN true;
END;
$$ LANGUAGE plpgsql;

-- Function: Updated_at trigger
CREATE OR REPLACE FUNCTION tenants.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at
CREATE TRIGGER update_tenants_updated_at
    BEFORE UPDATE ON tenants.tenants
    FOR EACH ROW
    EXECUTE FUNCTION tenants.update_updated_at();

CREATE TRIGGER update_tenant_settings_updated_at
    BEFORE UPDATE ON tenants.tenant_settings
    FOR EACH ROW
    EXECUTE FUNCTION tenants.update_updated_at();

-- ============================================================================
-- ROW LEVEL SECURITY (RLS)
-- ============================================================================

-- Enable RLS on all tenant tables
ALTER TABLE tenants.tenants ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants.tenant_schemas ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants.tenant_domains ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants.tenant_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenants.tenant_settings ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users can only see tenants they are members of
CREATE POLICY tenant_member_select ON tenants.tenants
    FOR SELECT
    USING (
        id = tenants.current_tenant_id()
        OR
        tenants.is_tenant_member(id, tenants.current_user_id())
    );

-- RLS Policy: Only tenant owners can update tenants
CREATE POLICY tenant_owner_update ON tenants.tenants
    FOR UPDATE
    USING (
        owner_user_id = tenants.current_user_id()
        OR
        tenants.get_user_tenant_role(id, tenants.current_user_id()) = 'owner'
    );

-- RLS Policy: Any authenticated user can create a tenant
CREATE POLICY tenant_create ON tenants.tenants
    FOR INSERT
    WITH CHECK (
        owner_user_id = tenants.current_user_id()
    );

-- RLS Policy: Only tenant owners can delete tenants
CREATE POLICY tenant_owner_delete ON tenants.tenants
    FOR DELETE
    USING (
        owner_user_id = tenants.current_user_id()
        OR
        tenants.get_user_tenant_role(id, tenants.current_user_id()) = 'owner'
    );

-- RLS Policy: Tenant domains - members can view
CREATE POLICY tenant_domains_select ON tenants.tenant_domains
    FOR SELECT
    USING (
        tenants.is_tenant_member(tenant_id, tenants.current_user_id())
    );

-- RLS Policy: Tenant domains - admins/owners can manage
CREATE POLICY tenant_domains_manage ON tenants.tenant_domains
    FOR ALL
    USING (
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- RLS Policy: Tenant members - can view own tenant's members
CREATE POLICY tenant_members_select ON tenants.tenant_members
    FOR SELECT
    USING (
        tenants.is_tenant_member(tenant_id, tenants.current_user_id())
    );

-- RLS Policy: Tenant members - admins/owners can manage
CREATE POLICY tenant_members_manage ON tenants.tenant_members
    FOR ALL
    USING (
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- RLS Policy: Tenant settings - members can view
CREATE POLICY tenant_settings_select ON tenants.tenant_settings
    FOR SELECT
    USING (
        tenants.is_tenant_member(tenant_id, tenants.current_user_id())
    );

-- RLS Policy: Tenant settings - admins/owners can manage
CREATE POLICY tenant_settings_manage ON tenants.tenant_settings
    FOR ALL
    USING (
        tenants.get_user_tenant_role(tenant_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- ============================================================================
-- VIEWS: Convenience views for common queries
-- ============================================================================

-- View: Active tenants with member count
CREATE OR REPLACE VIEW tenants.active_tenants_with_stats AS
SELECT
    t.id,
    t.slug,
    t.name,
    t.plan_id,
    t.created_at,
    COUNT(tm.id) as member_count,
    t.max_users,
    t.max_storage_gb,
    t.max_api_requests_per_month
FROM tenants.tenants t
LEFT JOIN tenants.tenant_members tm ON t.id = tm.tenant_id
WHERE t.status = 'active'
GROUP BY t.id, t.slug, t.name, t.plan_id, t.created_at, t.max_users, t.max_storage_gb, t.max_api_requests_per_month;

-- View: User's tenants
CREATE OR REPLACE VIEW tenants.user_tenants AS
SELECT
    t.*,
    tm.role as user_role,
    tm.joined_at
FROM tenants.tenants t
INNER JOIN tenants.tenant_members tm ON t.id = tm.tenant_id
WHERE tm.user_id = tenants.current_user_id()
AND t.status = 'active';

-- ============================================================================
-- GRANTS: Hasura permissions
-- ============================================================================

-- Create hasura role if it doesn't exist
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'hasura') THEN
    CREATE ROLE hasura WITH LOGIN PASSWORD 'hasura';
  END IF;
END
$$;

-- Grant usage on schema
GRANT USAGE ON SCHEMA tenants TO hasura;

-- Grant select on all tables
GRANT SELECT ON ALL TABLES IN SCHEMA tenants TO hasura;

-- Grant insert, update, delete on tables
GRANT INSERT, UPDATE, DELETE ON tenants.tenants TO hasura;
GRANT INSERT, UPDATE, DELETE ON tenants.tenant_domains TO hasura;
GRANT INSERT, UPDATE, DELETE ON tenants.tenant_members TO hasura;
GRANT INSERT, UPDATE, DELETE ON tenants.tenant_settings TO hasura;

-- Grant execute on functions
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA tenants TO hasura;

-- Grant usage on sequences
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA tenants TO hasura;

COMMIT;
