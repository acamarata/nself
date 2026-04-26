# nself login

> Log in to your ɳSelf account.

## Synopsis

```
nself login [flags]
```

## Description

`nself login` authenticates the CLI with your ɳSelf account (cloud.nself.org or a self-hosted ɳSelf Cloud instance). On success, a session token is stored in the local credential store (`~/.config/nself/credentials.json`) and used automatically by subsequent commands that require authentication.

Self-hosted users with no cloud account can skip this step, the CLI operates without a cloud login for local stack management. `nself login` is required for:

- Accessing `cloud.nself.org` to provision or manage hosted stacks
- Syncing license keys from your account
- Pushing plugin submissions via `nself plugin submit`

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--email` | — | Log in with email + password (non-interactive) |
| `--password` | — | Password for `--email` login (use with caution , prefer interactive prompt) |
| `--token` | — | Log in with a personal access token |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Interactive login (browser-based OAuth)
nself login

# Non-interactive login with PAT (for CI)
nself login --token nself_pat_xxxxxxxxxxxx

# Email + password login (non-interactive)
nself login --email you@example.com --password yourpassword
```

## See Also

- [[cmd-logout]], Log out and clear session
- [[cmd-license]], Manage plugin license keys
- [[cmd-version]], Show CLI version and build info
