# plugin-email

Transactional email for nSelf, backed by [Elastic Email](https://elasticemail.com). Sends notifications, receipts, password resets, and other system mail through a single internal API. Runs as a pro plugin on port 9008.

This is a core system plugin and belongs to no bundle. License gating applies (see [[Plugin-Licensing]]).

## What it does

- Sends HTML and plain-text email through the Elastic Email v4 API.
- Stores per-account sender config and a full message history in Postgres.
- Exposes delivery stats and a paginated message log.
- Isolates data per app via `source_account_id` row-level security in Hasura.

SSRF is not a concern here: the plugin only ever calls `api.elasticemail.com`. No user-supplied URLs are fetched.

## Install

```bash
nself license set <your-pro-key>
nself plugin install email
```

The CLI validates your license against `ping.nself.org` before downloading the signed plugin tarball. See [[Plugin-Licensing]] for the full gate flow.

Verify it is running:

```bash
curl http://localhost:9008/health
# {"status":"ok","plugin":"email"}
```

## Configuration

Set these in your environment (`.env` or your secrets manager). Only `DATABASE_URL` and `EMAIL_ELASTIC_API_KEY` are required.

| Variable | Default | Purpose |
|---|---|---|
| `EMAIL_ELASTIC_API_KEY` | (required) | Elastic Email API key. Never persisted to the database. |
| `DATABASE_URL` | (required) | Postgres connection string for config + message log. |
| `EMAIL_FROM_ADDRESS` | (none) | Default sender address. |
| `EMAIL_FROM_NAME` | `nSelf` | Default sender display name. |
| `EMAIL_PORT` | `9008` | HTTP listen port. |
| `EMAIL_HOST` | `0.0.0.0` | HTTP listen host. |
| `EMAIL_RATE_LIMIT_RPM` | `60` | Per-account send rate limit (requests per minute). |
| `EMAIL_RETENTION_DAYS` | `30` | How long message history is kept. |
| `EMAIL_INTERNAL_SECRET` | (none) | Bearer token guarding `/email/*` routes. When unset, routes are open. |

The API key is read from the environment only. The per-account config table stores a 4-character key hint (`api_key_hint`) for reference, never the secret itself.

## Sender config

Each app (`source_account_id`) can override the default sender. Store its config once:

```bash
curl -X PUT http://localhost:9008/email/config \
  -H "Authorization: Bearer $EMAIL_INTERNAL_SECRET" \
  -H "X-Hasura-Source-Account-Id: primary" \
  -H "Content-Type: application/json" \
  -d '{"from_address":"hello@yourapp.com","from_name":"Your App","rate_limit_rpm":120}'
```

When `enabled` is set to `false` on an account, send requests for that account return `403`.

## Send an email

```bash
curl -X POST http://localhost:9008/email/send \
  -H "Authorization: Bearer $EMAIL_INTERNAL_SECRET" \
  -H "X-Hasura-Source-Account-Id: primary" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "subject": "Welcome",
    "html": "<h1>Hi</h1>",
    "text": "Hi"
  }'
# {"status":"sent","message_id":"..."}
```

Every send is logged to `np_email_messages` with its delivery status, whether it succeeded or failed.

## API reference

All `/email/*` routes require the bearer token. Health checks do not.

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Liveness. Always 200 when the process is up. |
| `GET` | `/ready` | Readiness. Pings Postgres; 503 if the DB is down. |
| `GET` | `/email/config` | Read the sender config for the current account. |
| `PUT` | `/email/config` | Upsert the sender config for the current account. |
| `POST` | `/email/send` | Send an email and log it. |
| `GET` | `/email/messages` | List the 50 most recent messages for the account. |
| `GET` | `/email/messages/{id}` | Fetch one message by ID, scoped to the account. |
| `GET` | `/email/stats` | Message counts grouped by status. |

## Data model

Two tables, both prefixed `np_email_` and both carrying `source_account_id TEXT NOT NULL DEFAULT 'primary'`:

- `np_email_configs` — one sender config row per account.
- `np_email_messages` — append-only message log.

Hasura applies a row filter of `{"source_account_id": {"_eq": "X-Hasura-Source-Account-Id"}}` on every role, so one app can never read another app's mail.

## Troubleshooting

- **`EMAIL_ELASTIC_API_KEY not configured`** — the env var is empty. Set it and restart the plugin.
- **`401 unauthorized`** — the bearer token does not match `EMAIL_INTERNAL_SECRET`.
- **`403 email sending disabled`** — the account's config row has `enabled = false`. Update it via `PUT /email/config`.
- **`502` from `/email/send`** — Elastic Email rejected the request. The error body carries the upstream detail, and the failure is recorded in the message log.

See also: [[Plugin-Install]] · [[Plugin-Licensing]] · [[Plugin-Catalog]].
