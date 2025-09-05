#!/usr/bin/env bash

# functions.sh - Serverless functions management

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"

# Functions command
cmd_functions() {
  local subcommand="${1:-status}"
  shift || true

  case "$subcommand" in
    status)
      functions_status
      ;;
    enable)
      functions_enable
      ;;
    disable)
      functions_disable
      ;;
    list)
      functions_list
      ;;
    create)
      functions_create "$@"
      ;;
    delete)
      functions_delete "$@"
      ;;
    test)
      functions_test "$@"
      ;;
    logs)
      functions_logs "$@"
      ;;
    deploy)
      functions_deploy "$@"
      ;;
    help|--help|-h)
      functions_help
      ;;
    *)
      log_error "Unknown functions command: $subcommand"
      functions_help
      exit 1
      ;;
  esac
}

# Show functions status
functions_status() {
  show_command_header "Functions" "Checking serverless functions status"
  
  load_env_with_priority
  ensure_project_context
  
  if [[ "${FUNCTIONS_ENABLED:-false}" != "true" ]]; then
    log_info "Functions service is disabled"
    log_info "Enable with: nself functions enable"
    return 0
  fi
  
  if is_service_running "functions"; then
    log_success "Functions service is running"
    
    # Check health
    local health_url="http://localhost:${FUNCTIONS_PORT:-4300}/health"
    if curl -s "$health_url" >/dev/null 2>&1; then
      log_success "Functions service is healthy"
      
      # List available functions
      local functions=$(curl -s "http://localhost:${FUNCTIONS_PORT:-4300}/functions" | jq -r '.functions[]' 2>/dev/null || echo "none")
      if [[ -n "$functions" ]] && [[ "$functions" != "none" ]]; then
        log_info "Available functions:"
        echo "$functions" | while read -r func; do
          echo "  - $func"
        done
      else
        log_info "No functions deployed"
      fi
    else
      log_warning "Functions service is not responding"
    fi
  else
    log_warning "Functions service is not running"
    log_info "Start with: nself start"
  fi
}

# Enable functions
functions_enable() {
  show_command_header "Functions" "Enabling serverless functions"
  
  load_env_with_priority
  ensure_project_context
  
  # Update .env file
  if grep -q "^FUNCTIONS_ENABLED=" .env 2>/dev/null; then
    sed -i.bak 's/^FUNCTIONS_ENABLED=.*/FUNCTIONS_ENABLED=true/' .env
  else
    echo "FUNCTIONS_ENABLED=true" >> .env
  fi
  
  log_success "Functions enabled"
  log_info "Rebuild and restart to apply changes:"
  log_info "  nself build"
  log_info "  nself restart"
}

# Disable functions
functions_disable() {
  show_command_header "Functions" "Disabling serverless functions"
  
  load_env_with_priority
  ensure_project_context
  
  # Update .env file
  if grep -q "^FUNCTIONS_ENABLED=" .env 2>/dev/null; then
    sed -i.bak 's/^FUNCTIONS_ENABLED=.*/FUNCTIONS_ENABLED=false/' .env
  else
    echo "FUNCTIONS_ENABLED=false" >> .env
  fi
  
  log_success "Functions disabled"
  log_info "Rebuild and restart to apply changes:"
  log_info "  nself build"
  log_info "  nself restart"
}

# List functions
functions_list() {
  show_command_header "Functions" "Listing available functions"
  
  load_env_with_priority
  ensure_project_context
  
  if [[ "${FUNCTIONS_ENABLED:-false}" != "true" ]]; then
    log_error "Functions service is disabled"
    exit 1
  fi
  
  # Check local functions directory
  if [[ -d "./functions" ]]; then
    log_info "Local functions:"
    find ./functions -name "*.js" -type f | while read -r file; do
      local name=$(basename "$file" .js)
      echo "  - $name"
    done
  fi
  
  # Check deployed functions
  if is_service_running "functions"; then
    log_info ""
    log_info "Deployed functions:"
    curl -s "http://localhost:${FUNCTIONS_PORT:-4300}/functions" | jq -r '.functions[]' 2>/dev/null | while read -r func; do
      echo "  - $func (http://localhost:${FUNCTIONS_PORT:-4300}/function/$func)"
    done
  fi
}

# Create a new function
functions_create() {
  local name="${1:-}"
  local template="${2:-basic}"
  
  if [[ -z "$name" ]]; then
    log_error "Function name required"
    log_info "Usage: nself functions create <name> [template]"
    log_info "Templates: basic, webhook, api, scheduled"
    exit 1
  fi
  
  show_command_header "Functions" "Creating function: $name"
  
  load_env_with_priority
  ensure_project_context
  
  # Create functions directory if it doesn't exist
  mkdir -p ./functions
  
  local function_file="./functions/${name}.js"
  
  if [[ -f "$function_file" ]]; then
    log_error "Function already exists: $name"
    exit 1
  fi
  
  # Create function from template
  case "$template" in
    basic)
      cat >"$function_file" <<'EOF'
// Basic serverless function
async function handler(event, context) {
  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'application/json'
    },
    body: {
      message: 'Function executed successfully',
      timestamp: new Date().toISOString()
    }
  };
}
EOF
      ;;
      
    webhook)
      cat >"$function_file" <<'EOF'
// Webhook handler function
async function handler(event, context) {
  console.log('Webhook received:', event.body);
  
  const { action, data } = event.body || {};
  
  // Process webhook based on action
  switch (action) {
    case 'create':
      console.log('Processing create action:', data);
      break;
    case 'update':
      console.log('Processing update action:', data);
      break;
    case 'delete':
      console.log('Processing delete action:', data);
      break;
    default:
      console.log('Unknown action:', action);
  }
  
  return {
    statusCode: 200,
    body: { 
      received: true,
      action: action,
      processed: new Date().toISOString()
    }
  };
}
EOF
      ;;
      
    api)
      cat >"$function_file" <<'EOF'
// API endpoint function
async function handler(event, context) {
  const { method, path, query, headers, body } = event;
  
  // Handle different HTTP methods
  switch (method) {
    case 'GET':
      return handleGet(query);
    case 'POST':
      return handlePost(body);
    case 'PUT':
      return handlePut(body);
    case 'DELETE':
      return handleDelete(query);
    default:
      return {
        statusCode: 405,
        body: { error: 'Method not allowed' }
      };
  }
}

function handleGet(query) {
  const id = query.get('id');
  return {
    statusCode: 200,
    body: {
      method: 'GET',
      id: id,
      data: { /* your data */ }
    }
  };
}

function handlePost(body) {
  return {
    statusCode: 201,
    body: {
      method: 'POST',
      created: body,
      id: Date.now()
    }
  };
}

function handlePut(body) {
  return {
    statusCode: 200,
    body: {
      method: 'PUT',
      updated: body
    }
  };
}

function handleDelete(query) {
  const id = query.get('id');
  return {
    statusCode: 200,
    body: {
      method: 'DELETE',
      deleted: id
    }
  };
}
EOF
      ;;
      
    scheduled)
      cat >"$function_file" <<'EOF'
// Scheduled task function
async function handler(event, context) {
  console.log('Scheduled function executed at:', new Date().toISOString());
  
  try {
    // Perform scheduled task
    await performScheduledTask();
    
    return {
      statusCode: 200,
      body: {
        success: true,
        executedAt: new Date().toISOString()
      }
    };
  } catch (error) {
    console.error('Scheduled task failed:', error);
    return {
      statusCode: 500,
      body: {
        error: error.message
      }
    };
  }
}

async function performScheduledTask() {
  // Your scheduled task logic here
  console.log('Performing scheduled maintenance...');
  
  // Example: Clean up old data
  // Example: Send daily reports
  // Example: Sync with external services
  
  return true;
}
EOF
      ;;
      
    *)
      log_error "Unknown template: $template"
      log_info "Available templates: basic, webhook, api, scheduled"
      rm -f "$function_file"
      exit 1
      ;;
  esac
  
  log_success "Function created: $name"
  log_info "Function file: $function_file"
  log_info "Test with: nself functions test $name"
  
  # Reload if functions service is running
  if is_service_running "functions"; then
    log_info "Function will be automatically loaded"
  fi
}

# Delete a function
functions_delete() {
  local name="${1:-}"
  
  if [[ -z "$name" ]]; then
    log_error "Function name required"
    log_info "Usage: nself functions delete <name>"
    exit 1
  fi
  
  show_command_header "Functions" "Deleting function: $name"
  
  load_env_with_priority
  ensure_project_context
  
  local function_file="./functions/${name}.js"
  
  if [[ ! -f "$function_file" ]]; then
    log_error "Function not found: $name"
    exit 1
  fi
  
  rm -f "$function_file"
  log_success "Function deleted: $name"
}

# Test a function
functions_test() {
  local name="${1:-}"
  local data="${2:-'{}'}"
  
  if [[ -z "$name" ]]; then
    log_error "Function name required"
    log_info "Usage: nself functions test <name> [json-data]"
    exit 1
  fi
  
  show_command_header "Functions" "Testing function: $name"
  
  load_env_with_priority
  ensure_project_context
  
  if [[ "${FUNCTIONS_ENABLED:-false}" != "true" ]]; then
    log_error "Functions service is disabled"
    exit 1
  fi
  
  if ! is_service_running "functions"; then
    log_error "Functions service is not running"
    log_info "Start with: nself start"
    exit 1
  fi
  
  local url="http://localhost:${FUNCTIONS_PORT:-4300}/function/$name"
  
  log_info "Testing function at: $url"
  log_info "Request data: $data"
  
  local response
  if [[ "$data" == "{}" ]]; then
    response=$(curl -s "$url")
  else
    response=$(curl -s -X POST "$url" \
      -H "Content-Type: application/json" \
      -d "$data")
  fi
  
  if [[ -n "$response" ]]; then
    log_success "Function executed successfully"
    echo "$response" | jq . 2>/dev/null || echo "$response"
  else
    log_error "Function failed to execute"
    exit 1
  fi
}

# View function logs
functions_logs() {
  local follow="${1:-}"
  
  show_command_header "Functions" "Viewing functions logs"
  
  load_env_with_priority
  ensure_project_context
  
  if [[ "$follow" == "-f" ]] || [[ "$follow" == "--follow" ]]; then
    compose logs -f functions
  else
    compose logs --tail=50 functions
  fi
}

# Deploy functions (placeholder for future implementation)
functions_deploy() {
  show_command_header "Functions" "Deploying functions"
  
  load_env_with_priority
  ensure_project_context
  
  log_info "Deploying functions to production..."
  log_warning "Production deployment not yet implemented"
  log_info "Functions are currently deployed locally with 'nself build'"
}

# Show help
functions_help() {
  cat <<EOF
${COLOR_BLUE}nself functions${COLOR_RESET} - Manage serverless functions

${COLOR_YELLOW}Usage:${COLOR_RESET}
  nself functions <command> [options]

${COLOR_YELLOW}Commands:${COLOR_RESET}
  status              Show functions service status
  enable              Enable functions service
  disable             Disable functions service
  list                List available functions
  create <name> [tpl] Create a new function (templates: basic, webhook, api, scheduled)
  delete <name>       Delete a function
  test <name> [data]  Test a function with optional JSON data
  logs [-f]           View function logs (use -f to follow)
  deploy              Deploy functions to production
  help                Show this help message

${COLOR_YELLOW}Examples:${COLOR_RESET}
  nself functions enable                    # Enable functions
  nself functions create hello basic        # Create basic function
  nself functions create webhook webhook    # Create webhook handler
  nself functions test hello                # Test function
  nself functions test webhook '{"action":"test"}'
  nself functions logs -f                   # Follow logs

${COLOR_YELLOW}Function Structure:${COLOR_RESET}
  Functions are JavaScript files in ./functions/ directory
  Each function must export a handler function:

  async function handler(event, context) {
    return {
      statusCode: 200,
      headers: { 'Content-Type': 'application/json' },
      body: { message: 'Hello World' }
    };
  }

${COLOR_YELLOW}Access Functions:${COLOR_RESET}
  http://localhost:4300/function/{name}

${COLOR_YELLOW}Configuration:${COLOR_RESET}
  FUNCTIONS_ENABLED=true
  FUNCTIONS_PORT=4300

EOF
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_functions "$@"
fi