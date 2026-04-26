# nself stop

> Gracefully shut down all services or specific named services.

## Synopsis

```
nself stop [SERVICES...] [flags]
nself down [SERVICES...] [flags]
```

**Alias:** `down`

## Description

`nself stop` shuts down the ɳSelf stack gracefully. By default it sends SIGTERM to running containers and waits for them to exit cleanly before running `docker compose down`. The graceful shutdown timeout is 30 seconds by default and can be adjusted with `--graceful`.

You can stop specific services by passing their names as positional arguments. When stopping individual services, only those containers are affected, the rest of the stack remains running.

Data is always preserved. Volumes are only removed when `--volumes` is explicitly passed (this is destructive and will delete your database and all stored data). When `--volumes` or `--rmi` is set, the graceful stop phase is skipped for speed.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--graceful` | `30` | Graceful shutdown timeout in seconds |
| `--volumes`, `-v` | false | Remove volumes , **WARNING: deletes all data** |
| `--rmi` | false | Remove Docker images after stopping |
| `--remove-images` | false | Alias for `--rmi` |
| `--remove-orphans` | false | Remove orphaned containers |
| `--no-monorepo` | false | Disable automatic monorepo backend detection |
| `--verbose` | false | Show detailed output |
| `--help`, `-h` | — | Show help |

## Examples

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

← [[Commands]] | [[Home]] →
