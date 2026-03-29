# Mux Plugin

> AI-powered email and webhook pipeline with priority routing and dead-letter queue. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install mux
```

## What It Does

Routes incoming emails and webhook events through an AI classification pipeline. Assigns priority scores, routes messages to handlers based on rules, and auto-replies using configurable templates. Failed deliveries land in a dead-letter queue for manual review. Uses the `ai` plugin for classification and intent detection.

## Dependencies

Requires Redis (`REDIS_ENABLED=true`). Optionally uses the `ai` plugin for AI classification.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `MUX_PORT` | `3711` | Mux service port |
| `MUX_IMAP_HOST` | — | IMAP server for email ingestion |
| `MUX_IMAP_USER` | — | IMAP username |
| `MUX_IMAP_PASS` | — | IMAP password |
| `MUX_SMTP_HOST` | — | SMTP server for auto-replies |
| `MUX_AI_ENABLED` | `true` | Use AI plugin for classification |
| `MUX_DLQ_RETENTION` | `30` | Days to retain failed messages |

## Ports

| Port | Purpose |
|------|---------|
| 3711 | Mux REST API and webhook receiver |

## Database Tables

6 tables added to your Postgres database:
- `np_mux_messages` — ingested messages
- `np_mux_routes` — routing rules
- `np_mux_handlers` — message handlers
- `np_mux_replies` — auto-reply templates
- `np_mux_dlq` — dead-letter queue
- `np_mux_audit` — processing audit log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/mux/webhook` | Incoming webhook receiver |
| `/mux/` | Mux management API |
