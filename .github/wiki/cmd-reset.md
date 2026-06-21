# nself reset

> Stop containers, remove all data volumes, and clean generated files.

## Synopsis

```
nself reset [flags]
```

## Description

`nself reset` stops all running containers, removes Docker volumes (deleting all project data: database, storage, Redis), and removes generated files (`docker-compose.yml`, nginx site configs, SSL certificates, `.env.computed`). Your `.env` file is always preserved.

This operation is destructive and irreversible. All project data is deleted. Use `--yes` to skip the interactive confirmation prompt in CI/CD pipelines.

After `nself reset`, run `nself init` (if you want to reconfigure) or `nself build && nself start` to bring the stack back up with a clean state.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | false | Skip confirmation prompt (for CI/CD) |
| `--no-monorepo` | false | Disable automatic monorepo backend detection |
| `--help`, `-h` | — | Show help |

## What is Removed vs Preserved

| Item | Result |
|------|--------|
| Running containers | Stopped and removed |
| Docker volumes (database, storage, Redis) | Removed (all data deleted) |
| `docker-compose.yml` | Removed |
| `nginx/sites/` configs | Removed |
| `ssl/` certificate files | Removed |
| `.env.computed` | Removed |
| `.env` and all `.env.*` variants | **Preserved** |

## Examples

```bash
# Interactive reset — prompts for confirmation
nself reset

# Non-interactive reset for CI/CD pipelines
nself reset --yes

# After reset, rebuild and restart with a clean slate
nself reset --yes && nself build && nself start
```

← [[Commands]] | [[Home]] →
