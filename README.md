# nself - Nhost self-hosted stack and more, in seconds!

Deploy a feature-complete backend infrastructure on your own servers with PostgreSQL, Hasura GraphQL, Redis, Auth, Storage, and optional microservices. Works seamlessly across local development, staging, and production with automated SSL, smart defaults, and production-ready configurations.

**Based on [Nhost.io](https://nhost.io) for self-hosting!** and expanded with more features. Copy the below command in Terminal to install and get up and running in seconds!

```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

nself is *the* CLI for Nhost self-hosted deployments - with extras and an opinionated setup that makes everything smooth. From zero to production-ready backend in under 5 minutes. Just edit an env file with your preferences and build!

## 🚀 Why nself?

### ⚡ Lightning Fast Setup
- **Under 5 minutes** from zero to running backend
- One command installation, initialization, and deployment
- Smart defaults that just work™

### 🎯 Complete Feature Set
- **Full Nhost Stack**: PostgreSQL, Hasura GraphQL, Auth, Storage, Functions
- **Plus Extras**: Redis, TimescaleDB, PostGIS, pgvector extensions
- **Microservices Ready**: Built-in support for NestJS, BullMQ, GoLang, and Python services
- **Production SSL**: Automatic HTTPS with Let's Encrypt or self-signed certs

### 🛠️ Developer Experience
- **Single Config File**: One `.env.local` controls everything
- **Hot Reload**: Changes apply instantly in development
- **Multi-Environment**: Same setup works locally, staging, and production
- **No Lock-in**: Standard Docker Compose under the hood

### 🔐 Production Ready
- **Security First**: Automatic secure password generation with `nself prod`
- **Battle Tested**: Based on proven Nhost.io infrastructure
- **Scale Ready**: From hobby projects to enterprise deployments
- **Zero Downtime**: Rolling updates and health checks built-in

## 📋 Prerequisites

- Linux, macOS, or Windows with WSL
- Docker and Docker Compose (installer will help install these)
- `curl` (for installation)

## 🔧 Installation

Install nself with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
```

The installer will:
- Check and install Docker/Docker Compose if needed
- Download nself CLI to `~/.nself/bin`
- Add nself to your PATH
- Create a global `nself` command

## 🏁 Quick Start - 3 Commands to Backend Bliss

```bash
# 1. Create and enter project directory
mkdir my-awesome-backend && cd my-awesome-backend

# 2. Initialize with smart defaults
nself init

# 3. Build and launch everything
nself build && nself up
# URLs for enabled services will be shown in the output
```

**That's it!** Your complete backend is now running at:
- 🚀 GraphQL API: https://api.local.nself.org
- 🔐 Auth Service: https://auth.local.nself.org
- 📦 Storage: https://storage.local.nself.org
- 📊 And more...

*Tip:* These URLs are also printed after `nself build` and `nself up` so they're easy to copy.

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

| Command | Description |
|---------|-------------|
| `nself init` | Initialize a new project with `.env.local` |
| `nself build` | Generate project structure from configuration |
| `nself up` | Start all services (shows migration warnings) |
| `nself down` | Stop all services |
| `nself restart` | Restart all services (down + up) |
| `nself prod` | Create production .env from .env.local |
| `nself reset` | Delete all data and return to initial state |
| `nself update` | Update nself to the latest version |
| `nself db` | Database tools (migrations, schema, backups) |
| `nself version` | Show current version |
| `nself help` | Display help information |

## 🌐 Default Service URLs

When using the default `local.nself.org` domain:

- **GraphQL API**: https://api.local.nself.org
- **Authentication**: https://auth.local.nself.org
- **Storage**: https://storage.local.nself.org
- **Storage Console**: https://storage-console.local.nself.org
- **Functions** (if enabled): https://functions.local.nself.org
- **MailHog** (development): https://mailhog.local.nself.org

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

nself supports three SSL modes:

1. **Local Development** (default): Pre-generated certificates for `*.nself.org`
2. **Let's Encrypt**: Automatic certificates for production domains
3. **Custom**: Bring your own certificates

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
nself up
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
nself up

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
3. Run `nself up` to apply changes

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
For local development with `*.nself.org`, accept the self-signed certificate in your browser.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🎯 Perfect For

- **Startups**: Get your backend up fast, scale when you need to
- **Agencies**: Standardized backend setup for all client projects  
- **Enterprises**: Self-hosted solution with full control
- **Side Projects**: Production-grade infrastructure without the complexity
- **Learning**: See how modern backends work under the hood

## 🔗 Links

- [nself.org](https://nself.org) - Official Website
- [Nhost Documentation](https://docs.nhost.io) - Learn more about Nhost
- [Hasura Documentation](https://hasura.io/docs) - GraphQL engine docs
- [Report Issues](https://github.com/acamarata/nself/issues) - We'd love your feedback!

## Future Planned Commands

- nself doctor, status, logs, exec, resources
- nself scale, backup, config, ssl, network
- nself metrics, health, cleanup, optimize, tenant

---

Built with ❤️ for the self-hosting community by developers who were tired of complex setups
