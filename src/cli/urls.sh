#!/usr/bin/env bash
set -euo pipefail

# urls.sh - Display service URLs for nself stack

# Determine root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Source environment utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"

# Show help for urls command
show_urls_help() {
  echo "nself urls - Display service URLs"
  echo ""
  echo "Usage: nself urls [OPTIONS]"
  echo ""
  echo "Description:"
  echo "  Shows all accessible service URLs for your nself stack."
  echo "  Includes GraphQL API, Authentication, Storage, and optional services."
  echo ""
  echo "Options:"
  echo "  --simple            Show URLs only (no credentials)"
  echo "  -h, --help          Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself urls                     # Show all URLs with credentials"
  echo "  nself urls --simple            # Show URLs only"
  echo ""
  echo "Services Shown:"
  echo "  • GraphQL API (Hasura)"
  echo "  • Authentication service"
  echo "  • Storage API and console"
  echo "  • Functions (if enabled)"
  echo "  • Dashboard (if enabled)"
  echo "  • Email service (MailPit/MailHog)"
  echo "  • Custom app routes"
  echo ""
  echo "Notes:"
  echo "  • URLs are determined by your .env.local configuration"
  echo "  • Protocol (http/https) depends on SSL settings"
  echo "  • Credentials shown only in full mode"
}

# Main command function
cmd_urls() {
  local simple_mode=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
    --simple)
      simple_mode=true
      shift
      ;;
    -h | --help)
      show_urls_help
      return 0
      ;;
    *)
      log_error "Unknown option: $1"
      log_info "Use 'nself urls --help' for usage information"
      return 1
      ;;
    esac
  done

  # Show header
  show_command_header "nself urls" "Display service URLs"

  # Load environment safely from current directory
  if [ -f ".env.local" ]; then
    load_env_with_priority
  elif [ -f ".env" ]; then
    load_env_with_priority
  elif [ -f "$ROOT_DIR/.env.local" ]; then
    load_env_with_priority "$ROOT_DIR/.env.local"
  elif [ -f "$ROOT_DIR/.env" ]; then
    load_env_with_priority "$ROOT_DIR/.env"
  fi

  # Expand variables that contain references to other variables
  # Use eval safely to expand ${BASE_DOMAIN} references
  for var in HASURA_ROUTE AUTH_ROUTE STORAGE_ROUTE STORAGE_CONSOLE_ROUTE FUNCTIONS_ROUTE DASHBOARD_ROUTE MAILHOG_ROUTE MAILPIT_ROUTE MAIL_ROUTE; do
    value="${!var:-}"
    if [[ "$value" == *'${BASE_DOMAIN}'* ]]; then
      # Expand the variable reference
      expanded=$(echo "$value" | sed "s/\${BASE_DOMAIN}/$BASE_DOMAIN/g")
      eval "export $var='$expanded'"
    fi
  done

  # Set default routes if not defined
  HASURA_ROUTE="${HASURA_ROUTE:-api.${BASE_DOMAIN}}"
  AUTH_ROUTE="${AUTH_ROUTE:-auth.${BASE_DOMAIN}}"
  STORAGE_ROUTE="${STORAGE_ROUTE:-storage.${BASE_DOMAIN}}"
  STORAGE_CONSOLE_ROUTE="${STORAGE_CONSOLE_ROUTE:-storage-console.${BASE_DOMAIN}}"
  FUNCTIONS_ROUTE="${FUNCTIONS_ROUTE:-functions.${BASE_DOMAIN}}"
  DASHBOARD_ROUTE="${DASHBOARD_ROUTE:-dashboard.${BASE_DOMAIN}}"
  MAILHOG_ROUTE="${MAILHOG_ROUTE:-mailhog.${BASE_DOMAIN}}"
  MAILPIT_ROUTE="${MAILPIT_ROUTE:-mail.${BASE_DOMAIN}}"
  MAIL_ROUTE="${MAIL_ROUTE:-mail.${BASE_DOMAIN}}"

  # Determine protocol
  if [[ "${SSL_MODE:-}" == "local" ]] || [[ "${SSL_MODE:-}" == "letsencrypt" ]] || [[ -n "${SSL_CERT_PATH:-}" ]]; then
    PROTOCOL="https"
  else
    PROTOCOL="http"
  fi

  # Display URLs based on mode
  if [[ "$simple_mode" == "true" ]]; then
    # Simple mode - just URLs, no secrets
    log_info "Service URLs:"
    echo ""
    echo "  GraphQL API:     ${PROTOCOL}://${HASURA_ROUTE}"
    echo "  Authentication:  ${PROTOCOL}://${AUTH_ROUTE}"
    echo "  Storage:         ${PROTOCOL}://${STORAGE_ROUTE}"
    echo "  Storage Console: ${PROTOCOL}://${STORAGE_CONSOLE_ROUTE}"

    if [[ "${FUNCTIONS_ENABLED:-}" == "true" ]]; then
      echo "  Functions:       ${PROTOCOL}://${FUNCTIONS_ROUTE}"
    fi

    if [[ "${DASHBOARD_ENABLED:-}" == "true" ]]; then
      echo "  Dashboard:       ${PROTOCOL}://${DASHBOARD_ROUTE}"
    fi

    if [[ "${EMAIL_PROVIDER:-}" == "mailhog" ]]; then
      echo "  MailHog:         ${PROTOCOL}://${MAILHOG_ROUTE}"
    elif [[ "${EMAIL_PROVIDER:-}" == "mailpit" ]]; then
      echo "  MailPit:         ${PROTOCOL}://${MAIL_ROUTE}"
    fi

    # Show optional custom app routes
    for i in {1..10}; do
      var_name="APP_${i}_ROUTE"
      if [[ -n "${!var_name}" ]]; then
        # Extract subdomain from the route (format: port:subdomain.domain)
        route="${!var_name#*:}"
        echo "  App ${i}:           ${PROTOCOL}://${route}"
      fi
    done
  else
    # Full mode - with secrets (original behavior)
    log_info "Service URLs with credentials:"
    echo

    echo "  GraphQL API: ${PROTOCOL}://${HASURA_ROUTE}"
    echo "      Admin Secret: ${HASURA_GRAPHQL_ADMIN_SECRET}"
    echo

    echo "  Authentication: ${PROTOCOL}://${AUTH_ROUTE}"
    echo

    echo "  Storage API: ${PROTOCOL}://${STORAGE_ROUTE}"
    echo "      Console: ${PROTOCOL}://${STORAGE_CONSOLE_ROUTE}"
    echo "      Access Key: ${MINIO_ROOT_USER}"
    echo

    if [[ "${FUNCTIONS_ENABLED:-}" == "true" ]]; then
      echo "  Functions: ${PROTOCOL}://${FUNCTIONS_ROUTE}"
      echo
    fi

    if [[ "${DASHBOARD_ENABLED:-}" == "true" ]]; then
      echo "  Dashboard: ${PROTOCOL}://${DASHBOARD_ROUTE}"
      echo
    fi

    if [[ "${EMAIL_PROVIDER:-}" == "mailhog" ]]; then
      echo "  MailHog: ${PROTOCOL}://${MAILHOG_ROUTE}"
      echo
    elif [[ "${EMAIL_PROVIDER:-}" == "mailpit" ]]; then
      echo "  MailPit: ${PROTOCOL}://${MAIL_ROUTE}"
      echo
    fi
  fi
}

# Execute the command
cmd_urls "$@"
