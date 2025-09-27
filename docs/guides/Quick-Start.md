# Quick Start Guide

Get nself up and running in under 5 minutes! This guide will walk you through the fastest path to a working backend stack.

## 📋 Prerequisites

Before you begin, ensure you have:
- ✅ **Docker Desktop** or Docker Engine installed ([Install Docker](https://docs.docker.com/get-docker/))
- ✅ **Docker Compose v2** (included with Docker Desktop)
- ✅ **Git** for cloning the repository
- ✅ **4GB RAM** minimum available
- ✅ **10GB disk space** for Docker images

## 🚀 5-Minute Setup

### Step 1: Clone and Install

```bash
# Clone the repository
git clone https://github.com/acamarata/nself.git
cd nself

# Make the CLI executable
chmod +x bin/nself

# Optional: Add to PATH for global access
echo 'export PATH="$PATH:$HOME/nself/bin"' >> ~/.bashrc
source ~/.bashrc
```

### Step 2: Create Your First Project

```bash
# Create a new project directory
mkdir my-app && cd my-app

# Initialize with smart defaults
nself init

# This creates a .env file with:
# - Auto-generated secure passwords
# - Default ports configured
# - Local domain setup (*.local.nself.org)
```

### Step 3: Build and Start

```bash
# Build Docker images and generate configurations
nself build

# Start all services
nself start

# Check service status
nself status
```

### Step 4: Enable Admin UI (Optional but Recommended)

```bash
# Enable the web-based admin interface
nself admin enable

# Set admin password (optional, uses temp password by default)
nself admin password mySecurePassword123

# Open admin UI in browser
nself admin open
# Or navigate to: http://localhost:3100
```

## ✅ Verify Installation

Your backend stack is now running! Here are the default endpoints:

| Service | URL | Credentials |
|---------|-----|-------------|
| **Admin UI** | http://localhost:3100 | admin / (your password) |
| **Hasura Console** | http://localhost:8080 | No auth in dev mode |
| **PostgreSQL** | localhost:5432 | postgres / (auto-generated) |
| **MinIO Console** | http://localhost:9001 | minioadmin / minioadmin |
| **Auth Service** | http://localhost:4000 | JWT-based |
| **Storage API** | http://localhost:4002 | Via Auth JWT |
| **Mailpit (Dev Email)** | http://localhost:8025 | No auth required |

## 🎯 Next Steps

### Quick Commands to Try

```bash
# View logs for all services
nself logs

# View logs for specific service
nself logs postgres

# Stop all services
nself stop

# Restart a specific service
nself restart hasura

# Open database CLI
nself db

# Check system health
nself doctor
```

### Explore Key Features

1. **[Configure Your Environment](Configuration)**
   - Customize ports and domains
   - Enable optional services
   - Set up production configs

2. **[Add Microservices](Service-Templates)**
   ```bash
   # Enable Node.js microservices
   echo "SERVICES_ENABLED=true" >> .env
   echo "NODEJS_SERVICES=api,workers" >> .env
   nself build && nself restart
   ```

3. **[Set Up SSL](SSL)**
   ```bash
   # Generate local SSL certificates
   nself ssl
   
   # Trust certificates in browser
   nself trust
   ```

4. **[Configure Email](Email)**
   ```bash
   # Interactive email setup
   nself email
   ```

## 🏗️ Sample Project Structure

After initialization, your project will have:

```
my-app/
├── .env                     # Your configuration (git-ignored)
├── .env.dev                 # Development defaults
├── docker-compose.yml       # Generated orchestration
├── docker-compose.override.yml # Dev overrides
├── nginx/
│   └── default.conf        # Reverse proxy config
├── ssl/
│   └── certificates/       # SSL certificates
├── hasura/
│   ├── metadata/          # GraphQL metadata
│   └── migrations/        # Database migrations
└── services/              # Your microservices (if enabled)
    ├── nodejs/
    │   ├── api/
    │   └── workers/
    └── python/
        └── ml/
```

## 🔧 Common Tasks

### Create a Database Backup
```bash
nself backup create my-backup
nself backup list
```

### Reset to Clean State
```bash
nself reset
# Confirms before deleting data
```

### Deploy to Production
```bash
# Generate production config
nself prod

# Review and edit .env.prod
nano .env.prod

# Deploy (requires server setup)
nself deploy production
```

## 🚨 Troubleshooting Quick Fixes

### Docker Not Running?
```bash
nself doctor
# Provides platform-specific instructions
```

### Port Conflicts?
```bash
# Auto-fix will reassign ports
AUTO_FIX=true nself build
```

### Services Not Starting?
```bash
# Check logs for specific service
nself logs [service-name]

# Restart everything
nself restart
```

### Can't Access Services?
```bash
# Check URLs and ports
nself urls

# Verify services are healthy
nself status
```

## 📚 Learn More

- **[Complete Commands Reference](Commands)** - All 35+ CLI commands
- **[Configuration Guide](Configuration)** - Detailed environment setup
- **[Architecture Overview](Architecture)** - Understanding the stack
- **[Service Templates](Service-Templates)** - Adding microservices
- **[Production Deployment](Deployment)** - Going live

## 💡 Pro Tips

1. **Use Admin UI** - The web interface at port 3100 makes management much easier
2. **Enable Only What You Need** - Start with core services, add others as needed
3. **Use Smart Defaults** - The auto-configuration handles most complexity
4. **Check Doctor Command** - `nself doctor` diagnoses and fixes most issues
5. **Read the Logs** - `nself logs [service]` usually reveals the problem

## 🆘 Need Help?

- 📖 Check the [FAQ](FAQ) for common questions
- 🐛 See [Troubleshooting](Troubleshooting) for detailed solutions
- 💬 Ask in [GitHub Discussions](https://github.com/acamarata/nself/discussions)
- 🎫 Report bugs via [GitHub Issues](https://github.com/acamarata/nself/issues)

---

**Ready for more?** Continue to [Basic Configuration](Basic-Configuration) →