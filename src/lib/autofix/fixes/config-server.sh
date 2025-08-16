#!/bin/bash

fix_config_server_health() {
  # Check if files exist with health endpoint
  if [[ ! -f "config-server/index.js" ]]; then
    mkdir -p config-server
    if [[ -f "/Users/admin/Sites/nself/src/lib/auto-fix/dockerfile-generator.sh" ]]; then
      source "/Users/admin/Sites/nself/src/lib/auto-fix/dockerfile-generator.sh"
      generate_config_server "config-server" >/dev/null 2>&1
      nself build --force >/dev/null 2>&1
    fi
  fi

  # Restart the service with new files
  docker compose stop config-server >/dev/null 2>&1
  docker compose rm -f config-server >/dev/null 2>&1
  docker compose up -d config-server >/dev/null 2>&1
  sleep 3

  # Check if health endpoint works
  local port=$(docker port unity_config-server 4001 2>/dev/null | cut -d: -f2)
  if [[ -z "$port" ]]; then port=4001; fi

  if curl -s -f "http://localhost:$port/healthz" >/dev/null 2>&1; then
    return 0
  fi

  return 1
}
