# Content Safety Plugin

> Trust-and-safety backend: evidence collection, legal holds, spam detection, raid protection, and abuse scoring. **Pro plugin.**

> **Requires:** a Pro license tier or higher. Set it with `nself license set nself_pro_...` before installing.

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install content-safety
```

## What It Does

Content Safety is the moderation and abuse-defense service for an nSelf backend, running on port 3213. It is the operator-facing trust-safety product layer — distinct from the always-free core hardening that ships with `nself install`, and distinct from `nself-csam` (PhotoDNA hashing).

- **Trust-safety evidence:** capture moderation evidence records, list them, and generate exports for review or external reporting.
- **Legal holds:** create and list legal holds that preserve evidence under a retention policy so it cannot be purged while a hold is active.
- **Spam detection:** analyze content against operator-defined spam rules, manage rule sets, configs, and per-action rate limits.
- **Raid protection:** record raid events, read and update raid status, and trigger lockdowns to defend against coordinated bursts.
- **Abuse scoring:** register and update per-actor trust/abuse scores that downstream apps can gate on.
- **Multi-app isolation:** every record is scoped by `source_account_id` (via `X-Hasura-Source-Account-Id`), so independent apps in one nSelf deploy never see each other's evidence, rules, or scores.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CONTENT_SAFETY_PLUGIN_PORT` | `3213` | Service port |
| `CONTENT_SAFETY_PLUGIN_HOST` | `0.0.0.0` | Listen host |
| `DATABASE_URL` | — | Postgres connection string |
| `CONTENT_SAFETY_API_KEY` | — | Optional `X-API-Key` gate for the API |

## API

Health checks are at `GET /health` and `GET /ready`. All business endpoints live under `/api/v1`, grouped as evidence (`/evidence`, `/evidence/exports`), legal holds (`/legal-holds`), trust-safety stats (`/trust-safety/stats`), spam (`/spam/analyze`, `/spam/config`, `/spam/rules`, `/spam/rate-limits`), raid (`/raid/status`, `/raid/lockdown`), and abuse (`/abuse/trust`).

## Licensing

This is a Pro, license-gated plugin (`requires_license: true`). Under the Security-Always-Free doctrine, core deployment hardening is free and automatic; Content Safety is an operator-facing moderation product (evidence retention, legal holds, configurable spam rules, raid lockdowns, abuse scoring), so it is licensed as a Pro feature.

## See Also

- [Plugin-Install](Plugin-Install) — general plugin install flow
- [Plugin-Catalog](Plugin-Catalog) — full plugin list
