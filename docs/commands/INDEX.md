# Commands Documentation

Complete reference for all ɳSelf CLI commands.

## Overview

- **[All Commands](COMMANDS.md)** - Overview of 150+ commands
- **[Command Overview](README.md)** - Getting started with commands
- **[Command Tree v1](COMMAND-TREE-V1.md)** - Complete v1.0 command structure

## Core Commands (5)

Essential commands for everyday use:

- **[init](INIT.md)** - Initialize new project
- **[build](BUILD.md)** - Generate configuration files
- **[start](START.md)** - Start services
- **[stop](STOP.md)** - Stop services
- **[restart](RESTART.md)** - Restart services

## Database Commands

- **[db](DB.md)** - Database management (11 subcommands)
  - migrate, seed, mock, backup, restore, schema, types, reset, clean

## Multi-Tenant Commands

- **[tenant](TENANT.md)** - Multi-tenant management (50+ subcommands)
  - create, list, show, suspend, activate, delete
- **[billing](BILLING.md)** - Billing & subscriptions (now: `tenant billing`)
- **[branding](BRANDING.md)** - Brand customization
- **[domains](DOMAINS.md)** - Custom domains & SSL
- **[email](EMAIL.md)** - Email templates
- **[themes](THEMES.md)** - Theme management

## Deployment Commands

- **[deploy](DEPLOY.md)** - Deployment operations (23 subcommands)
- **[env](ENV.md)** - Environment management
- **[prod](PROD.md)** - Production deployment (now: `deploy prod`)
- **[staging](STAGING.md)** - Staging deployment (now: `deploy staging`)
- **[provision](PROVISION.md)** - Server provisioning (now: `deploy provision`)
- **[servers](SERVERS.md)** - Server management (now: `deploy server`)
- **[sync](SYNC.md)** - Configuration sync (now: `deploy sync`)

## Infrastructure Commands

- **[infra](SERVICE.md)** - Infrastructure operations (38 subcommands)
- **[provider](PROVIDER.md)** - Cloud providers (now: `infra provider`)
- **[k8s](K8S.md)** - Kubernetes (now: `infra k8s`)
- **[helm](HELM.md)** - Helm charts (now: `infra helm`)

## Service Commands

- **[service](SERVICE.md)** - Service management (43 subcommands)
- **[storage](storage.md)** - File storage (now: `service storage`)
- **[email](EMAIL.md)** - Email service (now: `service email`)
- **[search](SEARCH.md)** - Search service (now: `service search`)
- **[functions](FUNCTIONS.md)** - Serverless functions (now: `service functions`)
- **[mlflow](MLFLOW.md)** - MLflow tracking (now: `service mlflow`)
- **[realtime](REALTIME.md)** - Real-time features (now: `service realtime`)

## Configuration Commands

- **[config](CONFIG.md)** - Configuration management (20 subcommands)
- **[env](ENV.md)** - Environment variables (now: `config env`)

## Authentication & Security Commands

- **[auth](AUTH.md)** - Security operations (38 subcommands)
- **[oauth](OAUTH.md)** - OAuth providers (now: `auth oauth`)
- **[mfa](MFA.md)** - Multi-factor auth (now: `auth mfa`)
- **[ssl](SSL.md)** - SSL certificates (now: `auth ssl`)
- **[trust](TRUST.md)** - Trust management (now: `auth ssl trust`)
- **[devices](DEVICES.md)** - Device management (now: `auth devices`)

## Performance Commands

- **[perf](PERF.md)** - Performance operations (5 subcommands)
- **[bench](BENCH.md)** - Benchmarking (now: `perf bench`)
- **[scale](SCALE.md)** - Service scaling (now: `perf scale`)
- **[migrate](MIGRATE.md)** - Data migration (now: `perf migrate`)

## Backup & Recovery Commands

- **[backup](BACKUP.md)** - Backup operations (6 subcommands)
- **[restore](RESTORE.md)** - Restore from backup (now: `backup restore`)
- **[rollback](ROLLBACK.md)** - Rollback changes (now: `backup rollback`)
- **[reset](RESET.md)** - Reset state (now: `backup reset`)
- **[clean](CLEAN.md)** - Clean up (now: `backup clean`)

## Developer Commands

- **[dev](DEV.md)** - Developer tools (16 subcommands)
- **[frontend](FRONTEND.md)** - Frontend tools (now: `dev frontend`)
- **[ci](CI.md)** - CI/CD tools (now: `dev ci`)
- **[whitelabel](WHITELABEL.md)** - White-labeling (now: `dev whitelabel`)

## Plugin Commands

- **[plugin](PLUGIN.md)** - Plugin management (8+ subcommands)
  - install, uninstall, list, update, search, enable, disable

## Utility Commands (15)

- **[status](STATUS.md)** - Service health status
- **[logs](LOGS.md)** - View service logs
- **[help](HELP.md)** - Help system
- **[admin](ADMIN.md)** - Admin UI
- **[urls](URLS.md)** - Show service URLs
- **[exec](EXEC.md)** - Execute in container
- **[doctor](DOCTOR.md)** - Diagnostics
- **[monitor](MONITOR.md)** - Monitoring dashboards
- **[health](HEALTH.md)** - Health checks
- **[version](VERSION.md)** - Version info
- **[update](UPDATE.md)** - Update nself
- **[completion](COMPLETION.md)** - Shell completions
- **[metrics](METRICS.md)** - Metrics & profiling
- **[history](HISTORY.md)** - Audit trail
- **[audit](AUDIT.md)** - Audit logging

## Deprecated Commands

Commands that have been consolidated (still work with deprecation warnings):

- **[up](UP.md)** - Use `nself start` instead
- **[down](DOWN.md)** - Use `nself stop` instead
- **[destroy](DESTROY.md)** - Use `nself stop --remove-volumes` instead
- **[admin-dev](ADMIN-DEV.md)** - Use `nself admin --dev` instead

## Reference

- **[Auth Consolidation](auth-consolidation.md)** - Auth command changes
- **[Refactoring Summary](REFACTORING-SUMMARY.md)** - Command refactoring history

---

**[← Back to Documentation Home](../README.md)**
