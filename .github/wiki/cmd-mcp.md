# nself mcp

> Start the nSelf MCP server and expose infrastructure tools to Claude Code and other MCP clients.

## Synopsis

```
nself mcp [flags]
```

## Description

`nself mcp` starts a [Model Context Protocol](https://modelcontextprotocol.io) server that
exposes nSelf infrastructure operations as tools. MCP clients (including Claude Code) can
then call those tools directly, without leaving their coding environment.

By default the server runs over `stdio`, which is the correct mode for Claude Code's
`mcpServers` configuration. Use `--transport sse` to expose the server over HTTP on a local
port instead.

The server provides six tools:

| Tool | Description |
|------|-------------|
| `nself_list_plugins` | List the plugin catalog (installed and available plugins) |
| `nself_get_schema` | Hasura GraphQL schema introspection |
| `nself_get_permissions` | Hasura role permissions snapshot |
| `nself_run_migration` | Apply a SQL migration (requires `confirm: true`; DDL allowlist enforced — see below) |
| `nself_tail_logs` | Tail Docker logs for a named service or plugin container |
| `nself_doctor` | Run `nself doctor --deep` and return the diagnostic report |

The server advertises itself via mDNS as `_nself._tcp.local` so Claude Code and other
mDNS-aware clients can discover a running instance on the local network automatically.
Pass `--no-mdns` to suppress this.

`nself mcp` must be run from inside a directory that contains an nSelf project (i.e. one
initialised by `nself init`). The server exits immediately if no project is found.

## Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--transport` | `-t` | string | `stdio` | Transport mode: `stdio` or `sse` |
| `--port` | `-p` | int | `3825` | Port for SSE transport |
| `--no-mdns` | | bool | `false` | Disable mDNS service advertising |

## Examples

```bash
# Run as a Claude Code MCP server using stdio (recommended)
nself mcp
```

```bash
# Run as an SSE server for browser-based clients
nself mcp --transport sse
```

```bash
# Run SSE on a custom port
nself mcp --transport sse --port 4000
```

```bash
# Run without mDNS advertising (e.g. in CI)
nself mcp --no-mdns
```

```bash
# Run SSE on a custom port without mDNS
nself mcp --transport sse --port 4000 --no-mdns
```

### Claude Code configuration

Add the following to your project's `.claude/settings.json`:

```json
{
  "mcpServers": {
    "nself": {
      "command": "nself",
      "args": ["mcp"]
    }
  }
}
```

Claude Code will start `nself mcp` automatically and connect over stdio.

## nself_run_migration DDL Allowlist

The `nself_run_migration` tool enforces a DDL allowlist before executing any SQL. This
prevents AI Studio sessions or automated agents from running destructive statements via
`confirm: true` set programmatically.

Permitted statement types:

- `CREATE TABLE IF NOT EXISTS`
- `ALTER TABLE ADD COLUMN`
- `CREATE INDEX`
- `CREATE EXTENSION`
- `CREATE POLICY`
- `INSERT` with an explicit column list
- `UPDATE` with a `WHERE` clause
- `DROP POLICY`

Blocked statement types (return an error without executing):

| Blocked prefix | Why |
|----------------|-----|
| `DROP TABLE` | Irreversible data loss |
| `DROP DATABASE` | Irreversible data loss |
| `DROP SCHEMA` | Irreversible data loss |
| `TRUNCATE` | Irreversible data removal |
| `DELETE FROM` | Unbounded data removal |
| `ALTER ROLE` | Privilege escalation |
| `GRANT` | Privilege escalation |
| `REVOKE` | Privilege removal |
| `\copy` | psql meta-command, data exfil risk |
| `\connect` | psql meta-command, connection switching |

The check is case-insensitive and strips leading SQL comments (`-- ...`) so that
comment-prefix bypasses are also blocked.

For blocked statement types, run `nself db migrate` directly from the CLI.

## See Also

- [[cmd-doctor]] — run the nSelf diagnostics suite
- [[cmd-plugin]] — manage the plugin catalog
- [[cmd-start]] — start the nSelf stack

← [[Commands]] | [[Home]] →
