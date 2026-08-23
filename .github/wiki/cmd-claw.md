# nself claw

<!-- BEGIN PROSE:summary -->
> Manage ɳClaw AI assistant.
<!-- END PROSE:summary -->

## Synopsis

```
nself claw <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself claw` is the operator and end-user control plane for ɳClaw, the self-hosted AI assistant. It covers pairing client apps to a server, unlocking the web UI for first-time setup, sending one-shot prompts, opening interactive chats, browsing topics and memories, managing API keys, exposing an OpenAI-compatible local proxy, running an MCP server for tool integrations, exporting all data, and applying claw schema migrations.

The CLI talks to the `claw` plugin over its HTTP API. Authentication is by API key (env `NSELF_CLAW_API_KEY` or `~/.nself/claw/config.yaml`). For first-time setup, run `nself claw unlock` on the server to open a 10-minute window for account creation, then `nself claw pair --qr` to connect a mobile client.

`nself claw mcp` exposes ɳClaw memory, topics, and chat as Model Context Protocol tools so any MCP-aware client (Claude Code, IDEs) can read and write ɳClaw context. `nself claw proxy` exposes an OpenAI-compatible local endpoint so any client expecting OpenAI can route through ɳClaw without code changes.

### `claw pair`
### `claw unlock`
### `claw prompt [question]`
### `claw chat`
### `claw keys create`
### `claw mcp`
### `claw export`
### `claw migrate`
`claw config`, `claw config set`, `claw topics`, `claw topics search`, `claw memories`, `claw memories search`, `claw keys`, `claw keys revoke`, `claw status`, and `claw proxy` accept no flags beyond positional arguments.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `chat` | Start an interactive chat session with nClaw |
| `config` | Show or modify nClaw CLI configuration |
| `export` | Export all nClaw data |
| `keys` | Manage API keys |
| `mcp` | Start an MCP server for nClaw |
| `memories` | List or search memories |
| `migrate` | Apply pending claw schema migrations |
| `pair` | Generate a pairing code for nClaw clients |
| `prompt` | Send a single prompt to nClaw and print the response |
| `proxy` | Start a local OpenAI-compatible proxy |
| `session` | Manage Claude Code PTY sessions |
| `status` | Show nClaw server status and health |
| `topics` | List or search topics |
| `unlock` | Temporarily unlock the web UI for first-time account setup |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
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
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-ai]], AI plugin and local LLM stack
- [[cmd-plugin]], install or update the claw plugin
- [[cmd-status]], full stack status
- [[Commands]], full command index
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
