# nself upgrade

> Upgrade the nSelf CLI (detects install method).

## Synopsis

```
nself upgrade [flags]
```

## Description

`nself upgrade` upgrades the CLI binary. It detects the install method (Homebrew, direct download, or system package manager) and applies the appropriate strategy: Homebrew users get a `brew upgrade` instruction (or the canary tap for the canary channel), system-package users are pointed at their package manager, and direct-install users get an in-place binary swap with rollback support.

Use `--check` to inspect current version, install method, and channel without installing. Use `--channel` to switch between `stable` and `canary`; the choice is persisted in `~/.config/nself/channel.json`. Use `--version 1.2.3` to pin a specific release. Use `--rollback` to restore the previously installed binary from `~/.nself/bin/nself.prev`.

Before swapping the binary, the command warns about installed Pro plugins that may be incompatible with the new version. After a successful upgrade, the previous binary is preserved for one-shot rollback.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--check` | false | Check for updates without installing |
| `--channel` | `""` | Release channel: `stable` or `canary` |
| `--version` | `""` | Upgrade to a specific version (e.g. `1.2.3`) |
| `--rollback` | false | Revert to the previously installed version |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Check current version, channel, and install method
nself upgrade --check

# Upgrade to the latest stable
nself upgrade

# Switch to the canary channel and upgrade
nself upgrade --channel canary

# Pin to a specific version
nself upgrade --version 1.0.6

# Roll back to the previously installed binary
nself upgrade --rollback
```

## See Also

- [[cmd-update]] — update CLI and admin Docker image
- [[cmd-version]] — show current version and system info
- [[cmd-doctor]] — diagnose CLI/install issues
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
