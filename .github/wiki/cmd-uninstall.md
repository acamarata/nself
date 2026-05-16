# nself uninstall

> Remove nSelf-generated files and containers from the current project directory.

## Synopsis

```
nself uninstall [flags]
```

## Description

`nself uninstall` stops running containers and deletes files that nSelf generated for the current project: `docker-compose.yml`, the nginx site configs in `nginx/sites/`, and the `.nself/cache/` directory. Your `.env*` files, hand-managed `nginx/conf.d/` fragments, and application source code are never touched.

By default the command keeps Postgres database volumes so your data survives the removal. Add `--purge` to also delete the volumes. The command is interactive by default: it describes what it will do and prompts for confirmation before proceeding. Use `--yes` to skip prompts in scripted or CI environments.

> **Warning: Data Loss Risk.** Running `nself uninstall --purge` permanently deletes the Postgres database volumes for this project. This action cannot be undone. Keep a backup before running `--purge`, or omit the flag to preserve your data.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--keep-data` | | bool | `false` | Keep database volumes (the default behaviour when no flag is given) |
| `--purge` | | bool | `false` | Remove everything including database volumes. Destructive and permanent |
| `--yes` | `-y` | bool | `false` | Skip all confirmation prompts |

`--keep-data` and `--purge` are mutually exclusive.

## Examples

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

## See Also

- [cmd-stop.md](cmd-stop.md) — stop containers without removing any files
- [cmd-build.md](cmd-build.md) — regenerate docker-compose.yml
- [cmd-init.md](cmd-init.md) — initialise a new nSelf project
- [Commands.md](Commands.md) — full command index

← [[Commands]] | [[Home]] →
