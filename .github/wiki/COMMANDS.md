# Commands

ɳSelf CLI provides 46 top-level commands. Each command links to a dedicated page with full usage, flags, and examples.

## Lifecycle

| Command | Description |
|---------|-------------|
| [[cmd-init]] | Initialize a new ɳSelf project , generates `.env` configuration via interactive wizard |
| [[cmd-build]] | Compose infrastructure from `.env` , generates `docker-compose.yml`, Nginx config, and SSL certificates |
| [[cmd-start]] | Boot the ɳSelf stack with health checks and automatic database initialization (alias: `up`) |
| [[cmd-stop]] | Gracefully shut down all services or specific named services (alias: `down`) |
| [[cmd-restart]] | Smart restart with config change detection , only restarts affected services |
| [[cmd-dev]] | Start development environment with optional plugin hot-reload |
| [[cmd-reset]] | Stop containers and remove generated files (preserves `.env` and data) |
| [[cmd-clean]] | Remove Docker resources associated with the current project |

## Monitoring & Diagnostics

| Command | Description |
|---------|-------------|
| [[cmd-status]] | Show health status of all services with exit code signaling |
| [[cmd-logs]] | View and filter service logs with color, search, and tail support |
| [[cmd-health]] | Health check management , check, watch, history, single service or endpoint |
| [[cmd-doctor]] | Run full system diagnostics: infrastructure, Docker, config, network |
| [[cmd-urls]] | Display all service URLs grouped by type with route conflict detection |
| [[cmd-monitor]] | Monitoring stack management (Grafana dashboard upgrades) |
| [[cmd-alerts]] | Manage Prometheus alert rules and silences |
| [[cmd-watchdog]] | Self-healing container watchdog with circuit breaker |
| [[cmd-dogfood]] | Production dogfood audit and reporting |

## Data

| Command | Description |
|---------|-------------|
| [[cmd-db]] | Database operations , migrate, seed, backup, restore, shell, reset, Hasura metadata |
| [[cmd-backup]] | Backup operations: create, list, restore, verify, prune, config, status, init-key |
| [[cmd-dr]] | Disaster recovery: drill, promote-standby, reconfigure-dns, rollback, fence |
| [[cmd-queue]] | Manage async job queues (pg-boss) |
| [[cmd-webhooks]] | Manage webhook processing and outbox |

## Configuration & Environments

| Command | Description |
|---------|-------------|
| [[cmd-config]] | Manage project configuration , show, get, set, list, validate, export, import |
| [[cmd-env]] | Multi-environment management: switch, list, diff, copy |
| [[cmd-service]] | Enable, disable, and list optional services (Redis, MinIO, email, search, monitoring) |
| [[cmd-promote]] | Promote one environment to another (e.g. staging to prod) |

## Networking & SSL

| Command | Description |
|---------|-------------|
| [[cmd-ssl]] | Manage SSL certificates , status and force renewal |
| [[cmd-trust]] | Set up local dev trust (DNS, SSL, port forwarding) |
| [[cmd-dns-setup]] | Add project domains to `/etc/hosts` (run with sudo) |

## Security

| Command | Description |
|---------|-------------|
| [[cmd-security]] | Security audit, setup, and status for hardening your deployment |
| [[cmd-secrets]] | Manage encrypted project secrets (age encryption) |
| [[cmd-waf]] | Manage the Web Application Firewall (Coraza + OWASP CRS) |

## Multi-tenancy & Billing

| Command | Description |
|---------|-------------|
| [[cmd-tenant]] | Tenant management: create, upgrade, suspend, destroy, audit |
| [[cmd-billing]] | Billing operations: usage, invoice-preview, report, retry-event |

## Plugins & Licensing

| Command | Description |
|---------|-------------|
| [[cmd-plugin]] | Manage ɳSelf plugins , install, remove, update, list, start, stop, status |
| [[cmd-license]] | Manage Pro membership license key , set, show, validate, clear, upgrade |

## AI & ɳClaw

| Command | Description |
|---------|-------------|
| [[cmd-ai]] | Manage the ɳSelf AI plugin and local LLM stack |
| [[cmd-claw]] | Manage ɳClaw AI assistant |

## Utilities

| Command | Description |
|---------|-------------|
| [[cmd-exec]] | Execute a command inside a running service container |
| [[cmd-admin]] | Open the ɳSelf Admin dashboard in browser at `localhost:3021` |
| [[cmd-completion]] | Generate shell completion scripts for bash, zsh, fish, or PowerShell |
| [[cmd-update]] | Update the ɳSelf CLI binary and admin Docker image |
| [[cmd-upgrade]] | Upgrade the ɳSelf CLI (detects install method) |
| [[cmd-version]] | Show version and system information |
| [[cmd-migrate]] | Detect and migrate v1 (Bash CLI) artifacts to v2 |

---
← [[Home]] | [[_Sidebar]]
