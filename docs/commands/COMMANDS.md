# nself Commands Reference

**Version 0.4.5** | **34 Commands** | **Full CLI Reference**

---

## Quick Reference

```bash
# Core workflow
nself init                    # Initialize project
nself build                   # Generate configs
nself start                   # Start services
nself stop                    # Stop services

# Database (v0.4.4)
nself db migrate up           # Run migrations
nself db schema scaffold      # Create schema template
nself db schema apply         # Full workflow
nself db seed                 # Seed data
nself db backup               # Create backup

# Deployment (v0.4.3)
nself env create prod         # Create environment
nself deploy prod             # Deploy to production

# Provider Support (v0.4.5)
nself providers init hetzner  # Configure cloud provider
nself provision aws           # Provision infrastructure
nself sync pull staging       # Sync from staging
nself ci init github          # Generate CI/CD workflow
nself completion bash         # Shell completions
```

---

## Command Categories

| Category | Commands | Description |
|----------|----------|-------------|
| **Core** | 8 | Project lifecycle |
| **Status** | 6 | Monitoring and debugging |
| **Management** | 4 | Configuration and admin |
| **Services** | 6 | Service-specific operations |
| **Database** | 1 (10+ subcommands) | Database management |
| **Deployment** | 4 | Environment and deployment |
| **Provider** | 5 | Cloud providers and sync (v0.4.5) |

**Total: 34 commands**

---

## Core Commands (8)

### `nself init`

Initialize a new nself project.

```bash
nself init              # Basic setup
nself init --demo       # Full demo with all services
nself init --wizard     # Interactive configuration
nself init --full       # All environment files
```

**Options:**
| Flag | Description |
|------|-------------|
| `--demo` | All 25 services enabled |
| `--wizard` | Interactive prompts |
| `--full` | Creates .env.dev, .env.staging, .env.prod |
| `--force` | Reinitialize existing project |
| `--quiet` | Minimal output |

---

### `nself build`

Generate docker-compose.yml, nginx configs, and service files.

```bash
nself build             # Standard build
nself build --verbose   # Detailed output
nself build --clean     # Remove generated files first
```

**What it generates:**
- `docker-compose.yml` - Service definitions
- `nginx/` - Reverse proxy configs
- `postgres/` - Database init scripts
- `services/` - Custom service code (first time only)
- `ssl/` - Certificates

---

### `nself start`

Start all services.

```bash
nself start             # Start with smart defaults
nself start --fresh     # Force recreate all containers
nself start --verbose   # Detailed output
```

**Options:**
| Flag | Description |
|------|-------------|
| `--fresh` | Force recreate containers |
| `--verbose` | Detailed output |
| `--skip-health-checks` | Don't wait for health |
| `--timeout N` | Health check timeout (seconds) |

---

### `nself stop`

Stop all services.

```bash
nself stop              # Stop services
nself stop --remove     # Stop and remove containers
```

---

### `nself restart`

Restart services.

```bash
nself restart           # Restart all
nself restart postgres  # Restart specific service
```

---

### `nself reset`

Reset project to clean state.

```bash
nself reset             # With confirmation
nself reset --force     # Skip confirmation
```

**Warning:** Deletes all data. Blocked in production.

---

### `nself clean`

Remove generated files and Docker resources.

```bash
nself clean             # Remove generated files
nself clean --docker    # Also remove Docker resources
nself clean --all       # Everything including backups
```

---

### `nself version`

Show version information.

```bash
nself version           # Full version info
nself -v                # Short version
```

---

## Status Commands (6)

### `nself status`

Show service health and status.

```bash
nself status            # All services
nself status postgres   # Specific service
nself status --json     # JSON output
```

---

### `nself logs`

View service logs.

```bash
nself logs              # All services
nself logs postgres     # Specific service
nself logs -f           # Follow logs
nself logs --tail 100   # Last 100 lines
```

---

### `nself exec`

Execute command in a service container.

```bash
nself exec postgres bash
nself exec hasura hasura-cli console
```

---

### `nself urls`

Show all service URLs.

```bash
nself urls              # All URLs
nself urls --json       # JSON output
```

---

### `nself doctor`

Diagnose issues and suggest fixes.

```bash
nself doctor            # Full diagnosis
nself doctor --fix      # Auto-fix issues
```

---

### `nself help`

Show help for commands.

```bash
nself help              # General help
nself help db           # Command-specific help
nself db --help         # Also works
```

---

## Management Commands (4)

### `nself update`

Update nself to latest version.

```bash
nself update            # Update nself
nself update --check    # Check for updates only
```

---

### `nself ssl`

Manage SSL certificates.

```bash
nself ssl               # Generate/renew certs
nself ssl --staging     # Use Let's Encrypt staging
```

---

### `nself trust`

Trust SSL certificates in system keychain.

```bash
nself trust             # Add certs to keychain
```

---

### `nself admin`

Manage admin dashboard.

```bash
nself admin enable      # Enable admin UI
nself admin disable     # Disable admin UI
nself admin open        # Open in browser
nself admin password    # Set password
```

---

## Service Commands (6)

### `nself email`

Configure email service.

```bash
nself email             # Show current config
nself email setup       # Interactive setup
nself email test        # Send test email
```

---

### `nself search`

Configure search service.

```bash
nself search            # Show current config
nself search index      # Reindex data
nself search status     # Index status
```

---

### `nself functions`

Manage serverless functions.

```bash
nself functions deploy  # Deploy functions
nself functions logs    # View function logs
```

---

### `nself mlflow`

Manage MLflow service.

```bash
nself mlflow open       # Open UI
nself mlflow status     # Check status
```

---

### `nself metrics`

Configure monitoring stack.

```bash
nself metrics           # Status
nself metrics enable    # Enable monitoring
nself metrics disable   # Disable monitoring
```

---

### `nself monitor`

Access monitoring dashboards.

```bash
nself monitor           # Open Grafana
nself monitor prometheus # Open Prometheus
nself monitor logs      # Open Loki
```

---

## Database Command (v0.4.4)

### `nself db`

Comprehensive database management. See [DB.md](DB.md) for complete reference.

```bash
# Migrations
nself db migrate up           # Run pending migrations
nself db migrate down         # Rollback last migration
nself db migrate create NAME  # Create new migration
nself db migrate status       # Show migration status

# Schema (NEW in v0.4.4)
nself db schema scaffold basic  # Create from template
nself db schema import FILE     # Import DBML
nself db schema apply FILE      # Full workflow
nself db schema diagram         # Export to DBML

# Seeding
nself db seed                 # Run all seeds
nself db seed users           # Seed users (env-aware)
nself db seed create NAME     # Create seed file

# Mock Data
nself db mock                 # Generate mock data
nself db mock auto            # Auto-generate from schema
nself db mock --seed 123      # Reproducible data

# Backup/Restore
nself db backup               # Create backup
nself db backup list          # List backups
nself db restore              # Restore latest

# Types
nself db types                # Generate TypeScript
nself db types go             # Generate Go structs
nself db types python         # Generate Python

# Shell & Query
nself db shell                # Interactive psql
nself db shell --readonly     # Read-only shell
nself db query "SQL"          # Execute query

# Inspection
nself db inspect              # Overview
nself db inspect size         # Table sizes
nself db inspect slow         # Slow queries

# Data Operations
nself db data export users    # Export table
nself db data anonymize       # Anonymize PII
```

---

## Deployment Commands (v0.4.3)

### `nself env`

Manage environments.

```bash
nself env                     # List environments
nself env create prod         # Create environment
nself env switch staging      # Switch environment
nself env diff staging prod   # Compare environments
```

See [ENV.md](ENV.md) for complete reference.

---

### `nself deploy`

Deploy to remote servers.

```bash
nself deploy staging          # Deploy to staging
nself deploy prod             # Deploy to production
nself deploy prod --dry-run   # Preview deployment
nself deploy status           # Check deployment status
nself deploy rollback         # Rollback to previous
```

See [DEPLOY.md](DEPLOY.md) for complete reference.

---

### `nself prod`

Production configuration.

```bash
nself prod                    # Show prod config
nself prod check              # Validate config
nself prod harden             # Apply security settings
```

See [PROD.md](PROD.md) for complete reference.

---

### `nself staging`

Staging environment.

```bash
nself staging                 # Show staging config
nself staging create          # Create staging env
```

See [STAGING.md](STAGING.md) for complete reference.

---

## Provider Commands (v0.4.5)

### `nself providers`

Manage cloud provider credentials.

```bash
nself providers                 # List configured providers
nself providers init aws        # Configure AWS credentials
nself providers init hetzner    # Configure Hetzner
nself providers status          # Check provider status
nself providers costs --compare # Compare costs across providers
nself providers remove hetzner  # Remove provider config
```

**Supported Providers:**
AWS, GCP, Azure, DigitalOcean, Hetzner, Linode, Vultr, IONOS, OVH, Scaleway

---

### `nself provision`

One-command infrastructure provisioning.

```bash
nself provision hetzner         # Default small size
nself provision aws --size medium
nself provision do --size large
nself provision aws --estimate  # Show cost only
nself provision gcp --dry-run   # Preview resources
nself provision export terraform # Export as Terraform
```

**Sizes:** small (1-2 vCPU, 2GB), medium (2 vCPU, 4GB), large (4 vCPU, 8GB), xlarge (8 vCPU, 16GB)

---

### `nself sync`

Sync databases, config, and files between environments.

```bash
nself sync pull staging         # Pull database from staging
nself sync pull prod --anonymize # Pull prod with PII anonymization
nself sync push staging         # Push to staging
nself sync files pull staging uploads/  # Sync files
nself sync config diff staging  # Compare configs
```

See [SYNC.md](SYNC.md) for complete reference.

---

### `nself ci`

Generate CI/CD workflows.

```bash
nself ci init github            # Generate GitHub Actions
nself ci init gitlab            # Generate GitLab CI
nself ci validate               # Validate config
nself ci status                 # Check CI status
```

---

### `nself completion`

Generate shell completions.

```bash
nself completion bash >> ~/.bashrc
nself completion zsh >> ~/.zshrc
nself completion fish >> ~/.config/fish/completions/nself.fish
nself completion install bash   # Auto-install
```

---

## Environment Variables

All commands respect these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `ENV` | Current environment | `local` |
| `NSELF_LOG_LEVEL` | Log verbosity | `info` |
| `NSELF_AUTO_FIX` | Auto-fix issues | `false` |
| `NSELF_SKIP_HOOKS` | Skip git hooks | `false` |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Invalid arguments |
| `3` | Configuration error |
| `4` | Docker error |
| `5` | Database error |
| `126` | Permission denied |
| `127` | Command not found |

---

## Common Workflows

### New Project

```bash
mkdir myapp && cd myapp
nself init
nself build
nself start
nself db schema scaffold basic
nself db schema apply schema.dbml
```

### Daily Development

```bash
nself start                  # Start working
nself logs -f                # Watch logs
nself db migrate up          # Run new migrations
nself stop                   # Done for the day
```

### Deploy to Production

```bash
nself env create prod production
# Edit .environments/prod/server.json
nself deploy prod --dry-run  # Preview
nself deploy prod            # Deploy
```

### Database Backup

```bash
nself db backup              # Create backup
nself db backup list         # List backups
ENV=production nself db restore backup.sql  # Restore
```

---

## See Also

- [DB.md](DB.md) - Complete database command reference
- [DEPLOY.md](DEPLOY.md) - Deployment command reference
- [ENV.md](ENV.md) - Environment command reference
- [Quick Start](../guides/Quick-Start.md) - Getting started
- [Database Workflow](../guides/DATABASE-WORKFLOW.md) - Schema to production

---

*Last Updated: January 23, 2026 | Version: 0.4.5*
