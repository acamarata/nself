# Plugin: claw-budget

Token budget tracking and cost controls for AI assistant sessions. Integrates with the [[plugin-ai]] and [[plugin-claw]] plugins to enforce per-user, per-session, and per-tenant spending limits on LLM API calls.

## Overview

The claw-budget plugin sits between your AI plugins and the LLM providers, intercepting every inference request to track token usage and enforce configurable budget policies. When a budget threshold is reached, the plugin can warn, throttle, or block further requests depending on your configuration.

## Features

- **Per-user budgets**, set monthly or daily token/cost limits per user
- **Per-session budgets**, cap individual conversation sessions to prevent runaway costs
- **Per-tenant budgets**, enforce organization-level spending ceilings in multi-tenant deployments
- **Real-time dashboard**, token usage, cost breakdown by model, and budget utilization via Hasura subscriptions
- **Configurable actions**, warn, throttle (rate limit), or hard-block when thresholds are hit
- **Model cost mapping**, built-in cost tables for OpenAI, Anthropic, Mistral, and Gemini models with custom overrides
- **Historical reporting**, query usage by user, tenant, session, model, or time range via GraphQL
- **Webhook alerts**, fire webhooks when budgets reach configurable percentage thresholds (50%, 80%, 100%)

## Installation

```bash
nself plugin install claw-budget
nself build && nself restart
```

Requires a valid Pro license key. See [[Plugin-Licensing]].

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PLUGIN_CLAW_BUDGET_PORT` | `3720` | Service port |
| `PLUGIN_CLAW_BUDGET_DEFAULT_MONTHLY_LIMIT` | `10.00` | Default monthly cost limit per user (USD) |
| `PLUGIN_CLAW_BUDGET_DEFAULT_SESSION_LIMIT` | `1.00` | Default per-session cost limit (USD) |
| `PLUGIN_CLAW_BUDGET_ACTION` | `warn` | Action at limit: `warn`, `throttle`, `block` |
| `PLUGIN_CLAW_BUDGET_WEBHOOK_URL` | — | URL to POST budget alerts |
| `PLUGIN_CLAW_BUDGET_ALERT_THRESHOLDS` | `50,80,100` | Comma-separated percentage thresholds for alerts |

## Database Schema

Tables are created under the `np_claw_budget` schema:

| Table | Description |
|-------|-------------|
| `budgets` | Budget definitions (user, tenant, or global scope) |
| `usage_records` | Per-request token counts and computed costs |
| `alerts` | Alert history with delivery status |
| `model_costs` | Cost-per-token overrides by model name |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/claw-budget/usage` | Current usage summary for authenticated user |
| `GET` | `/claw-budget/usage/:userId` | Usage for specific user (admin only) |
| `GET` | `/claw-budget/budgets` | List all budget definitions |
| `POST` | `/claw-budget/budgets` | Create or update a budget |
| `DELETE` | `/claw-budget/budgets/:id` | Remove a budget |
| `GET` | `/claw-budget/reports` | Historical usage report with filters |

All endpoints are also available via Hasura GraphQL subscriptions for real-time updates.

## Integration with AI Plugins

claw-budget automatically hooks into [[plugin-ai]] and [[plugin-claw]] when all three are installed. No additional configuration is needed. The budget enforcement middleware intercepts requests at the AI gateway level, before they reach the LLM provider.

If claw-budget is installed without plugin-ai, it operates in tracking-only mode: it records usage via the GraphQL API but does not intercept or block requests.

## See Also

- [[plugin-ai]], LLM gateway with provider rotation
- [[plugin-claw]], autonomous AI assistant engine
- [[Plugin-Licensing]], license tiers and pricing
- [[Plugin-Overview]], all plugins by category

---
← [[Plugin-Overview]] | [[_Sidebar]]
