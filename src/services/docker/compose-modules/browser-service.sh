#!/usr/bin/env bash
set -euo pipefail
# browser-service.sh — generate np_browser service block for docker-compose.yml
#
# Included when PLUGIN_BROWSER_ENABLED=true in the project .env.
# The np_browser container runs the nself-browser Playwright plugin:
#   - Node.js Playwright worker on internal port 3717 (browser pool)
#   - Rust Axum HTTP server on port 3716 (proxies requests to worker)
#
# Chromium requires shared memory (shm_size: 256m).
# Binds to 127.0.0.1 only — nginx routes externally.
#
# Usage: sourced by compose-generate.sh alongside other compose-modules.
# Called via generate_browser_service (included in generate_all_plugin_services
# when PLUGIN_BROWSER_ENABLED=true).

# Generate the np_browser service YAML block.
generate_browser_service() {
  if [ "${PLUGIN_BROWSER_ENABLED:-false}" != "true" ]; then
    return 0
  fi

  printf '\n'
  printf '  # nself-browser — Headless Playwright/Chromium browser pool\n'
  printf '  # Requires PLUGIN_BROWSER_ENABLED=true\n'
  printf '  np_browser:\n'
  printf '    image: nself/nself-browser:${NSELF_BROWSER_VERSION:-latest}\n'
  printf '    container_name: ${PROJECT_NAME}_np_browser\n'

  cat <<'YAML'
    restart: unless-stopped
    shm_size: '256mb'
YAML

  cat <<YAML
    networks:
      - ${DOCKER_NETWORK}
    depends_on:
      - postgres
YAML

  cat <<'YAML'
    environment:
      - DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
      - PLUGIN_BROWSER_PORT=${PLUGIN_BROWSER_PORT:-3716}
      - PLUGIN_BROWSER_WORKER_PORT=${PLUGIN_BROWSER_WORKER_PORT:-3717}
      - PLUGIN_BROWSER_POOL_SIZE=${PLUGIN_BROWSER_POOL_SIZE:-3}
      - PLUGIN_BROWSER_TIMEOUT_MS=${PLUGIN_BROWSER_TIMEOUT_MS:-30000}
      - PLUGIN_BROWSER_STEALTH=${PLUGIN_BROWSER_STEALTH:-false}
      - PLUGIN_BROWSER_DEFAULT_POLICY=${PLUGIN_BROWSER_DEFAULT_POLICY:-allow_all}
      - PLUGIN_BROWSER_ALLOWLIST=${PLUGIN_BROWSER_ALLOWLIST:-}
      - PLUGIN_BROWSER_DENYLIST=${PLUGIN_BROWSER_DENYLIST:-}
    ports:
      # SECURITY: Bind to localhost only - access via nginx reverse proxy
      - "127.0.0.1:${PLUGIN_BROWSER_PORT:-3716}:3716"
    volumes:
      - np_browser_sessions:/app/sessions
      - np_browser_screenshots:/app/screenshots
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:3716/health"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 60s
YAML
}

# Output volume declarations for the browser service.
# Called from compose-generate.sh when building the volumes: section.
get_browser_volumes() {
  if [ "${PLUGIN_BROWSER_ENABLED:-false}" != "true" ]; then
    return 0
  fi
  printf '  np_browser_sessions:\n'
  printf '  np_browser_screenshots:\n'
}

# Export functions for sourcing by compose-generate.sh
export -f generate_browser_service
export -f get_browser_volumes
