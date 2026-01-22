#!/usr/bin/env bash

# mlflow.sh - MLflow ML experiment tracking management

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/utils/platform-compat.sh"

# MLflow command
cmd_mlflow() {
  local subcommand="${1:-status}"
  shift || true

  case "$subcommand" in
    status)
      mlflow_status
      ;;
    enable)
      mlflow_enable
      ;;
    disable)
      mlflow_disable
      ;;
    open|ui|dashboard)
      mlflow_open
      ;;
    configure)
      mlflow_configure "$@"
      ;;
    logs)
      mlflow_logs "$@"
      ;;
    test)
      mlflow_test
      ;;
    experiments|exp)
      mlflow_experiments "$@"
      ;;
    runs)
      mlflow_runs "$@"
      ;;
    help|--help|-h)
      mlflow_help
      ;;
    *)
      log_error "Unknown mlflow command: $subcommand"
      mlflow_help
      exit 1
      ;;
  esac
}

# Show MLflow status
mlflow_status() {
  show_command_header "MLflow" "Checking MLflow status"
  
  load_env_with_priority
  ensure_project_context
  
  if [[ "${MLFLOW_ENABLED:-false}" != "true" ]]; then
    log_info "MLflow is disabled"
    log_info "Enable with: nself mlflow enable"
    return 0
  fi
  
  if is_service_running "mlflow"; then
    log_success "MLflow is running"
    log_info "URL: http://localhost:${MLFLOW_PORT:-5000}"
    log_info "Username: ${MLFLOW_USERNAME:-admin}"
    
    # Check health
    if curl -s "http://localhost:${MLFLOW_PORT:-5000}/health" >/dev/null 2>&1; then
      log_success "MLflow is healthy"
    else
      log_warning "MLflow is not responding"
    fi
  else
    log_warning "MLflow is not running"
    log_info "Start with: nself start"
  fi
}

# Enable MLflow
mlflow_enable() {
  show_command_header "MLflow" "Enabling MLflow ML tracking"
  
  load_env_with_priority
  ensure_project_context
  
  # Update .env file
  if grep -q "^MLFLOW_ENABLED=" .env 2>/dev/null; then
    safe_sed_inline .env 's/^MLFLOW_ENABLED=.*/MLFLOW_ENABLED=true/'
  else
    echo "MLFLOW_ENABLED=true" >> .env
  fi
  
  # Set credentials if not already set
  if ! grep -q "^MLFLOW_USERNAME=" .env 2>/dev/null; then
    echo "MLFLOW_USERNAME=admin" >> .env
  fi
  
  if ! grep -q "^MLFLOW_PASSWORD=" .env 2>/dev/null; then
    local password=$(openssl rand -base64 12 | tr -d "=+/" | cut -c1-16)
    echo "MLFLOW_PASSWORD=$password" >> .env
    log_info "Generated MLflow password: $password"
  fi
  
  log_success "MLflow enabled"
  log_info "Storage and PostgreSQL must be enabled for MLflow to work properly"
  log_info "Rebuild and restart to apply changes:"
  log_info "  nself build"
  log_info "  nself restart"
}

# Disable MLflow
mlflow_disable() {
  show_command_header "MLflow" "Disabling MLflow"
  
  load_env_with_priority
  ensure_project_context
  
  # Update .env file
  if grep -q "^MLFLOW_ENABLED=" .env 2>/dev/null; then
    safe_sed_inline .env 's/^MLFLOW_ENABLED=.*/MLFLOW_ENABLED=false/'
  else
    echo "MLFLOW_ENABLED=false" >> .env
  fi
  
  log_success "MLflow disabled"
  log_info "Rebuild and restart to apply changes:"
  log_info "  nself build"
  log_info "  nself restart"
}

# Open MLflow UI
mlflow_open() {
  show_command_header "MLflow" "Opening MLflow UI"
  
  load_env_with_priority
  ensure_project_context
  
  if [[ "${MLFLOW_ENABLED:-false}" != "true" ]]; then
    log_error "MLflow is not enabled"
    exit 1
  fi
  
  if ! is_service_running "mlflow"; then
    log_error "MLflow is not running"
    log_info "Start with: nself start"
    exit 1
  fi
  
  local url="http://localhost:${MLFLOW_PORT:-5000}"
  log_info "Opening MLflow UI: $url"
  log_info "Username: ${MLFLOW_USERNAME:-admin}"
  log_info "Password: ${MLFLOW_PASSWORD:-[check .env file]}"
  
  if command -v open >/dev/null 2>&1; then
    open "$url"
  elif command -v xdg-open >/dev/null 2>&1; then
    xdg-open "$url"
  else
    log_info "Please open in browser: $url"
  fi
}

# Configure MLflow
mlflow_configure() {
  local setting="${1:-}"
  local value="${2:-}"
  
  if [[ -z "$setting" ]]; then
    log_error "Setting required"
    log_info "Usage: nself mlflow configure <setting> <value>"
    log_info "Settings: username, password, port"
    exit 1
  fi
  
  show_command_header "MLflow" "Configuring MLflow: $setting"
  
  load_env_with_priority
  ensure_project_context
  
  case "$setting" in
    username)
      if [[ -z "$value" ]]; then
        log_error "Username required"
        exit 1
      fi
      safe_sed_inline .env "s/^MLFLOW_USERNAME=.*/MLFLOW_USERNAME=$value/" 2>/dev/null || echo "MLFLOW_USERNAME=$value" >> .env
      log_success "MLflow username set to: $value"
      ;;
    password)
      if [[ -z "$value" ]]; then
        value=$(openssl rand -base64 12 | tr -d "=+/" | cut -c1-16)
        log_info "Generated password: $value"
      fi
      safe_sed_inline .env "s/^MLFLOW_PASSWORD=.*/MLFLOW_PASSWORD=$value/" 2>/dev/null || echo "MLFLOW_PASSWORD=$value" >> .env
      log_success "MLflow password updated"
      ;;
    port)
      if [[ -z "$value" ]]; then
        log_error "Port required"
        exit 1
      fi
      safe_sed_inline .env "s/^MLFLOW_PORT=.*/MLFLOW_PORT=$value/" 2>/dev/null || echo "MLFLOW_PORT=$value" >> .env
      log_success "MLflow port set to: $value"
      ;;
    *)
      log_error "Unknown setting: $setting"
      log_info "Valid settings: username, password, port"
      exit 1
      ;;
  esac
  
  log_info "Restart MLflow to apply changes:"
  log_info "  nself restart mlflow"
}

# View MLflow logs
mlflow_logs() {
  local follow="${1:-}"
  
  show_command_header "MLflow" "Viewing MLflow logs"
  
  load_env_with_priority
  ensure_project_context
  
  if [[ "$follow" == "-f" ]] || [[ "$follow" == "--follow" ]]; then
    compose logs -f mlflow
  else
    compose logs --tail=50 mlflow
  fi
}

# Test MLflow connection
mlflow_test() {
  show_command_header "MLflow" "Testing MLflow connection"
  
  load_env_with_priority
  ensure_project_context
  
  if [[ "${MLFLOW_ENABLED:-false}" != "true" ]]; then
    log_error "MLflow is not enabled"
    exit 1
  fi
  
  if ! is_service_running "mlflow"; then
    log_error "MLflow is not running"
    exit 1
  fi
  
  log_info "Testing MLflow API..."
  
  # Test health endpoint
  if curl -s "http://localhost:${MLFLOW_PORT:-5000}/health" | grep -q "OK"; then
    log_success "MLflow health check passed"
  else
    log_error "MLflow health check failed"
    exit 1
  fi
  
  # Test with Python if available
  if command -v python3 >/dev/null 2>&1; then
    log_info "Testing Python MLflow client..."
    python3 -c "
import sys
try:
    import mlflow
    mlflow.set_tracking_uri('http://localhost:${MLFLOW_PORT:-5000}')
    # Try to list experiments
    experiments = mlflow.search_experiments()
    print(f'Successfully connected. Found {len(experiments)} experiments.')
    sys.exit(0)
except ImportError:
    print('MLflow Python package not installed. Install with: pip install mlflow')
    sys.exit(1)
except Exception as e:
    print(f'Failed to connect: {e}')
    sys.exit(1)
" && log_success "Python client test passed" || log_warning "Python client test failed (install mlflow: pip install mlflow)"
  else
    log_info "Python not available, skipping client test"
  fi
  
  log_info ""
  log_info "To use MLflow in your code:"
  log_info "  Python: mlflow.set_tracking_uri('http://localhost:${MLFLOW_PORT:-5000}')"
  log_info "  R: mlflow_set_tracking_uri('http://localhost:${MLFLOW_PORT:-5000}')"
  log_info "  Environment: export MLFLOW_TRACKING_URI=http://localhost:${MLFLOW_PORT:-5000}"
}

# List/manage experiments
mlflow_experiments() {
  local action="${1:-list}"
  shift || true

  show_command_header "MLflow" "Experiments"

  load_env_with_priority
  ensure_project_context

  if [[ "${MLFLOW_ENABLED:-false}" != "true" ]]; then
    log_error "MLflow is not enabled"
    exit 1
  fi

  if ! is_service_running "mlflow"; then
    log_error "MLflow is not running"
    log_info "Start with: nself start"
    exit 1
  fi

  local mlflow_url="http://localhost:${MLFLOW_PORT:-5000}"

  case "$action" in
    list)
      log_info "Listing experiments from $mlflow_url..."
      echo

      if command -v curl >/dev/null 2>&1; then
        local response=$(curl -s "$mlflow_url/api/2.0/mlflow/experiments/search" \
          -H "Content-Type: application/json" \
          -d '{"max_results": 100}' 2>/dev/null)

        if echo "$response" | grep -q "experiments"; then
          echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
experiments = data.get('experiments', [])
if not experiments:
    print('  No experiments found')
    print('')
    print('  Create your first experiment:')
    print('    nself mlflow experiments create my-experiment')
else:
    print(f'  Found {len(experiments)} experiment(s):')
    print('')
    print(f'  {\"ID\":<8} {\"NAME\":<30} {\"STATE\":<10} ARTIFACT LOCATION')
    print(f'  {\"-\"*8} {\"-\"*30} {\"-\"*10} {\"----------\"}')
    for exp in experiments:
        exp_id = exp.get('experiment_id', 'N/A')
        name = exp.get('name', 'N/A')[:28]
        state = exp.get('lifecycle_stage', 'N/A')
        location = exp.get('artifact_location', 'N/A')[:40]
        print(f'  {exp_id:<8} {name:<30} {state:<10} {location}')
" 2>/dev/null || {
            # Fallback if python parsing fails
            log_info "Raw response:"
            echo "$response" | head -20
          }
        else
          log_warning "Could not fetch experiments"
          log_info "Response: $response"
        fi
      else
        log_warning "curl not available"
      fi
      ;;

    create)
      local exp_name="${1:-}"
      if [[ -z "$exp_name" ]]; then
        log_error "Experiment name required"
        log_info "Usage: nself mlflow experiments create <name>"
        exit 1
      fi

      log_info "Creating experiment: $exp_name"

      if command -v curl >/dev/null 2>&1; then
        local response=$(curl -s -X POST "$mlflow_url/api/2.0/mlflow/experiments/create" \
          -H "Content-Type: application/json" \
          -d "{\"name\": \"$exp_name\"}" 2>/dev/null)

        if echo "$response" | grep -q "experiment_id"; then
          local exp_id=$(echo "$response" | grep -o '"experiment_id":"[^"]*"' | cut -d'"' -f4)
          log_success "Experiment created: $exp_name (ID: $exp_id)"
        else
          log_error "Failed to create experiment"
          log_info "Response: $response"
        fi
      fi
      ;;

    delete)
      local exp_id="${1:-}"
      if [[ -z "$exp_id" ]]; then
        log_error "Experiment ID required"
        log_info "Usage: nself mlflow experiments delete <experiment_id>"
        exit 1
      fi

      log_info "Deleting experiment: $exp_id"

      if command -v curl >/dev/null 2>&1; then
        local response=$(curl -s -X POST "$mlflow_url/api/2.0/mlflow/experiments/delete" \
          -H "Content-Type: application/json" \
          -d "{\"experiment_id\": \"$exp_id\"}" 2>/dev/null)

        if [[ -z "$response" ]] || echo "$response" | grep -q "{}"; then
          log_success "Experiment deleted: $exp_id"
        else
          log_error "Failed to delete experiment"
          log_info "Response: $response"
        fi
      fi
      ;;

    *)
      log_error "Unknown experiments action: $action"
      log_info "Actions: list, create <name>, delete <id>"
      exit 1
      ;;
  esac
}

# List runs for an experiment
mlflow_runs() {
  local exp_id="${1:-}"

  show_command_header "MLflow" "Runs"

  load_env_with_priority
  ensure_project_context

  if [[ "${MLFLOW_ENABLED:-false}" != "true" ]]; then
    log_error "MLflow is not enabled"
    exit 1
  fi

  if ! is_service_running "mlflow"; then
    log_error "MLflow is not running"
    exit 1
  fi

  local mlflow_url="http://localhost:${MLFLOW_PORT:-5000}"

  if [[ -z "$exp_id" ]]; then
    log_info "Listing runs from all experiments..."
    exp_id=""
  else
    log_info "Listing runs for experiment: $exp_id"
  fi

  echo

  if command -v curl >/dev/null 2>&1; then
    local body="{\"max_results\": 50"
    if [[ -n "$exp_id" ]]; then
      body="$body, \"experiment_ids\": [\"$exp_id\"]"
    fi
    body="$body}"

    local response=$(curl -s -X POST "$mlflow_url/api/2.0/mlflow/runs/search" \
      -H "Content-Type: application/json" \
      -d "$body" 2>/dev/null)

    if echo "$response" | grep -q "runs"; then
      echo "$response" | python3 -c "
import sys, json
from datetime import datetime
data = json.load(sys.stdin)
runs = data.get('runs', [])
if not runs:
    print('  No runs found')
else:
    print(f'  Found {len(runs)} run(s):')
    print('')
    print(f'  {\"RUN ID\":<36} {\"STATUS\":<10} {\"START TIME\":<20} EXPERIMENT')
    print(f'  {\"-\"*36} {\"-\"*10} {\"-\"*20} ----------')
    for run in runs[:20]:
        info = run.get('info', {})
        run_id = info.get('run_id', 'N/A')[:34]
        status = info.get('status', 'N/A')
        start_ms = info.get('start_time', 0)
        if start_ms:
            start_time = datetime.fromtimestamp(start_ms/1000).strftime('%Y-%m-%d %H:%M')
        else:
            start_time = 'N/A'
        exp_id = info.get('experiment_id', 'N/A')
        print(f'  {run_id:<36} {status:<10} {start_time:<20} {exp_id}')
" 2>/dev/null || {
        log_info "Raw response:"
        echo "$response" | head -20
      }
    else
      log_warning "Could not fetch runs"
    fi
  else
    log_warning "curl not available"
  fi
}

# Show help
mlflow_help() {
  cat <<EOF
${COLOR_BLUE}nself mlflow${COLOR_RESET} - Manage MLflow ML experiment tracking

${COLOR_YELLOW}Usage:${COLOR_RESET}
  nself mlflow <command> [options]

${COLOR_YELLOW}Commands:${COLOR_RESET}
  status              Show MLflow status
  enable              Enable MLflow service
  disable             Disable MLflow service
  open                Open MLflow UI in browser
  configure <s> <v>   Configure MLflow settings
  experiments         List/manage experiments
  runs [exp_id]       List runs (optionally for specific experiment)
  logs [-f]           View MLflow logs
  test                Test MLflow connection
  help                Show this help message

${COLOR_YELLOW}Experiments:${COLOR_RESET}
  nself mlflow experiments             List all experiments
  nself mlflow experiments create <n>  Create new experiment
  nself mlflow experiments delete <id> Delete experiment
  nself mlflow runs                    List all runs
  nself mlflow runs <experiment_id>    List runs for experiment

${COLOR_YELLOW}Configuration:${COLOR_RESET}
  nself mlflow configure username <name>   Set username
  nself mlflow configure password [pass]   Set password
  nself mlflow configure port <port>       Set port

${COLOR_YELLOW}Examples:${COLOR_RESET}
  nself mlflow enable                        # Enable MLflow
  nself mlflow open                          # Open UI
  nself mlflow experiments                   # List experiments
  nself mlflow experiments create my-model   # Create experiment
  nself mlflow runs 0                        # List runs for experiment 0
  nself mlflow test                          # Test connection
  nself mlflow logs -f                       # Follow logs

${COLOR_YELLOW}MLflow Features:${COLOR_RESET}
  - Experiment tracking
  - Model registry
  - Model deployment
  - Artifact storage (MinIO)
  - Metrics visualization
  - Parameter tracking
  - Model versioning

${COLOR_YELLOW}Python Usage:${COLOR_RESET}
  import mlflow
  mlflow.set_tracking_uri('http://localhost:5000')
  
  with mlflow.start_run():
      mlflow.log_param("param1", value1)
      mlflow.log_metric("metric1", value1)
      mlflow.log_model(model, "model")

${COLOR_YELLOW}Requirements:${COLOR_RESET}
  - PostgreSQL (for backend store)
  - MinIO/S3 (for artifact storage)
  - Python with mlflow package (pip install mlflow)

${COLOR_YELLOW}Default Credentials:${COLOR_RESET}
  Username: admin
  Password: [auto-generated, check .env]
  Port: 5000

EOF
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_mlflow "$@"
fi