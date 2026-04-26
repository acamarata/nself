# nself doctor

> Run full system diagnostics.

## Synopsis

```
nself doctor [flags]
```

## Description

`nself doctor` checks everything ɳSelf needs to function correctly and reports issues with actionable fix suggestions. It covers infrastructure prerequisites (Docker, Docker Compose, Git), Docker daemon health and permissions, disk and memory availability, network connectivity, configuration correctness, running container health, and plugin schema placement.

Run `nself doctor` when something is not working as expected, before deploying to a new environment, or as part of an automated health check pipeline. The `--fix` flag enables automatic remediation of common problems.

The `--deep` flag runs all 12 subsystem checks including open port analysis, weak cipher detection, exposed service bindings, container-level security, and a CIS container benchmark subset. The deep scan runs without a license key: all hardening checks are free by design.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--full` | false | Run all checks including network and memory (slower) |
| `--deep` | false | Run all 12 subsystem checks: host, docker, postgres, hasura, nginx, ssl, ping, plugins, license, monitoring, backups, security |
| `--verbose` | false | Detailed diagnostics output |
| `--json` | false | JSON output |
| `--fix` | false | Auto-fix safe issues where a fix command is available |
| `--only <section>` | — | Run only one subsystem check section (see Subsections below) |
| `--check-legacy` | false | Scan this host for stale v0.9 global paths and print cleanup instructions. Exits with non-zero if any are found. See below. |
| `--install-check` | false | Run the 6-stage onboarding funnel check. Invoked automatically by the Homebrew post-install hook; safe to run at any time. Exits 0 when all 6 stages pass. See below. |
| `--ai` | false | Run the AI first-run wizard: install Ollama, set up Gemini pool, verify |
| `--yes` | false | Non-interactive mode: accept all defaults (for CI/scripts, used with `--ai`) |
| `--skip-ollama` | false | Skip local Ollama installation step (used with `--ai`) |
| `--skip-pool` | false | Skip Gemini pool setup step (used with `--ai`) |
| `--headless` | false | Print OAuth URL instead of opening a browser, for SSH or headless servers (used with `--ai`) |
| `--help`, `-h` | — | Show help |

## Exit Codes

### Standard mode (`nself doctor`)

| Code | Meaning |
|------|---------|
| `0` | All checks passed |
| `1` | One or more checks failed |
| `2` | Warnings only, no failures |

### Deep mode (`nself doctor --deep`)

| Code | Meaning |
|------|---------|
| `0` | All checks passed |
| `1` | One or more failures (no critical findings) |
| `2` | One or more CRITICAL security findings |

CRITICAL findings include: world-readable secret files, sensitive ports bound on `0.0.0.0`, missing JWT secrets, and weak SSL cipher suites. These indicate an immediate security risk and must be resolved before the service is considered safe to run.

## Checks Performed

### Standard mode

| Category | What is Checked |
|----------|----------------|
| Infrastructure | `docker`, `docker compose`, `git` are installed |
| Docker | Daemon running, BuildKit available, Compose v2 |
| Disk | At least 5 GB free space recommended |
| Memory | At least 2 GB RAM recommended (with `--full`) |
| Network | Internet connectivity, Docker Hub reachable (with `--full`) |
| Configuration | `.env` exists, required vars set, password strength meets requirements |
| Containers | Health status of running containers, error logs for unhealthy services |
| Plugin schemas | Warns if `np_*` tables are in the `public` schema instead of plugin schemas |
| License | License cache age and tier |

### Deep mode subsections (`--deep` or `--only <section>`)

| Section | What is Checked |
|---------|----------------|
| `host` | Disk free, swap usage, CPU load, clock sync, kernel tainted flag |
| `docker` | Storage driver, dangling images, container health |
| `postgres` | `pg_isready`, longest running query, dead tuples, last vacuum |
| `hasura` | `/healthz` endpoint, metadata consistency |
| `nginx` | Config syntax test, SSL cert expiry per domain |
| `ssl` | Certbot timer active, last renewal age |
| `ping` | `ping.nself.org` reachable |
| `plugins` | Plugin container health endpoints |
| `license` | License cache present and not in grace period |
| `monitoring` | Prometheus, Grafana, Loki reachable |
| `backups` | Last backup age (must be under 26 hours) |
| `security` | JWT secret, container user, secret file permissions, exposed ports, weak SSL ciphers |
| `ai-safety` | AI plugin moderation wire-up gap (see below) |
| `performance` | **PERF-POOL-01**: pgxpool connection cap vs postgres_max_connections |
| `security` | **PERM-RLS-01**: RLS enforcement for np_* tables (see below) |

### AI-SAFETY-01: ai+moderation Wire-Up Gap (S69-T05)

Fires during `nself doctor --deep` in the `ai-safety` section.

**What it checks:** If the `ai` plugin is loaded AND the deployment binds on a non-loopback address (e.g., `0.0.0.0`) AND the `moderation` plugin is NOT loaded, emit a `WARN`.

Self-hosted single-user deployments on loopback are explicitly exempted, this is a legitimate use case.

| Status | Meaning |
|--------|---------|
| `pass` | ai not loaded, or deployment is loopback-bound, or moderation is wired |
| `warn` | ai loaded on public-bound deployment without moderation , consider installing the moderation plugin |

**Fix:**
```bash
nself plugin install moderation
nself build && nself start
```

The check reads `NSELF_AI_LOADED`, `PLUGIN_AI_INTERNAL_URL`, `NSELF_MODERATION_LOADED`, `PLUGIN_MODERATION_INTERNAL_URL`, and `NSELF_BIND_ADDRESS` from the environment.

**Detection fixture:**
```bash
nself doctor --deep --config-file test/fixtures/ai-public-no-mod.env
# Emits: "moderation not wired on public-bound deployment with ai loaded"
```

---

### PERF-POOL-01: Pool Sizing Check

Fires during `nself doctor --deep` in the `performance` section.

**What it checks:** Total configured pgxpool `MaxConns` across all active services must not exceed `POSTGRES_MAX_CONNECTIONS`. Default Postgres ships with `max_connections=100`. With 23 enabled services at the default cap of 10 connections each, total = 230, exceeding the limit causes random `503` errors under load.

**Recommended cap formula:** `min(10, floor(postgres_max_connections / num_active_services))`

| Status | Meaning |
|--------|---------|
| `pass` | Total pool capacity ≤ postgres_max_connections |
| `warn` | Total pool capacity > postgres_max_connections , reduce per-service pool or raise `POSTGRES_MAX_CONNECTIONS` |

**Fix:** Raise `POSTGRES_MAX_CONNECTIONS` in `.env` or reduce per-plugin pool size. The check emits a `FixCmd` suggestion with the exact value to set.

```bash
# Skip the pool sizing check (not recommended in production)
nself doctor --deep --skip-pool
```

---

### PERM-RLS-01: RLS Enforcement Check (S74-T02 + S74-T-PERM-01)

Fires during `nself doctor --deep` in the `security` section.

**What it checks:**

1. For every `np_*` table: `pg_class.relrowsecurity = true` (RLS enabled) and at least one policy exists.
2. For every `np_*` table with a `tenant_id` column: `pg_class.relforcerowsecurity = true` (FORCE RLS, prevents table owner from bypassing policies).
3. For every `np_*` table with a `tenant_id` column: Hasura metadata has a `select` permission for the `user` role with a `tenant_id` row filter (`{"tenant_id": {"_eq": "X-Hasura-Tenant-Id"}}`).

Security-Always-Free Doctrine: this check runs without a license key.

| Status | Meaning |
|--------|---------|
| `pass` | All np_* tables have RLS enabled, at least one policy, and Hasura tenant_id filters where applicable |
| `warn` | Violation found (default); use `--strict` to escalate to `fail` |
| `fail` | Violation found with `--strict` flag |

**Violation output format:**
```
RLS-DISABLED table=np_chat_messages
RLS-FORCE-MISSING table=np_claw_cost_events role=user
HASURA-FILTER-MISSING table=np_claw_cost_events role=user
```

**Env vars read:**
- `NSELF_DB_URL` or `DATABASE_URL`, Postgres connection string
- `HASURA_GRAPHQL_URL`, Hasura endpoint (default: `http://127.0.0.1:8080`)
- `HASURA_GRAPHQL_ADMIN_SECRET`, required for Hasura metadata query; if absent, Hasura filter check is skipped with a warning

**Fix:**
```bash
# Fix missing FORCE RLS on a table
nself migrate apply --rls-force np_chat_messages

# Re-run the check after fixing
nself doctor --deep --only security
```

**See also:** [multi-tenant conventions](https://docs.nself.org/multi-tenancy/conventions) for the canonical wall doc on `source_account_id` vs `tenant_id`.

---

## --install-check: 6-stage onboarding funnel

`nself doctor --install-check` runs a focused readiness check that maps to the 6-stage onboarding funnel. It is invoked automatically by the Homebrew `post_install` hook after `brew install nself`. You can also run it manually at any time.

```
$ nself doctor --install-check

Onboarding Funnel Check
  Stage 1 — Install       v1.0.9 (darwin/arm64)
  Stage 2 — Activation    2 projects initialized
  Stage 3 — First-use     first start 3 days ago
  Stage 4 — First-plugin  no plugins installed
    → Run: nself plugin install ai  (or any plugin)
  Stage 5 — First-value   (skipped, prior stage failed)
  Stage 6 — Habit         (skipped, prior stage failed)

Funnel position: Stage 3/6. Next: nself plugin install ai  (or any plugin)
```

Each stage prints PASS, FAIL, UNKNOWN, or SKIPPED:

| Status | Meaning |
|--------|---------|
| PASS | Stage completed |
| FAIL | Not yet reached , remediation hint shown |
| UNKNOWN | Cannot determine (e.g. Stage 5 when Hasura telemetry hook is not wired) |
| SKIPPED | A prior stage failed; this stage was not evaluated |

Stage 5 shows UNKNOWN (not FAIL) when the `query-count` file is absent. This avoids penalising self-hosters who have not wired the optional Hasura telemetry hook.

Pass `--json` for machine-readable output:

```bash
nself doctor --install-check --json
```

Exit codes: 0 = all 6 stages pass; 1 = one or more stages fail.

## --check-legacy: v0.9 global host scan

`nself doctor --check-legacy` scans the host machine for stale v0.9 global paths that persist after migration:

| Path scanned | What it indicates |
|---|---|
| `~/.nself/` | v0.9 global config directory |
| `~/.nself/plugins/` | v0.9 plugin install directory |
| `~/.config/nself` | v0.9 XDG config file (v1 uses a directory) |
| `/usr/local/share/nself` | v0.9 shared data directory |

For each path found, the output shows the path, its type, and a cleanup hint. The check is read-only: it never deletes anything automatically.

```
$ nself doctor --check-legacy
WARNING: Found 2 v0.9 stale artifact(s) on this host:
  [dir]  /Users/me/.nself  — safe to remove: rm -rf ~/.nself (after verifying no custom config)
  [file] /usr/local/share/nself  — safe to remove: sudo rm -f /usr/local/share/nself

Run the cleanup commands above, then re-run `nself doctor --check-legacy` to confirm.
```

If no stale paths are found, the command exits 0 with `No v0.9 global artifacts detected. Clean install.`

This flag exits early without running any other doctor checks. Combine with other checks by running them separately.

## JSON Output

When `--json` is passed, the command writes a single JSON object to stdout and produces no other output. The JSON schema is:

```json
{
  "timestamp": "2026-04-17T12:00:00Z",
  "checks": [
    {
      "name": "[security] Exposed port 5432 (Postgres)",
      "status": "critical",
      "message": "Postgres port 5432 is bound on 0.0.0.0 — bind to 127.0.0.1",
      "detail": "docker stop nself_postgres && nself build && nself start"
    }
  ],
  "summary": {
    "total": 42,
    "passed": 39,
    "warnings": 1,
    "failed": 1,
    "critical": 1
  }
}
```

Status values: `pass`, `warn`, `fail`, `critical`.

## Examples

```bash
# Quick diagnostic
nself doctor

# Full diagnostic (includes network and memory checks)
nself doctor --full

# Full hardening scan across all 12 subsystems
nself doctor --deep

# Deep scan, one subsystem only
nself doctor --deep --only security

# Verbose output with details per check
nself doctor --verbose

# JSON output for automated monitoring
nself doctor --json

# Auto-fix safe issues
nself doctor --fix

# Deep scan with JSON output, useful for CI security gates
nself doctor --deep --json

# First-run AI wizard (installs Ollama, sets up Gemini pool)
nself doctor --ai

# First-run AI wizard, non-interactive
nself doctor --ai --yes
```

← [[Commands]] | [[Home]] →
