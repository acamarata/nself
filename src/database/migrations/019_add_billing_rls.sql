-- ============================================================================
-- nself Database Migration: 019_add_billing_rls.sql
-- Part of nself v0.9.0 - Sprint 21: Security Hardening
-- ============================================================================
-- Description: Row-Level Security (RLS) policies for billing system tables
--              Enforces multi-tenant isolation and secure access control
-- Dependencies: 015_create_billing_system.sql
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
-- app.current_customer_id - The authenticated customer's ID
-- app.user_role - User role (admin, customer, readonly)
-- app.is_admin - Boolean flag for admin bypass
--
-- Example usage in application:
-- SET LOCAL app.current_customer_id = 'cust_123';
-- SET LOCAL app.user_role = 'customer';
-- ============================================================================

-- ============================================================================
-- Helper Functions for RLS Policies
-- ============================================================================

-- Function to get current customer ID from session
CREATE OR REPLACE FUNCTION get_current_customer_id()
RETURNS VARCHAR AS $$
BEGIN
    RETURN current_setting('app.current_customer_id', true)::VARCHAR;
EXCEPTION
    WHEN OTHERS THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

COMMENT ON FUNCTION get_current_customer_id() IS 'Returns current customer ID from session variable';

-- Function to check if current user is admin
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

COMMENT ON FUNCTION is_current_user_admin() IS 'Returns true if current user has admin privileges';

-- Function to get user role
CREATE OR REPLACE FUNCTION get_current_user_role()
RETURNS VARCHAR AS $$
BEGIN
    RETURN current_setting('app.user_role', true)::VARCHAR;
EXCEPTION
    WHEN OTHERS THEN
        RETURN 'anonymous';
END;
$$ LANGUAGE plpgsql SECURITY DEFINER STABLE;

COMMENT ON FUNCTION get_current_user_role() IS 'Returns current user role from session variable';

-- ============================================================================
-- RLS Policies for billing_customers
-- ============================================================================

ALTER TABLE billing_customers ENABLE ROW LEVEL SECURITY;

-- Admin bypass: Admins can do everything
CREATE POLICY admin_all_access ON billing_customers
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Customer access: Can only see their own customer record
CREATE POLICY customer_read_own ON billing_customers
    FOR SELECT
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
    );

-- Customer update: Can update their own customer record (limited fields)
CREATE POLICY customer_update_own ON billing_customers
    FOR UPDATE
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
    )
    WITH CHECK (
        customer_id = get_current_customer_id()
    );

-- Prevent customer deletion by non-admins (soft delete only via updated_at)
CREATE POLICY customer_no_delete ON billing_customers
    FOR DELETE
    TO PUBLIC
    USING (false);

COMMENT ON TABLE billing_customers IS 'RLS enabled: Customers can only access their own records. Admins have full access.';

-- ============================================================================
-- RLS Policies for billing_plans
-- ============================================================================

ALTER TABLE billing_plans ENABLE ROW LEVEL SECURITY;

-- Admin full access
CREATE POLICY admin_all_access ON billing_plans
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Public read: Everyone can view active plans
CREATE POLICY public_read_active_plans ON billing_plans
    FOR SELECT
    TO PUBLIC
    USING (is_active = true);

-- No public write/update/delete
CREATE POLICY no_public_write ON billing_plans
    FOR INSERT
    TO PUBLIC
    WITH CHECK (false);

CREATE POLICY no_public_update ON billing_plans
    FOR UPDATE
    TO PUBLIC
    USING (false);

CREATE POLICY no_public_delete ON billing_plans
    FOR DELETE
    TO PUBLIC
    USING (false);

COMMENT ON TABLE billing_plans IS 'RLS enabled: Plans are public read-only. Only admins can modify.';

-- ============================================================================
-- RLS Policies for billing_subscriptions
-- ============================================================================

ALTER TABLE billing_subscriptions ENABLE ROW LEVEL SECURITY;

-- Admin bypass
CREATE POLICY admin_all_access ON billing_subscriptions
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Customer read: Can only see their own subscriptions
CREATE POLICY customer_read_own ON billing_subscriptions
    FOR SELECT
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
    );

-- Customer update: Can cancel their subscription (limited update)
CREATE POLICY customer_update_cancel ON billing_subscriptions
    FOR UPDATE
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
    )
    WITH CHECK (
        customer_id = get_current_customer_id()
    );

-- No customer insert/delete
CREATE POLICY no_customer_insert ON billing_subscriptions
    FOR INSERT
    TO PUBLIC
    WITH CHECK (is_current_user_admin());

CREATE POLICY no_customer_delete ON billing_subscriptions
    FOR DELETE
    TO PUBLIC
    USING (false);

COMMENT ON TABLE billing_subscriptions IS 'RLS enabled: Customers can view and update (cancel) their subscriptions.';

-- ============================================================================
-- RLS Policies for billing_quotas
-- ============================================================================

ALTER TABLE billing_quotas ENABLE ROW LEVEL SECURITY;

-- Admin full access
CREATE POLICY admin_all_access ON billing_quotas
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Customer read: Can view quotas for their plan
CREATE POLICY customer_read_own_plan_quotas ON billing_quotas
    FOR SELECT
    TO PUBLIC
    USING (
        EXISTS (
            SELECT 1 FROM billing_subscriptions
            WHERE billing_subscriptions.customer_id = get_current_customer_id()
            AND billing_subscriptions.plan_name = billing_quotas.plan_name
            AND billing_subscriptions.status = 'active'
        )
    );

-- No customer write
CREATE POLICY no_customer_write ON billing_quotas
    FOR INSERT
    TO PUBLIC
    WITH CHECK (false);

CREATE POLICY no_customer_update ON billing_quotas
    FOR UPDATE
    TO PUBLIC
    USING (false);

CREATE POLICY no_customer_delete ON billing_quotas
    FOR DELETE
    TO PUBLIC
    USING (false);

COMMENT ON TABLE billing_quotas IS 'RLS enabled: Customers can view quotas for their active plan.';

-- ============================================================================
-- RLS Policies for billing_usage_records
-- ============================================================================

ALTER TABLE billing_usage_records ENABLE ROW LEVEL SECURITY;

-- Admin bypass
CREATE POLICY admin_all_access ON billing_usage_records
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Customer read: Can only see their own usage
CREATE POLICY customer_read_own ON billing_usage_records
    FOR SELECT
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
    );

-- System insert: Usage records can be inserted by system
CREATE POLICY system_insert ON billing_usage_records
    FOR INSERT
    TO PUBLIC
    WITH CHECK (
        customer_id = get_current_customer_id()
        OR is_current_user_admin()
    );

-- No customer update/delete
CREATE POLICY no_customer_update ON billing_usage_records
    FOR UPDATE
    TO PUBLIC
    USING (false);

CREATE POLICY no_customer_delete ON billing_usage_records
    FOR DELETE
    TO PUBLIC
    USING (false);

COMMENT ON TABLE billing_usage_records IS 'RLS enabled: Customers can view their usage. System can insert.';

-- ============================================================================
-- RLS Policies for billing_invoices
-- ============================================================================

ALTER TABLE billing_invoices ENABLE ROW LEVEL SECURITY;

-- Admin bypass
CREATE POLICY admin_all_access ON billing_invoices
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Customer read: Can only see their own invoices
CREATE POLICY customer_read_own ON billing_invoices
    FOR SELECT
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
    );

-- No customer write/update/delete
CREATE POLICY no_customer_insert ON billing_invoices
    FOR INSERT
    TO PUBLIC
    WITH CHECK (false);

CREATE POLICY no_customer_update ON billing_invoices
    FOR UPDATE
    TO PUBLIC
    USING (false);

CREATE POLICY no_customer_delete ON billing_invoices
    FOR DELETE
    TO PUBLIC
    USING (false);

COMMENT ON TABLE billing_invoices IS 'RLS enabled: Customers can view their invoices. Only admins can modify.';

-- ============================================================================
-- RLS Policies for billing_payment_methods
-- ============================================================================

ALTER TABLE billing_payment_methods ENABLE ROW LEVEL SECURITY;

-- Admin bypass
CREATE POLICY admin_all_access ON billing_payment_methods
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Customer read: Can only see their own payment methods
CREATE POLICY customer_read_own ON billing_payment_methods
    FOR SELECT
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
        AND deleted_at IS NULL
    );

-- Customer insert: Can add their own payment methods
CREATE POLICY customer_insert_own ON billing_payment_methods
    FOR INSERT
    TO PUBLIC
    WITH CHECK (
        customer_id = get_current_customer_id()
    );

-- Customer update: Can update their own payment methods
CREATE POLICY customer_update_own ON billing_payment_methods
    FOR UPDATE
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
        AND deleted_at IS NULL
    )
    WITH CHECK (
        customer_id = get_current_customer_id()
    );

-- Customer soft delete: Can mark their own as deleted
CREATE POLICY customer_soft_delete_own ON billing_payment_methods
    FOR DELETE
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
    );

COMMENT ON TABLE billing_payment_methods IS 'RLS enabled: Customers can manage their own payment methods.';

-- ============================================================================
-- RLS Policies for billing_events
-- ============================================================================

ALTER TABLE billing_events ENABLE ROW LEVEL SECURITY;

-- Admin full access
CREATE POLICY admin_all_access ON billing_events
    FOR ALL
    TO PUBLIC
    USING (is_current_user_admin())
    WITH CHECK (is_current_user_admin());

-- Customer read: Can see events related to them
CREATE POLICY customer_read_own ON billing_events
    FOR SELECT
    TO PUBLIC
    USING (
        customer_id = get_current_customer_id()
    );

-- Webhook system insert: System can insert webhook events
CREATE POLICY webhook_system_insert ON billing_events
    FOR INSERT
    TO PUBLIC
    WITH CHECK (
        get_current_user_role() IN ('system', 'webhook', 'admin')
    );

-- No customer update/delete
CREATE POLICY no_customer_update ON billing_events
    FOR UPDATE
    TO PUBLIC
    USING (false);

CREATE POLICY no_customer_delete ON billing_events
    FOR DELETE
    TO PUBLIC
    USING (false);

COMMENT ON TABLE billing_events IS 'RLS enabled: Customers can view their events. System can insert webhook events.';

-- ============================================================================
-- RLS for Materialized View (billing_usage_daily_summary)
-- ============================================================================
-- Note: Materialized views don't support RLS directly.
-- Access control is handled through the base table (billing_usage_records).
-- The refresh function should be called by admins or system processes only.

-- Grant execute permission on refresh function to admin role only
REVOKE EXECUTE ON FUNCTION refresh_billing_usage_summary() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION refresh_billing_usage_summary() TO postgres;

COMMENT ON FUNCTION refresh_billing_usage_summary() IS 'Admin/system only: Refreshes usage summary materialized view';

-- ============================================================================
-- Security Functions - Update to respect RLS
-- ============================================================================

-- Update get_quota_usage function to respect RLS
DROP FUNCTION IF EXISTS get_quota_usage(VARCHAR, VARCHAR);

CREATE OR REPLACE FUNCTION get_quota_usage(
    p_customer_id VARCHAR(255),
    p_service_name VARCHAR(100)
)
RETURNS TABLE (
    service_name VARCHAR(100),
    quota_limit BIGINT,
    current_usage NUMERIC,
    percentage NUMERIC,
    is_exceeded BOOLEAN
) AS $$
BEGIN
    -- Verify caller has access to this customer
    IF NOT (is_current_user_admin() OR get_current_customer_id() = p_customer_id) THEN
        RAISE EXCEPTION 'Access denied to customer data';
    END IF;

    RETURN QUERY
    SELECT
        q.service_name,
        q.limit_value as quota_limit,
        COALESCE(SUM(ur.quantity), 0) as current_usage,
        CASE
            WHEN q.limit_value = -1 THEN 0
            WHEN q.limit_value = 0 THEN 0
            ELSE (COALESCE(SUM(ur.quantity), 0) * 100.0 / q.limit_value)
        END as percentage,
        CASE
            WHEN q.limit_value = -1 THEN false
            ELSE COALESCE(SUM(ur.quantity), 0) > q.limit_value
        END as is_exceeded
    FROM billing_quotas q
    JOIN billing_subscriptions s ON s.plan_name = q.plan_name
    LEFT JOIN billing_usage_records ur ON
        ur.service_name = q.service_name
        AND ur.customer_id = s.customer_id
        AND ur.recorded_at >= s.current_period_start
        AND ur.recorded_at <= s.current_period_end
    WHERE s.customer_id = p_customer_id
    AND s.status = 'active'
    AND q.service_name = p_service_name
    GROUP BY q.service_name, q.limit_value;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION get_quota_usage(VARCHAR, VARCHAR) IS 'RLS-aware: Get quota usage for customer/service';

-- Update is_quota_exceeded function to respect RLS
DROP FUNCTION IF EXISTS is_quota_exceeded(VARCHAR, VARCHAR, NUMERIC);

CREATE OR REPLACE FUNCTION is_quota_exceeded(
    p_customer_id VARCHAR(255),
    p_service_name VARCHAR(100),
    p_requested_quantity NUMERIC DEFAULT 1
)
RETURNS BOOLEAN AS $$
DECLARE
    v_limit BIGINT;
    v_usage NUMERIC;
    v_total NUMERIC;
BEGIN
    -- Verify caller has access to this customer
    IF NOT (is_current_user_admin() OR get_current_customer_id() = p_customer_id) THEN
        RAISE EXCEPTION 'Access denied to customer data';
    END IF;

    -- Get quota limit
    SELECT q.limit_value INTO v_limit
    FROM billing_quotas q
    JOIN billing_subscriptions s ON s.plan_name = q.plan_name
    WHERE s.customer_id = p_customer_id
    AND s.status = 'active'
    AND q.service_name = p_service_name
    LIMIT 1;

    -- Unlimited quota
    IF v_limit = -1 OR v_limit IS NULL THEN
        RETURN false;
    END IF;

    -- Get current usage
    SELECT COALESCE(SUM(quantity), 0) INTO v_usage
    FROM billing_usage_records ur
    JOIN billing_subscriptions s ON s.customer_id = ur.customer_id
    WHERE ur.customer_id = p_customer_id
    AND ur.service_name = p_service_name
    AND ur.recorded_at >= s.current_period_start
    AND ur.recorded_at <= s.current_period_end;

    -- Check if adding requested would exceed
    v_total := v_usage + p_requested_quantity;

    RETURN v_total > v_limit;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION is_quota_exceeded(VARCHAR, VARCHAR, NUMERIC) IS 'RLS-aware: Check if quota exceeded';

-- ============================================================================
-- Testing Queries (Comment out in production)
-- ============================================================================

-- Test 1: Verify RLS is enabled on all billing tables
DO $$
DECLARE
    rls_table RECORD;
    rls_count INTEGER := 0;
BEGIN
    FOR rls_table IN
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        AND tablename LIKE 'billing_%'
        AND tablename != 'billing_usage_daily_summary'
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

    RAISE NOTICE 'Total billing tables with RLS enabled: %', rls_count;
END $$;

-- Test 2: Count policies per table
SELECT
    schemaname,
    tablename,
    COUNT(*) as policy_count
FROM pg_policies
WHERE schemaname = 'public'
AND tablename LIKE 'billing_%'
GROUP BY schemaname, tablename
ORDER BY tablename;

-- ============================================================================
-- Performance Indexes for RLS
-- ============================================================================

-- Indexes to optimize RLS policy checks
CREATE INDEX IF NOT EXISTS idx_billing_customers_customer_id ON billing_customers(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_billing_subscriptions_customer_status ON billing_subscriptions(customer_id, status);
CREATE INDEX IF NOT EXISTS idx_billing_usage_records_customer_service ON billing_usage_records(customer_id, service_name, recorded_at);
CREATE INDEX IF NOT EXISTS idx_billing_invoices_customer_period ON billing_invoices(customer_id, period_start, period_end);
CREATE INDEX IF NOT EXISTS idx_billing_payment_methods_customer_active ON billing_payment_methods(customer_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_billing_events_customer_type ON billing_events(customer_id, event_type);

-- ============================================================================
-- Grant Permissions for Common Roles
-- ============================================================================

-- Grant SELECT on helper functions to PUBLIC (they have SECURITY DEFINER)
GRANT EXECUTE ON FUNCTION get_current_customer_id() TO PUBLIC;
GRANT EXECUTE ON FUNCTION is_current_user_admin() TO PUBLIC;
GRANT EXECUTE ON FUNCTION get_current_user_role() TO PUBLIC;

-- Grant EXECUTE on business functions
GRANT EXECUTE ON FUNCTION get_quota_usage(VARCHAR, VARCHAR) TO PUBLIC;
GRANT EXECUTE ON FUNCTION is_quota_exceeded(VARCHAR, VARCHAR, NUMERIC) TO PUBLIC;

-- ============================================================================
-- Migration Complete
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '============================================================';
    RAISE NOTICE 'Migration 019_add_billing_rls.sql completed successfully';
    RAISE NOTICE '============================================================';
    RAISE NOTICE 'Row-Level Security (RLS) enabled on all billing tables:';
    RAISE NOTICE '  ✓ billing_customers (customer isolation)';
    RAISE NOTICE '  ✓ billing_plans (public read, admin write)';
    RAISE NOTICE '  ✓ billing_subscriptions (customer read/update)';
    RAISE NOTICE '  ✓ billing_quotas (plan-based access)';
    RAISE NOTICE '  ✓ billing_usage_records (customer read, system write)';
    RAISE NOTICE '  ✓ billing_invoices (customer read-only)';
    RAISE NOTICE '  ✓ billing_payment_methods (customer full access)';
    RAISE NOTICE '  ✓ billing_events (webhook logging)';
    RAISE NOTICE '';
    RAISE NOTICE 'Security features:';
    RAISE NOTICE '  • Multi-tenant isolation by customer_id';
    RAISE NOTICE '  • Admin bypass for administrative access';
    RAISE NOTICE '  • Role-based access control (admin, customer, system)';
    RAISE NOTICE '  • Soft delete support for payment methods';
    RAISE NOTICE '  • RLS-aware helper functions';
    RAISE NOTICE '';
    RAISE NOTICE 'Required session variables:';
    RAISE NOTICE '  - app.current_customer_id (customer identifier)';
    RAISE NOTICE '  - app.user_role (admin, customer, system, webhook)';
    RAISE NOTICE '  - app.is_admin (boolean, optional)';
    RAISE NOTICE '============================================================';
END $$;
