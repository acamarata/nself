# Installation Guide

This comprehensive guide covers all installation methods and platform-specific instructions for nself.

## 📋 Table of Contents

- [System Requirements](#system-requirements)
- [Installation Methods](#installation-methods)
- [Platform-Specific Instructions](#platform-specific-instructions)
- [Verification](#verification)
- [Troubleshooting](#troubleshooting)
- [Upgrading](#upgrading)
- [Uninstalling](#uninstalling)

## System Requirements

### Minimum Requirements
- **CPU**: 2 cores
- **RAM**: 4GB
- **Storage**: 10GB free space
- **OS**: Linux, macOS, Windows (WSL2)
- **Docker**: Version 20.10+
- **Docker Compose**: Version 2.0+

### Recommended Requirements
- **CPU**: 4+ cores
- **RAM**: 8GB+
- **Storage**: 20GB+ free space
- **Network**: Stable internet for pulling images

### Software Dependencies
- **Required**:
  - Docker Engine or Docker Desktop
  - Docker Compose v2
  - Bash shell (4.0+)
  - Git

- **Optional**:
  - mkcert (for local SSL)
  - jq (for JSON processing)
  - curl or wget
  - Make (for Makefile targets)

## Installation Methods

### Method 1: Quick Install (Recommended)

```bash
# One-line installer (coming soon)
curl -sSL https://get.nself.org | bash

# Or with wget
wget -qO- https://get.nself.org | bash
```

### Method 2: Git Clone

```bash
# Clone the repository
git clone https://github.com/acamarata/nself.git
cd nself

# Make executable
chmod +x bin/nself

# Add to PATH (optional)
export PATH="$PATH:$(pwd)/bin"
echo 'export PATH="$PATH:'$(pwd)'/bin"' >> ~/.bashrc
```

### Method 3: Download Release

```bash
# Download latest release
wget https://github.com/acamarata/nself/releases/latest/download/nself-v0.3.9.tar.gz

# Extract
tar -xzf nself-v0.3.9.tar.gz
cd nself-v0.3.9

# Make executable
chmod +x bin/nself
```

### Method 4: Docker Image (Coming Soon)

```bash
# Pull the Docker image
docker pull nself/cli:latest

# Create alias
alias nself='docker run --rm -it -v $(pwd):/workspace -v /var/run/docker.sock:/var/run/docker.sock nself/cli:latest'
```

## Platform-Specific Instructions

### 🐧 Linux

#### Ubuntu/Debian

```bash
# Install Docker
curl -fsSL https://get.docker.com | bash
sudo usermod -aG docker $USER
newgrp docker

# Install Docker Compose v2
sudo apt-get update
sudo apt-get install docker-compose-plugin

# Install nself
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself
sudo ln -s $(pwd)/bin/nself /usr/local/bin/nself
```

#### RHEL/CentOS/Fedora

```bash
# Install Docker
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER

# Install nself
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself
sudo ln -s $(pwd)/bin/nself /usr/local/bin/nself
```

#### Arch Linux

```bash
# Install Docker
sudo pacman -S docker docker-compose
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER

# Install nself
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself
sudo ln -s $(pwd)/bin/nself /usr/local/bin/nself
```

### 🍎 macOS

#### Using Homebrew

```bash
# Install Docker Desktop
brew install --cask docker

# Start Docker Desktop
open /Applications/Docker.app

# Install nself
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself

# Add to PATH
echo 'export PATH="$PATH:'$(pwd)'/bin"' >> ~/.zshrc
source ~/.zshrc
```

#### Manual Installation

1. Download [Docker Desktop for Mac](https://www.docker.com/products/docker-desktop)
2. Install and start Docker Desktop
3. Clone and install nself:

```bash
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself
export PATH="$PATH:$(pwd)/bin"
```

### 🪟 Windows

#### WSL2 (Recommended)

1. **Enable WSL2**:
```powershell
# Run as Administrator
wsl --install
wsl --set-default-version 2
```

2. **Install Docker Desktop**:
   - Download [Docker Desktop for Windows](https://www.docker.com/products/docker-desktop)
   - Enable WSL2 backend in Docker Desktop settings

3. **Install nself in WSL2**:
```bash
# Inside WSL2 terminal
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself
export PATH="$PATH:$(pwd)/bin"
```

#### Git Bash

```bash
# In Git Bash
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself
export PATH="$PATH:$(pwd)/bin"
```

### 🐳 Docker Containers (LXC/Docker-in-Docker)

For containerized environments:

```bash
# Install Docker-in-Docker
apk add --no-cache docker docker-compose

# Start Docker daemon
dockerd &

# Install nself
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself
```

## Verification

### Verify Installation

```bash
# Check nself version
nself version

# Check dependencies
nself doctor

# Sample output:
# ✅ Docker: 24.0.5
# ✅ Docker Compose: v2.20.2
# ✅ Bash: 5.1.16
# ✅ Git: 2.34.1
# ✅ System: Ready
```

### Test Installation

```bash
# Create test project
mkdir test-nself && cd test-nself

# Initialize
nself init

# Build and start
nself build
nself start

# Check status
nself status
```

## Troubleshooting

### Common Issues

#### Docker Not Found

```bash
# Linux
curl -fsSL https://get.docker.com | bash

# macOS
brew install --cask docker

# Windows
# Install Docker Desktop with WSL2 backend
```

#### Permission Denied

```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Fix nself permissions
chmod +x bin/nself
```

#### Docker Compose v1 Instead of v2

```bash
# Check version
docker compose version

# If showing v1, install v2:
# Ubuntu/Debian
sudo apt-get install docker-compose-plugin

# Or install manually
DOCKER_CONFIG=${DOCKER_CONFIG:-$HOME/.docker}
mkdir -p $DOCKER_CONFIG/cli-plugins
curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 -o $DOCKER_CONFIG/cli-plugins/docker-compose
chmod +x $DOCKER_CONFIG/cli-plugins/docker-compose
```

#### WSL2 Issues

```powershell
# Update WSL2
wsl --update

# Set default version
wsl --set-default-version 2

# Restart WSL
wsl --shutdown
```

### Platform-Specific Issues

#### macOS Silicon (M1/M2)

```bash
# Ensure Rosetta is installed
softwareupdate --install-rosetta

# Use platform-specific images
export DOCKER_DEFAULT_PLATFORM=linux/amd64
```

#### Linux SELinux

```bash
# Temporarily disable for testing
sudo setenforce 0

# Or add Docker exceptions
sudo setsebool -P container_manage_cgroup on
```

## Upgrading

### Upgrade nself

```bash
# Pull latest changes
cd /path/to/nself
git pull origin main

# Or download new release
wget https://github.com/acamarata/nself/releases/latest/download/nself-latest.tar.gz
tar -xzf nself-latest.tar.gz --strip-components=1
```

### Upgrade Docker Images

```bash
# Pull latest images
nself update

# Or manually
docker compose pull
nself build --no-cache
```

### Migration Between Versions

```bash
# Backup before upgrading
nself backup create pre-upgrade

# Upgrade
git pull origin main

# Rebuild
nself build

# Restart services
nself restart
```

## Uninstalling

### Remove nself

```bash
# Stop all services
nself stop

# Clean up containers and volumes (optional)
nself clean --all

# Remove nself
rm -rf /path/to/nself

# Remove from PATH
# Edit ~/.bashrc or ~/.zshrc and remove nself PATH entry
```

### Remove Docker (Optional)

```bash
# Linux
sudo apt-get purge docker-ce docker-ce-cli containerd.io
sudo rm -rf /var/lib/docker

# macOS
brew uninstall --cask docker

# Windows
# Uninstall Docker Desktop from Control Panel
```

## Advanced Installation

### Custom Installation Path

```bash
# Install to custom location
NSELF_HOME=/opt/nself
sudo git clone https://github.com/acamarata/nself.git $NSELF_HOME
sudo chmod +x $NSELF_HOME/bin/nself
sudo ln -s $NSELF_HOME/bin/nself /usr/local/bin/nself
```

### System-Wide Installation

```bash
# Clone to system location
sudo git clone https://github.com/acamarata/nself.git /opt/nself

# Create system link
sudo ln -s /opt/nself/bin/nself /usr/local/bin/nself

# Set permissions
sudo chmod +x /opt/nself/bin/nself
```

### Air-Gapped Installation

For environments without internet:

1. **Download on connected machine**:
```bash
# Download nself and all images
./scripts/download-offline.sh
```

2. **Transfer to air-gapped system**:
```bash
# Copy archive
scp nself-offline.tar.gz user@airgapped:/tmp/
```

3. **Install on air-gapped system**:
```bash
# Extract and load
tar -xzf nself-offline.tar.gz
cd nself-offline
./install-offline.sh
```

## Next Steps

✅ Installation complete! Now you can:

1. **[Follow the Quick Start Guide](Quick-Start)** - Get running in 5 minutes
2. **[Configure Your Environment](Configuration)** - Customize settings
3. **[Explore Commands](Commands)** - Learn all CLI commands
4. **[Set Up Services](Services)** - Enable additional features

---

**Need help?** Check the [Troubleshooting Guide](Troubleshooting) or [create an issue](https://github.com/acamarata/nself/issues).