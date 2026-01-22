# nself Development Roadmap

## Quick Navigation
[Released](#released) | [Next (v0.4.2)](#next-v042) | [Planned (v0.4.x)](#planned-v04x-series) | [Future (v0.5+)](#future-v05)

## Current Status Summary
- **✅ v0.4.1 (Current)**: Platform compatibility fixes - stable release
- **📋 v0.4.2 (Next)**: Service & Monitoring management commands
- **📋 v0.4.3**: Database, Backup & Production features
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
**Focus**: Service & Monitoring Management Commands

This release adds **6 new commands** for managing optional services and monitoring.

### New Commands

#### Service Management
| Command | Purpose |
|---------|---------|
| `nself email` | Email service configuration, testing, provider switching |
| `nself search` | Search engine management (MeiliSearch, Typesense, Sonic) |
| `nself functions` | Serverless functions deployment, logs, testing |
| `nself mlflow` | MLflow experiment tracking, model management |

#### Monitoring Management
| Command | Purpose |
|---------|---------|
| `nself metrics` | Prometheus/metrics configuration and management |
| `nself monitor` | Dashboard access (Grafana, Prometheus, Alertmanager) |

### Enhancements
- Email provider setup wizards (MailPit, SMTP, SendGrid, Postmark)
- Search engine indexing and testing tools
- Functions hot-reload and live logs
- MLflow experiment browser CLI
- Grafana dashboard management
- Alert rule templates
- Metrics export and analysis

---

## Planned (v0.4.x Series)

### v0.4.3 - Database, Backup & Production
**Target**: Q2-Q3 2026

This consolidated release combines database operations, backup/restore, and production features.

#### New Commands
| Command | Purpose |
|---------|---------|
| `nself db` | Database migrations, backups, optimization, queries |
| `nself backup` | Create, schedule, and manage backups |
| `nself restore` | Restore from backups, point-in-time recovery |
| `nself prod` | Production environment configuration |
| `nself deploy` | SSH-based deployment to VPS/servers |

#### Enhancements
- **Database**
  - Schema migration tools
  - Performance analysis and query optimization
  - Connection pooling management
  - Database cloning for dev/staging

- **Backup & Restore**
  - S3-compatible backup storage (MinIO, AWS S3)
  - Automated backup scheduling (cron)
  - Point-in-time recovery
  - Backup encryption and verification

- **Production**
  - Security hardening wizard
  - Environment-specific configurations
  - GitHub webhook integration
  - Zero-downtime deployments
  - Health check endpoints

---

### v0.4.4 - Scaling & Advanced Features
**Target**: Q3-Q4 2026

Final release in the 0.4.x series before major 0.5 features.

#### New Commands
| Command | Purpose |
|---------|---------|
| `nself scale` | Horizontal/vertical scaling management |
| `nself perf` | Performance profiling and optimization |
| `nself migrate` | Cross-environment migration tools |

#### Enhancements
- Auto-scaling configuration
- Load balancing setup (nginx upstream)
- Resource optimization recommendations
- Performance benchmarking
- Multi-environment sync
- Configuration drift detection

---

## Future (v0.5+)
**Status**: 🔮 Long-term Vision | **Timeline**: 2027+
**Focus**: Advanced Features & Enterprise Capabilities

### Cloud & Orchestration (v0.5)
- **Kubernetes Support** - Generate K8s manifests, Helm charts
- **Container Orchestration** - Docker Swarm, Nomad support
- **Cloud Providers** - Native integrations (AWS, GCP, Azure, DigitalOcean)
- **Multi-Region** - Geographic distribution and failover
- **Terraform Integration** - Infrastructure as code exports

### Enterprise Features (v0.6)
- **Security & Compliance** - SSO/SAML, RBAC, audit logging, compliance templates
- **Advanced Database** - Multi-region replication, read replicas, automatic failover
- **High Availability** - Multi-node clustering, load balancing, circuit breakers
- **Developer Tools** - Code generation, API testing, CI/CD templates
- **Team Management** - Multi-user access, permissions, activity logs

### Innovation (v0.7+)
- **AI/ML Platform** - Vector database (pgvector), LLM integration, model serving
- **Edge Computing** - Edge functions, CDN integration, offline-first sync
- **Multi-Tenancy** - True multi-tenant architecture with isolation
- **Plugin System** - Community-contributed extensions and integrations
- **GUI Application** - Desktop app for visual management

---

## Development Principles

1. **Stability First** - Never break existing features
2. **Smart Defaults** - Everything works out of the box
3. **No Lock-in** - Standard Docker/PostgreSQL/GraphQL
4. **Progressive Disclosure** - Advanced features stay hidden until needed
5. **Auto-Fix** - Detect and resolve problems automatically
6. **Offline-First** - Works without internet connection
7. **Security by Default** - Production-ready security out of the box
8. **Cross-Platform** - Works on macOS, Linux, WSL (Bash 3.2+)

---

## Release Timeline

| Version | Status | Focus | Target |
|---------|--------|-------|--------|
| v0.4.0 | ✅ Released | Production-Ready Core | Oct 2025 |
| v0.4.1 | ✅ Released | Platform Compatibility | Jan 2026 |
| **v0.4.2** | 📋 **Next** | Service & Monitoring | Q1-Q2 2026 |
| v0.4.3 | 📋 Planned | Database, Backup, Production | Q2-Q3 2026 |
| v0.4.4 | 📋 Planned | Scaling & Advanced | Q3-Q4 2026 |
| v0.5.0 | 🔮 Future | Kubernetes & Cloud | 2027 |
| v0.6.0 | 🔮 Future | Enterprise Features | 2027+ |

---

## Command Summary by Release

### Currently Available (v0.4.1)
```
init, build, start, stop, restart, reset, clean, restore
status, logs, exec, urls, doctor, version, update, help
ssl, trust, admin
```

### Coming in v0.4.2
```
email, search, functions, mlflow, metrics, monitor
```

### Coming in v0.4.3
```
db, backup, restore, prod, deploy
```

### Coming in v0.4.4
```
scale, perf, migrate
```

**Total Commands After v0.4.4**: 30+

---

## Contributing

### Priority Areas
1. Test v0.4.1 in production environments
2. Report bugs and edge cases
3. Documentation improvements
4. Community feedback on roadmap priorities
5. Performance benchmarks

### How to Contribute
- **GitHub**: [github.com/acamarata/nself](https://github.com/acamarata/nself)
- **Issues**: [Report Bugs](https://github.com/acamarata/nself/issues)
- **Discussions**: [Feature Requests & Ideas](https://github.com/acamarata/nself/discussions)
- **Testing**: Help test new features in development

---

*This roadmap reflects actual implemented features and realistic future plans. Updated regularly based on development progress and community feedback.*

*Last Updated: January 22, 2026*
