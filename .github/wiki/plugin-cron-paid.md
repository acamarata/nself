# Plugin: Cron (Paid)

> Advanced cron scheduler for the ɳClaw bundle. Distributed job execution, multi-tenant support, tenant-level quota enforcement, failure alerts, and full run history. **Pro plugin (ɳClaw bundle).**

> **Requires:** Pro license tier or ɳClaw bundle. `nself license set nself_pro_...` or `nself license set nself_claw_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install cron
```

## What It Does

Extends the free `cron` plugin with production-grade scheduling for multi-tenant nSelf Cloud deployments. Features include distributed execution with atomic job locking to prevent duplicate runs across multiple instances, full job run history with timestamps and status tracking, per-tenant quota enforcement (max jobs, max run frequency), distributed leader election for cluster coordination, and zero-downtime job record snapshots for web dashboards.

The paid variant replaces the free cron plugin when installed and uses tenant-level row-level security to ensure paying customers can only see and modify their own scheduled jobs. All quota and usage tracking is automatically enforced at the database layer via Hasura permissions.

## Features

- **Distributed job execution**, atomic locks and leader election prevent duplicate runs
- **Multi-tenant isolation**, tenant_id row filters ensure data privacy
- **Per-tenant quotas**, configurable job count and execution rate limits
- **Full execution history**, every run recorded with status, duration, and output
- **Failure alerts**, configurable alerting and automatic retry logic
- **Zero-downtime snapshots**, web dashboard data synchronization
- **Cluster coordination**, distributed leader election via database consensus

## Installation

```bash
# Set your Pro license
nself license set nself_pro_xxxxx...

# Install the paid cron plugin
nself plugin install cron

# (Re)build and restart the stack
nself build && nself restart
```

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CRON_PORT` | `3713` | Cron service port (paid variant) |
| `CRON_LOCK_TTL_SECONDS` | `300` | Distributed lock time-to-live (seconds) |
| `CRON_LEADER_HEARTBEAT_INTERVAL_SECONDS` | `10` | Leader election heartbeat interval |
| `CRON_HISTORY_RETENTION_DAYS` | `90` | Number of days to retain run history |
| `CRON_MAX_JOB_PAYLOAD_BYTES` | `1048576` | Maximum webhook payload size (bytes) |

## Ports

| Port | Purpose |
|------|---------|
| 3713 | Cron service REST API (paid variant) |

## Database Tables

6 tables added to your Postgres database:

| Table | Purpose |
|-------|---------|
| `np_cron_jobs` | Job definitions with schedule, webhook URL, and metadata (tenant-scoped) |
| `np_cron_history` | Execution history with status, timestamps, and output (tenant-scoped) |
| `np_cron_account_quota` | Per-account quota limits (job count, execution rate) |
| `np_cron_leader` | Distributed leader election state (cluster coordination) |
| `np_cron_account_usage` | Current usage metrics per account (read-only view for admins) |
| `np_cron_web_snapshots` | Snapshot data for web dashboard/UI synchronization |

All tenant-visible tables (`np_cron_jobs`, `np_cron_history`) enforce row-level security via Hasura tenant row filters.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/cron/` | Cron service REST API |

## Multi-Tenant Security

This plugin implements the nSelf Security-Always-Free Doctrine. Row-level security is automatic and required in all deployments:

- **Tenant role**: SELECT/INSERT/UPDATE/DELETE only on rows where `tenant_id` matches the authenticated user's tenant (enforced by Hasura row filter on `X-Hasura-Tenant-Id` header).
- **Admin role**: Full read access to all tables including quota and usage data (for operations auditing).
- **Insert enforcement**: When a tenant creates a job, the `tenant_id` is automatically set from the session variable and cannot be spoofed.

Attempting to query another tenant's jobs returns an empty result set. Attempting to update another tenant's job is rejected by the database layer.

## API Endpoints

Via Hasura GraphQL (use the `graphql-client` package or standard GraphQL client):

```
# Create a job
mutation CreateJob($name: String!, $schedule: String!, $webhook_url: String!) {
  insert_np_cron_jobs_one(object: {
    name: $name,
    schedule: $schedule,
    webhook_url: $webhook_url
  }) {
    id
  }
}

# List your jobs (tenant-filtered)
query ListJobs {
  np_cron_jobs(order_by: {created_at: desc}) {
    id
    name
    schedule
    enabled
    next_run_at
    last_run_at
    run_count
  }
}

# View run history
query JobHistory($jobId: uuid!) {
  np_cron_history(where: {job_id: {_eq: $jobId}}, order_by: {run_at: desc}, limit: 50) {
    id
    status
    run_at
    duration_ms
    output
    error_message
  }
}
```

REST API endpoints are also exposed at `/cron/*` for direct HTTP access.

## Quota Enforcement

Each account has quota limits enforced at the database layer:

- `CRON_QUOTA_MAX_JOBS`: Maximum number of scheduled jobs (default: 100)
- `CRON_QUOTA_MAX_FREQUENCY`: Minimum seconds between consecutive executions of the same job (default: 1s, i.e., any frequency allowed)

Exceeding a quota results in an error when attempting to create or enable a job.

## See Also

- [[plugin-ai]], LLM gateway with provider rotation
- [[plugin-claw]], autonomous AI assistant engine
- [[plugin-cron]], free cron scheduler (limited features)
- [[Plugin-Licensing]], license tiers and pricing
- [[Plugin-Overview]], all plugins by category
- [[Security]], multi-tenant data isolation and row-level security

---

← [[Plugin-Overview]] | [[_Sidebar]]
