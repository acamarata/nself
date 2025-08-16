# nself Release Process Documentation

## Overview

This document describes the complete release process for nself, including all distribution channels, automation scripts, and procedures.

## Quick Release

For standard releases, use the automated script:

```bash
./scripts/release.sh v0.3.7
```

This handles GitHub tagging and release creation automatically.

## Documentation Structure

### Release Documentation Files

1. **`RELEASE_PROCESS.MD`** (this file) - Master release process documentation
2. **`RELEASE_GUIDE.MD`** - Comprehensive guide for all distribution channels
3. **`RELEASE_CHECKLIST.MD`** - Template checklist for releases
4. **`RELEASE_CHECKLIST_vX.X.X.MD`** - Version-specific checklists
5. **`CHANGELOG.MD`** - User-facing changelog
6. **`RELEASES.MD`** - Detailed release history

### Release Scripts

Located in `/scripts/`:
- `release.sh` - Main automated release script
- `update-homebrew.sh` - Update Homebrew formula
- `update-docker.sh` - Build and push Docker images
- `create-windows-installer.ps1` - Generate Windows installers

## Release Types

### 1. Patch Release (X.X.✓)
- Bug fixes only
- No breaking changes
- No new features
- Example: v0.3.7

### 2. Minor Release (X.✓.0)
- New features
- Backward compatible
- May include bug fixes
- Example: v0.4.0

### 3. Major Release (✓.0.0)
- Breaking changes
- Major refactoring
- API changes
- Example: v1.0.0

## Pre-Release Checklist

### 1. Code Preparation
- [ ] All tests passing
- [ ] No uncommitted changes
- [ ] On main branch
- [ ] Branch up to date with origin

### 2. Version Updates
- [ ] Update `src/VERSION`
- [ ] Update `CHANGELOG.MD`
- [ ] Update version in README if displayed
- [ ] Create version-specific release checklist

### 3. Testing
- [ ] Run full test suite: `cd src/tests && ./run_tests.sh`
- [ ] Test on Linux
- [ ] Test on macOS
- [ ] Test on WSL/Windows
- [ ] Test upgrade path from previous version

## Release Process

### Step 1: Prepare Release

```bash
# Ensure clean state
git status
git checkout main
git pull origin main

# Update version
echo "v0.3.7" > src/VERSION

# Update changelog
$EDITOR docs/CHANGELOG.MD

# Commit changes
git add -A
git commit -m "Prepare release v0.3.7"
```

### Step 2: Run Release Script

```bash
./scripts/release.sh v0.3.7
```

This will:
1. Validate repository state
2. Run tests
3. Create git tag
4. Push to GitHub
5. Create GitHub release (if `gh` CLI available)

### Step 3: Distribution Channels

#### Primary Channel: GitHub
- Automatic via release script
- Manual: https://github.com/acamarata/nself/releases/new

#### Package Managers
See `RELEASE_GUIDE.MD` for detailed instructions for:
- Homebrew
- Docker Hub
- AUR (Arch Linux)
- Snap (Ubuntu)
- Scoop (Windows)
- Chocolatey (Windows)

### Step 4: Verification

```bash
# Test direct installation
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash

# Test version
nself version  # Should show new version

# Test in clean environment
cd /tmp && rm -rf test-nself
mkdir test-nself && cd test-nself
nself init
nself build
nself doctor
```

## Post-Release Tasks

### Immediate (Day 0)
- [ ] Verify GitHub release is published
- [ ] Test installation script
- [ ] Update website/documentation if applicable
- [ ] Monitor GitHub Issues

### Day 1-3
- [ ] Address any critical issues
- [ ] Respond to user feedback
- [ ] Update package managers

### Week 1
- [ ] Analyze adoption metrics
- [ ] Plan hotfixes if needed
- [ ] Update roadmap

## Hotfix Process

For critical issues after release:

```bash
# Create hotfix branch
git checkout -b hotfix/v0.3.7.1 v0.3.7

# Make fixes
# ...

# Update version
echo "v0.3.7.1" > src/VERSION

# Commit and tag
git commit -am "Hotfix: <description>"
git tag -a v0.3.7.1 -m "Hotfix release v0.3.7.1"

# Push
git push origin hotfix/v0.3.7.1
git push origin v0.3.7.1

# Merge back to main
git checkout main
git merge hotfix/v0.3.7.1
git push origin main
```

## Version Numbering

Following Semantic Versioning 2.0.0:

```
MAJOR.MINOR.PATCH

MAJOR: Incompatible API changes
MINOR: Backwards-compatible functionality additions
PATCH: Backwards-compatible bug fixes
```

Pre-release versions:
- Alpha: v0.3.7-alpha.1
- Beta: v0.3.7-beta.1
- RC: v0.3.7-rc.1

## Announcement Templates

### GitHub Release
```markdown
## 🚀 nself vX.X.X

### ✨ Highlights
- Feature 1
- Feature 2

### 🐛 Bug Fixes
- Fix 1
- Fix 2

### 📦 Installation
\```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
\```

### 📝 Full Changelog
See [CHANGELOG.md](https://github.com/acamarata/nself/blob/main/docs/CHANGELOG.MD)
```

### Social Media
```
🚀 nself vX.X.X released!

✨ What's new:
• Feature 1
• Feature 2
• Bug fixes

📦 Install/Update:
curl -fsSL https://nself.org/install | bash

📖 Docs: https://github.com/acamarata/nself
```

## Troubleshooting

### Common Issues

1. **Tests failing**: Fix before release, no exceptions
2. **Tag already exists**: Delete with `git tag -d vX.X.X` and `git push origin :vX.X.X`
3. **Release script fails**: Check `gh` CLI authentication
4. **Docker build fails**: Ensure Docker daemon running and logged in

### Emergency Rollback

If critical issue found immediately after release:

```bash
# Delete release and tag
gh release delete vX.X.X --yes
git tag -d vX.X.X
git push origin :vX.X.X

# Fix issue and re-release with same version
# or increment patch version for transparency
```

## Distribution Channel Status

| Channel | Status | Priority | Maintainer |
|---------|--------|----------|------------|
| GitHub Release | ✅ Active | Essential | Core |
| Direct Install | ✅ Active | Essential | Core |
| Homebrew | 🔧 Ready | High | Community |
| Docker Hub | 🔧 Ready | High | Core |
| AUR | ❌ Planned | Medium | Community |
| Snap | ❌ Planned | Medium | Community |
| Scoop | 📝 Manifest | Medium | Community |
| Chocolatey | 📝 Package | Low | Community |

## References

- [Semantic Versioning](https://semver.org/)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Keep a Changelog](https://keepachangelog.com/)

## Contact

For release-related questions:
- GitHub Issues: https://github.com/acamarata/nself/issues
- Maintainer: @acamarata