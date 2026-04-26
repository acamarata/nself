# Workflows Plugin

> Trigger-action workflow automation with event-driven flows and scheduling. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install workflows
```

## What It Does

Enables trigger-action workflow automation without writing code. Define conditional logic, connect services with event-driven flows, and schedule automated tasks across your ɳSelf stack. Supports branching conditions, parallel steps, and secret injection for third-party service credentials.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `WORKFLOWS_PORT` | `3712` | Workflows service port |
| `WORKFLOWS_MAX_CONCURRENT` | `10` | Maximum simultaneously executing workflow runs |

## Ports

| Port | Purpose |
|------|---------|
| 3712 | Workflows REST API |

## Database Tables

9 tables added to your Postgres database:
- `np_workflows_definitions`, workflow definitions
- `np_workflows_triggers`, trigger configurations
- `np_workflows_actions`, action step definitions
- `np_workflows_conditions`, conditional logic rules
- `np_workflows_runs`, workflow execution runs
- `np_workflows_steps`, individual step execution records
- `np_workflows_logs`, execution log entries
- `np_workflows_schedules`, cron and time-based schedules
- `np_workflows_secrets`, encrypted secrets for workflow actions

## Nginx Routes

| Route | Target |
|-------|--------|
| `/workflows/` | Workflows API |
