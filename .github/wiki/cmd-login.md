# nself login

<!-- BEGIN PROSE:summary -->
> Log in to your ɳSelf account.
<!-- END PROSE:summary -->

## Synopsis

```
nself login [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself login` authenticates the CLI with your ɳSelf account (cloud.nself.org or a self-hosted ɳSelf Cloud instance). On success, a session token is stored in the local credential store (`~/.config/nself/credentials.json`) and used automatically by subsequent commands that require authentication.

Self-hosted users with no cloud account can skip this step, the CLI operates without a cloud login for local stack management. `nself login` is required for:

- Accessing `cloud.nself.org` to provision or manage hosted stacks
- Syncing license keys from your account
- Pushing plugin submissions via `nself plugin submit`
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Log in even if already logged in (replaces existing session) |
| `--no-browser` | `false` | Print the login URL instead of opening a browser |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Interactive login (browser-based OAuth)
nself login

# Non-interactive login with PAT (for CI)
nself login --token nself_pat_xxxxxxxxxxxx

# Email + password login (non-interactive)
nself login --email you@example.com --password yourpassword
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-logout]], Log out and clear session
- [[cmd-license]], Manage plugin license keys
- [[cmd-version]], Show CLI version and build info
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
