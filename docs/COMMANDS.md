# nself Commands Reference

Complete command reference for nself v0.3.9.

**Total Commands**: 36 (including aliases)

## Version Availability
- **✅ v0.3.9 (Current)**: 34 fully functional commands + 2 partial
- **🚧 v0.4.0 (Q1 2025)**: Complete `deploy` and `search` implementations  
- **🔮 Beyond**: Advanced cloud features, Kubernetes, multi-region

## Quick Reference

```bash
# Initialize a new project
nself init

# Build and start services
nself build
nself start

# Check status
nself status

# View logs
nself logs [service]

# Stop services
nself stop
```

## Core Commands

### nself init
Initialize a new nself project with environment configuration.

```bash
nself init [options]
```

**Options:**
- `--full` - Create all environment files and schema.dbml
- `--wizard` - Interactive setup wizard
- `--admin` - Minimal admin UI setup only
- `-h, --help` - Show help message

**Creates (Basic):**
- `.env.example` - Complete reference documentation
- `.env.local` - Personal development configuration
- `.gitignore` - Security ignore rules

**Creates (--full):**
All basic files plus:
- `.env.dev` - Team-shared development defaults
- `.env.staging` - Staging environment config
- `.env.prod` - Production config (non-secrets)
- `.env.secrets` - Sensitive data template
- `schema.dbml` - Example database schema

### nself build
Generate Docker Compose configuration and build services.

```bash
nself build [options]
```

**Options:**
- `--force` - Force rebuild all services
- `--verbose` - Show detailed output
- `--no-cache` - Build without Docker cache

**Generates:**
- `docker-compose.yml` - Main services configuration
- `docker-compose.override.yml` - Development overrides
- `nginx/` - Nginx configuration
- `ssl/` - SSL certificates

### nself start
Start all enabled services.

```bash
nself start [options]
```

**Options:**
- `--verbose` - Show detailed startup logs
- `--skip-checks` - Skip port availability checks
- `--attach` - Run in foreground (Ctrl+C to stop)
- `--no-deps` - Don't start linked services

### nself stop
Stop running services.

```bash
nself stop [options] [services...]
```

**Options:**
- `-v, --volumes` - Remove volumes (WARNING: deletes all data)
- `--rmi` - Remove Docker images
- `--remove-orphans` - Remove containers for services not in compose file
- `--verbose` - Show detailed output

**Examples:**
```bash
nself stop              # Stop all services, keep data
nself stop postgres     # Stop only postgres
nself stop --volumes    # Stop and remove all data
```

### nself restart
Restart services.

```bash
nself restart [service]
```

**Examples:**
```bash
nself restart           # Restart all services
nself restart postgres  # Restart only postgres
nself restart hasura    # Restart only hasura
```

### nself status
Show service health and resource usage.

```bash
nself status [options] [service]
```

**Options:**
- `-w, --watch` - Watch mode (refresh every 5s)
- `-i, --interval N` - Set refresh interval for watch mode
- `--no-resources` - Hide resource usage information
- `--show-ports` - Show detailed port information
- `--format FORMAT` - Output format: table, json

**Shows:**
- Service status (running/stopped/unhealthy)
- Resource usage (CPU, memory)
- Service URLs
- Health check status

### nself logs
View service logs.

```bash
nself logs [options] [service]
```

**Options:**
- `-f, --follow` - Follow log output
- `-t, --timestamps` - Show timestamps
- `--since TIME` - Show logs since timestamp
- `--tail N` - Number of lines to show (default: 50)
- `--no-color` - Disable colored output

**Examples:**
```bash
nself logs              # Show all service logs
nself logs postgres     # Show postgres logs
nself logs -f hasura    # Follow hasura logs
```

## Database Commands

### nself db
Database management interface.

```bash
nself db [subcommand]
```

**Subcommands:**

#### Schema Management
- `run` - Analyze schema.dbml and generate migrations
- `sync` - Pull schema from dbdiagram.io
- `sample` - Create sample schema

#### Migrations
- `migrate:create NAME` - Create new migration
- `migrate:up [N]` - Apply N migrations (default: all)
- `migrate:down [N]` - Rollback N migrations (default: 1)
- `migrate:status` - Show migration status

#### Data Operations
- `console` - Open PostgreSQL console
- `export [--format=sql|csv|json]` - Export database
- `import FILE` - Import data from file
- `clone SOURCE DEST` - Clone database

#### Maintenance
- `optimize` - Run full optimization (VACUUM, ANALYZE, REINDEX)
- `vacuum [--full]` - Reclaim storage space
- `analyze` - Update query planner statistics
- `reindex` - Rebuild all indexes

#### Monitoring
- `connections` - Show active connections
- `locks` - Show current locks
- `kill PID` - Terminate connection by PID
- `size` - Show database sizes

#### Lifecycle
- `seed` - Seed database with initial data
- `reset` - Drop and recreate database
- `update` - Apply migrations and seeds
- `status` - Show database status

## Backup Commands

### nself backup
Comprehensive backup system with local and cloud support.

```bash
nself backup [subcommand] [options]
```

**Subcommands:**

#### Creating Backups
- `create [--type=full|database|config]` - Create backup
- `create --name NAME` - Create named backup
- `create --compress` - Compress backup

#### Managing Backups
- `list` - List all backups
- `restore BACKUP_ID` - Restore from backup
- `verify BACKUP_ID` - Verify backup integrity
- `delete BACKUP_ID` - Delete specific backup

#### Backup Pruning
- `prune --age DAYS` - Remove backups older than N days
- `prune --gfs` - Apply Grandfather-Father-Son retention
- `prune --smart` - Smart retention (keeps important backups)
- `prune --dry-run` - Preview what would be deleted

#### Cloud Backups
- `cloud setup` - Configure cloud provider (S3, GCS, Azure)
- `cloud sync` - Sync local backups to cloud
- `cloud list` - List cloud backups
- `cloud restore` - Restore from cloud

#### Scheduling
- `schedule daily HH:MM` - Daily backups
- `schedule weekly DAY HH:MM` - Weekly backups
- `schedule monthly DD HH:MM` - Monthly backups
- `schedule list` - Show scheduled backups
- `schedule remove ID` - Remove schedule

## Admin UI Commands

### nself admin
Admin UI management for monitoring and configuration.

```bash
nself admin [subcommand]
```

**Subcommands:**
- `enable` - Enable admin UI (generates temporary password)
- `disable` - Disable admin UI
- `status` - Show admin UI status and configuration
- `password [PASSWORD]` - Set admin password
- `reset` - Reset admin to defaults
- `logs` - Show admin container logs
- `open` - Open admin UI in browser

**Admin UI Features:**
- Real-time service monitoring
- Docker container management
- Database query interface
- Log viewer
- Configuration editor
- Backup management

**Access:**
- Default URL: http://localhost:3100
- Default username: admin
- Password: Set with `nself admin password` or use temporary

## Configuration Commands

### nself validate
Validate configuration files and environment.

```bash
nself validate [options]
```

**Checks:**
- Environment variables
- Docker Compose syntax
- Service dependencies
- Port conflicts
- SSL certificates
- Database connections

### nself ssl
SSL certificate management.

```bash
nself ssl [options]
```

**Options:**
- `--generate` - Generate self-signed certificates
- `--import CERT KEY` - Import existing certificates
- `--renew` - Renew certificates
- `--verify` - Verify certificate configuration

**Generates certificates for:**
- localhost
- *.local.nself.org
- Custom domains

### nself trust
Install SSL certificates in system trust store.

```bash
nself trust
```

**Supports:**
- macOS Keychain
- Linux ca-certificates
- Windows Certificate Store
- Firefox/Chrome certificate stores

### nself email
Configure email service provider.

```bash
nself email [provider]
```

**Supported Providers:**
- Development: mailpit, mailhog
- Production: sendgrid, mailgun, ses, smtp
- Transactional: postmark, sparkpost, mandrill

### nself prod
Generate production configuration.

```bash
nself prod [options]
```

**Features:**
- Generates secure passwords
- Creates .env.prod file
- Configures SSL for production
- Sets resource limits
- Enables security features

## Monitoring Commands

### nself metrics
View service metrics and performance data.

```bash
nself metrics [service] [options]
```

**Options:**
- `--interval SECONDS` - Refresh interval
- `--export FORMAT` - Export metrics (json, csv)
- `--since TIME` - Show metrics since time

### nself monitor
Real-time monitoring dashboard.

```bash
nself monitor [options]
```

**Features:**
- Service health monitoring
- Resource usage graphs
- Log streaming
- Alert notifications

### nself doctor
System diagnostics and health checks.

```bash
nself doctor [options]
```

**Checks:**
- Docker installation and version
- System resources (CPU, memory, disk)
- Network connectivity
- Port availability
- Service health
- Configuration issues

**Options:**
- `--fix` - Attempt to fix issues automatically
- `--verbose` - Show detailed diagnostics

## Development Commands

### nself exec
Execute commands in service containers.

```bash
nself exec [options] <service> [command]
```

**Options:**
- `-it` - Interactive terminal
- `-T` - Disable pseudo-TTY
- `-u USER` - Run as specific user
- `-w DIR` - Working directory

**Examples:**
```bash
nself exec postgres psql -U postgres
nself exec hasura hasura-cli console
nself exec -it nginx /bin/bash
```

### nself diff
Show configuration differences.

```bash
nself diff [env1] [env2]
```

**Examples:**
```bash
nself diff              # Compare .env.local with defaults
nself diff dev prod     # Compare dev and prod configs
```

### nself reset
Reset project to clean state.

```bash
nself reset [options]
```

**Options:**
- `--hard` - Remove all data and configurations
- `--soft` - Keep configurations, reset data
- `--confirm` - Skip confirmation prompt

### nself clean
Clean up Docker resources.

```bash
nself clean [options]
```

**Options:**
- `--all` - Remove all nself containers and volumes
- `--images` - Remove Docker images
- `--orphans` - Remove orphaned containers
- `--system` - Run Docker system prune

## Deployment Commands

### nself deploy
Deploy to remote servers via SSH. *(Partial implementation - full features coming in v0.4.0)*

```bash
nself deploy [target] [options]
```

**Subcommands:**
- `init` - Initialize deployment configuration
- `ssh` - Deploy to VPS server via SSH
- `status` - Check deployment status

**Options:**
- `--host HOST` - Target host
- `--user USER` - SSH user
- `--key PATH` - SSH key path
- `--dry-run` - Preview deployment

**Note:** Full deployment automation with zero-downtime deployments, GitHub webhooks, and multi-environment sync planned for v0.4.0.

### nself scale
Scale service resources.

```bash
nself scale [service] [replicas]
```

**Examples:**
```bash
nself scale hasura 3    # Scale hasura to 3 replicas
nself scale postgres 2  # Scale postgres to 2 replicas
```

### nself rollback
Rollback to previous version.

```bash
nself rollback [options]
```

**Options:**
- `--version VERSION` - Specific version to rollback to
- `--list` - List available versions
- `--dry-run` - Preview rollback

## Utility Commands

### nself urls
Show all service URLs.

```bash
nself urls
```

**Shows:**
- GraphQL API endpoints
- Admin interfaces
- Storage URLs
- Custom service endpoints

### nself version
Show nself version information.

```bash
nself version [options]
```

**Options:**
- `--check` - Check for updates
- `--verbose` - Show detailed version info

### nself update
Update nself CLI to latest version.

```bash
nself update [options]
```

**Options:**
- `--check` - Check for updates only
- `--force` - Force update even if current
- `--beta` - Update to beta version

### nself help
Show help information.

```bash
nself help [command]
```

**Examples:**
```bash
nself help          # Show general help
nself help db       # Show database command help
nself help backup   # Show backup command help
```

### nself scaffold
Generate new service from template.

```bash
nself scaffold <type> <name> [options]
```

**Types:**
- `api` - REST API service
- `worker` - Background worker service
- `cron` - Scheduled job service
- `websocket` - WebSocket service

**Options:**
- `--language <lang>` - Language (js, ts, python, go)
- `--framework <fw>` - Framework to use
- `--port <port>` - Service port

**Examples:**
```bash
nself scaffold api user-service --language ts
nself scaffold worker email-processor --language python
```

### nself search
Manage search services with 6 different engine options. *(Partial implementation - full features coming in v0.4.0)*

```bash
nself search <subcommand> [options]
```

**Subcommands:**
- `status` - Show search service status
- `enable` - Enable search service
- `disable` - Disable search service
- `configure <engine>` - Configure search engine (meilisearch, typesense, zinc, elasticsearch, opensearch, sonic)
- `reindex [index]` - Rebuild search index
- `clear [index]` - Clear search index
- `import` - Import data into search
- `export` - Export search data
- `dashboard` - Open search dashboard (if available)
- `health` - Check search service health
- `logs` - View search service logs

**Supported Engines:**
- `meilisearch` - Fast, typo-tolerant search (default)
- `typesense` - Lightning-fast, typo-tolerant search
- `zinc` - Lightweight Elasticsearch alternative
- `elasticsearch` - Industry standard search & analytics
- `opensearch` - Open-source Elasticsearch fork
- `sonic` - Ultra-lightweight search

**Examples:**
```bash
nself search status                         # Check current search status
nself search configure meilisearch          # Switch to Meilisearch
nself search configure elasticsearch        # Switch to Elasticsearch
nself search reindex products               # Reindex products
nself search import --file=data.json        # Import data
nself search dashboard                      # Open dashboard
```

**Configuration:**
Set in `.env`:
```bash
SEARCH_ENABLED=true
SEARCH_ENGINE=meilisearch  # or typesense, zinc, elasticsearch, opensearch, sonic
```

See [SEARCH.md](./SEARCH.md) for complete search documentation.

### nself functions
Manage serverless functions for your application.

```bash
nself functions <subcommand> [options]
```

**Subcommands:**
- `status` - Show functions service status
- `enable` - Enable serverless functions
- `disable` - Disable serverless functions
- `list` - List all deployed functions
- `create <name>` - Create a new function
- `delete <name>` - Delete a function
- `test <name>` - Test a function locally
- `logs <name>` - View function logs
- `deploy [name]` - Deploy function(s) to production

**Examples:**
```bash
nself functions status              # Check functions service status
nself functions enable              # Enable serverless functions
nself functions create my-function  # Create new function
nself functions test my-function    # Test function locally
nself functions deploy              # Deploy all functions
nself functions logs my-function    # View function logs
```

**Configuration:**
Set in `.env`:
```bash
FUNCTIONS_ENABLED=true
FUNCTIONS_PORT=3030
```

### nself mlflow
Manage MLflow for machine learning experiment tracking and model registry.

```bash
nself mlflow <subcommand> [options]
```

**Subcommands:**
- `status` - Show MLflow service status
- `enable` - Enable MLflow tracking server
- `disable` - Disable MLflow tracking server
- `open` / `ui` / `dashboard` - Open MLflow UI in browser
- `configure` - Configure MLflow settings
- `logs` - View MLflow service logs
- `test` - Test MLflow connectivity

**Examples:**
```bash
nself mlflow status                 # Check MLflow status
nself mlflow enable                 # Enable MLflow tracking
nself mlflow open                   # Open MLflow UI (localhost:5000)
nself mlflow configure              # Configure MLflow settings
nself mlflow logs                   # View MLflow logs
```

**Configuration:**
Set in `.env`:
```bash
MLFLOW_ENABLED=true
MLFLOW_PORT=5000
MLFLOW_ARTIFACT_ROOT=./mlruns
MLFLOW_BACKEND_STORE_URI=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/mlflow
```

**Features:**
- Experiment tracking
- Model versioning
- Model registry
- Artifact storage
- Metrics visualization
- Parameter tracking
- Integration with Ray for distributed training

### nself up
Alias for `nself start` (Docker Compose compatibility).

```bash
nself up [options]
```

Same options as `nself start`.

### nself down
Alias for `nself stop` (Docker Compose compatibility).

```bash
nself down [options]
```

Same options as `nself stop`.

## Environment Variables

### Core Settings
```bash
# Project configuration
PROJECT_NAME=myproject
BASE_DOMAIN=local.nself.org
ENV=dev                     # dev, staging, prod

# Database
POSTGRES_DB=nhost
POSTGRES_USER=postgres
POSTGRES_PASSWORD=secure-password

# Hasura
HASURA_GRAPHQL_ADMIN_SECRET=admin-secret
HASURA_JWT_KEY=jwt-secret-minimum-32-chars
```

### Service Toggles
```bash
# Enable/disable services
POSTGRES_ENABLED=true
HASURA_ENABLED=true
AUTH_ENABLED=true
STORAGE_ENABLED=true
REDIS_ENABLED=false
FUNCTIONS_ENABLED=false
DASHBOARD_ENABLED=false
NSELF_ADMIN_ENABLED=true
```

### Advanced Configuration
```bash
# Resource limits
POSTGRES_MAX_CONNECTIONS=100
POSTGRES_SHARED_BUFFERS=256MB
HASURA_MAX_CONNECTIONS=50

# Monitoring
PROMETHEUS_ENABLED=false
GRAFANA_ENABLED=false
LOKI_ENABLED=false

# Custom services
SERVICES_ENABLED=true
NESTJS_SERVICES=api,workers
GOLANG_SERVICES=analytics
PYTHON_SERVICES=ml,data
```

## File Structure

```
project/
├── .env.local              # Development configuration
├── .env.secrets            # Sensitive data (git-ignored)
├── .env                    # Production override
├── .gitignore              # Git ignore rules
├── docker-compose.yml      # Generated services
├── docker-compose.override.yml  # Dev overrides
├── nginx/
│   ├── default.conf        # Nginx configuration
│   └── ssl/                # SSL configurations
├── ssl/
│   └── certificates/       # SSL certificates
├── backups/                # Local backups
├── migrations/             # Database migrations
├── seeds/                  # Database seeds
└── schemas/                # Database schemas
```

## Common Workflows

### Development Setup
```bash
# 1. Initialize project
nself init

# 2. Configure admin UI
nself admin enable
nself admin password mypassword

# 3. Build and start
nself build
nself start

# 4. Check status
nself status
nself admin open
```

### Production Deployment
```bash
# 1. Generate production config
nself prod

# 2. Set up SSL
nself ssl --generate
nself trust

# 3. Configure backups
nself backup schedule daily 02:00
nself backup cloud setup

# 4. Deploy
nself deploy production
```

### Database Management
```bash
# Create and apply migrations
nself db migrate:create add_users_table
nself db migrate:up

# Backup before changes
nself backup create --name pre-migration

# Seed data
nself db seed

# Monitor
nself db connections
nself db size
```

### Troubleshooting
```bash
# Run diagnostics
nself doctor

# Check logs
nself logs --tail 100
nself logs postgres -f

# Reset if needed
nself stop --volumes
nself reset --hard
nself build --force
```

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Misuse of command
- `126` - Command cannot execute
- `127` - Command not found
- `130` - Terminated by Ctrl+C

## Getting Help

```bash
# Command help
nself help
nself [command] --help

# Documentation
cat docs/COMMANDS.md
cat docs/TROUBLESHOOTING.md

# Version info
nself version
```

## See Also

- [Architecture Guide](ARCHITECTURE.md)
- [Backup Guide](BACKUP_GUIDE.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [Environment Configuration](ENVIRONMENT_CONFIGURATION.md)
- [Contributing](CONTRIBUTING.md)