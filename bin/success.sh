#!/bin/bash

# success.sh - Display success message with service URLs

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Load environment
if [ -f "$SCRIPT_DIR/../.env.local" ]; then
  set -o allexport
  source "$SCRIPT_DIR/../.env.local"
  set +o allexport
elif [ -f "$SCRIPT_DIR/../.env" ]; then
  set -o allexport
  source "$SCRIPT_DIR/../.env"
  set +o allexport
fi

# Color codes
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo
echo -e "${GREEN}✨ nself services started successfully!${NC}"
echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo
# Show service URLs
bash "$SCRIPT_DIR/urls.sh"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# Database connection
echo -e "${YELLOW}Database:${NC}"
echo -e "  Host: localhost"
echo -e "  Port: ${POSTGRES_PORT}"
echo -e "  Database: ${POSTGRES_DB}"
echo -e "  User: ${POSTGRES_USER}"
echo

# Redis if enabled
if [[ "$REDIS_ENABLED" == "true" ]]; then
  echo -e "${YELLOW}Redis Cache:${NC}"
  echo -e "  Host: localhost"
  echo -e "  Port: ${REDIS_PORT}"
  echo
fi

# Quick commands
echo -e "${CYAN}📝 Quick Commands:${NC}"
echo -e "  View logs:        docker compose logs -f [service]"
echo -e "  Stop services:    nself down"
echo -e "  Restart all:      nself restart"
echo -e "  View status:      docker compose ps"
echo -e "  Production:       nself prod"
echo

# SSL certificate notice for local development
if [[ "$SSL_MODE" == "local" ]] && [[ "$BASE_DOMAIN" == *"nself.org"* ]]; then
  echo -e "${YELLOW}⚠️  Note:${NC} Using self-signed certificates for *.nself.org"
  echo -e "   You may need to accept the certificate warning in your browser."
  echo
fi

# Check if all services are healthy
echo -e "${CYAN}🔍 Checking service health...${NC}"
sleep 5

# Function to check service health
check_service() {
  local service=$1
  local container="${PROJECT_NAME}_${service}"
  
  if docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "$container.*healthy"; then
    echo -e "  ✅ $service: ${GREEN}Healthy${NC}"
  elif docker ps --format "table {{.Names}}\t{{.Status}}" | grep -q "$container"; then
    echo -e "  ⏳ $service: ${YELLOW}Starting...${NC}"
  else
    echo -e "  ❌ $service: ${RED}Not running${NC}"
  fi
}

# Check each service
check_service "nginx"
check_service "postgres"
check_service "hasura"
check_service "auth"
check_service "minio"
check_service "storage"

if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
  check_service "functions"
fi
if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
  check_service "dashboard"
fi

if [[ "$REDIS_ENABLED" == "true" ]]; then
  check_service "redis"
fi

if [[ "$EMAIL_PROVIDER" == "mailhog" ]]; then
  check_service "mailhog"
fi

if [[ "$NESTJS_ENABLED" == "true" ]]; then
  check_service "microservice"
fi

echo
echo -e "${GREEN}🚀 Your nself backend is ready!${NC}"
echo
