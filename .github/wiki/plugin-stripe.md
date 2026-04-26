# Stripe Plugin

> Full Stripe billing integration, subscriptions, webhooks, and customer portal. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install stripe
```

## What It Does

Syncs your complete Stripe account state into Postgres in real time: customers, subscriptions, invoices, payment intents, products, prices, refunds, and disputes. Handles Stripe webhooks to keep data current. Exposes a customer portal API for self-service subscription management. With 24 synced tables, you can query all billing data directly via Hasura GraphQL without hitting the Stripe API.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `STRIPE_PORT` | `3070` | Stripe plugin port |
| `STRIPE_SECRET_KEY` | — | Stripe secret key (sk_live_... or sk_test_...) |
| `STRIPE_WEBHOOK_SECRET` | — | Webhook endpoint signing secret |
| `STRIPE_PORTAL_RETURN_URL` | — | Customer portal return URL |

## Ports

| Port | Purpose |
|------|---------|
| 3070 | Stripe integration REST API |

## Database Tables

24 tables added to your Postgres database, including:
- `np_stripe_customers`, customer records
- `np_stripe_subscriptions`, subscription state
- `np_stripe_invoices`, invoice history
- `np_stripe_payment_intents`, payment records
- `np_stripe_products`, product catalog
- `np_stripe_prices`, pricing tiers
- `np_stripe_refunds`, refund records
- `np_stripe_disputes`, dispute/chargeback records
- And 16 more for coupons, trials, tax rates, etc.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/stripe/webhook` | Stripe event webhook receiver |
| `/stripe/portal` | Customer portal session API |

## API

```
POST /stripe/webhook              — Stripe webhook receiver
POST /portal/session              — Create customer portal session
POST /checkout/session            — Create checkout session
GET  /subscriptions/{customer_id} — Get customer subscription state
```
