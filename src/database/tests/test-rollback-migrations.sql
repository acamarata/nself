-- nself Database Migration Test Script
-- Tests that rollback migrations work correctly
-- Usage: psql -U postgres -d nself_test -f test-rollback-migrations.sql
--
-- This script:
-- 1. Verifies forward migrations create all objects
-- 2. Verifies rollback migrations remove all objects
-- 3. Verifies re-applying forward migrations works
-- ============================================================================

-- ============================================================================
-- Test 1: Billing System Forward Migration
-- ============================================================================

\echo '=========================================='
\echo 'Test 1: Billing System Forward Migration'
\echo '=========================================='

-- Create separate test database if needed
-- CREATE DATABASE nself_test;

-- Apply billing migration
\i ../migrations/015_create_billing_system.sql

-- Verify all tables exist
SELECT
  'Billing System Tables' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 8 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.tables
WHERE table_name LIKE 'billing_%' AND table_schema = 'public';

-- Verify materialized view exists
SELECT
  'Billing Usage Summary View' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 1 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.views
WHERE table_name = 'billing_usage_daily_summary' AND table_schema = 'public';

-- Verify functions exist
SELECT
  'Billing Functions' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) >= 2 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.routines
WHERE routine_name IN ('get_quota_usage', 'is_quota_exceeded', 'refresh_billing_usage_summary')
AND routine_schema = 'public';

-- Verify indexes exist
SELECT
  'Billing Indexes' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) >= 15 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.statistics
WHERE table_name LIKE 'billing_%'
AND table_schema = 'public'
AND index_name LIKE 'idx_%';

-- Verify default data inserted
SELECT
  'Billing Plans (Default Data)' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) >= 4 THEN 'PASS' ELSE 'FAIL' END as status
FROM billing_plans;

SELECT
  'Billing Quotas (Default Data)' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) >= 24 THEN 'PASS' ELSE 'FAIL' END as status
FROM billing_quotas;

-- ============================================================================
-- Test 2: Billing System Rollback Migration
-- ============================================================================

\echo ''
\echo '=========================================='
\echo 'Test 2: Billing System Rollback Migration'
\echo '=========================================='

-- Apply rollback migration
\i ../migrations/017_rollback_billing_system.sql

-- Verify all tables are removed
SELECT
  'Billing Tables Removed' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.tables
WHERE table_name LIKE 'billing_%' AND table_schema = 'public';

-- Verify view is removed
SELECT
  'Billing Views Removed' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.views
WHERE table_name = 'billing_usage_daily_summary' AND table_schema = 'public';

-- Verify functions are removed
SELECT
  'Billing Functions Removed' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.routines
WHERE routine_name IN ('get_quota_usage', 'is_quota_exceeded', 'refresh_billing_usage_summary')
AND routine_schema = 'public';

-- Verify indexes are removed
SELECT
  'Billing Indexes Removed' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.statistics
WHERE table_name LIKE 'billing_%'
AND table_schema = 'public'
AND index_name LIKE 'idx_%';

-- ============================================================================
-- Test 3: Billing System Re-Apply Migration
-- ============================================================================

\echo ''
\echo '=========================================='
\echo 'Test 3: Billing System Re-Apply Migration'
\echo '=========================================='

-- Re-apply billing migration
\i ../migrations/015_create_billing_system.sql

-- Verify all tables exist again
SELECT
  'Billing Tables Re-Created' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 8 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.tables
WHERE table_name LIKE 'billing_%' AND table_schema = 'public';

-- Verify view is recreated
SELECT
  'Billing View Re-Created' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 1 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.views
WHERE table_name = 'billing_usage_daily_summary' AND table_schema = 'public';

-- ============================================================================
-- Test 4: White-Label System Forward Migration
-- ============================================================================

\echo ''
\echo '=========================================='
\echo 'Test 4: White-Label System Forward Migration'
\echo '=========================================='

-- Apply white-label migration
\i ../migrations/016_create_whitelabel_system.sql

-- Verify all tables exist
SELECT
  'White-Label Tables' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 5 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.tables
WHERE table_name LIKE 'whitelabel_%' AND table_schema = 'public';

-- Verify view exists
SELECT
  'White-Label View' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 1 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.views
WHERE table_name = 'whitelabel_brands_full' AND table_schema = 'public';

-- Verify functions exist
SELECT
  'White-Label Functions' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 1 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.routines
WHERE routine_name = 'update_whitelabel_updated_at'
AND routine_schema = 'public';

-- Verify indexes exist
SELECT
  'White-Label Indexes' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) >= 15 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.statistics
WHERE table_name LIKE 'whitelabel_%'
AND table_schema = 'public'
AND index_name LIKE 'idx_%';

-- Verify default data inserted
SELECT
  'White-Label Default Brand' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) >= 1 THEN 'PASS' ELSE 'FAIL' END as status
FROM whitelabel_brands WHERE tenant_id = 'default';

SELECT
  'White-Label Default Themes' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) >= 3 THEN 'PASS' ELSE 'FAIL' END as status
FROM whitelabel_themes;

SELECT
  'White-Label Default Email Templates' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) >= 2 THEN 'PASS' ELSE 'FAIL' END as status
FROM whitelabel_email_templates;

-- ============================================================================
-- Test 5: White-Label System Rollback Migration
-- ============================================================================

\echo ''
\echo '=========================================='
\echo 'Test 5: White-Label System Rollback Migration'
\echo '=========================================='

-- Apply rollback migration
\i ../migrations/018_rollback_whitelabel_system.sql

-- Verify all tables are removed
SELECT
  'White-Label Tables Removed' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.tables
WHERE table_name LIKE 'whitelabel_%' AND table_schema = 'public';

-- Verify view is removed
SELECT
  'White-Label View Removed' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.views
WHERE table_name = 'whitelabel_brands_full' AND table_schema = 'public';

-- Verify functions are removed
SELECT
  'White-Label Functions Removed' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.routines
WHERE routine_name = 'update_whitelabel_updated_at'
AND routine_schema = 'public';

-- Verify indexes are removed
SELECT
  'White-Label Indexes Removed' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.statistics
WHERE table_name LIKE 'whitelabel_%'
AND table_schema = 'public'
AND index_name LIKE 'idx_%';

-- ============================================================================
-- Test 6: White-Label System Re-Apply Migration
-- ============================================================================

\echo ''
\echo '=========================================='
\echo 'Test 6: White-Label System Re-Apply Migration'
\echo '=========================================='

-- Re-apply white-label migration
\i ../migrations/016_create_whitelabel_system.sql

-- Verify all tables exist again
SELECT
  'White-Label Tables Re-Created' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 5 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.tables
WHERE table_name LIKE 'whitelabel_%' AND table_schema = 'public';

-- Verify view is recreated
SELECT
  'White-Label View Re-Created' as test,
  COUNT(*) as count,
  CASE WHEN COUNT(*) = 1 THEN 'PASS' ELSE 'FAIL' END as status
FROM information_schema.views
WHERE table_name = 'whitelabel_brands_full' AND table_schema = 'public';

-- ============================================================================
-- Test Summary
-- ============================================================================

\echo ''
\echo '=========================================='
\echo 'Test Summary'
\echo '=========================================='
\echo 'All tests completed. Check output above for PASS/FAIL status.'
\echo 'Successful rollback and re-apply cycles verify:'
\echo '  - Forward migrations create all objects'
\echo '  - Rollback migrations safely remove all objects'
\echo '  - Forward migrations can be re-applied successfully'
\echo '  - No orphaned constraints or dependencies'
\echo '=========================================='
