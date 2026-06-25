# nself account

> Manage your ɳSelf account: session status, team membership, licenses, devices, and ownership transfer.

## Synopsis

```bash
nself account [subcommand] [flags]
nself account status
nself account team
nself account licenses
nself account devices
nself account transfer <new-owner-email>
```

## Description

`nself account` provides a read-only view into your authenticated session and account state. When invoked with no subcommand it prints the same output as `nself account status`.

Authentication state is stored in `~/.nself/auth.json`. Use `nself login` to establish a session and `nself logout` to clear it. The `account` command reads this file and optionally queries the auth server to verify token freshness.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `status` | Show session status, tier, MFA state, and email verification |
| `login` | Authenticate and write auth token (alias for `nself login`) |
| `logout` | Clear saved credentials (alias for `nself logout`) |
| `team` | List team members and their roles (Pro+ required) |
| `licenses` | Show all license keys associated with this account |
| `devices` | List devices with active sessions for this account |
| `transfer` | Transfer project ownership to another account |

## Flags

### `account status`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Emit status as JSON |

### `account team`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--json` | bool | false | Emit team list as JSON |

### `account transfer`

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--yes` | bool | false | Skip confirmation prompt |

## Examples

```bash
# Check login status
nself account status
```

```bash
# List team members
nself account team
```

```bash
# View all license keys on this account
nself account licenses
```

```bash
# List active device sessions
nself account devices
```

```bash
# Transfer project ownership (prompts for confirmation)
nself account transfer newowner@example.com
```

```bash
# JSON output for scripting
nself account status --json
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Authenticated but server error or malformed response |
| 2 | Not authenticated — run `nself login` |

## See Also

- [[cmd-login.md]] — establish a session
- [[cmd-logout.md]] — clear credentials
- [[cmd-license.md]] — manage license keys
- [[Licensing-Guide.md]] — license tiers and pricing

← [[Commands]] | [[Home]] →
