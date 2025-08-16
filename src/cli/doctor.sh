#!/usr/bin/env bash

# doctor.sh - System diagnostics and health checks for nself

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/progress.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

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

  if [[ -f ".env.local" ]]; then
    stop_spinner "success" ".env.local configuration file found"

    # Load environment safely
    start_spinner "Loading configuration"
    load_env_safe ".env.local" || {
      stop_spinner "error" "Failed to load .env.local - syntax error in configuration"
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
    stop_spinner "error" ".env.local configuration file not found"
    issue_found
    log_info "Run 'nself init' to create initial configuration"
  fi
}

# Function to check services status
check_services() {
  start_spinner "Checking docker-compose.yml"

  if [[ -f "docker-compose.yml" ]]; then
    stop_spinner "success" "docker-compose.yml found"

    # Check if any services are running
    start_spinner "Checking running services"
    local running_services=$(compose ps --services --filter "status=running" 2>/dev/null | wc -l)
    local total_services=$(compose config --services 2>/dev/null | wc -l)

    if [[ $running_services -gt 0 ]]; then
      stop_spinner "success" "$running_services/$total_services services are running"
    else
      stop_spinner "success" "No services are currently running"
      log_info "Run 'nself start' to start services"
    fi

    # Check for truly problematic containers (stopped when they should be running)
    start_spinner "Checking service health"
    local running_services=($(compose ps --services --filter "status=running" 2>/dev/null))
    local stopped_services=($(compose ps --services --filter "status=exited" 2>/dev/null))

    if [[ ${#stopped_services[@]} -gt 0 ]]; then
      stop_spinner "warning" "${#stopped_services[@]} services are stopped"
      warning_found
      log_info "Run 'nself status' for detailed service information"
    else
      stop_spinner "success" "All configured services are running"
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

# Function to check all service URLs
check_service_urls() {
  echo ""
  echo "Service URLs"
  echo "──────────────────────────────────────────────"

  if [[ ! -f ".env.local" ]]; then
    log_warning "No .env.local found - cannot determine service URLs"
    warning_found
    return
  fi

  load_env_safe ".env.local"
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
  echo ""
  echo "Recommendations"
  echo "──────────────────────────────────────────────"

  if [[ $ISSUES_FOUND -eq 0 ]] && [[ $WARNINGS_FOUND -eq 0 ]]; then
    log_success "Your nself installation looks great!"
    log_info "Ready for development. Run 'nself start' to start services."
    return
  fi

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
  log_info "  nself update           - Update to latest version"
  log_info "  nself status        - Check service details"
}

# Main function
main() {
  show_command_header "nself doctor" "System diagnostics and health checks"

  show_system_info

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

  check_service_urls
  show_recommendations

  # Show quick fix commands if issues found
  if [[ $ISSUES_FOUND -gt 0 ]]; then
    echo ""
    echo "Quick Fixes"
    echo "──────────────────────────────────────────────"

    if ! docker info >/dev/null 2>&1; then
      log_info "  Start Docker: 'sudo systemctl start docker' (Linux) or start Docker Desktop"
    fi

    if [[ ! -f ".env.local" ]]; then
      log_info "  Create config: 'nself init'"
    fi

    if [[ ! -f "docker-compose.yml" ]]; then
      log_info "  Generate structure: 'nself build'"
    fi

    log_info "  After fixes, run: 'nself doctor' again to verify"
  fi

  echo ""
  echo "──────────────────────────────────────────────"

  if [[ $ISSUES_FOUND -eq 0 ]]; then
    log_success "Health check completed - No critical issues found!"
    exit 0
  else
    log_error "Health check completed - $ISSUES_FOUND critical issue(s) found"
    exit 1
  fi
}

# Handle command line arguments
case "${1:-}" in
-h | --help)
  echo "nself doctor - System diagnostics and health checks"
  echo ""
  echo "Usage: nself doctor [options]"
  echo ""
  echo "Options:"
  echo "  -h, --help     Show this help message"
  echo "  -v, --verbose  Verbose output"
  echo ""
  echo "This command checks:"
  echo "  • System requirements (Docker, memory, disk)"
  echo "  • Network connectivity"
  echo "  • nself configuration"
  echo "  • Service status"
  echo "  • SSL certificates"
  echo ""
  echo "Exit codes:"
  echo "  0 - No critical issues"
  echo "  1 - Critical issues found"
  ;;
*)
  main "$@"
  ;;
esac
