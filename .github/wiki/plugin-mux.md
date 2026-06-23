# Mux Plugin

> Email and webhook pipeline engine with AI-powered classification and YAML rule routing. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|-----------------------|
| Free | $0 | $0 | No |
| Basic (ɳClaw bundle) | $0.99/mo | $9.99/yr | Yes |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** Basic via the ɳClaw bundle (per F07-PRICING-TIERS).

## Bundle membership

This plugin is included in the following bundles:

- **ɳClaw** ($0.99/mo) — see [[bundle-nclaw]]

Or get all bundles and all apps via **[ɳSelf+](https://nself.org/plus)** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install mux
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked server-side; insufficient tier returns an error.

## Description

The mux plugin routes incoming emails and webhook events through a configurable pipeline. Rules are defined in YAML and matched against incoming messages using pattern conditions. Each matched message passes through one or more of 9 handler groups: PatternNotify, InboxRouter, SilentTrash, ContentForward, ObserveNotify, CalendarSync, AutoReply, SheetLogger, and DonorHandler.

Messages that fail delivery land in a dead-letter queue (DLQ) for manual review or automated retry. The plugin integrates with the `ai` plugin for intent classification and priority scoring, and uses the `notify` plugin to dispatch alerts via email, SMS, Slack, or Telegram.

All outbound HTTP calls (webhook forwarding, body-fetch pipeline) are SSRF-guarded. Private RFC-1918 and loopback addresses are blocked by default. The `MUX_WEBHOOK_ALLOW_INTERNAL` flag unlocks Docker bridge access for development.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MUX_PORT` | `3711` | Mux service port |
| `MUX_AI_ENABLED` | `true` | Use `ai` plugin for classification |
| `MUX_TG_BOT_TOKEN` | — | Telegram bot token for alerts |
| `MUX_TG_CHAT_ID` | — | Telegram chat ID for alerts |
| `MUX_DLQ_RETENTION` | `30` | Days to retain failed messages in DLQ |
| `MUX_DLQ_MAX_RETRIES` | `3` | Max DLQ retry attempts |
| `MUX_DLQ_RETRY_WEBHOOK_URL` | — | Webhook URL to call on DLQ retry |
| `MUX_INTERNAL_SECRET` | — | Shared secret with ai/claw/google/notify plugins |
| `PLUGIN_MUX_SHADOW_MODE` | `0` | Run in shadow mode (log only, no actions) |
| `PLUGIN_MUX_RULES_YAML` | — | Absolute path to rules YAML file |
| `MUX_WEBHOOK_ALLOW_INTERNAL` | `false` | Allow RFC-1918 webhook targets (dev only) |
| `MUX_GLOBAL_COOLDOWN_SECS` | `0` | Global per-sender cooldown in seconds |
| `MUX_WORKFLOWS_ENABLED` | `false` | Enable multi-step workflow engine |

## Ports

| Port | Purpose |
|------|---------|
| `3711` | Mux REST API, webhook receiver, and health endpoint |

## Database schema

The mux plugin adds the following tables to your Postgres database (prefix: `np_mux_`):

- `np_mux_accounts` — linked email/webhook accounts
- `np_mux_rules` — routing rules (YAML-compiled)
- `np_mux_rule_logs` — per-rule execution log
- `np_mux_dedup_log` — deduplication fingerprint store
- `np_mux_cooldowns` — per-sender cooldown state
- `np_mux_runs` — pipeline run audit records
- `np_mux_audit_queue` — async audit event queue
- `np_mux_gmail_dlq` — Gmail-specific dead-letter queue
- `np_mux_newsletter_articles` — extracted newsletter article store

All tables include a `tenant_id UUID` column with Hasura row-level security enforced via `X-Hasura-Tenant-Id`.

## REST API (key endpoints)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | None | Health probe |
| `POST` | `/rules` | Bearer | Create routing rule |
| `GET` | `/rules` | Bearer | List routing rules |
| `PATCH` | `/rules/{id}` | Bearer | Update rule |
| `DELETE` | `/rules/{id}` | Bearer | Delete rule |
| `GET` | `/runs` | Bearer | List pipeline runs |
| `POST` | `/webhooks/endpoints` | Bearer | Register webhook endpoint |
| `GET` | `/webhooks/deliveries` | Bearer | View delivery history |
| `POST` | `/classify` | Bearer | AI-classify a message |
| `GET` | `/dlq` | Bearer | View dead-letter queue |

Full OpenAPI spec: `plugins-pro/paid/mux/openapi.yaml`

## Nginx routes

| Route | Target |
|-------|--------|
| `/mux/` | Mux REST API |
| `/mux/webhooks/` | Webhook receiver |

## Examples

**Install and verify:**
```bash
nself plugin install mux
nself start
curl http://localhost:3711/health
# {"status":"ok","version":"1.1.3"}
```

**Check SSRF guard (should return 403):**
```bash
curl -X POST http://localhost:3711/webhooks/endpoints \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"url":"http://169.254.169.254/latest/meta-data/"}'
# Error: webhook URL blocked by SSRF guard
```

**View DLQ:**
```bash
curl http://localhost:3711/dlq \
  -H "Authorization: Bearer $TOKEN"
```

## Source code

Source available at `plugins-pro/paid/mux/` in the private `nself-org/plugins-pro` repository. Access requires a valid nSelf license. See [[plugin-mux-contributing]] for development setup.

## See also

- [[plugin-ai]] — AI classification engine used by mux
- [[plugin-notify]] — Notification dispatch used by mux
- [[plugin-google]] — Gmail ingestion for mux pipeline
- [[plugin-cron]] — Schedule-triggered mux rule evaluation
- [[bundle-nclaw]] — ɳClaw bundle (includes mux)

← [[Plugins]] | [[Home]] →
