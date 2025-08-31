#!/usr/bin/env bash
set -euo pipefail

# scaffold.sh - Create new service from template

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Command function
cmd_scaffold() {
  local service_type="${1:-}"
  local service_name="${2:-}"
  local start_service=false

  # Parse options
  shift 2 2>/dev/null || true
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --start)
      start_service=true
      shift
      ;;
    --help | -h)
      show_scaffold_help
      return 0
      ;;
    *)
      log_error "Unknown option: $1"
      show_scaffold_help
      return 1
      ;;
    esac
  done

  # Show command header
  show_command_header "nself scaffold" "Create new service from template"

  # Validate inputs
  if [[ -z "$service_type" ]] || [[ -z "$service_name" ]]; then
    log_error "Service type and name are required"
    show_scaffold_help
    return 1
  fi

  # Validate service type
  case "$service_type" in
  nest | nestjs)
    service_type="nest"
    template_dir="$SCRIPT_DIR/../templates/services/nest"
    ;;
  bull | bullmq)
    service_type="bullmq"
    template_dir="$SCRIPT_DIR/../templates/services/bullmq"
    ;;
  go | golang)
    service_type="go"
    template_dir="$SCRIPT_DIR/../templates/services/go"
    ;;
  py | python)
    service_type="py"
    template_dir="$SCRIPT_DIR/../templates/services/py"
    ;;
  *)
    log_error "Invalid service type: $service_type"
    log_info "Valid types: nest, bull, go, py"
    return 1
    ;;
  esac

  # Check if template exists
  if [[ ! -d "$template_dir" ]]; then
    log_error "Template not found: $template_dir"
    return 1
  fi

  # Determine target directory
  local target_dir="services/$service_type/$service_name"

  # Check if service already exists
  if [[ -d "$target_dir" ]]; then
    log_error "Service already exists: $target_dir"
    return 1
  fi

  # Create service directory
  log_info "Creating service: $service_name (type: $service_type)"
  mkdir -p "$target_dir"

  # Copy template files
  log_info "Copying template files..."
  cp -r "$template_dir"/* "$target_dir/" 2>/dev/null || {
    log_error "Failed to copy template files"
    rm -rf "$target_dir"
    return 1
  }

  # Replace placeholders in template files
  log_info "Customizing service configuration..."
  find "$target_dir" -type f -name "*.template" | while read -r template_file; do
    output_file="${template_file%.template}"
    sed -e "s/{{SERVICE_NAME}}/$service_name/g" \
      -e "s/{{PROJECT_NAME}}/${PROJECT_NAME:-nself}/g" \
      -e "s/{{BASE_DOMAIN}}/${BASE_DOMAIN:-localhost}/g" \
      "$template_file" >"$output_file"
    rm "$template_file"
  done

  # Update package.json or go.mod with service name
  case "$service_type" in
  nest | bullmq)
    if [[ -f "$target_dir/package.json" ]]; then
      sed -i '' "s/\"name\": \".*\"/\"name\": \"$service_name\"/" "$target_dir/package.json"
    fi
    ;;
  go)
    if [[ -f "$target_dir/go.mod" ]]; then
      sed -i '' "s|module .*|module $service_name|" "$target_dir/go.mod"
    fi
    ;;
  py)
    if [[ -f "$target_dir/requirements.txt" ]]; then
      # Python requirements don't need name changes
      true
    fi
    ;;
  esac

  # Add service to environment configuration
  log_info "Registering service in environment..."
  case "$service_type" in
  nest)
    # Add to NESTJS_SERVICES
    if [[ -f ".env.local" ]]; then
      if grep -q "^NESTJS_SERVICES=" ".env.local"; then
        current_services=$(grep "^NESTJS_SERVICES=" ".env.local" | cut -d= -f2)
        if [[ -n "$current_services" ]]; then
          sed -i '' "s/^NESTJS_SERVICES=.*/NESTJS_SERVICES=$current_services,$service_name/" ".env.local"
        else
          sed -i '' "s/^NESTJS_SERVICES=.*/NESTJS_SERVICES=$service_name/" ".env.local"
        fi
      else
        echo "NESTJS_SERVICES=$service_name" >>".env.local"
      fi
      sed -i '' "s/^NESTJS_ENABLED=.*/NESTJS_ENABLED=true/" ".env.local"
    fi
    ;;
  bullmq)
    # Add to BULLMQ_WORKERS
    if [[ -f ".env.local" ]]; then
      if grep -q "^BULLMQ_WORKERS=" ".env.local"; then
        current_workers=$(grep "^BULLMQ_WORKERS=" ".env.local" | cut -d= -f2)
        if [[ -n "$current_workers" ]]; then
          sed -i '' "s/^BULLMQ_WORKERS=.*/BULLMQ_WORKERS=$current_workers,$service_name/" ".env.local"
        else
          sed -i '' "s/^BULLMQ_WORKERS=.*/BULLMQ_WORKERS=$service_name/" ".env.local"
        fi
      else
        echo "BULLMQ_WORKERS=$service_name" >>".env.local"
      fi
      sed -i '' "s/^BULLMQ_ENABLED=.*/BULLMQ_ENABLED=true/" ".env.local"
    fi
    ;;
  go)
    # Add to GOLANG_SERVICES
    if [[ -f ".env.local" ]]; then
      if grep -q "^GOLANG_SERVICES=" ".env.local"; then
        current_services=$(grep "^GOLANG_SERVICES=" ".env.local" | cut -d= -f2)
        if [[ -n "$current_services" ]]; then
          sed -i '' "s/^GOLANG_SERVICES=.*/GOLANG_SERVICES=$current_services,$service_name/" ".env.local"
        else
          sed -i '' "s/^GOLANG_SERVICES=.*/GOLANG_SERVICES=$service_name/" ".env.local"
        fi
      else
        echo "GOLANG_SERVICES=$service_name" >>".env.local"
      fi
      sed -i '' "s/^GOLANG_ENABLED=.*/GOLANG_ENABLED=true/" ".env.local"
    fi
    ;;
  py)
    # Add to PYTHON_SERVICES
    if [[ -f ".env.local" ]]; then
      if grep -q "^PYTHON_SERVICES=" ".env.local"; then
        current_services=$(grep "^PYTHON_SERVICES=" ".env.local" | cut -d= -f2)
        if [[ -n "$current_services" ]]; then
          sed -i '' "s/^PYTHON_SERVICES=.*/PYTHON_SERVICES=$current_services,$service_name/" ".env.local"
        else
          sed -i '' "s/^PYTHON_SERVICES=.*/PYTHON_SERVICES=$service_name/" ".env.local"
        fi
      else
        echo "PYTHON_SERVICES=$service_name" >>".env.local"
      fi
      sed -i '' "s/^PYTHON_ENABLED=.*/PYTHON_ENABLED=true/" ".env.local"
    fi
    ;;
  esac

  log_success "Service created successfully: $target_dir"

  # Show next steps
  echo
  show_section "Next Steps"
  echo "1. Review the generated service in: $target_dir"
  echo "2. Customize the service configuration as needed"
  echo "3. Run: nself build    # Regenerate docker-compose.yml"
  echo "4. Run: nself start       # Start all services"

  # Start service if requested
  if [[ "$start_service" == "true" ]]; then
    echo
    log_info "Rebuilding and starting services..."
    "$SCRIPT_DIR/build.sh" || return 1
    "$SCRIPT_DIR/start.sh" || return 1
  fi
}

# Show help
show_scaffold_help() {
  echo "Usage: nself scaffold <type> <name> [options]"
  echo
  echo "Create a new service from template"
  echo
  echo "Service Types:"
  echo "  nest, nestjs    Create a NestJS REST API service"
  echo "  bull, bullmq    Create a BullMQ worker service"
  echo "  go, golang      Create a Go service"
  echo "  py, python      Create a Python service"
  echo
  echo "Options:"
  echo "  --start         Build and start services after creation"
  echo "  --help, -h      Show this help message"
  echo
  echo "Examples:"
  echo "  nself scaffold nest api-gateway"
  echo "  nself scaffold bull email-worker --start"
  echo "  nself scaffold go websocket-server"
  echo "  nself scaffold py ml-service"
}

# Export for use as library
export -f cmd_scaffold

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "scaffold" || exit $?
  cmd_scaffold "$@"
  exit_code=$?
  post_command "scaffold" $exit_code
  exit $exit_code
fi
