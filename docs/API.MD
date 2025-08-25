# nself v0.3.9 Command Reference

Complete reference for all nself commands and their options in v0.3.9.

## Command Overview

```bash
nself <command> [options] [arguments]
```

## Core Commands

### init
Initialize a new nself project.

```bash
nself init [project_name] [domain]
```

**Arguments:**
- `project_name` - Project name (default: myproject)
- `domain` - Base domain (default: localhost)

**Examples:**
```bash
nself init                    # Initialize with defaults
nself init myapp              # Custom project name
nself init myapp example.com  # Custom domain
```

**Creates:**
- `.env.local` - Environment configuration
- Project directory structure
- Default database schema

---

### build
Build Docker images and generate configuration files.

```bash
nself build [--no-cache] [--parallel]
```

**Options:**
- `--no-cache` - Build without Docker cache
- `--parallel` - Build images in parallel

**Generates:**
- `docker-compose.yml`
- SSL certificates
- Nginx configuration
- Service configurations

---

### up
Start all services.

```bash
nself up [--verbose] [--skip-checks]
```

**Options:**
- `--verbose`, `-v` - Show detailed output
- `--skip-checks` - Skip pre-flight checks
- `--help`, `-h` - Show help

**Starts:**
- PostgreSQL database
- Hasura GraphQL engine
- Authentication service
- MinIO storage
- Nginx proxy
- Optional services (if enabled)

---

### down
Stop all services.

```bash
nself down [--volumes] [--rmi]
```

**Options:**
- `--volumes`, `-v` - Remove volumes (WARNING: deletes data)
- `--rmi` - Remove images
- `--help`, `-h` - Show help

---

### restart
Restart services.

```bash
nself restart [service_name]
```

**Arguments:**
- `service_name` - Specific service to restart (optional)

**Examples:**
```bash
nself restart          # Restart all services
nself restart hasura   # Restart only Hasura
```

---

### status
Show service status.

```bash
nself status [--format] [--services]
```

**Options:**
- `--format json|table` - Output format
- `--services` - Show only service names

**Shows:**
- Service health status
- Container state
- Resource usage
- Port bindings

---

### logs
View service logs.

```bash
nself logs [service] [--follow] [--tail]
```

**Arguments:**
- `service` - Service name (optional, all if omitted)

**Options:**
- `--follow`, `-f` - Follow log output
- `--tail N` - Show last N lines

**Examples:**
```bash
nself logs                # All service logs
nself logs hasura -f      # Follow Hasura logs
nself logs postgres --tail 100  # Last 100 lines
```

## Management Commands

### doctor
Run system diagnostics and health checks.

```bash
nself doctor [--verbose] [--fix]
```

**Options:**
- `--verbose`, `-v` - Detailed output
- `--fix` - Attempt auto-fixes

**Checks:**
- System requirements
- Docker configuration
- Port availability
- SSL certificates
- Service health
- Configuration validity

---

### db
Database operations.

```bash
nself db <subcommand> [options]
```

**Subcommands:**
- `migrate` - Run migrations
- `seed` - Seed database
- `reset` - Reset database
- `backup` - Create backup
- `restore` - Restore from backup
- `console` - Open database console

**Examples:**
```bash
nself db migrate          # Run pending migrations
nself db seed            # Load seed data
nself db backup          # Create backup
nself db console         # Interactive PostgreSQL
```

---

### email
Email service configuration.

```bash
nself email <provider> [--test]
```

**Providers:**
- `smtp` - Generic SMTP
- `sendgrid` - SendGrid API
- `mailgun` - Mailgun API
- `ses` - AWS SES
- `postmark` - Postmark API

**Options:**
- `--test` - Send test email

**Examples:**
```bash
nself email smtp          # Configure SMTP
nself email sendgrid --test  # Setup and test SendGrid
```

---

### urls
Display service URLs.

```bash
nself urls [--format] [--qr]
```

**Options:**
- `--format json|table` - Output format
- `--qr` - Generate QR codes

**Shows:**
- GraphQL endpoint
- Auth endpoints
- Storage URLs
- Admin console
- Service endpoints

---

### prod
Configure for production deployment.

```bash
nself prod [--domain] [--ssl] [--passwords]
```

**Options:**
- `--domain DOMAIN` - Production domain
- `--ssl` - Configure SSL
- `--passwords` - Generate secure passwords

**Configures:**
- Production environment variables
- SSL certificates
- Security headers
- Resource limits
- Backup strategy

---

### trust
Manage SSL certificates.

```bash
nself trust [--cert FILE] [--key FILE] [--ca FILE]
```

**Options:**
- `--cert` - Certificate file
- `--key` - Private key file
- `--ca` - CA certificate

**Features:**
- Install local CA
- Generate certificates
- Trust certificates
- Verify SSL setup

## Development Commands

### scaffold
Create new service from template.

```bash
nself scaffold <type> <name> [--start]
```

**Types:**
- `nest` - NestJS service
- `bull` - BullMQ worker
- `go` - Golang service
- `py` - Python service

**Options:**
- `--start` - Start service immediately

**Examples:**
```bash
nself scaffold nest api-gateway
nself scaffold bull email-worker --start
nself scaffold go websocket-server
nself scaffold py ml-service
```

---

### diff
Show configuration differences.

```bash
nself diff [file1] [file2]
```

**Arguments:**
- `file1` - First file (default: .env.local)
- `file2` - Second file (default: .env.example)

**Shows:**
- Variable differences
- Missing variables
- Value changes

---

### reset
Reset project to clean state.

```bash
nself reset [--hard] [--keep-data]
```

**Options:**
- `--hard` - Remove all data and volumes
- `--keep-data` - Preserve database data

**Resets:**
- Stops all services
- Removes containers
- Optionally removes volumes
- Optionally removes configuration

## Tool Commands

### validate-env
Validate environment configuration.

```bash
nself validate-env [file]
```

**Arguments:**
- `file` - Environment file (default: .env.local)

**Validates:**
- Required variables
- Variable format
- Value constraints
- Dependencies

---

### hot-reload
Enable hot reload for development.

```bash
nself hot-reload [service]
```

**Arguments:**
- `service` - Service to watch (optional)

**Features:**
- File watching
- Automatic rebuilds
- Service restarts
- Live updates

## System Commands

### update
Update nself to latest version.

```bash
nself update [--check] [--force]
```

**Options:**
- `--check` - Check for updates only
- `--force` - Force update

---

### version
Show version information.

```bash
nself version [--verbose]
```

**Options:**
- `--verbose`, `-v` - Detailed version info

---

### help
Show help information.

```bash
nself help [command]
```

**Arguments:**
- `command` - Show help for specific command

## Environment Variables

### Core Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PROJECT_NAME` | Project identifier | myproject |
| `BASE_DOMAIN` | Base domain for services | localhost |
| `ENV` | Environment (dev/prod) | dev |
| `COMPOSE_ENV_FILE` | Docker Compose env file | .env.local |

### Service URLs

| Variable | Description | Example |
|----------|-------------|---------|
| `HASURA_ROUTE` | Hasura subdomain | api.localhost |
| `AUTH_ROUTE` | Auth subdomain | auth.localhost |
| `STORAGE_ROUTE` | Storage subdomain | storage.localhost |

### Database Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `POSTGRES_DB` | Database name | postgres |
| `POSTGRES_USER` | Database user | postgres |
| `POSTGRES_PASSWORD` | Database password | (generated) |
| `POSTGRES_PORT` | Database port | 5432 |

### Hasura Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `HASURA_GRAPHQL_ADMIN_SECRET` | Admin secret | (generated) |
| `HASURA_GRAPHQL_JWT_SECRET` | JWT configuration | (generated) |
| `HASURA_GRAPHQL_ENABLE_CONSOLE` | Enable console | true |

### Optional Services

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_ENABLED` | Enable Redis | false |
| `FUNCTIONS_ENABLED` | Enable Functions | false |
| `DASHBOARD_ENABLED` | Enable Dashboard | false |
| `MONITORING_ENABLED` | Enable Monitoring | false |

## Hooks System

All commands support pre and post execution hooks.

### Pre-Command Hook
Executes before command:
- Validates environment
- Checks prerequisites  
- Creates directories
- Sets up logging

### Post-Command Hook
Executes after command:
- Cleanup operations
- Success/failure reporting
- Service URL display
- Metrics collection

### Disabling Hooks
```bash
NO_HOOKS=1 nself up  # Skip all hooks
```

## Auto-Fix System

Limited-scope automatic fixes for common issues:

1. **Docker Build Failures**
   - Clears cache and rebuilds

2. **Port Conflicts**
   - Stops conflicting containers (if owned by project)

3. **Missing Dependencies**
   - Installs from approved list

4. **Configuration Issues**
   - Fixes permissions and paths

### Disabling Auto-Fix
```bash
nself up --no-autofix
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Missing prerequisites |
| 3 | Configuration error |
| 4 | Docker error |
| 5 | Network error |
| 127 | Command not found |

## Logging

Logs are written to `logs/nself.log` with format:
```
[YYYY-MM-DD HH:MM:SS] LEVEL: Message
```

**Log Levels:**
- `INFO` - Informational messages
- `SUCCESS` - Successful operations
- `WARNING` - Warning messages
- `ERROR` - Error messages
- `DEBUG` - Debug information (if DEBUG=true)

## Examples

### Complete Project Setup
```bash
# Initialize project
nself init myapp api.example.com

# Build infrastructure
nself build

# Start services
nself up -d

# Check status
nself status

# View URLs
nself urls
```

### Development Workflow
```bash
# Enable hot reload
nself hot-reload

# Create new service
nself scaffold nest api-gateway

# View logs
nself logs -f

# Reset for fresh start
nself reset --keep-data
```

### Production Deployment
```bash
# Configure for production
nself prod --domain example.com --ssl

# Validate configuration
nself validate-env

# Start production services
nself up -d

# Monitor health
nself doctor
```

## Troubleshooting

### Common Issues

**Services won't start:**
```bash
nself doctor        # Run diagnostics
nself logs          # Check error logs
nself down && nself up  # Restart
```

**Port conflicts:**
```bash
nself doctor        # Identifies conflicts
nself up            # Auto-fix attempts resolution
```

**Build failures:**
```bash
nself build --no-cache  # Rebuild without cache
```

**Configuration issues:**
```bash
nself validate-env  # Check configuration
nself diff          # Compare with example
```

## Support

- **Documentation**: `/docs/` directory
- **Issues**: [GitHub Issues](https://github.com/acamarata/nself/issues)
- **Support Development**: [Patreon](https://patreon.com/acamarata)