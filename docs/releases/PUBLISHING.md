# nself Release Publishing

This document outlines where nself is published for each release.

---

## Distribution Channels

nself is published to the following platforms with each release:

### 1. GitHub Releases
- **URL**: https://github.com/acamarata/nself/releases
- **Includes**: Source code, release notes, changelog

### 2. Homebrew (macOS/Linux)
- **Tap**: `acamarata/nself`
- **Install**: `brew tap acamarata/nself && brew install nself`

### 3. npm (Node.js)
- **Package**: `@acamarata/nself`
- **Install**: `npm install -g @acamarata/nself`

### 4. Docker Hub
- **Image**: `acamarata/nself`
- **Pull**: `docker pull acamarata/nself:latest`

### 5. AUR (Arch Linux)
- **Package**: `nself`
- **Install**: `yay -S nself`

### 6. DEB Package (Debian/Ubuntu)
- **Download**: Available from GitHub Releases
- **Install**: `sudo dpkg -i nself_X.Y.Z_all.deb`

### 7. RPM Package (RHEL/CentOS/Fedora)
- **Download**: Available from GitHub Releases
- **Install**: `sudo rpm -i nself-X.Y.Z-1.noarch.rpm`

---

## Release Order

1. **GitHub Release** - Tag and publish release
2. **Homebrew** - Update formula with new SHA256
3. **npm** - Publish package
4. **Docker Hub** - Build and push images
5. **AUR** - Update PKGBUILD
6. **DEB/RPM** - Build and attach to GitHub Release

---

## Verification

After release, verify installation works:

```bash
# Homebrew
brew update && brew upgrade nself
nself version

# npm
npm update -g @acamarata/nself
nself version

# Docker
docker pull acamarata/nself:latest
docker run acamarata/nself:latest version
```

---

## Support

- **Issues**: https://github.com/acamarata/nself/issues
- **Discussions**: https://github.com/acamarata/nself/discussions
