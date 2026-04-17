# nself logs

> View and filter service logs with color and formatting.

## Synopsis

```
nself logs [SERVICE] [flags]
```

## Description

`nself logs` streams or displays logs from one or all nSelf services. Without a service name, it interleaves logs from all running services. Pass a service name as a positional argument to filter to that service. Use `--service` (repeatable) to target multiple specific services.

Output is colorized by level: errors in red, warnings in yellow, info in green. Use `--follow` for live streaming; press Ctrl+C to exit cleanly. Use `--quiet` to suppress healthcheck noise and focus on application-level events. Use `--plain` or redirect stdout to a file when piping output to other tools.

The `--status`, `--summary`, and `--top` flags provide aggregate views without raw log output, useful for quick operational checks.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--follow`, `-f` | false | Follow log output (live streaming) |
| `--tail`, `-n` | `10` | Number of lines to show |
| `--more` | false | Show last 50 lines |
| `--all` | false | Show last 100 lines |
| `--since` | `""` | Show logs since a time (e.g. `1h`, `30m`, or RFC3339) |
| `--until` | `""` | Show logs until a time (e.g. `1h` or RFC3339) |
| `--errors`, `-e` | false | Show only error lines |
| `--level`, `-l` | `""` | Filter by minimum level: `debug`, `info`, `warn`, `error` |
| `--search`, `-s` | `""` | Substring search pattern (case-insensitive) |
| `--grep` | `""` | Regex pattern filter |
| `--service`, `-S` | `[]` | Filter by service name; repeatable for multiple services |
| `--json` | false | Structured JSON output per line |
| `--compact`, `-c` | false | Compact output: `[service] message` |
| `--quiet`, `-q` | false | Filter out healthcheck noise |
| `--plain` | false | Plain output with no highlighting, for piping |
| `--no-color` | false | Disable colored output |
| `--status` | false | Show service status overview |
| `--summary` | false | Show recent errors by service |
| `--top` | false | Show most active services |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Last 10 lines from all services
nself logs
```

```bash
# Follow all logs live
nself logs -f
```

```bash
# Last 10 lines from a single service
nself logs hasura
```

```bash
# Follow postgres logs
nself logs -f postgres
```

```bash
# Show last 100 lines
nself logs --all
```

```bash
# Only error lines
nself logs -e
```

```bash
# Logs from the last hour
nself logs --since 1h
```

```bash
# Logs between one hour ago and thirty minutes ago
nself logs --since 1h --until 30m
```

```bash
# Filter to two specific services
nself logs -S hasura -S postgres
```

```bash
# Structured JSON output for log aggregation
nself logs --json
```

```bash
# Regex filter for timeout errors
nself logs --grep "error.*timeout"
```

```bash
# Plain output for piping to grep
nself logs --plain | grep migration
```

```bash
# Quick status overview
nself logs --status
```

```bash
# Recent errors by service
nself logs --summary
```

```bash
# Most active services
nself logs --top
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | No nSelf project found in current directory or parents |
| `1` | No containers running (stack not started) |
| `1` | Docker compose error |

## See Also

- [[cmd-status]] — check stack health and service states
- [[cmd-health]] — run health checks against all services
- [[cmd-restart]] — restart one or more services
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
