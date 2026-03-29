# WAF Plugin

> Web application firewall — rate limiting, IP blocking, and request filtering via nginx. **Pro plugin.**

> **Requires:** Basic license tier or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxx...
nself plugin install waf
```

## What It Does

Adds a web application firewall layer to your nginx configuration. Blocks known malicious IP ranges, filters requests matching OWASP attack signatures (SQLi, XSS, path traversal), enforces geo-blocking, and applies per-IP and per-endpoint rate limits. Rules are managed via a REST API and applied without nginx restart. Block decisions are logged to Postgres for analysis.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `WAF_PORT` | `3062` | WAF management service port |
| `WAF_MODE` | `block` | Mode: `detect` (log only) or `block` |
| `WAF_RATE_LIMIT_DEFAULT` | `100r/m` | Default rate limit |
| `WAF_GEO_BLOCK_ENABLED` | `false` | Enable geo-blocking |
| `WAF_OWASP_RULES` | `true` | Enable OWASP ruleset |
| `WAF_IP_BLOCKLIST_URL` | — | External IP blocklist URL |

## Ports

| Port | Purpose |
|------|---------|
| 3062 | WAF management REST API |

## Database Tables

4 tables added to your Postgres database:
- `np_waf_rules` — custom WAF rules
- `np_waf_ip_blocklist` — blocked IP ranges
- `np_waf_events` — blocked request log
- `np_waf_allowlist` — trusted IP allowlist

## Nginx Routes

None — WAF rules are injected into nginx configuration directly.
