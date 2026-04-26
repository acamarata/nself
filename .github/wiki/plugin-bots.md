# Bots Plugin

> Bot framework for chat, commands, subscriptions, and bot marketplace. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install bots
```

## What It Does

A framework for building and deploying chat bots that integrate with the `chat` plugin. Define bot commands (slash commands), event subscriptions (react to messages, joins, reactions), and publish bots to a shared marketplace. Supports webhook-triggered bots and native Go bot handlers.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `BOTS_PORT` | `3103` | Bots service port |
| `BOTS_WEBHOOK_TIMEOUT` | `5s` | Webhook delivery timeout |
| `BOTS_MARKETPLACE_ENABLED` | `true` | Enable bot marketplace |

## Ports

| Port | Purpose |
|------|---------|
| 3103 | Bots REST API |

## Database Tables

9 tables added to your Postgres database:
- `np_bots_bots`, registered bot definitions
- `np_bots_commands`, slash command registrations
- `np_bots_subscriptions`, event subscriptions
- `np_bots_webhooks`, webhook delivery configs
- `np_bots_marketplace`, marketplace listings
- `np_bots_installs`, bot install records
- `np_bots_tokens`, bot authentication tokens
- `np_bots_events`, event delivery log
- `np_bots_responses`, bot response cache

## Nginx Routes

| Route | Target |
|-------|--------|
| `/bots/` | Bot management API |
| `/bots/webhook/{bot_id}` | Bot webhook receiver |
