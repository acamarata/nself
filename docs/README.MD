# nself Documentation

> **The complete self-hosted Backend-as-a-Service platform**

[![Version](https://img.shields.io/badge/version-0.4.7-blue.svg)](releases/v0.4.7.md)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](../LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey.svg)]()

---

## Quick Start

```bash
# Install
curl -sSL https://install.nself.org | bash

# Create project
mkdir myapp && cd myapp
nself init && nself build && nself start

# Design database (optional)
nself db schema scaffold basic
nself db schema apply schema.dbml
```

**That's it.** Your backend is running with GraphQL API, authentication, and database.

---

## Documentation Index

### Getting Started

| Document | Description |
|----------|-------------|
| [Quick Start](guides/Quick-Start.md) | Get running in 5 minutes |
| [Installation](guides/Installation.md) | Detailed installation guide |
| [Database Workflow](guides/DATABASE-WORKFLOW.md) | Schema design to production |
| [Demo Setup](services/DEMO_SETUP.md) | Full demo with 25 services |
| [FAQ](guides/FAQ.md) | Frequently asked questions |

### Commands Reference

| Document | Description |
|----------|-------------|
| [Commands Overview](commands/COMMANDS.md) | All 55+ commands |
| [db](commands/DB.md) | Database tools (migrate, seed, mock, backup, schema, types) |
| [deploy](commands/DEPLOY.md) | SSH deployment with zero-downtime |
| [env](commands/ENV.md) | Environment management |
| [perf](commands/PERF.md) | Performance profiling and analysis |
| [bench](commands/BENCH.md) | Benchmarking and load testing |
| [health](commands/HEALTH.md) | Health check management |
| [config](commands/CONFIG.md) | Configuration management |
| [servers](commands/SERVERS.md) | Server infrastructure |

### Configuration

| Document | Description |
|----------|-------------|
| [Environment Variables](configuration/ENVIRONMENT-VARIABLES.md) | Complete variable reference |
| [Start Command Options](configuration/START-COMMAND-OPTIONS.md) | Startup configuration |
| [Admin UI](configuration/Admin-UI.md) | Web dashboard setup |

### Services

| Document | Description |
|----------|-------------|
| [Services Overview](services/SERVICES.md) | All available services |
| [Required Services](services/SERVICES_REQUIRED.md) | Core 4: PostgreSQL, Hasura, Auth, Nginx |
| [Optional Services](services/SERVICES_OPTIONAL.md) | 7 additional services |
| [Monitoring Bundle](services/MONITORING-BUNDLE.md) | 10-service observability stack |
| [Custom Services](services/SERVICES_CUSTOM.md) | 40+ templates |
| [Service Templates](services/SERVICE_TEMPLATES.md) | Template reference |

### Architecture

| Document | Description |
|----------|-------------|
| [Architecture Overview](architecture/ARCHITECTURE.md) | System design |
| [Project Structure](architecture/PROJECT_STRUCTURE.md) | File organization |
| [Build Architecture](architecture/BUILD_ARCHITECTURE.md) | How build works |
| [API Reference](architecture/API.md) | GraphQL and REST APIs |

### Guides

| Document | Description |
|----------|-------------|
| [Database Workflow](guides/DATABASE-WORKFLOW.md) | DBML to production |
| [Deployment](guides/Deployment.md) | Production deployment |
| [Backup Guide](guides/BACKUP_GUIDE.md) | Backup and recovery |
| [Multi-App Setup](guides/MULTI_APP_SETUP.md) | Multiple applications |
| [Troubleshooting](guides/TROUBLESHOOTING.md) | Common issues |
| [Examples](guides/EXAMPLES.md) | Code examples |

### Releases

| Document | Description |
|----------|-------------|
| [Roadmap](releases/ROADMAP.md) | Future plans |
| [v0.4.7](releases/v0.4.7.md) | Current release - Infrastructure Everywhere |
| [v0.4.6](releases/v0.4.6.md) | Scaling & Performance |
| [v0.4.5](releases/v0.4.5.md) | Provider Support |
| [v0.4.4](releases/v0.4.4.md) | Database Tools |
| [v0.4.3](releases/v0.4.3.md) | Deployment Pipeline |
| [v0.4.2](releases/v0.4.2.md) | Service & Monitoring |
| [Changelog](releases/CHANGELOG.md) | Full version history |

### Contributing

| Document | Description |
|----------|-------------|
| [Contributing Guide](CONTRIBUTING.md) | How to contribute |
| [Development Setup](contributing/DEVELOPMENT.md) | Dev environment |
| [Cross-Platform](contributing/CROSS-PLATFORM-COMPATIBILITY.md) | Bash 3.2+ compatibility |

---

## What is nself?

nself is a complete backend platform you can run anywhere:

```
┌─────────────────────────────────────────────────────────────────┐
│                         YOUR APP                                │
├─────────────────────────────────────────────────────────────────┤
│  Frontend (React, Next.js, Vue, etc.)                           │
│  ↓ GraphQL/REST                                                 │
├─────────────────────────────────────────────────────────────────┤
│                         nself                                   │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐     │
│  │  Hasura     │  Auth       │  Storage    │  Functions  │     │
│  │  GraphQL    │  JWT/OAuth  │  S3/MinIO   │  Serverless │     │
│  └─────────────┴─────────────┴─────────────┴─────────────┘     │
│  ┌─────────────┬─────────────┬─────────────┬─────────────┐     │
│  │  PostgreSQL │  Redis      │  Search     │  Mail       │     │
│  │  Database   │  Cache      │  Meili      │  SMTP       │     │
│  └─────────────┴─────────────┴─────────────┴─────────────┘     │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Monitoring: Prometheus, Grafana, Loki, Tempo           │   │
│  └─────────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────┤
│  Docker Compose │ Any Cloud │ Any Server │ Local Machine       │
└─────────────────────────────────────────────────────────────────┘
```

### Key Features

- **Zero DevOps** - One command to run a complete backend
- **Database-First** - Design in DBML, generate everything
- **Environment-Aware** - Safe defaults for local/staging/production
- **40+ Templates** - Custom services in any language
- **Full Observability** - Metrics, logs, traces, alerts
- **Production-Ready** - SSL, security, backups, deployment

---

## Service Counts

| Configuration | Services | Use Case |
|---------------|----------|----------|
| Minimal | 4 | Just the essentials |
| Typical Dev | 8-12 | Development with extras |
| Full Demo | 25 | Everything enabled |
| Maximum | 50+ | All optional + 10 custom |

### Required Services (4)
Always running: **PostgreSQL**, **Hasura**, **Auth**, **Nginx**

### Optional Services (7)
Enable with `*_ENABLED=true`: **Admin**, **Redis**, **MinIO**, **Functions**, **MLflow**, **Mail**, **Search**

### Monitoring Bundle (10)
Enable with `MONITORING_ENABLED=true`: **Prometheus**, **Grafana**, **Loki**, **Promtail**, **Tempo**, **Alertmanager**, **cAdvisor**, **Node Exporter**, **Postgres Exporter**, **Redis Exporter**

### Custom Services (Unlimited)
Build from 40+ templates: **Express**, **FastAPI**, **gRPC**, **BullMQ**, and more

---

## Version History

| Version | Date | Focus |
|---------|------|-------|
| **v0.4.7** | Jan 23, 2026 | Infrastructure Everywhere (current) |
| v0.4.6 | Jan 22, 2026 | Scaling & Performance |
| v0.4.5 | Jan 21, 2026 | Provider Support |
| v0.4.4 | Jan 20, 2026 | Database Tools |
| v0.4.3 | Jan 19, 2026 | Deployment Pipeline |
| v0.4.2 | Jan 18, 2026 | Service & Monitoring |
| v0.4.1 | Jan 17, 2026 | Platform Compatibility |
| v0.4.0 | Oct 2025 | Production-Ready Core |

---

## Links

- **GitHub**: [github.com/acamarata/nself](https://github.com/acamarata/nself)
- **Issues**: [Report bugs](https://github.com/acamarata/nself/issues)
- **Discussions**: [Ask questions](https://github.com/acamarata/nself/discussions)

---

*Last Updated: January 23, 2026 | Version: 0.4.7*
