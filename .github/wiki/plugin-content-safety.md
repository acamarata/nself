# Content Safety Plugin

> Trust-safety evidence collection, legal holds, spam detection, raid protection, and abuse scoring. **Pro plugin.**

> **Requires:** Pro license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install content-safety
nself build
```

## What It Does

Adds a complete trust-safety layer to your nSelf deployment. Covers six distinct capability areas:

**Evidence collection** — Record incidents of spam, abuse, or policy violations. Each evidence record tracks type, source, workspace, priority (low/normal/high/critical), and encryption status. Evidence can be associated with a legal hold for compliance export.

**Legal holds** — Freeze evidence records for regulatory or litigation purposes. Scope a hold by workspace, channel, or custom criteria. Export held records as a JSON package with a time-limited download URL.

**Spam detection** — Define pattern-based and condition-based rules, order them by priority, and analyze content against the active rule set in one API call. Per-workspace configuration controls sensitivity (low/medium/high), quarantine threshold, auto-delete, and moderator notifications.

**Rate-limit violation logging** — Record and query limit breaches with user, channel, workspace, and action metadata. Useful for surfacing repeated offenders or diagnosing bot activity.

**Raid protection** — Detect join spikes and coordinated attack events. Log raid events with severity and resolution times. Apply and deactivate workspace or channel lockdowns (partial or full) programmatically.

**Abuse scoring** — Maintain a per-user trust score from 0.0 (high risk) to 1.0 (fully trusted). Update scores from positive and negative events; query risk level (low/medium/high) at any time.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `CONTENT_SAFETY_PLUGIN_PORT` | `3213` | HTTP listen port |
| `CONTENT_SAFETY_PLUGIN_HOST` | `0.0.0.0` | HTTP bind address |
| `CONTENT_SAFETY_API_KEY` | — | Static API key; omit to run without auth |

## Ports

| Port | Purpose |
|------|---------|
| 3213 | Content Safety REST API |

## Database Tables

9 tables under the `np_cs_*` prefix, all with `source_account_id` isolation and Row Level Security:

- `np_cs_evidence` — abuse/spam/violation evidence records
- `np_cs_legal_holds` — legal hold definitions
- `np_cs_evidence_exports` — evidence export jobs with download URLs
- `np_cs_spam_rules` — pattern/condition spam detection rules
- `np_cs_spam_configs` — per-workspace spam configuration
- `np_cs_rate_limit_violations` — rate-limit breach log
- `np_cs_raid_events` — raid detection event log
- `np_cs_lockdowns` — workspace/channel lockdown records
- `np_cs_abuse_scores` — per-user trust/risk scores

## Key Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Liveness probe |
| `POST` | `/api/v1/evidence` | Collect a trust-safety evidence record |
| `POST` | `/api/v1/legal-holds` | Create a legal hold |
| `POST` | `/api/v1/evidence/exports` | Export held evidence |
| `POST` | `/api/v1/spam/analyze` | Analyze content against active spam rules |
| `PUT` | `/api/v1/spam/config` | Update workspace spam configuration |
| `POST` | `/api/v1/raid/status` | Record a raid event |
| `POST` | `/api/v1/raid/lockdown` | Apply a workspace/channel lockdown |
| `DELETE` | `/api/v1/raid/lockdown/{id}` | Deactivate a lockdown |
| `POST` | `/api/v1/abuse/trust` | Register user for abuse scoring |
| `PUT` | `/api/v1/abuse/trust` | Update user trust/risk score |

See the plugin README at `plugins-pro/paid/content-safety/README.md` for the full route list.

## Examples

Collect evidence and create a legal hold:

```bash
# Record evidence
curl -X POST http://localhost:3213/api/v1/evidence \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: $CONTENT_SAFETY_API_KEY' \
  -d '{"type":"harassment","content":{"text":"..."},"reason":"user_report","source":"nchat","workspace_id":"ws_001","priority":"high"}'

# Create a hold over that workspace
curl -X POST http://localhost:3213/api/v1/legal-holds \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: $CONTENT_SAFETY_API_KEY' \
  -d '{"name":"Legal Hold Q3","scope":{"workspace_id":"ws_001"},"criteria":{"after":"2026-01-01"}}'
```

Analyze content for spam:

```bash
curl -X POST http://localhost:3213/api/v1/spam/analyze \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: $CONTENT_SAFETY_API_KEY' \
  -d '{"workspace_id":"ws_001","content":"click this link now","user_id":"u_xyz"}'
```

Apply a raid lockdown:

```bash
curl -X POST http://localhost:3213/api/v1/raid/lockdown \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: $CONTENT_SAFETY_API_KEY' \
  -d '{"workspace_id":"ws_001","level":"full","reason":"join_spike_detected","activated_by":"admin"}'
```

## Pricing

| Tier | Price | Includes |
|------|-------|----------|
| Free | $0 | No |
| Any bundle | $0.99/mo | If bundle includes content-safety |
| ɳSelf+ | $3.99/mo | Yes |

**requires_license: true** — content-safety is a pro platform feature (trust-safety evidence, legal holds, spam engine, raid protection, abuse scoring). Basic security hardening (RLS, rate limits, WAF, TLS) ships free per the Security-Always-Free Doctrine; content safety is a product capability.

## See Also

- [[plugin-compliance]] — GDPR/CCPA/HIPAA compliance and DSARs
- [[plugin-moderation]] — real-time content moderation
- [[cmd-plugin]] — plugin management commands
- [[Home]]
