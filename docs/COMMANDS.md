# nself Commands Reference

Complete reference for all nself commands and subcommands.

## Command Tree

```
nself
├── 🚀 Core Commands
│   ├── init          Initialize a new project
│   ├── build         Build project structure and Docker images
│   ├── start         Start all services
│   ├── stop          Stop all services
│   ├── restart       Restart all services
│   ├── status        Show service status with health monitoring
│   └── logs          View service logs
│
├── ⚙️ Management Commands
│   ├── doctor        Run enterprise system diagnostics
│   ├── backup        Backup and restore with S3 support
│   │   ├── create    Create backups (full, database, config, incremental)
│   │   ├── list      List available backups
│   │   ├── restore   Restore from backup with point-in-time recovery
│   │   ├── prune     Remove old backups
│   │   ├── verify    Verify backup integrity
│   │   ├── schedule  Schedule automated backups
│   │   ├── export    Export backup to external location
│   │   ├── import    Import backup from external location
│   │   ├── snapshot  Create point-in-time snapshot
│   │   └── rollback  Rollback to specific point in time
│   │
│   ├── db            Database operations
│   ├── email         Email service configuration
│   │   ├── setup     Interactive email setup wizard
│   │   ├── list      Show all supported providers
│   │   ├── configure Configure specific provider
│   │   ├── validate  Check email configuration
│   │   ├── test      Send test email
│   │   └── docs      Show provider setup guide
│   │
│   ├── ssl           SSL certificate management (fully automatic)
│   │   ├── bootstrap Generate SSL certificates
│   │   ├── renew     Renew public wildcard certificate
│   │   ├── status    Show certificate status and expiry
│   │   ├── auto-renew Check and renew if needed
│   │   ├── schedule  Schedule automatic renewal checks
│   │   └── unschedule Remove automatic renewal
│   │
│   ├── urls          Show service URLs
│   ├── prod          Configure for production deployment
│   ├── trust         Install SSL root certificate
│   ├── validate      Validate configuration files
│   ├── exec          Execute commands in containers
│   ├── scale         Resource scaling management
│   ├── metrics       Metrics and observability
│   │   ├── enable    Enable metrics collection
│   │   ├── disable   Disable metrics collection
│   │   ├── status    Show metrics status
│   │   ├── dashboard Open metrics dashboard
│   │   ├── export    Export metrics data
│   │   └── configure Configure metrics providers
│   │
│   └── clean         Clean Docker resources
│
├── 🛠️ Development Commands
│   ├── diff          Show configuration differences
│   │   └── --drift   Configuration drift detection
│   ├── reset         Reset project to clean state
│   ├── rollback      Rollback to previous version
│   │   ├── latest    Rollback to latest backup
│   │   ├── backup    Rollback to specific backup
│   │   ├── migration Rollback database migrations
│   │   ├── deployment Rollback to previous deployment
│   │   └── config    Rollback configuration changes
│   │
│   └── monitor       Real-time monitoring dashboard
│       ├── dashboard Full monitoring dashboard
│       ├── services  Monitor service health
│       ├── resources Monitor resource usage
│       ├── logs      Monitor logs in real-time
│       └── alerts    Monitor active alerts
│
├── 🔧 Tool Commands
│   ├── scaffold      Create new service from template
│   └── hot_reload    Enable hot reload for development
│
└── 📋 Other Commands
    ├── update        Update nself to latest version
    ├── version       Show version information
    └── help          Show this help message
```

---

# 🚀 Core Commands

## init

Initialize a new nself project with smart defaults and configuration files.

Creates `.env.local` with sensible defaults, `.env.example` for reference, and sets up the basic project structure. This is always the first command to run in a new project directory.

**Usage:** `nself init`

**What it creates:**
- `.env.local` - Your main configuration file
- `.env.example` - Reference documentation for all options
- Backup of existing files with `.backup` suffix if they exist

## build

Generate all project infrastructure and configuration files from your `.env.local` settings.

Creates Docker Compose files, nginx configurations, SSL certificates, database initialization scripts, and service templates. This command is idempotent - it only rebuilds what's changed unless forced.

**Usage:** `nself build [--force]`

**What it generates:**
- `docker-compose.yml` - Main service orchestration
- `nginx/` - Web server configuration with SSL
- SSL certificates (automatic, fully managed)
- Database initialization scripts
- Service templates for enabled microservices
- Environment validation and auto-fix

## start

Start all services with comprehensive error handling and auto-fix capabilities.

Runs pre-flight checks, validates configuration, auto-maintains SSL certificates, and starts all Docker containers. Includes intelligent retry logic and automatic problem resolution.

**Usage:** `nself start [--verbose] [--skip-checks] [--attach]`

**Features:**
- Pre-flight system checks
- Automatic SSL certificate validation and renewal
- Port conflict resolution
- Service health monitoring
- Auto-fix for common issues
- Intelligent retry with backoff

## stop

Stop all running services gracefully.

Stops all Docker containers associated with the project while preserving data volumes and networks for quick restart.

**Usage:** `nself stop`

## restart

Restart all services (equivalent to stop + start).

Performs a clean restart of all services, useful when configuration has changed or services need a fresh start.

**Usage:** `nself restart`

## status

Show comprehensive service status with health monitoring, resource usage, and dependency visualization.

Provides real-time information about all services including health status, resource consumption, uptime, and service dependencies.

**Usage:** `nself status [--watch] [--json]`

**Information shown:**
- Service health and status indicators
- Resource usage (CPU, memory)
- Container restart counts
- Service dependencies
- Port mappings and URLs
- Health check results

## logs

View and follow service logs with advanced filtering and search capabilities.

Stream logs from one or all services with timestamps, filtering, and search functionality for debugging and monitoring.

**Usage:** `nself logs [service] [--follow] [--tail] [--grep] [--since]`

**Features:**
- Real-time log streaming
- Service-specific or aggregate logs
- Search and filtering with grep patterns
- Timestamp formatting
- Tail recent entries
- Follow mode for live monitoring

---

# ⚙️ Management Commands

## doctor

Run comprehensive enterprise-grade system diagnostics and health checks.

Performs deep system analysis including SSL certificate validation, DNS configuration, kernel parameters, network diagnostics, security compliance scanning, and resource predictions.

**Usage:** `nself doctor [--quick] [--fix] [--verbose]`

**Checks performed:**
- SSL certificate health and expiry
- DNS resolution and configuration
- Kernel parameter validation for production
- Network connectivity and latency
- Security compliance (HIPAA, SOC2 ready)
- Resource usage predictions
- Service dependency analysis
- Database performance metrics

## backup

Enterprise-grade backup and restore system with S3 support, point-in-time recovery, and automated scheduling.

Comprehensive backup solution supporting local and remote storage, incremental backups, integrity verification, and automated scheduling.

### backup create

Create various types of backups with optional compression and encryption.

**Usage:** `nself backup create [type] [name]`

**Types:** `full`, `database`, `config`, `incremental`

Creates timestamped backups with integrity checksums. Supports automatic compression and encryption for sensitive data.

### backup list

List all available backups with details including size, type, and integrity status.

**Usage:** `nself backup list [--remote] [--format json]`

Shows local and S3 backups with metadata, creation dates, and verification status.

### backup restore

Restore from backup with point-in-time recovery capabilities.

**Usage:** `nself backup restore <name> [type] [--point-in-time]`

Supports selective restoration of databases, configurations, or full system state with timestamp-based recovery.

### backup prune

Remove old backups based on retention policies.

**Usage:** `nself backup prune [days] [--dry-run]`

Automatically cleans up old backups while preserving important milestones and maintaining backup integrity.

### backup verify

Verify backup integrity and completeness.

**Usage:** `nself backup verify <name>`

Validates backup checksums, tests restoration capability, and ensures data integrity.

### backup schedule

Schedule automated backups with flexible frequency options.

**Usage:** `nself backup schedule <frequency>`

**Frequencies:** `hourly`, `daily`, `weekly`, `monthly`, or custom cron expressions

Sets up automated backup routines with proper logging and error notification.

### backup export

Export backups to external locations for disaster recovery.

**Usage:** `nself backup export <name> <path>`

Facilitates backup migration and off-site storage for compliance and disaster recovery.

### backup import

Import backups from external sources.

**Usage:** `nself backup import <path>`

Enables backup restoration from external storage or migration between environments.

### backup snapshot

Create lightweight point-in-time snapshots.

**Usage:** `nself backup snapshot [label]`

Quick snapshots for testing and development, useful before major changes.

### backup rollback

Rollback to specific point in time using backup data.

**Usage:** `nself backup rollback <timestamp>`

Time-based recovery with automatic service restart and health verification.

## db

Database operations including migrations, schema management, and team collaboration tools.

Comprehensive database management with support for migrations, seeding, schema generation from DBML, and team synchronization.

**Usage:** `nself db <command>`

**Commands include:**
- `run` - Generate migrations from schema.dbml
- `update` - Apply pending migrations safely
- `seed` - Apply environment-specific seed data
- `status` - Check database state and pending migrations
- `revert` - Restore from backup
- `sync` - Pull schema from dbdiagram.io

## email

Email service configuration supporting 16+ providers with zero-config development.

Complete email setup and management system with support for major email providers, testing, and validation.

### email setup

Interactive email setup wizard that guides you through provider selection and configuration.

**Usage:** `nself email setup`

Walks through provider selection, API key configuration, and testing to ensure email is working correctly.

### email list

Show all 16+ supported email providers with their features and requirements.

**Usage:** `nself email list`

**Supported providers:**
- SendGrid, AWS SES, Mailgun, Postmark
- Gmail, Outlook, Yahoo, SMTP
- Postfix, Sendmail, and more

### email configure

Configure specific email provider with guided setup.

**Usage:** `nself email configure <provider>`

Provider-specific configuration with validation and testing.

### email validate

Check current email configuration and test connectivity.

**Usage:** `nself email validate`

Validates API keys, SMTP settings, and tests actual email delivery.

### email test

Send test email to verify configuration.

**Usage:** `nself email test [email]`

Sends test message to verify email delivery is working correctly.

### email docs

Show detailed setup guide for specific provider.

**Usage:** `nself email docs <provider>`

Provider-specific documentation with step-by-step setup instructions.

## ssl

Fully automatic SSL certificate management with zero-configuration required.

SSL is completely automated - certificates are generated, renewed, and trusted automatically. These commands provide manual control for advanced users.

### ssl bootstrap

Generate SSL certificates for all detected domains and microservices.

**Usage:** `nself ssl bootstrap`

Auto-detects all domains from nginx configs, microservices, and environment variables. Generates certificates for everything found.

### ssl renew

Manually renew public wildcard certificates.

**Usage:** `nself ssl renew`

Forces renewal of Let's Encrypt certificates when DNS provider is configured.

### ssl status

Show detailed certificate status, expiry dates, and configuration.

**Usage:** `nself ssl status`

Displays certificate information, expiry dates, trust status, and renewal schedules.

### ssl auto-renew

Check certificate health and renew if expiring soon (7+ day threshold).

**Usage:** `nself ssl auto-renew [--force]`

Automated renewal with safer 7-day margin, used by cron for daily checks.

### ssl schedule

Schedule automatic certificate renewal checks.

**Usage:** `nself ssl schedule [frequency]`

Sets up daily monitoring and renewal automation via cron.

### ssl unschedule

Remove automatic renewal schedule.

**Usage:** `nself ssl unschedule`

Disables automated certificate monitoring and renewal.

## urls

Show all service URLs and endpoints.

**Usage:** `nself urls`

Displays URLs for all running services including API endpoints, dashboards, and management interfaces.

## prod

Configure production deployment with secure password generation.

**Usage:** `nself prod`

Generates production-ready configuration with secure passwords and production-optimized settings.

## trust

Install SSL root certificate for trusted HTTPS in browsers.

**Usage:** `nself trust`

One-time installation of root CA to enable green lock icons in browsers without warnings.

## validate

Validate configuration files with auto-fix capabilities.

**Usage:** `nself validate [--profile] [--strict] [--fix] [--quiet]`

Comprehensive configuration validation with automatic problem resolution and environment-specific checks.

## exec

Execute commands in running containers with full Docker exec capabilities.

**Usage:** `nself exec [options] <service> [command]`

**Features:**
- Interactive terminal support (-it)
- User and working directory options
- Environment variable injection
- Smart command defaults per service

## scale

Resource scaling and management with auto-scaling capabilities.

**Usage:** `nself scale <service> [--cpu] [--memory] [--replicas] [--auto]`

Configure CPU limits, memory constraints, replica counts, and auto-scaling parameters for optimal resource utilization.

## metrics

Metrics collection and observability with multiple provider support.

Comprehensive monitoring system supporting Prometheus, Datadog, New Relic, and custom metrics providers.

### metrics enable

Enable metrics collection with provider selection.

**Usage:** `nself metrics enable [--provider] [--port]`

Starts metrics collection with configurable providers and endpoints.

### metrics disable

Disable metrics collection.

**Usage:** `nself metrics disable`

Stops metrics collection and cleanup monitoring resources.

### metrics status

Show current metrics collection status and configuration.

**Usage:** `nself metrics status`

Displays active metrics providers, collection status, and dashboard URLs.

### metrics dashboard

Open metrics dashboard in browser.

**Usage:** `nself metrics dashboard`

Launches web-based metrics dashboard for real-time monitoring.

### metrics export

Export metrics data in various formats.

**Usage:** `nself metrics export [--format] [--output]`

Export historical metrics data for analysis or migration.

### metrics configure

Configure metrics providers and collection parameters.

**Usage:** `nself metrics configure <provider>`

Provider-specific configuration for Prometheus, Datadog, New Relic, etc.

## clean

Clean Docker resources including containers, images, volumes, and networks.

**Usage:** `nself clean [--containers] [--images] [--volumes] [--networks] [--all]`

Cleanup unused Docker resources to free disk space and reset environment.

---

# 🛠️ Development Commands

## diff

Show configuration differences with drift detection capabilities.

**Usage:** `nself diff [file1] [file2] [--drift] [--apply-changes]`

### diff --drift

Advanced configuration drift detection that identifies services not running that should be, modified files, and configuration inconsistencies.

**Features:**
- Service state drift detection
- File modification tracking via checksums
- Automatic reconciliation options
- Integration with build system

## reset

Reset project to clean state by removing all data and containers.

**Usage:** `nself reset [--confirm]`

Complete project reset while preserving configuration files. Useful for fresh starts and testing.

## rollback

Rollback to previous versions, deployments, or configurations with comprehensive recovery options.

### rollback latest

Rollback to the most recent backup automatically.

**Usage:** `nself rollback latest`

Quick recovery to last known good state with automatic service restart.

### rollback backup

Rollback to specific backup by ID or timestamp.

**Usage:** `nself rollback backup [id]`

Selective backup restoration with data integrity verification.

### rollback migration

Rollback database migrations by specified number of steps.

**Usage:** `nself rollback migration [steps]`

Database-specific rollback using Hasura CLI integration.

### rollback deployment

Rollback to previous deployment configuration.

**Usage:** `nself rollback deployment`

Restore previous deployment state including Docker configs and environment settings.

### rollback config

Rollback configuration file changes.

**Usage:** `nself rollback config`

Restore previous configuration from automatic backups.

## monitor

Real-time monitoring dashboard with interactive terminal interface.

Comprehensive monitoring system providing live service health, resource usage, logs, and alerts in an interactive terminal interface.

### monitor dashboard

Full-featured monitoring dashboard with all views and controls.

**Usage:** `nself monitor dashboard`

**Keyboard controls:**
- `q/Q` - Quit
- `r/R` - Refresh immediately  
- `s` - Switch to services view
- `c` - Switch to resources view
- `l` - Switch to logs view
- `a` - Switch to alerts view
- `Space` - Pause/resume auto-refresh

### monitor services

Monitor service health with detailed status information.

**Usage:** `nself monitor services`

Real-time service health monitoring with restart counts, health checks, and dependency status.

### monitor resources

Monitor resource usage including CPU, memory, and disk utilization.

**Usage:** `nself monitor resources`

System and container resource monitoring with usage graphs and alerts.

### monitor logs

Monitor logs in real-time with scrolling and filtering.

**Usage:** `nself monitor logs`

Live log streaming from all services with search and filtering capabilities.

### monitor alerts

Monitor active alerts and system notifications.

**Usage:** `nself monitor alerts`

Alert dashboard showing unhealthy services, resource warnings, and system notifications.

---

# 🔧 Tool Commands

## scaffold

Create new service from template with automatic integration.

**Usage:** `nself scaffold <type> <name>`

**Types:** `nestjs`, `golang`, `python`, `react`, `vue`, `angular`

Generates complete service templates with Docker integration, configuration, and example code.

## hot_reload

Enable hot reload for development with automatic file watching.

**Usage:** `nself hot_reload [service]`

Configures volume mounts and file watching for rapid development iteration.

---

# 📋 Other Commands

## update

Update nself to the latest version with automatic migration.

**Usage:** `nself update`

Downloads latest version from GitHub releases with configuration preservation and migration support.

## version

Show current nself version and build information.

**Usage:** `nself version`

Displays version number, build date, and system information.

## help

Show help information for commands.

**Usage:** `nself help [command]`

Displays general help or command-specific documentation with examples and options.

---

## Quick Reference

**Getting Started:**
```bash
nself init     # Initialize project
nself build    # Generate infrastructure  
nself start    # Start all services
```

**Development:**
```bash
nself status   # Check service health
nself logs     # View logs
nself monitor  # Real-time dashboard
```

**Production:**
```bash
nself prod     # Generate production config
nself backup   # Create backups
nself doctor   # Health diagnostics
```

**For help with any command:**
```bash
nself help <command>
nself <command> --help
```