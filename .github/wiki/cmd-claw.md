# nself claw

> Manage ɳClaw AI assistant.

## Synopsis

```
nself claw <subcommand> [flags] [args]
```

## Description

`nself claw` is the operator and end-user control plane for ɳClaw, the self-hosted AI assistant. It covers pairing client apps to a server, unlocking the web UI for first-time setup, sending one-shot prompts, opening interactive chats, browsing topics and memories, managing API keys, exposing an OpenAI-compatible local proxy, running an MCP server for tool integrations, exporting all data, and applying claw schema migrations.

The CLI talks to the `claw` plugin over its HTTP API. Authentication is by API key (env `NSELF_CLAW_API_KEY` or `~/.nself/claw/config.yaml`). For first-time setup, run `nself claw unlock` on the server to open a 10-minute window for account creation, then `nself claw pair --qr` to connect a mobile client.

`nself claw mcp` exposes ɳClaw memory, topics, and chat as Model Context Protocol tools so any MCP-aware client (Claude Code, IDEs) can read and write ɳClaw context. `nself claw proxy` exposes an OpenAI-compatible local endpoint so any client expecting OpenAI can route through ɳClaw without code changes.

## Subcommands

| Name | Description |
|------|-------------|
| `pair` | Generate a pairing code for ɳClaw clients |
| `unlock` | Temporarily enable the web UI for first-time account setup |
| `prompt [question]` | Send a single prompt and print the response |
| `chat` | Start an interactive chat session |
| `config` | Show or modify ɳClaw CLI configuration |
| `config set <key> <value>` | Set a configuration value (`api-key`, `server`) |
| `topics` | List topics |
| `topics search <query>` | Search topics by name |
| `memories` | List memories |
| `memories search <query>` | Search memories by content |
| `keys` | List API keys |
| `keys create` | Create a new API key |
| `keys revoke <id>` | Revoke an API key |
| `status` | Show ɳClaw server status and health |
| `proxy [port]` | Start a local OpenAI-compatible proxy (upstream: nself-ai-gateway :3761) |
| `session` | Manage nself-ai-cc PTY sessions |
| `session start <provider>` | Start a new PTY session for an AI provider (calls :3760 POST /sessions) |
| `session attach <id>` | Attach to an active session via WebSocket (:3760 WS /sessions/:id/stream) |
| `session stop <id>` | Stop (terminate) an active PTY session (calls :3760 DELETE /sessions/:id) |
| `session list` | List all PTY sessions (calls :3760 GET /sessions) |
| `mcp` | Start an MCP server for ɳClaw |
| `export` | Export all ɳClaw data |
| `migrate` | Apply pending claw schema migrations |

## Flags

### `claw pair`

| Flag | Default | Description |
|------|---------|-------------|
| `--qr` | false | Display QR code in terminal |
| `--direct` | false | Skip cloud relay, local pairing only |

### `claw unlock`

| Flag | Default | Description |
|------|---------|-------------|
| `--minutes` | `10` | Minutes the enable lasts (1-60) |

### `claw prompt [question]`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Output full JSON response |
| `--model` | `""` | Override model name |
| `--topic` | `""` | Topic context |
| `--no-memory` | false | Skip memory injection |
| `--system` | `""` | Custom system prompt |
| `--raw` | false | No formatting, raw text only |
| `--timeout` | `120s` | Request timeout |
| `--file`, `-f` | `""` | Read prompt from file |
| `--no-stream` | false | Wait for complete response |

### `claw chat`

| Flag | Default | Description |
|------|---------|-------------|
| `--topic` | `""` | Start in specific topic |
| `--model` | `""` | Use specific model |
| `--resume` | false | Resume last conversation |
| `--session` | `""` | Resume specific session ID |

### `claw keys create`

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | `""` | Name for the API key (required) |

### `claw mcp`

| Flag | Default | Description |
|------|---------|-------------|
| `--transport` | `stdio` | Transport mode: `stdio` or `sse` |
| `--port` | `8899` | Port for SSE transport |

### `claw export`

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `json` | Output format: `json` or `csv` |

### `claw migrate`

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | `""` | Skip migrations at or before this version (e.g. `003_add_index.sql`) |
| `--to` | current nself version | Stop after applying this version |

`claw config`, `claw config set`, `claw topics`, `claw topics search`, `claw memories`, `claw memories search`, `claw keys`, `claw keys revoke`, `claw status`, and `claw proxy` accept no flags beyond positional arguments.

## Examples

```bash
# First-time server setup: open the web UI for 10 minutes to create an account
nself claw unlock

# Pair a mobile or desktop client (prints QR code)
nself claw pair

# Save API key and server URL on a workstation
nself claw config set api-key nself_claw_xxxxxxxxxxxxxxxx
nself claw config set server https://claw.example.com

# Send a one-shot prompt and stream the answer
nself claw prompt "Summarize the project README"

# Pipe stdin through nClaw with a custom system prompt
cat notes.txt | nself claw prompt "Extract decisions" --system "You are a precise note-taker"

# Open the interactive chat REPL
nself claw chat

# Search memories
nself claw memories search "Postgres tuning"

# Run as an MCP server for Claude Code
nself claw mcp --transport stdio

# Run an OpenAI-compatible proxy on port 9000
nself claw proxy 9000

# Back up all nClaw data
nself claw export > claw-backup.json

# Apply all pending claw schema migrations
nself claw migrate

# Apply only migrations 003 through 005
nself claw migrate --from 002_add_topics.sql --to 005_index.sql
```

## See Also

- [[cmd-ai]], AI plugin and local LLM stack
- [[cmd-plugin]], install or update the claw plugin
- [[cmd-status]], full stack status
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
