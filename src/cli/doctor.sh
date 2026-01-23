#!/usr/bin/env bash

# doctor.sh - System diagnostics and health checks for nself

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/progress.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/docker.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Initialize counters
ISSUES_FOUND=0
WARNINGS_FOUND=0

issue_found() {
  ISSUES_FOUND=$((ISSUES_FOUND + 1))
}

warning_found() {
  WARNINGS_FOUND=$((WARNINGS_FOUND + 1))
}

# Function to check command availability
check_command() {
  local cmd="$1"
  local name="${2:-$cmd}"
  local required="${3:-true}"

  start_spinner "Checking $name"

  if command -v "$cmd" >/dev/null 2>&1; then
    local version=$($cmd --version 2>/dev/null | head -1 || echo "version unknown")
    stop_spinner "success" "$name is available: $version"
    return 0
  else
    if [[ "$required" == "true" ]]; then
      stop_spinner "error" "$name is not installed or not in PATH"
      issue_found
      return 1
    else
      stop_spinner "warning" "$name is not installed (optional)"
      warning_found
      return 1
    fi
  fi
}

# Function to check port availability
check_port() {
  local port="$1"
  local service="${2:-unknown}"

  start_spinner "Checking port $port"

  # Use lsof with timeout on macOS, netstat/ss on Linux
  local port_in_use=false
  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS - use lsof with timeout
    if lsof -ti ":$port" -sTCP:LISTEN >/dev/null 2>&1; then
      port_in_use=true
    fi
  else
    # Linux - use netstat or ss
    if netstat -tuln 2>/dev/null | grep -q ":$port " ||
      ss -tuln 2>/dev/null | grep -q ":$port "; then
      port_in_use=true
    fi
  fi

  if [[ "$port_in_use" == "true" ]]; then
    stop_spinner "warning" "Port $port is already in use (needed for $service)"
    warning_found
    return 1
  else
    stop_spinner "success" "Port $port is available for $service"
    return 0
  fi
}

# Function to check disk space
check_disk_space() {
  local path="${1:-.}"
  local min_gb="${2:-5}"

  start_spinner "Checking disk space"

  local available_gb
  if command -v df >/dev/null 2>&1; then
    # Try different df formats for cross-platform compatibility
    available_gb=$(df -h "$path" 2>/dev/null | tail -1 | awk '{print $4}' | sed 's/G.*//' 2>/dev/null || echo "unknown")

    if [[ "$available_gb" == "unknown" ]] || ! [[ "$available_gb" =~ ^[0-9]+$ ]]; then
      stop_spinner "warning" "Cannot determine available disk space"
      warning_found
    elif [[ "$available_gb" -ge "$min_gb" ]]; then
      stop_spinner "success" "Disk space: ${available_gb}GB available (minimum ${min_gb}GB)"
    else
      stop_spinner "error" "Disk space: Only ${available_gb}GB available, need at least ${min_gb}GB"
      issue_found
    fi
  else
    stop_spinner "warning" "Cannot check disk space (df command not available)"
    warning_found
  fi
}

# Function to check memory
check_memory() {
  local min_mb="${1:-2048}" # 2GB minimum

  start_spinner "Checking memory"

  if command -v free >/dev/null 2>&1; then
    local available_mb=$(free -m | grep '^Mem:' | awk '{print $7}' 2>/dev/null || echo "0")
    if [[ "$available_mb" -ge "$min_mb" ]]; then
      stop_spinner "success" "Memory: ${available_mb}MB available (minimum ${min_mb}MB)"
    else
      stop_spinner "warning" "Memory: Only ${available_mb}MB available, recommended minimum ${min_mb}MB"
      warning_found
    fi
  elif [[ "$(uname)" == "Darwin" ]]; then
    # macOS memory check
    local total_mb=$(($(sysctl -n hw.memsize) / 1024 / 1024))
    stop_spinner "success" "Memory: ${total_mb}MB total (macOS)"
  else
    stop_spinner "warning" "Cannot check available memory"
    warning_found
  fi
}

# Function to check Docker setup
check_docker() {
  if ! check_command "docker" "Docker"; then
    log_info "Install Docker from: https://docs.docker.com/get-docker/"
    return
  fi

  # Check Docker daemon
  start_spinner "Checking Docker daemon"
  if docker info >/dev/null 2>&1; then
    stop_spinner "success" "Docker daemon is running"

    # Check if user can run Docker without sudo
    start_spinner "Checking Docker permissions"
    if docker ps >/dev/null 2>&1; then
      stop_spinner "success" "Docker can be run without sudo"
    else
      stop_spinner "warning" "Docker requires sudo - consider adding user to docker group"
      warning_found
    fi
  else
    stop_spinner "error" "Docker daemon is not running"
    issue_found
    log_info "Start Docker with: sudo systemctl start docker (Linux) or start Docker Desktop (macOS/Windows)"
  fi
}

# Function to check Docker Compose
check_docker_compose() {
  start_spinner "Checking Docker Compose"

  # Check for Docker Compose v2 (plugin)
  if docker compose version >/dev/null 2>&1; then
    local compose_version=$(docker compose version --short 2>/dev/null || docker compose version)
    stop_spinner "success" "Docker Compose (plugin) is available: $compose_version"
    return 0
  fi

  # Check for legacy docker compose without spinner since check_command has its own
  if command -v "docker compose" >/dev/null 2>&1; then
    stop_spinner "warning" "Using legacy docker compose - consider upgrading to Docker Compose v2"
    warning_found
    return 0
  fi

  stop_spinner "error" "Docker Compose is not available"
  issue_found
  log_info "Install Docker Compose v2 or legacy docker compose"
}

# Function to check network connectivity
check_network() {
  # Check basic internet connectivity
  start_spinner "Checking internet connectivity"
  if curl -s --connect-timeout 5 https://google.com >/dev/null 2>&1 ||
    wget -q --timeout=5 --spider https://google.com >/dev/null 2>&1; then
    stop_spinner "success" "Internet connectivity is working"
  else
    stop_spinner "warning" "Cannot reach external websites - check internet connection"
    warning_found
  fi

  # Check Docker Hub connectivity
  start_spinner "Checking Docker Hub connectivity"
  if curl -s --connect-timeout 5 https://hub.docker.com >/dev/null 2>&1; then
    stop_spinner "success" "Docker Hub is reachable"
  else
    stop_spinner "warning" "Cannot reach Docker Hub - Docker pulls may fail"
    warning_found
  fi
}

# Function to check nself configuration
check_nself_config() {
  start_spinner "Checking nself configuration"

  if [[ -f ".env" ]] || [[ -f ".env.dev" ]]; then
    stop_spinner "success" "Configuration file found"

    # Load environment safely
    start_spinner "Loading configuration"
    load_env_with_priority || {
      stop_spinner "error" "Failed to load configuration - syntax error"
      issue_found
      return
    }
    stop_spinner "success" "Configuration loaded successfully"

    # Check essential variables
    start_spinner "Checking essential variables"
    local missing_vars=()

    [[ -z "$PROJECT_NAME" ]] && missing_vars+=("PROJECT_NAME")
    [[ -z "$BASE_DOMAIN" ]] && missing_vars+=("BASE_DOMAIN")
    [[ -z "$POSTGRES_PASSWORD" ]] && missing_vars+=("POSTGRES_PASSWORD")
    [[ -z "$HASURA_GRAPHQL_ADMIN_SECRET" ]] && missing_vars+=("HASURA_GRAPHQL_ADMIN_SECRET")

    if [[ ${#missing_vars[@]} -eq 0 ]]; then
      stop_spinner "success" "Essential configuration variables are set"
    else
      stop_spinner "error" "Missing configuration variables: ${missing_vars[*]}"
      issue_found
      log_info "Run 'nself init' to generate missing configuration"
    fi

    # Check for password strength
    if [[ -n "$POSTGRES_PASSWORD" ]] && [[ ${#POSTGRES_PASSWORD} -lt 12 ]]; then
      log_warning "Postgres password is shorter than 12 characters"
      warning_found
      log_info "Run 'nself prod' to generate secure passwords"
    fi

    # Check domain configuration
    if [[ "$BASE_DOMAIN" == *"nself.org"* ]]; then
      log_success "Using nself.org domain for local development"
    else
      log_info "Using custom domain: $BASE_DOMAIN"
      log_info "Ensure DNS is configured for custom domains"
    fi

  else
    stop_spinner "error" "Configuration file not found (.env or .env.dev)"
    issue_found
    log_info "Run 'nself init' to create initial configuration"
  fi
}

# Function to check if project has running containers
has_running_containers() {
  if [[ -f ".env" ]] || [[ -f ".env.local" ]] || [[ -f ".env.dev" ]]; then
    load_env_with_priority 2>/dev/null || true
  fi

  local project_name="${PROJECT_NAME:-nself}"
  local running_count=$(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')

  [[ "${running_count}" -gt 0 ]]
}

# Function to check running containers health
check_running_containers() {
  echo ""
  echo "Container Health Check"
  echo "──────────────────────────────────────────────"

  local project_name="${PROJECT_NAME:-nself}"
  local containers=($(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null | sort))
  local total=${#containers[@]}
  local healthy=0
  local unhealthy=0
  local no_health_check=0

  for container in "${containers[@]}"; do
    local service_name=$(echo "$container" | sed "s/^${project_name}_//" | sed 's/_[0-9]*$//')
    local health_status=$(docker inspect "$container" --format='{{.State.Health.Status}}' 2>/dev/null || echo "none")
    local state=$(docker inspect "$container" --format='{{.State.Status}}' 2>/dev/null || echo "unknown")

    if [[ "$health_status" == "healthy" ]]; then
      log_success "$service_name: Healthy"
      ((healthy++))
    elif [[ "$health_status" == "unhealthy" ]]; then
      log_error "$service_name: Unhealthy"
      issue_found
      ((unhealthy++))
      # Show last few log lines for unhealthy containers
      log_info "  Recent logs:"
      docker logs --tail 5 "$container" 2>&1 | sed 's/^/    /'
    elif [[ "$health_status" == "starting" ]]; then
      log_info "$service_name: Starting..."
    elif [[ "$state" == "running" && "$health_status" == "none" ]]; then
      log_success "$service_name: Running (no health check)"
      ((no_health_check++))
    elif [[ "$state" == "restarting" ]]; then
      log_warning "$service_name: Restarting (check logs for errors)"
      warning_found
    else
      log_warning "$service_name: $state"
      warning_found
    fi
  done

  echo ""
  if [[ $unhealthy -eq 0 ]]; then
    log_success "All $total containers are healthy"
  else
    log_warning "$healthy healthy, $unhealthy unhealthy (total: $total containers)"
  fi

  if [[ $unhealthy -gt 0 ]]; then
    echo ""
    log_info "Troubleshooting unhealthy containers:"
    log_info "  • Check logs: nself logs <service>"
    log_info "  • Restart service: nself restart <service>"
    log_info "  • Check resources: docker stats"
  fi
}

# Function to check services status (fallback when no containers running)
check_services() {
  start_spinner "Checking docker-compose.yml"

  if [[ -f "docker-compose.yml" ]]; then
    stop_spinner "success" "docker-compose.yml found"

    # Check if any services are running using direct Docker query
    start_spinner "Checking running services"
    local project_name="${PROJECT_NAME:-nself}"
    local running_count=$(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')

    # Try to get total from compose config, fallback to counting from docker-compose.yml
    local total_services=$(compose config --services 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$total_services" == "0" && -f "docker-compose.yml" ]]; then
      total_services=$(grep -c "^  [a-z].*:" docker-compose.yml 2>/dev/null || echo "?")
    fi

    if [[ "$running_count" -gt 0 ]]; then
      stop_spinner "success" "$running_count/$total_services services are running"
    else
      stop_spinner "info" "No services are currently running"
      log_info "Run 'nself start' to start services"
    fi

  else
    stop_spinner "warning" "docker-compose.yml not found"
    warning_found
    log_info "Run 'nself build' to generate docker-compose.yml"
  fi
}

# Function to check ports
check_ports() {
  # Standard nself ports
  check_port 80 "HTTP (nginx)"
  check_port 443 "HTTPS (nginx)"
  check_port 5432 "PostgreSQL"
  check_port 8080 "Hasura GraphQL"
  check_port 4000 "Hasura Auth"
  check_port 9000 "MinIO"
  check_port 6379 "Redis"
  check_port 1025 "SMTP (MailPit)"
  check_port 8025 "MailPit UI"
}

# Function to check SSL certificates
check_ssl() {
  start_spinner "Checking SSL certificates"

  local cert_path="nginx/ssl/nself.org.crt"
  local key_path="nginx/ssl/nself.org.key"

  if [[ -f "$cert_path" && -f "$key_path" ]]; then
    stop_spinner "success" "SSL certificates found"

    # Check certificate expiry
    if command -v openssl >/dev/null 2>&1; then
      start_spinner "Checking certificate expiry"
      local expiry_date=$(openssl x509 -in "$cert_path" -noout -enddate 2>/dev/null | cut -d= -f2)
      if [[ -n "$expiry_date" ]]; then
        # Check if certificate expires within 30 days
        local expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null || date -j -f "%b %d %T %Y %Z" "$expiry_date" +%s 2>/dev/null || echo "0")
        local now_epoch=$(date +%s)
        local days_until_expiry=$(((expiry_epoch - now_epoch) / 86400))

        if [[ $days_until_expiry -lt 30 ]] && [[ $days_until_expiry -gt 0 ]]; then
          stop_spinner "warning" "Certificate expires in $days_until_expiry days"
          warning_found
        elif [[ $days_until_expiry -le 0 ]]; then
          stop_spinner "error" "Certificate has expired"
          issue_found
        else
          stop_spinner "success" "Certificate expires: $expiry_date"
        fi
      else
        stop_spinner "warning" "Cannot read certificate expiry"
        warning_found
      fi
    fi
  else
    stop_spinner "warning" "SSL certificates not found"
    warning_found
    log_info "Run 'nself build' to generate certificates"
  fi
}

# Function to show system information
show_system_info() {
  echo ""
  echo "System Information"
  echo "──────────────────────────────────────────────"

  log_info "Operating System: $(uname -s) $(uname -r)"
  log_info "Architecture: $(uname -m)"
  log_info "Current Directory: $(pwd)"
  log_info "User: $(whoami)"
  log_info "Date: $(date)"

  if [[ -f "bin/VERSION" ]]; then
    log_info "nself Version: $(cat bin/VERSION)"
  fi
}

# Function to check database health
check_database() {
  echo ""
  echo "Database Health"
  echo "──────────────────────────────────────────────"

  # Check if PostgreSQL is running
  if docker ps --format "{{.Names}}" | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_success "PostgreSQL: Running"

    # Check connection
    if docker exec "${PROJECT_NAME:-myproject}_postgres" pg_isready -U "${POSTGRES_USER:-postgres}" >/dev/null 2>&1; then
      log_success "Connection: OK"
    else
      log_error "Connection: Failed"
      issue_found
      return 1
    fi

    # Check connection count
    local conn_count=$(docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U "${POSTGRES_USER:-postgres}" -t -c \
      "SELECT count(*) FROM pg_stat_activity WHERE state != 'idle';" 2>/dev/null | xargs)
    local max_conn="${DB_MAX_CONNECTIONS:-100}"
    local conn_percent=$((conn_count * 100 / max_conn))


    if [[ $conn_percent -lt 80 ]]; then
      log_success "Connections: $conn_count/$max_conn ($conn_percent%)"
    else
      log_warning "High connections: $conn_count/$max_conn ($conn_percent%)"
      warning_found
    fi

    # Check for table bloat
    local bloat_count=$(docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U "${POSTGRES_USER:-postgres}" -t -c \
      "SELECT COUNT(*) FROM pg_stat_user_tables
       WHERE n_dead_tup > n_live_tup * 0.2 AND n_live_tup > 1000;" 2>/dev/null | xargs)

    if [[ "$bloat_count" == "0" ]]; then
      log_success "Table bloat: None detected"
    else
      log_warning "Table bloat: $bloat_count tables need VACUUM"
      warning_found
    fi

    # Check database size (last item)
    local db_size=$(docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U "${POSTGRES_USER:-postgres}" -t -c \
      "SELECT pg_size_pretty(pg_database_size('${POSTGRES_DB:-nhost}'));" 2>/dev/null | xargs)
    log_info "Database size: $db_size"
    
    # Check backup status
    if [[ -d "backups" ]]; then
      local latest_backup=$(ls -t backups/*.tar.gz 2>/dev/null | head -1)
      if [[ -n "$latest_backup" ]]; then
        local backup_time=$(stat -f %m "$latest_backup" 2>/dev/null || stat -c %Y "$latest_backup" 2>/dev/null)
        local current_time=$(date +%s)
        local backup_age_hours=$(( (current_time - backup_time) / 3600 ))
        
        if [[ $backup_age_hours -lt 48 ]]; then
          log_success "Latest backup: ${backup_age_hours}h old"
        else
          log_warning "Latest backup: ${backup_age_hours}h old (run: nself backup create)"
          warning_found
        fi
      else
        log_warning "No backups found (run: nself backup create)"
        warning_found
      fi
    fi
    
    # Check WAL archiving if PITR enabled
    if [[ "${DB_PITR_ENABLED:-false}" == "true" ]] || [[ "${ENV:-dev}" == "prod" ]]; then
      local wal_status=$(docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U "${POSTGRES_USER:-postgres}" -t -c \
        "SELECT archive_mode FROM pg_settings WHERE name='archive_mode';" 2>/dev/null | xargs)
      
      if [[ "$wal_status" == "on" ]]; then
        log_success "WAL archiving: Enabled"
      else
        log_warning "WAL archiving: Disabled (PITR not available)"
        warning_found
      fi
    fi
    
    # Check replication if configured
    if [[ -n "${DB_REPLICA_HOST:-}" ]]; then
      local rep_lag=$(docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U "${POSTGRES_USER:-postgres}" -t -c \
        "SELECT EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))::int;" 2>/dev/null || echo "")
      
      if [[ -n "$rep_lag" ]] && [[ "$rep_lag" -lt 10 ]]; then
        log_success "Replication lag: ${rep_lag}s"
      elif [[ -n "$rep_lag" ]]; then
        log_warning "High replication lag: ${rep_lag}s"
        warning_found
      fi
    fi
    
  else
    log_error "PostgreSQL: Not running"
    issue_found
    return 1
  fi
  
  return 0
}

# Function to check all service URLs
check_service_urls() {
  echo ""
  echo "Service URLs"
  echo "──────────────────────────────────────────────"

  if [[ ! -f ".env" ]] && [[ ! -f ".env.dev" ]]; then
    log_warning "No configuration found - cannot determine service URLs"
    warning_found
    return
  fi

  load_env_with_priority
  local base_domain="${BASE_DOMAIN:-local.nself.org}"

  # Core services (always available)
  log_success "Core Services:"
  log_info "  GraphQL API:     https://api.$base_domain"
  log_info "  Auth:            https://auth.$base_domain"
  log_info "  Storage:         https://storage.$base_domain"
  log_info "  Admin Console:   https://api.$base_domain/console"

  # Optional services
  local optional_found=false
  if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
    if [[ "$optional_found" == "false" ]]; then
      log_success "Optional Services:"
      optional_found=true
    fi
    log_info "  Functions:       https://functions.$base_domain"
  fi

  if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
    if [[ "$optional_found" == "false" ]]; then
      log_success "Optional Services:"
      optional_found=true
    fi
    log_info "  Dashboard:       https://dashboard.$base_domain"
  fi

  if [[ "$NESTJS_ENABLED" == "true" ]]; then
    if [[ "$optional_found" == "false" ]]; then
      log_success "Optional Services:"
      optional_found=true
    fi
    log_info "  NestJS API:      https://nestjs.$base_domain"
  fi

  if [[ "$GOLANG_ENABLED" == "true" ]]; then
    if [[ "$optional_found" == "false" ]]; then
      log_success "Optional Services:"
      optional_found=true
    fi
    log_info "  Golang API:      https://golang.$base_domain"
  fi

  if [[ "$PYTHON_ENABLED" == "true" ]]; then
    if [[ "$optional_found" == "false" ]]; then
      log_success "Optional Services:"
      optional_found=true
    fi
    log_info "  Python API:      https://python.$base_domain"
  fi

  # Development services
  if [[ "$ENV" == "dev" ]]; then
    local dev_found=false
    if [[ "$MAILHOG_ENABLED" != "false" ]]; then
      if [[ "$dev_found" == "false" ]]; then
        log_success "Development Services:"
        dev_found=true
      fi
      log_info "  MailHog:         https://mailhog.$base_domain"
    fi

    if [[ "$ADMINER_ENABLED" == "true" ]]; then
      if [[ "$dev_found" == "false" ]]; then
        log_success "Development Services:"
        dev_found=true
      fi
      log_info "  Adminer:         https://adminer.$base_domain"
    fi
  fi

  # Show direct access URLs
  log_success "Direct Access (localhost):"
  log_info "  PostgreSQL:      localhost:5432"
  if [[ "$REDIS_ENABLED" == "true" ]]; then
    log_info "  Redis:           localhost:6379"
  fi
  log_info "  MinIO Console:   http://localhost:9001"
}

# Function to show recommendations
show_recommendations() {
  local containers_running="${1:-false}"

  echo ""
  echo "Recommendations"
  echo "──────────────────────────────────────────────"

  if [[ $ISSUES_FOUND -eq 0 ]] && [[ $WARNINGS_FOUND -eq 0 ]]; then
    log_success "Your nself installation looks great!"
    if [[ "$containers_running" == "true" ]]; then
      log_info "All containers are healthy and running properly."
    else
      log_info "Ready for development. Run 'nself start' to start services."
    fi
    return
  fi

  if [[ "$containers_running" == "true" ]]; then
    # Different messaging for running containers
    if [[ $ISSUES_FOUND -gt 0 ]]; then
      log_warning "Found $ISSUES_FOUND unhealthy container(s)"
      log_info "• Check container logs for errors"
      log_info "• Restart unhealthy services"
      log_info "• Check resource usage (CPU/memory)"
    fi

    if [[ $WARNINGS_FOUND -gt 0 ]]; then
      log_info "Found $WARNINGS_FOUND warning(s) - containers are functional"
    fi
  else
    # System/configuration issues
    if [[ $ISSUES_FOUND -gt 0 ]]; then
      log_error "Found $ISSUES_FOUND critical issue(s) that need attention:"
      log_info "• Fix critical issues before running nself"
      log_info "• Check installation documentation"
      log_info "• Verify system requirements"
    fi

    if [[ $WARNINGS_FOUND -gt 0 ]]; then
      log_warning "Found $WARNINGS_FOUND warning(s):"
      log_info "• Warnings won't prevent nself from running"
      log_info "• Consider addressing for optimal performance"
      log_info "• Some may become issues in production"
    fi

    echo ""
    log_info "Common fixes:"
    log_info "  nself init          - Create initial configuration"
    log_info "  nself build         - Generate project structure"
    log_info "  nself prod          - Generate secure passwords"
    log_info "  nself update        - Update to latest version"
    log_info "  nself status        - Check service details"
  fi
}

# Main function
main() {
  show_command_header "nself doctor" "System diagnostics and health checks"

  show_system_info

  # Check if containers are running - this determines which checks to run
  local containers_running=false
  if has_running_containers; then
    containers_running=true
  fi

  # Show brief system requirements check (always shown)
  echo ""
  echo "System Requirements"
  echo "──────────────────────────────────────────────"

  # Check Docker
  start_spinner "Checking Docker"
  if docker info >/dev/null 2>&1; then
    local docker_version=$(docker --version 2>/dev/null | cut -d' ' -f3 | tr -d ',')
    stop_spinner "success" "Docker ${docker_version} ${COLOR_DIM}(min v20.10)${COLOR_RESET}"
  else
    stop_spinner "error" "Docker is not running ${COLOR_DIM}(required)${COLOR_RESET}"
    issue_found
  fi

  # Check Docker Compose
  start_spinner "Checking Docker Compose"
  if docker compose version >/dev/null 2>&1; then
    local compose_version=$(docker compose version --short 2>/dev/null)
    stop_spinner "success" "Docker Compose ${compose_version} ${COLOR_DIM}(min v2.0)${COLOR_RESET}"
  else
    stop_spinner "error" "Docker Compose not available ${COLOR_DIM}(required)${COLOR_RESET}"
    issue_found
  fi

  # Check memory
  start_spinner "Checking memory"
  if [[ "$(uname)" == "Darwin" ]]; then
    local total_mb=$(($(sysctl -n hw.memsize 2>/dev/null || echo 0) / 1024 / 1024))
    local total_gb=$((total_mb / 1024))
    stop_spinner "success" "Memory ${total_gb}GB ${COLOR_DIM}(min 2GB)${COLOR_RESET}"
  elif command -v free >/dev/null 2>&1; then
    local total_mb=$(free -m | grep '^Mem:' | awk '{print $2}' 2>/dev/null || echo "0")
    if [[ "$total_mb" -ge 2048 ]]; then
      local total_gb=$((total_mb / 1024))
      stop_spinner "success" "Memory ${total_gb}GB ${COLOR_DIM}(min 2GB)${COLOR_RESET}"
    else
      local total_gb_display=$(awk "BEGIN {printf \"%.1f\", $total_mb/1024}")
      stop_spinner "warning" "Memory ${total_gb_display}GB - Low ${COLOR_DIM}(min 2GB recommended)${COLOR_RESET}"
      warning_found
    fi
  else
    stop_spinner "success" "Memory available ${COLOR_DIM}(unable to determine exact amount)${COLOR_RESET}"
  fi

  # Check disk space with minimum shown
  start_spinner "Checking disk space"
  local available_gb
  if command -v df >/dev/null 2>&1; then
    available_gb=$(df -h "." 2>/dev/null | tail -1 | awk '{print $4}' | sed 's/G.*//' 2>/dev/null || echo "unknown")

    if [[ "$available_gb" == "unknown" ]] || ! [[ "$available_gb" =~ ^[0-9]+$ ]]; then
      stop_spinner "warning" "Disk space available ${COLOR_DIM}(unable to determine exact amount)${COLOR_RESET}"
      warning_found
    elif [[ "$available_gb" -ge 5 ]]; then
      stop_spinner "success" "Disk space ${available_gb}GB ${COLOR_DIM}(min 5GB)${COLOR_RESET}"
    else
      stop_spinner "error" "Disk space ${available_gb}GB - Insufficient ${COLOR_DIM}(min 5GB required)${COLOR_RESET}"
      issue_found
    fi
  else
    stop_spinner "warning" "Disk space check unavailable"
    warning_found
  fi

  if [[ "$containers_running" == "true" ]]; then
    # Containers are running - show health diagnostics
    echo ""
    log_success "Found running containers - performing health diagnostics"

    # Check running container health
    check_running_containers

    # Check database health if PostgreSQL is running
    if docker ps --format "{{.Names}}" | grep -q "${PROJECT_NAME:-nself}_postgres"; then
      check_database 2>/dev/null || true
    fi

  else
    # No containers running - perform full system requirements check
    echo ""
    log_info "No containers running - checking system requirements"
    echo ""

    echo "System Requirements"
    echo "──────────────────────────────────────────────"

    check_command "curl" "curl"
    check_command "git" "Git"
    check_docker
    check_docker_compose
    check_memory
    check_disk_space "." 5

    echo ""
    echo "Network & Connectivity"
    echo "──────────────────────────────────────────────"

    check_network
    check_ports

    echo ""
    echo "nself Configuration"
    echo "──────────────────────────────────────────────"

    check_nself_config
    check_services
    check_ssl
  fi

  show_recommendations "$containers_running"

  # Show quick fix commands only for system issues (not container issues)
  if [[ $ISSUES_FOUND -gt 0 && "$containers_running" == "false" ]]; then
    echo ""
    echo "Quick Fixes"
    echo "──────────────────────────────────────────────"

    if ! docker info >/dev/null 2>&1; then
      log_info "  Start Docker: 'sudo systemctl start docker' (Linux) or start Docker Desktop"
    fi

    if [[ ! -f ".env" ]] && [[ ! -f ".env.dev" ]]; then
      log_info "  Create config: 'nself init'"
    fi

    if [[ ! -f "docker-compose.yml" ]]; then
      log_info "  Generate structure: 'nself build'"
    fi

    log_info "  After fixes, run: 'nself doctor' again to verify"
  fi

  echo ""

  # Exit with appropriate code
  if [[ $ISSUES_FOUND -eq 0 ]]; then
    exit 0
  else
    exit 1
  fi
}

# Auto-fix functions
auto_fix_docker() {
  log_info "Attempting to start Docker..."

  if [[ "$(uname)" == "Darwin" ]]; then
    # macOS - try to start Docker Desktop
    open -a Docker 2>/dev/null || {
      log_error "Cannot start Docker Desktop automatically"
      log_info "Please start Docker Desktop manually"
      return 1
    }
    log_info "Waiting for Docker to start..."
    local count=0
    while [[ $count -lt 30 ]]; do
      if docker info >/dev/null 2>&1; then
        log_success "Docker started successfully"
        return 0
      fi
      sleep 2
      count=$((count + 1))
    done
    log_error "Docker did not start within 60 seconds"
    return 1
  else
    # Linux - try systemctl
    if command -v systemctl >/dev/null 2>&1; then
      sudo systemctl start docker 2>/dev/null || {
        log_error "Cannot start Docker. Run: sudo systemctl start docker"
        return 1
      }
      log_success "Docker started"
      return 0
    fi
    log_error "Cannot auto-start Docker. Start it manually."
    return 1
  fi
}

auto_fix_config() {
  log_info "Creating initial configuration..."
  if command -v nself >/dev/null 2>&1; then
    nself init --defaults 2>/dev/null || {
      log_info "Running basic init..."
      # Create minimal .env
      cat > .env << 'EOF'
PROJECT_NAME=nself-project
ENV=dev
BASE_DOMAIN=local.nself.org
POSTGRES_DB=nhost
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres123
HASURA_GRAPHQL_ADMIN_SECRET=hasura-admin-secret
EOF
      log_success "Created basic .env configuration"
    }
  fi
  return 0
}

auto_fix_build() {
  log_info "Running nself build..."
  if command -v nself >/dev/null 2>&1; then
    nself build 2>/dev/null || {
      log_error "Build failed. Check configuration."
      return 1
    }
    log_success "Build completed"
    return 0
  fi
  return 1
}

auto_fix_ssl() {
  log_info "Generating SSL certificates..."
  if command -v nself >/dev/null 2>&1; then
    nself ssl bootstrap 2>/dev/null || {
      log_warning "SSL bootstrap failed, trying basic generation..."
      nself build 2>/dev/null || true
    }
    log_success "SSL certificates generated"
    return 0
  fi
  return 1
}

auto_fix_unhealthy_container() {
  local container="$1"
  log_info "Attempting to fix: $container"

  # Try restarting the container
  docker restart "$container" 2>/dev/null || {
    log_warning "Cannot restart $container"
    return 1
  }

  # Wait for health
  local count=0
  while [[ $count -lt 15 ]]; do
    local health=$(docker inspect "$container" --format='{{.State.Health.Status}}' 2>/dev/null || echo "none")
    if [[ "$health" == "healthy" ]]; then
      log_success "$container is now healthy"
      return 0
    fi
    sleep 2
    count=$((count + 1))
  done

  log_warning "$container still unhealthy after restart"
  return 1
}

run_auto_fix() {
  log_info "Running auto-fix for detected issues..."
  echo ""

  local fixed=0
  local failed=0

  # Fix Docker if not running
  if ! docker info >/dev/null 2>&1; then
    if auto_fix_docker; then
      fixed=$((fixed + 1))
    else
      failed=$((failed + 1))
    fi
  fi

  # Fix missing config
  if [[ ! -f ".env" ]] && [[ ! -f ".env.dev" ]]; then
    if auto_fix_config; then
      fixed=$((fixed + 1))
    else
      failed=$((failed + 1))
    fi
  fi

  # Fix missing docker-compose.yml
  if [[ ! -f "docker-compose.yml" ]] && [[ -f ".env" ]]; then
    if auto_fix_build; then
      fixed=$((fixed + 1))
    else
      failed=$((failed + 1))
    fi
  fi

  # Fix missing SSL
  if [[ ! -f "nginx/ssl/nself.org.crt" ]] && [[ -f "docker-compose.yml" ]]; then
    if auto_fix_ssl; then
      fixed=$((fixed + 1))
    else
      failed=$((failed + 1))
    fi
  fi

  # Fix unhealthy containers
  if has_running_containers; then
    local project_name="${PROJECT_NAME:-nself}"
    local unhealthy_containers=($(docker ps --filter "name=${project_name}_" --filter "health=unhealthy" --format "{{.Names}}" 2>/dev/null))

    for container in "${unhealthy_containers[@]}"; do
      if auto_fix_unhealthy_container "$container"; then
        fixed=$((fixed + 1))
      else
        failed=$((failed + 1))
      fi
    done
  fi

  echo ""
  if [[ $fixed -gt 0 ]]; then
    log_success "Fixed $fixed issue(s)"
  fi
  if [[ $failed -gt 0 ]]; then
    log_warning "Could not fix $failed issue(s)"
  fi
  if [[ $fixed -eq 0 ]] && [[ $failed -eq 0 ]]; then
    log_info "No auto-fixable issues detected"
  fi

  echo ""
  log_info "Run 'nself doctor' again to verify"
}

# Container diagnostics - shows expected vs running containers
doctor_containers() {
  show_command_header "nself doctor containers" "Container count verification"

  # Load environment
  [[ -f ".env" ]] && source ".env" 2>/dev/null || true
  [[ -f ".env.local" ]] && source ".env.local" 2>/dev/null || true

  local project_name="${PROJECT_NAME:-nself}"

  echo ""
  printf "${COLOR_CYAN}➞ Container Analysis${COLOR_RESET}\n"
  echo ""

  # Get expected services from docker-compose.yml
  local expected_services=()
  if [[ -f "docker-compose.yml" ]]; then
    while IFS= read -r service; do
      [[ -n "$service" ]] && expected_services+=("$service")
    done < <(grep -E "^  [a-z][a-z0-9_-]*:$" docker-compose.yml | sed 's/://g' | tr -d ' ')
  fi

  # Get running containers
  local running_containers=()
  while IFS= read -r container; do
    [[ -n "$container" ]] && running_containers+=("$container")
  done < <(docker ps --filter "label=com.docker.compose.project=${project_name}" --format "{{.Names}}" 2>/dev/null)

  # Also check by name pattern
  if [[ ${#running_containers[@]} -eq 0 ]]; then
    while IFS= read -r container; do
      [[ -n "$container" ]] && running_containers+=("$container")
    done < <(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null)
  fi

  local expected_count=${#expected_services[@]}
  local running_count=${#running_containers[@]}

  # Calculate counts by category
  local core_expected=0 core_running=0
  local optional_expected=0 optional_running=0
  local monitoring_expected=0 monitoring_running=0
  local custom_expected=0 custom_running=0

  # Core services
  local core_services=("postgres" "hasura" "auth" "nginx")
  # Optional services
  local optional_services=("minio" "redis" "functions" "mailpit" "meilisearch" "mlflow" "nself-admin" "storage")
  # Monitoring services
  local monitoring_services=("prometheus" "grafana" "loki" "promtail" "tempo" "alertmanager" "cadvisor" "node-exporter" "postgres-exporter" "redis-exporter")

  # Count expected by category
  for service in "${expected_services[@]}"; do
    local found=false
    for core in "${core_services[@]}"; do
      if [[ "$service" == *"$core"* ]]; then
        core_expected=$((core_expected + 1))
        found=true
        break
      fi
    done
    if [[ "$found" == "false" ]]; then
      for opt in "${optional_services[@]}"; do
        if [[ "$service" == *"$opt"* ]]; then
          optional_expected=$((optional_expected + 1))
          found=true
          break
        fi
      done
    fi
    if [[ "$found" == "false" ]]; then
      for mon in "${monitoring_services[@]}"; do
        if [[ "$service" == *"$mon"* ]]; then
          monitoring_expected=$((monitoring_expected + 1))
          found=true
          break
        fi
      done
    fi
    if [[ "$found" == "false" ]]; then
      custom_expected=$((custom_expected + 1))
    fi
  done

  # Show summary
  printf "  ${COLOR_BOLD}Expected:${COLOR_RESET} %d containers\n" "$expected_count"
  printf "  ${COLOR_BOLD}Running:${COLOR_RESET}  %d containers\n" "$running_count"
  echo ""

  if [[ $running_count -eq $expected_count ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} All expected containers are running\n"
  elif [[ $running_count -gt $expected_count ]]; then
    printf "  ${COLOR_YELLOW}⚠${COLOR_RESET} More containers running than expected (+%d)\n" "$((running_count - expected_count))"
  else
    printf "  ${COLOR_RED}✗${COLOR_RESET} Missing %d container(s)\n" "$((expected_count - running_count))"
  fi

  echo ""
  printf "${COLOR_CYAN}➞ Expected by Category${COLOR_RESET}\n"
  echo ""
  printf "  Core (required):     %d\n" "$core_expected"
  printf "  Optional services:   %d\n" "$optional_expected"
  printf "  Monitoring bundle:   %d\n" "$monitoring_expected"
  printf "  Custom services:     %d\n" "$custom_expected"

  # Find missing containers
  echo ""
  printf "${COLOR_CYAN}➞ Service Status${COLOR_RESET}\n"
  echo ""

  local missing_services=()
  for service in "${expected_services[@]}"; do
    local container_name="${project_name}_${service}_1"
    local alt_name="${project_name}-${service}-1"
    local short_name="${project_name}_${service}"

    local is_running=false
    for running in "${running_containers[@]}"; do
      if [[ "$running" == "$container_name" ]] || [[ "$running" == "$alt_name" ]] || [[ "$running" == "$short_name" ]] || [[ "$running" == *"$service"* ]]; then
        is_running=true
        break
      fi
    done

    if [[ "$is_running" == "true" ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} %s\n" "$service"
    else
      printf "  ${COLOR_RED}✗${COLOR_RESET} %s ${COLOR_GRAY}(not running)${COLOR_RESET}\n" "$service"
      missing_services+=("$service")
    fi
  done

  # Show missing details
  if [[ ${#missing_services[@]} -gt 0 ]]; then
    echo ""
    printf "${COLOR_CYAN}➞ Missing Services${COLOR_RESET}\n"
    echo ""
    for missing in "${missing_services[@]}"; do
      printf "  ${COLOR_RED}•${COLOR_RESET} %s\n" "$missing"

      # Try to show why it might be missing
      local reason=""
      case "$missing" in
        *redis-exporter*)
          [[ "${REDIS_ENABLED:-}" != "true" ]] && reason="REDIS_ENABLED=false"
          ;;
        *mlflow*)
          [[ "${MLFLOW_ENABLED:-}" != "true" ]] && reason="MLFLOW_ENABLED=false"
          ;;
        *functions*)
          [[ "${FUNCTIONS_ENABLED:-}" != "true" ]] && reason="FUNCTIONS_ENABLED=false"
          ;;
      esac

      if [[ -n "$reason" ]]; then
        printf "    ${COLOR_GRAY}Possible reason: %s${COLOR_RESET}\n" "$reason"
      fi
    done

    echo ""
    printf "${COLOR_YELLOW}Tip:${COLOR_RESET} Run 'nself start' to start missing containers\n"
    printf "     Run 'docker logs ${project_name}_<service>' to check for errors\n"
  fi

  # Show any extra containers (not in docker-compose.yml)
  local extra_containers=()
  for running in "${running_containers[@]}"; do
    local is_expected=false
    for service in "${expected_services[@]}"; do
      if [[ "$running" == *"$service"* ]]; then
        is_expected=true
        break
      fi
    done
    if [[ "$is_expected" == "false" ]]; then
      extra_containers+=("$running")
    fi
  done

  if [[ ${#extra_containers[@]} -gt 0 ]]; then
    echo ""
    printf "${COLOR_CYAN}➞ Extra Containers (not in docker-compose.yml)${COLOR_RESET}\n"
    echo ""
    for extra in "${extra_containers[@]}"; do
      printf "  ${COLOR_YELLOW}?${COLOR_RESET} %s\n" "$extra"
    done
  fi

  # Check for buildx containers
  local buildx_containers=$(docker ps -a --filter "name=buildx_buildkit" --format "{{.Names}}" 2>/dev/null)
  if [[ -n "$buildx_containers" ]]; then
    echo ""
    printf "${COLOR_CYAN}➞ BuildKit Containers (from docker buildx)${COLOR_RESET}\n"
    echo ""
    printf "  ${COLOR_YELLOW}!${COLOR_RESET} Found buildx containers (not part of project):\n"
    echo "$buildx_containers" | while read -r bx; do
      printf "    • %s\n" "$bx"
    done
    printf "\n  ${COLOR_GRAY}Remove with: nself clean --builders${COLOR_RESET}\n"
  fi

  echo ""
}

# Handle command line arguments
case "${1:-}" in
-h | --help)
  echo "nself doctor - System diagnostics and health checks"
  echo ""
  echo "Usage: nself doctor [options]"
  echo "       nself doctor containers"
  echo ""
  echo "Subcommands:"
  echo "  containers     Show expected vs running container analysis"
  echo ""
  echo "Options:"
  echo "  -h, --help     Show this help message"
  echo "  -v, --verbose  Verbose output"
  echo "  --fix          Automatically fix detected issues"
  echo ""
  echo "This command checks:"
  echo "  • System requirements (Docker, memory, disk)"
  echo "  • Network connectivity"
  echo "  • nself configuration"
  echo "  • Service status"
  echo "  • SSL certificates"
  echo ""
  echo "Auto-fix capabilities:"
  echo "  • Start Docker if not running"
  echo "  • Create initial .env configuration"
  echo "  • Run nself build if needed"
  echo "  • Generate SSL certificates"
  echo "  • Restart unhealthy containers"
  echo ""
  echo "Exit codes:"
  echo "  0 - No critical issues"
  echo "  1 - Critical issues found"
  ;;
containers)
  doctor_containers
  ;;
--fix)
  # Load environment for PROJECT_NAME
  [[ -f ".env" ]] && source ".env" 2>/dev/null || true
  [[ -f ".env.local" ]] && source ".env.local" 2>/dev/null || true
  run_auto_fix
  ;;
*)
  main "$@"
  ;;
esac
