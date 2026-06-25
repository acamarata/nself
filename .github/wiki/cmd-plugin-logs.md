# nself plugin logs

> Tail logs from a plugin container.

## Synopsis

```bash
nself plugin logs <name> [flags]
```

## Description

`nself plugin logs` streams or displays log output from a running plugin container. By default it prints all available logs; use `--follow` to stream continuously like `tail -f`.

Use `--tail` to limit output to the most recent N lines. Use `--since` to show logs from a relative time window. Use `--grep` to filter lines by a regex pattern.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-f`, `--follow` | `false` | Follow log output (like tail -f) |
| `--tail` | `0` | Number of lines to show from the end (0 = all) |
| `--since` | `""` | Show logs since duration (e.g. `1h`, `30m`, `5s`) |
| `--grep` | `""` | Regex pattern to filter log lines |

## Examples

```bash
# Show all logs for the ai plugin
nself plugin logs ai

# Follow logs in real time
nself plugin logs ai -f

# Show last 100 lines
nself plugin logs ai --tail 100

# Show logs from the last 30 minutes
nself plugin logs ai --since 30m

# Filter for error lines
nself plugin logs ai --grep "ERROR|WARN"

# Follow and filter together
nself plugin logs ai -f --grep "request"
```

## See Also

- [[cmd-plugin-debug]] — Attach a debugger to a running plugin
- [[cmd-plugin-status]] — Show plugin status
- [[cmd-plugin-dev]] — Start a plugin in development mode

← [[Commands]] | [[Home]] →
