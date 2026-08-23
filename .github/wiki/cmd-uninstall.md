# nself uninstall

<!-- BEGIN PROSE:summary -->
> Remove nSelf-generated files and containers from the current project directory.
<!-- END PROSE:summary -->

## Synopsis

```
nself uninstall [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself uninstall` stops running containers and deletes files that nSelf generated for the current project: `docker-compose.yml`, the nginx site configs in `nginx/sites/`, and the `.nself/cache/` directory. Your `.env*` files, hand-managed `nginx/conf.d/` fragments, and application source code are never touched.

By default the command keeps Postgres database volumes so your data survives the removal. Add `--purge` to also delete the volumes. The command is interactive by default: it describes what it will do and prompts for confirmation before proceeding. Use `--yes` to skip prompts in scripted or CI environments.

> **Warning: Data Loss Risk.** Running `nself uninstall --purge` permanently deletes the Postgres database volumes for this project. This action cannot be undone. Keep a backup before running `--purge`, or omit the flag to preserve your data.

`--keep-data` and `--purge` are mutually exclusive.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--keep-data` | `false` | Keep database volumes (default behaviour) |
| `--purge` | `false` | Remove everything including database volumes (DESTRUCTIVE) |
| `--yes`, `-y` | `false` | Skip confirmation prompts |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Interactive uninstall, keeps DB volumes (default)
nself uninstall
```

```bash
# Same as default, explicit flag
nself uninstall --keep-data
```

```bash
# Remove everything including the database (prompts for "purge" confirmation)
nself uninstall --purge
```

```bash
# Non-interactive full purge for CI teardown
nself uninstall --purge --yes
```

```bash
# Reinitialise from scratch after uninstall
nself init --force && nself build && nself start
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [cmd-stop.md](cmd-stop.md) — stop containers without removing any files
- [cmd-build.md](cmd-build.md) — regenerate docker-compose.yml
- [cmd-init.md](cmd-init.md) — initialise a new nSelf project
- [Commands.md](Commands.md) — full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
