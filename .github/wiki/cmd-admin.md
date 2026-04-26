# nself admin

> Manage the ɳSelf Admin UI: open, start, stop, inspect logs, or health-check.

## Synopsis

```
nself admin [subcommand] [flags]
```

## Description

`nself admin` manages the local ɳSelf Admin dashboard at `http://localhost:3021`. With no subcommand, it opens the dashboard in your default browser. Use subcommands to control the Admin container lifecycle directly.

The Admin UI is a local-only web interface (Docker container) on your machine. It is not a hosted service.

## Subcommands

| Subcommand | Description |
|------------|-------------|
| _(none)_ | Open the Admin UI in your default browser |
| `start` | Start the Admin service (enables, builds, boots if not running) |
| `stop` | Stop the Admin container gracefully |
| `logs` | Tail Admin container logs |
| `health` | Check Admin liveness via HTTP probe on `/health` |

## Flags

### `stop`

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | false | Skip graceful drain and force-stop |

### `logs`

| Flag | Default | Description |
|------|---------|-------------|
| `--follow` | false | Stream logs continuously |
| `--tail` | 100 | Number of recent lines to show |

## Examples

```bash
# Open the Admin UI in your browser
nself admin

# Start the Admin service (idempotent)
nself admin start

# Stop the Admin container
nself admin stop

# Force-stop immediately
nself admin stop --force

# Tail the last 50 log lines
nself admin logs --tail 50

# Stream logs continuously
nself admin logs --follow

# Check Admin health
nself admin health
```

← [[Commands]] | [[Home]] →
