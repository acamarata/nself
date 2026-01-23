# nself Commands Reference

**Version 0.4.6** | Complete CLI Reference

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
├── Deployment (nself deploy)
│   ├── (environment)           Deploy to environment
│   │   ├── staging            Deploy to staging
│   │   └── prod               Deploy to production
│   ├── init                    Initialize config
│   ├── check                   Pre-deploy validation (v0.4.6)
│   │   └── --fix              Auto-fix issues
│   ├── status                  Deployment status
│   ├── rollback               Rollback deployment
│   ├── logs                    Deployment logs
│   ├── webhook                 Setup webhooks
│   ├── health                  Check health
│   └── check-access           Verify SSH access
│
├── Environment (nself env)
│   ├── (default)               List environments
│   ├── create <name>           Create environment
│   ├── switch <name>           Switch environment
│   └── diff <env1> <env2>      Compare environments
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
├── Infrastructure
│   ├── servers                 Server management (v0.4.6)
│   │   ├── list               List servers
│   │   ├── add <name>         Add server
│   │   ├── remove <name>      Remove server
│   │   ├── status [name]      Server status
│   │   ├── ssh <name>         SSH to server
│   │   ├── logs <name>        Server logs
│   │   ├── update <name>      Update config
│   │   ├── reboot <name>      Reboot server
│   │   └── info <name>        Detailed info
│   ├── providers               Cloud providers
│   │   ├── (default)          List providers
│   │   ├── init <provider>    Configure provider
│   │   ├── status             Provider status
│   │   ├── costs              Cost comparison
│   │   └── remove <provider>  Remove provider
│   ├── provision               Provision infra
│   │   ├── <provider>         Provision on provider
│   │   ├── --estimate         Show cost only
│   │   ├── --dry-run          Preview resources
│   │   └── export terraform   Export as Terraform
│   └── sync                    Data sync
│       ├── pull <env>         Pull from env
│       ├── push <env>         Push to env
│       ├── files              File sync
│       └── config             Config sync
│
├── Services
│   ├── ssl                     SSL certificates
│   ├── trust                   Trust local certs
│   ├── admin                   Admin UI
│   │   ├── enable             Enable admin
│   │   ├── disable            Disable admin
│   │   ├── open               Open in browser
│   │   └── password           Set password
│   ├── email                   Email service
│   ├── search                  Search service
│   ├── functions               Serverless functions
│   ├── mlflow                  ML tracking
│   ├── metrics                 Monitoring stack
│   └── monitor                 Access dashboards
│
├── Utility
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
└── Legacy/Deprecated
    ├── staging                 → nself deploy staging
    ├── prod                    → nself deploy prod
    └── rollback                → nself deploy rollback
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

### Deployment

```bash
nself deploy check              # Pre-deploy validation
nself deploy staging            # Deploy to staging
nself deploy prod --dry-run     # Preview production
nself deploy prod               # Deploy to production
nself deploy rollback           # Rollback if needed
```

### Performance

```bash
nself perf profile              # System profile
nself perf slow-queries         # Find slow queries
nself bench run api             # Benchmark API
nself bench baseline            # Establish baseline
nself scale postgres --cpu 2    # Scale service
```

### Configuration

```bash
nself config show               # View config
nself config validate           # Validate config
nself config diff local staging # Compare envs
nself env create prod           # Create environment
```

---

## Command Categories

| Category | Commands | Description |
|----------|----------|-------------|
| **Core** | 7 | Project lifecycle |
| **Status** | 6 | Monitoring and debugging |
| **Database** | 10 | Database management |
| **Deployment** | 8 | Environment and deployment |
| **Performance** | 4 | Profiling and scaling (v0.4.6) |
| **Operations** | 3 | Config, frontend, history (v0.4.6) |
| **Infrastructure** | 4 | Servers, providers, sync |
| **Services** | 8 | Service-specific operations |
| **Utility** | 5 | CI, completions, updates |

**Total: 55+ commands and subcommands**

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

### Deployment
- [deploy](DEPLOY.md) | [env](ENV.md) | [sync](SYNC.md)

### Performance (v0.4.6)
- [perf](PERF.md) | [bench](BENCH.md) | [scale](SCALE.md) | [migrate](MIGRATE.md)

### Operations (v0.4.6)
- [health](HEALTH.md) | [frontend](FRONTEND.md) | [history](HISTORY.md) | [config](CONFIG.md)

### Infrastructure
- [servers](SERVERS.md) | [providers](PROVIDERS.md) | [provision](PROVISION.md)

---

## Version History

| Version | Commands Added |
|---------|----------------|
| **0.4.6** | perf, bench, scale, migrate, health, frontend, history, config, servers |
| **0.4.5** | providers, provision, sync, ci, completion |
| **0.4.4** | db schema, db mock, db types |
| **0.4.3** | env, deploy, staging, prod |

---

*Last Updated: January 23, 2026 | Version: 0.4.6*
