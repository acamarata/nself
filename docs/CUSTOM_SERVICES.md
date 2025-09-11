# Custom Services Configuration Guide

## Overview
nself supports 40+ service templates across multiple languages and frameworks. You can define custom backend services that are automatically containerized and integrated into your stack.

## Service Definition Format

### New Enhanced Format (Recommended)
Define services with specific frameworks using the format: `name:framework[:port]`

```bash
# In your .env file:
SERVICES_ENABLED=true

# Define services with framework selection
SERVICES="api:express-ts:3000,worker:bullmq-ts,ml:fastapi:8000,analytics:go:fiber"

# Or use individual language sections with framework specs
JS_SERVICES="api:express-ts:3000,worker:bullmq-ts,websocket:socketio-ts"
PY_SERVICES="ml:fastapi:8000,nlp:agent-llm,vision:agent-vision"
GO_SERVICES="metrics:fiber:3200,grpc-api:grpc:50051"
```

### Current Format (Still Supported)
```bash
SERVICES_ENABLED=true
NESTJS_ENABLED=true
NESTJS_SERVICES=api,worker
GOLANG_ENABLED=true
GOLANG_SERVICES=analytics
PYTHON_ENABLED=true
PYTHON_SERVICES=ml,data
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
SERVICES_ENABLED=true
SERVICES="api:nest-ts:3000,auth:express-ts:3001,ml:fastapi:8000,metrics:fiber:3200"
```

### Example 2: AI-Powered Application
```bash
SERVICES_ENABLED=true
PY_SERVICES="llm:agent-llm:8001,vision:agent-vision:8002,analytics:agent-analytics:8003"
JS_SERVICES="frontend-api:trpc:3000,realtime:socketio-ts:3001"
```

### Example 3: Event-Driven Architecture
```bash
SERVICES_ENABLED=true
SERVICES="api:fastify-ts:3000,worker:bullmq-ts,events:temporal-ts:3002,analytics:ray"
```

### Example 4: Traditional Microservices
```bash
SERVICES_ENABLED=true
JS_SERVICES="api:express-ts:3000,admin:express-ts:3001"
GO_SERVICES="auth:gin:3100,billing:echo:3101"
PY_SERVICES="reports:django-rest:8000"
```

## Port Assignment
- Automatic: Ports are assigned starting from language defaults
  - JavaScript/TypeScript: 3000+
  - Python: 8000+
  - Go: 3100+
  - Others: 4000+
- Manual: Specify port in definition `service:framework:port`

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

## Migration from Old Format
```bash
# Old format:
NESTJS_ENABLED=true
NESTJS_SERVICES=api,worker

# New format (automatic framework selection):
SERVICES="api:nest-ts,worker:bullmq-ts"

# Or specify exactly what you want:
SERVICES="api:express-ts:3000,worker:temporal-ts:3001"
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