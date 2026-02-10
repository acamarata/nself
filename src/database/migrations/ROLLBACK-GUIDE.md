# Database Migration Rollback Guide

This document describes the rollback migrations for the billing and white-label systems.

## Rollback Migrations

### 017_rollback_billing_system.sql

Safely removes all billing system objects created by `015_create_billing_system.sql`.

**Removed Objects:**
- Tables (8 total):
  - `billing_customers` - Customer accounts for billing
  - `billing_plans` - Available subscription plans
  - `billing_subscriptions` - Customer subscription records
  - `billing_quotas` - Quota limits per plan/service
  - `billing_usage_records` - Raw usage tracking records
  - `billing_invoices` - Customer invoices and bills
  - `billing_payment_methods` - Customer payment methods
  - `billing_events` - Stripe webhook events log

- Materialized Views:
  - `billing_usage_daily_summary` - Daily aggregated usage summary

- Functions:
  - `get_quota_usage()` - Get current quota usage for customer/service
  - `is_quota_exceeded()` - Check if quota exceeded
  - `refresh_billing_usage_summary()` - Refresh materialized view
  - `update_updated_at_column()` - Generic trigger function (if no other usage)

- Triggers (6 total):
  - `update_billing_customers_updated_at`
  - `update_billing_plans_updated_at`
  - `update_billing_subscriptions_updated_at`
  - `update_billing_quotas_updated_at`
  - `update_billing_invoices_updated_at`
  - `update_billing_payment_methods_updated_at`

- Indexes: All `idx_billing_*` indexes

**Drop Order (Dependency Chain):**
1. Triggers (depend on functions)
2. Functions (depend on tables)
3. Materialized Views (depend on tables)
4. Tables (in reverse dependency order)

### 018_rollback_whitelabel_system.sql

Safely removes all white-label system objects created by `016_create_whitelabel_system.sql`.

**Removed Objects:**
- Tables (5 total):
  - `whitelabel_brands` - White-label brand configurations
  - `whitelabel_domains` - Custom domains with SSL/DNS verification
  - `whitelabel_themes` - UI themes with CSS variables
  - `whitelabel_email_templates` - Custom email templates
  - `whitelabel_assets` - Logos, images, fonts, and other assets

- Views:
  - `whitelabel_brands_full` - Complete brand info with themes and domains

- Functions:
  - `update_whitelabel_updated_at()` - Update timestamp trigger function

- Triggers (5 total):
  - `trigger_whitelabel_brands_updated_at`
  - `trigger_whitelabel_domains_updated_at`
  - `trigger_whitelabel_themes_updated_at`
  - `trigger_whitelabel_email_templates_updated_at`
  - `trigger_whitelabel_assets_updated_at`

- Foreign Key Constraints:
  - All `fk_*` constraints from ALTER TABLE statements

- Indexes: All `idx_whitelabel_*` indexes

**Drop Order (Dependency Chain):**
1. Triggers (depend on functions)
2. Views (depend on tables)
3. Foreign Key Constraints (from ALTER statements)
4. Tables (in reverse dependency order)

## Safety Features

All rollback migrations use the following safety patterns:

### 1. Transactions
```sql
BEGIN TRANSACTION;
-- All DROP statements
COMMIT;
```
- Ensures atomic rollback: either all objects drop or none do
- Prevents partial deletions

### 2. IF EXISTS Clauses
```sql
DROP TABLE IF EXISTS table_name;
DROP FUNCTION IF EXISTS function_name();
DROP TRIGGER IF EXISTS trigger_name ON table_name;
```
- Prevents errors if objects don't exist
- Safe to run multiple times (idempotent)

### 3. Correct Dependency Order
```
Triggers → Functions → Materialized Views → Tables
```
- Functions are referenced by triggers (must drop triggers first)
- Materialized views depend on tables (must drop views first)
- Foreign keys automatically cascade (handled by ON DELETE CASCADE)

### 4. Explicit Constraint Removal
```sql
ALTER TABLE table_name
  DROP CONSTRAINT IF EXISTS constraint_name,
  DROP CONSTRAINT IF EXISTS constraint_name;
```
- Constraints added via ALTER must be dropped via ALTER
- Makes dependency chain explicit

## Usage

### Running Individual Rollback

```bash
# Rollback billing system
psql -U postgres -d nself -f src/database/migrations/017_rollback_billing_system.sql

# Rollback white-label system
psql -U postgres -d nself -f src/database/migrations/018_rollback_whitelabel_system.sql
```

### Running with Docker Compose

```bash
# Inside running PostgreSQL container
docker-compose exec postgres psql -U postgres -d nself -f /migrations/017_rollback_billing_system.sql

# Or via docker run
docker run --rm -v $(pwd)/src/database:/migrations \
  postgres:15 \
  psql -h host.docker.internal -U postgres -d nself \
  -f /migrations/017_rollback_billing_system.sql
```

### Testing Rollback + Re-Apply

See `tests/test-rollback-migrations.sql` for comprehensive testing script.

```bash
# Run all tests
psql -U postgres -d nself_test -f src/database/tests/test-rollback-migrations.sql
```

Tests verify:
1. Forward migration creates all objects
2. Rollback migration removes all objects
3. Forward migration can be re-applied successfully
4. No orphaned constraints or dependencies

## Verification

### Verify Billing System Removed

```sql
-- Should return 0 rows
SELECT COUNT(*) FROM information_schema.tables
WHERE table_name LIKE 'billing_%' AND table_schema = 'public';

-- Should return 0 rows
SELECT COUNT(*) FROM information_schema.views
WHERE table_name = 'billing_usage_daily_summary';

-- Should return 0 rows
SELECT COUNT(*) FROM information_schema.routines
WHERE routine_name IN ('get_quota_usage', 'is_quota_exceeded')
AND routine_schema = 'public';
```

### Verify White-Label System Removed

```sql
-- Should return 0 rows
SELECT COUNT(*) FROM information_schema.tables
WHERE table_name LIKE 'whitelabel_%' AND table_schema = 'public';

-- Should return 0 rows
SELECT COUNT(*) FROM information_schema.views
WHERE table_name = 'whitelabel_brands_full';

-- Should return 0 rows
SELECT COUNT(*) FROM information_schema.routines
WHERE routine_name = 'update_whitelabel_updated_at'
AND routine_schema = 'public';
```

## Recovery

If rollback fails partway through (should not happen with transactions):

1. **Identify remaining objects:**
   ```sql
   SELECT * FROM information_schema.tables
   WHERE table_name LIKE 'billing_%' OR table_name LIKE 'whitelabel_%';
   ```

2. **Clean up manually:**
   - Drop triggers
   - Drop functions
   - Drop views
   - Drop tables in dependency order

3. **Re-run rollback migration** with IF EXISTS clauses

## Rollback Considerations

### Data Loss

**WARNING:** Rollback migrations DROP ALL DATA:
- All customer records deleted
- All usage data deleted
- All invoices deleted
- All brand configurations deleted
- All custom domains deleted

**Before rollback:**
- Back up production database
- Export critical data if needed
- Notify stakeholders
- Plan re-deployment

### Production Safety

**Recommendations:**
1. Always test on staging first
2. Backup database before rollback
3. Run during maintenance window
4. Have forward migration ready for re-deployment
5. Monitor application logs after rollback

### Idempotency

All rollback migrations are **idempotent** (safe to run multiple times):
- Uses `DROP IF EXISTS` for all objects
- Wrapped in transaction for atomicity
- Can be re-run if connection drops mid-execution

## Integration with Migration System

### Migration Number Sequence

```
015 - Create Billing System (forward)
016 - Create White-Label System (forward)
017 - Rollback Billing System (reverse)
018 - Rollback White-Label System (reverse)
```

### Migration Framework

nself uses sequential migration numbering. When adding new migrations:

1. Increment migration number
2. Follow naming convention: `NNN_description.sql`
3. Create corresponding rollback migration
4. Place both in `src/database/migrations/`
5. Update migration tracker/version file

### Hasura Integration

If using Hasura:
1. Rollback removes tables from GraphQL schema
2. Hasura cache may need invalidation
3. Re-apply migration to restore schema
4. Consider Hasura metadata versioning

## Common Issues

### Issue: Foreign Key Constraint Violation

**Symptom:** `ERROR: cannot drop table because other objects depend on it`

**Solution:**
1. Verify all dependent objects are being dropped
2. Check for triggers/views not listed
3. Manually drop dependent objects first:
   ```sql
   DROP VIEW IF EXISTS dependent_view CASCADE;
   ```

### Issue: Function Still in Use

**Symptom:** `ERROR: cannot drop function because other objects depend on it`

**Solution:**
1. Drop triggers first (they call functions)
2. Drop materialized views (may use functions)
3. Then drop functions

### Issue: Transaction Rolled Back

**Symptom:** `ROLLBACK` instead of `COMMIT` appears in output

**Solution:**
1. Check error message in output
2. Fix the issue (e.g., missing IF EXISTS)
3. Re-run migration
4. Verify no partial state left

## Performance

### Execution Time Estimates

- **Billing system rollback:** 100-500ms
  - Depends on number of records in billing_usage_records
  - Cascading deletes may take time if many records

- **White-label system rollback:** 50-200ms
  - No large data tables
  - Quick constraint removal

- **With large data:** Can take 5-30+ seconds
  - Consider disabling triggers on source tables
  - Use VACUUM ANALYZE after rollback

### Optimization

For large tables, consider:

```sql
-- Disable triggers temporarily (if not ON DELETE CASCADE)
ALTER TABLE table_name DISABLE TRIGGER trigger_name;

-- Drop data more efficiently
TRUNCATE TABLE large_table CASCADE;

-- Drop object
DROP TABLE large_table;

-- Re-enable triggers
ALTER TABLE table_name ENABLE TRIGGER trigger_name;
```

## Troubleshooting

### Enable Detailed Output

```bash
# Show all SQL statements
psql -U postgres -d nself -f rollback.sql --echo-all

# Show query timing
psql -U postgres -d nself -f rollback.sql --echo-all -t
```

### Debug Mode

Add to rollback script:

```sql
-- Show what's being dropped
\dt billing_*
\df get_quota_usage
\dv billing_usage_daily_summary

-- Run rollback
\i 017_rollback_billing_system.sql

-- Verify removal
\dt billing_*
```

## See Also

- `015_create_billing_system.sql` - Billing system forward migration
- `016_create_whitelabel_system.sql` - White-label forward migration
- `tests/test-rollback-migrations.sql` - Rollback testing script
- `/.wiki/migrations/README.md` - Overall migration strategy

---

Last Updated: 2026-01-30
Version: 0.9.0
