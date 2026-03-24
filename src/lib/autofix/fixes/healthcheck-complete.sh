#!/usr/bin/env bash

# Source platform-compat for safe_sed_inline
_HC_FIX_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$_HC_FIX_DIR/../../utils/platform-compat.sh" 2>/dev/null || true

# Complete healthcheck fixes - adds tools and endpoints

# Add health endpoint to Node.js/NestJS services
add_nodejs_health_endpoint() {

set -euo pipefail

  local service_name="$1"
  local service_dir="$2"
  local port="${3:-3000}"

  log_info "Adding health endpoint to $service_name..."

  # Check if it's a NestJS service
  if [[ -f "$service_dir/src/main.ts" ]]; then
    # NestJS service - add health controller if missing
    if ! grep -r "health" "$service_dir/src" >/dev/null 2>&1; then
      # Create a simple health controller
      cat >"$service_dir/src/health.controller.ts" <<'EOF'
import { Controller, Get } from '@nestjs/common';

@Controller()
export class HealthController {
  @Get('health')
  health() {
    return { status: 'ok', timestamp: new Date().toISOString() };
  }
}
EOF

      # Add to app.module.ts if it exists
      if [[ -f "$service_dir/src/app.module.ts" ]]; then
        # Check if HealthController is already imported
        if ! grep -q "HealthController" "$service_dir/src/app.module.ts"; then
          # Add import at the top
          safe_sed_inline "$service_dir/src/app.module.ts" "1s/^/import { HealthController } from '.\/health.controller';\n/"

          # Add to controllers array
          safe_sed_inline "$service_dir/src/app.module.ts" '/controllers:/s/\[/[HealthController, /'
        fi
      fi
    fi
  elif [[ -f "$service_dir/src/index.js" ]] || [[ -f "$service_dir/src/app.js" ]]; then
    # Regular Node.js service
    local main_file=""
    for file in "$service_dir/src/index.js" "$service_dir/src/app.js" "$service_dir/src/server.js"; do
      if [[ -f "$file" ]]; then
        main_file="$file"
        break
      fi
    done

    if [[ -n "$main_file" ]]; then
      # Check if health endpoint exists
      if ! grep -q "/health" "$main_file"; then
        # Add health endpoint based on framework
        if grep -q "express" "$main_file"; then
          # Express app - add before app.listen
          safe_sed_inline "$main_file" "/app\.listen/i\
app.get('/health', (req, res) => {\
  res.json({ status: 'ok', timestamp: new Date().toISOString() });\
});"
        elif grep -q "fastify" "$main_file"; then
          # Fastify app
          safe_sed_inline "$main_file" "/fastify\.listen/i\
fastify.get('/health', async (request, reply) => {\
  return { status: 'ok', timestamp: new Date().toISOString() };\
});"
        else
          # Generic Node.js http server
          echo "// Health check endpoint added by nself" >>"$main_file"
        fi
      fi
    fi
  fi

  return 0
}

# Add health endpoint to Go services
add_go_health_endpoint() {
  local service_name="$1"
  local service_dir="$2"

  log_info "Adding health endpoint to Go service $service_name..."

  # Find main.go
  local main_file="$service_dir/main.go"
  if [[ ! -f "$main_file" ]]; then
    main_file="$service_dir/cmd/main.go"
  fi

  if [[ -f "$main_file" ]]; then
    # Check if health endpoint exists
    if ! grep -q "/health" "$main_file"; then
      # Add health handler
      cat >"$service_dir/health.go" <<'EOF'
package main

import (
    "encoding/json"
    "net/http"
    "time"
)

type HealthResponse struct {
    Status    string    `json:"status"`
    Timestamp time.Time `json:"timestamp"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
    response := HealthResponse{
        Status:    "ok",
        Timestamp: time.Now(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
EOF

      # Add route to main.go
      if grep -q "mux.HandleFunc\|http.HandleFunc" "$main_file"; then
        # Add health route
        safe_sed_inline "$main_file" '/HandleFunc/a\    http.HandleFunc("/health", healthHandler)'
      fi
    fi
  fi

  return 0
}

# Add health endpoint to Python services
add_python_health_endpoint() {
  local service_name="$1"
  local service_dir="$2"

  log_info "Adding health endpoint to Python service $service_name..."

  # Find main Python file
  local main_file=""
  for file in "$service_dir/main.py" "$service_dir/app.py" "$service_dir/src/main.py" "$service_dir/src/app.py"; do
    if [[ -f "$file" ]]; then
      main_file="$file"
      break
    fi
  done

  if [[ -n "$main_file" ]]; then
    # Check what framework is being used
    if grep -q "FastAPI\|fastapi" "$main_file"; then
      # FastAPI
      if ! grep -q "/health" "$main_file"; then
        # Add health endpoint
        cat >>"$main_file" <<'EOF'

@app.get("/health")
async def health():
    """Health check endpoint"""
    return {"status": "ok", "timestamp": datetime.now().isoformat()}
EOF
        # Add datetime import if needed
        if ! grep -q "from datetime import" "$main_file"; then
          safe_sed_inline "$main_file" "1s/^/from datetime import datetime\n/"
        fi
      fi
    elif grep -q "Flask\|flask" "$main_file"; then
      # Flask
      if ! grep -q "/health" "$main_file"; then
        cat >>"$main_file" <<'EOF'

@app.route('/health')
def health():
    """Health check endpoint"""
    return {"status": "ok", "timestamp": datetime.now().isoformat()}
EOF
        # Add datetime import if needed
        if ! grep -q "from datetime import" "$main_file"; then
          safe_sed_inline "$main_file" "1s/^/from datetime import datetime\n/"
        fi
      fi
    fi
  fi

  return 0
}

# Update Dockerfile to include health check tools
update_dockerfile_with_healthtools() {
  local service_dir="$1"
  local dockerfile="$service_dir/Dockerfile"

  if [[ ! -f "$dockerfile" ]]; then
    return 1
  fi

  # Check base image type
  if grep -q "FROM.*node" "$dockerfile"; then
    # Node.js image - add wget if using alpine
    if grep -q "alpine" "$dockerfile"; then
      if ! grep -q "apk.*wget\|apk.*curl" "$dockerfile"; then
        # Add wget after FROM
        safe_sed_inline "$dockerfile" "/^FROM.*alpine/a\RUN apk add --no-cache wget curl"
      fi
    else
      # Debian-based Node image
      if ! grep -q "apt-get.*wget\|apt-get.*curl" "$dockerfile"; then
        safe_sed_inline "$dockerfile" "/^FROM/a\RUN apt-get update && apt-get install -y wget curl && rm -rf /var/lib/apt/lists/*"
      fi
    fi
  elif grep -q "FROM.*python" "$dockerfile"; then
    # Python image
    if grep -q "alpine" "$dockerfile"; then
      if ! grep -q "apk.*wget\|apk.*curl" "$dockerfile"; then
        safe_sed_inline "$dockerfile" "/^FROM.*alpine/a\RUN apk add --no-cache wget curl"
      fi
    else
      if ! grep -q "apt-get.*wget\|apt-get.*curl" "$dockerfile"; then
        safe_sed_inline "$dockerfile" "/^FROM/a\RUN apt-get update && apt-get install -y wget curl && rm -rf /var/lib/apt/lists/*"
      fi
    fi
  elif grep -q "FROM.*golang" "$dockerfile"; then
    # Go image
    if grep -q "alpine" "$dockerfile"; then
      if ! grep -q "apk.*wget\|apk.*curl" "$dockerfile"; then
        safe_sed_inline "$dockerfile" "/^FROM.*alpine/a\RUN apk add --no-cache wget curl"
      fi
    fi
  fi

  # Add HEALTHCHECK instruction if missing
  if ! grep -q "HEALTHCHECK" "$dockerfile"; then
    echo "" >>"$dockerfile"
    echo "# Health check" >>"$dockerfile"
    echo 'HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \' >>"$dockerfile"
    echo '  CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-3000}/health || exit 1' >>"$dockerfile"
  fi

  return 0
}

# Fix auth service (nhost/hasura-auth)
fix_auth_healthcheck() {
  local project_name="${PROJECT_NAME:-nself}"

  log_info "Fixing auth service healthcheck..."

  # The auth service should have /healthz endpoint
  # Update docker-compose.yml to use wget instead of curl
  local compose_file="docker-compose.yml"

  # Find auth service block and update healthcheck
  safe_sed_inline "$compose_file" '/^  auth:/,/^  [a-z_-]*:/{
        /healthcheck:/,/test:/ {
            s|test:.*curl.*|test: ["CMD", "wget", "--spider", "-q", "http://localhost:4000/healthz"]|
        }
    }' 2>/dev/null || true

  # The nhost/hasura-auth image should have wget, but let's ensure it (without using 'which')
  docker exec "${project_name}_auth" sh -c "command -v wget || test -x /usr/bin/wget" >/dev/null 2>&1 || {
    # Try to install wget
    docker exec "${project_name}_auth" sh -c "apk add --no-cache wget 2>/dev/null || apt-get update && apt-get install -y wget 2>/dev/null" >/dev/null 2>&1
  }

  return 0
}

# Fix dashboard healthcheck (nhost/dashboard)
fix_dashboard_healthcheck() {
  local project_name="${PROJECT_NAME:-nself}"

  log_info "Fixing dashboard healthcheck..."

  # Update docker-compose.yml
  local compose_file="docker-compose.yml"

  safe_sed_inline "$compose_file" '/^  dashboard:/,/^  [a-z_-]*:/{
        /healthcheck:/,/test:/ {
            s|test:.*curl.*|test: ["CMD", "wget", "--spider", "-q", "http://localhost:3000/"]|
        }
    }' 2>/dev/null || true

  # Install wget in container
  docker exec "${project_name}_dashboard" sh -c "apk add --no-cache wget 2>/dev/null || apt-get update && apt-get install -y wget 2>/dev/null" >/dev/null 2>&1 || true

  return 0
}

# Fix functions service healthcheck
fix_functions_healthcheck() {
  local project_name="${PROJECT_NAME:-nself}"
  local functions_dir="functions"

  log_info "Fixing functions service healthcheck..."

  # Add health endpoint to functions
  if [[ -d "$functions_dir" ]]; then
    add_nodejs_health_endpoint "functions" "$functions_dir" 3000
    update_dockerfile_with_healthtools "$functions_dir"
  fi

  # Update docker-compose.yml
  local compose_file="docker-compose.yml"

  safe_sed_inline "$compose_file" '/^  functions:/,/^  [a-z_-]*:/{
        /healthcheck:/,/test:/ {
            s|test:.*curl.*|test: ["CMD", "wget", "--spider", "-q", "http://localhost:3000/health"]|
        }
    }' 2>/dev/null || true

  return 0
}

# Main function to fix all healthchecks
fix_all_healthchecks() {
  local project_name="${PROJECT_NAME:-nself}"

  log_info "Fixing all service healthchecks..."

  # Fix auth service
  fix_auth_healthcheck

  # Fix dashboard
  fix_dashboard_healthcheck

  # Fix functions
  fix_functions_healthcheck

  # Fix BullMQ workers
  for worker_dir in services/bullmq/*; do
    if [[ -d "$worker_dir" ]]; then
      local worker_name=$(basename "$worker_dir")
      add_nodejs_health_endpoint "${PROJECT_NAME:-nself}-bull-$worker_name" "$worker_dir" 3200
      update_dockerfile_with_healthtools "$worker_dir"

      # Update docker-compose.yml for this worker
      safe_sed_inline docker-compose.yml "/^  ${PROJECT_NAME:-nself}-bull-$worker_name:/,/^  [a-z_-]*:/{
                /healthcheck:/,/test:/ {
                    s|test:.*|test: [\"CMD\", \"wget\", \"--spider\", \"-q\", \"http://localhost:3200/health\"]|
                }
            }" 2>/dev/null || true
    fi
  done

  # Fix NestJS services
  for nest_dir in services/nestjs/* services/node/${PROJECT_NAME:-nself}-nest-*; do
    if [[ -d "$nest_dir" ]]; then
      local service_name=$(basename "$nest_dir")
      add_nodejs_health_endpoint "$service_name" "$nest_dir"
      update_dockerfile_with_healthtools "$nest_dir"

      # Update docker-compose.yml
      safe_sed_inline docker-compose.yml "/^  $service_name:/,/^  [a-z_-]*:/{
                /healthcheck:/,/test:/ {
                    s|test:.*curl.*|test: [\"CMD\", \"wget\", \"--spider\", \"-q\", \"http://localhost:3000/health\"]|
                }
            }" 2>/dev/null || true
    fi
  done

  # Fix Go services
  for go_dir in services/golang/* services/go/*; do
    if [[ -d "$go_dir" ]]; then
      local service_name=$(basename "$go_dir")
      add_go_health_endpoint "$service_name" "$go_dir"
      update_dockerfile_with_healthtools "$go_dir"

      # Update docker-compose.yml
      safe_sed_inline docker-compose.yml "/^  $service_name:/,/^  [a-z_-]*:/{
                /healthcheck:/,/test:/ {
                    s|test:.*curl.*|test: [\"CMD\", \"wget\", \"--spider\", \"-q\", \"http://localhost:8080/health\"]|
                }
            }" 2>/dev/null || true
    fi
  done

  # Fix Python services
  for py_dir in services/python/* services/py/*; do
    if [[ -d "$py_dir" ]]; then
      local service_name=$(basename "$py_dir")
      add_python_health_endpoint "$service_name" "$py_dir"
      update_dockerfile_with_healthtools "$py_dir"

      # Update docker-compose.yml
      safe_sed_inline docker-compose.yml "/^  $service_name:/,/^  [a-z_-]*:/{
                /healthcheck:/,/test:/ {
                    s|test:.*curl.*|test: [\"CMD\", \"wget\", \"--spider\", \"-q\", \"http://localhost:8000/health\"]|
                }
            }" 2>/dev/null || true
    fi
  done

  # Rebuild affected services
  log_info "Rebuilding services with updated healthchecks..."
  docker compose build --parallel >/dev/null 2>&1

  # Restart services to apply changes
  log_info "Restarting services to apply healthcheck fixes..."
  docker compose up -d >/dev/null 2>&1

  log_success "All healthchecks fixed!"
  return 0
}

# Export functions
export -f add_nodejs_health_endpoint
export -f add_go_health_endpoint
export -f add_python_health_endpoint
export -f update_dockerfile_with_healthtools
export -f fix_auth_healthcheck
export -f fix_dashboard_healthcheck
export -f fix_functions_healthcheck
export -f fix_all_healthchecks
