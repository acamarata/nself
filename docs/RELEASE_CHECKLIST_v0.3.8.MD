# nself v0.3.8 Release Checklist

## Pre-Release Status
- [x] Update `src/VERSION` to `v0.3.8`
- [x] Calculate SHA256 checksum: `bf6ee149699bddda77f72ceed9c0491eef8ba04431239eadb8b6b77616d260de`
- [x] Create package manager configurations
- [x] Create release automation scripts

## Package Manager Release Status

### 1. Homebrew (macOS/Linux)
**Status:** ⏳ Ready to publish  
**Files:** `/Users/admin/Sites/nself/packaging/homebrew/nself.rb`

**Steps to publish:**
```bash
# Option A: Personal Tap (Recommended)
1. Create/update homebrew-nself repository:
   git clone https://github.com/acamarata/homebrew-nself || git init homebrew-nself

2. Copy formula:
   mkdir -p homebrew-nself/Formula
   cp packaging/homebrew/nself.rb homebrew-nself/Formula/nself.rb

3. Commit and push:
   cd homebrew-nself
   git add Formula/nself.rb
   git commit -m "Update nself to v0.3.8"
   git push origin main

4. Users install with:
   brew tap acamarata/nself
   brew install nself
```

### 2. Docker Hub
**Status:** ⏳ Ready to publish  
**Files:** `/Users/admin/Sites/nself/packaging/docker/Dockerfile`, `/Users/admin/Sites/nself/packaging/docker/build.sh`

**Steps to publish:**
```bash
# Login to Docker Hub
docker login

# Build and push multi-arch images
./packaging/docker/build.sh

# Verify
docker run --rm acamarata/nself:0.3.8 version
```

### 3. AUR (Arch Linux)
**Status:** ⏳ Ready to publish  
**Files:** `/Users/admin/Sites/nself/packaging/aur/PKGBUILD`, `/Users/admin/Sites/nself/packaging/aur/.SRCINFO`

**Steps to publish:**
```bash
# Clone AUR repository
git clone ssh://aur@aur.archlinux.org/nself.git

# Copy files
cd nself
cp ../packaging/aur/PKGBUILD .
cp ../packaging/aur/.SRCINFO .

# Test build locally
makepkg -si

# Commit and push
git add PKGBUILD .SRCINFO
git commit -m "Update to v0.3.8"
git push
```

### 4. Ubuntu/Debian PPA
**Status:** ⏳ Ready to publish  
**Files:** `/Users/admin/Sites/nself/packaging/deb/DEBIAN/control`, `/Users/admin/Sites/nself/packaging/deb/build-deb.sh`

**Steps to publish:**
```bash
# Build .deb package
./packaging/deb/build-deb.sh

# Upload to Launchpad PPA (requires setup)
# 1. Create Launchpad account and PPA
# 2. Upload GPG key to Launchpad
# 3. Create source package with debuild
# 4. Upload with dput

# Alternative: Manual distribution
# The .deb file can be distributed manually or via GitHub releases
```

### 5. Fedora/RHEL (RPM)
**Status:** ⏳ Ready to publish  
**Files:** `/Users/admin/Sites/nself/packaging/rpm/nself.spec`, `/Users/admin/Sites/nself/packaging/rpm/build-rpm.sh`

**Steps to publish:**
```bash
# Build RPM package
./packaging/rpm/build-rpm.sh

# Submit to Fedora (requires contributor access)
# 1. Create Fedora account
# 2. Join packager group
# 3. Submit package review request

# Alternative: Manual distribution
# The RPM file can be distributed manually or via GitHub releases
```

### 6. NPM (Node.js)
**Status:** ⏳ Ready to publish  
**Files:** `/Users/admin/Sites/nself/packaging/npm/package.json`

**Steps to publish:**
```bash
# Login to NPM
npm login

# Publish package
./scripts/publish-npm.sh

# Users install with:
npm install -g nself-cli
npx nself-cli help
```

## Quick Start - Automated Release

Run the master release script:
```bash
./scripts/release-all.sh
```

This will:
1. Verify version consistency
2. Build Docker images (if Docker available)
3. Show status of all package files
4. Provide next steps for each platform

## Manual Publication Commands

### Homebrew
```bash
./scripts/update-homebrew.sh
```

### Docker
```bash
./packaging/docker/build.sh
```

### NPM
```bash
./scripts/publish-npm.sh
```

### Debian
```bash
./packaging/deb/build-deb.sh
```

### RPM
```bash
./packaging/rpm/build-rpm.sh
```

## Verification Steps

After publishing to each platform, verify installation:

```bash
# Homebrew
brew install acamarata/nself/nself
nself version

# Docker
docker run --rm acamarata/nself:0.3.8 version

# AUR (on Arch Linux)
yay -S nself
nself version

# NPM
npm install -g nself-cli
nself version

# Debian
dpkg -i nself_0.3.8.deb
nself version

# RPM
rpm -i nself-0.3.8-1.noarch.rpm
nself version
```

## Post-Release Tasks

- [ ] Update GitHub release with links to all package managers
- [ ] Update README.md with installation instructions for all platforms
- [ ] Announce release on social media/forums
- [ ] Monitor GitHub issues for installation problems
- [ ] Update documentation site if applicable

## Release Artifacts

All package files are located in `/Users/admin/Sites/nself/packaging/`:
- `homebrew/nself.rb` - Homebrew formula
- `docker/Dockerfile` - Docker image configuration
- `aur/PKGBUILD` - Arch Linux AUR package
- `deb/DEBIAN/control` - Debian package control
- `rpm/nself.spec` - RPM specification
- `npm/package.json` - NPM package configuration

## Checksums

**v0.3.8 Release Tarball:**
- URL: `https://github.com/acamarata/nself/archive/refs/tags/v0.3.8.tar.gz`
- SHA256: `bf6ee149699bddda77f72ceed9c0491eef8ba04431239eadb8b6b77616d260de`

## Troubleshooting

### Docker Build Issues
- Ensure Docker daemon is running
- Login with `docker login`
- Check Docker Hub permissions

### NPM Publish Issues
- Verify npm login: `npm whoami`
- Check package name availability
- Ensure version hasn't been published

### Package Manager Delays
- AUR: Updates appear immediately
- Homebrew: May take time for formula approval
- Debian/Ubuntu PPA: Requires build queue processing
- Fedora: Requires package review process

---

**Release prepared by:** Automated package manager setup  
**Date:** 2024-08-17  
**Version:** v0.3.8  
**Commit:** Based on GitHub release v0.3.8