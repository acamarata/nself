# ɳSelf

<div align="center">

**The Complete Self-Hosted Backend Platform**

[![Version](https://img.shields.io/badge/version-0.9.8-blue.svg)](releases/v0.9.8.md)
[![License](https://img.shields.io/badge/license-Personal%20Free%20%7C%20Commercial-green.svg)](LICENSE.md)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey.svg)]()
[![Status](https://img.shields.io/badge/status-stable-brightgreen.svg)]()

*Deploy a complete backend in minutes, not days with ɳSelf.*

**[Quick Start](getting-started/Quick-Start.md)** | **[Installation](getting-started/Installation.md)** | **[Commands](COMMANDS.md)** | **[Documentation](README.md)** | **[GitHub](https://github.com/acamarata/nself)**

</div>

---

## What's New in v0.9.8

> **Production Ready** - nself v0.9.8 is stable and production-grade. [View Release Notes](releases/v0.9.8.md)

- **80% Test Coverage** - 700+ tests across unit, integration, and E2E
- **Complete Documentation** - Every command and feature fully documented
- **UX Polish** - Standardized errors, progress indicators, input validation
- **Performance** - Build caching, batched operations, parallel execution
- **Production Features** - Health endpoints, graceful shutdown, automated backups
- **Security Hardened** - Full audit pass, SQL injection remediation, CSP headers

**Next:** v0.9.9 (QA freeze) → v1.0.0 LTS | **[Full Roadmap](releases/ROADMAP.md)**

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

**[View Full Quick Start Guide](getting-started/Quick-Start.md)**

---

## Quick Navigation

| I want to... | Go to... |
|-------------|----------|
| Get started in 5 minutes | **[Quick Start](getting-started/Quick-Start.md)** |
| Install nself | **[Installation](getting-started/Installation.md)** |
| Understand core concepts | **[Architecture](architecture/ARCHITECTURE.md)** |
| Look up a command | **[Commands Landing](COMMANDS.md)** |
| View command runtime matrix | **[SPORT Command Matrix](commands/SPORT-COMMAND-MATRIX.md)** |
| Configure my setup | **[Configuration](configuration/README.md)** |
| Deploy to production | **[Deployment](deployment/README.md)** |
| Fix a problem | **[Troubleshooting](guides/TROUBLESHOOTING.md)** |
| See examples | **[Examples](examples/README.md)** |
| Learn specific features | **[Tutorials](tutorials/README.md)** |

---

## SPORT Top-Level Categories

| Category | Landing Page | Purpose |
|---|---|---|
| Getting Started | [getting-started/README.md](getting-started/README.md) | First install and first project |
| Commands | [COMMANDS.md](COMMANDS.md) | Top-level command landing and index |
| Command Matrix | [commands/SPORT-COMMAND-MATRIX.md](commands/SPORT-COMMAND-MATRIX.md) | Full runtime command coverage, including wrappers |
| Architecture | [architecture/INDEX.md](architecture/INDEX.md) | System design and structure |
| Configuration | [configuration/README.md](configuration/README.md) | `.env`, secrets, vault, startup options |
| Deployment | [deployment/README.md](deployment/README.md) | Staging, production, infra workflows |
| Services | [services/INDEX.md](services/INDEX.md) | Required, optional, monitoring, custom |
| Security | [security/INDEX.md](security/INDEX.md) | Security guides and audits |
| Testing & QA | [testing/README.md](testing/README.md) | Test strategy and quality artifacts |
| Releases | [releases/INDEX.md](releases/INDEX.md) | Roadmap, release notes, changelog |

---

## What is ɳSelf?

ɳSelf is a complete self-hosted Backend-as-a-Service platform that provides all the features of commercial services like Supabase, Nhost, or Firebase, but runs entirely on your own infrastructure.

```
┌─────────────────────────────────────────────────────────────────────┐
│                          YOUR APPLICATION                            │
├─────────────────────────────────────────────────────────────────────┤
│   Frontend (React, Vue, Next.js, etc.)                              │
│   ↓ GraphQL queries and mutations                                   │
├─────────────────────────────────────────────────────────────────────┤
│                              ɳSelf                                   │
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
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              PLUGINS (v0.4.8)                             │      │
│   │   Stripe · GitHub · Shopify · and more                    │      │
│   └──────────────────────────────────────────────────────────┘      │
├─────────────────────────────────────────────────────────────────────┤
│   Runs on: Docker Compose · Any Cloud · Any Server · Laptop         │
└─────────────────────────────────────────────────────────────────────┘
```

**[Learn More](architecture/ARCHITECTURE.md)**

---

## Key Features

### Database-First Development
Design your schema visually, generate everything automatically.

```bash
nself db schema scaffold saas       # Start with a template
# Edit schema.dbml at dbdiagram.io
nself db schema apply schema.dbml   # Creates tables + mock data + users
```

**[Database Workflow Guide](guides/DATABASE-WORKFLOW.md)**

---

### Environment-Aware Safety
Different behavior for local, staging, and production.

| Command | Local | Staging | Production |
|---------|-------|---------|------------|
| `db mock` | Works | Works | Blocked |
| `db reset` | Works | Confirm | Blocked |
| `db seed users` | Mock users | QA users | Only explicit users |

**[Environment Management](guides/ENVIRONMENTS.md)**

---

### Smart Defaults Everywhere
Zero configuration required for common cases.

```bash
nself db migrate up     # Just works - no flags needed
nself db backup         # Auto-names: myapp_local_20260122.sql
nself db seed           # Knows your environment
nself db types          # Generates TypeScript by default
```

**[Commands Reference](commands/README.md)**

---

### 40+ Service Templates
Build custom services in any language.

**JavaScript/TypeScript:** Express, Fastify, NestJS, Hono, BullMQ
**Python:** FastAPI, Flask, Django, Celery
**Go:** Gin, Fiber, Echo, gRPC
**Others:** Rust, Java, PHP, Ruby, C#, Elixir

**[View All Templates](reference/SERVICE_TEMPLATES.md)**

---

### Multi-Tenant Platform (v0.9.0)

Complete multi-tenancy, billing, and white-labeling:

```bash
# Tenant management
nself tenant create "Acme Corp" --slug acme --plan pro

# Billing & subscriptions (v0.9.6: was "nself billing")
nself tenant billing usage
nself tenant billing subscription upgrade

# Organization management (v0.9.6: was "nself org")
nself tenant org list

# Custom domains & branding
nself tenant domains add app.example.com
nself tenant branding set-colors --primary #0066cc
```

> **v0.9.6 Update:** Commands consolidated under logical groupings. Old commands like `nself billing` and `nself org` are now `nself tenant billing` and `nself tenant org`. All old commands still work with deprecation warnings.

### OAuth & Storage (v0.9.0)

**OAuth Integration:**
- **[OAuth Management](commands/OAUTH.md)** - Google, GitHub, Microsoft, Slack
- Multiple provider support with easy configuration
- Commands: `nself auth oauth` (v0.9.6: consolidated from `nself oauth`)

**File Storage:**
- **[Storage System](commands/storage.md)** - Uploads, thumbnails, virus scanning
- GraphQL integration generation
- Commands: `nself service storage` (v0.9.6: consolidated from `nself storage`)

### Plugin System (v0.4.8)

**Available Plugins:**
- **[Stripe](plugins/stripe.md)** - Payment processing
- **[GitHub](plugins/github.md)** - Repository sync
- **[Shopify](plugins/shopify.md)** - E-commerce

**[View Plugin Documentation](plugins/README.md)**

---

## Documentation Structure

### 📚 Getting Started
- **[Quick Start](getting-started/Quick-Start.md)** - Get running in 5 minutes
- **[Installation](getting-started/Installation.md)** - Detailed installation
- **[Database Workflow](guides/DATABASE-WORKFLOW.md)** - DBML to production
- **[FAQ](getting-started/FAQ.md)** - Frequently asked questions

### 🛠️ Core Concepts
- **[Architecture](architecture/ARCHITECTURE.md)** - System design
- **[Services Overview](services/SERVICES.md)** - Available services
- **[Project Structure](architecture/PROJECT_STRUCTURE.md)** - File organization

### 💻 CLI Reference
- **[All Commands](commands/COMMANDS.md)** - Command reference
- **[SPORT Command Matrix](commands/SPORT-COMMAND-MATRIX.md)** - Runtime command coverage
- **[Quick Reference](reference/COMMAND-REFERENCE.md)** - Printable cheat sheet
- **[Core Commands](commands/README.md#core-commands)** - Essential commands

### ⚙️ Configuration
- **[Configuration Guide](configuration/README.md)** - Complete overview
- **[Environment Variables](configuration/ENVIRONMENT-VARIABLES.md)** - All variables
- **[Custom Services](configuration/CUSTOM-SERVICES-ENV-VARS.md)** - CS_N configuration

### 🚀 Deployment
- **[Deployment Guide](deployment/README.md)** - Production deployment
- **[SSH Deployment](guides/Deployment.md)** - Zero-downtime deployment
- **[Cloud Providers](deployment/CLOUD-PROVIDERS.md)** - 26+ providers

### 📖 Guides & Tutorials
- **[All Guides](guides/README.md)** - Usage guides
- **[All Tutorials](tutorials/README.md)** - Step-by-step tutorials
- **[Examples](examples/README.md)** - Real-world examples

### 🔌 Plugins
- **[Plugin Overview](plugins/README.md)** - Plugin system
- **[Plugin Development](plugins/development.md)** - Creating plugins

### 🔐 Security
- **[Security Overview](security/README.md)** - Security features
- **[Security Audit](security/SECURITY-AUDIT.md)** - Audit results

### 📡 API Reference
- **[API Overview](reference/api/README.md)** - All APIs
- **[GraphQL API](architecture/API.md)** - Hasura GraphQL

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

**[Complete Deployment Guide](deployment/README.md)**

---

## Service Summary

| Type | Count | Examples |
|------|-------|----------|
| **Required** | 4 | PostgreSQL, Hasura, Auth, Nginx |
| **Optional** | 7 | Redis, MinIO, Search, Mail, Functions, MLflow, Admin |
| **Monitoring** | 10 | Prometheus, Grafana, Loki, Tempo, Alertmanager, + exporters |
| **Plugins** | 3+ | Stripe, GitHub, Shopify, and more coming |
| **Custom** | Unlimited | Your services from 40+ templates |

**[Services Overview](services/SERVICES.md)**

---

## Version History

| Version | Date | Focus |
|---------|------|-------|
| **v0.9.8** | Feb 2026 | Production Ready (current) |
| v0.9.7 | Jan 31, 2026 | Security & CI/CD Complete |
| v0.9.6 | Jan 30, 2026 | Command Consolidation |
| v0.9.5 | Jan 30, 2026 | Feature Parity & Security |
| v0.9.0 | Jan 30, 2026 | Multi-Tenant Platform |
| v0.4.8 | Jan 24, 2026 | Plugin System & Registry |
| v0.4.7 | Jan 23, 2026 | Infrastructure Everywhere |
| v0.4.6 | Jan 22, 2026 | Scaling & Performance |

**[Full Roadmap](releases/ROADMAP.md)** | **[Changelog](releases/CHANGELOG.md)** | **[v1.0 Plan](releases/v1.0.0-PLAN.md)**

---

## Security

nself has been thoroughly audited for security vulnerabilities.

| Category | Status |
|----------|--------|
| Hardcoded Credentials | ✅ PASS |
| API Keys & Tokens | ✅ PASS |
| Command Injection | ✅ PASS |
| SQL Injection | ✅ PASS |
| Docker Security | ✅ PASS |
| Git History | ✅ PASS |

**[View Security Audit](security/SECURITY-AUDIT.md)**

---

## Links

- **[GitHub Repository](https://github.com/acamarata/nself)**
- **[Report Issues](https://github.com/acamarata/nself/issues)**
- **[Discussions](https://github.com/acamarata/nself/discussions)**
- **[Plugin Registry](https://plugins.nself.org)**
- **[Commands](COMMANDS.md)**
- **[Roadmap](releases/ROADMAP.md)**
- **[Changelog](releases/CHANGELOG.md)**
- **[License](LICENSE.md)**
- **[Root Structure Policy](ROOT-STRUCTURE-POLICY.md)**

---

## Contributing

We welcome contributions! Whether it's bug reports, feature requests, documentation improvements, or code contributions.

- **[Contributing Guide](contributing/CONTRIBUTING.md)** - How to contribute
- **[Development Setup](contributing/README.md)** - Dev environment
- **[Cross-Platform](contributing/CROSS-PLATFORM-COMPATIBILITY.md)** - Compatibility requirements

---

<div align="center">

**Version 0.9.8** · **February 2026**

*ɳSelf - The complete self-hosted backend platform*

**Quick Links:** [Quick Start](getting-started/Quick-Start.md) · [Commands](COMMANDS.md) · [Roadmap](releases/ROADMAP.md)

**Resources:** [Full Documentation](README.md) · [Changelog](releases/CHANGELOG.md) · [Security Audit](security/SECURITY-AUDIT.md) · [License](LICENSE.md)

**Community:** [GitHub](https://github.com/acamarata/nself) · [Issues](https://github.com/acamarata/nself/issues) · [Discussions](https://github.com/acamarata/nself/discussions)

**Path to v1.0:** [v0.9.9 Plan](releases/v0.9.9-PLAN.md) → [v1.0 Plan](releases/v1.0.0-PLAN.md)

</div>
