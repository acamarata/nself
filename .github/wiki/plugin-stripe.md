# Stripe Plugin (Pro)

> Deep Stripe billing sync: customers, subscriptions, invoices, webhooks, and customer portal. **Pro plugin.**

> **Requires:** Pro license or higher. `nself license set nself_pro_...`

## Two Stripe Plugins: stripe vs nself-stripe

nSelf has two Stripe-related plugins. They serve different purposes and are independently installable:

| Plugin | Port | Purpose |
|--------|------|---------|
| `stripe` (this page) | 3740 | Your app's Stripe billing data sync. Syncs customers, subscriptions, invoices, and 24 other tables from your Stripe account into Postgres. Use this for SaaS billing in your own app. |
| `nself-stripe` | 3830 | nSelf Cloud MAX infrastructure billing. Handles nSelf's own subscription management for multi-tenant cloud deployments. Use this only when operating nSelf Cloud. |

Install `stripe` for your app billing. Install `nself-stripe` only when you are running nSelf Cloud MAX.

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
| `STRIPE_PORT` | `3740` | Stripe plugin port |
| `STRIPE_SECRET_KEY` | — | Stripe secret key (`sk_live_...` or `sk_test_...`) |
| `STRIPE_WEBHOOK_SECRET` | — | Webhook endpoint signing secret |
| `STRIPE_PORTAL_RETURN_URL` | — | Customer portal return URL |

## Ports

| Port | Purpose |
|------|---------|
| 3740 | Stripe plugin REST API |

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
