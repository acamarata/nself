#!/usr/bin/env bash
# frontend-apps.sh - Generate frontend application service definitions
# This module handles all frontend app configurations dynamically

# Generate a single frontend app service
generate_frontend_app() {
  local index="$1"
  local app_name="$2"
  local app_port="$3"
  local app_framework="${4:-react}"
  local app_build_path="${5:-.}"
  local app_docker_context="${6:-.}"

  # Check if the frontend directory actually exists
  # Skip generation if it doesn't exist (likely a demo without actual frontend code)
  if [[ ! -d "$app_docker_context" ]] || [[ ! -f "$app_docker_context/Dockerfile" && ! -f "$app_docker_context/Dockerfile.$app_framework" && ! -f "$app_docker_context/package.json" ]]; then
    return 0
  fi

  # Sanitize app name for container naming
  local safe_name=$(echo "$app_name" | tr '[:upper:]' '[:lower:]' | tr -cd '[:alnum:]_-')

  cat <<EOF

  # Frontend Application: ${app_name}
  frontend-${safe_name}:
    container_name: \${PROJECT_NAME}_frontend_${safe_name}
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
EOF

  # Determine if using image or build
  local app_image_var="FRONTEND_APP_${index}_IMAGE"
  local app_image="${!app_image_var:-}"

  if [[ -n "$app_image" ]]; then
    cat <<EOF
    image: ${app_image}
EOF
  else
    # Use build configuration
    local dockerfile_path="Dockerfile"
    if [[ -f "${app_docker_context}/Dockerfile.${app_framework}" ]]; then
      dockerfile_path="Dockerfile.${app_framework}"
    elif [[ -f "${app_docker_context}/Dockerfile.${safe_name}" ]]; then
      dockerfile_path="Dockerfile.${safe_name}"
    fi

    cat <<EOF
    build:
      context: ${app_docker_context}
      dockerfile: ${dockerfile_path}
      args:
        NODE_ENV: \${NODE_ENV:-development}
        API_URL: \${API_URL:-http://localhost:8080/v1/graphql}
EOF
  fi

  # Add dependencies if needed
  if [[ "${HASURA_ENABLED:-false}" == "true" ]]; then
    cat <<EOF
    depends_on:
      hasura:
        condition: service_healthy
EOF
  fi

  # Add environment variables
  cat <<EOF
    environment:
      NODE_ENV: \${NODE_ENV:-development}
      PORT: ${app_port}
      NEXT_PUBLIC_API_URL: \${API_URL:-http://localhost:8080/v1/graphql}
      REACT_APP_API_URL: \${API_URL:-http://localhost:8080/v1/graphql}
      VITE_API_URL: \${API_URL:-http://localhost:8080/v1/graphql}
EOF

  # Add framework-specific environment variables
  case "$app_framework" in
    nextjs|next)
      cat <<EOF
      NEXT_TELEMETRY_DISABLED: 1
      NEXT_PUBLIC_BASE_URL: \${BASE_URL:-http://localhost:3000}
EOF
      ;;
    vue|nuxt)
      cat <<EOF
      NUXT_TELEMETRY_DISABLED: 1
      NUXT_PUBLIC_API_URL: \${API_URL:-http://localhost:8080/v1/graphql}
EOF
      ;;
    angular)
      cat <<EOF
      NG_CLI_ANALYTICS: false
EOF
      ;;
  esac

  # Add ports and volumes
  cat <<EOF
    ports:
      - "${app_port}:${app_port}"
    volumes:
      - ./frontend/${safe_name}:/app
      - /app/node_modules
EOF

  # Add healthcheck
  cat <<EOF
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${app_port}"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
EOF
}

# Generate all frontend apps based on configuration
generate_all_frontend_apps() {
  local app_count="${FRONTEND_APP_COUNT:-0}"

  [[ "$app_count" -le 0 ]] && return 0

  echo ""
  echo "  # ============================================"
  echo "  # Frontend Applications"
  echo "  # ============================================"

  for ((i=1; i<=app_count; i++)); do
    # Get app configuration
    local name_var="FRONTEND_APP_${i}_NAME"
    local port_var="FRONTEND_APP_${i}_PORT"
    local framework_var="FRONTEND_APP_${i}_FRAMEWORK"
    local build_path_var="FRONTEND_APP_${i}_BUILD_PATH"
    local docker_context_var="FRONTEND_APP_${i}_DOCKER_CONTEXT"

    local app_name="${!name_var:-app${i}}"
    local app_port="${!port_var:-$((3000 + i - 1))}"
    local app_framework="${!framework_var:-react}"
    local app_build_path="${!build_path_var:-.}"
    local app_docker_context="${!docker_context_var:-.}"

    generate_frontend_app "$i" "$app_name" "$app_port" "$app_framework" "$app_build_path" "$app_docker_context"
  done
}

# Generate frontend app from legacy FRONTEND_APPS variable
generate_legacy_frontend_apps() {
  local frontend_apps="${FRONTEND_APPS:-}"

  [[ -z "$frontend_apps" ]] && return 0

  echo ""
  echo "  # ============================================"
  echo "  # Frontend Applications (Legacy Format)"
  echo "  # ============================================"

  local index=1
  IFS=',' read -ra APPS <<< "$frontend_apps"

  for app_config in "${APPS[@]}"; do
    IFS=':' read -r name short prefix port framework <<< "$app_config"

    [[ -z "$name" ]] && continue

    # Use defaults if not provided
    port="${port:-$((3000 + index - 1))}"
    framework="${framework:-react}"

    generate_frontend_app "$index" "$name" "$port" "$framework" "." "."
    ((index++))
  done
}

# Main function to generate all frontend apps
generate_frontend_apps() {
  # Try new format first
  if [[ "${FRONTEND_APP_COUNT:-0}" -gt 0 ]]; then
    generate_all_frontend_apps
  # Fall back to legacy format
  elif [[ -n "${FRONTEND_APPS:-}" ]]; then
    generate_legacy_frontend_apps
  fi
}

# Export functions
export -f generate_frontend_app
export -f generate_all_frontend_apps
export -f generate_legacy_frontend_apps
export -f generate_frontend_apps