# nself Releases & Changelog

Complete release history and roadmap for nself - Self-Hosted Infrastructure Manager.

**Current Stable Version:** v0.4.2

---

## Latest Release

### [v0.4.2](v0.4.2.md) - Current Stable

**Released:** January 2026

**Status:** Production Ready

**Highlights:**
- 6 new service management commands (email, search, functions, mlflow, metrics, monitor)
- 16+ email provider support with SMTP pre-flight checks
- 6 search engines supported (PostgreSQL, MeiliSearch, Typesense, Elasticsearch, OpenSearch, Sonic)
- Serverless functions with TypeScript support
- Monitoring profiles (minimal, standard, full, auto)
- 92 unit tests, complete documentation

**[View Release Notes →](v0.4.2.md)**

---

## Recent Releases

### [v0.4.1](v0.4.1.md) - January 2026

**Status:** Stable

**Highlights:**
- Fixed Bash 3.2 compatibility for macOS
- Fixed cross-platform sed, stat, and timeout commands
- Fixed portable output formatting (POSIX compliance)
- All commands working on macOS, Linux, and WSL

**[View Release Notes →](v0.4.1.md)**

### [v0.4.0](v0.4.0.md) - October 2025

**Status:** Stable

**Highlights:**
- Production-ready release
- All core features complete and tested
- Enhanced cross-platform compatibility (Bash 3.2+)
- CI/CD pipeline passing all tests (12/12)

**[View Release Notes →](v0.4.0.md)**

### [v0.3.9](v0.3.9.md) - September 2025

**Status:** Stable

**Highlights:**
- Admin UI with comprehensive management features
- 25 services in demo configuration
- All 36 commands fully functional

**[View Release Notes →](v0.3.9.md)**

---

## Roadmap

See our [Roadmap](ROADMAP.md) for planned features and improvements.

### Upcoming Releases

| Version | Target | Focus |
|---------|--------|-------|
| **v0.4.3** | Q1 2026 | Deployment Pipeline (local → staging → prod) |
| v0.4.4 | Q1-Q2 2026 | Database, Backup & Restore |
| v0.4.5 | Q2 2026 | Mock Data & Seeding System |
| v0.4.6 | Q2-Q3 2026 | Scaling & Performance |
| v0.4.7 | Q3 2026 | Multi-Cloud Providers (AWS, GCP, Azure, DO, Linode) |
| v0.4.8 | Q3-Q4 2026 | Kubernetes Support |
| v0.4.9 | Q4 2026 | Polish & nself-admin Integration |
| **v0.5.0** | Q4 2026 / Q1 2027 | **Production Release + nself-admin v0.1** |

---

## Complete Changelog

See [CHANGELOG.md](../CHANGELOG.md) for complete version history with all changes.

### All Releases

| Version | Date | Status | Highlights |
|---------|------|--------|------------|
| [v0.4.2](v0.4.2.md) | Jan 2026 | **Current** | Service & monitoring commands |
| [v0.4.1](v0.4.1.md) | Jan 2026 | Stable | Platform compatibility fixes |
| [v0.4.0](v0.4.0.md) | Oct 2025 | Stable | Production-ready release |
| [v0.3.9](v0.3.9.md) | Sep 2025 | Stable | Admin UI, 36 commands |
| [v0.3.8](v0.3.8.md) | Aug 2024 | Stable | Backup system, SSL management |

---

## Version Guidelines

### Semantic Versioning

nself follows [Semantic Versioning](https://semver.org/):

**Format:** `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes, major feature additions
- **MINOR**: New features, backwards-compatible
- **PATCH**: Bug fixes, minor improvements

### Release Channels

- **Stable** (recommended): Production-ready releases (e.g., v0.4.1)
- **Beta**: Feature-complete pre-releases (e.g., v0.5.0-beta.1)
- **RC**: Release candidates (e.g., v0.5.0-rc.1)

### Updating nself

```bash
# Check current version
nself version

# Update to latest stable
nself update

# Update to specific version
NSELF_VERSION=v0.4.1 bash <(curl -sSL https://install.nself.org)
```

---

## Installation Methods

| Method | Command | Platforms |
|--------|---------|-----------|
| **curl (Primary)** | `curl -sSL https://install.nself.org \| bash` | macOS, Linux, WSL |
| **Homebrew** | `brew install acamarata/nself/nself` | macOS, Linux |
| **npm** | `npm install -g nself-cli` | All |
| **apt-get** | See Debian package | Ubuntu, Debian |
| **dnf/yum** | See RPM package | Fedora, RHEL |
| **AUR** | `yay -S nself` | Arch Linux |
| **Docker** | `docker pull acamarata/nself:latest` | All |

---

## Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| macOS (Bash 3.2) | ✅ Full | Default shell supported |
| Ubuntu/Debian | ✅ Full | All versions |
| Fedora/RHEL | ✅ Full | All versions |
| Arch Linux | ✅ Full | AUR package |
| Alpine Linux | ✅ Full | Docker-based |
| WSL/WSL2 | ✅ Full | Windows integration |

---

## Stay Updated

### Release Notifications

- **Watch Repository**: Get notified of new releases on GitHub
- **Release Notes**: Subscribe to [Discussions](https://github.com/acamarata/nself/discussions)
- **Changelog**: Check [CHANGELOG.md](../CHANGELOG.md) for detailed changes

---

## Documentation

- **[Home](../Home.md)** - Documentation homepage
- **[Commands Reference](../commands/COMMANDS.md)** - Complete CLI reference
- **[Contributing](../CONTRIBUTING.md)** - Contribution guidelines
- **[Architecture](../architecture/ARCHITECTURE.md)** - System design

---

## Support

- **Issues**: [Report bugs](https://github.com/acamarata/nself/issues)
- **Discussions**: [Ask questions](https://github.com/acamarata/nself/discussions)

---

**Last Updated:** January 2026 | **Current Version:** v0.4.2
