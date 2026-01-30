# Database Rollback - Quick Reference

## Quick Start

### Run Billing Rollback
```bash
psql -U postgres -d nself -f src/database/migrations/017_rollback_billing_system.sql
```

### Run White-Label Rollback
```bash
psql -U postgres -d nself -f src/database/migrations/018_rollback_whitelabel_system.sql
```

### Run All Tests
```bash
psql -U postgres -d nself_test -f src/database/tests/test-rollback-migrations.sql
```

## Verify Rollback

### Check Billing System Removed
```sql
SELECT * FROM information_schema.tables
WHERE table_name LIKE 'billing_%' AND table_schema = 'public';
-- Should return 0 rows
```

### Check White-Label System Removed
```sql
SELECT * FROM information_schema.tables
WHERE table_name LIKE 'whitelabel_%' AND table_schema = 'public';
-- Should return 0 rows
```

## Objects Removed

### Billing System (017)
| Type | Count | Names |
|------|-------|-------|
| Tables | 8 | customers, plans, subscriptions, quotas, usage_records, invoices, payment_methods, events |
| Views | 1 | billing_usage_daily_summary |
| Functions | 4 | get_quota_usage, is_quota_exceeded, refresh_billing_usage_summary, update_updated_at_column |
| Triggers | 6 | All billing_* updated_at triggers |
| Indexes | 30+ | All idx_billing_* indexes |

### White-Label System (018)
| Type | Count | Names |
|------|-------|-------|
| Tables | 5 | brands, domains, themes, email_templates, assets |
| Views | 1 | whitelabel_brands_full |
| Functions | 1 | update_whitelabel_updated_at |
| Triggers | 5 | All whitelabel_* updated_at triggers |
| Constraints | 6 | All fk_* foreign key constraints |
| Indexes | 15+ | All idx_whitelabel_* indexes |

## Docker Usage

```bash
# In Docker Compose
docker-compose exec postgres psql -U postgres -d nself \
  -f /migrations/017_rollback_billing_system.sql

# Via docker run
docker run --rm -v $(pwd)/src/database:/db \
  postgres:15 \
  psql -h host -U postgres -d nself -f /db/migrations/017_rollback_billing_system.sql
```

## Dependency Order (Why It Matters)

```
Triggers              ← Call functions
  ↓
Functions            ← Query tables & views
  ↓
Materialized Views   ← Depend on tables
  ↓
Views               ← Depend on tables
  ↓
Foreign Keys        ← Reference other tables
  ↓
Tables              ← Base objects
```

**Drop in REVERSE order!**

## Safety Features

| Feature | Purpose | Benefit |
|---------|---------|---------|
| `BEGIN/COMMIT` | Transaction wrapper | Atomic: all or nothing |
| `IF EXISTS` | Safe drop | Idempotent & no errors |
| Order | Dependency chain | No orphaned objects |
| Comments | Documentation | Clear intent |
| Verified | Tested thoroughly | Confidence in execution |

## Common Commands

### Full Rollback Both Systems
```bash
psql -U postgres -d nself \
  -f src/database/migrations/017_rollback_billing_system.sql \
  -f src/database/migrations/018_rollback_whitelabel_system.sql
```

### Re-Apply After Rollback
```bash
psql -U postgres -d nself \
  -f src/database/migrations/015_create_billing_system.sql \
  -f src/database/migrations/016_create_whitelabel_system.sql
```

### Test Full Cycle (Forward → Rollback → Re-Apply)
```bash
psql -U postgres -d nself_test -f src/database/tests/test-rollback-migrations.sql
```

### Backup Before Rollback
```bash
pg_dump -U postgres nself > nself-backup-$(date +%Y%m%d_%H%M%S).sql
```

### Restore From Backup
```bash
psql -U postgres -d nself < nself-backup-20260130_061000.sql
```

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `ERROR: cannot drop table` | Trigger/function still referenced - drop them first |
| `ROLLBACK` in output | Transaction failed - check error message |
| Some objects remain | Rerun the rollback (IF EXISTS makes it idempotent) |
| Function still in use | Drop triggers and views first |
| Foreign key error | Run ALTER to drop constraints first |

## Performance Estimates

| Operation | Time | Notes |
|-----------|------|-------|
| Billing rollback | 100-500ms | Fast - mostly schema drops |
| White-label rollback | 50-200ms | Very fast - fewer tables |
| Test suite | 30-60s | Includes forward/rollback/re-apply |
| With 1M rows | 5-30s+ | Cascade deletes may be slow |

## Pre-Flight Checklist

Before running rollback on production:

- [ ] Backed up database
- [ ] Exported critical data if needed
- [ ] Planned maintenance window
- [ ] Notified stakeholders
- [ ] Tested on staging first
- [ ] Have restore procedure ready
- [ ] Verified application handles missing tables
- [ ] Prepared re-deployment plan

## Post-Rollback Steps

After successful rollback:

1. Verify no objects remain (see "Verify Rollback" above)
2. Check application logs for errors
3. Monitor database connections
4. Plan re-deployment of forward migrations if needed

## More Information

- Detailed guide: `src/database/migrations/ROLLBACK-GUIDE.md`
- Full summary: `ROLLBACK_MIGRATION_SUMMARY.md`
- Forward migrations: `015_create_billing_system.sql`, `016_create_whitelabel_system.sql`
- Test suite: `src/database/tests/test-rollback-migrations.sql`

## File Locations

```
src/database/
├── migrations/
│   ├── 015_create_billing_system.sql       (forward)
│   ├── 016_create_whitelabel_system.sql    (forward)
│   ├── 017_rollback_billing_system.sql     (rollback) ← NEW
│   ├── 018_rollback_whitelabel_system.sql  (rollback) ← NEW
│   ├── ROLLBACK-GUIDE.md                   (detailed)
│   └── ROLLBACK-QUICKREF.md                (this file)
└── tests/
    └── test-rollback-migrations.sql        (test suite)
```

## Important Notes

⚠️ **DATA LOSS WARNING**
- Rollback deletes ALL data in billing and white-label systems
- Backup database before rolling back
- This is not reversible without restore

✓ **IDEMPOTENT & SAFE**
- Can run multiple times without errors
- IF EXISTS prevents failures
- Transaction wrapper ensures consistency
- Safe to retry on connection drop

📊 **PRODUCTION READY**
- Extensively tested
- Comprehensive documentation
- Clear error handling
- Performance optimized

---

Last Updated: 2026-01-30 | Version: 0.9.0
