# nself - Nhost self-hosted stack and more, in seconds!

[![Version](https://img.shields.io/badge/version-0.3.9-blue.svg)](https://github.com/acamarata/nself/releases)
[![License](https://img.shields.io/badge/license-Personal%20Free%20%7C%20Commercial-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey.svg)](https://github.com/acamarata/nself#-supported-platforms)
[![Docker](https://img.shields.io/badge/docker-required-blue.svg)](https://www.docker.com/get-started)
[![CI Status](https://github.com/acamarata/nself/actions/workflows/ci.yml/badge.svg)](https://github.com/acamarata/nself/actions)

Deploy a feature-complete backend infrastructure on your own servers with PostgreSQL, Hasura GraphQL, Redis, Auth, Storage, and optional microservices. Works seamlessly across local development, staging, and production with automated SSL, smart defaults, and production-ready configurations with enterprise monitoring.

**Based on [Nhost.io](https://nhost.io) for self-hosting!** and expanded with more features. Copy the below command in Terminal to install and get up and running in seconds!

```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

> **🚀 v0.3.9**: Admin UI integration, critical bug fixes, improved stability! Features web-based monitoring dashboard, real-time service health, Docker container management, and comprehensive backup UI. All major commands tested and working. [See release notes](docs/RELEASES.md)

📋 **[View Roadmap](docs/ROADMAP.md)** - See development roadmap and what's coming in v0.4.0!

nself is *the* CLI for Nhost self-hosted deployments - with extras and an opinionated setup that makes everything smooth. From zero to production-ready backend in under 5 minutes. Just edit an env file with your preferences and build!

## 🚀 Why nself?

### ⚡ Lightning Fast Setup
- **Under 5 minutes** from zero to running backend
- One command installation, initialization, and deployment
- Smart defaults that just work™

### 🎯 Complete Feature Set
- **Full Nhost Stack**: PostgreSQL, Hasura GraphQL, Auth, Storage, Functions
- **Plus Extras**: Redis, TimescaleDB, PostGIS, pgvector extensions
- **Email Management**: 16+ providers (SendGrid, AWS SES, Mailgun, etc.) with zero-config dev
- **Microservices Ready**: Built-in support for NestJS, BullMQ, GoLang, and Python services
- **Production SSL**: Automatic trusted certificates (no browser warnings!)

### 🛠️ Developer Experience
- **Admin Dashboard**: Web-based monitoring UI at localhost:3100
- **Single Config File**: One `.env` controls everything
- **Zero Configuration**: Email, SSL, and services work out of the box
- **Hot Reload**: Changes apply instantly without rebuild
- **Multi-Environment**: Same setup works locally, staging, and production
- **No Lock-in**: Standard Docker Compose under the hood
- **Debugging Tools**: `doctor`, `status`, `logs` commands for troubleshooting

### 🔐 Production Ready
- **Security First**: Automatic SSL trust + secure password generation
- **Email Ready**: Production email in 2 minutes with guided setup
- **Battle Tested**: Based on proven Nhost.io infrastructure
- **Scale Ready**: From hobby projects to enterprise deployments
- **Zero Downtime**: Rolling updates and health checks built-in

## 📋 Prerequisites

- Linux, macOS, or Windows with WSL
- Docker and Docker Compose (installer will help install these)
- `curl` (for installation)

## 🔧 Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

### Package Managers

#### macOS (Homebrew)
```bash
brew tap acamarata/nself
brew install nself
```

#### Debian/Ubuntu (.deb)
```bash
wget https://github.com/acamarata/nself/releases/download/v0.3.9/nself_0.3.9_all.deb
sudo dpkg -i nself_0.3.9_all.deb
```

#### RHEL/CentOS/Fedora (.rpm)
```bash
wget https://github.com/acamarata/nself/releases/download/v0.3.9/nself-0.3.9-1.noarch.rpm
sudo rpm -i nself-0.3.9-1.noarch.rpm
```

The installer will:
- ✅ Auto-detect existing installations and offer updates
- 📊 Show visual progress with loading spinners
- 🔍 Check and help install Docker/Docker Compose if needed
- 📦 Download nself CLI to `~/.nself/bin`
- 🔗 Add nself to your PATH automatically
- 🚀 Create a global `nself` command

### Updating nself

To update to the latest version:

```bash
nself update
```

The updater will:
- Check for new versions automatically
- Show version comparison (current → latest)
- Download and install updates seamlessly
- Preserve your existing configurations

## 🏁 Quick Start - 3 Commands to Backend Bliss

```bash
# 1. Create and enter project directory
mkdir my-awesome-backend && cd my-awesome-backend

# 2. Initialize with smart defaults (or use wizard)
nself init --wizard  # Interactive setup (NEW in v0.3.9)
# or: nself init     # Quick setup with defaults

# 3. Build and launch everything
nself build && nself start
# URLs for enabled services will be shown in the output
```

**That's it!** Your complete backend is now running at:
- 🚀 GraphQL API: https://api.local.nself.org
- 🔐 Auth Service: https://auth.local.nself.org
- 📦 Storage: https://storage.local.nself.org
- 📊 And more...

*Tip:* These URLs are also printed after `nself build` and `nself start` so they're easy to copy.

## 📧 Email Configuration

### Development (Zero Config)
Email works out of the box with MailPit - all emails are captured locally:
- 📧 View emails: https://mail.local.nself.org
- 🔧 No setup required
- 📨 Perfect for testing auth flows

### Production (2-Minute Setup)
```bash
nself email setup
```

Choose from 16+ providers:
- **SendGrid** - 100 emails/day free
- **AWS SES** - $0.10 per 1000 emails  
- **Mailgun** - First 1000 emails free
- **Postmark** - Transactional email specialist
- **Gmail** - Use your personal/workspace account
- **Postfix** - Full control self-hosted server
- And 10+ more!

The wizard guides you through everything. Example for SendGrid:
```bash
nself email configure sendgrid
# Add your API key to .env
nself build && nself restart
```

### Want to customize?

Edit `.env` to enable extras:
```bash
# Core settings at the top
ENV=dev                         # or 'prod' for production
PROJECT_NAME=myapp
BASE_DOMAIN=local.nself.org
DB_ENV_SEEDS=true               # Use dev/prod seed separation

# Enable all the goodies
REDIS_ENABLED=true              # Redis caching
NESTJS_ENABLED=true             # NestJS microservices
FUNCTIONS_ENABLED=true          # Serverless functions
POSTGRES_EXTENSIONS=timescaledb,postgis,pgvector  # DB superpowers
```

Then rebuild and restart:
```bash
nself build && nself restart
```

## 💪 What You Get vs Manual Setup

| Manual Nhost Self-hosting | With nself |
|--------------------------|------------|
| Hours of configuration | 5 minutes total |
| Multiple config files | Single `.env` |
| Complex networking setup | Automatic service discovery |
| Manual SSL certificates | Automatic HTTPS everywhere |
| Separate service installs | One command, all services |
| Production passwords? 🤷 | `nself prod` generates secure ones |
| Hope it works | Battle-tested configurations |

## 📚 Commands

### Complete Command Tree

```
nself (34 commands)
├── 🚀 Core Commands
│   ├── init          Initialize a new project
│   ├── build         Build project structure and Docker images
│   ├── start         Start all services
│   ├── stop          Stop all services
│   ├── restart       Restart all or specific services
│   ├── status        Show service status and health
│   ├── logs          View service logs
│   └── clean         Clean up Docker resources
│
├── 📊 Database & Backup
│   ├── db            Database operations
│   └── backup        Comprehensive backup system
│
├── 🔧 Configuration
│   ├── validate      Validate configuration files
│   ├── ssl           SSL certificate management
│   ├── trust         Install SSL certificates locally
│   ├── email         Email service configuration
│   ├── prod          Configure for production
│   └── urls          Show service URLs
│
├── 🎯 Admin & Monitoring
│   ├── admin         Admin UI management (localhost:3100)
│   ├── doctor        System diagnostics and fixes
│   ├── monitor       Real-time monitoring
│   ├── metrics       Metrics collection
│   └── mlflow        ML experiment tracking
│
├── 🚀 Deployment & Scaling
│   ├── deploy        Deploy to remote servers
│   ├── scale         Scale services up/down
│   ├── rollback      Rollback to previous version
│   └── update        Update nself CLI
├── 🛠️ Development Tools
│   ├── exec          Execute commands in containers
│   ├── diff          Show configuration changes
│   ├── reset         Reset project to clean state
│   ├── scaffold      Generate new service from template
│   └── search        Search service management
│
└── 📝 Utility Commands
    ├── version       Show version information
    ├── help          Display help information
    ├── up            Alias for 'start'
    └── down          Alias for 'stop'
```

## 🎯 Admin Dashboard

### Web-Based Monitoring Interface
The new admin dashboard provides complete visibility and control over your nself stack:

- **Service Health Monitoring**: Real-time status of all containers
- **Docker Management**: Start, stop, restart containers from UI
- **Database Query Interface**: Execute SQL queries directly
- **Log Viewer**: Filter and search through service logs
- **Backup Management**: Create and restore backups via UI
- **Configuration Editor**: Modify settings without SSH

### Quick Setup
```bash
# Enable admin UI
nself admin enable

# Set password
nself admin password mypassword

# Open in browser (localhost:3100)
nself admin open
```

## 📚 Documentation

- **[Commands Reference](docs/COMMANDS.md)** - All 34 available commands
- **[Release Notes](docs/RELEASES.md)** - Latest features and fixes
- **[Roadmap](docs/ROADMAP.md)** - Development roadmap and upcoming features
- **[Architecture](docs/ARCHITECTURE.md)** - System architecture and design
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Changelog](docs/CHANGELOG.md)** - Version history
- **[All Releases](docs/RELEASES.md)** - Complete release history

### Quick Reference

### Email Commands
| Command | Description |
|---------|-------------|
| `nself email setup` | Interactive email setup wizard |
| `nself email list` | Show all 16+ supported email providers |
| `nself email configure <provider>` | Configure specific email provider |
| `nself email validate` | Check email configuration |
| `nself email test [email]` | Send a test email |
| `nself email docs <provider>` | Show provider setup guide |

## 🌐 Default Service URLs

When using the default `local.nself.org` domain:

- **GraphQL API**: https://api.local.nself.org
- **Authentication**: https://auth.local.nself.org
- **Storage**: https://storage.local.nself.org
- **Storage Console**: https://storage-console.local.nself.org
- **Functions** (if enabled): https://functions.local.nself.org
- **Email** (development): https://mail.local.nself.org - MailPit email viewer
- **Admin UI**: http://localhost:3100 - Admin dashboard
- **Dashboard** (if enabled): https://dashboard.local.nself.org

All `*.nself.org` domains resolve to `127.0.0.1` for local development.

## 💡 Hello World Example

The included hello world example shows all services working together:

```bash
# Enable all services in .env
SERVICES_ENABLED=true
REDIS_ENABLED=true
NESTJS_ENABLED=true
NESTJS_SERVICES=weather-actions
BULLMQ_ENABLED=true
BULLMQ_WORKERS=weather-processor,currency-processor
GOLANG_ENABLED=true
GOLANG_SERVICES=currency-fetcher
PYTHON_ENABLED=true
PYTHON_SERVICES=data-analyzer
```

**Architecture:**
- **NestJS**: Hasura actions for weather API integration
- **BullMQ**: Background processing of weather/currency data
- **GoLang**: High-performance currency rate fetching
- **Python**: ML predictions on time-series data

All services communicate through:
- Shared PostgreSQL database (with TimescaleDB)
- Redis for queuing and caching
- Direct HTTP calls via Docker network

## 🔧 Core Services

### Required Services
- **PostgreSQL**: Primary database with optional extensions
- **Hasura GraphQL**: Instant GraphQL API for your database
- **Hasura Auth**: JWT-based authentication service
- **MinIO**: S3-compatible object storage
- **Nginx**: Reverse proxy with SSL termination

### Optional Services
- **Redis**: In-memory caching and queue management
- **Nhost Functions**: Serverless functions support
- **Nhost Dashboard**: Admin interface for managing your backend
- **MailHog**: Email testing for development
- **NestJS Run Service**: For constantly running microservices

### Backend Services Framework
- **NestJS Services**: TypeScript/JavaScript microservices for Hasura actions
- **BullMQ Workers**: Queue workers for background job processing
- **GoLang Services**: High-performance microservices
- **Python Services**: ML/AI and data analysis services

## 🔐 SSL/TLS Configuration

nself provides bulletproof SSL with green locks in browsers - no warnings!

### Two Domain Options (Both Work Perfectly)

1. **`*.localhost`** - Works offline, no DNS needed
2. **`*.local.nself.org`** - Our loopback domain (resolves to 127.0.0.1)

### Automatic Certificate Generation

```bash
nself build    # Automatically generates SSL certificates
nself trust    # Install root CA for green locks (one-time)
```

That's it! Your browser will show green locks for:
- https://localhost, https://api.localhost, etc.
- https://local.nself.org, https://api.local.nself.org, etc.

### Advanced: Public Wildcard Certificates

For teams or CI/CD, get globally-trusted certificates (no `nself trust` needed):

```bash
# Add to .env
DNS_PROVIDER=cloudflare        # or route53, digitalocean
DNS_API_TOKEN=your_api_token

# Generate public wildcard
nself ssl bootstrap
```

Supported DNS providers:
- Cloudflare (recommended)
- AWS Route53
- DigitalOcean
- And more via acme.sh

### SSL Commands

| Command | Description |
|---------|-------------|
| `nself ssl bootstrap` | Generate SSL certificates |
| `nself ssl renew` | Renew public certificates |
| `nself ssl status` | Check certificate status |
| `nself trust` | Install root CA to system |
| `nself trust status` | Check trust status |

## 💾 Backup & Restore

### Comprehensive Backup System

nself includes enterprise-grade backup capabilities with cloud storage support:

```bash
# Create backups
nself backup create              # Full backup (database, config, volumes)
nself backup create database     # Database only
nself backup create config       # Configuration only

# Restore from backup
nself backup restore backup_20240115_143022.tar.gz

# List all backups
nself backup list
```

### Cloud Storage Support

Configure automatic cloud uploads for offsite backup:

```bash
# Interactive cloud setup wizard
nself backup cloud setup

# Supported providers:
# - Amazon S3 / MinIO
# - Dropbox
# - Google Drive
# - OneDrive
# - 40+ providers via rclone (Box, MEGA, pCloud, etc.)

# Test cloud connection
nself backup cloud test

# View cloud configuration
nself backup cloud status
```

### Advanced Retention Policies

Intelligently manage backup storage with multiple retention strategies:

```bash
# Simple age-based cleanup (default)
nself backup prune age 30        # Remove backups older than 30 days

# Grandfather-Father-Son policy
nself backup prune gfs           # Keep 7 daily, 4 weekly, 12 monthly

# Smart retention (recommended)
nself backup prune smart         # Intelligent retention based on age

# Cloud backup cleanup
nself backup prune cloud 30      # Prune cloud backups
```

### Automated Backups

Schedule automatic backups with built-in cron integration:

```bash
# Schedule options
nself backup schedule hourly
nself backup schedule daily      # Recommended for production
nself backup schedule weekly
nself backup schedule monthly

# View schedule
crontab -l
```

### Backup Configuration

Environment variables for backup customization:

```bash
# Local storage
BACKUP_DIR=./backups             # Backup directory
BACKUP_RETENTION_DAYS=30         # Default retention
BACKUP_RETENTION_MIN=3           # Minimum backups to keep

# Cloud provider selection
BACKUP_CLOUD_PROVIDER=s3         # s3, dropbox, gdrive, onedrive, rclone

# Provider-specific settings
S3_BUCKET=my-backups
DROPBOX_TOKEN=xxx
GDRIVE_FOLDER_ID=xxx
RCLONE_REMOTE=myremote
```

### What Gets Backed Up

**Full Backup includes:**
- PostgreSQL databases (complete dump)
- All environment files (.env.dev, .env.staging, .env.prod, .env.secrets, .env)
- Docker-compose configurations
- Docker volumes (all project data)
- SSL certificates
- Hasura metadata
- Nginx configurations

## 🚀 Production Deployment

### Using nself prod Command

The `nself prod` command automatically generates secure passwords for production:

```bash
# 1. Generate production configuration with secure passwords
nself prod

# This creates:
# - .env.prod-template (ready-to-use production config)
# - .env.prod-secrets (backup of generated passwords)

# 2. Edit .env.prod-template to set your domain and email

# 3. Deploy to production
cp .env.prod-template .env
nself start
```

Environment files are loaded in priority order (highest priority last):
- `.env.dev` - Team defaults (always loaded)
- `.env.staging` - Staging environment (if ENV=staging)
- `.env.prod` - Production environment (if ENV=prod)
- `.env.secrets` - Production secrets (if ENV=prod)
- `.env` - Local overrides (highest priority)

### Production Checklist
1. Set `ENV=prod` (automatically configures security settings)
2. Use strong passwords (12+ characters, auto-generated by `nself prod`)
3. Configure your custom domain
4. Enable Let's Encrypt SSL
5. Set up automated backups
6. Configure monitoring alerts

## 📁 Project Structure

After running `nself build`:

```
my-backend/
├── .env.dev               # Team defaults
├── .env.staging           # Staging environment (optional)
├── .env.prod              # Production environment (optional)
├── .env.secrets           # Production secrets (optional)
├── .env                   # Local configuration (highest priority)
├── docker-compose.yml      # Generated Docker Compose file
├── nginx/                  # Nginx configuration
│   ├── nginx.conf
│   ├── conf.d/            # Service routing
│   └── ssl/               # SSL certificates
├── postgres/              # Database initialization
│   └── init/
├── hasura/                # GraphQL configuration
│   ├── metadata/
│   └── migrations/
├── functions/             # Optional serverless functions
└── services/              # Backend services (if enabled)
    ├── nest/              # NestJS microservices
    ├── bullmq/            # Queue workers
    ├── go/                # GoLang services
    └── py/                # Python services
```

## 🗄️ Database Management

nself includes comprehensive database tools for schema management, migrations, and team collaboration.

### For Lead Developers
```bash
# Design your schema
nano schema.dbml

# Generate migrations from schema
nself db run

# Test migrations locally
nself db migrate:up

# Commit to Git
git add schema.dbml hasura/migrations/
git commit -m "Add new tables"
git push
```

### For All Developers
```bash
# Pull latest code
git pull

# Start services
nself start

# If you see "DATABASE MIGRATIONS PENDING" warning:
nself db update  # Safely apply migrations with confirmation
```

### Database Commands
- `nself db` - Show all database commands
- `nself db run` - Generate migrations from schema.dbml
- `nself db update` - Safely apply pending migrations and seeds
- `nself db seed` - Apply seed data (dev or prod based on ENV)
- `nself db status` - Check database state
- `nself db revert` - Restore from backup
- `nself db sync` - Pull schema from dbdiagram.io

### Database Seeding Strategy

When `DB_ENV_SEEDS=true` (recommended - follows Hasura/PostgreSQL standards):
- `seeds/common/` - Shared data across all environments
- `seeds/development/` - Mock/test data (when ENV=dev)
- `seeds/staging/` - Staging data (when ENV=staging) 
- `seeds/production/` - Minimal production data (when ENV=prod)

When `DB_ENV_SEEDS=false` (no environment branching):
- `seeds/default/` - Single seed directory for all environments

See [DBTOOLS.md](DBTOOLS.md) for complete database documentation.

## 🔄 Updating Services

To update service configurations:

1. Edit `.env`
2. Run `nself build` to regenerate configurations
3. Run `nself start` to apply changes

## 🐛 Troubleshooting

### Common Issues

#### Build command hangs?
```bash
# Build includes 5-second timeout for validation
nself build --force  # Force rebuild if stuck
```

#### Services not starting?
```bash
# Run diagnostics first
nself doctor

# Check service logs
nself logs [service-name]

# Check service status
nself status
```

#### Auth service unhealthy?
Known issue: Auth health check reports unhealthy but service works (port 4001 vs 4000 mismatch).

#### Port conflicts?
Edit the port numbers in `.env` and rebuild.

#### SSL certificate warnings?
Run `nself trust` to install the root CA and get green locks in your browser. No more warnings!

#### Email test not working?
```bash
# SMTP testing uses swaks Docker container
nself email test recipient@example.com
```

## 🔄 Version History

### v0.3.9 (Current - Stable Release)
- ✅ Admin UI with web-based monitoring dashboard
- ✅ Fixed critical bugs (status, stop, exec commands)  
- ✅ SMTP email testing implementation
- ✅ 5-second timeout for build validation
- ✅ All 33 commands fully functional
- ⚠️ Known issue: Auth health check false negative

### v0.3.8 (Stable)
- Complete backup system with cloud support
- SSL certificate management
- Enterprise monitoring features

[Full Changelog](docs/CHANGELOG.md)

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

Source Available License - see [LICENSE](LICENSE) file for details.

## 🎯 Perfect For

- **Startups**: Get your backend up fast, scale when you need to
- **Agencies**: Standardized backend setup for all client projects  
- **Enterprises**: Self-hosted solution with full control
- **Side Projects**: Production-grade infrastructure without the complexity
- **Learning**: See how modern backends work under the hood

## 📄 License

nself is **free for personal use**. Commercial use requires a license.

- ✅ **Personal Projects**: Free forever
- ✅ **Learning & Education**: Free forever  
- ✅ **Open Source Projects**: Free forever
- 💼 **Commercial Use**: [Contact us for licensing](https://nself.org/commercial)

See [LICENSE](LICENSE) for full terms.

## 🔗 Links

- [nself.org](https://nself.org) - Official Website
- [Commercial Licensing](https://nself.org/commercial) - For business use
- [Nhost Documentation](https://docs.nhost.io) - Learn more about Nhost
- [Hasura Documentation](https://hasura.io/docs) - GraphQL engine docs
- [Report Issues](https://github.com/acamarata/nself/issues) - We'd love your feedback!

## v0.3.9 Admin UI

Enable the web-based administration interface:

```bash
nself admin enable     # Enable admin web interface
nself admin password   # Set admin password
nself admin open       # Open admin in browser
```

The admin UI provides:
- Real-time service monitoring
- Configuration file editing
- Log streaming and management
- Backup management interface
- Resource usage monitoring

## v0.3.9 Enterprise Search

Choose from 6 different search engines:

```bash
nself search enable    # Interactive engine selection
nself search setup     # Configure search settings
nself search test      # Test search functionality
```

**Available Engines:**
- PostgreSQL FTS (built-in)
- MeiliSearch (recommended)
- Typesense, Elasticsearch, OpenSearch, Sonic

## v0.3.9 SSH Deployment

Deploy to any VPS with one command:

```bash
nself deploy init      # Setup deployment config
nself deploy ssh       # Deploy to VPS server
nself deploy status    # Check deployment status
```

Supports DigitalOcean, Linode, Vultr, Hetzner, and any Ubuntu/Debian VPS.

---

Built with ❤️ for the self-hosting community by developers who were tired of complex setups


