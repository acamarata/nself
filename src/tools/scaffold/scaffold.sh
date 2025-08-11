#!/usr/bin/env bash
# scaffold.sh - Create new services from templates

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../../lib/utils/display.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../../lib/utils/env.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../../lib/utils/progress.sh" 2>/dev/null || true

# Command function
cmd_scaffold() {
    local service_type="${1:-}"
    local service_name="${2:-}"
    local start_after="${3:-}"
    
    # Check arguments
    if [[ -z "$service_type" ]] || [[ -z "$service_name" ]]; then
        show_scaffold_help
        return 1
    fi
    
    # Validate service type
    local valid_types="nest bull go py"
    if [[ ! " $valid_types " =~ " $service_type " ]]; then
        log_error "Invalid service type: $service_type"
        log_info "Valid types: $valid_types"
        return 1
    fi
    
    # Scaffold the service
    scaffold_service "$service_type" "$service_name"
    local result=$?
    
    # Start if requested
    if [[ "$start_after" == "--start" ]] && [[ $result -eq 0 ]]; then
        log_info "Starting the new service..."
        source "$SCRIPT_DIR/up.sh"
        cmd_up
    fi
    
    return $result
}

# Show scaffold help
show_scaffold_help() {
    echo "Usage: nself scaffold <type> <name> [--start]"
    echo
    echo "Service Types:"
    echo "  nest    NestJS service with TypeScript"
    echo "  bull    BullMQ worker service"
    echo "  go      Go service with Gin/Echo"
    echo "  py      Python service with FastAPI"
    echo
    echo "Options:"
    echo "  --start    Start services after scaffolding"
    echo
    echo "Examples:"
    echo "  nself scaffold nest api-gateway"
    echo "  nself scaffold go webhook-handler"
    echo "  nself scaffold py data-processor --start"
}

# Main scaffolding function
scaffold_service() {
    local service_type="$1"
    local service_name="$2"
    
    # Determine paths
    local template_dir="$SCRIPT_DIR/../templates/services/$service_type"
    local target_dir="services/$service_type/$service_name"
    
    # Check if template exists
    if [[ ! -d "$template_dir" ]]; then
        log_error "Template not found: $template_dir"
        return 1
    fi
    
    # Check if target already exists
    if [[ -d "$target_dir" ]]; then
        log_error "Service already exists: $target_dir"
        return 1
    fi
    
    show_section "Scaffolding $service_type Service: $service_name"
    
    # Create target directory
    log_info "Creating service directory..."
    mkdir -p "$target_dir"
    
    # Copy and process templates
    log_info "Copying templates..."
    copy_and_process_templates "$template_dir" "$target_dir" "$service_name"
    
    # Initialize service based on type
    log_info "Initializing service..."
    case "$service_type" in
        nest)
            init_nest_service "$target_dir" "$service_name"
            ;;
        bull)
            init_bull_service "$target_dir" "$service_name"
            ;;
        go)
            init_go_service "$target_dir" "$service_name"
            ;;
        py)
            init_python_service "$target_dir" "$service_name"
            ;;
    esac
    
    # Update environment configuration
    update_env_for_service "$service_type" "$service_name"
    
    # Rebuild docker-compose
    log_info "Rebuilding Docker configuration..."
    if [[ -f "$SCRIPT_DIR/build.sh" ]]; then
        source "$SCRIPT_DIR/build.sh"
        cmd_build --quiet
    fi
    
    log_success "Service scaffolded successfully: $service_name"
    echo
    echo "Next steps:"
    echo "  1. Review the generated code in $target_dir"
    echo "  2. Update the service configuration as needed"
    echo "  3. Run: nself up"
    
    return 0
}

# Copy and process templates
copy_and_process_templates() {
    local template_dir="$1"
    local target_dir="$2"
    local service_name="$3"
    
    # Copy all files from template
    cp -r "$template_dir"/* "$target_dir/" 2>/dev/null || true
    
    # Process template files
    for template_file in "$target_dir"/*.template; do
        if [[ -f "$template_file" ]]; then
            local output_file="${template_file%.template}"
            
            # Replace placeholders
            sed -e "s/__SERVICE_NAME__/$service_name/g" \
                -e "s/__PROJECT_NAME__/${PROJECT_NAME:-nself}/g" \
                -e "s/__BASE_DOMAIN__/${BASE_DOMAIN:-localhost}/g" \
                "$template_file" > "$output_file"
            
            rm "$template_file"
        fi
    done
}

# Initialize NestJS service
init_nest_service() {
    local service_dir="$1"
    local service_name="$2"
    
    # Create package.json if not exists
    if [[ ! -f "$service_dir/package.json" ]]; then
        cat > "$service_dir/package.json" << EOF
{
  "name": "$service_name",
  "version": "1.0.0",
  "scripts": {
    "build": "tsc",
    "start": "node dist/main.js",
    "dev": "ts-node-dev --respawn src/main.ts"
  },
  "dependencies": {
    "@nestjs/common": "^10.0.0",
    "@nestjs/core": "^10.0.0",
    "@nestjs/platform-express": "^10.0.0",
    "reflect-metadata": "^0.1.13",
    "rxjs": "^7.8.1"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0",
    "ts-node-dev": "^2.0.0"
  }
}
EOF
    fi
    
    # Create main.ts with health endpoint
    if [[ ! -f "$service_dir/src/main.ts" ]]; then
        mkdir -p "$service_dir/src"
        cat > "$service_dir/src/main.ts" << 'EOF'
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  app.enableCors();
  await app.listen(process.env.PORT || 3000);
  console.log(`Service running on port ${process.env.PORT || 3000}`);
}
bootstrap();
EOF
        
        # Create app.module.ts
        cat > "$service_dir/src/app.module.ts" << 'EOF'
import { Module } from '@nestjs/common';
import { HealthController } from './health.controller';

@Module({
  controllers: [HealthController],
})
export class AppModule {}
EOF
        
        # Create health controller
        cat > "$service_dir/src/health.controller.ts" << 'EOF'
import { Controller, Get } from '@nestjs/common';

@Controller()
export class HealthController {
  @Get('health')
  health() {
    return { status: 'ok', service: process.env.SERVICE_NAME || 'nest-service' };
  }
}
EOF
    fi
    
    # Create Dockerfile if not exists
    if [[ ! -f "$service_dir/Dockerfile" ]]; then
        create_nestjs_dockerfile "$service_dir"
    fi
}

# Initialize BullMQ service
init_bull_service() {
    local service_dir="$1"
    local service_name="$2"
    
    # Similar to NestJS but with BullMQ worker setup
    if [[ ! -f "$service_dir/package.json" ]]; then
        cat > "$service_dir/package.json" << EOF
{
  "name": "$service_name",
  "version": "1.0.0",
  "scripts": {
    "start": "node dist/worker.js",
    "build": "tsc",
    "dev": "ts-node-dev --respawn src/worker.ts"
  },
  "dependencies": {
    "bullmq": "^4.0.0",
    "ioredis": "^5.0.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0",
    "ts-node-dev": "^2.0.0"
  }
}
EOF
    fi
}

# Initialize Go service
init_go_service() {
    local service_dir="$1"
    local service_name="$2"
    
    if [[ ! -f "$service_dir/go.mod" ]]; then
        cat > "$service_dir/go.mod" << EOF
module $service_name

go 1.22

require (
    github.com/gin-gonic/gin v1.9.1
)
EOF
    fi
    
    if [[ ! -f "$service_dir/main.go" ]]; then
        cat > "$service_dir/main.go" << 'EOF'
package main

import (
    "net/http"
    "os"
    "github.com/gin-gonic/gin"
)

func main() {
    r := gin.Default()
    
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "ok",
            "service": os.Getenv("SERVICE_NAME"),
        })
    })
    
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    r.Run(":" + port)
}
EOF
    fi
    
    # Create Dockerfile if not exists
    if [[ ! -f "$service_dir/Dockerfile" ]]; then
        create_go_dockerfile "$service_dir"
    fi
}

# Initialize Python service
init_python_service() {
    local service_dir="$1"
    local service_name="$2"
    
    if [[ ! -f "$service_dir/requirements.txt" ]]; then
        cat > "$service_dir/requirements.txt" << 'EOF'
fastapi==0.104.1
uvicorn[standard]==0.24.0
pydantic==2.5.0
python-dotenv==1.0.0
EOF
    fi
    
    if [[ ! -f "$service_dir/main.py" ]]; then
        cat > "$service_dir/main.py" << 'EOF'
import os
from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

class HealthResponse(BaseModel):
    status: str
    service: str

@app.get("/health", response_model=HealthResponse)
async def health():
    return {
        "status": "ok",
        "service": os.getenv("SERVICE_NAME", "python-service")
    }

@app.get("/")
async def root():
    return {"message": "Service is running"}
EOF
    fi
    
    # Create Dockerfile if not exists
    if [[ ! -f "$service_dir/Dockerfile" ]]; then
        create_python_dockerfile "$service_dir"
    fi
}

# Update environment for new service
update_env_for_service() {
    local service_type="$1"
    local service_name="$2"
    local env_file=".env.local"
    
    if [[ ! -f "$env_file" ]]; then
        return
    fi
    
    # Add service to appropriate environment variable
    case "$service_type" in
        nest)
            local var_name="NESTJS_SERVICES"
            ;;
        bull)
            local var_name="BULLMQ_SERVICES"
            ;;
        go)
            local var_name="GO_SERVICES"
            ;;
        py)
            local var_name="PYTHON_SERVICES"
            ;;
    esac
    
    # Check if variable exists and append service
    if grep -q "^${var_name}=" "$env_file"; then
        # Append to existing list
        sed -i.bak "s/^${var_name}=\(.*\)/${var_name}=\1,$service_name/" "$env_file"
    else
        # Add new variable
        echo "${var_name}=$service_name" >> "$env_file"
    fi
    
    log_info "Added $service_name to $var_name in .env.local"
}

# Dockerfile creators (imported from auto-fix/docker.sh)
create_nestjs_dockerfile() {
    local dir="$1"
    cat > "$dir/Dockerfile" << 'EOF'
FROM node:18-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci || npm install
COPY . .
RUN npm run build

FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production || npm install --only=production
COPY --from=builder /app/dist ./dist
EXPOSE 3000
CMD ["node", "dist/main.js"]
EOF
}

create_go_dockerfile() {
    local dir="$1"
    cat > "$dir/Dockerfile" << 'EOF'
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod tidy && go mod download
COPY . .
ENV GO111MODULE=on
RUN go build -mod=mod -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
EOF
}

create_python_dockerfile() {
    local dir="$1"
    cat > "$dir/Dockerfile" << 'EOF'
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
EOF
}

# Export main function
export -f cmd_scaffold