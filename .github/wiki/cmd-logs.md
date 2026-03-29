# nself logs

> View and filter service logs with color and formatting.

## Synopsis

```
nself logs [SERVICE] [flags]
```

## Description

`nself logs` streams or displays logs from one or all nSelf services. Without a service name, it interleaves logs from all running services. Pass a service name to filter to a single service. Plugin service logs are read from `~/.nself/runtime/logs/{name}.log`.

Output is colorized by level: errors in red, warnings in yellow, info in green. Duplicate timestamps are stripped. Use `--follow` for live streaming — press Ctrl+C to exit cleanly. Use `--quiet` to suppress healthcheck noise and focus on application-level events.

The `--status`, `--summary`, and `--top` flags provide aggregate views without raw log output, useful for quick operational checks.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--follow`, `-f` | false | Follow log output (live streaming) |
| `--tail`, `-n` | `10` | Number of lines to show |
| `--more` | false | Show last 50 lines |
| `--all` | false | Show last 100 lines |
| `--errors`, `-e` | false | Show only error lines |
| `--level`, `-l` | `""` | Filter by level: `error`, `warn` |
| `--search`, `-s` | `""` | Search pattern (regex) |
| `--compact`, `-c` | false | Compact output: `[service] message` |
| `--quiet`, `-q` | false | Filter out healthcheck noise |
| `--status` | false | Show service status overview |
| `--summary` | false | Show recent errors by service |
| `--top` | false | Show most active services |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Last 10 lines from all services
nself logs

# Follow all logs live
nself logs -f

# Last 10 lines from hasura
nself logs hasura

# Follow postgres logs
nself logs -f postgres

# Show last 50 lines
nself logs --more

# Show last 100 lines
nself logs --all

# Only error lines
nself logs -e

# Search for a pattern (regex)
nself logs -s "migration"

# Live follow with error filter
nself logs -f -e

# Compact output for busy logs
nself logs --compact

# Quick status overview
nself logs --status

# Recent errors by service
nself logs --summary

# Most active services
nself logs --top
```

← [[Commands]] | [[Home]] →
