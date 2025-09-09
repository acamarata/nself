# nself Documentation Wiki

<div align="center">

![nself Logo](https://raw.githubusercontent.com/acamarata/nself/main/docs/assets/logo.png)

**The Complete Self-Hosted Backend Stack**

[![Version](https://img.shields.io/badge/version-0.3.9-blue.svg)](https://github.com/acamarata/nself/releases)
[![License](https://img.shields.io/badge/license-Source%20Available-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-compose%20v2-blue.svg)](https://docs.docker.com/compose/)
[![Status](https://img.shields.io/badge/status-production%20ready-brightgreen.svg)](https://github.com/acamarata/nself)

</div>

---

## 📚 Table of Contents

### 🚀 Getting Started
- **[Installation Guide](Installation)** - System requirements and setup
- **[Quick Start](Quick-Start)** - Get running in 5 minutes
- **[Basic Configuration](Basic-Configuration)** - Essential settings
- **[First Project](First-Project)** - Create your first nself project

### 📖 Core Documentation
- **[Commands Reference](Commands)** - Complete CLI command reference
- **[Configuration Guide](Configuration)** - Environment variables and settings
- **[Architecture Overview](Architecture)** - System design and components
- **[Service Templates](Service-Templates)** - Available microservice templates

### 🛠️ Features & Services
- **[Admin UI](Admin-UI)** - Web-based administration interface
- **[Database Management](Database)** - PostgreSQL with 60+ extensions
- **[Authentication](Authentication)** - JWT-based auth system
- **[Storage System](Storage)** - MinIO S3-compatible storage
- **[GraphQL API](GraphQL)** - Hasura GraphQL engine
- **[Email Service](Email)** - Multi-provider email configuration
- **[SSL/TLS Management](SSL)** - Certificate management

### 🚢 Deployment & Operations
- **[Production Deployment](Deployment)** - Deploy to production
- **[Backup & Recovery](Backup-Guide)** - Backup strategies
- **[Monitoring & Metrics](Monitoring)** - Prometheus, Grafana, Loki
- **[Scaling Guide](Scaling)** - Horizontal and vertical scaling
- **[Security Best Practices](Security)** - Hardening your deployment

### 🔧 Advanced Topics
- **[Microservices](Microservices)** - Adding custom services
- **[Docker Compose](Docker-Compose)** - Understanding the orchestration
- **[Environment Management](Environment-Management)** - Multi-environment setup
- **[Hooks & Automation](Hooks)** - Pre/post command hooks
- **[Custom Templates](Custom-Templates)** - Creating service templates

### 🐛 Troubleshooting & Support
- **[Troubleshooting Guide](Troubleshooting)** - Common issues and solutions
- **[FAQ](FAQ)** - Frequently asked questions
- **[Known Issues](Known-Issues)** - Current limitations
- **[Support](Support)** - Getting help

### 👥 Development
- **[Contributing Guide](Contributing)** - How to contribute
- **[Development Setup](Development)** - Setting up dev environment
- **[Testing](Testing)** - Running tests
- **[API Reference](API)** - Internal APIs

### 📋 Resources
- **[Changelog](Changelog)** - Version history
- **[Roadmap](Roadmap)** - Future plans
- **[Release Notes](Releases)** - Latest releases
- **[Examples](Examples)** - Code examples and recipes
- **[Directory Structure](Directory-Structure)** - Project layout

---

## 🎯 Quick Navigation

<table>
<tr>
<td width="33%" valign="top">

### 📦 Core Services
- [PostgreSQL 16](Services#postgresql)
- [Hasura GraphQL](Services#hasura)
- [Nhost Auth](Services#authentication)
- [MinIO Storage](Services#storage)
- [Redis Cache](Services#redis)
- [Nginx Proxy](Services#nginx)

</td>
<td width="33%" valign="top">

### 🔌 Optional Services
- [Prometheus](Services#prometheus)
- [Grafana](Services#grafana)
- [Loki Logging](Services#loki)
- [Jaeger Tracing](Services#jaeger)
- [Temporal Workflow](Services#temporal)
- [RabbitMQ](Services#rabbitmq)

</td>
<td width="33%" valign="top">

### 🎨 Templates
- [Node.js](Templates#nodejs)
- [Python](Templates#python)
- [Go](Templates#golang)
- [Rust](Templates#rust)
- [.NET](Templates#dotnet)
- [Java](Templates#java)

</td>
</tr>
</table>

---

## 🚦 Current Status

| Component | Status | Version | Health |
|-----------|--------|---------|--------|
| **nself CLI** | ✅ Stable | v0.3.9 | ![100%](https://img.shields.io/badge/health-100%25-brightgreen) |
| **PostgreSQL** | ✅ Stable | 16-alpine | ![100%](https://img.shields.io/badge/health-100%25-brightgreen) |
| **Hasura** | ✅ Stable | v2.44.0 | ![100%](https://img.shields.io/badge/health-100%25-brightgreen) |
| **Auth Service** | ⚠️ Works* | v0.36.0 | ![95%](https://img.shields.io/badge/health-95%25-yellow) |
| **Storage** | ✅ Stable | v0.6.1 | ![100%](https://img.shields.io/badge/health-100%25-brightgreen) |
| **Admin UI** | ✅ Stable | v0.0.3 | ![100%](https://img.shields.io/badge/health-100%25-brightgreen) |

*Auth service health check reports unhealthy but service works correctly on port 4001

---

## 🌟 Key Features

<div align="center">

| Feature | Description |
|---------|-------------|
| **🔧 Smart Defaults** | Auto-configuration with intelligent defaults |
| **🚀 One-Command Deploy** | `nself init && nself build && nself start` |
| **🎛️ Admin Dashboard** | Web-based management interface |
| **🔐 Built-in Auth** | JWT authentication with 20+ providers |
| **📦 S3 Storage** | MinIO with Hasura Storage integration |
| **🔄 Auto-SSL** | Automatic certificate generation |
| **📊 Monitoring** | Prometheus, Grafana, Loki integration |
| **🎯 Multi-Environment** | Dev, staging, production configs |
| **🔌 60+ Extensions** | PostgreSQL with all extensions |
| **📧 Email Service** | 16+ email provider integrations |

</div>

---

## 📊 Latest Release: v0.3.9

### What's New
- ✨ **Admin UI Integration** - Full web-based administration
- 🐛 **Major Bug Fixes** - Build, status, stop commands fixed
- 📦 **40+ Service Templates** - Expanded template library
- 🔧 **Auto-Fix System** - Intelligent error recovery
- 📚 **Enhanced Documentation** - Comprehensive guides

[View Full Changelog →](Changelog#v039)

---

## 🎓 Learning Path

### Beginner
1. Start with [Installation](Installation)
2. Follow the [Quick Start](Quick-Start)
3. Read [Basic Configuration](Basic-Configuration)
4. Try [Your First Project](First-Project)

### Intermediate
1. Explore [Commands Reference](Commands)
2. Understand [Architecture](Architecture)
3. Configure [Services](Services)
4. Setup [Admin UI](Admin-UI)

### Advanced
1. Deploy to [Production](Deployment)
2. Implement [Microservices](Microservices)
3. Configure [Monitoring](Monitoring)
4. Customize with [Templates](Custom-Templates)

---

## 🤝 Community & Support

<div align="center">

| Channel | Purpose | Link |
|---------|---------|------|
| **GitHub Issues** | Bug reports & features | [Create Issue](https://github.com/acamarata/nself/issues) |
| **Discussions** | Questions & help | [Join Discussion](https://github.com/acamarata/nself/discussions) |
| **Discord** | Real-time chat | Coming Soon |
| **Email** | Direct support | support@nself.org |

</div>

---

## 📝 License

nself is **Source Available** software:
- ✅ **Free** for personal and non-commercial use
- 💰 **Paid license** required for commercial use
- 📖 See [LICENSE](https://github.com/acamarata/nself/blob/main/LICENSE) for details

---

<div align="center">

**[Get Started](Quick-Start)** | **[Commands](Commands)** | **[Configuration](Configuration)** | **[Support](Support)**

Made with ❤️ by the nself team

</div>