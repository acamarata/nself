# Tokens Plugin

> HLS encryption keys, content delivery tokens, and entitlement checks. **Free, MIT licensed.**

## Install

```bash
nself plugin install tokens
```

## What It Does

Manages HLS encryption keys for video delivery, issues time-limited content delivery tokens, and performs entitlement checks before serving protected content. Used by video streaming setups to gate content behind authentication without requiring per-request auth server calls.

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `TOKENS_PORT` | `3108` | Service port |
| `TOKENS_SECRET` | *(auto-generated)* | Token signing secret |
| `TOKENS_TTL` | `3600` | Default token TTL in seconds |
| `TOKENS_HLS_KEY_ROTATION` | `3600` | HLS key rotation interval |

## Ports

| Port | Purpose |
|------|---------|
| 3108 | Tokens REST API |

## Database Tables

5 tables added to your Postgres database:
- `np_tokens_tokens`, issued token records
- `np_tokens_hls_keys`, HLS encryption key archive
- `np_tokens_entitlements`, user content entitlements
- `np_tokens_revocations`, revoked tokens
- `np_tokens_audit`, token issuance audit log

## Nginx Routes

| Route | Target |
|-------|--------|
| `/tokens/` | Token issuance and validation API |
| `/keys/` | HLS key delivery endpoint |
