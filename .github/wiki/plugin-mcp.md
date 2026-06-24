# ɳSelf MCP Plugin

> Exposes your nSelf infrastructure as a set of MCP tools for Claude Code and other
> MCP-compliant IDEs. Schema introspection, migration history, plugin catalog, log
> streaming, and env reading — all accessible from inside your coding environment.

**Bundle:** ɳClaw · ClawDE | **Port (SSE mode):** 3825 | **Transport:** stdio (default) or HTTP/SSE | **License:** pro

---

## Installation

The MCP plugin requires an active ɳClaw or ClawDE bundle license.

```bash
# Set your license key (if not already set)
nself license set <your-license-key>

# Install the plugin
nself plugin install mcp

# Verify
nself plugin list | grep mcp
```

`nself plugin install mcp` downloads the binary from `ping.nself.org`, verifies the
checksum and license, and registers the plugin with your local nSelf stack. No manual
Docker setup is required for stdio mode.

---

## Claude Code Configuration

Add the following to your project's `.claude/settings.json` (or global
`~/.claude/settings.json` to enable for all projects):

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

Claude Code starts `nself mcp serve` automatically when it launches and connects over
stdio. No port or auth token is needed for stdio mode.

To verify the connection:

```bash
# Inside a Claude Code session, run:
nself_plugin_list

# Or from the terminal, list available tools:
nself mcp status
```

---

## VS Code / Cursor / Windsurf Configuration

For IDEs that support the MCP HTTP/SSE transport, start the server in SSE mode first:

```bash
# Start the SSE server with an auth token
NSELF_MCP_AUTH_TOKEN=<your-token> nself mcp serve --port 3825
```

Then add the following to your IDE's MCP config (the location varies by IDE):

```json
{
  "mcpServers": {
    "nself": {
      "url": "http://127.0.0.1:3825/mcp/message",
      "headers": {
        "Authorization": "Bearer <your-token>"
      }
    }
  }
}
```

For Cursor, place this in `.cursor/mcp.json` at your project root. For VS Code with
the MCP extension, add to `.vscode/mcp.json`.

---

## Auth Token Setup

Stdio mode (Claude Code) does not need auth — the stdio pipe is inherently private.

For SSE mode (browser IDEs, remote access), set an auth token:

```bash
# In your .env.dev (or pass via environment)
NSELF_MCP_AUTH_TOKEN=<a-long-random-string>
```

All SSE requests must include `Authorization: Bearer <token>` or the server returns 401.
Leave `NSELF_MCP_AUTH_TOKEN` empty to run SSE without auth (only do this on loopback).

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `NSELF_FEATURE_MCP` | `false` | Enable auto-start in `nself start`. Set `true` to start alongside the stack. |
| `NSELF_MCP_PORT` | `3825` | HTTP/SSE bind port |
| `NSELF_MCP_HOST` | `127.0.0.1` | HTTP/SSE bind host |
| `NSELF_MCP_AUTH_TOKEN` | *(empty)* | Bearer token for SSE auth. Empty = no auth (stdio only recommended). |
| `NSELF_MCP_MDNS_ENABLED` | `true` | Broadcast `_nself-mcp._tcp.local.` via mDNS for auto-discovery |
| `NSELF_MCP_ALLOW_MUTATIONS` | `false` | Enable `nself_migration_apply`. Must be set explicitly. |
| `NSELF_MCP_ALLOW_SECRETS` | `false` | Allow `nself_env_get` to return secret-patterned env vars |

---

## Tool Reference

The MCP server exposes 9 tools. All tools follow the JSON-RPC 2.0 protocol.

### `nself_schema_list`

Lists all tables in the PostgreSQL public schema of the nSelf project.

**Parameters:** none

**Example output:**
```
Tables in nSelf database:

  public.np_ai_usage
  public.np_cron_jobs
  public.np_notify_events
  ...

14 tables total.
```

---

### `nself_schema_describe`

Describes columns, types, nullability, defaults, and constraints for a single table.

**Parameters:**

| Name | Type | Required | Description |
|---|---|---|---|
| `table` | string | yes | Table name, e.g. `np_ai_usage` |

**Example:**
```json
{ "table": "np_ai_usage" }
```

---

### `nself_migration_list`

Lists recent migration history from the `nself_migrations` tracking table.

**Parameters:**

| Name | Type | Required | Description |
|---|---|---|---|
| `limit` | integer | no | Max rows to return (default 20, max 200) |

---

### `nself_migration_apply`

Applies a SQL migration. Requires `NSELF_MCP_ALLOW_MUTATIONS=true`.

A DDL allowlist is enforced — only safe statement types are permitted. Destructive
statements (`DROP TABLE`, `TRUNCATE`, `DELETE FROM` without WHERE, `GRANT`, `REVOKE`)
are blocked and return an error without executing. See [[cmd-mcp]] for the full allowlist.

**Parameters:**

| Name | Type | Required | Description |
|---|---|---|---|
| `sql` | string | yes | SQL to execute |
| `label` | string | yes | Human-readable migration label for the history table |

---

### `nself_permission_list`

Returns a snapshot of Hasura row and column permissions for a given role.

**Parameters:**

| Name | Type | Required | Description |
|---|---|---|---|
| `role` | string | yes | Hasura role name, e.g. `user`, `admin`, `anonymous` |

---

### `nself_log_tail`

Returns recent log lines from a running nSelf service container.

**Parameters:**

| Name | Type | Required | Description |
|---|---|---|---|
| `service` | string | yes | Service name, e.g. `hasura`, `postgres`, `nself-ai`, `nself-notify` |
| `lines` | integer | no | Number of lines (default 50, max 500) |

---

### `nself_plugin_list`

Lists all installed nSelf plugins with version, category, and license type.

**Parameters:** none

---

### `nself_plugin_describe`

Returns detail for a single installed plugin: tables, capabilities, port, license.

**Parameters:**

| Name | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Plugin name, e.g. `ai`, `notify`, `mcp` |

---

### `nself_env_get`

Reads a non-secret environment variable from the active nSelf config.

Keys containing `SECRET`, `KEY`, `PASSWORD`, `TOKEN`, or `CREDENTIAL` are redacted by
default. Set `NSELF_MCP_ALLOW_SECRETS=true` to disable redaction (use with caution).

**Parameters:**

| Name | Type | Required | Description |
|---|---|---|---|
| `key` | string | yes | Env var name, e.g. `NSELF_PROJECT_NAME` |

---

## mDNS Auto-Discovery

When `NSELF_MCP_MDNS_ENABLED=true` (the default), the MCP server advertises itself as
`_nself-mcp._tcp.local.` on the local network. MCP clients that support mDNS discovery
can find a running instance without manual configuration.

Disable mDNS in CI or headless environments:

```bash
nself mcp serve --no-mdns
```

---

## Docker / SSE Mode

The plugin ships a multi-arch Docker image (`nself/plugin-mcp`) for environments where
running a Docker container is preferred over a local binary.

```bash
# Pull latest
docker pull nself/plugin-mcp:latest

# Run in SSE mode (stdio mode is not useful inside a container)
docker run --rm \
  -e DATABASE_URL="postgres://..." \
  -e HASURA_GRAPHQL_ENDPOINT="http://host.docker.internal:8080" \
  -e HASURA_GRAPHQL_ADMIN_SECRET="..." \
  -e NSELF_MCP_AUTH_TOKEN="<token>" \
  -p 3825:3825 \
  nself/plugin-mcp:latest
```

---

## Troubleshooting

**`nself mcp status` shows the server is not running**

Start it manually:
```bash
nself mcp serve
```
Or set `NSELF_FEATURE_MCP=true` in `.env.dev` so `nself start` launches it automatically.

**Claude Code shows "MCP server failed to start"**

Check that the `nself` binary is on your PATH:
```bash
which nself && nself --version
```

Run the server manually and look for errors:
```bash
nself mcp serve --log-level debug
```

Common causes: `DATABASE_URL` not set, `HASURA_GRAPHQL_ADMIN_SECRET` missing, or the
nSelf stack is not running (`nself start` to start it).

**Tool calls return "database connection failed"**

Make sure the nSelf stack is running and `DATABASE_URL` resolves:
```bash
nself status
nself doctor
```

**Secret env vars are redacted**

This is the safe default. To allow reading secrets via `nself_env_get`:
```bash
NSELF_MCP_ALLOW_SECRETS=true nself mcp serve
```

**Port 3825 already in use**

Check what is using the port and change the MCP port:
```bash
lsof -i :3825
NSELF_MCP_PORT=3826 nself mcp serve --port 3826
```

---

## See Also

- [[cmd-mcp]] — full command reference including DDL allowlist detail
- [[plugin-claw]] — ɳClaw bundle overview
- [[plugin-clawde]] — ClawDE bundle overview
- [[cmd-plugin]] — manage the plugin catalog
- [[cmd-doctor]] — run the nSelf diagnostics suite

← [[Plugin-Overview]] | [[Home]] →
