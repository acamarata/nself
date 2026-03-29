> **Planned Feature:** This plugin is not yet available. It is planned for a future release.
> Current available plugins: [Plugins Overview](./Plugin-Overview.md)

# Subscription Plugin

> Subscription billing management — plans, trials, upgrades, and cancellations. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install subscription
```

## What It Does

Manages subscription lifecycle on top of your payment processor. Define subscription plans, handle free trials, upgrades, downgrades, pauses, and cancellations. Tracks subscription state independently of the payment processor for fast access. Works with the `stripe` plugin for payment processing.

## Dependencies

Pairs with the `stripe` plugin for payment processing.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SUBSCRIPTION_PORT` | `3043` | Subscription service port |
| `SUBSCRIPTION_TRIAL_DAYS` | `14` | Default free trial length |
| `SUBSCRIPTION_GRACE_PERIOD_DAYS` | `3` | Grace period after payment failure |
| `SUBSCRIPTION_DUNNING_ENABLED` | `true` | Retry failed payments |

## Ports

| Port | Purpose |
|------|---------|
| 3043 | Subscription management REST API |

## Database Tables

5 tables added to your Postgres database:
- `np_subscription_plans` — plan definitions
- `np_subscription_subscriptions` — active subscriptions
- `np_subscription_trials` — trial records
- `np_subscription_events` — lifecycle events (upgraded, cancelled, etc.)
- `np_subscription_dunning` — payment retry schedule

## Nginx Routes

| Route | Target |
|-------|--------|
| `/subscription/` | Subscription management API |
