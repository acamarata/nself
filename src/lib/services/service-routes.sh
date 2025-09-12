#!/usr/bin/env bash
# service-routes.sh - Dynamic service discovery and route collection

set -euo pipefail

# Get the directory where this script is located
SERVICE_ROUTES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "$SERVICE_ROUTES_DIR/../.." && pwd)"

# Source utilities
source "$SERVICE_ROUTES_DIR/../utils/display.sh" 2>/dev/null || true

# Collect all service routes from environment
routes::collect_all() {
  local base_domain="${BASE_DOMAIN:-localhost}"
  local project_name="${PROJECT_NAME:-app}"
  local routes=()
  
  # Core domains
  routes+=("$base_domain")
  [[ "$base_domain" != "localhost" ]] && routes+=("localhost")
  routes+=("127.0.0.1" "::1")
  
  # Core nself services with their routes
  if [[ "${HASURA_ENABLED:-false}" == "true" ]]; then
    local hasura_route="${HASURA_ROUTE:-api.${base_domain}}"
    routes+=("$hasura_route")
  fi
  
  if [[ "${AUTH_ENABLED:-false}" == "true" ]]; then
    local auth_route="${AUTH_ROUTE:-auth.${base_domain}}"
    routes+=("$auth_route")
  fi
  
  if [[ "${STORAGE_ENABLED:-false}" == "true" ]]; then
    local storage_route="${STORAGE_ROUTE:-storage.${base_domain}}"
    routes+=("$storage_route")
    
    # Storage console if configured
    local storage_console_route="${STORAGE_CONSOLE_ROUTE:-storage-console.${base_domain}}"
    routes+=("$storage_console_route")
  fi
  
  # Additional services
  if [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]]; then
    local meilisearch_route="${MEILISEARCH_ROUTE:-search.${base_domain}}"
    routes+=("$meilisearch_route")
  fi
  
  if [[ "${MAILPIT_ENABLED:-true}" == "true" ]]; then
    local mailpit_route="${MAILPIT_ROUTE:-mail.${base_domain}}"
    routes+=("$mailpit_route")
  fi
  
  if [[ "${ADMINER_ENABLED:-false}" == "true" ]]; then
    local adminer_route="${ADMINER_ROUTE:-db.${base_domain}}"
    routes+=("$adminer_route")
  fi
  
  if [[ "${BULLMQ_DASHBOARD_ENABLED:-false}" == "true" ]]; then
    local bullmq_route="${BULLMQ_DASHBOARD_ROUTE:-queues.${base_domain}}"
    routes+=("$bullmq_route")
  fi
  
  if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
    local functions_route="${FUNCTIONS_ROUTE:-functions.${base_domain}}"
    routes+=("$functions_route")
  fi
  
  if [[ "${DASHBOARD_ENABLED:-false}" == "true" ]]; then
    local dashboard_route="${DASHBOARD_ROUTE:-dashboard.${base_domain}}"
    routes+=("$dashboard_route")
  fi
  
  if [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]]; then
    local admin_route="${NSELF_ADMIN_ROUTE:-admin.${base_domain}}"
    routes+=("$admin_route")
  fi
  
  if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
    local mlflow_route="${MLFLOW_ROUTE:-mlflow.${base_domain}}"
    routes+=("$mlflow_route")
  fi
  
  # Frontend applications
  local app_count="${FRONTEND_APP_COUNT:-0}"
  if [[ "$app_count" -gt 0 ]]; then
    for ((i=1; i<=app_count; i++)); do
      local route_var="FRONTEND_APP_${i}_ROUTE"
      local route="${!route_var:-}"
      
      if [[ -n "$route" ]]; then
        # Handle both full domain and subdomain formats
        if [[ "$route" == *".${base_domain}" ]]; then
          routes+=("$route")
        else
          routes+=("${route}.${base_domain}")
        fi
      fi
    done
  fi
  
  # Handle direct FRONTEND_APPS if no individual variables
  if [[ "$app_count" -eq 0 && -n "${FRONTEND_APPS:-}" ]]; then
    IFS=',' read -ra APPS <<<"$FRONTEND_APPS"
    for app_config in "${APPS[@]}"; do
      # Parse the format: name:short:prefix:port
      IFS=':' read -r app_name app_short app_prefix app_port <<<"$app_config"
      if [[ -n "$app_short" ]]; then
        routes+=("${app_short}.${base_domain}")
      elif [[ -n "$app_name" ]]; then
        routes+=("${app_name}.${base_domain}")
      fi
    done
  fi
  
  # CS_N services (Custom Services)
  for i in {1..20}; do
    local cs_var="CS_${i}"
    if [[ -n "${!cs_var:-}" ]]; then
      # Parse CS_N format: type:name:port[:route[:internal]]
      IFS=':' read -r cs_type cs_name cs_port cs_route cs_internal <<<"${!cs_var}"
      
      # Only add if it has an external route (not internal-only)
      if [[ -n "$cs_route" && "$cs_internal" != "true" ]]; then
        routes+=("${cs_route}.${base_domain}")
      fi
    fi
  done
  
  # Project main domain
  if [[ -n "$project_name" && "$project_name" != "app" ]]; then
    routes+=("${project_name}.${base_domain}")
  fi
  
  # Remove duplicates and sort
  printf '%s\n' "${routes[@]}" | sort -u
}

# Get service configuration for a specific service
routes::get_service_config() {
  local service_name="$1"
  local base_domain="${BASE_DOMAIN:-localhost}"
  
  case "$service_name" in
    hasura|graphql|api)
      if [[ "${HASURA_ENABLED:-false}" == "true" ]]; then
        echo "service_name=hasura"
        echo "route=${HASURA_ROUTE:-api.${base_domain}}"
        echo "container_name=hasura"
        echo "internal_port=8080"
        echo "needs_websocket=true"
        echo "upstream_name=hasura"
      fi
      ;;
    auth)
      if [[ "${AUTH_ENABLED:-false}" == "true" ]]; then
        echo "service_name=auth"
        echo "route=${AUTH_ROUTE:-auth.${base_domain}}"
        echo "container_name=auth"
        echo "internal_port=4000"
        echo "needs_websocket=false"
        echo "upstream_name=auth"
      fi
      ;;
    storage)
      if [[ "${STORAGE_ENABLED:-false}" == "true" ]]; then
        echo "service_name=storage"
        echo "route=${STORAGE_ROUTE:-storage.${base_domain}}"
        echo "container_name=storage"
        echo "internal_port=5001"
        echo "needs_websocket=false"
        echo "upstream_name=storage"
        echo "max_body_size=100M"
      fi
      ;;
    mailpit|mail)
      if [[ "${MAILPIT_ENABLED:-true}" == "true" ]]; then
        echo "service_name=mailpit"
        echo "route=${MAILPIT_ROUTE:-mail.${base_domain}}"
        echo "container_name=mailpit"
        echo "internal_port=8025"
        echo "needs_websocket=false"
        echo "upstream_name=mailpit"
      fi
      ;;
    meilisearch|search)
      if [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]]; then
        echo "service_name=meilisearch"
        echo "route=${MEILISEARCH_ROUTE:-search.${base_domain}}"
        echo "container_name=meilisearch"
        echo "internal_port=7700"
        echo "needs_websocket=false"
        echo "upstream_name=meilisearch"
      fi
      ;;
    adminer|db)
      if [[ "${ADMINER_ENABLED:-false}" == "true" ]]; then
        echo "service_name=adminer"
        echo "route=${ADMINER_ROUTE:-db.${base_domain}}"
        echo "container_name=adminer"
        echo "internal_port=8080"
        echo "needs_websocket=false"
        echo "upstream_name=adminer"
      fi
      ;;
    bullmq|queues)
      if [[ "${BULLMQ_DASHBOARD_ENABLED:-false}" == "true" ]]; then
        echo "service_name=bullmq"
        echo "route=${BULLMQ_DASHBOARD_ROUTE:-queues.${base_domain}}"
        echo "container_name=bullmq-dashboard"
        echo "internal_port=3000"
        echo "needs_websocket=false"
        echo "upstream_name=bullmq_dashboard"
      fi
      ;;
    functions)
      if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
        echo "service_name=functions"
        echo "route=${FUNCTIONS_ROUTE:-functions.${base_domain}}"
        echo "container_name=functions"
        echo "internal_port=3000"
        echo "needs_websocket=false"
        echo "upstream_name=functions"
      fi
      ;;
    dashboard)
      if [[ "${DASHBOARD_ENABLED:-false}" == "true" ]]; then
        echo "service_name=dashboard"
        echo "route=${DASHBOARD_ROUTE:-dashboard.${base_domain}}"
        echo "container_name=dashboard"
        echo "internal_port=3000"
        echo "needs_websocket=false"
        echo "upstream_name=dashboard"
      fi
      ;;
    nself-admin|admin)
      if [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]]; then
        echo "service_name=nself-admin"
        echo "route=${NSELF_ADMIN_ROUTE:-admin.${base_domain}}"
        echo "container_name=nself-admin"
        echo "internal_port=3100"
        echo "needs_websocket=false"
        echo "upstream_name=nself_admin"
      fi
      ;;
    mlflow)
      if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
        echo "service_name=mlflow"
        echo "route=${MLFLOW_ROUTE:-mlflow.${base_domain}}"
        echo "container_name=mlflow"
        echo "internal_port=5000"
        echo "needs_websocket=false"
        echo "upstream_name=mlflow"
      fi
      ;;
  esac
}

# Get all enabled services
routes::get_enabled_services() {
  local services=()
  
  [[ "${HASURA_ENABLED:-false}" == "true" ]] && services+=(hasura)
  [[ "${AUTH_ENABLED:-false}" == "true" ]] && services+=(auth)
  [[ "${STORAGE_ENABLED:-false}" == "true" ]] && services+=(storage)
  [[ "${MAILPIT_ENABLED:-true}" == "true" ]] && services+=(mailpit)
  [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]] && services+=(meilisearch)
  [[ "${ADMINER_ENABLED:-false}" == "true" ]] && services+=(adminer)
  [[ "${BULLMQ_DASHBOARD_ENABLED:-false}" == "true" ]] && services+=(bullmq)
  [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]] && services+=(functions)
  [[ "${DASHBOARD_ENABLED:-false}" == "true" ]] && services+=(dashboard)
  [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]] && services+=(nself-admin)
  [[ "${MLFLOW_ENABLED:-false}" == "true" ]] && services+=(mlflow)
  
  printf '%s\n' "${services[@]}"
}

# Get frontend applications
routes::get_frontend_apps() {
  local base_domain="${BASE_DOMAIN:-localhost}"
  local apps=()
  
  # Check FRONTEND_APP_N variables
  local app_count="${FRONTEND_APP_COUNT:-0}"
  if [[ "$app_count" -gt 0 ]]; then
    for ((i=1; i<=app_count; i++)); do
      local name_var="FRONTEND_APP_${i}_SYSTEM_NAME"
      local display_var="FRONTEND_APP_${i}_DISPLAY_NAME"
      local route_var="FRONTEND_APP_${i}_ROUTE"
      local port_var="FRONTEND_APP_${i}_PORT"
      
      local app_name="${!name_var:-${!display_var:-}}"
      local route="${!route_var:-}"
      local port="${!port_var:-}"
      
      if [[ -n "$app_name" ]]; then
        # Ensure route has proper domain
        if [[ -z "$route" ]]; then
          route="$app_name"
        fi
        
        if [[ "$route" != *".${base_domain}" ]]; then
          route="${route}.${base_domain}"
        fi
        
        # Auto-detect port if not specified
        if [[ -z "$port" ]]; then
          port=$(detect_app_port 3000)
        fi
        
        echo "app_name=${app_name}"
        echo "route=${route}"
        echo "port=${port}"
        echo "---"
      fi
    done
  fi
  
  # Check direct FRONTEND_APPS variable
  if [[ "$app_count" -eq 0 && -n "${FRONTEND_APPS:-}" ]]; then
    IFS=',' read -ra APPS <<<"$FRONTEND_APPS"
    for app_config in "${APPS[@]}"; do
      IFS=':' read -r app_name app_short app_prefix app_port <<<"$app_config"
      
      if [[ -n "$app_name" ]]; then
        local route="${app_short:-$app_name}.${base_domain}"
        
        echo "app_name=${app_name}"
        echo "route=${route}"
        echo "port=${app_port:-3000}"
        echo "---"
      fi
    done
  fi
}

# Get custom services (CS_N)
routes::get_custom_services() {
  local base_domain="${BASE_DOMAIN:-localhost}"
  
  for i in {1..20}; do
    local cs_var="CS_${i}"
    if [[ -n "${!cs_var:-}" ]]; then
      # Parse CS_N format: type:name:port[:route[:internal]]
      IFS=':' read -r cs_type cs_name cs_port cs_route cs_internal <<<"${!cs_var}"
      
      # Only include services with external routes
      if [[ -n "$cs_route" && "$cs_internal" != "true" ]]; then
        local full_route="${cs_route}.${base_domain}"
        
        echo "service_name=${cs_name}"
        echo "service_type=${cs_type}"
        echo "route=${full_route}"
        echo "internal_port=${cs_port}"
        echo "container_name=${PROJECT_NAME:-app}_${cs_name}"
        echo "---"
      fi
    fi
  done
}

# Generate SSL certificate domains list
routes::get_ssl_domains() {
  local base_domain="${BASE_DOMAIN:-localhost}"
  local domains=()
  
  # Collect all routes and convert to domains
  while IFS= read -r route; do
    [[ -n "$route" ]] && domains+=("$route")
  done < <(routes::collect_all)
  
  # Add wildcard for convenience (but explicit domains are primary)
  if [[ "$base_domain" == "localhost" ]]; then
    domains+=("*.localhost")
  else
    domains+=("*.$base_domain")
  fi
  
  printf '%s\n' "${domains[@]}" | sort -u
}

# Export functions
export -f routes::collect_all
export -f routes::get_service_config
export -f routes::get_enabled_services
export -f routes::get_frontend_apps
export -f routes::get_custom_services
export -f routes::get_ssl_domains