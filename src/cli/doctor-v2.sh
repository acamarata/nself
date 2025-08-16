#!/usr/bin/env bash

# doctor-v2.sh - Enhanced system diagnostics with enterprise features

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Initialize counters
ISSUES_FOUND=0
WARNINGS_FOUND=0
CHECKS_PASSED=0

# Check categories
declare -A CHECK_RESULTS
declare -A REMEDIATION_STEPS

# Function to record check result
record_check() {
  local category="$1"
  local check="$2"
  local status="$3"  # pass, warn, fail
  local message="$4"
  local remedy="${5:-}"
  
  CHECK_RESULTS["${category}:${check}"]="$status:$message"
  if [[ -n "$remedy" ]]; then
    REMEDIATION_STEPS["${category}:${check}"]="$remedy"
  fi
  
  case "$status" in
    pass) CHECKS_PASSED=$((CHECKS_PASSED + 1)) ;;
    warn) WARNINGS_FOUND=$((WARNINGS_FOUND + 1)) ;;
    fail) ISSUES_FOUND=$((ISSUES_FOUND + 1)) ;;
  esac
}

# Enhanced check functions
check_docker_setup() {
  echo ""
  show_section "Docker & Container Runtime"
  
  # Docker installation
  if command -v docker >/dev/null 2>&1; then
    local version=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    log_success "Docker installed: v$version"
    record_check "docker" "installation" "pass" "Docker v$version installed"
    
    # Docker daemon running
    if docker info >/dev/null 2>&1; then
      log_success "Docker daemon is running"
      record_check "docker" "daemon" "pass" "Docker daemon running"
    else
      log_error "Docker daemon is not running"
      record_check "docker" "daemon" "fail" "Docker daemon not running" \
        "Start Docker: sudo systemctl start docker (Linux) or open Docker Desktop (macOS)"
    fi
    
    # Docker Compose
    if docker compose version >/dev/null 2>&1; then
      local compose_version=$(docker compose version --short)
      log_success "Docker Compose installed: v$compose_version"
      record_check "docker" "compose" "pass" "Docker Compose v$compose_version"
    else
      log_error "Docker Compose not found"
      record_check "docker" "compose" "fail" "Docker Compose missing" \
        "Install Docker Compose plugin: docker compose version"
    fi
  else
    log_error "Docker not installed"
    record_check "docker" "installation" "fail" "Docker not installed" \
      "Install Docker: https://docs.docker.com/get-docker/"
  fi
}

check_ssl_certificates() {
  echo ""
  show_section "SSL/TLS Certificates"
  
  # Check for mkcert
  if command -v mkcert >/dev/null 2>&1; then
    log_success "mkcert is installed"
    record_check "ssl" "mkcert" "pass" "mkcert available"
    
    # Check if CA is installed
    if mkcert -check 2>/dev/null; then
      log_success "mkcert CA is trusted in system store"
      record_check "ssl" "ca_trust" "pass" "CA trusted"
    else
      log_warning "mkcert CA not trusted in system store"
      record_check "ssl" "ca_trust" "warn" "CA not trusted" \
        "Run: mkcert -install"
    fi
  else
    log_warning "mkcert not installed (needed for local HTTPS)"
    record_check "ssl" "mkcert" "warn" "mkcert not installed" \
      "Install mkcert: brew install mkcert (macOS) or see https://github.com/FiloSottile/mkcert"
  fi
  
  # Check certificate expiry if they exist
  local cert_dir="$PWD/certs/local"
  if [[ -f "$cert_dir/cert.pem" ]]; then
    local expiry=$(openssl x509 -enddate -noout -in "$cert_dir/cert.pem" 2>/dev/null | cut -d= -f2)
    local expiry_epoch=$(date -j -f "%b %d %T %Y %Z" "$expiry" "+%s" 2>/dev/null || date -d "$expiry" "+%s" 2>/dev/null || echo 0)
    local now_epoch=$(date "+%s")
    local days_left=$(( (expiry_epoch - now_epoch) / 86400 ))
    
    if [[ $days_left -gt 30 ]]; then
      log_success "Local certificates valid for $days_left days"
      record_check "ssl" "cert_expiry" "pass" "Certificates valid for $days_left days"
    elif [[ $days_left -gt 0 ]]; then
      log_warning "Local certificates expire in $days_left days"
      record_check "ssl" "cert_expiry" "warn" "Certificates expire soon" \
        "Run: nself ssl renew"
    else
      log_error "Local certificates have expired"
      record_check "ssl" "cert_expiry" "fail" "Certificates expired" \
        "Run: nself ssl renew"
    fi
  fi
}

check_network_ports() {
  echo ""
  show_section "Network Ports"
  
  local ports=(
    "80:HTTP/Nginx"
    "443:HTTPS/Nginx"
    "5432:PostgreSQL"
    "6379:Redis"
    "8080:Hasura GraphQL"
    "3000:Auth Service"
    "9090:Prometheus"
    "3100:Grafana"
  )
  
  for port_info in "${ports[@]}"; do
    IFS=':' read -r port service <<< "$port_info"
    
    local in_use=false
    if lsof -ti ":$port" >/dev/null 2>&1; then
      in_use=true
    fi
    
    if [[ "$in_use" == "true" ]]; then
      # Check if it's our container
      local container_using=$(docker ps --format "table {{.Names}}" | grep -E "(nself|nginx|postgres|redis|hasura)" | head -1 || echo "")
      if [[ -n "$container_using" ]]; then
        log_success "Port $port: In use by nself ($service)"
        record_check "ports" "port_$port" "pass" "Port $port used by nself"
      else
        log_warning "Port $port: In use by another process ($service)"
        record_check "ports" "port_$port" "warn" "Port $port blocked" \
          "Stop process using port $port or configure alternate port"
      fi
    else
      log_info "Port $port: Available ($service)"
      record_check "ports" "port_$port" "pass" "Port $port available"
    fi
  done
}

check_dns_configuration() {
  echo ""
  show_section "DNS & Hostname Resolution"
  
  # Check /etc/hosts entries
  local required_hosts=(
    "127.0.0.1 localhost"
    "127.0.0.1 api.localhost"
    "127.0.0.1 db.localhost"
    "127.0.0.1 auth.localhost"
  )
  
  for entry in "${required_hosts[@]}"; do
    local hostname=$(echo "$entry" | awk '{print $2}')
    if grep -q "$hostname" /etc/hosts 2>/dev/null; then
      log_success "Host entry exists: $hostname"
      record_check "dns" "hosts_$hostname" "pass" "$hostname configured"
    else
      log_warning "Missing /etc/hosts entry: $entry"
      record_check "dns" "hosts_$hostname" "warn" "Missing $hostname" \
        "Add to /etc/hosts: echo '$entry' | sudo tee -a /etc/hosts"
    fi
  done
  
  # Test DNS resolution
  if ping -c 1 -W 1 localhost >/dev/null 2>&1; then
    log_success "localhost resolves correctly"
    record_check "dns" "localhost_resolution" "pass" "localhost resolves"
  else
    log_error "Cannot resolve localhost"
    record_check "dns" "localhost_resolution" "fail" "localhost not resolving" \
      "Check /etc/hosts and DNS configuration"
  fi
}

check_system_resources() {
  echo ""
  show_section "System Resources"
  
  # Memory
  local total_mem_gb=0
  if [[ "$(uname)" == "Darwin" ]]; then
    total_mem_gb=$(( $(sysctl -n hw.memsize) / 1073741824 ))
  else
    total_mem_gb=$(( $(grep MemTotal /proc/meminfo | awk '{print $2}') / 1048576 ))
  fi
  
  if [[ $total_mem_gb -ge 8 ]]; then
    log_success "Memory: ${total_mem_gb}GB (recommended: 8GB+)"
    record_check "system" "memory" "pass" "${total_mem_gb}GB RAM available"
  elif [[ $total_mem_gb -ge 4 ]]; then
    log_warning "Memory: ${total_mem_gb}GB (minimum: 4GB, recommended: 8GB)"
    record_check "system" "memory" "warn" "${total_mem_gb}GB RAM (low)" \
      "Consider upgrading to 8GB+ for production"
  else
    log_error "Memory: ${total_mem_gb}GB (insufficient, need 4GB+)"
    record_check "system" "memory" "fail" "Insufficient RAM" \
      "Upgrade system memory to at least 4GB"
  fi
  
  # Disk space
  local available_gb=$(df -h . | tail -1 | awk '{print $4}' | sed 's/G//')
  if [[ $(echo "$available_gb > 10" | bc -l 2>/dev/null || echo 0) -eq 1 ]]; then
    log_success "Disk space: ${available_gb}GB available"
    record_check "system" "disk" "pass" "${available_gb}GB disk available"
  elif [[ $(echo "$available_gb > 5" | bc -l 2>/dev/null || echo 0) -eq 1 ]]; then
    log_warning "Disk space: ${available_gb}GB available (low)"
    record_check "system" "disk" "warn" "Low disk space" \
      "Free up disk space, need 10GB+ for comfortable operation"
  else
    log_error "Disk space: ${available_gb}GB (insufficient)"
    record_check "system" "disk" "fail" "Insufficient disk space" \
      "Free up disk space, need at least 5GB"
  fi
}

check_kernel_parameters() {
  echo ""
  show_section "Kernel Parameters (Production)"
  
  if [[ "$(uname)" == "Linux" ]]; then
    # Check important sysctl values for production
    local params=(
      "vm.max_map_count:262144:Elasticsearch/OpenSearch requirement"
      "fs.file-max:65536:File descriptor limit"
      "net.core.somaxconn:1024:Socket connection backlog"
    )
    
    for param_info in "${params[@]}"; do
      IFS=':' read -r param recommended description <<< "$param_info"
      local current=$(sysctl -n "$param" 2>/dev/null || echo "0")
      
      if [[ $current -ge $recommended ]]; then
        log_success "$param = $current (recommended: $recommended)"
        record_check "kernel" "$param" "pass" "$description configured"
      else
        log_warning "$param = $current (recommended: $recommended)"
        record_check "kernel" "$param" "warn" "$description not optimal" \
          "Run: sudo sysctl -w $param=$recommended"
      fi
    done
  else
    log_info "Kernel parameters check skipped (not Linux)"
  fi
}

generate_report() {
  echo ""
  echo ""
  show_header "nself Doctor Report"
  
  # Summary
  echo -e "${COLOR_BOLD}Summary:${COLOR_RESET}"
  echo -e "  ${COLOR_GREEN}✓${COLOR_RESET} Checks passed: $CHECKS_PASSED"
  if [[ $WARNINGS_FOUND -gt 0 ]]; then
    echo -e "  ${COLOR_YELLOW}⚠${COLOR_RESET} Warnings: $WARNINGS_FOUND"
  fi
  if [[ $ISSUES_FOUND -gt 0 ]]; then
    echo -e "  ${COLOR_RED}✗${COLOR_RESET} Issues: $ISSUES_FOUND"
  fi
  echo ""
  
  # Remediation steps
  if [[ ${#REMEDIATION_STEPS[@]} -gt 0 ]]; then
    echo -e "${COLOR_BOLD}Recommended Actions:${COLOR_RESET}"
    echo ""
    
    local step_num=1
    for key in "${!REMEDIATION_STEPS[@]}"; do
      local remedy="${REMEDIATION_STEPS[$key]}"
      echo -e "${COLOR_BLUE}$step_num.${COLOR_RESET} $remedy"
      step_num=$((step_num + 1))
    done
    echo ""
  fi
  
  # Overall status
  if [[ $ISSUES_FOUND -eq 0 && $WARNINGS_FOUND -eq 0 ]]; then
    log_success "System is healthy and ready for nself!"
  elif [[ $ISSUES_FOUND -eq 0 ]]; then
    log_warning "System is functional but has some warnings to address"
  else
    log_error "System has issues that need to be resolved"
  fi
}

# Main execution
cmd_doctor() {
  show_command_header "nself doctor" "System diagnostics and health checks"
  
  check_docker_setup
  check_ssl_certificates
  check_network_ports
  check_dns_configuration
  check_system_resources
  check_kernel_parameters
  
  generate_report
  
  # Return appropriate exit code
  if [[ $ISSUES_FOUND -gt 0 ]]; then
    return 1
  else
    return 0
  fi
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "doctor" || exit $?
  cmd_doctor
  exit_code=$?
  post_command "doctor" $exit_code
  exit $exit_code
fi