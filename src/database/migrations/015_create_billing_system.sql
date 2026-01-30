-- nself Database Migration: 015_create_billing_system.sql
-- Part of nself v0.9.0 - Sprint 13: Billing Integration & Usage Tracking
--
-- Creates comprehensive billing system with Stripe integration, usage tracking,
-- quotas, and invoice management.
--
-- Author: nself
-- Version: 0.9.0
-- Date: 2026-01-29

-- ============================================================================
-- Billing Customers
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_customers (
    customer_id VARCHAR(255) PRIMARY KEY,
    project_name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    name VARCHAR(255),
    company VARCHAR(255),
    stripe_customer_id VARCHAR(255) UNIQUE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_billing_customers_project ON billing_customers(project_name);
CREATE INDEX idx_billing_customers_email ON billing_customers(email);
CREATE INDEX idx_billing_customers_stripe ON billing_customers(stripe_customer_id);

COMMENT ON TABLE billing_customers IS 'Customer accounts for billing';
COMMENT ON COLUMN billing_customers.customer_id IS 'Internal customer identifier';
COMMENT ON COLUMN billing_customers.stripe_customer_id IS 'Stripe customer ID for payment processing';
COMMENT ON COLUMN billing_customers.metadata IS 'Additional customer metadata (JSON)';

-- ============================================================================
-- Billing Plans
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_plans (
    plan_id SERIAL PRIMARY KEY,
    plan_name VARCHAR(100) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    price_monthly DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    price_yearly DECIMAL(10, 2),
    stripe_price_id_monthly VARCHAR(255),
    stripe_price_id_yearly VARCHAR(255),
    features JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_billing_plans_name ON billing_plans(plan_name);
CREATE INDEX idx_billing_plans_active ON billing_plans(is_active);

COMMENT ON TABLE billing_plans IS 'Available subscription plans';
COMMENT ON COLUMN billing_plans.features IS 'List of plan features (JSON array)';
COMMENT ON COLUMN billing_plans.sort_order IS 'Display order (lower = shown first)';

-- Insert default plans
INSERT INTO billing_plans (plan_name, display_name, description, price_monthly, price_yearly, features, sort_order)
VALUES
    ('free', 'Free', 'Perfect for development and testing', 0.00, 0.00,
     '["10,000 API requests/month", "1GB storage", "Community support"]', 1),
    ('starter', 'Starter', 'For small projects and startups', 29.00, 290.00,
     '["100,000 API requests/month", "10GB storage", "Email support", "Custom domains"]', 2),
    ('pro', 'Professional', 'For production applications', 99.00, 990.00,
     '["1M API requests/month", "100GB storage", "Priority support", "Advanced features", "99.9% SLA"]', 3),
    ('enterprise', 'Enterprise', 'For large-scale deployments', 499.00, 4990.00,
     '["Unlimited API requests", "Unlimited storage", "24/7 dedicated support", "Custom contracts", "99.99% SLA"]', 4)
ON CONFLICT (plan_name) DO NOTHING;

-- ============================================================================
-- Billing Subscriptions
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_subscriptions (
    subscription_id VARCHAR(255) PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL REFERENCES billing_customers(customer_id) ON DELETE CASCADE,
    plan_name VARCHAR(100) NOT NULL REFERENCES billing_plans(plan_name),
    stripe_subscription_id VARCHAR(255) UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    -- Status: active, trialing, past_due, canceled, unpaid
    billing_cycle VARCHAR(20) DEFAULT 'monthly',
    -- Cycle: monthly, yearly
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    cancel_at_period_end BOOLEAN DEFAULT false,
    canceled_at TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_billing_subscriptions_customer ON billing_subscriptions(customer_id);
CREATE INDEX idx_billing_subscriptions_plan ON billing_subscriptions(plan_name);
CREATE INDEX idx_billing_subscriptions_status ON billing_subscriptions(status);
CREATE INDEX idx_billing_subscriptions_stripe ON billing_subscriptions(stripe_subscription_id);
CREATE INDEX idx_billing_subscriptions_period ON billing_subscriptions(current_period_start, current_period_end);

COMMENT ON TABLE billing_subscriptions IS 'Customer subscription records';
COMMENT ON COLUMN billing_subscriptions.status IS 'Subscription status (active, trialing, past_due, canceled, unpaid)';
COMMENT ON COLUMN billing_subscriptions.cancel_at_period_end IS 'Whether subscription will cancel at end of billing period';

-- ============================================================================
-- Billing Quotas
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_quotas (
    quota_id SERIAL PRIMARY KEY,
    plan_name VARCHAR(100) NOT NULL REFERENCES billing_plans(plan_name) ON DELETE CASCADE,
    service_name VARCHAR(100) NOT NULL,
    -- Service: api, storage, bandwidth, compute, database, functions
    limit_value BIGINT NOT NULL DEFAULT -1,
    -- -1 = unlimited
    limit_type VARCHAR(50) DEFAULT 'requests',
    -- Type: requests, gb, gb-hours, cpu-hours, invocations
    enforcement_mode VARCHAR(20) DEFAULT 'soft',
    -- Mode: soft (warn), hard (block)
    overage_price DECIMAL(10, 6) DEFAULT 0.00,
    -- Price per unit over quota
    reset_period VARCHAR(20) DEFAULT 'monthly',
    -- Period: daily, weekly, monthly
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(plan_name, service_name)
);

CREATE INDEX idx_billing_quotas_plan ON billing_quotas(plan_name);
CREATE INDEX idx_billing_quotas_service ON billing_quotas(service_name);

COMMENT ON TABLE billing_quotas IS 'Quota limits for each plan and service';
COMMENT ON COLUMN billing_quotas.limit_value IS 'Quota limit (-1 for unlimited)';
COMMENT ON COLUMN billing_quotas.enforcement_mode IS 'soft = warn only, hard = block requests';
COMMENT ON COLUMN billing_quotas.overage_price IS 'Price charged per unit over quota';

-- Insert default quotas
INSERT INTO billing_quotas (plan_name, service_name, limit_value, limit_type, enforcement_mode, overage_price)
VALUES
    -- Free plan
    ('free', 'api', 10000, 'requests', 'hard', 0.0001),
    ('free', 'storage', 1, 'gb', 'hard', 0.10),
    ('free', 'bandwidth', 10, 'gb', 'hard', 0.05),
    ('free', 'compute', 10, 'cpu-hours', 'hard', 0.05),
    ('free', 'database', 1000, 'connections', 'hard', 0.01),
    ('free', 'functions', 1000, 'invocations', 'hard', 0.0002),

    -- Starter plan
    ('starter', 'api', 100000, 'requests', 'soft', 0.0001),
    ('starter', 'storage', 10, 'gb', 'soft', 0.10),
    ('starter', 'bandwidth', 100, 'gb', 'soft', 0.05),
    ('starter', 'compute', 100, 'cpu-hours', 'soft', 0.05),
    ('starter', 'database', 10000, 'connections', 'soft', 0.01),
    ('starter', 'functions', 10000, 'invocations', 'soft', 0.0002),

    -- Pro plan
    ('pro', 'api', 1000000, 'requests', 'soft', 0.00005),
    ('pro', 'storage', 100, 'gb', 'soft', 0.08),
    ('pro', 'bandwidth', 1000, 'gb', 'soft', 0.04),
    ('pro', 'compute', 1000, 'cpu-hours', 'soft', 0.04),
    ('pro', 'database', 100000, 'connections', 'soft', 0.005),
    ('pro', 'functions', 100000, 'invocations', 'soft', 0.0001),

    -- Enterprise plan (unlimited)
    ('enterprise', 'api', -1, 'requests', 'soft', 0.00),
    ('enterprise', 'storage', -1, 'gb', 'soft', 0.00),
    ('enterprise', 'bandwidth', -1, 'gb', 'soft', 0.00),
    ('enterprise', 'compute', -1, 'cpu-hours', 'soft', 0.00),
    ('enterprise', 'database', -1, 'connections', 'soft', 0.00),
    ('enterprise', 'functions', -1, 'invocations', 'soft', 0.00)
ON CONFLICT (plan_name, service_name) DO NOTHING;

-- ============================================================================
-- Billing Usage Records
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_usage_records (
    usage_id BIGSERIAL PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL REFERENCES billing_customers(customer_id) ON DELETE CASCADE,
    service_name VARCHAR(100) NOT NULL,
    quantity DECIMAL(20, 6) NOT NULL DEFAULT 1.0,
    unit_cost DECIMAL(10, 6) DEFAULT 0.00,
    metadata JSONB DEFAULT '{}',
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    aggregated BOOLEAN DEFAULT false,
    invoice_id VARCHAR(255)
);

CREATE INDEX idx_billing_usage_customer ON billing_usage_records(customer_id);
CREATE INDEX idx_billing_usage_service ON billing_usage_records(service_name);
CREATE INDEX idx_billing_usage_recorded ON billing_usage_records(recorded_at);
CREATE INDEX idx_billing_usage_customer_service ON billing_usage_records(customer_id, service_name);
CREATE INDEX idx_billing_usage_customer_date ON billing_usage_records(customer_id, recorded_at);
CREATE INDEX idx_billing_usage_aggregated ON billing_usage_records(aggregated) WHERE aggregated = false;

COMMENT ON TABLE billing_usage_records IS 'Raw usage tracking records';
COMMENT ON COLUMN billing_usage_records.quantity IS 'Amount of service consumed';
COMMENT ON COLUMN billing_usage_records.unit_cost IS 'Cost per unit at time of usage';
COMMENT ON COLUMN billing_usage_records.aggregated IS 'Whether usage has been aggregated into invoice';
COMMENT ON COLUMN billing_usage_records.metadata IS 'Additional usage context (endpoint, method, etc)';

-- ============================================================================
-- Billing Invoices
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_invoices (
    invoice_id VARCHAR(255) PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL REFERENCES billing_customers(customer_id) ON DELETE CASCADE,
    stripe_invoice_id VARCHAR(255) UNIQUE,
    subscription_id VARCHAR(255) REFERENCES billing_subscriptions(subscription_id),
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    -- Status: draft, open, paid, void, uncollectible
    total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    subtotal DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    tax_amount DECIMAL(10, 2) DEFAULT 0.00,
    discount_amount DECIMAL(10, 2) DEFAULT 0.00,
    currency VARCHAR(3) DEFAULT 'USD',
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    due_date TIMESTAMP WITH TIME ZONE,
    paid_at TIMESTAMP WITH TIME ZONE,
    payment_method VARCHAR(100),
    line_items JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_billing_invoices_customer ON billing_invoices(customer_id);
CREATE INDEX idx_billing_invoices_subscription ON billing_invoices(subscription_id);
CREATE INDEX idx_billing_invoices_status ON billing_invoices(status);
CREATE INDEX idx_billing_invoices_stripe ON billing_invoices(stripe_invoice_id);
CREATE INDEX idx_billing_invoices_period ON billing_invoices(period_start, period_end);
CREATE INDEX idx_billing_invoices_created ON billing_invoices(created_at);

COMMENT ON TABLE billing_invoices IS 'Customer invoices and bills';
COMMENT ON COLUMN billing_invoices.line_items IS 'Invoice line items (JSON array)';
COMMENT ON COLUMN billing_invoices.status IS 'Invoice status (draft, open, paid, void, uncollectible)';

-- ============================================================================
-- Billing Payment Methods
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_payment_methods (
    payment_method_id VARCHAR(255) PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL REFERENCES billing_customers(customer_id) ON DELETE CASCADE,
    stripe_payment_method_id VARCHAR(255) UNIQUE,
    type VARCHAR(50) NOT NULL,
    -- Type: card, bank_account, paypal
    is_default BOOLEAN DEFAULT false,
    card_brand VARCHAR(50),
    card_last4 VARCHAR(4),
    card_exp_month INTEGER,
    card_exp_year INTEGER,
    billing_details JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_billing_payment_methods_customer ON billing_payment_methods(customer_id);
CREATE INDEX idx_billing_payment_methods_stripe ON billing_payment_methods(stripe_payment_method_id);
CREATE INDEX idx_billing_payment_methods_default ON billing_payment_methods(customer_id, is_default);

COMMENT ON TABLE billing_payment_methods IS 'Customer payment methods';
COMMENT ON COLUMN billing_payment_methods.is_default IS 'Whether this is the default payment method';

-- ============================================================================
-- Billing Events (Webhook Log)
-- ============================================================================

CREATE TABLE IF NOT EXISTS billing_events (
    event_id BIGSERIAL PRIMARY KEY,
    stripe_event_id VARCHAR(255) UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    -- Type: invoice.paid, customer.subscription.created, etc.
    customer_id VARCHAR(255) REFERENCES billing_customers(customer_id) ON DELETE CASCADE,
    payload JSONB NOT NULL,
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_billing_events_type ON billing_events(event_type);
CREATE INDEX idx_billing_events_customer ON billing_events(customer_id);
CREATE INDEX idx_billing_events_stripe ON billing_events(stripe_event_id);
CREATE INDEX idx_billing_events_processed ON billing_events(processed);
CREATE INDEX idx_billing_events_created ON billing_events(created_at);

COMMENT ON TABLE billing_events IS 'Stripe webhook events log';
COMMENT ON COLUMN billing_events.processed IS 'Whether event has been processed';

-- ============================================================================
-- Billing Usage Aggregations (Materialized View)
-- ============================================================================

CREATE MATERIALIZED VIEW IF NOT EXISTS billing_usage_daily_summary AS
SELECT
    customer_id,
    service_name,
    DATE(recorded_at) as usage_date,
    COUNT(*) as event_count,
    SUM(quantity) as total_quantity,
    SUM(quantity * unit_cost) as total_cost,
    MIN(recorded_at) as first_event,
    MAX(recorded_at) as last_event
FROM billing_usage_records
GROUP BY customer_id, service_name, DATE(recorded_at);

CREATE UNIQUE INDEX idx_billing_usage_daily_summary_unique
    ON billing_usage_daily_summary(customer_id, service_name, usage_date);
CREATE INDEX idx_billing_usage_daily_summary_date
    ON billing_usage_daily_summary(usage_date);

COMMENT ON MATERIALIZED VIEW billing_usage_daily_summary IS 'Daily aggregated usage summary for faster reporting';

-- Function to refresh materialized view
CREATE OR REPLACE FUNCTION refresh_billing_usage_summary()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY billing_usage_daily_summary;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Triggers
-- ============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_billing_customers_updated_at
    BEFORE UPDATE ON billing_customers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_billing_plans_updated_at
    BEFORE UPDATE ON billing_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_billing_subscriptions_updated_at
    BEFORE UPDATE ON billing_subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_billing_quotas_updated_at
    BEFORE UPDATE ON billing_quotas
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_billing_invoices_updated_at
    BEFORE UPDATE ON billing_invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_billing_payment_methods_updated_at
    BEFORE UPDATE ON billing_payment_methods
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Helper Functions
-- ============================================================================

-- Function: Get current quota usage for customer/service
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
$$ LANGUAGE plpgsql;

-- Function: Check if quota exceeded
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
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Grants (adjust as needed for your security model)
-- ============================================================================

-- Grant permissions to application user (adjust username as needed)
-- GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO your_app_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO your_app_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO your_app_user;

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON SCHEMA public IS 'nself billing system - comprehensive billing, usage tracking, and quota management';

-- ============================================================================
-- Migration Complete
-- ============================================================================

-- Log migration completion
DO $$
BEGIN
    RAISE NOTICE 'Migration 015_create_billing_system.sql completed successfully';
    RAISE NOTICE 'Billing system initialized with:';
    RAISE NOTICE '  - Customer management';
    RAISE NOTICE '  - Subscription tracking';
    RAISE NOTICE '  - Usage metering';
    RAISE NOTICE '  - Quota enforcement';
    RAISE NOTICE '  - Invoice generation';
    RAISE NOTICE '  - Payment methods';
    RAISE NOTICE '  - Webhook event logging';
END $$;
