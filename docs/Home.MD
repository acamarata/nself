# Welcome to nself

> **The modern self-hosted backend infrastructure platform**

nself provides a complete, production-ready backend infrastructure that you can deploy on your own servers in minutes. Built on proven technologies like PostgreSQL, Hasura GraphQL, Redis, and Docker, nself simplifies complex deployments while maintaining full control and flexibility.

## 🚀 Quick Start

```bash
# Install nself CLI
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash

# Initialize your project
nself init

# Build infrastructure
nself build

# Start everything
nself up
```

Your complete backend is now running! 🎉

## 📖 Documentation Overview

### Getting Started
- **[[Examples|EXAMPLES]]** - Complete command examples with real outputs
- **[[API Reference|API]]** - Comprehensive command reference
- **[[Troubleshooting|TROUBLESHOOTING]]** - Solve common issues

### Configuration & Setup
- **[[Environment Configuration|ENVIRONMENT_CONFIGURATION]]** - Configure your deployment
- **[[Output Standards|OUTPUT_STANDARDS]]** - UI and output standards

### Architecture & Development
- **[[Architecture|ARCHITECTURE]]** - System design and patterns
- **[[Directory Structure|DIRECTORY_STRUCTURE]]** - Project organization
- **[[Contributing|CONTRIBUTING]]** - Contribute to nself

### Reference
- **[[Changelog|CHANGELOG]]** - Version history
- **[[Releases|RELEASES]]** - Release notes
- **[[Migration Guide|MIGRATION_0.2.x_to_0.3.0]]** - Upgrade guides

## 🎯 Key Features

### Complete Stack
- **PostgreSQL** - Powerful relational database with extensions
- **Hasura GraphQL** - Instant GraphQL API over your database
- **Authentication** - Secure JWT-based authentication
- **Storage** - MinIO S3-compatible object storage
- **Redis** - High-performance caching and queuing
- **Nginx** - Production-ready reverse proxy with SSL

### Developer Experience
- **One Command Setup** - From zero to running in minutes
- **Smart Defaults** - Works out-of-the-box with sensible configurations
- **Hot Reload** - Changes apply instantly during development
- **Multi-Environment** - Same setup for local, staging, and production
- **Auto-Fix** - Automatic detection and resolution of common issues

### Production Ready
- **SSL Certificates** - Automatic trusted certificates (no warnings!)
- **Email Integration** - 16+ email providers with zero-config development
- **Database Tools** - Migrations, seeding, backups, and sync
- **Service Monitoring** - Health checks, logs, and status tracking
- **Docker Native** - Standard Docker Compose under the hood

## 🛠️ Core Commands

| Command | Description |
|---------|-------------|
| `nself init` | Initialize a new project |
| `nself build` | Build infrastructure and images |
| `nself up` | Start all services |
| `nself down` | Stop all services |
| `nself status` | Check service health |
| `nself logs` | View service logs |
| `nself doctor` | Run diagnostics |
| `nself db` | Database management |
| `nself email` | Email configuration |
| `nself scaffold` | Create new services |

## 🎨 Service Scaffolding

Create microservices instantly with built-in templates:

```bash
# Create a NestJS API
nself scaffold nest api-gateway

# Create a BullMQ worker
nself scaffold bull email-worker --start

# Create a Go service
nself scaffold go websocket-server

# Create a Python service
nself scaffold python ml-service
```

## 📊 System Requirements

- **OS**: Linux, macOS, or Windows with WSL
- **Docker**: Version 20.10+
- **Docker Compose**: Version 2.0+
- **Memory**: 2GB minimum (4GB recommended)
- **Disk**: 5GB free space

## 🤝 Community & Support

- **GitHub**: [github.com/acamarata/nself](https://github.com/acamarata/nself)
- **Issues**: [Report bugs or request features](https://github.com/acamarata/nself/issues)
- **Discussions**: [Join the community](https://github.com/acamarata/nself/discussions)

## 📄 License

nself is dual-licensed:
- **Personal/Open Source**: Free forever under MIT license
- **Commercial**: Requires license for commercial use

See [LICENSE](https://github.com/acamarata/nself/blob/main/LICENSE) for details.

---

**Current Version**: v0.3.2-alpha | **Last Updated**: August 2025