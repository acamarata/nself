# nself Architecture

Understanding how nself works internally.

---

## Overview

- **[Architecture Overview](ARCHITECTURE.md)** - Complete system architecture
- **[Project Structure](PROJECT_STRUCTURE.md)** - File and directory organization
- **[Build Architecture](BUILD_ARCHITECTURE.md)** - How the build system works
- **[API Reference](API.md)** - GraphQL and REST APIs

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          YOUR APPLICATION                            │
├─────────────────────────────────────────────────────────────────────┤
│   Frontend (React, Vue, Next.js, etc.)                              │
│   ↓ GraphQL queries and mutations                                   │
├─────────────────────────────────────────────────────────────────────┤
│                              nself                                   │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              REQUIRED SERVICES (4)                        │      │
│   │   PostgreSQL  ·  Hasura GraphQL  ·  Auth  ·  Nginx       │      │
│   └──────────────────────────────────────────────────────────┘      │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              OPTIONAL SERVICES (7)                        │      │
│   │   Redis  ·  MinIO  ·  Search  ·  Mail  ·  Functions      │      │
│   │   MLflow  ·  Admin Dashboard                              │      │
│   └──────────────────────────────────────────────────────────┘      │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              MONITORING BUNDLE (10)                       │      │
│   │   Prometheus · Grafana · Loki · Tempo · Alertmanager     │      │
│   │   cAdvisor · Node Exporter · Postgres/Redis Exporters    │      │
│   └──────────────────────────────────────────────────────────┘      │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────┐      │
│   │              CUSTOM SERVICES (unlimited)                  │      │
│   │   Your services built from 40+ templates                 │      │
│   └──────────────────────────────────────────────────────────┘      │
├─────────────────────────────────────────────────────────────────────┤
│   Runs on: Docker Compose · Any Cloud · Any Server · Laptop         │
└─────────────────────────────────────────────────────────────────────┘
```

**[View Full Architecture](ARCHITECTURE.md)**

---

## Core Components

### Required Services (4)

Always running, always enabled:

1. **PostgreSQL** - Database
2. **Hasura** - GraphQL Engine
3. **Auth** - nHost Authentication Service
4. **Nginx** - Reverse proxy and SSL termination

**[Learn More](../services/SERVICES_REQUIRED.md)**

### Optional Services (7)

Enable with `*_ENABLED=true`:

1. **nself-admin** - Web management UI
2. **MinIO** - S3-compatible storage
3. **Redis** - Cache and sessions
4. **Functions** - Serverless functions runtime
5. **MLflow** - ML experiment tracking
6. **Mail** - Email service (MailPit for dev)
7. **Search** - Search service (MeiliSearch)

**[Learn More](../services/SERVICES_OPTIONAL.md)**

### Monitoring Bundle (10)

All-or-nothing monitoring stack enabled with `MONITORING_ENABLED=true`:

- Prometheus - Metrics database
- Grafana - Visualization
- Loki - Log aggregation
- Promtail - Log shipping (required for Loki)
- Tempo - Distributed tracing
- Alertmanager - Alert routing
- cAdvisor - Container metrics
- Node Exporter - System metrics
- Postgres Exporter - PostgreSQL metrics
- Redis Exporter - Redis metrics

**[Learn More](../services/MONITORING-BUNDLE.md)**

---

## Project Structure

### Generated Files

When you run `nself build`, these files are generated:

```
project/
├── docker-compose.yml        # All service definitions
├── .env                      # Environment configuration
├── nginx/
│   ├── nginx.conf           # Main nginx config
│   ├── includes/            # Security headers, gzip
│   └── sites/              # Route configs for all services
├── postgres/
│   └── init/
│       └── 00-init.sql      # Database initialization
├── ssl/
│   ├── cert.pem            # Self-signed cert (local)
│   └── key.pem             # Private key
├── services/               # Custom services (CS_N)
│   ├── my_api/
│   ├── my_worker/
│   └── ...
└── monitoring/             # Monitoring configs
    ├── prometheus/
    ├── grafana/
    ├── loki/
    └── alertmanager/
```

**[View Full Project Structure](PROJECT_STRUCTURE.md)**

---

## Build System

### Build Process

```
nself build
    ↓
1. Load environment variables (.env)
2. Validate configuration
3. Generate docker-compose.yml
    - Required services (4)
    - Optional services (based on *_ENABLED)
    - Monitoring bundle (if MONITORING_ENABLED=true)
    - Custom services (CS_1 through CS_10)
4. Generate nginx configuration
    - SSL certificates
    - Route configs for each service
    - Security headers
5. Generate service files
    - Custom service templates
    - Monitoring configs
    - Database init scripts
6. Build Docker images (if needed)
```

**[View Full Build Architecture](BUILD_ARCHITECTURE.md)**

### Configuration Priority

Environment variables are loaded in this order (later overrides earlier):

1. `.env.dev` - Base configuration (committed to git)
2. `.env.local` - Developer overrides (gitignored)
3. `.env.staging` - Staging-specific (on staging server)
4. `.env.prod` - Production-specific (on prod server)
5. `.secrets` - Top-secret credentials (generated on server)

**[Learn More](../configuration/ENVIRONMENT-VARIABLES.md)**

---

## API Architecture

### GraphQL API (Hasura)

- Auto-generated from database schema
- Real-time subscriptions
- Role-based access control
- Custom business logic via Actions and Events

**Endpoint:** `https://api.{domain}`

### Authentication API

- JWT-based authentication
- OAuth providers (Google, GitHub, etc.)
- User management
- Session management

**Endpoint:** `https://auth.{domain}`

### REST APIs (Custom Services)

Your custom services can expose REST APIs:

```bash
# Define custom service
CS_1=my-api:express-js:8001

# Access at
https://my-api.{domain}
```

**[View Full API Reference](API.md)**

---

## Data Flow

### Client → Database

```
Frontend
    ↓ GraphQL Query
Nginx (SSL termination)
    ↓ Route to Hasura
Hasura GraphQL Engine
    ↓ Check permissions
    ↓ Execute query
PostgreSQL
    ↓ Return data
Hasura (transform/filter)
    ↓ GraphQL Response
Frontend
```

### Custom Service → Database

```
Custom Service (Express, FastAPI, etc.)
    ↓ SQL/ORM or GraphQL
PostgreSQL (direct) or Hasura (via GraphQL)
    ↓ Return data
Custom Service
    ↓ Business logic
    ↓ Response
Client
```

**[Learn More](../guides/SERVICE-TO-SERVICE-COMMUNICATION.md)**

---

## Network Architecture

### Docker Networks

All services run on a shared Docker network:

```
docker network: {PROJECT_NAME}_network

Services can communicate by container name:
- postgres:5432
- hasura:8080
- auth:4000
- redis:6379
```

### External Access

Nginx routes external requests to internal services:

```
https://api.{domain}      → hasura:8080
https://auth.{domain}     → auth:4000
https://admin.{domain}    → nself-admin:3000
https://my-api.{domain}   → my-api:8001
```

---

## Plugin Architecture (v0.4.8)

Plugins extend nself with external integrations:

```
Plugin Structure:
├── plugin.yaml             # Metadata and configuration
├── schema.sql             # Database tables (prefixed)
├── routes.yaml            # Nginx routes (webhooks)
├── cli.yaml               # CLI actions
├── services.yaml          # Docker services (optional)
└── views/                 # Analytics views
```

**Features:**
- Database schemas with automatic Hasura tracking
- Webhook handlers with signature verification
- CLI actions for data management
- Optional Docker services for background processing

**[Learn More](../plugins/index.md)**

---

## Security Architecture

### SSL/TLS

- Self-signed certificates for local development
- Let's Encrypt for production
- Automatic certificate renewal
- HTTP → HTTPS redirect

### Authentication

- JWT tokens with configurable expiry
- Refresh token rotation
- Role-based access control (RBAC)
- Row-level security (RLS) in PostgreSQL

### Secrets Management

- Environment-specific secrets files
- Never committed to git
- SSH-only access for production
- Automatic secret generation

**[View Security Guide](../guides/SECURITY.md)**

---

## Scaling Architecture

### Horizontal Scaling

```bash
# Scale specific services
nself scale api 3          # 3 Hasura instances
nself scale functions 5    # 5 function workers
```

### Vertical Scaling

Configure resource limits in `.env`:

```bash
# Memory limits
POSTGRES_MEMORY_LIMIT=4g
HASURA_MEMORY_LIMIT=2g
```

### Database Scaling

- Read replicas (manual setup)
- Connection pooling (PgBouncer)
- Query optimization with indexes

**[Learn More](../commands/SCALE.md)**

---

## Monitoring Architecture

### Metrics Flow

```
Services (expose /metrics)
    ↓
Prometheus (scrape metrics)
    ↓
Grafana (visualize)
```

### Logs Flow

```
Services (stdout/stderr)
    ↓
Docker (log driver)
    ↓
Promtail (collect logs)
    ↓
Loki (store logs)
    ↓
Grafana (view logs)
```

### Traces Flow

```
Services (instrumented)
    ↓
Tempo (collect traces)
    ↓
Grafana (view traces)
```

**[Learn More](../services/MONITORING-BUNDLE.md)**

---

## Deployment Architecture

### Local Development

```
Developer Machine
├── Docker Desktop
├── nself CLI
└── Project files
```

### Production Deployment

```
Production Server
├── Docker Engine
├── nself CLI (via SSH)
├── Project files (deployed)
├── SSL certificates (Let's Encrypt)
└── Backups (automated)
```

**Deployment Flow:**

```
Local Machine
    ↓ nself deploy prod
SSH Connection
    ↓ Transfer files
Production Server
    ↓ Build images
    ↓ Zero-downtime restart
    ↓ Health checks
Production Live
```

**[View Deployment Guide](../guides/Deployment.md)**

---

## Related Documentation

- **[Services Overview](../services/SERVICES.md)** - All available services
- **[Configuration](../configuration/README.md)** - Configuration options
- **[Commands](../commands/README.md)** - CLI commands
- **[Guides](../guides/README.md)** - Usage guides

---

**[Back to Documentation Home](../README.md)**
