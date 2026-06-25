# Functions V8 Plugin

> Edge Functions runtime using a V8/Deno isolate pool with TypeScript support, HTTP triggers, and strict SSRF protection. **Pro plugin, requires license.**

> **Requires:** Pro license or higher. `nself license set nself_pro_...`

## Install

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install functions-v8
nself build
```

The license is validated against `ping.nself.org/license/validate`. An insufficient tier returns an error and the purchase URL.

## What It Does

The functions-v8 plugin runs short-lived TypeScript functions in isolated Deno subprocess pools. Each function invocation gets a fresh Deno process, an allowlist-only environment (no OS env inheritance), a configurable memory ceiling, and a hard execution timeout. Functions are deployed as TypeScript source code, stored in Postgres, and invoked over HTTP.

Capabilities:

- TypeScript + Deno standard library support
- Per-function environment variables (stored encrypted in Postgres)
- Configurable isolate pool size for concurrency control
- Hard wall-clock execution timeout (returns HTTP 504 on breach)
- Prometheus metrics at `/metrics`
- Structured JSON logs with per-function log retention
- **SSRF protection enforced at the runtime level** (see below)

## Deploy a Function

```bash
# Deploy function source code
curl -X POST http://your-nself-host:3088/functions/v1 \
  -H 'Content-Type: application/json' \
  -H 'X-Internal-Secret: $PLUGIN_INTERNAL_SECRET' \
  -d '{
    "name": "hello-world",
    "source_code": "export default async function handler(req) { return new Response(JSON.stringify({hello: \"world\"})); }"
  }'
```

## Invoke a Function

```bash
curl -X POST http://your-nself-host:3088/functions/v1/hello-world/invoke \
  -H 'Content-Type: application/json' \
  -d '{"name":"nSelf"}'
```

The function receives the request body via `Deno.stdin` and writes its response to `stdout`. The HTTP status code is determined by the exit code and output format.

## TypeScript Runtime

Functions run under Deno with access to the Deno standard library (`https://deno.land/std@0.208.0/`). Example function:

```typescript
// Read request body from stdin
const body = await new Response(Deno.stdin.readable).text()
const input = JSON.parse(body || '{}')

const result = {
  message: `Hello, ${input.name || 'world'}!`,
  timestamp: new Date().toISOString(),
}

console.log(JSON.stringify(result))
```

## SSRF Protection

**Outbound network access is deny-by-default.** The plugin does NOT pass `--allow-net` unrestricted to Deno. Instead:

1. By default, `--deny-net` is set — user functions have no outbound network access.
2. Operators may set `NSELF_ALLOW_NET` to a comma-separated list of specific external hostnames to permit.
3. **RFC-1918 and loopback addresses are always blocked**, even if listed in `NSELF_ALLOW_NET`. The Go runtime filters them before constructing the Deno command.

This satisfies the Security-Always-Free Doctrine: no SSRF vector is exposed by default, and no paywall blocks the protection.

```bash
# Allow only a specific external API
NSELF_ALLOW_NET=api.openai.com,hooks.slack.com nself start
```

Attempting `fetch('http://192.168.1.1/')` from user code will fail with a Deno permission error. This is verified by the test suite (`TestIsPrivateHost_RFC1918`).

## Function Isolation

Each invocation runs in a separate Deno subprocess:

- No shared memory between invocations
- OS environment is NOT inherited — only explicitly permitted env vars are injected
- Memory ceiling enforced via `--v8-flags=--max-old-space-size=<MB>`
- `--no-npm` prevents npm: specifiers from downloading packages at runtime
- `--no-config` prevents `deno.json` from the host filesystem from being loaded

## Configuration

| Env Var | Required | Default | Description |
|---------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `FUNCTIONS_EDGE_PORT` | No | `3088` | HTTP server port |
| `FUNCTIONS_EDGE_POOL_SIZE` | No | `5` | Max concurrent function invocations |
| `FUNCTIONS_EDGE_MAX_DURATION_MS` | No | `30000` | Max execution time per invocation (ms) |
| `FUNCTIONS_EDGE_MAX_MEMORY_MB` | No | `128` | V8 heap limit per isolate (MB) |
| `FUNCTIONS_EDGE_LOG_RETENTION_DAYS` | No | `7` | Function log retention (days) |
| `DENO_BINARY_PATH` | No | `/usr/bin/deno` | Path to the Deno binary |
| `PLUGIN_INTERNAL_SECRET` | No | — | Shared secret for internal API calls |
| `NSELF_ALLOW_NET` | No | — | Comma-separated external hostname allowlist |

## Postgres Schema

| Table | Purpose |
|-------|---------|
| `np_edge_functions` | Function metadata (name, status, invoke/error counts) |
| `np_edge_function_versions` | Versioned source code; one active version per function |
| `np_edge_function_env` | Per-function environment key/value pairs |
| `np_function_logs` | Structured execution logs with TTL-based retention |

`np_edge_functions` includes `source_account_id TEXT NOT NULL DEFAULT 'primary'` for multi-app isolation (Convention A).

## HTTP Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | none | Health check |
| `GET` | `/metrics` | internal | Prometheus metrics |
| `GET` | `/functions/v1` | internal | List functions |
| `POST` | `/functions/v1` | internal | Deploy a function |
| `GET` | `/functions/v1/:name` | internal | Get function metadata |
| `DELETE` | `/functions/v1/:name` | internal | Delete a function |
| `POST` | `/functions/v1/:name/invoke` | internal | Invoke a function |
| `GET` | `/functions/v1/:name/logs` | internal | Fetch function logs |

Internal routes require the `X-Internal-Secret` header matching `PLUGIN_INTERNAL_SECRET`.

## Port

This plugin runs on port `3088` (CS_7 slot per the nSelf port registry).

## Bundle

This plugin is not included in a named bundle. Access is granted with any Pro license tier or ɳSelf+.

## See Also

- [Architecture: Microkernel](Architecture-Microkernel.md)
- [Security](security/)
- [Plugin Reference](Plugin-Reference.md)
