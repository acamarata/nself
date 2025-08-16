# nself v0.3.7 Complete Release Guide

## Pre-Release Checklist

### 1. Version Updates
- [ ] Update `src/VERSION` to `v0.3.7`
- [ ] Update `docs/CHANGELOG.md` with release notes
- [ ] Update `README.md` version badge (if applicable)
- [ ] Update `install.sh` default version (if hardcoded)

### 2. Testing
- [ ] Run full test suite: `cd src/tests && ./run_tests.sh`
- [ ] Test installation on clean Linux VM
- [ ] Test installation on clean macOS
- [ ] Test installation on WSL (Windows)
- [ ] Run integration tests in CI/CD

### 3. Documentation
- [ ] Update docs/CHANGELOG.md
- [ ] Update Wiki on GitHub
- [ ] Update nself.org website (if applicable)

---

## Release Channels

### 1. GitHub Release (Primary)

```bash
# 1. Ensure you're on main branch with latest changes
git checkout main
git pull origin main

# 2. Create and push tag
git tag -a v0.3.7 -m "Release v0.3.7: Major refactoring and critical fixes"
git push origin v0.3.7

# 3. Create GitHub Release
# Go to: https://github.com/acamarata/nself/releases/new
# - Choose tag: v0.3.7
# - Release title: nself v0.3.7 - Major Refactoring & Critical Fixes
# - Description: (see template below)
# - Attach binaries if applicable
# - Set as latest release
# - Publish
```

**GitHub Release Description Template:**
```markdown
## 🚀 nself v0.3.7 Release

### ✨ Highlights
- Fixed critical nginx SSL configuration issues
- Resolved logs and db command execution errors
- Improved command header standardization
- Enhanced error handling across all commands

### 🐛 Bug Fixes
- Fixed nginx SSL certificate path configuration
- Fixed http2 deprecation warnings in nginx configs
- Fixed logs command missing header source
- Fixed db help display not showing without arguments
- Improved error handling with `set +e` in critical commands

### 📝 Full Changelog
See [CHANGELOG.md](https://github.com/acamarata/nself/blob/main/CHANGELOG.md)

### 📦 Installation

**New Installation:**
```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

**Upgrade Existing:**
```bash
nself update
```
```

---

### 2. Homebrew (macOS/Linux)

**Prerequisites:**
- Homebrew tap repository: `homebrew-nself` or contribute to `homebrew-core`

```bash
# Option A: Personal Tap (Easier, Immediate)
# 1. Create/update homebrew-nself repository
git clone https://github.com/acamarata/homebrew-nself
cd homebrew-nself

# 2. Create/update Formula/nself.rb
cat > Formula/nself.rb << 'EOF'
class Nself < Formula
  desc "Self-hosted infrastructure manager for developers"
  homepage "https://nself.org"
  url "https://github.com/acamarata/nself/archive/refs/tags/v0.3.7.tar.gz"
  sha256 "ACTUAL_SHA256_HERE"  # Generate with: shasum -a 256 nself-v0.3.7.tar.gz
  license "MIT"
  head "https://github.com/acamarata/nself.git", branch: "main"

  depends_on "bash" => :build
  depends_on "docker"
  depends_on "docker-compose"

  def install
    # Install all source files
    libexec.install Dir["*"]
    
    # Create wrapper script
    (bin/"nself").write <<~EOS
      #!/bin/bash
      export NSELF_HOME="#{libexec}"
      exec "#{libexec}/bin/nself" "$@"
    EOS
    
    # Make executable
    chmod 0755, bin/"nself"
  end

  test do
    assert_match "v0.3.7", shell_output("#{bin}/nself version")
  end
end
EOF

# 3. Calculate SHA256
wget https://github.com/acamarata/nself/archive/refs/tags/v0.3.7.tar.gz
shasum -a 256 v0.3.7.tar.gz
# Update the sha256 in the formula

# 4. Test locally
brew install --build-from-source Formula/nself.rb
brew test nself

# 5. Commit and push
git add Formula/nself.rb
git commit -m "Update nself to v0.3.7"
git push

# 6. Users can now install via:
# brew tap acamarata/nself
# brew install nself
```

**Option B: Homebrew Core (Official, Takes Time)**
```bash
# 1. Fork homebrew-core
# 2. Create branch
git checkout -b nself-0.3.7

# 3. Update formula
brew edit nself  # If exists, or create new

# 4. Test extensively
brew audit --strict nself
brew test nself

# 5. Submit PR to homebrew-core
```

---

### 3. AUR (Arch Linux)

**Create/Update PKGBUILD:**
```bash
# Clone AUR package (or create new)
git clone ssh://aur@aur.archlinux.org/nself.git
cd nself

# Update PKGBUILD
cat > PKGBUILD << 'EOF'
# Maintainer: Your Name <email@example.com>
pkgname=nself
pkgver=0.3.7
pkgrel=1
pkgdesc="Self-hosted infrastructure manager for developers"
arch=('any')
url="https://nself.org"
license=('MIT')
depends=('bash' 'docker' 'docker-compose')
source=("$pkgname-$pkgver.tar.gz::https://github.com/acamarata/nself/archive/v$pkgver.tar.gz")
sha256sums=('ACTUAL_SHA256_HERE')

package() {
  cd "$srcdir/$pkgname-$pkgver"
  
  # Install to /opt
  install -dm755 "$pkgdir/opt/$pkgname"
  cp -r * "$pkgdir/opt/$pkgname/"
  
  # Create symlink
  install -dm755 "$pkgdir/usr/bin"
  ln -s "/opt/$pkgname/bin/nself" "$pkgdir/usr/bin/nself"
  
  # Install license
  install -Dm644 LICENSE "$pkgdir/usr/share/licenses/$pkgname/LICENSE"
}
EOF

# Generate checksums
makepkg -g >> PKGBUILD

# Test build
makepkg -si

# Update .SRCINFO
makepkg --printsrcinfo > .SRCINFO

# Commit and push
git add PKGBUILD .SRCINFO
git commit -m "Update to v0.3.7"
git push
```

---

### 4. Snap Package (Ubuntu/Linux)

**Create snapcraft.yaml:**
```yaml
name: nself
version: '0.3.7'
summary: Self-hosted infrastructure manager
description: |
  nself is a comprehensive CLI tool for deploying and managing 
  self-hosted backend infrastructure based on Nhost.io

base: core22
confinement: classic
grade: stable

apps:
  nself:
    command: bin/nself
    environment:
      NSELF_HOME: $SNAP

parts:
  nself:
    source: https://github.com/acamarata/nself.git
    source-tag: v0.3.7
    plugin: dump
    stage-packages:
      - bash
      - curl
      - git
    override-build: |
      cp -r * $SNAPCRAFT_PART_INSTALL/
      chmod +x $SNAPCRAFT_PART_INSTALL/bin/nself
```

**Build and publish:**
```bash
# Build snap
snapcraft

# Test locally
sudo snap install nself_0.3.7_amd64.snap --dangerous --classic

# Login to snapcraft
snapcraft login

# Push to store
snapcraft upload nself_0.3.7_amd64.snap
snapcraft release nself 0.3.7 stable
```

---

### 5. Docker Image

**Create multi-arch Docker image:**
```dockerfile
# Dockerfile
FROM alpine:latest

RUN apk add --no-cache \
    bash \
    curl \
    git \
    docker-cli \
    docker-compose

COPY . /opt/nself
RUN chmod +x /opt/nself/bin/nself && \
    ln -s /opt/nself/bin/nself /usr/local/bin/nself

WORKDIR /workspace
ENTRYPOINT ["nself"]
```

**Build and push:**
```bash
# Login to Docker Hub
docker login

# Build multi-arch
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t acamarata/nself:0.3.7 \
  -t acamarata/nself:latest \
  --push .
```

---

### 6. Windows Installers

#### A. PowerShell Script (Recommended)
```powershell
# install.ps1
$version = "0.3.7"
$installDir = "$env:LOCALAPPDATA\nself"

# Download and extract
Invoke-WebRequest -Uri "https://github.com/acamarata/nself/archive/v$version.zip" -OutFile "$env:TEMP\nself.zip"
Expand-Archive -Path "$env:TEMP\nself.zip" -DestinationPath $installDir -Force

# Add to PATH
$path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($path -notlike "*$installDir\bin*") {
    [Environment]::SetEnvironmentVariable("Path", "$path;$installDir\bin", "User")
}

Write-Host "nself v$version installed successfully!"
Write-Host "Please restart your terminal for PATH changes to take effect."
```

**Distribution:**
```powershell
# Users run:
irm https://raw.githubusercontent.com/acamarata/nself/main/install.ps1 | iex
```

#### B. Scoop Package
```json
{
    "version": "0.3.7",
    "description": "Self-hosted infrastructure manager",
    "homepage": "https://nself.org",
    "license": "MIT",
    "url": "https://github.com/acamarata/nself/archive/v0.3.7.zip",
    "hash": "SHA256_HERE",
    "extract_dir": "nself-0.3.7",
    "bin": "bin\\nself.bat",
    "checkver": "github",
    "autoupdate": {
        "url": "https://github.com/acamarata/nself/archive/v$version.zip",
        "extract_dir": "nself-$version"
    }
}
```

#### C. Chocolatey Package
```powershell
# nself.nuspec
<?xml version="1.0"?>
<package>
  <metadata>
    <id>nself</id>
    <version>0.3.7</version>
    <title>nself</title>
    <authors>acamarata</authors>
    <projectUrl>https://nself.org</projectUrl>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <description>Self-hosted infrastructure manager</description>
    <tags>docker infrastructure devops</tags>
  </metadata>
  <files>
    <file src="tools\**" target="tools" />
  </files>
</package>
```

#### D. WSL Instructions (Fallback)
```markdown
## Windows Installation via WSL

1. Install WSL2:
```powershell
wsl --install
```

2. Install nself in WSL:
```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

3. Use from Windows Terminal or VS Code with WSL extension
```

---

### 7. Package Managers Summary

| Platform | Package Manager | Command | Status |
|----------|----------------|---------|--------|
| macOS | Homebrew | `brew install nself` | Need to create |
| Ubuntu/Debian | Snap | `snap install nself` | Need to create |
| Arch Linux | AUR | `yay -S nself` | Need to create |
| Windows | Scoop | `scoop install nself` | Need to create |
| Windows | Chocolatey | `choco install nself` | Need to create |
| Windows | PowerShell | `irm install.ps1 \| iex` | Need to create |
| Docker | Docker Hub | `docker run acamarata/nself` | Need to create |
| Any | Direct | `curl install.sh \| bash` | ✅ Ready |

---

## Post-Release Tasks

### 1. Notifications
- [ ] Tweet/X announcement
- [ ] Discord/Slack announcement
- [ ] Reddit r/selfhosted post
- [ ] Dev.to article
- [ ] Update documentation site

### 2. Monitor
- [ ] Check GitHub Issues for installation problems
- [ ] Monitor package manager builds
- [ ] Verify download statistics

### 3. Hotfix Process
If critical issues found:
1. Create hotfix branch: `git checkout -b hotfix/0.3.7.1`
2. Fix issue
3. Tag as v0.3.7.1
4. Update only affected package managers

---

## Automation Scripts

### Create All Packages Script
```bash
#!/bin/bash
# release-all.sh

VERSION="0.3.7"

# Create GitHub release
gh release create "v$VERSION" \
  --title "nself v$VERSION" \
  --notes-file RELEASE_NOTES.md

# Build Docker image
docker buildx build --platform linux/amd64,linux/arm64 \
  -t acamarata/nself:$VERSION -t acamarata/nself:latest --push .

# Update Homebrew
./scripts/update-homebrew.sh $VERSION

# Update AUR
./scripts/update-aur.sh $VERSION

# Build Snap
snapcraft
snapcraft upload "nself_${VERSION}_amd64.snap"

echo "✅ All packages released!"
```

---

## Version Numbering

Follow Semantic Versioning:
- MAJOR.MINOR.PATCH (e.g., 0.3.7)
- MAJOR: Breaking changes
- MINOR: New features
- PATCH: Bug fixes

Hotfixes: 0.3.7.1, 0.3.7.2, etc.

---

## Missing Platforms to Consider

1. **FreeBSD Ports** - For BSD systems
2. **Nix/NixOS** - Declarative package manager
3. **Flatpak** - Desktop Linux apps
4. **AppImage** - Portable Linux apps
5. **winget** - Windows Package Manager (official)
6. **MacPorts** - Alternative to Homebrew
7. **pkg (npm)** - Distribute via npm for Node.js users
8. **Cargo (crates.io)** - If we add Rust components
9. **asdf** - Version manager
10. **SDKMAN** - For JVM ecosystem integration