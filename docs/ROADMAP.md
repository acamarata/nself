# nself Development Roadmap

## Vision
Transform nself from a powerful CLI tool into a complete self-hosted backend platform that rivals commercial BaaS offerings (Supabase, Nhost, Firebase) while maintaining simplicity, control, and the ability to run anywhere.

---

## 📍 Current Release (v0.3.8)
**Status**: Stable and production-ready for core features

Complete self-hosted backend featuring:
- **Stack**: PostgreSQL, Hasura GraphQL, Auth, Storage (MinIO)
- **Services**: 17+ optional services including Redis, email, monitoring
- **Backup**: Comprehensive system with S3 support and scheduling
- **SSL**: Automatic certificates with mkcert (dev) and Let's Encrypt (prod)
- **Smart Features**: Auto-fix capabilities, drift detection, validation
- **Commands**: 30+ commands including init, build, start, backup, ssl, monitor, doctor

[Full changelog →](./CHANGELOG.md)

---

## 🚧 In Development (v0.3.9)
**Target**: 5 weeks | **Focus**: Platform completion

### Core Features
- **[Admin UI](./v0.3.9.md#1-admin-ui)** - Web-based administration interface
- **[Deploy Command](./v0.3.9.md#2-deploy-command)** - VPS deployment with SSH
- **[Init Wizard](./v0.3.9.md#3-init-wizard)** - Interactive project setup
- **[Search](./v0.3.9.md#4-search-command)** - PostgreSQL FTS and MeiliSearch
- **[Environment Management](./v0.3.9.md#5-environment-management)** - Multi-environment configs

[Full v0.3.9 plan →](./v0.3.9.md)

---

## 📦 Next Release (v0.4.0)
**Target**: Public release | **Focus**: Polish and perfect

### Enhancements
- **[Apps](./v0.4.0.md#1-apps-enhancement)** - Multi-app with database isolation
- **[Codegen](./v0.4.0.md#2-codegen-command)** - TypeScript, React, GraphQL generation
- **[Functions](./v0.4.0.md#3-functions-polish)** - Edge functions with triggers
- **[Monitor](./v0.4.0.md#4-monitor-polish)** - Full observability stack
- **[Database](./v0.4.0.md#5-database-polish)** - Performance and migration tools
- **[Backup](./v0.4.0.md#6-backup-polish)** - Multi-cloud destinations
- **[SSL](./v0.4.0.md#7-ssl-polish)** - Production automation
- **[Environment](./v0.4.0.md#8-environment-polish)** - Enhanced management

[Full v0.4.0 plan →](./v0.4.0.md)

---

## 🚀 Future Releases (Beyond)
**Priority-ordered features for future development**

### Cloud & Infrastructure
- **Cloud Providers** - AWS, GCP, Azure, DigitalOcean native integration
- **Kubernetes** - Manifest generation, Helm charts, GitOps
- **Container Registry** - DockerHub, GitHub, ECR, GCR push support
- **CI/CD Workflows** - GitHub Actions, GitLab CI templates

### Enterprise Features
- **Database Visualizer** - Visual schema designer and query builder
- **Email Templates** - HTML email templates with MJML support
- **Enterprise Security** - SSO/SAML, RBAC, audit logging, compliance
- **Advanced Monitoring** - APM integration, SLO monitoring, distributed tracing

### Scale & Innovation
- **Global Scale** - Multi-region, edge computing, geo-replication
- **AI/ML Integration** - Vector databases, LLM support, model serving
- **Template System** - Reusable starters (SaaS, E-commerce, Blog)
- **Plugin Architecture** - Extensibility for community contributions

### Platform Ecosystem
- **nself Cloud** - Optional managed hosting service
- **Marketplace** - Community templates and extensions

---

## Development Principles

1. **Simplicity First** - Every feature must work with smart defaults
2. **No Lock-in** - Standard Docker/PostgreSQL/GraphQL under the hood
3. **Progressive Disclosure** - Basic users never see advanced features
4. **Auto-Fix Everything** - Detect and resolve issues automatically
5. **Offline-First** - Everything works locally without internet
6. **Type-Safe** - Generated code with full TypeScript support
7. **Secure by Default** - Production-ready security out of the box

---

## Release Schedule

| Version | Status | Focus | Timeline |
|---------|--------|-------|----------|
| v0.3.8 | ✅ Released | Core platform | Current |
| v0.3.9 | 🚧 Development | Feature complete | 5 weeks |
| v0.4.0 | 📋 Planned | Public release | Q2 2025 |
| Beyond | 🔮 Future | Enterprise & scale | 2025+ |

---

## Getting Involved

### For Users
- **Try it**: `npm install -g @acamarata/nself`
- **Report issues**: [GitHub Issues](https://github.com/acamarata/nself/issues)
- **Join community**: [Discord](https://discord.gg/nself)

### For Contributors
Priority areas for contribution:
1. Testing v0.3.9 features
2. Documentation improvements
3. Cloud provider integrations
4. Language-specific code generators
5. Example applications

### Resources
- **GitHub**: [github.com/acamarata/nself](https://github.com/acamarata/nself)
- **Documentation**: [Getting Started](../README.md)
- **Admin UI Spec**: [admin.md](../admin.md)

---

*This roadmap is updated regularly based on community feedback and development progress. Features may be reprioritized based on user needs.*