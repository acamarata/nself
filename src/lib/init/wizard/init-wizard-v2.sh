#!/usr/bin/env bash
# init-wizard-v2.sh - Practical configuration wizard for nself

# Determine directories
WIZARD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INIT_LIB_DIR="$(dirname "$WIZARD_DIR")"
ROOT_DIR="$(dirname "$(dirname "$(dirname "$INIT_LIB_DIR")")")"

# Source required modules with existence checks
for module in "$WIZARD_DIR/prompts.sh" "$WIZARD_DIR/detection.sh" "$WIZARD_DIR/templates.sh" "$WIZARD_DIR/hosts-helper.sh"; do
  if [[ ! -f "$module" ]]; then
    echo "Error: Required wizard module not found: $module" >&2
    exit 1
  fi
  source "$module"
done

# Source from lib/utils and lib/wizard
for lib in "$INIT_LIB_DIR/../utils/display.sh" "$INIT_LIB_DIR/../utils/env.sh" "$INIT_LIB_DIR/../wizard/environment-manager.sh"; do
  if [[ ! -f "$lib" ]]; then
    echo "Error: Required library not found: $lib" >&2
    exit 1
  fi
  source "$lib"
done

# Main wizard function
run_config_wizard() {
  # Set up trap for Ctrl+C
  trap 'echo ""; echo ""; log_info "Wizard cancelled"; echo "Run nself init --wizard to try again."; echo ""; exit 0' INT TERM

  clear
  show_wizard_header "nself Configuration Wizard" "Setup Your Project Step by Step"

  echo "Welcome to nself! Let's configure your project."
  echo "This wizard will walk you through the essential settings."
  echo ""
  echo "📝 We'll configure:"
  echo "  • Project name and domain"
  echo "  • Database settings"
  echo "  • Service passwords"
  echo "  • Admin dashboard"
  echo "  • Optional services (Redis, search, etc.)"
  echo "  • Custom backend services"
  echo "  • Frontend applications"
  echo ""
  echo "(Press Ctrl+C anytime to exit)"
  echo ""
  press_any_key
  
  # Configuration variables
  local config=()
  
  # ==========================================
  # STEP 1: Core Project Settings
  # ==========================================
  clear
  show_wizard_step 1 10 "Core Project Settings"
  
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
  echo ""

  local base_domain
  if [[ "$env_mode" == "dev" ]]; then
    echo "Development domain:"
    local domain_options=(
      "local.nself.org (recommended) - Zero configuration, automatic SSL"
      "localhost - Works everywhere, SSL auto-configured"
      "Custom prefix - e.g., myapp.local.nself.org or myapp.localhost"
    )
    local selected_domain
    select_option "Select domain" domain_options selected_domain

    case $selected_domain in
      0)
        base_domain="local.nself.org"
        log_success "Using local.nself.org"
        echo "  ✓ Automatic wildcard SSL certificate"
        echo "  ✓ No setup required - works immediately"
        echo "  ✓ Services: api.local.nself.org, admin.local.nself.org"
        ;;
      1)
        base_domain="localhost"
        log_success "Using localhost"
        echo "  ✓ Works on all systems"
        echo "  ✓ Subdomains: api.localhost, admin.localhost"
        echo "  ✓ SSL configured automatically during build"
        ;;
      2)
        echo ""
        echo "Custom prefix options:"
        echo "  • [prefix].local.nself.org - No setup needed"
        echo "  • [prefix].localhost - Auto-resolves in browsers"
        echo ""
        echo "Examples: myapp.local.nself.org, test.localhost"
        echo ""

        local custom_domain
        prompt_input "Enter your domain" "myapp.local.nself.org" custom_domain

        # Validate and normalize
        if [[ "$custom_domain" =~ \.local\.nself\.org$ ]]; then
          base_domain="$custom_domain"
          log_success "Using $base_domain"
          echo "  ✓ Works immediately with automatic SSL"
        elif [[ "$custom_domain" =~ \.localhost$ ]]; then
          base_domain="$custom_domain"
          log_success "Using $base_domain"
          echo "  ✓ Auto-resolves in modern browsers"
          echo "  ✓ SSL configured during build"
        else
          # Default to prefixing local.nself.org
          base_domain="${custom_domain}.local.nself.org"
          log_info "Using $base_domain"
          echo "  ✓ Prefixed to local.nself.org for zero config"
        fi
        ;;
    esac
  else
    # Production mode
    echo "Production domain (e.g., yourdomain.com):"
    prompt_input "Base domain" "yourdomain.com" base_domain
  fi
  config+=("BASE_DOMAIN=$base_domain")
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 2: Database Configuration
  # ==========================================
  clear
  show_wizard_step 2 10 "Database Configuration"
  
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

  local db_password
  if [[ "$env_mode" == "dev" ]]; then
    local password_options=(
      "Use simple development password (recommended for dev)"
      "Generate secure password"
      "Let me set a custom password"
    )
    local selected_password
    select_option "Password option" password_options selected_password

    case $selected_password in
      0)
        db_password="${project_name}-dev-password"
        log_info "Using development password: $db_password"
        ;;
      1)
        db_password=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
        log_success "Generated secure password: $db_password"
        ;;
      2)
        echo -n "Enter password: "
        read -s db_password
        echo ""
        ;;
    esac
  else
    # Production mode - always generate secure
    db_password=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)
    log_success "Generated secure password for production: $db_password"
  fi
  config+=("POSTGRES_PASSWORD=$db_password")
  
  echo ""
  
  # PostgreSQL Extensions
  echo "PostgreSQL extensions to enable:"
  echo "(Select multiple with space, enter to confirm)"
  echo ""
  local extensions=(
    "uuid-ossp - UUID generation (recommended for all apps)"
    "pgcrypto - Encryption/hashing for secure apps"
    "pg_trgm - Full-text search and fuzzy matching"
    "timescaledb - Time-series data (IoT, metrics, logs)"
    "pgvector - Vector similarity for AI/ML embeddings"
    "postgis - Geographic/location data and queries"
    "pg_stat_statements - Query performance monitoring"
    "hstore - Key-value pairs within PostgreSQL"
    "pg_cron - Schedule database jobs"
    "btree_gin - Better indexing for full-text search"
  )
  local selected_extensions=()
  multi_select extensions selected_extensions

  # Always include uuid-ossp
  local has_uuid=false
  for ext in "${selected_extensions[@]}"; do
    if [[ "$ext" == *"uuid-ossp"* ]]; then
      has_uuid=true
      break
    fi
  done

  if [[ ${#selected_extensions[@]} -gt 0 ]]; then
    local ext_string=$(IFS=,; echo "${selected_extensions[@]}" | sed 's/ - [^,]*//g')
    if [[ "$has_uuid" == false ]]; then
      ext_string="uuid-ossp,$ext_string"
    fi
    config+=("POSTGRES_EXTENSIONS=$ext_string")
  else
    config+=("POSTGRES_EXTENSIONS=uuid-ossp,pgcrypto,pg_trgm")
  fi
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 3: Required Services
  # ==========================================
  clear
  show_wizard_step 3 10 "Required Services"

  echo "📦 Core Services (always enabled):"
  echo ""
  echo "  ✅ PostgreSQL - Primary database"
  echo "  ✅ Hasura - GraphQL API engine"
  echo "  ✅ Auth - Authentication service"
  echo "  ✅ Storage - File storage (MinIO)"
  echo "  ✅ Nginx - Reverse proxy & SSL"
  echo ""
  echo "These services form the foundation of your backend."
  echo ""
  press_any_key

  # ==========================================
  # STEP 4: Service Authentication
  # ==========================================
  clear
  show_wizard_step 4 10 "Service Passwords"

  echo "🔐 Service Authentication"
  echo ""
  
  # Hasura Admin Secret
  echo "Hasura GraphQL admin secret:"

  local hasura_secret
  if [[ "$env_mode" == "dev" ]]; then
    # Dev mode - default to simple
    hasura_secret="hasura-admin-secret-dev"
    log_info "Using development secret: $hasura_secret"
  else
    # Production - generate secure
    hasura_secret=$(openssl rand -hex 32)
    log_success "Generated secure secret for production"
  fi
  config+=("HASURA_GRAPHQL_ADMIN_SECRET=$hasura_secret")
  
  echo ""
  
  # JWT Key
  echo ""
  echo "JWT signing key:"

  local jwt_key
  if [[ "$env_mode" == "dev" ]]; then
    jwt_key="development-secret-key-minimum-32-characters-long"
    log_info "Using development JWT key"
  else
    jwt_key=$(openssl rand -base64 64 | tr -d '\n')
    log_success "Generated secure JWT key for production"
  fi
  config+=("HASURA_JWT_KEY=$jwt_key")
  config+=("HASURA_JWT_TYPE=HS256")
  
  echo ""
  press_any_key
  
  # Remove duplicate step 4 - merge with step 3
  
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

  local enable_console
  if [[ "$env_mode" == "dev" ]]; then
    enable_console="true"
    log_info "Console enabled for development"
  else
    echo -n "Enable in production? (y/N): "
    local console_choice
    read console_choice
    if [[ "$console_choice" == "y" ]] || [[ "$console_choice" == "Y" ]]; then
      enable_console="true"
      log_warning "Console enabled in production - ensure it's protected"
    else
      enable_console="false"
      log_info "Console disabled for production"
    fi
  fi
  config+=("HASURA_GRAPHQL_ENABLE_CONSOLE=$enable_console")
  
  echo ""
  press_any_key
  
  # Admin dashboard continues from step 3

  echo "🎛️ nself Admin Dashboard"
  echo ""
  echo "The admin dashboard provides:"
  echo "  • Visual service management"
  echo "  • Database schema editor"
  echo "  • API testing tools"
  echo "  • Log viewer"
  echo "  • Configuration management"
  echo ""

  echo -n "Enable admin dashboard? (Y/n): "
  local enable_admin
  read enable_admin
  enable_admin="${enable_admin:-y}"

  if [[ "$enable_admin" == "y" ]] || [[ "$enable_admin" == "Y" ]]; then
    config+=("DASHBOARD_ENABLED=true")
    log_success "Admin dashboard enabled"
  else
    config+=("DASHBOARD_ENABLED=false")
  fi

  echo ""
  press_any_key

  # ==========================================
  # STEP 5: Admin Dashboard
  # ==========================================
  clear
  show_wizard_step 5 10 "Admin Dashboard"

  echo "🎛️ nself Admin Dashboard"
  echo ""
  echo "Web-based management interface provides:"
  echo "  • Service health monitoring"
  echo "  • Database schema management"
  echo "  • GraphQL API explorer"
  echo "  • Real-time logs viewer"
  echo "  • Docker container management"
  echo ""

  echo -n "Enable nself admin dashboard? (Y/n): "
  local enable_admin
  read enable_admin
  enable_admin="${enable_admin:-y}"

  if [[ "$enable_admin" == "y" ]] || [[ "$enable_admin" == "Y" ]]; then
    config+=("NSELF_ADMIN_ENABLED=true")
    log_success "Admin dashboard enabled"
  else
    config+=("NSELF_ADMIN_ENABLED=false")
  fi

  echo ""
  press_any_key

  # ==========================================
  # STEP 6: Optional Services
  # ==========================================
  clear
  show_wizard_step 6 10 "Optional Services"

  echo "🔧 Optional Services"
  echo "Select services you want to enable (space to select, enter to confirm):"
  echo ""
  
  local optional_services=(
    "Redis - Caching, sessions, and queues"
    "BullMQ - Job queue management (requires Redis)"
    "Functions - Serverless functions runtime"
    "MLflow - ML experiment tracking"
    "Temporal - Workflow orchestration"
    "Monitoring - Prometheus & Grafana"
  )
  
  local selected_optional=()
  multi_select optional_services selected_optional
  
  # Process selections (handle empty array safely)
  if [[ ${#selected_optional[@]} -gt 0 ]]; then
    for service in "${selected_optional[@]}"; do
    case "$service" in
      *Redis*)
        config+=("REDIS_ENABLED=true")
        config+=("REDIS_VERSION=7-alpine")
        ;;
      *BullMQ*)
        config+=("BULLMQ_ENABLED=true")
        config+=("REDIS_ENABLED=true")  # BullMQ needs Redis
        ;;
      *MLflow*)
        config+=("MLFLOW_ENABLED=true")
        ;;
      *Temporal*)
        config+=("TEMPORAL_ENABLED=true")
        ;;
      *Functions*)
        config+=("FUNCTIONS_ENABLED=true")
        ;;
      *Monitoring*)
        config+=("MONITORING_ENABLED=true")
        ;;
    esac
    done
  fi
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 7: Email & Search
  # ==========================================
  clear
  show_wizard_step 7 10 "Email & Search"

  # Email configuration
  echo "📧 Email Service"
  echo ""
  if [[ "$env_mode" == "dev" ]]; then
    echo -n "Enable MailPit for local email testing? (Y/n): "
    local enable_mailpit
    read enable_mailpit
    enable_mailpit="${enable_mailpit:-y}"

    if [[ "$enable_mailpit" == "y" ]] || [[ "$enable_mailpit" == "Y" ]]; then
      config+=("MAILPIT_ENABLED=true")
      log_success "MailPit enabled at mailpit.$base_domain"
    fi
  else
    echo "Configure production email provider later in .env"
    config+=("# EMAIL_PROVIDER=sendgrid")
    config+=("# EMAIL_API_KEY=your-api-key")
  fi

  echo ""

  # Search configuration
  echo "🔍 Search Service"
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
  # STEP 8: Frontend Applications
  # ==========================================
  clear
  show_wizard_step 8 10 "Custom Backend Services"

  echo "🎨 Frontend Applications"
  echo ""
  echo "Do you have frontend applications to configure?"
  echo "(Next.js, React, Vue, Angular, Svelte, etc.)"
  echo ""

  echo -n "Add custom services? (y/N): "
  local add_custom
  read add_custom

  if [[ "$add_custom" == "y" ]] || [[ "$add_custom" == "Y" ]]; then
    echo ""
    echo "How many frontend applications?"
    local num_apps
    prompt_input "Number of apps" "1" num_apps "^[0-9]+$"

    local frontend_apps=()
    for ((i=1; i<=num_apps; i++)); do
      echo ""
      echo "Frontend App $i:"
      local app_name
      prompt_input "App name" "frontend-$i" app_name "^[a-z][a-z0-9-]*$"

      echo "Framework:"
      local framework_options=("Next.js" "React (CRA)" "React (Vite)" "Vue" "Angular" "Svelte" "Static HTML" "Other")
      local selected_framework
      select_option "Select framework" framework_options selected_framework

      # Smart port assignment based on framework
      local default_port
      case ${framework_options[$selected_framework]} in
        "Next.js") default_port=$((3000 + i - 1)) ;;
        "React (CRA)") default_port=$((3000 + i - 1)) ;;
        "React (Vite)") default_port=$((5173 + i - 1)) ;;
        "Vue") default_port=$((8080 + i - 1)) ;;
        "Angular") default_port=$((4200 + i - 1)) ;;
        "Svelte") default_port=$((5173 + i - 1)) ;;
        *) default_port=$((3000 + i - 1)) ;;
      esac

      local app_port
      prompt_input "Dev server port" "$default_port" app_port "^[0-9]+$"

      # Handle subdomain based on base_domain type
      if [[ "$base_domain" == "localhost" ]]; then
        log_info "For localhost, app will be at http://localhost:$app_port"
        frontend_apps+=("$app_name:${framework_options[$selected_framework]}:$app_port:localhost")
      else
        echo "Subdomain (e.g., 'app' for app.$base_domain):"
        local default_subdomain
        if [[ $i -eq 1 ]]; then
          # First app gets main subdomain
          default_subdomain="$([[ "$app_name" == "admin" ]] && echo "admin" || echo "app")"
        else
          default_subdomain="${app_name//-/}"
        fi
        local subdomain
        prompt_input "Subdomain" "$default_subdomain" subdomain "^[a-z][a-z0-9-]*$"
        frontend_apps+=("$app_name:${framework_options[$selected_framework]}:$app_port:$subdomain")
      fi
    done

    # Store frontend apps configuration
    config+=("FRONTEND_APPS=${frontend_apps[*]}")
  fi

  echo ""
  press_any_key

  # ==========================================
  # STEP 9: Custom Services
  # ==========================================
  clear
  show_wizard_step 9 10 "Frontend Applications"
  
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
    echo "Custom services will be added to the services/ directory"
    echo "Each service gets its own subdirectory with Dockerfile"
    echo ""

    echo "How many custom services do you want to add?"
    local num_services
    prompt_input "Number of services" "1" num_services "^[0-9]+$"

    local custom_services=()
    for ((i=1; i<=num_services; i++)); do
      echo ""
      echo "Service $i:"
      local service_name
      prompt_input "Service name" "api-service-$i" service_name "^[a-z][a-z0-9-]*$"

      echo "Service type:"
      local service_types=(
        "API Service - REST/GraphQL backend"
        "Worker Service - Background jobs/queues"
        "WebSocket Service - Real-time connections"
        "Proxy Service - API gateway/middleware"
        "ML Service - Machine learning/AI"
        "Data Service - ETL/processing"
        "Other - Custom service"
      )
      local selected_type
      select_option "Select type" service_types selected_type

      echo "Language/framework:"
      local lang_options
      case $selected_type in
        0|3) # API or Proxy
          lang_options=("Node.js Express" "Node.js Fastify" "Python FastAPI" "Python Flask" "Go Gin" "Go Fiber" "Ruby Rails" "Java Spring" "Rust Actix" "C# ASP.NET")
          ;;
        1|5) # Worker or Data
          lang_options=("Node.js" "Python" "Go" "Java" "Rust" "Ruby")
          ;;
        2) # WebSocket
          lang_options=("Node.js Socket.io" "Node.js WS" "Python WebSockets" "Go Gorilla" "Java Spring WebSocket")
          ;;
        4) # ML
          lang_options=("Python TensorFlow" "Python PyTorch" "Python Scikit-learn" "Python FastAPI + ML" "R Plumber")
          ;;
        *) # Other
          lang_options=("Node.js" "Python" "Go" "Ruby" "Java" "Rust" "C#/.NET" "PHP" "Other")
          ;;
      esac

      local selected_lang
      select_option "Select language" lang_options selected_lang

      # Smart port assignment
      local default_port
      case $selected_type in
        0) default_port=$((8000 + i)) ;;  # API services
        1) default_port=0 ;;              # Workers don't need ports
        2) default_port=$((3001 + i)) ;;  # WebSocket services
        3) default_port=$((8080 + i)) ;;  # Proxy services
        4) default_port=$((5000 + i)) ;;  # ML services
        5) default_port=0 ;;              # Data services
        *) default_port=$((8000 + i)) ;;
      esac

      local service_port=""
      if [[ $default_port -ne 0 ]]; then
        prompt_input "Service port" "$default_port" service_port "^[0-9]+$"
      else
        log_info "Worker/data service - no port needed"
        service_port="none"
      fi

      # Ask about database access
      echo -n "Will this service need database access? (Y/n): "
      local needs_db
      read needs_db
      needs_db="${needs_db:-y}"

      # Store service configuration
      custom_services+=("$service_name:${lang_options[$selected_lang]}:$service_port:$needs_db:${service_types[$selected_type]}")
    done

    # Store custom services info for docker-compose generation
    config+=("CUSTOM_SERVICES=${custom_services[*]}")
    config+=("CUSTOM_SERVICES_COUNT=$num_services")
  fi
  
  echo ""
  press_any_key
  
  # ==========================================
  # STEP 10: Review & Generate
  # ==========================================
  clear
  show_wizard_step 10 10 "Review Configuration"
  
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
  if [[ " ${config[@]} " =~ "DASHBOARD_ENABLED=true" ]]; then
    echo "  ✅ Admin Dashboard"
  fi
  if [[ " ${config[@]} " =~ "REDIS_ENABLED=true" ]]; then
    echo "  ✅ Redis"
  fi
  if [[ " ${config[@]} " =~ "MAILPIT_ENABLED=true" ]]; then
    echo "  ✅ MailPit (dev email)"
  fi
  if [[ " ${config[@]} " =~ "SEARCH_ENABLED=true" ]]; then
    echo "  ✅ Search ($([[ " ${config[@]} " =~ "meilisearch" ]] && echo "MeiliSearch" || echo "Other"))"
  fi
  if [[ " ${config[@]} " =~ "FRONTEND_APPS=" ]]; then
    echo "  ✅ Frontend applications configured"
  fi
  if [[ " ${config[@]} " =~ "CUSTOM_SERVICES=" ]]; then
    echo "  ✅ Custom backend services configured"
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
  
  # Generate configuration files
  echo ""
  log_info "Generating configuration files..."

  # For dev environment, put config in .env.dev
  if [[ "$env_mode" == "dev" ]]; then
    # Generate .env.dev with all wizard configuration
    {
      echo "# nself Development Configuration"
      echo "# Generated by wizard on $(date)"
      echo "# This file contains team-shared development settings"
      echo "# Can be overridden in .env or .env.local"
      echo ""
      echo "# Core Settings"
      for item in "${config[@]}"; do
        echo "$item"
      done
    } > .env.dev
    log_success "Created .env.dev (wizard configuration)"

    # Create mostly empty .env for local overrides
    {
      echo "# nself Local Configuration Overrides"
      echo "# Add your personal overrides here (higher priority than .env.dev)"
      echo "# Generated on $(date)"
      echo ""
      echo "# Example: Override project name"
      echo "# PROJECT_NAME=my-custom-name"
      echo ""
      echo "# Example: Use different ports"
      echo "# POSTGRES_PORT=5433"
      echo ""
    } > .env
    log_success "Created .env (for your local overrides)"
  else
    # For production, put config directly in .env
    {
      echo "# nself Production Configuration"
      echo "# Generated by wizard on $(date)"
      echo ""
      echo "# Core Settings"
      for item in "${config[@]}"; do
        echo "$item"
      done
    } > .env
    log_success "Created .env (production configuration)"
  fi

  # Copy .env.example from templates
  local templates_dir="${INIT_LIB_DIR}/../../templates"
  if [[ -f "$templates_dir/.env.example" ]]; then
    cp "$templates_dir/.env.example" .env.example
    log_success "Created .env.example (reference documentation)"
  elif [[ -f "${ROOT_DIR}/src/templates/.env.example" ]]; then
    cp "${ROOT_DIR}/src/templates/.env.example" .env.example
    log_success "Created .env.example (reference documentation)"
  else
    # Fallback: create basic example
    {
      echo "# nself Configuration Example"
      echo "# Copy this file to .env and customize"
      echo ""
      echo "# Project Settings"
      echo "PROJECT_NAME=myapp"
      echo "ENV=dev"
      echo "BASE_DOMAIN=local.nself.org"
      echo ""
      echo "# Database Settings"
      echo "POSTGRES_DB=myapp_db"
      echo "POSTGRES_USER=postgres"
      echo "POSTGRES_PASSWORD=postgres"
      echo ""
      echo "# Service Settings"
      echo "HASURA_GRAPHQL_ADMIN_SECRET=hasura-admin-secret"
      echo "HASURA_GRAPHQL_ENABLE_CONSOLE=true"
      echo ""
      echo "# Optional Services"
      echo "# REDIS_ENABLED=true"
      echo "# DASHBOARD_ENABLED=true"
    } > .env.example
    log_success "Created .env.example (basic template)"
  fi

  # Create or update .gitignore
  if [[ -f "$INIT_LIB_DIR/gitignore.sh" ]]; then
    source "$INIT_LIB_DIR/gitignore.sh"
    ensure_gitignore
    log_success "Created/updated .gitignore"
  else
    # Fallback: create basic .gitignore
    if [[ ! -f ".gitignore" ]]; then
      cat > .gitignore << 'EOF'
# Environment files (sensitive)
.env
.env.local
.env.*.local
.env.secrets

# But allow
!.env.example
!.env.dev
!.env.staging
!.env.production

# Docker volumes
.volumes/

# Logs
logs/
*.log

# Dependencies
node_modules/

# System files
.DS_Store
.idea/
.vscode/
*.swp
*.swo
*~

# Build artifacts
dist/
build/
*.pid
EOF
      log_success "Created .gitignore"
    else
      log_info ".gitignore already exists"
    fi
  fi

  log_success "Configuration generated successfully!"
  
  # Note: File generation will be handled by 'nself build' command
  
  echo ""
  echo "✅ Setup complete!"
  echo ""
  echo "Files created:"
  if [[ "$env_mode" == "dev" ]]; then
    echo "  • .env.dev - Wizard configuration (can be committed)"
    echo "  • .env - Your local overrides (git-ignored, mostly empty)"
  else
    echo "  • .env - Production configuration (git-ignored)"
  fi
  echo "  • .env.example - Reference documentation (from templates)"
  echo "  • .gitignore - Git ignore rules"
  echo ""
  echo "Next steps:"
  echo "  1. Review .env to adjust settings (optional)"
  echo "  2. Run: nself build"

  # Add setup notes if needed
  if [[ "$base_domain" != "local.nself.org" ]] && [[ "$env_mode" == "dev" ]]; then
    if [[ "$base_domain" != "localhost" ]] && ! check_domain_resolution "$base_domain" 2>/dev/null; then
      echo "     • Will configure /etc/hosts automatically (may need sudo)"
    fi
    if [[ "$base_domain" != "local.nself.org" ]]; then
      echo "     • Will set up SSL certificates automatically (may need sudo)"
    fi
  fi

  echo "  3. Run: nself start"
  echo "  4. Access your services:"

  # Show correct URL based on domain
  if [[ " ${config[@]} " =~ "USE_PORTS_ONLY=true" ]]; then
    echo "     • API: http://localhost:8080"
    echo "     • Hasura: http://localhost:9695"
    if [[ " ${config[@]} " =~ "DASHBOARD_ENABLED=true" ]]; then
      echo "     • Admin: http://localhost:3005"
    fi
  elif [[ "$base_domain" == "localhost" ]]; then
    # localhost will get SSL via mkcert automatically
    echo "     • API: https://api.localhost"
    echo "     • Hasura: https://hasura.localhost"
    if [[ " ${config[@]} " =~ "DASHBOARD_ENABLED=true" ]]; then
      echo "     • Admin: https://admin.localhost"
    fi
  else
    # All other domains get HTTPS (local.nself.org automatic, others via mkcert)
    echo "     • API: https://api.$base_domain"
    echo "     • Hasura: https://hasura.$base_domain"
    if [[ " ${config[@]} " =~ "DASHBOARD_ENABLED=true" ]]; then
      echo "     • Admin: https://admin.$base_domain"
    fi
  fi
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
    echo "These are saved in .env but you may want to store them securely."
  fi
}

# File generation removed - handled by 'nself build' command
# Template functions removed as they should not be in the wizard

# Execute if run directly
# Disabled due to syntax error - wizard is called from init.sh
# if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
#   run_config_wizard
# fi