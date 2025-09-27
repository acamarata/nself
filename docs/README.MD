# nself Documentation

> **Complete documentation for the nself Backend-as-a-Service platform**

This documentation is organized for both local browsing and GitHub Wiki integration. All documentation is maintained in Markdown format for maximum compatibility.

## 📚 Documentation Structure

```
/docs/
├── Home.md                    # Main documentation homepage
├── README.md                  # This file - documentation index
├── services/                  # Service documentation
│   ├── SERVICES.md           # Overview of all services
│   ├── SERVICES_REQUIRED.md  # Core required services
│   ├── SERVICES_OPTIONAL.md  # Optional services
│   ├── SERVICES_CUSTOM.md    # Custom service templates
│   ├── MONITORING_BUNDLE.md  # Monitoring stack (10 services)
│   ├── NSELF_ADMIN.md        # Admin UI documentation
│   └── DEMO_SETUP.md         # Complete demo walkthrough
├── configuration/            # Configuration guides
│   ├── ENVIRONMENT-VARIABLES.md
│   ├── ENV-COMPLETE-REFERENCE.md
│   └── ENVIRONMENT_CONFIGURATION.md
├── commands/                 # CLI command reference
│   ├── COMMANDS.md
│   └── COMMAND-TREE-FINAL.md
├── architecture/             # System design documentation
│   ├── ARCHITECTURE.md
│   ├── PROJECT_STRUCTURE.md
│   └── BUILD_ARCHITECTURE.md
├── guides/                   # How-to guides and tutorials
│   ├── Quick-Start.md
│   ├── TROUBLESHOOTING.md
│   ├── FAQ.md
│   ├── EXAMPLES.md
│   └── BACKUP_GUIDE.md
└── releases/                 # Release notes and changelogs
    ├── CHANGELOG.md
    ├── RELEASES.md
    └── v0.4.0.md
```

## 🎯 Quick Start

New to nself? Start here:

1. **[Quick Start Guide](guides/Quick-Start)** - Get running in 5 minutes
2. **[Demo Setup](services/DEMO_SETUP)** - Try the complete demo with 24 services
3. **[Services Overview](services/SERVICES)** - Understand available services
4. **[Commands Reference](commands/COMMANDS)** - Learn the CLI

## 📖 Documentation by Category

### Services & Components
- **[All Services](services/SERVICES)** - Complete service catalog
- **[Required Services](services/SERVICES_REQUIRED)** - Core infrastructure (4 services)
- **[Optional Services](services/SERVICES_OPTIONAL)** - Additional capabilities (16+ services)
- **[Custom Services](services/SERVICES_CUSTOM)** - Build your own with 40+ templates
- **[Monitoring Bundle](services/MONITORING_BUNDLE)** - Full observability stack (10 services)
- **[nself Admin](services/NSELF_ADMIN)** - Web-based management interface

### Configuration
- **[Environment Variables](configuration/ENVIRONMENT-VARIABLES)** - All configuration options
- **[Complete ENV Reference](configuration/ENV-COMPLETE-REFERENCE)** - Every variable documented
- **[Configuration Guide](configuration/ENVIRONMENT_CONFIGURATION)** - Setup walkthrough

### Commands & CLI
- **[Commands Reference](commands/COMMANDS)** - All CLI commands
- **[Command Tree](commands/COMMAND-TREE-FINAL)** - Visual command structure

### Architecture
- **[System Architecture](architecture/ARCHITECTURE)** - Design principles and patterns
- **[Project Structure](architecture/PROJECT_STRUCTURE)** - File organization
- **[Build System](architecture/BUILD_ARCHITECTURE)** - How nself build works

### Guides & Tutorials
- **[Quick Start](guides/Quick-Start)** - Get started fast
- **[Troubleshooting](guides/TROUBLESHOOTING)** - Fix common issues
- **[FAQ](guides/FAQ)** - Frequently asked questions
- **[Examples](guides/EXAMPLES)** - Code examples and patterns
- **[Backup Guide](guides/BACKUP_GUIDE)** - Backup and recovery

### Releases
- **[Changelog](releases/CHANGELOG)** - Version history
- **[Current Release](releases/RELEASES)** - Latest version details
- **[v0.4.0](releases/v0.4.0)** - Current stable release

## 🎆 Demo Project

Experience the full power of nself with our demo:

```bash
nself init --demo   # Initialize with all services
nself build         # Generate custom services
nself start         # Launch 24 services
```

The demo includes:
- ✅ All 4 required services
- ✅ 16 optional services (including 10 monitoring)
- ✅ 4 custom services from templates
- ✅ Full monitoring and observability
- ✅ Web-based admin interface

**[→ Complete Demo Documentation](services/DEMO_SETUP)**

## 📊 Service Counts

### Minimal Setup
- 4 services (required only)

### Typical Development
- 8-12 services (required + Redis + Mail + Search)

### Full Demo
- 24 services (everything enabled)

### Maximum Configuration
- 40+ services (all optional + 10 custom)

## 🔧 Key Concepts

### Service Types
1. **Required** - Essential infrastructure
2. **Optional** - Enable as needed
3. **Custom** - Your microservices

### Environments
- `dev` - Local development
- `staging` - Pre-production testing
- `production` - Live deployment

### Templates
40+ service templates in multiple languages:
- **JavaScript/TypeScript**: Express, Fastify, NestJS, BullMQ
- **Python**: FastAPI, Flask, Django, Celery
- **Go**: Gin, Fiber, Echo, gRPC
- **Others**: Rust, Ruby, PHP, Java, C#, Elixir

## 🤝 Contributing

Want to contribute? Check out:
- [Contributing Guide](guides/CONTRIBUTING)
- [Development Roadmap](guides/ROADMAP)
- [GitHub Issues](https://github.com/nself/nself/issues)

## 📝 License

nself is open-source software licensed under the MIT License.

## 🔗 Links

- **GitHub**: [github.com/nself/nself](https://github.com/nself/nself)
- **Wiki**: [github.com/nself/nself/wiki](https://github.com/nself/nself/wiki)
- **Issues**: [github.com/nself/nself/issues](https://github.com/nself/nself/issues)
- **Discussions**: [github.com/nself/nself/discussions](https://github.com/nself/nself/discussions)

---

**Version**: 0.4.0 | **Last Updated**: September 2024