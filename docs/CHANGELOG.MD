# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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