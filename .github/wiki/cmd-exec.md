# nself exec

<!-- BEGIN PROSE:summary -->
> Execute a command inside a running service container.
<!-- END PROSE:summary -->

## Synopsis

```
nself exec [FLAGS] <SERVICE> [COMMAND...] [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself exec` runs a command inside a running service container. It is a thin wrapper around `docker compose exec` with ɳSelf-aware defaults: if no command is given, it opens a sensible interactive shell for the service (`psql` for postgres, `redis-cli` for redis, `/bin/sh` for everything else).

The command supports piping, you can feed SQL files or other input through stdin using shell pipes. Allocate a pseudo-TTY by default for interactive sessions; pass `--no-tty` when piping input.

## Default Commands by Service

| Service | Default Command |
|---------|----------------|
| `postgres` | `psql -U postgres` |
| `redis` | `redis-cli` |
| (all others) | `/bin/sh` |
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--env`, `-e` | `—` | Set environment variables (KEY=VALUE) |
| `--no-tty`, `-T` | `false` | Disable pseudo-TTY allocation |
| `--user`, `-u` | `""` | Run command as this user |
| `--workdir`, `-w` | `""` | Working directory inside the container |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Open an interactive psql session
nself exec postgres

# Open redis-cli
nself exec redis

# Open a shell in the hasura container
nself exec hasura

# Run a specific command
nself exec postgres -- pg_dump mydb

# Run as root in nginx
nself exec -u root nginx /bin/bash

# Pipe a SQL file into psql (disable TTY for piping)
cat schema.sql | nself exec -T postgres psql

# Set environment variables
nself exec -e DEBUG=true functions /bin/sh

# Use a specific working directory
nself exec -w /app functions /bin/sh
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
