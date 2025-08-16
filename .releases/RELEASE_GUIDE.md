# nself v0.3.7 Release Guide

All package files have been prepared for the v0.3.7 release across multiple platforms.

## 1. Homebrew (macOS/Linux)

**Status**: Repository prepared at `~/homebrew-nself`

### To publish:
1. Create GitHub repository: https://github.com/new
   - Name: `homebrew-nself`
   - Public repository
2. Push the tap:
   ```bash
   cd ~/homebrew-nself
   git push -u origin main
   ```
3. Users install with:
   ```bash
   brew tap acamarata/nself
   brew install nself
   ```

## 2. Docker Hub

**Status**: Dockerfile and scripts ready in `.releases/docker/`

### To publish:
```bash
# Build the image
cd /Users/admin/Sites/nself
./.releases/docker/build-and-push.sh 0.3.7

# Login to Docker Hub
docker login -u acamarata

# Push images
docker push acamarata/nself:0.3.7
docker push acamarata/nself:latest
```

Users install with:
```bash
docker pull acamarata/nself:latest
```

## 3. Ubuntu PPA

**Status**: Debian package files ready in `.releases/debian/`

### To publish:
1. Install build tools:
   ```bash
   sudo apt-get install devscripts debhelper dput
   ```

2. Build source package:
   ```bash
   cd /Users/admin/Sites/nself
   debuild -S -sa
   ```

3. Upload to Launchpad PPA:
   ```bash
   dput ppa:acamarata/nself ../nself_0.3.7-1_source.changes
   ```

Users install with:
```bash
sudo add-apt-repository ppa:acamarata/nself
sudo apt update
sudo apt install nself
```

## 4. Fedora COPR

**Status**: RPM spec file ready in `.releases/rpm/`

### To publish:
1. Create SRPM:
   ```bash
   cd /Users/admin/Sites/nself
   tar czf ~/rpmbuild/SOURCES/nself-0.3.7.tar.gz --transform 's,^,nself-0.3.7/,' *
   rpmbuild -bs .releases/rpm/nself.spec
   ```

2. Upload to COPR:
   - Go to: https://copr.fedorainfracloud.org/
   - Create new project: `nself`
   - Upload SRPM from `~/rpmbuild/SRPMS/`

Users install with:
```bash
sudo dnf copr enable acamarata/nself
sudo dnf install nself
```

## 5. Arch Linux AUR

**Status**: PKGBUILD ready in `.releases/aur/`

### To publish:
1. Clone AUR repository:
   ```bash
   git clone ssh://aur@aur.archlinux.org/nself.git ~/aur-nself
   ```

2. Copy files and push:
   ```bash
   cp /Users/admin/Sites/nself/.releases/aur/* ~/aur-nself/
   cd ~/aur-nself
   git add .
   git commit -m "Release v0.3.7"
   git push
   ```

Users install with:
```bash
yay -S nself
# or
git clone https://aur.archlinux.org/nself.git
cd nself
makepkg -si
```

## Version Information

- **Version**: 0.3.7
- **SHA256**: 842e571cba1c5d0bdd7a50f066e560a53cde99fd6225f87438b71d1c112bc3c4
- **Source**: https://github.com/acamarata/nself/archive/refs/tags/v0.3.7.tar.gz

## Testing Commands

After publishing, test each platform:

```bash
# Homebrew
brew install nself && nself version

# Docker
docker run acamarata/nself:latest version

# Ubuntu/Debian
sudo apt install nself && nself version

# Fedora
sudo dnf install nself && nself version

# Arch
yay -S nself && nself version
```

## Support

- GitHub: https://github.com/acamarata/nself
- Email: aric.camarata@gmail.com