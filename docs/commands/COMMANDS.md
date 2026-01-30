# nself Commands Reference

**Version 0.9.0** | Complete CLI Reference

---

## Command Tree Overview

```
nself
├── Core Lifecycle
│   ├── init                    Initialize project
│   ├── build                   Generate configs
│   ├── start                   Start services
│   ├── stop                    Stop services
│   ├── restart                 Restart services
│   ├── reset                   Reset to clean state
│   └── clean                   Clean Docker resources
│
├── Status & Monitoring
│   ├── status                  Service health
│   │   ├── --all-envs         All environments
│   │   ├── --json             JSON output
│   │   └── --watch            Continuous monitoring
│   ├── logs                    View logs
│   ├── exec                    Execute in container
│   ├── urls                    Service URLs
│   │   ├── --env              Specific environment
│   │   ├── --diff             Compare environments
│   │   └── --json             JSON output
│   ├── doctor                  Diagnostics
│   │   └── --fix              Auto-repair
│   └── health                  Health checks (v0.4.6)
│       ├── check              Run all checks
│       ├── service <name>     Check specific service
│       ├── endpoint <url>     Check custom endpoint
│       ├── watch              Continuous monitoring
│       ├── history            Check history
│       └── config             Health configuration
│
├── Database (nself db)
│   ├── migrate                 Migration management
│   │   ├── up                 Run migrations
│   │   ├── down               Rollback
│   │   ├── create <name>      Create migration
│   │   └── status             Migration status
│   ├── schema                  Schema operations
│   │   ├── scaffold <template> Create from template
│   │   ├── import <file>      Import DBML
│   │   ├── apply <file>       Full workflow
│   │   └── diagram            Export to DBML
│   ├── seed                    Seed data
│   │   ├── (default)          Run all seeds
│   │   ├── users              Seed users
│   │   └── create <name>      Create seed file
│   ├── mock                    Generate mock data
│   │   ├── (default)          Generate mocks
│   │   ├── auto               Auto from schema
│   │   └── --seed N           Reproducible
│   ├── backup                  Backup database
│   │   ├── (default)          Create backup
│   │   └── list               List backups
│   ├── restore                 Restore database
│   ├── shell                   Interactive psql
│   │   └── --readonly         Read-only mode
│   ├── query <sql>             Execute SQL
│   ├── types                   Generate types
│   │   ├── (default)          TypeScript
│   │   ├── go                 Go structs
│   │   └── python             Python classes
│   ├── inspect                 Database inspection
│   │   ├── (default)          Overview
│   │   ├── size               Table sizes
│   │   └── slow               Slow queries
│   └── data                    Data operations
│       ├── export <table>     Export table
│       └── anonymize          Anonymize PII
│
├── Multi-Tenant (nself tenant) - NEW v0.9.0
│   ├── init                    Initialize multi-tenancy system
│   ├── create <name>           Create a new tenant
│   ├── list                    List all tenants
│   ├── show <id>               Show tenant details
│   ├── suspend <id>            Suspend a tenant
│   ├── activate <id>           Activate a suspended tenant
│   ├── delete <id>             Delete a tenant
│   ├── stats                   Show tenant statistics
│   ├── member                  Member management
│   │   ├── add <tenant> <user> [role]    Add user to tenant
│   │   ├── remove <tenant> <user>        Remove user from tenant
│   │   └── list <tenant>                 List tenant members
│   ├── setting                 Settings management
│   │   ├── set <tenant> <key> <value>    Set tenant setting
│   │   ├── get <tenant> <key>            Get tenant setting
│   │   └── list <tenant>                 List tenant settings
│   ├── billing                 Billing management
│   │   ├── usage              Show usage statistics
│   │   ├── invoice            Manage invoices (list, show, download, pay)
│   │   ├── subscription       Manage subscriptions (show, upgrade, downgrade)
│   │   ├── payment            Manage payment methods (list, add, remove)
│   │   ├── quota              Check quota limits
│   │   ├── plan               Manage plans (list, show, compare, current)
│   │   ├── export             Export billing data
│   │   └── customer           Manage customer info (show, update, portal)
│   ├── branding                Brand customization
│   │   ├── create <name>      Create new brand
│   │   ├── set-colors         Set brand colors
│   │   ├── set-fonts          Set brand fonts
│   │   ├── upload-logo        Upload brand logo
│   │   ├── set-css            Set custom CSS
│   │   └── preview            Preview branding
│   ├── domains                 Custom domains & SSL
│   │   ├── add <domain>       Add custom domain
│   │   ├── verify <domain>    Verify domain ownership
│   │   ├── ssl <domain>       Provision SSL certificate
│   │   ├── health <domain>    Check domain health
│   │   └── remove <domain>    Remove custom domain
│   ├── email                   Email templates
│   │   ├── list               List all email templates
│   │   ├── edit <template>    Edit email template
│   │   ├── preview <template> Preview email template
│   │   ├── test <template>    Send test email
│   │   └── set-language       Set email language
│   └── themes                  Theme management
│       ├── create <name>      Create new theme
│       ├── edit <name>        Edit theme
│       ├── activate <name>    Activate theme
│       ├── preview <name>     Preview theme
│       ├── export <name>      Export theme
│       └── import <path>      Import theme
│
├── OAuth (nself oauth) - NEW v0.9.0
│   ├── install                 Install OAuth handlers service
│   ├── enable                  Enable OAuth providers
│   │   └── --providers=<list> Comma-separated list (google,github,slack,microsoft)
│   ├── disable                 Disable OAuth providers
│   │   └── --providers=<list> Providers to disable
│   ├── config <provider>       Configure OAuth provider credentials
│   │   ├── --client-id=<id>   OAuth client ID
│   │   ├── --client-secret    OAuth client secret
│   │   ├── --tenant-id        Tenant ID (Microsoft only)
│   │   └── --callback-url     Custom callback URL
│   ├── test <provider>         Test OAuth provider configuration
│   ├── list                    List all OAuth providers
│   └── status                  Show OAuth service status
│
├── Storage (nself storage) - NEW v0.9.0
│   ├── init                    Initialize storage system
│   ├── upload <file>           Upload a file to storage
│   │   ├── --dest <path>      Destination path in storage
│   │   ├── --thumbnails       Generate image thumbnails
│   │   ├── --virus-scan       Scan file for viruses
│   │   ├── --compression      Compress large files
│   │   └── --all-features     Enable all features
│   ├── list [prefix]           List uploaded files
│   ├── delete <path>           Delete an uploaded file
│   ├── config                  Configure upload pipeline
│   ├── status                  Show pipeline status
│   ├── test                    Test upload functionality
│   └── graphql-setup           Generate GraphQL integration package
│
├── Deployment (nself deploy) - Enhanced v0.4.7
│   ├── (environment)           Deploy to environment
│   │   ├── staging            Deploy to staging
│   │   └── production         Deploy to production
│   ├── preview                 Preview environments (NEW)
│   │   ├── (default)          Create preview
│   │   ├── list               List previews
│   │   └── destroy <id>       Destroy preview
│   ├── canary                  Canary deployment (NEW)
│   │   ├── (default)          Start canary
│   │   ├── promote            Promote to 100%
│   │   ├── rollback           Rollback canary
│   │   └── status             Canary status
│   ├── blue-green              Blue-green deploy (NEW)
│   │   ├── (default)          Deploy inactive
│   │   ├── switch             Switch traffic
│   │   ├── rollback           Rollback switch
│   │   └── status             Show active
│   ├── rollback                Rollback deployment
│   ├── check                   Pre-deploy validation
│   │   └── --fix              Auto-fix issues
│   └── status                  Deployment status
│
├── Environment (nself env)
│   ├── (default)               List environments
│   ├── create <name>           Create environment
│   ├── switch <name>           Switch environment
│   └── diff <env1> <env2>      Compare environments
│
├── Cloud Infrastructure (nself cloud) - v0.4.7
│   ├── provider                Provider management
│   │   ├── list               List 26 providers
│   │   ├── init <provider>    Configure credentials
│   │   ├── validate           Validate config
│   │   └── info <provider>    Provider details
│   ├── server                  Server management
│   │   ├── create <provider>  Provision server
│   │   ├── destroy <server>   Destroy server
│   │   ├── list               List servers
│   │   ├── status [server]    Server status
│   │   ├── ssh <server>       SSH to server
│   │   ├── add <ip>           Add existing server
│   │   └── remove <server>    Remove from registry
│   ├── cost                    Cost management
│   │   ├── estimate <prov>    Estimate costs
│   │   └── compare            Compare all providers
│   └── deploy                  Quick deployment
│       ├── quick              Provision + deploy
│       └── full               Full production setup
│
├── Service Management (nself service) - Enhanced v0.9.0
│   ├── list                    List optional services
│   ├── enable <service>        Enable service
│   ├── disable <service>       Disable service
│   ├── status [service]        Service status
│   ├── restart <service>       Restart service
│   ├── logs <service>          Service logs
│   ├── init                    Initialize service from template
│   ├── scaffold                Scaffold new service
│   ├── wizard                  Service creation wizard
│   ├── search                  Search services
│   ├── admin                   Admin UI management
│   │   ├── status             Admin UI status
│   │   ├── open               Open admin UI
│   │   ├── users              User management
│   │   ├── config             Admin configuration
│   │   └── dev                Development mode
│   ├── email                   Email service
│   │   ├── test               Send test email
│   │   ├── inbox              Open MailPit
│   │   └── config             Email config
│   ├── search                  Search service
│   │   ├── index              Reindex data
│   │   ├── query <term>       Run query
│   │   └── stats              Index stats
│   ├── functions               Serverless
│   │   ├── deploy             Deploy all
│   │   ├── invoke <fn>        Invoke function
│   │   ├── logs [fn]          View logs
│   │   └── list               List functions
│   ├── mlflow                  ML tracking
│   │   ├── ui                 Open UI
│   │   ├── experiments        List experiments
│   │   ├── runs               List runs
│   │   └── artifacts          Browse artifacts
│   ├── storage                 Object storage
│   │   ├── buckets            List buckets
│   │   ├── upload             Upload file
│   │   ├── download           Download file
│   │   └── presign            Generate URL
│   └── cache                   Redis cache
│       ├── stats              Statistics
│       ├── flush              Flush cache
│       └── keys               List keys
│
├── Kubernetes (nself k8s) - v0.4.7
│   ├── init                    Initialize K8s config
│   ├── convert                 Compose to manifests
│   │   ├── --output <dir>     Custom output
│   │   └── --namespace <ns>   Custom namespace
│   ├── apply                   Apply manifests
│   │   └── --dry-run          Preview changes
│   ├── deploy                  Full deployment
│   │   └── --env <env>        Environment
│   ├── status                  Deployment status
│   ├── logs <service>          Pod logs
│   │   └── -f                 Follow logs
│   ├── scale <svc> <n>         Scale deployment
│   ├── rollback <service>      Rollback deployment
│   ├── delete                  Delete deployment
│   ├── cluster                 Cluster management
│   │   ├── list               List clusters
│   │   ├── connect <name>     Connect to cluster
│   │   └── info               Cluster info
│   └── namespace               Namespace management
│       ├── list               List namespaces
│       ├── create <name>      Create namespace
│       ├── delete <name>      Delete namespace
│       └── switch <name>      Switch namespace
│
├── Helm Charts (nself helm) - v0.4.7
│   ├── init                    Initialize chart
│   │   └── --from-compose     From docker-compose
│   ├── generate                Generate/update chart
│   ├── install                 Install to cluster
│   │   └── --env <env>        With env values
│   ├── upgrade                 Upgrade release
│   ├── rollback                Rollback release
│   ├── uninstall               Remove release
│   ├── list                    List releases
│   ├── status                  Release status
│   ├── values                  Show/edit values
│   ├── template                Render locally
│   ├── package                 Package chart
│   └── repo                    Repository mgmt
│       ├── add <name> <url>   Add repository
│       ├── remove <name>      Remove repository
│       ├── update             Update repos
│       └── list               List repos
│
├── Sync (nself sync) - Enhanced v0.4.7
│   ├── db <env>                Sync database
│   ├── files <env>             Sync files
│   ├── config <env>            Sync configuration
│   ├── full <env>              Full sync
│   ├── auto                    Auto-sync (NEW)
│   │   ├── --setup            Configure service
│   │   └── --stop             Stop auto-sync
│   ├── watch                   Watch mode (NEW)
│   │   ├── --path <dir>       Watch specific path
│   │   └── --interval <s>     Polling interval
│   ├── status                  Sync status
│   └── history                 Sync history
│
├── Performance (v0.4.6)
│   ├── perf                    Performance profiling
│   │   ├── profile [service]  System/service profile
│   │   ├── analyze            Analyze performance
│   │   ├── slow-queries       Slow query analysis
│   │   ├── report             Generate report
│   │   ├── dashboard          Real-time dashboard
│   │   └── suggest            Optimization tips
│   ├── bench                   Benchmarking
│   │   ├── run [target]       Run benchmark
│   │   ├── baseline           Establish baseline
│   │   ├── compare [file]     Compare to baseline
│   │   ├── stress [target]    Stress test
│   │   └── report             Benchmark report
│   ├── scale                   Service scaling
│   │   ├── (service)          Scale service
│   │   ├── status             Scale status
│   │   └── --auto             Autoscaling
│   └── migrate                 Cross-env migration
│       ├── <src> <target>     Migrate environments
│       ├── diff <s> <t>       Show differences
│       ├── sync <s> <t>       Continuous sync
│       └── rollback           Undo migration
│
├── Operations (v0.4.6)
│   ├── frontend                Frontend management
│   │   ├── status             Frontend status
│   │   ├── list               List frontends
│   │   ├── add <name>         Add frontend
│   │   ├── remove <name>      Remove frontend
│   │   ├── deploy <name>      Deploy frontend
│   │   ├── logs <name>        Deploy logs
│   │   └── env <name>         Environment vars
│   ├── history                 Audit trail
│   │   ├── show               Recent history
│   │   ├── deployments        Deploy history
│   │   ├── migrations         Migration history
│   │   ├── rollbacks          Rollback history
│   │   ├── commands           Command history
│   │   ├── search <query>     Search history
│   │   ├── export             Export history
│   │   └── clear              Clear history
│   └── config                  Configuration
│       ├── show               Show config
│       ├── get <key>          Get value
│       ├── set <key> <val>    Set value
│       ├── list               List keys
│       ├── edit               Open in editor
│       ├── validate           Validate config
│       ├── diff <e1> <e2>     Compare envs
│       ├── export             Export config
│       ├── import <file>      Import config
│       └── reset              Reset to defaults
│
├── Plugins (v0.4.8)
│   └── plugin                 Plugin management
│       ├── list               List available plugins
│       │   ├── --installed    Show installed only
│       │   └── --category     Filter by category
│       ├── install <name>     Install plugin
│       ├── remove <name>      Remove plugin
│       │   └── --keep-data    Keep database tables
│       ├── update [name]      Update plugin(s)
│       │   └── --all          Update all plugins
│       ├── updates            Check for plugin updates
│       ├── refresh            Refresh registry cache
│       ├── status [name]      Plugin status
│       └── <plugin> <action>  Run plugin action
│           ├── stripe         Payment processing
│           │   ├── sync       Sync data
│           │   ├── customers  Customer management
│           │   ├── subscriptions Subscriptions
│           │   ├── invoices   Invoice management
│           │   └── webhook    Webhook events
│           ├── github         DevOps integration
│           │   ├── sync       Sync repositories
│           │   ├── repos      Repository list
│           │   ├── issues     Issue tracking
│           │   ├── prs        Pull requests
│           │   ├── workflows  GitHub Actions
│           │   └── webhook    Webhook events
│           └── shopify        E-commerce
│               ├── sync       Sync store data
│               ├── products   Product catalog
│               ├── orders     Order management
│               ├── customers  Customer data
│               └── webhook    Webhook events
│
├── Utility
│   ├── ssl                     SSL certificates
│   ├── trust                   Trust local certs
│   ├── admin                   Admin UI
│   ├── ci                      CI/CD generation
│   │   ├── init <platform>    Generate workflow
│   │   ├── validate           Validate config
│   │   └── status             CI status
│   ├── completion              Shell completions
│   │   ├── bash               Bash completions
│   │   ├── zsh                Zsh completions
│   │   ├── fish               Fish completions
│   │   └── install <shell>    Auto-install
│   ├── update                  Update nself
│   │   └── --check            Check only
│   ├── version                 Version info
│   │   └── --json             JSON output
│   └── help [command]          Show help
│
└── Legacy (backward compatible)
    ├── providers               → nself cloud provider
    ├── provision               → nself cloud server create
    ├── servers                 → nself cloud server
    ├── email                   → nself service email
    ├── search                  → nself service search
    ├── functions               → nself service functions
    ├── mlflow                  → nself service mlflow
    ├── staging                 → nself deploy staging
    ├── prod                    → nself deploy production
    ├── billing                 → nself tenant billing
    └── whitelabel              → nself tenant branding/domains/email/themes
```

---

## Quick Reference

### Daily Development

```bash
nself start                     # Start services
nself status                    # Check health
nself logs [service]            # View logs
nself db shell                  # Database shell
nself stop                      # Stop services
```

### Database Operations

```bash
nself db migrate up             # Run migrations
nself db migrate create NAME    # Create migration
nself db seed                   # Seed data
nself db backup                 # Create backup
nself db types                  # Generate types
```

### Multi-Tenant Management (NEW v0.9.0)

```bash
# Initialize multi-tenancy
nself tenant init

# Create tenant
nself tenant create "Acme Corp" --slug acme --plan pro

# Manage billing
nself tenant billing usage
nself tenant billing invoice list
nself tenant billing subscription show

# Custom domains
nself tenant domains add app.example.com
nself tenant domains verify app.example.com
nself tenant domains ssl app.example.com

# Branding
nself tenant branding set-colors --primary #0066cc
nself tenant branding upload-logo logo.png

# Email templates
nself tenant email list
nself tenant email edit welcome
```

### OAuth Management (NEW v0.9.0)

```bash
# Install OAuth service
nself oauth install

# Enable providers
nself oauth enable --providers google,github,slack

# Configure provider
nself oauth config google \
  --client-id=xxx.apps.googleusercontent.com \
  --client-secret=GOCSPX-xxx

# Test configuration
nself oauth test google
nself oauth list
nself oauth status
```

### Storage Management (NEW v0.9.0)

```bash
# Initialize storage
nself storage init

# Upload files
nself storage upload photo.jpg
nself storage upload avatar.png --thumbnails
nself storage upload doc.pdf --all-features

# Manage uploads
nself storage list
nself storage list users/123/
nself storage delete users/123/file.txt

# Configuration
nself storage config
nself storage status
nself storage test
```

### Deployment

```bash
nself deploy check              # Pre-deploy validation
nself deploy staging            # Deploy to staging
nself deploy production         # Deploy to production
nself deploy canary             # Start canary deployment
nself deploy blue-green         # Blue-green deployment
nself deploy rollback           # Rollback if needed
```

### Kubernetes

```bash
nself k8s convert               # Generate K8s manifests
nself k8s deploy                # Deploy to cluster
nself helm install              # Install Helm chart
nself helm upgrade              # Upgrade release
```

### Cloud Infrastructure

```bash
nself cloud provider list       # List 26 providers
nself cloud server create       # Provision server
nself cloud cost compare        # Compare provider costs
```

### Service Management

```bash
nself service list              # List optional services
nself service enable redis      # Enable Redis
nself service email test        # Test email
nself service functions deploy  # Deploy functions
nself service admin open        # Open admin UI
```

### Plugins

```bash
nself plugin list               # List available plugins
nself plugin install stripe     # Install Stripe plugin
nself plugin stripe sync        # Sync Stripe data
nself plugin stripe customers list  # List customers
nself plugin updates            # Check for updates
nself plugin refresh            # Refresh registry cache
nself plugin status             # Plugin status
```

### Performance

```bash
nself perf profile              # System profile
nself perf slow-queries         # Find slow queries
nself bench run api             # Benchmark API
nself scale postgres --cpu 2    # Scale service
```

---

## Command Categories

| Category | Commands | Description |
|----------|----------|-------------|
| **Core** | 7 | Project lifecycle |
| **Status** | 6 | Monitoring and debugging |
| **Database** | 10 | Database management |
| **Multi-Tenant** | 32+ | Tenant, billing, branding (NEW v0.9.0) |
| **OAuth** | 7 | OAuth provider management (NEW v0.9.0) |
| **Storage** | 8 | File storage and uploads (NEW v0.9.0) |
| **Deployment** | 12 | Environment and deployment (enhanced v0.4.7) |
| **Cloud** | 4 | Infrastructure management (v0.4.7) |
| **Service** | 15+ | Optional service management (enhanced v0.9.0) |
| **Kubernetes** | 11 | K8s operations (v0.4.7) |
| **Helm** | 12 | Helm chart management (v0.4.7) |
| **Sync** | 8 | Data sync (enhanced v0.4.7) |
| **Performance** | 4 | Profiling and scaling (v0.4.6) |
| **Operations** | 3 | Config, frontend, history (v0.4.6) |
| **Plugins** | 6+ | Third-party integrations (v0.4.8) |
| **Utility** | 7 | CI, completions, updates |

**Total: 150+ commands and subcommands**

---

## Global Options

All commands support these options:

| Option | Description |
|--------|-------------|
| `-h, --help` | Show help for command |
| `--version` | Show version |
| `--json` | JSON output (where supported) |
| `--quiet` | Minimal output |
| `--verbose` | Detailed output |

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ENV` | Current environment | `local` |
| `NSELF_LOG_LEVEL` | Log verbosity | `info` |
| `NSELF_AUTO_FIX` | Auto-fix issues | `false` |
| `NSELF_SKIP_HOOKS` | Skip git hooks | `false` |
| `NSELF_DEFAULT_PROVIDER` | Default cloud provider | - |

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

## Command Documentation

Each command has detailed documentation:

### Core Commands
- [init](INIT.md) | [build](BUILD.md) | [start](START.md) | [stop](STOP.md)

### Status Commands
- [status](STATUS.md) | [urls](URLS.md) | [logs](LOGS.md) | [doctor](DOCTOR.md)

### Database
- [db](DB.md) - Complete database reference

### Multi-Tenant (NEW v0.9.0)
- [tenant](TENANT.md) - Multi-tenant management
- [tenant billing](BILLING.md) - Billing and subscriptions
- [tenant branding](BRANDING.md) - Brand customization
- [tenant domains](DOMAINS.md) - Custom domains and SSL
- [tenant email](EMAIL-TEMPLATES.md) - Email templates
- [tenant themes](THEMES.md) - Theme management

### OAuth (NEW v0.9.0)
- [oauth](OAUTH.md) - OAuth provider management

### Storage (NEW v0.9.0)
- [storage](STORAGE.md) - File storage and uploads

### Deployment
- [deploy](DEPLOY.md) | [env](ENV.md) | [sync](SYNC.md)

### Cloud & Kubernetes (v0.4.7)
- [cloud](CLOUD.md) | [k8s](K8S.md) | [helm](HELM.md) | [service](SERVICE.md)

### Performance (v0.4.6)
- [perf](PERF.md) | [bench](BENCH.md) | [scale](SCALE.md) | [migrate](MIGRATE.md)

### Operations (v0.4.6)
- [health](HEALTH.md) | [frontend](FRONTEND.md) | [history](HISTORY.md) | [config](CONFIG.md)

### Plugins (v0.4.8)
- [plugin](PLUGIN.md) - Plugin management and third-party integrations

---

## Version History

| Version | Commands Added |
|---------|----------------|
| **0.9.0** | tenant (32+ subcommands), oauth (7 commands), storage (8 commands) |
| **0.4.8** | plugin (list, install, remove, update, status), stripe/github/shopify actions |
| **0.4.7** | cloud, service, k8s, helm, deploy preview/canary/blue-green, sync auto/watch |
| **0.4.6** | perf, bench, scale, migrate, health, frontend, history, config, servers |
| **0.4.5** | providers, provision, sync, ci, completion |
| **0.4.4** | db schema, db mock, db types |
| **0.4.3** | env, deploy, staging, prod |

---

## Future: Command Reorganization (Proposed)

A comprehensive reorganization proposal is in progress to reduce top-level commands from 77 to 13 logical categories:

**See:**
- [Command Reorganization Proposal](../architecture/COMMAND-REORGANIZATION-PROPOSAL.md) - Full proposal with rationale
- [Visual Command Guide](../architecture/COMMAND-REORGANIZATION-VISUAL.md) - Before/after visual comparison
- [Command Consolidation Map](../architecture/COMMAND-CONSOLIDATION-MAP.md) - Detailed command flow diagrams
- [Implementation Checklist](../architecture/COMMAND-REORGANIZATION-CHECKLIST.md) - Development roadmap

**Proposed Structure:**
- **13 categories** instead of 77 top-level commands (71% reduction)
- **Backward compatible** with legacy aliases for 2+ versions
- **Logical grouping**: `observe` (monitoring), `secure` (security), expanded `auth`, `service`, `deploy`, `cloud`, `dev`, `config`
- **4-phase rollout** over 6-12 months with deprecation warnings

This reorganization aims to improve discoverability, reduce cognitive load, and create a clearer mental model while maintaining full backward compatibility during the transition.

---

*Last Updated: January 30, 2026 | Version: 0.9.0*
