# ɳSelf - Complete Self-Hosted Backend Platform

[![Go](https://img.shields.io/badge/go-1.25%2B-00ADD8.svg)](https://go.dev/)
[![Status](https://img.shields.io/badge/status-stable-brightgreen.svg)](https://github.com/nself-org/cli/releases)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey.svg)](#prerequisites)
[![Docker](https://img.shields.io/badge/docker-required-blue.svg)](https://www.docker.com/get-started)
[![CI Status](https://github.com/nself-org/cli/actions/workflows/ci.yml/badge.svg)](https://github.com/nself-org/cli/actions)
[![Security Scan](https://github.com/nself-org/cli/actions/workflows/security-scan.yml/badge.svg)](https://github.com/nself-org/cli/actions/workflows/security-scan.yml)
[![Test Coverage](https://codecov.io/gh/nself-org/cli/branch/main/graph/badge.svg)](https://codecov.io/gh/nself-org/cli)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

<div align="center">

**Deploy a production-ready backend in 5 minutes**

Complete self-hosted Backend-as-a-Service with PostgreSQL, GraphQL API, Authentication, Storage, Real-time features, and unlimited custom services. Local to production workflow with automated SSL, intelligent defaults, and enterprise monitoring. 87 plugins (25 free, 62 Pro). MIT licensed core, forever.

```bash
curl -sSL https://install.nself.org | bash
```

**One command. Complete backend. Your infrastructure.**

[Quick Start](#quick-start) • [Features](#why-nself) • [Documentation](#documentation) • [Roadmap](.github/wiki/releases/ROADMAP.md)

</div>

---

## What is ɳSelf?

ɳSelf is a **complete self-hosted Backend-as-a-Service platform** that gives you the same powerful features as commercial services like Supabase and Nhost, but runs entirely on your own infrastructure.

**Get the power of commercial BaaS, plus:**
- **True Data Ownership** - Your data never leaves your infrastructure
- **Multi-Tenancy Built-In** - Enterprise SaaS features out of the box
- **Integrated Billing** - Stripe/Paddle integration included
- **Deploy Anywhere** - Your laptop, VPS, cloud, or Kubernetes
- **No Vendor Lock-In** - Standard open-source components
- **Complete Control** - Customize everything

From zero to production-ready backend in under 5 minutes. Really.

## Why ɳSelf?

### Fast Setup
- Under 5 minutes from zero to running backend
- One command installation, initialization, and deployment
- Smart defaults that work out of the box
- Interactive wizard or quick setup mode

### Complete Feature Set

**Core Backend Stack:**
- **PostgreSQL** - Production-ready database with pgvector, PostGIS, TimescaleDB
- **Hasura GraphQL** - Instant GraphQL API with permissions and subscriptions
- **Authentication** - JWT-based auth with 13 OAuth providers (Google, GitHub, Microsoft, etc.)
- **Storage (MinIO)** - S3-compatible object storage with CDN integration
- **Real-Time** - WebSocket channels, database subscriptions (CDC), presence tracking

**Enterprise Features (Unique to ɳSelf):**
- **Multi-Tenancy** - Complete tenant isolation, row-level security, org management
- **Billing Integration** - Stripe/Paddle subscriptions, usage tracking, invoicing
- **White-Label** - Custom domains, branding, email templates, legal documents
- **Migration Tools** - One-command migration from Supabase, Nhost, Firebase

**Production Infrastructure:**
- **Monitoring Bundle** - Prometheus, Grafana, Loki, Tempo (10 services)
- **Security Hardening** - `nself security` audit, setup, and status. SQL injection prevention, CSP framework, rate limiting
- **Automated Backups** - Intelligent pruning, cloud storage, 3-2-1 rule verification
- **SSL Automation** - Trusted certificates with zero configuration

**Developer Experience:**
- **40+ Service Templates** - Express, FastAPI, Flask, Gin, Rust, NestJS, gRPC, and more
- **25 Commands, 300+ Subcommands** - Complete control from the terminal
- **Admin GUI** (localhost) - Local management UI
- **Email Management** - 16+ providers with zero-config development mode

### ɳSelf vs Others

| Feature | ɳSelf | Supabase | Nhost | DIY |
|---------|:-----:|:--------:|:-----:|:---:|
| **Full Backend Stack** | :white_check_mark: | :white_check_mark: | :white_check_mark: | :warning: Manual |
| **Self-Hosted** | :white_check_mark: | :yellow_circle: Limited | :yellow_circle: Limited | :white_check_mark: |
| **Multi-Tenancy** | :white_check_mark: Built-in | :x: | :x: | :warning: DIY |
| **Built-in Billing** | :white_check_mark: Stripe/Paddle | :x: | :x: | :warning: DIY |
| **White-Label** | :white_check_mark: Complete | :x: | :x: | :warning: DIY |
| **Deploy Anywhere** | :white_check_mark: Any infra | :yellow_circle: Cloud-first | :yellow_circle: Cloud-first | :white_check_mark: |
| **Setup Time** | :white_check_mark: 5 minutes | :yellow_circle: 30+ min | :yellow_circle: 30+ min | :x: Hours/Days |
| **One Command Deploy** | :white_check_mark: | :x: | :x: | :x: |
| **Data Ownership** | :white_check_mark: Complete | :yellow_circle: Shared | :yellow_circle: Shared | :white_check_mark: |
| **Pricing** | :white_check_mark: Free core | :yellow_circle: Paid tiers | :yellow_circle: Paid tiers | :white_check_mark: Free |

## Prerequisites

- **macOS, Linux, or Windows with WSL**
- **Docker and Docker Compose** (installer helps install if needed)
- **curl** (for installation)

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://install.nself.org | bash
```

The installer will:
- Auto-detect existing installations and offer updates
- Check and help install Docker/Docker Compose if needed
- Download the nself binary to `~/.nself/bin`
- Add nself to your PATH automatically

### Alternative Methods

**macOS/Linux (Homebrew)**
```bash
brew tap nself-org/nself
brew install nself
```

**Direct from GitHub**
```bash
curl -fsSL https://raw.githubusercontent.com/nself-org/cli/main/.github/install.sh | bash
```

### Updating ɳSelf

```bash
nself update
```

Checks for new versions, shows a comparison, downloads and installs the update, and preserves your existing configurations.

## Quick Start

```bash
# 1. Create and enter project directory
mkdir my-backend && cd my-backend

# 2. Initialize with interactive wizard (or quick mode)
nself init --wizard  # Interactive setup
# or: nself init     # Quick setup with defaults

# 3. Build and launch everything
nself build && nself start
```

Your complete backend is now running at:
- GraphQL API: https://api.local.nself.org
- Auth Service: https://auth.local.nself.org
- Storage: https://storage.local.nself.org
- Email UI (dev): https://mail.local.nself.org
- Admin Dashboard: http://localhost:3021

All URLs work with automatic SSL, no browser warnings.

---

## What You Get

Every `nself start` brings up a complete backend stack:

**Core (always on):**
- PostgreSQL - primary database
- Hasura GraphQL - API and metadata engine
- Auth (nHost) - user authentication and JWTs
- Nginx - reverse proxy and SSL termination

**Optional services (enable per project):**
- Redis - caching, sessions, queues
- MinIO - S3-compatible object storage
- Search - full-text search (MeiliSearch or Typesense)
- Functions - serverless runtime
- Email - local mail (Mailpit in dev, your provider in prod)
- MLflow - ML experiment tracking
- Admin - local GUI at `localhost:3021`

Run `nself service list` to see all available services and which are enabled.

---

## Default Service URLs

When using the default `local.nself.org` domain (resolves to 127.0.0.1):

**Core Services:**
- **GraphQL API**: https://api.local.nself.org
- **Authentication**: https://auth.local.nself.org
- **Storage**: https://storage.local.nself.org
- **Storage Console**: https://storage-console.local.nself.org

**Optional Services (when enabled):**
- **Functions**: https://functions.local.nself.org
- **Email (MailPit)**: https://mail.local.nself.org
- **Search (MeiliSearch)**: https://search.local.nself.org
- **MLflow**: https://mlflow.local.nself.org

**Monitoring (when enabled):**
- **Grafana**: https://grafana.local.nself.org
- **Prometheus**: https://prometheus.local.nself.org

**Management:**
- **Admin UI**: http://localhost:3021

All `*.local.nself.org` domains automatically resolve to `127.0.0.1` for zero-config local development.

---

## Free vs Pro

The core CLI and 25 free plugins are MIT-licensed, free forever, including commercial use. Pro plugins require a membership key.

| Tier | Monthly | Annual | What's included |
|------|---------|--------|-----------------|
| Free | $0 | $0 | Core CLI + 25 free plugins |
| Basic | $0.99/mo | $9.99/yr | All 62 Pro plugins |
| Pro | $1.99/mo | $19.99/yr | Basic + AI suite (ai, claw, mux, voice, browser) |
| Elite | $4.99/mo | $49.99/yr | Pro + email support |
| Business | $9.99/mo | $99.99/yr | Elite + 24h support + priority feature requests |
| Business+ | $49.99/mo | $499.99/yr | Business + dedicated support channel |
| Enterprise | $99.99/mo | $999.99/yr | Business+ + managed DevOps |

Annual pricing is ~17% cheaper than monthly. Existing `$9.99/yr` keys grandfather to the Basic tier.

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install ai
nself plugin install livekit
```

---

## Email Configuration

### Development (Zero Config)
Email works out of the box with MailPit. All emails are captured locally:
- View emails: https://mail.local.nself.org
- No setup required
- Works for testing auth flows

### Production (2-Minute Setup)
```bash
nself service email config
```

Choose from 16+ providers:
- **SendGrid** - 100 emails/day free
- **AWS SES** - $0.10 per 1000 emails
- **Mailgun** - First 1000 emails free
- **Postmark** - Transactional email specialist
- **Gmail** - Use your personal/workspace account
- **Postfix** - Full control self-hosted server
- And 10+ more

---

## Customize Your Stack

Edit `.env` to enable additional services:

```bash
# Core settings
ENV=dev                         # or 'prod' for production
PROJECT_NAME=myapp
BASE_DOMAIN=local.nself.org

# Optional Services
REDIS_ENABLED=true              # Redis caching and queues
MINIO_ENABLED=true              # S3-compatible storage
FUNCTIONS_ENABLED=true          # Serverless functions
MLFLOW_ENABLED=true             # ML experiment tracking
MEILISEARCH_ENABLED=true        # Search engine

# Monitoring Bundle (10 services)
MONITORING_ENABLED=true         # Prometheus, Grafana, Loki, etc.

# Database Extensions
POSTGRES_EXTENSIONS=timescaledb,postgis,pgvector

# Custom Services (40+ templates)
CS_1=api:fastapi:3001           # Python FastAPI
CS_2=worker:bullmq-ts:3002      # Background jobs
CS_3=grpc:grpc:3003             # gRPC service
```

Then rebuild and restart:
```bash
nself build && nself restart
```

---

## Service Templates - 40+ Ready-to-Use Microservices

Add custom backend services with one line in your `.env`:

```bash
SERVICES_ENABLED=true
CS_1=api:fastapi:3001           # Python FastAPI
CS_2=auth:nest-ts:3002          # TypeScript NestJS
CS_3=jobs:bullmq-ts:3003        # Background jobs (BullMQ)
CS_4=ml:ray:3004                # ML model serving (Ray)
CS_5=chat:socketio-ts:3005      # Real-time WebSocket
```

### Available Templates by Language

- **JavaScript/TypeScript (19)**: Express, Fastify, NestJS, Hono, Socket.IO, BullMQ, Temporal, Bun, Deno, tRPC
- **Python (7)**: Flask, FastAPI, Django REST, Celery, Ray, AI Agents (LLM and Data)
- **Go (4)**: Gin, Echo, Fiber, gRPC
- **Other (10)**: Rust, Java, C#, C++, Ruby, Elixir, PHP, Kotlin, Swift

Every template includes:
- Production Docker setup with multi-stage builds
- Security headers and CORS configuration
- Health checks and graceful shutdown
- Language-specific optimizations
- Template variables for customization

---

## Command Reference

ɳSelf provides a **25-command canonical runtime surface** with **300+ subcommands** organized by domain:

### Core Commands
```bash
nself init          # Initialize project with wizard
nself build         # Generate Docker configs
nself start         # Start services
nself stop          # Stop services
nself restart       # Restart services
```

### Utilities
```bash
nself status        # Service health status
nself logs          # View service logs
nself admin         # Open admin UI
nself urls          # List all service URLs
nself doctor        # System diagnostics
nself monitor       # Monitoring dashboards
nself health        # Health checks
nself version       # Version info
nself update        # Update nself
```

### Advanced Commands
```bash
nself db            # Database operations (migrate, seed, backup, restore, shell, reset)
nself service       # Service management (enable, disable, list)
nself config        # Configuration (show, get, set, list, validate, export, import)
nself plugin        # Plugin system (install, remove, update, list, start, stop, status)
nself ssl           # SSL certificate management (status, renew [domain])
nself security      # Security audit, setup, and status
nself license       # Pro license management (set, show, validate, clear, upgrade)
nself doctor        # System diagnostics (--fix for auto-repair)
nself health        # Health checks (check, watch, history)
nself migrate       # Detect and migrate v0.x projects
```

Run `nself help <command>` for subcommand details.

---

## Plugin System

**25 free MIT plugins** - no key required:

```bash
nself plugin install monitoring    # Prometheus + Grafana + Loki
nself plugin install cron          # Scheduled jobs
nself plugin install notify        # Push notifications
nself plugin install search        # MeiliSearch full-text
nself plugin install commerce      # Stripe billing
```

**62 Pro plugins** - requires a membership key (Basic tier, $9.99/yr):

```bash
nself license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
nself plugin install ai            # LLM inference (OpenAI, Anthropic, local)
nself plugin install claw          # AI assistant backend
nself plugin install livekit       # Live video and audio
nself plugin install mux           # Email pipeline with AI routing
nself plugin install voice         # Voice synthesis and transcription
```

See [Plugins](.github/wiki/Plugin-Architecture.md) for the full list.

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                     ɳSelf CLI                       │
│          init  →  build  →  start                   │
└────────────────────────┬────────────────────────────┘
                         │ generates
                         ▼
              docker-compose.yml + nginx.conf
                         │ starts
                         ▼
┌──────────┐ ┌────────┐ ┌──────┐ ┌───────┐
│ Postgres │ │ Hasura │ │ Auth │ │ Nginx │   ← always on
└──────────┘ └────────┘ └──────┘ └───────┘
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
┌──────────────┐ ┌────────────┐ ┌──────────────┐
│ Redis, MinIO │ │  Search,   │ │   Plugins    │ ← optional
│ Email, Funcs │ │   MLflow   │ │  (77 total)  │
└──────────────┘ └────────────┘ └──────────────┘
```

All services bind to `127.0.0.1`. Nginx is the only external-facing process.

---

## SSL/TLS Configuration

ɳSelf provides automatic SSL with green locks in browsers, no warnings.

### Automatic Certificate Generation

```bash
nself build              # Automatically generates SSL certificates
```

Your browser will show green locks for:
- https://localhost, https://api.localhost
- https://local.nself.org, https://api.local.nself.org

### Two Domain Options (Both Work)

1. **`*.localhost`** - Works offline, no DNS needed
2. **`*.local.nself.org`** - Loopback domain (resolves to 127.0.0.1)

### SSL Commands

```bash
nself ssl status         # Show certificate expiry, covered domains, and validity
nself ssl renew          # Reload nginx with existing certificates
nself ssl renew <domain> # Reload nginx and run certbot renewal for a domain
```

---

## Backup and Restore

### Database Backup and Restore

```bash
# Create a pg_dump backup (timestamped filename)
nself db backup

# Backup to a specific file
nself db backup /tmp/mybackup.sql

# List available backups
nself db backup list

# Restore from a backup
nself db restore /tmp/mybackup.sql
```

---

## Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| Docker | 24+ | latest |
| macOS | 12 (Monterey) | 14+ |
| Linux | Ubuntu 20.04 / Debian 11 | Ubuntu 22.04+ |
| RAM | 2 GB | 4 GB |
| Disk | 5 GB free | 10 GB free |

Docker must be running before any `nself` command that starts services.

---

## Configuration

Config lives in `.env` (dev), `.env.staging`, and `.env.prod`. The `nself build` command reads the active env file and generates `docker-compose.yml` and nginx config.

Never hand-edit `docker-compose.yml` directly. All changes go through the env file, then `nself build`.

```bash
nself config show        # Show current config
nself config set POSTGRES_PORT 5433
nself config validate    # Validate before building
```

### Environment File Priority

Files loaded in order (later files override earlier):
1. `.env.dev` - Team defaults (always loaded)
2. `.env.staging` - Staging environment (if ENV=staging)
3. `.env.prod` - Production environment (if ENV=prod)
4. `.env.secrets` - Production secrets (if ENV=prod)
5. `.env` - Local overrides (highest priority)

---

## Environments

| Environment | How to activate | Typical use |
|-------------|----------------|-------------|
| `dev` | default | Local development |
| `staging` | `nself start --env staging` | Pre-production testing |
| `prod` | `nself start --env prod` | Production server |

Your `.env.prod` should never be checked into source control. Use your secrets manager to inject it on the server.

---

## Production Deployment

```bash
# 1. Copy your project to the server
rsync -az --exclude .volumes/ ./ user@server:/opt/myapp/

# 2. On the server: set production environment and start
ssh user@server
cd /opt/myapp
nself start --env prod

# 3. Check service health
nself status
nself health check
```

### Production Checklist
1. Set `ENV=prod` (automatically configures security settings)
2. Use strong passwords (12+ characters, auto-generated)
3. Configure your custom domain
4. Enable Let's Encrypt SSL
5. Set up automated backups
6. Configure monitoring alerts

---

## Project Structure

After running `nself build`:

```
my-backend/
├── .env.dev               # Team defaults
├── .env.staging           # Staging environment (optional)
├── .env.prod              # Production environment (optional)
├── .env.secrets           # Production secrets (optional)
├── .env                   # Local configuration (highest priority)
├── docker-compose.yml     # Generated Docker Compose file
├── nginx/                 # Nginx configuration
│   ├── nginx.conf
│   ├── conf.d/           # Service routing
│   └── ssl/              # SSL certificates
├── postgres/             # Database initialization
│   └── init/
├── hasura/               # GraphQL configuration
│   ├── metadata/
│   └── migrations/
├── functions/            # Optional serverless functions
└── services/             # Custom services (if enabled)
    ├── api/              # CS_1 service
    ├── worker/           # CS_2 service
    └── grpc/             # CS_3 service
```

---

## Database Management

### For Lead Developers
```bash
nano schema.dbml                    # Design your schema
nself db run                        # Generate migrations from schema
nself db migrate:up                 # Test migrations locally
git add schema.dbml hasura/migrations/
git commit -m "Add new tables"
```

### For All Developers
```bash
git pull
nself start
# If you see "DATABASE MIGRATIONS PENDING" warning:
nself db update                     # Safely apply migrations
```

### Database Commands
```bash
nself db                    # Show all database commands
nself db migrate up         # Apply pending migrations
nself db migrate down       # Revert last migration
nself db migrate status     # Show migration status
nself db migrate create <name>  # Create new migration file
nself db seed               # Apply seed data
nself db backup             # Create pg_dump backup (timestamped)
nself db backup list        # List available backups with size and date
nself db restore <file>     # Restore database from backup file
nself db shell              # Open interactive psql shell
nself db drop               # Drop the project database (DESTRUCTIVE)
nself db reset              # Drop and recreate database (DESTRUCTIVE)
```

---

## Admin GUI

```bash
nself admin start
```

Runs at `http://localhost:3021`. Features:
- Service health monitoring with real-time status
- Docker management: start, stop, restart containers
- Database query interface
- Log viewer with filtering and search
- Backup management
- Configuration editor

Local only. Never deployed to a server.

---

## Migrating from v1

```bash
nself migrate detect   # Scan for v1 artifacts (read-only)
nself migrate run      # Migrate with automatic backup
nself migrate rollback # Restore the most recent backup
```

---

## Troubleshooting

**Services not starting?**
```bash
nself doctor            # Run diagnostics
nself logs [service]    # Check service logs
nself status            # Check service status
```

**Port conflicts?**
Edit port numbers in `.env` and rebuild:
```bash
nself build && nself restart
```

**SSL certificate warnings?**
```bash
nself ssl status        # Check certificate status
nself ssl renew         # Reload nginx with existing certificates
```

**Email test not working?**
```bash
nself service email test recipient@example.com
```

**Build command hangs?**
```bash
nself build --force     # Force rebuild
```

---

## Documentation

### Getting Started
- [Quick Start Tutorial](.github/wiki/getting-started/Quick-Start.md) - 5-minute tutorial
- [Installation Guide](.github/wiki/getting-started/Installation.md) - Detailed installation
- [Configuration Reference](.github/wiki/configuration/README.md) - Complete .env settings
- [Command Reference](.github/wiki/commands/COMMAND-TREE.md) - All 300+ commands

### Features and Services
- [Service Templates](.github/wiki/reference/SERVICE_TEMPLATES.md) - 40+ microservice templates
- [Database Workflow](.github/wiki/guides/DATABASE-WORKFLOW.md) - DBML to production
- [Multi-Tenancy](.github/wiki/architecture/MULTI-TENANCY.md) - Enterprise SaaS features
- [Email Setup](.github/wiki/commands/EMAIL.md) - 16+ provider configuration

### Deployment and Operations
- [Production Deployment](.github/wiki/deployment/README.md) - Production guide
- [Backup Guide](.github/wiki/guides/BACKUP_GUIDE.md) - Comprehensive backup system
- [Monitoring Setup](.github/wiki/guides/MONITORING-COMPLETE.md) - Grafana dashboards
- [Security Hardening](.github/wiki/security/README.md) - Security best practices

### Migration and Compatibility
- [Migration from Supabase](.github/wiki/migrations/FROM-SUPABASE.md) - Step-by-step migration
- [Migration from Nhost](.github/wiki/migrations/FROM-NHOST.md) - One-command migration
- [Migration from Firebase](.github/wiki/migrations/FROM-FIREBASE.md) - Auth and Firestore

---

## Contributing

See [Contributing](.github/wiki/contributing/CONTRIBUTING.md).

The CLI is written in Go (1.25+). Tests live in `internal/` alongside the packages they test. Run `make test` from the repo root.

---

## Perfect For

- **Startups** - Get your backend up fast, scale when you need to
- **Agencies** - Standardized backend setup for all client projects
- **Enterprises** - Self-hosted solution with full control
- **Side Projects** - Production-grade infrastructure without the complexity
- **Learning** - See how modern backends work under the hood

---

## License

MIT, free for personal and commercial use. See [LICENSE](LICENSE).

ɳSelf CLI is MIT licensed. The 62 Pro plugins are source-available under a separate commercial license.

---

## Links

- [Official Website](https://nself.org) - Project homepage
- [Documentation](https://docs.nself.org) - Complete documentation
- [GitHub Repository](https://github.com/nself-org/cli) - Source code
- [Report Issues](https://github.com/nself-org/cli/issues) - Bug reports and feedback
- [Discussions](https://github.com/nself-org/cli/discussions) - Community discussions

---

<div align="center">

**ɳSelf CLI v1.0.3** - Built by [nself](https://nself.org) - [GitHub](https://github.com/nself-org/cli)

[Get Started](#quick-start) - [Documentation](#documentation) - [Roadmap](.github/wiki/releases/ROADMAP.md)

</div>
