# nself v0.3.9 Release Notes

**Version**: 0.3.9  
**Release Date**: September 2, 2025  
**Type**: Major Feature Release

## 🎯 Release Highlights

### 1. Admin UI Integration
- **New web-based admin interface** at `localhost:3100`
- Real-time service monitoring and management
- Database query interface
- Log viewer and health checks
- Backup management system
- Enable with: `nself admin enable`

### 2. Simplified Project Initialization
- **Minimal default setup**: `nself init` now creates only essential files
- **Full setup option**: `nself init --full` for complete environment
- **Interactive wizard**: `nself init --wizard` for guided setup
- **Smart defaults**: Works out-of-the-box without configuration

### 3. Enhanced Reset Command
- **Organized backups**: Creates timestamped folders in `_backup/`
- **Complete cleanup**: Removes all files except backup folder
- **Version history**: Maintains multiple backup snapshots
- Example: `_backup/20250902_175548/`

### 4. Environment System Overhaul
- **Fixed loading priority** (dev → staging/prod → local → .env → secrets)
- **Minimal templates** with smart defaults
- **Clean headers** (60 chars wide) for readability
- **Comprehensive .env.example** with all options documented

### 5. Config-Server Removal
- **Removed obsolete Nhost dashboard integration**
- Replaced entirely with nself-admin
- Cleaner codebase with no legacy dependencies

## 📦 What's New

### Commands (35+ Total)
- `nself admin` - Complete admin UI management
- `nself db` - Interactive database operations menu
- `nself init --full` - Complete project setup
- `nself init --wizard` - Interactive configuration
- Enhanced `nself reset` with better backup strategy
- Improved `nself doctor` diagnostics

### Templates & Examples
- **New database schema**: `schema.dbml` with example tables
- **25+ service templates**: Express, FastAPI, Flask, Gin, Rust, Ruby, and more
- **Nginx configuration** templates
- **Updated SSL certificates** for local development

### Bug Fixes
- ✅ Fixed hanging status command
- ✅ Fixed stop command compose wrapper
- ✅ Fixed exec command container detection
- ✅ Fixed build command timeout issues
- ✅ Fixed doctor command function references
- ✅ Fixed SSL certificate generation for nginx
- ✅ Fixed environment loading priority order

## 💔 Breaking Changes

### 1. Init Command Behavior
- **Old**: Created all environment files by default
- **New**: Creates minimal setup (use `--full` for old behavior)
- **Migration**: Run `nself init --full` for complete setup

### 2. Environment Loading Order
- Priority order has been corrected
- May affect deployments relying on incorrect precedence
- Review your environment files if upgrading

### 3. Config-Server Removed
- Any projects using config-server must migrate to nself-admin
- No automatic migration available

## 📚 Documentation Updates

### New Documentation
- Comprehensive command reference (`docs/COMMANDS.md`)
- Release checklist (`docs/RELEASE-CHECKLIST.md`)
- Command tree visualization (`docs/COMMAND-TREE-FINAL.md`)
- Updated changelog and roadmap

### Updated Files
- `README.md` - Simplified and clarified
- `CHANGELOG.md` - Complete version history
- `ROADMAP.md` - Future development plans

## 🔧 Installation & Upgrade

### New Installation
```bash
curl -sSL https://install.nself.org | bash
```

### Upgrade from Previous Version
```bash
nself update
```

### Manual Upgrade
```bash
cd /path/to/nself
git pull origin main
./install.sh
```

## 🚀 Quick Start

### New Project
```bash
# Minimal setup (recommended)
mkdir myproject && cd myproject
nself init
nself build
nself start

# Full setup with all features
nself init --full
nself admin enable
nself build
nself start
```

### Migrating from v0.3.8
```bash
# Backup current project
cp -r myproject myproject.backup

# Reset and reinitialize
cd myproject
nself reset
nself init --full
# Copy your custom settings from backup
cp ../myproject.backup/.env.local .env.local
nself build
nself start
```

## 📋 Complete Change Log

### Added
- Admin UI integration via nself-admin Docker image
- Database management interactive menu
- Schema.dbml template with example tables
- 25+ service template stubs for various languages
- Gitignore management in init command
- Backup versioning with timestamps
- --full, --wizard, --admin flags for init

### Changed
- Init command defaults to minimal setup
- Reset command uses timestamp-based backups
- Environment loading priority corrected
- All environment templates updated with clean headers
- Build command improved error handling
- SSL certificates regenerated

### Removed
- Config-server and all related code
- Obsolete Nhost dashboard integration
- Old JavaScript service templates

### Fixed
- Status command hanging issues
- Stop command compose wrapper calls
- Exec command container detection
- Build command validation timeout
- Doctor command function references
- Display.sh unbound variables
- Environment loading priority bugs

## 🧪 Testing Coverage

### Tested Commands
✅ Core: init, build, start, stop, restart, status, logs  
✅ Management: doctor, db, admin, email, validate  
✅ Development: reset, diff, exec  
✅ Production: prod, ssl, trust  
✅ Utilities: version, help, update

### Test Environments
- macOS (Apple Silicon & Intel)
- Ubuntu 22.04, 24.04
- Docker Desktop 4.30+
- Docker Engine 24.0+

## 🐛 Known Issues

### Minor Issues
- Auth service health check may report unhealthy (service works correctly on port 4001)
- Some commands may require PROJECT_NAME environment variable to be set
- Build validation may occasionally timeout (workaround: increase timeout)

### Workarounds
- For auth health check: Ignore the warning, service is functional
- For PROJECT_NAME: Add to .env.local: `PROJECT_NAME=myproject`
- For build timeout: Use `timeout 10 nself build` if needed

## 🙏 Acknowledgments

### Contributors
- Lead Developer: @acamarata
- Community testers and bug reporters
- Documentation contributors

### Special Thanks
- Hasura team for GraphQL engine
- Nhost team for auth/storage services
- Docker team for container platform

## 📞 Support

### Getting Help
- GitHub Issues: https://github.com/acamarata/nself/issues
- Documentation: https://github.com/acamarata/nself/docs/
- Discord: Coming soon

### Reporting Issues
When reporting issues, please include:
- nself version (`nself version`)
- Operating system and Docker version
- Steps to reproduce
- Error messages or logs

## 🔮 What's Next (v0.4.0)

### Planned Features
- Kubernetes deployment options
- Multi-node clustering support
- Advanced monitoring with Prometheus/Grafana
- Automated backup verification
- Cloud provider integrations

### In Development
- Plugin system for custom services
- CI/CD pipeline templates
- Performance optimizations
- Enhanced security features

---

**Thank you for using nself!** 🚀

*For the complete changelog, see [CHANGELOG.md](./CHANGELOG.md)*  
*For future plans, see [ROADMAP.md](../ROADMAP.md)*