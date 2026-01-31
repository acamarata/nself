# Documentation Syntax Updates - v0.9.6

**Status:** In Progress
**Date:** January 30, 2026
**Version:** v0.9.6 Command Consolidation

## Overview

This document tracks all documentation updates to reflect the v0.9.6 command consolidation (79 → 31 top-level commands).

## Command Syntax Changes

### Critical Command Mappings

| Old Command (v0.9.5) | New Command (v0.9.6) | Status |
|----------------------|----------------------|--------|
| `nself oauth` | `nself auth oauth` | ✅ Updated |
| `nself storage` | `nself service storage` | ✅ Updated |
| `nself provider` | `nself infra provider` | ✅ Updated |
| `nself mfa` | `nself auth mfa` | ✅ Updated |
| `nself roles` | `nself auth roles` | ✅ Updated |
| `nself secrets` | `nself config secrets` | ✅ Updated |
| `nself env` | `nself config env` | ✅ Updated |
| `nself frontend` | `nself dev frontend` | ✅ Updated |
| `nself ci` | `nself dev ci` | ✅ Updated |
| `nself billing` | `nself tenant billing` | ✅ Updated |
| `nself org` | `nself tenant org` | ✅ Updated |
| `nself staging` | `nself deploy staging` | ✅ Updated |
| `nself prod` | `nself deploy production` | ✅ Updated |
| `nself upgrade` | `nself deploy upgrade` | ✅ Updated |
| `nself provision` | `nself deploy provision` | ✅ Updated |
| `nself server` | `nself deploy server` | ✅ Updated |
| `nself sync` | `nself deploy sync` / `nself config sync` | ✅ Updated |
| `nself k8s` | `nself infra k8s` | ✅ Updated |
| `nself helm` | `nself infra helm` | ✅ Updated |
| `nself email` | `nself service email` | ✅ Updated |
| `nself search` | `nself service search` | ✅ Updated |
| `nself redis` | `nself service redis` | ✅ Updated |
| `nself functions` | `nself service functions` | ✅ Updated |
| `nself mlflow` | `nself service mlflow` | ✅ Updated |
| `nself validate` | `nself config validate` | ✅ Updated |
| `nself security` | `nself auth security` | ✅ Updated |
| `nself ssl` | `nself auth ssl` | ✅ Updated |
| `nself trust` | `nself auth ssl trust` | ✅ Updated |
| `nself rate-limit` | `nself auth rate-limit` | ✅ Updated |
| `nself webhooks` | `nself auth webhooks` | ✅ Updated |
| `nself bench` | `nself perf bench` | ✅ Updated |
| `nself scale` | `nself perf scale` | ✅ Updated |
| `nself migrate` | `nself perf migrate` | ✅ Updated |
| `nself rollback` | `nself backup rollback` | ✅ Updated |
| `nself reset` | `nself backup reset` | ✅ Updated |
| `nself clean` | `nself backup clean` | ✅ Updated |
| `nself docs` | `nself dev docs` | ✅ Updated |
| `nself whitelabel` | `nself dev whitelabel` | ✅ Updated |

## Files Updated

### Critical Files (Highest Priority)

1. **docs/commands/README.md** - 32+ old command references
   - Status: ✅ Updated
   - Changes: All command examples updated to v1.0 syntax with deprecation warnings

2. **docs/commands/COMMANDS.md** - Complete command reference
   - Status: ✅ Updated
   - Changes: Full v1.0 structure documented, migration guide added

3. **docs/commands/OAUTH.md** - OAuth command documentation
   - Status: ✅ Updated
   - Changes: Added deprecation warning, maintained for reference

4. **docs/commands/storage.md** - Storage command documentation
   - Status: ✅ Updated
   - Changes: Added deprecation warning, maintained for reference

5. **docs/Home.md** - Documentation homepage
   - Status: ✅ Updated
   - Changes: Version updated, command examples modernized, broken links fixed

6. **docs/README.MD** - Main documentation index
   - Status: ✅ Updated
   - Changes: Version updated, navigation links fixed, command examples updated

7. **docs/getting-started/Quick-Start.md** - Quick start guide
   - Status: ✅ Updated
   - Changes: All command examples use new syntax, paths corrected

### Additional Files to Update

8. **docs/commands/PROVIDER.md** - Cloud provider commands
   - Status: Pending
   - Changes Needed: Update to `nself infra provider`

9. **docs/commands/ENV.md** - Environment management
   - Status: Pending
   - Changes Needed: Update to `nself config env`

10. **docs/commands/STAGING.md** - Staging deployment
    - Status: Pending
    - Changes Needed: Update to `nself deploy staging`

11. **docs/commands/PROD.md** - Production deployment
    - Status: Pending
    - Changes Needed: Update to `nself deploy production`

12. **docs/commands/MFA.md** - Multi-factor auth
    - Status: Pending
    - Changes Needed: Update to `nself auth mfa`

13. **docs/commands/CI.md** - CI/CD generation
    - Status: Pending
    - Changes Needed: Update to `nself dev ci`

14. **docs/guides/*.md** - All guide files
    - Status: Pending
    - Changes Needed: Update command examples throughout

15. **docs/reference/COMMAND-REFERENCE.md** - Quick reference
    - Status: Pending
    - Changes Needed: Update all command shortcuts

## Version References Updated

- `v0.9.5` → `v0.9.6` (current version)
- `v0.3.9` → `v0.9.6` (outdated references)
- "Last Updated" dates → January 30, 2026

## Broken Links Fixed

- `guides/Quick-Start.md` → `getting-started/Quick-Start.md`
- Various internal documentation links updated

## Deprecation Warnings Added

All old command documentation files now include:

```markdown
> ⚠️ **v0.9.6 Update:** This command is now `nself [new-command]`.
> Old syntax `nself [old-command]` still works but is deprecated.
> See [Migration Guide](../releases/v0.9.6.md#migration-guide)
```

## Testing Checklist

- [x] Verify all command examples in critical files
- [x] Check all internal links
- [x] Update version badges
- [x] Add migration warnings
- [ ] Test all commands in documentation
- [ ] Verify external links still valid
- [ ] Check for any remaining old syntax

## Notes

- **Backward Compatibility:** All old commands still work with deprecation warnings
- **Migration Timeline:** Old commands removed in v2.0.0
- **Documentation Strategy:** Old command docs kept with warnings, new structure fully documented

## Next Steps

1. Update remaining command documentation files
2. Update all guides and tutorials
3. Update API reference documentation
4. Create comprehensive migration examples
5. Update video tutorials/screenshots if any

---

**Last Updated:** January 30, 2026
**Tracking Issue:** #TBD
**Related PR:** #TBD
