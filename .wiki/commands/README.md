# nself Commands

Complete reference for all nself CLI commands.

---

## Documentation

- **[Complete Commands Reference](COMMANDS.md)** - Comprehensive guide to all 31 top-level commands and their subcommands
- **[SPORT Command Matrix](SPORT-COMMAND-MATRIX.md)** - Full runtime command + wrapper coverage
- **[Quick Reference](../reference/COMMAND-REFERENCE.md)** - Printable cheat sheet
- **[Top-Level Commands Landing](../COMMANDS.md)** - SPORT landing page from wiki root

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
nself auth oauth install         # Install OAuth service
nself auth oauth enable          # Enable providers
nself auth oauth config          # Configure credentials
nself auth oauth test            # Test provider
```

> **v0.9.6 Update:** OAuth commands moved to `nself auth oauth`. Old syntax `nself oauth` still works but is deprecated.

**Providers:** Google, GitHub, Slack, Microsoft

**See:** [OAUTH.md](OAUTH.md), [OAuth Commands](COMMANDS.md#oauth-commands)

---

### 5. Storage (1 command, 7 subcommands) - v0.9.0

File storage and upload pipeline:

```bash
nself service storage init          # Initialize storage
nself service storage upload        # Upload files
nself service storage list          # List files
nself service storage config        # Configure pipeline
```

> **v0.9.6 Update:** Storage commands moved to `nself service storage`. Old syntax `nself storage` still works but is deprecated.

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
nself infra provider list         # List 26+ providers
nself infra provider server       # Server management
nself infra provider cost         # Cost estimation
```

> **v0.9.6 Update:** Infrastructure commands moved to `nself infra provider`. Old syntax `nself provider` still works but is deprecated.

**Providers:** AWS, GCP, Azure, DigitalOcean, Linode, Vultr, Hetzner, and 19+ more

**See:** [PROVIDER.md](PROVIDER.md), [Cloud Infrastructure](COMMANDS.md#cloud-infrastructure)

---

### 9. Kubernetes & Helm (2 commands) - v0.4.7

Kubernetes and Helm chart management:

```bash
nself infra k8s convert           # Generate K8s manifests
nself infra k8s deploy            # Deploy to cluster
nself infra helm install          # Install Helm chart
nself infra helm upgrade          # Upgrade release
```

> **v0.9.6 Update:** K8s/Helm commands moved to `nself infra k8s` and `nself infra helm`. Old syntax still works but is deprecated.

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
nself auth security         # Security scanning
nself auth                  # Authentication
nself auth mfa              # Multi-factor auth
nself auth roles            # Role management
nself auth devices          # Device management
nself config secrets        # Secrets management
nself config vault          # Vault integration
nself auth ssl              # SSL certificates
nself auth ssl trust        # Trust local certs
```

> **v0.9.6 Update:** Security commands consolidated under `nself auth` and `nself config`. Old syntax still works but is deprecated.

**See:** [Security Commands](COMMANDS.md#security-commands)

---

### 12. Performance & Optimization (4 commands) - v0.4.6

Performance profiling, benchmarking, and scaling:

```bash
nself perf                  # Performance profiling
nself perf bench            # Benchmarking
nself perf scale            # Service scaling
nself perf migrate          # Cross-env migration
```

> **v0.9.6 Update:** Performance commands consolidated under `nself perf`. Old top-level commands still work but are deprecated.

**See:** [PERF.md](PERF.md), [BENCH.md](BENCH.md), [SCALE.md](SCALE.md), [Performance](COMMANDS.md#performance--optimization)

---

### 13. Developer Tools (6 commands)

Tools for developers:

```bash
nself dev                   # Developer tools (v0.8.0)
nself dev frontend          # Frontend management
nself dev ci                # CI/CD generation
nself completion            # Shell completions
nself dev docs              # Documentation
```

> **v0.9.6 Update:** Developer commands consolidated under `nself dev`. Old syntax still works but is deprecated.

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
nself config env            # Environment mgmt
nself deploy sync           # Data synchronization
nself config validate       # Validate config
```

> **v0.9.6 Update:** Configuration commands consolidated. `env` → `config env`, `sync` → `deploy sync`, `validate` → `config validate`.

**See:** [CONFIG.md](CONFIG.md), [ENV.md](ENV.md), [Configuration](COMMANDS.md#configuration)

---

### 16. Utilities (5 commands)

Essential utilities:

```bash
nself help                  # Show help
nself version               # Version info
nself update                # Update nself
nself deploy upgrade        # Zero-downtime upgrades
nself admin                 # Admin UI
```

> **v0.9.6 Update:** Upgrade command moved to `nself deploy upgrade`. Old syntax still works but is deprecated.

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
- [checklist](checklist.md) - Production readiness checks
- [ci](CI.md) - CI/CD generation
- [clean](CLEAN.md) - Clean resources
- [cloud](cloud.md) - Cloud wrapper (deprecated)
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
- [org](org.md) - Organization wrapper (deprecated)
- [perf](PERF.md) - Performance profiling
- [plugin](PLUGIN.md) - Plugin system
- [prod](PROD.md) - Production deploy
- [provider](PROVIDER.md) - Cloud providers
- [providers](PROVIDERS.md) - Provider config
- [provision](PROVISION.md) - Provision server
- [realtime](REALTIME.md) - Real-time features
- [rate-limit](rate-limit.md) - Rate-limit wrapper (deprecated)
- [redis](redis.md) - Redis wrapper (deprecated)
- [reset](RESET.md) - Reset project
- [restart](RESTART.md) - Restart services
- [restore](RESTORE.md) - Restore database
- [rollback](ROLLBACK.md) - Rollback deploy
- [roles](roles.md) - Roles wrapper (deprecated)
- [scale](SCALE.md) - Service scaling
- [search](SEARCH.md) - Search service
- [secrets](secrets.md) - Secrets wrapper (deprecated)
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
- [vault](vault.md) - Vault wrapper (deprecated)
- [version](VERSION.md) - Version info
- [webhooks](webhooks.md) - Webhooks wrapper (deprecated)
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

*Last Updated: January 30, 2026 | Version: 0.9.6*
