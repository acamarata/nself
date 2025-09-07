# nself v0.3.9 - Patch Update (September 7, 2024)

## Installation & Release Optimization Patch

This patch update to v0.3.9 optimizes the installation process and release distribution without changing any runtime functionality.

### Key Improvements

#### 1. Minimal Release Tarballs
- **87% size reduction** - from 3.4MB to 432KB
- Contains only runtime-essential files
- Excludes development, test, and documentation files
- New build script: `.releases/scripts/build-release-tarball.sh`

#### 2. Smart Installation
- `install.sh` now detects version types automatically
- Release versions download minimal tarballs (432KB)
- Development versions (`main` branch) download full source
- Automatic fallback for compatibility

#### 3. Release Structure Cleanup
- Consolidated all releases under `.releases/` directory
- Removed duplicate `releases/` directory
- Better organization of release tooling

### Installation Methods

```bash
# Standard installation (minimal tarball)
curl -sSL https://install.nself.org | bash

# Development installation (full source)
NSELF_VERSION=main curl -sSL https://install.nself.org | bash

# Specific version
NSELF_VERSION=v0.3.9 curl -sSL https://install.nself.org | bash
```

### What's Excluded from Release Tarballs
- Test suites (`src/tests/`)
- Development tools (`.releases/`)
- GitHub workflows (`.github/`)
- Example and demo files
- IDE configurations
- Full documentation (keeping only essentials)

### What's Included
- Core executable (`bin/nself`)
- CLI commands (`src/cli/`)
- Libraries (`src/lib/`)
- Docker generation (`src/services/docker/`)
- Essential templates
- License and README

### No Breaking Changes
- Fully backward compatible
- No runtime functionality changes
- Existing projects continue working

### For Developers
To get the full source with tests and development tools:
```bash
git clone https://github.com/acamarata/nself.git
```

---
*This is a patch update to v0.3.9. The version number remains 0.3.9.*