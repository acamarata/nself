# nself Configuration Guide

This is the complete configuration reference for nself - a comprehensive self-hosted backend stack.

## Table of Contents

1. [Basic Settings](\1) - Project configuration and core settings
2. [Required Services](\1) - PostgreSQL, Hasura, Auth, Nginx
3. [Optional Services](\1) - Admin UI, MinIO, Redis, MLflow, etc.
4. [User Services](\1) - Custom backend services (CS_N configuration)
5. [Frontend Apps](\1) - Frontend application configuration (FA_N)
6. [Environment Variables](\1) - Complete environment variable reference
7. [How-To Guides](\1) - Common configuration scenarios

## Quick Start

The minimum configuration required to run nself:

```bash
# .env.local (minimum required)
PROJECT_NAME=myproject
BASE_DOMAIN=local.nself.org
ENV=development

# That's it! Everything else has smart defaults
```

## Configuration Files

nself uses a hierarchical environment file system:

1. `.env` - Base configuration (committed to git)
2. `.env.dev` - Development overrides
3. `.env.staging` - Staging overrides  
4. `.env.prod` - Production overrides
5. `.env.local` - Local overrides (git-ignored, highest priority)
6. `.env.secrets` - Production secrets (server-only)

## Service Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        NGINX (Reverse Proxy)                 │
│                     SSL, Routing, Load Balancing             │
└─────────────────────────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
┌───────▼──────┐       ┌────────▼──────┐      ┌────────▼──────┐
│   REQUIRED   │       │   OPTIONAL    │      │     USER      │
│   SERVICES   │       │   SERVICES    │      │   SERVICES    │
├──────────────┤       ├───────────────┤      ├───────────────┤
│ • PostgreSQL │       │ • Admin UI    │      │ • CS_1 (API)  │
│ • Hasura     │       │ • MinIO (S3)  │      │ • CS_2 (Auth) │
│ • Auth       │       │ • Redis       │      │ • CS_3 (...)  │
│ • Nginx      │       │ • MLflow      │      │ • CS_N        │
└──────────────┘       │ • Mailpit     │      └───────────────┘
                       │ • Prometheus  │
                       │ • Grafana     │
                       │ • Loki        │
                       └───────────────┘
                                │
                    ┌───────────▼───────────┐
                    │    FRONTEND APPS      │
                    ├───────────────────────┤
                    │ • FA_1 (Todo App)     │
                    │ • FA_2 (Dashboard)    │
                    │ • FA_3 (Admin Panel)  │
                    │ • FA_N                │
                    └───────────────────────┘
```

## Configuration Categories

### 1. Basic Settings
Core project configuration that affects all services:
- Project name, domain, environment
- Ports, SSL, networking
- Global resource limits

### 2. Required Services
Always-on services that form the core stack:
- **PostgreSQL**: Primary database with 60+ extensions
- **Hasura**: Instant GraphQL API
- **Auth**: JWT authentication service
- **Nginx**: Reverse proxy and SSL termination

### 3. Optional Services
Enable based on your needs:
- **Admin UI**: Web-based monitoring dashboard
- **MinIO**: S3-compatible object storage
- **Redis**: In-memory cache and queues
- **MLflow**: Machine learning lifecycle
- **Monitoring**: Prometheus, Grafana, Loki
- **Email**: Mailpit for development

### 4. User Services (Backend)
Custom backend services running in Docker:
- Microservices, APIs, workers
- Any language/framework
- Auto-configured with nginx routing
- Full access to core services

### 5. Frontend Apps
Frontend applications with backend integration:
- Automatic Hasura remote schemas
- Table prefix isolation
- Custom domains
- API routing

## Quick Examples

### Enable Optional Services
```bash
# .env.local
REDIS_ENABLED=true
MINIO_ENABLED=true
NSELF_ADMIN_ENABLED=true
```

### Add Custom Backend Services
```bash
# .env.local
SERVICES_ENABLED=true
CS_1=api:fastapi:3001:api.myapp.com
CS_2=worker:celery:3002
CS_3=webhook:node-ts:3003:webhooks
```

### Configure Frontend Apps
```bash
# .env.local
FRONTEND_APPS_ENABLED=true
FA_1=todoapp:todotracker.com:tdt_:api.todotracker.com
FA_2=dashboard:dashboard.myapp.com:dash_:api.dashboard.myapp.com
```

## Environment Loading Order

Variables are loaded in this priority (highest to lowest):
1. `.env.local` (your local overrides)
2. `.env.${ENV}` (environment-specific)
3. `.env` (base configuration)
4. Built-in defaults

## Next Steps

- [Basic Settings](\1) - Configure your project
- [Required Services](\1) - Customize core services
- [Optional Services](\1) - Enable additional features
- [User Services](\1) - Add custom backend services
- [Frontend Apps](\1) - Configure frontend applications
- [How-To Guides](\1) - Common scenarios and examples