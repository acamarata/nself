# nself deploy

> Deploy the stack to a target environment (local, staging, production).

## Synopsis

```
nself deploy [target] [flags]
nself deploy --env <target> [flags]
nself deploy status [--env <target>] [--blue-green]
nself deploy rollback [target]
nself deploy promote
nself deploy logs [target]
nself deploy health [target]
nself deploy check-access
```

## Description

`nself deploy` builds your stack and deploys it to a target environment using a per-service
sequenced rolling restart. It chains `nself build` then restarts services in dependency order
(postgres → hasura → auth → storage → plugins), waiting for each service to pass a health check
before restarting the next.

The target environment can be supplied as a positional argument or via `--env`. The flag takes
priority when both are given. The three supported values are `local`, `staging`, and `prod`
(also accepted as `production`).

When `NSELF_DEPLOY_HOST_STAGING` or `NSELF_DEPLOY_HOST_PROD` is set, the CLI rsyncs the compose
file and env to the remote host, pulls updated images, then runs the rolling restart via SSH.
When no host is configured, the deploy runs on the current host (single-region model).

Targets accept both short and long forms:

| Target | Accepts | Description |
|---|---|---|
| local | `local` | Build and rolling-restart on this machine |
| staging | `staging` | Staging environment (uses `NSELF_DEPLOY_HOST_STAGING` if set) |
| prod | `prod`, `production` | Production (uses `NSELF_DEPLOY_HOST_PROD` if set; requires `--force` or `--dry-run`) |

## Deploy Strategies

| Strategy | Status | Behavior |
|---|---|---|
| `rolling` | **Default (v1.1.1)** | Per-service sequenced restart with health-gating. Each service waits up to 60s for `service_healthy` before the next restarts. |
| `blue-green / canary` | **Available via `--canary N` (Y17)** | Zero-downtime: green containers run alongside blue, Nginx shifts N% traffic to green during soak, then promotes to 100%. Requires `NSELF_FEATURE_BLUE_GREEN_DEPLOY=true`. |
| `preview` | Not yet implemented | Falls back to rolling with an explicit warning. |

The `--strategy=blue-green` and `--strategy=canary` flags still fall back to rolling for backwards
compatibility with existing scripts. Use `--canary N` to activate the implemented blue/green path.

See [[deploy-strategies]] for the full per-strategy behavior spec and downtime expectations.

## Blue/Green Canary Deploy (Y17)

Enable with `NSELF_FEATURE_BLUE_GREEN_DEPLOY=true` (feature flag `blue_green_deploy` via `nself flag list`).

### How it works

```
1. Pull new images tagged as green
2. Start green containers (docker compose -p nself-green up -d)
3. Health check green (30s timeout)
4. Nginx upstream: route N% to green (via upstream weights)
5. Canary soak period (default 5 min)
   - Monitor error rate from green containers
   - If error_rate > NSELF_CANARY_ERROR_THRESHOLD (default 1%) → auto-rollback
6. Promote Nginx upstream to 100% green
7. Health check green at 100%
8. Stop and remove blue containers
9. Green becomes blue for the next deploy
```

### Port assignment

| Color | Port offset | Example (Hasura) |
|---|---|---|
| Blue | 0 (base ports) | 8080 |
| Green | +100 | 8180 |

### Examples

```bash
# Canary at 10% (requires NSELF_FEATURE_BLUE_GREEN_DEPLOY=true)
nself deploy local --canary 10

# Canary at 25%
nself deploy staging --canary 25 --force

# Skip canary — flip directly to 100% green
nself deploy local --skip-canary

# Promote a canary to 100% after manual review
nself deploy promote

# Rollback to blue in < 5 seconds
nself deploy rollback local
```

### Backward-incompatible migrations

If a pending migration is not safe to run during canary (DROP COLUMN, DROP TABLE, RENAME COLUMN,
ALTER COLUMN TYPE, NOT NULL without DEFAULT), the deploy is blocked:

```
ERROR: Migration 0042_drop_column_old_name.sql is not backward-compatible.
Run with --force-migration to apply (disables canary, full downtime deploy).
```

Use `--force-migration` to proceed with a full downtime deploy.

### State file

Blue/green state is persisted to `.nself/bluegreen/state.json`. Run `nself deploy status --blue-green`
to inspect the current active environment and canary traffic split.

## Rolling Restart, Service Order and Downtime

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
| `--env` | — | Target environment: `local`, `staging`, or `prod`. Takes priority over the positional argument. Required env vars for remote targets: `NSELF_DEPLOY_HOST`, `NSELF_DEPLOY_USER`, `NSELF_DEPLOY_KEY_PATH` |
| `--follow` | false | Stream container logs after deploy until Ctrl-C (staging and prod only) |
| `--yes` | false | Skip the production confirmation prompt (alias for `--force`) |
| `--canary` | `0` | Start a canary deploy at N% traffic to green (Y17; requires `NSELF_FEATURE_BLUE_GREEN_DEPLOY=true`) |
| `--skip-canary` | false | Skip canary phase and flip directly to 100% green |
| `--force-migration` | false | Force deploy even with backward-incompatible migrations (disables canary) |
| `--help`, `-h` | — | Show help |

## Subcommands

- `nself deploy status [--blue-green]`, report current deploy state; `--blue-green` adds blue/green slot info
- `nself deploy rollback [target]`, roll back the last deployment (see below)
- `nself deploy promote`, flip Nginx to 100% green after a manual canary review
- `nself deploy logs [target]`, tail the last 200 lines of Docker logs on the target host
- `nself deploy health [target]`, run `nself doctor` against the deployment
- `nself deploy check-access`, verify `NSELF_DEPLOY_HOST_*` values resolve

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
# Local build + rolling restart (positional form)
nself deploy local

# Local build + rolling restart (flag form)
nself deploy --env local

# Staging dry-run — shows what would happen without executing
nself deploy staging --dry-run

# Staging dry-run using --env flag
nself deploy --env staging --dry-run

# Staging deploy with JSON output
nself deploy staging --json

# Production deploy (rolling)
nself deploy production --force

# Production deploy using --env, skipping the confirmation prompt via --yes
nself deploy --env prod --yes

# Production deploy and stream logs until Ctrl-C
nself deploy --env prod --force --follow

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

| Var | Default | Purpose |
|---|---|---|
| `NSELF_DEPLOY_HOST_STAGING` | — | SSH/rsync target for staging: `user@host:/path` |
| `NSELF_DEPLOY_HOST_PROD` | — | SSH/rsync target for production: `user@host:/path` |
| `NSELF_DEPLOY_HOST` | — | Generic remote host used when target-specific vars are unset: `user@host:/path` |
| `NSELF_DEPLOY_USER` | — | SSH user for remote deployments (used when host is specified without a user prefix) |
| `NSELF_DEPLOY_KEY_PATH` | `~/.ssh/id_ed25519` | SSH private key path for remote deployments (alias: `NSELF_DEPLOY_SSH_KEY`) |
| `STAGING_DEPLOY_HOST` | — | Fallback alias for `NSELF_DEPLOY_HOST_STAGING` |
| `PROD_DEPLOY_HOST` | — | Fallback alias for `NSELF_DEPLOY_HOST_PROD` |
| `NSELF_DEPLOY_SSH_KEY` | `~/.ssh/id_ed25519` | SSH key path (superseded by `NSELF_DEPLOY_KEY_PATH`; both accepted) |
| `HEALTHCHECK_TIMEOUT_<SERVICE>` | `60s` | Per-service health-check timeout |
| `NSELF_FEATURE_BLUE_GREEN_DEPLOY` | `false` | Enable blue/green canary path (Y17). Set to `true` to activate. |
| `NSELF_DEPLOY_STRATEGY` | `canary` | Deploy strategy when blue/green is active: `canary`, `blue-green`, `direct` |
| `NSELF_CANARY_PERCENT` | `10` | Initial canary traffic % (used when `--canary` is not passed) |
| `NSELF_CANARY_SOAK_MINUTES` | `5` | Soak duration before auto-promote |
| `NSELF_CANARY_ERROR_THRESHOLD` | `1.0` | Error rate % that triggers auto-rollback |
| `NSELF_DEPLOY_HEALTH_TIMEOUT` | `30` | Seconds to wait for green container health |
| `NSELF_BLUE_PORT_OFFSET` | `0` | Port offset for blue containers |
| `NSELF_GREEN_PORT_OFFSET` | `100` | Port offset for green containers |
| `NSELF_DEPLOY_ENV` | `production` | Deploy target environment set by `nself deploy` after resolving `--env` / positional argument. Values: `local`, `staging`, `production`. Exposed for subprocesses and plugins. |

When no host is configured, the CLI deploys to the current host. This is the
single-region model. Multi-region orchestration is available via the dedicated `nself region` command (`list`, `add`, `status`, `promote`).

## Maintenance Banner

`--maintenance-banner` is deferred to a future release (DEP-07). To show a maintenance page manually,
configure an nginx static page via `nginx/conf.d/`.

## Safety

- Production deploys without `--force` are rejected with a clear message
- Use `--dry-run` first to preview steps on any target
- Agent forwarding is disabled by default for all SSH connections
- SSH keys are never logged
- `--rollback` validates the promote tag's env against current code before applying; drift
 causes an explicit error rather than a silent misapply

## Cross-references

- [[cmd-build|nself build]], generates `docker-compose.yml` and nginx configs
- [[cmd-start|nself start]], boot the stack with health checks
- [[cmd-promote|nself promote]], promote env-to-env with rollback support
- [[deploy-strategies]], full per-strategy spec and downtime expectations
- [[Home]]
