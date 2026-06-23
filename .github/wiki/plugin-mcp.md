# nself-mcp Plugin

Model Context Protocol (MCP) server for nSelf. Exposes your nSelf instance to Claude Code and any MCP-compliant IDE with zero configuration. Integrates nSelf schema, migrations, permissions, logs, plugins, and environment variables into Claude Code.

**License:** Source-Available  
**Tier:** Pro  
**Port:** 3825 (HTTP/SSE transport)  
**Entry point:** `nself mcp serve`

## Overview

The nself-mcp plugin is an MCP (Model Context Protocol) server that provides Claude Code and other MCP-compliant IDEs with deep context about your nSelf deployment:

- Full database schema introspection (tables, columns, types)
- Migration history and schema evolution tracking
- Hasura permissions and row-level security (RLS) configuration
- Service logs streaming (container logs from all nSelf services)
- Installed plugin inventory and metadata
- Environment variable reading (with secret redaction by default)

This enables AI assistants to understand your full nSelf context and provide schema-aware suggestions without manual setup.

## Installation

```bash
# Set your Pro license
nself license set nself_pro_...

# Install the MCP plugin
nself plugin install mcp
```

The plugin runs locally on port 3825 and communicates via stdio (for Claude Code) or HTTP/Server-Sent Events (SSE) for other IDEs.

## Quick Start

### Claude Code Integration (Recommended)

Add to your project's `.claude/settings.json`:

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

Claude Code will auto-discover and initialize the connection on startup. You'll have full nSelf schema and context access within the same message.

### Stdio Mode (Default)

Start the MCP server in stdio mode for direct IDE integration:

```bash
nself mcp serve
```

This is the recommended mode for Claude Code. The server reads JSON-RPC 2.0 requests from stdin and writes responses to stdout.

### HTTP/SSE Mode

For HTTP/Server-Sent Events mode (useful for web-based IDEs or remote connections):

```bash
# Start on default port 3825
nself mcp serve --port 3825

# Start on a custom port
NSELF_MCP_PORT=3900 nself mcp serve

# With optional bearer token authentication
NSELF_MCP_AUTH_TOKEN="your-secret-token" nself mcp serve
```

The server will listen on `127.0.0.1:3825` by default and expose the following HTTP endpoints:

- `GET /mcp/health` — Health check
- `POST /mcp/message` — JSON-RPC 2.0 messages
- `GET /mcp/tools/list` — List available tools
- `POST /mcp/tools/call` — Call a specific tool

### Status Check

```bash
nself mcp status
```

Displays whether the MCP server is running, its PID, and the port it's listening on.

## Tools

The MCP plugin exposes the following tools via the MCP protocol:

### Schema Introspection

| Tool | Description | Input |
|------|-------------|-------|
| `nself_schema_list` | List all tables in the database | None |
| `nself_schema_describe` | Describe a specific table (columns, types, constraints) | `table_name: string` |

### Migrations & Schema Evolution

| Tool | Description | Input |
|------|-------------|-------|
| `nself_migration_list` | List all applied migrations with timestamps | None |
| `nself_migration_apply` | Apply a SQL migration to the database | `migration_sql: string` (requires `NSELF_MCP_ALLOW_MUTATIONS=true`) |

### Permissions & Access Control

| Tool | Description | Input |
|------|-------------|-------|
| `nself_permission_list` | List Hasura permissions for a role | `role: string, table: string` |

### Logs & Monitoring

| Tool | Description | Input |
|------|-------------|-------|
| `nself_log_tail` | Stream recent logs from a service container | `service: string, lines?: number` |

### Plugins & Extensions

| Tool | Description | Input |
|------|-------------|-------|
| `nself_plugin_list` | List all installed plugins with versions | None |
| `nself_plugin_describe` | Describe a plugin (metadata, tools, env vars) | `plugin_name: string` |

### Environment & Configuration

| Tool | Description | Input |
|------|-------------|-------|
| `nself_env_get` | Read an environment variable | `key: string` |

Environment variables matching `*SECRET*`, `*KEY*`, `*TOKEN*`, `*PASSWORD*` are automatically redacted by default unless `NSELF_MCP_ALLOW_SECRETS=true`.

## Configuration

The nself-mcp plugin is configured via environment variables:

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `NSELF_MCP_PORT` | `3825` | No | HTTP/SSE transport port |
| `NSELF_MCP_HOST` | `127.0.0.1` | No | HTTP/SSE bind host (localhost only by default) |
| `NSELF_MCP_AUTH_TOKEN` | (empty) | No | Optional bearer token for SSE mode (empty = no auth, stdio recommended) |
| `NSELF_MCP_MDNS_ENABLED` | `true` | No | Broadcast `_nself-mcp._tcp.local.` via mDNS for auto-discovery |
| `NSELF_MCP_ALLOW_MUTATIONS` | `false` | No | Allow the `nself_migration_apply` tool (dangerous — requires explicit opt-in) |
| `NSELF_MCP_ALLOW_SECRETS` | `false` | No | Allow `nself_env_get` to return secret env vars (off by default for security) |
| `NSELF_FEATURE_MCP` | `false` | No | Feature flag to enable MCP server (default off in v1.0.x) |

## Security

The nself-mcp plugin follows nSelf's Security-Always-Free doctrine. Key security features:

- **Local-only by default:** Binds to `127.0.0.1` and is never exposed externally. Access over HTTP/SSE requires conscious host binding and optional bearer token.
- **Secret redaction:** Any environment variable matching `*SECRET*`, `*KEY*`, `*TOKEN*`, or `*PASSWORD*` is automatically redacted from `nself_env_get` output unless explicitly enabled with `NSELF_MCP_ALLOW_SECRETS=true`.
- **Mutation gatekeeping:** The `nself_migration_apply` tool is disabled by default. Migrations can only be applied if `NSELF_MCP_ALLOW_MUTATIONS=true` is explicitly set.
- **Stdio recommended:** For Claude Code integration, use stdio mode (the default `nself mcp serve`). This is the safest and requires no network exposure.
- **Optional bearer token:** HTTP/SSE mode supports optional bearer-token authentication via `NSELF_MCP_AUTH_TOKEN`. Set a strong token if exposing over the network.

## Requires

- nSelf CLI v1.0.9 or later
- Claude Code or an MCP-compliant IDE
- A valid nSelf Pro license

## Port Registry

The nself-mcp plugin uses **port 3825**, which is reserved in the nSelf port registry (block 3820–3829 for MCP-related services).

## Bundles

The nself-mcp plugin is included in:

- **ɳClaw bundle** (Pro tier) — for ɳClaw + ɳClawDE users
- **ɳSelf+ subscription** — all-access tier with all plugins and apps

## See Also

- [MCP (Model Context Protocol) Specification](https://modelcontextprotocol.io/)
- [Claude Code Documentation](https://claude.ai/docs/claude-code)
- [nself CLI Reference](./commands.md)
- [nSelf Permissions & RLS](../../Architecture.md#permissions-and-row-level-security)

## License

Source-Available. Requires a valid nSelf Pro license or ɳSelf+ subscription.
