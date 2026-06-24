# nself gauth

**Manage server-side Google OAuth tokens via plugin-gauth.**

The `gauth` command provides operator-facing tools to manage server-side Google OAuth refresh tokens. It delegates token management to the plugin-gauth service, which handles encrypted storage and token refresh without requiring any browser interaction.

**Subcommands:**
- `nself gauth status` — Show token expiry and cache status for all accounts
- `nself gauth refresh` — Force-refresh a specific account's access token
- `nself gauth revoke` — Revoke and remove a stored refresh token

---

## nself gauth status

Show token expiry and cache status for all provisioned Google OAuth accounts.

**Syntax**

```bash
nself gauth status [--json] [--account <id>]
```

**Flags**

| Flag | Description |
|------|-------------|
| `--json` | Emit JSON output instead of a table |
| `--account <id>` | Filter to one account ID (optional) |

**Examples**

```bash
# Show all accounts
nself gauth status

# Show all accounts in JSON format
nself gauth status --json

# Show one account only
nself gauth status --account my-service-account
```

**Output**

Without `--json`, displays a table:

```
ACCOUNT ID                    EXPIRES AT                     CACHED
----------------------------------------------------------------------
my-service-account            2026-12-31T23:59:59Z           yes
another-account               2026-06-30T12:00:00Z           no
```

With `--json`, returns:

```json
{
  "accounts": [
    {
      "account_id": "my-service-account",
      "expires_at": "2026-12-31T23:59:59Z",
      "cached": true
    }
  ]
}
```

---

## nself gauth refresh

Force-refresh the access token for a specific Google OAuth account.

**Syntax**

```bash
nself gauth refresh --account <id> [--force]
```

**Flags**

| Flag | Description |
|------|-------------|
| `--account <id>` | Account ID to refresh (required) |
| `--force` | Bypass cache and fetch a fresh token from Google |

**Examples**

```bash
# Refresh with cache (returns cached token if still valid)
nself gauth refresh --account my-service-account

# Force a fresh token from Google (bypass cache)
nself gauth refresh --account my-service-account --force
```

**Output**

```
Account my-service-account refreshed. Expires at: 2026-06-25T10:00:00Z
```

**Error handling**

If the refresh token is invalid or revoked, you will see:

```
plugin-gauth error 401: refresh failed/revoked
```

In this case, you must re-provision the account by storing a new refresh token via `nself secrets`.

---

## nself gauth revoke

Revoke a Google OAuth refresh token for a specific account and remove it from encrypted storage.

**Syntax**

```bash
nself gauth revoke --account <id>
```

**Flags**

| Flag | Description |
|------|-------------|
| `--account <id>` | Account ID to revoke (required) |

**Examples**

```bash
nself gauth revoke --account my-service-account
```

**Output**

```
Token revoked and removed for account my-service-account
```

After revocation, the account is no longer available and cannot be used for API calls until a new refresh token is provisioned.

---

## Setup

Before using `gauth` commands, ensure the plugin-gauth service is installed and running:

```bash
nself plugin install plugin-gauth
nself start
```

Then provision a refresh token (obtained from Google's OAuth 2.0 console):

```bash
nself secrets set GAUTH_REFRESH_my-service-account "...refresh-token-value..."
```

Once provisioned, `nself gauth status` will show the account.

---

## See also

- [[cmd-plugin]] — Plugin management
- [[cmd-secrets]] — Secret management
- [[cmd-service]] — Service management
- [[Home]]
