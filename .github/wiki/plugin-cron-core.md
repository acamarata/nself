# Cron Core Plugin

> Dedicated cron scheduler for ɳClaw AI agent system. pg_cron driver with HMAC-signed webhook delivery, per-tenant RLS isolation, and distributed leader locking. **Included in ɳClaw bundle.**

> **Requires:** ɳClaw bundle ($0.99/month) or ɳSelf+ ($39.99/year). `nself plugin install cron` (core)

## Install

```bash
nself license set <claw_license_key>
nself plugin install cron
```

## What It Does

Cron Core is the system scheduler for ɳClaw AI agents and other bundle services. Unlike the free cron plugin (which uses BullMQ in Redis), Cron Core is backed by PostgreSQL pg_cron — a battle-tested distributed job scheduler that requires zero external dependencies beyond Postgres.

Features:
- **pg_cron driver**: leverage Postgres native scheduling, no external queue
- **HMAC-signed webhooks**: cryptographically authenticated job callbacks
- **Per-tenant isolation**: row-level security (RLS) via Hasura tenant_id column
- **Distributed locking**: leader election prevents duplicate runs across ɳClaw instances
- **Automatic retry**: exponential backoff with configurable max retries
- **Job history**: complete audit trail of all scheduled job executions
- **Health monitoring**: webhook health checks and failure alerts

## Architecture

Cron Core runs as a service (port 9005) and uses Postgres `pg_cron` extension to schedule job execution. When a job triggers, the service fires an HMAC-signed HTTP POST to a configured webhook endpoint. All job metadata, history, and state are stored in `np_cron_*` tables with full tenant isolation.

## Install & Configure

After `nself license set`, install the plugin:

```bash
nself plugin install cron
nself build
nself start
```

Cron Core enables automatically. If you need to explicitly configure it, set these env vars:

| Env Var | Default | Description |
|---------|---------|-------------|
| `CRON_PORT` | `9005` | Cron Core service port |
| `CRON_HOST` | `0.0.0.0` | Service listen address |
| `CRON_WEBHOOK_SIGNING_SECRET` | *(auto-generated)* | HMAC secret for webhook signatures |
| `CRON_INTERNAL_SECRET` | *(auto-generated)* | Internal service auth token |
| `CRON_MAX_RETRIES` | `3` | Max automatic retries on failure |
| `CRON_TIMEOUT_SECS` | `30` | HTTP callback timeout |
| `CRON_RETENTION_DAYS` | `30` | Days to retain job history |
| `CRON_DEFAULT_QUOTA_PER_ACCOUNT` | `100` | Max concurrent jobs per tenant |
| `CRON_POLL_INTERVAL_SECS` | `5` | Poll interval for job execution |
| `CRON_LEADER_ENABLED` | `true` | Enable distributed leader locking |
| `CRON_INSTANCE_ID` | *(auto-generated)* | Unique instance ID for leader election |

## Database Tables

Cron Core adds five tables to your Postgres database:

| Table | Purpose |
|-------|---------|
| `np_cron_jobs` | Job definitions and schedules |
| `np_cron_history` | Execution history and results |
| `np_cron_account_quota` | Per-tenant job quotas and usage |
| `np_cron_leader` | Distributed leader election state |
| `np_cron_web_snapshots` | Internal state for web UI |

All tables include `tenant_id` (UUID) for multi-tenant row-level security via Hasura. Hasura applies row filters: `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}`.

## Configuration Examples

### Enable Cron Core for ɳClaw

ɳClaw agents automatically create cron jobs for:
- Periodic model retraining
- Memory compaction
- Health checks and diagnostics
- Chat message archival
- Training data cleanup

No manual configuration needed. Jobs are auto-created on agent startup.

### Manual Job Creation via API

Create a job via REST API (internal endpoint, not exposed to nginx):

```bash
curl -X POST http://localhost:9005/cron/jobs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $CRON_INTERNAL_SECRET" \
  -d '{
    "name": "custom-cleanup",
    "schedule": "0 2 * * *",
    "webhook_url": "http://my-service:8080/tasks/cleanup",
    "payload": {"type": "full"},
    "timezone": "UTC"
  }'
```

### Inspect Job Status

```bash
# List all jobs
curl http://localhost:9005/cron/jobs \
  -H "Authorization: Bearer $CRON_INTERNAL_SECRET"

# Check job history
curl http://localhost:9005/cron/history \
  -H "Authorization: Bearer $CRON_INTERNAL_SECRET"

# Health check
curl http://localhost:9005/health
```

## Multi-Instance / Distributed Locking

When running multiple ɳClaw instances (for HA), Cron Core uses Postgres-based distributed locks to ensure jobs run only once, even if multiple instances have the same job scheduled.

The leader election process:
1. Each instance inserts/updates its row in `np_cron_leader` with a heartbeat timestamp
2. The instance with the most recent heartbeat becomes the leader
3. Only the leader polls pg_cron and fires webhooks
4. Failover is automatic (non-leader detects stale heartbeat, takes over)

This requires zero external coordination service (no Redis, no Etcd). All state lives in Postgres.

## Webhook HMAC Signing

Every webhook callback is signed with HMAC-SHA256 using `CRON_WEBHOOK_SIGNING_SECRET`. Webhook handlers should verify the signature before processing:

```javascript
// Example: verify signature in Node.js
const crypto = require('crypto');

app.post('/tasks/my-callback', (req, res) => {
  const signature = req.headers['x-cron-signature'];
  const body = JSON.stringify(req.body);
  const secret = process.env.CRON_WEBHOOK_SIGNING_SECRET;
  
  const hash = crypto.createHmac('sha256', secret).update(body).digest('hex');
  if (hash !== signature) {
    return res.status(401).send('Unauthorized');
  }
  
  // Process job
  res.status(200).send('OK');
});
```

## Troubleshooting

### Job Not Triggering

1. Check Postgres connection: `nself doctor --deep` displays Postgres health.
2. Verify leader is active: query `np_cron_leader` for a recent heartbeat timestamp.
3. Check webhook endpoint accessibility: Cron Core logs all webhook attempts.
4. Review `np_cron_history` for failure details.

### Quota Exceeded

If jobs are rejected with quota error, check or increase `CRON_DEFAULT_QUOTA_PER_ACCOUNT`.

### Memory Usage

Cron Core is lightweight (~64 MB minimum). If memory grows unbounded, check `CRON_RETENTION_DAYS` and consider reducing history retention.

## Ports

| Port | Purpose |
|------|---------|
| 9005 | Cron Core REST API (internal only, not exposed on nginx) |

## Related

- [[plugin-cron]], free MIT-licensed cron (BullMQ-backed)
- [[plugin-cron-pro]], distributed locks + complex syntax + dashboard (planned)
- [[bundle-claw]], ɳClaw bundle containing Cron Core

[[Home]]
