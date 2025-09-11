# Custom Services Configuration Guide

## Overview
nself supports 40+ service templates across multiple languages and frameworks. You can define custom backend services that are automatically containerized and integrated into your stack.

## Service Definition Format (CS_N)

Define custom services using the CS_N pattern where N is an incremental number (1, 2, 3, etc.).

### Format
```bash
CS_N=name:framework[:port][:route]
```

### Examples
```bash
# Basic services
CS_1=api:express-ts              # API service using Express + TypeScript
CS_2=worker:bullmq-ts            # Background worker using BullMQ

# With port specified
CS_3=ml:fastapi:8000             # ML service on port 8000
CS_4=analytics:fiber:8001        # Analytics service on port 8001

# With port and route
CS_5=chat:socketio-ts:3001:ws    # WebSocket service at ws.domain.com
CS_6=api:express-ts:3000:api     # API service at api.domain.com

# Additional configuration for each service
CS_1_MEMORY=512M                 # Memory limit
CS_1_CPU=0.5                    # CPU cores
CS_1_PUBLIC=true                # Expose via nginx
CS_1_HEALTHCHECK=/health        # Health endpoint
CS_1_TABLE_PREFIX=api_          # Database table prefix
CS_1_REDIS_PREFIX=api:          # Redis key prefix
CS_1_RATE_LIMIT=100             # Requests per minute
```

## Available Templates

### JavaScript/TypeScript (18 templates)
| Framework | Template ID | Description | Best For |
|-----------|------------|-------------|----------|
| Express.js | `express-js` | Classic Node.js framework | REST APIs |
| Express + TypeScript | `express-ts` | Type-safe Express | Enterprise APIs |
| Fastify | `fastify-js` / `fastify-ts` | High-performance framework | High-throughput APIs |
| Hono | `hono-js` / `hono-ts` | Ultrafast web framework | Edge computing |
| NestJS | `nest-js` / `nest-ts` | Enterprise Node.js framework | Microservices |
| Socket.io | `socketio-js` / `socketio-ts` | Real-time bidirectional | Chat, notifications |
| Temporal | `temporal-js` / `temporal-ts` | Workflow orchestration | Complex workflows |
| tRPC | `trpc` | End-to-end typesafe APIs | TypeScript monorepos |
| BullMQ | `bullmq-js` / `bullmq-ts` | Redis-based queue | Background jobs |
| Bun | `bun` | All-in-one JavaScript runtime | Fast startups |
| Deno | `deno` | Secure TypeScript runtime | Secure services |
| Node.js | `node-js` / `node-ts` | Vanilla Node.js | Custom implementations |

### Python (10 templates)
| Framework | Template ID | Description | Best For |
|-----------|------------|-------------|----------|
| FastAPI | `fastapi` | Modern async framework | REST/GraphQL APIs |
| Flask | `flask` | Lightweight framework | Simple services |
| Django REST | `django-rest` | Batteries-included | Full-featured APIs |
| Celery | `celery` | Distributed task queue | Background jobs |
| Ray | `ray` | Distributed computing | ML workloads |
| Agent LLM | `agent-llm` | LLM integration service | AI chatbots |
| Agent Vision | `agent-vision` | Computer vision service | Image processing |
| Agent Analytics | `agent-analytics` | Data analytics service | Data pipelines |
| Agent Training | `agent-training` | ML training service | Model training |
| Agent TimeSeries | `agent-timeseries` | Time series analysis | Forecasting |

### Go (4 templates)
| Framework | Template ID | Description | Best For |
|-----------|------------|-------------|----------|
| Gin | `gin` | Fast HTTP framework | REST APIs |
| Echo | `echo` | High performance framework | Microservices |
| Fiber | `fiber` | Express-inspired framework | Web apps |
| gRPC | `grpc` | RPC framework | Service mesh |

### Other Languages
| Language | Framework | Template ID | Description |
|----------|-----------|------------|-------------|
| Ruby | Rails | `rails` | Full-stack framework |
| Ruby | Sinatra | `sinatra` | Lightweight framework |
| Rust | Actix Web | `actix-web` | Powerful actor framework |
| Java | Spring Boot | `spring-boot` | Enterprise framework |
| C# | ASP.NET Core | `aspnet` | Microsoft framework |
| PHP | Laravel | `laravel` | Elegant PHP framework |
| Elixir | Phoenix | `phoenix` | Productive framework |
| Kotlin | Ktor | `ktor` | Asynchronous framework |
| Swift | Vapor | `vapor` | Server-side Swift |
| C++ | Oat++ | `oatpp` | Modern web framework |
| Lua | Lapis | `lapis` | OpenResty framework |
| Zig | Zap | `zap` | Blazingly fast framework |

## Examples

### Example 1: Multi-Language Microservices
```bash
# API and worker services in different languages
CS_1=api:nest-ts:3000:api
CS_2=auth:express-ts:3001:auth
CS_3=ml:fastapi:8000:ml
CS_4=metrics:fiber:3200:metrics
```

### Example 2: AI-Powered Application
```bash
# AI services with Python, frontend with TypeScript
CS_1=llm:agent-llm:8001
CS_2=vision:agent-vision:8002
CS_3=analytics:agent-analytics:8003
CS_4=frontend-api:trpc:3000
CS_5=realtime:socketio-ts:3001:ws
```

### Example 3: Event-Driven Architecture
```bash
# Event-driven services with queues and workers
CS_1=api:fastify-ts:3000
CS_2=worker:bullmq-ts
CS_3=events:temporal-ts:3002
CS_4=analytics:ray:8000

# Configure the worker
CS_2_PUBLIC=false
CS_2_MEMORY=1G
```

### Example 4: Production Configuration
```bash
# Production-ready configuration with all settings
CS_1=api:express-ts:3000:api
CS_1_MEMORY=2G
CS_1_CPU=1.0
CS_1_REPLICAS=3
CS_1_PUBLIC=true
CS_1_HEALTHCHECK=/health
CS_1_TABLE_PREFIX=api_
CS_1_REDIS_PREFIX=api:
CS_1_RATE_LIMIT=1000
CS_1_DEV_DOMAIN=api.local.nself.org
CS_1_PROD_DOMAIN=api.production.com

CS_2=worker:bullmq-ts
CS_2_PUBLIC=false
CS_2_MEMORY=1G
CS_2_REDIS_PREFIX=queue:
```

## Port Assignment
- Automatic: If not specified, ports are assigned as 8000 + service number
  - CS_1: port 8001
  - CS_2: port 8002
  - CS_3: port 8003
- Manual: Specify port in definition `CS_N=name:framework:port`
- Best practice: Always specify ports explicitly for production

## Generated Structure
Each service gets:
```
services/
├── <language>/
│   └── <service-name>/
│       ├── Dockerfile
│       ├── package.json / requirements.txt / go.mod / etc.
│       ├── src/
│       │   └── [framework-specific structure]
│       └── [framework-specific config files]
```

## Environment Variables
Each service automatically receives:
- `DATABASE_URL` - PostgreSQL connection
- `HASURA_ENDPOINT` - GraphQL endpoint
- `HASURA_ADMIN_SECRET` - Admin secret
- `REDIS_HOST/PORT` - If Redis enabled
- `S3_*` - Storage configuration
- Service-specific ports and routes

## Deployment
All services are:
- Automatically containerized with optimized Dockerfiles
- Added to docker-compose.yml
- Connected to the internal network
- Configured with health checks
- Exposed through Nginx reverse proxy (if PUBLIC=true)

## Migration from Legacy Formats

If you were using older formats, migrate to CS_N:

```bash
# Old format:
NESTJS_ENABLED=true
NESTJS_SERVICES=api,worker
GOLANG_ENABLED=true
GOLANG_SERVICES=analytics

# New CS_N format:
CS_1=api:nest-ts:3000
CS_2=worker:nest-ts:3001
CS_3=analytics:fiber:3100

# Or with different frameworks:
CS_1=api:express-ts:3000
CS_2=worker:bullmq-ts
CS_3=analytics:gin:3100
```

## Tips
1. Choose frameworks based on your needs:
   - REST API: express-ts, fastify-ts, nest-ts
   - Real-time: socketio-ts, grpc
   - Background jobs: bullmq-ts, celery, temporal-ts
   - AI/ML: agent-*, ray, fastapi
   
2. Mix languages for best tool for each job:
   - Node.js for APIs
   - Python for ML/AI
   - Go for high-performance services
   - Rust for system-critical components

3. Use specific ports to avoid conflicts:
   - Always specify ports for production
   - Let auto-assignment handle development

4. Group related services:
   - Use prefixes: auth-api, auth-worker, auth-admin
   - Makes it easier to manage related services