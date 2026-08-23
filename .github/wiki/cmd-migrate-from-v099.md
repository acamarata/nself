# nself migrate-from-v099

<!-- BEGIN PROSE:summary -->
> Migrate v0.9.9 home-level state (license key, update channel, SSH keys) to the v1.x layout.
<!-- END PROSE:summary -->

## Synopsis

```
nself migrate-from-v099 [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself migrate-from-v099` is a one-time migration command for operators who ran the
v0.9.9 Bash-era CLI and are upgrading to v1.x.

The v0.9.9 CLI stored persistent home-level state under `~/.nself/v099-state/`. The Go
CLI (v1.x) moved that state to `~/.config/nself/` and `~/.nself/`. This command detects
the legacy directory, backs everything up, and moves each recognised artifact to its
v1.x location.

What gets migrated:

| Legacy path | v1.x destination | Notes |
|-------------|-----------------|-------|
| `~/.nself/v099-state/license.key` | `~/.config/nself/license.json` | Saved with `verified: false`; run `nself license verify` after |
| `~/.nself/v099-state/channel` | `~/.config/nself/channel.json` | Update channel preference (stable/edge) |
| `~/.nself/v099-state/id_nself` | `~/.nself/ssh/id_nself` | Permissions set to 0600 |
| `~/.nself/v099-state/id_nself.pub` | `~/.nself/ssh/id_nself.pub` | Public key |

All migrated files are backed up to `~/.nself/v099-state-backup/<timestamp>/` before any
changes are made. After migration the legacy directory is renamed to
`~/.nself/v099-state.migrated.<timestamp>/` and left in place.

This command does not touch any per-project directories. For project-level Bash-era
artifacts, use `nself migrate from-bash`.

Run with `--check` first to see what would be migrated without making any changes. Run
interactively (the default) to confirm each migration, or use `--yes` for scripted
environments.

After the command completes, re-validate your license:

```bash
nself license verify
```

## Breaking Changes

### Home directory layout change (v0.9.9 → v1.x)

The most significant breaking change between v0.9.9 and v1.x is the location of
home-level CLI state:

- **v0.9.9:** `~/.nself/v099-state/` (flat directory, Bash-written files)
- **v1.x:** `~/.config/nself/` for configuration (license, channel) and `~/.nself/` for
  runtime state (SSH keys)

If you run `nself license verify` or `nself update` before migrating, the CLI will
report that no license or channel preference is configured, because it reads from the v1.x
paths only.

### License key format change

v0.9.9 stored the raw license key string in `license.key`. v1.x stores a JSON object in
`license.json` with additional metadata (`verified`, `tier`, `validatedAt`). The migrated
file is written with `verified: false`; the `nself license verify` call re-validates the
key against `ping.nself.org` and fills in the remaining fields.

### SSH key permissions

v0.9.9 did not enforce strict permissions on SSH keys. v1.x requires 0600. The migration
sets 0600 automatically; if a key already exists at the destination with different
permissions, it is backed up and replaced.

## Data Migration

Before running the migration command, review what is present in your legacy directory:

```bash
nself migrate-from-v099 --check
```

The output lists each artifact found and the action that would be taken:
`migrate → <destination>` or `backup-only (no v1.x destination)`.

Artifacts not recognised by the migration shim are placed in the backup directory and left
in the renamed legacy directory. They are never deleted.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--check` | `false` | Detect legacy state without migrating |
| `--yes` | `false` | Skip the confirmation prompt (non-interactive) |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
Detect legacy state without making any changes — run this first to see what would be migrated:

```bash
nself migrate-from-v099 --check
```

Run the interactive migration, which prompts before moving each artifact:

```bash
nself migrate-from-v099
```

Skip confirmation prompts for scripted or CI environments:

```bash
nself migrate-from-v099 --yes
```

Check what would migrate, review the output, then run the migration in one sequence:

```bash
nself migrate-from-v099 --check
nself migrate-from-v099 --yes
```

After migration, re-validate the license key against the v1.x license store:

```bash
nself license verify
```

Confirm the legacy directory was renamed and a backup was created:

```bash
ls ~/.nself/ | grep v099
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [cmd-migrate.md](cmd-migrate.md) — `nself migrate from-bash` for per-project Bash-era artifacts
- [cmd-license.md](cmd-license.md) — manage and verify license keys
- [cmd-update.md](cmd-update.md) — update the CLI binary
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
