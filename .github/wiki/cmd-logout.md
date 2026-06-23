# nself logout

> Log out of your ɳSelf account.

## Synopsis

```
nself logout [flags]
```

## Description

`nself logout` clears the stored session token from the local credential store (`~/.config/nself/credentials.json`). After logging out, commands that require cloud authentication will prompt you to log in again.

Plugin license keys are NOT removed by `nself logout`. To clear license keys, use `nself license clear`.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--all`, `-a` | `false` | Clear all stored sessions and tokens, not just the current one |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Log out of your nSelf account
nself logout

# Clear all stored sessions and tokens
nself logout --all
```

## See Also

- [[cmd-login]], Log in to your ɳSelf account
- [[cmd-license]], Manage plugin license keys
