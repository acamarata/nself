# ɳStripe Plugin (`nself-stripe`)

The `nself-stripe` plugin provides Stripe Connect Express billing for nSelf Cloud MAX. It handles connected-account onboarding, checkout session creation, webhook ingestion (platform and Connect), and entitlement caching. It is part of the **ɳSentry** bundle and is license-gated: the plugin refuses to start unless the active license includes the ɳSentry bundle.

> **Cloud MAX only.** This plugin isolates all data by `tenant_id` (Cloud Multi-Tenancy convention). It is not intended for self-hosted multi-app deployments. A self-host variant would be a separate plugin keyed by `source_account_id` — see `SPEC.md`.

## Purpose

- Onboard tenant operators to Stripe Connect Express and store their connected account.
- Create Checkout sessions for tenant customers.
- Verify and process Stripe webhooks (HMAC-signed) for both the platform account and connected accounts, with idempotent dedup.
- Maintain a per-tenant entitlement cache so the rest of the stack can answer "does this entity have capability X?" without round-tripping to Stripe.

## Port and endpoints

The plugin listens on **port 3830** (`PORT` env overrides). All routes are served via `chi`.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/health` | none | Liveness probe (used by Docker HEALTHCHECK). |
| `GET` | `/ready` | none | Readiness probe (checks DB pool). |
| `GET` | `/metrics` | none | Operational counters. |
| `POST` | `/onboard` | bearer | Begin Stripe Connect Express onboarding. |
| `GET` | `/onboard/callback` | none | Stripe OAuth return URL. |
| `POST` | `/checkout` | bearer | Create a Checkout session. |
| `POST` | `/stripe/webhook/platform` | HMAC | Platform-account webhook ingest. |
| `POST` | `/stripe/webhook/connect` | HMAC | Connected-account webhook ingest. |
| `GET` | `/entitlement/{entity_id}/{capability}` | bearer | Entitlement check (cache-first). |
| `GET` | `/account/{entity_id}` | bearer | Connected-account status for an entity. |
| `POST` | `/account/{entity_id}/disconnect` | bearer | Deauthorize a connected account. |

Webhook routes verify the Stripe signature using HMAC-SHA256 against the configured signing secret, with a configurable timestamp tolerance, before any handler runs. After signature verification the handler attempts to record the event ID in `np_stripe_processed_events` with `ON CONFLICT DO NOTHING`; if the row already existed it returns `200` immediately (replay-safe).

## Configuration (environment variables)

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | yes | Postgres connection string. |
| `STRIPE_PLATFORM_SECRET_KEY` | yes | Stripe secret key for the platform account. |
| `STRIPE_PLATFORM_WEBHOOK_SECRET` | yes | Signing secret for `/stripe/webhook/platform`. |
| `STRIPE_CONNECT_WEBHOOK_SECRET` | yes | Signing secret for `/stripe/webhook/connect`. |
| `STRIPE_CLIENT_ID` | yes | Stripe Connect OAuth client ID. |
| `NSELF_DB_ENCRYPTION_KEY` | recommended | pgcrypto key for at-rest encryption of access/refresh tokens. |
| `NSELF_TENANT_ID` | optional | Default tenant context for single-tenant deployments. |
| `STRIPE_API_VERSION` | optional | Pin the Stripe API version. |
| `STRIPE_ENTITLEMENT_CACHE_TTL_SECONDS` | optional | Entitlement cache TTL. |
| `STRIPE_WEBHOOK_TOLERANCE_SECONDS` | optional | Max age of a webhook timestamp. |
| `PORT` | optional | Listen port (defaults to 3830). |

## Data model and tenant isolation

The plugin owns three tables, all prefixed `np_stripe_`:

- `np_stripe_accounts` — connected Stripe accounts, one per tenant operator. Access/refresh tokens are stored encrypted when `NSELF_DB_ENCRYPTION_KEY` is set.
- `np_stripe_entitlements` — capability grants per entity per tenant, with `valid_until`.
- `np_stripe_processed_events` — webhook dedup ledger keyed on the Stripe event ID.

Every table carries a `tenant_id UUID` column and a Hasura row filter `{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}` on the `tenant_operator` and `tenant_user` roles. The `admin` and `service` roles bypass the filter (the plugin itself uses `service`). This is enforced in `metadata/hasura/tables.yaml`; cross-tenant reads are blocked at the GraphQL layer.

## Install

```bash
nself plugin install nself-stripe
```

The plugin requires an active license that includes the ɳSentry bundle. After install, set the environment variables above and run `nself build && nself start`. Verify the service is healthy:

```bash
curl -s http://localhost:3830/health
```

The bundled Docker image declares a `HEALTHCHECK` against `/health`, so orchestrators report the container unhealthy if the plugin cannot serve requests.
