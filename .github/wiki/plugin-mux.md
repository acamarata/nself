# plugin-mux — Email & Webhook Pipeline

> **Bundle:** ɳClaw | **Tier:** Pro | **Port:** 3711 | **License required:** yes

The mux plugin is an email and webhook pipeline engine for nSelf. It applies a YAML rule file
to incoming Gmail messages and dispatches them through 9 handler groups: PatternNotify,
InboxRouter, SilentTrash, ContentForward, ObserveNotify, CalendarSync, AutoReply,
SheetLogger, and DonorHandler. All outbound HTTP calls are protected by an SSRF guard that
blocks RFC-1918, loopback, and link-local addresses.

---

## Install

```bash
# Set your license key first (ɳClaw bundle or ɳSelf+)
nself license set nself_pro_xxxxx...

# Install the plugin
nself plugin install mux
```

---

## Dependencies

| Plugin | Purpose |
|--------|---------|
| `notify` | Push notifications for matched rules |
| `google` | Gmail push subscription + OAuth token refresh |
| `ai` | AI classification (optional, enabled with `MUX_AI_ENABLED=true`) |

---

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | none | Liveness probe |
| `POST` | `/gmail/push` | bearer | Gmail Pub/Sub push receiver |
| `POST` | `/gmail/import` | bearer | Bulk import existing messages |
| `GET` | `/rules` | bearer | List configured rules |
| `POST` | `/rules` | bearer | Create or replace rule set |
| `GET` | `/rules/export` | bearer | Export rules as YAML |
| `POST` | `/rules/import` | bearer | Import rules from YAML |
| `GET` | `/runs` | bearer | List recent pipeline runs |
| `GET` | `/runs/{id}` | bearer | Single run detail |
| `GET` | `/accounts` | bearer | List registered Gmail accounts |
| `POST` | `/accounts` | bearer | Register a Gmail account |
| `DELETE` | `/accounts/{id}` | bearer | Remove an account |
| `POST` | `/webhooks/endpoints` | bearer | Register a webhook endpoint |
| `GET` | `/webhooks/endpoints` | bearer | List webhook endpoints |
| `DELETE` | `/webhooks/endpoints/{id}` | bearer | Remove a webhook endpoint |
| `GET` | `/workflows` | bearer | List automation workflows |
| `POST` | `/workflows` | bearer | Create a workflow |
| `PATCH` | `/workflows/{id}` | bearer | Update a workflow |
| `DELETE` | `/workflows/{id}` | bearer | Remove a workflow |
| `GET` | `/workflow-instances` | bearer | List workflow instances |
| `POST` | `/workflow-instances/{id}/advance` | bearer | Advance an instance |
| `POST` | `/workflow-instances/{id}/cancel` | bearer | Cancel an instance |

---

## Database Tables

Six tables are created in your Postgres database, all prefixed `np_mux_`:

| Table | Purpose |
|-------|---------|
| `np_mux_accounts` | Registered Gmail accounts (OAuth state) |
| `np_mux_rules` | Pipeline rule definitions |
| `np_mux_rule_logs` | Per-rule execution log |
| `np_mux_dedup_log` | Message deduplication records |
| `np_mux_cooldowns` | Per-sender cooldown state |
| `np_mux_runs` | Pipeline run history and DLQ entries |

All tables carry a `tenant_id UUID` column with Hasura row-filter
`{"tenant_id":{"_eq":"X-Hasura-Tenant-Id"}}` for multi-tenant isolation.

---

## Environment Variables

### Required

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | Postgres connection string |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `MUX_PORT` | `3711` | HTTP listen port |
| `MUX_HOST` | `0.0.0.0` | HTTP listen host |
| `MUX_RULES_PATH` | `rules.yaml` | Path to YAML rule file |
| `MUX_AI_ENABLED` | `false` | Enable AI classification via `ai` plugin |
| `PLUGIN_MUX_AI_TOKEN` | — | Bearer token for `ai` plugin calls |
| `MUX_PUBSUB_SUBSCRIPTION` | — | GCP Pub/Sub subscription name |
| `PLUGIN_MUX_SHADOW_MODE` | `false` | Log-only mode, no side effects |
| `PLUGIN_MUX_RULES_YAML` | — | Inline YAML rules (overrides file) |
| `MUX_DLQ_MAX_RETRIES` | `3` | Max DLQ retry attempts |
| `MUX_DLQ_RETRY_WEBHOOK_URL` | — | Webhook for DLQ retry notifications |
| `MUX_WEBHOOK_ALLOW_INTERNAL` | `false` | Allow RFC-1918 webhook targets (dev only) |
| `PLUGIN_VOICE_INTERNAL_URL` | — | Internal URL of voice plugin |
| `PLUGIN_VOICE_TTS_PROVIDER` | — | TTS provider identifier |

---

## Quickstart

1. Register a Gmail account:
   ```bash
   curl -X POST http://localhost:3711/accounts \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"email":"you@gmail.com"}'
   ```

2. Upload a rule file:
   ```bash
   curl -X POST http://localhost:3711/rules/import \
     -H "Authorization: Bearer $TOKEN" \
     -F "file=@rules.yaml"
   ```

3. Point your Gmail Pub/Sub subscription push URL at `/gmail/push`.

4. Verify the health endpoint:
   ```bash
   curl http://localhost:3711/health
   # {"status":"ok"}
   ```

---

## SSRF Guard

All outbound HTTP calls (webhook delivery, `also_notify` side-channels, AI classify forwards)
run through `safety.ValidateWebhookURL`. Requests targeting RFC-1918 ranges
(`10.x`, `172.16-31.x`, `192.168.x`), loopback (`127.x`, `::1`), and link-local
(`169.254.x`, `fe80::`) are rejected.

Set `MUX_WEBHOOK_ALLOW_INTERNAL=true` to bypass in development (for example, Docker bridge routing).

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `webhook URL blocked` in logs | Target URL resolves to a private IP. Set `MUX_WEBHOOK_ALLOW_INTERNAL=true` for dev. |
| Rules not matching | Run `GET /runs/{id}` to see match trace. Check rule syntax in `rules.yaml`. |
| Gmail push not received | Verify Pub/Sub push URL is `https://<host>/gmail/push`. Check bearer token. |
| DLQ growing | Check `np_mux_runs` for failure reason. Raise `MUX_DLQ_MAX_RETRIES` or fix downstream. |
| `license required` on install | Run `nself license set <key>`, requires ɳClaw bundle or ɳSelf+. |

---

## See Also

- [[plugin-ai]] — AI classification backend
- [[plugin-google]] — Gmail OAuth and Calendar integration
- [[plugin-notify]] — Push notification delivery
- [[Home]]
