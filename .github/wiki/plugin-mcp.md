# plugin-mcp — MCP Server for nSelf (ɳClaw + ClawDE)

## Overview

`plugin-mcp` exposes your nSelf instance as a Model Context Protocol (MCP) server.
Connect Claude Code, Cursor, Windsurf, or any MCP-compliant IDE to your nSelf backend
with zero configuration — no API keys, no env setup.

**Port**: 3825 | **Bundle**: ɳClaw + ClawDE | **License**: Required

## Why Use This?

- **AI assistants query your live database** — Claude Code can run `nself_query` to inspect your PostgreSQL schema and data
- **Live nSelf operations** — install plugins, check health, query logs from inside your IDE
- **Zero config** — MCP auto-discovered via `nself.json` or stdio transport
- **Security audit** — all tool calls logged; SQL injection payloads detected and blocked

## Installation

```bash
nself plugin install mcp
```

## Configuration in Claude Code

Add to `.claude/settings.json`:

```json
{
  "mcpServers": {
    "nself": {
      "command": "nself",
      "args": ["mcp", "--stdio"]
    }
  }
}
```

Or run as HTTP server:

```bash
nself plugin install mcp
# Plugin starts on port 3825 automatically
```

## Available MCP Tools

| Tool | Description |
|------|-------------|
| `nself_health` | Check health of all installed plugins |
| `nself_plugins_list` | List installed plugins with versions and status |
| `nself_plugin_install` | Install a plugin by name |
| `nself_plugin_uninstall` | Uninstall a plugin |
| `nself_query` | Execute read-only SQL against the nSelf database |
| `nself_schema_list` | List all PostgreSQL tables (np_* prefix) |
| `nself_schema_describe` | Describe columns of a specific table |
| `nself_logs` | Tail recent plugin logs |
| `nself_config_get` | Get a config value |
| `nself_config_set` | Set a config value |

## Security

- **SQL injection protection** — `nself_query` rejects payloads containing DDL (`DROP`, `TRUNCATE`) and common injection patterns
- **Read-only queries** — `nself_query` only accepts `SELECT` statements
- **License gate** — all tool calls require a valid ɳClaw or ɳSelf+ license
- **Audit log** — every tool call logged with caller identity and result

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL DSN |
| `NSELF_LICENSE_KEY` | Yes | — | ɳClaw or ɳSelf+ license key |
| `PORT` | No | `3825` | HTTP listen port |
| `MCP_TRANSPORT` | No | `http` | `http` or `stdio` |

## Example: Using in Claude Code

Once configured, Claude Code can:

```
> What tables does this nSelf instance have?
→ (calls nself_schema_list)
→ Returns: np_ai_usage, np_cron_jobs, np_voice_sessions, ...

> Show me the last 10 cron job failures
→ (calls nself_query with SELECT)
→ Returns: structured result rows
```

## Docker

```bash
docker run -p 3825:3825 \
  -e DATABASE_URL=postgres://... \
  -e NSELF_LICENSE_KEY=... \
  nself/plugin-mcp:latest
```

## Hasura Integration

The MCP plugin itself has no `np_*` tables (it is a tool connector, not a data store).
It reads from other plugins' tables via `nself_query` under the read-only constraint.

## Changelog

- **1.0.0** — Initial release (ɳClaw + ClawDE, port 3825, 10 MCP tools)
- **1.1.0** — SQL injection blocking, audit log, stdio transport
