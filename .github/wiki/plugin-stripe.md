# Stripe Plugin

> Stripe billing data sync with full webhook handling. **Pro plugin, requires license.**

> **Not the same as** `nself-stripe` (port 3830), which handles nSelf Cloud MAX subscription billing. See [[plugin-nself-stripe-pro]] if you need nSelf's own billing infrastructure.

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install stripe
nself build
```

The license is validated against `ping.nself.org/license/validate`. Stripe plugin is available at pro tier and above.

## What It Does

Syncs your Stripe account into Postgres in real time: customers, subscriptions, invoices, payment intents, products, prices, refunds, and disputes. Handles incoming Stripe webhooks (signature-verified) and keeps all 24 tables current. Exposes a customer portal API for self-service subscription management. Query all billing data via Hasura GraphQL without hitting the Stripe API on every request.

## Distinction from nself-stripe

| Plugin | Port | Purpose |
|--------|------|---------|
| `stripe` (this page) | 3740 | Sync **your app's** Stripe account into Postgres |
| `nself-stripe` | 3830 | nSelf Cloud MAX subscription billing infrastructure |

Use `stripe` when you are building a product that takes Stripe payments and want that data in your local database. Use `nself-stripe` only when operating a self-hosted nSelf Cloud deployment.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `STRIPE_PORT` | `3740` | Stripe plugin port |
| `STRIPE_API_KEY` | — | Stripe secret key (`sk_live_...` or `sk_test_...`) |
| `STRIPE_WEBHOOK_SECRET` | — | Webhook endpoint signing secret |
| `STRIPE_API_KEYS` | — | Comma-separated keys for multi-account mode |
| `STRIPE_ACCOUNT_LABELS` | — | Labels matching `STRIPE_API_KEYS` order |
| `STRIPE_WEBHOOK_SECRETS` | — | Webhook secrets matching `STRIPE_API_KEYS` order |
| `STRIPE_ACCOUNT_ID` | — | Default Stripe account ID |

Either `STRIPE_API_KEY` (single account) or `STRIPE_API_KEYS` (multi-account) must be set.

## Ports

| Port | Purpose |
|------|---------|
| 3740 | Stripe REST API and webhook receiver |

## Webhook Events Handled

The plugin processes all major Stripe event types via `POST /stripe/webhook`:

- **Customers:** `customer.created`, `customer.updated`, `customer.deleted`
- **Subscriptions:** `customer.subscription.*` (created, updated, deleted, trial_will_end)
- **Invoices:** `invoice.created`, `invoice.paid`, `invoice.payment_failed`, `invoice.finalized`
- **Payments:** `payment_intent.created`, `payment_intent.succeeded`, `payment_intent.payment_failed`
- **Charges:** `charge.succeeded`, `charge.failed`, `charge.refunded`, `charge.dispute.*`

All webhook payloads are signature-verified with `STRIPE_WEBHOOK_SECRET` before processing.

## Database Tables

24 tables prefixed `np_stripe_`:

- `np_stripe_customers`, customer records
- `np_stripe_subscriptions`, subscription state
- `np_stripe_invoices`, invoice history
- `np_stripe_payment_intents`, payment records
- `np_stripe_products`, product catalog
- `np_stripe_prices`, pricing tiers
- `np_stripe_refunds`, refund records
- `np_stripe_disputes`, dispute and chargeback records
- `np_stripe_coupons`, discount coupons
- `np_stripe_tax_rates`, tax rate definitions
- `np_stripe_webhook_events`, raw incoming webhook payloads
- Plus 13 more for charges, payment methods, trials, transfers, and balance transactions

Each table includes `source_account_id` for multi-app isolation.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/stripe/webhook` | Stripe event webhook receiver (signature-verified) |
| `/stripe/portal` | Customer portal session API |

## API

```
POST /stripe/webhook              — Stripe webhook receiver
POST /portal/session              — Create customer portal session
POST /checkout/session            — Create checkout session
GET  /subscriptions/{customer_id} — Get customer subscription state
```

## Docker Image

```bash
docker pull nself/plugin-stripe:latest
```

## See Also

- [[plugin-nself-stripe-pro]], nSelf Cloud billing infrastructure
- [[plugin-paypal-pro]], PayPal payment sync
- [[plugin-shopify-pro]], Shopify store sync
- [[Plugin-Overview]], full plugin index
- [[Plugin-Licensing]], tier comparison

← [[Plugin-Overview]] | [[Home]] →
