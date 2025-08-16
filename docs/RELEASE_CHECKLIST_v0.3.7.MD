# nself v0.3.7 Release Checklist

## Pre-Release Testing ✅
- [x] Fixed nginx SSL configuration issues
- [x] Fixed logs command execution
- [x] Fixed db help display  
- [x] Version updated to v0.3.7
- [x] All commands tested and working

## Release Steps

### 1. Prepare Release
```bash
# Ensure you're on main branch
git checkout main
git pull origin main

# Run the automated release script
./scripts/release.sh v0.3.7
```

This will:
- Update VERSION file
- Run tests
- Create git tag
- Push to GitHub
- Create GitHub release (if gh CLI installed)

### 2. GitHub Release
If not created automatically, go to:
https://github.com/acamarata/nself/releases/new

- Tag: v0.3.7
- Title: nself v0.3.7 - Critical Fixes & Improvements
- Description:
```markdown
## 🚀 nself v0.3.7

### 🐛 Bug Fixes
- Fixed nginx SSL certificate path configuration and http2 deprecation warnings
- Resolved logs command missing header source and error handling
- Fixed db help display not showing without arguments
- Improved error handling with proper `set +e` usage

### ✨ Improvements
- Standardized command headers across all commands
- Enhanced SSL certificate generation and copying
- Better error messages and debugging

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

### 3. Package Managers

#### Homebrew (macOS/Linux)
```bash
# Create/update your tap
./scripts/update-homebrew.sh v0.3.7

# Users install with:
brew tap acamarata/nself
brew install nself
```

#### Docker Hub
```bash
# Build and push multi-arch image
./scripts/update-docker.sh v0.3.7

# Users run with:
docker run -v /var/run/docker.sock:/var/run/docker.sock acamarata/nself:latest
```

#### Windows (PowerShell)
```powershell
# Create Windows installers
powershell -ExecutionPolicy Bypass -File scripts/create-windows-installer.ps1 v0.3.7

# Upload install-windows.ps1 to your server
# Users install with:
irm https://raw.githubusercontent.com/acamarata/nself/main/install-windows.ps1 | iex
```

### 4. Future Package Managers (To Create)

#### AUR (Arch Linux)
```bash
# Create AUR package
git clone ssh://aur@aur.archlinux.org/nself.git
cd nself
# Update PKGBUILD with v0.3.7
makepkg -si
git push
```

#### Snap (Ubuntu)
```bash
# Build snap package
snapcraft
snapcraft upload nself_0.3.7_amd64.snap
snapcraft release nself 0.3.7 stable
```

#### Scoop (Windows)
```json
// Submit PR to scoop-extras with nself.json
// Users install with:
scoop bucket add extras
scoop install nself
```

#### Chocolatey (Windows)
```powershell
cd chocolatey
choco pack
choco push nself.0.3.7.nupkg --source https://push.chocolatey.org/
```

### 5. Verification
- [ ] Test direct install: `curl -fsSL ... | bash`
- [ ] Test Docker image: `docker run acamarata/nself:latest version`
- [ ] Test Homebrew (if available)
- [ ] Test Windows installer (if available)
- [ ] Check GitHub release page

### 6. Announcement
- [ ] Update nself.org website (if exists)
- [ ] Post on Twitter/X
- [ ] Post on r/selfhosted
- [ ] Discord/Slack announcement
- [ ] Dev.to article (optional)

## Package Manager Status

| Platform | Manager | Status | Priority |
|----------|---------|--------|----------|
| Linux/macOS | Direct Install | ✅ Ready | Essential |
| macOS | Homebrew | 🔧 Script Ready | High |
| Docker | Docker Hub | 🔧 Script Ready | High |
| Windows | PowerShell | 🔧 Script Ready | High |
| Windows | WSL | ✅ Works via install.sh | High |
| Arch | AUR | ❌ Not Created | Medium |
| Ubuntu | Snap | ❌ Not Created | Medium |
| Windows | Scoop | 📝 Manifest Ready | Medium |
| Windows | Chocolatey | 📝 Package Ready | Low |
| Windows | winget | ❌ Not Created | Low |
| BSD | Ports | ❌ Not Created | Low |
| Linux | Flatpak | ❌ Not Created | Low |
| Multi | Nix | ❌ Not Created | Low |

## Post-Release
- [ ] Monitor GitHub Issues for problems
- [ ] Check download statistics
- [ ] Prepare hotfix process if needed (v0.3.7.1)

## Notes
- Main distribution channel: GitHub + direct install script
- Docker image provides cross-platform support
- Windows users can use WSL2 as primary method
- Package managers can be added incrementally