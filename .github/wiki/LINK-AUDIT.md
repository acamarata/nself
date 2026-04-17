# Wiki Link Audit (S07-T03)

**Date:** 2026-04-17
**Auditor:** S07-T03 automated repair
**Scope:** All `[[wiki-link]]` references in `cli/.github/wiki/`

---

## Summary

E2 UAT (V04) identified 35 broken wiki-links in the cli wiki. The E2 Fleet I
documentation sync commit (0c91d73) resolved the majority before this audit ran.
This audit (S07-T03) performed a complete scan and resolved the 3 remaining issues.

**Final state:** 0 broken links. All `[[wiki-link]]` occurrences resolve to existing pages.

---

## Prior Repair (E2 Fleet I — 0c91d73): 32 Links Fixed by Creating Target Pages

These links had no target file. New pages were created to resolve them.

### Bundle pages (4 created)

| Link | File Created | Notes |
|---|---|---|
| `[[bundle-clawde]]` | `bundle-clawde.md` | ClawDE+ bundle reference |
| `[[bundle-nchat]]` | `bundle-nchat.md` | nChat bundle reference |
| `[[bundle-nclaw]]` | `bundle-nclaw.md` | nClaw bundle reference |
| `[[bundle-nfamily]]` | `bundle-nfamily.md` | nFamily bundle reference |

### Command pages (22 created)

| Link | File Created | Notes |
|---|---|---|
| `[[cmd-ai]]` | `cmd-ai.md` | `nself ai` command reference |
| `[[cmd-alerts]]` | `cmd-alerts.md` | `nself alerts` command reference |
| `[[cmd-backup]]` | `cmd-backup.md` | `nself backup` command reference |
| `[[cmd-billing]]` | `cmd-billing.md` | `nself billing` command reference |
| `[[cmd-claw]]` | `cmd-claw.md` | `nself claw` command reference |
| `[[cmd-dev]]` | `cmd-dev.md` | `nself dev` command reference |
| `[[cmd-dns-setup]]` | `cmd-dns-setup.md` | `nself dns-setup` command reference |
| `[[cmd-dogfood]]` | `cmd-dogfood.md` | `nself dogfood` command reference |
| `[[cmd-down]]` | `cmd-down.md` | `nself down` alias page (redirects to cmd-stop) |
| `[[cmd-dr]]` | `cmd-dr.md` | `nself dr` command reference |
| `[[cmd-env]]` | `cmd-env.md` | `nself env` command reference |
| `[[cmd-monitor]]` | `cmd-monitor.md` | `nself monitor` command reference |
| `[[cmd-promote]]` | `cmd-promote.md` | `nself promote` command reference |
| `[[cmd-queue]]` | `cmd-queue.md` | `nself queue` command reference |
| `[[cmd-secrets]]` | `cmd-secrets.md` | `nself secrets` command reference |
| `[[cmd-tenant]]` | `cmd-tenant.md` | `nself tenant` command reference |
| `[[cmd-trust]]` | `cmd-trust.md` | `nself trust` command reference |
| `[[cmd-up]]` | `cmd-up.md` | `nself up` alias page (redirects to cmd-start) |
| `[[cmd-upgrade]]` | `cmd-upgrade.md` | `nself upgrade` command reference |
| `[[cmd-waf]]` | `cmd-waf.md` | `nself waf` command reference |
| `[[cmd-watchdog]]` | `cmd-watchdog.md` | `nself watchdog` command reference |
| `[[cmd-webhooks]]` | `cmd-webhooks.md` | `nself webhooks` command reference |

### Plugin pages (3 net-new, 2 expanded)

| Link | File Created | Notes |
|---|---|---|
| `[[plugin-claw-news]]` | `plugin-claw-news.md` | Expanded from stub |
| `[[plugin-linkedin]]` | `plugin-linkedin.md` | Expanded from stub |
| `[[plugin-paypal-pro]]` | `plugin-paypal-pro.md` | New pro variant page |
| `[[plugin-shopify-pro]]` | `plugin-shopify-pro.md` | New pro variant page |
| `[[plugin-stripe-pro]]` | `plugin-stripe-pro.md` | New pro variant page |

### Index page (1 created)

| Link | File Created | Notes |
|---|---|---|
| `[[Plugin-Catalog]]` | `Plugin-Catalog.md` | Plugin + bundle index with tier table |

---

## Prior Repair (E2 Fleet I — 0c91d73): Orphaned Pages Removed

These pages existed but had no wiki-link pointing to them and referenced
plugins that are not in the canonical plugin registry (SPORT F03/F04). They were
deleted to prevent confusion.

| Page Deleted | Reason |
|---|---|
| `plugin-audit.md` | Plugin not in canonical registry |
| `plugin-blog.md` | Plugin not in canonical registry |
| `plugin-checkout.md` | Plugin not in canonical registry |
| `plugin-commerce.md` | Plugin not in canonical registry |
| `plugin-drm.md` | Plugin not in canonical registry |
| `plugin-export.md` | Plugin not in canonical registry |
| `plugin-flow.md` | Plugin not in canonical registry |
| `plugin-import.md` | Plugin not in canonical registry |
| `plugin-ldap.md` | Plugin not in canonical registry |
| `plugin-mailgun.md` | Plugin not in canonical registry |
| `plugin-media.md` | Plugin not in canonical registry (superseded by plugin-media-processing) |
| `plugin-oauth-providers.md` | Plugin not in canonical registry |
| `plugin-pages.md` | Plugin not in canonical registry |
| `plugin-postmark.md` | Plugin not in canonical registry |
| `plugin-rate-limit.md` | Plugin not in canonical registry |
| `plugin-reports.md` | Plugin not in canonical registry |
| `plugin-saml.md` | Plugin not in canonical registry |
| `plugin-scheduler.md` | Plugin not in canonical registry (superseded by plugin-cron) |
| `plugin-sendgrid.md` | Plugin not in canonical registry |
| `plugin-sso.md` | Plugin not in canonical registry |
| `plugin-subscription.md` | Plugin not in canonical registry |
| `plugin-thumb.md` | Plugin not in canonical registry |
| `plugin-transcoder.md` | Plugin not in canonical registry (superseded by plugin-media-processing) |
| `plugin-twilio.md` | Plugin not in canonical registry |
| `plugin-waf.md` | Plugin not in canonical registry (WAF is a CLI command, not a plugin) |
| `plugin-watermark.md` | Plugin not in canonical registry |

---

## S07-T03 Repair (2026-04-17): 3 Remaining Issues Resolved

### Fixed: `[[Commands]]` filename case mismatch

| Action | Classification | Details |
|---|---|---|
| Renamed `COMMANDS.md` to `Commands.md` | **fixed** | 68 occurrences of `[[Commands]]` across cmd-*.md pages, Home.md, Getting-Started.md, Plugin-Catalog.md, and _Sidebar.md all linked to `[[Commands]]`. The target file was named `COMMANDS.md` (all-caps). GitHub Wiki resolves links case-insensitively at runtime, but the canonical filename should match the link convention. Renamed to `Commands.md` per wiki naming rules (cross-cutting pages use title-case). |

### Deleted: Raw `[[wiki-links]]` and `[[wiki-link]]` in LINK-AUDIT.md body prose

| Action | Classification | Details |
|---|---|---|
| Removed `[[wiki-links]]` from LINK-AUDIT.md line 11 | **deleted** | The text `[[wiki-links]]` appeared in a prose sentence describing the audit scope. It is not a navigation link. Replaced with plain text `wiki-links`. |
| Removed `[[wiki-link]]` from LINK-AUDIT.md line 79 | **deleted** | The text `[[wiki-link]]` appeared in a prose sentence describing orphaned pages. It is not a navigation link. Replaced with plain text `wiki-link`. |

Note: Other occurrences of `[[wiki-link]]` in this file (lines 5, 15, 118) are already inside backtick code spans and are not parsed as wiki links by GitHub.

---

## Verification

Post-repair scan result: **0 broken links**.

All `[[wiki-link]]` occurrences across the wiki resolve to existing pages.
Backtick-enclosed `[[...]]` patterns (code spans) are excluded from link resolution per GitHub Wiki behavior.

Canonical reference for plugin membership: `.claude/docs/sport/F03-F04` (25 free + 62 pro).
