#!/usr/bin/env bash

# urls.sh - Display all configured service URLs organized by category
set -euo pipefail

# Get script directory
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$CLI_SCRIPT_DIR/../.." && pwd)"

# Source required utilities
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/header.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/services.sh" 2>/dev/null || true

# Define colors - always set them to avoid unbound variable errors
export COLOR_GREEN="${COLOR_GREEN:-\033[0;32m}"
export COLOR_RED="${COLOR_RED:-\033[0;31m}"
export COLOR_YELLOW="${COLOR_YELLOW:-\033[0;33m}"
export COLOR_BLUE="${COLOR_BLUE:-\033[0;34m}"
export COLOR_CYAN="${COLOR_CYAN:-\033[0;36m}"
export COLOR_GRAY="${COLOR_GRAY:-\033[0;90m}"
export COLOR_RESET="${COLOR_RESET:-\033[0m}"
export BOLD="${BOLD:-\033[1m}"

# Track detected conflicts
declare -a route_conflicts=()

# Main function
main() {
  local show_all=false
  local format="table"
  local check_conflicts=false
  local target_env=""
  local diff_env=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
      --all | -a)
        show_all=true
        shift
        ;;
      --json)
        format="json"
        shift
        ;;
      --env)
        target_env="$2"
        shift 2
        ;;
      --diff)
        diff_env="$2"
        shift 2
        ;;
      --check-conflicts)
        check_conflicts=true
        shift
        ;;
      --help | -h)
        show_help
        exit 0
        ;;
      *)
        echo "Unknown option: $1" >&2
        show_help
        exit 1
        ;;
    esac
  done

  # Handle --env flag: load specific environment
  if [[ -n "$target_env" ]]; then
    case "$target_env" in
      local | dev)
        [[ -f ".env" ]] && source ".env"
        [[ -f ".env.dev" ]] && source ".env.dev"
        ;;
      staging)
        [[ -f ".env.staging" ]] && source ".env.staging"
        # Check for remote config
        if [[ -f ".environments/staging/server.json" ]]; then
          printf "${COLOR_CYAN}→ Remote Environment: staging${COLOR_RESET}\n"
          local host=$(grep '"host"' ".environments/staging/server.json" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
          local domain=$(grep '"domain"' ".environments/staging/server.json" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
          [[ -n "$domain" ]] && BASE_DOMAIN="$domain"
          echo ""
        fi
        ;;
      prod | production)
        [[ -f ".env.prod" ]] && source ".env.prod"
        [[ -f ".env.production" ]] && source ".env.production"
        # Check for remote config
        if [[ -f ".environments/prod/server.json" ]]; then
          printf "${COLOR_CYAN}→ Remote Environment: production${COLOR_RESET}\n"
          local domain=$(grep '"domain"' ".environments/prod/server.json" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
          [[ -n "$domain" ]] && BASE_DOMAIN="$domain"
          echo ""
        fi
        ;;
      *)
        # Try to load from .environments directory
        if [[ -f ".environments/$target_env/server.json" ]]; then
          printf "${COLOR_CYAN}→ Remote Environment: $target_env${COLOR_RESET}\n"
          local domain=$(grep '"domain"' ".environments/$target_env/server.json" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
          [[ -n "$domain" ]] && BASE_DOMAIN="$domain"
          echo ""
        else
          echo "Unknown environment: $target_env" >&2
          exit 1
        fi
        ;;
    esac
  else
    # Load environment normally
    load_env_with_priority
  fi

  # Handle --diff flag: compare URLs between environments
  if [[ -n "$diff_env" ]]; then
    show_url_diff "$target_env" "$diff_env"
    exit 0
  fi

  # Load environment if not already loaded
  [[ -z "$target_env" ]] && load_env_with_priority

  # Get base domain
  local domain="${BASE_DOMAIN:-localhost}"
  local protocol="https"

  # Check if SSL is disabled
  if [[ "${SSL_ENABLED:-true}" == "false" ]]; then
    protocol="http"
  fi

  # Check for route conflicts if requested
  if [[ "$check_conflicts" == "true" ]]; then
    check_route_conflicts
    exit $?
  fi

  if [[ "$format" == "json" ]]; then
    output_json "$protocol" "$domain" "$show_all"
  else
    output_table "$protocol" "$domain" "$show_all"
  fi
}

# Show help
show_help() {
  cat <<EOF
Usage: nself urls [OPTIONS]

Display all configured service URLs organized by category

Options:
  -a, --all             Show all routes including internal services
  --env NAME            Show URLs for specific environment (staging, prod)
  --diff ENV            Compare URLs between current and specified environment
  --json                Output in JSON format
  --check-conflicts     Check for route conflicts (used by build)
  -h, --help           Show this help message

Examples:
  nself urls              # Show all service URLs
  nself urls --json       # Output as JSON
  nself urls --all        # Include internal services
  nself urls --env staging    # Show staging URLs
  nself urls --env prod       # Show production URLs
  nself urls --diff staging   # Compare local vs staging URLs

Categories:
  • Required Services   - Core infrastructure (PostgreSQL, Hasura, Auth, Nginx)
  • Optional Services   - Additional enabled services
  • Custom Services     - Your microservices from templates
  • Frontend Routes     - External frontend applications
  • Plugins             - Third-party integrations (Stripe, GitHub, Shopify)
EOF
}

# Show URL diff between environments
show_url_diff() {
  local env1="${1:-local}"
  local env2="$2"

  if [[ -z "$env2" ]]; then
    echo "Error: Second environment required for diff" >&2
    exit 1
  fi

  printf "${COLOR_CYAN}→ URL Comparison: $env1 ↔ $env2${COLOR_RESET}\n"
  echo ""

  # Get domains
  local domain1="${BASE_DOMAIN:-local.nself.org}"
  local domain2=""

  case "$env2" in
    staging)
      if [[ -f ".environments/staging/server.json" ]]; then
        domain2=$(grep '"domain"' ".environments/staging/server.json" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
      elif [[ -f ".env.staging" ]]; then
        domain2=$(grep "^BASE_DOMAIN=" ".env.staging" 2>/dev/null | cut -d'=' -f2)
      fi
      ;;
    prod | production)
      if [[ -f ".environments/prod/server.json" ]]; then
        domain2=$(grep '"domain"' ".environments/prod/server.json" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
      elif [[ -f ".env.prod" ]]; then
        domain2=$(grep "^BASE_DOMAIN=" ".env.prod" 2>/dev/null | cut -d'=' -f2)
      fi
      ;;
    *)
      if [[ -f ".environments/$env2/server.json" ]]; then
        domain2=$(grep '"domain"' ".environments/$env2/server.json" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
      fi
      ;;
  esac

  domain2="${domain2:-unknown}"

  printf "  %-20s %-35s %-35s\n" "Service" "$env1" "$env2"
  printf "  %-20s %-35s %-35s\n" "-------" "$(printf '%0.s-' {1..30})" "$(printf '%0.s-' {1..30})"

  # Compare key services
  local services=("api" "auth" "admin" "storage" "grafana")
  for svc in "${services[@]}"; do
    local url1="https://${svc}.${domain1}"
    local url2="https://${svc}.${domain2}"
    printf "  %-20s %-35s %-35s\n" "$svc" "$url1" "$url2"
  done

  echo ""
  printf "${COLOR_GRAY}Note: Only showing common services. Use --json for full comparison.${COLOR_RESET}\n"
}

# Check for route conflicts
check_route_conflicts() {
  # Use parallel arrays for bash 3.2 compatibility
  local -a routes=()
  local -a services=()
  local has_conflicts=false

  # Helper function to add route
  add_route() {
    local route="$1"
    local service="$2"
    local i
    for i in "${!routes[@]}"; do
      if [[ "${routes[$i]}" == "$route" ]]; then
        printf "${COLOR_RED}✗ Route conflict detected!${COLOR_RESET}\n" >&2
        printf "  Route '${COLOR_YELLOW}$route${COLOR_RESET}' used by both:\n" >&2
        printf "    - ${services[$i]}\n" >&2
        printf "    - $service\n" >&2
        has_conflicts=true
        return 1
      fi
    done
    routes+=("$route")
    services+=("$service")
    return 0
  }

  # Register required service routes
  if [[ "${HASURA_ENABLED:-true}" == "true" ]]; then
    local route="${HASURA_ROUTE:-api}"
    route="${route%%.*}" # Strip domain
    add_route "$route" "Hasura GraphQL"
  fi

  if [[ "${AUTH_ENABLED:-true}" == "true" ]]; then
    local route="${AUTH_ROUTE:-auth}"
    route="${route%%.*}" # Strip domain
    add_route "$route" "Authentication"
  fi

  # Register optional service routes - strip domain from all
  if [[ "${STORAGE_ENABLED:-false}" == "true" ]]; then
    local route="${STORAGE_ROUTE:-storage}"
    route="${route%%.*}"
    add_route "$route" "Storage API"
  fi

  if [[ "${MINIO_ENABLED:-false}" == "true" ]]; then
    local route="${STORAGE_CONSOLE_ROUTE:-storage-console}"
    route="${route%%.*}"
    add_route "$route" "MinIO Console"
  fi

  if [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]]; then
    local route="${NSELF_ADMIN_ROUTE:-admin}"
    route="${route%%.*}"
    add_route "$route" "nself Admin"
  fi

  if [[ "${MAILPIT_ENABLED:-false}" == "true" ]]; then
    local route="${MAILPIT_ROUTE:-mail}"
    route="${route%%.*}"
    add_route "$route" "MailPit"
  fi

  if [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]]; then
    local route="${MEILISEARCH_ROUTE:-search}"
    route="${route%%.*}"
    add_route "$route" "MeiliSearch"
  fi

  if [[ "${GRAFANA_ENABLED:-false}" == "true" ]]; then
    local route="${GRAFANA_ROUTE:-grafana}"
    route="${route%%.*}"
    add_route "$route" "Grafana"
  fi

  if [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]]; then
    local route="${PROMETHEUS_ROUTE:-prometheus}"
    route="${route%%.*}"
    add_route "$route" "Prometheus"
  fi

  if [[ "${ALERTMANAGER_ENABLED:-false}" == "true" ]]; then
    local route="${ALERTMANAGER_ROUTE:-alertmanager}"
    route="${route%%.*}"
    add_route "$route" "Alertmanager"
  fi

  if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
    local route="${MLFLOW_ROUTE:-mlflow}"
    route="${route%%.*}"
    add_route "$route" "MLflow"
  fi

  if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
    local route="${FUNCTIONS_ROUTE:-functions}"
    route="${route%%.*}"
    add_route "$route" "Functions"
  fi

  if [[ "${BULLMQ_UI_ENABLED:-false}" == "true" ]]; then
    local route="${BULLMQ_UI_ROUTE:-bullmq}"
    route="${route%%.*}"
    add_route "$route" "BullMQ Dashboard"
  fi

  if [[ "${WEBHOOK_SERVICE_ENABLED:-false}" == "true" ]]; then
    local route="${WEBHOOK_SERVICE_ROUTE:-webhooks}"
    route="${route%%.*}"
    add_route "$route" "Webhooks"
  fi

  if [[ "${NESTJS_ENABLED:-false}" == "true" ]]; then
    local route="${NESTJS_ROUTE:-nestjs-api}"
    route="${route%%.*}"
    add_route "$route" "NestJS API"
  fi

  # Check custom services
  for i in {1..10}; do
    local cs_var="CS_${i}"
    local cs_value="${!cs_var:-}"

    if [[ -n "$cs_value" ]]; then
      IFS=':' read -r service_name template port <<<"$cs_value"
      local route_var="CS_${i}_ROUTE"
      local route="${!route_var:-$service_name}"

      # Remove domain if included
      route="${route%%.*}"

      if ! add_route "$route" "Custom service: $service_name (CS_${i})"; then
        route_conflicts+=("CS_${i}:${route}")
      fi
    fi
  done

  # Check frontend routes
  local frontend_count="${FRONTEND_APP_COUNT:-0}"
  for i in $(seq 1 $frontend_count); do
    local route_var="FRONTEND_APP_${i}_ROUTE"
    local route="${!route_var:-}"

    if [[ -n "$route" ]]; then
      route="${route%%.*}"
      if ! add_route "$route" "Frontend App $i"; then
        has_conflicts=true
      fi
    fi
  done

  if [[ "$has_conflicts" == "true" ]]; then
    printf "${COLOR_YELLOW}⚠ Route conflicts must be resolved before building${COLOR_RESET}\n"
    echo "Suggested fixes:"

    for conflict in "${route_conflicts[@]}"; do
      IFS=':' read -r cs_var route <<<"$conflict"
      local service_num="${cs_var#CS_}"
      local new_route=$(suggest_route_fix "$route" "$service_num")
      printf "  In your .env file, add: ${COLOR_GREEN}${cs_var}_ROUTE=${new_route}${COLOR_RESET}\n"
    done

    return 1
  else
    printf "${COLOR_GREEN}✓ No route conflicts detected${COLOR_RESET}\n"
    return 0
  fi
}

# Suggest a fix for route conflict
suggest_route_fix() {
  local conflicting_route="$1"
  local service_num="$2"
  local cs_var="CS_${service_num}"
  local cs_value="${!cs_var:-}"

  if [[ -n "$cs_value" ]]; then
    IFS=':' read -r service_name template port <<<"$cs_value"

    # Suggest route based on service name or template
    if [[ "$conflicting_route" == "api" ]]; then
      case "$template" in
        express*) echo "${service_name//_/-}-api" ;;
        fastapi*) echo "py-api" ;;
        nestjs*) echo "nest-api" ;;
        *) echo "${service_name//_/-}" ;;
      esac
    else
      echo "${service_name//_/-}"
    fi
  else
    echo "service-${service_num}"
  fi
}

# Show plugin webhook URLs
show_plugin_urls() {
  local protocol="$1"
  local domain="$2"
  local plugin_dir="${NSELF_PLUGIN_DIR:-$HOME/.nself/plugins}"
  local plugin_runtime="${NSELF_PLUGIN_RUNTIME:-$HOME/.nself/runtime}"

  printf "%b%b➞ Plugins%b\n" "${BOLD}" "${COLOR_BLUE}" "${COLOR_RESET}"

  # Check if plugin directory exists
  if [[ ! -d "$plugin_dir" ]]; then
    printf "  %bNone installed%b\n" "${COLOR_GRAY}" "${COLOR_RESET}"
    echo
    return
  fi

  # Source plugin runtime functions if available
  if [[ -f "$CLI_SCRIPT_DIR/../lib/plugin/runtime.sh" ]]; then
    source "$CLI_SCRIPT_DIR/../lib/plugin/runtime.sh" 2>/dev/null || true
  fi

  # Find installed plugins
  local has_plugins=false
  for plugin_path in "$plugin_dir"/*/plugin.json; do
    if [[ -f "$plugin_path" ]]; then
      local plugin_name
      plugin_name=$(dirname "$plugin_path")
      plugin_name=$(basename "$plugin_name")

      # Skip shared utilities
      if [[ "$plugin_name" == "_shared" ]]; then
        continue
      fi

      has_plugins=true

      # Get plugin metadata from manifest
      local version="" port="" route=""
      if command -v jq >/dev/null 2>&1; then
        version=$(jq -r '.version // "unknown"' "$plugin_path" 2>/dev/null)
        port=$(jq -r '.port // ""' "$plugin_path" 2>/dev/null)
        route=$(jq -r '.route // ""' "$plugin_path" 2>/dev/null)
      else
        version=$(grep '"version"' "$plugin_path" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
        port=$(grep '"port"' "$plugin_path" 2>/dev/null | head -1 | sed 's/[^0-9]//g')
        route=$(grep '"route"' "$plugin_path" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
      fi

      # Get actual port from .env if plugin is configured
      local env_file="$plugin_dir/$plugin_name/ts/.env"
      if [[ -f "$env_file" ]]; then
        local env_port=$(grep "^PORT=" "$env_file" | cut -d= -f2)
        [[ -n "$env_port" ]] && port="$env_port"
      fi

      # Check if plugin is running
      local status_indicator=""
      if declare -f is_plugin_running >/dev/null 2>&1 && is_plugin_running "$plugin_name" 2>/dev/null; then
        status_indicator="${COLOR_GREEN}●${COLOR_RESET} "
      else
        status_indicator="${COLOR_DIM}○${COLOR_RESET} "
      fi

      # Default route if not specified
      [[ -z "$route" ]] && route="webhooks/$plugin_name"

      local padded_name
      padded_name=$(printf "%-15s" "${plugin_name}:")

      # Show URL based on route configuration
      if [[ -n "$port" ]]; then
        # Plugin has a port - show local URL
        printf "  %s%s %bhttp://localhost:%s%b %b(v%s)%b\n" "$status_indicator" "$padded_name" "${COLOR_GREEN}" "$port" "${COLOR_RESET}" "${COLOR_GRAY}" "$version" "${COLOR_RESET}"
      else
        # No port configured - show as webhook route
        printf "  %s%s %b%s://%s%b %b(v%s)%b\n" "$status_indicator" "$padded_name" "${COLOR_GREEN}" "$protocol" "${route}.${domain}" "${COLOR_RESET}" "${COLOR_GRAY}" "$version" "${COLOR_RESET}"
      fi
    fi
  done

  if [[ "$has_plugins" == "false" ]]; then
    printf "  %bNone installed%b\n" "${COLOR_GRAY}" "${COLOR_RESET}"
  else
    printf "\n  %b●%b = running | %b○%b = stopped | %bManage: nself plugin list --installed --detailed%b\n" \
      "${COLOR_GREEN}" "${COLOR_RESET}" \
      "${COLOR_DIM}" "${COLOR_RESET}" \
      "${COLOR_GRAY}" "${COLOR_RESET}"
  fi
  echo
}

# Output URLs in table format
output_table() {
  local protocol="$1"
  local domain="$2"
  local show_all="$3"

  show_command_header "nself urls" "Service URLs and routes"
  echo

  # Show base application URL (nginx default page)
  printf "  ${BOLD}Base URL:${COLOR_RESET}       ${COLOR_GREEN}${protocol}://${domain}${COLOR_RESET} ${COLOR_GRAY}(nginx default page)${COLOR_RESET}\n"
  echo

  # Required Services
  printf "${BOLD}${COLOR_BLUE}➞ Required Services${COLOR_RESET}\n"

  if [[ "${HASURA_ENABLED:-true}" == "true" ]]; then
    local hasura_route="${HASURA_ROUTE:-api}"
    printf "  GraphQL API:    ${COLOR_GREEN}${protocol}://${hasura_route}.${domain}${COLOR_RESET}\n"
    printf "   - Console:     ${COLOR_GRAY}${protocol}://${hasura_route}.${domain}/console${COLOR_RESET}\n"
  fi

  if [[ "${AUTH_ENABLED:-true}" == "true" ]]; then
    local auth_route="${AUTH_ROUTE:-auth}"
    printf "  Auth:           ${COLOR_GREEN}${protocol}://${auth_route}.${domain}${COLOR_RESET}\n"
  fi

  # Note: PostgreSQL and Nginx don't have public URLs
  [[ "$show_all" == "true" ]] && printf "  PostgreSQL:     ${COLOR_GRAY}Internal only (port 5432)${COLOR_RESET}\n"
  [[ "$show_all" == "true" ]] && printf "  Nginx:          ${COLOR_GRAY}Reverse proxy (ports 80/443)${COLOR_RESET}\n"
  echo

  # Optional Services
  local has_optional=false
  printf "${BOLD}${COLOR_BLUE}➞ Optional Services${COLOR_RESET}\n"

  # Storage
  if [[ "${STORAGE_ENABLED:-false}" == "true" ]]; then
    local storage_route="${STORAGE_ROUTE:-storage}"
    printf "  Storage:        ${COLOR_GREEN}${protocol}://${storage_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  if [[ "${MINIO_ENABLED:-false}" == "true" ]]; then
    local minio_console="${STORAGE_CONSOLE_ROUTE:-storage-console}"
    printf "  MinIO Console:  ${COLOR_GREEN}${protocol}://${minio_console}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  # Mail
  if [[ "${MAILPIT_ENABLED:-false}" == "true" ]]; then
    local mail_route="${MAILPIT_ROUTE:-mail}"
    printf "  Mail UI:        ${COLOR_GREEN}${protocol}://${mail_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  # Search
  if [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]]; then
    local search_route="${MEILISEARCH_ROUTE:-search}"
    printf "  MeiliSearch:    ${COLOR_GREEN}${protocol}://${search_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  # Monitoring
  if [[ "${GRAFANA_ENABLED:-false}" == "true" ]]; then
    local grafana_route="${GRAFANA_ROUTE:-grafana}"
    printf "  Grafana:        ${COLOR_GREEN}${protocol}://${grafana_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  if [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]]; then
    local prom_route="${PROMETHEUS_ROUTE:-prometheus}"
    printf "  Prometheus:     ${COLOR_GREEN}${protocol}://${prom_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  if [[ "${ALERTMANAGER_ENABLED:-false}" == "true" ]]; then
    local alert_route="${ALERTMANAGER_ROUTE:-alertmanager}"
    printf "  Alertmanager:   ${COLOR_GREEN}${protocol}://${alert_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  # Admin
  if [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]]; then
    local admin_route="${NSELF_ADMIN_ROUTE:-admin}"
    printf "  nself Admin:    ${COLOR_GREEN}${protocol}://${admin_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  if [[ "${BULLMQ_UI_ENABLED:-false}" == "true" ]]; then
    local bullmq_route="${BULLMQ_UI_ROUTE:-bullmq}"
    printf "  BullMQ UI:      ${COLOR_GREEN}${protocol}://${bullmq_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  # ML
  if [[ "${MLFLOW_ENABLED:-false}" == "true" ]]; then
    local mlflow_route="${MLFLOW_ROUTE:-mlflow}"
    printf "  MLflow:         ${COLOR_GREEN}${protocol}://${mlflow_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  # Other
  if [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]]; then
    local functions_route="${FUNCTIONS_ROUTE:-functions}"
    printf "  Functions:      ${COLOR_GREEN}${protocol}://${functions_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  if [[ "${WEBHOOK_SERVICE_ENABLED:-false}" == "true" ]]; then
    local webhook_route="${WEBHOOK_SERVICE_ROUTE:-webhooks}"
    printf "  Webhooks:       ${COLOR_GREEN}${protocol}://${webhook_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  if [[ "${NESTJS_ENABLED:-false}" == "true" ]]; then
    local nestjs_route="${NESTJS_ROUTE:-nestjs-api}"
    printf "  NestJS API:     ${COLOR_GREEN}${protocol}://${nestjs_route}.${domain}${COLOR_RESET}\n"
    has_optional=true
  fi

  # Show Redis if enabled (internal)
  if [[ "$show_all" == "true" && "${REDIS_ENABLED:-false}" == "true" ]]; then
    printf "  Redis:          ${COLOR_GRAY}Internal only (port 6379)${COLOR_RESET}\n"
    has_optional=true
  fi

  [[ "$has_optional" == "false" ]] && printf "  ${COLOR_GRAY}None enabled${COLOR_RESET}\n"
  echo

  # Custom Services
  local custom_count=0
  for i in {1..10}; do
    local cs_var="CS_${i}"
    local cs_value="${!cs_var:-}"

    if [[ -n "$cs_value" ]]; then
      if [[ $custom_count -eq 0 ]]; then
        printf "${BOLD}${COLOR_BLUE}➞ Custom Services${COLOR_RESET}\n"
      fi
      custom_count=$((custom_count + 1))

      IFS=':' read -r service_name template port <<<"$cs_value"
      local route_var="CS_${i}_ROUTE"
      local public_var="CS_${i}_PUBLIC"
      local route="${!route_var:-$service_name}"
      local is_public="${!public_var:-true}"

      # Clean route (remove domain if accidentally included)
      route="${route%%.*}"

      if [[ "$is_public" == "true" ]]; then
        # Pad service name for alignment
        local padded_name=$(printf "%-15s" "$service_name:")
        printf "  ${padded_name} ${COLOR_GREEN}${protocol}://${route}.${domain}${COLOR_RESET} ${COLOR_GRAY}(${template})${COLOR_RESET}\n"
      elif [[ "$show_all" == "true" ]]; then
        local padded_name=$(printf "%-15s" "$service_name:")
        printf "  ${padded_name} ${COLOR_GRAY}Internal only - port ${port}${COLOR_RESET}\n"
      fi
    fi
  done
  [[ $custom_count -eq 0 ]] && printf "${BOLD}${COLOR_BLUE}➞ Custom Services${COLOR_RESET}\n" && printf "  ${COLOR_GRAY}None configured${COLOR_RESET}\n"
  echo

  # Frontend Applications - detect dynamically
  printf "${BOLD}${COLOR_BLUE}➞ Frontend Routes${COLOR_RESET}\n"
  local has_frontend=false

  # Check for frontend apps (FRONTEND_APP_1, FRONTEND_APP_2, etc.)
  for i in {1..10}; do
    local route_var="FRONTEND_APP_${i}_ROUTE"
    local name_var="FRONTEND_APP_${i}_NAME"
    local port_var="FRONTEND_APP_${i}_PORT"
    local route="${!route_var:-}"
    local name="${!name_var:-app${i}}"
    local port="${!port_var:-}"

    if [[ -n "$route" ]]; then
      has_frontend=true
      local padded_name=$(printf "%-15s" "${name}:")
      # Handle root route "/" — display as base domain, not /.domain
      if [[ "$route" == "/" ]]; then
        printf "  ${padded_name} ${COLOR_GREEN}${protocol}://${domain}${COLOR_RESET} ${COLOR_GRAY}(external)${COLOR_RESET}\n"
      else
        printf "  ${padded_name} ${COLOR_GREEN}${protocol}://${route}.${domain}${COLOR_RESET} ${COLOR_GRAY}(external)${COLOR_RESET}\n"
      fi
    fi
  done

  if [[ "$has_frontend" == "false" ]]; then
    printf "  ${COLOR_GRAY}None configured${COLOR_RESET}\n"
  fi
  echo

  # Plugins
  show_plugin_urls "$protocol" "$domain"

  # Summary
  printf "${BOLD}${COLOR_GRAY}────────────────────────────────────────${COLOR_RESET}\n"
  local active_count=$(count_active_routes)
  printf "  ${COLOR_GRAY}Total routes: ${active_count} | Domain: ${domain} | Protocol: ${protocol}${COLOR_RESET}\n"

  # SSL/Trust Status
  if [[ "${protocol}" == "https" ]]; then
    # Determine correct SSL certificate path based on domain
    local cert_path=""
    if [[ "${domain}" == "localhost" ]] || [[ "${domain}" == *".localhost" ]]; then
      cert_path="ssl/certificates/localhost"
    elif [[ "${domain}" == *".nself.org" ]] || [[ "${domain}" == "nself.org" ]]; then
      cert_path="ssl/certificates/nself-org"
    else
      cert_path="ssl/certificates/${domain}"
    fi

    # Also check nginx/ssl directory as alternative location
    local nginx_cert_path="nginx/ssl/localhost"
    if [[ "${domain}" == *".nself.org" ]] || [[ "${domain}" == "nself.org" ]]; then
      nginx_cert_path="nginx/ssl/nself-org"
    fi

    if [[ -f "${cert_path}/fullchain.pem" ]]; then
      printf "  %s✓ SSL: Self-signed certificate installed & trusted via /etc/hosts%s\n" "${COLOR_GRAY}" "${COLOR_RESET}"
    elif [[ -f "${nginx_cert_path}/fullchain.pem" ]]; then
      printf "  %s✓ SSL: Self-signed certificate installed & trusted via /etc/hosts%s\n" "${COLOR_GRAY}" "${COLOR_RESET}"
    else
      printf "  %s⚠ SSL: Certificate not found (run 'nself build' to generate)%s\n" "${COLOR_GRAY}" "${COLOR_RESET}"
    fi
  fi

  printf "  ${COLOR_GRAY}Use 'nself urls --all' to see internal services${COLOR_RESET}\n"
  echo
}

# Count active routes
count_active_routes() {
  local count=1 # Application root

  # Required (public only)
  [[ "${HASURA_ENABLED:-true}" == "true" ]] && count=$((count + 1))
  [[ "${AUTH_ENABLED:-true}" == "true" ]] && count=$((count + 1))

  # Optional services
  [[ "${STORAGE_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${MINIO_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${NSELF_ADMIN_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${MAILPIT_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${MEILISEARCH_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${GRAFANA_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${PROMETHEUS_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${ALERTMANAGER_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${MLFLOW_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${FUNCTIONS_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${BULLMQ_UI_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${WEBHOOK_SERVICE_ENABLED:-false}" == "true" ]] && count=$((count + 1))
  [[ "${NESTJS_ENABLED:-false}" == "true" ]] && count=$((count + 1))

  # Custom services with public routes
  for i in {1..10}; do
    local cs_var="CS_${i}"
    local public_var="CS_${i}_PUBLIC"
    local cs_value="${!cs_var:-}"
    local is_public="${!public_var:-true}"

    [[ -n "$cs_value" && "$is_public" == "true" ]] && count=$((count + 1))
  done

  # Frontend apps
  local frontend_count="${FRONTEND_APP_COUNT:-0}"
  count=$((count + frontend_count))

  # Plugins (count webhooks route if any plugins installed)
  local plugin_dir="${NSELF_PLUGIN_DIR:-$HOME/.nself/plugins}"
  if [[ -d "$plugin_dir" ]]; then
    local has_plugins=false
    for plugin_path in "$plugin_dir"/*/plugin.json; do
      if [[ -f "$plugin_path" ]]; then
        local pname
        pname=$(basename "$(dirname "$plugin_path")")
        if [[ "$pname" != "_shared" ]]; then
          has_plugins=true
          break
        fi
      fi
    done
    [[ "$has_plugins" == "true" ]] && count=$((count + 1))
  fi

  echo "$count"
}

# JSON output (simplified for now)
output_json() {
  local protocol="$1"
  local domain="$2"

  echo "{"
  echo "  \"base_domain\": \"$domain\","
  echo "  \"protocol\": \"$protocol\","
  echo "  \"routes\": {"

  # Add routes in JSON format...
  echo "    \"application\": \"${protocol}://${domain}\""

  echo "  }"
  echo "}"
}

# Run main function
main "$@"
