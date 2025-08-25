# nself Development Roadmap

## Quick Navigation
[Previous (v0.3.8)](#previous-v038) | [Current (v0.3.9)](#current-v039) | [Next (v0.4.0)](#next-v040) | [Beyond](#beyond-future)

---

## Vision
Transform nself from a powerful CLI tool into a complete self-hosted backend platform that rivals commercial BaaS offerings (Supabase, Nhost, Firebase) while maintaining simplicity, control, and the ability to run anywhere.

---

## Previous (v0.3.8)
**Status**: ✅ Released and Stable

### Features Available Now
- **PostgreSQL** - Full database with 60+ extensions
- **Hasura GraphQL** - Instant GraphQL API
- **Auth Service** - JWT authentication
- **Storage** - S3-compatible MinIO
- **Backup System** - S3 support, scheduling, incremental
- **SSL** - Automatic certificates (mkcert + Let's Encrypt)
- **Email** - 16+ providers with MailPit for dev
- **Monitoring** - Prometheus, Grafana, Loki ready
- **Auto-Fix** - Smart defaults and problem resolution
- **Docker Compose** - Simple container orchestration

### Available Commands
init, build, start, stop, restart, status, logs, backup (10 subcommands), db, email, ssl, doctor, validate, exec, scale, metrics, clean, diff, reset, rollback, monitor, scaffold, hot_reload, update, version, help, urls, prod, trust

[View Changelog →](./CHANGELOG.md)

---

## Current (v0.3.9)
**Status**: ✅ Released | **Release Date**: August 2024

### New Features
- **Admin UI** - Web-based administration interface
- **Deploy** - SSH deployment to VPS servers  
- **Init Wizard** - Interactive setup for new projects
- **Search** - Enterprise search with 6 engine options
- **Environment Management** - Multi-environment configuration system

### Admin UI Details
- Separate Docker container with host mount
- Next.js + TypeScript + Tailwind (Protocol template)
- Real-time service monitoring
- Configuration editor
- Log streaming
- Backup management

### Deploy Command Details
- SSH key authentication
- Automatic Docker installation
- Git repository setup
- Environment compilation (.env.prod + .env.secrets)
- Let's Encrypt SSL automation
- GitHub webhook support
- Supports: DigitalOcean, Linode, Vultr, Hetzner, any Ubuntu/Debian VPS

### Init Wizard Details
- Project type detection (SaaS, E-commerce, Blog, API)
- Service recommendations
- Database setup with owner user
- Framework detection
- Sample data generation

### Search Options
- **PostgreSQL FTS** (Default) - No extra container, built-in
- **MeiliSearch** - Best search UX, typo-tolerant
- **Typesense** - Fast instant search
- **Elasticsearch** - Enterprise standard
- **OpenSearch** - AWS maintained
- **Sonic** - Lightweight option

[Full v0.3.9 Documentation →](./v0.3.9.md)

---

## Next (v0.4.0)
**Status**: 📋 Planned | **Focus**: Polish for Public Release

### Enhancements to Existing Features
- **Apps** - Expand routing to full multi-app with DB isolation
- **Codegen** - Generate TypeScript, React, GraphQL clients
- **Functions** - Complete edge functions with triggers
- **Monitor** - Full observability with dashboards
- **Database** - Performance tools and optimization
- **Backup** - Multi-cloud destinations (B2, GCS, Azure)
- **SSL** - Production Let's Encrypt automation
- **Environment** - Enhanced configuration management

### Apps Enhancement Details
- Database table prefixing (app1_users, app2_products)
- Per-app Hasura metadata
- Isolated GraphQL schemas
- Cross-app data sharing options

### Codegen Details
- TypeScript interfaces from schema
- React hooks with SWR/React Query
- GraphQL client with full typing
- OpenAPI 3.0 specifications
- Watch mode for development

### Functions Enhancement Details
- Node.js runtime (Python/Deno coming)
- Hot reload in development
- Database event triggers
- Scheduled functions (cron)
- Webhook endpoints

[Full v0.4.0 Documentation →](./v0.4.0.md)

---

## Beyond (Future)
**Status**: 🔮 Future Plans | **Priority Ordered**

### Cloud & Infrastructure
- **Cloud Providers** - AWS, GCP, Azure, DigitalOcean integration
- **Kubernetes** - Manifests, Helm charts, GitOps
- **Container Registry** - Push to DockerHub, GitHub, ECR
- **CI/CD** - GitHub Actions, GitLab CI templates

### Enterprise Features
- **Database Visualizer** - Schema designer and query builder
- **Email Templates** - HTML templates with MJML
- **Security** - SSO/SAML, RBAC, audit logging
- **Advanced Monitoring** - APM, SLO, distributed tracing
- **Compliance** - HIPAA, SOC2, GDPR templates

### Scale & Innovation
- **Global Scale** - Multi-region, edge computing
- **AI/ML** - Vector databases, LLM support
- **Templates** - SaaS, E-commerce, Blog starters
- **Plugins** - Community extensions
- **nself Cloud** - Optional managed hosting
- **Marketplace** - Templates and plugins

---

## Development Principles

1. **Simplicity First** - Smart defaults for everything
2. **No Lock-in** - Standard Docker/PostgreSQL/GraphQL
3. **Progressive Disclosure** - Advanced features hidden
4. **Auto-Fix** - Detect and resolve automatically
5. **Offline-First** - Works without internet
6. **Type-Safe** - Full TypeScript support
7. **Secure by Default** - Production-ready security

---

## Release Timeline

| Version | Status | Focus | Timeline |
|---------|--------|-------|----------|
| [v0.3.8](#current-v038) | ✅ Released | Core Platform | Available Now |
| [v0.3.9](#developing-v039) | 🚧 Development | New Features | 5 weeks |
| [v0.4.0](#pending-v040) | 📋 Planned | Polish & Perfect | Q2 2025 |
| [Beyond](#beyond-future) | 🔮 Future | Enterprise & Scale | 2025+ |

---

## Contributing

### Priority Areas
1. Testing v0.3.9 features
2. Documentation improvements
3. Cloud provider integrations
4. Code generators for more languages
5. Example applications

### Resources
- **GitHub**: [github.com/acamarata/nself](https://github.com/acamarata/nself)
- **Discord**: [Join Community](https://discord.gg/nself)
- **Issues**: [Report Bugs](https://github.com/acamarata/nself/issues)

---

*This roadmap is updated regularly based on community feedback. Features may be reprioritized based on user needs.*