# nself Release History

## Overview

This document provides an index of all nself releases with links to detailed release notes.

---

## [v0.4.0] - October 13, 2025

**Status**: ✅ Current Stable Release
**[Full Release Notes](./v0.4.0.md)** | **[Changelog](./CHANGELOG.md#040---2025-10-13)**

### Highlights
- 🎯 **Production-Ready Core** - Stable, tested, ready for real-world use
- 🐛 **Critical Bug Fixes** - Fixed unbound variables and Bash 4+ compatibility issues
- 🖥️ **Cross-Platform** - Bash 3.2+ compatible (macOS, Linux, WSL)
- ✅ **CI/CD Passing** - 12/12 tests passing on all platforms
- 📚 **Complete Documentation** - Comprehensive guides and examples
- 🔧 **15 Core Commands** - All essential commands functional and tested
- 🎨 **40+ Service Templates** - Production-ready templates for all major languages
- 📊 **Monitoring Bundle** - Complete 10-service observability stack

### What's New in v0.4.0
- **Fixed**: Unbound variable error in change-detection.sh
- **Fixed**: Bash 4+ uppercase expansion compatibility (replaced ${var^^} with tr)
- **Enhanced**: Cross-platform shell script compatibility
- **Improved**: Error handling with proper defaults
- **Tested**: Full QA cycle with all core workflows verified

### Quick Stats
- 15 core commands (all functional)
- 40 service templates
- 10-service monitoring bundle
- 4 required services (PostgreSQL, Hasura, Auth, Nginx)
- 7 optional services (Redis, MinIO, Admin, MailPit, MeiliSearch, Functions, MLflow)
- Quality score: 10/10

### What v0.4.0 Means
This release marks the **production-ready milestone** for nself. All core functionality is:
- ✅ Fully implemented and tested
- ✅ Cross-platform compatible
- ✅ Documented with examples
- ✅ Ready for real-world deployment

Future releases (v0.4.1+) will add management commands for existing services and polish features, but v0.4.0 provides a solid, stable foundation for production use.

---

## [v0.3.9] - September 3, 2025 (Patched September 16, 2025)

**Status**: Previous Stable Release
**[Full Release Notes](./v0.3.9.md)** | **[Installation Patch](./RELEASE-v0.3.9-patch3.md)**

### Highlights
- 🎯 **Admin UI Integration** - Web-based monitoring dashboard at localhost:3100
- 🐛 **Critical Bug Fixes** - Fixed status, stop, exec, build commands
- 📧 **SMTP Testing** - Email test functionality with swaks
- ⏱️ **Build Timeout** - 5-second validation timeout prevents hangs
- 📦 **36 Commands** - 34 fully functional + 2 partial (deploy, search)
- 🚀 **40+ Service Templates** - Production-ready templates for all major languages
- 🧪 **Serverless Functions** - Built-in functions runtime
- 🤖 **MLflow Integration** - ML experiment tracking and model registry
- 📦 **87% Smaller Releases** - Minimal tarballs (432KB vs 3.3MB) for faster installs
- 🚀 **Smart Installation** - Auto-detects version type for optimal downloads

### Quick Stats
- 9 bugs fixed
- 36 total commands (34 full + 2 partial)
- 40 service templates
- 76 library files
- ~58,000 lines of code
- Quality score: 9.8/10

---

## [v0.3.8] - August 15, 2025

**Status**: Previous Stable Release

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

## [v0.3.7] - July 30, 2025

**Status**: Legacy Release

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

- **Current stable (v0.4.0)**: Full support
- **Previous stable (v0.3.9)**: Security updates for 6 months
- **Beta releases**: Community support
- **Alpha releases**: No support

## Upcoming Releases

### v0.4.1 (Q1 2026) - Service Management
- `nself email` - Email service configuration
- `nself search` - Search engine management
- `nself functions` - Serverless functions management
- `nself mlflow` - MLflow management

### v0.4.2 (Q2 2026) - Monitoring Management
- `nself metrics` - Complete monitoring stack management
- `nself monitor` - Access monitoring dashboards

### v0.4.3 (Q2 2026) - Database Operations
- `nself db` - Database operations and optimization

### v0.4.4 (Q3 2026) - Backup & Restore
- `nself backup` - Backup creation and management
- `nself rollback` - Rollback capabilities

### v0.4.5 (Q3 2026) - Production Features
- `nself prod` - Production configuration
- `nself deploy` - SSH deployment automation

### v0.4.6 (Q4 2026) - Scaling
- `nself scale` - Scaling management

### v0.5+ (2027+) - Advanced Features
- Kubernetes deployment support
- Multi-node clustering
- Plugin system
- Cloud provider abstractions
- CI/CD integrations

## Archives

Older releases and their documentation can be found in the [GitHub Releases](https://github.com/acamarata/nself/releases) page.

---

*For the latest version, run `nself version` or check [src/VERSION](../../src/VERSION)*
