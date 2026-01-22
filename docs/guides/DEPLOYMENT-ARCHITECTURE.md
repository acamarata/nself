# Deployment Architecture

This guide explains how nself deploys services across different environments and the distinction between staging and production deployments.

## Service Categories

nself organizes services into five distinct categories:

### 1. Core Services (4) - Always Deployed

These are the foundational services required for any nself deployment:

| Service | Purpose | Subdomain |
|---------|---------|-----------|
| **PostgreSQL** | Primary database | Internal only |
| **Hasura** | GraphQL API engine | `api.domain.com` |
| **Auth** | Authentication service (nhost-compatible) | `auth.domain.com` |
| **Nginx** | Reverse proxy & SSL termination | Main entry point |

### 2. Optional Services (7) - Based on `*_ENABLED` Vars

Enable these based on your application needs:

| Service | Enable Flag | Subdomain | Purpose |
|---------|-------------|-----------|---------|
| **nself-admin** | `NSELF_ADMIN_ENABLED=true` | `admin.domain.com` | Web management UI |
| **MinIO** | `MINIO_ENABLED=true` | `minio.domain.com` | S3-compatible storage |
| **Redis** | `REDIS_ENABLED=true` | Internal | Cache & sessions |
| **Functions** | `FUNCTIONS_ENABLED=true` | `functions.domain.com` | Serverless runtime |
| **MLflow** | `MLFLOW_ENABLED=true` | `mlflow.domain.com` | ML experiment tracking |
| **Mailpit** | `MAILPIT_ENABLED=true` | `mail.domain.com` | Dev email testing |
| **Meilisearch** | `MEILISEARCH_ENABLED=true` | `search.domain.com` | Full-text search |

### 3. Monitoring Bundle (10) - All or Nothing

When `MONITORING_ENABLED=true`, all 10 services are deployed:

| Service | Purpose |
|---------|---------|
| Prometheus | Metrics collection |
| Grafana | Dashboards & visualization |
| Loki | Log aggregation |
| Promtail | Log shipping (required for Loki) |
| Tempo | Distributed tracing |
| Alertmanager | Alert routing |
| cAdvisor | Container metrics |
| Node Exporter | System metrics |
| Postgres Exporter | Database metrics |
| Redis Exporter | Redis metrics |

### 4. Custom Services (CS_N) - Your Backend Applications

Completely independent backend applications with custom business logic:

```bash
# In .env
CS_1=order-api:express-js:8001    # Order processing API
CS_2=webhooks:nestjs:8002         # Webhook handler service
CS_3=payments:fastapi-py:8003     # Payment processing
CS_4=ml-inference:fastapi-py:8004 # ML model serving
```

Custom services are **separate applications** that:
- Have their own codebase and logic
- May or may not connect to the database
- Run as independent Docker containers
- Can integrate with Hasura via Actions/Event Triggers

**Examples:**
- Order processing API
- Payment processing service
- Email notification worker
- ML inference endpoint
- Third-party integration handlers (webhooks, callbacks)

### 5. Remote Schemas - Multi-App Hasura Endpoints

**Different from Custom Services!** Remote Schemas are multiple Hasura GraphQL endpoints for different apps/tenants, all accessing the **same database** with different exposed schemas:

```
Same nself instance, same PostgreSQL, different GraphQL APIs:

api.app1.com → Hasura endpoint exposing App1 tables/permissions
api.app2.com → Hasura endpoint exposing App2 tables/permissions
```

**Use case**: One nself deployment serving multiple applications:
- `www.app1.com` uses `api.app1.com` (sees users, products, orders)
- `www.app2.com` uses `api.app2.com` (sees different tables/fields)

Configuration in `.env`:
```bash
# Remote Schemas (multiple Hasura endpoints)
REMOTE_SCHEMA_1_NAME=app1
REMOTE_SCHEMA_1_DOMAIN=api.app1.com

REMOTE_SCHEMA_2_NAME=app2
REMOTE_SCHEMA_2_DOMAIN=api.app2.com
```

### 6. Frontend Apps (FRONTEND_APP_N) - External Applications

Frontend applications configured for Nginx routing:

```bash
# In .env
FRONTEND_APP_1_NAME=web
FRONTEND_APP_1_PORT=3000
FRONTEND_APP_1_ROUTE=app

FRONTEND_APP_2_NAME=admin
FRONTEND_APP_2_PORT=3001
FRONTEND_APP_2_ROUTE=dashboard
```

**Key Point**: These are **NOT Docker containers** - they run outside Docker and Nginx routes to them.

---

## Deployment Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                    LOCAL DEVELOPMENT                             │
│  nself init → nself build → nself start                         │
│  All services run in Docker on localhost                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       STAGING                                    │
│  nself staging deploy                                           │
│                                                                  │
│  Deploys: Core + Optional + Monitoring + Custom + Frontends    │
│                                                                  │
│  Frontend apps served by Nginx on subdomains:                   │
│    app.staging.example.com → Frontend App 1                     │
│    dashboard.staging.example.com → Frontend App 2               │
│                                                                  │
│  Complete replica for testing                                    │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      PRODUCTION                                  │
│  nself deploy prod                                              │
│                                                                  │
│  Deploys: Core + Optional + Monitoring + Custom                 │
│  (Frontend apps excluded - they're external)                    │
│                                                                  │
│  Frontend apps deployed separately:                              │
│    ├── Vercel (Next.js, React)                                  │
│    ├── Cloudflare Pages (static sites)                          │
│    ├── Mobile apps (App Store, Play Store)                      │
│    └── Any CDN/hosting platform                                 │
│                                                                  │
│  API endpoints exposed:                                          │
│    api.example.com → Hasura GraphQL                             │
│    auth.example.com → Authentication                            │
└─────────────────────────────────────────────────────────────────┘
```

---

## Staging vs Production

| Aspect | Staging | Production |
|--------|---------|------------|
| **Frontend Apps** | ✅ Deployed (Nginx serves them) | ❌ External (Vercel, CDN) |
| **Hasura Console** | ✅ Enabled | ❌ Disabled |
| **Debug Mode** | ❌ Off | ❌ Off |
| **Log Level** | `info` | `warning` |
| **Mailpit** | ✅ Available | ❌ Use real email |
| **Monitoring** | ⚠️ Optional | ✅ Required |
| **Purpose** | Testing & QA | Live users |

### Why This Distinction?

**Staging**: You want a complete replica to test everything together - frontend, backend, APIs, integrations. Nginx serves all frontend apps on staging subdomains.

**Production**: Frontend apps have different scaling needs and are typically deployed on:
- **Vercel/Netlify**: Automatic scaling, edge caching, preview deployments
- **Cloudflare Pages**: Global CDN, fast worldwide access
- **Mobile Apps**: App Store / Google Play (can't be on your VPS anyway)

Your VPS focuses on what it does best: running the backend services, APIs, and databases.

---

## Deployment Commands

### Staging (Full Stack)

```bash
# Initialize staging
nself staging init staging.example.com --email admin@example.com

# Configure server
# Edit .environments/staging/server.json

# Generate secrets
nself staging secrets generate

# Deploy everything
nself staging deploy

# What gets deployed:
#   ✓ Core Services (PostgreSQL, Hasura, Auth, Nginx)
#   ✓ Optional Services (based on *_ENABLED)
#   ✓ Monitoring Bundle (if enabled)
#   ✓ Custom Services (CS_1, CS_2, ...)
#   ✓ Frontend Apps (FRONTEND_APP_1, FRONTEND_APP_2, ...)
```

### Production (Backend Only)

```bash
# Initialize production
nself prod init example.com --email admin@example.com

# Configure server
# Edit .environments/prod/server.json

# Generate secrets
nself prod secrets generate

# Security audit
nself prod check

# Deploy backend only
nself deploy prod

# What gets deployed:
#   ✓ Core Services (PostgreSQL, Hasura, Auth, Nginx)
#   ✓ Optional Services (based on *_ENABLED)
#   ✓ Monitoring Bundle (if enabled)
#   ✓ Custom Services (CS_1, CS_2, ...)
#   ○ Frontend Apps (excluded - deploy to Vercel/CDN)
```

### Override Default Behavior

```bash
# Force include frontends in production (unusual)
nself deploy prod --include-frontends

# Exclude frontends in staging (e.g., testing backend only)
nself staging deploy --exclude-frontends
```

---

## Hasura Integration

### Remote Schemas (Multi-App Endpoints)

nself can serve multiple applications from one deployment using Remote Schemas - different Hasura GraphQL endpoints with different exposed schemas, all hitting the same database:

```
┌─────────────────────────────────────────────────────────────┐
│                    nself Deployment                          │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐  │
│  │ api.app1.com │    │ api.app2.com │    │  api.main    │  │
│  │  (Remote 1)  │    │  (Remote 2)  │    │  (Default)   │  │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘  │
│         │                   │                   │           │
│         └───────────────────┼───────────────────┘           │
│                             ▼                               │
│                    ┌─────────────────┐                      │
│                    │    PostgreSQL   │                      │
│                    │  (Same Database)│                      │
│                    └─────────────────┘                      │
└─────────────────────────────────────────────────────────────┘
```

**Use case**: SaaS platform serving multiple white-label apps from one backend.

### Custom Services Integration

Custom Services (CS_N) are **separate backend applications** that can integrate with Hasura:

#### Hasura Actions

Your Custom Service handles business logic called from GraphQL:

```bash
# Custom service for complex operations
CS_1=order-service:nestjs:8001
```

In Hasura Console → Actions:
- Handler URL: `http://order-service:8001/validate-and-process`
- Expose as: `processOrder` mutation
- Hasura handles auth, your service handles logic

#### Hasura Event Triggers

Your Custom Service reacts to database changes:

```bash
# Worker service for async processing
CS_2=notification-worker:express-js:8002
```

In Hasura Console → Events:
- Webhook URL: `http://notification-worker:8002/on-order-created`
- Trigger on: `INSERT` into `orders` table
- Your service sends emails, updates analytics, etc.

#### Standalone APIs

Custom Services can also be completely independent:

```bash
# Telemetry API (no Hasura integration)
CS_3=telemetry:express-js:8003

# ML inference (no Hasura integration)
CS_4=ml-api:fastapi-py:8004
```

These run alongside nself but handle their own routing and logic.

---

## Example Project Structure

```
my-project/
├── .environments/
│   ├── staging/
│   │   ├── .env              # Staging config
│   │   ├── .env.secrets      # Staging secrets
│   │   └── server.json       # Staging VPS SSH config
│   └── prod/
│       ├── .env              # Production config
│       ├── .env.secrets      # Production secrets
│       └── server.json       # Production VPS SSH config
├── .env.dev                   # Local development config
├── docker-compose.yml         # Generated by nself build
├── nginx/                     # Generated nginx configs
├── services/                  # Generated custom services
│   ├── payment_api/          # CS_1
│   ├── event_worker/         # CS_2
│   └── ml_inference/         # CS_3
└── frontend/                  # Your frontend apps (not in Docker)
    ├── web/                   # Next.js app → Vercel in prod
    └── mobile/                # React Native → App stores
```

---

## Best Practices

1. **Staging mirrors production config** - Same services, same structure
2. **Test everything in staging** - Including frontend integrations
3. **Production is backend-focused** - Let specialized platforms handle frontends
4. **Use Hasura for API** - Custom services extend, don't replace it
5. **Monitor everything** - Enable monitoring bundle in staging and production
6. **Secure secrets** - Different secrets per environment, never commit them

---

## Related Documentation

- [nself env](../commands/ENV.md) - Environment management
- [nself staging](../commands/STAGING.md) - Staging commands
- [nself prod](../commands/PROD.md) - Production commands
- [nself deploy](../commands/DEPLOY.md) - Deployment commands
- [Custom Services](../services/CUSTOM-SERVICES.md) - CS_N configuration
