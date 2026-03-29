# Guide: Security Hardening

Follow this guide after deploying nSelf to production. For a quick checklist, see [[Security-Hardening]].

## Firewall

Close all ports except 80, 443, and 22. All internal service ports (Postgres 5432, Hasura 8080, Auth 4000) must not be reachable from the internet.

```bash
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw default deny incoming
ufw enable
ufw status verbose
```

## Rotate Default Secrets

Before going live, replace all generated default secrets with strong random values:

```bash
# Generate strong secrets (32+ characters)
openssl rand -hex 32   # use output for each secret
```

Set in `.env.secrets`:
```env
POSTGRES_PASSWORD=<strong-random>
HASURA_GRAPHQL_ADMIN_SECRET=<strong-random>
HASURA_JWT_KEY=<strong-random>
AUTH_JWT_SECRET=<strong-random>
```

Never commit `.env.secrets` to git. Verify it is in `.gitignore`.

## Disable Hasura Console in Production

The Hasura GraphQL console exposes schema information and must be disabled in production:

```env
# .env.prod
HASURA_GRAPHQL_ENABLE_CONSOLE=false
```

Rebuild after changing:
```bash
nself build && nself restart
```

## Enable Rate Limiting

Add rate limiting to Auth endpoints to prevent brute-force attacks. The default rate limit is 30 requests/minute.

To customise:
```env
AUTH_RATE_LIMIT=10r/m    # 10 requests per minute
```

For WAF-level protection, install the rate-limit plugin:
```bash
nself plugin install rate-limit
```

## Keep Everything Updated

```bash
nself update          # update nSelf CLI
nself build           # regenerate configs with latest service images
nself restart         # apply updates
```

## Enable Monitoring and Alerts

See [[Guide-Monitoring-Setup]] for step-by-step instructions. Set up CPU, memory, and error-rate alerts in Alertmanager.

## See Also

- [[Security-Hardening]] — production checklist
- [[Security-Architecture]] — how nSelf is secured by design
- [[Guide-Monitoring-Setup]] — enable alerts

---
← [[Home]] | [[_Sidebar]]
