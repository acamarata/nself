# nself oauth

> Manage OAuth provider tokens for nSelf plugins.

## Synopsis

```
nself oauth <subcommand>
nself oauth refresh <provider> [flags]
nself oauth refresh --all [flags]
```

## Description

`nself oauth` manages OAuth provider tokens. Use `nself oauth refresh` to trigger
an immediate token refresh outside the scheduled cron window — for example after
a provider outage or a B-21-style token expiry incident.

## Subcommands

| Subcommand | Description |
|---|---|
| `refresh` | Trigger immediate OAuth token refresh for a provider |

---

## nself oauth refresh

Force an immediate OAuth token refresh outside the cron schedule.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--account-id <id>` | — | Scope refresh to a single account |
| `--all` | false | Refresh all providers in parallel |
| `--base-url <url>` | `$NSELF_API_URL` | nSelf API base URL |

### Behavior

- Posts to the provider plugin's `/oauth/refresh-now` endpoint
- Runs providers in **parallel** when `--all` is used
- Reports success/failure **per account** — not aggregated
- Never exposes OAuth token values in output (only key prefixes)
- Success is visible in OAuth refresh Prometheus metrics within 5s (requires S41-T03 exporter)
- Exits non-zero if any account fails

### Examples

```bash
# Refresh Google OAuth tokens
nself oauth refresh google

# Refresh a specific account
nself oauth refresh google --account-id user@example.com

# Refresh all configured providers (parallel)
nself oauth refresh --all
```

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | All accounts refreshed successfully |
| 1 | One or more accounts failed — see per-account output |

### See Also

- [Observability](https://docs.nself.org/operations/observability) — oauth_token_refresh_total metric
- [google plugin](https://docs.nself.org/plugins/google) — Google OAuth configuration
