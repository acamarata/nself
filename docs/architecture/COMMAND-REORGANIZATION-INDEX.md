# nself CLI Command Reorganization - Documentation Index

**Comprehensive proposal to reduce 77 top-level commands to 13 logical categories**

---

## Quick Start

**New to this proposal?** Start here:

1. **[Executive Summary](../../COMMAND-REORGANIZATION-SUMMARY.md)** (9.4 KB)
   - The problem and solution in ~5 minutes
   - Before/after examples
   - Migration strategy overview
   - **READ THIS FIRST**

2. **[Visual Guide](./COMMAND-REORGANIZATION-VISUAL.md)** (23 KB)
   - Before/after visual comparison
   - The 13 categories explained
   - Quick reference card
   - Common workflows and FAQ

3. **[Consolidation Map](./COMMAND-CONSOLIDATION-MAP.md)** (19 KB)
   - Visual flow diagrams
   - Command mapping tables
   - Impact analysis
   - Migration complexity guide

---

## Detailed Documentation

### For Decision Makers

**[Complete Proposal](./COMMAND-REORGANIZATION-PROPOSAL.md)** (39 KB)
- Full rationale and analysis
- Detailed category breakdown
- 4-phase migration strategy
- Risk analysis and mitigation
- Success metrics
- Alternatives considered

**Key Sections:**
- Current State Analysis
- Proposed Reorganization (13 categories)
- Detailed Reorganization (category-by-category)
- Migration Strategy (4 phases)
- Backward Compatibility (50+ legacy aliases)
- Benefits and Risks

---

### For Developers

**[Implementation Checklist](./COMMAND-REORGANIZATION-CHECKLIST.md)** (14 KB)
- Phase-by-phase tasks
- File structure changes
- Code changes required
- Testing requirements
- Documentation updates
- Success criteria
- Rollback plan

**Key Sections:**
- Phase 1: Add New Commands (weeks 1-2)
- Phase 2: Add Deprecation Warnings (weeks 3-6)
- Phase 3: Legacy Alias System (ongoing)
- Phase 4: Remove Old Commands (6-12 months)

---

## Document Overview

| Document | Size | Purpose | Audience |
|----------|------|---------|----------|
| [Executive Summary](../../COMMAND-REORGANIZATION-SUMMARY.md) | 9.4 KB | Quick overview | Everyone |
| [Visual Guide](./COMMAND-REORGANIZATION-VISUAL.md) | 23 KB | Visual comparison | Users, PMs |
| [Consolidation Map](./COMMAND-CONSOLIDATION-MAP.md) | 19 KB | Command flows | Users, Devs |
| [Complete Proposal](./COMMAND-REORGANIZATION-PROPOSAL.md) | 39 KB | Full details | Decision makers |
| [Implementation Checklist](./COMMAND-REORGANIZATION-CHECKLIST.md) | 14 KB | Development tasks | Developers |
| **TOTAL** | **104.4 KB** | **Complete documentation** | - |

---

## The Reorganization at a Glance

### The Problem
```
77 top-level commands → Confusion, cognitive overload, poor discoverability
```

### The Solution
```
13 logical categories → Clear grouping, easy discovery, backward compatible
```

### Reduction
```
77 commands → 22 top-level (71% reduction)
```

---

## The 13 Categories

```
Core (6 commands)
├─ init, build, start, stop, restart, status

Data & Business Logic (4 categories)
├─ db          All database operations
├─ auth        All authentication & authorization
├─ tenant      All multi-tenancy
└─ service     All optional services

Infrastructure (3 categories)
├─ deploy      All deployment operations
├─ cloud       All cloud infrastructure
└─ observe     All monitoring & observability (NEW)

Security & Tools (3 categories)
├─ secure      All security operations (NEW)
├─ plugin      Plugin management
└─ dev         All developer tools

Configuration (2 categories)
├─ config      Configuration management
└─ help        Utilities (help, version, update, upgrade)
```

---

## Key Changes Summary

### NEW Categories (2)
1. **`observe`** - Consolidates 9 observability commands
   - logs, metrics, monitor, health, doctor, history, audit, urls, exec

2. **`secure`** - Consolidates 6 security commands
   - security, secrets, vault, ssl, trust

### EXPANDED Categories (6)
1. **`auth`** - Absorbs 7 auth-related commands
   - oauth, mfa, devices, roles, webhooks

2. **`service`** - Absorbs 10 optional service commands
   - admin, admin-dev, email, search, functions, mlflow, storage, redis, realtime, rate-limit

3. **`deploy`** - Absorbs 7 deployment commands
   - staging, prod, rollback, env, sync, validate, migrate

4. **`cloud`** - Absorbs 7 infrastructure commands
   - providers, provision, server, servers, k8s, helm

5. **`dev`** - Absorbs 6 developer tool commands
   - perf, bench, scale, frontend, ci, completion

6. **`config`** - Absorbs 2 configuration commands
   - clean, reset

### KEPT Categories (5)
- Core lifecycle: init, build, start, stop, restart, status
- Database: db
- Multi-tenant: tenant
- Plugins: plugin
- Utilities: help, version, update, upgrade

---

## Migration Examples

### Before (Current)
```bash
# Scattered across 77 top-level commands
nself logs postgres
nself oauth enable
nself admin open
nself staging
nself k8s deploy
nself perf profile
```

### After (Proposed)
```bash
# Organized into 13 categories
nself observe logs postgres
nself auth oauth enable
nself service admin open
nself deploy staging
nself cloud k8s deploy
nself dev perf profile
```

---

## Backward Compatibility

**All existing commands supported via legacy aliases for 2+ major versions:**

- Phase 1: New commands work, old commands work (no warnings)
- Phase 2: Old commands show deprecation warning, still work
- Phase 3: Legacy aliases maintained, usage tracked
- Phase 4: Old commands removed (v1.0), helpful errors shown

**Example:**
```bash
$ nself logs postgres
⚠️  DEPRECATED: Use 'nself observe logs postgres'
[logs continue normally...]
```

---

## Timeline

| Phase | Duration | Status | Description |
|-------|----------|--------|-------------|
| **Phase 1** | Weeks 1-2 | Proposed | Add new commands alongside old |
| **Phase 2** | Weeks 3-6 | Proposed | Add deprecation warnings |
| **Phase 3** | Ongoing | Proposed | Maintain legacy aliases |
| **Phase 4** | 6-12 months | Proposed | Remove old commands (v1.0) |

**Total Migration**: 6-12 months from start to completion

---

## Benefits

### For Users
- ✅ Easier discovery (13 categories vs 77 commands)
- ✅ Logical grouping (related features together)
- ✅ Less cognitive load (clear mental model)
- ✅ Better help (organized by category)
- ✅ Backward compatible (scripts won't break)

### For Developers
- ✅ Easier maintenance (related code together)
- ✅ Clear ownership (category-based)
- ✅ Better testing (test by category)
- ✅ Less duplication (shared logic)

---

## Statistics

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| Top-Level Commands | 77 | 22 | **-71%** |
| Categories | N/A | 13 | New structure |
| Help Text Length | ~300 lines | ~150 lines | **-50%** |
| Auth Commands | 8 scattered | 1 category | **-87%** |
| Service Commands | 11 scattered | 1 category | **-91%** |
| Deploy Commands | 8 scattered | 1 category | **-87%** |
| Observe Commands | 9 scattered | 1 category | **-89%** |
| Secure Commands | 6 scattered | 1 category | **-83%** |

---

## Implementation Effort

**Total Estimated Effort**: 100-120 hours over 6-12 months

- Phase 1 (Add new): 40-60 hours
- Phase 2 (Deprecate): 20-30 hours
- Phase 3 (Maintain): 5-10 hours/month
- Phase 4 (Remove): 10-15 hours

**Risk Level**: Low (backward compatible, phased approach)

---

## Risks & Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking scripts | Legacy aliases for 2+ versions |
| User confusion | Both systems work simultaneously |
| Longer commands | Shell aliases, tab completion |
| Implementation complexity | Phased rollout, extensive testing |

---

## Success Criteria

### Quantitative
- ✅ 71% reduction in top-level commands
- ✅ 50% reduction in help text
- ✅ Faster tab completion
- ✅ Reduced support tickets

### Qualitative
- ✅ Users find commands without docs
- ✅ Logical grouping makes sense
- ✅ Help text is scannable
- ✅ New users onboard faster

---

## How to Read This Documentation

### For a Quick Overview (5 minutes)
1. Read [Executive Summary](../../COMMAND-REORGANIZATION-SUMMARY.md)
2. Skim [Visual Guide](./COMMAND-REORGANIZATION-VISUAL.md) - look at diagrams

### For Decision Making (30 minutes)
1. Read [Executive Summary](../../COMMAND-REORGANIZATION-SUMMARY.md)
2. Read [Complete Proposal](./COMMAND-REORGANIZATION-PROPOSAL.md) - focus on:
   - Current State Analysis
   - Proposed Reorganization
   - Migration Strategy
   - Benefits and Risks

### For Implementation (2-3 hours)
1. Read [Complete Proposal](./COMMAND-REORGANIZATION-PROPOSAL.md) in full
2. Study [Consolidation Map](./COMMAND-CONSOLIDATION-MAP.md) - understand flows
3. Follow [Implementation Checklist](./COMMAND-REORGANIZATION-CHECKLIST.md) step-by-step

### For Migration Planning (1 hour)
1. Read [Visual Guide](./COMMAND-REORGANIZATION-VISUAL.md) - command mappings
2. Study [Consolidation Map](./COMMAND-CONSOLIDATION-MAP.md) - flow diagrams
3. Review migration examples in [Complete Proposal](./COMMAND-REORGANIZATION-PROPOSAL.md)

---

## Related Documentation

- **[Current Commands Reference](../commands/COMMANDS.md)** - Current v0.9.0 command structure
- **[Architecture Overview](./ARCHITECTURE.md)** - nself architecture documentation
- **[Release Roadmap](../releases/ROADMAP.md)** - Future release plans

---

## Contributing to This Proposal

### Provide Feedback
- Open a GitHub issue with tag `command-reorganization`
- Comment on specific sections that need revision
- Suggest alternative category names or groupings

### Questions?
- Review the FAQ in [Visual Guide](./COMMAND-REORGANIZATION-VISUAL.md)
- Check command mappings in [Consolidation Map](./COMMAND-CONSOLIDATION-MAP.md)
- Open a discussion on GitHub

---

## Status

**Current Status**: Proposal ready for stakeholder review

**Next Steps**:
1. Stakeholder review and feedback
2. Finalize categories and naming
3. Approve for Phase 1 implementation
4. Begin development (create observe.sh, secure.sh, expand existing)

---

## Document Changelog

| Date | Change | Author |
|------|--------|--------|
| 2026-01-30 | Initial proposal created | Automated Analysis |
| 2026-01-30 | All 5 documents completed (104 KB) | Automated Analysis |
| 2026-01-30 | Added to main commands documentation | Automated Analysis |

---

## Summary

This reorganization represents a **significant improvement** to nself's CLI usability:

- **Problem**: 77 scattered top-level commands creating confusion
- **Solution**: 13 logical categories with clear grouping
- **Impact**: 71% reduction in top-level commands
- **Risk**: Low - backward compatible with phased migration
- **Timeline**: 6-12 months for full migration
- **Effort**: 100-120 hours total

**Recommendation**: Proceed with Phase 1 implementation.

---

**Documentation Complete** | Total: 104.4 KB across 5 documents
