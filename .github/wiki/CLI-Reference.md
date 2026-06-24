# CLI Reference — E6 AI Gateway Commands

This page covers the ten new or updated CLI commands added in P4-E6 (AI Gateway Unification). For the full command index, see [[COMMANDS]].

---

## nself claw session

Manage nself-ai-cc PTY sessions. The `session` subgroup controls AI provider sessions (`claude`, `codex`, `gemini` providers) running in a PTY relay on port 3760.

```bash
nself claw session <subcommand> [flags]
```

### nself claw session start

Start a new PTY session for an AI provider.

```bash
nself claw session start <provider>
```

Calls `POST /sessions` on nself-ai-cc (port 3760). Returns a `session_id` used by `attach` and `stop`.

Supported providers: `claude`, `codex`, `gemini`

**Example:**

```bash
nself claw session start claude
# Output: session_id: sess_abc123
```

### nself claw session attach

Attach to an active PTY session via WebSocket.

```bash
nself claw session attach <session_id>
```

Opens `WS /sessions/:id/stream` on port 3760. Streams bidirectional I/O to the terminal. Press `Ctrl+D` to detach without killing the session.

**Example:**

```bash
nself claw session attach sess_abc123
```

### nself claw session stop

Stop (terminate) an active PTY session.

```bash
nself claw session stop <session_id>
```

Calls `DELETE /sessions/:id` on nself-ai-cc (port 3760). The PTY process is sent SIGTERM then SIGKILL after 5 seconds.

**Example:**

```bash
nself claw session stop sess_abc123
```

### nself claw session list

List all PTY sessions (active and recently terminated).

```bash
nself claw session list
```

Calls `GET /sessions` on port 3760. Columns: `ID`, `PROVIDER`, `STATUS`, `STARTED`, `DURATION`.

**Example output:**

```
ID           PROVIDER   STATUS   STARTED              DURATION
-----------  ---------  -------  -------------------  --------
sess_abc123  claude     active   2026-06-24T10:00:00  5m12s
sess_def456  codex      stopped  2026-06-24T09:45:00  2m03s
```

---

## nself claw proxy

Start a local OpenAI-compatible proxy. Routes requests through nself-ai-gateway (port 3761).

```bash
nself claw proxy [port] [flags]
```

The proxy translates OpenAI-compat requests (`/v1/chat/completions`, `/v1/models`) to the upstream gateway. Any client expecting the OpenAI API can route through nClaw without code changes.

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `""` | Default model to use for requests |
| `--upstream` | `http://localhost:3761` | Gateway upstream URL |

**Example:**

```bash
nself claw proxy --model claude-sonnet-4-6 "hello"
```

---

## nself gateway

Manage the nSelf AI gateway service: provider key vault, quota tracking, routing rules, and service health. See [[cmd-gateway]] for full documentation.

```bash
nself gateway <subcommand> [flags]
```

### nself gateway status

Health-check all three AI services (`nself-ai-cc` port 3760, `nself-ai-gateway` port 3761, `nself-ai-mcp` port 3762) in parallel. Exits 0 only when all three are healthy.

```bash
nself gateway status
```

**Example output:**

```
SERVICE            PORT   STATUS
-------            ----   ------
nself-ai-cc        3760   ok
nself-ai-gateway   3761   ok
nself-ai-mcp       3762   ok

3/3 services healthy
```

Exits 1 if any service is unreachable.

### nself gateway keys list

List stored provider keys. Key material (actual API key values) is never displayed.

```bash
nself gateway keys list
```

Columns: `ID`, `PROVIDER`, `LABEL`, `ACTIVE`, `CREATED`

### nself gateway keys add

Add a provider API key to the vault.

```bash
nself gateway keys add --provider anthropic
nself gateway keys add --provider openai --label "production" --key sk-...
```

Supported providers: `anthropic`, `openai`, `google`, `custom`

| Flag | Required | Description |
|------|----------|-------------|
| `--provider` | yes | Provider identifier |
| `--key` | no | API key value (hidden prompt if omitted) |
| `--label` | no | Optional label for the key |

### nself gateway keys remove

Remove a provider key by its UUID.

```bash
nself gateway keys remove <id>
```

Use `nself gateway keys list` to find the key ID.

### nself gateway quota

Display today's token and request usage from nself-ai-gateway.

```bash
nself gateway quota
nself gateway quota --provider anthropic
nself gateway quota --provider openai --model gpt-4o
```

Columns: `PROVIDER`, `MODEL`, `DATE`, `TOKENS_IN`, `TOKENS_OUT`, `REQUESTS`, `LIMIT`

### nself gateway routes

List the active routing rules that determine which provider and model handle each request.

```bash
nself gateway routes
```

Columns: `NAME`, `PROVIDER`, `MODEL`, `PRIORITY`, `ENABLED`

---

## Command Summary

| Command | Port | Description |
|---------|------|-------------|
| `nself claw session start <provider>` | 3760 | Start a PTY session for an AI provider |
| `nself claw session attach <id>` | 3760 | Attach to a running PTY session via WebSocket |
| `nself claw session stop <id>` | 3760 | Terminate a PTY session |
| `nself claw session list` | 3760 | List all PTY sessions |
| `nself claw proxy [port]` | 3761 | Local OpenAI-compat proxy via gateway |
| `nself gateway status` | 3760-3762 | Health-check all 3 AI services |
| `nself gateway keys list` | 3761 | List stored provider keys |
| `nself gateway keys add` | 3761 | Add a provider API key to the vault |
| `nself gateway keys remove <id>` | 3761 | Remove a provider key by ID |
| `nself gateway quota` | 3761 | Show daily token and request usage |
| `nself gateway routes` | 3761 | List active routing rules |

> Note: `keys list`, `keys add`, and `keys remove` are subcommands of `gateway keys`. The ten distinct documented actions are: session start, session attach, session stop, session list, claw proxy, gateway status, gateway keys list, gateway keys add, gateway keys remove, gateway quota, gateway routes (11 documented; 10 are new to E6, with `claw proxy` updated to target :3761).

---

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `NSELF_GATEWAY_MASTER_KEY` | Yes | 32-byte hex AES-256-GCM key for provider key encryption |
| `NSELF_MCP_TOKEN` | Yes | Service JWT for nself-ai-mcp to call nself-ai-gateway |
| `CLAUDE_BINARY_PATH` | No | Path to `claude` binary; falls back to PATH lookup |
| `GEMINI_BINARY_PATH` | No | Path to `gemini` binary; falls back to PATH lookup |
| `NSELF_GATEWAY_PORT` | No | Port override for gateway (test only; default 3761) |
| `NSELF_AICC_PORT` | No | Port override for cc relay (test only; default 3760) |
| `NSELF_MCP_PORT` | No | Port override for MCP SSE (test only; default 3762) |

Full env var reference: [[Config-Env-Vars]]

---

## Related

- [[cmd-claw]] — full ɳClaw command reference
- [[cmd-gateway]] — gateway command reference
- [[COMMANDS]] — full command index
- [[Home]]
