#!/usr/bin/env bash
set -euo pipefail
# plugin-services.sh - Generate Docker services for installed nself plugins

# Constants
PLUGIN_DIR="${NSELF_PLUGIN_DIR:-$HOME/.nself/plugins}"

# Generate Docker service for a single plugin.
# Supports two install modes:
#   GHCR mode:  plugin was installed via `nself plugin install` (Rust/Docker plugins)
#               and has a .ghcr-image file containing the full image reference.
#               Generates  image: ghcr.io/nself-org/nself-<name>:<version>
#   Build mode: plugin was installed locally from source (has Dockerfile).
#               Generates  build: context: ...
#
# Service name is always nself-<plugin> to match expected container naming.
generate_plugin_service() {
  local plugin_name="$1"
  local plugin_dir="$PLUGIN_DIR/$plugin_name"
  local manifest="$plugin_dir/plugin.json"

  # Skip if plugin directory or manifest doesn't exist
  [[ ! -d "$plugin_dir" ]] && return 0
  [[ ! -f "$manifest" ]] && return 0

  # Determine image source: GHCR (pre-built) or local build
  local ghcr_image_file="$plugin_dir/.ghcr-image"
  local ghcr_image=""
  local dockerfile_path=""

  if [[ -f "$ghcr_image_file" ]]; then
    # GHCR install: read the stored image reference (trim whitespace)
    ghcr_image=$(tr -d '[:space:]' < "$ghcr_image_file" 2>/dev/null || true)
  fi

  if [[ -z "$ghcr_image" ]]; then
    # Fall back to local build mode
    if [[ -f "$plugin_dir/Dockerfile" ]]; then
      dockerfile_path="Dockerfile"
    elif [[ -f "$plugin_dir/docker/Dockerfile" ]]; then
      dockerfile_path="docker/Dockerfile"
    else
      # No Dockerfile and no GHCR image — skip
      return 0
    fi
  fi

  # Parse plugin metadata
  local port
  local replicas
  local memory
  local cpu
  port=$(grep -A5 '"service"' "$manifest" | grep '"port"' | head -1 | sed 's/.*"port"[[:space:]]*:[[:space:]]*\([0-9]*\).*/\1/')
  replicas=$(grep -A5 '"service"' "$manifest" | grep '"replicas"' | head -1 | sed 's/.*"replicas"[[:space:]]*:[[:space:]]*\([0-9]*\).*/\1/')
  memory=$(grep -A5 '"service"' "$manifest" | grep '"memory"' | head -1 | sed 's/.*"memory"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
  cpu=$(grep -A5 '"service"' "$manifest" | grep '"cpu"' | head -1 | sed 's/.*"cpu"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

  # Default values
  [[ -z "$port" ]] && port=3000
  [[ -z "$replicas" ]] && replicas=1

  # Resource limits: per-plugin env vars override global defaults.
  # Per-plugin: PLUGIN_{NAME}_MEMORY_LIMIT / PLUGIN_{NAME}_CPU_LIMIT
  # Global default: PLUGIN_DEFAULT_MEMORY_LIMIT / PLUGIN_DEFAULT_CPU_LIMIT
  # Special default for plugin-ai: 1g / 1.0
  local plugin_name_upper
  plugin_name_upper=$(printf '%s' "$plugin_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
  local per_plugin_memory per_plugin_cpu
  per_plugin_memory=$(eval "printf '%s' \"\${PLUGIN_${plugin_name_upper}_MEMORY_LIMIT:-}\"" 2>/dev/null || true)
  per_plugin_cpu=$(eval "printf '%s' \"\${PLUGIN_${plugin_name_upper}_CPU_LIMIT:-}\"" 2>/dev/null || true)

  # Determine defaults: ai plugin gets larger defaults
  local default_memory default_cpu
  if [[ "$plugin_name" == "ai" ]]; then
    default_memory="${PLUGIN_DEFAULT_MEMORY_LIMIT:-1g}"
    default_cpu="${PLUGIN_DEFAULT_CPU_LIMIT:-1.0}"
  else
    default_memory="${PLUGIN_DEFAULT_MEMORY_LIMIT:-512m}"
    default_cpu="${PLUGIN_DEFAULT_CPU_LIMIT:-0.5}"
  fi

  # Per-plugin env var wins; fall back to manifest value, then default
  if [[ -n "$per_plugin_memory" ]]; then
    memory="$per_plugin_memory"
  elif [[ -z "$memory" ]]; then
    memory="$default_memory"
  fi

  if [[ -n "$per_plugin_cpu" ]]; then
    cpu="$per_plugin_cpu"
  elif [[ -z "$cpu" ]]; then
    cpu="$default_cpu"
  fi

  # Service name: nself-<plugin> for GHCR plugins, plugin-<name> for local builds
  # Tests reference nself-ai, nself-mux etc. — use nself- prefix for all Docker plugins
  local safe_name
  safe_name=$(printf '%s' "$plugin_name" | tr '[:upper:]' '[:lower:]' | tr -cd '[:alnum:]_-')

  # Start service definition
  printf '\n'
  printf '  # Plugin: %s\n' "${plugin_name}"
  printf '  nself-%s:\n' "${safe_name}"

  if [[ -n "$ghcr_image" ]]; then
    # GHCR mode: use pre-built image
    printf '    image: %s\n' "${ghcr_image}"
  else
    # Local build mode
    printf '    build:\n'
    printf '      context: %s\n' "${plugin_dir}"
    printf '      dockerfile: %s\n' "${dockerfile_path}"
  fi

  # Only add container_name if not using replicas
  if [[ "$replicas" == "1" ]]; then
    printf '    container_name: ${PROJECT_NAME}_nself_%s\n' "${safe_name}"
  fi

  cat <<YAML
    restart: unless-stopped
    networks:
      - ${DOCKER_NETWORK}
YAML

  # Add ports if specified — bind to 127.0.0.1 only (nginx proxies externally)
  if [[ -n "$port" && "$port" != "0" ]]; then
    cat <<YAML
    ports:
      - "127.0.0.1:${port}:${port}"
YAML
  fi

  # Add environment variables
  cat <<YAML
    environment:
      - ENV=\${ENV:-dev}
      - NODE_ENV=\${ENV:-dev}
      - APP_ENV=\${ENV:-dev}
      - PROJECT_NAME=\${PROJECT_NAME}
      - BASE_DOMAIN=\${BASE_DOMAIN:-localhost}
      - DOCKER_NETWORK=${DOCKER_NETWORK}
      - SERVICE_NAME=${plugin_name}
      - SERVICE_PORT=${port}
      - PORT=${port}
      - POSTGRES_HOST=postgres
      - POSTGRES_PORT=5432
      - POSTGRES_DB=\${POSTGRES_DB}
      - POSTGRES_USER=\${POSTGRES_USER}
      - POSTGRES_PASSWORD=\${POSTGRES_PASSWORD}
      - DATABASE_URL=postgres://\${POSTGRES_USER}:\${POSTGRES_PASSWORD}@postgres:5432/\${POSTGRES_DB}
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=\${REDIS_PASSWORD:-}
      - HASURA_GRAPHQL_ENDPOINT=http://hasura:8080/v1/graphql
      - HASURA_ADMIN_SECRET=\${HASURA_GRAPHQL_ADMIN_SECRET}
YAML

  # Add plugin-specific environment variables from manifest
  local env_vars
  env_vars=$(grep -A20 '"environment"' "$manifest" | grep -o '"[A-Z_]*"[[:space:]]*:' | tr -d '":' | tr -d ' ')
  if [[ -n "$env_vars" ]]; then
    for var in $env_vars; do
      printf '      - %s=${%s:-}\n' "${var}" "${var}"
    done
  fi

  # Inter-plugin internal URLs — allow plugins to call each other over the
  # Docker network.  Only emitted for plugins that are actually installed
  # (directory exists in PLUGIN_DIR).  Bash 3.2 compatible: parallel arrays.
  local _ipu_names="ai mux notify google cron browser claw claw-web"
  local _ipu_ports="3705 3711 3712 3714 3717 3718 3710 3004"
  local _ipu_idx=0
  for _ipu_name in $_ipu_names; do
    _ipu_idx=$((_ipu_idx + 1))
    # Skip self — a plugin does not need its own URL
    if [[ "$_ipu_name" = "$plugin_name" ]]; then
      continue
    fi
    # Only emit if the peer plugin is installed
    if [[ -d "$PLUGIN_DIR/$_ipu_name" ]]; then
      local _ipu_port
      _ipu_port=$(printf '%s' "$_ipu_ports" | tr ' ' '\n' | awk "NR==$_ipu_idx")
      local _ipu_var
      _ipu_var=$(printf '%s' "$_ipu_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
      printf '      - PLUGIN_%s_INTERNAL_URL=http://nself-%s:%s\n' "$_ipu_var" "$_ipu_name" "$_ipu_port"
    fi
  done

  # Add resource limits
  cat <<YAML
    deploy:
      resources:
        limits:
          memory: ${memory}
          cpus: '${cpu}'
      replicas: ${replicas}
YAML

  # Add dependencies
  cat <<YAML
    depends_on:
      - postgres
YAML

  # Add optional dependencies if enabled
  [[ "${REDIS_ENABLED:-false}" == "true" ]] && printf '      - redis\n'
  [[ "${HASURA_ENABLED:-true}" == "true" ]] && printf '      - hasura\n'

  # Add healthcheck if port is exposed
  if [[ -n "$port" && "$port" != "0" ]]; then
    cat <<YAML
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:${port}/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
YAML
  fi
}

# Generate all plugin services.
# Includes both:
#   - GHCR-installed plugins (have .ghcr-image file, may have no Dockerfile)
#   - Locally-built plugins (have Dockerfile)
generate_all_plugin_services() {
  local plugins_found=false

  # Check if plugin directory exists
  [[ ! -d "$PLUGIN_DIR" ]] && return 0

  # Iterate through installed plugins
  for plugin_path in "$PLUGIN_DIR"/*/; do
    # Skip if not a directory or no plugin.json
    [[ ! -d "$plugin_path" ]] && continue
    [[ ! -f "$plugin_path/plugin.json" ]] && continue

    local plugin_name
    plugin_name=$(basename "$plugin_path")

    # Include plugin if it has a GHCR image reference OR a Dockerfile
    local has_ghcr=false
    local has_dockerfile=false
    [[ -f "$plugin_path/.ghcr-image" ]] && has_ghcr=true
    [[ -f "$plugin_path/Dockerfile" ]] || [[ -f "$plugin_path/docker/Dockerfile" ]] && has_dockerfile=true

    if [[ "$has_ghcr" == "true" ]] || [[ "$has_dockerfile" == "true" ]]; then
      if [[ "$plugins_found" == "false" ]]; then
        printf '\n'
        printf '  # ============================================\n'
        printf '  # Plugin Services (with Docker support)\n'
        printf '  # ============================================\n'
        plugins_found=true
      fi

      generate_plugin_service "$plugin_name"
    fi
  done

  # If plugins were found, add note about non-Dockerized plugins
  if [[ "$plugins_found" == "true" ]]; then
    printf '\n'
    printf '  # Note: Plugins without Dockerfiles run externally (host or remote)\n'
  fi
}

# Export functions
export -f generate_plugin_service
export -f generate_all_plugin_services
