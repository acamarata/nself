# ɳSelf Documentation

<div align="center">

**The Complete Self-Hosted Backend Platform**

[![Version](https://img.shields.io/badge/version-0.9.0-blue.svg)](releases/v0.9.0.md)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](../LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20WSL-lightgrey.svg)]()

*Deploy a complete backend in minutes, not days with ɳSelf.*

</div>

---

## Quick Navigation

| I want to... | Go to... |
|-------------|----------|
| Get started in 5 minutes | [Quick Start Guide](guides/Quick-Start.md) |
| Install nself | [Installation](guides/Installation.md) |
| Understand core concepts | [Architecture](architecture/ARCHITECTURE.md) |
| Look up a command | [Command Reference](commands/COMMANDS.md) |
| Configure my setup | [Configuration Guide](configuration/README.md) |
| Deploy to production | [Deployment Guide](deployment/README.md) |
| Fix a problem | [Troubleshooting](guides/TROUBLESHOOTING.md) |
| See examples | [Examples](examples/README.md) |
| Learn specific features | [Tutorials](tutorials/README.md) |

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

**[Learn More About Architecture](architecture/ARCHITECTURE.md)**

---

## Documentation Structure

### Getting Started

**Start here if you're new to ɳSelf**

- **[Quick Start](guides/Quick-Start.md)** - Get running in 5 minutes
- **[Installation](guides/Installation.md)** - Detailed installation guide
- **[Database Workflow](guides/DATABASE-WORKFLOW.md)** - DBML to production in one command
- **[Demo Setup](services/DEMO_SETUP.md)** - Full demo with 25 services
- **[FAQ](guides/FAQ.md)** - Frequently asked questions

### Core Concepts

**Understand how ɳSelf works**

- **[Architecture](architecture/ARCHITECTURE.md)** - System design and components
- **[Project Structure](architecture/PROJECT_STRUCTURE.md)** - File organization
- **[Build System](architecture/BUILD_ARCHITECTURE.md)** - How build works
- **[Services Overview](services/SERVICES.md)** - All available services

### CLI Reference

**Complete command documentation**

- **[All Commands](commands/COMMANDS.md)** - Overview of 150+ commands
- **[Core Commands](commands/README.md#core-commands)** - init, build, start, stop, status
- **[Database Commands](commands/DB.md)** - migrate, seed, mock, backup, schema, types
- **[Deployment Commands](commands/DEPLOY.md)** - SSH deployment, environments
- **[Performance Commands](commands/PERF.md)** - Profiling and benchmarking
- **[Plugin Commands](commands/PLUGIN.md)** - Plugin management
- **[Quick Reference](COMMAND-REFERENCE.md)** - Printable cheat sheet

### Configuration

**Configure your ɳSelf setup**

- **[Configuration Guide](configuration/README.md)** - Complete configuration overview
- **[Environment Variables](configuration/ENVIRONMENT-VARIABLES.md)** - All .env variables
- **[Start Command Options](configuration/START-COMMAND-OPTIONS.md)** - Startup configuration
- **[Custom Services](configuration/CUSTOM-SERVICES-ENV-VARS.md)** - CS_N variables
- **[Admin UI](configuration/Admin-UI.md)** - Web dashboard setup

### Deployment

**Deploy to production**

- **[Deployment Guide](deployment/README.md)** - Production deployment overview
- **[SSH Deployment](guides/Deployment.md)** - Zero-downtime deployment
- **[Environment Management](guides/ENVIRONMENTS.md)** - local, staging, production
- **[Security Guide](guides/SECURITY.md)** - Production security
- **[Backup & Recovery](guides/BACKUP_GUIDE.md)** - Database backups

### Guides & Tutorials

**Learn specific features and workflows**

- **[Database Workflow](guides/DATABASE-WORKFLOW.md)** - Design-first development
- **[Multi-App Setup](guides/MULTI_APP_SETUP.md)** - Multiple apps in one stack
- **[Service Communication](guides/SERVICE-TO-SERVICE-COMMUNICATION.md)** - Microservices patterns
- **[OAuth Setup](guides/OAUTH-SETUP.md)** - Social login configuration
- **[Real-Time Features](guides/REALTIME-FEATURES.md)** - Subscriptions and live data
- **[Organization Management](guides/ORGANIZATION-MANAGEMENT.md)** - Multi-tenancy
- **[Billing & Usage](guides/BILLING-AND-USAGE.md)** - Usage tracking and billing
- **[All Tutorials](tutorials/README.md)** - Step-by-step tutorials

### Services

**Available services and configuration**

- **[Services Overview](services/SERVICES.md)** - All available services
- **[Required Services](services/SERVICES_REQUIRED.md)** - Core 4: PostgreSQL, Hasura, Auth, Nginx
- **[Optional Services](services/SERVICES_OPTIONAL.md)** - 7 additional services
- **[Monitoring Bundle](services/MONITORING-BUNDLE.md)** - 10-service observability stack
- **[Custom Services](services/SERVICES_CUSTOM.md)** - Build from 40+ templates
- **[Service Templates](services/SERVICE_TEMPLATES.md)** - Template reference

### Plugins (v0.4.8)

**Extend ɳSelf with third-party integrations**

- **[Plugin Overview](plugins/index.md)** - Plugin system introduction
- **[Plugin Development](plugins/development.md)** - Creating custom plugins
- **[Stripe Plugin](plugins/stripe.md)** - Payment processing
- **[GitHub Plugin](plugins/github.md)** - Repository sync
- **[Shopify Plugin](plugins/shopify.md)** - E-commerce integration

### API Reference

**GraphQL and REST APIs**

- **[API Overview](api/README.md)** - API documentation
- **[GraphQL API](architecture/API.md)** - Hasura GraphQL reference
- **[Authentication API](api/AUTH-API.md)** - Auth endpoints

### Troubleshooting

**Fix common issues**

- **[Troubleshooting Guide](guides/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Doctor Command](commands/DOCTOR.md)** - Automated diagnostics
- **[Billing Troubleshooting](troubleshooting/BILLING-TROUBLESHOOTING.md)** - Billing issues
- **[White-Label Troubleshooting](troubleshooting/WHITE-LABEL-TROUBLESHOOTING.md)** - Branding issues

### Examples

**Real-world examples and code**

- **[Examples Index](examples/README.md)** - All examples
- **[Database Examples](examples/DATABASE-EXAMPLES.md)** - Schema patterns
- **[Deployment Examples](deployment/examples/README.md)** - Production setups
- **[Service Examples](examples/CUSTOM-SERVICE-EXAMPLES.md)** - Custom service patterns

---

## What's New in v0.9.0

### Multi-Tenant Platform

Complete multi-tenancy with billing, white-labeling, and tenant isolation:

```bash
# Initialize multi-tenancy
nself tenant init

# Create tenant
nself tenant create "Acme Corp" --slug acme --plan pro

# Billing management
nself tenant billing usage
nself tenant billing subscription upgrade pro

# Custom domains and SSL
nself tenant domains add app.example.com
nself tenant domains ssl app.example.com

# Brand customization
nself tenant branding set-colors --primary #0066cc
```

### OAuth Management

Comprehensive OAuth provider integration:

```bash
# Enable multiple providers
nself oauth enable --providers google,github,slack,microsoft

# Configure credentials
nself oauth config google --client-id=xxx --client-secret=xxx

# Test configuration
nself oauth test google
```

### File Storage & Uploads

Advanced file storage with thumbnails, virus scanning, and compression:

```bash
# Upload with all features
nself storage upload photo.jpg --all-features

# Generate GraphQL integration
nself storage graphql-setup

# Manage uploads
nself storage list users/123/
```

**[View v0.9.0 Release Notes](releases/v0.9.0.md)**

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

**[View All Templates](services/SERVICE_TEMPLATES.md)**

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

**[View Complete Deployment Guide](deployment/README.md)**

---

## Service Summary

| Type | Count | Examples |
|------|-------|----------|
| **Required** | 4 | PostgreSQL, Hasura, Auth, Nginx |
| **Optional** | 7 | Redis, MinIO, Search, Mail, Functions, MLflow, Admin |
| **Monitoring** | 10 | Prometheus, Grafana, Loki, Tempo, Alertmanager, + exporters |
| **Plugins** | 3+ | Stripe, GitHub, Shopify, and more coming |
| **Custom** | Unlimited | Your services from 40+ templates |

**[View Services Overview](services/SERVICES.md)**

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
| v0.4.3 | Jan 19, 2026 | Deployment Pipeline |

**[View Roadmap](releases/ROADMAP.md)** | **[View Changelog](releases/CHANGELOG.md)**

---

## Security

nself has been thoroughly audited for security vulnerabilities. The codebase is safe for production use.

| Category | Status |
|----------|--------|
| Hardcoded Credentials | ✅ PASS |
| API Keys & Tokens | ✅ PASS |
| Command Injection | ✅ PASS |
| SQL Injection | ✅ PASS |
| Docker Security | ✅ PASS |
| Git History | ✅ PASS |

**[View Complete Security Audit](security/SECURITY-AUDIT.md)**

---

## Contributing

We welcome contributions! Whether it's bug reports, feature requests, documentation improvements, or code contributions.

- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute
- **[Development Setup](contributing/DEVELOPMENT.md)** - Dev environment
- **[Cross-Platform Compatibility](contributing/CROSS-PLATFORM-COMPATIBILITY.md)** - Bash 3.2+ requirements

---

## Links

- **GitHub**: [github.com/acamarata/nself](https://github.com/acamarata/nself)
- **Issues**: [Report bugs](https://github.com/acamarata/nself/issues)
- **Discussions**: [Ask questions](https://github.com/acamarata/nself/discussions)
- **Plugin Registry**: [plugins.nself.org](https://plugins.nself.org)
- **Roadmap**: [Future plans](releases/ROADMAP.md)

---

<div align="center">

**Version 0.9.0** · **January 2026** · **[Changelog](releases/CHANGELOG.md)**

*ɳSelf - The complete self-hosted backend platform*

</div>
