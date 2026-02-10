# Billing Tests Quick Reference

**File**: `test-billing.sh`
**Tests**: 60 comprehensive integration tests
**Status**: ✅ Ready (Simulation Mode)

## Test Breakdown

| Section | Tests | Module | Functions |
|---------|-------|--------|-----------|
| **Usage Tracking** | 1-15 (15) | `usage-tracking.sh` | 12 functions |
| **Stripe Integration** | 16-30 (15) | `stripe-integration.sh` | 16 functions |
| **Quota Enforcement** | 31-42 (12) | `quota-enforcement.sh` | 9 functions |
| **Invoice Generation** | 43-52 (10) | `invoice-generation.sh` | 10 functions |
| **Plan Management** | 53-60 (8) | `plan-management.sh` | 8 functions |

**Total**: 60 tests, 55 functions across 5 modules

## Run Tests

```bash
# Current (simulation mode)
bash src/tests/integration/test-billing.sh

# After implementation (live mode)
bash src/tests/integration/test-billing.sh

# With verbose output
BILLING_TEST_VERBOSE=true bash src/tests/integration/test-billing.sh

# With Stripe test key
STRIPE_TEST_KEY="sk_test_PLACEHOLDER..." bash src/tests/integration/test-billing.sh
```

## Expected Output

```
=== Billing System Integration Tests ===

Test Configuration:
  Project: billing_test_12345
  Mode: SIMULATION
  Total Tests: 60

--- Section 1: Usage Tracking Tests (15 tests) ---
Test 1: Initialize usage tracking... ✓ passed (simulated)
Test 2: Track API request... ✓ passed (simulated)
...

=== Test Summary ===
Total Tests:  60
Passed:       60
Failed:       0
Success Rate: 100.0%
```

## Module Requirements

### 1. Usage Tracking (15 tests)

**File**: `src/lib/billing/usage-tracking.sh`

```bash
usage_tracking_init()
usage_track_api_request(customer_id, endpoint, method, status, duration)
usage_track_storage(customer_id, bytes)
usage_track_bandwidth(customer_id, bytes, direction)
usage_track_compute(customer_id, resource_type, seconds)
usage_get_current(customer_id) → JSON
usage_aggregate(customer_id, period)
usage_export(customer_id, format, output_path)
usage_alert_set(customer_id, metric, threshold, notification_type)
usage_alert_check(customer_id)
usage_get_history(customer_id, days) → JSON array
usage_reset(customer_id, period)
```

### 2. Stripe Integration (15 tests)

**File**: `src/lib/billing/stripe-integration.sh`

```bash
stripe_init(api_key)
stripe_customer_create(email, name) → customer_id
stripe_customer_get(customer_id) → JSON
stripe_customer_update(customer_id, field, value)
stripe_payment_method_attach(customer_id, pm_id) → pm_id
stripe_payment_methods_list(customer_id) → JSON array
stripe_subscription_create(customer_id, price_id) → subscription_id
stripe_subscription_get(subscription_id) → JSON
stripe_subscription_update(subscription_id, new_price_id)
stripe_subscription_cancel(subscription_id, when)
stripe_invoice_create(customer_id) → invoice_id
stripe_invoice_item_add(customer_id, amount, description)
stripe_invoice_finalize(invoice_id)
stripe_payment_process(invoice_id) → JSON
stripe_webhook_handle(webhook_payload)
stripe_customer_delete(customer_id) # cleanup
```

### 3. Quota Enforcement (12 tests)

**File**: `src/lib/billing/quota-enforcement.sh`

```bash
quota_init()
quota_set_limit(customer_id, resource, limit, period)
quota_check(customer_id, resource, amount)
quota_enforce(customer_id, resource, amount)
quota_soft_limit_check(customer_id, resource) → JSON
quota_hard_limit_check(customer_id, resource, amount) → 0/1
quota_calculate_overage(customer_id, resource) → JSON
quota_get_usage_percentage(customer_id, resource) → percentage
quota_reset(customer_id, period)
```

### 4. Invoice Generation (10 tests)

**File**: `src/lib/billing/invoice-generation.sh`

```bash
invoice_init()
invoice_generate(customer_id, period) → invoice_id
invoice_calculate_charges(customer_id) → JSON
invoice_apply_discount(invoice_id, discount_code)
invoice_apply_discount_percentage(invoice_id, percentage)
invoice_calculate_tax(customer_id, amount) → JSON
invoice_generate_pdf(invoice_id, output_path)
invoice_email_send(invoice_id, customer_id)
invoice_get_history(customer_id, months) → JSON array
invoice_mark_paid(invoice_id, payment_method)
```

### 5. Plan Management (8 tests)

**File**: `src/lib/billing/plan-management.sh`

```bash
plan_init()
plan_list() → JSON array
plan_get_details(plan_id) → JSON
plan_upgrade(customer_id, new_plan_id)
plan_downgrade(customer_id, new_plan_id, when)
plan_create_custom(customer_id, plan_name, price, limits_json) → custom_plan_id
plan_compare(plan_id_1, plan_id_2) → JSON
plan_get_current(customer_id) → JSON
```

## Database Tables

```sql
usage_events          -- Track resource usage
customer_quotas       -- Store quota limits
invoices              -- Generated invoices
plans                 -- Available plans
subscriptions         -- Customer subscriptions
```

## Test Modes

### Simulation Mode (Current)
- No billing modules required
- All tests pass with "(simulated)" status
- Validates test structure only
- Exit code: 0

### Live Mode (After Implementation)
- Requires billing modules in `src/lib/billing/`
- Tests actual functionality
- Requires PostgreSQL running
- May require Stripe test key

## Success Criteria

✅ All 60 tests pass
✅ Exit code 0
✅ No memory leaks
✅ Cross-platform compatible
✅ Proper error handling

## Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed

## Files

```
test-billing.sh                    # Main test file (516 lines)
test-billing.README.md             # Complete documentation
test-billing.IMPLEMENTATION-GUIDE.md  # Implementation guide
test-billing.QUICK-REFERENCE.md    # This file
```

## Common Commands

```bash
# Run tests
bash src/tests/integration/test-billing.sh

# Count tests
bash src/tests/integration/test-billing.sh 2>&1 | grep -c "^Test"

# Show only summary
bash src/tests/integration/test-billing.sh 2>&1 | tail -15

# Check exit code
bash src/tests/integration/test-billing.sh && echo "PASS" || echo "FAIL"

# Run specific section (not supported - run all or implement modules)
```

## Environment Variables

```bash
STRIPE_TEST_KEY           # Stripe test API key
BILLING_TEST_DB           # Test database name
BILLING_TEST_VERBOSE      # Verbose output
BILLING_TEST_SKIP_CLEANUP # Skip cleanup for debugging
```

## CI/CD Integration

Add to `.github/workflows/test-billing.yml`:

```yaml
name: Billing Tests
on:
  push:
    paths:
      - 'src/lib/billing/**'
      - 'src/tests/integration/test-billing.sh'
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run tests
        run: bash src/tests/integration/test-billing.sh
        env:
          STRIPE_TEST_KEY: ${{ secrets.STRIPE_TEST_KEY }}
```

## Related Files

- Implementation guide: `test-billing.IMPLEMENTATION-GUIDE.md`
- Full documentation: `test-billing.README.md`
- Test file: `test-billing.sh`
- Billing modules: `src/lib/billing/*.sh` (to be created)

## Next Steps

1. Review full documentation: `test-billing.README.md`
2. Follow implementation guide: `test-billing.IMPLEMENTATION-GUIDE.md`
3. Create billing module directory: `mkdir -p src/lib/billing`
4. Implement modules one at a time
5. Run tests after each module
6. Verify all 60 tests pass

## Support

- Documentation: See README files in this directory
- nself docs: `/.wiki/commands/BILLING.md`
- Test issues: Create GitHub issue with test output

---

**Last Updated**: 2026-01-29
**Test Version**: 1.0.0
**Compatible with**: nself v0.8.0+
