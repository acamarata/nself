#!/usr/bin/env bash
# init-wizard-v2.sh - Practical configuration wizard for nself

# Determine directories
WIZARD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(dirname "$WIZARD_DIR")"
ROOT_DIR="$(dirname "$(dirname "$(dirname "$WIZARD_DIR")")")"

# Source required modules with existence checks
for module in "$WIZARD_DIR/prompts.sh" "$WIZARD_DIR/detection.sh" "$WIZARD_DIR/templates.sh"; do
  if [[ ! -f "$module" ]]; then
    echo "Error: Required wizard module not found: $module" >&2
    exit 1
  fi
  source "$module"
done

for lib in "$CLI_DIR/../lib/utils/display.sh" "$CLI_DIR/../lib/utils/env.sh" "$CLI_DIR/../lib/wizard/environment-manager.sh"; do
  if [[ ! -f "$lib" ]]; then
    echo "Error: Required library not found: $lib" >&2
    exit 1
  fi
  source "$lib"
done

# Main wizard function
run_config_wizard() {
  clear
  show_wizard_header "nself Configuration Wizard" "Setup Your Project Step by Step"
  
  echo "Welcome to nself! Let's configure your project."
  echo "This wizard will walk you through the essential settings."
  echo ""
  echo "📝 We'll configure:"
  echo "  • Project name and domain"
  echo "  • Database settings"
  echo "  • Service passwords"
  echo "  • Optional services (Redis, search, etc.)"
  echo "  • Custom services if needed"
  echo ""
  press_any_key
  
  # Configuration variables
  local config=()
  
  # ==========================================
  # STEP 1: Core Project Settings
  # ==========================================
  clear
  show_wizard_step 1 8 "Core Project Settings"
  
  echo "📋 Basic Configuration"
  echo ""
  
  # Project Name
  echo "Project name:"
  echo "  Used for: Docker containers, database names, resource prefixes"
  echo "  Format: lowercase letters, numbers, hyphens (e.g., my-app)"
  echo ""
  local project_name
  prompt_input "Project name" "myapp" project_name "^[a-z][a-z0-9-]*$"
  config+=("PROJECT_NAME=$project_name")
  
  echo ""
  
  # Environment Mode
  echo "Environment mode:"
  local env_options=(
    "dev - Development (debug tools, hot reload, verbose logging)"
    "prod - Production (optimized, secure, minimal logging)"
  )
  local selected_env
  select_option "Select environment" env_options selected_env
  local env_mode=$([[ $selected_env -eq 0 ]] && echo "dev" || echo "prod")
  config+=("ENV=$env_mode")
  
  echo ""
  
  # Base Domain
  echo "Base domain:"
  echo "  All services will be subdomains of this domain"
  echo "  Examples:"
  echo "    • Development: local.nself.org (automatic SSL)"
  echo "    • Production: yourdomain.com"
  echo ""
  local base_domain
  local default_domain=$([[ "$env_mode" == "dev" ]] && echo "local.nself.org" || echo "yourdomain.com")
  prompt_input "Base domain" "$default_domain" base_domain
  config+=("BASE_DOMAIN=$base_domain")
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 2: Database Configuration
  # ==========================================
  clear
  show_wizard_step 2 8 "Database Configuration"
  
  echo "🗄️ PostgreSQL Database Setup"
  echo ""
  
  # Database Name
  echo "Database name:"
  local db_name
  prompt_input "Database name" "${project_name//-/_}" db_name "^[a-z][a-z0-9_]*$"
  config+=("POSTGRES_DB=$db_name")
  
  echo ""
  
  # Database User
  echo "Database user:"
  local db_user
  prompt_input "Database user" "postgres" db_user
  config+=("POSTGRES_USER=$db_user")
  
  echo ""
  
  # Database Password
  echo "Database password:"
  echo ""
  local password_options=(
    "Generate secure password (recommended)"
    "Use simple password for development"
    "Let me set a custom password"
  )
  local selected_password
  select_option "Password option" password_options selected_password
  
  local db_password
  case $selected_password in
    0)
      db_password=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
      log_success "Generated secure password: $db_password"
      ;;
    1)
      db_password="${project_name}-dev-password"
      log_info "Using development password: $db_password"
      ;;
    2)
      echo -n "Enter password: "
      read -s db_password
      echo ""
      ;;
  esac
  config+=("POSTGRES_PASSWORD=$db_password")
  
  echo ""
  
  # PostgreSQL Extensions
  echo "PostgreSQL extensions to enable:"
  local extensions=(
    "uuid-ossp - UUID generation (recommended)"
    "pgcrypto - Cryptographic functions"
    "postgis - Geographic objects"
    "pg_trgm - Text similarity"
    "hstore - Key-value storage"
    "citext - Case-insensitive text"
  )
  local selected_extensions=()
  multi_select extensions selected_extensions
  
  if [[ ${#selected_extensions[@]} -gt 0 ]]; then
    local ext_string=$(IFS=,; echo "${selected_extensions[@]}" | sed 's/ - [^,]*//g')
    config+=("POSTGRES_EXTENSIONS=$ext_string")
  else
    config+=("POSTGRES_EXTENSIONS=uuid-ossp")
  fi
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 3: Core Services Passwords
  # ==========================================
  clear
  show_wizard_step 3 8 "Service Passwords"
  
  echo "🔐 Service Authentication"
  echo ""
  
  # Hasura Admin Secret
  echo "Hasura GraphQL admin secret:"
  local hasura_options=(
    "Generate secure secret (recommended)"
    "Use simple secret for development"
    "Set custom secret"
  )
  local selected_hasura
  select_option "Hasura admin secret" hasura_options selected_hasura
  
  local hasura_secret
  case $selected_hasura in
    0)
      hasura_secret=$(openssl rand -hex 32)
      log_success "Generated secure secret"
      ;;
    1)
      hasura_secret="hasura-admin-secret-dev"
      log_info "Using development secret"
      ;;
    2)
      echo -n "Enter secret: "
      read -s hasura_secret
      echo ""
      ;;
  esac
  config+=("HASURA_GRAPHQL_ADMIN_SECRET=$hasura_secret")
  
  echo ""
  
  # JWT Key
  echo "JWT signing key:"
  local jwt_options=(
    "Generate secure key (recommended)"
    "Use development key"
    "Set custom key"
  )
  local selected_jwt
  select_option "JWT key" jwt_options selected_jwt
  
  local jwt_key
  case $selected_jwt in
    0)
      jwt_key=$(openssl rand -base64 64 | tr -d '\n')
      log_success "Generated secure JWT key"
      ;;
    1)
      jwt_key="development-secret-key-minimum-32-characters-long"
      log_info "Using development key"
      ;;
    2)
      echo -n "Enter JWT key (min 32 chars): "
      read -s jwt_key
      echo ""
      ;;
  esac
  config+=("HASURA_JWT_KEY=$jwt_key")
  config+=("HASURA_JWT_TYPE=HS256")
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 4: Core Services Selection
  # ==========================================
  clear
  show_wizard_step 4 8 "Core Services"
  
  echo "📦 Core Services (always enabled):"
  echo ""
  echo "  ✅ PostgreSQL - Primary database"
  echo "  ✅ Hasura - GraphQL API engine"
  echo "  ✅ Auth - Authentication service"
  echo "  ✅ Storage - File storage service"
  echo "  ✅ Nginx - Reverse proxy & SSL"
  echo ""
  
  # Ask about console access
  echo "Enable Hasura console? (GraphQL playground)"
  local console_options=("Yes - Enable console" "No - Disable for production")
  local selected_console
  select_option "Hasura console" console_options selected_console
  local enable_console=$([[ $selected_console -eq 0 ]] && echo "true" || echo "false")
  config+=("HASURA_GRAPHQL_ENABLE_CONSOLE=$enable_console")
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 5: Optional Services
  # ==========================================
  clear
  show_wizard_step 5 8 "Optional Services"
  
  echo "🔧 Optional Services"
  echo "Select services you want to enable (space to select, enter to confirm):"
  echo ""
  
  local optional_services=(
    "Redis - Caching and sessions"
    "MailPit - Local email testing (dev only)"
    "Functions - Serverless functions"
    "Config Server - Configuration UI"
    "Dashboard - Admin dashboard"
    "Monitoring - Prometheus & Grafana"
  )
  
  local selected_optional=()
  multi_select optional_services selected_optional
  
  # Process selections
  for service in "${selected_optional[@]}"; do
    case "$service" in
      *Redis*)
        config+=("REDIS_ENABLED=true")
        config+=("REDIS_VERSION=7-alpine")
        ;;
      *MailPit*)
        if [[ "$env_mode" == "dev" ]]; then
          config+=("MAILPIT_ENABLED=true")
        fi
        ;;
      *Functions*)
        config+=("FUNCTIONS_ENABLED=true")
        ;;
      *Dashboard*)
        config+=("DASHBOARD_ENABLED=true")
        ;;
      *Monitoring*)
        config+=("MONITORING_ENABLED=true")
        ;;
    esac
  done
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 6: Search Service
  # ==========================================
  clear
  show_wizard_step 6 8 "Search Service"
  
  echo "🔍 Search Service Configuration"
  echo ""
  echo "Do you need search functionality?"
  
  local search_options=(
    "No search needed"
    "PostgreSQL full-text search (built-in, simple)"
    "MeiliSearch (fast, typo-tolerant)"
    "Elasticsearch (powerful, complex)"
    "Typesense (modern, developer-friendly)"
  )
  
  local selected_search
  select_option "Search engine" search_options selected_search
  
  case $selected_search in
    1)
      config+=("SEARCH_ENGINE=postgres")
      log_info "Will use PostgreSQL full-text search"
      ;;
    2)
      config+=("SEARCH_ENGINE=meilisearch")
      config+=("SEARCH_ENABLED=true")
      # Generate API key
      local search_key=$(openssl rand -hex 32)
      config+=("SEARCH_API_KEY=$search_key")
      log_success "MeiliSearch configured"
      ;;
    3)
      config+=("SEARCH_ENGINE=elasticsearch")
      config+=("SEARCH_ENABLED=true")
      log_info "Elasticsearch configured"
      ;;
    4)
      config+=("SEARCH_ENGINE=typesense")
      config+=("SEARCH_ENABLED=true")
      local search_key=$(openssl rand -hex 32)
      config+=("SEARCH_API_KEY=$search_key")
      log_success "Typesense configured"
      ;;
  esac
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 7: Custom Services
  # ==========================================
  clear
  show_wizard_step 7 8 "Custom Services"
  
  echo "🚀 Custom Services"
  echo ""
  echo "Do you have custom services to add?"
  echo "(You can code them in any language - Node.js, Python, Go, etc.)"
  echo ""
  
  echo -n "Add custom services? (y/N): "
  local add_custom
  read add_custom
  
  if [[ "$add_custom" == "y" ]] || [[ "$add_custom" == "Y" ]]; then
    echo ""
    echo "Custom services can be added to docker-compose.override.yml"
    echo "We'll create a template for you."
    echo ""
    
    echo "How many custom services do you want to add?"
    local num_services
    prompt_input "Number of services" "1" num_services "^[0-9]+$"
    
    local custom_services=()
    for ((i=1; i<=num_services; i++)); do
      echo ""
      echo "Service $i:"
      local service_name
      prompt_input "Service name" "custom-service-$i" service_name "^[a-z][a-z0-9-]*$"
      
      echo "Language/framework:"
      local lang_options=("Node.js" "Python" "Go" "Ruby" "Java" "Other")
      local selected_lang
      select_option "Select language" lang_options selected_lang
      
      local service_port
      prompt_input "Service port" "$((8000 + i))" service_port "^[0-9]+$"
      
      custom_services+=("$service_name:${lang_options[$selected_lang]}:$service_port")
    done
    
    # Store custom services info for docker-compose generation
    config+=("CUSTOM_SERVICES=${custom_services[*]}")
  fi
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 8: Review & Generate
  # ==========================================
  clear
  show_wizard_step 8 8 "Review Configuration"
  
  echo "📄 Configuration Summary"
  echo "========================"
  echo ""
  echo "Core Settings:"
  echo "  Project Name:  $project_name"
  echo "  Environment:   $env_mode"
  echo "  Base Domain:   $base_domain"
  echo "  Database:      $db_name"
  echo ""
  
  echo "Services Enabled:"
  echo "  ✅ PostgreSQL, Hasura, Auth, Storage, Nginx"
  if [[ " ${config[@]} " =~ "REDIS_ENABLED=true" ]]; then
    echo "  ✅ Redis"
  fi
  if [[ " ${config[@]} " =~ "MAILPIT_ENABLED=true" ]]; then
    echo "  ✅ MailPit (dev email)"
  fi
  if [[ " ${config[@]} " =~ "SEARCH_ENABLED=true" ]]; then
    echo "  ✅ Search ($([[ " ${config[@]} " =~ "meilisearch" ]] && echo "MeiliSearch" || echo "Other"))"
  fi
  if [[ " ${config[@]} " =~ "CUSTOM_SERVICES=" ]]; then
    echo "  ✅ Custom services configured"
  fi
  
  echo ""
  echo "Security:"
  if [[ "$db_password" == *"openssl"* ]] || [[ ${#db_password} -gt 20 ]]; then
    echo "  🔒 Secure passwords generated"
  else
    echo "  ⚠️  Using development passwords"
  fi
  
  echo ""
  echo -n "Generate this configuration? (Y/n): "
  local confirm
  read confirm
  confirm="${confirm:-y}"
  
  if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
    log_info "Configuration cancelled"
    return 1
  fi
  
  # Generate .env.local
  echo ""
  log_info "Generating .env.local..."
  
  {
    echo "# nself Configuration"
    echo "# Generated by wizard on $(date)"
    echo ""
    echo "# Core Settings"
    for item in "${config[@]}"; do
      echo "$item"
    done
  } > .env.local
  
  log_success "Configuration generated successfully!"
  
  # Generate docker-compose.override.yml if custom services
  if [[ " ${config[@]} " =~ "CUSTOM_SERVICES=" ]]; then
    log_info "Generating docker-compose.override.yml for custom services..."
    generate_custom_services_compose "$custom_services"
  fi
  
  echo ""
  echo "✅ Setup complete!"
  echo ""
  echo "Next steps:"
  echo "  1. Run: nself build"
  echo "  2. Run: nself start"
  echo "  3. Access your services at: https://api.$base_domain"
  echo ""
  
  # Save passwords if secure ones were generated
  if [[ "$db_password" == *"openssl"* ]] || [[ ${#db_password} -gt 20 ]]; then
    echo "⚠️  Important: Save these generated passwords:"
    echo ""
    echo "PostgreSQL Password: $db_password"
    if [[ -n "$hasura_secret" ]] && [[ ${#hasura_secret} -gt 30 ]]; then
      echo "Hasura Admin Secret: $hasura_secret"
    fi
    echo ""
    echo "These are saved in .env.local but you may want to store them securely."
  fi
}

# Generate docker-compose.override.yml for custom services
generate_custom_services_compose() {
  local services=("$@")
  
  cat > docker-compose.override.yml << 'EOF'
# Custom Services Configuration
# Generated by nself wizard

version: '3.8'

services:
EOF
  
  for service_info in "${services[@]}"; do
    IFS=':' read -r name language port <<< "$service_info"
    
    cat >> docker-compose.override.yml << EOF
  $name:
    build: ./services/$name
    ports:
      - "$port:$port"
    environment:
      - NODE_ENV=\${ENV}
      - DATABASE_URL=postgresql://\${POSTGRES_USER}:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}
    networks:
      - nself
    restart: unless-stopped
    volumes:
      - ./services/$name:/app
    depends_on:
      - postgres

EOF
    
    # Create service directory with basic template
    mkdir -p "services/$name"
    
    case "$language" in
      "Node.js")
        create_nodejs_template "$name" "$port"
        ;;
      "Python")
        create_python_template "$name" "$port"
        ;;
      "Go")
        create_go_template "$name" "$port"
        ;;
      *)
        create_generic_template "$name" "$port"
        ;;
    esac
  done
  
  cat >> docker-compose.override.yml << 'EOF'
networks:
  nself:
    external: true
EOF
}

# Create Node.js service template
create_nodejs_template() {
  local name="$1"
  local port="$2"
  
  cat > "services/$name/package.json" << EOF
{
  "name": "$name",
  "version": "1.0.0",
  "main": "index.js",
  "scripts": {
    "start": "node index.js",
    "dev": "nodemon index.js"
  },
  "dependencies": {
    "express": "^4.18.0",
    "pg": "^8.11.0"
  },
  "devDependencies": {
    "nodemon": "^3.0.0"
  }
}
EOF
  
  cat > "services/$name/index.js" << EOF
const express = require('express');
const { Pool } = require('pg');

const app = express();
const port = $port;

// Database connection
const pool = new Pool({
  connectionString: process.env.DATABASE_URL
});

app.use(express.json());

app.get('/health', (req, res) => {
  res.json({ status: 'healthy', service: '$name' });
});

app.get('/', async (req, res) => {
  res.json({ 
    message: 'Custom service $name is running',
    timestamp: new Date().toISOString()
  });
});

app.listen(port, () => {
  console.log(\`$name service listening on port \${port}\`);
});
EOF
  
  cat > "services/$name/Dockerfile" << EOF
FROM node:18-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE $port
CMD ["npm", "start"]
EOF
}

# Create Python service template
create_python_template() {
  local name="$1"
  local port="$2"
  
  cat > "services/$name/requirements.txt" << EOF
flask==3.0.0
psycopg2-binary==2.9.9
gunicorn==21.2.0
EOF
  
  cat > "services/$name/app.py" << EOF
from flask import Flask, jsonify
import os
import psycopg2
from datetime import datetime

app = Flask(__name__)
port = $port

# Database connection
DATABASE_URL = os.environ.get('DATABASE_URL')

@app.route('/health')
def health():
    return jsonify({'status': 'healthy', 'service': '$name'})

@app.route('/')
def index():
    return jsonify({
        'message': 'Custom service $name is running',
        'timestamp': datetime.now().isoformat()
    })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=port, debug=os.environ.get('ENV') == 'dev')
EOF
  
  cat > "services/$name/Dockerfile" << EOF
FROM python:3.11-alpine
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
EXPOSE $port
CMD ["gunicorn", "-b", "0.0.0.0:$port", "app:app"]
EOF
}

# Create Go service template
create_go_template() {
  local name="$1"
  local port="$2"
  
  cat > "services/$name/go.mod" << EOF
module $name

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/lib/pq v1.10.9
)
EOF
  
  cat > "services/$name/main.go" << EOF
package main

import (
    "database/sql"
    "fmt"
    "net/http"
    "os"
    "time"
    
    "github.com/gin-gonic/gin"
    _ "github.com/lib/pq"
)

func main() {
    r := gin.Default()
    
    // Database connection
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        fmt.Println("Database connection error:", err)
    }
    defer db.Close()
    
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "healthy",
            "service": "$name",
        })
    })
    
    r.GET("/", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "message": "Custom service $name is running",
            "timestamp": time.Now().Format(time.RFC3339),
        })
    })
    
    r.Run(":$port")
}
EOF
  
  cat > "services/$name/Dockerfile" << EOF
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE $port
CMD ["./main"]
EOF
}

# Create generic service template
create_generic_template() {
  local name="$1"
  local port="$2"
  
  cat > "services/$name/README.md" << EOF
# $name Service

Custom service running on port $port

## Setup

1. Add your application code here
2. Update the Dockerfile for your language/framework
3. Rebuild: nself build
4. Restart: nself restart

## Environment Variables

- DATABASE_URL: PostgreSQL connection string
- NODE_ENV/ENV: Environment mode (dev/prod)
- PORT: $port

## Endpoints

- GET /health - Health check
- GET / - Main endpoint
EOF
  
  cat > "services/$name/Dockerfile" << EOF
# Add your Dockerfile here for your chosen language/framework
# Example for a generic web service:

FROM alpine:latest
WORKDIR /app
# Add your build steps here
EXPOSE $port
# Add your CMD here
EOF
}

# Execute if run directly
# Disabled due to syntax error - wizard is called from init.sh
# if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
#   run_config_wizard
# fi