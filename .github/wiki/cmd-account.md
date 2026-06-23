# nself account

> Manage your ɳSelf account, sessions, licenses, team, and devices.

## Synopsis

```
nself account [subcommand] [flags]
```

## Description

`nself account` provides account management for ɳSelf cloud services. Running it with no subcommand shows your current account status (equivalent to `nself account status`).

Use the subcommands to log in and out, inspect licenses and devices, manage team members, or transfer a license to another account.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `login` | Log in via device-code OAuth flow |
| `logout` | Revoke current session and clear local credentials |
| `status` | Show current account summary (default) |
| `team` | List team members. Supports `--invite`, `--remove`, `--role` |
| `licenses` | List active licenses. Supports `--activate <id>` |
| `devices` | List registered devices. Supports `--revoke <id>` |
| `transfer` | Move a license to another account |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Show account status (default)
nself account

# Log in via browser-based device flow
nself account login

# Log out and revoke session
nself account logout

# Show detailed status
nself account status

# List team members
nself account team

# Invite a team member
nself account team --invite teammate@example.com

# List active licenses
nself account licenses

# Activate a license by ID
nself account licenses --activate lic_abc123

# List registered devices
nself account devices

# Revoke a device
nself account devices --revoke dev_xyz789

# Transfer a license to another account
nself account transfer
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error (auth failure, server error) |
| `2` | Misuse (bad arguments or flags) |

## See Also

- [[cmd-login]], Log in to your ɳSelf account
- [[cmd-logout]], Log out and clear session
- [[cmd-license]], Manage plugin license keys
- [[cmd-billing]], Billing operations
