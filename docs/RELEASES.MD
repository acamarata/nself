# nself Releases

## [v0.3.9] - August 17, 2024

### 🚀 Platform Evolution Release

nself v0.3.9 transforms nself from a CLI tool into a comprehensive backend platform with visual administration, enterprise search, and automated deployment capabilities.

### ✨ What's New

#### Admin UI Integration
- **Web-based Administration Interface** for complete visual stack management
- Real-time service monitoring and control through browser
- Configuration file editing via web interface
- Password-protected access with secure authentication
- Seamless integration with existing nself projects

#### Enterprise Search Capabilities
- **Multi-engine search support** with 6 different options
- PostgreSQL FTS, MeiliSearch, Typesense, Elasticsearch, OpenSearch, and Sonic
- Interactive setup wizard for optimal engine selection
- Built-in testing and configuration validation
- Zero-config setup for most use cases

#### VPS Deployment System
- **SSH-based deployment** to any Ubuntu/Debian server
- Support for major VPS providers (DigitalOcean, Linode, Vultr, Hetzner)
- Automated Docker installation and service setup
- GitHub webhook integration for continuous deployment
- Built-in rollback and monitoring capabilities

#### Interactive Project Setup
- **Init Wizard** for guided project configuration
- Automatic framework and language detection (JavaScript, Python, Go, Ruby, PHP, Java, .NET)
- Smart service recommendations based on project type
- 6 pre-configured templates: SaaS, E-commerce, Blog/CMS, API, Mobile, Custom
- Multi-environment support (development, staging, production)

#### Enhanced Environment Management
- **Multi-environment configuration system**
- Environment-specific files (.env.local, .env.staging, .env.prod)
- Secure secrets management with .env.secrets
- Environment compilation and validation
- Production security checks and warnings

### 🛠️ New Commands

```bash
# Admin UI Management
nself admin enable          # Enable admin web interface
nself admin status          # Check admin status
nself admin password        # Set admin password
nself admin open            # Open admin in browser

# Enterprise Search
nself search enable         # Enable search with engine selection
nself search setup          # Interactive search configuration
nself search test           # Test search functionality
nself search docs           # View search documentation

# VPS Deployment  
nself deploy init           # Setup deployment configuration
nself deploy ssh            # Deploy to VPS server
nself deploy status         # Check deployment status
nself deploy webhook        # Setup continuous deployment

# Interactive Setup
nself init --wizard         # Launch interactive project setup
```

### 🔄 Migration from v0.3.8

nself v0.3.9 is **fully backward compatible**:
- All existing functionality unchanged
- New features are opt-in and disabled by default
- Existing `.env.local` files continue to work
- No configuration changes required

**To upgrade:**
```bash
nself update
```

### 📦 Installation

```bash
curl -fsSL https://get.nself.org | bash
```

### 🔗 Documentation

- [Complete Commands Reference](./COMMANDS.md)
- [v0.3.9 Release Notes](./v0.3.9.md)
- [Environment Configuration Guide](./ENVIRONMENT_CONFIGURATION.md)

---

## [v0.3.8] - August 17, 2024

### 🎯 Intelligent Auto-Fix System

This release introduces automatic fixing as the default behavior, making nself more intelligent and user-friendly. The system now automatically resolves configuration issues, port conflicts, and missing services without manual intervention.

### ✨ What's New

#### Default Auto-Fix Behavior
- **ALWAYS_AUTOFIX=true** is now the default setting
- System automatically resolves issues during startup
- Up to 30 retry attempts for comprehensive problem resolution
- Concise output mode with inline status updates
- Interactive mode still available with `ALWAYS_AUTOFIX=false`

#### Docker Resource Management
- **New `nself clean` command** for resource cleanup
- Proper image tagging prevents duplicate buildup
- Clean containers, images, volumes, and networks
- Project-specific cleanup with safety checks

#### Comprehensive Service Generation
- Automatic creation of missing services during startup
- Support for NestJS, BullMQ, Go, and Python services
- Complete hello-world implementations with Dockerfiles
- Seamless continuation without manual intervention

#### Advanced Port Management
- Upfront port checking before Docker startup
- Interactive numbered options for conflict resolution
- Automatic port reassignment in auto-fix mode
- Intelligent mapping for all service types

### 🔧 Improvements

- Silent build failures now properly report errors
- Fully idempotent build command
- Automatic recovery from unhealthy services
- Missing Dockerfile detection and generation
- Go module dependency resolution
- Lowercase headers following nself standards
- Compact status indicators in auto-fix mode

### 📦 Installation

```bash
curl -fsSL https://get.nself.org | bash
```

### 💡 Usage

```bash
# Default behavior - auto-fixes everything
nself up

# Use interactive mode
ALWAYS_AUTOFIX=false nself up

# Clean Docker resources
nself clean
```

---

## [v0.3.2] - August 12, 2025

### 🐛 Critical Command Resolution Fix

Fixed a critical bug where the SCRIPT_DIR variable was being corrupted during utility sourcing, causing commands like `nself --version` and `nself help` to fail.

### ✨ Improvements

- Fixed command resolution reliability
- Enhanced install script formatting
- Improved banner alignment

---

## [v0.3.0] - August 11, 2025

### 🚀 Major Architecture Refactor

This release represents a complete architectural overhaul, moving from a monolithic structure to a clean, modular src-centric design. This is a breaking change that requires updating existing installations.

### ✨ What's New

#### Complete Architectural Redesign
- **Modular Structure**: All implementation code now lives in `/src` with organized subdirectories
- **Thin Shims**: `/bin` contains only lightweight delegation scripts pointing to `/src/cli`
- **Clean Separation**: Templates, certificates, and libraries properly organized under `/src`
- **Maintainable Code**: 1078-line monolithic build.sh broken into focused, manageable modules

#### Comprehensive Error Handling System
- **Smart Detection**: Modular error detection and reporting system
- **Auto-Fix Capabilities**: Automatic resolution for common issues
- **Interactive Recovery**: User-friendly prompts for complex fixes
- **Port Management**: Automatic alternative port configuration when conflicts detected
- **Build Recovery**: Intelligent analysis and recovery from Docker build errors
- **Dependency Resolution**: Automatic Go module and Node.js dependency fixes

#### Enhanced Developer Experience
- **Doctor Command**: Comprehensive system health checks with auto-fix suggestions
- **Improved Status**: Real-time health monitoring with color-coded indicators
- **Better Logging**: Enhanced log filtering and progress indicators
- **Performance**: Optimized quick checks for faster operations

### 🔧 Technical Improvements

- Bash 3.2 compatibility for macOS (removed associative arrays)
- Proper port conflict detection using configured values
- Robust Go module build error handling
- Docker Desktop auto-start on macOS
- Standardized shebangs (`#!/usr/bin/env bash`)
- Absolute path usage in auto-fix subsystem

### 📦 Installation

For new installations:
```bash
curl -sSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

For upgrading from v0.2.x:
```bash
nself update
```

### ⚠️ Breaking Changes

This is a major version with breaking changes. The entire directory structure has been reorganized. Existing installations will be automatically updated when running `nself update`.

---

## [v0.2.4] - August 9, 2025

### 📧 Comprehensive Email Provider Support

This release adds extensive email provider support with automatic configuration for multiple services.

### ✨ Features

- **Multiple Email Providers**: Support for SendGrid, Mailgun, AWS SES, Postmark, and more
- **Automatic Configuration**: Smart detection and setup based on provider selection
- **Development Tools**: Enhanced MailPit integration replacing deprecated MailHog
- **Provider Templates**: Pre-configured templates for each email service
- **Environment Detection**: Different configurations for development vs production

### 🔧 Improvements

- Simplified email configuration in `.env.local`
- Better error messages for email configuration issues
- Automatic SMTP settings based on provider
- Backward compatibility with MailHog installations

---

## [v0.2.3] - August 9, 2025

### 🔒 Critical Fixes and SSL Improvements

Security and reliability improvements with automatic SSL certificate trust.

### ✨ Features

- **Automatic SSL Trust**: Self-signed certificates automatically trusted on macOS
- **Installation Improvements**: More robust installation process
- **Error Recovery**: Better handling of installation failures

### 🐛 Bug Fixes

- Fixed certificate generation on fresh installations
- Resolved permission issues during installation
- Fixed Docker network creation race conditions
- Improved error messages for common issues

### 🔧 Improvements

- Faster installation process
- Better detection of existing installations
- Cleaner upgrade path from older versions

---

## [v0.2.2] - August 8, 2025

### 🎨 UI Improvements and Fixes

Focus on user interface enhancements and bug fixes.

### ✨ Features

- **Better Output Formatting**: Improved command output with better colors and formatting
- **Progress Indicators**: Visual feedback for long-running operations
- **Status Command**: Enhanced status display with service health indicators

### 🐛 Bug Fixes

- Fixed output alignment issues in status command
- Resolved color display problems on certain terminals
- Fixed progress bar display during builds
- Corrected service health check timing

### 🔧 Improvements

- Cleaner console output
- Better error message formatting
- Improved command response times

---

## [v0.2.1] - August 7, 2025

### 🗄️ Database Tools Enhancement

Enhanced database management tools for different environments.

### ✨ Features

- **DB Sync Tools**: New `nself db sync` command for database synchronization
- **Environment Support**: Separate handling for development, staging, and production
- **DBML Integration**: Database schema management with DBML support
- **Migration Tools**: Improved Hasura migration handling

### 🐛 Bug Fixes

- Fixed database connection string generation
- Resolved migration ordering issues
- Fixed schema initialization problems

### 🔧 Improvements

- Better database error messages
- Faster schema synchronization
- Improved migration rollback support

---

## [v0.2.0] - August 6, 2025

### 🏗️ Initial Modular Refactoring

First step towards modular architecture with basic error handling.

### ✨ Features

- **Modular Commands**: Separated commands into individual files
- **Basic Error Handling**: Initial error detection and reporting
- **Docker Compose Generation**: Dynamic docker-compose.yml generation
- **Service Management**: Improved service start/stop operations

### 🔧 Improvements

- Better code organization
- Reduced code duplication
- Improved maintainability
- Basic auto-fix for common issues

### 🐛 Bug Fixes

- Fixed service dependency ordering
- Resolved environment variable expansion issues
- Fixed Docker network creation problems

---

## [v0.1.0] - August 6, 2025

### 🎉 Initial Release

The first public release of nself - a comprehensive local development environment for modern web applications.

### ✨ Core Features

- **One-Command Setup**: Initialize a complete development stack with `nself init`
- **Service Integration**: PostgreSQL, Hasura, Auth, Storage, and Nginx pre-configured
- **SSL Support**: Automatic SSL certificate generation for local development
- **Docker-Based**: Fully containerized for consistency across environments
- **Multi-Service Support**: Built-in support for NestJS, Go, Python, and BullMQ workers

### 📦 Included Services

- **PostgreSQL**: Database with automatic initialization
- **Hasura**: GraphQL engine with migrations and metadata
- **Authentication**: Hasura Auth with JWT support
- **Storage**: MinIO for S3-compatible object storage
- **Nginx**: Reverse proxy with automatic routing
- **Redis**: Optional caching and queue support
- **Email**: Development email with MailHog

### 🛠️ Commands

- `nself init` - Initialize a new project
- `nself build` - Build the project infrastructure
- `nself up` - Start all services
- `nself down` - Stop all services
- `nself status` - Check service status
- `nself logs` - View service logs
- `nself reset` - Reset the project

### 📚 Documentation

- Complete installation guide
- Service configuration documentation
- Troubleshooting guide
- Example projects

### 🎯 Target Audience

Developers building modern web applications who want a production-like development environment without the complexity of manual setup.

---

## Version History Summary

| Version | Date         | Type  | Focus                            |
|---------|--------------|-------|----------------------------------|
| v0.3.0  | 2025-08-11   | Major | Complete architectural refactor |
| v0.2.4  | 2025-08-09   | Minor | Email provider support           |
| v0.2.3  | 2025-08-09   | Patch | SSL and installation fixes      |
| v0.2.2  | 2025-08-08   | Patch | UI improvements                  |
| v0.2.1  | 2025-08-07   | Patch | Database tools                   |
| v0.2.0  | 2025-08-06   | Minor | Initial modularization           |
| v0.1.0  | 2025-08-06   | Major | Initial release                  |