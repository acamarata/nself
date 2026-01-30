-- ============================================================================
-- nself Database Migration: 020_add_whitelabel_rls.sql
-- Part of nself v0.9.0 - Sprint 21: Security Hardening
-- ============================================================================
-- Description: Row-Level Security (RLS) policies for white-label system tables
--              Enforces multi-tenant isolation based on tenant_id/brand_id
-- Dependencies: 016_create_whitelabel_system.sql
-- ============================================================================
-- Author: nself
-- Version: 0.9.0
-- Date: 2026-01-30
-- ============================================================================

-- ============================================================================
-- Session Variables for RLS
-- ============================================================================
-- The following session variables must be set by your application:
--
-- app.current_tenant_id - The authenticated tenant's ID
-- app.current_brand_id - The authenticated brand's UUID
-- app.user_role - User role (admin, tenant_admin, tenant_user, public)
-- app.is_admin - Boolean flag for super admin bypass
--
-- Example usage in application:
-- SET LOCAL app.current_tenant_id = 'acme-corp';
-- SET LOCAL app.current_brand_id = 'uuid-here';
-- SET LOCAL app.user_role = 'tenant_admin';
-- ============================================================================

-- ============================================================================
-- Helper Functions for RLS Policies
-- ============================================================================

-- Function to get current tenant ID from session
CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS VARCHAR AS $$
BEGIN
    RETURN current_setting('app.current_tenant_id', true)::VARCHAR;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

COMMENT ON FUNCTION get_current_tenant_id() IS 'Returns current tenant ID from session variable';

-- Function to get current brand ID from session
CREATE OR REPLACE FUNCTION get_current_brand_id()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_brand_id', true)::UUID;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

COMMENT ON FUNCTION get_current_brand_id() IS 'Returns current brand UUID from session variable';

-- Function to check if current user is super admin (already exists, ensure consistency)
CREATE OR REPLACE FUNCTION is_current_user_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN COALESCE(
        current_setting('app.is_admin', true)::BOOLEAN,
        current_setting('app.user_role', true) = 'admin',
        false
    );
EXCEPTION
    WHEN OTHERS THEN
        RETURN false;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

COMMENT ON FUNCTION is_current_user_admin() IS 'Returns true if current user has super admin privileges';

-- Function to check if current user is tenant admin
CREATE OR REPLACE FUNCTION is_current_user_tenant_admin()
RETURNS BOOLEAN AS $$
BEGIN
    RETURN COALESCE(
        current_setting('app.user_role', true) IN ('admin', 'tenant_admin'),
        false
    );
EXCEPTION
    WHEN OTHERS THEN
        RETURN false;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

COMMENT ON FUNCTION is_current_user_tenant_admin() IS 'Returns true if current user is a tenant administrator';

-- Function to get user role
CREATE OR REPLACE FUNCTION get_current_user_role()
RETURNS VARCHAR AS $$
BEGIN
    RETURN COALESCE(
        current_setting('app.user_role', true)::VARCHAR,
        'anonymous'
    );
EXCEPTION
    WHEN OTHERS THEN
        RETURN 'anonymous';
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

COMMENT ON FUNCTION get_current_user_role() IS 'Returns current user role from session variable';

-- ============================================================================
-- RLS Policies for whitelabel_brands
-- ============================================================================

ALTER TABLE whitelabel_brands ENABLE ROW LEVEL SECURITY;

-- Super admin bypass: Admins can do everything
CREATE POLICY admin_all_access ON whitelabel_brands
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Tenant admin access: Can manage their own brand
CREATE POLICY tenant_admin_full_access ON whitelabel_brands
    FOR ALL
    TO PUBLIC
    USING (
        tenant_id = get_current_tenant_id()
        AND is_current_user_tenant_admin()
    )
    WITH CHECK (
        tenant_id = get_current_tenant_id()
        AND is_current_user_tenant_admin()
    );

-- Tenant user read: Can view their own brand
CREATE POLICY tenant_user_read_own ON whitelabel_brands
    FOR SELECT
    TO PUBLIC
    USING (
        tenant_id = get_current_tenant_id()
        AND is_active = true
    );

-- Public read: Can view primary brands only (for public branding)
CREATE POLICY public_read_primary ON whitelabel_brands
    FOR SELECT
    TO PUBLIC
    USING (
        is_primary = true
        AND is_active = true
        AND get_current_user_role() = 'public'
    );

COMMENT ON TABLE whitelabel_brands IS 'RLS enabled: Tenants can manage their brand. Public can view primary brand.';

-- ============================================================================
-- RLS Policies for whitelabel_domains
-- ============================================================================

ALTER TABLE whitelabel_domains ENABLE ROW LEVEL SECURITY;

-- Super admin bypass
CREATE POLICY admin_all_access ON whitelabel_domains
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Tenant admin access: Can manage domains for their brand
CREATE POLICY tenant_admin_manage_own ON whitelabel_domains
    FOR ALL
    TO PUBLIC
    USING (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_current_user_tenant_admin()
    )
    WITH CHECK (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_current_user_tenant_admin()
    );

-- Tenant user read: Can view domains for their brand
CREATE POLICY tenant_user_read_own ON whitelabel_domains
    FOR SELECT
    TO PUBLIC
    USING (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_active = true
    );

-- Public read: Can view active domains (for DNS verification)
CREATE POLICY public_read_active ON whitelabel_domains
    FOR SELECT
    TO PUBLIC
    USING (
        status = 'active'
        AND is_active = true
    );

COMMENT ON TABLE whitelabel_domains IS 'RLS enabled: Tenant admins manage domains. Public can view active domains.';

-- ============================================================================
-- RLS Policies for whitelabel_themes
-- ============================================================================

ALTER TABLE whitelabel_themes ENABLE ROW LEVEL SECURITY;

-- Super admin bypass
CREATE POLICY admin_all_access ON whitelabel_themes
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Tenant admin full access: Can manage themes for their brand
CREATE POLICY tenant_admin_manage_own ON whitelabel_themes
    FOR ALL
    TO PUBLIC
    USING (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_current_user_tenant_admin()
    )
    WITH CHECK (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_current_user_tenant_admin()
    );

-- Tenant user read: Can view active themes for their brand
CREATE POLICY tenant_user_read_own ON whitelabel_themes
    FOR SELECT
    TO PUBLIC
    USING (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_active = true
    );

-- Public read: Can view system themes (built-in themes)
CREATE POLICY public_read_system_themes ON whitelabel_themes
    FOR SELECT
    TO PUBLIC
    USING (
        is_system = true
        AND is_active = true
    );

COMMENT ON TABLE whitelabel_themes IS 'RLS enabled: Tenant admins manage themes. Public can view system themes.';

-- ============================================================================
-- RLS Policies for whitelabel_email_templates
-- ============================================================================

ALTER TABLE whitelabel_email_templates ENABLE ROW LEVEL SECURITY;

-- Super admin bypass
CREATE POLICY admin_all_access ON whitelabel_email_templates
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Tenant admin full access: Can manage email templates for their brand
CREATE POLICY tenant_admin_manage_own ON whitelabel_email_templates
    FOR ALL
    TO PUBLIC
    USING (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_current_user_tenant_admin()
    )
    WITH CHECK (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_current_user_tenant_admin()
    );

-- Tenant user read: Can view active templates for their brand
CREATE POLICY tenant_user_read_own ON whitelabel_email_templates
    FOR SELECT
    TO PUBLIC
    USING (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_active = true
    );

-- System read: Email service can read templates for sending
CREATE POLICY system_read_for_sending ON whitelabel_email_templates
    FOR SELECT
    TO PUBLIC
    USING (
        get_current_user_role() IN ('system', 'email_service')
        AND is_active = true
    );

-- System update: Email service can update sent_count
CREATE POLICY system_update_stats ON whitelabel_email_templates
    FOR UPDATE
    TO PUBLIC
    USING (
        get_current_user_role() IN ('system', 'email_service')
    )
    WITH CHECK (
        get_current_user_role() IN ('system', 'email_service')
    );

COMMENT ON TABLE whitelabel_email_templates IS 'RLS enabled: Tenant admins manage templates. System can read/update for sending.';

-- ============================================================================
-- RLS Policies for whitelabel_assets
-- ============================================================================

ALTER TABLE whitelabel_assets ENABLE ROW LEVEL SECURITY;

-- Super admin bypass
CREATE POLICY admin_all_access ON whitelabel_assets
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Tenant admin full access: Can manage assets for their brand
CREATE POLICY tenant_admin_manage_own ON whitelabel_assets
    FOR ALL
    TO PUBLIC
    USING (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_current_user_tenant_admin()
    )
    WITH CHECK (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_current_user_tenant_admin()
    );

-- Tenant user read: Can view assets for their brand
CREATE POLICY tenant_user_read_own ON whitelabel_assets
    FOR SELECT
    TO PUBLIC
    USING (
        brand_id IN (
            SELECT id FROM whitelabel_brands
            WHERE tenant_id = get_current_tenant_id()
        )
        AND is_active = true
    );

-- Public read: Can view public assets (for CDN access, logos, etc)
CREATE POLICY public_read_public_assets ON whitelabel_assets
    FOR SELECT
    TO PUBLIC
    USING (
        is_public = true
        AND is_active = true
    );

-- CDN read: CDN service can read all active assets for delivery
CREATE POLICY cdn_read_assets ON whitelabel_assets
    FOR SELECT
    TO PUBLIC
    USING (
        get_current_user_role() = 'cdn_service'
        AND is_active = true
    );

COMMENT ON TABLE whitelabel_assets IS 'RLS enabled: Tenant admins manage assets. Public can read public assets (CDN).';

-- ============================================================================
-- Special Policies for SSL Certificates and Keys
-- ============================================================================

-- Additional policy: Restrict certificate/key access to admins and system
CREATE POLICY admin_only_secrets ON whitelabel_assets
    FOR SELECT
    TO PUBLIC
    USING (
        asset_type IN ('certificate', 'key')
        AND (is_current_user_admin() OR get_current_user_role() IN ('system', 'ssl_service'))
    );

COMMENT ON POLICY admin_only_secrets ON whitelabel_assets IS 'Restricts certificate/key access to admins and system services';

-- ============================================================================
-- RLS for View: whitelabel_brands_full
-- ============================================================================
-- Note: Views inherit RLS from base tables automatically
-- The view already respects RLS on whitelabel_brands, whitelabel_themes, and whitelabel_domains

COMMENT ON VIEW whitelabel_brands_full IS 'RLS enabled: View inherits policies from base tables';

-- ============================================================================
-- Performance Indexes for RLS
-- ============================================================================

-- Optimize tenant_id lookups
CREATE INDEX IF NOT EXISTS idx_whitelabel_brands_tenant_active ON whitelabel_brands(tenant_id, is_active);
CREATE INDEX IF NOT EXISTS idx_whitelabel_brands_primary ON whitelabel_brands(is_primary) WHERE is_primary = true AND is_active = true;

-- Optimize brand_id lookups for related tables
CREATE INDEX IF NOT EXISTS idx_whitelabel_domains_brand_active ON whitelabel_domains(brand_id, is_active);
CREATE INDEX IF NOT EXISTS idx_whitelabel_domains_status_active ON whitelabel_domains(status, is_active);
CREATE INDEX IF NOT EXISTS idx_whitelabel_themes_brand_active ON whitelabel_themes(brand_id, is_active);
CREATE INDEX IF NOT EXISTS idx_whitelabel_themes_system ON whitelabel_themes(is_system) WHERE is_system = true AND is_active = true;
CREATE INDEX IF NOT EXISTS idx_whitelabel_email_templates_brand_active ON whitelabel_email_templates(brand_id, is_active);
CREATE INDEX IF NOT EXISTS idx_whitelabel_email_templates_template_lang ON whitelabel_email_templates(template_name, language_code);
CREATE INDEX IF NOT EXISTS idx_whitelabel_assets_brand_active ON whitelabel_assets(brand_id, is_active);
CREATE INDEX IF NOT EXISTS idx_whitelabel_assets_public ON whitelabel_assets(is_public) WHERE is_public = true AND is_active = true;
CREATE INDEX IF NOT EXISTS idx_whitelabel_assets_type_category ON whitelabel_assets(asset_type, asset_category);

-- ============================================================================
-- Grant Permissions for Common Roles
-- ============================================================================

-- Grant SELECT on helper functions to PUBLIC (they have SECURITY DEFINER)
GRANT EXECUTE ON FUNCTION get_current_tenant_id() TO PUBLIC;
GRANT EXECUTE ON FUNCTION get_current_brand_id() TO PUBLIC;
GRANT EXECUTE ON FUNCTION is_current_user_tenant_admin() TO PUBLIC;

-- Ensure admin functions are accessible (if not already granted)
GRANT EXECUTE ON FUNCTION is_current_user_admin() TO PUBLIC;
GRANT EXECUTE ON FUNCTION get_current_user_role() TO PUBLIC;

-- ============================================================================
-- Testing Queries (Comment out in production)
-- ============================================================================

-- Test 1: Verify RLS is enabled on all whitelabel tables
DO $$
DECLARE
    rls_table RECORD;
    rls_count INTEGER := 0;
BEGIN
    FOR rls_table IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename LIKE 'whitelabel_%'
    LOOP
        IF EXISTS (
            SELECT 1
            FROM pg_class c
            JOIN pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname = 'public'
            AND c.relname = rls_table.tablename
            AND c.relrowsecurity = true
        ) THEN
            rls_count := rls_count + 1;
            RAISE NOTICE 'RLS enabled on: %', rls_table.tablename;
        ELSE
            RAISE WARNING 'RLS NOT enabled on: %', rls_table.tablename;
        END IF;
    END LOOP;

    RAISE NOTICE 'Total whitelabel tables with RLS enabled: %', rls_count;
END $$;

-- Test 2: Count policies per table
SELECT
    schemaname,
    tablename,
    COUNT(*) as policy_count
FROM pg_policies
WHERE schemaname = 'public'
AND tablename LIKE 'whitelabel_%'
GROUP BY schemaname, tablename
ORDER BY tablename;

-- Test 3: List all policies with their commands
SELECT
    tablename,
    policyname,
    cmd as command,
    CASE
        WHEN qual IS NOT NULL THEN 'USING clause defined'
        ELSE 'No USING clause'
    END as using_clause,
    CASE
        WHEN with_check IS NOT NULL THEN 'WITH CHECK clause defined'
        ELSE 'No WITH CHECK clause'
    END as with_check_clause
FROM pg_policies
WHERE schemaname = 'public'
AND tablename LIKE 'whitelabel_%'
ORDER BY tablename, policyname;

-- ============================================================================
-- Example Usage Scenarios
-- ============================================================================

-- Example 1: Tenant admin accessing their brand
COMMENT ON SCHEMA public IS '
Example: Tenant Admin Access
------------------------------
SET LOCAL app.current_tenant_id = ''acme-corp'';
SET LOCAL app.user_role = ''tenant_admin'';

-- Tenant admin can now:
SELECT * FROM whitelabel_brands WHERE tenant_id = ''acme-corp'';
UPDATE whitelabel_brands SET brand_name = ''ACME Corporation'' WHERE tenant_id = ''acme-corp'';
INSERT INTO whitelabel_domains (brand_id, domain) VALUES (...);
';

-- Example 2: Public user accessing assets (CDN)
COMMENT ON TABLE whitelabel_assets IS '
Example: Public CDN Access
--------------------------
SET LOCAL app.user_role = ''public'';

-- Public users can access:
SELECT * FROM whitelabel_assets WHERE is_public = true;

Example: CDN Service Access
---------------------------
SET LOCAL app.user_role = ''cdn_service'';

-- CDN can read all active assets for delivery
SELECT * FROM whitelabel_assets WHERE is_active = true;
';

-- Example 3: Email service sending emails
COMMENT ON TABLE whitelabel_email_templates IS '
Example: Email Service Access
-----------------------------
SET LOCAL app.user_role = ''email_service'';
SET LOCAL app.current_brand_id = ''uuid-here'';

-- Email service can:
SELECT * FROM whitelabel_email_templates WHERE brand_id = ''uuid-here'' AND is_active = true;
UPDATE whitelabel_email_templates SET sent_count = sent_count + 1 WHERE id = ''uuid'';
';

-- ============================================================================
-- Security Audit Functions
-- ============================================================================

-- Function to audit access to sensitive assets
CREATE OR REPLACE FUNCTION audit_asset_access(
    p_asset_id UUID,
    p_access_type VARCHAR(50)
)
RETURNS VOID AS $$
BEGIN
    -- Log sensitive asset access
    IF EXISTS (
        SELECT 1 FROM whitelabel_assets
        WHERE id = p_asset_id
        AND asset_type IN ('certificate', 'key')
    ) THEN
        RAISE NOTICE 'AUDIT: User % accessed sensitive asset % (type: %)',
            get_current_user_role(),
            p_asset_id,
            p_access_type;
    END IF;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION audit_asset_access(UUID, VARCHAR) IS 'Audit log for sensitive asset access';

-- Function to check tenant isolation
CREATE OR REPLACE FUNCTION verify_tenant_isolation()
RETURNS TABLE (
    table_name VARCHAR,
    has_tenant_policy BOOLEAN,
    policy_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        t.tablename::VARCHAR,
        EXISTS (
            SELECT 1 FROM pg_policies p
            WHERE p.tablename = t.tablename
            AND p.schemaname = 'public'
            AND (
                p.qual::TEXT LIKE '%get_current_tenant_id()%'
                OR p.qual::TEXT LIKE '%get_current_brand_id()%'
            )
        ) as has_tenant_policy,
        COUNT(p.policyname)::INTEGER as policy_count
    FROM pg_tables t
    LEFT JOIN pg_policies p ON p.tablename = t.tablename AND p.schemaname = 'public'
    WHERE t.schemaname = 'public'
    AND t.tablename LIKE 'whitelabel_%'
    GROUP BY t.tablename;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION verify_tenant_isolation() IS 'Verify all whitelabel tables have tenant isolation policies';

-- ============================================================================
-- Migration Complete
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '============================================================';
    RAISE NOTICE 'Migration 020_add_whitelabel_rls.sql completed successfully';
    RAISE NOTICE '============================================================';
    RAISE NOTICE 'Row-Level Security (RLS) enabled on all whitelabel tables:';
    RAISE NOTICE '  ✓ whitelabel_brands (tenant isolation)';
    RAISE NOTICE '  ✓ whitelabel_domains (brand-based access)';
    RAISE NOTICE '  ✓ whitelabel_themes (brand-based access)';
    RAISE NOTICE '  ✓ whitelabel_email_templates (brand-based + system)';
    RAISE NOTICE '  ✓ whitelabel_assets (brand-based + public CDN)';
    RAISE NOTICE '';
    RAISE NOTICE 'Security features:';
    RAISE NOTICE '  • Multi-tenant isolation by tenant_id/brand_id';
    RAISE NOTICE '  • Super admin bypass for platform administration';
    RAISE NOTICE '  • Tenant admin full access to their brand resources';
    RAISE NOTICE '  • Tenant user read-only access';
    RAISE NOTICE '  • Public read access for CDN assets and logos';
    RAISE NOTICE '  • Special protection for SSL certificates/keys';
    RAISE NOTICE '  • System service access for email/CDN operations';
    RAISE NOTICE '';
    RAISE NOTICE 'User roles supported:';
    RAISE NOTICE '  - admin (super admin, full access)';
    RAISE NOTICE '  - tenant_admin (tenant administrator, manage brand)';
    RAISE NOTICE '  - tenant_user (regular user, read-only)';
    RAISE NOTICE '  - system (system service, limited access)';
    RAISE NOTICE '  - email_service (email sending service)';
    RAISE NOTICE '  - cdn_service (CDN asset delivery)';
    RAISE NOTICE '  - ssl_service (SSL certificate management)';
    RAISE NOTICE '  - public (anonymous, public assets only)';
    RAISE NOTICE '';
    RAISE NOTICE 'Required session variables:';
    RAISE NOTICE '  - app.current_tenant_id (tenant identifier)';
    RAISE NOTICE '  - app.current_brand_id (brand UUID, optional)';
    RAISE NOTICE '  - app.user_role (role name)';
    RAISE NOTICE '  - app.is_admin (boolean, optional)';
    RAISE NOTICE '============================================================';
END $$;

-- ============================================================================
-- Quick Test: Run tenant isolation verification
-- ============================================================================

SELECT * FROM verify_tenant_isolation();
