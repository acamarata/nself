# nself env

> Multi-environment management: switch, list, diff, copy.

## Synopsis

```
nself env <subcommand> [flags] [args]
```

## Description

`nself env` manages multiple environments (dev, staging, prod) on a single project. Each environment gets its own Docker project name, network, volume namespace, and port range, so they can coexist on one host without conflicts.

`env use <name>` switches the active environment. After switching, run `nself build` so the generated `docker-compose.yml` and nginx config reflect the new environment's `.env.<name>` cascade. `env list` shows all environments with the active one marked. `env show` prints the resolved config for the current environment with secrets redacted. `env diff a b` compares two environments side-by-side. `env copy from to` scaffolds a new environment from an existing one as the starting template.

The active environment is stored in a project-local marker (`.env.active` or equivalent) so subsequent commands like `nself start`, `nself logs`, and `nself db` all target the right stack.

## Subcommands

| Name | Description |
|------|-------------|
| `use <env>` | Switch active environment |
| `list` | List available environments |
| `show` | Show current environment config (secrets redacted) |
| `diff <env-a> <env-b>` | Compare two environment configs side-by-side |
| `copy <from> <to>` | Scaffold a new environment from an existing one |

## Flags

The `env` subcommands take no flags beyond positional arguments. Use `--help` on any subcommand to confirm.

## Examples

```bash
# See which environments exist
nself env list

# Switch to staging and rebuild
nself env use staging
nself build

# Show the resolved config for the current environment
nself env show

# Compare dev and prod env configs
nself env diff dev prod

# Scaffold a new "qa" environment from staging
nself env copy staging qa
nself env use qa
```

## See Also

- [[cmd-build]], regenerate compose for the active environment
- [[cmd-promote]], promote one environment to another
- [[cmd-secrets]], environment-scoped secrets
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
