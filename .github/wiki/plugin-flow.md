# Flow Plugin

> Visual workflow automation — connect services with trigger-action flows. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install flow
```

## What It Does

Adds visual workflow automation to your nSelf stack. Define trigger-action flows that connect your services without writing code: when a new user signs up, send a welcome email and create a CRM record; when a payment succeeds, provision access and notify the team. Supports conditional logic, loops, and scheduled execution. Compatible with n8n-style workflow definitions.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `FLOW_PORT` | `3712` | Flow service port |
| `FLOW_MAX_CONCURRENT` | `10` | Max concurrent workflow executions |
| `FLOW_EXECUTION_TIMEOUT` | `300s` | Max execution time per workflow |
| `FLOW_HISTORY_RETENTION` | `30` | Days to retain execution history |

## Ports

| Port | Purpose |
|------|---------|
| 3712 | Flow automation REST API |

## Database Tables

9 tables added to your Postgres database:
- `np_flow_workflows` — workflow definitions
- `np_flow_triggers` — trigger configurations
- `np_flow_actions` — action step definitions
- `np_flow_conditions` — conditional logic rules
- `np_flow_executions` — execution history
- `np_flow_execution_steps` — per-step execution log
- `np_flow_variables` — workflow variables
- `np_flow_schedules` — scheduled trigger configs
- `np_flow_integrations` — external service connections

## Nginx Routes

| Route | Target |
|-------|--------|
| `/flow/` | Flow automation API and UI |
