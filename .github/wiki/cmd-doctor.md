# nself doctor

> Run full system diagnostics.

## Synopsis

```
nself doctor [flags]
```

## Description

`nself doctor` checks everything nSelf needs to function correctly and reports issues with actionable fix suggestions. It covers infrastructure prerequisites (Docker, Docker Compose, Git), Docker daemon health and permissions, disk and memory availability, network connectivity, configuration correctness, running container health, and plugin schema placement.

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
