#!/usr/bin/env bash
# custom-services.sh - Generate custom service definitions
# This module handles dynamic custom service generation

# Generate a single custom service
generate_custom_service() {
  local index="$1"
  local service_name="$2"
  local service_image="$3"
  local service_port="$4"
  local service_env="${5:-}"
  local service_volumes="${6:-}"
  local service_command="${7:-}"

  # Sanitize service name
  local safe_name=$(echo "$service_name" | tr '[:upper:]' '[:lower:]' | tr -cd '[:alnum:]_-')

  cat <<EOF

  # Custom Service: ${service_name}
  custom-${safe_name}:
    container_name: \${PROJECT_NAME}_custom_${safe_name}
    image: ${service_image}
    restart: unless-stopped
    networks:
      - \${DOCKER_NETWORK}
EOF

  # Add command if specified
  if [[ -n "$service_command" ]]; then
    cat <<EOF
    command: ${service_command}
EOF
  fi

  # Add environment variables if specified
  if [[ -n "$service_env" ]]; then
    cat <<EOF
    environment:
EOF
    # Parse environment variables (format: KEY=value;KEY2=value2)
    IFS=';' read -ra ENV_VARS <<< "$service_env"
    for env_var in "${ENV_VARS[@]}"; do
      [[ -n "$env_var" ]] && echo "      $env_var"
    done
  fi

  # Add volumes if specified
  if [[ -n "$service_volumes" ]]; then
    cat <<EOF
    volumes:
EOF
    # Parse volumes (format: host:container;host2:container2)
    IFS=';' read -ra VOLUMES <<< "$service_volumes"
    for volume in "${VOLUMES[@]}"; do
      [[ -n "$volume" ]] && echo "      - $volume"
    done
  fi

  # Add ports if specified
  if [[ -n "$service_port" && "$service_port" != "0" ]]; then
    cat <<EOF
    ports:
      - "${service_port}:${service_port}"
EOF
  fi

  # Add basic healthcheck if port is exposed
  if [[ -n "$service_port" && "$service_port" != "0" ]]; then
    cat <<EOF
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:${service_port}/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 30s
      disable: \${HEALTHCHECK_DISABLED:-false}
EOF
  fi
}

# Generate all custom services based on configuration
generate_all_custom_services() {
  local service_count="${CUSTOM_SERVICE_COUNT:-0}"

  [[ "$service_count" -le 0 ]] && return 0

  echo ""
  echo "  # ============================================"
  echo "  # Custom Services"
  echo "  # ============================================"

  for ((i=1; i<=service_count; i++)); do
    # Get service configuration
    local name_var="CUSTOM_SERVICE_${i}_NAME"
    local image_var="CUSTOM_SERVICE_${i}_IMAGE"
    local port_var="CUSTOM_SERVICE_${i}_PORT"
    local env_var="CUSTOM_SERVICE_${i}_ENV"
    local volumes_var="CUSTOM_SERVICE_${i}_VOLUMES"
    local command_var="CUSTOM_SERVICE_${i}_COMMAND"
    local depends_var="CUSTOM_SERVICE_${i}_DEPENDS_ON"

    local service_name="${!name_var:-custom${i}}"
    local service_image="${!image_var:-alpine:latest}"
    local service_port="${!port_var:-}"
    local service_env="${!env_var:-}"
    local service_volumes="${!volumes_var:-}"
    local service_command="${!command_var:-}"

    # Skip if no image specified
    [[ -z "$service_image" ]] && continue

    generate_custom_service "$i" "$service_name" "$service_image" \
      "$service_port" "$service_env" "$service_volumes" "$service_command"
  done
}

# Generate custom services from legacy CS_ variables
generate_legacy_custom_services() {
  echo ""
  echo "  # ============================================"
  echo "  # Custom Services (Legacy Format)"
  echo "  # ============================================"

  for i in {1..20}; do
    local cs_var="CS_${i}"
    local cs_value="${!cs_var:-}"

    [[ -z "$cs_value" ]] && continue

    # Parse CS_ format: type:name:port:route:internal:image:command
    IFS=':' read -r cs_type cs_name cs_port cs_route cs_internal cs_image cs_command <<< "$cs_value"

    # Skip if essential fields are missing
    [[ -z "$cs_name" || -z "$cs_image" ]] && continue

    # Skip internal-only services
    [[ "$cs_internal" == "true" ]] && continue

    generate_custom_service "$i" "$cs_name" "$cs_image" "$cs_port" "" "" "$cs_command"
  done
}

# Main function to generate all custom services
generate_custom_services() {
  # Try new format first
  if [[ "${CUSTOM_SERVICE_COUNT:-0}" -gt 0 ]]; then
    generate_all_custom_services
  fi

  # Also check for legacy CS_ variables
  local has_legacy=false
  for i in {1..20}; do
    local cs_var="CS_${i}"
    [[ -n "${!cs_var:-}" ]] && has_legacy=true && break
  done

  [[ "$has_legacy" == "true" ]] && generate_legacy_custom_services
}

# Export functions
export -f generate_custom_service
export -f generate_all_custom_services
export -f generate_legacy_custom_services
export -f generate_custom_services