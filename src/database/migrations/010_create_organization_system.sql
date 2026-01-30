-- Migration 010: Organization & Team System
-- Creates organization hierarchy and team-based access control

BEGIN;

-- ============================================================================
-- SCHEMA: organizations
-- Organization hierarchy and workspace management
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS organizations;

-- Organizations table
CREATE TABLE organizations.organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'deleted')),

    -- Billing
    billing_email TEXT,
    billing_plan TEXT DEFAULT 'free',

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Owner
    owner_user_id UUID NOT NULL,

    -- Settings
    settings JSONB DEFAULT '{}'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_orgs_slug ON organizations.organizations(slug);
CREATE INDEX idx_orgs_status ON organizations.organizations(status);
CREATE INDEX idx_orgs_owner ON organizations.organizations(owner_user_id);

-- Organization members
CREATE TABLE organizations.org_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member', 'guest')),

    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    invited_by UUID,

    UNIQUE (org_id, user_id)
);

CREATE INDEX idx_org_members_org ON organizations.org_members(org_id);
CREATE INDEX idx_org_members_user ON organizations.org_members(user_id);

-- Teams within organizations
CREATE TABLE organizations.teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    slug TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    settings JSONB DEFAULT '{}'::jsonb,

    UNIQUE (org_id, slug)
);

CREATE INDEX idx_teams_org ON organizations.teams(org_id);
CREATE INDEX idx_teams_slug ON organizations.teams(slug);

-- Team members
CREATE TABLE organizations.team_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID NOT NULL REFERENCES organizations.teams(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('lead', 'member')),

    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    added_by UUID,

    UNIQUE (team_id, user_id)
);

CREATE INDEX idx_team_members_team ON organizations.team_members(team_id);
CREATE INDEX idx_team_members_user ON organizations.team_members(user_id);

-- Organization-tenant relationship (one org can have multiple tenants)
CREATE TABLE organizations.org_tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants.tenants(id) ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (org_id, tenant_id)
);

CREATE INDEX idx_org_tenants_org ON organizations.org_tenants(org_id);
CREATE INDEX idx_org_tenants_tenant ON organizations.org_tenants(tenant_id);

-- ============================================================================
-- SCHEMA: permissions
-- Advanced RBAC system
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS permissions;

-- Roles (custom role definitions)
CREATE TABLE permissions.roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,

    -- Built-in roles cannot be deleted
    is_builtin BOOLEAN DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (org_id, name)
);

CREATE INDEX idx_roles_org ON permissions.roles(org_id);

-- Permissions (granular permissions)
CREATE TABLE permissions.permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL, -- e.g., 'tenant.create', 'user.delete', 'team.manage'
    description TEXT,
    resource_type TEXT NOT NULL, -- e.g., 'tenant', 'user', 'team'
    action TEXT NOT NULL, -- e.g., 'create', 'read', 'update', 'delete', 'manage'

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_permissions_resource ON permissions.permissions(resource_type);
CREATE INDEX idx_permissions_action ON permissions.permissions(action);

-- Role-Permission assignments
CREATE TABLE permissions.role_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES permissions.roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions.permissions(id) ON DELETE CASCADE,

    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (role_id, permission_id)
);

CREATE INDEX idx_role_perms_role ON permissions.role_permissions(role_id);
CREATE INDEX idx_role_perms_permission ON permissions.role_permissions(permission_id);

-- User-Role assignments (user can have multiple roles)
CREATE TABLE permissions.user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role_id UUID NOT NULL REFERENCES permissions.roles(id) ON DELETE CASCADE,
    org_id UUID NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,

    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID,

    -- Scope: 'global' (all resources), 'tenant' (specific tenant), 'team' (specific team)
    scope TEXT NOT NULL DEFAULT 'global' CHECK (scope IN ('global', 'tenant', 'team')),
    scope_id UUID, -- tenant_id or team_id if scope is not global

    UNIQUE (user_id, role_id, org_id, scope, scope_id)
);

CREATE INDEX idx_user_roles_user ON permissions.user_roles(user_id);
CREATE INDEX idx_user_roles_role ON permissions.user_roles(role_id);
CREATE INDEX idx_user_roles_org ON permissions.user_roles(org_id);

-- Permission audit log
CREATE TABLE permissions.permission_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    org_id UUID REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    action TEXT NOT NULL, -- 'grant', 'revoke'
    resource_type TEXT NOT NULL,
    resource_id UUID,
    permission_name TEXT,

    performed_by UUID,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_perm_audit_user ON permissions.permission_audit(user_id);
CREATE INDEX idx_perm_audit_org ON permissions.permission_audit(org_id);
CREATE INDEX idx_perm_audit_timestamp ON permissions.permission_audit(timestamp);

-- ============================================================================
-- FUNCTIONS: Organization Management
-- ============================================================================

-- Function: Get current organization ID from session
CREATE OR REPLACE FUNCTION organizations.current_org_id()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('hasura.user.x-hasura-org-id', true)::uuid;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Check if user is member of organization
CREATE OR REPLACE FUNCTION organizations.is_org_member(p_org_id UUID, p_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM organizations.org_members
        WHERE org_id = p_org_id
        AND user_id = p_user_id
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Get user's organization role
CREATE OR REPLACE FUNCTION organizations.get_user_org_role(p_org_id UUID, p_user_id UUID)
RETURNS TEXT AS $$
DECLARE
    v_role TEXT;
BEGIN
    SELECT role INTO v_role
    FROM organizations.org_members
    WHERE org_id = p_org_id
    AND user_id = p_user_id;

    RETURN v_role;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Check if user is member of team
CREATE OR REPLACE FUNCTION organizations.is_team_member(p_team_id UUID, p_user_id UUID)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM organizations.team_members
        WHERE team_id = p_team_id
        AND user_id = p_user_id
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Check if user has permission
CREATE OR REPLACE FUNCTION permissions.has_permission(
    p_user_id UUID,
    p_org_id UUID,
    p_permission_name TEXT,
    p_scope TEXT DEFAULT 'global',
    p_scope_id UUID DEFAULT NULL
)
RETURNS BOOLEAN AS $$
DECLARE
    v_has_permission BOOLEAN;
BEGIN
    -- Check if user has the permission through any of their roles
    SELECT EXISTS (
        SELECT 1
        FROM permissions.user_roles ur
        INNER JOIN permissions.role_permissions rp ON ur.role_id = rp.role_id
        INNER JOIN permissions.permissions p ON rp.permission_id = p.id
        WHERE ur.user_id = p_user_id
        AND ur.org_id = p_org_id
        AND p.name = p_permission_name
        AND (ur.scope = 'global' OR (ur.scope = p_scope AND ur.scope_id = p_scope_id))
    ) INTO v_has_permission;

    RETURN v_has_permission;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Get user's all permissions
CREATE OR REPLACE FUNCTION permissions.get_user_permissions(
    p_user_id UUID,
    p_org_id UUID
)
RETURNS TABLE (
    permission_name TEXT,
    resource_type TEXT,
    action TEXT,
    scope TEXT,
    scope_id UUID
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT
        p.name as permission_name,
        p.resource_type,
        p.action,
        ur.scope,
        ur.scope_id
    FROM permissions.user_roles ur
    INNER JOIN permissions.role_permissions rp ON ur.role_id = rp.role_id
    INNER JOIN permissions.permissions p ON rp.permission_id = p.id
    WHERE ur.user_id = p_user_id
    AND ur.org_id = p_org_id;
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Trigger: Update updated_at
CREATE OR REPLACE FUNCTION organizations.update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_orgs_updated_at
    BEFORE UPDATE ON organizations.organizations
    FOR EACH ROW
    EXECUTE FUNCTION organizations.update_updated_at();

CREATE TRIGGER update_teams_updated_at
    BEFORE UPDATE ON organizations.teams
    FOR EACH ROW
    EXECUTE FUNCTION organizations.update_updated_at();

CREATE TRIGGER update_roles_updated_at
    BEFORE UPDATE ON permissions.roles
    FOR EACH ROW
    EXECUTE FUNCTION organizations.update_updated_at();

-- ============================================================================
-- ROW LEVEL SECURITY
-- ============================================================================

-- Enable RLS
ALTER TABLE organizations.organizations ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations.org_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations.teams ENABLE ROW LEVEL SECURITY;
ALTER TABLE organizations.team_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE permissions.roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE permissions.user_roles ENABLE ROW LEVEL SECURITY;

-- RLS: Organizations - members can view
CREATE POLICY org_member_select ON organizations.organizations
    FOR SELECT
    USING (
        organizations.is_org_member(id, tenants.current_user_id())
    );

-- RLS: Organizations - owners can manage
CREATE POLICY org_owner_manage ON organizations.organizations
    FOR ALL
    USING (
        owner_user_id = tenants.current_user_id()
        OR
        organizations.get_user_org_role(id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- RLS: Teams - org members can view
CREATE POLICY team_member_select ON organizations.teams
    FOR SELECT
    USING (
        organizations.is_org_member(org_id, tenants.current_user_id())
    );

-- RLS: Teams - org admins can manage
CREATE POLICY team_admin_manage ON organizations.teams
    FOR ALL
    USING (
        organizations.get_user_org_role(org_id, tenants.current_user_id()) IN ('owner', 'admin')
    );

-- ============================================================================
-- DEFAULT PERMISSIONS
-- ============================================================================

-- Insert default permissions
INSERT INTO permissions.permissions (name, description, resource_type, action) VALUES
-- Tenant permissions
('tenant.create', 'Create new tenants', 'tenant', 'create'),
('tenant.read', 'View tenant details', 'tenant', 'read'),
('tenant.update', 'Update tenant settings', 'tenant', 'update'),
('tenant.delete', 'Delete tenants', 'tenant', 'delete'),
('tenant.manage', 'Full tenant management', 'tenant', 'manage'),

-- User permissions
('user.create', 'Create new users', 'user', 'create'),
('user.read', 'View user details', 'user', 'read'),
('user.update', 'Update user information', 'user', 'update'),
('user.delete', 'Delete users', 'user', 'delete'),
('user.manage', 'Full user management', 'user', 'manage'),

-- Team permissions
('team.create', 'Create new teams', 'team', 'create'),
('team.read', 'View team details', 'team', 'read'),
('team.update', 'Update team settings', 'team', 'update'),
('team.delete', 'Delete teams', 'team', 'delete'),
('team.manage', 'Full team management', 'team', 'manage'),

-- Organization permissions
('org.billing', 'Manage organization billing', 'org', 'billing'),
('org.settings', 'Manage organization settings', 'org', 'settings'),
('org.members', 'Manage organization members', 'org', 'members')

ON CONFLICT DO NOTHING;

-- ============================================================================
-- VIEWS
-- ============================================================================

-- View: User's organizations
CREATE OR REPLACE VIEW organizations.user_organizations AS
SELECT
    o.*,
    om.role as user_role,
    om.joined_at
FROM organizations.organizations o
INNER JOIN organizations.org_members om ON o.id = om.org_id
WHERE om.user_id = tenants.current_user_id()
AND o.status = 'active';

-- View: User's teams
CREATE OR REPLACE VIEW organizations.user_teams AS
SELECT
    t.*,
    tm.role as user_role,
    tm.joined_at,
    o.name as org_name
FROM organizations.teams t
INNER JOIN organizations.team_members tm ON t.id = tm.team_id
INNER JOIN organizations.organizations o ON t.org_id = o.id
WHERE tm.user_id = tenants.current_user_id();

-- View: Organization with tenant count
CREATE OR REPLACE VIEW organizations.org_stats AS
SELECT
    o.id,
    o.slug,
    o.name,
    o.billing_plan,
    COUNT(DISTINCT om.user_id) as member_count,
    COUNT(DISTINCT ot.tenant_id) as tenant_count,
    COUNT(DISTINCT t.id) as team_count
FROM organizations.organizations o
LEFT JOIN organizations.org_members om ON o.id = om.org_id
LEFT JOIN organizations.org_tenants ot ON o.id = ot.org_id
LEFT JOIN organizations.teams t ON o.id = t.org_id
WHERE o.status = 'active'
GROUP BY o.id;

-- ============================================================================
-- GRANTS
-- ============================================================================

-- Grant usage
GRANT USAGE ON SCHEMA organizations TO hasura;
GRANT USAGE ON SCHEMA permissions TO hasura;

-- Grant select on all tables
GRANT SELECT ON ALL TABLES IN SCHEMA organizations TO hasura;
GRANT SELECT ON ALL TABLES IN SCHEMA permissions TO hasura;

-- Grant insert, update, delete
GRANT INSERT, UPDATE, DELETE ON organizations.organizations TO hasura;
GRANT INSERT, UPDATE, DELETE ON organizations.org_members TO hasura;
GRANT INSERT, UPDATE, DELETE ON organizations.teams TO hasura;
GRANT INSERT, UPDATE, DELETE ON organizations.team_members TO hasura;
GRANT INSERT, UPDATE, DELETE ON permissions.roles TO hasura;
GRANT INSERT, UPDATE, DELETE ON permissions.role_permissions TO hasura;
GRANT INSERT, UPDATE, DELETE ON permissions.user_roles TO hasura;

-- Grant execute on functions
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA organizations TO hasura;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA permissions TO hasura;

COMMIT;
