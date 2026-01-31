# ɳSelf - Complete Self-Hosted Backend Platform

[![Version](https://img.shields.io/badge/version-0.9.5-blue.svg)](https://github.com/acamarata/nself/releases)
[![Status](https://img.shields.io/badge/status-production--ready-green.svg)](#-important-note)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey.svg)](https://github.com/acamarata/nself#-supported-platforms)
[![Docker](https://img.shields.io/badge/docker-required-blue.svg)](https://www.docker.com/get-started)
[![CI Status](https://github.com/acamarata/nself/actions/workflows/ci.yml/badge.svg)](https://github.com/acamarata/nself/actions)
[![Security Scan](https://github.com/acamarata/nself/actions/workflows/security-scan.yml/badge.svg)](https://github.com/acamarata/nself/actions/workflows/security-scan.yml)
[![License](https://img.shields.io/badge/license-Personal%20Free%20%7C%20Commercial-green.svg)](LICENSE)

> **✅ PRODUCTION READY**: ɳSelf v0.9.5 is production-ready! With 60% test coverage (445 tests), enterprise security hardening, complete feature parity with Supabase/Nhost, and battle-tested infrastructure, ɳSelf is ready for production deployment. Security fixes include SQL injection prevention, Content Security Policy framework, and automated dependency scanning.
>
> ɳSelf provides a complete orchestration layer that generates Docker Compose configurations and manages services - the same tasks required for any self-hosted backend (Nhost, Supabase, or DIY). The underlying services (PostgreSQL, Hasura, etc.) are production-grade; ɳSelf adds enterprise security, multi-tenancy, billing, and 150+ CLI commands on top.
>
> We welcome bug reports and contributions as we continue toward v1.0!

---

Deploy a feature-complete backend infrastructure on your own servers with PostgreSQL, Hasura GraphQL, Redis, Auth, Storage, and optional microservices. Works seamlessly across local development, staging, and production with automated SSL, smart defaults, and production-ready configurations with enterprise monitoring.

**Based on [Nhost.io](https://nhost.io) for self-hosting!** and expanded with more features. Copy the below command in Terminal to install and get up and running in seconds!

```bash
curl -sSL https://install.nself.org | bash
```

> **🎉 v0.9.5 - Feature Parity & Security Hardening!**: Complete real-time communication system (WebSocket, CDC, presence), enhanced OAuth with 13 providers, SQL injection fixes, Content Security Policy framework, Typesense search integration, intelligent backup pruning, migration tools from Supabase/Nhost/Firebase, and 368 new tests. [See v0.9.5 release](docs/releases/v0.9.5.md) | [View all releases](https://github.com/acamarata/nself/releases)

📋 **[View Roadmap](docs/ROADMAP.md)** - See development roadmap and future releases!

ɳSelf is *the* CLI for Nhost self-hosted deployments - with extras and an opinionated setup that makes everything smooth. From zero to production-ready backend in under 5 minutes. Just edit an env file with your preferences and build!

## 🚀 Why ɳSelf?

### ⚡ Lightning Fast Setup
- **Under 5 minutes** from zero to running backend
- One command installation, initialization, and deployment
- Smart defaults that just work™

### 🎯 Complete Feature Set
- **Full Nhost Stack**: PostgreSQL, Hasura GraphQL, Auth, Storage, Functions
- **Plus Extras**: Redis, TimescaleDB, PostGIS, pgvector extensions
- **Real-Time System (NEW v0.9.5)**: WebSocket channels, Database CDC subscriptions, Presence tracking, Broadcast messaging
- **OAuth Providers (ENHANCED v0.9.5)**: 13 providers (Google, GitHub, Microsoft, Facebook, Apple, Slack, Discord, Twitch, Twitter, LinkedIn, GitLab, Bitbucket, Spotify) with PKCE support
- **Multi-Tenancy**: Complete tenant isolation, organization management, role-based access, per-tenant billing
- **Security Hardening (NEW v0.9.5)**: SQL injection prevention, Content Security Policy framework, Dependency scanning (ShellCheck, Gitleaks, Trivy, Semgrep)
- **Migration Tools (NEW v0.9.5)**: One-command migration from Supabase, Nhost, and Firebase
- **Backup System (ENHANCED v0.9.5)**: Intelligent pruning (age, count, size, GFS), 3-2-1 rule verification, cloud backup support
- **Enterprise Search**: MeiliSearch, Typesense, Elasticsearch, OpenSearch, Zinc, Sonic with provider switching
- **Plugin Ecosystem**: Extensible architecture with marketplace integration
- **Email Management**: 16+ providers (SendGrid, AWS SES, Mailgun, etc.) with zero-config dev
- **40+ Service Templates**: Express, FastAPI, Flask, Gin, Rust, NestJS, Socket.IO, Celery, Ray, and more
- **Microservices Ready**: Production-ready templates for JS/TS, Python, Go, Rust, Java, C#, Ruby, Elixir, PHP
- **Serverless Functions**: Built-in functions runtime with hot reload and deployment
- **ML Platform**: MLflow integration for experiment tracking and model registry
- **Production SSL**: Automatic trusted certificates (no browser warnings!)

### 🛠️ Developer Experience
- **150+ CLI Commands**: Complete feature control from terminal
- **Admin Dashboard**: Web-based monitoring UI at localhost:3021
- **Developer Console**: Interactive development console with live REPL
- **Real-Time CLI (NEW v0.9.5)**: 40+ commands for channels, presence, subscriptions, broadcast
- **Security Audit (NEW v0.9.5)**: One-command security assessment with auto-fix
- **Local Tunneling**: Expose local services to internet for testing
- **API Mocking**: Mock external APIs for faster development
- **Single Config File**: One `.env` controls everything
- **Zero Configuration**: Email, SSL, and services work out of the box
- **Automated SSL**: Certificates generated automatically (one-time sudo for trust)
- **Smart Domains**: Use local.nself.org (zero config) or localhost with auto-SSL
- **Hot Reload**: Changes apply instantly without rebuild
- **Multi-Environment**: Same setup works locally, staging, and production
- **No Lock-in**: Standard Docker Compose under the hood
- **Debugging Tools**: `doctor`, `status`, `logs`, `observe` commands for troubleshooting
- **445 Automated Tests**: 60% code coverage ensuring reliability

### 🔐 Production Ready
- **Security Hardened (NEW v0.9.5)**: SQL injection prevention, Content Security Policy, OWASP Top 10 compliance
- **Dependency Scanning (NEW v0.9.5)**: Automated ShellCheck, Gitleaks, Trivy, Semgrep in CI/CD
- **Security Audit System (NEW v0.9.5)**: One-command production readiness check with auto-fix
- **Safe Destruction (NEW v0.9.6)**: Intelligent infrastructure teardown with multi-level safety and selective removal
- **Server Management (NEW v0.9.6)**: Complete VPS lifecycle management - init, health checks, diagnostics, SSH
- **Kubernetes Abstraction (NEW v0.9.6)**: Deploy to 8 cloud providers with unified commands (AWS, GCP, Azure, DO, Linode, Vultr, Hetzner, Scaleway)
- **Automated SSL**: Certificates generated and trusted automatically (mkcert or Let's Encrypt)
- **Security Scanning**: Automated vulnerability detection and auditing
- **Firewall Management**: Simplified network security configuration
- **Email Ready**: Production email in 2 minutes with 16+ provider support
- **Battle Tested**: Based on proven Nhost.io infrastructure with enterprise enhancements
- **Multi-Tenant Ready**: Enterprise-grade tenant isolation with Row Level Security
- **445 Tests**: Comprehensive test coverage (60%+) ensuring reliability
- **Scale Ready**: From hobby projects to enterprise deployments
- **Zero Downtime**: Rolling updates and health checks built-in
- **Migration Ready (NEW v0.9.5)**: Migrate from Supabase, Nhost, or Firebase in minutes

## 📋 Prerequisites

- **Bash 3.2+** (default on macOS and most Linux distributions)
- **Linux, macOS, or Windows with WSL**
- **Docker and Docker Compose** (installer will help install these)
- **curl** (for installation)

**Note:** nself is fully compatible with Bash 3.2+, the default shell on macOS. No Bash upgrade needed!

## 🔧 Installation

### Quick Install (Recommended)

```bash
curl -sSL https://install.nself.org | bash
```

### Alternative Methods

#### Package Managers

**macOS/Linux (Homebrew)**
```bash
brew tap acamarata/nself
brew install nself
```

**Direct from GitHub**
```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

**Docker**
```bash
docker pull acamarata/nself:latest
docker run -it acamarata/nself:latest version
```

The installer will:
- ✅ Auto-detect existing installations and offer updates
- 📊 Show visual progress with loading spinners
- 🔍 Check and help install Docker/Docker Compose if needed
- 📦 Download nself CLI to `~/.nself/bin`
- 🔗 Add nself to your PATH automatically
- 🚀 Create a global `nself` command

### Updating ɳSelf

To update to the latest version:

```bash
nself update
```

The updater will:
- Check for new versions automatically
- Show version comparison (current → latest)
- Download and install updates seamlessly
- Preserve your existing configurations

## 🏁 Quick Start - 3 Commands to Backend Bliss

```bash
# 1. Create and enter project directory
mkdir my-awesome-backend && cd my-awesome-backend

# 2. Initialize with smart defaults (or use wizard)
nself init --wizard  # Interactive setup (NEW in v0.3.9)
# or: nself init     # Quick setup with defaults

# 3. Build and launch everything
nself build && nself start
# URLs for enabled services will be shown in the output
```

**That's it!** Your complete backend is now running at:
- 🚀 GraphQL API: https://api.local.nself.org
- 🔐 Auth Service: https://auth.local.nself.org
- 📦 Storage: https://storage.local.nself.org
- 📊 And more...

*Tip:* These URLs are also printed after `nself build` and `nself start` so they're easy to copy.

## 📧 Email Configuration

### Development (Zero Config)
Email works out of the box with MailPit - all emails are captured locally:
- 📧 View emails: https://mail.local.nself.org
- 🔧 No setup required
- 📨 Perfect for testing auth flows

### Production (2-Minute Setup)
```bash
nself service email config
```

> **Note:** v0.9.6+ uses consolidated v1.0 command structure. See [Command Tree](docs/commands/COMMAND-TREE-V1.md) for details.

Choose from 16+ providers:
- **SendGrid** - 100 emails/day free
- **AWS SES** - $0.10 per 1000 emails  
- **Mailgun** - First 1000 emails free
- **Postmark** - Transactional email specialist
- **Gmail** - Use your personal/workspace account
- **Postfix** - Full control self-hosted server
- And 10+ more!

The wizard guides you through everything. Example for SendGrid:
```bash
nself service email config sendgrid
# Add your API key to .env
nself build && nself restart
```

### Want to customize?

Edit `.env` to enable extras:
```bash
# Core settings at the top
ENV=dev                         # or 'prod' for production
PROJECT_NAME=myapp
BASE_DOMAIN=local.nself.org
DB_ENV_SEEDS=true               # Use dev/prod seed separation

# Enable all the goodies
REDIS_ENABLED=true              # Redis caching
NESTJS_ENABLED=true             # NestJS microservices
FUNCTIONS_ENABLED=true          # Serverless functions
POSTGRES_EXTENSIONS=timescaledb,postgis,pgvector  # DB superpowers
```

Then rebuild and restart:
```bash
nself build && nself restart
```

## 🚀 Service Templates - 40+ Ready-to-Use Microservices

Add custom backend services with one line:

```bash
# Enable custom services
SERVICES_ENABLED=true

# Add microservices (examples)
CS_1=api:fastapi:3001:/api        # Python FastAPI
CS_2=auth:nest-ts:3002:/auth      # TypeScript NestJS
CS_3=jobs:bullmq-ts:3003          # Background jobs
CS_4=ml:ray:3004:/models          # ML model serving
CS_5=chat:socketio-ts:3005        # Real-time WebSocket
```

### Available Templates by Language

- **JavaScript/TypeScript (19)**: Node.js, Express, Fastify, NestJS, Hono, Socket.IO, BullMQ, Temporal, Bun, Deno, tRPC
- **Python (7)**: Flask, FastAPI, Django REST, Celery, Ray, AI Agents (LLM & Data)
- **Go (4)**: Gin, Echo, Fiber, gRPC
- **Other (10)**: Rust, Java, C#, C++, Ruby, Elixir, PHP, Kotlin, Swift

**📖 [View Complete Service Templates Documentation](docs/services/SERVICE_TEMPLATES.md)**

Every template includes:
- 🐳 Production Docker setup with multi-stage builds
- 🛡️ Security headers and CORS configuration  
- 📊 Health checks and graceful shutdown
- ⚡ Language-specific optimizations
- 🔧 Template variables for customization

## 💪 What You Get vs Manual Setup

| Manual Nhost Self-hosting | With ɳSelf |
|--------------------------|------------|
| Hours of configuration | 5 minutes total |
| Multiple config files | Single `.env` |
| Complex networking setup | Automatic service discovery |
| Manual SSL certificates | Automatic HTTPS everywhere |
| Separate service installs | One command, all services |
| Production passwords? 🤷 | `nself prod` generates secure ones |
| Hope it works | Battle-tested configurations |

## 📚 Commands

### Version Status
- **✅ v1.0.0 (Current)**: 32 top-level commands with 295+ subcommands - Production-ready with complete feature set
- **🔮 v1.1.0 (Next)**: Performance optimizations, enhanced plugin marketplace
- **🎯 v2.0.0**: Advanced AI features, distributed deployment

### New in v0.9.6
- **`destroy`** - Safe infrastructure destruction with selective targeting
- **`deploy server`** - Complete server lifecycle management (10 subcommands)
- **`deploy sync`** - Environment synchronization (pull, push, full)
- **`infra provider k8s-*`** - Unified Kubernetes management across 8 cloud providers

### Complete Command Tree (v1.0)

> **v0.9.6 Consolidation:** Old commands like `nself billing`, `nself org`, `nself staging`, etc. have been consolidated into this streamlined structure. See [Command Consolidation Map](docs/architecture/COMMAND-CONSOLIDATION-MAP.md) for the full mapping.

```
nself (32 top-level commands, 295+ subcommands)
├── 🚀 Core Commands (5)
│   ├── init          Initialize project with wizard
│   ├── build         Generate Docker configs
│   ├── start         Start services
│   ├── stop          Stop services
│   └── restart       Restart services
│
├── 🗑️  Infrastructure Management (1)
│   └── destroy       Safe infrastructure destruction
│
├── 📊 Utilities (15)
│   ├── status        Service health status
│   ├── logs          View service logs
│   ├── help          Help system
│   ├── admin         Admin UI
│   ├── urls          Service URLs
│   ├── exec          Execute in container
│   ├── doctor        System diagnostics
│   ├── monitor       Monitoring dashboards
│   ├── health        Health checks
│   ├── version       Version info
│   ├── update        Update nself
│   ├── completion    Shell completions
│   ├── metrics       Metrics & profiling
│   ├── history       Audit trail
│   └── audit         Audit logging
│
└── 🎯 Other Commands (11)
    ├── db            Database operations (11 subcommands)
    ├── tenant        Multi-tenancy (50+ subcommands)
    │   ├── billing   → Billing management (was: nself billing)
    │   └── org       → Organization management (was: nself org)
    ├── deploy        Deployment (33 subcommands)
    │   ├── staging   → Deploy to staging (was: nself staging)
    │   ├── production → Deploy to production (was: nself prod)
    │   ├── upgrade   → Upgrade deployment (was: nself upgrade)
    │   ├── server    → Server management - 10 subcommands (was: nself servers)
    │   │   ├── init      Initialize VPS for nself
    │   │   ├── check     Verify server readiness
    │   │   ├── status    Quick status of all servers
    │   │   ├── diagnose  Comprehensive diagnostics
    │   │   ├── list      List configured servers
    │   │   ├── add       Add server configuration
    │   │   ├── remove    Remove server configuration
    │   │   ├── ssh       Quick SSH connection
    │   │   ├── info      Display server details
    │   │   └── sync      → Use 'nself deploy sync' instead
    │   └── sync      → Environment sync - 4 subcommands (was: nself sync)
    │       ├── pull      Pull config from remote
    │       ├── push      Push config to remote
    │       ├── status    Show sync status
    │       └── full      Complete synchronization
    ├── infra         Infrastructure (48 subcommands)
    │   ├── provider  → Cloud providers (was: nself cloud/provider)
    │   │   ├── k8s-create     Create managed K8s cluster
    │   │   ├── k8s-delete     Delete managed K8s cluster
    │   │   ├── k8s-kubeconfig Get kubeconfig credentials
    │   │   └── ... (8 providers: AWS, GCP, Azure, DO, Linode, Vultr, Hetzner, Scaleway)
    │   ├── k8s       → Kubernetes operations (was: nself k8s)
    │   └── helm      → Helm charts (was: nself helm)
    ├── service       Service management (43 subcommands)
    │   ├── storage   → Storage service (was: nself storage)
    │   ├── email     → Email service (was: nself email)
    │   ├── search    → Search service (was: nself search)
    │   ├── redis     → Redis cache (was: nself redis)
    │   ├── functions → Serverless functions (was: nself functions)
    │   ├── mlflow    → ML tracking (was: nself mlflow)
    │   └── realtime  → Real-time features (was: nself realtime)
    ├── config        Configuration (20 subcommands)
    │   ├── env       → Environment management (was: nself env)
    │   ├── secrets   → Secrets management (was: nself secrets)
    │   ├── vault     → Vault integration (was: nself vault)
    │   └── validate  → Config validation (was: nself validate)
    ├── auth          Security (38 subcommands)
    │   ├── mfa       → Multi-factor auth (was: nself mfa)
    │   ├── roles     → Role management (was: nself roles)
    │   ├── devices   → Device management (was: nself devices)
    │   ├── oauth     → OAuth providers (was: nself oauth)
    │   ├── security  → Security scanning (was: nself security)
    │   ├── ssl       → SSL management (was: nself ssl/trust)
    │   ├── rate-limit → Rate limiting (was: nself rate-limit)
    │   └── webhooks  → Webhook management (was: nself webhooks)
    ├── perf          Performance (5 subcommands)
    │   ├── bench     → Benchmarking (was: nself bench)
    │   ├── scale     → Scaling (was: nself scale)
    │   └── migrate   → Migration tools (was: nself migrate)
    ├── backup        Backup & recovery (6 subcommands)
    │   ├── rollback  → Rollback changes (was: nself rollback)
    │   ├── reset     → Reset state (was: nself reset)
    │   └── clean     → Clean resources (was: nself clean)
    ├── dev           Developer tools (16 subcommands)
    │   ├── frontend  → Frontend management (was: nself frontend)
    │   ├── ci        → CI/CD config (was: nself ci)
    │   ├── docs      → Documentation (was: nself docs)
    │   └── whitelabel → White-label config (was: nself whitelabel)
    └── plugin        Plugin system (8+ subcommands)
        ├── install   → Install plugins
        ├── list      → List plugins
        └── update    → Update plugins
```

## 🎯 Admin Dashboard

### Web-Based Monitoring Interface
The new admin dashboard provides complete visibility and control over your nself stack:

- **Service Health Monitoring**: Real-time status of all containers
- **Docker Management**: Start, stop, restart containers from UI
- **Database Query Interface**: Execute SQL queries directly
- **Log Viewer**: Filter and search through service logs
- **Backup Management**: Create and restore backups via UI
- **Configuration Editor**: Modify settings without SSH

### Quick Setup
```bash
# Open admin UI in browser
nself admin

# Or open in development mode
nself admin --dev

# Served at localhost:3021
```

## 📚 Documentation

### Core Documentation
- **[Commands Reference](docs/commands/COMMAND-TREE-V1.md)** - Complete v1.0 command tree
- **[Command Mapping](docs/architecture/COMMAND-CONSOLIDATION-MAP.md)** - Old → New command reference
- **[Release Notes](docs/releases/INDEX.md)** - Latest features and fixes
- **[Roadmap](docs/releases/ROADMAP.md)** - Development roadmap and upcoming features
- **[Architecture](docs/architecture/README.md)** - System architecture and design
- **[Troubleshooting](docs/guides/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Changelog](docs/releases/CHANGELOG.md)** - Version history
- **[All Releases](docs/releases/INDEX.md)** - Complete release history

### New Feature Guides (v0.9.6)
- **[Destroy Command](docs/commands/DESTROY.md)** - Safe infrastructure destruction
- **[Server Management](docs/deployment/SERVER-MANAGEMENT.md)** - Complete VPS lifecycle management
- **[Kubernetes Guide](docs/infrastructure/K8S-IMPLEMENTATION-GUIDE.md)** - Multi-cloud K8s abstraction layer

### Quick Reference

### Email Commands (v1.0)
| Command | Description |
|---------|-------------|
| `nself service email send` | Send email interactively |
| `nself service email config` | Configure email provider |
| `nself service email test` | Test email configuration |
| `nself service email template` | Manage email templates |

## 🌐 Default Service URLs

When using the default `local.nself.org` domain:

- **GraphQL API**: https://api.local.nself.org
- **Authentication**: https://auth.local.nself.org
- **Storage**: https://storage.local.nself.org
- **Storage Console**: https://storage-console.local.nself.org
- **Functions** (if enabled): https://functions.local.nself.org
- **Email** (development): https://mail.local.nself.org - MailPit email viewer
- **Admin UI**: http://localhost:3021 - Admin dashboard
- **Dashboard** (if enabled): https://dashboard.local.nself.org

All `*.nself.org` domains resolve to `127.0.0.1` for local development.

## 💡 Hello World Example

The included hello world example shows all services working together:

```bash
# Enable all services in .env
SERVICES_ENABLED=true
REDIS_ENABLED=true
NESTJS_ENABLED=true
NESTJS_SERVICES=weather-actions
BULLMQ_ENABLED=true
BULLMQ_WORKERS=weather-processor,currency-processor
GOLANG_ENABLED=true
GOLANG_SERVICES=currency-fetcher
PYTHON_ENABLED=true
PYTHON_SERVICES=data-analyzer
```

**Architecture:**
- **NestJS**: Hasura actions for weather API integration
- **BullMQ**: Background processing of weather/currency data
- **GoLang**: High-performance currency rate fetching
- **Python**: ML predictions on time-series data

All services communicate through:
- Shared PostgreSQL database (with TimescaleDB)
- Redis for queuing and caching
- Direct HTTP calls via Docker network

## 🔧 Core Services

### Required Services
- **PostgreSQL**: Primary database with optional extensions
- **Hasura GraphQL**: Instant GraphQL API for your database
- **Hasura Auth**: JWT-based authentication service
- **MinIO**: S3-compatible object storage
- **Nginx**: Reverse proxy with SSL termination

### Optional Services
- **Redis**: In-memory caching and queue management
- **Nhost Functions**: Serverless functions support
- **Nhost Dashboard**: Admin interface for managing your backend
- **MailHog**: Email testing for development
- **NestJS Run Service**: For constantly running microservices

### Backend Services Framework
- **NestJS Services**: TypeScript/JavaScript microservices for Hasura actions
- **BullMQ Workers**: Queue workers for background job processing
- **GoLang Services**: High-performance microservices
- **Python Services**: ML/AI and data analysis services

## 🔐 SSL/TLS Configuration

nself provides bulletproof SSL with green locks in browsers - no warnings!

### Two Domain Options (Both Work Perfectly)

1. **`*.localhost`** - Works offline, no DNS needed
2. **`*.local.nself.org`** - Our loopback domain (resolves to 127.0.0.1)

### Automatic Certificate Generation

```bash
nself build              # Automatically generates SSL certificates
nself auth ssl trust     # Install root CA for green locks (one-time)
```

That's it! Your browser will show green locks for:
- https://localhost, https://api.localhost, etc.
- https://local.nself.org, https://api.local.nself.org, etc.

### Advanced: Public Wildcard Certificates

For teams or CI/CD, get globally-trusted certificates (no `nself auth ssl trust` needed):

```bash
# Add to .env
DNS_PROVIDER=cloudflare        # or route53, digitalocean
DNS_API_TOKEN=your_api_token

# Generate public wildcard
nself auth ssl generate
```

Supported DNS providers:
- Cloudflare (recommended)
- AWS Route53
- DigitalOcean
- And more via acme.sh

### SSL Commands (v1.0)

| Command | Description |
|---------|-------------|
| `nself auth ssl generate` | Generate SSL certificates |
| `nself auth ssl renew` | Renew certificates |
| `nself auth ssl info` | Check certificate status |
| `nself auth ssl trust` | Install root CA to system |
| `nself auth ssl install` | Install certificate |

## 💾 Backup & Restore

### Comprehensive Backup System

nself includes enterprise-grade backup capabilities with cloud storage support:

```bash
# Create backups
nself backup create              # Full backup (database, config, volumes)
nself backup create database     # Database only
nself backup create config       # Configuration only

# Restore from backup
nself backup restore backup_20240115_143022.tar.gz

# List all backups
nself backup list
```

### Cloud Storage Support

Configure automatic cloud uploads for offsite backup:

```bash
# Interactive cloud setup wizard
nself backup cloud setup

# Supported providers:
# - Amazon S3 / MinIO
# - Dropbox
# - Google Drive
# - OneDrive
# - 40+ providers via rclone (Box, MEGA, pCloud, etc.)

# Test cloud connection
nself backup cloud test

# View cloud configuration
nself backup cloud status
```

### Advanced Retention Policies

Intelligently manage backup storage with multiple retention strategies:

```bash
# Simple age-based cleanup (default)
nself backup clean --age 30      # Remove backups older than 30 days

# List backups with filtering
nself backup list --filter DATE  # View backups by date

# Create full or incremental backups
nself backup create --full       # Full backup
nself backup create --incremental # Incremental backup
```

### Automated Backups

Schedule automatic backups:

```bash
# Create backups
nself backup create              # Full backup
nself backup create --incremental # Incremental backup

# Restore from backup
nself backup restore <backup-id>

# Clean old backups
nself backup clean --age 30      # Remove backups older than 30 days
```

### Backup Configuration

Environment variables for backup customization:

```bash
# Local storage
BACKUP_DIR=./backups             # Backup directory
BACKUP_RETENTION_DAYS=30         # Default retention
BACKUP_RETENTION_MIN=3           # Minimum backups to keep

# Cloud provider selection
BACKUP_CLOUD_PROVIDER=s3         # s3, dropbox, gdrive, onedrive, rclone

# Provider-specific settings
S3_BUCKET=my-backups
DROPBOX_TOKEN=xxx
GDRIVE_FOLDER_ID=xxx
RCLONE_REMOTE=myremote
```

### What Gets Backed Up

**Full Backup includes:**
- PostgreSQL databases (complete dump)
- All environment files (.env.dev, .env.staging, .env.prod, .env.secrets, .env)
- Docker-compose configurations
- Docker volumes (all project data)
- SSL certificates
- Hasura metadata
- Nginx configurations

## 🚀 Production Deployment

### Using Production Deployment

Deploy to production environments:

```bash
# 1. Configure production environment
nself config env create production

# 2. Deploy to production
nself deploy production

# 3. Check deployment status
nself deploy status

# 4. Monitor deployment
nself monitor
```

> **Command Update:** In v0.9.6+, use `nself deploy staging` and `nself deploy production` instead of the old `nself staging` and `nself prod` commands.

Environment files are loaded in priority order (highest priority last):
- `.env.dev` - Team defaults (always loaded)
- `.env.staging` - Staging environment (if ENV=staging)
- `.env.prod` - Production environment (if ENV=prod)
- `.env.secrets` - Production secrets (if ENV=prod)
- `.env` - Local overrides (highest priority)

### Production Checklist
1. Set `ENV=prod` (automatically configures security settings)
2. Use strong passwords (12+ characters, auto-generated by `nself prod`)
3. Configure your custom domain
4. Enable Let's Encrypt SSL
5. Set up automated backups
6. Configure monitoring alerts

## 📁 Project Structure

After running `nself build`:

```
my-backend/
├── .env.dev               # Team defaults
├── .env.staging           # Staging environment (optional)
├── .env.prod              # Production environment (optional)
├── .env.secrets           # Production secrets (optional)
├── .env                   # Local configuration (highest priority)
├── docker-compose.yml      # Generated Docker Compose file
├── docker-compose.custom.yml # Custom services (if CS_N variables defined)
├── _backup/               # Timestamped backups from build/reset
│   └── YYYYMMDD_HHMMSS/  # Each backup in its own timestamp folder
├── nginx/                  # Nginx configuration
│   ├── nginx.conf
│   ├── conf.d/            # Service routing
│   └── ssl/               # SSL certificates
├── postgres/              # Database initialization
│   └── init/
├── hasura/                # GraphQL configuration
│   ├── metadata/
│   └── migrations/
├── functions/             # Optional serverless functions
└── services/              # Backend services (if enabled)
    ├── nest/              # NestJS microservices
    ├── bullmq/            # Queue workers
    ├── go/                # GoLang services
    └── py/                # Python services
```

## 🗄️ Database Management

nself includes comprehensive database tools for schema management, migrations, and team collaboration.

### For Lead Developers
```bash
# Design your schema
nano schema.dbml

# Generate migrations from schema
nself db run

# Test migrations locally
nself db migrate:up

# Commit to Git
git add schema.dbml hasura/migrations/
git commit -m "Add new tables"
git push
```

### For All Developers
```bash
# Pull latest code
git pull

# Start services
nself start

# If you see "DATABASE MIGRATIONS PENDING" warning:
nself db update  # Safely apply migrations with confirmation
```

### Database Commands
- `nself db` - Show all database commands
- `nself db run` - Generate migrations from schema.dbml
- `nself db update` - Safely apply pending migrations and seeds
- `nself db seed` - Apply seed data (dev or prod based on ENV)
- `nself db status` - Check database state
- `nself db revert` - Restore from backup
- `nself db sync` - Pull schema from dbdiagram.io

### Database Seeding Strategy

When `DB_ENV_SEEDS=true` (recommended - follows Hasura/PostgreSQL standards):
- `seeds/common/` - Shared data across all environments
- `seeds/development/` - Mock/test data (when ENV=dev)
- `seeds/staging/` - Staging data (when ENV=staging) 
- `seeds/production/` - Minimal production data (when ENV=prod)

When `DB_ENV_SEEDS=false` (no environment branching):
- `seeds/default/` - Single seed directory for all environments

See [DBTOOLS.md](DBTOOLS.md) for complete database documentation.

## 🔄 Updating Services

To update service configurations:

1. Edit `.env`
2. Run `nself build` to regenerate configurations
3. Run `nself start` to apply changes

## 🐛 Troubleshooting

### Common Issues

#### Build command hangs?
```bash
# Build includes 5-second timeout for validation
nself build --force  # Force rebuild if stuck
```

#### Services not starting?
```bash
# Run diagnostics first
nself doctor

# Check service logs
nself logs [service-name]

# Check service status
nself status
```

#### Auth service unhealthy?
Known issue: Auth health check reports unhealthy but service works (port 4001 vs 4000 mismatch).

#### Port conflicts?
Edit the port numbers in `.env` and rebuild.

#### SSL certificate warnings?
Run `nself auth ssl trust` to install the root CA and get green locks in your browser. No more warnings!

#### Email test not working?
```bash
# Test email configuration
nself service email test recipient@example.com
```

## 🔄 Version History

### v0.8.0 (Current - Jan 29, 2026)
- ✅ Complete multi-tenant architecture with tenant and organization management
- ✅ Plugin ecosystem with marketplace integration
- ✅ Real-time collaboration features (presence, sync, broadcast)
- ✅ Enhanced security tools (scan, audit, firewall)
- ✅ Developer experience improvements (console, tunnel, mock)
- ✅ Cross-environment migration tools
- ✅ Performance profiling and benchmarking
- ✅ 56 commands total

### v0.7.0 (Jan 25, 2026)
- Real-time collaboration infrastructure
- Enhanced monitoring and observability
- Security hardening features

### v0.6.0 (Jan 24, 2026)
- Plugin system foundation
- Multi-tenancy support
- Advanced developer tools

### v0.5.0 (Jan 23, 2026)
- First production-ready release
- 36 commands, 40+ service templates
- Full monitoring stack
- Kubernetes and Helm support

[Full Changelog](docs/releases/CHANGELOG.md)

## 🧪 Quality Assurance

### Dedicated QA Team

ɳSelf has a dedicated QA team that ensures the highest quality for every release:

- **Release Testing**: Every release, patch, and update is thoroughly tested before deployment
- **Issue Reproduction**: All user-reported issues are reproduced and verified by the QA team
- **Fix Verification**: When maintainer Aric Camarata (acamarata) pushes a fix, QA confirms it resolves the issue
- **Regression Testing**: Ensures new changes don't break existing functionality
- **Multi-Platform Testing**: Validates functionality across macOS, Linux, WSL2, and Docker environments

This systematic QA process ensures that ɳSelf remains stable and reliable for production use.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

Source Available License - see [LICENSE](LICENSE) file for details.

## 🎯 Perfect For

- **Startups**: Get your backend up fast, scale when you need to
- **Agencies**: Standardized backend setup for all client projects  
- **Enterprises**: Self-hosted solution with full control
- **Side Projects**: Production-grade infrastructure without the complexity
- **Learning**: See how modern backends work under the hood

## 📄 License

ɳSelf is **free for personal use**. Commercial use requires a license.

- ✅ **Personal Projects**: Free forever
- ✅ **Learning & Education**: Free forever  
- ✅ **Open Source Projects**: Free forever
- 💼 **Commercial Use**: [Contact us for licensing](https://nself.org/commercial)

See [LICENSE](LICENSE) for full terms.

## 🔗 Links

- [nself.org](https://nself.org) - Official Website
- [Commercial Licensing](https://nself.org/commercial) - For business use
- [Nhost Documentation](https://docs.nhost.io) - Learn more about Nhost
- [Hasura Documentation](https://hasura.io/docs) - GraphQL engine docs
- [Report Issues](https://github.com/acamarata/nself/issues) - We'd love your feedback!

## Admin UI

Access the web-based administration interface:

```bash
nself admin           # Open admin in browser
nself admin --dev     # Open in development mode
```

The admin UI provides:
- Real-time service monitoring
- Configuration file editing
- Log streaming and management
- Backup management interface
- Resource usage monitoring

## Enterprise Search

Choose from 6 different search engines:

```bash
nself service search init <provider>    # Initialize search provider
nself service search config             # Configure search settings
nself service search query <text>       # Test search functionality
```

**Available Engines:**
- PostgreSQL FTS (built-in)
- MeiliSearch (recommended)
- Typesense, Elasticsearch, OpenSearch, Sonic

## Remote Deployment

Deploy to any VPS with one command:

```bash
nself deploy provision <provider>  # Provision new server
nself deploy staging              # Deploy to staging
nself deploy production           # Deploy to production
nself deploy status               # Check deployment status
```

Supports DigitalOcean, Linode, Vultr, Hetzner, and any Ubuntu/Debian VPS.

## 📚 Documentation

### Getting Started
- **[Installation Guide](docs/EXAMPLES.md#installation)** - Step-by-step installation
- **[Quick Start Tutorial](docs/EXAMPLES.md#basic-usage)** - Zero to running in 5 minutes
- **[Configuration Reference](docs/ENVIRONMENT_CONFIGURATION.md)** - Complete `.env` settings guide
- **[Command Reference](docs/API.md)** - All 35+ CLI commands
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Fix common issues

### Features & Services
- **[Service Templates](docs/SERVICE_TEMPLATES.md)** - 40+ microservice templates (JS/TS, Python, Go, Rust, Java, etc.)
- **[Admin Dashboard](docs/EXAMPLES.md#admin-ui)** - Web-based monitoring and management
- **[Email Setup](docs/EXAMPLES.md#email-configuration)** - 16+ provider configuration
- **[SSL Certificates](docs/EXAMPLES.md#ssl-setup)** - Automatic HTTPS setup
- **[Backup System](docs/BACKUP_GUIDE.md)** - Comprehensive backup and restore

### Advanced Topics
- **[Architecture Overview](docs/ARCHITECTURE.md)** - System design and components
- **[Environment Cascade](docs/CONFIG.md)** - Multi-environment configuration
- **[Directory Structure](docs/DIRECTORY_STRUCTURE.md)** - Complete file organization
- **[Contributing](docs/contributing/CONTRIBUTING.md)** - Development guidelines

### Release Information
- **[Release Notes](docs/RELEASES.md)** - Complete version history
- **[Changelog](docs/CHANGELOG.md)** - User-facing changes
- **[Roadmap](docs/ROADMAP.md)** - Future development plans

---

Built with ❤️ for the self-hosting community by developers who were tired of complex setups


