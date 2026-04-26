# nself exec

> Execute a command inside a running service container.

## Synopsis

```
nself exec [flags] <SERVICE> [COMMAND...]
```

## Description

`nself exec` runs a command inside a running service container. It is a thin wrapper around `docker compose exec` with ɳSelf-aware defaults: if no command is given, it opens a sensible interactive shell for the service (`psql` for postgres, `redis-cli` for redis, `/bin/sh` for everything else).

The command supports piping, you can feed SQL files or other input through stdin using shell pipes. Allocate a pseudo-TTY by default for interactive sessions; pass `--no-tty` when piping input.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--user`, `-u` | `""` | Run the command as this user inside the container |
| `--workdir`, `-w` | `""` | Working directory inside the container |
| `--no-tty`, `-T` | false | Disable pseudo-TTY allocation |
| `--env`, `-e` | `[]` | Set environment variables (`KEY=VALUE`, repeatable) |
| `--help`, `-h` | — | Show help |

## Default Commands by Service

| Service | Default Command |
|---------|----------------|
| `postgres` | `psql -U postgres` |
| `redis` | `redis-cli` |
| (all others) | `/bin/sh` |

## Examples

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

← [[Commands]] | [[Home]] →
