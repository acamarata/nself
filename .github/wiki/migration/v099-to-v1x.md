# Migrating from nSelf v0.9.9 to v1.x

> Operator-facing guide for upgrading from the Bash-era nSelf CLI (v0.9.9) to the v1.x Go binary. Two migrations apply. Both are opt-in. Run them in order.

The v0.9.9 → v1.x upgrade has two independent migration paths. They cover different state and can be run in either order, though most operators run the home-state shim first.

## 1. Home-Level State (license, channel, ssh keys)

The v0.9.9 CLI persisted state under `~/.nself/v099-state/`. The v1.x CLI uses `~/.config/nself/` for typed config and `~/.nself/` for binaries and ssh keys. Run the home-state shim:

```bash
nself migrate-from-v099 --check    # Detect, do not migrate
nself migrate-from-v099            # Interactive prompt
nself migrate-from-v099 --yes      # Non-interactive (CI)
```

What the shim does:

| Legacy path | New path | Notes |
|-------------|----------|-------|
| `~/.nself/v099-state/license.key` | `~/.config/nself/license.json` | Marked `verified:false`. Re-run `nself license verify`. |
| `~/.nself/v099-state/channel` | `~/.config/nself/channel.json` | Same channel preserved (`stable` or `canary`). |
| `~/.nself/v099-state/ssh/` | `~/.nself/ssh/` | Perms preserved at 0600. |
| `~/.nself/v099-state/plugin-install.log` | (backup only) | v1.x manages plugin state per-project. |

Everything is backed up to `~/.nself/v099-state-backup/<timestamp>/` first. The legacy directory is renamed to `~/.nself/v099-state.migrated.<timestamp>/` so re-runs are no-ops.

After migration, re-validate your license:

```bash
nself license verify
```

## 2. Per-Project Artifacts (config.sh, compose, .envrc)

Each v0.9.9 project directory has its own state. Run inside each project directory:

```bash
cd path/to/your-v099-project
nself migrate from-bash            # Detect and print step-by-step guide
nself migrate from-bash --auto     # Run automatable steps after confirmation
```

This handles `.nself/config.sh`, hand-written `docker-compose.yml`, `nself.sh`, and `.envrc`. See [[Migrate-From-Bash]] for the full per-project flow.

## Plugin Renames and Deprecated Commands

A handful of plugin and command names changed between v0.9.9 and v1.x. Re-install plugins after migration to pick up the current names:

```bash
nself plugin list                  # Current state
nself plugin install <name>        # Re-install if needed
```

For the full plugin and command-rename map, see [[Plugin-Catalog]] and [[Commands]].

## Environment Variables

v0.9.9 exported env vars from `.nself/config.sh` via `source`. v1.x reads the cascade: `.env.dev → .env.local → .env.staging/.env.prod → .env.secrets → .env.computed`. Use the per-project migration in step 2 above to translate `config.sh` exports into `.env.dev` lines.

## Rollback

If anything goes wrong, the home-state shim leaves the legacy directory in place under `~/.nself/v099-state.migrated.<timestamp>/`. Move it back if needed:

```bash
mv ~/.nself/v099-state.migrated.<timestamp> ~/.nself/v099-state
```

Per-project rollback uses the existing `nself migrate rollback` command on the project directory.

## Cross-References

- [[Migrate-From-Bash]], per-project Bash-era migration
- [[Upgrading]], general upgrade flow
- [[License-Workflow]], license verification after migration
- [[Home]]
