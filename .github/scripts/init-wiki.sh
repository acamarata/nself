#!/bin/bash
# Initialize or update GitHub Wiki with documentation from /docs

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DOCS_DIR="$REPO_ROOT/docs"
WIKI_REPO="https://github.com/acamarata/nself.wiki.git"
WIKI_DIR="/tmp/nself-wiki-$$"

echo "═══════════════════════════════════════════════════════════════"
echo "           nself Wiki Initialization Script"
echo "═══════════════════════════════════════════════════════════════"
echo

# Clone wiki repo
echo "→ Cloning wiki repository..."
if git clone "$WIKI_REPO" "$WIKI_DIR" 2>/dev/null; then
    echo "✓ Wiki repository cloned"
else
    echo "✗ Wiki doesn't exist yet. Creating..."
    mkdir -p "$WIKI_DIR"
    cd "$WIKI_DIR"
    git init
    git remote add origin "$WIKI_REPO"
    echo "✓ Wiki repository initialized"
fi

cd "$WIKI_DIR"

# Copy all documentation
echo "→ Copying documentation files..."
cp -r "$DOCS_DIR"/* "$WIKI_DIR/" 2>/dev/null || true

# Rename files for wiki compatibility (wiki prefers dashes)
echo "→ Renaming files for wiki format..."
for file in *.MD *.md; do
    if [[ -f "$file" ]]; then
        # Convert underscores to dashes for better wiki URLs
        newname=$(echo "$file" | sed 's/_/-/g' | sed 's/\.MD$/.md/')
        if [[ "$file" != "$newname" ]]; then
            mv "$file" "$newname"
            echo "  Renamed: $file → $newname"
        fi
    fi
done

# Create comprehensive Home page
echo "→ Creating wiki home page..."
cat > Home.md << 'EOF'
# nself Documentation Wiki

> **Self-Hosted Infrastructure Made Simple** - Deploy Hasura, PostgreSQL, Auth, and Storage with one command.

[![GitHub](https://img.shields.io/github/stars/acamarata/nself?style=social)](https://github.com/acamarata/nself)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/acamarata/nself/blob/main/LICENSE)
[![Version](https://img.shields.io/badge/version-0.3.0-green.svg)](https://github.com/acamarata/nself/releases)
[![Support](https://img.shields.io/badge/support-Patreon-orange.svg)](https://patreon.com/acamarata)

## 📚 Documentation

### Getting Started
- **[[Installation|Installation]]** - Get nself up and running
- **[[Quick Start|Quick-Start]]** - Your first nself project
- **[[Examples|EXAMPLES]]** - Complete command examples with outputs
- **[[Troubleshooting|TROUBLESHOOTING]]** - Fix common issues

### Reference
- **[[API Reference|API]]** - Complete command documentation
- **[[Configuration|Configuration]]** - Environment variables and settings
- **[[Architecture|ARCHITECTURE]]** - System design and components
- **[[Directory Structure|DIRECTORY-STRUCTURE]]** - Project organization

### Development
- **[[Contributing|CONTRIBUTING]]** - How to contribute to nself
- **[[Code Style|CODE-STYLE]]** - Coding standards and conventions
- **[[Testing Strategy|TESTING-STRATEGY]]** - Testing approaches
- **[[Decisions|DECISIONS]]** - Architectural decision records

### Advanced Topics
- **[[Production Deployment|Production]]** - Deploy to production
- **[[Scaling|Scaling]]** - Scale your infrastructure
- **[[Security|Security]]** - Security best practices
- **[[Monitoring|Monitoring]]** - Monitor your services

### Changes
- **[[Changelog|CHANGELOG]]** - Version history
- **[[Roadmap|REFACTORING-ROADMAP]]** - Future plans

## 🚀 Quick Commands

```bash
# Initialize a new project
nself init myproject

# Build and start services
nself build
nself up

# Check status
nself status
nself urls

# View logs
nself logs -f
```

## 🏗️ Architecture Overview

nself manages a complete backend infrastructure:

```
┌─────────────────────────────────────────────────────────────┐
│                         NGINX Proxy                         │
│                    (SSL, Routing, Load Balancing)           │
└─────────────┬───────────┬───────────┬───────────┬──────────┘
              │           │           │           │
     ┌────────▼────┐ ┌────▼────┐ ┌───▼────┐ ┌───▼────┐
     │   Hasura    │ │  Auth   │ │ MinIO  │ │Custom  │
     │  GraphQL    │ │Service  │ │Storage │ │Services│
     └────────┬────┘ └────┬────┘ └───┬────┘ └───┬────┘
              │           │           │           │
     ┌────────▼───────────▼───────────▼───────────▼────┐
     │              PostgreSQL Database                 │
     └──────────────────────────────────────────────────┘
              │                             │
     ┌────────▼────┐                ┌──────▼─────┐
     │    Redis    │                │   Backup   │
     │   (Cache)   │                │  Storage   │
     └─────────────┘                └────────────┘
```

## 📖 Documentation Standards

This wiki is automatically synchronized from the `/docs` folder in the main repository.

### For Contributors
1. Edit files in `/docs` folder of main repo
2. Submit PR with changes
3. Wiki updates automatically on merge

### For Users
- Use the search feature to find specific topics
- Check [[Examples|EXAMPLES]] for practical usage
- Refer to [[Troubleshooting|TROUBLESHOOTING]] for issues

## 🔧 Core Features

- **🚀 One-Command Setup** - `nself init && nself up`
- **🔐 Built-in Authentication** - JWT-based auth service
- **📊 GraphQL API** - Hasura GraphQL engine
- **💾 Object Storage** - MinIO S3-compatible storage
- **🗄️ PostgreSQL Database** - Reliable data persistence
- **🔄 Hot Reload** - Development mode with auto-restart
- **🛡️ SSL Certificates** - Automatic local SSL
- **🎯 Service Scaffolding** - Generate new services
- **🔍 Health Monitoring** - Built-in health checks
- **📦 Docker Compose v2** - Modern container orchestration

## 💡 Common Use Cases

### Local Development
```bash
nself init myapp localhost
nself build
nself up
nself hot-reload
```

### Production Deployment
```bash
nself init myapp api.example.com
nself prod --domain api.example.com --ssl
nself validate-env
ENV=production nself up -d
```

### Adding Services
```bash
nself scaffold nest api-gateway
nself scaffold bull worker
nself build
nself up
```

## 🆘 Getting Help

- **Documentation**: This wiki
- **Issues**: [GitHub Issues](https://github.com/acamarata/nself/issues)
- **Discussions**: [GitHub Discussions](https://github.com/acamarata/nself/discussions)
- **Support Development**: [Patreon](https://patreon.com/acamarata)

## 📊 Status

- ✅ Core functionality complete
- ✅ Modular architecture (v0.3.0)
- ✅ Auto-fix system
- ✅ Comprehensive documentation
- 🚧 Kubernetes support (planned)
- 🚧 Cloud provider integrations (planned)

## 📄 License

MIT License - See [LICENSE](https://github.com/acamarata/nself/blob/main/LICENSE)

---

*This wiki is automatically generated from `/docs`. Do not edit directly.*
EOF

echo "✓ Home page created"

# Create Installation page
echo "→ Creating Installation page..."
cat > Installation.md << 'EOF'
# Installation

## Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

## Prerequisites

- **Operating System**: macOS, Linux, WSL2
- **Docker**: 20.10 or higher
- **Docker Compose**: v2 (included with Docker Desktop)
- **Bash**: 4.0 or higher
- **curl**: For downloading
- **Git**: For updates

## Installation Methods

### Method 1: Automated Install (Recommended)

```bash
# Download and run installer
curl -sSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash

# Reload shell configuration
source ~/.bashrc  # or ~/.zshrc for zsh
```

### Method 2: Manual Installation

```bash
# Clone repository
git clone https://github.com/acamarata/nself.git ~/.nself

# Add to PATH
echo 'export PATH="$HOME/.nself/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# Make executable
chmod +x ~/.nself/bin/nself.sh

# Create symlink
ln -sf ~/.nself/bin/nself.sh ~/.nself/bin/nself
```

### Method 3: System-wide Installation

```bash
# Download installer with sudo
sudo curl -sSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | sudo bash

# This installs to /usr/local/nself
```

## Verify Installation

```bash
# Check version
nself version

# Run diagnostics
nself doctor

# Get help
nself help
```

## Update nself

```bash
# Check for updates
nself update --check

# Update to latest version
nself update
```

## Uninstall

```bash
# Remove nself directory
rm -rf ~/.nself

# Remove from PATH (edit ~/.bashrc or ~/.zshrc)
# Remove the line: export PATH="$HOME/.nself/bin:$PATH"
```

## Platform-Specific Notes

### macOS
- Install Docker Desktop from [docker.com](https://docker.com)
- Xcode Command Line Tools required: `xcode-select --install`

### Linux
- Install Docker: [Docker Installation Guide](https://docs.docker.com/engine/install/)
- Add user to docker group: `sudo usermod -aG docker $USER`

### Windows (WSL2)
- Install WSL2: `wsl --install`
- Install Docker Desktop with WSL2 backend
- Run nself inside WSL2 terminal

## Troubleshooting Installation

See [[Troubleshooting|TROUBLESHOOTING#installation-issues]] for common installation problems.

## Next Steps

After installation, see [[Quick Start|Quick-Start]] to create your first project.
EOF

echo "✓ Installation page created"

# Create Quick Start page
echo "→ Creating Quick Start page..."
cat > Quick-Start.md << 'EOF'
# Quick Start Guide

Get your first nself project running in 5 minutes!

## Step 1: Initialize Project

```bash
# Create project directory
mkdir ~/myproject
cd ~/myproject

# Initialize nself project
nself init
```

Output:
```
╔════════════════════════════════════════════════════════════════╗
║                NSELF PROJECT INITIALIZATION                    ║
╚════════════════════════════════════════════════════════════════╝

[INFO] Initializing project: myproject
[SUCCESS] Created .env.local
[SUCCESS] Created project structure
```

## Step 2: Build Infrastructure

```bash
nself build
```

This will:
- Generate SSL certificates
- Create docker-compose.yml
- Configure services
- Build Docker images

## Step 3: Start Services

```bash
nself up
```

Output:
```
╔════════════════════════════════════════════════════════════════╗
║                    ALL SERVICES RUNNING                        ║
╚════════════════════════════════════════════════════════════════╝

Service URLs:
  GraphQL API:     https://api.localhost
  Authentication:  https://auth.localhost
  Storage:         https://storage.localhost
  Admin Console:   https://api.localhost/console
```

## Step 4: Access Services

Open your browser and visit:
- **GraphQL Console**: https://api.localhost/console
- **Authentication**: https://auth.localhost
- **Storage**: https://storage.localhost

## Step 5: Check Status

```bash
# View service status
nself status

# View logs
nself logs -f

# Get all URLs
nself urls
```

## What's Next?

### Development Workflow

1. **Enable hot reload for development:**
   ```bash
   nself hot-reload
   ```

2. **Create a new service:**
   ```bash
   nself scaffold nest api-gateway
   ```

3. **Work with the database:**
   ```bash
   nself db console
   nself db migrate
   nself db seed
   ```

### Production Deployment

1. **Configure for production:**
   ```bash
   nself prod --domain api.example.com
   ```

2. **Validate configuration:**
   ```bash
   nself validate-env
   ```

3. **Deploy:**
   ```bash
   ENV=production nself up -d
   ```

## Common Commands

| Command | Description |
|---------|-------------|
| `nself up` | Start all services |
| `nself down` | Stop all services |
| `nself status` | Check service status |
| `nself logs` | View logs |
| `nself doctor` | Run diagnostics |
| `nself urls` | Show service URLs |
| `nself db console` | Access database |

## Examples

See [[Examples|EXAMPLES]] for comprehensive command examples with outputs.

## Troubleshooting

If you encounter issues, see [[Troubleshooting|TROUBLESHOOTING]] or run:
```bash
nself doctor
```

## Learn More

- [[API Reference|API]] - All commands
- [[Architecture|ARCHITECTURE]] - System design
- [[Contributing|CONTRIBUTING]] - Contribute to nself
EOF

echo "✓ Quick Start page created"

# Create sidebar navigation
echo "→ Creating sidebar navigation..."
cat > _Sidebar.md << 'EOF'
## Getting Started
- [[Home]]
- [[Installation]]
- [[Quick Start|Quick-Start]]

## Documentation
- [[API Reference|API]]
- [[Examples|EXAMPLES]]
- [[Configuration]]
- [[Troubleshooting|TROUBLESHOOTING]]

## Architecture
- [[System Design|ARCHITECTURE]]
- [[Directory Structure|DIRECTORY-STRUCTURE]]
- [[Decisions|DECISIONS]]

## Development
- [[Contributing|CONTRIBUTING]]
- [[Code Style|CODE-STYLE]]
- [[Testing|TESTING-STRATEGY]]

## Reference
- [[Changelog|CHANGELOG]]
- [[Roadmap|REFACTORING-ROADMAP]]
- [[Output Format|OUTPUT-FORMATTING]]

## Support
- [GitHub](https://github.com/acamarata/nself)
- [Issues](https://github.com/acamarata/nself/issues)
- [Patreon](https://patreon.com/acamarata)
EOF

echo "✓ Sidebar created"

# Create footer
echo "→ Creating footer..."
cat > _Footer.md << 'EOF'
---
*nself v0.3.0* | [GitHub](https://github.com/acamarata/nself) | [Support on Patreon](https://patreon.com/acamarata) | [Report Issues](https://github.com/acamarata/nself/issues)

*This wiki is automatically synchronized from `/docs`. To contribute, submit PRs to the main repository.*
EOF

echo "✓ Footer created"

# Stage all changes
echo "→ Staging changes..."
git add -A

# Check for changes
if git diff --staged --quiet; then
    echo "✓ Wiki is already up to date"
else
    # Commit and push
    echo "→ Committing changes..."
    git commit -m "Sync documentation from main repository

Updated: $(date '+%Y-%m-%d %H:%M:%S')
Source: https://github.com/acamarata/nself/tree/main/docs"

    echo "→ Pushing to GitHub..."
    if git push origin master 2>/dev/null || git push origin main 2>/dev/null; then
        echo "✓ Wiki updated successfully"
    else
        echo "⚠ Wiki push failed - you may need to create the wiki first"
        echo "  Visit: https://github.com/acamarata/nself/wiki"
        echo "  Create a page, then run this script again"
    fi
fi

# Cleanup
cd "$REPO_ROOT"
rm -rf "$WIKI_DIR"

echo
echo "═══════════════════════════════════════════════════════════════"
echo "                    Wiki Sync Complete!"
echo "═══════════════════════════════════════════════════════════════"
echo
echo "View your wiki at: https://github.com/acamarata/nself/wiki"
echo
echo "To update the wiki in the future:"
echo "  1. Edit files in /docs"
echo "  2. Run: .github/scripts/init-wiki.sh"
echo "  3. Or use GitHub Actions (automatic)"
echo