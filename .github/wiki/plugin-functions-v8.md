# Plugin: functions-v8

**Bundle:** ɳClaw · **Port:** 3088 · **Type:** paid

The `functions-v8` plugin runs user-supplied TypeScript functions inside a V8/Deno isolate pool. Each invocation gets a fresh Deno subprocess, a per-request timeout, and a hard memory ceiling. Functions are stored in Postgres and served via a single HTTP endpoint.

---

## TypeScript runtime

Functions run under **Deno 2.3.3** with access to the full Deno standard library and URL imports. The runtime enforces the following by default:

| Constraint | Value | Env var |
|---|---|---|
| Max duration | 30 000 ms | `FUNCTIONS_EDGE_MAX_DURATION_MS` |
| Max memory | 128 MB | `FUNCTIONS_EDGE_MAX_MEMORY_MB` |
| Pool slots | 5 | `FUNCTIONS_EDGE_POOL_SIZE` |
| Log retention | 7 days | `FUNCTIONS_EDGE_LOG_RETENTION_DAYS` |

Functions receive the HTTP request method and path via `NSELF_REQUEST_METHOD` and `NSELF_REQUEST_PATH` environment variables and a request body on stdin. They write their response to stdout.

---

## Installation

```bash
nself plugin install functions-v8
```

The CLI pulls the signed image from Docker Hub (`nself-org/plugin-functions-v8:latest`), registers it in the plugin registry, and starts the service on port 3088.

Verify the service is healthy:

```bash
curl http://localhost:3088/health
```

---

## SSRF restrictions

All outbound network access from user functions is routed through an internal SSRF-filtering HTTP proxy. The proxy blocks the following destination ranges:

| Range | CIDR | Reason |
|---|---|---|
| Loopback | `127.0.0.0/8`, `::1/128` | Prevents self-SSRF to the host |
| RFC-1918 private | `10.0.0.0/8` | Internal LAN |
| RFC-1918 private | `172.16.0.0/12` | Internal LAN |
| RFC-1918 private | `192.168.0.0/16` | Internal LAN |
| Link-local / APIPA | `169.254.0.0/16` | Cloud metadata (e.g. AWS IMDSv1) |
| Link-local IPv6 | `fe80::/10` | Link-local |
| CGNAT | `100.64.0.0/10` | Carrier-grade NAT (RFC 6598) |
| Multicast | `224.0.0.0/4`, `ff00::/8` | Not routable |

**How it works:** Deno's `--allow-net` flag is scoped to the proxy's loopback address only. All HTTP and HTTPS traffic (`HTTP_PROXY` / `HTTPS_PROXY`) flows through the proxy, which resolves the destination hostname and blocks any address that falls in the ranges above. DNS rebinding attacks are mitigated because the IP check happens at connect time after resolution — not at the URL level.

A user function that attempts to reach a blocked address will receive a connection error from Deno (the proxy returns HTTP 403). Example:

```typescript
// This will fail with a network error — 192.168.1.1 is RFC-1918.
const resp = await fetch("http://192.168.1.1/admin");
```

Public internet addresses are permitted. The proxy only enforces the deny list; no allowlist of specific external domains is required.

---

## Deploying a function

```bash
# Deploy inline TypeScript
nself functions deploy --name hello --source 'Deno.stdout.write(new TextEncoder().encode("Hello!"))'

# Deploy from a file
nself functions deploy --name greet --file ./greet.ts

# Invoke
nself functions invoke hello --method GET --path /functions/v1/hello
```

---

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `FUNCTIONS_EDGE_PORT` | `3088` | HTTP listen port |
| `FUNCTIONS_EDGE_POOL_SIZE` | `5` | Max concurrent isolates |
| `FUNCTIONS_EDGE_MAX_DURATION_MS` | `30000` | Per-function timeout (ms) |
| `FUNCTIONS_EDGE_MAX_MEMORY_MB` | `128` | V8 heap ceiling (MB) |
| `FUNCTIONS_EDGE_LOG_RETENTION_DAYS` | `7` | Log line TTL |
| `DENO_BINARY_PATH` | `/usr/local/bin/deno` | Path to the Deno binary |

---

## Security notes

- **No OS env inheritance.** The Deno subprocess does not inherit the host's environment. Only variables explicitly passed by the orchestrator (`NSELF_REQUEST_METHOD`, `NSELF_REQUEST_PATH`) and tenant-scoped user-defined env vars are available.
- **Filesystem read restricted to `/tmp`.** Functions cannot access the host filesystem beyond a per-invocation temp file.
- **npm imports disabled.** `--no-npm` prevents `npm:` specifiers from reaching the npm registry.
- **No persistent Deno config.** `--no-config` blocks any `deno.json` on the host from being loaded.
- **SSRF is blocked by default** — see the section above. The Security-Always-Free Doctrine applies: SSRF protection ships free and cannot be downgraded.

---

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `{"error":"function timeout"}` | Function exceeded `FUNCTIONS_EDGE_MAX_DURATION_MS` | Reduce function duration or increase the env var |
| `network error` on fetch to RFC-1918 | SSRF proxy blocked the destination | Target is a private IP — use a public endpoint |
| 429 on `/functions/v1/invoke` | Pool exhausted | Increase `FUNCTIONS_EDGE_POOL_SIZE` or reduce concurrency |
| Empty response body | Function wrote nothing to stdout | Ensure function writes a response |

See also: [`nself functions`](cmd-functions.md) · [`Feature-Functions.md`](Feature-Functions.md)
