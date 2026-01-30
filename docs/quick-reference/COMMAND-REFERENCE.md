# nself CLI Command Reference

> Quick reference guide for all nself commands - Optimized for printing

**Version:** v0.8.0+ (Current)
**Total Commands:** 70+ commands across 14 categories

---

## Core Commands (8)

### init
Initialize a new nself project with interactive wizard
```bash
nself init                 # Interactive wizard
nself init --demo          # Demo configuration
nself init --simple        # Simple wizard (fewer options)
```

### build
Generate configuration files and build Docker images
```bash
nself build                # Build all services
nself build --clean        # Clean build (remove existing)
nself build --no-cache     # Build without cache
```

### start
Start all configured services
```bash
nself start                # Smart start (default)
nself start --fresh        # Force recreate containers
nself start --verbose      # Detailed output
```
**Key Flags:** `--skip-health-checks`, `--timeout <seconds>`, `--clean-start`

### stop
Stop all running services
```bash
nself stop                 # Stop all services
nself stop <service>       # Stop specific service
```

### restart
Restart services
```bash
nself restart              # Restart all services
nself restart <service>    # Restart specific service
```

### reset
Reset project to clean state
```bash
nself reset                # Interactive confirmation
nself reset --force        # Skip confirmation
```

### clean
Clean up Docker resources
```bash
nself clean                # Clean unused resources
nself clean --all          # Remove everything
```

### version
Show version information
```bash
nself version              # Show version
nself version --check      # Check for updates
```

**Related:** `update`, `doctor`, `help`

---

## Status & Monitoring (6)

### status
Show service status
```bash
nself status               # All services
nself status <service>     # Specific service
nself status --json        # JSON output
nself status --all-envs    # All environments
```

### logs
View service logs
```bash
nself logs                 # All services
nself logs <service>       # Specific service
nself logs -f              # Follow logs
nself logs --tail 100      # Last 100 lines
```

### exec
Execute commands in containers
```bash
nself exec <service> <command>
nself exec postgres psql -U postgres
nself exec hasura hasura-cli console
```

### urls
Show service URLs
```bash
nself urls                 # All URLs
nself urls --env staging   # Environment-specific
nself urls --diff          # Compare environments
```

### doctor
Run system diagnostics
```bash
nself doctor               # Full diagnostic
nself doctor --fix         # Auto-fix issues
nself doctor --check deps  # Check dependencies
```

### help
Show help information
```bash
nself help                 # General help
nself help <command>       # Command-specific help
```

**Related:** `monitor`, `metrics`

---

## Management (4)

### update
Update nself to latest version
```bash
nself update               # Update to latest
nself update --check       # Check for updates only
nself update --version X.Y.Z  # Specific version
```

### ssl
Manage SSL certificates
```bash
nself ssl generate         # Generate self-signed cert
nself ssl renew            # Renew Let's Encrypt cert
nself ssl info             # Show certificate info
```

### trust
Trust local SSL certificates
```bash
nself trust                # Trust local certificates
nself trust --system       # Add to system keychain
```

### admin
Admin UI management
```bash
nself admin                # Open admin UI
nself admin start          # Start admin UI
nself admin stop           # Stop admin UI
```

**Related:** `config`, `validate`

---

## Multi-Tenancy (2) - NEW in v0.8.0

### tenant
Multi-tenant management
```bash
nself tenant create <name>      # Create tenant
nself tenant list                # List tenants
nself tenant suspend <id>        # Suspend tenant
nself tenant activate <id>       # Activate tenant
nself tenant member add <tenant> <user> [role]
nself tenant domain add <tenant> <domain>
nself tenant setting set <tenant> <key> <value>
```

### org
Organization & team management
```bash
nself org create <name>          # Create organization
nself org teams <org>            # List teams
nself org members <org>          # List members
nself org member add <org> <user> [role]
nself org team create <org> <name>
nself org role create <org> <name>
nself org permission grant <role> <perm>
```

**Related:** `roles`, `auth`

---

## Billing (1) - NEW in v0.9.0

### billing
Billing & subscription management
```bash
nself billing usage              # Current usage
nself billing invoice list       # List invoices
nself billing subscription show  # Current plan
nself billing subscription upgrade <plan>
nself billing payment list       # Payment methods
nself billing quota              # Quota limits
nself billing export --format csv
```

**Related:** `plugin stripe`

---

## White-Label (1) - NEW in v0.9.0

### whitelabel
Custom branding & themes
```bash
nself whitelabel branding create <name>
nself whitelabel branding set-colors --primary #hex
nself whitelabel branding upload-logo <path>
nself whitelabel domain add <domain>
nself whitelabel email edit <template>
nself whitelabel theme create <name>
nself whitelabel theme activate <name>
```

**Related:** `tenant`, `org`

---

## Real-Time (1) - NEW in v0.8.0

### realtime
Real-time communication management
```bash
nself realtime init              # Initialize WebSocket
nself realtime status            # Server status
nself realtime connections       # Active connections
nself realtime channel create <name>
nself realtime channel list      # List channels
nself realtime stats             # Statistics
```

**Related:** `monitor`, `logs`

---

## Plugin System (1) - v0.4.8

### plugin
Plugin management & execution
```bash
nself plugin list                # Available plugins
nself plugin install <name>      # Install plugin
nself plugin status              # Installed plugins
nself plugin update [name]       # Update plugins
nself plugin config <name>       # Configure plugin
nself plugin init                # Create plugin template
```

**First Plugin:**
```bash
nself stripe init                # Setup Stripe integration
nself stripe sync                # Sync Stripe data
nself stripe webhook status      # Webhook status
nself stripe check               # Verify DB matches Stripe
```

**Related:** `billing`

---

## Security (1) - NEW in v0.8.0

### security
Security scanning & management
```bash
nself security scan              # Full security scan
nself security scan passwords    # Check password strength
nself security scan mfa          # Check MFA coverage
nself security scan suspicious   # Detect suspicious activity
nself security devices           # List all devices
nself security incidents         # Security incidents
nself security events <user>     # User security events
nself security webauthn          # Manage WebAuthn keys
```

**Related:** `mfa`, `auth`, `audit`, `devices`

---

## Performance (4)

### perf
Performance profiling & analysis
```bash
nself perf profile               # Full system profile
nself perf analyze               # Analyze performance
nself perf slow-queries          # Slow query analysis
nself perf report                # Generate report
nself perf dashboard             # Real-time dashboard
nself perf suggest               # Optimization suggestions
```

### bench
Benchmarking & load testing
```bash
nself bench api                  # Benchmark API
nself bench db                   # Database performance
nself bench load --users 1000    # Load test
nself bench compare              # Compare to baseline
nself bench baseline save        # Save baseline
```

### scale
Service scaling & autoscaling
```bash
nself scale up <service>         # Vertical scaling
nself scale out <service>        # Horizontal scaling
nself scale auto <service> --min 2 --max 10
nself scale pooler enable        # Enable PgBouncer
nself scale redis cluster        # Redis cluster
nself scale status               # Scaling status
```

### migrate
Cross-environment migration
```bash
nself migrate staging prod       # Migrate environments
nself migrate --dry-run          # Preview migration
nself migrate sync <from> <to>   # Keep in sync
nself migrate diff staging prod  # Show differences
nself migrate rollback           # Rollback migration
```

**Also includes:** `nself migrate from firebase|supabase` (v0.8.0)

**Related:** `deploy`, `env`, `upgrade`

---

## Database (1 with 10 subcommands)

### db
Unified database management
```bash
# Migrations
nself db migrate                 # Run migrations
nself db migrate status          # Migration status
nself db migrate down            # Rollback one migration
nself db migrate create <name>   # Create migration
nself db migrate fresh           # Drop and recreate

# Seeding & Mock Data
nself db seed                    # Seed database
nself db mock <entity> <count>   # Generate mock data

# Backup & Restore
nself db backup                  # Create backup
nself db restore <file>          # Restore backup

# Schema & Types
nself db schema                  # Show schema
nself db types typescript        # Generate types

# Operations
nself db shell                   # PostgreSQL shell
nself db inspect                 # Database analysis
nself db data export             # Export data
```

**Related:** `backup`, `restore`

---

## Deployment (4)

### env
Environment management
```bash
nself env list                   # List environments
nself env switch <env>           # Switch environment
nself env create <name>          # Create environment
nself env diff staging prod      # Compare environments
nself env validate               # Validate config
nself env access                 # Show access level
```

### deploy
Deploy with advanced strategies
```bash
nself deploy staging             # Deploy to staging
nself deploy production          # Deploy to production
nself deploy rollback            # Rollback deployment
nself deploy preview             # Preview environment
nself deploy canary --percentage 20
nself deploy blue-green          # Zero-downtime
nself deploy check               # Pre-deployment validation
```

### prod
Production configuration shortcut
```bash
nself prod                       # Deploy to production
nself prod status                # Production status
```

### staging
Staging environment shortcut
```bash
nself staging                    # Deploy to staging
nself staging status             # Staging status
```

**Related:** `sync`, `migrate`, `upgrade`

---

## Cloud Providers (2)

### providers
Cloud provider management
```bash
nself providers list             # List providers
nself providers init <provider>  # Configure provider
nself providers validate         # Validate credentials
nself providers info <provider>  # Provider info
```

**Supported:** AWS, GCP, Azure, DigitalOcean, Linode, Vultr, Hetzner, OVH, Scaleway, UpCloud, and 15+ more

### provision
One-command infrastructure provisioning
```bash
nself provision <provider>       # Provision infrastructure
nself provision <provider> --estimate  # Cost estimate
nself provision <provider> --size medium
nself provision compare          # Compare providers
```

**Related:** `cloud`, `servers`

---

## Kubernetes (2)

### k8s
Kubernetes operations
```bash
nself k8s generate               # Generate K8s manifests
nself k8s apply                  # Apply to cluster
nself k8s status                 # Cluster status
nself k8s logs <service>         # Pod logs
nself k8s exec <service>         # Exec into pod
nself k8s scale <service> 3      # Scale replicas
nself k8s rollout <service>      # Rolling update
nself k8s rollback <service>     # Rollback deployment
```

### helm
Helm chart management
```bash
nself helm init                  # Initialize chart
nself helm package               # Package chart
nself helm install               # Install to cluster
nself helm upgrade               # Upgrade release
nself helm list                  # List releases
nself helm rollback              # Rollback release
nself helm repo add <url>        # Add chart repo
```

**Related:** `k8s`, `deploy`

---

## Developer Tools (1) - NEW in v0.8.0

### dev
Developer experience tools
```bash
# SDK Generation
nself dev sdk generate typescript
nself dev sdk generate python ./my-sdk

# Documentation
nself dev docs generate          # API documentation
nself dev docs openapi           # OpenAPI spec

# Testing
nself dev test init              # Initialize test env
nself dev test fixtures users 50 # Generate fixtures
nself dev test factory users     # Mock data factory
nself dev test snapshot create baseline
nself dev test run               # Run integration tests

# Mock Data
nself dev mock users 100         # Generate mock users
```

**Related:** `db mock`, `db seed`

---

## Services (6)

### email
Email service management
```bash
nself email init                 # Configure email provider
nself email test                 # Send test email
nself email inbox                # View inbox (MailPit)
nself email config               # Show configuration
```

### search
Search service management
```bash
nself search init                # Initialize search
nself search index <entity>      # Index data
nself search query <term>        # Search query
nself search stats               # Search statistics
```

**Engines:** PostgreSQL, MeiliSearch, Typesense, Sonic, ElasticSearch, Algolia

### functions
Serverless functions runtime
```bash
nself functions deploy <path>    # Deploy function
nself functions invoke <name>    # Invoke function
nself functions logs <name>      # View logs
nself functions list             # List functions
```

### mlflow
ML experiment tracking
```bash
nself mlflow ui                  # Open MLflow UI
nself mlflow experiments         # List experiments
nself mlflow runs <experiment>   # List runs
nself mlflow artifacts <run>     # Show artifacts
```

### metrics
Monitoring profiles
```bash
nself metrics profile minimal    # Minimal monitoring
nself metrics profile standard   # Standard monitoring
nself metrics profile full       # Full monitoring
nself metrics profile auto       # Auto-configure
```

### monitor
Dashboard access
```bash
nself monitor                    # Open Grafana
nself monitor prometheus         # Open Prometheus
nself monitor alertmanager       # Open Alertmanager
```

**Related:** `status`, `logs`, `perf`

---

## Additional Commands

### storage
File storage and upload management
```bash
nself storage upload <file>       # Upload a file
nself storage list [prefix]       # List files
nself storage delete <path>       # Delete file
nself storage config              # Show configuration
nself storage status              # Pipeline status
nself storage test                # Test uploads
nself storage init                # Initialize storage
nself storage graphql-setup       # Generate GraphQL integration
```
**Key Features:** Multipart upload, thumbnails, virus scanning, compression
**See Also:** [storage command](commands/storage.md), [File Upload Guide](guides/file-upload-pipeline.md)

### sync
Data synchronization
```bash
nself sync db staging prod       # Sync database
nself sync files staging prod    # Sync files
nself sync config staging prod   # Sync configuration
nself sync pull staging          # Pull from staging
nself sync auto                  # Continuous sync
```

### ci
CI/CD workflow generation
```bash
nself ci init                    # Generate CI workflow
nself ci init github             # GitHub Actions
nself ci init gitlab             # GitLab CI
```

### completion
Shell completion
```bash
nself completion bash            # Bash completion
nself completion zsh             # Zsh completion
nself completion fish            # Fish completion
```

### health
Health check management
```bash
nself health                     # Check all services
nself health <service>           # Check specific service
nself health --watch             # Watch health status
```

### frontend
Frontend application management
```bash
nself frontend list              # List apps
nself frontend add <name>        # Add frontend app
nself frontend remove <name>     # Remove app
```

### history
Deployment audit trail
```bash
nself history                    # Deployment history
nself history --limit 20         # Last 20 deployments
nself history --filter success   # Filter by status
```

### config
Configuration management
```bash
nself config list                # List all config
nself config get <key>           # Get value
nself config set <key> <value>   # Set value
```

### servers
Server infrastructure
```bash
nself servers list               # List servers
nself servers add <name>         # Add server
nself servers ssh <name>         # SSH to server
nself servers status <name>      # Server status
```

### upgrade
Zero-downtime upgrades (v0.8.0)
```bash
nself upgrade perform            # Blue-green deployment
nself upgrade rolling            # Rolling update
nself upgrade rollback           # Instant rollback
nself upgrade status             # Upgrade status
```

### auth
Authentication management
```bash
nself auth users                 # List users
nself auth roles                 # List roles
nself auth providers             # Auth providers
```

### mfa
Multi-factor authentication
```bash
nself mfa enable                 # Enable MFA
nself mfa disable                # Disable MFA
nself mfa status                 # MFA status
```

### roles
Role management
```bash
nself roles list                 # List roles
nself roles create <name>        # Create role
nself roles assign <user> <role> # Assign role
```

### secrets
Secrets management
```bash
nself secrets list               # List secrets
nself secrets add <key> <value>  # Add secret
nself secrets rotate             # Rotate secrets
```

### vault
Vault integration
```bash
nself vault init                 # Initialize Vault
nself vault status               # Vault status
nself vault unseal               # Unseal Vault
```

### rate-limit
Rate limiting
```bash
nself rate-limit config          # Configure limits
nself rate-limit status          # Show limits
```

### webhooks
Webhook management
```bash
nself webhooks list              # List webhooks
nself webhooks test <url>        # Test webhook
```

### devices
Device management
```bash
nself devices list               # List devices
nself devices approve <id>       # Approve device
nself devices revoke <id>        # Revoke device
```

### audit
Audit logging
```bash
nself audit logs                 # Audit logs
nself audit events <user>        # User events
```

### redis
Redis management
```bash
nself redis cli                  # Redis CLI
nself redis stats                # Redis statistics
nself redis flush                # Flush cache
```

---

## Command Flags Summary

### Global Flags (Available on most commands)
- `--help, -h` - Show help message
- `--verbose, -v` - Verbose output
- `--debug` - Debug mode
- `--json` - JSON output
- `--format <format>` - Output format (table, json, csv)
- `--env <env>` - Target environment (local, staging, prod)

### Common Patterns

**Service-specific:**
```bash
nself <command> <service>        # Target specific service
nself logs postgres
nself restart hasura
```

**Environment-specific:**
```bash
nself <command> --env <env>      # Target environment
nself status --env staging
nself deploy --env prod
```

**Output formats:**
```bash
nself <command> --json           # JSON output
nself <command> --format csv     # CSV output
nself <command> --format table   # Table output
```

---

## Quick Start Commands

**New Project:**
```bash
nself init --demo
nself build
nself start
```

**Check Status:**
```bash
nself status
nself urls
nself doctor
```

**View Logs:**
```bash
nself logs -f
nself logs postgres
```

**Database Operations:**
```bash
nself db migrate
nself db seed
nself db backup
```

**Deployment:**
```bash
nself env switch staging
nself deploy staging
nself deploy production --blue-green
```

**Monitoring:**
```bash
nself monitor
nself perf dashboard
nself security scan
```

---

## Environment Variables

Key configuration via `.env`:

```bash
# Core
PROJECT_NAME=myapp
ENV=dev|staging|prod
BASE_DOMAIN=localhost

# Optional Services
REDIS_ENABLED=true
MINIO_ENABLED=true
NSELF_ADMIN_ENABLED=true

# Monitoring
MONITORING_ENABLED=true

# Multi-Tenancy (v0.8.0)
MULTI_TENANCY_ENABLED=true
REALTIME_ENABLED=true

# Custom Services
CS_1=api:express-js:8001
CS_2=worker:bullmq-js:8002
```

---

## Help Resources

**Command Help:**
```bash
nself help                       # General help
nself help <command>             # Command-specific help
nself <command> --help           # Alternative syntax
```

**Diagnostics:**
```bash
nself doctor                     # System check
nself doctor --fix               # Auto-fix issues
nself version --check            # Check for updates
```

**Documentation:**
- GitHub: https://github.com/acamarata/nself
- Wiki: https://github.com/acamarata/nself/wiki
- Issues: https://github.com/acamarata/nself/issues

---

## Version Information

- **Current Version:** v0.8.0 (Multi-Tenancy & Enterprise)
- **Next Version:** v0.9.0 (Billing & White-Label)
- **LTS Target:** v0.5.0 (Production Ready)

**Update:**
```bash
nself update                     # Update to latest
nself version                    # Show current version
```

---

*Last Updated: January 30, 2026*
*nself v0.8.0 - Self-Hosted Infrastructure Manager*
