# Changelog

All notable changes to nself will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased] - Post v0.9.8

### 🔴 Critical Fixes

#### Installation System
- **CRITICAL**: Fixed `install.sh` not deploying latest code from main branch
  - Added `INSTALL_LATEST=true` environment variable support
  - Fixed branch archive URL (was using `/tags/` instead of `/heads/`)
  - Users can now get latest fixes: `curl -fsSL install.sh | INSTALL_LATEST=true bash`
  - Commit: `0050dca`

#### Deployment System
- **CRITICAL**: Fixed infinite hang at .env sync during deployment
  - Added SSH connection test with 10-second timeout before file transfers
  - Added 30-second timeout to all scp operations
  - Proper error capture using temp files instead of blocking command substitution
  - Clear timeout indicators in error messages
  - Commit: `0050dca`

- **CRITICAL**: Fixed bash syntax error in `secure-defaults.sh`
  - Command substitution was capturing unicode characters (✓) causing arithmetic errors
  - Changed to redirect stderr to parent process: `2>&2`
  - Error: `syntax error: operand expected (error token is "✓ POSTGRES_PASSWORD: Set...")`
  - Commit: `97bf107`

- **CRITICAL**: Fixed database deployment not executing during deployment
  - Step 6 condition was checking for local `hasura` directory instead of remote
  - Now properly checks remote server for hasura files before attempting deployment
  - Ensures migrations, seeds, and metadata apply on remote during `nself deploy sync full`
  - Database automation now runs reliably on all deployment targets
  - Commit: `65d3196`

- **CRITICAL**: Fixed environment-aware security validation
  - Security check was blocking staging deployments (ports on 0.0.0.0 needed for nginx)
  - Now only runs strict port validation in production environments
  - Staging/dev: Informational message only (allows nginx proxying)
  - Production: Fails deployment if sensitive ports exposed (redis, postgres, meilisearch, minio)
  - Prevents false-positive security rollbacks in non-production environments
  - Commit: `65d3196`

- **CRITICAL**: Fixed .env file sync failure (Round 4 - Bug #9)
  - scp requires `-P` (capital) for port flag, ssh uses `-p` (lowercase)
  - Error: "scp: stat local '22': No such file or directory"
  - Port number was treated as filename causing all file transfers to fail
  - Created separate `ssh_args` and `scp_args` arrays with correct flags
  - .env and .env.secrets now sync correctly to remote server
  - Commit: `0590a36`

- **CRITICAL**: Fixed services not ready during database automation (Round 4 - Bug #7)
  - `docker compose up -d` returns immediately but containers need 5-30s to start
  - Step 6 database automation ran before postgres was accepting connections
  - Added health check wait using `pg_isready` between Step 5 and Step 6
  - Waits up to 60 seconds for database to be ready before proceeding
  - Database commands now execute when services are actually healthy
  - Commit: `0590a36`

- **CRITICAL**: Fixed environment variables not available during deployment (Round 4 - Bug #8)
  - SSH non-interactive shells don't automatically source .env files
  - Database commands failed with "unbound variable" errors (HASURA_GRAPHQL_ADMIN_SECRET, etc.)
  - Now explicitly loads .env and .env.secrets before running nself CLI commands
  - Uses `set -a / set +a` to auto-export all environment variables
  - Database automation commands now have access to all required configuration
  - Commit: `0590a36`

### 🆕 Features

#### Plugin System
- **Plugin Dependency Management** (`nself plugin check-deps`, `nself plugin install-deps`)
  - Auto-detect package manager (apt, brew, yum, pacman)
  - Support for space-delimited package strings: `"apt": "python3 python3-pip"`
  - Support for custom install commands: `"custom_install": "curl -sSL https://install.sh | bash"`
  - Required vs recommended dependency semantics
  - Interactive confirmation for custom installs
  - Dry-run mode: `--check-only`
  - Commits: `7d62a66`, `2f3d6df`, `1b416af`

#### Database Automation
- **Database Deployment Workflow** (integrated with `nself deploy`)
  - Automatic migrations application on deployment
  - Environment-aware seeding (dev: all seeds, staging: 000-004, prod: 000-001 only)
  - Hasura metadata application with fallback chain
  - Step 6 in deployment: Database → Migrations → Seeds → Metadata
  - Commits: `341debf`, `424e8ed`, `1d3872b`

- **Database Commands** (`nself db`)
  - `nself db shell` - Interactive psql session
  - `nself db backup [file]` - Create database backup
  - `nself db restore <file>` - Restore from backup
  - `nself db reset` - Drop all tables (dev only, requires confirmation)
  - Commits: `424e8ed`, `e23ac35`

- **Hasura Commands** (`nself hasura`)
  - `nself hasura metadata apply` - Apply Hasura metadata
  - `nself hasura metadata export` - Export current metadata
  - `nself hasura metadata reload` - Reload metadata cache
  - `nself hasura console` - Open Hasura console
  - Commit: `424e8ed`

#### Deployment Enhancements
- **Environment-Aware Configuration Rebuild** (Step 1.5)
  - Automatically rebuilds nginx configs with correct `BASE_DOMAIN` on remote
  - Runs `nself build --force` on target server after .env sync
  - Fixes issue where staging got localhost configs instead of staging domain
  - Commit: `1d3872b`

- **Hasura Metadata Fallback Chain**
  - Try 1: `nself hasura metadata apply` (preferred if available)
  - Try 2: `hasura metadata apply` (hasura CLI directly)
  - Try 3: Direct HTTP API call with curl
  - Handles minimal CLI installations gracefully
  - Commit: `97bf107`

### 🔒 Security

#### Secure-by-Default Implementation (Phase 1.2)
- **Removed Weak Default Credentials**
  - Removed `minioadmin` from MinIO defaults
  - Removed `hasura-admin-secret-dev` from Hasura defaults
  - Removed `development-secret-key-minimum-32-characters-long` from JWT defaults
  - All secrets now empty in `.env.example` with clear security warnings
  - Commit: `fc40ae2`

- **Enhanced Security Validation**
  - Added `GRAFANA_ADMIN_PASSWORD` validation (when monitoring enabled)
  - Added `MEILISEARCH_MASTER_KEY` to insecure values check
  - Auto-generates secure passwords in dev environments
  - Blocks production builds if secrets missing or weak
  - Environment-aware password strength (dev: 32 chars, staging: 40, prod: 48+)
  - Commit: `fc40ae2`

- **Security Validation Improvements**
  - Shows password strength assessment (weak/moderate/strong/secure)
  - Provides clear fix instructions: `nself config secrets generate`
  - Checks all sensitive environment variables
  - Validates port bindings (ensures 127.0.0.1 for sensitive ports)
  - Commit: `fc40ae2`

### 🐛 Bug Fixes

#### CLI Output & Display
- **Standardized Color Variables** (`COLOR_* → CLI_*`)
  - Fixed inconsistent color variable naming across codebase
  - All files now use `CLI_GREEN`, `CLI_RED`, `CLI_BLUE`, etc.
  - Ensures colors work consistently in all commands
  - Commit: `d73e49b`

- **Added Missing Display Functions**
  - Added `cli_subheader()` function to `cli-output.sh`
  - Fixed missing imports in `db.sh` and `hasura.sh`
  - Resolved "command not found" errors in database commands
  - Commits: `5e25485`, `e23ac35`

#### Environment & Configuration
- **Fixed Environment Variable Export**
  - Ensured all vars exported before `compose-generate.sh` runs
  - Fixed services not getting correct environment values
  - Commit: `7b8d345`

- **Better Error Handling**
  - Deployment now provides clear error messages with troubleshooting hints
  - Failed steps show actual error output (not just "FAILED")
  - Timeout indicators for network operations
  - Commits: `97bf107`, `0050dca`

### 📚 Documentation

#### User Feedback Integration
- Comprehensive feedback from production deployments (nself-web team)
- Detailed testing reports identifying real-world issues
- 9/9 services healthy validation
- Commits: `23664ff`, `1a1ef26`, `7d5d958`, `3aad1c8`, `d293a80`

### 🔧 Internal Improvements

#### Code Quality
- Cross-platform compatibility maintained (Bash 3.2+)
- Proper error handling in all network operations
- Timeouts for blocking operations
- Better separation of concerns (display vs logic)

#### Testing
- Validated on fresh VPS installations
- Production deployment testing
- Multi-environment testing (dev, staging, production)
- Real-world dogfooding with actual projects

---

## [0.9.8] - 2026-02-11

### Release Highlights
- Comprehensive wiki documentation
- Package build automation (DEB, RPM)
- AUR package support
- Docker build improvements

### Added
- GitHub Actions workflow for package builds
- Debian package configuration
- RPM spec file
- AUR PKGBUILD
- Comprehensive wiki update

### Fixed
- Docker build script paths
- RPM spec formatting
- Debian control file structure

---

## Migration Guide: v0.9.8 → v0.9.9 (Unreleased)

### Breaking Changes
**None** - All changes are backward compatible

### New Requirements
- **For Deployment**: SSH access must be configured (non-interactive)
- **For Security**: Empty secrets in .env.example must be generated during init

### Recommended Actions

#### 1. Update Installation Method
```bash
# New installations - get latest fixes:
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | INSTALL_LATEST=true bash

# Or wait for v0.9.9 release tag
```

#### 2. Security Audit
```bash
# Check for weak secrets:
nself config secrets generate --check-only

# Regenerate weak secrets:
nself config secrets generate
```

#### 3. Test Deployment
```bash
# Ensure deployment completes without hanging:
nself deploy sync full staging

# Should complete within 5 minutes
```

#### 4. Database Automation
```bash
# New database commands available:
nself db shell              # Interactive psql
nself db backup            # Create backup
nself hasura metadata apply # Apply Hasura metadata
```

#### 5. Plugin Dependencies
```bash
# Check plugin system dependencies:
nself plugin check-deps <plugin-name>

# Install missing dependencies:
nself plugin install-deps <plugin-name>
```

---

## Release Timeline

- **v0.9.8**: Released 2026-02-11
- **v0.9.9**: Planned after nself-web team QA approval
  - Waiting for fresh VPS testing with `INSTALL_LATEST=true`
  - All critical bugs fixed, awaiting production validation
  - Expected: Late February 2026

---

## Credits

### Testing & Feedback
- **nself-web team**: Comprehensive production testing, detailed bug reports
- **nself-tv team**: Plugin dependency management feedback
- **nself-chat team**: Monorepo migration insights

### Contributors
- **CLI Team**: Core development, bug fixes, security hardening
- **QA Team**: Real-world deployment testing, edge case discovery

---

## Links

- [GitHub Repository](https://github.com/acamarata/nself)
- [Issue Tracker](https://github.com/acamarata/nself/issues)
- [Documentation](https://github.com/acamarata/nself/wiki)
- [Homebrew Formula](https://github.com/acamarata/homebrew-nself)

---

*This changelog tracks all changes since v0.9.8 for the upcoming v0.9.9 release.*
*Last updated: 2026-02-12*
