# Browser Plugin

> Headless Chromium pool for screenshots, scraping, PDF generation, and JS execution. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install browser
```

## What It Does

Runs a managed pool of headless Chromium instances accessible via a REST API. Capture screenshots, scrape JavaScript-rendered pages, generate PDFs, run arbitrary JS in a sandboxed browser context, and extract structured data from web pages. Sessions are isolated and cleaned up automatically after each task.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `BROWSER_PORT` | `3716` | Browser service port |
| `BROWSER_POOL_SIZE` | `3` | Number of browser instances |
| `BROWSER_TIMEOUT` | `30s` | Max time per browser task |
| `BROWSER_MAX_CONCURRENT` | `10` | Max concurrent requests |

## Ports

| Port | Purpose |
|------|---------|
| 3716 | Browser service REST API |

## Database Tables

2 tables added to your Postgres database:
- `np_browser_sessions` — browser session log
- `np_browser_results` — cached task results

## Nginx Routes

| Route | Target |
|-------|--------|
| `/browser/` | Browser automation API |

## API

```
POST /screenshot      — Capture page screenshot
POST /scrape          — Extract page content
POST /pdf             — Generate PDF from URL or HTML
POST /execute         — Run JavaScript in browser context
POST /extract         — Extract structured data with CSS selectors
```
