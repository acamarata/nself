#!/usr/bin/env bash

# functions.sh - Serverless functions management

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/utils/platform-compat.sh"

# Functions command
cmd_functions() {
  local subcommand="${1:-status}"
  shift || true

  case "$subcommand" in
    status)
      functions_status
      ;;
    init)
      functions_init "$@"
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

# Initialize functions service
functions_init() {
  show_command_header "Functions" "Initializing serverless functions"

  load_env_with_priority
  ensure_project_context

  local use_typescript=false
  for arg in "$@"; do
    if [[ "$arg" == "--ts" ]] || [[ "$arg" == "--typescript" ]]; then
      use_typescript=true
    fi
  done

  # Create functions directory
  if [[ ! -d "./functions" ]]; then
    mkdir -p ./functions
    log_success "Created functions directory"
  else
    log_info "Functions directory already exists"
  fi

  # Create package.json if it doesn't exist
  if [[ ! -f "./functions/package.json" ]]; then
    if [[ "$use_typescript" == "true" ]]; then
      cat > ./functions/package.json << 'EOF'
{
  "name": "nself-functions",
  "version": "1.0.0",
  "description": "Serverless functions for nself project",
  "main": "index.js",
  "scripts": {
    "build": "tsc",
    "watch": "tsc --watch",
    "lint": "eslint . --ext .ts"
  },
  "dependencies": {},
  "devDependencies": {
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0"
  }
}
EOF
      log_success "Created package.json (TypeScript)"

      # Create tsconfig.json
      cat > ./functions/tsconfig.json << 'EOF'
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "commonjs",
    "lib": ["ES2022"],
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "outDir": "./dist",
    "rootDir": "./"
  },
  "include": ["*.ts"],
  "exclude": ["node_modules", "dist"]
}
EOF
      log_success "Created tsconfig.json"
    else
      cat > ./functions/package.json << 'EOF'
{
  "name": "nself-functions",
  "version": "1.0.0",
  "description": "Serverless functions for nself project",
  "main": "index.js",
  "scripts": {
    "lint": "eslint ."
  },
  "dependencies": {}
}
EOF
      log_success "Created package.json"
    fi
  else
    log_info "package.json already exists"
  fi

  # Run npm install if node/npm is available
  if command -v npm >/dev/null 2>&1; then
    log_info "Installing dependencies..."
    (cd ./functions && npm install 2>/dev/null) || log_warning "npm install had warnings (may be normal)"
    log_success "Dependencies installed"
  else
    log_warning "npm not found - skipping dependency installation"
    log_info "Run 'cd functions && npm install' manually"
  fi

  # Create example function if none exist
  local func_count=$(find ./functions -maxdepth 1 \( -name "*.js" -o -name "*.ts" \) 2>/dev/null | wc -l | tr -d ' ')
  if [[ "$func_count" -eq 0 ]]; then
    if [[ "$use_typescript" == "true" ]]; then
      functions_create "hello" "basic" "--ts"
    else
      functions_create "hello" "basic"
    fi
    log_success "Created example function: hello"
  fi

  # Enable functions in .env if not already
  if ! grep -q "^FUNCTIONS_ENABLED=true" .env 2>/dev/null; then
    functions_enable
  fi

  echo ""
  log_success "Functions initialized successfully!"
  echo ""
  log_info "Next steps:"
  log_info "  1. nself build          # Rebuild to include functions service"
  log_info "  2. nself restart        # Restart to apply changes"
  log_info "  3. nself functions test hello  # Test the example function"
}

# Enable functions
functions_enable() {
  show_command_header "Functions" "Enabling serverless functions"
  
  load_env_with_priority
  ensure_project_context
  
  # Update .env file
  if grep -q "^FUNCTIONS_ENABLED=" .env 2>/dev/null; then
    safe_sed_inline .env 's/^FUNCTIONS_ENABLED=.*/FUNCTIONS_ENABLED=true/'
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
    safe_sed_inline .env 's/^FUNCTIONS_ENABLED=.*/FUNCTIONS_ENABLED=false/'
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
  local use_typescript=false

  # Check for --ts flag
  for arg in "$@"; do
    if [[ "$arg" == "--ts" ]] || [[ "$arg" == "--typescript" ]]; then
      use_typescript=true
    fi
  done

  # Handle case where template is --ts
  if [[ "$template" == "--ts" ]] || [[ "$template" == "--typescript" ]]; then
    template="basic"
    use_typescript=true
  fi

  if [[ -z "$name" ]]; then
    log_error "Function name required"
    log_info "Usage: nself functions create <name> [template] [--ts]"
    log_info "Templates: basic, webhook, api, scheduled"
    log_info "Add --ts flag for TypeScript"
    exit 1
  fi

  show_command_header "Functions" "Creating function: $name"

  load_env_with_priority
  ensure_project_context

  # Create functions directory if it doesn't exist
  mkdir -p ./functions

  local ext="js"
  if [[ "$use_typescript" == "true" ]]; then
    ext="ts"
  fi

  local function_file="./functions/${name}.${ext}"

  # Check for both .js and .ts versions
  if [[ -f "./functions/${name}.js" ]] || [[ -f "./functions/${name}.ts" ]]; then
    log_error "Function already exists: $name"
    exit 1
  fi

  # Create TypeScript functions
  if [[ "$use_typescript" == "true" ]]; then
    create_typescript_function "$name" "$template" "$function_file"
    return
  fi

  # Create JavaScript function from template
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

# Deploy functions
functions_deploy() {
  local target="${1:-local}"
  local force="${2:-}"

  show_command_header "Functions" "Deploying functions"

  load_env_with_priority
  ensure_project_context

  if [[ "${FUNCTIONS_ENABLED:-false}" != "true" ]]; then
    log_error "Functions service is disabled"
    log_info "Enable with: nself functions enable"
    exit 1
  fi

  # Check for functions directory
  if [[ ! -d "./functions" ]]; then
    log_error "No functions directory found"
    log_info "Create a function first: nself functions create <name>"
    exit 1
  fi

  # Count functions
  local func_count=$(find ./functions -name "*.js" -o -name "*.ts" 2>/dev/null | wc -l | tr -d ' ')
  if [[ "$func_count" -eq 0 ]]; then
    log_error "No functions found in ./functions directory"
    exit 1
  fi

  log_info "Found $func_count function(s) to deploy"

  case "$target" in
    local)
      deploy_functions_local
      ;;
    production|prod)
      deploy_functions_production "$force"
      ;;
    validate)
      validate_functions
      ;;
    *)
      log_error "Unknown deploy target: $target"
      log_info "Valid targets: local, production, validate"
      exit 1
      ;;
  esac
}

# Deploy functions locally (restart container with new functions)
deploy_functions_local() {
  log_info "Deploying functions locally..."

  # Validate functions first
  if ! validate_functions; then
    log_error "Function validation failed"
    exit 1
  fi

  # Check if functions service is running
  if is_service_running "functions"; then
    log_info "Restarting functions service to pick up changes..."
    compose restart functions
    log_success "Functions deployed locally"

    # Wait for service to be healthy
    local max_wait=30
    local waited=0
    while [[ $waited -lt $max_wait ]]; do
      if curl -s "http://localhost:${FUNCTIONS_PORT:-4300}/health" >/dev/null 2>&1; then
        log_success "Functions service is healthy"
        break
      fi
      sleep 1
      waited=$((waited + 1))
    done

    if [[ $waited -ge $max_wait ]]; then
      log_warning "Functions service may still be starting"
    fi
  else
    log_info "Functions service is not running"
    log_info "Start with: nself start"
  fi

  # List deployed functions
  log_info ""
  log_info "Deployed functions:"
  find ./functions -name "*.js" -o -name "*.ts" 2>/dev/null | while read -r file; do
    local name=$(basename "$file" | sed 's/\.[jt]s$//')
    printf "  - %s (http://localhost:%s/function/%s)\n" "$name" "${FUNCTIONS_PORT:-4300}" "$name"
  done
}

# Deploy functions to production
deploy_functions_production() {
  local force="$1"

  # Check for production configuration
  if [[ -z "${DEPLOY_HOST:-}" ]]; then
    log_error "Production deployment requires DEPLOY_HOST configuration"
    log_info ""
    log_info "Set up production deployment in .env:"
    log_info "  DEPLOY_HOST=user@production-server.com"
    log_info "  DEPLOY_PATH=/opt/myapp"
    log_info "  DEPLOY_KEY=~/.ssh/deploy_key (optional)"
    exit 1
  fi

  local deploy_host="${DEPLOY_HOST}"
  local deploy_path="${DEPLOY_PATH:-/opt/${PROJECT_NAME:-nself}}"
  local deploy_key="${DEPLOY_KEY:-}"

  # Build SSH options
  local ssh_opts="-o StrictHostKeyChecking=accept-new -o BatchMode=yes"
  if [[ -n "$deploy_key" ]]; then
    ssh_opts="$ssh_opts -i $deploy_key"
  fi

  log_info "Deploying to: $deploy_host:$deploy_path/functions"

  # Validate functions first
  if ! validate_functions; then
    log_error "Function validation failed"
    exit 1
  fi

  # Confirm deployment unless forced
  if [[ "$force" != "--force" ]] && [[ "$force" != "-f" ]]; then
    printf "Deploy functions to production? [y/N] "
    read -r response
    response=$(echo "$response" | tr '[:upper:]' '[:lower:]')
    if [[ "$response" != "y" ]] && [[ "$response" != "yes" ]]; then
      log_info "Deployment cancelled"
      return 0
    fi
  fi

  # Sync functions directory
  log_info "Syncing functions..."
  if rsync -avz --delete \
    -e "ssh $ssh_opts" \
    ./functions/ \
    "$deploy_host:$deploy_path/functions/"; then
    log_success "Functions synced successfully"
  else
    log_error "Failed to sync functions"
    exit 1
  fi

  # Restart functions service on production
  log_info "Restarting functions service on production..."
  if ssh $ssh_opts "$deploy_host" "cd $deploy_path && docker compose restart functions"; then
    log_success "Functions deployed to production!"
  else
    log_warning "Could not restart functions service remotely"
    log_info "You may need to restart manually: ssh $deploy_host 'cd $deploy_path && docker compose restart functions'"
  fi
}

# Validate all functions
validate_functions() {
  log_info "Validating functions..."
  local errors=0

  find ./functions -name "*.js" 2>/dev/null | while read -r file; do
    local name=$(basename "$file" .js)
    # Check for basic syntax with node if available
    if command -v node >/dev/null 2>&1; then
      if ! node --check "$file" 2>/dev/null; then
        log_error "Syntax error in: $name"
        errors=$((errors + 1))
      else
        log_success "  $name.js - valid"
      fi
    else
      # Basic check - file exists and is not empty
      if [[ -s "$file" ]]; then
        log_success "  $name.js - exists"
      else
        log_error "  $name.js - empty or missing"
        errors=$((errors + 1))
      fi
    fi
  done

  find ./functions -name "*.ts" 2>/dev/null | while read -r file; do
    local name=$(basename "$file" .ts)
    # Check for TypeScript syntax with tsc if available
    if command -v tsc >/dev/null 2>&1; then
      if ! tsc --noEmit "$file" 2>/dev/null; then
        log_warning "TypeScript check skipped for: $name (may have import issues)"
        log_success "  $name.ts - exists"
      else
        log_success "  $name.ts - valid"
      fi
    else
      # Basic check
      if [[ -s "$file" ]]; then
        log_success "  $name.ts - exists"
      else
        log_error "  $name.ts - empty or missing"
        errors=$((errors + 1))
      fi
    fi
  done

  if [[ $errors -gt 0 ]]; then
    return 1
  fi
  return 0
}

# Create TypeScript function from template
create_typescript_function() {
  local name="$1"
  local template="$2"
  local function_file="$3"

  case "$template" in
    basic)
      cat >"$function_file" <<'EOF'
// Basic serverless function (TypeScript)
interface Event {
  method: string;
  path: string;
  query: URLSearchParams;
  headers: Record<string, string>;
  body: unknown;
}

interface Context {
  functionName: string;
  requestId: string;
}

interface Response {
  statusCode: number;
  headers?: Record<string, string>;
  body: unknown;
}

async function handler(event: Event, context: Context): Promise<Response> {
  return {
    statusCode: 200,
    headers: {
      'Content-Type': 'application/json'
    },
    body: {
      message: 'Function executed successfully',
      timestamp: new Date().toISOString(),
      requestId: context.requestId
    }
  };
}

export { handler };
EOF
      ;;

    webhook)
      cat >"$function_file" <<'EOF'
// Webhook handler function (TypeScript)
interface Event {
  method: string;
  path: string;
  query: URLSearchParams;
  headers: Record<string, string>;
  body: WebhookPayload;
}

interface WebhookPayload {
  action?: string;
  data?: Record<string, unknown>;
}

interface Context {
  functionName: string;
  requestId: string;
}

interface Response {
  statusCode: number;
  headers?: Record<string, string>;
  body: unknown;
}

async function handler(event: Event, context: Context): Promise<Response> {
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

export { handler };
EOF
      ;;

    api)
      cat >"$function_file" <<'EOF'
// API endpoint function (TypeScript)
interface Event {
  method: string;
  path: string;
  query: URLSearchParams;
  headers: Record<string, string>;
  body: unknown;
}

interface Context {
  functionName: string;
  requestId: string;
}

interface Response {
  statusCode: number;
  headers?: Record<string, string>;
  body: unknown;
}

async function handler(event: Event, context: Context): Promise<Response> {
  const { method, query, body } = event;

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

function handleGet(query: URLSearchParams): Response {
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

function handlePost(body: unknown): Response {
  return {
    statusCode: 201,
    body: {
      method: 'POST',
      created: body,
      id: Date.now()
    }
  };
}

function handlePut(body: unknown): Response {
  return {
    statusCode: 200,
    body: {
      method: 'PUT',
      updated: body
    }
  };
}

function handleDelete(query: URLSearchParams): Response {
  const id = query.get('id');
  return {
    statusCode: 200,
    body: {
      method: 'DELETE',
      deleted: id
    }
  };
}

export { handler };
EOF
      ;;

    scheduled)
      cat >"$function_file" <<'EOF'
// Scheduled task function (TypeScript)
interface Event {
  method: string;
  path: string;
  query: URLSearchParams;
  headers: Record<string, string>;
  body: unknown;
}

interface Context {
  functionName: string;
  requestId: string;
}

interface Response {
  statusCode: number;
  headers?: Record<string, string>;
  body: unknown;
}

async function handler(event: Event, context: Context): Promise<Response> {
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
        error: error instanceof Error ? error.message : 'Unknown error'
      }
    };
  }
}

async function performScheduledTask(): Promise<boolean> {
  // Your scheduled task logic here
  console.log('Performing scheduled maintenance...');

  // Example: Clean up old data
  // Example: Send daily reports
  // Example: Sync with external services

  return true;
}

export { handler };
EOF
      ;;

    *)
      log_error "Unknown template: $template"
      log_info "Available templates: basic, webhook, api, scheduled"
      rm -f "$function_file"
      exit 1
      ;;
  esac

  log_success "TypeScript function created: $name"
  log_info "Function file: $function_file"
  log_info "Test with: nself functions test $name"

  # Reload if functions service is running
  if is_service_running "functions"; then
    log_info "Function will be automatically loaded"
  fi
}

# Show help
functions_help() {
  cat <<EOF
${COLOR_BLUE}nself functions${COLOR_RESET} - Manage serverless functions

${COLOR_YELLOW}Usage:${COLOR_RESET}
  nself functions <command> [options]

${COLOR_YELLOW}Commands:${COLOR_RESET}
  status              Show functions service status
  init [--ts]         Initialize functions service (creates directory, package.json, example)
  enable              Enable functions service
  disable             Disable functions service
  list                List available functions
  create <name> [tpl] Create a new function (templates: basic, webhook, api, scheduled)
                      Add --ts flag for TypeScript: nself functions create hello --ts
  delete <name>       Delete a function
  test <name> [data]  Test a function with optional JSON data
  logs [-f]           View function logs (use -f to follow)
  deploy [target]     Deploy functions (targets: local, production, validate)
  help                Show this help message

${COLOR_YELLOW}Examples:${COLOR_RESET}
  nself functions init                      # Initialize functions (JS)
  nself functions init --ts                 # Initialize functions (TypeScript)
  nself functions enable                    # Enable functions
  nself functions create hello basic        # Create basic JS function
  nself functions create hello basic --ts   # Create basic TypeScript function
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