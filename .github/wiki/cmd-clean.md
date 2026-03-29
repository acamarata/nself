# nself clean

> Remove Docker resources associated with the current nSelf project.

## Synopsis

```
nself clean [flags]
```

## Description

`nself clean` removes Docker resources (stopped containers, unused images, dangling volumes, orphaned networks) scoped to the current nSelf project. By default it uses Docker label filters to only remove resources belonging to the current project — other Docker projects on your machine are not affected.

`nself clean` does **not** delete your `.env` files, data directories, or any user data. It only removes generated Docker artifacts. To remove generated configuration files (nginx configs, `docker-compose.yml`), use `nself reset` instead.

Use `--all` for a system-wide Docker cleanup (`docker system prune`). This requires confirmation and affects all Docker resources on the host, not just the nSelf project.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--all` | false | System-wide Docker cleanup (requires confirmation) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Project-scoped cleanup (safe, recommended)
nself clean

# System-wide Docker cleanup (affects all projects)
nself clean --all
```

**What is preserved:**
- `.env` and all `.env.*` variant files
- `data/` and `volumes/` directories
- Your application source code

**What is removed:**
- Stopped nSelf project containers
- Unused images tagged to this project
- Dangling volumes (with `--all`: all Docker system resources)

← [[Commands]] | [[Home]] →
