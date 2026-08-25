# nself stop

<!-- BEGIN PROSE:summary -->
> Gracefully shut down all services or specific named services.
<!-- END PROSE:summary -->

## Synopsis

```
nself stop [SERVICES...] [flags]
```

**Alias:** `nself down`

## Description

<!-- BEGIN PROSE:description -->
`nself stop` shuts down the ɳSelf stack gracefully. By default it sends SIGTERM to running containers and waits for them to exit cleanly before running `docker compose down`. The graceful shutdown timeout is 30 seconds by default and can be adjusted with `--graceful`.

You can stop specific services by passing their names as positional arguments. When stopping individual services, only those containers are affected, the rest of the stack remains running.

Data is always preserved. Volumes are only removed when `--volumes` is explicitly passed (this is destructive and will delete your database and all stored data). When `--volumes` or `--rmi` is set, the graceful stop phase is skipped for speed.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--graceful` | `30` | Graceful shutdown timeout in seconds |
| `--no-monorepo` | `false` | Disable automatic monorepo backend detection |
| `--remove-images` | `false` | Alias for --rmi |
| `--remove-orphans` | `false` | Remove orphaned containers |
| `--rmi` | `false` | Remove Docker images |
| `--verbose` | `false` | Show detailed output |
| `--volumes`, `-v` | `false` | Remove volumes (WARNING: deletes data) |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Stop all services gracefully
nself stop

# Using the alias
nself down

# Stop only specific services
nself stop postgres redis

# Stop and remove volumes (destroys all data)
nself stop --volumes

# Stop and remove Docker images
nself stop --rmi

# Clean up orphaned containers
nself stop --remove-orphans

# Extended graceful shutdown timeout
nself stop --graceful 60
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
