# Pages Plugin

> Landing page builder with page templates, sections, and form capture. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install pages
```

## What It Does

Build and manage marketing landing pages through a structured API. Define page templates with named sections (hero, features, testimonials, CTA), compose pages from sections, and capture form submissions. Pages are served as structured JSON for any frontend to render, or as pre-rendered HTML snippets.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `PAGES_PORT` | `3046` | Pages service port |
| `PAGES_FORM_SPAM_PROTECTION` | `true` | Enable honeypot spam filtering |
| `PAGES_PREVIEW_ENABLED` | `true` | Enable draft page preview |

## Ports

| Port | Purpose |
|------|---------|
| 3046 | Pages REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_pages_templates` — page template definitions
- `np_pages_pages` — page instances
- `np_pages_sections` — page sections
- `np_pages_form_submissions` — form capture records

## Nginx Routes

| Route | Target |
|-------|--------|
| `/pages/` | Pages management API |
