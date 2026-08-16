# nself update

> Update the ɳSelf CLI binary and admin UI.

## Synopsis

```
nself update [flags]
```

## Description

`nself update` checks for new releases of the ɳSelf CLI on GitHub (`nself-org/cli`) and installs the latest version. It replaces the current binary in-place and optionally restarts running services after the update.

By default, `nself update` updates both the CLI binary and the admin Docker image. Use `--cli` or `--admin` to update only one component. Use `--check` to see if an update is available without installing anything.

If you are running an active ɳSelf project and update the CLI, you may want to run `nself build --force && nself restart` afterward to regenerate `docker-compose.yml` with any new defaults from the updated version.

## v0.9 legacy detection

When running on a v0.9 CLI (any `0.x.y` binary), `nself update --check` queries `ping.nself.org/version`, receives a migration block in the response, and surfaces the migration guide URL directly in the terminal:

```
WARNING: You are running a legacy v0.9 CLI. Migration to v1.1.1 is required.
Upgrade guide: https://nself.org/docs/migrate/from-v0.9
```

The check is read-only and non-destructive. It does not install anything. See [[Upgrade-From-v0.9]] for the full migration guide.

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
