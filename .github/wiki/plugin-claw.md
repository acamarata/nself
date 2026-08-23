# ɳClaw Plugin

Smart AI assistant and agent runtime with WebSocket channels, session management, persistent memory, identity, tool execution, and the Canvas A2UI protocol. **Pro plugin** — requires ɳClaw or ɳSelf+ bundle.

## Install

```bash
# Set your license key first
nself license set nself_pro_xxxxx...

# Install claw and its required dependencies
nself plugin install claw
```

Dependencies (`ai`, `mux`, `notify`, `cron`) are resolved and installed automatically.

## What It Does

Provides the backend runtime for ɳClaw. Core capabilities:

- **Channels**: persistent WebSocket sessions with topic routing
- **Agent loop**: ReAct (Reasoning + Acting) with tool discovery and streaming output
- **Memory**: per-session and cross-session persistent memory backed by Postgres
- **Identity**: per-user context and identity management
- **Tools**: extensible tool registry, callable from any connected client
- **Canvas A2UI**: structured UI protocol for rich agent-driven interfaces

## Dependencies

| Plugin | Required | Purpose |
|--------|----------|---------|
| `ai` | Required | LLM provider routing (OpenAI, Anthropic, Ollama) |
| `mux` | Required | HTTP multiplexer and WebSocket upgrade |
| `notify` | Required | Event fan-out and push notifications |
| `cron` | Required | Scheduled tasks and background jobs |
| `voice` | Optional | TTS/STT integration |
| `browser` | Optional | Headless browser tools |

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CLAW_PORT` | `3710` | Service port |
| `CLAW_MAX_STEPS` | `20` | Max agent loop steps per run |
| `CLAW_TIMEOUT` | `120s` | Max wall time per agent run |
| `CLAW_TOOLS_ENABLED` | `search,code,data` | Enabled tool categories |
| `CLAW_MEMORY_TTL` | `30d` | Memory retention window |
| `CLAW_CANVAS_ENABLED` | `true` | Enable Canvas A2UI protocol |

## Ports

| Port | Purpose |
|------|---------|
| 3710 | Agent REST API, WebSocket channels, SSE streaming |

## Database Tables

Tables added to your Postgres database (all under `np_claw_*`, isolated by `source_account_id`):

- `np_claw_sessions` — conversation sessions
- `np_claw_messages` — message history
- `np_claw_tool_calls` — tool invocation log
- `np_claw_tools` — registered tool definitions
- `np_claw_contexts` — injected context sources
- `np_claw_feedback` — user feedback on responses
- `np_claw_memory` — persistent agent memory
- `np_claw_channels` — WebSocket channel registry
- `np_claw_identities` — per-user identity records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/claw/` | Agent REST API |
| `/claw/stream` | SSE streaming |
| `/claw/ws` | WebSocket channels |
| `/claw/canvas` | Canvas A2UI protocol |

## Docker

The official image is published to Docker Hub with multi-arch support (linux/amd64 and linux/arm64):

```bash
docker pull nself/plugin-claw:latest
```

`nself plugin install claw` pulls and starts this image automatically.

## Bundle

Part of the **ɳClaw bundle** (`$0.99/mo`). Also included in **ɳSelf+** (`$3.99/mo`).

See [[bundle-nclaw]] for the full bundle catalog.

---

Related: [[plugin-claw-web]] | [[plugin-claw-budget]] | [[plugin-nself-ai-gateway]] | [[plugin-mux]] | [[Home]]
