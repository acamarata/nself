# Embedded PostgreSQL (pglite/wasmtime)

> Run the full ɳSelf stack without a Docker postgres container — PostgreSQL executes inside a WebAssembly runtime on the same host process.

## How it works

`nself start --embedded-pg` replaces the Docker postgres container with a pglite WebAssembly module executing inside [wasmtime](https://wasmtime.dev/) (via `wasmtime-go`). A Unix-domain socket bridge proxies the Postgres wire protocol between Hasura, Auth, and the WASM runtime. From the perspective of any service that talks to postgres, the connection is identical to a standard socket connection.

```
Hasura / Auth
     │
     │  Postgres wire protocol (Unix socket)
     ▼
PGSocketBridge  ─── intercepts unsupported commands (COPY, LISTEN, NOTIFY)
     │
     │  Postgres wire protocol (internal socket)
     ▼
EmbeddedPGRuntime (wasmtime + pglite WASM)
     │
     │  WASI filesystem preopen
     ▼
$NSELF_RUNTIME_DIR/pgdata/   ← durable Postgres data directory
```

pglite v0.2.17 ships with pgvector support included. No separate extension install is needed.

## Requirements

- macOS (darwin/amd64 or darwin/arm64) or Linux (amd64 or arm64)
- Internet access on first run to download the pglite WASM binary (cached at `~/.nself/cache/pglite/`)
- `CGO_ENABLED=1` build (the CLI binary from Homebrew satisfies this automatically)

The Docker postgres image is NOT required. Docker is still needed for Hasura, Auth, and Nginx.

## First run

On the first `nself start --embedded-pg`, the CLI:

1. Downloads `pglite.wasm` from `ping.nself.org/assets/pglite/0.2.17/pglite.wasm` (verifies sha256)
2. Compiles the WASM module to native machine code via wasmtime (takes 5-15 seconds on cold start)
3. Caches the compiled module at `~/.nself/cache/pglite/0.2.17/pglite.wasm.compiled`
4. Starts the pglite runtime with data at `.nself/embedded-pg/pgdata/`
5. Starts the socket bridge at `.nself/embedded-pg/pglite.sock.bridge`
6. Proceeds with the normal stack startup (Hasura, Auth, Nginx)

Subsequent starts reuse the compiled module cache. Warm start time is typically under 5 seconds.

## Environment variables

| Variable | Default | Purpose |
|---|---|---|
| `NSELF_EMBEDDED_PG` | `false` | Set to `true` to enable embedded PG without passing `--embedded-pg` every run |
| `NSELF_RUNTIME_DIR` | `.nself/embedded-pg` (project-relative) | Override the runtime directory path |

## Data persistence

Postgres data lives in `$NSELF_RUNTIME_DIR/pgdata/` — by default `.nself/embedded-pg/pgdata/` inside the project directory. This directory persists across restarts.

To start fresh, stop the stack and remove the `pgdata/` directory:

```bash
nself stop
rm -rf .nself/embedded-pg/pgdata
nself start --embedded-pg
```

## Backup

`nself backup` works with embedded PG. When `NSELF_EMBEDDED_PG=true`, the backup command targets the Unix-domain socket instead of the Docker container. It attempts `pg_dump --format=custom` first (producing a file compatible with `pg_restore`), then falls back to a SQL plain-text dump if `pg_dump` is not on PATH.

Backup files are written to `~/.nself/backups/` by default.

## Limitations

- COPY TO / COPY FROM, LISTEN, and NOTIFY are not supported via the bridge. Any attempt returns `SQLSTATE 0A000` (feature not supported). Standard INSERT/UPDATE/DELETE/SELECT work without restriction.
- Windows is not supported. The prebuilt libwasmtime static library is available for macOS and Linux only.
- The embedded runtime uses a single-process WASM instance. High-concurrency workloads may see higher latency than a native Postgres process.

## Troubleshooting

**"pglite WASM not found"** — run `nself start --embedded-pg` with internet access to trigger the download. Or set `PGLITE_WASM_PATH` to an existing `pglite.wasm` file.

**"Socket already in use"** — a previous embedded PG instance did not shut down cleanly. Remove the socket file: `rm -f .nself/embedded-pg/pglite.sock*`

**Slow warm start** — delete the compiled module cache to force recompile: `rm -f ~/.nself/cache/pglite/*/pglite.wasm.compiled`

## Related pages

- [[cmd-start]] — `nself start` flag reference
- [[Architecture]] — full ɳSelf stack architecture
- [[backup]] — backup and restore

← [[Architecture]] | [[Home]] →
