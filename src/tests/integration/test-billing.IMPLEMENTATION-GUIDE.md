# Billing System Implementation Guide

This guide helps developers implement the billing system modules to pass all 60 integration tests.

## Quick Start

1. Create the billing module directory:
```bash
mkdir -p src/lib/billing
```

2. Implement the 5 required modules (see skeleton templates below)

3. Run tests to verify:
```bash
bash src/tests/integration/test-billing.sh
```

## Module 1: Usage Tracking (`usage-tracking.sh`)

**Purpose**: Track customer resource usage for metering-based billing

**Required Functions**:

```bash
#!/usr/bin/env bash
# usage-tracking.sh - Usage metering and tracking

usage_tracking_init() {
  # Initialize usage tracking tables and indexes
  # Return: 0 on success, 1 on failure
}

usage_track_api_request() {
  local customer_id="$1"
  local endpoint="$2"
  local method="$3"
  local status="$4"
  local duration="$5"

  # Record API request in usage_events table
  # Return: 0 on success
}

usage_track_storage() {
  local customer_id="$1"
  local bytes="$2"

  # Record storage usage
  # Return: 0 on success
}

usage_track_bandwidth() {
  local customer_id="$1"
  local bytes="$2"
  local direction="$3"  # "egress" or "ingress"

  # Record bandwidth usage
  # Return: 0 on success
}

usage_track_compute() {
  local customer_id="$1"
  local resource_type="$2"
  local seconds="$3"

  # Record compute time
  # Return: 0 on success
}

usage_get_current() {
  local customer_id="$1"

  # Return current usage as JSON
  # Output: {"api_requests":100,"storage_gb":5.2,...}
}

usage_aggregate() {
  local customer_id="$1"
  local period="$2"  # "hour", "day", or "month"

  # Aggregate usage by period
  # Return: 0 on success
}

usage_export() {
  local customer_id="$1"
  local format="$2"  # "csv" or "json"
  local output_path="$3"

  # Export usage data to file
  # Return: 0 on success
}

usage_alert_set() {
  local customer_id="$1"
  local metric="$2"
  local threshold="$3"
  local notification_type="$4"

  # Set usage alert threshold
  # Return: 0 on success
}

usage_alert_check() {
  local customer_id="$1"

  # Check if any alerts should trigger
  # Return: 0 on success
}

usage_get_history() {
  local customer_id="$1"
  local days="$2"

  # Return usage history as JSON array
  # Output: [{"date":"2026-01-01","api_requests":1000}]
}

usage_reset() {
  local customer_id="$1"
  local period="$2"  # "monthly" or "daily"

  # Reset usage counters
  # Return: 0 on success
}
```

## Module 2: Stripe Integration (`stripe-integration.sh`)

**Purpose**: Integrate with Stripe for payments and subscriptions

**Required Functions**:

```bash
#!/usr/bin/env bash
# stripe-integration.sh - Stripe API integration

STRIPE_API_KEY=""
STRIPE_API_URL="https://api.stripe.com/v1"

stripe_init() {
  local api_key="$1"
  STRIPE_API_KEY="$api_key"
  # Verify API key works
  # Return: 0 on success
}

stripe_customer_create() {
  local email="$1"
  local name="$2"

  # Create Stripe customer via API
  # Output: customer_id (e.g., "cus_123abc")
  # Return: 0 on success
}

stripe_customer_get() {
  local customer_id="$1"

  # Get customer details from Stripe
  # Output: JSON customer object
}

stripe_customer_update() {
  local customer_id="$1"
  local field="$2"
  local value="$3"

  # Update customer field
  # Return: 0 on success
}

stripe_payment_method_attach() {
  local customer_id="$1"
  local payment_method_id="$2"

  # Attach payment method to customer
  # Output: payment_method_id
}

stripe_payment_methods_list() {
  local customer_id="$1"

  # List all payment methods for customer
  # Output: JSON array of payment methods
}

stripe_subscription_create() {
  local customer_id="$1"
  local price_id="$2"

  # Create subscription
  # Output: subscription_id (e.g., "sub_123abc")
}

stripe_subscription_get() {
  local subscription_id="$1"

  # Get subscription details
  # Output: JSON subscription object
}

stripe_subscription_update() {
  local subscription_id="$1"
  local new_price_id="$2"

  # Update subscription to new price
  # Return: 0 on success
}

stripe_subscription_cancel() {
  local subscription_id="$1"
  local when="$2"  # "now" or "at_period_end"

  # Cancel subscription
  # Return: 0 on success
}

stripe_invoice_create() {
  local customer_id="$1"

  # Create draft invoice
  # Output: invoice_id (e.g., "in_123abc")
}

stripe_invoice_item_add() {
  local customer_id="$1"
  local amount="$2"  # in cents
  local description="$3"

  # Add line item to invoice
  # Return: 0 on success
}

stripe_invoice_finalize() {
  local invoice_id="$1"

  # Finalize invoice (can't be edited after)
  # Return: 0 on success
}

stripe_payment_process() {
  local invoice_id="$1"

  # Charge customer for invoice
  # Output: JSON payment result
}

stripe_webhook_handle() {
  local webhook_payload="$1"

  # Process Stripe webhook event
  # Return: 0 on success
}

stripe_customer_delete() {
  local customer_id="$1"

  # Delete customer (for cleanup)
  # Return: 0 on success
}
```

## Module 3: Quota Enforcement (`quota-enforcement.sh`)

**Purpose**: Enforce resource limits and prevent abuse

**Required Functions**:

```bash
#!/usr/bin/env bash
# quota-enforcement.sh - Quota limits and enforcement

quota_init() {
  # Initialize quota system
  # Return: 0 on success
}

quota_set_limit() {
  local customer_id="$1"
  local resource="$2"
  local limit="$3"
  local period="$4"  # "per_month", "per_day", "absolute"

  # Set resource limit for customer
  # Return: 0 on success
}

quota_check() {
  local customer_id="$1"
  local resource="$2"
  local amount="$3"

  # Check if operation would exceed quota
  # Return: 0 if within quota, 1 if would exceed
}

quota_enforce() {
  local customer_id="$1"
  local resource="$2"
  local amount="$3"

  # Check and record usage (fail if over quota)
  # Return: 0 if allowed, 1 if blocked
}

quota_soft_limit_check() {
  local customer_id="$1"
  local resource="$2"

  # Check if at 80% of quota (warning threshold)
  # Output: JSON with warning flag
}

quota_hard_limit_check() {
  local customer_id="$1"
  local resource="$2"
  local amount="$3"

  # Check if at 100% of quota (block threshold)
  # Return: 0 if under limit, 1 if over
}

quota_calculate_overage() {
  local customer_id="$1"
  local resource="$2"

  # Calculate overage amount and cost
  # Output: JSON with overage and cost
}

quota_get_usage_percentage() {
  local customer_id="$1"
  local resource="$2"

  # Get usage as percentage of quota
  # Output: percentage number (e.g., "75")
}

quota_reset() {
  local customer_id="$1"
  local period="$2"  # "monthly" or "daily"

  # Reset quota counters
  # Return: 0 on success
}
```

## Module 4: Invoice Generation (`invoice-generation.sh`)

**Purpose**: Generate invoices with tax and discounts

**Required Functions**:

```bash
#!/usr/bin/env bash
# invoice-generation.sh - Invoice creation and delivery

invoice_init() {
  # Initialize invoice system
  # Return: 0 on success
}

invoice_generate() {
  local customer_id="$1"
  local period="$2"  # "monthly", "quarterly", etc.

  # Generate invoice for period
  # Output: invoice_id
}

invoice_calculate_charges() {
  local customer_id="$1"

  # Calculate all charges for customer
  # Output: JSON with total and breakdown
}

invoice_apply_discount() {
  local invoice_id="$1"
  local discount_code="$2"

  # Apply discount code to invoice
  # Return: 0 on success
}

invoice_apply_discount_percentage() {
  local invoice_id="$1"
  local percentage="$2"

  # Apply percentage discount
  # Return: 0 on success
}

invoice_calculate_tax() {
  local customer_id="$1"
  local amount="$2"

  # Calculate sales tax
  # Output: JSON with tax_rate and tax_amount
}

invoice_generate_pdf() {
  local invoice_id="$1"
  local output_path="$2"

  # Generate PDF invoice
  # Return: 0 on success
}

invoice_email_send() {
  local invoice_id="$1"
  local customer_id="$2"

  # Email invoice to customer
  # Return: 0 on success
}

invoice_get_history() {
  local customer_id="$1"
  local months="$2"

  # Get past invoices
  # Output: JSON array of invoices
}

invoice_mark_paid() {
  local invoice_id="$1"
  local payment_method="$2"

  # Mark invoice as paid
  # Return: 0 on success
}
```

## Module 5: Plan Management (`plan-management.sh`)

**Purpose**: Manage subscription plans and upgrades

**Required Functions**:

```bash
#!/usr/bin/env bash
# plan-management.sh - Subscription plan management

plan_init() {
  # Initialize plan system
  # Return: 0 on success
}

plan_list() {
  # List all available plans
  # Output: JSON array of plans
}

plan_get_details() {
  local plan_id="$1"

  # Get plan features and limits
  # Output: JSON plan object
}

plan_upgrade() {
  local customer_id="$1"
  local new_plan_id="$2"

  # Upgrade customer to new plan
  # Return: 0 on success
}

plan_downgrade() {
  local customer_id="$1"
  local new_plan_id="$2"
  local when="$3"  # "now" or "at_period_end"

  # Downgrade customer plan
  # Return: 0 on success
}

plan_create_custom() {
  local customer_id="$1"
  local plan_name="$2"
  local price="$3"
  local limits_json="$4"

  # Create custom enterprise plan
  # Output: custom_plan_id
}

plan_compare() {
  local plan_id_1="$1"
  local plan_id_2="$2"

  # Compare two plans
  # Output: JSON comparison table
}

plan_get_current() {
  local customer_id="$1"

  # Get customer's current plan
  # Output: JSON plan object
}
```

## Database Schema

Create tables for billing data:

```sql
-- src/lib/billing/schema.sql

-- Usage events
CREATE TABLE IF NOT EXISTS usage_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  customer_id TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  amount NUMERIC NOT NULL,
  metadata JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_usage_customer ON usage_events(customer_id);
CREATE INDEX idx_usage_created ON usage_events(created_at);

-- Customer quotas
CREATE TABLE IF NOT EXISTS customer_quotas (
  customer_id TEXT PRIMARY KEY,
  api_requests_limit INTEGER DEFAULT 1000,
  storage_limit BIGINT DEFAULT 1073741824,
  bandwidth_limit BIGINT DEFAULT 10737418240,
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Invoices
CREATE TABLE IF NOT EXISTS invoices (
  id TEXT PRIMARY KEY,
  customer_id TEXT NOT NULL,
  amount NUMERIC NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  period_start TIMESTAMP,
  period_end TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Plans
CREATE TABLE IF NOT EXISTS plans (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  price NUMERIC NOT NULL,
  limits JSONB NOT NULL DEFAULT '{}',
  features JSONB DEFAULT '{}',
  created_at TIMESTAMP DEFAULT NOW()
);

-- Subscriptions
CREATE TABLE IF NOT EXISTS subscriptions (
  id TEXT PRIMARY KEY,
  customer_id TEXT NOT NULL,
  plan_id TEXT REFERENCES plans(id),
  stripe_subscription_id TEXT UNIQUE,
  status TEXT NOT NULL DEFAULT 'active',
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Seed default plans
INSERT INTO plans (id, name, price, limits, features) VALUES
  ('free', 'Free', 0, '{"api_requests":1000,"storage":1073741824}', '{"support":"community"}'),
  ('pro', 'Pro', 29, '{"api_requests":10000,"storage":10737418240}', '{"support":"email"}'),
  ('enterprise', 'Enterprise', 299, '{"api_requests":100000,"storage":107374182400}', '{"support":"priority"}')
ON CONFLICT (id) DO NOTHING;
```

## Testing Your Implementation

### Step 1: Run in simulation mode (should pass)
```bash
bash src/tests/integration/test-billing.sh
# Output: 60/60 passed (simulated)
```

### Step 2: Implement one module at a time
```bash
# Implement usage-tracking.sh first
bash src/tests/integration/test-billing.sh
# Output: Tests 1-15 should start running in live mode
```

### Step 3: Verify each section passes
```bash
# After implementing all modules
bash src/tests/integration/test-billing.sh
# Output: 60/60 passed (live mode)
```

### Step 4: Test with real Stripe credentials (optional)
```bash
export STRIPE_TEST_KEY="sk_test_PLACEHOLDER_key_here"
bash src/tests/integration/test-billing.sh
```

## Common Implementation Patterns

### Database Queries

Use nself's database utilities:

```bash
source "$SCRIPT_DIR/../database/query.sh"

# Execute query
result=$(db_query "SELECT * FROM usage_events WHERE customer_id='$1'")
```

### API Calls

Use curl with proper error handling:

```bash
stripe_api_call() {
  local endpoint="$1"
  local method="${2:-GET}"
  local data="$3"

  curl -s -X "$method" \
    "https://api.stripe.com/v1/$endpoint" \
    -H "Authorization: Bearer $STRIPE_API_KEY" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    ${data:+-d "$data"}
}
```

### Error Handling

Always use proper error handling:

```bash
set -euo pipefail

my_function() {
  local customer_id="$1"

  # Validate input
  if [[ -z "$customer_id" ]]; then
    echo "Error: customer_id required" >&2
    return 1
  fi

  # Execute with error handling
  if ! result=$(some_command); then
    echo "Error: command failed" >&2
    return 1
  fi

  echo "$result"
  return 0
}
```

## Next Steps

1. ✅ Create `src/lib/billing/` directory
2. ✅ Implement `usage-tracking.sh` (run tests 1-15)
3. ✅ Implement `stripe-integration.sh` (run tests 16-30)
4. ✅ Implement `quota-enforcement.sh` (run tests 31-42)
5. ✅ Implement `invoice-generation.sh` (run tests 43-52)
6. ✅ Implement `plan-management.sh` (run tests 53-60)
7. ✅ Run full test suite: `bash src/tests/integration/test-billing.sh`
8. ✅ Verify 60/60 tests pass
9. ✅ Add CI/CD integration
10. ✅ Document in main docs

## Support

- Test documentation: `test-billing.README.md`
- nself docs: `/docs/features/BILLING.md`
- Stripe docs: https://stripe.com/docs/api

## License

Part of the nself project. See LICENSE file for details.
