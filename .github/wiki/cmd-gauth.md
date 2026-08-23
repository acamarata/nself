# nself gauth

<!-- BEGIN PROSE:summary -->
> Manage Google OAuth tokens for ɳSelf AI services.
<!-- END PROSE:summary -->

## Synopsis

```
nself gauth <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Manage headless Google OAuth tokens for nSelf AI services.

`nself gauth` delegates to `plugin-gauth` (default port `3762`). No OAuth logic runs in the CLI — it calls the plugin's HTTP endpoints. This is for operator use after provisioning refresh tokens via `nself secret set`.

### `nself gauth status`
Show token expiry and cache state for all provisioned Google accounts.
```
nself gauth status [--account <id>] [--json]
```
**Flags:**
**Output columns:** `ACCOUNT`, `STATUS`, `EXPIRES`, `CACHED`
### `nself gauth refresh`
Force-refresh a Google OAuth access token for a specific account.
```
nself gauth refresh --account <id> [--force]
```
**Flags:**
On success, prints: `Refreshed: account=<id> expires_at=<time>`
**Error cases:**
- `token revoked` — the refresh token was rejected by Google; re-provision via `nself secret set`
- `account not found` — no token stored for this account
### `nself gauth revoke`
Revoke a stored refresh token and clear it from the cache.
```
nself gauth revoke --account <id>
```
**Flags:**
After revocation the account must be re-provisioned via `nself secret set plugin-gauth/GAUTH_REFRESH_<account_id>`.
## Plugin connection

`nself gauth` connects to `plugin-gauth` at `localhost:3762` by default.

Override: set `GAUTH_URL` or `GAUTH_PORT` in your environment.

## Token provisioning

Refresh tokens are stored encrypted in Postgres by `plugin-gauth`. To provision:

```bash
nself secret set plugin-gauth/GAUTH_REFRESH_<account_id> <refresh_token>
```

The refresh token is stored encrypted (AES-256-GCM). It is never returned by any API response or logged.

## Security notes

- `gauth revoke` never prints the refresh token in error messages.
- `gauth status --json` output contains only metadata (`account_id`, `status`, `expires_hint`, `cached`) — never token values.
- All communication between CLI and `plugin-gauth` is local (localhost only).
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `refresh` | Force-refresh a Google OAuth access token for an account |
| `revoke` | Revoke and remove a stored Google OAuth refresh token |
| `status` | Show token expiry for all provisioned gauth accounts |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
<!-- TODO(docs): needs human prose -->

```bash
nself gauth
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
