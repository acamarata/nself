# nself plugin logs

> Tail logs from a plugin container.

## Synopsis

```
nself plugin logs <name> [flags]
```

## Description

`nself plugin logs` streams or prints log output from a named plugin container. It resolves the container name from the plugin name using the nSelf naming convention (`nself-plugin-<name>`) or via `docker compose ps` lookup.

The flags mirror `docker logs` for familiarity. Use `-f` to follow the stream, `--tail` to limit output to the last N lines, `--since` to filter by time window, and `--grep` to filter by a regex pattern applied to each log line.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--follow`, `-f` | false | Follow log output (stream continuously) |
| `--tail` | `0` | Number of lines to show from the end of the log (0 = all) |
| `--since` | `""` | Show logs since duration (e.g. `1h`, `30m`, `5s`) |
| `--grep` | `""` | Regex pattern to filter log lines |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Show all logs from a plugin container
nself plugin logs my-plugin
```

```bash
# Follow logs in real time
nself plugin logs my-plugin -f
```

```bash
# Show the last 50 lines
nself plugin logs my-plugin --tail 50
```

```bash
# Filter logs from the past 10 minutes
nself plugin logs my-plugin --since 10m
```

```bash
# Show only error lines
nself plugin logs my-plugin --grep "ERROR"
```

```bash
# Follow and filter simultaneously
nself plugin logs my-plugin -f --grep "panic"
```

## See Also

- [[cmd-plugin-debug]] — attach a Delve debugger to a running plugin
- [[cmd-plugin-status]] — show plugin container state
- [[cmd-plugin]] — plugin command overview
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
