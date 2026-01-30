# nself Commands

Complete reference for all nself CLI commands.

---

## Documentation

- **[Complete Commands Reference](COMMANDS.md)** - Comprehensive guide to all 80+ commands
- **[Quick Reference](../quick-reference/COMMAND-REFERENCE.md)** - Printable cheat sheet

---

## Command Categories

nself organizes commands into logical categories for easy discovery:

### 1. Core Lifecycle (7 commands)

Essential daily commands:

```bash
nself init          # Initialize project
nself build         # Generate configs
nself start         # Start services
nself stop          # Stop services
nself restart       # Restart services
nself reset         # Reset to clean state
nself clean         # Clean Docker resources
```

**See:** [Core Commands](COMMANDS.md#core-commands)

---

### 2. Database (1 command, 11 subcommands)

Comprehensive database management:

```bash
nself db migrate    # Migration management
nself db schema     # Schema operations
nself db seed       # Seed data
nself db mock       # Generate mock data
nself db backup     # Backup database
nself db restore    # Restore database
nself db shell      # Interactive psql
nself db query      # Execute SQL
nself db types      # Generate types
nself db inspect    # Database inspection
nself db data       # Data operations
```

**See:** [DB.md](DB.md), [Database Commands](COMMANDS.md#database-commands)

---

### 3. Multi-Tenant (1 command, 32+ subcommands) - v0.9.0

Multi-tenancy with billing, branding, and domains:

```bash
nself tenant init           # Initialize multi-tenancy
nself tenant create         # Create tenant
nself tenant billing        # Billing management
nself tenant branding       # Brand customization
nself tenant domains        # Custom domains & SSL
nself tenant email          # Email templates
nself tenant themes         # Theme management
```

**See:** [TENANT.md](TENANT.md), [BILLING.md](BILLING.md), [Multi-Tenant Commands](COMMANDS.md#multi-tenant-commands)

---

### 4. OAuth (1 command, 6 subcommands) - v0.9.0

OAuth provider management:

```bash
nself oauth install         # Install OAuth service
nself oauth enable          # Enable providers
nself oauth config          # Configure credentials
nself oauth test            # Test provider
```

**Providers:** Google, GitHub, Slack, Microsoft

**See:** [OAUTH.md](OAUTH.md), [OAuth Commands](COMMANDS.md#oauth-commands)

---

### 5. Storage (1 command, 7 subcommands) - v0.9.0

File storage and upload pipeline:

```bash
nself storage init          # Initialize storage
nself storage upload        # Upload files
nself storage list          # List files
nself storage config        # Configure pipeline
```

**Features:** Multipart upload, thumbnails, virus scanning, compression

**See:** [storage.md](storage.md), [Storage Commands](COMMANDS.md#storage-commands)

---

### 6. Service Management (1 command, 15+ subcommands)

Manage optional services:

```bash
nself service list          # List services
nself service enable        # Enable service
nself service admin         # Admin UI
nself service email         # Email service
nself service search        # Search service
nself service functions     # Serverless functions
nself service mlflow        # ML tracking
nself service cache         # Redis cache
```

**See:** [SERVICE.md](SERVICE.md), [Service Management](COMMANDS.md#service-management)

---

### 7. Deployment (1 command, 12 subcommands)

Advanced deployment strategies:

```bash
nself deploy staging        # Deploy to staging
nself deploy production     # Deploy to production
nself deploy preview        # Preview environments
nself deploy canary         # Canary deployment
nself deploy blue-green     # Zero-downtime deploy
nself deploy rollback       # Rollback deployment
```

**See:** [DEPLOY.md](DEPLOY.md), [Deployment Commands](COMMANDS.md#deployment-commands)

---

### 8. Cloud Infrastructure (1 command, 9+ subcommands) - v0.4.7

Manage cloud providers and servers:

```bash
nself provider list         # List 26+ providers
nself provider server       # Server management
nself provider cost         # Cost estimation
```

**Providers:** AWS, GCP, Azure, DigitalOcean, Linode, Vultr, Hetzner, and 19+ more

**See:** [PROVIDER.md](PROVIDER.md), [Cloud Infrastructure](COMMANDS.md#cloud-infrastructure)

---

### 9. Kubernetes & Helm (2 commands) - v0.4.7

Kubernetes and Helm chart management:

```bash
nself k8s convert           # Generate K8s manifests
nself k8s deploy            # Deploy to cluster
nself helm install          # Install Helm chart
nself helm upgrade          # Upgrade release
```

**See:** [K8S.md](K8S.md), [HELM.md](HELM.md), [Kubernetes & Helm](COMMANDS.md#kubernetes--helm)

---

### 10. Observability & Monitoring (10 commands)

Service monitoring, logging, and diagnostics:

```bash
nself status                # Service health
nself logs                  # View logs
nself exec                  # Execute in container
nself urls                  # Service URLs
nself doctor                # Diagnostics
nself health                # Health checks
nself monitor               # Dashboard access
nself metrics               # Monitoring profiles
nself history               # Audit trail
```

**See:** [STATUS.md](STATUS.md), [LOGS.md](LOGS.md), [Observability](COMMANDS.md#observability--monitoring)

---

### 11. Security (10 commands)

Security scanning, secrets, and access control:

```bash
nself security              # Security scanning
nself auth                  # Authentication
nself mfa                   # Multi-factor auth
nself roles                 # Role management
nself devices               # Device management
nself secrets               # Secrets management
nself vault                 # Vault integration
nself ssl                   # SSL certificates
nself trust                 # Trust local certs
```

**See:** [Security Commands](COMMANDS.md#security-commands)

---

### 12. Performance & Optimization (4 commands) - v0.4.6

Performance profiling, benchmarking, and scaling:

```bash
nself perf                  # Performance profiling
nself bench                 # Benchmarking
nself scale                 # Service scaling
nself migrate               # Cross-env migration
```

**See:** [PERF.md](PERF.md), [BENCH.md](BENCH.md), [SCALE.md](SCALE.md), [Performance](COMMANDS.md#performance--optimization)

---

### 13. Developer Tools (6 commands)

Tools for developers:

```bash
nself dev                   # Developer tools (v0.8.0)
nself frontend              # Frontend management
nself ci                    # CI/CD generation
nself completion            # Shell completions
nself docs                  # Documentation
```

**See:** [DEV.md](DEV.md), [Developer Tools](COMMANDS.md#developer-tools)

---

### 14. Plugin System (1 command) - v0.4.8

Third-party integrations:

```bash
nself plugin list           # List available plugins
nself plugin install        # Install plugin
nself plugin stripe         # Stripe integration
nself plugin github         # GitHub integration
nself plugin shopify        # Shopify integration
```

**See:** [PLUGIN.md](PLUGIN.md), [Plugin System](COMMANDS.md#plugin-system)

---

### 15. Configuration (4 commands)

Configuration and environment management:

```bash
nself config                # Configuration mgmt
nself env                   # Environment mgmt
nself sync                  # Data synchronization
nself validate              # Validate config
```

**See:** [CONFIG.md](CONFIG.md), [ENV.md](ENV.md), [Configuration](COMMANDS.md#configuration)

---

### 16. Utilities (5 commands)

Essential utilities:

```bash
nself help                  # Show help
nself version               # Version info
nself update                # Update nself
nself upgrade               # Zero-downtime upgrades
nself admin                 # Admin UI
```

**See:** [Utilities](COMMANDS.md#utilities)

---

## Quick Start

### New Project

```bash
nself init --demo           # Interactive wizard with demo config
nself build                 # Generate configs
nself start                 # Start all services
nself urls                  # Show service URLs
```

### Daily Development

```bash
nself start                 # Start services
nself status                # Check health
nself logs -f               # Follow logs
nself db shell              # Database shell
nself stop                  # Stop when done
```

### Common Tasks

```bash
# Database operations
nself db migrate up
nself db seed
nself db backup

# Service management
nself service enable redis
nself service email test
nself service admin open

# Deployment
nself deploy staging
nself deploy production --blue-green

# Monitoring
nself monitor
nself perf dashboard
```

---

## Getting Help

```bash
# General help
nself help

# Command-specific help
nself help <command>
nself <command> --help

# Examples
nself help db
nself deploy --help

# System diagnostics
nself doctor
nself doctor --fix
```

---

## Command Index

### A-D
- [admin](ADMIN.md) - Admin UI
- [admin-dev](ADMIN-DEV.md) - Admin dev mode
- [audit](AUDIT.md) - Audit logging
- [auth](AUTH.md) - Authentication
- [backup](BACKUP.md) - Database backup (legacy)
- [bench](BENCH.md) - Benchmarking
- [billing](BILLING.md) - Billing management
- [build](BUILD.md) - Build configs
- [ci](CI.md) - CI/CD generation
- [clean](CLEAN.md) - Clean resources
- [completion](COMPLETION.md) - Shell completion
- [config](CONFIG.md) - Configuration
- [db](DB.md) - Database management
- [deploy](DEPLOY.md) - Deployment
- [dev](DEV.md) - Developer tools
- [devices](DEVICES.md) - Device management
- [doctor](DOCTOR.md) - Diagnostics
- [docs](docs.md) - Documentation

### E-M
- [email](EMAIL.md) - Email service
- [env](ENV.md) - Environment mgmt
- [exec](EXEC.md) - Execute in container
- [frontend](FRONTEND.md) - Frontend mgmt
- [functions](FUNCTIONS.md) - Serverless functions
- [health](HEALTH.md) - Health checks
- [helm](HELM.md) - Helm charts
- [help](HELP.md) - Show help
- [history](HISTORY.md) - Audit trail
- [init](INIT.md) - Initialize project
- [k8s](K8S.md) - Kubernetes
- [logs](LOGS.md) - View logs
- [metrics](METRICS.md) - Monitoring profiles
- [mfa](MFA.md) - Multi-factor auth
- [migrate](MIGRATE.md) - Migration tool
- [mlflow](MLFLOW.md) - ML tracking
- [monitor](MONITOR.md) - Dashboard access

### O-Z
- [oauth](OAUTH.md) - OAuth providers
- [perf](PERF.md) - Performance profiling
- [plugin](PLUGIN.md) - Plugin system
- [prod](PROD.md) - Production deploy
- [provider](PROVIDER.md) - Cloud providers
- [providers](PROVIDERS.md) - Provider config
- [provision](PROVISION.md) - Provision server
- [realtime](REALTIME.md) - Real-time features
- [reset](RESET.md) - Reset project
- [restart](RESTART.md) - Restart services
- [restore](RESTORE.md) - Restore database
- [rollback](ROLLBACK.md) - Rollback deploy
- [scale](SCALE.md) - Service scaling
- [search](SEARCH.md) - Search service
- [security](security.md) - Security scanning
- [server](server.md) - Server management
- [servers](SERVERS.md) - Server list
- [service](SERVICE.md) - Service mgmt
- [ssl](SSL.md) - SSL certificates
- [staging](STAGING.md) - Staging deploy
- [start](START.md) - Start services
- [status](STATUS.md) - Service health
- [stop](STOP.md) - Stop services
- [storage](storage.md) - File storage
- [sync](SYNC.md) - Data sync
- [tenant](TENANT.md) - Multi-tenancy
- [trust](TRUST.md) - Trust certs
- [update](UPDATE.md) - Update nself
- [upgrade](upgrade.md) - Zero-downtime upgrades
- [urls](URLS.md) - Service URLs
- [validate](validate.md) - Validate config
- [version](VERSION.md) - Version info
- [whitelabel](WHITELABEL.md) - White-label (legacy)

---

## Version History

| Version | Commands Added |
|---------|----------------|
| **0.9.5** | Feature parity & security hardening |
| **0.9.0** | `tenant`, `oauth`, `storage` (50+ new subcommands) |
| **0.8.0** | `dev`, `realtime`, `org`, `security`, `upgrade` |
| **0.4.8** | `plugin` with Stripe/GitHub/Shopify integrations |
| **0.4.7** | `provider`, `service`, `k8s`, `helm`, advanced deploy |
| **0.4.6** | `perf`, `bench`, `scale`, `migrate`, `health` |
| **0.4.5** | `providers`, `provision`, `sync`, `ci`, `completion` |
| **0.4.4** | `db schema`, `db mock`, `db types` |
| **0.4.3** | `env`, `deploy`, `staging`, `prod` |

---

**Total: 80+ top-level commands with 200+ subcommands**

**[View Complete Commands Reference →](COMMANDS.md)**

---

*Last Updated: January 30, 2026 | Version: 0.9.5*
