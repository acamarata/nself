#!/bin/bash
# urls.sh - Display service URLs for nself stack

# Determine root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Load environment if not already loaded
if [ -z "$HASURA_ROUTE" ]; then
  if [ -f "$ROOT_DIR/.env.local" ]; then
    set -o allexport
    source "$ROOT_DIR/.env.local"
    set +o allexport
  elif [ -f "$ROOT_DIR/.env" ]; then
    set -o allexport
    source "$ROOT_DIR/.env"
    set +o allexport
  fi
fi

# Colors
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Determine protocol
if [[ "$SSL_MODE" == "local" ]] || [[ "$SSL_MODE" == "letsencrypt" ]] || [[ -n "$SSL_CERT_PATH" ]]; then
  PROTOCOL="https"
else
  PROTOCOL="http"
fi

echo -e "${BLUE}🌐 Service URLs:${NC}"
echo

echo -e "  ${YELLOW}GraphQL API:${NC} ${PROTOCOL}://${HASURA_ROUTE}"
echo -e "      Admin Secret: ${HASURA_GRAPHQL_ADMIN_SECRET}"
echo

echo -e "  ${YELLOW}Authentication:${NC} ${PROTOCOL}://${AUTH_ROUTE}"
echo

echo -e "  ${YELLOW}Storage API:${NC} ${PROTOCOL}://${STORAGE_ROUTE}"
echo -e "      Console: ${PROTOCOL}://${STORAGE_CONSOLE_ROUTE}"
echo -e "      Access Key: ${MINIO_ROOT_USER}"
echo

if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
  echo -e "  ${YELLOW}Functions:${NC} ${PROTOCOL}://${FUNCTIONS_ROUTE}"
  echo
fi

if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
  echo -e "  ${YELLOW}Dashboard:${NC} ${PROTOCOL}://${DASHBOARD_ROUTE}"
  echo
fi

if [[ "$EMAIL_PROVIDER" == "mailhog" ]]; then
  echo -e "  ${YELLOW}MailHog:${NC} ${PROTOCOL}://${MAILHOG_ROUTE}"
  echo
fi
