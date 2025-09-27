# nself - Backend-as-a-Service Platform

<div align="center">

**The Complete Backend-as-a-Service Platform**

[![Version](https://img.shields.io/badge/version-0.4.0-blue.svg)](RELEASES)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-compose%20v2-blue.svg)](https://docs.docker.com/compose/)
[![Status](https://img.shields.io/badge/status-production%20ready-brightgreen.svg)](https://github.com/nself/nself)

</div>

---

## 🚀 Quick Start

```bash
# Install nself
curl -fsSL https://raw.githubusercontent.com/nself/nself/main/install.sh | bash

# Initialize a demo project with all services
nself init --demo

# Build services from templates
nself build

# Start everything
nself start
```

→ **[View Complete Demo Setup](DEMO_SETUP)** - 24 services showcasing all capabilities

## 📚 Documentation

### Getting Started
- **[Quick Start Guide](Quick-Start)** - Get up and running in 5 minutes
- **[Demo Setup](DEMO_SETUP)** - Complete demo with all 24 services
- **[FAQ](FAQ)** - Frequently asked questions
- **[Examples](EXAMPLES)** - Code examples and patterns

### Services Documentation
- **[Services Overview](SERVICES)** - All available services
- **[Required Services](SERVICES_REQUIRED)** - Core infrastructure (4 services)
- **[Optional Services](SERVICES_OPTIONAL)** - Additional capabilities (16+ services)
- **[Custom Services](SERVICES_CUSTOM)** - Build your own microservices
- **[Monitoring Bundle](MONITORING_BUNDLE)** - Complete observability (10 services)
- **[nself Admin](NSELF_ADMIN)** - Web-based management interface

### Configuration & Commands
- **[Commands Reference](COMMANDS)** - Complete CLI reference
- **[Command Tree](COMMAND-TREE-FINAL)** - Visual command structure
- **[Environment Variables](ENVIRONMENT-VARIABLES)** - Configuration reference
- **[Complete ENV Reference](ENV-COMPLETE-REFERENCE)** - All variables
- **[Environment Setup](ENVIRONMENT_CONFIGURATION)** - Configuration guide

### Architecture & Guides
- **[Architecture Overview](ARCHITECTURE)** - System design
- **[Project Structure](PROJECT_STRUCTURE)** - File organization
- **[Troubleshooting](TROUBLESHOOTING)** - Common issues and solutions
- **[Backup Guide](BACKUP_GUIDE)** - Backup and recovery
- **[Domain Selection](domain-selection-guide)** - Choosing your domain

### Releases & Updates
- **[Changelog](CHANGELOG)** - Version history
- **[Latest Release Notes](RELEASES)** - Current version details
- **[v0.3.9 Release](RELEASE-v0.3.9)** - Previous stable release

## 🎯 Service Overview

### Core Services (Required - 4)
- **PostgreSQL** - Primary database with 60+ extensions
- **Hasura** - Instant GraphQL APIs
- **Auth** - JWT-based authentication
- **Nginx** - Reverse proxy and SSL

### Optional Services (16 in demo)
- **Monitoring Bundle (10)** - Prometheus, Grafana, Loki, Promtail, Tempo, Alertmanager, exporters
- **Redis** - Caching and sessions
- **MinIO** - S3-compatible storage
- **nself Admin** - Web management UI
- **MailPit** - Email testing
- **MeiliSearch** - Full-text search
- **Storage API** - File management

### Custom Services (Your code)
- Build from 40+ templates
- Express, FastAPI, Go, Rust, and more
- Automatic Docker integration
- Full monitoring and logging

## 🎆 Demo Project

The demo setup showcases all capabilities with **24 running services**:

```bash
nself init --demo  # Creates complete demo environment
nself build        # Generates 4 custom services from templates
nself start        # Launches all 24 services
```

**Demo includes:**
- Express.js REST API
- BullMQ job worker
- Go gRPC service
- Python FastAPI
- Full monitoring stack
- All optional services

## 🌟 Key Features

- **🔧 Zero DevOps** - Focus on code, not infrastructure
- **🚀 One Command** - `nself init && nself build && nself start`
- **🎛️ Admin Dashboard** - Complete web-based management
- **🔐 Authentication** - JWT with social providers
- **📦 S3 Storage** - MinIO with CDN support
- **🔄 Auto-SSL** - Let's Encrypt integration
- **📊 Full Monitoring** - Metrics, logs, traces
- **🎯 Multi-Environment** - Dev, staging, production
- **🔌 60+ Extensions** - PostgreSQL fully loaded
- **📧 Email Service** - Multiple providers

## 📊 Common Use Cases

### SaaS Applications
Complete backend with auth, billing, multi-tenancy

### API Platforms
RESTful and GraphQL APIs with documentation

### Microservices
Service mesh with discovery and monitoring

### ML Platforms
MLflow integration for model training and serving

## 🤝 Community & Support

- **GitHub Issues** - [Report bugs and request features](https://github.com/nself/nself/issues)
- **Discussions** - [Ask questions and share ideas](https://github.com/nself/nself/discussions)
- **Discord** - [Join our community](https://discord.gg/nself)
- **Twitter** - [@nself](https://twitter.com/nself)

## 📝 License

nself is open-source software licensed under the MIT License.

---

**Current Version:** v0.4.0 | **Last Updated:** September 2024 | **Status:** Production Ready