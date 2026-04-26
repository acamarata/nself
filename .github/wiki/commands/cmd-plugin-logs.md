# nself plugin logs

Tail or print logs from a plugin container.

## Synopsis

```bash
nself plugin logs <name> [flags]
```

## Description

Wraps `docker logs` with plugin-aware container name resolution. Resolves the container
from the plugin name using the ɳSelf naming convention (`nself-plugin-<name>`) then falls
back to a `docker compose ps` lookup.

Supports live streaming, line limits, time filters, and regex filtering.

## Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--follow` | `-f` | false | Follow log output (live stream) |
| `--tail` | | 100 | Number of lines from the end (0 = all; default 100 when not following) |
| `--since` | | | Show logs since duration (e.g. `1h`, `30m`, `5s`) |
| `--grep` | | | Regex pattern to filter log lines |

## Examples

```bash
# Show last 100 lines (default)
nself plugin logs myplugin

# Live stream
nself plugin logs myplugin --follow
nself plugin logs myplugin -f

# Last 50 lines
nself plugin logs myplugin --tail 50

# Logs from the last 30 minutes
nself plugin logs myplugin --since 30m

# Filter for errors
nself plugin logs myplugin --grep "ERROR"

# Live stream + filter
nself plugin logs myplugin -f --grep "ERROR|WARN"
```

## Container Resolution

The command tries these names in order until one matches:

1. `nself-plugin-<name>`
2. `nself_plugin_<name>`
3. `<name>` (bare name)
4. `docker compose ps` lookup (scans service names for a match)

If no container is found, the command exits with a descriptive error.

## Production Debugging

For production log analysis, use the Grafana/Loki stack included in the ɳSelf monitoring
bundle. Equivalent Loki query:

```logql
{container=~"nself-plugin-myplugin"} |= "ERROR"
```

See [Observability guide](https://docs.nself.org/observability) for the Grafana dashboard.

## See Also

- [[cmd-plugin-dev]], hot-reload watcher
- [[cmd-plugin-debug]], attach Delve debugger
- [[cmd-plugin-status]], check plugin runtime state
- [[Home]]
