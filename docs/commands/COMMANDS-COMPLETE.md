# nself Commands - Complete Reference

This is the authoritative, complete reference for all nself commands, their options, environment variables, and behavior.

## Table of Contents

1. [Core Commands](#core-commands)
   - [init](#nself-init) - Initialize a new project
   - [build](#nself-build) - Generate and build configuration
   - [start](#nself-start) - Start services with smart defaults
   - [stop](#nself-stop) - Stop services and cleanup
   - [restart](#nself-restart) - Smart restart of services
   - [reset](#nself-reset) - Reset project to clean state
2. [Service Management](#service-management)
   - [status](#nself-status) - Check service health
   - [logs](#nself-logs) - View service logs
   - [exec](#nself-exec) - Execute commands in containers
   - [scale](#nself-scale) - Scale service replicas
3. [Configuration](#configuration)
   - [urls](#nself-urls) - Display service URLs
   - [admin](#nself-admin) - Admin UI management
   - [doctor](#nself-doctor) - System diagnostics
4. [Environment Variables](#environment-variables)

---

## Core Commands

### nself init

Initialize a new nself project with environment configuration.

#### Usage
```bash
nself init [OPTIONS]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `--demo` | Initialize with demo configuration (all services enabled) | false |
| `--full` | Create all environment files (.dev, .staging, .prod, schema.dbml) | false |
| `--wizard` | Interactive setup wizard | false |
| `--admin` | Minimal admin UI setup only | false |
| `-h, --help` | Show help message | - |

#### Environment Variables
None specific to init command.

#### Files Created

**Basic Mode (default):**
- `.env` - Local development configuration (git-ignored)
- `.env.example` - Complete reference with all options
- `.gitignore` - Security-focused ignore rules

**Demo Mode (`--demo`):**
- All basic files plus pre-configured `.env` with:
  - All optional services enabled
  - Monitoring stack enabled
  - 4 custom services (CS_1 to CS_4)
  - 2 frontend apps configured

**Full Mode (`--full`):**
- All basic files plus:
  - `.env.dev` - Development environment defaults
  - `.env.staging` - Staging environment configuration
  - `.env.prod` - Production configuration
  - `.env.secrets` - Secrets template (git-ignored)
  - `schema.dbml` - Example database schema

#### Examples
```bash
# Basic initialization
nself init

# Demo with all services
nself init --demo

# Full environment setup
nself init --full

# Interactive wizard
nself init --wizard
```

---

### nself build

Generate Docker Compose configuration and build all services.

#### Usage
```bash
nself build [OPTIONS]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `--force` | Force rebuild, overwrite existing services | false |
| `--verbose` | Show detailed build output | false |
| `--no-cache` | Build Docker images without cache | false |
| `-h, --help` | Show help message | - |

#### Environment Variables
| Variable | Description | Default |
|----------|-------------|---------|
| `ENV` | Environment (dev/staging/prod) | dev |
| `PROJECT_NAME` | Project identifier | (directory name) |
| `BASE_DOMAIN` | Base domain for services | localhost |
| `*_ENABLED` | Enable optional services | false |
| `CS_N` | Custom service definitions | - |

#### Generated Files
```
docker-compose.yml         # Main services configuration
nginx/
  ├── nginx.conf          # Main nginx configuration
  ├── includes/           # Security headers, gzip settings
  └── sites/              # Per-service routing configs
postgres/
  └── init/
      └── 00-init.sql     # Database initialization
ssl/
  ├── cert.pem           # Self-signed certificate
  └── key.pem            # Private key
services/                 # Custom services from templates
monitoring/               # Monitoring configurations
  ├── prometheus/
  ├── grafana/
  ├── loki/
  └── alertmanager/
```

#### Examples
```bash
# Standard build
nself build

# Force rebuild with verbose output
nself build --force --verbose

# Clean build without cache
nself build --no-cache
```

---

### nself start

Start all services with smart defaults and configurable health checks.

#### Usage
```bash
nself start [OPTIONS]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `-v, --verbose` | Show detailed Docker output | false |
| `-d, --debug` | Show debug information (implies verbose) | false |
| `-h, --help` | Show help message | - |
| `--skip-health-checks` | Skip health check validation | false |
| `--timeout N` | Health check timeout in seconds | 120 |
| `--fresh` | Force recreate all containers | false |
| `--clean-start` | Remove all containers before starting | false |
| `--quick` | Quick start with relaxed health checks | false |

#### Environment Variables (Smart Defaults)

All these variables are **OPTIONAL** - the command works with sensible defaults.

| Variable | Description | Default | Options |
|----------|-------------|---------|---------|
| `NSELF_START_MODE` | Container start strategy | smart | smart, fresh, force |
| `NSELF_HEALTH_CHECK_TIMEOUT` | Max seconds to wait for health | 120 | 30-600 |
| `NSELF_HEALTH_CHECK_INTERVAL` | Seconds between health checks | 2 | 1-10 |
| `NSELF_HEALTH_CHECK_REQUIRED` | Percent services required healthy | 80 | 0-100 |
| `NSELF_SKIP_HEALTH_CHECKS` | Skip all health validation | false | true/false |
| `NSELF_DOCKER_BUILD_TIMEOUT` | Max seconds for Docker builds | 300 | 60-1800 |
| `NSELF_CLEANUP_ON_START` | Container cleanup strategy | auto | auto, always, never |
| `NSELF_PARALLEL_LIMIT` | Parallel container starts | 5 | 1-20 |
| `NSELF_LOG_LEVEL` | Output verbosity | info | debug, info, warn, error |

#### Start Modes

- **smart** (default): Intelligently handles existing containers
  - Resumes stopped containers
  - Keeps running healthy containers
  - Only recreates problematic containers

- **fresh**: Force recreates all containers
  - Uses `docker-compose up --force-recreate`
  - Good for configuration changes

- **force**: Most aggressive cleanup
  - Removes all containers first
  - Starts completely fresh

#### Health Check Behavior

- Shows real-time progress of services becoming healthy
- Accepts partial success (default 80% healthy)
- Doesn't fail on timeout if services are running
- Can be skipped entirely with `--skip-health-checks`

#### Common Configurations

```bash
# Development - Quick iteration
NSELF_HEALTH_CHECK_REQUIRED=60 NSELF_HEALTH_CHECK_TIMEOUT=60 nself start

# Production - Full validation
NSELF_HEALTH_CHECK_REQUIRED=100 NSELF_HEALTH_CHECK_TIMEOUT=180 nself start

# CI/CD - Clean state
NSELF_START_MODE=fresh NSELF_CLEANUP_ON_START=always nself start

# Debugging - Maximum visibility
NSELF_LOG_LEVEL=debug NSELF_SKIP_HEALTH_CHECKS=true nself start
```

#### Examples
```bash
# Start with smart defaults
nself start

# Quick development start
nself start --quick

# Fresh start with all containers recreated
nself start --fresh

# Skip health checks for fastest startup
nself start --skip-health-checks

# Custom timeout
nself start --timeout 180

# Debug mode with full output
nself start --debug
```

#### Runtime Files

- `.env.runtime` - Merged environment configuration (created on start, deleted on stop)

---

### nself stop

Stop running services and optionally clean up resources.

#### Usage
```bash
nself stop [OPTIONS] [SERVICES...]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `-v, --volumes` | Remove volumes (WARNING: deletes all data) | false |
| `--rmi` | Remove Docker images | false |
| `--remove-orphans` | Remove orphaned containers | false |
| `--graceful [N]` | Graceful shutdown timeout in seconds | 30 |
| `--verbose` | Show detailed output | false |
| `-h, --help` | Show help message | - |

#### Behavior

1. Optionally performs graceful stop with timeout (default 30s)
2. Stops all running containers for the project
3. Removes `.env.runtime` file (ensures env changes take effect on restart)
4. Optionally removes volumes (persistent data)
5. Optionally removes images
6. Cleans up orphaned containers if requested

#### Environment Variables
| Variable | Description | Default |
|----------|-------------|---------|
| `NSELF_STOP_TIMEOUT` | Default graceful shutdown timeout | 30 |

#### Examples
```bash
# Stop all services, preserve data
nself stop

# Stop with graceful 60 second timeout
nself stop --graceful 60

# Stop specific service
nself stop postgres

# Stop and remove all data
nself stop --volumes

# Full cleanup
nself stop --volumes --rmi

# Stop with detailed output
nself stop --verbose

# Quick stop with custom timeout
NSELF_STOP_TIMEOUT=10 nself stop
```

---

### nself restart

Smart restart of services with minimal downtime.

#### Usage
```bash
nself restart [OPTIONS] [SERVICES...]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `--all, -a` | Restart all services (not just unhealthy) | false |
| `--smart, -s` | Smart mode - only restart unhealthy services | true |
| `--verbose, -v` | Show detailed output | false |
| `-h, --help` | Show help message | - |

#### Behavior

**Smart Mode (default):**
- Detects changes to .env files and docker-compose.yml
- Automatically rebuilds configuration if changes detected
- Only restarts unhealthy or stopped services
- Keeps healthy services running
- Minimal downtime
- Verifies health status after restart

**All Mode:**
- Forces restart of all services regardless of health
- Complete refresh of all containers
- Does not check for configuration changes
- Verifies health status after restart

**Change Detection:**
- Compares modification times of .env files vs container start times
- Compares docker-compose.yml modification vs container start times
- Automatically runs `nself build --force` if changes detected
- Applies changes with `docker compose up -d --build`

**Health Verification:**
- After restarting, waits 5 seconds for services to initialize
- Checks and reports how many services are healthy
- Shows warning if services are still pending health checks

#### Examples
```bash
# Smart restart (only unhealthy)
nself restart

# Restart all services
nself restart --all

# Restart specific service
nself restart postgres

# Verbose restart
nself restart --verbose
```

---

### nself reset

Reset project to clean state by removing all generated files and containers.

#### Usage
```bash
nself reset [OPTIONS]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `--soft` | Keep Docker volumes (preserve data) | false |
| `--hard` | Remove everything including .env files | false |
| `--keep-env` | Preserve environment files | false |
| `--no-backup` | Skip creating backup archive | false |
| `--force` | Skip confirmation prompt | false |
| `-h, --help` | Show help message | - |

#### What Gets Removed

**Default Reset:**
- All Docker containers for project
- Generated docker-compose.yml
- nginx/ directory
- ssl/ directory
- services/ directory (generated from templates)
- monitoring/ directory
- .env.runtime file
- *.log files and logs/ directory
- .nself/ and .cache/ directories
- docker-compose.override.yml

**Soft Reset (`--soft`):**
- Same as default but keeps Docker volumes (data preserved)

**With `--no-backup`:**
- Skips creating backup archive before reset

**With `--keep-env`:**
- Preserves all .env files during reset

**Hard Reset (`--hard`):**
- Everything from default reset
- All .env files (except .env.example)
- Docker volumes (all data)
- Docker images for project
- Essentially returns to pre-init state

#### Examples
```bash
# Standard reset with backup
nself reset

# Reset without creating backup
nself reset --no-backup

# Soft reset (keeps volumes/data)
nself reset --soft

# Hard reset (complete cleanup)
nself reset --hard

# Reset but keep env files
nself reset --keep-env

# Force without confirmation
nself reset --force

# Quick reset without backup or confirmation
nself reset --force --no-backup
```

---

## Service Management

### nself status

Check health status of all running services.

#### Usage
```bash
nself status [OPTIONS] [SERVICES...]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Output in JSON format | false |
| `--watch` | Continuous monitoring mode | false |
| `--interval N` | Watch interval in seconds | 5 |
| `-h, --help` | Show help message | - |

#### Output Shows
- Service name and status (running/stopped/unhealthy)
- Container health check status
- CPU and memory usage
- Restart count
- Uptime

---

### nself logs

View service logs with filtering and following options.

#### Usage
```bash
nself logs [OPTIONS] [SERVICE]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `-f, --follow` | Follow log output | false |
| `--tail N` | Number of lines to show | 100 |
| `--since TIME` | Show logs since timestamp | - |
| `--timestamps` | Show timestamps | false |
| `--no-color` | Disable colored output | false |
| `-h, --help` | Show help message | - |

---

### nself exec

Execute commands inside running containers.

#### Usage
```bash
nself exec [OPTIONS] SERVICE COMMAND [ARGS...]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `-it` | Interactive terminal | false |
| `--user USER` | Run as specific user | - |
| `--workdir DIR` | Working directory | - |
| `-h, --help` | Show help message | - |

---

## Configuration

### nself urls

Display all service URLs and endpoints.

#### Usage
```bash
nself urls [OPTIONS]
```

#### Options
| Flag | Description | Default |
|------|-------------|---------|
| `--json` | Output in JSON format | false |
| `--qr` | Show QR codes for URLs | false |
| `--public` | Show only public URLs | false |
| `-h, --help` | Show help message | - |

#### Output Format
```
╔════════════════════════════════════════════════════════════════╗
║                     Service URLs                               ║
╠════════════════════════════════════════════════════════════════╣
║ Service          │ URL                                          ║
╟─────────────────┼──────────────────────────────────────────────╢
║ Hasura GraphQL  │ http://api.localhost                         ║
║ Auth Service    │ http://auth.localhost                        ║
║ Admin UI        │ http://admin.localhost                       ║
╚════════════════════════════════════════════════════════════════╝
```

---

## Environment Variables

### Loading Order

Environment files are loaded in this priority order (later overrides earlier):

1. `.env.example` - Reference defaults
2. `.env.[environment]` - Environment-specific (dev/staging/prod)
3. `.env.local` - Personal overrides
4. `.env` - Highest priority
5. `.env.runtime` - Generated at start (merged result)

### Core Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PROJECT_NAME` | Project identifier | (directory name) | Yes |
| `ENV` | Environment (dev/staging/prod) | dev | Yes |
| `BASE_DOMAIN` | Base domain for services | localhost | Yes |

### Service Enable Flags

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_ENABLED` | Enable Redis cache | false |
| `MINIO_ENABLED` | Enable MinIO storage | false |
| `FUNCTIONS_ENABLED` | Enable serverless functions | false |
| `NSELF_ADMIN_ENABLED` | Enable admin UI | false |
| `MLFLOW_ENABLED` | Enable MLflow | false |
| `MAILPIT_ENABLED` | Enable MailPit (dev email) | false |
| `MEILISEARCH_ENABLED` | Enable MeiliSearch | false |
| `MONITORING_ENABLED` | Enable full monitoring stack | false |

### Custom Services (CS_N)

Format: `CS_N=name:template:port[:route]`

Examples:
```bash
CS_1=api:express-js:8001:api
CS_2=worker:bullmq-js:8002
CS_3=grpc:grpc:8003:grpc
CS_4=ml:fastapi:8004:ml
```

### Database Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_VERSION` | PostgreSQL version | 16-alpine |
| `POSTGRES_HOST` | Database host | postgres |
| `POSTGRES_PORT` | Database port | 5432 |
| `POSTGRES_DB` | Database name | nself |
| `POSTGRES_USER` | Database user | postgres |
| `POSTGRES_PASSWORD` | Database password | (required) |

### Secrets (Production)

These must be changed from defaults in production:

| Variable | Description |
|----------|-------------|
| `HASURA_GRAPHQL_ADMIN_SECRET` | Hasura admin secret |
| `HASURA_JWT_KEY` | JWT signing key (min 32 chars) |
| `AUTH_JWT_SECRET` | Auth service JWT secret |
| `MINIO_ROOT_PASSWORD` | MinIO admin password |
| `GRAFANA_ADMIN_PASSWORD` | Grafana admin password |

---

## Command Execution Flow

### Typical Workflow

```bash
# 1. Initialize project
nself init --demo

# 2. Customize .env file
vim .env

# 3. Build configuration
nself build

# 4. Start services
nself start

# 5. Check status
nself status

# 6. View URLs
nself urls

# 7. Monitor logs
nself logs -f

# 8. Stop when done
nself stop

# 9. Restart to apply changes
nself restart
```

### Environment Change Workflow

```bash
# 1. Stop services
nself stop

# 2. Edit environment
vim .env

# 3. Rebuild if needed
nself build

# 4. Start with changes
nself start
```

### Complete Reset Workflow

```bash
# 1. Hard reset everything
nself reset --hard

# 2. Re-initialize
nself init

# 3. Restore configuration
cp .env.backup .env

# 4. Rebuild and start
nself build && nself start
```

---

## Troubleshooting

### Services Not Starting

```bash
# Check with debug output
nself start --debug

# Skip health checks
nself start --skip-health-checks

# Increase timeout
nself start --timeout 300

# Force fresh start
nself start --fresh
```

### Port Conflicts

```bash
# Force cleanup before start
NSELF_CLEANUP_ON_START=always nself start

# Or stop everything first
nself stop --volumes
nself start --fresh
```

### Health Check Issues

```bash
# Lower health requirements
NSELF_HEALTH_CHECK_REQUIRED=60 nself start

# Or skip health checks entirely
NSELF_SKIP_HEALTH_CHECKS=true nself start
```

### Configuration Not Applied

```bash
# Ensure .env.runtime is regenerated
nself stop
nself start

# Or force rebuild
nself build --force
nself start --fresh
```

---

## See Also

- [Environment Variables Reference](../configuration/ENVIRONMENT-VARIABLES.md)
- [Start Command Options](../configuration/START-COMMAND-OPTIONS.md)
- [Docker Compose Configuration](../architecture/Docker-Compose.md)
- [Template System](../guides/Templates.md)