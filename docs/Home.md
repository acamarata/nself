# nself

<div align="center">

**The Complete Self-Hosted Backend Platform**

[![Version](https://img.shields.io/badge/version-0.9.0-blue.svg)](releases/v0.9.0.md)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](../LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey.svg)]()
[![Status](https://img.shields.io/badge/status-stable-brightgreen.svg)]()

*Deploy a complete backend in minutes, not days.*

**[Quick Start](guides/Quick-Start.md)** | **[Installation](guides/Installation.md)** | **[Documentation](README.md)** | **[GitHub](https://github.com/acamarata/nself)**

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

**[View Full Quick Start Guide](guides/Quick-Start.md)**

---

## Quick Navigation

| I want to... | Go to... |
|-------------|----------|
| Get started in 5 minutes | **[Quick Start](guides/Quick-Start.md)** |
| Install nself | **[Installation](guides/Installation.md)** |
| Understand core concepts | **[Architecture](architecture/ARCHITECTURE.md)** |
| Look up a command | **[Command Reference](commands/COMMANDS.md)** |
| Configure my setup | **[Configuration](configuration/README.md)** |
| Deploy to production | **[Deployment](deployment/README.md)** |
| Fix a problem | **[Troubleshooting](guides/TROUBLESHOOTING.md)** |
| See examples | **[Examples](examples/README.md)** |
| Learn specific features | **[Tutorials](tutorials/README.md)** |

---

## What is nself?

nself is a complete self-hosted Backend-as-a-Service platform that provides all the features of commercial services like Supabase, Nhost, or Firebase, but runs entirely on your own infrastructure.

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

**[View All Templates](services/SERVICE_TEMPLATES.md)**

---

### Multi-Tenant Platform (v0.9.0)

Complete multi-tenancy, billing, and white-labeling:

```bash
# Tenant management
nself tenant create "Acme Corp" --slug acme --plan pro

# Billing & subscriptions
nself tenant billing usage
nself tenant billing subscription upgrade

# Custom domains & branding
nself tenant domains add app.example.com
nself tenant branding set-colors --primary #0066cc
```

### OAuth & Storage (v0.9.0)

**OAuth Integration:**
- **[OAuth Management](commands/OAUTH.md)** - Google, GitHub, Microsoft, Slack
- Multiple provider support with easy configuration

**File Storage:**
- **[Storage System](commands/STORAGE.md)** - Uploads, thumbnails, virus scanning
- GraphQL integration generation

### Plugin System (v0.4.8)

**Available Plugins:**
- **[Stripe](plugins/stripe.md)** - Payment processing
- **[GitHub](plugins/github.md)** - Repository sync
- **[Shopify](plugins/shopify.md)** - E-commerce

**[View Plugin Documentation](plugins/README.md)**

---

## Documentation Structure

### 📚 Getting Started
- **[Quick Start](guides/Quick-Start.md)** - Get running in 5 minutes
- **[Installation](guides/Installation.md)** - Detailed installation
- **[Database Workflow](guides/DATABASE-WORKFLOW.md)** - DBML to production
- **[FAQ](guides/FAQ.md)** - Frequently asked questions

### 🛠️ Core Concepts
- **[Architecture](architecture/ARCHITECTURE.md)** - System design
- **[Services Overview](services/SERVICES.md)** - Available services
- **[Project Structure](architecture/PROJECT_STRUCTURE.md)** - File organization

### 💻 CLI Reference
- **[All Commands](commands/COMMANDS.md)** - 150+ commands
- **[Quick Reference](quick-reference/COMMAND-REFERENCE.md)** - Printable cheat sheet
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
- **[API Overview](api/README.md)** - All APIs
- **[GraphQL API](architecture/API.md)** - Hasura GraphQL

---

## What's New in v0.9.0

### Multi-Tenant Platform

Complete platform for building SaaS applications:

- **Tenant Management** - Create, suspend, activate, delete tenants
- **Billing Integration** - Stripe-based usage tracking and invoicing
- **White-Labeling** - Custom domains, branding, email templates, themes
- **OAuth Providers** - Google, GitHub, Microsoft, Slack integration
- **File Storage** - Advanced upload pipeline with thumbnails and virus scanning
- **Member Management** - Role-based access control per tenant

**[v0.9.0 Documentation](releases/v0.9.0.md)** | **[Release Notes](releases/v0.9.0.md)**

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
| **v0.9.0** | Jan 30, 2026 | Multi-Tenant Platform (current) |
| v0.4.8 | Jan 24, 2026 | Plugin System & Registry |
| v0.4.7 | Jan 23, 2026 | Infrastructure Everywhere |
| v0.4.6 | Jan 22, 2026 | Scaling & Performance |
| v0.4.5 | Jan 21, 2026 | Provider Support |
| v0.4.4 | Jan 20, 2026 | Database Tools |

**[Roadmap](releases/ROADMAP.md)** | **[Changelog](releases/CHANGELOG.md)**

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
- **[Roadmap](releases/ROADMAP.md)**

---

## Contributing

We welcome contributions! Whether it's bug reports, feature requests, documentation improvements, or code contributions.

- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute
- **[Development Setup](contributing/README.md)** - Dev environment
- **[Cross-Platform](contributing/CROSS-PLATFORM-COMPATIBILITY.md)** - Compatibility requirements

---

<div align="center">

**Version 0.9.0** · **January 2026** · **[Full Documentation](README.md)**

*nself - The complete self-hosted backend platform*

</div>
