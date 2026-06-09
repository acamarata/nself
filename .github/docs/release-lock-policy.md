# Release Lock Policy — nSelf Ecosystem

**Document type:** Ecosystem-wide policy (applies to all 12 repos under nself-org)
**Authored:** 2026-06-01 (P1-E1-W1-S01-T02 version reconciliation)
**Authority:** GCI Version & Release Lock Hard Rule + phase-end-shipping.md

---

## Purpose

This document records the version reconciliation outcome from P1-T02, establishes the release lock policy for Phase 1, and serves as the canonical reference for any agent or engineer making release decisions.

---

## Version Reconciliation Outcome (2026-06-01)

Reconciliation was performed against disk version files and git tags. No version bumps were applied. Results:

| Repo | Documented | Actual | Delta | Status |
|---|---|---|---|---|
| cli | v1.1.1 (PPI) | v1.1.5 (disk+tag) | +4 patches | MASTER-VERSIONS.md updated |
| admin | v1.1.1 (PPI) | v1.1.5 (disk) | +4 patches | MASTER-VERSIONS.md updated |
| homebrew-nself | v1.1.1 (PPI) | v1.1.5 (formula) | +4 patches | MASTER-VERSIONS.md updated |
| plugins-pro | v1.1.10 (MV) | v1.1.12 (tag) | +2 patches | MASTER-VERSIONS.md updated |
| plugins (free) | undocumented | v1.1.3 (tag) | added | MASTER-VERSIONS.md updated |
| web | v1.0.13 (MV) | v1.1.3 (tag) | +minor (own semver) | MASTER-VERSIONS.md updated |
| nchat | undocumented | v1.1.4 (tag) | added | MASTER-VERSIONS.md updated |
| nclaw | undocumented | v1.1.3 (tag) | added | MASTER-VERSIONS.md updated |
| ntask | undocumented | v1.1.4 (tag) | added | MASTER-VERSIONS.md updated |
| ntv | v1.0.9 (MV) | v1.1.3 (tag) | +patches (own semver) | MASTER-VERSIONS.md updated |
| nfamily | v0.1.0 (MV) | v0.1.1 (tag) | +1 patch | MASTER-VERSIONS.md updated |
| clawde | undocumented | v0.3.2 Tauri (tag) | added | MASTER-VERSIONS.md updated |

MV = MASTER-VERSIONS.md prior state

---

## CLI ↔ Admin Lockstep Status

**Status: MAINTAINED at v1.1.5.** Both CLI and Admin are at v1.1.5. The homebrew formula references v1.1.5. No lockstep violation exists; MASTER-VERSIONS.md documentation was stale only.

Lockstep rule (hard): When CLI tags vX.Y.Z, Admin Docker must publish on the same release day. Any desync = `nself doctor --deep` hard error.

---

## Release Lock Rules for P1

### What is locked
- **No version bumps** during the planning phase (P1 is currently in planning). GCI Version & Release Lock Hard Rule prohibits bumps without a Plan phase-approved Release Plan.
- **No tags**, **no GitHub releases**, **no publishes** until Build phase close.

### What is auto-authorized (P1 Build phase close)
Per `phase-end-shipping.md § Version Bump Authority`:

| Action | Authorized | Requires |
|---|---|---|
| Patch bump (x.y.Z+1) for any touched artifact | YES — auto-authorized | Just Build phase close |
| Minor bump (x.Y.0) | NO | Explicit user instruction in current message turn |
| Major bump (X.0.0) | NO | Plan phase-approved Release Plan |
| git tag + GitHub release (patch) | YES — auto-authorized at Build close | Just Build phase close |
| git tag + GitHub release (minor/major) | NO | As above |
| Docker Hub push (admin :latest) | YES — auto-authorized at Build close | Same day as CLI tag |
| homebrew formula update | YES — auto-authorized at Build close | Tracks CLI tag automatically |
| npm publish (patch) | YES — auto-authorized at Build close | Just Build phase close |
| cargo publish (patch) | YES — auto-authorized at Build close | Just Build phase close |

### Standing authorization for P1 Build
At P1 Build phase start, the user may grant a standing authorization covering the full patch release cascade for all repos touched during the phase. This removes per-action confirmation for the above auto-authorized items. The standing authorization must be recorded in `.claude/phases/current/p1/standing-authorizations.md`.

---

## Multi-Phase Guidance

- MASTER-VERSIONS.md is the single authoritative version source. Update it at every release.
- REGISTRY-PACKAGES.md (SPORT) mirrors the package version rows.
- The PPI "Current version" line in CLAUDE.md should reflect CLI version and be updated at every CLI release.
- Future phases: run `pbd-verify-ready` which checks MASTER-VERSIONS.md consistency as part of the Ready gate.

---

## Reference

- `~/.claude/CLAUDE.md` § Version & Release Lock — Hard Rule
- `~/.claude/rules/phase-end-shipping.md` § Version Bump Authority
- `.claude/docs/MASTER-VERSIONS.md` — authoritative version table
- `.opencode/phases/sport/REGISTRY-PACKAGES.md` — SPORT package mirror
