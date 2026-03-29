# Quick Start

Get a full backend running in under 5 minutes.

## Step 1 — Initialize

```bash
nself init myapp
```

The interactive wizard asks:

| Prompt | What it sets |
|--------|-------------|
| Project name | `PROJECT_NAME` in `.env` — used as Docker network prefix |
| Base domain | `BASE_DOMAIN` — e.g. `myapp.dev` or `localhost` |
| Services to enable | Redis, MinIO storage, email, search, monitoring |
| Postgres password | Auto-generated secure password (accept or override) |
| Hasura admin secret | Auto-generated (accept or override) |

Accept defaults for a fast local setup. Run `nself init --fast` to skip all prompts.

## Step 2 — Build

```bash
nself build
```

Generates `docker-compose.yml`, Nginx config, and SSL certificates from your `.env`. Takes a few seconds.

## Step 3 — Start

```bash
nself start
```

Boots the stack in order: PostgreSQL first, then Hasura, Auth, Nginx. You'll see each service reach healthy status:

```
✓ postgres   healthy (2s)
✓ hasura     healthy (8s)
✓ auth       healthy (5s)
✓ nginx      healthy (1s)
```

## Step 4 — View URLs

```bash
nself urls
```

Output:

```
Service        URL
─────────────────────────────────────────
Hasura         https://hasura.local.nself.org
Auth           https://auth.local.nself.org
GraphQL API    https://api.local.nself.org/v1/graphql
```

## Step 5 — Run a GraphQL Query

Open the Hasura Console at `https://hasura.local.nself.org` (or the URL shown by `nself urls`).

In the GraphQL Explorer, run:

```graphql
query {
  __typename
}
```

Expected response:

```json
{
  "data": {
    "__typename": "query_root"
  }
}
```

Your GraphQL API is live.

## Step 6 — Stop

```bash
nself stop
```

Gracefully shuts down all containers. Data is preserved in Docker volumes.

---

Next: [[First-Project]] · [[Commands]]
