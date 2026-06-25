# Payments Plugin (Pro)

> Unified payments abstraction for Stripe, PayPal, and Apple/Google Pay with normalized webhook handling. **Pro plugin.**

> **Requires:** Pro license or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install payments
```

## What It Does

Provides a single consistent payments API across multiple providers. Normalizes orders, transactions, and webhook events from Stripe and PayPal into a unified Postgres schema. Includes dunning management for failed subscription charges and Prometheus metrics.

> **Two payments plugins:** `payments` (port 3086, this plugin) is the multi-provider abstraction layer — use it when you need Stripe + PayPal in one API. The `stripe` plugin (port 3740) is a dedicated deep Stripe billing sync (customers, subscriptions, invoices, 24 tables) — use it for Stripe-only deployments that want raw Stripe data in Postgres. The two plugins serve different purposes and are independently installable.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PAYMENTS_PORT` | `3086` | Service port |
| `DATABASE_URL` | — | Postgres connection string (required) |
| `STRIPE_SECRET_KEY` | — | Stripe secret key (`sk_live_...` or `sk_test_...`) |
| `PAYPAL_CLIENT_ID` | — | PayPal OAuth2 client ID |
| `PAYPAL_CLIENT_SECRET` | — | PayPal OAuth2 client secret |

## Ports

| Port | Purpose |
|------|---------|
| 3086 | Payments REST API |

## Database Tables

| Table | Purpose |
|-------|---------|
| `np_payments_orders` | Normalized order records across all providers |
| `np_payments_transactions` | Individual transaction and webhook event log |

## API

```
GET  /health                    — Liveness check
POST /orders                    — Create a payment order
GET  /orders/{id}               — Get order and transaction status
POST /orders/{id}/capture       — Capture a PayPal order
POST /webhook/stripe            — Stripe event webhook receiver
POST /webhook/paypal            — PayPal IPN/webhook receiver
GET  /metrics                   — Prometheus metrics endpoint
```
