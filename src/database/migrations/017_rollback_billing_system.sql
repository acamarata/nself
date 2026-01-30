-- nself Database Migration: 017_rollback_billing_system.sql
-- Rollback of 015_create_billing_system.sql
--
-- Safely removes all billing system tables, views, and functions
-- in reverse dependency order with proper error handling.
--
-- Author: nself
-- Version: 0.9.0
-- Date: 2026-01-30
-- ============================================================================

BEGIN TRANSACTION;

-- ============================================================================
-- Drop Triggers
-- ============================================================================

DROP TRIGGER IF EXISTS update_billing_customers_updated_at ON billing_customers;
DROP TRIGGER IF EXISTS update_billing_plans_updated_at ON billing_plans;
DROP TRIGGER IF EXISTS update_billing_subscriptions_updated_at ON billing_subscriptions;
DROP TRIGGER IF EXISTS update_billing_quotas_updated_at ON billing_quotas;
DROP TRIGGER IF EXISTS update_billing_invoices_updated_at ON billing_invoices;
DROP TRIGGER IF EXISTS update_billing_payment_methods_updated_at ON billing_payment_methods;

-- ============================================================================
-- Drop Functions (in reverse dependency order)
-- ============================================================================

-- Functions that depend on tables
DROP FUNCTION IF EXISTS is_quota_exceeded(
    p_customer_id VARCHAR(255),
    p_service_name VARCHAR(100),
    p_requested_quantity NUMERIC
);

DROP FUNCTION IF EXISTS get_quota_usage(
    p_customer_id VARCHAR(255),
    p_service_name VARCHAR(100)
);

DROP FUNCTION IF EXISTS refresh_billing_usage_summary();

-- Generic trigger function (check if still used elsewhere before dropping)
DROP FUNCTION IF EXISTS update_updated_at_column();

-- ============================================================================
-- Drop Materialized Views (must be before dependent tables)
-- ============================================================================

DROP MATERIALIZED VIEW IF EXISTS billing_usage_daily_summary;

-- ============================================================================
-- Drop Tables (in reverse dependency order)
-- ============================================================================

-- Tables with no dependent tables
DROP TABLE IF EXISTS billing_events;
DROP TABLE IF EXISTS billing_payment_methods;
DROP TABLE IF EXISTS billing_invoices;
DROP TABLE IF EXISTS billing_usage_records;

-- Dependent tables (references to other tables)
DROP TABLE IF EXISTS billing_subscriptions;
DROP TABLE IF EXISTS billing_quotas;

-- Base tables (referenced by other tables)
DROP TABLE IF EXISTS billing_plans;
DROP TABLE IF EXISTS billing_customers;

-- ============================================================================
-- Log Migration Completion
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 017_rollback_billing_system.sql completed successfully';
    RAISE NOTICE 'Removed billing system with:';
    RAISE NOTICE '  - All billing tables';
    RAISE NOTICE '  - All triggers';
    RAISE NOTICE '  - All functions';
    RAISE NOTICE '  - All views';
END $$;

COMMIT;

-- ============================================================================
-- Verification (uncomment to check state after rollback)
-- ============================================================================

-- SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'billing_customers');
-- SELECT EXISTS (SELECT 1 FROM information_schema.views WHERE table_name = 'billing_usage_daily_summary');
-- SELECT EXISTS (SELECT 1 FROM information_schema.routines WHERE routine_name = 'get_quota_usage');
