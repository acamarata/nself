# nself Commands

Complete reference for all nself CLI commands.

---

## Quick Links

- **[Commands Overview](COMMANDS.md)** - All 55+ commands
- **[Quick Reference](../quick-reference/COMMAND-REFERENCE.md)** - Printable cheat sheet

---

## Core Commands

Essential commands for daily use:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **init** | Initialize a new project | [INIT.md](INIT.md) |
| **build** | Generate configuration and build images | [BUILD.md](BUILD.md) |
| **start** | Start all services | [START.md](START.md) |
| **stop** | Stop all services | [STOP.md](STOP.md) |
| **restart** | Restart services | [RESTART.md](RESTART.md) |
| **status** | Check service status | [STATUS.md](STATUS.md) |
| **urls** | List service URLs | [URLS.md](URLS.md) |
| **logs** | View service logs | [LOGS.md](LOGS.md) |

---

## Database Commands

Database management and development:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **db** | Database tools (main command) | [DB.md](DB.md) |
| **db migrate** | Run migrations | [DB.md](DB.md#migrations) |
| **db seed** | Seed database | [DB.md](DB.md#seeding) |
| **db mock** | Generate mock data | [DB.md](DB.md#mock-data) |
| **db backup** | Create backups | [DB.md](DB.md#backups) |
| **db schema** | Schema management | [DB.md](DB.md#schema) |
| **db types** | Generate TypeScript types | [DB.md](DB.md#types) |

---

## Deployment Commands

Production deployment and environment management:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **deploy** | Deploy to server via SSH | [DEPLOY.md](DEPLOY.md) |
| **env** | Environment management | [ENV.md](ENV.md) |
| **prod** | Production shortcuts | [PROD.md](PROD.md) |
| **staging** | Staging shortcuts | [STAGING.md](STAGING.md) |
| **servers** | Server infrastructure management | [SERVERS.md](SERVERS.md) |

---

## Performance Commands

Performance profiling and optimization:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **perf** | Performance profiling | [PERF.md](PERF.md) |
| **bench** | Benchmarking and load testing | [BENCH.md](BENCH.md) |
| **scale** | Service scaling | [SCALE.md](SCALE.md) |
| **metrics** | Metrics and monitoring | [METRICS.md](METRICS.md) |

---

## Plugin Commands (v0.4.8)

Plugin management and operations:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **plugin** | Plugin management | [PLUGIN.md](PLUGIN.md) |
| **plugin list** | List available plugins | [PLUGIN.md](PLUGIN.md#list) |
| **plugin install** | Install a plugin | [PLUGIN.md](PLUGIN.md#install) |
| **plugin updates** | Check for updates | [PLUGIN.md](PLUGIN.md#updates) |

**Available Plugins:**
- [Stripe](../plugins/stripe.md) - Payment processing
- [GitHub](../plugins/github.md) - Repository sync
- [Shopify](../plugins/shopify.md) - E-commerce

---

## Service Commands

Service-specific operations:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **email** | Email service management | [EMAIL.md](EMAIL.md) |
| **search** | Search service management | [SEARCH.md](SEARCH.md) |
| **functions** | Functions management | [FUNCTIONS.md](FUNCTIONS.md) |
| **mlflow** | MLflow operations | [MLFLOW.md](MLFLOW.md) |
| **admin** | Admin UI operations | [ADMIN.md](ADMIN.md) |

---

## Billing Commands

Billing, payment, and usage management:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **billing** | Billing and usage management | [BILLING.md](BILLING.md) |
| **billing usage** | View usage and metrics | [BILLING.md](BILLING.md#usage-tracking) |
| **billing invoice** | Manage invoices | [BILLING.md](BILLING.md#invoices) |
| **billing subscription** | Manage subscriptions | [BILLING.md](BILLING.md#subscriptions) |
| **billing payment** | Manage payment methods | [BILLING.md](BILLING.md#payment-methods) |
| **billing quota** | Check quota limits | [BILLING.md](BILLING.md#quota-management) |
| **billing plan** | View billing plans | [BILLING.md](BILLING.md#billing-plans) |
| **billing export** | Export billing data | [BILLING.md](BILLING.md#export-billing-data) |

---

## Infrastructure Commands (v0.4.7)

Cloud and infrastructure management:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **cloud** | Cloud provider operations | [CLOUD.md](CLOUD.md) |
| **k8s** | Kubernetes operations | [K8S.md](K8S.md) |
| **helm** | Helm chart management | [HELM.md](HELM.md) |
| **providers** | Provider configuration | [PROVIDERS.md](PROVIDERS.md) |

---

## Operations Commands

Operational tools and utilities:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **health** | Health check management | [HEALTH.md](HEALTH.md) |
| **doctor** | Automated diagnostics | [DOCTOR.md](DOCTOR.md) |
| **config** | Configuration management | [CONFIG.md](CONFIG.md) |
| **history** | Audit trail | [HISTORY.md](HISTORY.md) |
| **monitor** | Monitoring operations | [MONITOR.md](MONITOR.md) |

---

## Utility Commands

Utilities and helpers:

| Command | Description | Documentation |
|---------|-------------|---------------|
| **exec** | Execute commands in containers | [EXEC.md](EXEC.md) |
| **clean** | Clean up resources | [CLEAN.md](CLEAN.md) |
| **completion** | Shell completion | [COMPLETION.md](COMPLETION.md) |
| **version** | Show version | [VERSION.md](VERSION.md) |
| **help** | Show help | Built-in |

---

## Command Categories

### By Use Case

**Getting Started:**
- init, build, start, urls, status

**Development:**
- db, logs, exec, restart

**Testing:**
- db mock, db seed, bench

**Deployment:**
- deploy, env, prod, staging

**Monitoring:**
- status, health, metrics, logs

**Optimization:**
- perf, bench, scale

---

## Common Workflows

### Local Development
```bash
nself init
nself build
nself start
nself db schema apply schema.dbml
nself urls
```

### Database Development
```bash
nself db schema scaffold saas
# Edit schema.dbml
nself db schema apply schema.dbml
nself db types generate
```

### Production Deployment
```bash
nself env create prod production
# Edit server.json
nself deploy prod
nself prod status
```

### Performance Testing
```bash
nself bench api
nself perf profile
nself metrics view
```

---

## Getting Help

```bash
# General help
nself help

# Command-specific help
nself <command> --help

# Examples
nself db --help
nself deploy --help
```

---

**[Back to Documentation Home](../README.md)**
