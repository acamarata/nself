# nself env

> Multi-environment management: switch, list, diff, copy.

## Synopsis

```
nself env <subcommand> [flags] [args]
```

## Description

`nself env` manages multiple environments (dev, staging, prod) on a single project. Each environment gets its own Docker project name, network, volume namespace, and port range, so they can coexist on one host without conflicts.

`env use <name>` switches the active environment. After switching, run `nself build` so the generated `docker-compose.yml` and nginx config reflect the new environment's `.env.<name>` cascade. `env list` shows all environments with the active one marked. `env show` prints the resolved config for the current environment with secrets redacted. `env diff a b` compares two environments side-by-side. `env copy from to` scaffolds a new environment from an existing one as the starting template.

The active environment is stored in a project-local marker (`.env.active` or equivalent) so subsequent commands like `nself start`, `nself logs`, and `nself db` all target the right stack.

## Subcommands

| Name | Description |
|------|-------------|
| `use <env>` | Switch active environment |
| `list` | List available environments |
| `show` | Show current environment config (secrets redacted) |
| `diff <env-a> <env-b>` | Compare two environment configs side-by-side |
| `copy <from> <to>` | Scaffold a new environment from an existing one |
| `target` | Manage control-plane server targets (see below) |

## Flags

The `env` subcommands take no flags beyond positional arguments. Use `--help` on any subcommand to confirm.

## Examples

```bash
# See which environments exist
nself env list

# Switch to staging and rebuild
nself env use staging
nself build

# Show the resolved config for the current environment
nself env show

# Compare dev and prod env configs
nself env diff dev prod

# Scaffold a new "qa" environment from staging
nself env copy staging qa
nself env use qa
```

---

## env target

> Manage the multi-server inventory stored in `.nself/control-plane.yaml`.

### Synopsis

```
nself env target <subcommand> [flags] [args]
```

### Description

`nself env target` manages a fleet of servers across environments. The inventory is persisted in `.nself/control-plane.yaml` (mode 0600). Each server entry records its role, SSH target, SSH key reference (env-var name only — never an inline key), remote installation path, and whether it is the primary app server.

The `probe` subcommand runs SSH + Docker checks against live servers and classifies each as `manage` (full control), `read-only` (SSH reachable but Docker unavailable), or `hidden` (SSH unreachable). A 60-second on-disk cache avoids repeated probe latency; use `--refresh` to bypass it.

The `migrate` subcommand is a one-time helper for teams that previously configured deployments via `NSELF_DEPLOY_HOST_*` environment variables. It synthesizes a `control-plane.yaml` from those variables without overwriting any existing manual entries.

### Subcommands

| Name | Args | Description |
|------|------|-------------|
| `list [env]` | optional env name | List servers in the inventory; `--json` for machine-readable output; `--all` to include hidden servers |
| `add <env> <name>` | env name, server name | Add a server to an environment |
| `remove <env> <name>` | env name, server name | Remove a server from an environment |
| `probe [env[:server]]` | optional filter | Run SSH + Docker probes and print a capability matrix |
| `migrate` | none | Migrate legacy `NSELF_DEPLOY_HOST_*` env vars to `control-plane.yaml` |

### Flags — list

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Emit JSON output |
| `--all` | false | Include hidden servers (no host configured) |

### Flags — add

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | — | SSH target in `user@host` form (required for remote servers) |
| `--role` | `app` | Server role: `app`, `lb`, `observability`, `db`, `worker` |
| `--key-ref` | — | Name of the env var that holds the SSH key path (never the key path itself) |
| `--remote-path` | `/opt/nself` | Absolute path on the remote host where nSelf is installed |
| `--primary` | false | Mark this server as the primary app server |
| `--upstreams` | — | Upstream app server names (LB role only, comma-separated) |

### Flags — probe

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Emit JSON output |
| `--refresh` | false | Bypass the 60-second probe cache |

### Security invariants

- `--key-ref` accepts an env-var **name** only (e.g. `NSELF_SSH_KEY_STAGING`). Supplying a file path or inline key material is rejected on input.
- SSH probes use `BatchMode=yes StrictHostKeyChecking=accept-new ForwardAgent=no`. No interactive prompts, no agent forwarding.
- Secret values are never written to or read from the inventory file. `list` and `probe` output shows `SSHKeyRef` names, not values.
- All writes to `control-plane.yaml` use mode 0600.

### Examples

```bash
# List all servers (table)
nself env target list

# List only staging servers, JSON output
nself env target list staging --json

# Add a primary app server to staging
nself env target add staging web1 \
  --host ubuntu@staging.example.com \
  --role app \
  --key-ref NSELF_SSH_KEY_STAGING \
  --remote-path /opt/nself \
  --primary

# Add an LB server that routes to web1 and web2
nself env target add staging lb1 \
  --host ubuntu@lb.example.com \
  --role lb \
  --key-ref NSELF_SSH_KEY_STAGING \
  --upstreams web1,web2

# Remove a server from staging
nself env target remove staging web1

# Probe all servers (capability matrix)
nself env target probe

# Probe only staging, JSON, bypassing cache
nself env target probe staging --json --refresh

# Probe one specific server
nself env target probe staging:web1

# Migrate legacy NSELF_DEPLOY_HOST_* env vars
nself env target migrate
```

### Output format — probe table

```
ENV      SERVER  CAPABILITY  SSH  DOCKER  LATENCY  REASON
staging  web1    manage      ok   ok      42ms
staging  lb1     read-only   ok   -       38ms     docker socket permission denied
prod     db1     manage      ok   ok      91ms
```

### Output format — probe JSON

```json
[
  {
    "env": "staging",
    "server": "web1",
    "capability": "manage",
    "ssh_reachable": true,
    "docker_ok": true,
    "latency_ms": 42
  }
]
```

### Capability values

| Value | Meaning |
|-------|---------|
| `manage` | Full control: SSH reachable, Docker socket accessible |
| `read-only` | SSH reachable but Docker unavailable (permission or socket issue) |
| `hidden` | SSH unreachable; server excluded from deploy pipeline |

### Inventory file format

`.nself/control-plane.yaml` (mode 0600):

```yaml
version: 1
environments:
  staging:
    name: staging
    kind: remote
    servers:
      - name: web1
        role: app
        host: ubuntu@staging.example.com
        ssh_key_ref: NSELF_SSH_KEY_STAGING
        remote_path: /opt/nself
        primary: true
  prod:
    name: prod
    kind: remote
    servers:
      - name: web1
        role: app
        host: deploy@prod.example.com
        ssh_key_ref: NSELF_SSH_KEY_PROD
        remote_path: /opt/nself
        primary: true
```

---

## See Also

- [[cmd-build]], regenerate compose for the active environment
- [[cmd-promote]], promote one environment to another
- [[cmd-secrets]], environment-scoped secrets
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
