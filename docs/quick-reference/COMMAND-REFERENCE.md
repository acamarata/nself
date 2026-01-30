# nself CLI Command Reference

> Quick reference guide for all nself commands - Optimized for printing

**Version:** v0.9.5 | **Total Commands:** 80+ commands with 200+ subcommands

---

## Core Commands (7)

```bash
nself init [--demo|--simple]           # Initialize project
nself build [--clean|--no-cache]       # Generate configs
nself start [--fresh|--verbose]        # Start services
nself stop [service]                   # Stop services
nself restart [service]                # Restart services
nself reset [--force]                  # Reset to clean state
nself clean [--all]                    # Clean Docker resources
```

**Key flags:**
- `--verbose` - Detailed output
- `--skip-health-checks` - Skip validation
- `--timeout <seconds>` - Health check timeout

---

## Database Commands (1 with 11 subcommands)

```bash
# Migrations
nself db migrate [up|down|create <name>|status|fresh]

# Schema
nself db schema [scaffold <template>|import <file>|apply <file>|diagram]

# Data
nself db seed [dataset]                # Seed data
nself db mock [auto|--seed N]          # Generate mock data
nself db backup [--name <name>]        # Create backup
nself db restore <file>                # Restore backup

# Operations
nself db shell [--readonly]            # Interactive psql
nself db query <sql>                   # Execute SQL
nself db types [typescript|go|python]  # Generate types
nself db inspect [size|slow]           # Database inspection
nself db data [export <table>|anonymize] # Data operations
```

**Schema templates:** `saas`, `ecommerce`, `blog`, `analytics`

---

## Multi-Tenant Commands (1 with 32+ subcommands) - v0.9.0

```bash
# Core
nself tenant init                      # Initialize multi-tenancy
nself tenant create <name> [--plan]    # Create tenant
nself tenant list                      # List all tenants
nself tenant show <id>                 # Show details
nself tenant suspend|activate <id>     # Lifecycle
nself tenant delete <id>               # Delete tenant
nself tenant stats                     # Statistics

# Members
nself tenant member add <tenant> <user> [role]
nself tenant member remove <tenant> <user>
nself tenant member list <tenant>

# Settings
nself tenant setting set <tenant> <key> <value>
nself tenant setting get <tenant> <key>
nself tenant setting list <tenant>

# Billing
nself tenant billing usage             # Usage statistics
nself tenant billing invoice [list|show|download|pay]
nself tenant billing subscription [show|upgrade|downgrade]
nself tenant billing payment [list|add|remove]
nself tenant billing quota             # Check quota
nself tenant billing plan [list|show|compare]
nself tenant billing export --format csv

# Branding
nself tenant branding create <name>
nself tenant branding set-colors --primary #hex
nself tenant branding set-fonts --heading "Font"
nself tenant branding upload-logo <file>
nself tenant branding set-css <file>
nself tenant branding preview

# Domains
nself tenant domains add <domain>
nself tenant domains verify <domain>
nself tenant domains ssl <domain>
nself tenant domains health <domain>
nself tenant domains remove <domain>

# Email Templates
nself tenant email list
nself tenant email edit <template>
nself tenant email preview <template>
nself tenant email test <template> <email>

# Themes
nself tenant themes create <name>
nself tenant themes edit <name>
nself tenant themes activate <name>
nself tenant themes preview <name>
nself tenant themes export <name>
nself tenant themes import <path>
```

---

## OAuth Commands (1 with 6 subcommands) - v0.9.0

```bash
nself oauth install                    # Install OAuth service
nself oauth enable --providers <list>  # Enable providers
nself oauth disable --providers <list> # Disable providers
nself oauth config <provider>          # Configure credentials
  --client-id=<id>
  --client-secret=<secret>
  --tenant-id=<id>                     # Microsoft only
nself oauth test <provider>            # Test configuration
nself oauth list                       # List all providers
nself oauth status                     # Service status
```

**Providers:** `google`, `github`, `slack`, `microsoft`

---

## Storage Commands (1 with 7 subcommands) - v0.9.0

```bash
nself storage init                     # Initialize storage
nself storage upload <file>            # Upload file
  --dest <path>                        # Destination path
  --thumbnails                         # Generate thumbnails
  --virus-scan                         # Scan for viruses
  --compression                        # Compress large files
  --all-features                       # Enable all features
nself storage list [prefix]            # List files
nself storage delete <path>            # Delete file
nself storage config                   # Configure pipeline
nself storage status                   # Pipeline status
nself storage test                     # Test uploads
nself storage graphql-setup            # Generate GraphQL integration
```

---

## Service Management (1 with 15+ subcommands)

```bash
# Core
nself service list                     # List optional services
nself service enable <service>         # Enable service
nself service disable <service>        # Disable service
nself service status [service]         # Service status
nself service restart <service>        # Restart service
nself service logs <service> [-f]      # Service logs

# Templates
nself service init                     # Initialize from template
nself service scaffold                 # Scaffold new service
nself service wizard                   # Creation wizard
nself service search                   # Search services

# Admin UI
nself service admin status|open|users|config|dev

# Email
nself service email test|inbox|config

# Search
nself service search index|query <term>|stats

# Functions
nself service functions deploy|invoke <fn>|logs|list

# MLflow
nself service mlflow ui|experiments|runs|artifacts

# Storage (MinIO)
nself service storage buckets|upload|download|presign

# Cache (Redis)
nself service cache stats|flush|keys
```

---

## Deployment Commands (1 with 12 subcommands)

```bash
# Basic Deployment
nself deploy staging                   # Deploy to staging
nself deploy production                # Deploy to production

# Preview Environments
nself deploy preview                   # Create preview
nself deploy preview list              # List previews
nself deploy preview destroy <id>      # Destroy preview

# Canary Deployment
nself deploy canary [--percentage N]   # Start canary
nself deploy canary promote            # Promote to 100%
nself deploy canary rollback           # Rollback canary
nself deploy canary status             # Canary status

# Blue-Green Deployment
nself deploy blue-green                # Deploy to inactive
nself deploy blue-green switch         # Switch traffic
nself deploy blue-green rollback       # Rollback switch
nself deploy blue-green status         # Show active

# Utilities
nself deploy rollback                  # Rollback deployment
nself deploy check [--fix]             # Pre-deploy validation
nself deploy status                    # Deployment status
```

**Legacy shortcuts:**
```bash
nself staging                          # = nself deploy staging
nself prod                             # = nself deploy production
```

---

## Cloud Infrastructure (1 with 9+ subcommands) - v0.4.7

```bash
# Providers
nself provider list                    # List 26+ providers
nself provider init <provider>         # Configure credentials
nself provider validate                # Validate config
nself provider info <provider>         # Provider details

# Server Management
nself provider server create <provider> [--size]
nself provider server destroy <server>
nself provider server list
nself provider server status [server]
nself provider server ssh <server>
nself provider server add <ip>
nself provider server remove <server>

# Cost Management
nself provider cost estimate <provider>
nself provider cost compare

# Quick Deploy
nself provider deploy quick <provider>
nself provider deploy full <provider>
```

**Providers:** AWS, GCP, Azure, DigitalOcean, Linode, Vultr, Hetzner, OVH, and 18+ more

**Legacy aliases:**
```bash
nself providers                        # = nself provider
nself provision <provider>             # = nself provider server create
nself servers                          # = nself provider server
```

---

## Kubernetes & Helm (2 commands) - v0.4.7

```bash
# Kubernetes
nself k8s init                         # Initialize K8s config
nself k8s convert [--output|--namespace]
nself k8s apply [--dry-run]
nself k8s deploy [--env <env>]
nself k8s status
nself k8s logs <service> [-f]
nself k8s scale <service> <replicas>
nself k8s rollback <service>
nself k8s delete
nself k8s cluster [list|connect|info]
nself k8s namespace [list|create|delete|switch]

# Helm
nself helm init [--from-compose]
nself helm generate
nself helm install [--env <env>]
nself helm upgrade
nself helm rollback
nself helm uninstall
nself helm list
nself helm status
nself helm values
nself helm template
nself helm package
nself helm repo [add|remove|update|list]
```

---

## Observability & Monitoring (10 commands)

```bash
# Status & Health
nself status [service] [--json|--watch|--all-envs]
nself health [check|service <name>|watch|history]
nself doctor [--fix|--check <category>]

# Logs & Execution
nself logs [service] [-f|--tail N|--since <time>]
nself exec <service> <command>

# URLs & Monitoring
nself urls [--env|--diff|--json]
nself monitor [grafana|prometheus|alertmanager]
nself metrics [profile <type>|view]

# History & Audit
nself history [show|deployments|migrations|search]
nself audit [logs|events <user>]
```

---

## Security Commands (10 commands)

```bash
# Security Scanning
nself security scan [passwords|mfa|suspicious]
nself security devices|incidents|events <user>|webauthn

# Authentication
nself auth users|roles|providers

# MFA & Devices
nself mfa enable|disable|status
nself devices list|approve <id>|revoke <id>

# Roles & Permissions
nself roles list|create <name>|assign <user> <role>

# Secrets & Vault
nself secrets list|add <key> <value>|rotate
nself vault init|status|unseal

# SSL & Trust
nself ssl generate|renew|info
nself trust [--system]

# Rate Limiting & Webhooks
nself rate-limit config|status
nself webhooks list|test <url>
```

---

## Performance & Optimization (4 commands) - v0.4.6

```bash
# Performance Profiling
nself perf profile [service]
nself perf analyze
nself perf slow-queries
nself perf report
nself perf dashboard
nself perf suggest

# Benchmarking
nself bench run [target]
nself bench baseline
nself bench compare [file]
nself bench stress [target] --users N
nself bench report

# Scaling
nself scale <service> [--cpu N|--memory N|--replicas N]
nself scale <service> --auto --min N --max N
nself scale status

# Migration
nself migrate <source> <target> [--dry-run]
nself migrate diff <source> <target>
nself migrate sync <source> <target>
nself migrate rollback
```

---

## Developer Tools (6 commands)

```bash
# Dev Tools - v0.8.0
nself dev sdk generate [typescript|python]
nself dev docs [generate|openapi]
nself dev test [init|fixtures|factory|snapshot|run]
nself dev mock <entity> <count>

# Frontend Management
nself frontend [status|list|add|remove|deploy|logs|env]

# CI/CD Generation
nself ci init [github|gitlab|circleci]
nself ci validate
nself ci status

# Shell Completion
nself completion [bash|zsh|fish]
nself completion install <shell>

# Documentation
nself docs generate|openapi
```

---

## Plugin System (1 command) - v0.4.8

```bash
# Plugin Management
nself plugin list [--installed|--category]
nself plugin install <name>
nself plugin remove <name> [--keep-data]
nself plugin update [name|--all]
nself plugin updates
nself plugin refresh
nself plugin status [name]
nself plugin init                      # Create plugin template

# Stripe Plugin
nself plugin stripe init|sync|check
nself plugin stripe customers [list|show <id>]
nself plugin stripe subscriptions [list|show <id>]
nself plugin stripe invoices [list|show <id>]
nself plugin stripe webhook status|test

# GitHub Plugin
nself plugin github init|sync
nself plugin github repos list
nself plugin github issues [list|show <id>]
nself plugin github prs [list|show <id>]
nself plugin github workflows list
nself plugin github webhook status

# Shopify Plugin
nself plugin shopify init|sync
nself plugin shopify products [list|show <id>]
nself plugin shopify orders [list|show <id>]
nself plugin shopify customers list
nself plugin shopify webhook status
```

---

## Configuration (4 commands)

```bash
# Configuration
nself config show|get <key>|set <key> <value>
nself config list|edit|validate
nself config diff <env1> <env2>
nself config export|import <file>
nself config reset

# Environment
nself env [list]
nself env create <name> <type>
nself env switch <env>
nself env diff <env1> <env2>
nself env validate
nself env access [--check <env>]

# Sync
nself sync db|files|config|full <source> <target>
nself sync pull <env>
nself sync auto [--setup|--stop]
nself sync watch [--path|--interval]
nself sync status|history

# Validation
nself validate [--fix|--strict]
```

---

## Utilities (5 commands)

```bash
# Help & Version
nself help [command]
nself version [--short|--json|--check]

# Updates
nself update [--check|--version <ver>|--force]

# Zero-Downtime Upgrades - v0.8.0
nself upgrade perform|rolling|rollback|status

# Admin UI
nself admin [start|stop]
```

**Aliases:**
```bash
nself up                               # = nself start
nself down                             # = nself stop
nself -v                               # = nself version --short
```

---

## Global Flags

Available on most commands:

```bash
-h, --help                             # Show help
--version                              # Show version
--json                                 # JSON output
--quiet                                # Minimal output
--verbose                              # Detailed output
--debug                                # Debug mode
--env <env>                            # Target environment
--format <format>                      # Output format (table, json, csv)
```

---

## Environment Variables

### Core Configuration

```bash
PROJECT_NAME=myapp
ENV=dev|staging|prod
BASE_DOMAIN=localhost

POSTGRES_DB=myapp_db
POSTGRES_PASSWORD=secure
HASURA_GRAPHQL_ADMIN_SECRET=secret
```

### Optional Services

```bash
REDIS_ENABLED=true
MINIO_ENABLED=true
NSELF_ADMIN_ENABLED=true
MAILPIT_ENABLED=true
MEILISEARCH_ENABLED=true
MLFLOW_ENABLED=true
FUNCTIONS_ENABLED=true
```

### Monitoring

```bash
MONITORING_ENABLED=true                # Enables all 10 services
```

### Multi-Tenancy (v0.9.0)

```bash
MULTI_TENANCY_ENABLED=true
REALTIME_ENABLED=true
```

### Custom Services

```bash
CS_1=api:express-js:8001
CS_2=worker:bullmq-js:8002
CS_3=grpc:grpc:8003
```

### nself Behavior

```bash
NSELF_START_MODE=smart|fresh|force
NSELF_HEALTH_CHECK_TIMEOUT=120
NSELF_HEALTH_CHECK_REQUIRED=80
NSELF_SKIP_HEALTH_CHECKS=false
NSELF_LOG_LEVEL=info|debug|warn|error
```

---

## Exit Codes

```bash
0   # Success
1   # General error
2   # Invalid arguments
3   # Configuration error
4   # Docker error
5   # Database error
126 # Permission denied
127 # Command not found
```

---

## Quick Workflows

### New Project Setup

```bash
nself init --demo
nself build
nself start
nself urls
```

### Daily Development

```bash
nself start
nself status
nself logs -f
nself db shell
nself stop
```

### Database Development

```bash
nself db schema scaffold saas
# Edit schema.dbml
nself db schema apply schema.dbml
nself db types
```

### Deployment

```bash
nself env switch staging
nself deploy check
nself deploy staging
nself deploy production --blue-green
```

### Multi-Tenant SaaS

```bash
nself tenant init
nself tenant create "Acme Corp" --plan pro
nself tenant billing usage
nself tenant domains add app.example.com
nself tenant branding upload-logo logo.png
```

### Monitoring

```bash
nself status --watch
nself monitor
nself perf dashboard
nself security scan
```

---

## Service Count Summary

### Docker Containers (Demo Config)

- **Required Services:** 4 (Postgres, Hasura, Auth, Nginx)
- **Optional Services:** 7 (Admin, MinIO, Redis, Functions, MLflow, Email, Search)
- **Monitoring Bundle:** 10 (Prometheus, Grafana, Loki, Promtail, Tempo, Alertmanager, cAdvisor, Node Exporter, Postgres Exporter, Redis Exporter)
- **Custom Services:** 4 (CS_1 through CS_4)

**Total Docker Containers:** 25
**Frontend Apps:** 2 (external, not in Docker)
**Total Routes:** 21

---

## Common Service Routes

### Required
```
/                                      # Application root
api.local.nself.org                    # Hasura GraphQL
auth.local.nself.org                   # Authentication
```

### Optional Services
```
admin.local.nself.org                  # nself Admin
minio.local.nself.org                  # MinIO Console
functions.local.nself.org              # Functions runtime
mail.local.nself.org                   # MailPit UI
search.local.nself.org                 # MeiliSearch
mlflow.local.nself.org                 # MLflow
```

### Monitoring
```
grafana.local.nself.org                # Grafana
prometheus.local.nself.org             # Prometheus
alertmanager.local.nself.org           # Alertmanager
```

### Custom Services
```
express-api.local.nself.org            # Express API (CS_1)
grpc-api.local.nself.org               # gRPC service (CS_3)
ml-api.local.nself.org                 # Python API (CS_4)
```

### Frontend Apps
```
app1.local.nself.org                   # Frontend App 1
app2.local.nself.org                   # Frontend App 2
```

---

## Version History

| Version | Commands Added |
|---------|----------------|
| **0.9.5** | Feature parity & security hardening |
| **0.9.0** | `tenant` (32+), `oauth` (7), `storage` (8) |
| **0.8.0** | `dev`, `realtime`, `org`, `security`, `upgrade` |
| **0.4.8** | `plugin` with Stripe/GitHub/Shopify |
| **0.4.7** | `provider`, `service`, `k8s`, `helm` |
| **0.4.6** | `perf`, `bench`, `scale`, `migrate`, `health` |
| **0.4.5** | `providers`, `provision`, `sync`, `ci` |
| **0.4.4** | `db schema`, `db mock`, `db types` |

---

## Getting Help

```bash
nself help                             # General help
nself help <command>                   # Command help
nself <command> --help                 # Alternative
nself doctor                           # System check
nself doctor --fix                     # Auto-fix
nself version --check                  # Check updates
```

**Documentation:**
- GitHub: https://github.com/acamarata/nself
- Wiki: https://github.com/acamarata/nself/wiki
- Complete Reference: [docs/commands/COMMANDS.md](../commands/COMMANDS.md)

---

*Print this page for your desk!*

*Last Updated: January 30, 2026 | Version: 0.9.5*
*nself - Self-Hosted Infrastructure Manager*
