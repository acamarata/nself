# Browser Plugin

> Headless Chromium pool for screenshots, scraping, PDF generation, and JS execution. **Pro plugin — requires ɳClaw bundle or ɳSelf+.**

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install browser
```

## What It Does

Runs a managed pool of headless Chromium instances (via Playwright) accessible via a REST API. Capture screenshots, scrape JavaScript-rendered pages, generate PDFs, run arbitrary JS in a sandboxed browser context, and extract structured data from web pages. Sessions are isolated and cleaned up automatically after each task.

Built-in safety: RFC-1918 / loopback SSRF guard, stealth anti-fingerprinting mode, DB-backed URL allowlist.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PLUGIN_BROWSER_PORT` | `3719` | Browser service port |
| `PLUGIN_BROWSER_POOL_SIZE` | `4` | Concurrent Chromium instances |
| `PLUGIN_BROWSER_TIMEOUT` | `30` | Page-load timeout (seconds) |
| `PLUGIN_BROWSER_STEALTH` | `true` | Stealth / anti-fingerprinting mode |
| `PLUGIN_BROWSER_ALLOWLIST` | `` | Comma-separated domain allowlist (empty = allow all) |
| `PLUGIN_BROWSER_COOKIES_PERSIST` | `false` | Persist cookies to Postgres across restarts |

## Ports

| Port | Purpose |
|------|---------|
| 3719 | Browser service REST API |

## Database Tables

5 tables added to your Postgres database:

| Table | Purpose |
|-------|---------|
| `np_browser_sessions` | Operation audit log (per-account, isolated via `source_account_id`) |
| `np_browser_screenshots` | Screenshot metadata (per-account, isolated via `source_account_id`) |
| `np_browser_cookies` | Persisted cookie store (per-account, isolated via `source_account_id`) |
| `np_browser_allowlist` | Admin-controlled URL allowlist patterns |
| `np_browser_config` | Runtime configuration key/value store |

All tables tracked in Hasura with row-level access control.

## Nginx Routes

| Route | Target |
|-------|--------|
| `/browser/` | Browser automation API |

## API

```
GET  /health              — Liveness probe
POST /screenshot          — Capture page screenshot (PNG/JPEG)
POST /scrape              — Extract page HTML or text content
POST /pdf                 — Generate PDF from URL or raw HTML
POST /execute             — Run JavaScript in browser context
POST /automate            — Run multi-step scripted flow (YAML/JSON)
GET  /sessions            — List recent session history
GET  /config              — Get current runtime config
PUT  /config              — Update runtime config
GET  /allowlist           — List URL allowlist entries
POST /allowlist           — Add URL pattern to allowlist
DELETE /allowlist/:id     — Remove URL pattern from allowlist
```

## Safety

- **SSRF guard**: RFC-1918 ranges (`10.x`, `172.16-31.x`, `192.168.x`) and loopback (`127.x`, `::1`) are always blocked, even if listed in the allowlist.
- **Stealth mode**: Anti-fingerprinting patches applied by default (`PLUGIN_BROWSER_STEALTH=true`).
- **URL allowlist**: When `np_browser_allowlist` has any enabled rows, only matching domains are permitted. When empty, all domains (minus SSRF guard) are permitted.

## See Also

- [[plugin-claw]] — AI assistant that orchestrates browser tasks
- [[Commands]] — Full command reference
- [[Home]]
