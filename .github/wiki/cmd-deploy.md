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

`nself deploy` builds your stack and deploys it to a target environment using a per-service
sequenced rolling restart. It chains `nself build` then restarts services in dependency order
(postgres → hasura → auth → storage → plugins), waiting for each service to pass a health check
before restarting the next.

When `NSELF_DEPLOY_HOST_STAGING` or `NSELF_DEPLOY_HOST_PROD` is set, the CLI rsyncs the compose
file and env to the remote host, pulls updated images, then runs the rolling restart via SSH.
When no host is configured, the deploy runs on the current host (the v1.0.9 single-region model).

Targets accept both short and long forms:

| Target | Accepts | Description |
|---|---|---|
| local | `local` | Build and rolling-restart on this machine |
| staging | `staging` | Staging environment (uses `NSELF_DEPLOY_HOST_STAGING` if set) |
| prod | `prod`, `production` | Production (uses `NSELF_DEPLOY_HOST_PROD` if set; requires `--force` or `--dry-run`) |

## Deploy Strategies

| Strategy | Status | Behavior |
|---|---|---|
| `rolling` | **Implemented (v1.0.9)** | Per-service sequenced restart with health-gating. Each service waits up to 60s for `service_healthy` before the next restarts. |
| `blue-green` | Not yet implemented. v1.1.0 target | Falls back to rolling with an explicit warning. |
| `canary` | Not yet implemented. v1.1.0 target | Falls back to rolling with an explicit warning. |
| `preview` | Not yet implemented. v1.1.0 target | Falls back to rolling with an explicit warning. |

When `--strategy=blue-green` (or canary/preview) is passed, the CLI emits:

```
WARN: Strategy "blue-green" is not yet implemented in v1.0.9. Tracked for v1.1.0; falling back to rolling.
```

See [[deploy-strategies]] for the full per-strategy behavior spec and downtime expectations.

## Rolling Restart — Service Order and Downtime

The rolling strategy restarts services in dependency order. Each service restart is health-gated
(max 60s wait). If a service does not become healthy within 60s, the deploy halts and reports
which service failed.

| Service | Restart order | Expected downtime |
|---|---|---|
| postgres | 1st | 5–30s (WAL recovery time) |
| hasura | 2nd | 0–10s (waits for postgres) |
| auth | 3rd | 0–5s |
| storage | 4th | 0–3s |
| plugins | 5th | 0–5s per plugin |

Total deploy time: approximately 75s minimum (5 services × ~15s each). Total user-visible
downtime is lower than the deploy duration because each service continues serving until its
replacement becomes healthy.

Use `--skip-health-check` to bypass the 60s gate for known-slow services like MeiliSearch or
Grafana. Set `HEALTHCHECK_TIMEOUT_<SERVICE>` (e.g. `HEALTHCHECK_TIMEOUT_GRAFANA=120s`) to
extend the per-service timeout.

## Remote Push

Set `NSELF_DEPLOY_HOST_STAGING` or `NSELF_DEPLOY_HOST_PROD` to a `user@host:/remote/path`
value to enable remote deployments. The CLI:

1. rsyncs `docker-compose.yml` and `.env.<target>` to the remote
2. Runs `docker compose pull` on the remote to fetch updated images
3. Runs the rolling restart on the remote via SSH

```bash
export NSELF_DEPLOY_HOST_STAGING=ubuntu@167.235.233.65:/opt/nself-staging
nself deploy staging
```

SSH key defaults to `~/.ssh/id_ed25519`. Override with `NSELF_DEPLOY_SSH_KEY`.

Agent forwarding is disabled by default. The CLI uses
`-o ForwardAgent=no -o StrictHostKeyChecking=accept-new` for all SSH connections.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--strategy` | `rolling` | Deploy strategy: `rolling`, `blue-green`, `canary`, `preview` |
| `--dry-run` | false | Preview the deploy without executing |
| `--force` | false | Skip confirmation prompts (required for prod when not dry-run) |
| `--rolling` | false | Alias for `--strategy=rolling` |
| `--skip-health` | false | Skip post-deploy health checks (visible warning emitted) |
| `--include-frontends` | false | Include frontend apps in the deploy |
| `--exclude-frontends` | false | Exclude frontend apps from the deploy |
| `--json` | false | Emit structured JSON output |
| `--env` | — | Override target (alias for the positional argument) |
| `--help`, `-h` | — | Show help |

## Subcommands

- `nself deploy status` — report current deploy state for a target
- `nself deploy rollback [target]` — roll back the last deployment (see below)
- `nself deploy logs [target]` — tail the last 200 lines of Docker logs on the target host
- `nself deploy health [target]` — run `nself doctor` against the deployment
- `nself deploy check-access` — verify `NSELF_DEPLOY_HOST_*` values resolve

## Rollback

`nself deploy rollback` is wired to the last `nself promote` tag. Every `nself promote <src> <target>`
call creates a backup snapshot tagged `pre-promote-<target>-<unix>` and writes a promotion record
to `.nself/promotions/<id>.json`. Rollback reads the most recent record and restores the env,
migration state, and Hasura metadata from that snapshot.

**Rollback does not roll back container images.** If you need an older image, pull it manually
(`docker pull nself/...:v1.0.8`) before running rollback.

```bash
# After a failed or unwanted production promotion:
nself deploy rollback prod
# Finds last pre-promote-prod-* backup tag
# Runs: nself backup restore --tag pre-promote-prod-<ts>
# Prints: Rollback for prod completed — prior promote state restored
```

When no promote history exists:

```
Error: rollback failed: find backup: no promotion records found
Run 'nself promote staging' before a deploy to create a rollback point.
```

When the last promote tag is older than 7 days, rollback prompts for confirmation. A stale rollback
may re-apply migrations that no longer match the current code. Review the diff shown before confirming.

**Requires:** `nself promote` must have run at least once against the target environment to create
a rollback point. On a fresh production host, run `nself promote staging prod` before the first
`nself deploy production` to ensure a rollback baseline exists.

## Health Checks

After each deploy (unless `--skip-health` is set), the CLI calls `nself doctor` and checks the
output. If any service is `unhealthy` or has not started within 60s, the deploy result is `failed`
and the specific service name is reported:

```
  [running] Health checks (calling nself health)
  [failed] Health check failed (service: hasura). Run 'nself doctor --verbose' for details.
```

Use `--skip-health` as a break-glass escape hatch. A visible warning is always emitted when used.

## Examples

```bash
# Local build + rolling restart
nself deploy local

# Staging dry-run (shows what would happen)
nself deploy staging --dry-run

# Staging deploy with JSON output
nself deploy staging --json

# Production deploy (rolling)
nself deploy production --force

# Production deploy with explicit strategy (blue-green falls back to rolling with warning)
nself deploy production --strategy=blue-green --force

# Roll back to the last promoted state
nself deploy rollback prod

# Check SSH access to configured targets
nself deploy check-access
```

## Output format

When `--json` is set, the command writes a structured result:

```json
{
  "target": "staging",
  "strategy": "rolling",
  "steps": [
    {"name": "Build images", "status": "done"},
    {"name": "Restart postgres", "status": "done"},
    {"name": "Restart hasura", "status": "done"},
    {"name": "Restart auth", "status": "done"},
    {"name": "Restart storage", "status": "done"},
    {"name": "Restart plugins", "status": "done"},
    {"name": "Health checks", "status": "done"}
  ],
  "durationMs": 78234,
  "success": true
}
```

Human output uses the same step vocabulary (`done`, `failed`, `skipped`, `running`, `pending`,
`unhealthy`) inside `[...]` brackets so [[Admin|admin]]'s deploy API route can parse it without
a separate protocol.

## Environment variables

| Var | Purpose |
|---|---|
| `NSELF_DEPLOY_HOST_STAGING` | SSH/rsync target for staging: `user@host:/path` |
| `NSELF_DEPLOY_HOST_PROD` | SSH/rsync target for production: `user@host:/path` |
| `STAGING_DEPLOY_HOST` | Fallback alias for `NSELF_DEPLOY_HOST_STAGING` |
| `PROD_DEPLOY_HOST` | Fallback alias for `NSELF_DEPLOY_HOST_PROD` |
| `NSELF_DEPLOY_SSH_KEY` | SSH key path (default: `~/.ssh/id_ed25519`) |
| `HEALTHCHECK_TIMEOUT_<SERVICE>` | Per-service health-check timeout (default: 60s) |

When no host is configured, the CLI deploys to the current host. This is the v1.0.9 LTS
single-region model. Multi-region is deferred to v1.1.0.

## Maintenance Banner

`--maintenance-banner` is deferred to v1.1.0 (DEP-07). To show a maintenance page manually,
configure an nginx static page via `nginx/conf.d/`.

## Safety

- Production deploys without `--force` are rejected with a clear message
- Use `--dry-run` first to preview steps on any target
- Agent forwarding is disabled by default for all SSH connections
- SSH keys are never logged
- `--rollback` validates the promote tag's env against current code before applying; drift
  causes an explicit error rather than a silent misapply

## Cross-references

- [[cmd-build|nself build]] — generates `docker-compose.yml` and nginx configs
- [[cmd-start|nself start]] — boot the stack with health checks
- [[cmd-promote|nself promote]] — promote env-to-env with rollback support
- [[deploy-strategies]] — full per-strategy spec and downtime expectations
- [[Home]]
