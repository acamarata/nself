#!/usr/bin/env bash

# metrics.sh - Metrics and observability management

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/header.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Show help
show_metrics_help() {
  echo "nself metrics - Metrics and observability management"
  echo ""
  echo "Usage: nself metrics <command> [options]"
  echo ""
  echo "Commands:"
  echo "  enable             Enable metrics collection"
  echo "  disable            Disable metrics collection"
  echo "  status             Show metrics status"
  echo "  dashboard          Open metrics dashboard"
  echo "  export             Export metrics data"
  echo "  configure          Configure metrics providers"
  echo ""
  echo "Options:"
  echo "  --provider <name>  Metrics provider (prometheus, datadog, newrelic)"
  echo "  --port <port>      Metrics endpoint port (default: 9090)"
  echo "  --interval <sec>   Collection interval in seconds (default: 30)"
  echo "  --retention <days> Data retention in days (default: 30)"
  echo "  -h, --help         Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself metrics enable"
  echo "  nself metrics enable --provider prometheus"
  echo "  nself metrics status"
  echo "  nself metrics dashboard"
  echo "  nself metrics export --format json --output metrics.json"
}

# Check metrics status
check_metrics_status() {
  local enabled=false
  local provider="none"
  local endpoint=""
  
  # Check if metrics are enabled in .env.local
  if [[ -f ".env.local" ]]; then
    set -a
    source .env.local
    set +a
    
    if [[ "${METRICS_ENABLED:-false}" == "true" ]]; then
      enabled=true
      provider="${METRICS_PROVIDER:-prometheus}"
      endpoint="${METRICS_ENDPOINT:-http://localhost:9090}"
    fi
  fi
  
  # Check if metrics config exists
  if [[ -f ".nself/metrics.conf" ]]; then
    set -a
    source .nself/metrics.conf
    set +a
    enabled="${METRICS_ENABLED:-false}"
    provider="${METRICS_PROVIDER:-prometheus}"
  fi
  
  echo "$enabled|$provider|$endpoint"
}

# Enable metrics
enable_metrics() {
  local provider="${1:-prometheus}"
  local port="${2:-9090}"
  local interval="${3:-30}"
  local retention="${4:-30}"
  
  show_command_header "nself metrics enable" "Enabling metrics collection"
  echo ""
  
  printf "${COLOR_CYAN}➞ Configuration${COLOR_RESET}\n"
  echo "  Provider: $provider"
  echo "  Port: $port"
  echo "  Collection interval: ${interval}s"
  echo "  Data retention: ${retention} days"
  echo ""
  
  # Update .env.local
  if [[ -f ".env.local" ]]; then
    # Add or update metrics configuration
    if grep -q "^METRICS_ENABLED=" .env.local; then
      if ! sed -i.bak "s/^METRICS_ENABLED=.*/METRICS_ENABLED=true/" .env.local; then
        log_error "Failed to update METRICS_ENABLED"
        return 1
      fi
    else
      echo "" >> .env.local
      echo "# Metrics configuration" >> .env.local
      echo "METRICS_ENABLED=true" >> .env.local
    fi
    
    if grep -q "^METRICS_PROVIDER=" .env.local; then
      sed -i.bak "s/^METRICS_PROVIDER=.*/METRICS_PROVIDER=$provider/" .env.local
    else
      echo "METRICS_PROVIDER=$provider" >> .env.local
    fi
    
    if grep -q "^METRICS_PORT=" .env.local; then
      sed -i.bak "s/^METRICS_PORT=.*/METRICS_PORT=$port/" .env.local
    else
      echo "METRICS_PORT=$port" >> .env.local
    fi
    
    if grep -q "^METRICS_INTERVAL=" .env.local; then
      sed -i.bak "s/^METRICS_INTERVAL=.*/METRICS_INTERVAL=$interval/" .env.local
    else
      echo "METRICS_INTERVAL=$interval" >> .env.local
    fi
    
    if grep -q "^METRICS_RETENTION=" .env.local; then
      sed -i.bak "s/^METRICS_RETENTION=.*/METRICS_RETENTION=$retention/" .env.local
    else
      echo "METRICS_RETENTION=$retention" >> .env.local
    fi
    
    rm -f .env.local.bak
  fi
  
  # Create metrics config
  mkdir -p .nself
  cat > .nself/metrics.conf <<EOF
METRICS_ENABLED=true
METRICS_PROVIDER=$provider
METRICS_PORT=$port
METRICS_INTERVAL=$interval
METRICS_RETENTION=$retention
METRICS_ENDPOINT=http://localhost:$port
EOF
  
  # Add provider-specific configuration
  case "$provider" in
    prometheus)
      cat > .nself/prometheus.yml <<EOF
global:
  scrape_interval: ${interval}s
  evaluation_interval: ${interval}s

scrape_configs:
  - job_name: 'nself'
    static_configs:
      - targets: ['localhost:9090']
    
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres:9187']
    
  - job_name: 'nginx'
    static_configs:
      - targets: ['nginx:9113']
    
  - job_name: 'hasura'
    static_configs:
      - targets: ['hasura:8080']
    
  - job_name: 'redis'
    static_configs:
      - targets: ['redis:9121']
EOF
      
      log_info "Prometheus configuration created"
      ;;
    
    datadog)
      cat > .nself/datadog.yaml <<EOF
api_key: YOUR_DATADOG_API_KEY
site: datadoghq.com
hostname: nself-local
tags:
  - env:development
  - project:${PROJECT_NAME:-nself}

logs_enabled: true
process_config:
  enabled: true

apm_config:
  enabled: true
  env: development
EOF
      
      log_warning "Update .nself/datadog.yaml with your API key"
      ;;
    
    newrelic)
      cat > .nself/newrelic.yml <<EOF
license_key: YOUR_NEW_RELIC_LICENSE_KEY
app_name: ${PROJECT_NAME:-nself}
monitor_mode: true
developer_mode: false
log_level: info

distributed_tracing:
  enabled: true

slow_sql:
  enabled: true
  record_sql: obfuscated
  explain_threshold: 500
EOF
      
      log_warning "Update .nself/newrelic.yml with your license key"
      ;;
  esac
  
  echo ""
  log_success "Metrics collection enabled"
  
  # Add metrics service to docker-compose.override.yml
  if [[ "$provider" == "prometheus" ]]; then
    local override_file="docker-compose.override.yml"
    
    if [[ ! -f "$override_file" ]]; then
      cat > "$override_file" <<EOF
version: '3.8'
services:
EOF
    fi
    
    # Check if prometheus already exists
    if ! grep -q "^  prometheus:" "$override_file"; then
      cat >> "$override_file" <<EOF
  prometheus:
    image: prom/prometheus:latest
    container_name: \${PROJECT_NAME:-nself}-prometheus
    volumes:
      - ./.nself/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=${retention}d'
      - '--web.console.libraries=/usr/share/prometheus/console_libraries'
      - '--web.console.templates=/usr/share/prometheus/consoles'
    ports:
      - "${port}:9090"
    restart: unless-stopped
    networks:
      - default

volumes:
  prometheus_data:
EOF
      
      log_info "Prometheus service added to docker-compose.override.yml"
    fi
    
    echo ""
    log_info "Run 'nself build && nself restart' to apply changes"
  fi
  
  return 0
}

# Disable metrics
disable_metrics() {
  show_command_header "nself metrics disable" "Disabling metrics collection"
  echo ""
  
  # Update .env.local
  if [[ -f ".env.local" ]]; then
    if grep -q "^METRICS_ENABLED=" .env.local; then
      sed -i.bak "s/^METRICS_ENABLED=.*/METRICS_ENABLED=false/" .env.local
      rm -f .env.local.bak
    fi
  fi
  
  # Update metrics config
  if [[ -f ".nself/metrics.conf" ]]; then
    sed -i.bak "s/^METRICS_ENABLED=.*/METRICS_ENABLED=false/" .nself/metrics.conf
    rm -f .nself/metrics.conf.bak
  fi
  
  # Stop metrics container if running
  if docker compose ps 2>/dev/null | grep -q prometheus; then
    log_info "Stopping prometheus container..."
    docker compose stop prometheus 2>/dev/null || true
  fi
  
  log_success "Metrics collection disabled"
  echo ""
  log_info "Metrics configuration preserved in .nself/"
  log_info "Run 'nself metrics enable' to re-enable"
  
  return 0
}

# Show metrics status
show_metrics_status() {
  show_command_header "nself metrics status" "Metrics collection status"
  echo ""
  
  local status_info
  status_info=$(check_metrics_status)
  
  IFS='|' read -r enabled provider endpoint <<< "$status_info"
  
  printf "${COLOR_CYAN}➞ Status${COLOR_RESET}\n"
  echo ""
  
  if [[ "$enabled" == "true" ]]; then
    printf "  ${COLOR_GREEN}●${COLOR_RESET} Metrics: Enabled\n"
    echo "  Provider: $provider"
    echo "  Endpoint: $endpoint"
    
    # Check if provider container is running
    case "$provider" in
      prometheus)
        if docker compose ps 2>/dev/null | grep -q prometheus; then
          printf "  Container: ${COLOR_GREEN}Running${COLOR_RESET}\n"
        else
          printf "  Container: ${COLOR_RED}Not running${COLOR_RESET}\n"
          echo ""
          log_info "Run 'nself restart' to start metrics collection"
        fi
        ;;
    esac
    
    echo ""
    
    # Show collected metrics summary
    printf "${COLOR_CYAN}➞ Collected Metrics${COLOR_RESET}\n"
    echo ""
    
    # List services being monitored
    local services=(postgres nginx hasura redis)
    for service in "${services[@]}"; do
      if docker compose ps 2>/dev/null | grep -q "$service"; then
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} %s\n" "$service"
      else
        printf "  ${COLOR_DIM}○${COLOR_RESET} %s\n" "$service"
      fi
    done
    
  else
    printf "  ${COLOR_RED}●${COLOR_RESET} Metrics: Disabled\n"
    echo ""
    log_info "Run 'nself metrics enable' to start collecting metrics"
  fi
  
  return 0
}

# Open metrics dashboard
open_dashboard() {
  local status_info
  status_info=$(check_metrics_status)
  
  IFS='|' read -r enabled provider endpoint <<< "$status_info"
  
  if [[ "$enabled" != "true" ]]; then
    log_error "Metrics collection is not enabled"
    echo ""
    log_info "Run 'nself metrics enable' first"
    return 1
  fi
  
  show_command_header "nself metrics dashboard" "Opening metrics dashboard"
  echo ""
  
  case "$provider" in
    prometheus)
      local url="http://localhost:${METRICS_PORT:-9090}"
      printf "${COLOR_CYAN}➞ Dashboard URL:${COLOR_RESET} %s\n" "$url"
      echo ""
      
      # Try to open in browser
      if command -v open >/dev/null 2>&1; then
        open "$url"
        log_success "Opening dashboard in browser..."
      elif command -v xdg-open >/dev/null 2>&1; then
        xdg-open "$url"
        log_success "Opening dashboard in browser..."
      else
        log_info "Open this URL in your browser: $url"
      fi
      ;;
    
    datadog|newrelic)
      log_info "Dashboard available in $provider web console"
      ;;
    
    *)
      log_error "Unknown provider: $provider"
      return 1
      ;;
  esac
  
  return 0
}

# Export metrics
export_metrics() {
  local format="${1:-json}"
  local output="${2:-metrics.json}"
  
  show_command_header "nself metrics export" "Exporting metrics data"
  echo ""
  
  local status_info
  status_info=$(check_metrics_status)
  
  IFS='|' read -r enabled provider endpoint <<< "$status_info"
  
  if [[ "$enabled" != "true" ]]; then
    log_error "Metrics collection is not enabled"
    return 1
  fi
  
  printf "${COLOR_CYAN}➞ Export Configuration${COLOR_RESET}\n"
  echo "  Format: $format"
  echo "  Output: $output"
  echo ""
  
  case "$provider" in
    prometheus)
      # Query prometheus API for metrics
      local api_url="http://localhost:${METRICS_PORT:-9090}/api/v1"
      
      log_info "Querying metrics from Prometheus..."
      
      # Check if curl is available
      if ! command -v curl >/dev/null 2>&1; then
        log_error "curl is required for exporting metrics"
        return 1
      fi
      
      # Get current metrics
      if curl -s "${api_url}/query?query=up" > "$output" 2>/dev/null; then
        log_success "Metrics exported to $output"
      else
        log_error "Failed to export metrics"
        return 1
      fi
      ;;
    
    *)
      log_error "Export not supported for provider: $provider"
      return 1
      ;;
  esac
  
  return 0
}

# Configure metrics provider
configure_metrics() {
  local provider="${1:-prometheus}"
  
  show_command_header "nself metrics configure" "Configure metrics provider"
  echo ""
  
  case "$provider" in
    prometheus)
      log_info "Editing Prometheus configuration..."
      
      if [[ -f ".nself/prometheus.yml" ]]; then
        # Try to find an available editor
        local editor="${EDITOR:-}"
        
        if [[ -z "$editor" ]]; then
          if command -v nano >/dev/null 2>&1; then
            editor="nano"
          elif command -v vim >/dev/null 2>&1; then
            editor="vim"
          elif command -v vi >/dev/null 2>&1; then
            editor="vi"
          else
            log_error "No text editor found. Set EDITOR environment variable."
            return 1
          fi
        fi
        
        "$editor" .nself/prometheus.yml
      else
        log_error "Prometheus configuration not found"
        log_info "Run 'nself metrics enable --provider prometheus' first"
        return 1
      fi
      ;;
    
    datadog)
      log_info "Editing Datadog configuration..."
      
      if [[ -f ".nself/datadog.yaml" ]]; then
        # Try to find an available editor
        local editor="${EDITOR:-}"
        
        if [[ -z "$editor" ]]; then
          if command -v nano >/dev/null 2>&1; then
            editor="nano"
          elif command -v vim >/dev/null 2>&1; then
            editor="vim"
          elif command -v vi >/dev/null 2>&1; then
            editor="vi"
          else
            log_error "No text editor found. Set EDITOR environment variable."
            return 1
          fi
        fi
        
        "$editor" .nself/datadog.yaml
      else
        log_error "Datadog configuration not found"
        log_info "Run 'nself metrics enable --provider datadog' first"
        return 1
      fi
      ;;
    
    newrelic)
      log_info "Editing New Relic configuration..."
      
      if [[ -f ".nself/newrelic.yml" ]]; then
        # Try to find an available editor
        local editor="${EDITOR:-}"
        
        if [[ -z "$editor" ]]; then
          if command -v nano >/dev/null 2>&1; then
            editor="nano"
          elif command -v vim >/dev/null 2>&1; then
            editor="vim"
          elif command -v vi >/dev/null 2>&1; then
            editor="vi"
          else
            log_error "No text editor found. Set EDITOR environment variable."
            return 1
          fi
        fi
        
        "$editor" .nself/newrelic.yml
      else
        log_error "New Relic configuration not found"
        log_info "Run 'nself metrics enable --provider newrelic' first"
        return 1
      fi
      ;;
    
    *)
      log_error "Unknown provider: $provider"
      return 1
      ;;
  esac
  
  log_success "Configuration updated"
  log_info "Restart services to apply changes: nself restart"
  
  return 0
}

# Main metrics command
cmd_metrics() {
  local subcommand="${1:-status}"
  shift || true
  
  case "$subcommand" in
    enable)
      local provider=""
      local port=""
      local interval=""
      local retention=""
      
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --provider)
            provider="$2"
            shift 2
            ;;
          --port)
            port="$2"
            shift 2
            ;;
          --interval)
            interval="$2"
            shift 2
            ;;
          --retention)
            retention="$2"
            shift 2
            ;;
          *)
            shift
            ;;
        esac
      done
      
      enable_metrics "$provider" "$port" "$interval" "$retention"
      ;;
    
    disable)
      disable_metrics
      ;;
    
    status)
      show_metrics_status
      ;;
    
    dashboard)
      open_dashboard
      ;;
    
    export)
      local format="json"
      local output="metrics.json"
      
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --format)
            format="$2"
            shift 2
            ;;
          --output)
            output="$2"
            shift 2
            ;;
          *)
            shift
            ;;
        esac
      done
      
      export_metrics "$format" "$output"
      ;;
    
    configure)
      local provider="${1:-prometheus}"
      configure_metrics "$provider"
      ;;
    
    -h|--help|help)
      show_metrics_help
      ;;
    
    *)
      log_error "Unknown command: $subcommand"
      echo ""
      show_metrics_help
      return 1
      ;;
  esac
}

# Export for use as library
export -f cmd_metrics

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "metrics" || exit $?
  cmd_metrics "$@"
  exit_code=$?
  post_command "metrics" $exit_code
  exit $exit_code
fi