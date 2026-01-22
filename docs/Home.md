# nself - Self-Hosted Infrastructure Manager

<div align="center">

**Complete Backend-as-a-Service Platform**

[![Version](https://img.shields.io/badge/version-0.4.1-blue.svg)](releases/v0.4.1)
[![License](https://img.shields.io/badge/license-Personal%20Free%20%7C%20Commercial-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-compose%20v2-blue.svg)](https://docs.docker.com/compose/)
[![Status](https://img.shields.io/badge/status-stable-brightgreen.svg)](https://github.com/acamarata/nself)

*Current: v0.4.1 (Stable) | Platform: macOS, Linux, WSL*

</div>

---

## Quick Start

```bash
# Install nself
curl -sSL https://install.nself.org | bash

# Initialize a demo project with all services
nself init --demo

# Build services from templates
nself build

# Start everything
nself start
```

> **[View Complete Demo Setup](services/DEMO_SETUP)** - 25 services showcasing all capabilities

---

## Documentation

### Getting Started
Start here if you're new to nself.

- **[Quick Start Guide](guides/Quick-Start)** - Get up and running in 5 minutes
- **[Installation Guide](guides/Installation)** - Detailed installation instructions
- **[Demo Setup](services/DEMO_SETUP)** - Complete demo with all services
- **[FAQ](guides/FAQ)** - Frequently asked questions
- **[Troubleshooting](guides/TROUBLESHOOTING)** - Common issues and solutions

### Core Documentation
Essential information for using nself.

- **[Commands Reference](commands/COMMANDS)** - Complete CLI reference (36 commands)
- **[Command Tree](commands/COMMAND-TREE-FINAL)** - Visual command structure
- **[Environment Variables](configuration/ENVIRONMENT-VARIABLES)** - Configuration reference
- **[Environment Setup](configuration/ENVIRONMENT_CONFIGURATION)** - Configuration guide

### Architecture
Understand how nself works.

- **[Architecture Overview](architecture/ARCHITECTURE)** - System design and principles
- **[Project Structure](architecture/PROJECT_STRUCTURE)** - File organization
- **[Build Architecture](architecture/BUILD_ARCHITECTURE)** - How the build system works

### Services
All available services and how to use them.

- **[Services Overview](services/SERVICES)** - All available services
- **[Required Services](services/SERVICES_REQUIRED)** - Core infrastructure (4 services)
- **[Optional Services](services/SERVICES_OPTIONAL)** - Additional capabilities (7+ services)
- **[Custom Services](services/SERVICES_CUSTOM)** - Build your own microservices (40+ templates)
- **[Monitoring Bundle](services/MONITORING-BUNDLE)** - Complete observability (10 services)
- **[nself Admin](services/NSELF_ADMIN)** - Web-based management interface

### Guides
Step-by-step instructions for common tasks.

- **[Backup Guide](guides/BACKUP_GUIDE)** - Backup and recovery strategies
- **[Domain Selection](guides/domain-selection-guide)** - Choosing your domain strategy
- **[Deployment](guides/Deployment)** - Deploy to production
- **[Examples](guides/EXAMPLES)** - Code examples and patterns

### Contributing
Help make nself better!

- **[Contributing Guide](CONTRIBUTING)** - Start here for contribution guidelines
- **[Development Setup](contributing/DEVELOPMENT)** - Development environment and standards
- **[Cross-Platform Compatibility](contributing/CROSS-PLATFORM-COMPATIBILITY)** - Bash 3.2+, POSIX compliance
- **[Code of Conduct](contributing/CODE_OF_CONDUCT)** - Community guidelines

### Releases & Updates
Stay up to date with nself development.

- **[Changelog](CHANGELOG)** - Complete version history
- **[Roadmap](releases/ROADMAP)** - Planned features and improvements
- **[v0.4.1 Release](releases/v0.4.1)** - Current stable release
- **[All Releases](releases/INDEX)** - View all releases

---

## Service Overview

### Core Services (Required - 4)
Always enabled, form the foundation of your infrastructure.

- **PostgreSQL** - Primary database with 60+ extensions
- **Hasura** - Instant GraphQL APIs with real-time subscriptions
- **Auth** - JWT-based authentication with social providers
- **Nginx** - Reverse proxy, SSL termination, routing

### Optional Services (7 types)
Enable what you need with `*_ENABLED=true`.

- **nself Admin** - Web-based management interface
- **Redis** - Caching, sessions, pub/sub
- **MinIO** - S3-compatible object storage
- **Functions** - Serverless functions runtime
- **MLflow** - ML experiment tracking and model registry
- **Mail** - Email service (MailPit for dev, SMTP for prod)
- **Search** - Full-text search (MeiliSearch, Typesense, Sonic)

### Monitoring Bundle (10 services)
Complete observability stack, enabled with `MONITORING_ENABLED=true`.

- **Prometheus** - Metrics collection and storage
- **Grafana** - Visualization and dashboards
- **Loki** - Log aggregation
- **Promtail** - Log shipping (required for Loki)
- **Tempo** - Distributed tracing
- **Alertmanager** - Alert routing and management
- **cAdvisor** - Container metrics
- **Node Exporter** - System metrics
- **Postgres Exporter** - Database metrics
- **Redis Exporter** - Cache metrics

### Custom Services (Your code)
Build from 40+ templates with automatic Docker integration.

**Languages:** JavaScript/TypeScript, Python, Go, Rust, Java, PHP
**Frameworks:** Express, FastAPI, gRPC, BullMQ, GraphQL, REST
**Templates:** API servers, workers, queues, microservices, and more

---

## Demo Project

The demo setup showcases all capabilities with **25 running services**:

```bash
nself init --demo  # Creates complete demo environment
nself build        # Generates 4 custom services from templates
nself start        # Launches all 25 services
```

**Demo includes:**
- 4 Required Services (PostgreSQL, Hasura, Auth, Nginx)
- 7 Optional Services (Redis, MinIO, Admin, Mail, Search, MLflow, Functions)
- 10 Monitoring Services (complete observability stack)
- 4 Custom Services (Express API, BullMQ worker, Go gRPC, Python FastAPI)

**Access everything at:** `*.local.nself.org` (21 routes configured)

---

## Key Features

- **Zero DevOps** - Focus on code, not infrastructure
- **One Command** - `nself init && nself build && nself start`
- **Admin Dashboard** - Complete web-based management
- **Authentication** - JWT with social login providers
- **S3 Storage** - MinIO with CDN support
- **Auto-SSL** - Self-signed for local, Let's Encrypt for production
- **Full Monitoring** - Metrics, logs, traces, alerts
- **Multi-Environment** - Dev, staging, production configurations
- **60+ Extensions** - PostgreSQL fully loaded
- **Email Service** - Multiple provider support
- **Search Engine** - Multiple providers (MeiliSearch, Typesense, Sonic)
- **Functions** - Serverless runtime for Node.js/Python
- **ML Tracking** - MLflow for experiment tracking

---

## Community & Support

- **GitHub Issues** - [Report bugs and request features](https://github.com/acamarata/nself/issues)
- **GitHub Discussions** - [Ask questions and share ideas](https://github.com/acamarata/nself/discussions)

---

## License

nself is source-available software. Free for personal use, commercial license required for business use.

---

**Current Version:** v0.4.1 | **Status:** Stable | **Platforms:** macOS, Linux, WSL

> **[View Changelog](CHANGELOG)** for version history | **[View Roadmap](releases/ROADMAP)** for upcoming features
