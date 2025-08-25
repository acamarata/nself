#!/usr/bin/env bash
# service-builder.sh - Build custom services from CUSTOM_SERVICES definition

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../utils/display.sh"
source "$SCRIPT_DIR/../utils/env.sh"

# Parse custom services definition
parse_custom_services() {
  local services_def="${CUSTOM_SERVICES:-}"
  
  if [[ -z "$services_def" ]]; then
    return 0
  fi
  
  # Array to store parsed services
  PARSED_SERVICES=()
  
  # Split by comma
  IFS=',' read -ra SERVICES <<< "$services_def"
  
  for service in "${SERVICES[@]}"; do
    # Trim whitespace
    service=$(echo "$service" | xargs)
    
    # Parse SERVICE_NAME:LANGUAGE:ROUTING
    IFS=':' read -r name language routing <<< "$service"
    
    # Default routing to service name if not provided
    routing="${routing:-$name}"
    
    # Convert to uppercase for env var lookups
    name_upper=$(echo "$name" | tr '[:lower:]' '[:upper:]')
    
    # Get environment-specific domain override
    local current_env="${ENV:-dev}"
    local domain_override=""
    
    if [[ "$current_env" == "prod" ]] || [[ "$current_env" == "production" ]]; then
      domain_override=$(eval echo "\${${name_upper}_DOMAIN_PROD:-}")
    else
      domain_override=$(eval echo "\${${name_upper}_DOMAIN_DEV:-}")
    fi
    
    # Determine final domain
    local final_domain=""
    if [[ -n "$domain_override" ]]; then
      # Use environment-specific override if set
      final_domain="$domain_override"
    elif [[ "$routing" == *"."* ]] && [[ "$routing" == *".com"* || "$routing" == *".org"* || "$routing" == *".net"* || "$routing" == *".io"* ]]; then
      # Full domain provided (e.g., metals.goldprices.com)
      final_domain="$routing"
    elif [[ "$routing" == *"."* ]]; then
      # Multi-level subdomain (e.g., metals.api → metals.api.${BASE_DOMAIN})
      final_domain="${routing}.${BASE_DOMAIN}"
    else
      # Simple subdomain (e.g., metals → metals.${BASE_DOMAIN})
      final_domain="${routing}.${BASE_DOMAIN}"
    fi
    
    # Get service-specific configuration
    local port=$(eval echo "\${${name_upper}_PORT:-}")
    local replicas=$(eval echo "\${${name_upper}_REPLICAS:-1}")
    local memory=$(eval echo "\${${name_upper}_MEMORY:-256M}")
    local cpu=$(eval echo "\${${name_upper}_CPU:-0.25}")
    local env_vars=$(eval echo "\${${name_upper}_ENV:-}")
    local healthcheck=$(eval echo "\${${name_upper}_HEALTHCHECK:-/health}")
    local public=$(eval echo "\${${name_upper}_PUBLIC:-true}")
    local rate_limit=$(eval echo "\${${name_upper}_RATE_LIMIT:-}")
    
    # Auto-assign port if not set
    if [[ -z "$port" ]]; then
      port=$((8000 + ${#PARSED_SERVICES[@]}))
    fi
    
    # Store parsed service (using final_domain instead of subdomain)
    PARSED_SERVICES+=("$name|$language|$final_domain|$port|$replicas|$memory|$cpu|$env_vars|$healthcheck|$public|$rate_limit")
  done
}

# Generate docker-compose service definition
generate_service_compose() {
  local service_info="$1"
  IFS='|' read -r name language domain port replicas memory cpu env_vars healthcheck public rate_limit <<< "$service_info"
  
  cat << EOF

  ${name}:
    build: 
      context: ./services/${name}
      dockerfile: Dockerfile
    container_name: \${PROJECT_NAME}_${name}
    restart: unless-stopped
    networks:
      - nself
    ports:
      - "${port}:${port}"
    environment:
      - NODE_ENV=\${ENV}
      - PORT=${port}
      - SERVICE_NAME=${name}
      - DATABASE_URL=postgresql://\${POSTGRES_USER}:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}
      - REDIS_URL=redis://redis:6379
      - BASE_URL=https://${domain}
EOF
  
  # Add custom environment variables
  if [[ -n "$env_vars" ]]; then
    echo "      # Custom environment variables"
    IFS=',' read -ra ENVS <<< "$env_vars"
    for env in "${ENVS[@]}"; do
      echo "      - $env"
    done
  fi
  
  # Add resource limits
  cat << EOF
    deploy:
      replicas: ${replicas}
      resources:
        limits:
          memory: ${memory}
          cpus: '${cpu}'
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${port}${healthcheck}"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    depends_on:
      - postgres
EOF
  
  # Add Redis dependency if enabled
  if [[ "${REDIS_ENABLED:-false}" == "true" ]]; then
    echo "      - redis"
  fi
  
  # Add volumes for development
  if [[ "${ENV:-dev}" == "dev" ]]; then
    cat << EOF
    volumes:
      - ./services/${name}:/app
      - /app/node_modules  # Prevent node_modules from being overwritten
EOF
  fi
}

# Generate Nginx configuration for service
generate_service_nginx() {
  local service_info="$1"
  IFS='|' read -r name language domain port replicas memory cpu env_vars healthcheck public rate_limit <<< "$service_info"
  
  # Skip if not public
  if [[ "$public" != "true" ]]; then
    return
  fi
  
  # Determine SSL certificate path based on domain
  local ssl_cert_path="/etc/nginx/ssl/certs/\${BASE_DOMAIN}/fullchain.pem"
  local ssl_key_path="/etc/nginx/ssl/certs/\${BASE_DOMAIN}/privkey.pem"
  
  # Check if using a custom domain (not a subdomain of BASE_DOMAIN)
  if [[ "$domain" != *"\${BASE_DOMAIN}"* ]]; then
    # For custom domains, use domain-specific certs if available, otherwise fallback
    ssl_cert_path="/etc/nginx/ssl/certs/${domain}/fullchain.pem"
    ssl_key_path="/etc/nginx/ssl/certs/${domain}/privkey.pem"
  fi
  
  cat << EOF

# ${name} Service
server {
    listen 80;
    listen 443 ssl http2;
    server_name ${domain};

    # SSL Configuration
    include /etc/nginx/ssl/ssl.conf;
    ssl_certificate ${ssl_cert_path};
    ssl_certificate_key ${ssl_key_path};

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

EOF

  # Add rate limiting if configured
  if [[ -n "$rate_limit" ]]; then
    cat << EOF
    # Rate Limiting
    limit_req_zone \$binary_remote_addr zone=${name}_limit:10m rate=${rate_limit}r/m;
    limit_req zone=${name}_limit burst=10 nodelay;
    limit_req_status 429;

EOF
  fi
  
  cat << EOF
    # Proxy Configuration
    location / {
        proxy_pass http://${name}:${port};
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # Buffering
        proxy_buffering off;
        proxy_request_buffering off;
    }
    
    # Health check endpoint (internal only)
    location ${healthcheck} {
        access_log off;
        proxy_pass http://${name}:${port}${healthcheck};
    }
}
EOF
}

# Create service directory and template
create_service_template() {
  local service_info="$1"
  IFS='|' read -r name language subdomain port replicas memory cpu env_vars healthcheck public rate_limit <<< "$service_info"
  
  local service_dir="./services/${name}"
  
  # Create directory if it doesn't exist
  if [[ ! -d "$service_dir" ]]; then
    mkdir -p "$service_dir"
    log_info "Creating service template for ${name} (${language})"
    
    # Try to copy from template files first
    if copy_template_files "$name" "$language" "$port" "$healthcheck"; then
      log_success "Service template created from template files"
    else
      # Fall back to inline templates
      case "$language" in
        nodejs|node)
          create_nodejs_template "$name" "$port" "$healthcheck"
          ;;
        python)
          create_python_template "$name" "$port" "$healthcheck"
          ;;
        go|golang)
          create_go_template "$name" "$port" "$healthcheck"
          ;;
        ruby)
          create_ruby_template "$name" "$port" "$healthcheck"
          ;;
        rust)
          create_rust_template "$name" "$port" "$healthcheck"
          ;;
        java)
          create_java_template "$name" "$port" "$healthcheck"
          ;;
        dotnet|csharp)
          create_dotnet_template "$name" "$port" "$healthcheck"
          ;;
        *)
          log_warning "Unknown language: $language. Creating generic template."
          create_generic_template "$name" "$port" "$healthcheck"
          ;;
      esac
    fi
  else
    log_info "Service directory already exists: ${service_dir}"
  fi
}

# Copy template files from templates directory
copy_template_files() {
  local name="$1"
  local language="$2"
  local port="$3"
  local health="$4"
  
  # Get the script directory - handle both sourced and executed contexts
  local script_dir
  if [[ -n "${BASH_SOURCE[0]}" ]]; then
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  else
    # Fallback for when script is sourced from build.sh
    script_dir="$(dirname "$(which nself)")/src/lib/services"
    if [[ ! -d "$script_dir" ]]; then
      script_dir="/usr/local/nself/src/lib/services"
    fi
    if [[ ! -d "$script_dir" ]]; then
      script_dir="$HOME/.nself/src/lib/services"
    fi
  fi
  
  local template_file="$script_dir/templates/${language}.template"
  
  # Check if template file exists
  if [[ ! -f "$template_file" ]]; then
    return 1
  fi
  
  # Convert name to various cases
  local name_upper=$(echo "$name" | tr '[:lower:]' '[:upper:]')
  local name_lower=$(echo "$name" | tr '[:upper:]' '[:lower:]')
  local name_pascal=$(echo "$name" | sed 's/\b\(.\)/\u\1/g')
  
  # Copy and process template
  local service_dir="./services/${name}"
  
  # Main service file
  local service_ext
  case "$language" in
    nodejs|node) service_ext="js" ;;
    python) service_ext="py" ;;
    go|golang) service_ext="go" ;;
    ruby) service_ext="rb" ;;
    rust) service_ext="rs" ;;
    java) service_ext="java" ;;
    dotnet|csharp) service_ext="cs" ;;
    *) service_ext="txt" ;;
  esac
  
  # Process template and save to service file
  sed -e "s/{{SERVICE_NAME}}/${name}/g" \
      -e "s/{{SERVICE_NAME_UPPER}}/${name_upper}/g" \
      -e "s/{{SERVICE_NAME_LOWER}}/${name_lower}/g" \
      -e "s/{{SERVICE_NAME_PASCAL}}/${name_pascal}/g" \
      -e "s/{{SERVICE_PORT}}/${port}/g" \
      -e "s/{{HEALTHCHECK}}/${health}/g" \
      "$template_file" > "${service_dir}/main.${service_ext}"
  
  # Create Dockerfile based on language
  create_dockerfile_for_language "$name" "$language" "$port"
  
  # Create package/dependency files based on language
  create_dependency_files "$name" "$language"
  
  return 0
}

# Create Dockerfile based on language
create_dockerfile_for_language() {
  local name="$1"
  local language="$2"
  local port="$3"
  local dir="./services/${name}"
  
  case "$language" in
    nodejs|node)
      cat > "${dir}/Dockerfile" << EOF
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN addgroup -g 1001 -S nodejs && adduser -S nodejs -u 1001
USER nodejs
EXPOSE ${port}
CMD ["node", "main.js"]
EOF
      ;;
    python)
      cat > "${dir}/Dockerfile" << EOF
FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN useradd -m -u 1001 python && chown -R python:python /app
USER python
EXPOSE ${port}
CMD ["python", "main.py"]
EOF
      ;;
    go|golang)
      cat > "${dir}/Dockerfile" << EOF
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
RUN addgroup -g 1001 -S app && adduser -S app -u 1001
USER app
EXPOSE ${port}
CMD ["./main"]
EOF
      ;;
    ruby)
      cat > "${dir}/Dockerfile" << EOF
FROM ruby:3.2-slim
WORKDIR /app
COPY Gemfile Gemfile.lock ./
RUN bundle install --without development test
COPY . .
RUN useradd -m -u 1001 ruby && chown -R ruby:ruby /app
USER ruby
EXPOSE ${port}
CMD ["ruby", "main.rb"]
EOF
      ;;
    rust)
      cat > "${dir}/Dockerfile" << EOF
FROM rust:1.75 AS builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
COPY src ./src
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/main /usr/local/bin/main
RUN useradd -m -u 1001 rust
USER rust
EXPOSE ${port}
CMD ["main"]
EOF
      ;;
    java)
      cat > "${dir}/Dockerfile" << EOF
FROM maven:3.9-openjdk-17 AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline
COPY src ./src
RUN mvn package

FROM openjdk:17-slim
COPY --from=builder /app/target/*.jar app.jar
RUN useradd -m -u 1001 java
USER java
EXPOSE ${port}
CMD ["java", "-jar", "app.jar"]
EOF
      ;;
    dotnet|csharp)
      cat > "${dir}/Dockerfile" << EOF
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS builder
WORKDIR /app
COPY *.csproj ./
RUN dotnet restore
COPY . .
RUN dotnet publish -c Release -o out

FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /app
COPY --from=builder /app/out .
RUN useradd -m -u 1001 dotnet
USER dotnet
EXPOSE ${port}
CMD ["dotnet", "main.dll"]
EOF
      ;;
  esac
}

# Create dependency files based on language
create_dependency_files() {
  local name="$1"
  local language="$2"
  local dir="./services/${name}"
  
  case "$language" in
    nodejs|node)
      cat > "${dir}/package.json" << EOF
{
  "name": "${name}-service",
  "version": "1.0.0",
  "main": "main.js",
  "scripts": {
    "start": "node main.js"
  },
  "dependencies": {
    "express": "^4.18.2"
  }
}
EOF
      ;;
    python)
      cat > "${dir}/requirements.txt" << EOF
flask==3.0.0
requests==2.31.0
EOF
      ;;
    go|golang)
      cat > "${dir}/go.mod" << EOF
module ${name}

go 1.21

require (
  net/http v0.0.0
)
EOF
      cat > "${dir}/go.sum" << EOF
# Go dependencies will be resolved on first build
EOF
      ;;
    ruby)
      cat > "${dir}/Gemfile" << EOF
source 'https://rubygems.org'
gem 'sinatra', '~> 3.0'
gem 'puma', '~> 6.0'
gem 'json'
EOF
      cat > "${dir}/Gemfile.lock" << EOF
# Ruby dependencies will be locked on first install
EOF
      ;;
    rust)
      cat > "${dir}/Cargo.toml" << EOF
[package]
name = "${name}"
version = "1.0.0"
edition = "2021"

[dependencies]
actix-web = "4"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
chrono = { version = "0.4", features = ["serde"] }
tokio = { version = "1", features = ["full"] }
EOF
      mkdir -p "${dir}/src"
      mv "${dir}/main.rs" "${dir}/src/main.rs" 2>/dev/null || true
      ;;
    java)
      cat > "${dir}/pom.xml" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>
    
    <groupId>com.nself</groupId>
    <artifactId>${name}</artifactId>
    <version>1.0.0</version>
    
    <properties>
        <maven.compiler.source>17</maven.compiler.source>
        <maven.compiler.target>17</maven.compiler.target>
    </properties>
    
    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
            <version>3.2.0</version>
        </dependency>
    </dependencies>
</project>
EOF
      mkdir -p "${dir}/src/main/java/com/nself/${name}"
      mv "${dir}/main.java" "${dir}/src/main/java/com/nself/${name}/Application.java" 2>/dev/null || true
      ;;
    dotnet|csharp)
      cat > "${dir}/${name}.csproj" << EOF
<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
  </PropertyGroup>
</Project>
EOF
      mv "${dir}/main.cs" "${dir}/Program.cs" 2>/dev/null || true
      ;;
  esac
}

# Node.js service template
create_nodejs_template() {
  local name="$1"
  local port="$2"
  local health="$3"
  local dir="./services/${name}"
  
  # package.json
  cat > "${dir}/package.json" << EOF
{
  "name": "${name}-service",
  "version": "1.0.0",
  "description": "${name} service for nself",
  "main": "index.js",
  "scripts": {
    "start": "node index.js",
    "dev": "nodemon index.js",
    "test": "jest"
  },
  "dependencies": {
    "express": "^4.18.2",
    "pg": "^8.11.3",
    "redis": "^4.6.10",
    "helmet": "^7.1.0",
    "cors": "^2.8.5",
    "dotenv": "^16.3.1",
    "winston": "^3.11.0"
  },
  "devDependencies": {
    "nodemon": "^3.0.1",
    "jest": "^29.7.0"
  }
}
EOF

  # index.js
  cat > "${dir}/index.js" << 'EOF'
const express = require('express');
const helmet = require('helmet');
const cors = require('cors');
const { Pool } = require('pg');
const redis = require('redis');
const winston = require('winston');

// Initialize Express app
const app = express();
const port = process.env.PORT || 3000;

// Logger setup
const logger = winston.createLogger({
  level: process.env.NODE_ENV === 'production' ? 'info' : 'debug',
  format: winston.format.json(),
  transports: [
    new winston.transports.Console({
      format: winston.format.simple()
    })
  ]
});

// Database connection
const pool = new Pool({
  connectionString: process.env.DATABASE_URL
});

// Redis connection (optional)
let redisClient;
if (process.env.REDIS_URL) {
  redisClient = redis.createClient({
    url: process.env.REDIS_URL
  });
  redisClient.on('error', (err) => logger.error('Redis Client Error', err));
  redisClient.connect().catch(console.error);
}

// Middleware
app.use(helmet());
app.use(cors());
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

// Request logging
app.use((req, res, next) => {
  logger.info(`${req.method} ${req.path}`);
  next();
});

// Health check
app.get(process.env.HEALTHCHECK || '/health', async (req, res) => {
  try {
    // Check database connection
    await pool.query('SELECT 1');
    
    // Check Redis if configured
    if (redisClient) {
      await redisClient.ping();
    }
    
    res.json({ 
      status: 'healthy',
      service: process.env.SERVICE_NAME,
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    logger.error('Health check failed:', error);
    res.status(503).json({ 
      status: 'unhealthy',
      error: error.message 
    });
  }
});

// Root endpoint
app.get('/', (req, res) => {
  res.json({
    message: `${process.env.SERVICE_NAME} service is running`,
    version: '1.0.0',
    environment: process.env.NODE_ENV
  });
});

// Example API endpoint
app.get('/api/example', async (req, res) => {
  try {
    // Example database query
    const result = await pool.query('SELECT NOW()');
    
    // Example Redis cache
    if (redisClient) {
      await redisClient.set('last_request', new Date().toISOString());
    }
    
    res.json({
      data: result.rows[0],
      cached: redisClient ? true : false
    });
  } catch (error) {
    logger.error('API error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Error handling middleware
app.use((err, req, res, next) => {
  logger.error('Unhandled error:', err);
  res.status(500).json({ error: 'Something went wrong!' });
});

// Start server
app.listen(port, () => {
  logger.info(`${process.env.SERVICE_NAME} service listening on port ${port}`);
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  logger.info('SIGTERM signal received: closing HTTP server');
  await pool.end();
  if (redisClient) {
    await redisClient.quit();
  }
  process.exit(0);
});
EOF

  # Dockerfile
  cat > "${dir}/Dockerfile" << EOF
FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy application code
COPY . .

# Create non-root user
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nodejs -u 1001
USER nodejs

EXPOSE ${port}

CMD ["node", "index.js"]
EOF

  # .dockerignore
  cat > "${dir}/.dockerignore" << EOF
node_modules
npm-debug.log
.env
.git
.gitignore
README.md
.eslintrc
.prettierrc
coverage
.nyc_output
EOF
}

# Python service template
create_python_template() {
  local name="$1"
  local port="$2"
  local health="$3"
  local dir="./services/${name}"
  
  # requirements.txt
  cat > "${dir}/requirements.txt" << EOF
fastapi==0.104.1
uvicorn[standard]==0.24.0
psycopg2-binary==2.9.9
redis==5.0.1
python-dotenv==1.0.0
pydantic==2.5.0
httpx==0.25.2
EOF

  # main.py
  cat > "${dir}/main.py" << 'EOF'
import os
import logging
from datetime import datetime
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import psycopg2
from psycopg2.extras import RealDictCursor
import redis
import uvicorn

# Configure logging
logging.basicConfig(
    level=logging.INFO if os.getenv('ENV') == 'prod' else logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Database connection
def get_db():
    return psycopg2.connect(
        os.getenv('DATABASE_URL'),
        cursor_factory=RealDictCursor
    )

# Redis connection (optional)
redis_client = None
if os.getenv('REDIS_URL'):
    try:
        redis_client = redis.from_url(os.getenv('REDIS_URL'))
        redis_client.ping()
        logger.info("Redis connected successfully")
    except Exception as e:
        logger.warning(f"Redis connection failed: {e}")

# Lifespan context manager
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logger.info(f"{os.getenv('SERVICE_NAME')} service starting")
    yield
    # Shutdown
    logger.info(f"{os.getenv('SERVICE_NAME')} service shutting down")

# Create FastAPI app
app = FastAPI(
    title=f"{os.getenv('SERVICE_NAME', 'Service')} API",
    version="1.0.0",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Health check endpoint
@app.get(os.getenv('HEALTHCHECK', '/health'))
async def health_check():
    try:
        # Check database
        with get_db() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT 1")
        
        # Check Redis if configured
        if redis_client:
            redis_client.ping()
        
        return {
            "status": "healthy",
            "service": os.getenv('SERVICE_NAME'),
            "timestamp": datetime.utcnow().isoformat()
        }
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(status_code=503, detail=str(e))

# Root endpoint
@app.get("/")
async def root():
    return {
        "message": f"{os.getenv('SERVICE_NAME')} service is running",
        "version": "1.0.0",
        "environment": os.getenv('ENV', 'dev')
    }

# Example API endpoint
@app.get("/api/example")
async def example_endpoint():
    try:
        # Example database query
        with get_db() as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT NOW() as current_time")
                result = cur.fetchone()
        
        # Example Redis cache
        if redis_client:
            redis_client.set('last_request', datetime.utcnow().isoformat())
        
        return {
            "data": result,
            "cached": redis_client is not None
        }
    except Exception as e:
        logger.error(f"API error: {e}")
        raise HTTPException(status_code=500, detail="Internal server error")

if __name__ == "__main__":
    port = int(os.getenv('PORT', 8000))
    uvicorn.run(app, host="0.0.0.0", port=port)
EOF

  # Dockerfile
  cat > "${dir}/Dockerfile" << EOF
FROM python:3.11-slim

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Create non-root user
RUN useradd -m -u 1001 python && chown -R python:python /app
USER python

EXPOSE ${port}

CMD ["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "${port}"]
EOF

  # .dockerignore
  cat > "${dir}/.dockerignore" << EOF
__pycache__
*.pyc
*.pyo
*.pyd
.Python
env/
venv/
.venv/
pip-log.txt
pip-delete-this-directory.txt
.tox/
.coverage
.coverage.*
.cache
*.log
.git
.gitignore
.pytest_cache
EOF
}

# Go service template
create_go_template() {
  local name="$1"
  local port="$2"
  local health="$3"
  local dir="./services/${name}"
  
  # go.mod
  cat > "${dir}/go.mod" << EOF
module ${name}

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/lib/pq v1.10.9
    github.com/redis/go-redis/v9 v9.3.0
    github.com/joho/godotenv v1.5.1
)
EOF

  # main.go
  cat > "${dir}/main.go" << 'EOF'
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
    "github.com/redis/go-redis/v9"
)

var (
    db *sql.DB
    rdb *redis.Client
    ctx = context.Background()
)

func init() {
    // Database connection
    var err error
    db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Printf("Database connection error: %v", err)
    } else {
        db.SetMaxOpenConns(25)
        db.SetMaxIdleConns(5)
        db.SetConnMaxLifetime(5 * time.Minute)
    }
    
    // Redis connection (optional)
    if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
        opt, err := redis.ParseURL(redisURL)
        if err == nil {
            rdb = redis.NewClient(opt)
            if err := rdb.Ping(ctx).Err(); err != nil {
                log.Printf("Redis connection error: %v", err)
                rdb = nil
            }
        }
    }
}

func main() {
    // Set Gin mode
    if os.Getenv("ENV") == "prod" {
        gin.SetMode(gin.ReleaseMode)
    }
    
    r := gin.Default()
    
    // Middleware
    r.Use(gin.Recovery())
    r.Use(corsMiddleware())
    
    // Health check
    healthPath := os.Getenv("HEALTHCHECK")
    if healthPath == "" {
        healthPath = "/health"
    }
    r.GET(healthPath, healthCheck)
    
    // Routes
    r.GET("/", rootHandler)
    r.GET("/api/example", exampleHandler)
    
    // Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8000"
    }
    
    srv := &http.Server{
        Addr:    ":" + port,
        Handler: r,
    }
    
    // Graceful shutdown
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %s\n", err)
        }
    }()
    
    log.Printf("%s service listening on port %s", os.Getenv("SERVICE_NAME"), port)
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down server...")
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }
    
    if db != nil {
        db.Close()
    }
    if rdb != nil {
        rdb.Close()
    }
    
    log.Println("Server exited")
}

func corsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }
        
        c.Next()
    }
}

func healthCheck(c *gin.Context) {
    status := "healthy"
    
    // Check database
    if db != nil {
        if err := db.Ping(); err != nil {
            status = "unhealthy"
        }
    }
    
    // Check Redis
    if rdb != nil {
        if err := rdb.Ping(ctx).Err(); err != nil {
            status = "degraded"
        }
    }
    
    statusCode := http.StatusOK
    if status == "unhealthy" {
        statusCode = http.StatusServiceUnavailable
    }
    
    c.JSON(statusCode, gin.H{
        "status":    status,
        "service":   os.Getenv("SERVICE_NAME"),
        "timestamp": time.Now().Format(time.RFC3339),
    })
}

func rootHandler(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "message":     fmt.Sprintf("%s service is running", os.Getenv("SERVICE_NAME")),
        "version":     "1.0.0",
        "environment": os.Getenv("ENV"),
    })
}

func exampleHandler(c *gin.Context) {
    // Example database query
    var now time.Time
    if db != nil {
        err := db.QueryRow("SELECT NOW()").Scan(&now)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
            return
        }
    }
    
    // Example Redis cache
    cached := false
    if rdb != nil {
        err := rdb.Set(ctx, "last_request", time.Now().Format(time.RFC3339), 0).Err()
        cached = err == nil
    }
    
    c.JSON(http.StatusOK, gin.H{
        "data":   now.Format(time.RFC3339),
        "cached": cached,
    })
}
EOF

  # Dockerfile
  cat > "${dir}/Dockerfile" << EOF
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

# Create non-root user
RUN addgroup -g 1001 -S app && \
    adduser -S app -u 1001
USER app

EXPOSE ${port}

CMD ["./main"]
EOF

  # .dockerignore
  cat > "${dir}/.dockerignore" << EOF
.git
.gitignore
README.md
*.log
tmp/
vendor/
.env
EOF
}

# Create generic template for unsupported languages
create_generic_template() {
  local name="$1"
  local port="$2"
  local health="$3"
  local dir="./services/${name}"
  
  cat > "${dir}/README.md" << EOF
# ${name} Service

This is a placeholder for your ${name} service.

## Configuration

- Port: ${port}
- Health Check: ${health}
- Database URL: Available as DATABASE_URL environment variable
- Redis URL: Available as REDIS_URL environment variable (if enabled)

## Setup

1. Add your application code here
2. Create a Dockerfile for your chosen language/framework
3. Implement the health check endpoint at ${health}
4. Run \`nself build\` to rebuild containers
5. Run \`nself restart\` to apply changes

## Environment Variables

- NODE_ENV / ENV: Environment mode (dev/prod)
- PORT: ${port}
- SERVICE_NAME: ${name}
- DATABASE_URL: PostgreSQL connection string
- REDIS_URL: Redis connection string (if Redis is enabled)
- BASE_URL: The public URL for this service

## Required Endpoints

### Health Check (${health})
Should return 200 OK when the service is healthy:
\`\`\`json
{
  "status": "healthy",
  "service": "${name}",
  "timestamp": "2024-01-15T12:00:00Z"
}
\`\`\`

## Docker Requirements

Your Dockerfile should:
1. Expose port ${port}
2. Run as non-root user (uid 1001)
3. Handle SIGTERM for graceful shutdown
4. Include health check command
EOF

  cat > "${dir}/Dockerfile" << EOF
# Add your Dockerfile here
# Example structure:

# FROM your-base-image:version
# WORKDIR /app
# COPY . .
# RUN your-build-commands
# EXPOSE ${port}
# USER 1001
# CMD ["your-start-command"]
EOF
}

# Build all custom services
build_custom_services() {
  parse_custom_services
  
  if [[ ${#PARSED_SERVICES[@]} -eq 0 ]]; then
    return 0
  fi
  
  log_info "Building ${#PARSED_SERVICES[@]} custom service(s)..."
  
  # Generate docker-compose.custom.yml
  {
    echo "# Custom Services Configuration"
    echo "# Auto-generated by nself"
    echo ""
    echo "version: '3.8'"
    echo ""
    echo "services:"
    
    for service_info in "${PARSED_SERVICES[@]}"; do
      generate_service_compose "$service_info"
    done
    
    echo ""
    echo "networks:"
    echo "  nself:"
    echo "    external: true"
  } > docker-compose.custom.yml
  
  # Generate nginx configurations
  local nginx_config=""
  for service_info in "${PARSED_SERVICES[@]}"; do
    nginx_config+=$(generate_service_nginx "$service_info")
  done
  
  # Write to nginx custom services config
  if [[ -n "$nginx_config" ]]; then
    mkdir -p ./nginx/conf.d
    echo "$nginx_config" > ./nginx/conf.d/custom-services.conf
    log_success "Generated Nginx configuration for custom services"
  fi
  
  # Create service templates
  for service_info in "${PARSED_SERVICES[@]}"; do
    create_service_template "$service_info"
  done
  
  log_success "Custom services configuration complete"
  
  # Show summary
  echo ""
  echo "Custom Services Summary:"
  echo "========================"
  for service_info in "${PARSED_SERVICES[@]}"; do
    IFS='|' read -r name language domain port <<< "$service_info"
    echo "  • ${name} (${language}) → https://${domain}"
  done
  echo ""
  
  # Check for custom domains that may need SSL certs
  local custom_domains=()
  for service_info in "${PARSED_SERVICES[@]}"; do
    IFS='|' read -r name language domain port replicas memory cpu env_vars healthcheck public <<< "$service_info"
    if [[ "$public" == "true" ]] && [[ "$domain" != *"${BASE_DOMAIN}"* ]]; then
      custom_domains+=("$domain")
    fi
  done
  
  if [[ ${#custom_domains[@]} -gt 0 ]]; then
    echo "⚠️  Custom domains detected that will need SSL certificates:"
    for domain in "${custom_domains[@]}"; do
      echo "   - $domain"
    done
    echo ""
    echo "  For production, ensure you have valid SSL certificates for these domains."
    echo "  Place them in: nginx/ssl/certs/<domain>/"
    echo ""
  fi
  
  echo "Next steps:"
  echo "  1. Edit service code in ./services/<name>/"
  echo "  2. Run 'docker-compose -f docker-compose.yml -f docker-compose.custom.yml build'"
  echo "  3. Run 'nself restart' to apply changes"
}

# Export functions
export -f parse_custom_services
export -f build_custom_services