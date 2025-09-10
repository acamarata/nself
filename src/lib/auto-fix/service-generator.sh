#!/usr/bin/env bash

# service-generator.sh - Auto-generate missing service templates

# Source utilities - don't override parent SCRIPT_DIR
SERVICE_GEN_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SERVICE_GEN_SCRIPT_DIR/../utils/display.sh" 2>/dev/null || true

# Generate a basic Node.js Express service
generate_basic_node_service() {
  local service_name="$1"
  local service_path="services/node/${service_name}"
  
  mkdir -p "$service_path/src"
  
  # package.json
  cat > "$service_path/package.json" <<EOF
{
  "name": "${service_name}",
  "version": "1.0.0",
  "description": "Node.js microservice",
  "main": "src/index.js",
  "scripts": {
    "start": "node src/index.js",
    "dev": "nodemon src/index.js"
  },
  "dependencies": {
    "express": "^4.18.0",
    "cors": "^2.8.5",
    "dotenv": "^16.0.0"
  },
  "devDependencies": {
    "nodemon": "^3.0.0"
  }
}
EOF

  # Main application file
  cat > "$service_path/src/index.js" <<'EOF'
const express = require('express');
const cors = require('cors');

const app = express();
const PORT = process.env.PORT || 3000;

app.use(cors());
app.use(express.json());

app.get('/', (req, res) => {
  res.json({ message: `Hello from ${process.env.SERVICE_NAME || 'service'}!` });
});

app.get('/health', (req, res) => {
  res.json({ 
    status: 'healthy',
    service: process.env.SERVICE_NAME || 'service',
    uptime: process.uptime()
  });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Service running on port ${PORT}`);
});
EOF

  # Dockerfile
  cat > "$service_path/Dockerfile" <<'EOF'
FROM node:18-alpine
RUN apk add --no-cache curl wget
WORKDIR /app
COPY package*.json ./
RUN npm install --production
COPY . .
ARG PORT=3000
ENV PORT=${PORT}
EXPOSE ${PORT}
CMD ["node", "src/index.js"]
EOF
}

# Generate a NestJS hello world service
generate_nest_service() {
  local service_name="$1"
  local service_path="services/nest/${service_name}"

  # Silent generation, no log output

  # Create directory structure
  mkdir -p "$service_path/src"

  # Create package.json
  cat >"$service_path/package.json" <<'EOF'
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
  cat >"$service_path/src/main.ts" <<'EOF'
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
  cat >"$service_path/src/app.module.ts" <<'EOF'
import { Module } from '@nestjs/common';
import { AppController } from './app.controller';

@Module({
  imports: [],
  controllers: [AppController],
})
export class AppModule {}
EOF

  # Create app.controller.ts
  cat >"$service_path/src/app.controller.ts" <<'EOF'
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
  cat >"$service_path/nest-cli.json" <<'EOF'
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
  cat >"$service_path/tsconfig.json" <<'EOF'
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
  cat >"$service_path/Dockerfile" <<'EOF'
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
  cat >"$service_path/package.json" <<'EOF'
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
  cat >"$service_path/src/index.js" <<'EOF'
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
  cat >"$service_path/Dockerfile" <<'EOF'
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
  cat >"$service_path/go.mod" <<EOF
module ${service_name}

go 1.21

require github.com/gorilla/mux v1.8.0
EOF

  # Create go.sum with the mux dependency
  cat >"$service_path/go.sum" <<'EOF'
github.com/gorilla/mux v1.8.0 h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=
github.com/gorilla/mux v1.8.0/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
EOF

  # Create main.go
  cat >"$service_path/main.go" <<'EOF'
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
  cat >"$service_path/Dockerfile" <<'EOF'
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
  cat >"$service_path/requirements.txt" <<'EOF'
fastapi==0.104.0
uvicorn==0.24.0
python-dotenv==1.0.0
EOF

  # Create main.py
  cat >"$service_path/main.py" <<'EOF'
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
  cat >"$service_path/Dockerfile" <<'EOF'
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

# Generate a Rust service
generate_rust_service() {
  local service_name="$1"
  local service_path="services/rust/${service_name}"

  # Create directory structure
  mkdir -p "$service_path/src"

  # Create Cargo.toml
  cat >"$service_path/Cargo.toml" <<EOF
[package]
name = "${service_name}"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4"
tokio = { version = "1", features = ["full"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
env_logger = "0.10"
EOF

  # Create main.rs
  cat >"$service_path/src/main.rs" <<'EOF'
use actix_web::{web, App, HttpResponse, HttpServer, Result};
use serde::Serialize;

#[derive(Serialize)]
struct HealthResponse {
    status: String,
    service: String,
}

async fn index() -> Result<HttpResponse> {
    Ok(HttpResponse::Ok().json(&serde_json::json!({
        "message": format!("Hello from SERVICE_NAME service!")
    })))
}

async fn health() -> Result<HttpResponse> {
    Ok(HttpResponse::Ok().json(&HealthResponse {
        status: "ok".to_string(),
        service: "SERVICE_NAME".to_string(),
    }))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    env_logger::init();

    let port = std::env::var("PORT").unwrap_or_else(|_| "8000".to_string());
    let addr = format!("0.0.0.0:{}", port);

    println!("Starting SERVICE_NAME service on {}", addr);

    HttpServer::new(|| {
        App::new()
            .route("/", web::get().to(index))
            .route("/health", web::get().to(health))
    })
    .bind(&addr)?
    .run()
    .await
}
EOF
  sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/src/main.rs" && rm -f "$service_path/src/main.rs.bak"

  # Create Dockerfile
  cat >"$service_path/Dockerfile" <<'EOF'
FROM rust:1.75 as builder
WORKDIR /app
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release
RUN rm -rf src
COPY src ./src
RUN touch src/main.rs
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates curl wget && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/target/release/SERVICE_NAME /app/
ARG PORT=8000
ENV PORT=${PORT}
EXPOSE ${PORT}
CMD ["./SERVICE_NAME"]
EOF
  sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/Dockerfile" && rm -f "$service_path/Dockerfile.bak"
}

# Generate a Java service
generate_java_service() {
  local service_name="$1"
  local service_path="services/java/${service_name}"

  # Create directory structure
  mkdir -p "$service_path/src/main/java/com/nself/${service_name}"

  # Create pom.xml
  cat >"$service_path/pom.xml" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0
         http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>com.nself</groupId>
    <artifactId>${service_name}</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>

    <parent>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-parent</artifactId>
        <version>3.2.0</version>
    </parent>

    <properties>
        <java.version>17</java.version>
    </properties>

    <dependencies>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-web</artifactId>
        </dependency>
        <dependency>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-starter-actuator</artifactId>
        </dependency>
    </dependencies>

    <build>
        <plugins>
            <plugin>
                <groupId>org.springframework.boot</groupId>
                <artifactId>spring-boot-maven-plugin</artifactId>
            </plugin>
        </plugins>
    </build>
</project>
EOF

  # Create Application.java
  cat >"$service_path/src/main/java/com/nself/${service_name}/Application.java" <<'EOF'
package com.nself.SERVICE_NAME;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import java.util.HashMap;
import java.util.Map;

@SpringBootApplication
@RestController
public class Application {

    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }

    @GetMapping("/")
    public Map<String, String> index() {
        Map<String, String> response = new HashMap<>();
        response.put("message", "Hello from SERVICE_NAME service!");
        return response;
    }

    @GetMapping("/health")
    public Map<String, String> health() {
        Map<String, String> response = new HashMap<>();
        response.put("status", "ok");
        response.put("service", "SERVICE_NAME");
        return response;
    }
}
EOF
  sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/src/main/java/com/nself/${service_name}/Application.java" && rm -f "$service_path/src/main/java/com/nself/${service_name}/Application.java.bak"

  # Create Dockerfile
  cat >"$service_path/Dockerfile" <<'EOF'
FROM maven:3.9-eclipse-temurin-17 as builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline
COPY src ./src
RUN mvn package -DskipTests

FROM eclipse-temurin:17-jre
RUN apt-get update && apt-get install -y curl wget && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/target/*.jar app.jar
ARG PORT=8000
ENV PORT=${PORT}
EXPOSE ${PORT}
CMD ["java", "-jar", "app.jar", "--server.port=${PORT}"]
EOF
}

# Generate a .NET service
generate_dotnet_service() {
  local service_name="$1"
  local service_path="services/dotnet/${service_name}"

  # Create directory structure
  mkdir -p "$service_path"

  # Create .csproj file
  cat >"$service_path/${service_name}.csproj" <<'EOF'
<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <Nullable>enable</Nullable>
    <ImplicitUsings>enable</ImplicitUsings>
  </PropertyGroup>
</Project>
EOF

  # Create Program.cs
  cat >"$service_path/Program.cs" <<'EOF'
var builder = WebApplication.CreateBuilder(args);
var app = builder.Build();

app.MapGet("/", () => new { message = "Hello from SERVICE_NAME service!" });
app.MapGet("/health", () => new { status = "ok", service = "SERVICE_NAME" });

var port = Environment.GetEnvironmentVariable("PORT") ?? "8000";
app.Urls.Add($"http://0.0.0.0:{port}");

app.Run();
EOF
  sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/Program.cs" && rm -f "$service_path/Program.cs.bak"

  # Create Dockerfile
  cat >"$service_path/Dockerfile" <<'EOF'
FROM mcr.microsoft.com/dotnet/sdk:8.0 AS builder
WORKDIR /app
COPY *.csproj ./
RUN dotnet restore
COPY . ./
RUN dotnet publish -c Release -o out

FROM mcr.microsoft.com/dotnet/aspnet:8.0
RUN apt-get update && apt-get install -y curl wget && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/out .
ARG PORT=8000
ENV PORT=${PORT}
ENV ASPNETCORE_URLS=http://0.0.0.0:${PORT}
EXPOSE ${PORT}
CMD ["dotnet", "SERVICE_NAME.dll"]
EOF
  sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/Dockerfile" && rm -f "$service_path/Dockerfile.bak"
}

# Generate a Ruby service
generate_ruby_service() {
  local service_name="$1"
  local service_path="services/ruby/${service_name}"

  # Create directory structure
  mkdir -p "$service_path"

  # Create Gemfile
  cat >"$service_path/Gemfile" <<'EOF'
source 'https://rubygems.org'

gem 'sinatra', '~> 3.0'
gem 'puma', '~> 6.0'
gem 'json', '~> 2.6'
EOF

  # Create app.rb
  cat >"$service_path/app.rb" <<'EOF'
require 'sinatra'
require 'json'

set :bind, '0.0.0.0'
set :port, ENV['PORT'] || 8000

get '/' do
  content_type :json
  { message: 'Hello from SERVICE_NAME service!' }.to_json
end

get '/health' do
  content_type :json
  { status: 'ok', service: 'SERVICE_NAME' }.to_json
end
EOF
  sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/app.rb" && rm -f "$service_path/app.rb.bak"

  # Create config.ru
  cat >"$service_path/config.ru" <<'EOF'
require './app'
run Sinatra::Application
EOF

  # Create Dockerfile
  cat >"$service_path/Dockerfile" <<'EOF'
FROM ruby:3.2-slim
RUN apt-get update && apt-get install -y build-essential curl wget && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY Gemfile Gemfile.lock* ./
RUN bundle install
COPY . .
ARG PORT=8000
ENV PORT=${PORT}
EXPOSE ${PORT}
CMD ["ruby", "app.rb"]
EOF
}

# Generate a PHP service
generate_php_service() {
  local service_name="$1"
  local service_path="services/php/${service_name}"

  # Create directory structure
  mkdir -p "$service_path/public"

  # Create composer.json
  cat >"$service_path/composer.json" <<EOF
{
    "name": "nself/${service_name}",
    "type": "project",
    "require": {
        "slim/slim": "^4.12",
        "slim/psr7": "^1.6"
    }
}
EOF

  # Create index.php
  cat >"$service_path/public/index.php" <<'EOF'
<?php
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Factory\AppFactory;

require __DIR__ . '/../vendor/autoload.php';

$app = AppFactory::create();

$app->get('/', function (Request $request, Response $response, $args) {
    $data = ['message' => 'Hello from SERVICE_NAME service!'];
    $response->getBody()->write(json_encode($data));
    return $response->withHeader('Content-Type', 'application/json');
});

$app->get('/health', function (Request $request, Response $response, $args) {
    $data = ['status' => 'ok', 'service' => 'SERVICE_NAME'];
    $response->getBody()->write(json_encode($data));
    return $response->withHeader('Content-Type', 'application/json');
});

$app->run();
EOF
  sed -i.bak "s/SERVICE_NAME/${service_name}/g" "$service_path/public/index.php" && rm -f "$service_path/public/index.php.bak"

  # Create Dockerfile
  cat >"$service_path/Dockerfile" <<'EOF'
FROM composer:2 as builder
WORKDIR /app
COPY composer.json composer.lock* ./
RUN composer install --no-dev --optimize-autoloader

FROM php:8.2-apache
RUN apt-get update && apt-get install -y curl wget && rm -rf /var/lib/apt/lists/*
RUN a2enmod rewrite
WORKDIR /var/www
COPY --from=builder /app/vendor ./vendor
COPY . .
RUN chown -R www-data:www-data /var/www
ENV APACHE_DOCUMENT_ROOT /var/www/public
RUN sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf
RUN sed -ri -e 's!/var/www/!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/apache2.conf /etc/apache2/conf-available/*.conf
ARG PORT=8000
ENV PORT=${PORT}
RUN sed -i "s/80/${PORT}/g" /etc/apache2/ports.conf /etc/apache2/sites-available/*.conf
EXPOSE ${PORT}
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

  # Parse NODEJS_SERVICES for basic Node.js services
  local nodejs_list="${NODEJS_SERVICES:-}"
  if [[ -n "$nodejs_list" ]]; then
    IFS=',' read -ra services <<<"$nodejs_list"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs) # Trim whitespace
      if [[ ! -d "services/node/$service" ]]; then
        generate_basic_node_service "$service"
        ((generated_count++))
      fi
    done
  fi

  # Parse NEST_SERVICES or NESTJS_SERVICES (support both)
  local nest_list="${NEST_SERVICES:-${NESTJS_SERVICES:-}}"
  if [[ -n "$nest_list" ]]; then
    IFS=',' read -ra services <<<"$nest_list"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs) # Trim whitespace
      if [[ ! -d "services/nest/$service" ]]; then
        generate_nest_service "$service"
        ((generated_count++))
      fi
    done
  fi

  # Parse BULL_SERVICES or BULLMQ_WORKERS (support both)
  local bull_list="${BULL_SERVICES:-${BULLMQ_WORKERS:-}}"
  if [[ -n "$bull_list" ]]; then
    IFS=',' read -ra services <<<"$bull_list"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs)
      if [[ ! -d "services/bull/$service" ]]; then
        generate_bull_service "$service"
        ((generated_count++))
      fi
    done
  fi

  # Parse GO_SERVICES or GOLANG_SERVICES
  local go_list="${GO_SERVICES:-${GOLANG_SERVICES:-}}"
  if [[ -n "$go_list" ]]; then
    IFS=',' read -ra services <<<"$go_list"
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
    IFS=',' read -ra services <<<"$PYTHON_SERVICES"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs)
      if [[ ! -d "services/py/$service" ]]; then
        generate_python_service "$service"
        ((generated_count++))
      fi
    done
  fi

  # Parse RUST_SERVICES
  if [[ -n "${RUST_SERVICES:-}" ]]; then
    IFS=',' read -ra services <<<"$RUST_SERVICES"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs)
      if [[ ! -d "services/rust/$service" ]]; then
        generate_rust_service "$service"
        ((generated_count++))
      fi
    done
  fi

  # Parse JAVA_SERVICES
  if [[ -n "${JAVA_SERVICES:-}" ]]; then
    IFS=',' read -ra services <<<"$JAVA_SERVICES"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs)
      if [[ ! -d "services/java/$service" ]]; then
        generate_java_service "$service"
        ((generated_count++))
      fi
    done
  fi

  # Parse DOTNET_SERVICES
  if [[ -n "${DOTNET_SERVICES:-}" ]]; then
    IFS=',' read -ra services <<<"$DOTNET_SERVICES"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs)
      if [[ ! -d "services/dotnet/$service" ]]; then
        generate_dotnet_service "$service"
        ((generated_count++))
      fi
    done
  fi

  # Parse RUBY_SERVICES
  if [[ -n "${RUBY_SERVICES:-}" ]]; then
    IFS=',' read -ra services <<<"$RUBY_SERVICES"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs)
      if [[ ! -d "services/ruby/$service" ]]; then
        generate_ruby_service "$service"
        ((generated_count++))
      fi
    done
  fi

  # Parse PHP_SERVICES
  if [[ -n "${PHP_SERVICES:-}" ]]; then
    IFS=',' read -ra services <<<"$PHP_SERVICES"
    for service in "${services[@]}"; do
      service=$(echo "$service" | xargs)
      if [[ ! -d "services/php/$service" ]]; then
        generate_php_service "$service"
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
    nest | nestjs)
      generate_nest_service_at "$service_name" "$actual_dir"
      ;;
    bull | bullmq)
      generate_bull_service_at "$service_name" "$actual_dir"
      ;;
    go | golang)
      generate_go_service_at "$service_name" "$actual_dir"
      ;;
    py | python)
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
  cat >"$service_path/package.json" <<'EOF'
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
  cat >"$service_path/src/main.ts" <<'EOF'
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

  cat >"$service_path/src/app.module.ts" <<'EOF'
import { Module } from '@nestjs/common';
import { AppController } from './app.controller';

@Module({
  imports: [],
  controllers: [AppController],
})
export class AppModule {}
EOF

  cat >"$service_path/src/app.controller.ts" <<'EOF'
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

  cat >"$service_path/tsconfig.json" <<'EOF'
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

  cat >"$service_path/Dockerfile" <<'EOF'
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

  cat >"$service_path/package.json" <<'EOF'
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

  cat >"$service_path/src/index.js" <<'EOF'
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

  cat >"$service_path/Dockerfile" <<'EOF'
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

  cat >"$service_path/go.mod" <<EOF
module ${service_name}

go 1.21

require github.com/gorilla/mux v1.8.0
EOF

  # Create go.sum with the mux dependency
  cat >"$service_path/go.sum" <<'EOF'
github.com/gorilla/mux v1.8.0 h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=
github.com/gorilla/mux v1.8.0/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
EOF

  cat >"$service_path/main.go" <<'EOF'
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

  cat >"$service_path/Dockerfile" <<'EOF'
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
  cat >"$service_path/package.json" <<'EOF'
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

  cat >"$service_path/index.js" <<'EOF'
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

  cat >"$service_path/Dockerfile" <<'EOF'
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

  cat >"$service_path/requirements.txt" <<'EOF'
fastapi==0.104.0
uvicorn==0.24.0
python-dotenv==1.0.0
EOF

  cat >"$service_path/main.py" <<'EOF'
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

  cat >"$service_path/Dockerfile" <<'EOF'
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
