# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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