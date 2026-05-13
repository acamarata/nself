# Custom Services

Add your own backend services to any nSelf project using `nself service add`. This is the correct path for custom Docker services — not raw `docker-compose.yml` edits.

## Overview

nSelf supports up to 10 user-defined custom services (CS_1 through CS_10). Each slot maps to a named service in your generated `docker-compose.yml`. The CLI handles slot assignment, port allocation, and `.env.dev` updates automatically.

## Add a custom service

```bash
nself service add <name>
nself service add <name> --lang <language>
```

Supported languages: `go` (default), `node`, `python`, `rust`, `other`.

### Examples

```bash
# Go service (default)
nself service add my-api

# Python FastAPI service
nself service add analytics --lang python

# Node/TypeScript service
nself service add webhooks --lang node

# Preview without writing any files
nself service add my-api --lang rust --dry-run
```

### What the command does

1. Finds the next free `CS_N` slot in `.env.dev`
2. Creates `services/<name>/` with starter files for the chosen language
3. Writes `CS_N=<name>:<lang>:<port>` and `<NAME>_PORT=<port>` into `.env.dev`

After running, apply the change:

```bash
nself build   # regenerates docker-compose.yml with your new service
nself start   # launches the full stack including your service
```

## Scaffold at project init

Use `--cs-template` with `nself init` to scaffold a custom service at the same time as initialising a new project:

```bash
nself init --cs-template go
nself init --name my-project --cs-template python
```

This is equivalent to running `nself init` followed by `nself service add`.

## Slot assignment

Slots are assigned automatically (lowest free slot first). You can see your current slots with:

```bash
nself service list
```

Custom service slots appear as `CS_1` through `CS_10` in `.env.dev`. Each slot reserves a port starting at `8001` through `8010` (auto-assigned as `8000 + N`).

## Editing your service

After scaffolding, edit the files in `services/<name>/` to implement your logic. The `Dockerfile` is pre-configured to build and expose the correct port.

Key files by language:

| Language | Entry point |
|---|---|
| Go | `main.go` |
| Node | `src/index.ts` |
| Python | `main.py` |
| Rust | `src/main.rs` |
| Other | `main.sh` |

## Environment variables

The scaffold writes two env vars into `.env.dev`:

| Variable | Example | Purpose |
|---|---|---|
| `CS_N` | `CS_1=my-api:go:8001` | Registers the service with the nSelf stack |
| `<NAME>_PORT` | `MY_API_PORT=8001` | Available inside the service container |

Add additional env vars for your service under `CS_N_ENV` or directly in `.env.dev`.

## See also

- [[cmd-service]] — full `nself service` command reference
- [[cmd-init]] — `nself init` reference including `--cs-template`
- [[Home]]
