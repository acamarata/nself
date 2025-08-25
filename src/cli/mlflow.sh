#!/usr/bin/env bash
# mlflow.sh - Manage MLflow ML experiment tracking service

# Source shared utilities (only if not already sourced by wrapper)
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
[[ -z "${ENV_SOURCED:-}" ]] && source "$SCRIPT_DIR/../lib/utils/env.sh"
[[ -z "${DISPLAY_SOURCED:-}" ]] && source "$SCRIPT_DIR/../lib/utils/display.sh"
[[ -z "${DOCKER_UTILS_SOURCED:-}" ]] && source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Load environment with priority
load_env_with_priority

# Command function
cmd_mlflow() {
  local subcommand="${1:-status}"
  shift
  
  case "$subcommand" in
    enable)
      enable_mlflow "$@"
      ;;
    disable)
      disable_mlflow "$@"
      ;;
    status)
      show_mlflow_status "$@"
      ;;
    init)
      init_mlflow "$@"
      ;;
    url|urls)
      show_mlflow_urls "$@"
      ;;
    logs)
      show_mlflow_logs "$@"
      ;;
    restart)
      restart_mlflow "$@"
      ;;
    help|--help|-h)
      show_mlflow_help
      ;;
    *)
      echo_error "Unknown mlflow subcommand: $subcommand"
      show_mlflow_help
      return 1
      ;;
  esac
}

# Enable MLflow service
enable_mlflow() {
  show_header "Enabling MLflow"
  
  # Update .env or .env.local
  local env_file=".env.local"
  [[ -f ".env" ]] && env_file=".env"
  
  echo_info "Updating $env_file to enable MLflow..."
  
  # Check if MLFLOW_ENABLED exists
  if grep -q "^MLFLOW_ENABLED=" "$env_file" 2>/dev/null; then
    sed -i.bak 's/^MLFLOW_ENABLED=.*/MLFLOW_ENABLED=true/' "$env_file"
  else
    echo "MLFLOW_ENABLED=true" >> "$env_file"
  fi
  
  # Set default auth if not present
  if ! grep -q "^MLFLOW_AUTH_ENABLED=" "$env_file" 2>/dev/null; then
    echo "MLFLOW_AUTH_ENABLED=false" >> "$env_file"
  fi
  
  if ! grep -q "^MLFLOW_VERSION=" "$env_file" 2>/dev/null; then
    echo "MLFLOW_VERSION=2.9.2" >> "$env_file"
  fi
  
  echo_success "MLflow enabled in $env_file"
  
  # Rebuild and start
  echo_info "Rebuilding with MLflow enabled..."
  "$SCRIPT_DIR/build.sh" || return 1
  
  # Initialize MLflow (database and storage)
  echo_info "Initializing MLflow..."
  "$SCRIPT_DIR/../services/mlflow/init-mlflow.sh" || return 1
  
  # Start services
  echo_info "Starting services..."
  "$SCRIPT_DIR/start.sh" || return 1
  
  echo_success "MLflow enabled successfully!"
  show_mlflow_urls
}

# Disable MLflow service
disable_mlflow() {
  show_header "Disabling MLflow"
  
  # Update .env or .env.local
  local env_file=".env.local"
  [[ -f ".env" ]] && env_file=".env"
  
  echo_info "Updating $env_file to disable MLflow..."
  
  if grep -q "^MLFLOW_ENABLED=" "$env_file" 2>/dev/null; then
    sed -i.bak 's/^MLFLOW_ENABLED=.*/MLFLOW_ENABLED=false/' "$env_file"
  else
    echo "MLFLOW_ENABLED=false" >> "$env_file"
  fi
  
  echo_success "MLflow disabled in $env_file"
  
  # Stop MLflow container if running
  if docker ps --format "{{.Names}}" | grep -q "${PROJECT_NAME}-mlflow"; then
    echo_info "Stopping MLflow container..."
    docker stop "${PROJECT_NAME}-mlflow-1" 2>/dev/null || true
    docker rm "${PROJECT_NAME}-mlflow-1" 2>/dev/null || true
  fi
  
  # Rebuild without MLflow
  echo_info "Rebuilding without MLflow..."
  "$SCRIPT_DIR/build.sh" || return 1
  
  echo_success "MLflow disabled successfully!"
}

# Show MLflow status
show_mlflow_status() {
  show_header "MLflow Status"
  
  # Check if enabled
  local enabled="${MLFLOW_ENABLED:-false}"
  if [[ "$enabled" == "true" ]]; then
    echo_success "MLflow is ENABLED"
    
    # Check container status
    if docker ps --format "{{.Names}}" | grep -q "${PROJECT_NAME}-mlflow"; then
      echo_success "MLflow container is RUNNING"
      
      # Show container details
      echo
      echo "Container Details:"
      docker ps --filter "name=${PROJECT_NAME}-mlflow" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
      
      # Check health
      echo
      echo -n "Health Check: "
      if curl -s -o /dev/null -w "%{http_code}" "http://localhost:${MLFLOW_PORT}/health" | grep -q "200"; then
        echo_success "HEALTHY"
      else
        echo_warning "NOT RESPONDING"
      fi
      
      # Show URLs
      echo
      show_mlflow_urls
      
    else
      echo_warning "MLflow container is NOT RUNNING"
      echo_info "Run 'nself mlflow init' to initialize MLflow"
    fi
  else
    echo_info "MLflow is DISABLED"
    echo_info "Run 'nself mlflow enable' to enable MLflow"
  fi
  
  # Show configuration
  echo
  echo "Configuration:"
  echo "  Version: ${MLFLOW_VERSION:-2.9.2}"
  echo "  Port: ${MLFLOW_PORT:-5000}"
  echo "  Route: ${MLFLOW_ROUTE:-mlflow.${BASE_DOMAIN}}"
  echo "  Database: ${MLFLOW_DB_NAME:-mlflow}"
  echo "  Artifacts Bucket: ${MLFLOW_ARTIFACTS_BUCKET:-mlflow-artifacts}"
  echo "  Authentication: ${MLFLOW_AUTH_ENABLED:-false}"
}

# Initialize MLflow
init_mlflow() {
  if [[ "${MLFLOW_ENABLED:-false}" != "true" ]]; then
    echo_error "MLflow is not enabled. Run 'nself mlflow enable' first."
    return 1
  fi
  
  show_header "Initializing MLflow"
  "$SCRIPT_DIR/../services/mlflow/init-mlflow.sh"
}

# Show MLflow URLs
show_mlflow_urls() {
  echo "MLflow URLs:"
  echo "  Web UI: https://${MLFLOW_ROUTE:-mlflow.${BASE_DOMAIN}}"
  echo "  API Endpoint: http://localhost:${MLFLOW_PORT:-5000}"
  echo "  Tracking URI: postgresql://${POSTGRES_USER}@postgres:5432/${MLFLOW_DB_NAME:-mlflow}"
  echo "  Artifact Store: s3://${MLFLOW_ARTIFACTS_BUCKET:-mlflow-artifacts}/"
}

# Show MLflow logs
show_mlflow_logs() {
  local follow="${1:-}"
  
  if ! docker ps --format "{{.Names}}" | grep -q "${PROJECT_NAME}-mlflow"; then
    echo_error "MLflow container is not running"
    return 1
  fi
  
  if [[ "$follow" == "-f" ]] || [[ "$follow" == "--follow" ]]; then
    docker logs -f "${PROJECT_NAME}-mlflow-1"
  else
    docker logs --tail 50 "${PROJECT_NAME}-mlflow-1"
  fi
}

# Restart MLflow
restart_mlflow() {
  if [[ "${MLFLOW_ENABLED:-false}" != "true" ]]; then
    echo_error "MLflow is not enabled"
    return 1
  fi
  
  show_header "Restarting MLflow"
  
  if docker ps --format "{{.Names}}" | grep -q "${PROJECT_NAME}-mlflow"; then
    echo_info "Stopping MLflow container..."
    docker restart "${PROJECT_NAME}-mlflow-1"
    echo_success "MLflow restarted successfully"
  else
    echo_warning "MLflow container not found. Starting services..."
    "$SCRIPT_DIR/start.sh"
  fi
}

# Show MLflow help
show_mlflow_help() {
  show_header "nself mlflow - MLflow ML Experiment Tracking"
  echo "Usage: nself mlflow <subcommand> [options]"
  echo
  echo "Subcommands:"
  echo "  enable     Enable MLflow service"
  echo "  disable    Disable MLflow service"
  echo "  status     Show MLflow status (default)"
  echo "  init       Initialize MLflow database and storage"
  echo "  urls       Show MLflow URLs"
  echo "  logs       Show MLflow container logs"
  echo "  restart    Restart MLflow container"
  echo "  help       Show this help message"
  echo
  echo "Examples:"
  echo "  nself mlflow enable        # Enable and start MLflow"
  echo "  nself mlflow status        # Check MLflow status"
  echo "  nself mlflow urls          # Show MLflow access URLs"
  echo "  nself mlflow logs -f       # Follow MLflow logs"
  echo
  echo "MLflow provides:"
  echo "  • Experiment tracking and comparison"
  echo "  • Model versioning and registry"
  echo "  • Model deployment and serving"
  echo "  • Integration with popular ML frameworks"
  echo
  echo "Configuration (in .env or .env.local):"
  echo "  MLFLOW_ENABLED=true                    # Enable MLflow"
  echo "  MLFLOW_VERSION=2.9.2                    # MLflow version"
  echo "  MLFLOW_AUTH_ENABLED=true                # Enable authentication"
  echo "  MLFLOW_AUTH_USERNAME=admin              # Auth username"
  echo "  MLFLOW_AUTH_PASSWORD=secure-password    # Auth password"
}

# Run command if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_mlflow "$@"
fi