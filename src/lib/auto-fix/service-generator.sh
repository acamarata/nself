#!/usr/bin/env bash

# service-generator.sh - Auto-generate missing service templates

# Source utilities - don't override parent SCRIPT_DIR
SERVICE_GEN_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SERVICE_GEN_SCRIPT_DIR/../utils/display.sh" 2>/dev/null || true

# Generate a NestJS hello world service
generate_nest_service() {
    local service_name="$1"
    local service_path="services/nest/${service_name}"
    
    # Silent generation, no log output
    
    # Create directory structure
    mkdir -p "$service_path/src"
    
    # Create package.json
    cat > "$service_path/package.json" << 'EOF'
{
  "name": "SERVICE_NAME",
  "version": "1.0.0",
  "description": "NestJS microservice",
  "scripts": {
    "dev": "nest start --watch",
    "build": "nest build",
    "start": "node dist/main"
  },
  "dependencies": {
    "@nestjs/common": "^10.0.0",
    "@nestjs/core": "^10.0.0",
    "@nestjs/platform-express": "^10.0.0",
    "reflect-metadata": "^0.1.13",
    "rxjs": "^7.8.1"
  },
  "devDependencies": {
    "@nestjs/cli": "^10.0.0",
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0"
  }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/package.json" && rm -f "$service_path/package.json.bak"
    
    # Create main.ts
    cat > "$service_path/src/main.ts" << 'EOF'
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  const port = process.env.PORT || 3000;
  await app.listen(port, '0.0.0.0');
  console.log(`Service is running on port ${port}`);
}
bootstrap();
EOF
    
    # Create app.module.ts
    cat > "$service_path/src/app.module.ts" << 'EOF'
import { Module } from '@nestjs/common';
import { AppController } from './app.controller';

@Module({
  imports: [],
  controllers: [AppController],
})
export class AppModule {}
EOF
    
    # Create app.controller.ts
    cat > "$service_path/src/app.controller.ts" << 'EOF'
import { Controller, Get } from '@nestjs/common';

@Controller()
export class AppController {
  @Get()
  getHello(): string {
    return 'Hello from SERVICE_NAME service!';
  }

  @Get('health')
  getHealth(): object {
    return { status: 'ok', service: 'SERVICE_NAME' };
  }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/src/app.controller.ts" && rm -f "$service_path/src/app.controller.ts.bak"
    
    # Create nest-cli.json
    cat > "$service_path/nest-cli.json" << 'EOF'
{
  "$schema": "https://json.schemastore.org/nest-cli",
  "collection": "@nestjs/schematics",
  "sourceRoot": "src",
  "compilerOptions": {
    "deleteOutDir": true
  }
}
EOF
    
    # Create tsconfig.json
    cat > "$service_path/tsconfig.json" << 'EOF'
{
  "compilerOptions": {
    "module": "commonjs",
    "declaration": true,
    "removeComments": true,
    "emitDecoratorMetadata": true,
    "experimentalDecorators": true,
    "target": "es2021",
    "sourceMap": true,
    "outDir": "./dist",
    "baseUrl": "./",
    "incremental": true,
    "skipLibCheck": true,
    "strictNullChecks": false,
    "noImplicitAny": false,
    "strictBindCallApply": false,
    "forceConsistentCasingInFileNames": false,
    "noFallthroughCasesInSwitch": false
  }
}
EOF
    
    # Create Dockerfile
    cat > "$service_path/Dockerfile" << 'EOF'
FROM node:18-alpine AS development
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM node:18-alpine AS production
# Install health check tools
RUN apk add --no-cache curl wget
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY --from=development /app/dist ./dist
# Use PORT from build arg and environment
ARG PORT=3000
ENV PORT=${PORT}
EXPOSE ${PORT}
CMD ["node", "dist/main"]
EOF
}

# Generate a Bull queue worker service
generate_bull_service() {
    local service_name="$1"
    local service_path="services/bull/${service_name}"
    
    
    # Create directory structure
    mkdir -p "$service_path/src"
    
    # Create package.json
    cat > "$service_path/package.json" << 'EOF'
{
  "name": "SERVICE_NAME",
  "version": "1.0.0",
  "description": "BullMQ queue worker",
  "scripts": {
    "dev": "nodemon src/index.js",
    "start": "node src/index.js"
  },
  "dependencies": {
    "bullmq": "^4.12.0",
    "ioredis": "^5.3.0"
  },
  "devDependencies": {
    "nodemon": "^3.0.0"
  }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/package.json" && rm -f "$service_path/package.json.bak"
    
    # Create index.js
    cat > "$service_path/src/index.js" << 'EOF'
const { Queue, Worker } = require('bullmq');

const queueName = 'SERVICE_NAME';
const connection = {
  host: process.env.REDIS_HOST || 'redis',
  port: process.env.REDIS_PORT || 6379,
};

// Create queue
const queue = new Queue(queueName, { connection });

// Create worker
const worker = new Worker(
  queueName,
  async (job) => {
    console.log(`Processing job ${job.id}:`, job.data);
    // Add your job processing logic here
    return { processed: true, timestamp: new Date() };
  },
  { connection }
);

// Event listeners
worker.on('completed', (job, result) => {
  console.log(`Job ${job.id} completed:`, result);
});

worker.on('failed', (job, err) => {
  console.error(`Job ${job.id} failed:`, err.message);
});

console.log(`SERVICE_NAME worker started, waiting for jobs...`);

// Graceful shutdown
process.on('SIGTERM', async () => {
  await worker.close();
  process.exit(0);
});
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/src/index.js" && rm -f "$service_path/src/index.js.bak"
    
    # Create Dockerfile
    cat > "$service_path/Dockerfile" << 'EOF'
FROM node:18-alpine
# Install health check tools
RUN apk add --no-cache curl wget
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
# BullMQ dashboard port (dynamically set via environment)
ARG DASHBOARD_PORT=4200
ENV DASHBOARD_PORT=${DASHBOARD_PORT}
EXPOSE ${DASHBOARD_PORT}
CMD ["node", "src/index.js"]
EOF
    
}

# Generate a Go service
generate_go_service() {
    local service_name="$1"
    local service_path="services/go/${service_name}"
    
    
    # Create directory structure
    mkdir -p "$service_path"
    
    # Create go.mod
    cat > "$service_path/go.mod" << EOF
module ${service_name}

go 1.21

require github.com/gorilla/mux v1.8.0
EOF
    
    # Create go.sum with the mux dependency
    cat > "$service_path/go.sum" << 'EOF'
github.com/gorilla/mux v1.8.0 h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=
github.com/gorilla/mux v1.8.0/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
EOF
    
    # Create main.go
    cat > "$service_path/main.go" << 'EOF'
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/gorilla/mux"
)

type HealthResponse struct {
    Status  string `json:"status"`
    Service string `json:"service"`
}

func main() {
    r := mux.NewRouter()

    r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello from SERVICE_NAME service!")
    })

    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(HealthResponse{
            Status:  "ok",
            Service: "SERVICE_NAME",
        })
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("SERVICE_NAME service starting on port %s", port)
    if err := http.ListenAndServe(":"+port, r); err != nil {
        log.Fatal(err)
    }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/main.go" && rm -f "$service_path/main.go.bak"
    
    # Create Dockerfile
    cat > "$service_path/Dockerfile" << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum* ./
RUN go mod download || go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl wget
WORKDIR /root/
COPY --from=builder /app/main .
# Use PORT from build arg and environment
ARG PORT=8080
ENV PORT=${PORT}
EXPOSE ${PORT}
CMD ["./main"]
EOF
    
}

# Generate a Python service
generate_python_service() {
    local service_name="$1"
    local service_path="services/py/${service_name}"
    
    
    # Create directory structure
    mkdir -p "$service_path"
    
    # Create requirements.txt
    cat > "$service_path/requirements.txt" << 'EOF'
fastapi==0.104.0
uvicorn==0.24.0
python-dotenv==1.0.0
EOF
    
    # Create main.py
    cat > "$service_path/main.py" << 'EOF'
from fastapi import FastAPI
from fastapi.responses import JSONResponse
import os

app = FastAPI(title="SERVICE_NAME")

@app.get("/")
def read_root():
    return {"message": "Hello from SERVICE_NAME service!"}

@app.get("/health")
def health_check():
    return {"status": "ok", "service": "SERVICE_NAME"}

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", 8000))
    uvicorn.run(app, host="0.0.0.0", port=port)
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/main.py" && rm -f "$service_path/main.py.bak"
    
    # Create Dockerfile
    cat > "$service_path/Dockerfile" << 'EOF'
FROM python:3.11-slim
# Install health check tools
RUN apt-get update && apt-get install -y curl wget && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
# Use PORT from build arg and environment
ARG PORT=8000
ENV PORT=${PORT}
EXPOSE ${PORT}
CMD uvicorn main:app --host 0.0.0.0 --port ${PORT}
EOF
    
}

# Auto-detect and generate missing services
auto_generate_services() {
    local generated_count=0
    local silent="${1:-false}"
    
    # Check environment for enabled services
    if [[ -f ".env.local" ]]; then
        source .env.local
    fi
    
    # Check if services are enabled
    if [[ "${SERVICES_ENABLED:-false}" != "true" ]]; then
        return 0
    fi
    
    [[ "$silent" != "true" ]] && log_info "Checking for missing services..."
    
    # Parse NEST_SERVICES or NESTJS_SERVICES (support both)
    local nest_list="${NEST_SERVICES:-${NESTJS_SERVICES:-}}"
    if [[ -n "$nest_list" ]]; then
        IFS=',' read -ra services <<< "$nest_list"
        for service in "${services[@]}"; do
            service=$(echo "$service" | xargs)  # Trim whitespace
            if [[ ! -d "services/nest/$service" ]]; then
                generate_nest_service "$service"
                ((generated_count++))
            fi
        done
    fi
    
    # Parse BULL_SERVICES or BULLMQ_WORKERS (support both)
    local bull_list="${BULL_SERVICES:-${BULLMQ_WORKERS:-}}"
    if [[ -n "$bull_list" ]]; then
        IFS=',' read -ra services <<< "$bull_list"
        for service in "${services[@]}"; do
            service=$(echo "$service" | xargs)
            if [[ ! -d "services/bull/$service" ]]; then
                generate_bull_service "$service"
                ((generated_count++))
            fi
        done
    fi
    
    # Parse GO_SERVICES
    if [[ -n "${GO_SERVICES:-}" ]]; then
        IFS=',' read -ra services <<< "$GO_SERVICES"
        for service in "${services[@]}"; do
            service=$(echo "$service" | xargs)
            if [[ ! -d "services/go/$service" ]]; then
                generate_go_service "$service"
                ((generated_count++))
            fi
        done
    fi
    
    # Parse PYTHON_SERVICES
    if [[ -n "${PYTHON_SERVICES:-}" ]]; then
        IFS=',' read -ra services <<< "$PYTHON_SERVICES"
        for service in "${services[@]}"; do
            service=$(echo "$service" | xargs)
            if [[ ! -d "services/py/$service" ]]; then
                generate_python_service "$service"
                ((generated_count++))
            fi
        done
    fi
    
    # Silent - let caller handle reporting
    return 0
}

# Check if a specific service directory is missing
check_missing_service() {
    local missing_path="$1"
    
    # Extract service type and name from path - handle ALL naming conventions
    if [[ "$missing_path" =~ services/([^/]+)/([^/]+) ]]; then
        local service_type="${BASH_REMATCH[1]}"
        local service_name="${BASH_REMATCH[2]}"
        local actual_dir=""
        
        
        # Create in the exact directory that Docker Compose expects
        actual_dir="services/${service_type}/${service_name}"
        mkdir -p "$actual_dir"
        
        # Map various names to our standard generator functions
        case "$service_type" in
            nest|nestjs)
                generate_nest_service_at "$service_name" "$actual_dir"
                ;;
            bull|bullmq)
                generate_bull_service_at "$service_name" "$actual_dir"
                ;;
            go|golang)
                generate_go_service_at "$service_name" "$actual_dir"
                ;;
            py|python)
                generate_python_service_at "$service_name" "$actual_dir"
                ;;
            *)
                # If we don't recognize the type, still try to generate something reasonable
                # Default to a basic Node.js service
                # Unknown service type, generating basic Node.js service
                generate_basic_node_service_at "$service_name" "$actual_dir"
                ;;
        esac
        
        return 0
    fi
    
    return 1
}

# Generate services at specific paths (for flexibility)
generate_nest_service_at() {
    local service_name="$1"
    local service_path="$2"
    
    # Create directory structure
    mkdir -p "$service_path/src"
    
    # Use the same generation logic but with custom path
    # Create package.json
    cat > "$service_path/package.json" << 'EOF'
{
  "name": "SERVICE_NAME",
  "version": "1.0.0",
  "description": "NestJS microservice",
  "scripts": {
    "dev": "nest start --watch",
    "build": "nest build",
    "start": "node dist/main"
  },
  "dependencies": {
    "@nestjs/common": "^10.0.0",
    "@nestjs/core": "^10.0.0",
    "@nestjs/platform-express": "^10.0.0",
    "reflect-metadata": "^0.1.13",
    "rxjs": "^7.8.1"
  },
  "devDependencies": {
    "@nestjs/cli": "^10.0.0",
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0"
  }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/package.json" && rm -f "$service_path/package.json.bak"
    
    # Rest of the files...
    cat > "$service_path/src/main.ts" << 'EOF'
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  const port = process.env.PORT || 3000;
  await app.listen(port, '0.0.0.0');
  console.log(`Service is running on port ${port}`);
}
bootstrap();
EOF
    
    cat > "$service_path/src/app.module.ts" << 'EOF'
import { Module } from '@nestjs/common';
import { AppController } from './app.controller';

@Module({
  imports: [],
  controllers: [AppController],
})
export class AppModule {}
EOF
    
    cat > "$service_path/src/app.controller.ts" << 'EOF'
import { Controller, Get } from '@nestjs/common';

@Controller()
export class AppController {
  @Get()
  getHello(): string {
    return 'Hello from SERVICE_NAME service!';
  }

  @Get('health')
  getHealth(): object {
    return { status: 'ok', service: 'SERVICE_NAME' };
  }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/src/app.controller.ts" && rm -f "$service_path/src/app.controller.ts.bak"
    
    cat > "$service_path/tsconfig.json" << 'EOF'
{
  "compilerOptions": {
    "module": "commonjs",
    "declaration": true,
    "removeComments": true,
    "emitDecoratorMetadata": true,
    "experimentalDecorators": true,
    "target": "es2021",
    "sourceMap": true,
    "outDir": "./dist",
    "baseUrl": "./",
    "incremental": true,
    "skipLibCheck": true,
    "strictNullChecks": false,
    "noImplicitAny": false,
    "strictBindCallApply": false,
    "forceConsistentCasingInFileNames": false,
    "noFallthroughCasesInSwitch": false
  }
}
EOF
    
    cat > "$service_path/Dockerfile" << 'EOF'
FROM node:18-alpine AS development
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM node:18-alpine AS production
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY --from=development /app/dist ./dist
EXPOSE 3000
CMD ["node", "dist/main"]
EOF
    
    # Successfully generated
}

generate_bull_service_at() {
    local service_name="$1"
    local service_path="$2"
    
    # Create directory structure
    mkdir -p "$service_path/src"
    
    cat > "$service_path/package.json" << 'EOF'
{
  "name": "SERVICE_NAME",
  "version": "1.0.0",
  "description": "Bull queue worker",
  "scripts": {
    "dev": "nodemon src/index.js",
    "start": "node src/index.js"
  },
  "dependencies": {
    "bull": "^4.11.0",
    "bullmq": "^4.12.0",
    "ioredis": "^5.3.0"
  },
  "devDependencies": {
    "nodemon": "^3.0.0"
  }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/package.json" && rm -f "$service_path/package.json.bak"
    
    cat > "$service_path/src/index.js" << 'EOF'
const { Queue, Worker } = require('bullmq');

const queueName = 'SERVICE_NAME';
const connection = {
  host: process.env.REDIS_HOST || 'redis',
  port: process.env.REDIS_PORT || 6379,
};

// Create queue
const queue = new Queue(queueName, { connection });

// Create worker
const worker = new Worker(
  queueName,
  async (job) => {
    console.log(`Processing job ${job.id}:`, job.data);
    // Add your job processing logic here
    return { processed: true, timestamp: new Date() };
  },
  { connection }
);

// Event listeners
worker.on('completed', (job, result) => {
  console.log(`Job ${job.id} completed:`, result);
});

worker.on('failed', (job, err) => {
  console.error(`Job ${job.id} failed:`, err.message);
});

console.log(`SERVICE_NAME worker started, waiting for jobs...`);

// Graceful shutdown
process.on('SIGTERM', async () => {
  await worker.close();
  process.exit(0);
});
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/src/index.js" && rm -f "$service_path/src/index.js.bak"
    
    cat > "$service_path/Dockerfile" << 'EOF'
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
CMD ["node", "src/index.js"]
EOF
    
    # Successfully generated
}

generate_go_service_at() {
    local service_name="$1"
    local service_path="$2"
    
    cat > "$service_path/go.mod" << EOF
module ${service_name}

go 1.21

require github.com/gorilla/mux v1.8.0
EOF
    
    # Create go.sum with the mux dependency
    cat > "$service_path/go.sum" << 'EOF'
github.com/gorilla/mux v1.8.0 h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=
github.com/gorilla/mux v1.8.0/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
EOF
    
    cat > "$service_path/main.go" << 'EOF'
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/gorilla/mux"
)

type HealthResponse struct {
    Status  string `json:"status"`
    Service string `json:"service"`
}

func main() {
    r := mux.NewRouter()

    r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello from SERVICE_NAME service!")
    })

    r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(HealthResponse{
            Status:  "ok",
            Service: "SERVICE_NAME",
        })
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("SERVICE_NAME service starting on port %s", port)
    if err := http.ListenAndServe(":"+port, r); err != nil {
        log.Fatal(err)
    }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/main.go" && rm -f "$service_path/main.go.bak"
    
    cat > "$service_path/Dockerfile" << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum* ./
RUN go mod download || go mod tidy
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
EOF
    
    # Successfully generated
}

generate_basic_node_service_at() {
    local service_name="$1"
    local service_path="$2"
    
    # Create a basic Node.js service
    cat > "$service_path/package.json" << 'EOF'
{
  "name": "SERVICE_NAME",
  "version": "1.0.0",
  "description": "Node.js service",
  "scripts": {
    "start": "node index.js"
  },
  "dependencies": {
    "express": "^4.18.0"
  }
}
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/package.json" && rm -f "$service_path/package.json.bak"
    
    cat > "$service_path/index.js" << 'EOF'
const express = require('express');
const app = express();
const port = process.env.PORT || 3000;

app.get('/', (req, res) => {
  res.json({ message: 'Hello from SERVICE_NAME service!' });
});

app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'SERVICE_NAME' });
});

app.listen(port, () => {
  console.log(`SERVICE_NAME service listening on port ${port}`);
});
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/index.js" && rm -f "$service_path/index.js.bak"
    
    cat > "$service_path/Dockerfile" << 'EOF'
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
EXPOSE 3000
CMD ["node", "index.js"]
EOF
    
    # Successfully generated
}

generate_python_service_at() {
    local service_name="$1"
    local service_path="$2"
    
    cat > "$service_path/requirements.txt" << 'EOF'
fastapi==0.104.0
uvicorn==0.24.0
python-dotenv==1.0.0
EOF
    
    cat > "$service_path/main.py" << 'EOF'
from fastapi import FastAPI
from fastapi.responses import JSONResponse
import os

app = FastAPI(title="SERVICE_NAME")

@app.get("/")
def read_root():
    return {"message": "Hello from SERVICE_NAME service!"}

@app.get("/health")
def health_check():
    return {"status": "ok", "service": "SERVICE_NAME"}

if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", 8000))
    uvicorn.run(app, host="0.0.0.0", port=port)
EOF
    sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/main.py" && rm -f "$service_path/main.py.bak"
    
    cat > "$service_path/Dockerfile" << 'EOF'
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
EOF
    
    # Successfully generated
}

# Export functions
export -f generate_nest_service
export -f generate_bull_service
export -f generate_go_service
export -f generate_python_service
export -f auto_generate_services
export -f check_missing_service