# nself Releases & Changelog

Complete release history and roadmap for nself - Self-Hosted Infrastructure Manager.

**Current Stable Version:** v0.3.9

---

## 📦 Latest Release

### [v0.3.9](RELEASE-v0.3.9.md) - Current Stable

**Released:** September 2024

**Status:** Production Ready

**Highlights:**
- Complete admin UI overhaul with comprehensive management features
- Enhanced cross-platform compatibility (Bash 3.2+, POSIX compliant)
- Improved start command with smart health checking and configurable behavior
- SSL certificate management improvements
- 25 services in demo configuration

**[View Release Notes →](RELEASE-v0.3.9.md)**

---

## 🗺️ Roadmap

See our [Roadmap](ROADMAP.md) for planned features and improvements.

### Upcoming Releases

#### v0.4.1 (Q1 2025)
- **Service Management Commands**: `email`, `search`, `functions`, `mlflow`
- **Enhanced Service Configuration**: Simplified provider switching
- **Template Improvements**: More service templates

#### v0.4.2 (Q2 2025)
- **Monitoring Commands**: `metrics`, `monitor`
- **Integrated Dashboards**: Terminal-based monitoring
- **Performance Tracking**: Built-in metrics collection

#### v0.4.3 (Q2 2025)
- **Database Management**: `db` command with migrations, console, optimization
- **Schema Management**: dbdiagram.io integration
- **Backup Integration**: Database-specific backups

#### v0.4.4 (Q3 2025)
- **Backup System**: Comprehensive local and cloud backups
- **Rollback Support**: Point-in-time recovery
- **Automated Scheduling**: Cron-based backup schedules

#### v0.4.5 (Q3 2025)
- **Production Tools**: `prod` command for production configuration
- **SSH Deployment**: Zero-downtime deployments
- **Let's Encrypt**: Automatic SSL certificate provisioning

#### v0.4.6 (Q4 2025)
- **Scaling**: Service scaling and resource management
- **Auto-scaling**: Automatic resource adjustment
- **Load Balancing**: Multi-replica support

---

## 📋 Complete Changelog

See [CHANGELOG.md](CHANGELOG.md) for complete version history with all changes.

### Recent Releases

- **[v0.3.9](RELEASE-v0.3.9.md)** - September 2024 - Current stable
- **[v0.3.9-patch3](RELEASE-v0.3.9-patch3.md)** - Bug fixes and improvements
- **[v0.3.9-patch2](RELEASE-v0.3.9-patch2.md)** - Hotfixes
- **[v0.3.9-final](RELEASE-v0.3.9-final.md)** - Final release candidate

---

## 🔍 Version Guidelines

### Semantic Versioning

nself follows [Semantic Versioning](https://semver.org/):

**Format:** `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes, major feature additions
- **MINOR**: New features, backwards-compatible
- **PATCH**: Bug fixes, minor improvements

### Release Channels

- **Stable** (recommended): Production-ready releases (e.g., v0.3.9)
- **Beta**: Feature-complete pre-releases (e.g., v0.4.0-beta.1)
- **RC**: Release candidates (e.g., v0.4.0-rc.1)

### Updating nself

```bash
# Check current version
nself version

# Check for updates
nself update --check

# Update to latest stable
nself update

# Update to specific version
nself update --version 0.4.0

# Update to beta
nself update --beta
```

---

## 📊 Release Statistics

### v0.3.x Series

- **Total Releases**: 8
- **Release Cadence**: ~2 weeks
- **Bug Fixes**: 47
- **New Features**: 12
- **Documentation Updates**: 34

### Development Activity

- **Contributors**: Growing community
- **Commits**: 500+
- **Test Coverage**: 12 automated test jobs
- **Platform Support**: macOS, Linux (all distros), WSL

---

## 🎯 Feature Status

### ✅ Implemented (v0.3.9)

**Core Commands:**
- `init` - Project initialization
- `build` - Infrastructure generation
- `start` - Service orchestration
- `stop` - Graceful shutdown
- `restart` - Service restart
- `status` - Health monitoring
- `logs` - Log viewing
- `exec` - Container execution
- `urls` - Service endpoints
- `reset` - Project reset
- `clean` - Resource cleanup
- `restore` - Configuration restore

**Management Commands:**
- `ssl` - SSL certificate management
- `trust` - Certificate trust installation
- `admin` - Admin UI management
- `doctor` - System diagnostics
- `version` - Version information
- `update` - CLI updates
- `help` - Help system

### 🚧 In Development

- Enhanced admin UI features
- Additional service templates
- Improved error handling
- Performance optimizations

### 🔮 Planned (v0.4.0+)

See [Roadmap](ROADMAP.md) for complete list of planned features.

---

## 📝 Release Process

### For Contributors

See [Release Checklist](RELEASE-CHECKLIST.md) for the complete release process.

### For Users

1. **Stable Releases**: Automatically available via `nself update`
2. **Beta Releases**: Opt-in with `nself update --beta`
3. **Security Updates**: Announced in release notes and discussions

---

## 🔔 Stay Updated

### Release Notifications

- **Watch Repository**: Get notified of new releases on GitHub
- **Release Notes**: Subscribe to [Discussions](https://github.com/acamarata/nself/discussions)
- **Changelog**: Check [CHANGELOG.md](CHANGELOG.md) for detailed changes

### Breaking Changes

Major version changes (e.g., 0.x to 1.x) may include breaking changes. We will:
- Document all breaking changes in release notes
- Provide migration guides
- Maintain backwards compatibility when possible

---

## 📚 Documentation

- **[Home](../Home.md)** - Documentation homepage
- **[Commands Reference](../commands/COMMANDS.md)** - Complete CLI reference
- **[Contributing](../contributing/)** - Contribution guidelines
- **[Architecture](../architecture/)** - System design

---

## 🤝 Support

- **Issues**: [Report bugs](https://github.com/acamarata/nself/issues)
- **Discussions**: [Ask questions](https://github.com/acamarata/nself/discussions)
- **Support Development**: [Patreon](https://patreon.com/acamarata)

---

**Last Updated:** October 2024 | **Current Version:** v0.3.9
