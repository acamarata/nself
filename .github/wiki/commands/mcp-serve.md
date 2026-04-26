# nself mcp serve

Start a Model Context Protocol (MCP) server for the local ɳSelf instance. Enables Claude Code and other MCP-compliant IDEs to introspect your ɳSelf project with zero configuration.

## Synopsis

```
nself mcp serve [--port <port>] [--host <host>] [--log-level <level>]
nself mcp status
```

## Description

`nself mcp serve` starts the MCP server in **stdio mode** by default, the native transport that Claude Code expects. When `--port` is supplied (or `NSELF_MCP_PORT` is set), it also starts an HTTP/SSE transport for web IDEs and API access.

On startup the server:

1. Validates `DATABASE_URL` and `HASURA_GRAPHQL_ADMIN_SECRET` are present.
2. Registers all 9 MCP tools against the tool registry.
3. Prints `MCP server ready on stdio` to stderr.
4. Optionally broadcasts `_nself-mcp._tcp.local.` via mDNS for auto-discovery.

## Options

| Flag | Default | Description |
|---|---|---|
| `--port` | `0` (stdio only) | HTTP/SSE port. Set to `3825` for web IDE support. |
| `--host` | `127.0.0.1` | HTTP/SSE bind host. |
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error`. Logs go to stderr. |

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `NSELF_MCP_PORT` | `3825` | HTTP/SSE port |
| `NSELF_MCP_HOST` | `127.0.0.1` | HTTP/SSE bind host |
| `NSELF_MCP_AUTH_TOKEN` | (empty) | Optional bearer token for SSE requests |
| `NSELF_MCP_MDNS_ENABLED` | `true` | Broadcast via mDNS |
| `NSELF_MCP_ALLOW_MUTATIONS` | `false` | Enable `nself_migration_apply` tool |
| `NSELF_MCP_ALLOW_SECRETS` | `false` | Allow `nself_env_get` to return secret keys |
| `NSELF_FEATURE_MCP` | `false` | Feature flag , set to `true` to enable auto-start with `nself start` |

## Claude Code integration

Add to `.claude/settings.json` in your project root:

```json
{
  "mcpServers": {
    "nself": {
      "command": "nself",
      "args": ["mcp", "serve"]
    }
  }
}
```

Claude Code will start the server automatically and expose all 9 tools in its tool context. You can then ask Claude Code questions like:

- "What tables does this ɳSelf project have?"
- "Show me the schema for np_ai_usage"
- "Which plugins are installed?"
- "Tail the hasura logs"
- "What permissions does the 'user' role have?"

## Available MCP tools

| Tool | Description |
|---|---|
| `nself_schema_list` | List all PostgreSQL tables in the project |
| `nself_schema_describe` | Describe columns, types, and constraints for a table |
| `nself_migration_list` | List migration history from `nself_migrations` |
| `nself_migration_apply` | Apply a SQL migration (requires `NSELF_MCP_ALLOW_MUTATIONS=true`) |
| `nself_permission_list` | List Hasura permissions for a Hasura role |
| `nself_log_tail` | Tail recent logs from a Docker service container |
| `nself_plugin_list` | List installed ɳSelf plugins |
| `nself_plugin_describe` | Full details for a specific plugin |
| `nself_env_get` | Read an env var (secret keys redacted unless `NSELF_MCP_ALLOW_SECRETS=true`) |

## Security

- HTTP/SSE transport binds to `127.0.0.1` by default. Never expose to `0.0.0.0` in production.
- Secret env vars (`*SECRET*`, `*KEY*`, `*TOKEN*`, `*PASSWORD*`) are redacted by default.
- `nself_migration_apply` requires explicit opt-in. Treat it as a privileged operation.
- mDNS operates on the local network segment only.

## Status check

```bash
nself mcp status
```

Reports the configured port, whether it is in use (server likely running), mDNS status, and mutation/secret flags.

## Doctor integration

`nself doctor` reports MCP server status:

- `MCP-01`, MCP server process running (info)
- `MCP-02`, Port 3825 not in conflict (warning)

## Examples

```bash
# Stdio only (Claude Code uses this)
nself mcp serve

# With HTTP/SSE on the default port
nself mcp serve --port 3825

# Debug logging
nself mcp serve --log-level debug

# Check if running
nself mcp status

# Curl the tool list (requires HTTP/SSE mode)
curl http://localhost:3825/mcp/tools/list

# Curl a tool call
curl -X POST http://localhost:3825/mcp/tools/call \
  -H 'Content-Type: application/json' \
  -d '{"name":"nself_schema_list","arguments":{}}'
```

## Plugin requirement

Requires the `mcp` plugin (`plugins-pro/paid/mcp`). Install:

```bash
nself license set nself_pro_...
nself plugin install mcp
```

## See also

- [Plugin documentation](https://nself.org/plugins/mcp)
- [MCP specification](https://modelcontextprotocol.io/specification/2025-03-26)
- [nself doctor](./cmd-doctor.md)
