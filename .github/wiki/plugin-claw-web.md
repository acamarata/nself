# Claw Web Plugin

> Self-hosted browser UI for the nSelf Claw AI assistant. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install claw-web
```

## What It Does

Provides a full-featured web interface for conversations, tool results, and agent history powered by the nSelf Claw AI assistant. Built on Next.js, it renders SSE-streamed responses in real time and displays tool call traces alongside assistant replies. Requires the `claw` plugin to supply the agent backend.

## Dependencies

Requires the `claw` plugin.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `CLAW_WEB_PORT` | `3004` | Claw web UI service port |
| `CLAW_API_URL` | — | URL of the running claw plugin API |
| `NEXTAUTH_SECRET` | — | NextAuth.js secret for session signing |
| `NEXTAUTH_URL` | — | Public base URL of this UI |

## Ports

| Port | Purpose |
|------|---------|
| 3004 | Claw web UI (Next.js) |

## Database Tables

0 tables added. All conversational state is stored by the `claw` plugin.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/` | Claw web UI |
