# Mux Plugin

> Email and webhook pipeline engine with YAML rule routing, AI classification, auto-reply, and a dead-letter queue. **Pro plugin (ɳClaw bundle).**

> **Requires:** Pro license entitlement. `nself license set nself_pro_...`

## Overview

`plugin-mux` (service `plugin-mux`, port **3711**) is the routing and automation core of the ɳClaw bundle. It ingests email (Gmail/IMAP/Outlook) and webhook events (GitHub, Slack, Telegram, generic), evaluates them against YAML-defined rules, and dispatches actions through 9 handler groups:

- **PatternNotify**: match a sender/subject pattern, fire a notification.
- **InboxRouter**: route a message to a labeled bucket or downstream handler.
- **SilentTrash**: drop noise without notifying.
- **ContentForward**: forward the message body to a webhook or another address (SSRF-guarded).
- **ObserveNotify**: observe-only notifications for audit.
- **CalendarSync**: sync calendar invites via the `google` plugin.
- **AutoReply**: generate and (optionally) send a reply, with a human-approval queue.
- **SheetLogger**: append rows to a Google Sheet.
- **DonorHandler**: special-case donor/contact handling.

Classification and reply drafting use the `ai` plugin; notifications use the `notify` plugin; Google/Gmail integration uses the `google` plugin. All outbound HTTP (ContentForward, webhook dispatch, DLQ retry) passes through the shared SSRF guard, which blocks RFC-1918, loopback, link-local, and cloud metadata addresses.

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install mux
nself restart
```

The license gate (`requires_license: true`, entitlement `pro`) is enforced by the `ping_api` middleware on every `/mux/*` route. Unauthenticated or unentitled calls receive `403`.

## Dependencies

| Dependency | Required | Purpose |
|------------|----------|---------|
| `notify` plugin | Yes | Outbound notifications |
| `google` plugin | Yes | Gmail ingestion, Calendar/Sheets actions |
| `ai` plugin | Yes | Classification, rule suggestion, reply drafting |
| PostgreSQL | Yes | All state (`np_mux_*` tables) |
| Redis | Optional | Dedup/cooldown acceleration |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `DATABASE_URL` | — (required) | Postgres connection string |
| `MUX_PORT` | `3711` | Mux service port |
| `MUX_HOST` | `0.0.0.0` | Bind address |
| `MUX_RULES_PATH` | — | Path to the YAML rules file |
| `MUX_AI_ENABLED` | `true` | Use the `ai` plugin for classification |
| `MUX_PUBSUB_SUBSCRIPTION` | — | Google Pub/Sub subscription for Gmail push |
| `MUX_DLQ_MAX_RETRIES` | `5` | Max retries before a message is parked in the DLQ |
| `MUX_DLQ_RETRY_WEBHOOK_URL` | — | Webhook notified on DLQ retry (SSRF-guarded) |
| `PLUGIN_MUX_SHADOW_MODE` | `false` | Evaluate rules but suppress side effects (dry run) |
| `PLUGIN_MUX_RULES_YAML` | — | Inline YAML rules (alternative to `MUX_RULES_PATH`) |
| `PLUGIN_MUX_AI_TOKEN` | — | Override token for the `ai` plugin call |
| `PLUGIN_VOICE_INTERNAL_URL` | — | ɳVoice internal URL for TTS notifications |
| `PLUGIN_VOICE_INTERNAL_SECRET` | — | ɳVoice internal auth secret |
| `PLUGIN_VOICE_TTS_PROVIDER` | — | TTS provider selector for voice notifications |

## Ports

| Port | Purpose |
|------|---------|
| 3711 | Mux REST API, webhook receiver, and health/metrics |

## Endpoints

All routes are mounted under `/mux` and gated by the `ping_api` license middleware (except `/health`, `/ready`, and `/metrics`).

### Health & diagnostics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` · `/mux/health` | Liveness probe |
| GET | `/ready` · `/mux/ready` | Readiness probe (DB reachable) |
| GET | `/mux/health/migrations` | Migration status |
| GET | `/mux/health/push-status` | Gmail push subscription health |
| GET | `/mux/diagnostics` | Runtime diagnostics |
| GET | `/mux/diagnostics/unmatched` | Messages that matched no rule |
| GET | `/mux/metrics` · `/metrics` | Prometheus metrics |
| GET | `/internal/tools` · `/mux/ai/instructions` | AI tool/instruction manifest |
| GET | `/mux/config/schema` | JSON schema for the rule config |

### Ingest

| Method | Path | Description |
|--------|------|-------------|
| POST | `/mux/ingest` | Generic message ingest |
| POST | `/mux/ingest/webhook` | Generic webhook ingest |
| POST | `/mux/trigger` | Token-auth generic trigger |
| POST | `/mux/ingest/github` | GitHub webhook (signature-verified) |
| POST | `/mux/ingest/slack` | Slack events (signature-verified) |
| POST | `/mux/ingest/telegram` | Telegram webhook |
| GET | `/mux/ingest/status` | Ingest pipeline status |
| GET | `/mux/ingest/rss/feeds` | Configured RSS feeds |
| POST | `/mux/gmail/push` · `/mux/outlook/push` | Provider push notifications |
| POST | `/mux/push` | Generic push receiver |
| POST | `/mux/tokens/import` | Import Gmail OAuth tokens |

### Accounts, rules & runs

| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/mux/accounts` | List / create accounts |
| PATCH/DELETE | `/mux/accounts/{id}` | Update / delete an account |
| GET/POST | `/mux/rules` | List / create rules |
| GET/PATCH/DELETE | `/mux/rules/{id}` | Get / update / delete a rule |
| GET | `/mux/rules/export` | Export rules as YAML |
| POST | `/mux/rules/import` | Import rules from YAML |
| GET | `/mux/rules/{id}/test-history` | Past test results for a rule |
| GET/POST | `/mux/rules/suggest` | List / generate AI rule suggestions |
| POST | `/mux/rules/test` | Dry-run rules against samples |
| POST | `/mux/rules/{id}/compile` | Compile a suggestion into a live rule |
| POST | `/mux/reload` · `/mux/test` · `/mux/replay` | Reload config / test / replay |
| GET | `/mux/runs` | Rule execution history |

### Messages, digests & analytics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/mux/messages` · `/mux/messages/search` | Query / search ingested messages |
| POST | `/mux/messages/summarize-today` | AI summary of today's messages |
| POST | `/mux/messages/draft-reply` | Draft an AI reply |
| POST | `/mux/digest/generate` · `/mux/digests/{id}/send-now` | Generate / send digests |
| GET | `/mux/analytics/{volume,rules,senders,latency,actions}` | Analytics dashboards |
| GET | `/mux/classifications/categories` · `/mux/classifications/runs` | Classification data |
| POST | `/mux/classifications/classify` · `/mux/classifications/feedback` | Classify / give feedback |

### Auto-reply & DLQ

| Method | Path | Description |
|--------|------|-------------|
| POST | `/mux/pending-replies/{id}/approve` · `/discard` | Approve / discard a queued reply |
| GET | `/mux/auto-reply/audit` | Auto-reply audit log |
| GET | `/mux/dlq` | List dead-letter items |
| POST | `/mux/dlq/{id}/retry` · `/mux/dlq/clear-failed` | Retry / clear DLQ items |
| DELETE | `/mux/dlq/{id}` | Delete a DLQ item |

## Database Tables

Core tables declared in `plugin.json`:

- `np_mux_accounts`: connected email/webhook accounts
- `np_mux_rules`: routing rules
- `np_mux_rule_logs`: per-rule evaluation log
- `np_mux_dedup_log`: deduplication ledger
- `np_mux_cooldowns`: per-rule cooldown windows
- `np_mux_runs`: rule execution runs

Supporting tables created by migrations include `np_mux_messages`, `np_mux_dlq`, `np_mux_audit_queue`, `np_mux_gmail_dlq`, `np_mux_pending_replies`, `np_mux_digests`, `np_mux_classifications_feedback`, `np_mux_threads`, `np_mux_newsletter_articles`, and `np_mux_workflows`.

Multi-app isolation (`source_account_id TEXT NOT NULL DEFAULT 'primary'` with a Hasura row filter `{"source_account_id":{"_eq":"X-Hasura-Source-Account-Id"}}`) is applied per the Multi-Tenant Convention Wall on the tables that carry it (`np_mux_messages`, `np_mux_deferred_actions`, `np_mux_newsletter_articles`); remaining tables are scoped through their parent account or rule. Cloud-tenant tables (`np_mux_gmail_dlq`, `np_mux_audit_queue`) enforce `{"tenant_id":{"_eq":"X-Hasura-Tenant-Id"}}`.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/mux/ingest/*` | Inbound webhook/event receivers |
| `/mux/` | Mux management API |

## Quickstart

1. **Set a Pro license and install:**
   ```bash
   nself license set nself_pro_xxxxx...
   nself plugin install mux
   ```
2. **Configure a rules file** at `MUX_RULES_PATH` (or set `PLUGIN_MUX_RULES_YAML` inline):
   ```yaml
   rules:
     - name: urgent-support
       match:
         subject: "(?i)urgent|outage"
       action: PatternNotify
       notify:
         channel: ops
   ```
3. **Restart and verify health:**
   ```bash
   nself restart
   curl -fsS http://localhost:3711/health
   ```
4. **Reload rules without a restart** after editing:
   ```bash
   curl -X POST http://localhost:3711/mux/reload
   ```
5. **Send a test event** and confirm a run is recorded:
   ```bash
   curl -X POST http://localhost:3711/mux/trigger \
     -H "Authorization: Bearer $MUX_TOKEN" \
     -d '{"subject":"urgent outage","body":"db down"}'
   curl http://localhost:3711/mux/runs
   ```

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `403` on every `/mux/*` call | Missing/invalid Pro license | `nself license set nself_pro_...`, then `nself restart` |
| `/mux/ingest/webhook` returns `403` for an outbound forward | SSRF guard blocked an RFC-1918/loopback/metadata target | Point `ContentForward`/`MUX_DLQ_RETRY_WEBHOOK_URL` at a public host; the guard rejects `10.0.0.0/8`, `127.0.0.1`, `169.254.169.254` by design |
| Messages never match a rule | Rules file not loaded | Check `MUX_RULES_PATH`/`PLUGIN_MUX_RULES_YAML`; `GET /mux/diagnostics/unmatched`; `POST /mux/reload` |
| GitHub/Slack webhooks rejected | Signature verification failed | Confirm the webhook secret matches; check the `X-Hub-Signature-256` / `X-Slack-Signature` headers |
| Gmail ingest stops | Push subscription expired | `GET /mux/health/push-status`; re-import tokens via `POST /mux/tokens/import` |
| Messages stuck in the DLQ | Downstream handler failing | `GET /mux/dlq`; fix the handler; `POST /mux/dlq/{id}/retry` or `POST /mux/dlq/clear-failed` |
| Auto-replies not sending | Pending-approval queue | `GET /mux/auto-reply/audit`; approve via `POST /mux/pending-replies/{id}/approve` |
| Rules evaluate but nothing happens | Shadow mode on | Unset `PLUGIN_MUX_SHADOW_MODE` (dry-run flag) and restart |

## See Also

- ɳClaw bundle overview: `nself plugin list --bundle nclaw`
- Security-Always-Free doctrine (SSRF guard ships free)
- `ai`, `notify`, and `google` plugin docs for the action backends
