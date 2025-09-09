# Frequently Asked Questions (FAQ)

## 📚 Table of Contents

- [General Questions](#general-questions)
- [Installation & Setup](#installation--setup)
- [Configuration](#configuration)
- [Services](#services)
- [Troubleshooting](#troubleshooting)
- [Production & Deployment](#production--deployment)
- [Security](#security)
- [Performance](#performance)
- [Licensing & Support](#licensing--support)

---

## General Questions

### What is nself?

nself is a comprehensive self-hosted backend stack that provides all the features of Backend-as-a-Service (BaaS) platforms like Nhost.io, Supabase, and Firebase, but runs entirely on your own infrastructure using Docker Compose.

### How is nself different from Supabase or Nhost?

| Feature | nself | Supabase | Nhost |
|---------|-------|----------|-------|
| **Self-hosted** | ✅ Full control | ⚠️ Limited | ⚠️ Limited |
| **No vendor lock-in** | ✅ Yes | ❌ No | ❌ No |
| **Customizable** | ✅ Fully | ⚠️ Partial | ⚠️ Partial |
| **Cost** | ✅ Infrastructure only | 💰 Subscription | 💰 Subscription |
| **Data ownership** | ✅ 100% yours | ⚠️ Shared | ⚠️ Shared |
| **Microservices** | ✅ Unlimited | ❌ Limited | ❌ Limited |

### What services are included?

**Core Services:**
- PostgreSQL 16 with 60+ extensions
- Hasura GraphQL Engine
- Authentication service (JWT-based)
- S3-compatible storage (MinIO)
- Email service (16+ providers)
- Admin UI dashboard

**Optional Services:**
- Redis cache
- Monitoring stack (Prometheus, Grafana, Loki)
- Custom microservices (40+ templates)

### What are the system requirements?

**Minimum:**
- 2 CPU cores
- 4GB RAM
- 10GB storage
- Docker & Docker Compose v2

**Recommended:**
- 4+ CPU cores
- 8GB+ RAM
- 20GB+ storage

---

## Installation & Setup

### How do I install nself?

```bash
# Quick install
git clone https://github.com/acamarata/nself.git
cd nself
chmod +x bin/nself

# Create project
mkdir my-app && cd my-app
nself init
nself build
nself start
```

See [Installation Guide](Installation) for detailed instructions.

### Do I need to install all services?

No! nself uses smart defaults. Only PostgreSQL and Nginx are required. You can enable/disable services in your `.env` file:

```bash
HASURA_ENABLED=true
AUTH_ENABLED=true
STORAGE_ENABLED=false
REDIS_ENABLED=false
```

### Can I use an existing PostgreSQL database?

Yes! Configure external database in `.env`:

```bash
POSTGRES_EXTERNAL=true
POSTGRES_HOST=your-db-host.com
POSTGRES_PORT=5432
POSTGRES_USER=your-user
POSTGRES_PASSWORD=your-password
POSTGRES_DB=your-database
```

### How do I add nself to PATH?

```bash
# Bash
echo 'export PATH="$PATH:/path/to/nself/bin"' >> ~/.bashrc
source ~/.bashrc

# Zsh (macOS)
echo 'export PATH="$PATH:/path/to/nself/bin"' >> ~/.zshrc
source ~/.zshrc
```

---

## Configuration

### Where is the configuration stored?

Configuration is stored in environment files in your project directory:
- `.env` - Main configuration (highest priority)
- `.env.dev` - Development defaults
- `.env.staging` - Staging overrides
- `.env.prod` - Production settings
- `.env.secrets` - Sensitive data (never commit!)

### How do environment files work?

Files are loaded in cascade order:
1. `.env.dev` (always loaded first)
2. `.env.staging` (if ENV=staging)
3. `.env.prod` (if ENV=prod)
4. `.env.secrets` (if ENV=prod)
5. `.env` (always loaded last, highest priority)

### How do I change ports?

Edit `.env` file:

```bash
# Service ports
POSTGRES_PORT=5433
HASURA_PORT=8081
AUTH_PORT=4001
STORAGE_PORT=4003
ADMIN_PORT=3101
```

### How do I enable SSL?

```bash
# Generate certificates
nself ssl

# Trust in browser
nself trust

# For production (Let's Encrypt)
nself ssl --production
```

### Can I use a custom domain?

Yes! Configure in `.env`:

```bash
BASE_DOMAIN=api.myapp.com
HASURA_SUBDOMAIN=graphql
AUTH_SUBDOMAIN=auth
STORAGE_SUBDOMAIN=storage
```

---

## Services

### How do I add microservices?

1. Enable services in `.env`:
```bash
SERVICES_ENABLED=true
NODEJS_SERVICES=api,workers
PYTHON_SERVICES=ml,analytics
```

2. Rebuild and restart:
```bash
nself build
nself restart
```

See [Service Templates](Service-Templates) for available templates.

### How do I scale services?

```bash
# Scale specific service
nself scale api=3

# Or use Docker Compose directly
docker compose up -d --scale api=3
```

### Can I use my own Docker images?

Yes! Create `docker-compose.custom.yml`:

```yaml
services:
  my-service:
    image: my-custom-image:latest
    environment:
      - API_KEY=${MY_API_KEY}
```

Then enable:
```bash
CUSTOM_COMPOSE_FILES=docker-compose.custom.yml
```

### How do I connect services?

Services communicate via Docker network. Use service names as hostnames:

```javascript
// In your Node.js service
const pgClient = new Client({
  host: 'postgres',  // Service name
  port: 5432,
  database: process.env.POSTGRES_DB
});

const hasuraEndpoint = 'http://hasura:8080/v1/graphql';
```

---

## Troubleshooting

### Services won't start

```bash
# Check Docker
nself doctor

# View logs
nself logs [service-name]

# Common fixes
docker system prune -a  # Clean Docker
AUTO_FIX=true nself build  # Auto-fix issues
```

### Port already in use

The auto-fix system will reassign ports automatically:

```bash
AUTO_FIX=true nself build
```

Or manually change in `.env`:
```bash
HASURA_PORT=8081  # Instead of 8080
```

### Auth service shows unhealthy

This is a known issue. The health check expects port 4000 but service runs on 4001. The service works correctly despite the health status.

### Database connection refused

```bash
# Check if PostgreSQL is running
nself status

# Check logs
nself logs postgres

# Restart PostgreSQL
nself restart postgres

# Reset database (WARNING: deletes data)
nself reset
```

### Can't access Admin UI

```bash
# Ensure enabled
grep NSELF_ADMIN_ENABLED .env

# Enable if not
nself admin enable

# Check status
nself admin status

# View URL
nself admin open
```

---

## Production & Deployment

### How do I deploy to production?

1. Generate production config:
```bash
nself prod
```

2. Review and edit `.env.prod`:
```bash
ENV=prod
BASE_DOMAIN=api.myapp.com
SSL_ENABLED=true
```

3. Deploy:
```bash
nself deploy production
```

See [Deployment Guide](Deployment) for details.

### How do I backup data?

```bash
# Create backup
nself backup create my-backup

# Schedule automatic backups
nself backup schedule --daily --retain 7

# Restore backup
nself backup restore my-backup
```

### Can I use Kubernetes?

Kubernetes support is planned for v0.4.0. Currently, you can:
- Export Docker Compose to Kubernetes using Kompose
- Use the Docker images directly in K8s manifests

### How do I monitor services?

Enable monitoring stack:

```bash
PROMETHEUS_ENABLED=true
GRAFANA_ENABLED=true
LOKI_ENABLED=true
```

Access Grafana at http://localhost:3000

### How do I handle secrets?

1. Never commit `.env` or `.env.secrets`
2. Use secret management tools:
```bash
# HashiCorp Vault
SECRETS_PROVIDER=vault
VAULT_ADDR=https://vault.myapp.com

# AWS Secrets Manager
SECRETS_PROVIDER=aws
AWS_REGION=us-east-1
```

---

## Security

### Is nself secure for production?

Yes, with proper configuration:
- ✅ Use strong passwords (auto-generated by default)
- ✅ Enable SSL/TLS
- ✅ Configure firewall rules
- ✅ Regular security updates
- ✅ Enable 2FA for admin access
- ✅ Rotate secrets regularly

### How do I secure the Admin UI?

```bash
# Strong password
nself admin password $(openssl rand -base64 32)

# IP whitelisting
NSELF_ADMIN_IP_WHITELIST=10.0.0.0/8

# SSL only
NSELF_ADMIN_SSL_ONLY=true
```

### What about GDPR compliance?

nself gives you full control over data:
- Data stays on your infrastructure
- You control backups and retention
- Built-in data export capabilities
- Audit logging available

### How do I enable audit logging?

```bash
AUDIT_ENABLED=true
AUDIT_LOG_PATH=/var/log/nself/audit.log
AUDIT_RETENTION_DAYS=90
```

---

## Performance

### How many concurrent users can nself handle?

Depends on your infrastructure. Typical performance:
- **Small** (2 CPU, 4GB RAM): ~100 concurrent users
- **Medium** (4 CPU, 8GB RAM): ~500 concurrent users
- **Large** (8 CPU, 16GB RAM): ~2000 concurrent users

Scale horizontally for more capacity.

### How do I optimize performance?

```bash
# Enable Redis cache
REDIS_ENABLED=true

# Increase PostgreSQL connections
POSTGRES_MAX_CONNECTIONS=200

# Enable connection pooling
HASURA_POOL_SIZE=50

# Use CDN for static assets
CDN_ENABLED=true
CDN_URL=https://cdn.myapp.com
```

### Can I use a CDN?

Yes! Configure in Nginx:

```nginx
location /static {
    proxy_pass https://cdn.myapp.com;
    proxy_cache_valid 200 1d;
}
```

---

## Licensing & Support

### Is nself free?

- ✅ **Free** for personal and non-commercial use
- 💰 **Paid license** for commercial use
- See [LICENSE](https://github.com/acamarata/nself/blob/main/LICENSE)

### How do I get commercial license?

Contact: license@nself.org

Includes:
- Commercial use rights
- Priority support
- Custom features
- SLA guarantees

### Where can I get help?

1. 📖 [Documentation](Home)
2. 🐛 [GitHub Issues](https://github.com/acamarata/nself/issues)
3. 💬 [GitHub Discussions](https://github.com/acamarata/nself/discussions)
4. 📧 Email: support@nself.org
5. 💼 Commercial support: enterprise@nself.org

### How can I contribute?

We welcome contributions!

1. Fork the repository
2. Create feature branch
3. Make changes
4. Submit pull request

See [Contributing Guide](Contributing) for details.

### What's on the roadmap?

See [Roadmap](Roadmap) for planned features:
- v0.4.0: Kubernetes support
- v0.5.0: Multi-tenant architecture
- v0.6.0: Enterprise features

---

## Quick Fixes

### Reset Everything
```bash
nself clean --all
nself init
nself build
nself start
```

### Update to Latest
```bash
cd /path/to/nself
git pull
nself update
```

### Emergency Stop
```bash
docker compose down
docker stop $(docker ps -q)
```

### View All Logs
```bash
nself logs > all-logs.txt
```

### Export Configuration
```bash
nself config > config-backup.json
```

---

**Still have questions?** [Create an issue](https://github.com/acamarata/nself/issues) or [join the discussion](https://github.com/acamarata/nself/discussions)!