# nself Commands Reference v0.3.9

**Complete command reference for nself - Self-Hosted Infrastructure Manager**

## Table of Contents

- [Overview](#overview)
- [Command Status Legend](#command-status-legend)
- [Core Commands](#core-commands)
- [Status & Monitoring Commands](#status--monitoring-commands)
- [Management Commands](#management-commands)
- [Service-Specific Commands](#service-specific-commands)
- [Configuration Commands](#configuration-commands)
- [Database Commands](#database-commands-roadmap)
- [Backup & Recovery Commands](#backup--recovery-commands-roadmap)
- [Deployment Commands](#deployment-commands-roadmap)
- [Utility Commands](#utility-commands)
- [Common Workflows](#common-workflows)
- [Environment Variables](#environment-variables)
- [Exit Codes](#exit-codes)

---

## Overview

nself provides a comprehensive CLI for managing your entire backend infrastructure. Commands follow a consistent structure with helpful flags and detailed help messages.

**Quick Start:**
```bash
nself init --demo    # Initialize demo project with all services
nself build          # Build infrastructure and services
nself start          # Start all services
nself status         # Check service health
nself urls           # Show all service URLs
```

---

## Command Status Legend

| Symbol | Status | Description |
|--------|--------|-------------|
| ✅ | **Implemented** | Fully functional in v0.3.9 |
| 🚧 | **Partial** | Basic functionality implemented, more features coming |
| 🔮 | **Planned** | Roadmapped for future version |
| 📋 | **Designed** | Specification complete, implementation pending |

---

## Core Commands

### ✅ `nself init`

Initialize a new nself project with environment configuration.

**Usage:**
```bash
nself init [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--full` | Create all environment files (.env.dev, .env.staging, .env.prod, .env.secrets) and schema.dbml |
| `--wizard` | Launch interactive setup wizard with prompts for all services |
| `--demo` | Create complete demo app with all 25 services enabled |
| `--admin` | Setup minimal admin UI only (no other services) |
| `--force` | Reinitialize even if .env exists (safe, won't overwrite) |
| `--quiet`, `-q` | Minimal output (for automation/CI) |
| `-h`, `--help` | Show help message |

**Files Created (Basic):**
- `.env` - Your personal development configuration
- `.env.example` - Complete reference documentation with all variables
- `.gitignore` - Security rules (ignores .env.secrets, sensitive files)

**Files Created (--full):**
All basic files plus:
- `.env.dev` - Team-shared development defaults
- `.env.staging` - Staging environment configuration
- `.env.prod` - Production configuration (public variables only)
- `.env.secrets` - Sensitive data template (git-ignored)
- `schema.dbml` - Example database schema

**Examples:**
```bash
# Basic setup - works out of the box
mkdir myproject && cd myproject
nself init

# Complete setup with all environment files
nself init --full

# Interactive wizard for guided configuration
nself init --wizard

# Full demo with all 25 services
nself init --demo
```

**Next Steps:**
1. (Optional) Edit `.env` to customize configuration
2. Run `nself build` to generate infrastructure
3. Run `nself start` to launch services

---

### ✅ `nself build`

Generate Docker Compose configuration, build custom services, and prepare infrastructure.

**Usage:**
```bash
nself build [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--force` | Force rebuild all services (ignores cache) |
| `--verbose` | Show detailed output from Docker builds |
| `--no-cache` | Build without using Docker cache |
| `--debug` | Enable debug mode with maximum verbosity |

**What It Does:**
1. Reads environment configuration from `.env` files
2. Generates `docker-compose.yml` with all enabled services
3. Creates nginx configuration with routing rules
4. Generates SSL certificates (self-signed for local dev)
5. Copies custom service templates (CS_1 through CS_10)
6. Builds Docker images for custom services
7. Sets up monitoring configurations (if enabled)

**Generated Files:**
```
docker-compose.yml           # Main services definition (25+ services)
nginx/
  ├── nginx.conf            # Main nginx config
  ├── includes/             # Security headers, gzip, etc.
  └── sites/                # Route configs for all services
ssl/
  ├── cert.pem             # Self-signed certificate
  └── key.pem              # Private key
postgres/
  └── init/
      └── 00-init.sql       # Database initialization
services/                   # Custom services (if using CS_N)
  ├── express_api/          # CS_1 (example)
  ├── bullmq_worker/        # CS_2 (example)
  └── ...
monitoring/                 # If MONITORING_ENABLED=true
  ├── prometheus/
  ├── grafana/
  └── loki/
```

**Examples:**
```bash
# Standard build
nself build

# Force rebuild everything
nself build --force

# Rebuild with detailed output
nself build --verbose --no-cache

# Debug mode for troubleshooting
DEBUG=true nself build
```

---

### ✅ `nself start`

Start all enabled services with health checking and monitoring.

**Usage:**
```bash
nself start [OPTIONS] [SERVICES...]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--verbose` | Show detailed startup logs |
| `--debug` | Maximum verbosity with debug information |
| `--skip-health-checks` | Skip health validation (faster startup) |
| `--skip-checks` | Skip all pre-flight checks (port availability, etc.) |
| `--quick` | Quick start mode (minimal checks, fast startup) |
| `--fresh` | Force recreate all containers |
| `--clean-start` | Remove all containers before starting |
| `--attach` | Run in foreground (Ctrl+C to stop) |
| `--no-deps` | Don't start linked services |
| `--timeout SECONDS` | Health check timeout (default: 120) |
| `--health-required PERCENT` | Percent of services required healthy (default: 80) |

**Environment Variables:**
```bash
# Start behavior control
NSELF_START_MODE=smart              # smart, fresh, force
NSELF_HEALTH_CHECK_TIMEOUT=120      # Seconds to wait
NSELF_HEALTH_CHECK_INTERVAL=2       # Check interval
NSELF_HEALTH_CHECK_REQUIRED=80      # Percent required healthy
NSELF_SKIP_HEALTH_CHECKS=false      # Skip validation
NSELF_LOG_LEVEL=info                # debug, info, warn, error
```

**Start Modes:**
- **smart** (default): Intelligently resumes stopped containers, keeps running ones
- **fresh**: Force recreates all containers
- **force**: Most aggressive - removes everything first

**Examples:**
```bash
# Standard start
nself start

# Quick start (development)
nself start --quick

# Full validation (production)
NSELF_HEALTH_CHECK_REQUIRED=100 nself start

# Start specific services only
nself start postgres hasura

# Debugging startup issues
nself start --verbose --debug
```

---

### ✅ `nself stop`

Stop running services with optional cleanup.

**Usage:**
```bash
nself stop [OPTIONS] [SERVICES...]
```

**Options:**
| Flag | Description |
|------|-------------|
| `-v`, `--volumes` | ⚠️ Remove volumes (deletes all data) |
| `--rmi` | Remove Docker images |
| `--remove-orphans` | Remove containers for services not in compose file |
| `--verbose` | Show detailed output |
| `--timeout SECONDS` | Shutdown timeout (default: 10) |

**Examples:**
```bash
# Stop all services, keep data
nself stop

# Stop specific service
nself stop postgres

# Stop and remove all data (⚠️ DESTRUCTIVE)
nself stop --volumes

# Stop with custom timeout
nself stop --timeout 30
```

---

### ✅ `nself restart`

Restart services (stop then start).

**Usage:**
```bash
nself restart [OPTIONS] [SERVICES...]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--all` | Restart all services |
| `--timeout SECONDS` | Restart timeout |

**Examples:**
```bash
# Restart all services
nself restart

# Restart specific services
nself restart postgres hasura

# Restart with custom timeout
nself restart --timeout 30 postgres
```

---

### ✅ `nself reset`

Reset project to clean state.

**Usage:**
```bash
nself reset [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--hard` | ⚠️ Remove all data and generated configurations |
| `--soft` | Keep configurations, reset data only |
| `--confirm` | Skip confirmation prompt |

**Examples:**
```bash
# Standard reset (prompts for confirmation)
nself reset

# Hard reset - remove everything
nself reset --hard --confirm

# Soft reset - keep configs
nself reset --soft
```

---

### ✅ `nself clean`

Clean up Docker resources.

**Usage:**
```bash
nself clean [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--all` | Remove all nself containers and volumes |
| `--images` | Remove Docker images |
| `--orphans` | Remove orphaned containers |
| `--system` | Run Docker system prune |

**Examples:**
```bash
# Standard cleanup
nself clean

# Deep cleanup
nself clean --all --images --system
```

---

### 🚧 `nself restore`

Restore configuration from backup (basic implementation).

**Usage:**
```bash
nself restore [BACKUP_FILE]
```

**Examples:**
```bash
nself restore backup-2024-01-15.tar.gz
```

**Note:** Full backup/restore system coming in v0.4.4.

---

## Status & Monitoring Commands

### ✅ `nself status`

Show service health and resource usage with real-time monitoring.

**Usage:**
```bash
nself status [OPTIONS] [SERVICE]
```

**Options:**
| Flag | Description |
|------|-------------|
| `-w`, `--watch` | Watch mode (refresh every 5s) |
| `-i`, `--interval N` | Set refresh interval for watch mode |
| `--no-resources` | Hide resource usage information |
| `--show-ports` | Show detailed port mappings |
| `--format FORMAT` | Output format: `table` (default), `json`, `compact` |
| `--filter STATUS` | Filter by status: `running`, `stopped`, `unhealthy` |

**Shows:**
- Service status (✓ running, ✗ stopped, ⚠ unhealthy)
- Resource usage (CPU %, Memory usage)
- Uptime
- Health check status
- Container restart count

**Examples:**
```bash
# Show all services
nself status

# Watch mode (refresh every 5s)
nself status --watch

# Custom refresh interval
nself status --watch --interval 2

# Specific service
nself status postgres

# JSON output for automation
nself status --format json

# Show only running services
nself status --filter running
```

---

### ✅ `nself logs`

View service logs with filtering and following.

**Usage:**
```bash
nself logs [OPTIONS] [SERVICE]
```

**Options:**
| Flag | Description |
|------|-------------|
| `-f`, `--follow` | Follow log output (live stream) |
| `-t`, `--timestamps` | Show timestamps |
| `--since TIME` | Show logs since timestamp (e.g., `2h`, `2024-01-15T10:00:00`) |
| `--tail N` | Number of lines to show (default: 50) |
| `--no-color` | Disable colored output |
| `--grep PATTERN` | Filter logs by pattern |

**Examples:**
```bash
# Show all service logs
nself logs

# Show postgres logs
nself logs postgres

# Follow hasura logs
nself logs -f hasura

# Last 100 lines with timestamps
nself logs --tail 100 --timestamps postgres

# Logs from last 2 hours
nself logs --since 2h

# Filter logs
nself logs postgres --grep ERROR
```

---

### ✅ `nself exec`

Execute commands in service containers.

**Usage:**
```bash
nself exec [OPTIONS] <SERVICE> [COMMAND]
```

**Options:**
| Flag | Description |
|------|-------------|
| `-it` | Interactive terminal (default for shell access) |
| `-T` | Disable pseudo-TTY |
| `-u USER` | Run as specific user |
| `-w DIR` | Working directory inside container |
| `--index N` | Target specific replica (if scaled) |

**Examples:**
```bash
# PostgreSQL console
nself exec postgres psql -U postgres

# Interactive bash in nginx
nself exec -it nginx /bin/bash

# Hasura CLI console
nself exec hasura hasura-cli console

# Run command as specific user
nself exec -u www-data nginx ls -la /var/www

# Execute in custom service
nself exec express_api npm run test
```

---

### ✅ `nself urls`

Show all service URLs and access points.

**Usage:**
```bash
nself urls [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--internal` | Show internal Docker network URLs |
| `--external` | Show only external URLs (default) |
| `--format FORMAT` | Output format: `table`, `list`, `json` |
| `--open SERVICE` | Open specific service URL in browser |

**Shows:**
- Application root URL
- GraphQL API endpoint (Hasura)
- Authentication service URL
- Admin UIs (if enabled)
- Custom service URLs
- Frontend app URLs
- Monitoring dashboards (if enabled)

**Examples:**
```bash
# Show all URLs
nself urls

# Show internal URLs
nself urls --internal

# JSON output
nself urls --format json

# Open Hasura console
nself urls --open hasura
```

---

### ✅ `nself doctor`

Run system diagnostics and health checks.

**Usage:**
```bash
nself doctor [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--fix` | Attempt to fix issues automatically |
| `--verbose` | Show detailed diagnostics |
| `--check TYPE` | Run specific check: `docker`, `network`, `services`, `config` |

**Checks:**
1. **Docker**: Installation, version, daemon status
2. **System Resources**: CPU, memory, disk space
3. **Network**: Connectivity, port availability, DNS
4. **Services**: Container health, dependencies
5. **Configuration**: Environment files, required variables
6. **SSL**: Certificate validity
7. **Database**: Connection, permissions

**Examples:**
```bash
# Full diagnostic
nself doctor

# Auto-fix issues
nself doctor --fix

# Check specific component
nself doctor --check docker
nself doctor --check services
```

---

### ✅ `nself version`

Show version information.

**Usage:**
```bash
nself version [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `-v` | Short version number only |
| `--check` | Check for updates |
| `--verbose` | Show detailed version info (Git commit, build date) |

**Examples:**
```bash
# Full version info
nself version

# Just the version number
nself version -v
# or
nself -v

# Check for updates
nself version --check
```

---

### ✅ `nself update`

Update nself CLI to latest version.

**Usage:**
```bash
nself update [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--check` | Check for updates only (don't install) |
| `--force` | Force update even if current version |
| `--beta` | Update to beta version |
| `--version VERSION` | Update to specific version |

**Examples:**
```bash
# Update to latest
nself update

# Check if update available
nself update --check

# Update to specific version
nself update --version 0.4.0

# Force reinstall current version
nself update --force
```

---

### ✅ `nself help`

Show help information.

**Usage:**
```bash
nself help [COMMAND]
```

**Examples:**
```bash
# General help
nself help

# Command-specific help
nself help init
nself help admin
nself help start

# Alternative syntax
nself init --help
```

---

## Management Commands

### ✅ `nself ssl`

Manage SSL certificates for local development and production.

**Usage:**
```bash
nself ssl [OPTIONS]
```

**Options:**
| Flag | Description |
|------|-------------|
| `--generate` | Generate self-signed certificates |
| `--import CERT KEY` | Import existing certificates |
| `--renew` | Renew certificates |
| `--verify` | Verify certificate configuration |
| `--force` | Force regenerate certificates |

**Generates Certificates For:**
- `localhost`
- `*.local.nself.org` (wildcard)
- Custom domains (from BASE_DOMAIN)

**Examples:**
```bash
# Generate self-signed certs
nself ssl --generate

# Import existing certs
nself ssl --import /path/to/cert.pem /path/to/key.pem

# Verify configuration
nself ssl --verify
```

---

### ✅ `nself trust`

Install SSL certificates in system trust store.

**Usage:**
```bash
nself trust [OPTIONS]
```

**Platform Support:**
- ✅ macOS Keychain
- ✅ Linux ca-certificates
- 🚧 Windows Certificate Store (partial)
- 🚧 Firefox/Chrome certificate stores (partial)

**Examples:**
```bash
# Trust local certificates
nself trust

# May require sudo on some systems
sudo nself trust
```

---

### ✅ `nself admin`

Manage nself Admin UI - web-based management interface.

**Usage:**
```bash
nself admin [SUBCOMMAND] [OPTIONS]
```

**Subcommands:**

#### `nself admin enable`
Enable admin UI and generate temporary password.

**Options:**
- `--port PORT` - Custom port (default: 3100)
- `--password PASS` - Set custom password
- `--no-open` - Don't open browser automatically

#### `nself admin disable`
Disable and stop admin UI.

#### `nself admin status`
Show admin UI status and configuration.

**Shows:**
- Status (running/stopped)
- URL
- Port
- Container information
- Resource usage

#### `nself admin password [PASSWORD]`
Set or reset admin password.

**Examples:**
```bash
# Set specific password
nself admin password mypassword123

# Generate random password (prompted)
nself admin password
```

#### `nself admin reset`
Reset admin to defaults (regenerate password).

#### `nself admin logs`
Show admin container logs.

**Options:**
- `-f` - Follow logs
- `--tail N` - Show last N lines

#### `nself admin open`
Open admin UI in default browser.

**Admin UI Features:**
- 📊 Real-time service monitoring
- 🐳 Docker container management
- 🗄️ Database query interface
- 📝 Log viewer with filtering
- ⚙️ Configuration editor
- 💾 Backup management
- 📈 Resource usage graphs
- 🔧 Service control (start/stop/restart)

**Access:**
- Default URL: `http://localhost:3100` or `http://admin.local.nself.org`
- Default username: `admin`
- Password: Set with `nself admin password` or use generated temporary

**Examples:**
```bash
# Enable admin UI
nself admin enable

# Enable with custom password
nself admin enable --password mypassword

# Check status
nself admin status

# Open in browser
nself admin open

# View logs
nself admin logs -f

# Disable admin UI
nself admin disable
```

---

## Service-Specific Commands

### 🔮 `nself email` (Roadmap: v0.4.1)

Configure email service provider.

**Usage:**
```bash
nself email [SUBCOMMAND] [OPTIONS]
```

**Planned Subcommands:**
- `configure <provider>` - Configure email provider
- `test` - Send test email
- `templates` - Manage email templates
- `logs` - View email service logs

**Supported Providers:**
- Development: `mailpit`, `mailhog`
- Production: `sendgrid`, `mailgun`, `ses`, `smtp`
- Transactional: `postmark`, `sparkpost`, `mandrill`

---

### 🔮 `nself search` (Roadmap: v0.4.1)

Manage search services with multiple engine options.

**Usage:**
```bash
nself search [SUBCOMMAND] [OPTIONS]
```

**Planned Subcommands:**
- `enable` - Enable search service
- `disable` - Disable search service
- `configure <engine>` - Switch search engine
- `reindex [index]` - Rebuild search index
- `clear [index]` - Clear search index
- `import` - Import data into search
- `export` - Export search data
- `dashboard` - Open search dashboard
- `health` - Check search service health

**Supported Engines:**
- `meilisearch` - Fast, typo-tolerant (default)
- `typesense` - Lightning-fast search
- `sonic` - Ultra-lightweight
- `zinc` - Lightweight Elasticsearch alternative
- `elasticsearch` - Industry standard (requires more resources)
- `opensearch` - Open-source Elasticsearch fork

---

### 🔮 `nself functions` (Roadmap: v0.4.1)

Manage serverless functions runtime.

**Usage:**
```bash
nself functions [SUBCOMMAND] [OPTIONS]
```

**Planned Subcommands:**
- `enable` - Enable serverless functions
- `disable` - Disable serverless functions
- `list` - List all deployed functions
- `create <name>` - Create new function
- `delete <name>` - Delete function
- `test <name>` - Test function locally
- `logs <name>` - View function logs
- `deploy [name]` - Deploy function(s)

**Supports:**
- Node.js functions
- Python functions
- Auto-scaling
- Event triggers

---

### 🔮 `nself mlflow` (Roadmap: v0.4.1)

Manage MLflow for ML experiment tracking.

**Usage:**
```bash
nself mlflow [SUBCOMMAND] [OPTIONS]
```

**Planned Subcommands:**
- `enable` - Enable MLflow tracking server
- `disable` - Disable MLflow tracking server
- `open` / `ui` / `dashboard` - Open MLflow UI
- `configure` - Configure MLflow settings
- `logs` - View MLflow service logs
- `test` - Test MLflow connectivity

**Features:**
- Experiment tracking
- Model versioning
- Model registry
- Artifact storage
- Metrics visualization
- Parameter tracking

---

### 🔮 `nself metrics` (Roadmap: v0.4.2)

View service metrics and performance data.

**Usage:**
```bash
nself metrics [SERVICE] [OPTIONS]
```

**Planned Options:**
- `--interval SECONDS` - Refresh interval
- `--export FORMAT` - Export metrics (json, csv)
- `--since TIME` - Show metrics since time
- `--prometheus` - Open Prometheus UI
- `--grafana` - Open Grafana dashboards

---

### 🔮 `nself monitor` (Roadmap: v0.4.2)

Real-time monitoring dashboard in terminal.

**Usage:**
```bash
nself monitor [OPTIONS]
```

**Planned Features:**
- Service health monitoring
- Resource usage graphs
- Log streaming
- Alert notifications
- Interactive TUI interface

---

## Configuration Commands

### 🔮 `nself prod` (Roadmap: v0.4.5)

Generate production-ready configuration.

**Usage:**
```bash
nself prod [OPTIONS]
```

**Planned Features:**
- Generate secure random passwords
- Create `.env.prod` file
- Configure SSL for production domains
- Set resource limits
- Enable security features
- Harden nginx configuration
- Configure backups

**Options:**
- `--domain DOMAIN` - Production domain
- `--ssl-email EMAIL` - Let's Encrypt email
- `--db-password` - Custom database password
- `--interactive` - Interactive configuration

---

## Database Commands (Roadmap)

### 🔮 `nself db` (Roadmap: v0.4.3)

Comprehensive database management interface.

**Usage:**
```bash
nself db [SUBCOMMAND] [OPTIONS]
```

**Planned Subcommands:**

#### Schema Management
- `run` - Analyze schema.dbml and generate migrations
- `sync` - Pull schema from dbdiagram.io
- `sample` - Create sample schema
- `diff` - Show schema differences

#### Migrations
- `migrate:create NAME` - Create new migration
- `migrate:up [N]` - Apply N migrations (default: all)
- `migrate:down [N]` - Rollback N migrations
- `migrate:status` - Show migration status
- `migrate:reset` - Reset all migrations

#### Data Operations
- `console` - Open PostgreSQL console (psql)
- `export [--format=sql|csv|json]` - Export database
- `import FILE` - Import data from file
- `clone SOURCE DEST` - Clone database
- `dump` - Create SQL dump
- `restore DUMP` - Restore from SQL dump

#### Maintenance
- `optimize` - Run full optimization (VACUUM, ANALYZE, REINDEX)
- `vacuum [--full]` - Reclaim storage space
- `analyze` - Update query planner statistics
- `reindex` - Rebuild all indexes

#### Monitoring
- `connections` - Show active connections
- `locks` - Show current locks
- `kill PID` - Terminate connection by PID
- `size` - Show database and table sizes
- `stats` - Show database statistics

#### Lifecycle
- `seed` - Seed database with initial data
- `reset` - Drop and recreate database
- `update` - Apply migrations and seeds
- `status` - Show database status

---

## Backup & Recovery Commands (Roadmap)

### 🔮 `nself backup` (Roadmap: v0.4.4)

Comprehensive backup system with local and cloud support.

**Usage:**
```bash
nself backup [SUBCOMMAND] [OPTIONS]
```

**Planned Subcommands:**

#### Creating Backups
- `create [--type=full|database|config]` - Create backup
- `create --name NAME` - Create named backup
- `create --compress` - Compress backup with gzip

#### Managing Backups
- `list` - List all backups with details
- `restore BACKUP_ID` - Restore from backup
- `verify BACKUP_ID` - Verify backup integrity
- `delete BACKUP_ID` - Delete specific backup
- `info BACKUP_ID` - Show backup details

#### Backup Pruning
- `prune --age DAYS` - Remove backups older than N days
- `prune --gfs` - Apply Grandfather-Father-Son retention
- `prune --smart` - Smart retention (keeps important backups)
- `prune --dry-run` - Preview what would be deleted
- `prune --keep-last N` - Keep last N backups

#### Cloud Backups
- `cloud setup` - Configure cloud provider (S3, GCS, Azure, Backblaze)
- `cloud sync` - Sync local backups to cloud
- `cloud list` - List cloud backups
- `cloud restore` - Restore from cloud backup
- `cloud prune` - Prune cloud backups

#### Scheduling
- `schedule daily HH:MM` - Schedule daily backups
- `schedule weekly DAY HH:MM` - Schedule weekly backups
- `schedule monthly DD HH:MM` - Schedule monthly backups
- `schedule list` - Show scheduled backups
- `schedule remove ID` - Remove backup schedule
- `schedule run ID` - Manually trigger scheduled backup

---

### 🔮 `nself rollback` (Roadmap: v0.4.4)

Rollback to previous version or backup.

**Usage:**
```bash
nself rollback [OPTIONS]
```

**Planned Options:**
- `--version VERSION` - Rollback to specific version
- `--backup BACKUP_ID` - Rollback to specific backup
- `--list` - List available rollback points
- `--dry-run` - Preview rollback changes

---

## Deployment Commands (Roadmap)

### 🔮 `nself deploy` (Roadmap: v0.4.5)

Deploy to remote servers via SSH with zero-downtime.

**Usage:**
```bash
nself deploy [TARGET] [OPTIONS]
```

**Planned Subcommands:**
- `init` - Initialize deployment configuration
- `ssh` - Deploy to VPS server via SSH
- `status` - Check deployment status
- `rollback` - Rollback deployment
- `logs` - View deployment logs

**Planned Options:**
- `--host HOST` - Target host
- `--user USER` - SSH user
- `--key PATH` - SSH key path
- `--dry-run` - Preview deployment
- `--no-downtime` - Zero-downtime deployment
- `--backup-first` - Backup before deploying

**Planned Features:**
- Zero-downtime deployments
- Health check verification
- Automatic rollback on failure
- GitHub webhook integration
- Multi-environment sync
- SSL certificate provisioning

---

### 🔮 `nself scale` (Roadmap: v0.4.6)

Scale service resources and replicas.

**Usage:**
```bash
nself scale [SERVICE] [REPLICAS] [OPTIONS]
```

**Planned Options:**
- `--replicas N` - Number of replicas
- `--cpu LIMIT` - CPU limit
- `--memory LIMIT` - Memory limit
- `--auto` - Enable auto-scaling

**Examples:**
```bash
nself scale hasura 3              # Scale hasura to 3 replicas
nself scale postgres --cpu 2      # Set CPU limit
nself scale worker --auto         # Enable auto-scaling
```

---

## Utility Commands

### ✅ `nself up`

Alias for `nself start` (Docker Compose compatibility).

```bash
nself up [OPTIONS]
```

Same options as `nself start`.

---

### ✅ `nself down`

Alias for `nself stop` (Docker Compose compatibility).

```bash
nself down [OPTIONS]
```

Same options as `nself stop`.

---

## Common Workflows

### Quick Start - Development

```bash
# 1. Initialize project with demo
mkdir myproject && cd myproject
nself init --demo

# 2. Build infrastructure
nself build

# 3. Start services
nself start

# 4. Check status
nself status

# 5. View URLs
nself urls

# 6. Open admin UI
nself admin enable
nself admin open
```

### Custom Service Development

```bash
# 1. Initialize with wizard
nself init --wizard

# 2. Configure custom services in .env
# CS_1=api:express-ts:8001
# CS_2=worker:bullmq-js:8002

# 3. Build (generates services from templates)
nself build

# 4. Edit generated services
# services/api/...
# services/worker/...

# 5. Rebuild and start
nself build --force
nself start

# 6. View logs
nself logs api -f
nself logs worker -f
```

### Production Deployment (v0.4.5+)

```bash
# 1. Generate production config
nself prod --domain api.example.com

# 2. Configure SSL
nself ssl --generate

# 3. Set up backups
nself backup schedule daily 02:00
nself backup cloud setup

# 4. Test locally
nself build
nself start
nself doctor

# 5. Deploy to production
nself deploy production --host 203.0.113.10
```

### Database Workflow (v0.4.3+)

```bash
# 1. Create schema
nself db sample

# 2. Generate migration
nself db migrate:create add_users_table

# 3. Apply migrations
nself db migrate:up

# 4. Seed data
nself db seed

# 5. Backup before changes
nself backup create --name pre-migration

# 6. Monitor
nself db connections
nself db size
nself db stats
```

### Troubleshooting

```bash
# 1. Run diagnostics
nself doctor --verbose

# 2. Check logs
nself logs --tail 200
nself logs postgres -f

# 3. Check specific service
nself status postgres
nself exec postgres psql -U postgres -c "SELECT version();"

# 4. Reset if needed
nself stop
nself clean --all
nself build --force
nself start --verbose

# 5. Complete reset (⚠️ deletes data)
nself stop --volumes
nself reset --hard
```

---

## Environment Variables

### Core Configuration

```bash
# Project Settings
PROJECT_NAME=myproject              # Docker project name
ENV=dev                             # Environment: dev, staging, prod
BASE_DOMAIN=local.nself.org         # Base domain for routing

# Database
POSTGRES_DB=nhost                   # Database name
POSTGRES_USER=postgres              # Database user
POSTGRES_PASSWORD=secure-password   # Database password
POSTGRES_PORT=5432                  # External port (default: 5432)

# Hasura
HASURA_GRAPHQL_ADMIN_SECRET=admin-secret  # Admin secret
HASURA_JWT_KEY=jwt-secret-minimum-32-chars  # JWT secret
HASURA_GRAPHQL_ENABLE_CONSOLE=true  # Enable console

# Auth
AUTH_JWT_SECRET_KEY=jwt-secret-key  # Auth JWT secret
```

### Service Toggles

```bash
# Required Services (always enabled)
# POSTGRES_ENABLED=true  # Not needed, always on
# HASURA_ENABLED=true    # Not needed, always on
# AUTH_ENABLED=true      # Not needed, always on
# NGINX_ENABLED=true     # Not needed, always on

# Optional Services (must enable)
NSELF_ADMIN_ENABLED=true           # Admin UI
REDIS_ENABLED=true                 # Redis cache
MINIO_ENABLED=true                 # S3 storage
FUNCTIONS_ENABLED=true             # Serverless functions
MLFLOW_ENABLED=true                # ML tracking
MAILPIT_ENABLED=true               # Email (dev)
MEILISEARCH_ENABLED=true           # Search

# Monitoring Bundle (all 10 services)
MONITORING_ENABLED=true            # Enable all monitoring
# Individual overrides (if needed):
# PROMETHEUS_ENABLED=true
# GRAFANA_ENABLED=true
# LOKI_ENABLED=true
# PROMTAIL_ENABLED=true
# TEMPO_ENABLED=true
# ALERTMANAGER_ENABLED=true
# CADVISOR_ENABLED=true
# NODE_EXPORTER_ENABLED=true
# POSTGRES_EXPORTER_ENABLED=true
# REDIS_EXPORTER_ENABLED=true
```

### Custom Services

```bash
# Define up to 10 custom services
CS_1=api:express-ts:8001           # Express TypeScript API
CS_2=worker:bullmq-js:8002         # BullMQ worker
CS_3=grpc:grpc-go:8003             # Go gRPC service
CS_4=ml:fastapi-python:8004        # Python FastAPI
# ... CS_5 through CS_10
```

### Frontend Applications

```bash
# External applications (not in Docker)
FRONTEND_APP_1_NAME=webapp
FRONTEND_APP_1_PORT=3000
FRONTEND_APP_1_ROUTE=app

FRONTEND_APP_2_NAME=dashboard
FRONTEND_APP_2_PORT=3001
FRONTEND_APP_2_ROUTE=dashboard
```

### Start Command Configuration

```bash
# Smart defaults - all optional
NSELF_START_MODE=smart             # smart, fresh, force
NSELF_HEALTH_CHECK_TIMEOUT=120     # Seconds
NSELF_HEALTH_CHECK_REQUIRED=80     # Percent healthy
NSELF_SKIP_HEALTH_CHECKS=false     # Skip validation
NSELF_LOG_LEVEL=info               # debug, info, warn, error
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Misuse of command (invalid arguments) |
| `126` | Command cannot execute (permissions) |
| `127` | Command not found |
| `130` | Terminated by Ctrl+C (SIGINT) |

---

## See Also

- [Architecture Guide](../architecture/ARCHITECTURE.md)
- [Service Documentation](../services/SERVICES.md)
- [Environment Configuration](../configuration/ENVIRONMENT-VARIABLES.md)
- [Troubleshooting](../guides/TROUBLESHOOTING.md)
- [Contributing - Development Guide](../contributing/DEVELOPMENT.md)
- [Contributing - Cross-Platform Compatibility](../contributing/CROSS-PLATFORM-COMPATIBILITY.md)

---

**Version:** v0.3.9 | **Last Updated:** October 2024

For the latest documentation, visit: https://github.com/acamarata/nself/wiki
