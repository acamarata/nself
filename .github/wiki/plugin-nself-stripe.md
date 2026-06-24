# nself-stripe

**Bundle:** ɳSentry | **Port:** 3830 | **Language:** Go | **License:** Source-Available (commercial)

The `nself-stripe` plugin provides Stripe Connect Express scaffolding for nSelf Cloud MAX. It handles platform account onboarding, dual-webhook processing (platform + Connect), entitlement caching, and idempotent event deduplication for multi-tenant deployments.

This plugin is gated by the `cloud` entitlement and requires an active ɳSentry bundle license.

---

## Purpose

nself-stripe bridges your nSelf Cloud MAX instance to Stripe Connect Express, enabling:

- **Operator onboarding** — guides tenant operators through the Stripe Connect OAuth flow so they can receive revenue splits.
- **Entitlement cache** — stores and serves capability grants (e.g., `premium`, `api_access`) derived from Stripe subscription events, avoiding live Stripe API calls on every request.
- **Webhook fanout** — handles both platform-level events (charges, disputes, transfers) and Connect-level account events (account.updated, deauthorized) with strict HMAC validation and per-event idempotency.
- **Idempotency guard** — deduplicates Stripe webhook retries via the `np_stripe_processed_events` table, keyed on Stripe's globally unique `evt_*` ID.

---

## Installation

```bash
nself plugin install nself-stripe
```

Requires a valid ɳSentry bundle license:

```bash
nself license activate <your-license-key>
```

---

## Configuration

Set the following environment variables before starting the plugin:

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | Yes | PostgreSQL connection string for the nSelf database |
| `STRIPE_PLATFORM_SECRET_KEY` | Yes | Stripe secret key for the platform account (`sk_live_*` or `sk_test_*`) |
| `STRIPE_PLATFORM_WEBHOOK_SECRET` | Yes | Webhook signing secret for platform events (`whsec_*`) |
| `STRIPE_CONNECT_WEBHOOK_SECRET` | Yes | Webhook signing secret for Connect account events (`whsec_*`) |
| `STRIPE_CLIENT_ID` | Yes | OAuth client ID for Stripe Connect Express (`ca_*`) |
| `NSELF_CLOUD_BASE_URL` | Yes | Public base URL of your nSelf Cloud instance (e.g., `https://cloud.nself.org`) |
| `NSELF_TENANT_ID` | No | UUID of the tenant context for standalone runs; Cloud MAX sets this automatically |
| `PORT` | No | Override the default port (default: 3830) |
| `NSELF_BEARER_TOKEN` | No | Static bearer token for authenticated endpoint validation in dev |

---

## Endpoints

### Health and readiness

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/health` | None | Returns plugin status and timestamp |
| GET | `/ready` | None | Returns `{"ready":true}` once database is reachable |
| GET | `/metrics` | None | Minimal Prometheus-style text metrics |

### Stripe Connect OAuth

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/onboard` | None (Stripe OAuth) | Initiates Connect Express onboarding for an operator |
| GET | `/onboard/callback` | None (Stripe OAuth) | Handles OAuth callback and stores account credentials |

### Webhooks (Stripe HMAC only — no JWT)

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/stripe/webhook/platform` | Stripe HMAC | Processes platform-level Stripe events |
| POST | `/stripe/webhook/connect` | Stripe HMAC | Processes Connect account-level events |

### Entitlement and account management (Bearer JWT required)

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/entitlement/{entity_id}/{capability}` | Bearer JWT | Check whether an entity holds a given capability |
| GET | `/account/{entity_id}` | Bearer JWT | Get Stripe Connect account details for an entity |
| POST | `/account/{entity_id}/disconnect` | Bearer JWT | Deauthorize and remove a connected Stripe account |

---

## Database tables

All tables use the `np_` prefix and are scoped by `tenant_id` (Cloud multi-tenancy):

| Table | Purpose |
|---|---|
| `np_stripe_accounts` | One row per tenant operator's connected Stripe Express account |
| `np_stripe_entitlements` | Capability grant cache keyed by `(tenant_id, entity_id, capability)` |
| `np_stripe_processed_events` | Idempotency log keyed by Stripe `evt_*` ID |

Hasura row filters enforce `tenant_id = X-Hasura-Tenant-Id` on all read roles. The `service_role` (used by the Go service internally) has unrestricted CRUD access.

---

## Migrations

Migrations run automatically during `nself plugin install`:

| File | Direction | Description |
|---|---|---|
| `002_billing_extension.sql` | Up | Creates `np_stripe_accounts`, `np_stripe_entitlements`, `np_stripe_processed_events` |
| `002_billing_extension_down.sql` | Down | Drops all three base tables |
| `003_stripe_connect_author_id_up.sql` | Up | Adds `author_id UUID` to `np_stripe_accounts` for Connect revenue splits |
| `003_stripe_connect_author_id_down.sql` | Down | Removes `author_id` column |

---

## Security

- **Webhook HMAC validation** — every webhook request is verified against the Stripe-Signature header before any processing. Invalid signatures return 401 immediately.
- **Idempotency** — duplicate Stripe events (retries) are silently acknowledged (200) without reprocessing.
- **Token encryption** — `access_token` and `refresh_token` in `np_stripe_accounts` should be encrypted at rest using `NSELF_DB_ENCRYPTION_KEY` in production (see SPEC.md §15).
- **Tenant isolation** — all reads are scoped by `tenant_id` via Hasura row filters; cross-tenant data access is impossible through the GraphQL layer.

---

## Related

- [plugin-stripe.md](plugin-stripe.md) — Free Stripe plugin (checkout + portal, no Connect)
- [plugin-stripe-pro.md](plugin-stripe-pro.md) — Pro Stripe plugin (advanced billing features)
- SPEC.md in the plugin source for full architecture and security rationale
