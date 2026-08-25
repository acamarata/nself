# nself functions

<!-- BEGIN PROSE:summary -->
> Deploy, list, invoke, tail logs for, and delete serverless functions.
<!-- END PROSE:summary -->

## Synopsis

```
nself functions <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself functions` manages serverless functions running inside the ɳSelf functions service. Functions are deployed from local TypeScript, JavaScript, Deno, or Python files and are served at `http://localhost:3008/v1/<name>`.

The functions service must be enabled before use:

```bash
nself service enable functions
nself build && nself start
```

---
### deploy
Copies the source file or directory into `./functions/<name>/` and signals the container to reload.
```
nself functions deploy <file|dir> [--name <name>] [--runtime node|deno|python] [--env KEY=VALUE]
```
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
---
### invoke
Sends an HTTP request to a deployed function and prints the response.
```
nself functions invoke <name> [--payload <json>] [--method GET|POST|...] [--auth <token>]
```
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

- [[Feature-Functions]], functions service overview and configuration
- [[Config-Optional-Services]], FUNCTIONS_RUNTIME, FUNCTIONS_MEMORY, FUNCTIONS_CPU, FUNCTIONS_TIMEOUT
- [[cmd-service]], `nself service enable functions`
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `delete` | Delete a deployed function |
| `deploy` | Deploy a function from a file or directory |
| `invoke` | Invoke a deployed function |
| `list` | List deployed functions |
| `logs` | Stream logs for a function |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
<!-- TODO(docs): needs human prose -->

```bash
nself functions
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
