# PayPal Plugin

> PayPal payment sync with webhooks and order management. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install paypal
```

## What It Does

Syncs PayPal payment data into your Postgres database via the PayPal REST API and webhooks. Tracks orders, captures, refunds, disputes, and subscriptions. Provides a webhook receiver for real-time payment event processing. With 14 synced tables, query PayPal data directly via Hasura GraphQL.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PAYPAL_PORT` | `3071` | PayPal plugin port |
| `PAYPAL_CLIENT_ID` | — | PayPal REST API client ID |
| `PAYPAL_CLIENT_SECRET` | — | PayPal REST API client secret |
| `PAYPAL_MODE` | `sandbox` | Mode: `sandbox` or `live` |
| `PAYPAL_WEBHOOK_ID` | — | PayPal webhook ID for validation |

## Ports

| Port | Purpose |
|------|---------|
| 3071 | PayPal integration REST API |

## Database Tables

14 tables added to your Postgres database, including:
- `np_paypal_orders`, order records
- `np_paypal_captures`, payment captures
- `np_paypal_refunds`, refund records
- `np_paypal_disputes`, dispute/chargeback records
- `np_paypal_subscriptions`, subscription state
- `np_paypal_plans`, subscription plan definitions
- `np_paypal_webhook_events`, raw event log
- And 7 more for customers, invoices, payouts, etc.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/paypal/webhook` | PayPal webhook receiver |
