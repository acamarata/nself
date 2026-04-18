# nself deploy

> Deploy the stack to a target environment (local, staging, production).

## Synopsis

```
nself deploy <target> [flags]
nself deploy status [--env <target>]
nself deploy rollback [target]
nself deploy logs [target]
nself deploy health [target]
nself deploy check-access
```

## Description

`nself deploy` builds your stack and starts it against a target environment. It chains `nself build` and `nself start`, then applies strategy-aware orchestration for staging and production. When a deploy host is configured (`NSELF_DEPLOY_HOST_STAGING`, `NSELF_DEPLOY_HOST_PROD`), the CLI prepares the push step; otherwise it runs the stack on the current host.

Targets accept both short and long forms:

| Target | Accepts | Description |
|---|---|---|
| local | `local` | Build and start on this machine |
| staging | `staging` | Staging environment (uses `NSELF_DEPLOY_HOST_STAGING` if set) |
| prod | `prod`, `production` | Production (uses `NSELF_DEPLOY_HOST_PROD` if set; requires `--force` or `--dry-run`) |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--strategy` | `rolling` | Deploy strategy: `rolling`, `blue-green`, `canary`, `preview` |
| `--dry-run` | false | Preview the deploy without executing |
| `--force` | false | Skip confirmation prompts (required for prod when not dry-run) |
| `--rolling` | false | Alias for `--strategy=rolling` |
| `--skip-health` | false | Skip post-deploy health checks |
| `--include-frontends` | false | Include frontend apps in the deploy |
| `--exclude-frontends` | false | Exclude frontend apps from the deploy |
| `--json` | false | Emit structured JSON output |
| `--env` | — | Override target (alias for the positional argument) |
| `--help`, `-h` | — | Show help |

## Subcommands

- `nself deploy status` — report current deploy state for a target
- `nself deploy rollback [target]` — roll back the last deployment (promotions use `nself promote rollback`)
- `nself deploy logs [target]` — tail the last 200 lines of Docker logs on the target host
- `nself deploy health [target]` — run `nself doctor` against the deployment
- `nself deploy check-access` — verify `NSELF_DEPLOY_HOST_*` values resolve

## Examples

```bash
# Local build + start
nself deploy local

# Staging dry-run with JSON output
nself deploy staging --dry-run --json

# Production blue-green deploy
nself deploy production --strategy=blue-green --force

# Production with skipped health checks
nself deploy prod --force --skip-health

# Inspect a target
nself deploy status --env staging
```

## Output format

When `--json` is set, the command writes a structured result:

```json
{
  "target": "staging",
  "strategy": "rolling",
  "steps": [
    {"name": "Build images", "status": "done"},
    {"name": "Start stack", "status": "skipped"}
  ],
  "durationMs": 1234,
  "success": true
}
```

Human output uses the same step vocabulary (`done`, `failed`, `skipped`, `running`, `pending`) inside `[...]` brackets so [[Admin|admin]]'s deploy API route can parse it without a separate protocol.

## Environment variables

| Var | Purpose |
|---|---|
| `NSELF_DEPLOY_HOST_STAGING` | SSH/rsync target for staging pushes |
| `NSELF_DEPLOY_HOST_PROD` | SSH/rsync target for production pushes |
| `STAGING_DEPLOY_HOST` | Fallback alias (staging) |
| `PROD_DEPLOY_HOST` | Fallback alias (production) |

When no host is configured, the CLI runs the deploy on the current host. This matches the v0.9.x behaviour where deploys were triggered from a session on the target machine itself.

## Safety

Production deploys without `--force` are rejected with a clear message. Use `--dry-run` first to preview steps on any target. Rollback for promotions is handled by `nself promote rollback`; the `deploy rollback` subcommand is a stub until the full rollback pipeline ships.

## See also

- [[cmd-build|nself build]] — generates `docker-compose.yml` and nginx configs
- [[cmd-start|nself start]] — boot the stack with health checks
- [[cmd-promote|nself promote]] — promote env-to-env with rollback support
- [[Home]]
