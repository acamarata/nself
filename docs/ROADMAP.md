# nself Development Roadmap

## Quick Navigation
[Released (v0.3.9)](#released-v039) | [Next (v0.4.0)](#next-v040) | [Beyond](#beyond-future)

---

## Vision
Transform nself from a powerful CLI tool into a complete self-hosted backend platform that rivals commercial BaaS offerings (Supabase, Nhost, Firebase) while maintaining simplicity, control, and the ability to run anywhere.

---

## Released (v0.3.9)
**Status**: ✅ Released | **Release Date**: September 3, 2024

### Core Features Available Now
- **Full Nhost Stack** - PostgreSQL with 60+ extensions, Hasura GraphQL, Auth, Storage
- **Admin UI** - Web-based monitoring dashboard at localhost:3100
- **40+ Service Templates** - Production-ready microservice templates across 10 languages
- **Email System** - 16+ providers with MailPit for dev, SMTP testing
- **SSL Management** - Automatic certificates with mkcert and Let's Encrypt ready
- **Backup System** - S3 support, scheduling, incremental backups
- **Environment Cascade** - Multi-environment configuration (.env, .env.dev, .env.staging, .env.prod)
- **Auto-Fix System** - Smart defaults and automatic problem resolution
- **Docker Compose** - Simple container orchestration
- **Custom Services (CS_N)** - Easy microservice configuration pattern

### Service Templates (40 Total)
- **JavaScript/TypeScript (19)**: Node.js, Express, Fastify, NestJS, Hono, Socket.IO, BullMQ, Temporal, Bun, Deno, tRPC
- **Python (7)**: Flask, FastAPI, Django REST, Celery, Ray, AI Agents
- **Go (4)**: Gin, Echo, Fiber, gRPC  
- **Other (10)**: Rust, Java, C#, C++, Ruby, Elixir, PHP, Kotlin, Swift

### Available Commands (39 Total)
admin, backup, build, clean, config, db, deploy, diff, doctor, email, exec, help, init, logs, metrics, mlflow, monitor, prod, reset, restart, rollback, scale, scaffold, search, ssl, start, status, stop, trust, update, urls, validate, version, wizard, up, down

### Bug Fixes in v0.3.9
- Fixed status command hanging issue
- Fixed stop command compose wrapper
- Fixed exec command container detection
- Fixed build command timeout (5 seconds)
- Fixed email SMTP testing
- Fixed doctor command function names
- Fixed init command .env creation
- Fixed environment loading cascade
- Fixed display library aliases

[View Full Release Notes →](./RELEASES.md)

---

## Next (v0.4.0)
**Status**: 📋 Planned | **Target**: Q1 2025  
**Focus**: Refinement and Polish

### Goals
- **Perfect existing features** - Make everything rock solid
- **Complete partial implementations** - Finish deploy, search, mlflow commands
- **Enhanced stability** - More comprehensive testing
- **Better developer experience** - Improved error messages and documentation

### Planned Improvements
- **Deploy Command** - Complete SSH deployment implementation
  - Automated VPS setup
  - GitHub webhook integration
  - Zero-downtime deployments
  
- **Search Command** - Finish enterprise search integration
  - MeiliSearch, Typesense, Elasticsearch support
  - Auto-indexing from PostgreSQL
  - Search UI components
  
- **MLflow Integration** - ML experiment tracking
  - Model versioning
  - Metrics visualization
  - Integration with Ray

- **Enhanced Monitoring** - Complete observability
  - Prometheus + Grafana dashboards
  - Log aggregation with Loki
  - Performance metrics

- **Production Hardening**
  - Kubernetes manifests generation
  - Multi-node PostgreSQL support
  - Advanced backup strategies
  - Security scanning

---

## Beyond (Future)
**Status**: 🔮 Future Plans | **Timeline**: 2025+  
**Focus**: Cloud Management & Enterprise Features

### Cloud & Deployment
- **nself deploy** - Complete deployment automation
  - Multi-environment sync (dev → staging → prod)
  - Automated rollbacks
  - Blue-green deployments
  - GitOps integration
  
- **Cloud Providers** - Native integrations
  - AWS (ECS, RDS, S3)
  - Google Cloud (GKE, Cloud SQL)
  - Azure (AKS, PostgreSQL)
  - DigitalOcean Apps Platform
  
- **Container Orchestration**
  - Kubernetes operators
  - Helm charts
  - Docker Swarm mode
  - Nomad support

### Enterprise Features
- **Security & Compliance**
  - SSO/SAML integration
  - RBAC with fine-grained permissions
  - Audit logging
  - HIPAA/SOC2/GDPR compliance templates
  
- **Advanced Database**
  - Multi-region replication
  - Read replicas
  - Connection pooling with PgBouncer
  - Automatic failover
  
- **Developer Tools**
  - Code generation (TypeScript, GraphQL, OpenAPI)
  - Database migration tools
  - API testing framework
  - CI/CD templates

### Innovation
- **AI/ML Platform**
  - Vector database support (pgvector)
  - LLM integration helpers
  - ML model serving
  - Data pipeline tools
  
- **Edge Computing**
  - Edge function runtime
  - CDN integration
  - Global database sync
  - Offline-first capabilities
  
- **nself Cloud** (Optional)
  - Managed hosting option
  - One-click deployments
  - Automatic updates
  - Enterprise support

---

## Development Principles

1. **Stability First** - Never break existing features
2. **Smart Defaults** - Everything works out of the box
3. **No Lock-in** - Standard Docker/PostgreSQL/GraphQL
4. **Progressive Disclosure** - Advanced features stay hidden
5. **Auto-Fix** - Detect and resolve problems automatically
6. **Offline-First** - Works without internet
7. **Security by Default** - Production-ready security

---

## Release Timeline

| Version | Status | Focus | Timeline |
|---------|--------|-------|----------|
| [v0.3.9](#released-v039) | ✅ Released | Service Templates & Admin UI | Available Now |
| [v0.4.0](#next-v040) | 📋 Planned | Polish & Refinement | Q1 2025 |
| [Beyond](#beyond-future) | 🔮 Future | Cloud & Enterprise | 2025+ |

---

## Contributing

### Priority Areas for v0.4.0
1. Complete partial command implementations (deploy, search, mlflow)
2. Improve test coverage
3. Documentation improvements
4. Bug fixes and stability
5. Performance optimizations

### How to Contribute
- **GitHub**: [github.com/acamarata/nself](https://github.com/acamarata/nself)
- **Issues**: [Report Bugs](https://github.com/acamarata/nself/issues)
- **Testing**: Help test v0.4.0 features in development

---

*This roadmap reflects actual implemented features and realistic future plans. Updated regularly based on development progress and community feedback.*