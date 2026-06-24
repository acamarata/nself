# PayPal Plugin

> PayPal payment data sync with webhook handling and order management. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Any bundle | $0.99/mo | $9.99/yr | If in bundle |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** Basic (this is a `tier: pro` plugin per F07-PRICING-TIERS).

## Bundle membership

Not currently in a named bundle. Purchase any tier subscription (Basic and up) for access.

Or get all bundles + all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install paypal
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked server-side; insufficient tier returns an error.

## Description

Syncs PayPal payment data into your Postgres database via the PayPal REST API and webhooks. Tracks orders, captures, refunds, disputes, and subscriptions. Provides a webhook receiver for real-time payment event processing. With 14 synced tables, query PayPal data directly via Hasura GraphQL.

Webhook events are verified using PayPal's signature validation (`PAYPAL_WEBHOOK_ID` required). Key events processed include `payment.capture.completed`, `payment.capture.refunded`, `billing.subscription.activated`, `billing.subscription.cancelled`, and dispute lifecycle events. Each event is persisted to `np_paypal_webhook_events` and fanned out to the relevant domain tables.

Supports sandbox and live environments. Set `PAYPAL_ENVIRONMENT=sandbox` for development and switch to `live` for production. Multi-account configuration is available via the array env vars (`PAYPAL_CLIENT_IDS`, `PAYPAL_CLIENT_SECRETS`).

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string (auto-provided by nself) |
| `PAYPAL_CLIENT_ID` | Yes | — | PayPal REST API client ID |
| `PAYPAL_CLIENT_SECRET` | Yes | — | PayPal REST API client secret |
| `PAYPAL_WEBHOOK_ID` | Yes | — | PayPal webhook ID for signature validation |
| `PAYPAL_ENVIRONMENT` | No | `sandbox` | Mode: `sandbox` or `live` |
| `PAYPAL_SYNC_INTERVAL` | No | `300` | Sync interval in seconds |
| `PAYPAL_CLIENT_IDS` | No | — | Comma-separated client IDs for multi-account |
| `PAYPAL_CLIENT_SECRETS` | No | — | Comma-separated secrets for multi-account |
| `PAYPAL_ACCOUNT_LABELS` | No | — | Labels for multi-account display |
| `PAYPAL_WEBHOOK_IDS` | No | — | Comma-separated webhook IDs for multi-account |
| `PAYPAL_WEBHOOK_SECRETS` | No | — | Comma-separated webhook secrets for multi-account |

Reference vault credentials via `nself secrets`. Never hardcode secrets.

## Ports

| Port | Purpose |
|------|---------|
| 3741 | PayPal plugin REST API and webhook receiver |

> **Port conflict note:** Port 3741 is also allocated to the `byok` (Bring Your Own Key) plugin. Running both `paypal` and `byok` simultaneously on the same host will cause a port collision. This conflict is tracked in F10-PORT-REGISTRY and requires resolution before both plugins can coexist. Workaround: assign a custom port to one plugin via its env configuration, or avoid running both concurrently.

## Database Schema

14 tables prefixed `np_paypal_`:

- `np_paypal_transactions` — transaction log
- `np_paypal_orders` — order records
- `np_paypal_captures` — payment captures
- `np_paypal_authorizations` — payment authorizations
- `np_paypal_refunds` — refund records
- `np_paypal_disputes` — dispute and chargeback records
- `np_paypal_subscriptions` — subscription state
- `np_paypal_subscription_plans` — subscription plan definitions
- `np_paypal_products` — product catalog
- `np_paypal_payouts` — payout records
- `np_paypal_invoices` — invoice history
- `np_paypal_payers` — payer (customer) records
- `np_paypal_balances` — account balance snapshots
- `np_paypal_webhook_events` — raw incoming webhook payloads

All tables use `source_account_id` for multi-app isolation.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/paypal/webhook` | PayPal webhook receiver (POST) |
| `/paypal/orders/` | Order creation and capture |
| `/paypal/subscriptions/` | Subscription management |
| `/paypal/sync` | Manual sync trigger |

## Examples

### Trigger a manual sync

```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  https://api.example.com/paypal/sync
```

### Query recent transactions via Hasura

```graphql
query RecentTransactions {
  np_paypal_transactions(
    order_by: {created_at: desc}
    limit: 20
  ) {
    id
    amount
    currency
    status
    created_at
  }
}
```

### Query active subscriptions

```graphql
query ActiveSubscriptions {
  np_paypal_subscriptions(where: {status: {_eq: "ACTIVE"}}) {
    id
    payer_id
    plan_id
    next_billing_time
  }
}
```

### Check webhook event log

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.example.com/paypal/webhook-events?limit=10"
```

## Docker Hub

```bash
docker pull nself-org/plugin-paypal:latest
```

Image: `nself-org/plugin-paypal:latest`

## Source

Source-available (license required to run): [`plugins-pro/paid/paypal/`](https://github.com/nself-org/plugins-pro/tree/main/paid/paypal)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers and Enterprise customers.

## See Also

- [[plugin-stripe-pro]] — Stripe billing integration
- [[plugin-shopify-pro]] — Shopify store sync
- [[plugin-donorbox-pro]] — Donorbox donation sync
- [[plugin-byok]] — shares port 3741 (conflict — see Ports note above)
- [[Plugin-Overview]] — full plugin index
- [[Plugin-Licensing]] — tier comparison

← [[Plugins]] | [[Home]] →
