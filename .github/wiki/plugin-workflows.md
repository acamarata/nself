# Workflows Plugin

> Trigger-action workflow automation with conditional logic, event-driven flows, scheduling, and cross-plugin orchestration. **Pro plugin.**

> **Requires:** Pro license. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install workflows
```

## What It Does

The workflows plugin is a full automation engine built into your ɳSelf stack. You define workflows as trigger-condition-action chains stored in Postgres. Workflows run on demand, on schedule, or in response to events from other plugins without external automation services.

## Trigger Types

| Type | Description |
|------|-------------|
| `webhook` | Fires when an inbound HTTP POST hits `/workflows/trigger/{workflow_id}` |
| `schedule` | Fires on a cron expression (requires the cron plugin) |
| `event` | Fires when an nSelf internal event is published, e.g. `user.created`, `plugin.started` |
| `manual` | Fires only via an explicit API call to `/workflows/{id}/execute` |

## Action Types

| Type | Description |
|------|-------------|
| `http_call` | Outbound HTTP request to any URL with configurable method, headers, and body |
| `db_update` | Write or upsert a row in a Postgres table via parameterized SQL |
| `plugin_call` | Invoke another nSelf plugin's REST endpoint |
| `wait` | Pause execution for N seconds before the next step |
| `emit_event` | Publish an internal nSelf event for other workflows to react to |

## Condition Syntax

Conditions are JSON objects evaluated at runtime against the trigger payload.

Single condition:
```json
{
  "field": "body.status",
  "op": "eq",
  "value": "active"
}
```

Supported operators: `eq`, `neq`, `gt`, `lt`, `contains`, `exists`.

Combine with `all` (AND) or `any` (OR):
```json
{
  "any": [
    {"field": "body.role", "op": "eq", "value": "admin"},
    {"field": "body.role", "op": "eq", "value": "owner"}
  ]
}
```

Steps accept an `if` field with a condition object. Steps that fail their condition are skipped, not failed.

## Scheduling (cron plugin integration)

Schedule triggers use the cron plugin to fire workflows on a time-based cadence. The trigger config accepts a standard cron expression:

```json
{
  "trigger_type": "schedule",
  "config": {
    "cron": "0 9 * * 1-5",
    "timezone": "UTC"
  }
}
```

Install the cron plugin alongside workflows if you use schedule triggers:

```bash
nself plugin install cron workflows
```

## Secret Injection

Workflow variables with `is_secret: true` are stored encrypted. At execution time the runtime decrypts and injects them into action configs. Secrets never appear in list or get responses.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PORT` | `3733` | Workflows service port |
| `DATABASE_URL` | postgres://... | Postgres connection URL |
| `WORKFLOWS_MAX_CONCURRENT_EXECUTIONS` | `10` | Maximum simultaneously executing workflow runs |

## Ports

| Port | Purpose |
|------|---------|
| 3733 | Workflows REST API |

## Database Tables

9 tables added to your Postgres database (all prefixed `np_workflows_`):

| Table | Purpose |
|-------|---------|
| `np_workflows_workflows` | Workflow definitions (draft / active / archived) |
| `np_workflows_executions` | Workflow execution runs |
| `np_workflows_execution_steps` | Per-step execution records |
| `np_workflows_triggers` | Trigger configurations per workflow |
| `np_workflows_actions` | Action step definitions |
| `np_workflows_templates` | Shareable workflow templates |
| `np_workflows_variables` | Workflow variables and secrets |
| `np_workflows_webhook_logs` | Inbound webhook call log |
| `np_workflows_approvals` | Pending and resolved approval gates |

All tables include `source_account_id` for multi-app isolation.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/workflows/` | Workflows REST API |

## Webhooks Emitted

| Event | When |
|-------|------|
| `workflow.created` | A workflow was created |
| `workflow.published` | A workflow was published (status set to active) |
| `workflow.archived` | A workflow was archived |
| `execution.started` | A workflow execution started |
| `execution.completed` | Execution completed successfully |
| `execution.failed` | Execution failed |
| `execution.timeout` | Execution timed out |
| `approval.required` | An approval gate is waiting for response |
| `approval.responded` | An approval was resolved |
| `trigger.fired` | A trigger fired |

## Docker

```bash
docker pull acamarata/plugin-workflows:latest
```

---

[[Plugin-Overview]] | [[Home]]
