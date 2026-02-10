# Billing System Integration Tests

**File**: `/Users/admin/Sites/nself/src/tests/integration/test-billing.sh`
**Version**: nself v0.8.0+
**Status**: Ready for implementation (Simulation Mode)
**Total Tests**: 60 comprehensive tests across 5 categories

## Overview

This test suite provides comprehensive integration testing for the nself billing system. The tests are designed to validate usage tracking, Stripe payment integration, quota enforcement, invoice generation, and subscription plan management.

### Current Status

The test suite runs in **SIMULATION MODE** by default since the billing modules are not yet implemented. Once the billing system is implemented at `/Users/admin/Sites/nself/src/lib/billing/`, the tests will automatically switch to **LIVE MODE** and test real functionality.

## Test Categories

### 1. Usage Tracking Tests (15 tests)

Tests the metering and usage tracking system that records customer resource consumption.

| Test # | Description | Purpose |
|--------|-------------|---------|
| 1 | Initialize usage tracking | Verify system initialization |
| 2 | Track API request | Record API call with metadata |
| 3 | Track storage usage | Monitor storage consumption |
| 4 | Track bandwidth consumption | Track egress/ingress data |
| 5 | Track compute time | Record function execution time |
| 6 | Get current usage | Retrieve real-time usage stats |
| 7 | Aggregate usage by hour | Hourly usage rollups |
| 8 | Aggregate usage by day | Daily usage rollups |
| 9 | Aggregate usage by month | Monthly usage rollups |
| 10 | Export usage data (CSV) | Export to CSV format |
| 11 | Export usage data (JSON) | Export to JSON format |
| 12 | Set usage alert threshold | Configure usage alerts |
| 13 | Check alert threshold | Verify alert triggering |
| 14 | Get usage history | Retrieve historical usage |
| 15 | Reset usage counters | Reset monthly/daily counters |

**Expected Functions**:
- `usage_tracking_init()`
- `usage_track_api_request(customer_id, endpoint, method, status, duration)`
- `usage_track_storage(customer_id, bytes)`
- `usage_track_bandwidth(customer_id, bytes, direction)`
- `usage_track_compute(customer_id, resource_type, seconds)`
- `usage_get_current(customer_id)`
- `usage_aggregate(customer_id, period)`
- `usage_export(customer_id, format, output_path)`
- `usage_alert_set(customer_id, metric, threshold, notification_type)`
- `usage_alert_check(customer_id)`
- `usage_get_history(customer_id, days)`
- `usage_reset(customer_id, period)`

### 2. Stripe Integration Tests (15 tests)

Tests integration with Stripe payment platform for customer management, subscriptions, and payments.

| Test # | Description | Purpose |
|--------|-------------|---------|
| 16 | Initialize Stripe client | Setup Stripe SDK |
| 17 | Create Stripe customer | Register new customer |
| 18 | Get Stripe customer | Retrieve customer details |
| 19 | Update Stripe customer | Modify customer info |
| 20 | Add payment method | Attach card/payment method |
| 21 | List payment methods | Show all payment methods |
| 22 | Create subscription | Start new subscription |
| 23 | Get subscription | Retrieve subscription details |
| 24 | Update subscription | Change subscription plan |
| 25 | Cancel subscription | End subscription |
| 26 | Create invoice | Generate new invoice |
| 27 | Add invoice item | Add line items to invoice |
| 28 | Finalize invoice | Lock invoice for payment |
| 29 | Process payment | Charge payment method |
| 30 | Handle webhook event | Process Stripe webhooks |

**Expected Functions**:
- `stripe_init(api_key)`
- `stripe_customer_create(email, name)`
- `stripe_customer_get(customer_id)`
- `stripe_customer_update(customer_id, field, value)`
- `stripe_payment_method_attach(customer_id, payment_method_id)`
- `stripe_payment_methods_list(customer_id)`
- `stripe_subscription_create(customer_id, price_id)`
- `stripe_subscription_get(subscription_id)`
- `stripe_subscription_update(subscription_id, new_price_id)`
- `stripe_subscription_cancel(subscription_id, when)`
- `stripe_invoice_create(customer_id)`
- `stripe_invoice_item_add(customer_id, amount, description)`
- `stripe_invoice_finalize(invoice_id)`
- `stripe_payment_process(invoice_id)`
- `stripe_webhook_handle(webhook_payload)`

### 3. Quota Enforcement Tests (12 tests)

Tests quota limits and enforcement to prevent resource abuse and manage plan limits.

| Test # | Description | Purpose |
|--------|-------------|---------|
| 31 | Initialize quota system | Setup quota tracking |
| 32 | Set API rate limit | Configure API limits |
| 33 | Set storage limit | Configure storage limits |
| 34 | Set bandwidth limit | Configure bandwidth limits |
| 35 | Check quota before operation | Pre-flight quota check |
| 36 | Enforce API rate limit (within limit) | Allow when under limit |
| 37 | Enforce storage limit | Block when over limit |
| 38 | Handle soft limit (warning) | Warn at 80% usage |
| 39 | Handle hard limit (block) | Block at 100% usage |
| 40 | Calculate overage | Compute overage charges |
| 41 | Get quota usage percentage | Show usage percentage |
| 42 | Reset quota (monthly schedule) | Reset monthly quotas |

**Expected Functions**:
- `quota_init()`
- `quota_set_limit(customer_id, resource, limit, period)`
- `quota_check(customer_id, resource, amount)`
- `quota_enforce(customer_id, resource, amount)`
- `quota_soft_limit_check(customer_id, resource)`
- `quota_hard_limit_check(customer_id, resource, amount)`
- `quota_calculate_overage(customer_id, resource)`
- `quota_get_usage_percentage(customer_id, resource)`
- `quota_reset(customer_id, period)`

### 4. Invoice Generation Tests (10 tests)

Tests automated invoice generation, tax calculation, and delivery.

| Test # | Description | Purpose |
|--------|-------------|---------|
| 43 | Initialize invoice system | Setup invoice engine |
| 44 | Generate monthly invoice | Create monthly bill |
| 45 | Calculate usage charges | Compute usage-based fees |
| 46 | Apply discount code | Apply promo codes |
| 47 | Apply percentage discount | Apply percentage off |
| 48 | Calculate tax | Compute sales tax |
| 49 | Generate invoice PDF | Create PDF document |
| 50 | Email invoice to customer | Send invoice via email |
| 51 | Get invoice history | Retrieve past invoices |
| 52 | Mark invoice as paid | Update payment status |

**Expected Functions**:
- `invoice_init()`
- `invoice_generate(customer_id, period)`
- `invoice_calculate_charges(customer_id)`
- `invoice_apply_discount(invoice_id, discount_code)`
- `invoice_apply_discount_percentage(invoice_id, percentage)`
- `invoice_calculate_tax(customer_id, amount)`
- `invoice_generate_pdf(invoice_id, output_path)`
- `invoice_email_send(invoice_id, customer_id)`
- `invoice_get_history(customer_id, months)`
- `invoice_mark_paid(invoice_id, payment_method)`

### 5. Plan Management Tests (8 tests)

Tests subscription plan management, upgrades, downgrades, and custom enterprise plans.

| Test # | Description | Purpose |
|--------|-------------|---------|
| 53 | Initialize plan system | Setup plan management |
| 54 | List available plans | Show all plans |
| 55 | Get plan details | Show plan features/limits |
| 56 | Upgrade to Pro plan | Upgrade customer plan |
| 57 | Downgrade to Free plan | Downgrade customer plan |
| 58 | Create custom enterprise plan | Create custom plan |
| 59 | Compare plans | Show plan comparison |
| 60 | Get current plan for customer | Show active plan |

**Expected Functions**:
- `plan_init()`
- `plan_list()`
- `plan_get_details(plan_id)`
- `plan_upgrade(customer_id, new_plan_id)`
- `plan_downgrade(customer_id, new_plan_id, when)`
- `plan_create_custom(customer_id, plan_name, price, limits_json)`
- `plan_compare(plan_id_1, plan_id_2)`
- `plan_get_current(customer_id)`

## Running the Tests

### Simulation Mode (Current)

Tests run automatically in simulation mode when billing modules don't exist:

```bash
bash src/tests/integration/test-billing.sh
```

**Output**: All 60 tests pass with "(simulated)" status.

### Live Mode (After Implementation)

Once you implement the billing modules at:
- `/Users/admin/Sites/nself/src/lib/billing/usage-tracking.sh`
- `/Users/admin/Sites/nself/src/lib/billing/stripe-integration.sh`
- `/Users/admin/Sites/nself/src/lib/billing/quota-enforcement.sh`
- `/Users/admin/Sites/nself/src/lib/billing/invoice-generation.sh`
- `/Users/admin/Sites/nself/src/lib/billing/plan-management.sh`

The tests will automatically switch to live mode and test real functionality.

### CI/CD Integration

Add to `.github/workflows/test-billing.yml`:

```yaml
name: Billing Tests

on:
  push:
    paths:
      - 'src/lib/billing/**'
      - 'src/tests/integration/test-billing.sh'
  pull_request:
    paths:
      - 'src/lib/billing/**'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run billing tests
        run: bash src/tests/integration/test-billing.sh
        env:
          STRIPE_TEST_KEY: ${{ secrets.STRIPE_TEST_KEY }}
```

## Test Environment Variables

The following environment variables can be used to configure test behavior:

```bash
# Stripe test credentials
export STRIPE_TEST_KEY="sk_test_PLACEHOLDER..."
export STRIPE_TEST_SECRET="whsec_..."

# Test database (if separate from main DB)
export BILLING_TEST_DB="billing_test"

# Enable verbose output
export BILLING_TEST_VERBOSE=true

# Skip cleanup (for debugging)
export BILLING_TEST_SKIP_CLEANUP=true
```

## Expected Module Structure

The billing implementation should follow this structure:

```
src/lib/billing/
├── usage-tracking.sh       # Metering and usage tracking
├── stripe-integration.sh   # Stripe API integration
├── quota-enforcement.sh    # Quota limits and enforcement
├── invoice-generation.sh   # Invoice creation and delivery
├── plan-management.sh      # Subscription plan management
└── README.md               # Billing module documentation
```

## Database Schema

The billing system will require the following database tables:

```sql
-- Usage tracking
CREATE TABLE usage_events (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  customer_id TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  amount NUMERIC NOT NULL,
  metadata JSONB,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Quotas
CREATE TABLE quotas (
  customer_id TEXT PRIMARY KEY,
  api_requests_limit INTEGER,
  storage_limit BIGINT,
  bandwidth_limit BIGINT,
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Invoices
CREATE TABLE invoices (
  id TEXT PRIMARY KEY,
  customer_id TEXT NOT NULL,
  amount NUMERIC NOT NULL,
  status TEXT NOT NULL,
  period_start TIMESTAMP,
  period_end TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Plans
CREATE TABLE plans (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  price NUMERIC NOT NULL,
  limits JSONB NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Customer subscriptions
CREATE TABLE subscriptions (
  id TEXT PRIMARY KEY,
  customer_id TEXT NOT NULL,
  plan_id TEXT REFERENCES plans(id),
  stripe_subscription_id TEXT,
  status TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

## Success Criteria

All 60 tests must pass before billing system is considered production-ready:

- ✅ All tests pass in live mode
- ✅ No memory leaks or resource exhaustion
- ✅ Proper error handling for all edge cases
- ✅ Cross-platform compatibility (macOS, Linux, WSL)
- ✅ CI/CD integration working
- ✅ Documentation complete

## Cross-Platform Compatibility

Tests follow nself portability standards:

- ✅ Uses `printf` instead of `echo -e`
- ✅ No Bash 4+ features (compatible with Bash 3.2)
- ✅ Platform-agnostic date handling
- ✅ Proper error handling with `set -euo pipefail`

## Troubleshooting

### All tests fail immediately

**Problem**: Billing modules not found
**Solution**: Implement billing modules or verify running in simulation mode

### Stripe tests fail

**Problem**: Missing Stripe API key
**Solution**: Set `STRIPE_TEST_KEY` environment variable

### Database connection errors

**Problem**: PostgreSQL not running or misconfigured
**Solution**: Verify nself database is running: `nself status`

### Permission errors

**Problem**: File permissions issue
**Solution**: Ensure test file is executable: `chmod +x test-billing.sh`

## Contributing

When adding new billing features:

1. Add corresponding test(s) to appropriate section
2. Update `TOTAL_TESTS` count at top of file
3. Update this README with new test descriptions
4. Ensure simulation mode stubs exist
5. Test in both simulation and live modes
6. Update expected functions list

## Related Documentation

- [nself Billing System Documentation](/.wiki/commands/BILLING.md)
- [Stripe Plugin Documentation](/.wiki/plugins/stripe.md)
- [Usage Tracking Guide](/.wiki/guides/USAGE-TRACKING.md)
- [Quota Management Guide](/.wiki/guides/QUOTAS.md)

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-29 | Initial test suite with 60 tests |

## License

Part of the nself project. See LICENSE file for details.
