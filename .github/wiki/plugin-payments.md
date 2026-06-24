# Plugin: payments

**Tier:** Pro (requires license)
**Port:** 3086
**Category:** Commerce / Billing
**Language:** Go
**Bundle:** Unbundled (cross-bundle pro plugin)

## Overview

The `payments` plugin is the unified payment abstraction layer for ɳSelf deployments. It provides a single API surface across Stripe, Lemon Squeezy, and Paddle, normalizing webhook events from all three providers into a canonical subscription model stored in Postgres.

Key capabilities:

- Checkout session creation (provider-agnostic)
- Subscription lifecycle management (create, pause, cancel, portal)
- Webhook ingestion with HMAC-SHA256 signature verification per provider
- Dunning engine: configurable retry schedule with grace period before cancellation
- Usage metering records for metered billing plans
- Apple Pay and Google Pay token forwarding to the active provider
- Prometheus `/metrics` endpoint for observability

## Install

```bash
nself plugin install payments
```

Requires a valid `pro` license key. To verify:

```bash
nself license info
```

## Configuration

Set the required env vars before starting the plugin. The minimum required set:

```bash
DATABASE_URL=postgres://...
PAYMENTS_DEFAULT_PROVIDER=stripe   # or: lemonsqueezy, paddle
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

All env vars:

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `PAYMENTS_DEFAULT_PROVIDER` | Yes | — | Active provider: `stripe`, `lemonsqueezy`, `paddle` |
| `PAYMENTS_PORT` | No | `3086` | HTTP listen port |
| `STRIPE_SECRET_KEY` | Conditional | — | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Conditional | — | Stripe webhook HMAC secret |
| `STRIPE_PRICE_IDS` | No | — | Comma-separated Stripe price IDs |
| `LEMONSQUEEZY_API_KEY` | Conditional | — | Lemon Squeezy API key |
| `LEMONSQUEEZY_STORE_ID` | Conditional | — | Lemon Squeezy store ID |
| `LEMONSQUEEZY_WEBHOOK_SECRET` | Conditional | — | Lemon Squeezy signing secret |
| `PADDLE_API_KEY` | Conditional | — | Paddle API key |
| `PADDLE_WEBHOOK_SECRET` | Conditional | — | Paddle signing secret |
| `PADDLE_PUBLIC_KEY` | Conditional | — | Paddle RSA public key |
| `PAYMENTS_DUNNING_GRACE_DAYS` | No | `7` | Grace period before subscription cancels |
| `PAYMENTS_DUNNING_RETRY_INTERVALS` | No | `1,3,7` | Day offsets for dunning retries |
| `LOG_LEVEL` | No | `info` | `debug`, `info`, `warn`, `error` |

## API Endpoints

All endpoints are served on port 3086.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/health` | None | Health check — returns `{"status":"ok"}` |
| `GET` | `/metrics` | Internal | Prometheus metrics |
| `POST` | `/checkout` | License | Create a checkout session; returns redirect URL |
| `GET` | `/subscription/{id}` | License | Fetch current subscription state |
| `POST` | `/portal` | License | Create a billing portal session URL |
| `DELETE` | `/subscription/{id}` | License | Cancel a subscription |
| `POST` | `/subscription/{id}/pause` | License | Pause a subscription |
| `GET` | `/plans` | License | List available plans from the active provider |
| `POST` | `/webhook/stripe` | Signature | Stripe webhook ingress |
| `POST` | `/webhook/lemonsqueezy` | Signature | Lemon Squeezy webhook ingress |
| `POST` | `/webhook/paddle` | Signature | Paddle webhook ingress |

## Webhook Setup

Configure your payment provider's webhook to POST to:

```
https://your-domain.com/webhook/<provider>
```

The plugin verifies the provider-specific HMAC-SHA256 signature on every incoming webhook before any processing. Invalid signatures return `401`. Duplicate event IDs (already written to `np_payment_events`) are silently dropped (idempotent).

### Stripe

Set the webhook secret as `STRIPE_WEBHOOK_SECRET`. Register the following events in the Stripe dashboard:
- `customer.subscription.created`
- `customer.subscription.updated`
- `customer.subscription.deleted`
- `invoice.payment_failed`
- `invoice.payment_succeeded`

### Lemon Squeezy

Set `LEMONSQUEEZY_WEBHOOK_SECRET`. Register:
- `subscription_created`
- `subscription_updated`
- `subscription_payment_failed`
- `subscription_payment_success`

### Paddle

Set `PADDLE_WEBHOOK_SECRET` and `PADDLE_PUBLIC_KEY` (RSA public key for signature verification). Register:
- `subscription.created`
- `subscription.updated`
- `subscription.canceled`
- `transaction.payment_failed`
- `transaction.completed`

## Database Tables

| Table | Description |
|---|---|
| `np_subscriptions` | Canonical subscription state, synced via webhooks |
| `np_usage_records` | Metered billing records |
| `np_payment_events` | Raw webhook event log (idempotency + audit trail) |

## Dunning Engine

When a payment fails, the dunning engine schedules retry attempts at the day offsets set in `PAYMENTS_DUNNING_RETRY_INTERVALS` (default: 1, 3, 7 days). After `PAYMENTS_DUNNING_GRACE_DAYS` (default: 7) without success, the subscription moves to `canceled` in `np_subscriptions`.

To disable dunning, set `PAYMENTS_DUNNING_GRACE_DAYS=0`.

## Observability

The `/metrics` endpoint exposes Prometheus counters and histograms for:
- Checkout sessions created per provider
- Webhook events received, verified, and processed per provider
- Subscription state transitions
- Dunning retries and cancellations
- HTTP request latencies per endpoint

## Section I vs Section J Note

The `payments` plugin appears in both Section I (Unbundled / Cross-Bundle Pro) and Section J (P95–P101 Infrastructure / Enterprise) of the ɳSelf plugin inventory. These are **the same plugin** (`plugins-pro/paid/payments/`). The Section J entry was added during P95 infrastructure expansion and references the same codebase and port. There is one `payments` plugin on disk.

## Related Plugins

- `stripe` — free-standing Stripe-only variant (lighter weight, no provider abstraction)
- `paypal` — free-standing PayPal-only variant
- `shopify` — Shopify orders and product sync (separate plugin, no subscription management)
