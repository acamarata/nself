-- ============================================================================
-- RLS Policy Testing Script
-- ============================================================================
-- Description: Comprehensive test suite for Row-Level Security policies
--              on billing and whitelabel tables
-- Usage: Run this after applying migrations 019 and 020
-- ============================================================================

-- ============================================================================
-- Setup Test Environment
-- ============================================================================

-- Create test roles
DO $$
BEGIN
    -- Drop roles if they exist (for idempotency)
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'test_admin') THEN
        DROP OWNED BY test_admin CASCADE;
        DROP ROLE test_admin;
    END IF;

    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'test_customer') THEN
        DROP OWNED BY test_customer CASCADE;
        DROP ROLE test_customer;
    END IF;

    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'test_tenant_admin') THEN
        DROP OWNED BY test_tenant_admin CASCADE;
        DROP ROLE test_tenant_admin;
    END IF;

    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'test_public') THEN
        DROP OWNED BY test_public CASCADE;
        DROP ROLE test_public;
    END IF;

    -- Create fresh test roles
    CREATE ROLE test_admin WITH LOGIN PASSWORD 'test123';
    CREATE ROLE test_customer WITH LOGIN PASSWORD 'test123';
    CREATE ROLE test_tenant_admin WITH LOGIN PASSWORD 'test123';
    CREATE ROLE test_public WITH LOGIN PASSWORD 'test123';
END $$;

-- Grant necessary permissions
GRANT USAGE ON SCHEMA public TO test_admin, test_customer, test_tenant_admin, test_public;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO test_admin, test_customer, test_tenant_admin, test_public;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO test_admin, test_customer, test_tenant_admin, test_public;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO test_admin, test_customer, test_tenant_admin, test_public;

-- ============================================================================
-- Insert Test Data
-- ============================================================================

-- Insert test billing customers
INSERT INTO billing_customers (customer_id, project_name, email, name)
VALUES
    ('test_cust_001', 'test_project_1', 'customer1@test.com', 'Test Customer 1'),
    ('test_cust_002', 'test_project_2', 'customer2@test.com', 'Test Customer 2')
ON CONFLICT (customer_id) DO NOTHING;

-- Insert test subscriptions
INSERT INTO billing_subscriptions (subscription_id, customer_id, plan_name, status, current_period_start, current_period_end)
VALUES
    ('test_sub_001', 'test_cust_001', 'pro', 'active', NOW(), NOW() + INTERVAL '30 days'),
    ('test_sub_002', 'test_cust_002', 'starter', 'active', NOW(), NOW() + INTERVAL '30 days')
ON CONFLICT (subscription_id) DO NOTHING;

-- Insert test usage records
INSERT INTO billing_usage_records (customer_id, service_name, quantity)
VALUES
    ('test_cust_001', 'api', 1000),
    ('test_cust_002', 'api', 500);

-- Insert test whitelabel brands
INSERT INTO whitelabel_brands (tenant_id, brand_name, is_active)
VALUES
    ('test_tenant_1', 'Test Brand 1', true),
    ('test_tenant_2', 'Test Brand 2', true)
ON CONFLICT (tenant_id) DO NOTHING;

-- Get brand IDs for subsequent inserts
DO $$
DECLARE
    brand1_id UUID;
    brand2_id UUID;
BEGIN
    SELECT id INTO brand1_id FROM whitelabel_brands WHERE tenant_id = 'test_tenant_1';
    SELECT id INTO brand2_id FROM whitelabel_brands WHERE tenant_id = 'test_tenant_2';

    -- Insert test domains
    INSERT INTO whitelabel_domains (brand_id, domain, status, is_active)
    VALUES
        (brand1_id, 'test1.example.com', 'active', true),
        (brand2_id, 'test2.example.com', 'active', true)
    ON CONFLICT (domain) DO NOTHING;

    -- Insert test themes
    INSERT INTO whitelabel_themes (brand_id, theme_name, display_name, is_active)
    VALUES
        (brand1_id, 'test_theme_1', 'Test Theme 1', true),
        (brand2_id, 'test_theme_2', 'Test Theme 2', true)
    ON CONFLICT (brand_id, theme_name) DO NOTHING;

    -- Insert test assets
    INSERT INTO whitelabel_assets (brand_id, asset_name, asset_type, file_name, file_path, is_public, is_active)
    VALUES
        (brand1_id, 'test_logo_1', 'logo', 'logo1.png', '/assets/logo1.png', true, true),
        (brand2_id, 'test_logo_2', 'logo', 'logo2.png', '/assets/logo2.png', false, true)
    ON CONFLICT DO NOTHING;
END $$;

-- ============================================================================
-- Test Suite 1: Billing RLS Policies
-- ============================================================================

\echo ''
\echo '============================================================'
\echo 'TEST SUITE 1: BILLING RLS POLICIES'
\echo '============================================================'
\echo ''

-- Test 1.1: Admin can see all customers
\echo 'Test 1.1: Admin access to all customers'
SET LOCAL app.user_role = 'admin';
SET LOCAL app.is_admin = true;

SELECT
    CASE
        WHEN COUNT(*) >= 2 THEN '✓ PASS: Admin can see all customers (' || COUNT(*) || ' found)'
        ELSE '✗ FAIL: Admin should see all customers'
    END as result
FROM billing_customers;

-- Test 1.2: Customer can only see their own data
\echo ''
\echo 'Test 1.2: Customer isolation (customer 1)'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';
SET LOCAL app.is_admin = false;

SELECT
    CASE
        WHEN COUNT(*) = 1 AND MAX(customer_id) = 'test_cust_001'
        THEN '✓ PASS: Customer can only see their own record'
        ELSE '✗ FAIL: Customer isolation broken'
    END as result
FROM billing_customers;

-- Test 1.3: Customer cannot see other customer's data
\echo ''
\echo 'Test 1.3: Customer cannot access other customer data'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';

SELECT
    CASE
        WHEN COUNT(*) = 0 THEN '✓ PASS: Customer cannot see other customers'
        ELSE '✗ FAIL: Customer can see other customers'
    END as result
FROM billing_customers
WHERE customer_id = 'test_cust_002';

-- Test 1.4: Customer can see their subscriptions
\echo ''
\echo 'Test 1.4: Customer can view their subscriptions'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';

SELECT
    CASE
        WHEN COUNT(*) = 1 AND MAX(customer_id) = 'test_cust_001'
        THEN '✓ PASS: Customer can view their subscriptions'
        ELSE '✗ FAIL: Customer cannot view subscriptions'
    END as result
FROM billing_subscriptions;

-- Test 1.5: Customer can see their usage records
\echo ''
\echo 'Test 1.5: Customer can view their usage records'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';

SELECT
    CASE
        WHEN COUNT(*) >= 1 AND MAX(customer_id) = 'test_cust_001'
        THEN '✓ PASS: Customer can view their usage'
        ELSE '✗ FAIL: Customer cannot view usage'
    END as result
FROM billing_usage_records;

-- Test 1.6: Everyone can see active billing plans
\echo ''
\echo 'Test 1.6: Public access to active billing plans'
SET LOCAL app.user_role = 'anonymous';
SET LOCAL app.current_customer_id = '';

SELECT
    CASE
        WHEN COUNT(*) >= 4 THEN '✓ PASS: Public can view active plans (' || COUNT(*) || ' plans)'
        ELSE '✗ FAIL: Public cannot view plans'
    END as result
FROM billing_plans
WHERE is_active = true;

-- Test 1.7: Customer can view quotas for their plan
\echo ''
\echo 'Test 1.7: Customer can view quotas for their active plan'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';

SELECT
    CASE
        WHEN COUNT(*) >= 1 THEN '✓ PASS: Customer can view plan quotas (' || COUNT(*) || ' quotas)'
        ELSE '✗ FAIL: Customer cannot view quotas'
    END as result
FROM billing_quotas q
JOIN billing_subscriptions s ON s.plan_name = q.plan_name
WHERE s.customer_id = 'test_cust_001' AND s.status = 'active';

-- Test 1.8: Customer cannot insert usage records (should fail or require system role)
\echo ''
\echo 'Test 1.8: Customer cannot insert usage for other customers'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';

DO $$
BEGIN
    BEGIN
        INSERT INTO billing_usage_records (customer_id, service_name, quantity)
        VALUES ('test_cust_002', 'api', 999);

        RAISE NOTICE '✗ FAIL: Customer should not insert usage for others';
    EXCEPTION
        WHEN insufficient_privilege OR check_violation THEN
            RAISE NOTICE '✓ PASS: Customer blocked from inserting usage for others';
    END;
END $$;

-- ============================================================================
-- Test Suite 2: Whitelabel RLS Policies
-- ============================================================================

\echo ''
\echo '============================================================'
\echo 'TEST SUITE 2: WHITELABEL RLS POLICIES'
\echo '============================================================'
\echo ''

-- Test 2.1: Admin can see all brands
\echo 'Test 2.1: Admin access to all brands'
SET LOCAL app.user_role = 'admin';
SET LOCAL app.is_admin = true;

SELECT
    CASE
        WHEN COUNT(*) >= 2 THEN '✓ PASS: Admin can see all brands (' || COUNT(*) || ' found)'
        ELSE '✗ FAIL: Admin should see all brands'
    END as result
FROM whitelabel_brands;

-- Test 2.2: Tenant admin can see their own brand
\echo ''
\echo 'Test 2.2: Tenant admin can view their brand'
SET LOCAL app.current_tenant_id = 'test_tenant_1';
SET LOCAL app.user_role = 'tenant_admin';
SET LOCAL app.is_admin = false;

SELECT
    CASE
        WHEN COUNT(*) = 1 AND MAX(tenant_id) = 'test_tenant_1'
        THEN '✓ PASS: Tenant admin can view their brand'
        ELSE '✗ FAIL: Tenant admin cannot view brand'
    END as result
FROM whitelabel_brands;

-- Test 2.3: Tenant admin cannot see other tenants
\echo ''
\echo 'Test 2.3: Tenant admin cannot view other tenant brands'
SET LOCAL app.current_tenant_id = 'test_tenant_1';
SET LOCAL app.user_role = 'tenant_admin';

SELECT
    CASE
        WHEN COUNT(*) = 0 THEN '✓ PASS: Tenant admin cannot see other brands'
        ELSE '✗ FAIL: Tenant isolation broken'
    END as result
FROM whitelabel_brands
WHERE tenant_id = 'test_tenant_2';

-- Test 2.4: Tenant admin can see their domains
\echo ''
\echo 'Test 2.4: Tenant admin can view their domains'
SET LOCAL app.current_tenant_id = 'test_tenant_1';
SET LOCAL app.user_role = 'tenant_admin';

SELECT
    CASE
        WHEN COUNT(*) >= 1 THEN '✓ PASS: Tenant admin can view domains (' || COUNT(*) || ' domains)'
        ELSE '✗ FAIL: Tenant admin cannot view domains'
    END as result
FROM whitelabel_domains d
JOIN whitelabel_brands b ON b.id = d.brand_id
WHERE b.tenant_id = 'test_tenant_1';

-- Test 2.5: Public can see active domains
\echo ''
\echo 'Test 2.5: Public can view active domains'
SET LOCAL app.user_role = 'public';
SET LOCAL app.current_tenant_id = '';

SELECT
    CASE
        WHEN COUNT(*) >= 1 THEN '✓ PASS: Public can view active domains (' || COUNT(*) || ' domains)'
        ELSE '✗ FAIL: Public cannot view domains'
    END as result
FROM whitelabel_domains
WHERE status = 'active' AND is_active = true;

-- Test 2.6: Public can see public assets
\echo ''
\echo 'Test 2.6: Public can view public assets (CDN)'
SET LOCAL app.user_role = 'public';

SELECT
    CASE
        WHEN COUNT(*) >= 1 THEN '✓ PASS: Public can view public assets (' || COUNT(*) || ' assets)'
        ELSE '✗ FAIL: Public cannot view assets'
    END as result
FROM whitelabel_assets
WHERE is_public = true AND is_active = true;

-- Test 2.7: Public cannot see private assets
\echo ''
\echo 'Test 2.7: Public cannot view private assets'
SET LOCAL app.user_role = 'public';

SELECT
    CASE
        WHEN COUNT(*) = 0 THEN '✓ PASS: Public cannot see private assets'
        ELSE '✗ FAIL: Private assets are exposed'
    END as result
FROM whitelabel_assets
WHERE is_public = false;

-- Test 2.8: Tenant admin can see system themes
\echo ''
\echo 'Test 2.8: Tenant admin can view system themes'
SET LOCAL app.current_tenant_id = 'test_tenant_1';
SET LOCAL app.user_role = 'tenant_admin';

SELECT
    CASE
        WHEN COUNT(*) >= 1 THEN '✓ PASS: Can view system themes (' || COUNT(*) || ' themes)'
        ELSE '✗ FAIL: Cannot view system themes'
    END as result
FROM whitelabel_themes
WHERE is_system = true AND is_active = true;

-- ============================================================================
-- Test Suite 3: Helper Functions
-- ============================================================================

\echo ''
\echo '============================================================'
\echo 'TEST SUITE 3: HELPER FUNCTIONS'
\echo '============================================================'
\echo ''

-- Test 3.1: get_current_customer_id function
\echo 'Test 3.1: get_current_customer_id() function'
SET LOCAL app.current_customer_id = 'test_cust_001';

SELECT
    CASE
        WHEN get_current_customer_id() = 'test_cust_001'
        THEN '✓ PASS: Function returns correct customer ID'
        ELSE '✗ FAIL: Function returns: ' || COALESCE(get_current_customer_id(), 'NULL')
    END as result;

-- Test 3.2: is_current_user_admin function
\echo ''
\echo 'Test 3.2: is_current_user_admin() function'
SET LOCAL app.is_admin = true;

SELECT
    CASE
        WHEN is_current_user_admin() = true
        THEN '✓ PASS: Admin flag detected correctly'
        ELSE '✗ FAIL: Admin flag not detected'
    END as result;

-- Test 3.3: get_current_tenant_id function
\echo ''
\echo 'Test 3.3: get_current_tenant_id() function'
SET LOCAL app.current_tenant_id = 'test_tenant_1';

SELECT
    CASE
        WHEN get_current_tenant_id() = 'test_tenant_1'
        THEN '✓ PASS: Function returns correct tenant ID'
        ELSE '✗ FAIL: Function returns: ' || COALESCE(get_current_tenant_id(), 'NULL')
    END as result;

-- Test 3.4: get_quota_usage function
\echo ''
\echo 'Test 3.4: get_quota_usage() function'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';
SET LOCAL app.is_admin = false;

SELECT
    CASE
        WHEN EXISTS (
            SELECT 1 FROM get_quota_usage('test_cust_001', 'api')
        )
        THEN '✓ PASS: Quota usage function works'
        ELSE '✗ FAIL: Quota usage function failed'
    END as result;

-- Test 3.5: is_quota_exceeded function
\echo ''
\echo 'Test 3.5: is_quota_exceeded() function'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';

SELECT
    CASE
        WHEN is_quota_exceeded('test_cust_001', 'api', 1) IS NOT NULL
        THEN '✓ PASS: Quota exceeded function works (result: ' || is_quota_exceeded('test_cust_001', 'api', 1)::TEXT || ')'
        ELSE '✗ FAIL: Quota exceeded function failed'
    END as result;

-- ============================================================================
-- Test Suite 4: Cross-Table Relationships
-- ============================================================================

\echo ''
\echo '============================================================'
\echo 'TEST SUITE 4: CROSS-TABLE RELATIONSHIPS'
\echo '============================================================'
\echo ''

-- Test 4.1: Customer can join subscriptions with plans
\echo 'Test 4.1: Customer can join subscriptions with plans'
SET LOCAL app.current_customer_id = 'test_cust_001';
SET LOCAL app.user_role = 'customer';

SELECT
    CASE
        WHEN COUNT(*) >= 1
        THEN '✓ PASS: Customer can join subscriptions with plans'
        ELSE '✗ FAIL: Join failed'
    END as result
FROM billing_subscriptions s
JOIN billing_plans p ON p.plan_name = s.plan_name;

-- Test 4.2: Tenant admin can join brands with domains
\echo ''
\echo 'Test 4.2: Tenant admin can join brands with domains'
SET LOCAL app.current_tenant_id = 'test_tenant_1';
SET LOCAL app.user_role = 'tenant_admin';

SELECT
    CASE
        WHEN COUNT(*) >= 1
        THEN '✓ PASS: Tenant admin can join brands with domains'
        ELSE '✗ FAIL: Join failed'
    END as result
FROM whitelabel_brands b
JOIN whitelabel_domains d ON d.brand_id = b.id;

-- Test 4.3: View whitelabel_brands_full respects RLS
\echo ''
\echo 'Test 4.3: whitelabel_brands_full view respects RLS'
SET LOCAL app.current_tenant_id = 'test_tenant_1';
SET LOCAL app.user_role = 'tenant_admin';

SELECT
    CASE
        WHEN COUNT(*) = 1 AND MAX(tenant_id) = 'test_tenant_1'
        THEN '✓ PASS: View respects RLS policies'
        ELSE '✗ FAIL: View bypasses RLS'
    END as result
FROM whitelabel_brands_full;

-- ============================================================================
-- Summary Report
-- ============================================================================

\echo ''
\echo '============================================================'
\echo 'RLS POLICY TEST SUMMARY'
\echo '============================================================'
\echo ''

-- Count total policies
SELECT
    COUNT(DISTINCT tablename) as tables_with_rls,
    COUNT(*) as total_policies
FROM pg_policies
WHERE schemaname = 'public'
AND (tablename LIKE 'billing_%' OR tablename LIKE 'whitelabel_%');

-- List policy coverage
\echo ''
\echo 'Policy Coverage by Table:'
SELECT
    tablename as table_name,
    COUNT(*) as policy_count,
    string_agg(DISTINCT cmd::TEXT, ', ') as commands_covered
FROM pg_policies
WHERE schemaname = 'public'
AND (tablename LIKE 'billing_%' OR tablename LIKE 'whitelabel_%')
GROUP BY tablename
ORDER BY tablename;

-- ============================================================================
-- Cleanup
-- ============================================================================

\echo ''
\echo '============================================================'
\echo 'CLEANUP'
\echo '============================================================'
\echo ''

-- Note: Uncomment to cleanup test data
-- DELETE FROM billing_usage_records WHERE customer_id LIKE 'test_cust_%';
-- DELETE FROM billing_subscriptions WHERE customer_id LIKE 'test_cust_%';
-- DELETE FROM billing_customers WHERE customer_id LIKE 'test_cust_%';
-- DELETE FROM whitelabel_assets WHERE asset_name LIKE 'test_%';
-- DELETE FROM whitelabel_themes WHERE theme_name LIKE 'test_%';
-- DELETE FROM whitelabel_domains WHERE domain LIKE 'test%';
-- DELETE FROM whitelabel_brands WHERE tenant_id LIKE 'test_tenant_%';

\echo 'Test data preserved for manual inspection.'
\echo 'Run cleanup queries manually if needed.'
\echo ''
\echo '============================================================'
\echo 'TEST SUITE COMPLETE'
\echo '============================================================'
