# nself doctor

> Run comprehensive system diagnostics.

## Synopsis

```
nself doctor [flags]
```

## Description

`nself doctor` checks everything nSelf needs to function correctly and reports issues with actionable fix suggestions. It covers infrastructure prerequisites (Docker, Docker Compose, Git), Docker daemon health and permissions, disk and memory availability, network connectivity, configuration correctness, running container health, and plugin schema placement.

Run `nself doctor` when something is not working as expected, before deploying to a new environment, or as part of an automated health check pipeline. The `--fix` flag enables automatic remediation of common problems.

Exit codes: `0` all checks passed, `1` one or more checks failed, `2` warnings only.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--full` | false | Run all checks including network and memory (slower) |
| `--verbose` | false | Detailed diagnostics output |
| `--json` | false | JSON output |
| `--fix` | false | Suggest auto-fix for common issues |
| `--help`, `-h` | — | Show help |

## Checks Performed

| Category | What is Checked |
|----------|----------------|
| Infrastructure | `docker`, `docker compose`, `git` are installed |
| Docker | Daemon running, user permissions, BuildKit available |
| Disk | ≥5 GB free space recommended |
| Memory | ≥2 GB RAM recommended |
| Network | Internet connectivity, Docker Hub reachable |
| Configuration | `.env` exists, required vars set, password strength meets requirements |
| Containers | Health status of running containers, error logs for unhealthy services |
| Plugin schemas | Warns if `np_*` tables are in the `public` schema instead of plugin schemas |

## Examples

```bash
# Quick diagnostic
nself doctor

# Full diagnostic (includes network and memory checks)
nself doctor --full

# Verbose output with details per check
nself doctor --verbose

# JSON output for automated monitoring
nself doctor --json

# Suggest fixes for detected issues
nself doctor --fix
```

← [[Commands]] | [[Home]] →
