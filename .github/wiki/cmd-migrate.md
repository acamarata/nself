# nself migrate

<!-- BEGIN PROSE:summary -->
> Detect and migrate ɳSelf v0.9.x projects to the current v1.x format.
<!-- END PROSE:summary -->

## Synopsis

```
nself migrate <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself migrate` detects and migrates v0.9.x project artifacts to the current v1.x format. Running `nself migrate` without a subcommand performs a detection scan, the same as `nself migrate detect`, and reports which v0.9 artifacts are present.

The `run` subcommand performs the full automated migration: it stops running containers, backs up the current project state to `.nself/backup/{timestamp}/`, moves nginx configs from the flat `nginx/` layout to `nginx/sites/` (v1 layout), regenerates `docker-compose.yml`, and prints a summary of every change made. The migration is idempotent: running it on an already-migrated project exits cleanly with no changes.

After migration, the CLI prints the exact `nself plugin install` commands for every v0.9 plugin detected in your `.env`. Run those commands to re-install your plugins using the signed v1 bundle system.

If anything goes wrong, use `nself migrate rollback` to restore from the automatic backup.

> **Note:** `nself migrate` manages v0.9→v1 project migration. For database schema migrations within a v1 project, see [[cmd-db]].

## Plugin re-install warning

After `nself migrate run` completes, the output includes a plugin re-install block:

```
┌─────────────────────────────────────────────────────────┐
│  v0.9 PLUGINS DETECTED — RE-INSTALL REQUIRED            │
│                                                         │
│  v0.9 plugin code is not compatible with v1 signed      │
│  bundles. Re-install your plugins using:                │
│                                                         │
│    nself plugin install ai mux notify cron              │
│                                                         │
│  Your license key is already set.                       │
└─────────────────────────────────────────────────────────┘
```

The plugin list is generated from `PLUGIN_<NAME>=true` entries in your v0.9 `.env`. Plugin names are mapped from v0.9 naming to v1 bundle naming automatically.

## CI fixture

A v0.9 test fixture lives at `internal/migration/testdata/v0.9-fixture/`. The GitHub Actions workflow `.github/workflows/migration-fixture.yml` runs migration regression tests on every push to main and nightly. To run locally:

```bash
go test -mod=vendor -run TestE2E ./internal/migration/...
```
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--from-bash` | `false` | Migrate from a v0.9.9 Bash-era project (alias for: nself migrate from-bash) |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `detect` | Detect v1 artifacts in the current project |
| `firebase` | Generate ɳSelf migration artifacts from a Firebase export |
| `from-bash` | Migrate a v0.9.9 Bash-era project to the current ɳSelf CLI |
| `from-v099` | Migrate v0.9.9 home-level state (license key, channel, ssh keys) to v1.x layout |
| `generate` | Generate a SQL migration from a natural-language description |
| `rollback` | Restore a v1 backup created by migrate run |
| `run` | Migrate v1 project to v2 |
| `supabase` | Migrate a Supabase project to ɳSelf |
| `watch` | Watch model files and propose SQL migrations on change |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Scan for v0.9 artifacts (non-destructive)
nself migrate
nself migrate detect

# Perform the v0.9→v1 migration
nself migrate run

# Restore from the most recent backup
nself migrate rollback

# List available backups
nself migrate rollback --list

# Restore from a specific backup
nself migrate rollback --backup 20260417-143022
```

See [[Upgrade-From-v0.9]] for the full step-by-step migration guide.
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
