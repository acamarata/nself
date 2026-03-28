# nSelf CLI

![Build](https://img.shields.io/github/actions/workflow/status/nself-org/cli/ci.yml?branch=main&label=build)
![License](https://img.shields.io/badge/license-MIT-blue)
![Go](https://img.shields.io/badge/go-1.23%2B-00ADD8)

nself is a self-hosted backend you install in five minutes. Postgres, Hasura, auth, storage, and nginx. No cloud required.

---

## Quick Install

**macOS**

```bash
brew install nself-org/nself/nself
```

**Linux**

```bash
curl -fsSL https://install.nself.org | sh
```

After installation, run `nself version` to confirm it worked.

---

## Quick Start

Five commands to a running backend:

```bash
nself init myproject
cd myproject
nself build
nself start
open http://localhost:8080
```

`nself init` generates your project config. `nself build` produces a `docker-compose.yml` from that config. `nself start` brings up all services. When it finishes, PostgreSQL, Hasura GraphQL, auth, and nginx are all running locally.

---

## Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| Docker | 24+ (Desktop or Engine) | latest |
| macOS | 12 (Monterey) | 14+ |
| Linux | Ubuntu 20.04 or Debian 11 | Ubuntu 22.04+ |
| RAM | 2 GB | 4 GB |
| Disk | 5 GB free | 10 GB free |

Docker must be running before you use any nself command that starts services.

---

## Core Commands

| Command | What it does |
|---------|-------------|
| `nself init <name>` | Create a new project |
| `nself build` | Generate docker-compose from your config |
| `nself start` | Start all services |
| `nself stop` | Stop all services |
| `nself restart` | Restart services |
| `nself status` | Show service health |
| `nself logs [service]` | View logs |
| `nself exec <service> <cmd>` | Run a command in a container |
| `nself backup` | Back up Postgres |
| `nself restore <file>` | Restore from a backup |
| `nself update` | Update nself to the latest version |
| `nself plugin list` | List available plugins |

The full command list is wider. Run `nself help` or see the [command reference](https://docs.nself.org/commands) for every subcommand.

---

## Full Command Reference

```
nself init         Create a new project
nself build        Generate docker-compose.yml and configs
nself start        Build (if needed) and start all services
nself stop         Stop all services
nself restart      Smart restart with change detection
nself status       Show service health and container status
nself urls         Display all service URLs
nself logs         View service logs
nself exec         Execute commands in containers
nself doctor       Run system diagnostics
nself health       Health check management
nself config       Configuration management (show/get/set/list/validate/export/import)
nself service      Manage optional services (enable/disable/list)
nself ssl          SSL certificate management (status/renew)
nself db           Database operations (migrate/backup/restore/seed/shell/reset/hasura)
nself plugin       Manage plugins (list/install/remove/update)
nself license      Manage Pro license (set/show/validate/clear)
nself migrate      Migrate from nSelf v1 (detect/run/rollback)
nself reset        Stop containers and remove generated files
nself clean        Remove unused Docker resources
nself update       Check for and install CLI updates
nself version      Show version info
nself completion   Shell completions
nself admin        nSelf Admin GUI management
```

---

## Architecture

nself always starts four core services:

| Service | Role |
|---------|------|
| PostgreSQL | Primary database |
| Hasura | GraphQL API and metadata engine |
| Auth (nHost) | User authentication and JWTs |
| Nginx | Reverse proxy and SSL termination |

Six optional services are available. Enable them in your project config before running `nself build`:

- **Redis** - caching and queues
- **MinIO** - S3-compatible object storage
- **Email** - local mail (Mailpit in dev, your provider in prod)
- **Search** - full-text search via MeiliSearch or Typesense
- **Functions** - serverless runtime
- **Admin** - the nSelf Admin GUI at `localhost:3021`

Run `nself service list` to see what is available and what is enabled.

---

## Plugin System

The core CLI is MIT-licensed and free forever. Sixteen free plugins are included. Install any of them with:

```bash
nself plugin install monitoring
nself plugin install cron
nself plugin install notify
```

Pro plugins require a membership key. The Basic tier ($9.99/yr) unlocks all 59 Pro plugins. Once you have a key:

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install ai
nself plugin install livekit
nself plugin install commerce
```

The key is validated server-side. The CLI never stores credentials in plaintext. See [plugin docs](https://docs.nself.org/plugins) for the full list.

---

## Configuration

Config lives in `.env` (dev), `.env.staging`, and `.env.prod`. Each environment file controls service settings, ports, domains, and credentials for that environment. The `nself build` command reads the active env file and generates your `docker-compose.yml` and nginx config.

Never hand-edit `docker-compose.yml` directly. All changes go through the env file, then `nself build`. This keeps the generated config in sync with the CLI's expectations.

See [configuration docs](https://docs.nself.org/configuration) for every available variable.

---

## Environments

nself supports three environments out of the box:

| Environment | How to activate | Typical use |
|-------------|----------------|-------------|
| `dev` | default | Local development |
| `staging` | `nself start --env staging` | Pre-production testing |
| `prod` | `nself start --env prod` | Production server |

Switching environments rebuilds the config for that target. Your `.env.prod` should never exist in source control. Use your secrets manager to inject it on the server.

---

## Migrating from v1

If you have an existing nSelf v1 project, the `migrate` command handles the upgrade.

```bash
nself migrate detect   # Scan for v1 artifacts (safe, read-only)
nself migrate run      # Migrate to v2 with automatic backup
```

Migration steps run automatically:

- Stops running containers gracefully
- Backs up current state to `.nself/backup/{timestamp}/`
- Moves nginx configs to the v2 layout
- Regenerates docker-compose.yml, nginx, and SSL configs

To roll back:

```bash
nself migrate rollback                           # Restore the most recent backup
nself migrate rollback --backup 20260328-143022  # Restore a specific backup
nself migrate rollback --list                    # List available backups
```

---

## Deployment

See [Deployment Guide](https://docs.nself.org/deployment) for Hetzner, DigitalOcean, and bare metal setup.

The short version: provision a Linux server, install Docker, install nself, copy your project, run `nself start --env prod`. Nginx handles SSL via Let's Encrypt automatically when you set `DOMAIN` in `.env.prod`.

---

## Admin GUI

nself ships a local admin UI. Start it with:

```bash
nself admin start
```

It runs at `http://localhost:3021`. Use it to inspect service health, manage databases, and view logs without memorizing commands. It is a local tool only. It is never deployed to a server.

---

## Diagnostics

If something is not working, run:

```bash
nself doctor
```

This checks Docker availability, service health, port conflicts, config validity, and SSL status. It prints actionable output for each issue it finds.

---

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) to get started.

The CLI is written in Go. Tests live in `internal/` alongside the packages they test. Run `go test ./...` from the repo root. All CLI commands follow the pattern documented in `docs/COMMAND_SPEC.md`.

---

## License

MIT - free for personal and commercial use. See [LICENSE](LICENSE).
