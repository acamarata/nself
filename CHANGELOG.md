# Changelog

## [0.2.1] - 2025-08-08

### Added
- **Simplified Environment Management**: New `ENV` variable (dev/prod) for user simplicity
  - `ENV=dev` for development (default)
  - `ENV=prod` for production (auto-configures security)
  - Core project variables now at top of `.env.example`
  - Maps internally to Hasura/PostgreSQL standards (development/production)
- **Standards-Compliant Database Seeding**: Enhanced `DB_ENV_SEEDS` option
  - When `true`: Uses Hasura/PostgreSQL standard structure (common/ + environment-specific)
  - When `false`: Single default/ directory for all environments
  - Follows industry best practices while keeping configuration simple

### Changed
- **Configuration Organization**: Core settings (ENV, PROJECT_NAME, BASE_DOMAIN, DB_ENV_SEEDS) now at top of `.env.example`
- **Auto-Configuration**: Many settings now automatically adjust based on ENV (Hasura console, dev mode, etc.)
- **Improved Documentation**: Updated README and DBTOOLS with new ENV and seeding strategies

### Fixed
- Environment variable consistency across all scripts
- Backward compatibility maintained with ENVIRONMENT variable

## [0.2.0] - 2025-08-06

### Added
- **Database Tools**: Complete database management system under `nself db`
  - `nself db run` - Generate migrations from schema.dbml
  - `nself db update` - Safe migration updates for team members
  - `nself db sync` - Sync from dbdiagram.io
  - `nself db revert` - Rollback to previous backup
  - Automatic backups to `/bin/dbsyncs/` with timestamps
  - Migration warnings in `nself up` output
- **Enhanced Terminal UI**: Professional output with spinners and progress indicators
- **Comprehensive Documentation**: Added DBTOOLS.md with 1500+ lines of database documentation

### Changed
- **Improved Reset Command**: Complete Docker resource cleanup
  - Removes all project containers, volumes, and networks
  - Backs up .env.local to .env.old
  - Requires typing "RESET" for safety
- **Script Organization**: Removed 'nself-' prefix from all bin scripts
- **Better Build Output**: Shows progress with spinner animations
- **Cleaner Service Management**: All commands now show professional progress indicators

### Fixed
- Docker container namespacing with PROJECT_NAME
- Environment variable loading in various scenarios
- Migration detection and warning system

### Security
- Removed auto-migration feature for production safety
- All database changes now require explicit confirmation
- Enhanced production environment checks

## [0.1.0] - 2025-08-03

### Initial Release
- Core nself CLI for self-hosted Nhost stack
- Docker Compose generation
- Basic service management (up, down, restart)
- SSL/TLS support with Let's Encrypt
- Multi-environment configuration