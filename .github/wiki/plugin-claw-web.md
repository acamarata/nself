# plugin-claw-web

> Self-hosted browser UI for the ɳClaw AI assistant. **Pro plugin — requires license.**

Part of the **ɳClaw bundle** ($0.99/mo or $9.99/yr). Also included in ɳSelf+ ($3.99/mo or $39.99/yr).

---

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install claw
nself plugin install claw-web
nself build
nself start
```

`claw-web` depends on the `claw` plugin. Install both before running `nself build`.

---

## What It Does

Provides a full-featured browser interface for ɳClaw AI assistant conversations. Key surfaces:

| Page | Route | Description |
|------|-------|-------------|
| Chat | `/` | Real-time AI chat with SSE streaming and tool call traces |
| Projects | `/projects` | Browse and manage ɳClaw projects and workspaces |
| Data Browser | `/data` | Inspect memory entries, documents, and retrieved context |
| Settings | `/settings` | Profile, model selection, voice, plugins, security settings |
| Login | `/login` | Auth gate — JWT issued by the `claw` plugin |

All pages require authentication. Unauthenticated requests redirect to `/login`.

---

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `CLAW_INTERNAL_URL` | No | `http://plugin-claw:3710` | Internal URL of the `claw` plugin service |
| `CLAW_DOMAIN` | Yes | — | Public domain (e.g., `claw.example.com`) |
| `ORIGIN` | Yes | — | App origin (typically `https://$CLAW_DOMAIN`) |

Set these in your `.env.prod` before running `nself build`.

---

## Ports

| Port | Purpose |
|------|---------|
| `3004` | claw-web container (internal, proxied via Nginx) |

Port 3004 is registered in the canonical port registry (F10-PORT-REGISTRY, ɳClaw bundle, requires_license=true).

---

## Quick Start

```bash
# 1. Set license
nself license set nself_pro_xxxxxxxx

# 2. Install plugins
nself plugin install claw
nself plugin install claw-web

# 3. Configure environment
echo "CLAW_DOMAIN=claw.example.com" >> .env.prod
echo "ORIGIN=https://claw.example.com" >> .env.prod

# 4. Build and start
nself build
nself start
```

Open `https://claw.example.com` in any browser.

---

## Docker

The plugin ships as a multi-arch Docker image:

```bash
docker pull nself/plugin-claw-web:latest
```

Supported platforms: `linux/amd64`, `linux/arm64`. Managed by `nself build`. Do not run standalone.

---

## Database Tables

No tables added. All conversational state and memory are stored by the `claw` plugin.

---

## Nginx Routes

`nself build` generates a Nginx server block for `CLAW_DOMAIN` that proxies all traffic to port 3004 with WebSocket upgrade headers. ɳClaw streams AI responses over WebSocket.

| Route | Target |
|-------|--------|
| `/` | claw-web container (port 3004) |

---

## License Gate

This plugin requires an active license with `max` entitlement.

| Tier | Included |
|------|----------|
| Free | No |
| ɳClaw bundle ($0.99/mo or $9.99/yr) | Yes |
| ɳSelf+ ($3.99/mo or $39.99/yr) | Yes |

Installing without a valid license returns an error with a purchase link.

---

## Dependencies

| Plugin | Required | Notes |
|--------|----------|-------|
| `claw` | Yes | Provides AI backend, auth, memory, and tool APIs |

Install `claw` before `claw-web`. Running `claw-web` without `claw` causes the UI to fail at startup.

---

## Source

Source-available (license required to run):
[`plugins-pro/paid/claw-web/`](https://github.com/nself-org/plugins-pro/tree/main/paid/claw-web)

Source access is granted to ɳSelf+ subscribers and Enterprise customers.

---

## See Also

- [[plugin-claw]] - the AI assistant runtime that claw-web depends on
- [[bundle-nclaw]] - full ɳClaw bundle documentation
- [[Feature-nClaw]] - ɳClaw product feature overview
- [[cmd-claw]] - `nself claw` CLI commands
- [[Home]]
