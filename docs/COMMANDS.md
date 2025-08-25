# nself Commands Reference

Complete reference for all nself commands in v0.3.9.

## Command Tree

Based on actual implemented commands:

```
nself
├── 🚀 Core Commands
│   ├── init          Initialize a new project
│   │   └── --wizard  Interactive setup wizard (basic implementation)
│   ├── build         Build project structure and Docker images
│   ├── start         Start all services
│   ├── stop          Stop all services
│   ├── restart       Restart all services
│   ├── status        Show service status
│   └── logs          View service logs
│
├── ⚙️ Management Commands
│   ├── doctor        Run system diagnostics
│   ├── backup        Backup and restore system
│   ├── db            Database operations
│   ├── email         Email service configuration
│   ├── admin         Admin UI management (planned for v0.4.0)
│   ├── search        Search service management (planned for v0.4.0)
│   ├── deploy        SSH deployment (planned for v0.4.0)
│   ├── ssl           SSL certificate management
│   ├── urls          Show service URLs
│   ├── prod          Configure for production
│   ├── trust         Install SSL certificates
│   ├── validate      Validate configuration
│   ├── exec          Execute commands in containers
│   ├── scale         Resource scaling
│   ├── metrics       Metrics and monitoring
│   └── clean         Clean Docker resources
│
├── 🛠️ Development Commands
│   ├── diff          Show configuration differences
│   ├── reset         Reset project to clean state
│   ├── rollback      Rollback to previous version
│   ├── monitor       Real-time monitoring
│   └── scaffold      Create new service from template
│
├── 🔧 Tool Commands
│   ├── hot_reload    Enable hot reload for development
│   └── validate-env  Validate environment configuration
│
└── 📋 Other Commands
    ├── up            Alias for start (backward compatibility)
    ├── down          Alias for stop (backward compatibility)
    ├── update        Update nself to latest version
    ├── version       Show version information
    └── help          Show help information
```

---

# 🚀 Core Commands

## init

Initialize a new nself project with smart defaults.

**Usage:** `nself init [--wizard]`

**Options:**
- `--wizard` - Launch interactive setup wizard (NEW in v0.3.9)

**What it creates:**
- `.env.local` - Your main configuration file
- `.env.example` - Reference documentation for all options

**Example:**
```bash
nself init              # Quick setup with defaults
nself init --wizard     # Interactive guided setup (v0.3.9)
```

## build

Generate project infrastructure from your `.env.local` settings.

**Usage:** `nself build [--force]`

**What it generates:**
- `docker-compose.yml` - Main service orchestration
- `nginx/` - Web server configuration with SSL
- SSL certificates (automatic)
- Database initialization scripts

## start / stop / restart

**Usage:**
```bash
nself start [--verbose] [--skip-checks]
nself stop
nself restart
```

- `start` - Start all services with health checks
- `stop` - Stop all services gracefully  
- `restart` - Restart all services (stop + start)

**Aliases:**
- `nself up` - Same as `nself start`
- `nself down` - Same as `nself stop`

## status

Show comprehensive service status with health monitoring.

**Usage:** `nself status [--watch] [--json]`

## logs

View and follow service logs with filtering.

**Usage:** `nself logs [service] [--follow] [--tail N] [--grep PATTERN]`

---

# ⚙️ Management Commands

## doctor

Run comprehensive system diagnostics.

**Usage:** `nself doctor [--quick] [--fix] [--verbose]`

Performs health checks on SSL certificates, DNS, network, and system resources.

## backup

Enterprise-grade backup and restore system with cloud storage support.

**Usage:** `nself backup <subcommand> [options]`

**Subcommands:**
- `create [type] [name]` - Create backups (full, database, config)
- `list` - List available backups (local and cloud)
- `restore <name> [type]` - Restore from backup
- `prune [policy] [days]` - Remove old backups
  - Policies: `age` (default), `gfs` (Grandfather-Father-Son), `smart`, `cloud`
- `cloud [action]` - Manage cloud backups
  - Actions: `setup`, `status`, `test`
- `schedule [frequency]` - Schedule automatic backups
  - Frequencies: `hourly`, `daily`, `weekly`, `monthly`

**Cloud Providers Supported:**
- Amazon S3 / MinIO
- Dropbox
- Google Drive
- OneDrive
- 40+ providers via rclone (Box, MEGA, pCloud, etc.)

**Examples:**
```bash
# Create full backup
nself backup create

# Create database backup with custom name
nself backup create database pre-migration

# Setup cloud backups
nself backup cloud setup

# Apply smart retention policy
nself backup prune smart

# Schedule daily backups
nself backup schedule daily
```

**Environment Variables:**
- `BACKUP_DIR` - Backup directory (default: ./backups)
- `BACKUP_RETENTION_DAYS` - Days to keep backups (default: 30)
- `BACKUP_RETENTION_MIN` - Minimum backups to keep (default: 3)
- `BACKUP_CLOUD_PROVIDER` - Cloud provider (s3, dropbox, gdrive, onedrive, rclone)

## db

Database operations and management.

**Usage:** `nself db <command>`

Supports migrations, schema management, seeding, and team collaboration.

## email

Email service configuration.

**Usage:** `nself email <subcommand>`

**Subcommands:**
- `setup` - Interactive email setup wizard
- `list` - Show supported providers
- `configure <provider>` - Configure specific provider
- `validate` - Check email configuration
- `test [email]` - Send test email
- `docs <provider>` - Show provider setup guide

Supports 16+ email providers including SendGrid, AWS SES, Mailgun, and SMTP.

## admin *(planned for v0.4.0)*

Admin UI management will provide visual administration in a future release.

**Status:** Not yet implemented. Planned for v0.4.0.

## search *(planned for v0.4.0)*

Enterprise search service management will be available in a future release.

**Status:** Not yet implemented. Planned for v0.4.0.

## deploy *(planned for v0.4.0)*

SSH deployment to VPS servers will be available in a future release.

**Status:** Not yet implemented. Planned for v0.4.0.

## ssl

Fully automatic SSL certificate management.

**Usage:** `nself ssl <subcommand>`

**Subcommands:**
- `bootstrap` - Generate certificates for all domains
- `status` - Show certificate status and expiry
- `renew` - Manually renew certificates
- `auto-renew` - Check and renew if needed
- `schedule` - Schedule automatic renewal
- `unschedule` - Remove automatic renewal

SSL is completely automated by default.

## Other Management Commands

- **urls** - Show all service URLs and endpoints
- **prod** - Configure production deployment with secure passwords
- **trust** - Install SSL root certificate for browsers
- **validate** - Validate configuration files with auto-fix
- **exec** - Execute commands in running containers
- **scale** - Resource scaling and management
- **metrics** - Metrics collection and observability
- **clean** - Clean Docker resources

---

# 🛠️ Development Commands

## diff

Show configuration differences with drift detection.

**Usage:** `nself diff [--drift]`

## reset

Reset project to clean state.

**Usage:** `nself reset [--confirm]`

## rollback

Rollback to previous versions or deployments.

**Usage:** `nself rollback <type>`

**Types:**
- `latest` - Rollback to latest backup
- `backup [id]` - Rollback to specific backup
- `migration [steps]` - Rollback database migrations

## monitor

Real-time monitoring dashboard.

**Usage:** `nself monitor [view]`

**Views:**
- `dashboard` - Full monitoring dashboard
- `services` - Monitor service health
- `resources` - Monitor resource usage
- `logs` - Monitor logs in real-time
- `alerts` - Monitor active alerts

## scaffold

Create new service from template.

**Usage:** `nself scaffold <type> <name>`

**Types:** `nestjs`, `golang`, `python`, `react`, `vue`, `angular`

---

# 🔧 Tool Commands

## hot_reload

Enable hot reload for development.

**Usage:** `nself hot_reload [service]`

## validate-env

Validate environment configuration.

**Usage:** `nself validate-env [--profile] [--strict] [--fix]`

---

# 📋 Other Commands

## update

Update nself to the latest version.

**Usage:** `nself update`

## version

Show current nself version and build information.

**Usage:** `nself version`

## help

Show help information.

**Usage:** `nself help [command]`

---

## Quick Reference

**Getting Started:**
```bash
nself init --wizard  # Interactive setup (v0.3.9)
nself build          # Generate infrastructure  
nself start          # Start all services
```

**Development:**
```bash
nself status         # Check service health
nself logs           # View logs
nself monitor        # Real-time dashboard
```

**Production:**
```bash
nself prod           # Generate production config
nself deploy init    # Setup deployment (v0.3.9)
nself deploy ssh     # Deploy to VPS (v0.3.9)
```

**v0.3.9 New Features:**
```bash
# Custom Services (CS_N pattern)
CS_1=api,js,3000     # Define in .env.local
nself build          # Generates custom services

# Frontend Applications
FRONTEND_APPS="app:short:prefix:port"  # Define in .env.local
nself build          # Generates frontend routing

# Multi-environment support
nself init           # Creates .env.local, .env.dev, .env.staging, .env.prod
```

**For help with any command:**
```bash
nself help <command>
nself <command> --help
```