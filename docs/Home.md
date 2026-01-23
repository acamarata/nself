# nself

<div align="center">

**The Complete Self-Hosted Backend Platform**

[![Version](https://img.shields.io/badge/version-0.4.7-blue.svg)](releases/v0.4.7)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](../LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey.svg)]()
[![Status](https://img.shields.io/badge/status-stable-brightgreen.svg)]()

*Deploy a complete backend in minutes, not days.*

</div>

---

## 5-Minute Quick Start

```bash
# 1. Install nself
curl -sSL https://install.nself.org | bash

# 2. Create your project
mkdir myapp && cd myapp
nself init

# 3. Build and start
nself build && nself start

# 4. Design your database (optional but recommended)
nself db schema scaffold basic    # Creates schema.dbml
nself db schema apply schema.dbml # Import → migrate → seed
```

**Done!** You now have:
- GraphQL API at `api.local.nself.org`
- Authentication at `auth.local.nself.org`
- Database with your schema
- Sample users to test with

---

## What You Get

```
┌─────────────────────────────────────────────────────────────────────┐
│                          YOUR APPLICATION                            │
├─────────────────────────────────────────────────────────────────────┤
│   Frontend (React, Vue, Next.js, etc.)                              │
│   ↓ GraphQL queries and mutations                                   │
├─────────────────────────────────────────────────────────────────────┤
│                              nself                                   │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              ALWAYS RUNNING (4 services)                  │      │
│   │   PostgreSQL  ·  Hasura GraphQL  ·  Auth  ·  Nginx       │      │
│   └──────────────────────────────────────────────────────────┘      │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              OPTIONAL (enable as needed)                  │      │
│   │   Redis  ·  MinIO  ·  Search  ·  Mail  ·  Functions      │      │
│   │   MLflow  ·  Admin Dashboard                              │      │
│   └──────────────────────────────────────────────────────────┘      │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              MONITORING (all-or-nothing bundle)           │      │
│   │   Prometheus · Grafana · Loki · Tempo · Alertmanager     │      │
│   └──────────────────────────────────────────────────────────┘      │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              YOUR CUSTOM SERVICES                         │      │
│   │   Express · FastAPI · gRPC · BullMQ · and 40+ more       │      │
│   └──────────────────────────────────────────────────────────┘      │
├─────────────────────────────────────────────────────────────────────┤
│   Runs on: Docker Compose · Any Cloud · Any Server · Laptop         │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Key Features

### Database-First Development
Design your schema visually, generate everything automatically.

```bash
nself db schema scaffold saas       # Start with a template
# Edit schema.dbml at dbdiagram.io
nself db schema apply schema.dbml   # Creates tables + mock data + users
```

### Environment-Aware Safety
Different behavior for local, staging, and production.

| Command | Local | Staging | Production |
|---------|-------|---------|------------|
| `db mock` | Works | Works | Blocked |
| `db reset` | Works | Confirm | Blocked |
| `db seed users` | Mock users | QA users | Only explicit users |

### Smart Defaults Everywhere
Zero configuration required for common cases.

```bash
nself db migrate up     # Just works - no flags needed
nself db backup         # Auto-names: myapp_local_20260122.sql
nself db seed           # Knows your environment
nself db types          # Generates TypeScript by default
```

### 40+ Service Templates
Build custom services in any language.

**JavaScript/TypeScript:** Express, Fastify, NestJS, Hono, BullMQ
**Python:** FastAPI, Flask, Django, Celery
**Go:** Gin, Fiber, Echo, gRPC
**Others:** Rust, Java, PHP, Ruby, C#, Elixir

---

## Documentation

### Start Here

| Guide | Description |
|-------|-------------|
| **[Quick Start](guides/Quick-Start)** | Get running in 5 minutes |
| **[Database Workflow](guides/DATABASE-WORKFLOW)** | DBML to production in one command |
| **[Demo Setup](services/DEMO_SETUP)** | Full demo with 25 services |

### Commands Reference

| Command | Description |
|---------|-------------|
| **[db](commands/DB)** | Database tools - migrate, seed, mock, backup, schema, types |
| **[deploy](commands/DEPLOY)** | SSH deployment with zero-downtime |
| **[env](commands/ENV)** | Environment management |
| **[perf](commands/PERF)** | Performance profiling and analysis |
| **[bench](commands/BENCH)** | Benchmarking and load testing |
| **[All Commands](commands/COMMANDS)** | Complete reference (55+ commands) |

### Services

| Guide | Description |
|-------|-------------|
| **[Services Overview](services/SERVICES)** | All available services |
| **[Custom Services](services/SERVICES_CUSTOM)** | Build your own with templates |
| **[Monitoring Bundle](services/MONITORING-BUNDLE)** | Full observability stack |

### Deep Dives

| Guide | Description |
|-------|-------------|
| **[Architecture](architecture/ARCHITECTURE)** | How nself works |
| **[Deployment Guide](guides/Deployment)** | Go to production |
| **[Troubleshooting](guides/TROUBLESHOOTING)** | Fix common issues |

---

## Version 0.4.7 - Kubernetes Support

The current release adds full Kubernetes and container orchestration support:

### Kubernetes Commands
- **`nself k8s`** - Kubernetes operations (generate, apply, status, pods, logs, exec, rollout, scale)
- **`nself helm`** - Helm chart management (init, package, lint, install, upgrade, rollback)

### Key Features
- **Automatic Conversion** - docker-compose.yml → K8s manifests
- **Multi-Platform** - EKS, GKE, AKS, DOKS, LKE, k3s, minikube
- **Helm Charts** - Generate, package, and deploy Helm charts
- **Rolling Deployments** - Zero-downtime updates with HPA support

### Cloud Providers
26 cloud providers now supported with normalized sizing and one-command provisioning.

**[View Full Release Notes](releases/v0.4.7)**

---

## Minimal Path to Production

```bash
# 1. Local development (3 commands)
nself init && nself build && nself start

# 2. Design database
nself db schema scaffold saas
# Edit schema.dbml
nself db schema apply schema.dbml

# 3. To production (2 commands)
nself env create prod production
# Edit .environments/prod/server.json with your server
nself deploy prod
```

**5-6 commands** from blank folder to production.

---

## Service Summary

| Type | Count | Examples |
|------|-------|----------|
| **Required** | 4 | PostgreSQL, Hasura, Auth, Nginx |
| **Optional** | 7 | Redis, MinIO, Search, Mail, Functions, MLflow, Admin |
| **Monitoring** | 10 | Prometheus, Grafana, Loki, Tempo, Alertmanager, + exporters |
| **Custom** | Unlimited | Your services from 40+ templates |

---

## Links

- **[GitHub Repository](https://github.com/acamarata/nself)**
- **[Report Issues](https://github.com/acamarata/nself/issues)**
- **[Discussions](https://github.com/acamarata/nself/discussions)**
- **[Roadmap](releases/ROADMAP)**

---

<div align="center">

**Version 0.4.7** · **January 2026** · **[Changelog](releases/CHANGELOG)**

</div>
