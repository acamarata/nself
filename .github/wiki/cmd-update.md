# nself update

> Update the nSelf CLI binary and admin UI.

## Synopsis

```
nself update [flags]
```

## Description

`nself update` checks for new releases of the nSelf CLI on GitHub (`nself-org/cli`) and installs the latest version. It replaces the current binary in-place and optionally restarts running services after the update.

By default, `nself update` updates both the CLI binary and the admin Docker image. Use `--cli` or `--admin` to update only one component. Use `--check` to see if an update is available without installing anything.

If you are running an active nSelf project and update the CLI, you may want to run `nself build --force && nself restart` afterward to regenerate `docker-compose.yml` with any new defaults from the updated version.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--check` | false | Check for updates without installing |
| `--cli` | false | Only update the CLI binary |
| `--admin` | false | Only update the admin UI Docker image |
| `--force` | false | Force update even if already on the latest version |
| `--restart` | false | Restart services after update |
| `--version` | `""` | Download and install a specific version (e.g. `v1.2.3`) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Check for updates without installing
nself update --check

# Update CLI and admin UI to latest
nself update

# Update only the CLI binary
nself update --cli

# Update only the admin UI
nself update --admin

# Force update even if already up to date
nself update --force

# Install a specific version
nself update --version v1.0.0

# Update and restart services
nself update --restart
```

← [[Commands]] | [[Home]] →
