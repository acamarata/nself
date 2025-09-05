# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.9] - 2025-09-05 (Patch Updates)

### Fixed (Patch)
- **Build Command**: Fixed docker-compose generation hanging issue
  - Added timeout protections for service generation scripts
  - Fixed .env.local sourcing error in compose-generate.sh
  - Resolved infinite loops in service builder scripts
- **Smart Defaults**: Added missing `generate_password()` function
- **Environment Loading**: Fixed recursive environment loading in build process
- **Service Generation**: Restored functionality with proper timeout handling

## [0.3.9] - 2025-09-03

### Added
- **Admin UI Integration**: Full integration with nself-admin Docker image
  - Web-based monitoring at localhost:3100
  - Real-time service health checks
  - Docker container management interface
  - Database query interface
  - Log viewer with filtering
  - Backup management UI
- **Enhanced Init Command**:
  - Minimal default setup (just .env.example + .env.local)
  - `--full` flag for complete environment setup
  - `--wizard` flag for interactive configuration
  - `--admin` flag for minimal admin UI setup
  - Automatic .gitignore management
- **Database Schema Template**: Added schema.dbml with example tables
- **Service Templates**: Added 40 production-ready service templates for various languages and frameworks
- **Improved Reset Command**: Timestamp-based backup folders (_backup/YYYYMMDD_HHMMSS/)

### Changed
- **Environment Loading Priority**: Fixed to match documentation (dev → staging/prod → local → .env → secrets)
- **Environment Templates**: All templates updated with clean 60-char headers and smart defaults
- **Build Command**: Simplified validation to prevent timeout issues
- **Init Command**: Default behavior now creates minimal setup (breaking change)
- **Reset Command**: Now uses organized timestamp subfolders in _backup/

### Removed
- **Config-Server**: Completely removed obsolete Nhost dashboard integration
- Old JavaScript service templates (replaced with stubs)
- All config-server related code and references

### Fixed
- Status command hanging issue
- Stop command compose wrapper calls
- Exec command container detection
- Build command validation timeout
- Doctor command function references
- SSL certificate generation for nginx
- Container name resolution issues
- Environment loading priority bugs
- Display.sh unbound variable errors
- Diff.sh unbound variable error

### Known Issues
- Auth service health check reports unhealthy (service works correctly on port 4001)
- Some commands may require PROJECT_NAME environment variable

## [0.3.8] - 2024-08-17

### Added
- **Admin UI Integration**: Web-based administration interface
  - `nself admin` - Complete admin UI management system
  - Visual service monitoring and control
  - Configuration editing through web interface
  - Real-time status dashboard
  - Password management and authentication

- **Enterprise Search**: Multi-engine search capabilities
  - `nself search` - Search service management with 6 engine options
  - PostgreSQL FTS, MeiliSearch, Typesense, Elasticsearch, OpenSearch, Sonic support
  - Interactive setup wizard for optimal engine selection
  - Built-in testing and configuration validation

- **VPS Deployment**: SSH-based deployment system
  - `nself deploy` - Complete deployment management
  - Support for DigitalOcean, Linode, Vultr, Hetzner, and Ubuntu/Debian VPS
  - Automated Docker installation and service setup
  - GitHub webhook integration for continuous deployment
  - Built-in rollback and monitoring capabilities

- **Interactive Project Setup**: Enhanced initialization experience
  - `nself init --wizard` - Interactive setup wizard
  - Automatic framework and language detection
  - Smart service recommendations based on project type
  - 6 pre-configured templates: SaaS, E-commerce, Blog/CMS, API, Mobile, Custom
  - Project-specific environment configuration

- **Custom Services Framework (CS_N Pattern)**: Simplified service creation
  - CS_N=name,framework pattern for defining services
  - 20+ supported frameworks (js, py, go, rb, php, java, rust, lua, etc.)
  - Popular framework support (nest, django, flask, rails, laravel, fastify)
  - Queue worker support (bull, celery, sidekiq)
  - Complete hello-world templates for all frameworks
  - Auto-generated docker-compose.custom.yml

- **Frontend Applications Support**: SPA and frontend project configuration
  - FRONTEND_APPS configuration for multiple frontend applications
  - Independent routing and deployment per app
  - Database isolation with table prefixes
  - Production deployment to Vercel, Netlify, Cloudflare
  - Remote schema integration with Hasura
  - Environment-specific routing

- **Enhanced Environment Management**: Multi-environment support
  - Environment-specific files (.env.local, .env.staging, .env.prod)
  - Secure secrets management with .env.secrets
  - Environment compilation and validation
  - Production security checks and warnings
  - Hierarchical configuration system

### Changed
- **File Structure**: Added wizard directory with interactive components
- **Environment Variables**: Extended .env.example with 150+ new configuration options
- **Help System**: Updated to include all v0.3.9 commands
- **Command Router**: Enhanced nself.sh to support new command structure

### Security
- **Password Management**: Secure password generation for production environments
- **Environment Secrets**: Separate .env.secrets file for sensitive data
- **Admin Authentication**: Password hashing and session management
- **Production Validation**: Security checks for production deployments

## [0.3.8] - 2024-08-17

### Added
- **Enterprise Monitoring & Management**:
  - `nself validate` - Configuration validation with auto-fix capabilities
  - `nself exec` - Execute commands in containers with full Docker exec support
  - `nself metrics` - Metrics collection and observability with provider support
  - `nself monitor` - Real-time monitoring dashboard with interactive terminal UI
  - `nself rollback` - Comprehensive rollback system for migrations, deployments, and configs
  - `nself scale` - Resource scaling and management with auto-scaling capabilities
  
- **Enhanced Backup System**:
  - 10 backup subcommands (create, list, restore, prune, verify, schedule, export, import, snapshot, rollback)
  - S3 backup support with multiple providers
  - Incremental and differential backups
  - Point-in-time recovery
  - Backup integrity verification
  
- **Automatic SSL System**:
  - Fully automatic SSL certificate generation
  - Auto-detection of all domains from nginx configs and microservices
  - Support for wildcard certificates
  - Let's Encrypt production support with DNS providers
  - Zero-configuration trusted certificates

### Changed
- **Commands**: Complete refactoring of command structure for consistency
- **Documentation**: Updated all documentation to reflect new features
- **Error Handling**: Improved error messages and recovery options

### Fixed
- **SSL**: Fixed certificate generation for complex domain setups
- **Backup**: Fixed S3 backup compatibility issues
- **Monitor**: Fixed terminal UI rendering on different terminals

## [0.3.7] - 2025-08-15

### Added
- **Enterprise-Ready Features**: Major improvements for production deployments
  - Comprehensive backup and restore system with local and S3 support
  - Full disaster recovery capabilities with point-in-time backups
  - Automated backup pruning and scheduled backup support
  
- **CI/CD Pipeline**: Complete GitHub Actions workflow
  - ShellCheck linting for all shell scripts
  - Bats unit testing framework integration
  - Integration tests across Ubuntu and macOS
  - Security scanning with Trivy
  - Automated testing on push and pull requests
  
- **Multiple Installation Methods**: Expanded platform support
  - Homebrew formula for macOS users
  - .deb packages for Debian/Ubuntu
  - .rpm packages for RHEL/CentOS/Fedora
  - Package build automation scripts
  
- **Enhanced CLI Commands**:
  - `nself backup` - Create, restore, list, and prune backups
  - `nself doctor` - Comprehensive system diagnostics with actionable fixes
  - `nself logs` - Advanced log viewing with filtering, grep, and follow mode
  - Enhanced status command with service health visualization
  
- **Production Improvements**:
  - System health diagnostics with remediation suggestions
  - Memory and disk space checks
  - Port conflict detection with process identification
  - DNS resolution verification
  - SSL certificate validation

### Fixed
- **License Clarity**: Resolved inconsistency between README and LICENSE file
  - Changed from conflicting "MIT License" reference to clear "Source Available License"
  - Aligned all license documentation for legal compliance
  
### Changed
- **Documentation**: Clarified licensing terms as Source Available
- **Commands**: Enhanced existing commands with better error handling and output
- **Testing**: Improved test coverage and CI/CD reliability

### Security
- Added Trivy security scanning in CI pipeline
- Improved secret handling in backup/restore operations
- Enhanced SSL certificate management and validation

## [0.3.6] - 2025-08-15

### Added
- **Major Refactoring**: Complete overhaul of core systems for improved reliability
- **Critical Bug Fixes**: Resolved multiple stability issues affecting production deployments

## [0.3.5] - 2025-08-15

### Added
- **Complete SSL/HTTPS Support**: Full HTTPS implementation for development and production
  - Automatic SSL certificate generation using mkcert for development environments
  - Support for both `*.localhost` and `*.local.nself.org` domain patterns
  - Wildcard SSL certificates with proper Subject Alternative Name (SAN) configuration
  - HTTP to HTTPS redirect for all services
  - Modern TLS configuration with secure cipher suites and protocols

- **SSL Certificate Management**
  - `nself trust` command to install SSL certificates in system trust store
  - Automatic certificate generation during build process
  - Domain-specific certificate selection (localhost vs nself.org patterns)
  - Certificate validation and health checking
  - Browser-ready green lock compatibility

- **Enhanced Service Architecture**
  - 100% service success rate (17/17 services running reliably)
  - Improved volume mount strategy preserving node_modules and dependencies
  - Reserved port allocation (3000-3099 for user apps, 3100+ for services)
  - Enhanced service health monitoring and auto-recovery

### Fixed
- **Critical Volume Mount Issues**
  - Node.js services now preserve installed dependencies during development
  - Changed volume mounts from entire directories to source-only mounting
  - Resolved "Cannot find module" errors in NestJS, BullMQ, and Functions services

- **SSL and Nginx Configuration**
  - Fixed nginx upstream definition conflicts causing restart loops
  - Resolved SSL certificate Subject Alternative Name (SAN) configuration
  - Fixed dynamic nginx includes with static configuration approach
  - Corrected certificate path references in nginx configurations

- **Service Reliability**
  - Docker health checks now work without 'which' command dependency
  - Eliminated port allocation conflicts between user and system services
  - Fixed service startup reliability from 76% to 100% success rate
  - Resolved container restart loops and networking issues

### Changed
- **Default HTTPS Everywhere**: All services now use HTTPS by default in development
- **Smart Port Allocation**: System services moved to 3100+ range, 3000-3099 reserved for users
- **Volume Mount Strategy**: Source-only mounting preserves container dependencies
- **SSL Trust Integration**: Certificates automatically trusted during build process

### Enhanced
- **Developer Experience**
  - Seamless HTTPS development workflow with automatic certificate trust
  - Green lock icons in browsers without manual certificate installation
  - Improved error handling and auto-fix capabilities for SSL issues
  - Enhanced service status reporting with SSL health indicators

## [0.3.4] - 2025-08-14

### Added
- **Standardized Command Headers**: All nself commands now use consistent header formatting
  - Unified blue borders with bold titles and dim subtitles
  - Centralized `show_command_header` function for consistency
  - Professional boxed headers across all CLI commands

### Fixed
- **Header Formatting Issues**
  - Removed extra spacing after title and subtitle lines
  - Fixed color rendering in non-TTY environments
  - Corrected header positioning in `nself up` to show before pre-flight checks
  - Eliminated artifacts in database schema ready messages

- **Service Detection**
  - Fixed `nself down` container detection using direct docker ps instead of docker compose
  - Resolved false "stopped" status for running containers
  - Improved container state detection accuracy

- **Auto-fix Syntax Errors**
  - Fixed integer comparison errors in comprehensive.sh
  - Cleaned grep output to prevent multi-line variable issues
  - Resolved "syntax error in expression" during health checks

- **nginx Configuration**
  - Fixed storage service port mismatch (5000 → 5001)
  - Updated deprecated http2 directive to modern syntax
  - Resolved nginx restart loops during auto-fix

### Improved
- **Command Output Standards**
  - Consistent header formatting across init, build, up, down, status, doctor, clean, restart
  - Proper ANSI color preservation in all environments
  - Better visual hierarchy with standardized spacing

## [0.3.3] - 2025-08-14

### Added
- **ALWAYS_AUTOFIX Mode**: Automatic fixing is now the default behavior
  - System defaults to `ALWAYS_AUTOFIX=true` unless explicitly set to `false`
  - Concise output mode with inline status updates when auto-fix is enabled
  - Up to 30 retry attempts in auto-fix mode for comprehensive resolution
  - Interactive mode still available by setting `ALWAYS_AUTOFIX=false`

- **Docker Resource Management**
  - New `nself clean` command for Docker cleanup operations
  - Proper image tagging with project name and environment
  - Support for cleaning containers, images, volumes, and networks
  - Prevents duplicate image buildup with consistent naming conventions

- **Comprehensive Service Auto-Generation**
  - Automatic generation of missing NestJS, BullMQ, Go, and Python services
  - Smart detection of service types from directory structure
  - Complete hello-world implementations with proper Dockerfiles
  - Seamless continuation after service generation without manual intervention

- **Advanced Port Conflict Resolution**
  - Upfront port checking before Docker startup
  - Interactive numbered options for conflict resolution
  - Automatic port reassignment in ALWAYS_AUTOFIX mode
  - Intelligent port mapping for Redis, Mailpit, and other services

### Fixed
- **Build System Robustness**
  - Silent build failures now properly report errors
  - Build command is fully idempotent and handles partial builds
  - Validation runs in subshell to prevent script exits
  - Proper handling of existing infrastructure

- **Service Startup Reliability**
  - Automatic recovery from unhealthy services
  - Missing Dockerfile detection and generation
  - Go module dependency resolution
  - Continuous retry mechanism for transient failures

### Improved
- **Output Formatting**
  - Lowercase headers following nself theme standards
  - Compact status indicators (✓, ⚡, ⠋, ✗) in auto-fix mode
  - Cleaner spacing and alignment in all outputs
  - Reduced verbosity for automated deployments

### Changed
- **Default Behavior**
  - ALWAYS_AUTOFIX is now `true` by default
  - System auto-fixes issues unless explicitly disabled
  - More automation-friendly for CI/CD environments

## [0.3.2] - 2025-08-12

### Fixed
- **Command Resolution Bug**: Fixed critical SCRIPT_DIR variable corruption during sourcing
  - Commands like `nself --version` and `nself help` now work correctly
  - Resolved issue where sourced utility files overwrote CLI script directory path
  - Improved command routing reliability across all nself commands

### Improved  
- **Install Script Enhancements**
  - Fixed banner alignment and formatting in installation output
  - Corrected version display (removed duplicate "v" prefix)
  - Shortened "Prerequisites Check" header to fit standard terminal widths
  - Enhanced installation script robustness and professional appearance

## [0.3.1] - 2025-01-12

### Added
- **Enhanced Configuration Validation System**
  - 25+ comprehensive validation checks for .env files
  - Automatic detection and fixing of 22+ common configuration issues
  - Empty file handling with automatic minimal config creation
  - Port range validation (1-65535) with conflict detection
  - Special character escaping in passwords
  - Quote mismatch detection and repair
  - Duplicate variable detection and removal
  - Inline comment cleanup for proper parsing

- **Comprehensive Auto-Fix System** (`config-validator-v2.sh`, `auto-fixer-v2.sh`)
  - 25+ automated fix strategies for common configuration mistakes
  - Safe backup creation before applying fixes
  - Detailed progress reporting during fix application
  - Port conflict resolution with alternative port suggestions
  - Docker naming convention validation and correction
  - Memory format validation (512M, 2G format checking)

- **Professional Output Formatting**
  - Perfect text alignment in all boxed messages and banners
  - Consistent 3-space indentation after border characters
  - Properly aligned right-side borders (`║`) throughout interface
  - Beautiful Unicode box-drawing characters
  - Color-coded indicators for errors, warnings, and success

- **Advanced Validation Features**
  - IP address format validation
  - Timezone format checking
  - SSL configuration validation (cert/key path verification)
  - File path existence checking
  - Service list format validation
  - Boolean value normalization
  - JWT key length validation (32+ character minimum)

### Changed
- **Environment File Priority System**
  - Strict priority enforcement: `.env` > `.env.local` > `.env.dev`
  - Complete file isolation - no merging between environment files
  - Higher priority files completely override lower priority ones
  - Smart defaults applied only after loading the chosen file

- **Update System**
  - Now pulls exclusively from GitHub releases instead of development branch
  - Ensures users only get stable, tested versions
  - Updated rsync paths to work with new src structure

- **Template Formatting**
  - All `.env.example` and `.env.local` templates updated with perfect alignment
  - Consistent header formatting across all template files
  - Professional appearance suitable for public screenshots

### Fixed
- Empty `.env.local` files causing build failures
- Port numbers >65535 or negative values crashing services
- Leading/trailing whitespace in configuration values
- Quote mismatches breaking configuration parsing
- Special characters in passwords causing shell escaping issues
- Duplicate variables causing undefined behavior
- Inline comments interfering with value parsing
- Invalid IP addresses in host configurations
- Malformed memory specifications causing container issues
- Docker naming violations preventing container creation
- Missing file references breaking SSL configurations
- Mixed case boolean values causing validation failures
- Service list formatting issues with commas and spaces
- Project names with spaces breaking Docker networks
- Weak password detection and automatic replacement
- Environment priority confusion when multiple files exist
- Text alignment issues in all UI components

### Documentation
- Added comprehensive environment configuration guide (`ENVIRONMENT_CONFIGURATION.md`)
- Updated validation error reference with solutions
- Enhanced troubleshooting documentation
- Auto-fix strategy documentation

## [0.3.0] - 2025-01-11

### Changed (BREAKING)
- **Major Architecture Refactor**: Moved from monolithic to modular src-centric architecture
  - All implementation code now lives in `/src` with organized subdirectories
  - `/bin` contains only thin shim scripts that delegate to `/src/cli`
  - Templates, certificates, and libraries moved to `/src`
  - Clean separation of concerns with modular error handling and auto-fix systems

### Added
- **Comprehensive Error Handling System** (`/src/lib/errors/`)
  - Modular error detection and reporting
  - Auto-fix capabilities for common issues
  - Interactive user prompts for complex fixes
  - Port conflict resolution with automatic alternative port configuration
  - Docker build error analysis and recovery
  - Go module dependency resolution
  - Node.js build error handling

- **Modular Build System** (`/src/cli/build/`)
  - Broke down 1078-line monolithic build.sh into maintainable modules
  - Separate modules for environment, Docker, services, nginx, SSL, and Hasura
  - Improved error detection and recovery during build process

- **Enhanced Status and Logging** (`/src/cli/status.sh`, `/src/cli/logs.sh`)
  - Real-time service health monitoring
  - Color-coded status indicators
  - Improved log filtering and display
  - Progress indicators for long-running operations

- **Doctor Command** (`/src/cli/doctor.sh`)
  - Comprehensive system health checks
  - Environment validation
  - Service connectivity testing
  - Auto-fix suggestions for common issues

### Fixed
- Installation upgrade path from v0.2.x to v0.3.0
- Bash 3.2 compatibility for macOS (removed associative arrays)
- Port conflict detection now uses configured ports from .env.local
- Go module build errors with automatic dependency resolution
- Template path references updated for new src structure
- Safety checks for nself repository detection
- Shebang standardization to `#!/usr/bin/env bash`
- Auto-fix subsystem now uses absolute paths

### Improved
- Performance optimizations for quick checks
- Docker Desktop auto-start on macOS
- Interactive menus for complex configuration choices
- Error messages are more descriptive and actionable
- Build process is more resilient to failures
- Version detection now properly reads from /src/VERSION

## [0.2.4] - 2025-01-10
- Comprehensive email provider support
- Multiple provider configurations

## [0.2.3] - 2025-01-09
- Critical fixes and automatic SSL trust
- Installation improvements

## [0.2.2] - 2025-01-08
- UI improvements and fixes

## [0.2.1] - 2025-01-07
- Database tools for different environments
- Enhanced dbsyncs functionality

## [0.2.0] - 2025-01-06
- Initial modular refactoring
- Basic error handling
- Docker compose generation

## [0.1.0] - 2025-01-01
- Initial release
- Basic nself functionality
- Docker-based development environment