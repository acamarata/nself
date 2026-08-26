# nself logout

<!-- BEGIN PROSE:summary -->
> Log out of your ɳSelf account.
<!-- END PROSE:summary -->

## Synopsis

```
nself logout [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself logout` clears the stored session token from the local credential store (`~/.nself/auth.json`). After logging out, commands that require cloud authentication will prompt you to log in again.

Plugin license keys are NOT removed by `nself logout`. To clear license keys, use `nself license clear`.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--all` | `false` | Revoke all active sessions, not just the current one |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Log out of your nSelf account
nself logout
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-login]], Log in to your ɳSelf account
- [[cmd-license]], Manage plugin license keys
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
