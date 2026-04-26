# Tiny VPS Setup

Running ɳSelf on a server with less than 1 GB of RAM requires the tiny profile.
The tiny profile starts Postgres and nginx only. Hasura and Auth are opt-in.

---

## When to use the tiny profile

| RAM | Recommendation |
|-----|----------------|
| < 512 MB | Tiny profile required |
| 512 MB – 1 GB | Tiny profile recommended |
| 1 GB + | Standard profile works |

ɳSelf detects available RAM at init time and prints a warning when it is below 512 MB.

---

## Getting started

```bash
nself init --profile=tiny --accept-defaults
nself build
nself start
```

This generates a `.env.dev` with only Postgres and nginx enabled.

---

## What the tiny profile includes

| Service | Status |
|---------|--------|
| Postgres | Enabled |
| Nginx | Enabled |
| Hasura | Disabled (opt-in) |
| Auth | Disabled (requires Hasura) |
| Redis | Disabled |
| MinIO | Disabled |
| Functions | Disabled |
| Monitoring | Disabled |
| Plugins | Disabled (enable per-plugin after adding RAM) |

---

## Enabling Hasura when you have more RAM

Once you upgrade to a server with 1 GB+ RAM:

1. Edit `.env.dev` and set:

```bash
ENABLE_HASURA=true
HASURA_GRAPHQL_ADMIN_SECRET=your_secret_here
HASURA_JWT_KEY=your_jwt_key_here
```

2. Rebuild and restart:

```bash
nself build
nself start
```

---

## Tradeoffs

- No GraphQL API (Hasura disabled). You can query Postgres directly via raw SQL.
- No authentication service. Build your own auth layer or enable Hasura Auth later.
- No plugin support. Plugins require at least Hasura for schema management.

---

## Upgrade path

When you upgrade your VPS:

1. Increase RAM to at least 1 GB.
2. Re-run init with the standard profile:

```bash
nself init --profile=standard --force
nself build
nself start
```

---

## Related pages

- [[cmd-init]], full `nself init` reference
- [[Errors]], troubleshooting guide
- [[Support]], get help

---

← [[Home]] | [[_Sidebar]]
