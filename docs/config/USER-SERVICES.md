# User Services Configuration (Custom Backend Services)

User services are custom Docker containers that run as part of your nself backend stack. These are YOUR microservices, APIs, workers, and other backend components.

## Table of Contents
- [Overview](#overview)
- [Basic Configuration](#basic-configuration)
- [Service Definition Format](#service-definition-format)
- [Advanced Configuration](#advanced-configuration)
- [Service Templates](#service-templates)
- [Networking and Communication](#networking-and-communication)
- [Examples](#examples)
- [Best Practices](#best-practices)

---

## Overview

User services in nself:
- Run as Docker containers within your project
- Have full access to all core services (PostgreSQL, Redis, etc.)
- Can be exposed publicly via nginx or kept internal
- Support any language or framework
- Are configured via `CS_N` environment variables

## Basic Configuration

### Enable User Services

```bash
# Enable custom services
SERVICES_ENABLED=true
# Default: false
```

### Service Definition Format

```bash
# Format: CS_N=name:framework[:port][:route_or_domain]
CS_1=api:fastapi:3001:api.myapp.com
```

Where:
- **`N`** = Service number (1-99)
- **`name`** = Service name (alphanumeric + hyphens)
- **`framework`** = Template to use or "custom"
- **`port`** = Port number (optional, auto-assigns 8001+ if not set)
- **`route_or_domain`** = Route prefix or full domain (optional)

### Simple Examples

```bash
# Minimal - just name and framework
CS_1=myapi:custom
# Creates: myapi.local.nself.org on port 8001

# With specific port
CS_2=worker:python:3002
# Creates: worker service on port 3002 (not publicly exposed by default)

# With subdomain
CS_3=api:node-ts:3003:api
# Creates: api.local.nself.org on port 3003

# With full custom domain
CS_4=backend:fastapi:3004:api.mycompany.com
# Creates: service accessible at api.mycompany.com
```

---

## Advanced Configuration

Each service can have additional configuration via `CS_N_*` variables:

### Resource Limits

```bash
CS_1=api:fastapi:3001:api

# Memory limit
CS_1_MEMORY=512M
# Default: 256M

# CPU limit
CS_1_CPU=1.0
# Default: 0.25

# Replicas (for scaling)
CS_1_REPLICAS=3
# Default: 1
```

### Networking

```bash
# Public exposure via nginx
CS_1_PUBLIC=true
# Default: true

# Custom domain for development
CS_1_DEV_DOMAIN=api.local.nself.org
# Default: uses route parameter

# Custom domain for production
CS_1_PROD_DOMAIN=api.production.com
# Default: uses route parameter

# Health check endpoint
CS_1_HEALTHCHECK=/health
# Default: /health

# Rate limiting (requests per minute)
CS_1_RATE_LIMIT=100
# Default: none
```

### Database Configuration

```bash
# Table prefix (for multi-tenant or service isolation)
CS_1_TABLE_PREFIX=api_
# Default: none
# Results in tables like: api_users, api_products, etc.

# Database name (if using separate database)
CS_1_DATABASE=api_db
# Default: uses main database
```

### Redis Configuration

```bash
# Redis key prefix
CS_1_REDIS_PREFIX=api:
# Default: none
# Results in keys like: api:sessions:123, api:cache:data

# Redis database number (0-15)
CS_1_REDIS_DB=1
# Default: 0
```

### Environment Variables

```bash
# Additional environment variables
CS_1_ENV=KEY1=value1,KEY2=value2,SECRET_KEY=${SECRET_KEY}
# Default: none

# Example with multiple vars
CS_1_ENV=NODE_ENV=production,LOG_LEVEL=info,API_KEY=${API_KEY},FEATURE_FLAG=true
```

---

## Service Templates

nself provides 40+ pre-configured templates:

### JavaScript/TypeScript Templates

```bash
# Node.js
CS_1=api:node-js:3001      # JavaScript
CS_2=api:node-ts:3002      # TypeScript

# Express
CS_3=api:express-js:3003   # JavaScript
CS_4=api:express-ts:3004   # TypeScript

# Fastify
CS_5=api:fastify-js:3005   # JavaScript
CS_6=api:fastify-ts:3006   # TypeScript

# NestJS
CS_7=api:nest-js:3007      # JavaScript
CS_8=api:nest-ts:3008      # TypeScript

# Other frameworks
CS_9=api:hono-js:3009      # Hono (JS)
CS_10=api:hono-ts:3010     # Hono (TS)
CS_11=ws:socketio-js:3011  # Socket.IO (JS)
CS_12=ws:socketio-ts:3012  # Socket.IO (TS)
CS_13=queue:bullmq-js:3013 # BullMQ (JS)
CS_14=queue:bullmq-ts:3014 # BullMQ (TS)
CS_15=workflow:temporal-js:3015  # Temporal (JS)
CS_16=workflow:temporal-ts:3016  # Temporal (TS)

# Runtime-specific
CS_17=api:bun:3017         # Bun runtime
CS_18=api:deno:3018        # Deno runtime
CS_19=api:trpc:3019        # tRPC (TypeScript only)
```

### Python Templates

```bash
CS_20=api:flask:3020           # Flask
CS_21=api:fastapi:3021         # FastAPI
CS_22=api:django-rest:3022     # Django REST
CS_23=worker:celery:3023       # Celery workers
CS_24=ml:ray:3024              # Ray distributed computing
CS_25=ai:agent-llm:3025        # LLM agent service
CS_26=data:agent-data:3026     # Data processing agent
```

### Go Templates

```bash
CS_27=api:gin:3027         # Gin framework
CS_28=api:echo:3028        # Echo framework
CS_29=api:fiber:3029       # Fiber framework
CS_30=rpc:grpc:3030        # gRPC service
```

### Other Languages

```bash
CS_31=api:rust:3031        # Rust (Actix-Web)
CS_32=api:ruby:3032        # Ruby on Rails
CS_33=api:sinatra:3033     # Ruby Sinatra
CS_34=api:php:3034         # PHP Laravel
CS_35=api:java:3035        # Java Spring Boot
CS_36=api:csharp:3036      # C# ASP.NET
CS_37=api:elixir:3037      # Elixir Phoenix
CS_38=api:kotlin:3038      # Kotlin Ktor
CS_39=api:swift:3039       # Swift Vapor
CS_40=api:cpp:3040         # C++ Oat++
```

### Custom Template

```bash
CS_41=myservice:custom:3041
# Uses the blank custom template - you provide everything
```

---

## Networking and Communication

### Internal Communication

Services can communicate with each other and core services using Docker networking:

```javascript
// From any user service, call another user service
const response = await fetch('http://api:3001/endpoint');

// Call PostgreSQL
const pg = new Pool({
  host: 'postgres',
  port: 5432,
  database: process.env.POSTGRES_DB
});

// Call Redis
const redis = new Redis('redis://redis:6379');

// Call Hasura GraphQL
const graphql = await fetch('http://hasura:8080/v1/graphql', {
  headers: {
    'x-hasura-admin-secret': process.env.HASURA_GRAPHQL_ADMIN_SECRET
  }
});

// Call MinIO/S3
const s3 = new AWS.S3({
  endpoint: 'http://minio:9000',
  accessKeyId: process.env.S3_ACCESS_KEY,
  secretAccessKey: process.env.S3_SECRET_KEY
});

// Call Auth service
const auth = await fetch('http://auth:4000/verify', {
  headers: { 'Authorization': `Bearer ${token}` }
});
```

### Environment Variables Available

All user services automatically receive:

```bash
# Core
SERVICE_NAME=<from-CS_N>
PROJECT_NAME=<from-env>
BASE_DOMAIN=<from-env>
PORT=<from-CS_N>
ENV=<development|staging|production>

# Database
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=<project-name>
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<from-env>
DATABASE_URL=postgresql://...

# Redis (if enabled)
REDIS_URL=redis://redis:6379
REDIS_PASSWORD=<from-env>

# Hasura
HASURA_GRAPHQL_URL=http://hasura:8080/v1/graphql
HASURA_GRAPHQL_ADMIN_SECRET=<from-env>

# S3/MinIO (if enabled)
S3_ENDPOINT=http://minio:9000
S3_ACCESS_KEY=<from-env>
S3_SECRET_KEY=<from-env>
S3_BUCKET=<project>-storage

# Auth
AUTH_URL=http://auth:4000
HASURA_AUTH_URL=http://auth:4000
JWT_SECRET=<from-env>

# Custom environment variables from CS_N_ENV
<KEY>=<VALUE>
```

---

## Examples

### Example 1: Simple API Service

```bash
# Basic API service
CS_1=api:fastapi:3001:api
CS_1_MEMORY=512M
CS_1_CPU=1
CS_1_RATE_LIMIT=1000
```

### Example 2: Microservices Architecture

```bash
# Enable services
SERVICES_ENABLED=true

# API Gateway
CS_1=gateway:node-ts:3000:api.myapp.com
CS_1_MEMORY=1G
CS_1_CPU=2
CS_1_REPLICAS=2
CS_1_RATE_LIMIT=5000

# Authentication Service
CS_2=auth:fastapi:3001:auth
CS_2_TABLE_PREFIX=auth_
CS_2_REDIS_PREFIX=auth:
CS_2_ENV=JWT_EXPIRES_IN=3600,REFRESH_EXPIRES_IN=86400

# User Service
CS_3=users:gin:3002:users
CS_3_TABLE_PREFIX=users_
CS_3_REDIS_PREFIX=users:
CS_3_REDIS_DB=1

# Product Service
CS_4=products:java:3003:products
CS_4_TABLE_PREFIX=products_
CS_4_MEMORY=2G
CS_4_ENV=CATALOG_SIZE=10000,ENABLE_CACHE=true

# Order Service
CS_5=orders:nest-ts:3004:orders
CS_5_TABLE_PREFIX=orders_
CS_5_DATABASE=orders_db
CS_5_ENV=PAYMENT_GATEWAY_URL=https://payment.example.com

# Notification Worker (internal, not publicly exposed)
CS_6=notifications:celery:3005
CS_6_PUBLIC=false
CS_6_REDIS_PREFIX=notifications:
CS_6_ENV=EMAIL_PROVIDER=sendgrid,SMS_PROVIDER=twilio

# Analytics Service
CS_7=analytics:ray:3006:analytics
CS_7_MEMORY=4G
CS_7_CPU=4
CS_7_TABLE_PREFIX=analytics_

# Webhook Handler
CS_8=webhooks:express-ts:3007:webhooks.myapp.com
CS_8_RATE_LIMIT=100
CS_8_ENV=WEBHOOK_SECRET=${WEBHOOK_SECRET}

# Background Jobs
CS_9=jobs:bullmq-ts:3008
CS_9_PUBLIC=false
CS_9_REDIS_PREFIX=jobs:
CS_9_REDIS_DB=2

# ML Model Service
CS_10=ml:agent-llm:3009:ai.myapp.com
CS_10_MEMORY=4G
CS_10_CPU=2
CS_10_ENV=OPENAI_API_KEY=${OPENAI_API_KEY},MODEL=gpt-4
```

### Example 3: Multi-Tenant SaaS

```bash
# Tenant-specific services with table isolation
CS_1=tenant1-api:custom:3001:tenant1.myapp.com
CS_1_TABLE_PREFIX=t1_
CS_1_REDIS_PREFIX=t1:
CS_1_DATABASE=tenant1_db

CS_2=tenant2-api:custom:3002:tenant2.myapp.com
CS_2_TABLE_PREFIX=t2_
CS_2_REDIS_PREFIX=t2:
CS_2_DATABASE=tenant2_db

CS_3=tenant3-api:custom:3003:tenant3.myapp.com
CS_3_TABLE_PREFIX=t3_
CS_3_REDIS_PREFIX=t3:
CS_3_DATABASE=tenant3_db

# Shared services
CS_4=billing:stripe-webhook:3004
CS_4_PUBLIC=false
CS_4_ENV=STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}

CS_5=admin:nest-ts:3005:admin.myapp.com
CS_5_ENV=ADMIN_SECRET=${ADMIN_SECRET}
```

### Example 4: Event-Driven Architecture

```bash
# Event producers
CS_1=api:fastapi:3001:api
CS_1_ENV=EVENT_BUS=redis://redis:6379

# Event consumers
CS_2=order-processor:node-ts:3002
CS_2_PUBLIC=false
CS_2_ENV=SUBSCRIBE_TO=order.created,order.updated

CS_3=inventory-updater:python:3003
CS_3_PUBLIC=false
CS_3_ENV=SUBSCRIBE_TO=order.completed

CS_4=email-sender:go:3004
CS_4_PUBLIC=false
CS_4_ENV=SUBSCRIBE_TO=user.registered,order.shipped

# Event store
CS_5=event-store:custom:3005
CS_5_TABLE_PREFIX=events_
CS_5_ENV=RETENTION_DAYS=90
```

### Example 5: Development vs Production

```bash
# Base configuration
CS_1=api:node-ts:3001
CS_2=worker:python:3002

# Development overrides
CS_1_DEV_DOMAIN=api.local.nself.org
CS_1_ENV=NODE_ENV=development,LOG_LEVEL=debug,MOCK_DATA=true
CS_2_DEV_DOMAIN=worker.local.nself.org

# Production overrides
CS_1_PROD_DOMAIN=api.production.com
CS_1_ENV=NODE_ENV=production,LOG_LEVEL=warn,ENABLE_METRICS=true
CS_1_MEMORY=4G
CS_1_CPU=2
CS_1_REPLICAS=5

CS_2_PROD_DOMAIN=worker.production.com
CS_2_MEMORY=2G
CS_2_REPLICAS=3
```

---

## Directory Structure

After running `nself build`, your services are created at:

```
services/
├── api/              # CS_1
│   ├── Dockerfile
│   ├── README.md
│   └── [your code]
├── auth/             # CS_2
│   ├── Dockerfile
│   ├── README.md
│   └── [your code]
├── users/            # CS_3
│   ├── Dockerfile
│   ├── README.md
│   └── [your code]
└── ...
```

---

## Best Practices

### 1. Service Design

```bash
# Good: Focused, single-purpose services
CS_1=user-api:fastapi:3001:users
CS_2=auth-api:node-ts:3002:auth
CS_3=payment-api:java:3003:payments

# Bad: Monolithic service
CS_1=everything:custom:3001:api
```

### 2. Resource Allocation

```bash
# Allocate resources based on load
CS_1=api:fastapi:3001
CS_1_MEMORY=2G      # High-traffic API
CS_1_CPU=2
CS_1_REPLICAS=3

CS_2=worker:python:3002
CS_2_MEMORY=512M    # Background worker
CS_2_CPU=0.5
CS_2_PUBLIC=false   # No public access needed
```

### 3. Database Isolation

```bash
# Use table prefixes for service isolation
CS_1=orders:node-ts:3001
CS_1_TABLE_PREFIX=orders_

CS_2=inventory:python:3002
CS_2_TABLE_PREFIX=inventory_

# Or use separate databases for complete isolation
CS_3=analytics:ray:3003
CS_3_DATABASE=analytics_db
```

### 4. Security

```bash
# Keep internal services private
CS_1=public-api:fastapi:3001:api.myapp.com
CS_1_PUBLIC=true

CS_2=internal-worker:celery:3002
CS_2_PUBLIC=false  # Not exposed to internet

# Use environment variables for secrets
CS_3=payment:node-ts:3003
CS_3_ENV=STRIPE_KEY=${STRIPE_SECRET_KEY},WEBHOOK_SECRET=${WEBHOOK_SECRET}
```

### 5. Health Checks

Always implement health endpoints:

```python
# FastAPI example
@app.get("/health")
async def health():
    return {
        "status": "healthy",
        "service": os.getenv("SERVICE_NAME"),
        "timestamp": datetime.utcnow().isoformat()
    }
```

### 6. Logging

Use structured logging to stdout/stderr:

```javascript
// Node.js example
console.log(JSON.stringify({
  level: 'info',
  service: process.env.SERVICE_NAME,
  message: 'Request processed',
  requestId: req.id,
  duration: Date.now() - start
}));
```

### 7. Rate Limiting

Protect public APIs:

```bash
CS_1=public-api:fastapi:3001:api
CS_1_RATE_LIMIT=100  # 100 requests per minute per IP
```

### 8. Domain Strategy

```bash
# Development: Use subdomains
CS_1=api:fastapi:3001:api     # → api.local.nself.org

# Production: Use custom domains
CS_1_PROD_DOMAIN=api.mycompany.com
```

---

## Lifecycle Management

### Building Services

```bash
# After adding CS_N variables to .env.local
nself build

# This will:
# 1. Parse all CS_N definitions
# 2. Create service directories
# 3. Generate Dockerfiles from templates
# 4. Create docker-compose configurations
# 5. Configure nginx routing
```

### Starting Services

```bash
# Start all services
nself start

# Start specific service
nself start api
```

### Monitoring Services

```bash
# Check status
nself status

# View logs
nself logs api
nself logs worker -f  # Follow logs

# Check health
curl http://api.local.nself.org/health
```

### Updating Services

```bash
# After code changes
nself restart api

# After configuration changes
nself build
nself restart api
```

### Scaling Services

```bash
# Update replica count
CS_1_REPLICAS=5

# Apply changes
nself build
nself restart api
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check logs
nself logs api

# Common issues:
# - Port already in use (change CS_N port)
# - Missing environment variables
# - Dockerfile errors
# - Dependency services not running
```

### Can't Access Service

```bash
# Check if service is running
nself status

# Check nginx configuration
nself logs nginx

# Verify domain resolves
nslookup api.local.nself.org

# Check if publicly exposed
# CS_N_PUBLIC=true
```

### Database Connection Issues

```bash
# Verify PostgreSQL is running
nself status postgres

# Check connection from service
nself exec api
> psql -h postgres -U postgres -d ${POSTGRES_DB}
```

### Performance Issues

```bash
# Increase resources
CS_1_MEMORY=2G
CS_1_CPU=2
CS_1_REPLICAS=3

# Enable caching with Redis
CS_1_REDIS_PREFIX=api:

# Add rate limiting
CS_1_RATE_LIMIT=100
```

---

## Next Steps

- [Frontend Apps](./FRONTEND-APPS.md) - Configure frontend applications
- [How-To Guides](./HOW-TO.md) - Common scenarios and examples
- [Environment Reference](./ENV-REFERENCE.md) - Complete variable list
- [Service Templates](../templates/services/) - Browse available templates