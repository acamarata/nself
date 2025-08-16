# nself - Nhost self-hosted stack and more, in seconds!

[![Version](https://img.shields.io/badge/version-0.3.7-blue.svg)](https://github.com/acamarata/nself/releases)
[![License](https://img.shields.io/badge/license-Personal%20Free%20%7C%20Commercial-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey.svg)](https://github.com/acamarata/nself#-supported-platforms)
[![Docker](https://img.shields.io/badge/docker-required-blue.svg)](https://www.docker.com/get-started)
[![CI Status](https://github.com/acamarata/nself/actions/workflows/ci.yml/badge.svg)](https://github.com/acamarata/nself/actions)

Deploy a feature-complete backend infrastructure on your own servers with PostgreSQL, Hasura GraphQL, Redis, Auth, Storage, and optional microservices. Works seamlessly across local development, staging, and production with automated SSL, smart defaults, and production-ready configurations.

**Based on [Nhost.io](https://nhost.io) for self-hosting!** and expanded with more features. Copy the below command in Terminal to install and get up and running in seconds!

```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

> **🚀 v0.3.7 NEW**: Enterprise-ready with backup/restore, CI/CD pipeline, Homebrew/apt/rpm packages, enhanced doctor diagnostics, and production-grade logging! [See full release notes](docs/CHANGELOG.md#037---2025-08-15)

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
- **Single Config File**: One `.env.local` controls everything
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
wget https://github.com/acamarata/nself/releases/download/v0.3.7/nself_0.3.7_all.deb
sudo dpkg -i nself_0.3.7_all.deb
```

#### RHEL/CentOS/Fedora (.rpm)
```bash
wget https://github.com/acamarata/nself/releases/download/v0.3.7/nself-0.3.7-1.noarch.rpm
sudo rpm -i nself-0.3.7-1.noarch.rpm
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

# 2. Initialize with smart defaults
nself init

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
# Add your API key to .env.local
nself build && nself restart
```

### Want to customize?

Edit `.env.local` to enable extras:
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
| Multiple config files | Single `.env.local` |
| Complex networking setup | Automatic service discovery |
| Manual SSL certificates | Automatic HTTPS everywhere |
| Separate service installs | One command, all services |
| Production passwords? 🤷 | `nself prod` generates secure ones |
| Hope it works | Battle-tested configurations |

## 📚 Commands

### Core Commands
| Command | Description |
|---------|-------------|
| `nself init` | Initialize a new project with `.env.local` |
| `nself build` | Generate project structure from configuration |
| `nself start` | Start all services (--apply-changes, --dry-run) |
| `nself stop` | Stop all services |
| `nself restart` | Restart all services (down + up) |
| `nself diff` | Show configuration changes since last build |
| `nself reset` | Delete all data and return to initial state |
| `nself backup` | Create and manage backups (local/S3) |
| `nself trust` | Install SSL certificate (fixes browser warnings) |
| `nself ssl` | SSL certificate management (bootstrap, renew, status) |

### Management Commands
| Command | Description |
|---------|-------------|
| `nself prod` | Create production .env with secure passwords |
| `nself update` | Update nself to the latest version |
| `nself db` | Database tools (migrations, schema, backups) |
| `nself email` | Email provider setup and management |
| `nself doctor` | Run system diagnostics and health checks |
| `nself logs` | View and follow service logs with filtering |
| `nself status` | Show service status and health |
| `nself version` | Show current version |
| `nself help` | Display help information |

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
- **Dashboard** (if enabled): https://dashboard.local.nself.org

All `*.nself.org` domains resolve to `127.0.0.1` for local development.

## 💡 Hello World Example

The included hello world example shows all services working together:

```bash
# Enable all services in .env.local
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
# Add to .env.local
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

The system prioritizes `.env` over `.env.local` when both exist:
- `.env.local` - Base configuration (development defaults)
- `.env` - Production overrides (only changed values)

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
├── .env.local              # Your configuration
├── .env                   # Production overrides (optional)
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

1. Edit `.env.local`
2. Run `nself build` to regenerate configurations
3. Run `nself start` to apply changes

## 🐛 Troubleshooting

### Services not starting?
```bash
# Check service logs
docker compose logs [service-name]

# Check service status
docker compose ps
```

### Port conflicts?
Edit the port numbers in `.env.local` and rebuild.

### SSL certificate warnings?
Run `nself trust` to install the root CA and get green locks in your browser. No more warnings!

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

## Future Planned Commands

- nself doctor, status, logs, exec, resources
- nself scale, backup, config, ssl, network
- nself metrics, health, cleanup, optimize, tenant

---

Built with ❤️ for the self-hosting community by developers who were tired of complex setups
