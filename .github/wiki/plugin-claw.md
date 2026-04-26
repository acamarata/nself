# Claw Plugin

> AI agent runtime with ReAct loop, tool discovery, and SSE streaming. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install claw
```

## What It Does

Provides the backend for ɳSelf Claw, an AI assistant that can use tools, browse the web, run code, and interact with your data. Implements a ReAct (Reasoning + Acting) loop where the agent plans steps, executes tools, and observes results iteratively. Streams responses via SSE for real-time display. Connects to any chat interface or the companion `claw-web` Next.js frontend.

## Dependencies

Requires the `ai` plugin.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CLAW_PORT` | `3710` | Claw service port |
| `CLAW_MAX_STEPS` | `20` | Max ReAct loop steps per query |
| `CLAW_TIMEOUT` | `120s` | Max time per agent run |
| `CLAW_TOOLS_ENABLED` | `search,code,data` | Enabled tool categories |

## Ports

| Port | Purpose |
|------|---------|
| 3710 | Claw agent REST API + SSE streaming |

## Database Tables

6 tables added to your Postgres database:
- `np_claw_sessions`, conversation sessions
- `np_claw_messages`, message history
- `np_claw_tool_calls`, tool invocation log
- `np_claw_tools`, registered tool definitions
- `np_claw_contexts`, injected context sources
- `np_claw_feedback`, user feedback on responses

## Nginx Routes

| Route | Target |
|-------|--------|
| `/claw/` | Claw agent API |
| `/claw/stream` | SSE streaming endpoint |
