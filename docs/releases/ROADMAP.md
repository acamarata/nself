# nself Development Roadmap

## Quick Navigation
[Released](#released) | [Next (v0.4.2)](#next-v042) | [Future (v0.5+)](#future-v05)

## Current Status Summary
- **✅ v0.4.1 (Current)**: Platform compatibility fixes - stable release
- **📋 v0.4.2 (Next)**: Service & Monitoring management commands
- **🔮 v0.5+**: Advanced features (Kubernetes, multi-cloud, enterprise)

---

## Vision
Transform nself from a powerful CLI tool into a complete self-hosted backend platform that rivals commercial BaaS offerings (Supabase, Nhost, Firebase) while maintaining simplicity, control, and the ability to run anywhere.

---

## Released

### v0.4.1 - Platform Compatibility Fixes
**Status**: ✅ Released | **Release Date**: January 22, 2026

Bug fix release addressing 5 critical platform compatibility issues:

- ✅ **Bash 3.2 Compatibility** - Fixed array declaration syntax
- ✅ **Cross-Platform sed** - Fixed in-place editing for macOS/Linux
- ✅ **Cross-Platform stat** - Fixed file stat commands
- ✅ **Portable timeout** - Added guards for timeout command
- ✅ **Portable output** - Converted echo -e to printf

**22 files fixed** across the codebase. Full details: [v0.4.1 Release Notes](./v0.4.1.md)

---

### v0.4.0 - Production-Ready Core
**Status**: ✅ Released | **Release Date**: October 13, 2025

v0.4.0 represents the **stable, production-ready release** of nself with all core features complete, tested, and ready for real-world use.

### Core Features (Fully Functional)
- **✅ Full Nhost Stack** - PostgreSQL with 60+ extensions, Hasura GraphQL, Auth, Storage
- **✅ Admin UI** - Web-based monitoring dashboard at localhost:3100
- **✅ 40+ Service Templates** - Production-ready microservice templates across 10 languages
- **✅ SSL Management** - Automatic certificates with mkcert (Let's Encrypt ready)
- **✅ Environment Management** - Multi-environment configuration with smart defaults
- **✅ Auto-Fix System** - Smart defaults and automatic problem resolution
- **✅ Docker Compose** - Battle-tested container orchestration
- **✅ Custom Services (CS_N)** - Easy microservice configuration pattern
- **✅ Optional Services** - Redis, MinIO, MailPit, MeiliSearch, MLflow, Functions
- **✅ Monitoring Bundle** - Complete 10-service monitoring stack (Prometheus, Grafana, Loki, etc.)

### Available Commands (15 Core)
**Core**: init, build, start, stop, restart, reset, clean, restore
**Status**: status, logs, exec, urls, doctor, version, update, help
**Management**: ssl, trust, admin

### Service Templates (40 Total)
- **JavaScript/TypeScript (19)**: Node.js, Express, Fastify, NestJS, Hono, Socket.IO, BullMQ, Temporal, Bun, Deno, tRPC
- **Python (7)**: Flask, FastAPI, Django REST, Celery, Ray, AI Agents
- **Go (4)**: Gin, Echo, Fiber, gRPC
- **Other (10)**: Rust, Java, C#, C++, Ruby, Elixir, PHP, Kotlin, Swift

[View Full Release Notes](./v0.4.0.md)

---

## Next (v0.4.2)
**Status**: 📋 Planned | **Target**: Q1-Q2 2026
**Focus**: Service Management & Monitoring Commands

### New Commands

#### Service Management
- `nself email` - Email service configuration and testing
- `nself search` - Search engine management (MeiliSearch, Typesense, etc.)
- `nself functions` - Serverless functions management
- `nself mlflow` - MLflow ML experiment tracking management

#### Monitoring Management
- `nself metrics` - Complete monitoring stack management
- `nself monitor` - Access monitoring dashboards (Grafana, Prometheus, etc.)

### Enhancements
- Enhanced email provider configuration
- Search engine setup wizards
- Functions deployment and testing
- MLflow integration improvements
- Monitoring stack configuration wizards
- Dashboard management
- Alert rule templates
- Metrics export and analysis

---

## Planned (v0.4.x Series)

### v0.4.3 - Database Operations
**Target**: Q2 2026

**New Commands**:
- `nself db` - Database operations (migrations, backups, optimization)

**Enhancements**:
- Schema migration tools
- Database performance analysis
- Query optimization recommendations
- Connection pooling management

### v0.4.4 - Backup & Restore
**Target**: Q3 2026

**New Commands**:
- `nself backup` - Create and manage backups
- `nself rollback` - Rollback to previous state

**Enhancements**:
- S3-compatible backup storage
- Automated backup scheduling
- Point-in-time recovery
- Backup encryption

### v0.4.5 - Production Features
**Target**: Q3 2026

**New Commands**:
- `nself prod` - Configure for production environments
- `nself deploy` - SSH-based deployment automation

**Enhancements**:
- Production security hardening
- VPS deployment automation
- GitHub webhook integration
- Zero-downtime deployments

### v0.4.6 - Scaling & Performance
**Target**: Q4 2026

**New Commands**:
- `nself scale` - Horizontal and vertical scaling management

**Enhancements**:
- Auto-scaling configuration
- Load balancing setup
- Resource optimization
- Performance monitoring

---

## Future (v0.5+)
**Status**: 🔮 Long-term Vision | **Timeline**: 2027+
**Focus**: Advanced Features & Enterprise Capabilities

### Cloud & Orchestration (v0.5)
- **Kubernetes Support** - Generate K8s manifests, Helm charts
- **Container Orchestration** - Docker Swarm, Nomad support
- **Cloud Providers** - Native integrations (AWS, GCP, Azure, DigitalOcean)
- **Multi-Region** - Geographic distribution and failover

### Enterprise Features (v0.6)
- **Security & Compliance** - SSO/SAML, RBAC, audit logging, compliance templates
- **Advanced Database** - Multi-region replication, read replicas, automatic failover
- **High Availability** - Multi-node clustering, load balancing
- **Developer Tools** - Code generation, API testing, CI/CD templates

### Innovation (v0.7+)
- **AI/ML Platform** - Vector database, LLM integration, model serving
- **Edge Computing** - Edge functions, CDN integration, offline-first
- **Multi-Tenancy** - True multi-tenant architecture with isolation
- **Plugin System** - Community-contributed extensions

---

## Development Principles

1. **Stability First** - Never break existing features
2. **Smart Defaults** - Everything works out of the box
3. **No Lock-in** - Standard Docker/PostgreSQL/GraphQL
4. **Progressive Disclosure** - Advanced features stay hidden
5. **Auto-Fix** - Detect and resolve problems automatically
6. **Offline-First** - Works without internet
7. **Security by Default** - Production-ready security
8. **Cross-Platform** - Works on macOS, Linux, WSL

---

## Release Timeline

| Version | Status | Focus | Timeline |
|---------|--------|-------|----------|
| [v0.4.0](#v040---production-ready-core) | ✅ Released | Production-Ready Core | Oct 2025 |
| [v0.4.1](#v041---platform-compatibility-fixes) | ✅ Released | Platform Compatibility | Jan 2026 |
| [v0.4.2](#next-v042) | 📋 Planned | Service & Monitoring | Q1-Q2 2026 |
| [v0.4.3](#v043---database-operations) | 📋 Planned | Database Tools | Q2 2026 |
| [v0.4.4](#v044---backup--restore) | 📋 Planned | Backup & Restore | Q3 2026 |
| [v0.4.5](#v045---production-features) | 📋 Planned | Production & Deploy | Q3 2026 |
| [v0.4.6](#v046---scaling--performance) | 📋 Planned | Scaling | Q4 2026 |
| [v0.5+](#future-v05) | 🔮 Future | Advanced Features | 2027+ |

---

## Contributing

### Priority Areas for v0.4.x
1. Test v0.4.1 in production environments
2. Report bugs and edge cases
3. Documentation improvements
4. Community feedback on roadmap priorities
5. Performance benchmarks

### How to Contribute
- **GitHub**: [github.com/acamarata/nself](https://github.com/acamarata/nself)
- **Issues**: [Report Bugs](https://github.com/acamarata/nself/issues)
- **Discussions**: [Feature Requests & Ideas](https://github.com/acamarata/nself/discussions)
- **Testing**: Help test v0.4.x features in development

---

*This roadmap reflects actual implemented features and realistic future plans. Updated regularly based on development progress and community feedback.*

*Last Updated: January 22, 2026*
