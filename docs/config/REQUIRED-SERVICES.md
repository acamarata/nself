# Required Services Configuration

These are the core services that are always running in nself. While they have sensible defaults, you can customize every aspect.

## Table of Contents
- [PostgreSQL](#postgresql)
- [Hasura GraphQL](#hasura-graphql)
- [Authentication Service](#authentication-service)
- [Nginx Reverse Proxy](#nginx-reverse-proxy)

---

## PostgreSQL

The primary database with 60+ extensions available.

### Basic Configuration

```bash
# Enable/Disable (always true for required service)
POSTGRES_ENABLED=true
# Default: true (cannot be disabled)

# Version
POSTGRES_VERSION=16
# Default: 16

# Database name
POSTGRES_DB=${PROJECT_NAME}
# Default: ${PROJECT_NAME}

# Username
POSTGRES_USER=postgres
# Default: postgres

# Password (auto-generated if not set)
POSTGRES_PASSWORD=$(openssl rand -hex 16)
# Default: auto-generated secure password
```

### Connection Settings

```bash
# Port (internal)
POSTGRES_PORT=5432
# Default: 5432

# External port (for direct access)
POSTGRES_EXTERNAL_PORT=5432
# Default: 5432

# Host (internal Docker network)
POSTGRES_HOST=postgres
# Default: postgres

# Connection pool settings
POSTGRES_MAX_CONNECTIONS=200
# Default: 200

POSTGRES_SHARED_BUFFERS=256MB
# Default: 256MB

POSTGRES_EFFECTIVE_CACHE_SIZE=1GB
# Default: 1GB

POSTGRES_WORK_MEM=4MB
# Default: 4MB
```

### Extensions

```bash
# Core extensions (always enabled)
# - uuid-ossp (UUID generation)
# - pgcrypto (Cryptographic functions)
# - citext (Case-insensitive text)
# - btree_gist (GiST index support)

# Optional extensions
POSTGRES_EXTENSIONS_ENABLED=true
# Default: true

# TimescaleDB (time-series data)
TIMESCALEDB_ENABLED=true
# Default: false

# PostGIS (geospatial data)
POSTGIS_ENABLED=false
# Default: false

# pgvector (ML embeddings)
PGVECTOR_ENABLED=false
# Default: false

# pg_cron (scheduled jobs)
PG_CRON_ENABLED=false
# Default: false

# Full list of available extensions
POSTGRES_EXTENSIONS="uuid-ossp,pgcrypto,citext,btree_gist,hstore,pg_trgm,unaccent"
# Default: "uuid-ossp,pgcrypto,citext"
```

### Performance Tuning

```bash
# Memory settings
POSTGRES_SHARED_BUFFERS=1GB      # 25% of RAM
POSTGRES_EFFECTIVE_CACHE_SIZE=3GB # 75% of RAM
POSTGRES_MAINTENANCE_WORK_MEM=256MB
POSTGRES_WORK_MEM=10MB
POSTGRES_HUGE_PAGES=try

# Checkpoint settings
POSTGRES_CHECKPOINT_SEGMENTS=32
POSTGRES_CHECKPOINT_COMPLETION_TARGET=0.9
POSTGRES_WAL_BUFFERS=16MB

# Query tuning
POSTGRES_DEFAULT_STATISTICS_TARGET=100
POSTGRES_RANDOM_PAGE_COST=1.1
POSTGRES_EFFECTIVE_IO_CONCURRENCY=200
POSTGRES_MIN_WAL_SIZE=1GB
POSTGRES_MAX_WAL_SIZE=4GB

# Connection settings
POSTGRES_MAX_CONNECTIONS=200
POSTGRES_SUPERUSER_RESERVED_CONNECTIONS=3
POSTGRES_MAX_PREPARED_TRANSACTIONS=0
POSTGRES_MAX_LOCKS_PER_TRANSACTION=64

# Autovacuum settings
POSTGRES_AUTOVACUUM=on
POSTGRES_AUTOVACUUM_MAX_WORKERS=3
POSTGRES_AUTOVACUUM_NAPTIME=1min
POSTGRES_AUTOVACUUM_VACUUM_THRESHOLD=50
POSTGRES_AUTOVACUUM_ANALYZE_THRESHOLD=50
```

### Backup Settings

```bash
# Backup enabled
POSTGRES_BACKUP_ENABLED=true
# Default: true

# Backup schedule (cron format)
POSTGRES_BACKUP_SCHEDULE="0 2 * * *"
# Default: "0 2 * * *" (2 AM daily)

# Backup retention
POSTGRES_BACKUP_RETENTION_DAYS=30
# Default: 30

# WAL archiving
POSTGRES_ARCHIVE_MODE=on
POSTGRES_ARCHIVE_COMMAND='test ! -f /archive/%f && cp %p /archive/%f'
# Default: off
```

### Resource Limits

```bash
# Memory limit
POSTGRES_MEMORY_LIMIT=4G
# Default: no limit

# CPU limit
POSTGRES_CPU_LIMIT=2
# Default: no limit

# Storage limit
POSTGRES_STORAGE_LIMIT=100G
# Default: no limit
```

---

## Hasura GraphQL

Instant GraphQL API for your PostgreSQL database.

### Basic Configuration

```bash
# Enable/Disable (always true for required service)
HASURA_ENABLED=true
# Default: true

# Version
HASURA_VERSION=v2.44.0
# Default: v2.44.0

# Port (internal)
HASURA_PORT=8080
# Default: 8080

# External port
HASURA_EXTERNAL_PORT=8080
# Default: 8080

# Admin secret (required for production)
HASURA_GRAPHQL_ADMIN_SECRET=${ADMIN_SECRET}
# Default: ${ADMIN_SECRET}
```

### Features

```bash
# Console enabled (dev only)
HASURA_GRAPHQL_ENABLE_CONSOLE=true
# Default: true (dev), false (prod)

# Dev mode
HASURA_GRAPHQL_DEV_MODE=true
# Default: true (dev), false (prod)

# Telemetry
HASURA_GRAPHQL_ENABLE_TELEMETRY=false
# Default: false

# Allow list (GraphQL query allowlisting)
HASURA_GRAPHQL_ENABLE_ALLOWLIST=false
# Default: false (dev), true (prod)

# Schema introspection
HASURA_GRAPHQL_ENABLE_SCHEMA_INTROSPECTION=true
# Default: true (dev), false (prod)

# Remote schema permissions
HASURA_GRAPHQL_ENABLE_REMOTE_SCHEMA_PERMISSIONS=true
# Default: true
```

### Authentication

```bash
# JWT secret configuration
HASURA_GRAPHQL_JWT_SECRET='{
  "type": "HS256",
  "key": "${JWT_SECRET}"
}'
# Default: HS256 with ${JWT_SECRET}

# Unauthorized role
HASURA_GRAPHQL_UNAUTHORIZED_ROLE=public
# Default: public

# Admin role
HASURA_GRAPHQL_ADMIN_ROLE=admin
# Default: admin

# Auth webhook (alternative to JWT)
HASURA_GRAPHQL_AUTH_HOOK=http://auth:4000/webhook
# Default: not set (uses JWT)

# Webhook secret
HASURA_GRAPHQL_AUTH_HOOK_MODE=POST
# Default: GET
```

### Performance

```bash
# Connection pool settings
HASURA_GRAPHQL_PG_CONNECTIONS=50
# Default: 50

HASURA_GRAPHQL_PG_TIMEOUT=180
# Default: 180

HASURA_GRAPHQL_PG_CONN_LIFETIME=600
# Default: 600

HASURA_GRAPHQL_PG_POOL_TIMEOUT=360
# Default: 360

# Query limits
HASURA_GRAPHQL_QUERY_PLAN_CACHE_SIZE=4000
# Default: 4000

HASURA_GRAPHQL_MAX_CACHE_SIZE=1000
# Default: 1000

# Live queries
HASURA_GRAPHQL_LIVE_QUERIES_MULTIPLEXED_REFETCH_INTERVAL=1000
# Default: 1000 (ms)

HASURA_GRAPHQL_LIVE_QUERIES_MULTIPLEXED_BATCH_SIZE=100
# Default: 100

# Websocket settings
HASURA_GRAPHQL_WS_READ_COOKIE=false
# Default: false

HASURA_GRAPHQL_WEBSOCKET_KEEPALIVE=30
# Default: 30 (seconds)

HASURA_GRAPHQL_WEBSOCKET_CONNECTION_INIT_TIMEOUT=3
# Default: 3 (seconds)
```

### CORS Configuration

```bash
# CORS enabled
HASURA_GRAPHQL_CORS_DOMAIN="*"
# Default: "*" (dev), specific domains (prod)

# CORS credentials
HASURA_GRAPHQL_CORS_CREDENTIALS=true
# Default: true

# CORS max age
HASURA_GRAPHQL_CORS_MAX_AGE=86400
# Default: 86400
```

### Logging

```bash
# Log level
HASURA_GRAPHQL_LOG_LEVEL=info
# Default: info (options: debug, info, warn, error)

# Console log format
HASURA_GRAPHQL_CONSOLE_LOG_LEVEL=info
# Default: info

# Server log format
HASURA_GRAPHQL_SERVER_LOG_LEVEL=info
# Default: info

# Enable request logging
HASURA_GRAPHQL_ENABLE_REQUEST_LOG=true
# Default: true (dev), false (prod)
```

### Events & Actions

```bash
# Event triggers
HASURA_GRAPHQL_EVENTS_HTTP_POOL_SIZE=100
# Default: 100

HASURA_GRAPHQL_EVENTS_FETCH_INTERVAL=10
# Default: 10 (seconds)

HASURA_GRAPHQL_EVENTS_FETCH_BATCH_SIZE=100
# Default: 100

# Actions
HASURA_GRAPHQL_ACTION_TIMEOUT=30
# Default: 30 (seconds)

HASURA_GRAPHQL_ACTION_HANDLER_WEBHOOK_BASEURL=http://actions:3000
# Default: not set
```

### Resource Limits

```bash
# Memory limit
HASURA_MEMORY_LIMIT=2G
# Default: no limit

# CPU limit
HASURA_CPU_LIMIT=1
# Default: no limit
```

---

## Authentication Service

Nhost Auth service for JWT-based authentication.

### Basic Configuration

```bash
# Enable/Disable (always true for required service)
AUTH_ENABLED=true
# Default: true

# Version
AUTH_VERSION=0.36.0
# Default: 0.36.0

# Port (internal)
AUTH_PORT=4000
# Default: 4000

# External port
AUTH_EXTERNAL_PORT=4001
# Default: 4001

# Server URL
AUTH_SERVER_URL=http://localhost:4000
# Default: http://localhost:4000
```

### JWT Configuration

```bash
# JWT secret (must match Hasura)
AUTH_JWT_SECRET=${JWT_SECRET}
# Default: ${JWT_SECRET}

# JWT algorithm
AUTH_JWT_ALGORITHM=HS256
# Default: HS256

# JWT expires in
AUTH_ACCESS_TOKEN_EXPIRES_IN=900
# Default: 900 (15 minutes)

# Refresh token expires in
AUTH_REFRESH_TOKEN_EXPIRES_IN=2592000
# Default: 2592000 (30 days)
```

### Email Configuration

```bash
# SMTP enabled
AUTH_SMTP_ENABLED=true
# Default: false (dev), true (prod)

# SMTP settings (production)
AUTH_SMTP_HOST=smtp.sendgrid.net
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=apikey
AUTH_SMTP_PASS=${SENDGRID_API_KEY}
AUTH_SMTP_SECURE=false
AUTH_SMTP_FROM=noreply@${BASE_DOMAIN}

# Email templates
AUTH_EMAIL_TEMPLATE_VERIFY_EMAIL=true
AUTH_EMAIL_TEMPLATE_PASSWORD_RESET=true
AUTH_EMAIL_TEMPLATE_MAGIC_LINK=true
```

### Authentication Methods

```bash
# Email/password authentication
AUTH_EMAIL_PASSWORD_ENABLED=true
# Default: true

# Magic link authentication
AUTH_MAGIC_LINK_ENABLED=true
# Default: false

# Anonymous authentication
AUTH_ANONYMOUS_ENABLED=false
# Default: false

# OAuth providers
AUTH_OAUTH_PROVIDERS_ENABLED=false
# Default: false

# GitHub OAuth
AUTH_GITHUB_ENABLED=false
AUTH_GITHUB_CLIENT_ID=${GITHUB_CLIENT_ID}
AUTH_GITHUB_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}

# Google OAuth
AUTH_GOOGLE_ENABLED=false
AUTH_GOOGLE_CLIENT_ID=${GOOGLE_CLIENT_ID}
AUTH_GOOGLE_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}

# Facebook OAuth
AUTH_FACEBOOK_ENABLED=false
AUTH_FACEBOOK_CLIENT_ID=${FACEBOOK_CLIENT_ID}
AUTH_FACEBOOK_CLIENT_SECRET=${FACEBOOK_CLIENT_SECRET}

# Twitter OAuth
AUTH_TWITTER_ENABLED=false
AUTH_TWITTER_CONSUMER_KEY=${TWITTER_CONSUMER_KEY}
AUTH_TWITTER_CONSUMER_SECRET=${TWITTER_CONSUMER_SECRET}

# LinkedIn OAuth
AUTH_LINKEDIN_ENABLED=false
AUTH_LINKEDIN_CLIENT_ID=${LINKEDIN_CLIENT_ID}
AUTH_LINKEDIN_CLIENT_SECRET=${LINKEDIN_CLIENT_SECRET}
```

### Security Settings

```bash
# Password requirements
AUTH_PASSWORD_MIN_LENGTH=8
# Default: 8

AUTH_PASSWORD_REQUIRE_UPPERCASE=true
# Default: false

AUTH_PASSWORD_REQUIRE_LOWERCASE=true
# Default: false

AUTH_PASSWORD_REQUIRE_NUMBER=true
# Default: false

AUTH_PASSWORD_REQUIRE_SPECIAL=false
# Default: false

# Account security
AUTH_MAX_FAILED_LOGIN_ATTEMPTS=5
# Default: 5

AUTH_LOCKOUT_DURATION=900
# Default: 900 (15 minutes)

# Email verification
AUTH_EMAIL_VERIFICATION_REQUIRED=true
# Default: false (dev), true (prod)

# MFA/2FA
AUTH_MFA_ENABLED=false
# Default: false

AUTH_MFA_TOTP_ENABLED=false
# Default: false
```

### Rate Limiting

```bash
# Global rate limit
AUTH_RATE_LIMIT_ENABLED=true
# Default: true

AUTH_RATE_LIMIT_MAX_REQUESTS=100
# Default: 100

AUTH_RATE_LIMIT_WINDOW=60000
# Default: 60000 (1 minute)

# Per-endpoint limits
AUTH_RATE_LIMIT_SIGNUP=5
AUTH_RATE_LIMIT_LOGIN=10
AUTH_RATE_LIMIT_PASSWORD_RESET=3
```

### Resource Limits

```bash
# Memory limit
AUTH_MEMORY_LIMIT=512M
# Default: no limit

# CPU limit
AUTH_CPU_LIMIT=0.5
# Default: no limit
```

---

## Nginx Reverse Proxy

The main entry point for all HTTP/HTTPS traffic.

### Basic Configuration

```bash
# Enable/Disable (always true for required service)
NGINX_ENABLED=true
# Default: true

# Version
NGINX_VERSION=alpine
# Default: alpine

# Worker processes
NGINX_WORKER_PROCESSES=auto
# Default: auto

# Worker connections
NGINX_WORKER_CONNECTIONS=1024
# Default: 1024

# Server names hash bucket size
NGINX_SERVER_NAMES_HASH_BUCKET_SIZE=64
# Default: 64
```

### SSL Configuration

```bash
# SSL protocols
NGINX_SSL_PROTOCOLS="TLSv1.2 TLSv1.3"
# Default: "TLSv1.2 TLSv1.3"

# SSL ciphers
NGINX_SSL_CIPHERS="HIGH:!aNULL:!MD5"
# Default: Mozilla Intermediate

# SSL session cache
NGINX_SSL_SESSION_CACHE="shared:SSL:10m"
# Default: "shared:SSL:10m"

# SSL session timeout
NGINX_SSL_SESSION_TIMEOUT=10m
# Default: 10m

# OCSP stapling
NGINX_SSL_STAPLING=on
# Default: on

NGINX_SSL_STAPLING_VERIFY=on
# Default: on

# HSTS
NGINX_HSTS_ENABLED=true
# Default: true (prod)

NGINX_HSTS_MAX_AGE=31536000
# Default: 31536000

NGINX_HSTS_INCLUDE_SUBDOMAINS=true
# Default: true

NGINX_HSTS_PRELOAD=false
# Default: false
```

### Proxy Settings

```bash
# Proxy timeouts
NGINX_PROXY_CONNECT_TIMEOUT=60s
# Default: 60s

NGINX_PROXY_SEND_TIMEOUT=60s
# Default: 60s

NGINX_PROXY_READ_TIMEOUT=60s
# Default: 60s

# Proxy buffers
NGINX_PROXY_BUFFERING=off
# Default: off

NGINX_PROXY_BUFFER_SIZE=4k
# Default: 4k

NGINX_PROXY_BUFFERS="8 4k"
# Default: "8 4k"

# Client settings
NGINX_CLIENT_MAX_BODY_SIZE=100M
# Default: 100M

NGINX_CLIENT_BODY_TIMEOUT=60s
# Default: 60s

NGINX_CLIENT_HEADER_TIMEOUT=60s
# Default: 60s

# Keepalive
NGINX_KEEPALIVE_TIMEOUT=65
# Default: 65

NGINX_KEEPALIVE_REQUESTS=100
# Default: 100
```

### Caching

```bash
# Cache enabled
NGINX_CACHE_ENABLED=false
# Default: false

# Cache path
NGINX_CACHE_PATH=/var/cache/nginx
# Default: /var/cache/nginx

# Cache size
NGINX_CACHE_SIZE=1g
# Default: 1g

# Cache time
NGINX_CACHE_TIME=1h
# Default: 1h

# Cache key
NGINX_CACHE_KEY='$scheme$request_method$host$request_uri'
# Default: '$scheme$request_method$host$request_uri'

# Cache bypass
NGINX_CACHE_BYPASS='$http_pragma $http_authorization'
# Default: '$http_pragma $http_authorization'
```

### Rate Limiting

```bash
# Rate limiting enabled
NGINX_RATE_LIMIT_ENABLED=true
# Default: true

# Rate limit zones
NGINX_RATE_LIMIT_ZONES='
  limit_req_zone $binary_remote_addr zone=global:10m rate=10r/s;
  limit_req_zone $binary_remote_addr zone=api:10m rate=100r/s;
  limit_req_zone $binary_remote_addr zone=auth:10m rate=5r/s;
'

# Rate limit burst
NGINX_RATE_LIMIT_BURST=20
# Default: 20

# Rate limit delay
NGINX_RATE_LIMIT_NODELAY=true
# Default: true

# Rate limit status code
NGINX_RATE_LIMIT_STATUS=429
# Default: 429
```

### Gzip Compression

```bash
# Gzip enabled
NGINX_GZIP_ENABLED=true
# Default: true

# Gzip level
NGINX_GZIP_LEVEL=6
# Default: 6

# Gzip types
NGINX_GZIP_TYPES="text/plain text/css text/xml text/javascript application/json application/javascript application/xml+rss application/rss+xml application/atom+xml image/svg+xml text/x-js text/x-cross-domain-policy application/x-font-ttf application/x-font-opentype application/vnd.ms-fontobject image/x-icon"
# Default: common text types

# Gzip min length
NGINX_GZIP_MIN_LENGTH=1000
# Default: 1000
```

### Logging

```bash
# Access log
NGINX_ACCESS_LOG=/var/log/nginx/access.log
# Default: /var/log/nginx/access.log

# Error log
NGINX_ERROR_LOG=/var/log/nginx/error.log
# Default: /var/log/nginx/error.log

# Log level
NGINX_LOG_LEVEL=warn
# Default: warn (options: debug, info, notice, warn, error, crit, alert, emerg)

# Log format
NGINX_LOG_FORMAT='$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"'
# Default: combined
```

### Security Headers

```bash
# Security headers
NGINX_SECURITY_HEADERS=true
# Default: true

# X-Frame-Options
NGINX_X_FRAME_OPTIONS=SAMEORIGIN
# Default: SAMEORIGIN

# X-Content-Type-Options
NGINX_X_CONTENT_TYPE_OPTIONS=nosniff
# Default: nosniff

# X-XSS-Protection
NGINX_X_XSS_PROTECTION="1; mode=block"
# Default: "1; mode=block"

# Referrer-Policy
NGINX_REFERRER_POLICY="strict-origin-when-cross-origin"
# Default: "strict-origin-when-cross-origin"

# Content-Security-Policy
NGINX_CSP="default-src 'self'"
# Default: not set (customize per application)

# Permissions-Policy
NGINX_PERMISSIONS_POLICY="geolocation=(), microphone=(), camera=()"
# Default: restrictive
```

### Resource Limits

```bash
# Memory limit
NGINX_MEMORY_LIMIT=256M
# Default: no limit

# CPU limit
NGINX_CPU_LIMIT=0.5
# Default: no limit
```

---

## Service Dependencies

The required services have specific startup dependencies:

```
nginx → depends on → all other services
hasura → depends on → postgres, auth
auth → depends on → postgres
postgres → no dependencies (starts first)
```

## Health Checks

All required services include health checks:

```bash
# PostgreSQL
curl -f http://localhost:5432 || pg_isready

# Hasura
curl -f http://localhost:8080/healthz

# Auth
curl -f http://localhost:4000/healthz

# Nginx
curl -f http://localhost/health
```

## Complete Example

### Production Configuration

```bash
# ============================================
# REQUIRED SERVICES - PRODUCTION
# ============================================

# PostgreSQL
POSTGRES_VERSION=16
POSTGRES_DB=myapp
POSTGRES_USER=postgres
POSTGRES_PASSWORD=$(openssl rand -hex 32)
POSTGRES_MAX_CONNECTIONS=500
POSTGRES_SHARED_BUFFERS=4GB
POSTGRES_EFFECTIVE_CACHE_SIZE=12GB
TIMESCALEDB_ENABLED=true
POSTGIS_ENABLED=true
PGVECTOR_ENABLED=true
POSTGRES_BACKUP_ENABLED=true
POSTGRES_BACKUP_SCHEDULE="0 2 * * *"
POSTGRES_MEMORY_LIMIT=16G
POSTGRES_CPU_LIMIT=8

# Hasura
HASURA_VERSION=v2.44.0
HASURA_GRAPHQL_ADMIN_SECRET=$(openssl rand -hex 32)
HASURA_GRAPHQL_ENABLE_CONSOLE=false
HASURA_GRAPHQL_DEV_MODE=false
HASURA_GRAPHQL_ENABLE_ALLOWLIST=true
HASURA_GRAPHQL_ENABLE_SCHEMA_INTROSPECTION=false
HASURA_GRAPHQL_PG_CONNECTIONS=100
HASURA_GRAPHQL_CORS_DOMAIN="https://myapp.com"
HASURA_GRAPHQL_LOG_LEVEL=warn
HASURA_MEMORY_LIMIT=4G
HASURA_CPU_LIMIT=2

# Auth
AUTH_VERSION=0.36.0
AUTH_JWT_SECRET=$(openssl rand -hex 64)
AUTH_ACCESS_TOKEN_EXPIRES_IN=900
AUTH_REFRESH_TOKEN_EXPIRES_IN=2592000
AUTH_SMTP_ENABLED=true
AUTH_SMTP_HOST=smtp.sendgrid.net
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=apikey
AUTH_SMTP_PASS=${SENDGRID_API_KEY}
AUTH_EMAIL_VERIFICATION_REQUIRED=true
AUTH_PASSWORD_MIN_LENGTH=12
AUTH_PASSWORD_REQUIRE_UPPERCASE=true
AUTH_PASSWORD_REQUIRE_LOWERCASE=true
AUTH_PASSWORD_REQUIRE_NUMBER=true
AUTH_PASSWORD_REQUIRE_SPECIAL=true
AUTH_RATE_LIMIT_ENABLED=true
AUTH_MEMORY_LIMIT=1G
AUTH_CPU_LIMIT=1

# Nginx
NGINX_WORKER_PROCESSES=auto
NGINX_WORKER_CONNECTIONS=2048
NGINX_SSL_PROTOCOLS="TLSv1.2 TLSv1.3"
NGINX_HSTS_ENABLED=true
NGINX_HSTS_MAX_AGE=31536000
NGINX_CLIENT_MAX_BODY_SIZE=1G
NGINX_RATE_LIMIT_ENABLED=true
NGINX_GZIP_ENABLED=true
NGINX_SECURITY_HEADERS=true
NGINX_MEMORY_LIMIT=512M
NGINX_CPU_LIMIT=1
```

## Next Steps

- [Optional Services](./OPTIONAL-SERVICES.md) - Enable MinIO, Redis, Admin UI, etc.
- [User Services](./USER-SERVICES.md) - Add custom backend services
- [Frontend Apps](./FRONTEND-APPS.md) - Configure frontend applications
- [How-To Guides](./HOW-TO.md) - Common scenarios and examples