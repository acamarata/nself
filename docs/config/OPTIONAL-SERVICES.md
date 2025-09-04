# Optional Services Configuration

Enable these services based on your needs. Each service can be independently enabled/disabled.

## Table of Contents
- [Admin UI](#admin-ui)
- [MinIO (S3 Storage)](#minio-s3-storage)
- [Redis](#redis)
- [Mailpit (Development Email)](#mailpit-development-email)
- [MLflow](#mlflow)
- [Prometheus](#prometheus)
- [Grafana](#grafana)
- [Loki](#loki)
- [Temporal](#temporal)
- [Elasticsearch](#elasticsearch)
- [Kibana](#kibana)
- [Jaeger](#jaeger)
- [Webhook Service](#webhook-service)
- [Functions Runtime](#functions-runtime)
- [Dashboard](#dashboard)

---

## Admin UI

Web-based monitoring and management interface for nself.

### Basic Configuration

```bash
# Enable Admin UI
NSELF_ADMIN_ENABLED=true
# Default: false

# Version
NSELF_ADMIN_VERSION=0.0.3
# Default: 0.0.3

# Port
NSELF_ADMIN_PORT=3100
# Default: 3100

# Authentication provider
NSELF_ADMIN_AUTH_PROVIDER=basic
# Default: basic (options: basic, oauth, saml)

# Admin password (auto-generated if not set)
ADMIN_PASSWORD=$(openssl rand -hex 16)
# Default: auto-generated

# Admin secret key
ADMIN_SECRET_KEY=$(openssl rand -hex 32)
# Default: auto-generated
```

### Features Configuration

```bash
# Enable features
NSELF_ADMIN_FEATURES_MONITORING=true      # Service monitoring
NSELF_ADMIN_FEATURES_LOGS=true           # Log viewer
NSELF_ADMIN_FEATURES_DATABASE=true       # Database management
NSELF_ADMIN_FEATURES_BACKUP=true         # Backup management
NSELF_ADMIN_FEATURES_USERS=true          # User management
NSELF_ADMIN_FEATURES_CONFIG=true         # Configuration editor
NSELF_ADMIN_FEATURES_TERMINAL=false      # Web terminal (security risk)
# Default: all true except terminal

# Session timeout (minutes)
NSELF_ADMIN_SESSION_TIMEOUT=30
# Default: 30

# Max login attempts
NSELF_ADMIN_MAX_LOGIN_ATTEMPTS=5
# Default: 5
```

### OAuth Configuration (if using OAuth provider)

```bash
NSELF_ADMIN_OAUTH_PROVIDER=github
NSELF_ADMIN_OAUTH_CLIENT_ID=${GITHUB_CLIENT_ID}
NSELF_ADMIN_OAUTH_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}
NSELF_ADMIN_OAUTH_CALLBACK_URL=http://localhost:3100/auth/callback
```

---

## MinIO (S3 Storage)

S3-compatible object storage for files and assets.

### Basic Configuration

```bash
# Enable MinIO
MINIO_ENABLED=true
# Default: false

# Version
MINIO_VERSION=latest
# Default: latest

# Ports
MINIO_PORT=9000        # API port
MINIO_CONSOLE_PORT=9001 # Console port
# Defaults: 9000, 9001

# Root credentials (auto-generated if not set)
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=$(openssl rand -hex 32)
# Default: auto-generated

# Default bucket
MINIO_DEFAULT_BUCKET=${PROJECT_NAME}
# Default: ${PROJECT_NAME}
```

### Storage Configuration

```bash
# Storage path
MINIO_DATA_PATH=/data
# Default: /data

# Storage class
MINIO_STORAGE_CLASS=STANDARD
# Default: STANDARD (options: STANDARD, REDUCED_REDUNDANCY)

# Erasure coding (for multi-disk)
MINIO_ERASURE_SET_DRIVE_COUNT=4
# Default: 4

# Disk usage threshold
MINIO_DISK_USAGE_THRESHOLD=90
# Default: 90 (percent)

# Retention policy
MINIO_RETENTION_DAYS=90
# Default: 90

# Versioning
MINIO_VERSIONING=true
# Default: false
```

### Security

```bash
# TLS/SSL
MINIO_TLS_ENABLED=false
# Default: false

MINIO_TLS_CERT=/path/to/cert.pem
MINIO_TLS_KEY=/path/to/key.pem

# Encryption
MINIO_ENCRYPTION_ENABLED=true
# Default: false

MINIO_ENCRYPTION_MASTER_KEY=$(openssl rand -hex 32)
# Required if encryption enabled

# Access policies
MINIO_DEFAULT_POLICY=readwrite
# Default: readwrite (options: none, download, upload, public, readwrite)

# Anonymous access
MINIO_ANONYMOUS_ACCESS=false
# Default: false
```

### Performance

```bash
# Cache settings
MINIO_CACHE_ENABLED=true
MINIO_CACHE_SIZE=10GB
MINIO_CACHE_WATERMARK_LOW=70
MINIO_CACHE_WATERMARK_HIGH=90
MINIO_CACHE_QUOTA=80

# Connection pool
MINIO_CONN_READ_DEADLINE=10s
MINIO_CONN_WRITE_DEADLINE=10s
MINIO_CONN_POOL_SIZE=100
```

### Integration with Hasura Storage

```bash
# Hasura Storage enabled (requires MinIO)
STORAGE_ENABLED=true
# Default: false

# Storage version
STORAGE_VERSION=0.6.1
# Default: 0.6.1

# Storage port
STORAGE_PORT=8000
# Default: 8000

# Storage endpoint
S3_ENDPOINT=http://minio:9000
S3_ACCESS_KEY=${MINIO_ROOT_USER}
S3_SECRET_KEY=${MINIO_ROOT_PASSWORD}
S3_BUCKET=${MINIO_DEFAULT_BUCKET}
S3_REGION=us-east-1
```

---

## Redis

In-memory data store for caching, sessions, and queues.

### Basic Configuration

```bash
# Enable Redis
REDIS_ENABLED=true
# Default: false

# Version
REDIS_VERSION=7-alpine
# Default: 7-alpine

# Port
REDIS_PORT=6379
# Default: 6379

# Password (auto-generated if not set)
REDIS_PASSWORD=$(openssl rand -hex 16)
# Default: no password (dev), auto-generated (prod)

# Database count
REDIS_DATABASES=16
# Default: 16
```

### Persistence

```bash
# Persistence enabled
REDIS_PERSISTENCE_ENABLED=true
# Default: true

# AOF (Append Only File)
REDIS_AOF_ENABLED=yes
REDIS_AOF_FILENAME=appendonly.aof
REDIS_AOF_FSYNC=everysec
# Default: yes, appendonly.aof, everysec

# RDB (Redis Database Backup)
REDIS_RDB_ENABLED=yes
REDIS_RDB_FILENAME=dump.rdb
REDIS_RDB_SAVE="900 1 300 10 60 10000"
# Default: yes, dump.rdb, "900 1 300 10 60 10000"

# Backup
REDIS_BACKUP_ENABLED=true
REDIS_BACKUP_SCHEDULE="0 3 * * *"
REDIS_BACKUP_RETENTION_DAYS=7
```

### Memory Management

```bash
# Max memory
REDIS_MAXMEMORY=2gb
# Default: no limit

# Memory policy
REDIS_MAXMEMORY_POLICY=allkeys-lru
# Default: noeviction
# Options: volatile-lru, allkeys-lru, volatile-random, allkeys-random, volatile-ttl, noeviction

# Memory samples
REDIS_MAXMEMORY_SAMPLES=5
# Default: 5
```

### Performance

```bash
# TCP settings
REDIS_TCP_BACKLOG=511
REDIS_TCP_KEEPALIVE=300
REDIS_TIMEOUT=0

# Threading
REDIS_IO_THREADS=4
REDIS_IO_THREADS_DO_READS=yes

# Slow log
REDIS_SLOWLOG_LOG_SLOWER_THAN=10000
REDIS_SLOWLOG_MAX_LEN=128
```

### Clustering (optional)

```bash
# Cluster enabled
REDIS_CLUSTER_ENABLED=false
# Default: false

# Cluster nodes
REDIS_CLUSTER_NODES=6
REDIS_CLUSTER_REPLICAS=1
REDIS_CLUSTER_CONFIG_FILE=nodes.conf
```

---

## Mailpit (Development Email)

Email testing server for development (replaces MailHog).

### Basic Configuration

```bash
# Enable Mailpit (dev only)
MAILPIT_ENABLED=true
# Default: true (dev), false (prod)

# Version
MAILPIT_VERSION=latest
# Default: latest

# Ports
MAILPIT_SMTP_PORT=1025  # SMTP server
MAILPIT_UI_PORT=8025     # Web interface
# Defaults: 1025, 8025

# Hostname
MAILPIT_HOSTNAME=mailpit
# Default: mailpit
```

### Features

```bash
# Web UI authentication
MAILPIT_UI_AUTH_ENABLED=false
MAILPIT_UI_USERNAME=admin
MAILPIT_UI_PASSWORD=admin
# Default: false (no auth)

# Message retention
MAILPIT_MAX_MESSAGES=5000
# Default: 5000

# Message storage
MAILPIT_DATA_PATH=/data
# Default: /data

# API enabled
MAILPIT_API_ENABLED=true
# Default: true

# Webhook notifications
MAILPIT_WEBHOOK_ENABLED=false
MAILPIT_WEBHOOK_URL=http://webhook:3000/email
# Default: false
```

---

## MLflow

Machine learning lifecycle management platform.

### Basic Configuration

```bash
# Enable MLflow
MLFLOW_ENABLED=false
# Default: false

# Version
MLFLOW_VERSION=2.10.0
# Default: 2.10.0

# Port
MLFLOW_PORT=5000
# Default: 5000

# Backend store (PostgreSQL)
MLFLOW_BACKEND_STORE_URI=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/mlflow
# Default: uses main PostgreSQL

# Artifact store (MinIO)
MLFLOW_DEFAULT_ARTIFACT_ROOT=s3://mlflow/artifacts
# Default: s3://mlflow/artifacts
```

### S3 Configuration (uses MinIO)

```bash
AWS_ACCESS_KEY_ID=${MINIO_ROOT_USER}
AWS_SECRET_ACCESS_KEY=${MINIO_ROOT_PASSWORD}
MLFLOW_S3_ENDPOINT_URL=http://minio:9000
MLFLOW_S3_IGNORE_TLS=true
```

### Authentication

```bash
# Basic auth
MLFLOW_AUTH_ENABLED=false
MLFLOW_AUTH_USERNAME=admin
MLFLOW_AUTH_PASSWORD=$(openssl rand -hex 16)

# OAuth (optional)
MLFLOW_OAUTH_ENABLED=false
MLFLOW_OAUTH_PROVIDER=github
MLFLOW_OAUTH_CLIENT_ID=${GITHUB_CLIENT_ID}
MLFLOW_OAUTH_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}
```

---

## Prometheus

Metrics collection and monitoring system.

### Basic Configuration

```bash
# Enable Prometheus
PROMETHEUS_ENABLED=false
# Default: false

# Version
PROMETHEUS_VERSION=latest
# Default: latest

# Port
PROMETHEUS_PORT=9090
# Default: 9090

# Data retention
PROMETHEUS_RETENTION_TIME=15d
# Default: 15d

# Data path
PROMETHEUS_DATA_PATH=/prometheus
# Default: /prometheus
```

### Scrape Configuration

```bash
# Scrape interval
PROMETHEUS_SCRAPE_INTERVAL=15s
# Default: 15s

# Scrape timeout
PROMETHEUS_SCRAPE_TIMEOUT=10s
# Default: 10s

# Evaluation interval
PROMETHEUS_EVALUATION_INTERVAL=15s
# Default: 15s

# Targets (auto-configured for nself services)
PROMETHEUS_TARGETS="
  - postgres-exporter:9187
  - node-exporter:9100
  - nginx-exporter:9113
  - redis-exporter:9121
"
```

### Alerting

```bash
# Alertmanager enabled
PROMETHEUS_ALERTMANAGER_ENABLED=false
PROMETHEUS_ALERTMANAGER_URL=http://alertmanager:9093

# Alert rules path
PROMETHEUS_RULES_PATH=/etc/prometheus/rules
```

---

## Grafana

Visualization and dashboards for metrics.

### Basic Configuration

```bash
# Enable Grafana
GRAFANA_ENABLED=false
# Default: false

# Version
GRAFANA_VERSION=latest
# Default: latest

# Port
GRAFANA_PORT=3000
# Default: 3000

# Admin credentials
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=$(openssl rand -hex 16)
# Default: admin/admin
```

### Data Sources

```bash
# Auto-configure Prometheus
GRAFANA_PROMETHEUS_URL=http://prometheus:9090
# Default: http://prometheus:9090

# Auto-configure Loki
GRAFANA_LOKI_URL=http://loki:3100
# Default: http://loki:3100

# PostgreSQL metrics
GRAFANA_POSTGRES_ENABLED=true
GRAFANA_POSTGRES_URL=postgres:5432
GRAFANA_POSTGRES_USER=${POSTGRES_USER}
GRAFANA_POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
```

### Features

```bash
# Anonymous access
GRAFANA_ANONYMOUS_ENABLED=false
GRAFANA_ANONYMOUS_ORG_ROLE=Viewer

# OAuth
GRAFANA_OAUTH_ENABLED=false
GRAFANA_OAUTH_PROVIDER=github
GRAFANA_OAUTH_CLIENT_ID=${GITHUB_CLIENT_ID}
GRAFANA_OAUTH_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}

# SMTP
GRAFANA_SMTP_ENABLED=false
GRAFANA_SMTP_HOST=${AUTH_SMTP_HOST}
GRAFANA_SMTP_PORT=${AUTH_SMTP_PORT}
GRAFANA_SMTP_USER=${AUTH_SMTP_USER}
GRAFANA_SMTP_PASSWORD=${AUTH_SMTP_PASS}
```

---

## Loki

Log aggregation system (works with Grafana).

### Basic Configuration

```bash
# Enable Loki
LOKI_ENABLED=false
# Default: false

# Version
LOKI_VERSION=latest
# Default: latest

# Port
LOKI_PORT=3100
# Default: 3100

# Data path
LOKI_DATA_PATH=/loki
# Default: /loki
```

### Storage

```bash
# Storage backend
LOKI_STORAGE_BACKEND=filesystem
# Default: filesystem (options: filesystem, s3, gcs, azure)

# S3 storage (if backend=s3)
LOKI_S3_BUCKET=loki-logs
LOKI_S3_ENDPOINT=http://minio:9000
LOKI_S3_ACCESS_KEY=${MINIO_ROOT_USER}
LOKI_S3_SECRET_KEY=${MINIO_ROOT_PASSWORD}

# Retention
LOKI_RETENTION_ENABLED=true
LOKI_RETENTION_PERIOD=720h
# Default: 720h (30 days)
```

### Limits

```bash
# Ingestion rate limit
LOKI_INGESTION_RATE_LIMIT=10MB
# Default: 10MB

# Ingestion burst size
LOKI_INGESTION_BURST_SIZE=20MB
# Default: 20MB

# Query timeout
LOKI_QUERY_TIMEOUT=5m
# Default: 5m
```

---

## Temporal

Workflow orchestration platform.

### Basic Configuration

```bash
# Enable Temporal
TEMPORAL_ENABLED=false
# Default: false

# Version
TEMPORAL_VERSION=latest
# Default: latest

# Ports
TEMPORAL_FRONTEND_PORT=7233
TEMPORAL_WEB_PORT=8088
# Defaults: 7233, 8088

# Namespace
TEMPORAL_DEFAULT_NAMESPACE=default
# Default: default

# Database (uses PostgreSQL)
TEMPORAL_DB_NAME=temporal
TEMPORAL_DB_USER=${POSTGRES_USER}
TEMPORAL_DB_PASSWORD=${POSTGRES_PASSWORD}
```

### Persistence

```bash
# Number of history shards
TEMPORAL_NUM_HISTORY_SHARDS=4
# Default: 4

# Retention period
TEMPORAL_RETENTION_DAYS=30
# Default: 30

# Archive enabled
TEMPORAL_ARCHIVE_ENABLED=false
TEMPORAL_ARCHIVE_BUCKET=temporal-archive
```

---

## Elasticsearch

Full-text search and analytics engine.

### Basic Configuration

```bash
# Enable Elasticsearch
ELASTICSEARCH_ENABLED=false
# Default: false

# Version
ELASTICSEARCH_VERSION=8.12.0
# Default: 8.12.0

# Ports
ELASTICSEARCH_PORT=9200      # HTTP
ELASTICSEARCH_TCP_PORT=9300   # TCP/Transport
# Defaults: 9200, 9300

# Cluster name
ELASTICSEARCH_CLUSTER_NAME=${PROJECT_NAME}
# Default: ${PROJECT_NAME}

# Node name
ELASTICSEARCH_NODE_NAME=${PROJECT_NAME}-es-1
# Default: ${PROJECT_NAME}-es-1
```

### Memory and Performance

```bash
# JVM heap size
ELASTICSEARCH_HEAP_SIZE=2g
# Default: 2g

# Memory lock
ELASTICSEARCH_MEMORY_LOCK=true
# Default: true

# Index settings
ELASTICSEARCH_INDEX_NUMBER_OF_SHARDS=1
ELASTICSEARCH_INDEX_NUMBER_OF_REPLICAS=0
```

### Security

```bash
# Security enabled
ELASTICSEARCH_SECURITY_ENABLED=true
# Default: true

# Authentication
ELASTICSEARCH_USERNAME=elastic
ELASTICSEARCH_PASSWORD=$(openssl rand -hex 16)
# Default: auto-generated

# TLS
ELASTICSEARCH_TLS_ENABLED=false
ELASTICSEARCH_TLS_CERT=/path/to/cert
ELASTICSEARCH_TLS_KEY=/path/to/key
```

---

## Kibana

Elasticsearch visualization and management.

### Basic Configuration

```bash
# Enable Kibana (requires Elasticsearch)
KIBANA_ENABLED=false
# Default: false

# Version (should match Elasticsearch)
KIBANA_VERSION=8.12.0
# Default: 8.12.0

# Port
KIBANA_PORT=5601
# Default: 5601

# Elasticsearch connection
KIBANA_ELASTICSEARCH_URL=http://elasticsearch:9200
KIBANA_ELASTICSEARCH_USERNAME=${ELASTICSEARCH_USERNAME}
KIBANA_ELASTICSEARCH_PASSWORD=${ELASTICSEARCH_PASSWORD}
```

---

## Jaeger

Distributed tracing system.

### Basic Configuration

```bash
# Enable Jaeger
JAEGER_ENABLED=false
# Default: false

# Version
JAEGER_VERSION=latest
# Default: latest

# Ports
JAEGER_AGENT_PORT=6831       # UDP
JAEGER_COLLECTOR_PORT=14268   # HTTP
JAEGER_QUERY_PORT=16686       # UI
# Defaults: 6831, 14268, 16686

# Storage backend
JAEGER_STORAGE_TYPE=memory
# Default: memory (options: memory, elasticsearch, cassandra)

# Elasticsearch storage
JAEGER_ES_SERVER_URLS=http://elasticsearch:9200
JAEGER_ES_USERNAME=${ELASTICSEARCH_USERNAME}
JAEGER_ES_PASSWORD=${ELASTICSEARCH_PASSWORD}
```

---

## Webhook Service

Service for handling webhooks.

### Basic Configuration

```bash
# Enable webhook service
WEBHOOK_SERVICE_ENABLED=false
# Default: false

# Version
WEBHOOK_VERSION=latest
# Default: latest

# Port
WEBHOOK_PORT=9000
# Default: 9000

# Secret for webhook validation
WEBHOOK_SECRET=$(openssl rand -hex 32)
# Default: auto-generated

# Retry configuration
WEBHOOK_MAX_RETRIES=3
WEBHOOK_RETRY_DELAY=1000
# Defaults: 3, 1000ms
```

---

## Functions Runtime

Serverless functions runtime.

### Basic Configuration

```bash
# Enable functions
FUNCTIONS_ENABLED=false
# Default: false

# Runtime
FUNCTIONS_RUNTIME=node
# Default: node (options: node, python, go)

# Port
FUNCTIONS_PORT=3000
# Default: 3000

# Workers
FUNCTIONS_WORKERS=4
# Default: 4

# Timeout
FUNCTIONS_TIMEOUT=30000
# Default: 30000ms

# Memory limit per function
FUNCTIONS_MEMORY_LIMIT=256M
# Default: 256M
```

---

## Dashboard

Optional Hasura console alternative.

### Basic Configuration

```bash
# Enable dashboard
DASHBOARD_ENABLED=false
# Default: false

# Version
DASHBOARD_VERSION=latest
# Default: latest

# Port
DASHBOARD_PORT=3001
# Default: 3001

# Authentication
DASHBOARD_AUTH_ENABLED=true
DASHBOARD_AUTH_SECRET=${ADMIN_SECRET}
```

---

## Complete Example

### Development Environment

```bash
# ============================================
# OPTIONAL SERVICES - DEVELOPMENT
# ============================================

# Admin UI
NSELF_ADMIN_ENABLED=true
NSELF_ADMIN_PORT=3100
ADMIN_PASSWORD=admin123

# Storage
MINIO_ENABLED=true
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
STORAGE_ENABLED=true

# Cache
REDIS_ENABLED=true
REDIS_PASSWORD=redis123

# Email testing
MAILPIT_ENABLED=true

# Monitoring (basic)
PROMETHEUS_ENABLED=true
GRAFANA_ENABLED=true
GRAFANA_ADMIN_PASSWORD=admin123
```

### Production Environment

```bash
# ============================================
# OPTIONAL SERVICES - PRODUCTION
# ============================================

# Admin UI
NSELF_ADMIN_ENABLED=true
NSELF_ADMIN_PORT=3100
NSELF_ADMIN_AUTH_PROVIDER=oauth
NSELF_ADMIN_OAUTH_PROVIDER=github
NSELF_ADMIN_OAUTH_CLIENT_ID=${GITHUB_CLIENT_ID}
NSELF_ADMIN_OAUTH_CLIENT_SECRET=${GITHUB_CLIENT_SECRET}
ADMIN_SECRET_KEY=$(openssl rand -hex 32)

# Storage
MINIO_ENABLED=true
MINIO_ROOT_USER=$(openssl rand -hex 16)
MINIO_ROOT_PASSWORD=$(openssl rand -hex 32)
MINIO_VERSIONING=true
MINIO_ENCRYPTION_ENABLED=true
MINIO_ENCRYPTION_MASTER_KEY=$(openssl rand -hex 32)
STORAGE_ENABLED=true

# Cache
REDIS_ENABLED=true
REDIS_PASSWORD=$(openssl rand -hex 32)
REDIS_PERSISTENCE_ENABLED=true
REDIS_MAXMEMORY=4gb
REDIS_MAXMEMORY_POLICY=allkeys-lru

# Email (production SMTP)
MAILPIT_ENABLED=false

# Full monitoring stack
PROMETHEUS_ENABLED=true
PROMETHEUS_RETENTION_TIME=30d
GRAFANA_ENABLED=true
GRAFANA_ADMIN_PASSWORD=$(openssl rand -hex 16)
LOKI_ENABLED=true
LOKI_RETENTION_PERIOD=720h

# ML platform
MLFLOW_ENABLED=true
MLFLOW_AUTH_ENABLED=true
MLFLOW_AUTH_PASSWORD=$(openssl rand -hex 16)

# Search
ELASTICSEARCH_ENABLED=true
ELASTICSEARCH_HEAP_SIZE=4g
ELASTICSEARCH_PASSWORD=$(openssl rand -hex 16)
KIBANA_ENABLED=true

# Tracing
JAEGER_ENABLED=true
JAEGER_STORAGE_TYPE=elasticsearch

# Workflow
TEMPORAL_ENABLED=true
TEMPORAL_NUM_HISTORY_SHARDS=8
```

## Service Dependencies

Optional services may depend on:
- **Storage services** → require MinIO
- **Monitoring services** → may require Prometheus
- **Search services** → Kibana requires Elasticsearch
- **All services** → can use Redis for caching

## Enabling Services

To enable any optional service:

1. Set `SERVICE_ENABLED=true` in `.env.local`
2. Configure any required settings
3. Run `nself build` to generate configurations
4. Run `nself start` to start services

## Next Steps

- [User Services](./USER-SERVICES.md) - Add custom backend services
- [Frontend Apps](./FRONTEND-APPS.md) - Configure frontend applications
- [How-To Guides](./HOW-TO.md) - Common scenarios and examples
- [Environment Reference](./ENV-REFERENCE.md) - Complete variable list