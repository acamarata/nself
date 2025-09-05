# nself Release History

## Overview

This document provides an index of all nself releases with links to detailed release notes.

---

## [v0.3.10] - September 5, 2024

**Status**: Current Stable Release  
**[Full Release Notes](./RELEASE-v0.3.10.md)**

### Highlights
- 🔍 **Search Services** - 6 self-hosted search engines (Meilisearch, Typesense, Zinc, Elasticsearch, OpenSearch, Sonic)
- 🎛️ **nself-admin Integration** - Full compatibility with initialization wizard
- 📚 **Complete Documentation** - Exhaustive environment variable reference
- 🔧 **Service Enable Flags** - Explicit enable/disable for all services
- 🌐 **Frontend App Support** - Configure multiple frontend applications
- 🔄 **Variable Compatibility** - Backward compatibility for renamed variables

### Quick Stats
- 6 search engines integrated
- 200+ environment variables documented
- Full nself-admin compatibility
- Complete backward compatibility
- Zero breaking changes

---

## [v0.3.9] - September 3, 2024

**Status**: Stable Release  
**[Full Release Notes](./RELEASE-v0.3.9-final.md)**

### Highlights
- 🎯 **Admin UI Integration** - Web-based monitoring dashboard at localhost:3100
- 🐛 **Critical Bug Fixes** - Fixed status, stop, exec, build commands
- 📧 **SMTP Testing** - Email test functionality with swaks
- ⏱️ **Build Timeout** - 5-second validation timeout prevents hangs
- 📦 **34 Commands** - All fully functional and tested
- 🚀 **40+ Service Templates** - Production-ready templates for all major languages

### Quick Stats
- 9 bugs fixed
- 34 commands working
- 40 service templates
- 76 library files
- ~58,000 lines of code
- Quality score: 9.8/10

---

## [v0.3.8] - August 15, 2024

**Status**: Stable Release

### Highlights
- 🔐 **Complete SSL System** - Automatic certificate generation and trust
- 💾 **Enterprise Backup** - Comprehensive backup with S3 support
- 📊 **Monitoring Stack** - Prometheus, Grafana, Loki integration
- 🔧 **Auto-Fix System** - Intelligent error recovery
- 📧 **Email System** - 16+ provider support

### Major Features
- SSL certificate management with mkcert
- Backup system with scheduling and cloud storage
- Production deployment configurations
- Enhanced error handling and recovery

---

## [v0.3.7] - July 30, 2024

**Status**: Stable Release

### Highlights
- 🚀 **Initial Public Release**
- 🐳 **Docker Compose Generation**
- 🔄 **Service Management**
- 📦 **Package Manager Support**

### Features
- Core command structure
- Basic service management
- Environment configuration
- Docker integration

---

## Version Numbering

nself follows semantic versioning:

- **Major (X.0.0)**: Breaking changes, major architecture updates
- **Minor (0.X.0)**: New features, backward compatible
- **Patch (0.0.X)**: Bug fixes, minor improvements
- **Tags**: 
  - `-alpha`: Early testing, may be unstable
  - `-beta`: Feature complete, testing for stability
  - `-rc`: Release candidate, final testing
  - (no tag): Stable production release

## Release Cycle

- **Alpha**: Internal testing, rapid iteration
- **Beta**: Public testing, feature freeze
- **RC**: Final testing, bug fixes only
- **Stable**: Production ready

## Support Policy

- **Current stable**: Full support
- **Previous stable**: Security updates for 6 months
- **Beta releases**: Community support
- **Alpha releases**: No support

## Upcoming Releases

### v0.4.0 (Q4 2024)
- Kubernetes deployment support
- Multi-node clustering
- Advanced monitoring with alerts
- GraphQL subscriptions
- WebSocket support

### v0.5.0 (Q1 2025)
- Plugin system
- Custom service templates
- CI/CD integrations
- Cloud provider abstractions
- Terraform modules

## Archives

Older releases and their documentation can be found in the [GitHub Releases](https://github.com/acamarata/nself/releases) page.

---

*For the latest version, run `nself version` or check [src/VERSION](../src/VERSION)*