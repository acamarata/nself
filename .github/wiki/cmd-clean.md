# nself clean

<!-- BEGIN PROSE:summary -->
> Remove Docker resources associated with the current ɳSelf project.
<!-- END PROSE:summary -->

## Synopsis

```
nself clean [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself clean` removes Docker resources (stopped containers, unused images, dangling volumes, orphaned networks) scoped to the current ɳSelf project. By default it uses Docker label filters to only remove resources belonging to the current project, other Docker projects on your machine are not affected.

`nself clean` does **not** delete your `.env` files, data directories, or any user data. It only removes generated Docker artifacts. To remove generated configuration files (nginx configs, `docker-compose.yml`), use `nself reset` instead.

Use `--all` for a system-wide Docker cleanup (`docker system prune`). This requires confirmation and affects all Docker resources on the host, not just the ɳSelf project.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
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
- Stopped ɳSelf project containers
- Unused images tagged to this project
- Dangling volumes (with `--all`: all Docker system resources)
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
