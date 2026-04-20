# nself functions

> Deploy, list, invoke, tail logs for, and delete serverless functions.

## Synopsis

```
nself functions <subcommand> [flags]
```

## Description

`nself functions` manages serverless functions running inside the nSelf functions service. Functions are deployed from local TypeScript, JavaScript, Deno, or Python files and are served at `http://localhost:3008/v1/<name>`.

The functions service must be enabled before use:

```bash
nself service enable functions
nself build && nself start
```

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `deploy <file\|dir>` | Deploy a function from a file or directory |
| `list` | List all deployed functions |
| `invoke <name>` | Send an HTTP request to a deployed function |
| `logs <name>` | Stream logs for a deployed function |
| `delete <name>` | Remove a deployed function |

---

### deploy

Copies the source file or directory into `./functions/<name>/` and signals the container to reload.

```
nself functions deploy <file|dir> [--name <name>] [--runtime node|deno|python] [--env KEY=VALUE]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--name` | filename (no extension) | Function name (lowercase alphanumeric + hyphens) |
| `--runtime` | node | Runtime override: `node`, `deno`, `python` |
| `--env` | — | Environment variable KEY=VALUE (repeatable) |

```bash
nself functions deploy hello-world.ts
nself functions deploy ./my-fn/ --name my-fn
nself functions deploy handler.py --runtime python --env DB_URL=postgres://...
```

---

### list

Lists all function directories under `./functions/`. Probes each function's health endpoint to report status.

```
nself functions list [--json] [--runtime <runtime>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Output as JSON array |
| `--runtime` | — | Filter by runtime |

---

### invoke

Sends an HTTP request to a deployed function and prints the response.

```
nself functions invoke <name> [--payload <json>] [--method GET|POST|...] [--auth <token>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--payload` | — | JSON body to send |
| `--method` | POST | HTTP method |
| `--auth` | — | Bearer token (not logged) |

```bash
nself functions invoke hello-world
nself functions invoke hello-world --payload '{"name":"Alice"}'
nself functions invoke hello-world --method GET
```

---

### logs

Tails Docker logs for the functions container, filtered to lines matching the function name.

```
nself functions logs <name> [--follow] [--since <duration>] [--tail <n>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--follow` | false | Stream continuously |
| `--since` | — | Show logs since duration (e.g. `1h`, `30m`) |
| `--tail` | 100 | Number of recent lines |

---

### delete

Removes the function directory. Requires `--confirm`.

```
nself functions delete <name> --confirm
```

---

## Runtimes

The runtime is determined by `FUNCTIONS_RUNTIME` in your `.env` (or `--runtime` on deploy):

| Runtime | Image | Use for |
|---------|-------|---------|
| `node` (default) | `nhost/functions:latest` | TypeScript, JavaScript (nhost-compatible) |
| `deno` | `denoland/deno:alpine` | Deno TypeScript/JavaScript |
| `python` | `python:3.12-slim` | Python 3 with `requirements.txt` |

Resource limits are controlled by `FUNCTIONS_MEMORY` (default `256M`) and `FUNCTIONS_CPU` (default `0.5`).

## Related

- [[Feature-Functions]] — functions service overview and configuration
- [[Config-Optional-Services]] — FUNCTIONS_RUNTIME, FUNCTIONS_MEMORY, FUNCTIONS_CPU, FUNCTIONS_TIMEOUT
- [[cmd-service]] — `nself service enable functions`

← [[Commands]] | [[Home]] →
