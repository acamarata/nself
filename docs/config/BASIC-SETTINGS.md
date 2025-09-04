# Basic Settings Configuration

Core configuration that affects your entire nself stack.

## Table of Contents
- [Required Settings](#required-settings)
- [Optional Settings](#optional-settings)
- [Environment Configuration](#environment-configuration)
- [Networking](#networking)
- [Security](#security)
- [Resource Limits](#resource-limits)
- [Examples](#examples)

## Required Settings

These MUST be set in your `.env.local` file:

```bash
# Project identifier (alphanumeric, hyphens allowed)
PROJECT_NAME=myproject

# Base domain for services
BASE_DOMAIN=local.nself.org

# Environment (development, staging, production)
ENV=development
```

## Optional Settings

### Project Configuration

```bash
# Project display name (can contain spaces)
PROJECT_DISPLAY_NAME="My Amazing Project"
# Default: ${PROJECT_NAME}

# Project description
PROJECT_DESCRIPTION="A comprehensive backend platform"
# Default: "nself project"

# Project version
PROJECT_VERSION="1.0.0"
# Default: "0.1.0"

# Organization name
ORG_NAME="My Company"
# Default: ${PROJECT_NAME}

# Support email
SUPPORT_EMAIL="support@example.com"
# Default: "admin@${BASE_DOMAIN}"
```

### Environment Configuration

```bash
# Environment type
ENV=development  # or: staging, production, test
# Default: development

# Debug mode (verbose logging)
DEBUG=false
# Default: true (dev), false (prod)

# Log level
LOG_LEVEL=info  # debug, info, warn, error
# Default: info

# Timezone
TZ=UTC  # or: America/New_York, Europe/London, etc.
# Default: UTC
```

### Networking

```bash
# Docker network settings
NETWORK_NAME=${PROJECT_NAME}_default
# Default: ${PROJECT_NAME}_default

# Network driver
NETWORK_DRIVER=bridge  # or: overlay, host, macvlan
# Default: bridge

# Network subnet (if custom network needed)
NETWORK_SUBNET=172.20.0.0/16
# Default: Docker assigns

# External network (use existing network)
EXTERNAL_NETWORK=false
# Default: false

# DNS servers (comma-separated)
DNS_SERVERS=8.8.8.8,8.8.4.4
# Default: Docker's default
```

### Ports Configuration

```bash
# HTTP port
HTTP_PORT=80
# Default: 80

# HTTPS port
HTTPS_PORT=443
# Default: 443

# Port range for services
SERVICE_PORT_START=3000
SERVICE_PORT_END=3999
# Default: 3000-3999

# Port range for user services
USER_SERVICE_PORT_START=8000
USER_SERVICE_PORT_END=8999
# Default: 8000-8999
```

### SSL Configuration

```bash
# SSL enabled
SSL_ENABLED=true
# Default: true

# SSL provider (mkcert, letsencrypt, custom)
SSL_PROVIDER=mkcert
# Default: mkcert (dev), letsencrypt (prod)

# Let's Encrypt email (production only)
LETSENCRYPT_EMAIL=admin@example.com
# Required for production SSL

# SSL certificate paths (custom provider)
SSL_CERT_PATH=/path/to/cert.pem
SSL_KEY_PATH=/path/to/key.pem
# Required if SSL_PROVIDER=custom

# Force HTTPS redirect
FORCE_HTTPS=true
# Default: false (dev), true (prod)

# HSTS (HTTP Strict Transport Security)
HSTS_ENABLED=true
HSTS_MAX_AGE=31536000
# Default: true (prod only)
```

### Security Settings

```bash
# Admin secret (for Hasura and other services)
ADMIN_SECRET=$(openssl rand -hex 32)
# Default: auto-generated

# JWT secret
JWT_SECRET=$(openssl rand -hex 64)
# Default: auto-generated

# Encryption key
ENCRYPTION_KEY=$(openssl rand -hex 32)
# Default: auto-generated

# API keys
API_KEY=$(openssl rand -hex 32)
# Default: auto-generated

# Session secret
SESSION_SECRET=$(openssl rand -hex 32)
# Default: auto-generated

# Cookie secret
COOKIE_SECRET=$(openssl rand -hex 32)
# Default: auto-generated

# Rate limiting (requests per minute)
GLOBAL_RATE_LIMIT=1000
# Default: 1000

# IP whitelist (comma-separated)
IP_WHITELIST=""
# Default: empty (allow all)

# IP blacklist (comma-separated)
IP_BLACKLIST=""
# Default: empty

# CORS origins (comma-separated)
CORS_ORIGINS="http://localhost:3000,https://${BASE_DOMAIN}"
# Default: "*" (dev), specific origins (prod)
```

### Resource Limits

```bash
# Global memory limit
GLOBAL_MEMORY_LIMIT=8G
# Default: no limit

# Global CPU limit
GLOBAL_CPU_LIMIT=4
# Default: no limit

# Disk quota
DISK_QUOTA=100G
# Default: no limit

# Max file upload size
MAX_UPLOAD_SIZE=100M
# Default: 100M

# Request timeout (seconds)
REQUEST_TIMEOUT=30
# Default: 30

# Idle timeout (seconds)
IDLE_TIMEOUT=600
# Default: 600

# Max connections
MAX_CONNECTIONS=1000
# Default: 1000
```

### Backup Configuration

```bash
# Backup enabled
BACKUP_ENABLED=true
# Default: true

# Backup schedule (cron format)
BACKUP_SCHEDULE="0 2 * * *"  # Daily at 2 AM
# Default: "0 2 * * *"

# Backup retention days
BACKUP_RETENTION_DAYS=30
# Default: 30

# Backup location
BACKUP_PATH=/backups
# Default: ./_backup

# S3 backup enabled
S3_BACKUP_ENABLED=false
# Default: false

# S3 backup bucket
S3_BACKUP_BUCKET=myproject-backups
# Required if S3_BACKUP_ENABLED=true

# S3 endpoint (for non-AWS S3)
S3_BACKUP_ENDPOINT=https://s3.amazonaws.com
# Default: AWS S3
```

### Monitoring

```bash
# Health check interval (seconds)
HEALTH_CHECK_INTERVAL=30
# Default: 30

# Health check timeout (seconds)
HEALTH_CHECK_TIMEOUT=5
# Default: 5

# Health check retries
HEALTH_CHECK_RETRIES=3
# Default: 3

# Metrics enabled
METRICS_ENABLED=true
# Default: false

# Metrics port
METRICS_PORT=9090
# Default: 9090

# Tracing enabled
TRACING_ENABLED=false
# Default: false

# Tracing endpoint
TRACING_ENDPOINT=http://jaeger:14268
# Required if TRACING_ENABLED=true
```

### Development Settings

```bash
# Hot reload enabled
HOT_RELOAD=true
# Default: true (dev), false (prod)

# Source maps enabled
SOURCE_MAPS=true
# Default: true (dev), false (prod)

# Mock data enabled
MOCK_DATA=false
# Default: false

# Seed database on start
SEED_DATABASE=false
# Default: false

# Development tools enabled
DEV_TOOLS=true
# Default: true (dev), false (prod)
```

## Complete Example

### Development Environment (.env.local)

```bash
# ============================================
# BASIC SETTINGS - DEVELOPMENT
# ============================================

# Required
PROJECT_NAME=myapp
BASE_DOMAIN=local.nself.org
ENV=development

# Project
PROJECT_DISPLAY_NAME="My Application"
PROJECT_DESCRIPTION="Next-generation platform"
PROJECT_VERSION="0.1.0"
ORG_NAME="My Startup"
SUPPORT_EMAIL="support@myapp.com"

# Environment
DEBUG=true
LOG_LEVEL=debug
TZ=America/New_York

# Networking
HTTP_PORT=80
HTTPS_PORT=443

# SSL (local development)
SSL_ENABLED=true
SSL_PROVIDER=mkcert
FORCE_HTTPS=false

# Security (development - weak for convenience)
ADMIN_SECRET=devsecret123
JWT_SECRET=devjwtsecret456
GLOBAL_RATE_LIMIT=10000
CORS_ORIGINS="*"

# Resources (development - limited)
GLOBAL_MEMORY_LIMIT=4G
GLOBAL_CPU_LIMIT=2
MAX_UPLOAD_SIZE=10M

# Backup
BACKUP_ENABLED=false

# Monitoring
METRICS_ENABLED=true
HEALTH_CHECK_INTERVAL=60

# Development
HOT_RELOAD=true
SOURCE_MAPS=true
DEV_TOOLS=true
SEED_DATABASE=true
```

### Production Environment (.env.prod)

```bash
# ============================================
# BASIC SETTINGS - PRODUCTION
# ============================================

# Required
PROJECT_NAME=myapp
BASE_DOMAIN=myapp.com
ENV=production

# Project
PROJECT_DISPLAY_NAME="My Application"
PROJECT_DESCRIPTION="Enterprise Platform"
PROJECT_VERSION="1.0.0"
ORG_NAME="My Company Inc."
SUPPORT_EMAIL="support@myapp.com"

# Environment
DEBUG=false
LOG_LEVEL=warn
TZ=UTC

# Networking
HTTP_PORT=80
HTTPS_PORT=443
DNS_SERVERS=1.1.1.1,1.0.0.1

# SSL (production with Let's Encrypt)
SSL_ENABLED=true
SSL_PROVIDER=letsencrypt
LETSENCRYPT_EMAIL=admin@myapp.com
FORCE_HTTPS=true
HSTS_ENABLED=true
HSTS_MAX_AGE=31536000

# Security (production - strong)
ADMIN_SECRET=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 64)
ENCRYPTION_KEY=$(openssl rand -hex 32)
API_KEY=$(openssl rand -hex 32)
SESSION_SECRET=$(openssl rand -hex 32)
COOKIE_SECRET=$(openssl rand -hex 32)
GLOBAL_RATE_LIMIT=100
CORS_ORIGINS="https://myapp.com,https://www.myapp.com"

# Resources (production - full)
GLOBAL_MEMORY_LIMIT=32G
GLOBAL_CPU_LIMIT=16
MAX_UPLOAD_SIZE=1G
REQUEST_TIMEOUT=60
MAX_CONNECTIONS=10000

# Backup
BACKUP_ENABLED=true
BACKUP_SCHEDULE="0 2 * * *"
BACKUP_RETENTION_DAYS=90
S3_BACKUP_ENABLED=true
S3_BACKUP_BUCKET=myapp-backups

# Monitoring
METRICS_ENABLED=true
TRACING_ENABLED=true
TRACING_ENDPOINT=http://jaeger:14268
HEALTH_CHECK_INTERVAL=10
HEALTH_CHECK_TIMEOUT=3
HEALTH_CHECK_RETRIES=5

# Production
HOT_RELOAD=false
SOURCE_MAPS=false
DEV_TOOLS=false
SEED_DATABASE=false
```

## Environment-Specific Overrides

You can create environment-specific files that override base settings:

- `.env` - Base configuration (all environments)
- `.env.dev` - Development overrides
- `.env.staging` - Staging overrides
- `.env.prod` - Production overrides
- `.env.local` - Local overrides (highest priority, git-ignored)

## Validation

Run validation to check your configuration:

```bash
nself validate
```

This will check:
- Required variables are set
- Values are in valid format
- Ports are available
- Domains are resolvable
- Certificates are valid
- Resources are sufficient

## Next Steps

- [Required Services](./REQUIRED-SERVICES.md) - Configure PostgreSQL, Hasura, Auth, Nginx
- [Optional Services](./OPTIONAL-SERVICES.md) - Enable additional services
- [User Services](./USER-SERVICES.md) - Add custom backend services
- [Frontend Apps](./FRONTEND-APPS.md) - Configure frontend applications