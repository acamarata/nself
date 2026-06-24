# ɳClaw Web UI Plugin

> Full browser interface for the ɳClaw AI assistant: chat, projects, data browser, and settings from any device. **Pro plugin — requires license.**

## Tier required

| Tier | Monthly | Annual | Includes this plugin? |
|------|---------|--------|----------------------|
| Free | $0 | $0 | No |
| Any bundle | $0.99/mo | $9.99/yr | If in bundle |
| ɳSelf+ | $3.99/mo | $39.99/yr | Yes |

**Minimum tier:** ɳClaw Bundle or ɳSelf+ (this is a `tier: max` plugin per F07-PRICING-TIERS).

## Bundle membership

This plugin is included in the following bundles:

- **ɳClaw Bundle** ($0.99/mo or $9.99/yr) — see [[bundle-nclaw]]

Or get all bundles + all apps via **ɳSelf+** ($3.99/mo or $39.99/yr).

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install claw-web
nself build
```

The license is validated against `ping.nself.org/license/validate`. Tier is checked
server-side; insufficient tier returns an error.

**Prerequisite:** the `claw` plugin must be installed and running before `claw-web` starts.

```bash
nself plugin install claw
nself plugin install claw-web
nself build
nself start
```

## Description

The ɳClaw Web UI plugin provides a React 19 + Vite browser front end that connects to the
`claw` plugin API. Once installed, it serves a full interface at your configured domain
(port 3004 locally) covering chat sessions, project management, a data browser, and
account settings.

All pages communicate with `plugin-claw` over the internal `CLAW_INTERNAL_URL`. Traffic
from the public internet goes through nginx, which terminates TLS and proxies both HTTP
and WebSocket connections to port 3004. SSE streaming is enabled on all chat routes so
responses appear token-by-token.

The license gate runs at startup. If the installed license does not cover the `max` tier,
the service exits immediately with a clear error message. No data is served until the
license passes.

## Configuration

| Env Var | Default | Required | Description |
|---------|---------|----------|-------------|
| `CLAW_INTERNAL_URL` | `http://plugin-claw:3710` | No | Internal URL of the `claw` plugin API |
| `CLAW_DOMAIN` | — | Yes | Public domain for claw-web, e.g. `claw.myserver.com` |
| `ORIGIN` | — | Yes | App origin for CORS/CSP (same as `https://${CLAW_DOMAIN}`) |

Set env vars via:

```bash
nself env set claw-web CLAW_DOMAIN=claw.example.com
nself env set claw-web ORIGIN=https://claw.example.com
nself build
```

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 3004 | HTTP | ɳClaw Web UI (React 19 + Vite, registered in F10-PORT-REGISTRY.md) |

## Database Schema

No database tables are added by this plugin. All conversational state, project data,
and session storage are managed by the `claw` plugin (`np_claw_*` tables). This plugin
is a pure front end.

## Nginx Routes

| Route | Method | Purpose |
|-------|--------|---------|
| `/` | GET | React app shell and all page routes |
| `/api/claw/ws` | GET (WS upgrade) | WebSocket endpoint for real-time push |

The nginx fragment is at `nginx-claw-web.conf`. It is copied into place by `nself build`
and must not be hand-edited.

## Pages

| Page | Path | Description |
|------|------|-------------|
| Chat | `/chat` | Real-time chat with ɳClaw, streaming via SSE |
| Projects | `/projects` | Create and manage AI-assisted projects |
| Data browser | `/data` | Browse and inspect data indexed by ɳClaw |
| Settings | `/settings` | User preferences, API keys, theme |

## Examples

**Start claw-web on a fresh server:**

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install claw claw-web
nself env set claw-web CLAW_DOMAIN=claw.example.com
nself env set claw-web ORIGIN=https://claw.example.com
nself build
nself start
# Visit https://claw.example.com
```

**Check plugin status:**

```bash
nself plugin status claw-web
```

**View logs:**

```bash
nself logs claw-web
```

**Update to latest:**

```bash
nself plugin update claw-web
nself build
nself restart claw-web
```

## Source

Source-available (license required to run): [`plugins-pro/paid/claw-web/`](https://github.com/nself-org/plugins-pro/tree/main/paid/claw-web)

Note: `plugins-pro` is a private repository. Source access is granted to ɳSelf+ subscribers
and Enterprise customers.

## See Also

- [[plugin-claw]] — the `claw` AI backend this plugin connects to
- [[bundle-nclaw]] — bundle that includes both `claw` and `claw-web`
- [[Pricing]] — tier comparison
- [[Plugins]] — full plugin index

← [[Plugins]] | [[Home]] →
