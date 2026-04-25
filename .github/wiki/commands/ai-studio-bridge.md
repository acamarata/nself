# `nself ai-studio bridge`

Start a secure Google AI Studio bridge via Cloudflare Tunnel.

---

## Summary

Opens a local HTTP reverse proxy and exposes it to Google AI Studio through an ephemeral Cloudflare Tunnel (trycloudflare.com — no account required). Gemini can query your local Postgres schema and run GraphQL reads against your nSelf instance without any cloud deployment.

---

## Usage

```bash
nself ai-studio bridge [flags]
```

---

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8890` | Local proxy port |
| `--no-context` | `false` | Disable `X-Nself-Schema-Context` header injection |
| `--dry-run` | `false` | Print tunnel info without starting proxy |
| `--idle-timeout` | `30` | Auto-close tunnel after N minutes of inactivity |
| `--ip-allowlist` | `` | Comma-separated CIDRs to restrict tunnel access |
| `--region` | `auto` | Cloudflare tunnel region |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `NSELF_AISTUDIO_PROXY_PORT` | Local proxy port (overridden by `--port`) |
| `NSELF_AISTUDIO_TUNNEL_REGION` | Cloudflare tunnel region |
| `NSELF_AISTUDIO_SCHEMA_CONTEXT` | Set to `false` to disable schema injection |
| `NSELF_AISTUDIO_AUTH_TOKEN` | Pre-shared auth token (auto-generated when unset) |
| `HASURA_GRAPHQL_ENDPOINT` | Hasura endpoint (default: `http://localhost:8080`) |
| `HASURA_GRAPHQL_ADMIN_SECRET` | Hasura admin secret |

---

## How It Works

1. `cloudflared` is downloaded automatically to `~/.nself/bin/cloudflared` if not present.
2. A local HTTP proxy starts on `--port` (default 8890).
3. Cloudflare Tunnel connects the proxy to a `*.trycloudflare.com` URL.
4. Every GraphQL request is forwarded to Hasura using the `ai_studio_read` role — mutations and DDL are blocked at the proxy layer.
5. The `X-Nself-Schema-Context` response header carries a base64-encoded JSON snapshot of your table structure so Gemini can understand your schema without a round-trip.
6. The tunnel auto-closes after the idle timeout (default 30 minutes).

---

## Security

- All requests require an `Authorization: Bearer <token>` header.
- The token is auto-generated (64-char hex) each session unless `NSELF_AISTUDIO_AUTH_TOKEN` is set.
- Only read operations (`query`) are allowed. Mutations are blocked with HTTP 403.
- Use `--ip-allowlist 192.168.1.0/24` to restrict access to specific CIDRs.
- The Hasura role `ai_studio_read` must exist in your Hasura metadata (SELECT only).

---

## Examples

```bash
# Start bridge with defaults
nself ai-studio bridge

# Disable schema header injection
nself ai-studio bridge --no-context

# Test what the bridge would do (no real tunnel)
nself ai-studio bridge --dry-run

# Restrict to local network only
nself ai-studio bridge --ip-allowlist 192.168.1.0/24

# Extend idle timeout to 60 minutes
nself ai-studio bridge --idle-timeout 60
```

---

## Connecting AI Studio

After starting the bridge:

1. Copy the `AI Studio bridge ready: https://xxxx.trycloudflare.com` URL.
2. In Google AI Studio, open **Custom connector**.
3. Paste `https://xxxx.trycloudflare.com/v1/graphql` as the endpoint.
4. Add header: `Authorization: Bearer <token>` (token is printed at startup).
5. Gemini will now have access to your local schema via `X-Nself-Schema-Context`.

---

## Related

- [[cmd-ai]] — `nself ai` — Manage the AI plugin and local LLM stack
- [[Home]]
