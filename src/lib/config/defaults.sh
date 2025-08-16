#!/usr/bin/env bash
# defaults.sh - Default configuration values

# Project defaults
export PROJECT_NAME="${PROJECT_NAME:-nself}"
export BASE_DOMAIN="${BASE_DOMAIN:-localhost}"
export ENV_FILE="${ENV_FILE:-.env.local}"

# Directory paths
export NSELF_ROOT="${NSELF_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)}"
export NSELF_BIN="${NSELF_BIN:-$NSELF_ROOT/bin}"
export NSELF_SRC="${NSELF_SRC:-$NSELF_ROOT/src}"
export NSELF_TEMPLATES="${NSELF_TEMPLATES:-$NSELF_SRC/templates}"
export NSELF_CERTS="${NSELF_CERTS:-$NSELF_SRC/certs}"
export NSELF_LIB="${NSELF_LIB:-$NSELF_SRC/lib}"
export NSELF_LOGS="${NSELF_LOGS:-$NSELF_ROOT/logs}"

# Docker defaults
export COMPOSE_PROJECT_NAME="${PROJECT_NAME}"
export COMPOSE_ENV_FILE="${ENV_FILE}"
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

# Timeouts and retries
export STARTUP_TIMEOUT="${STARTUP_TIMEOUT:-60}"
export HEALTH_CHECK_INTERVAL="${HEALTH_CHECK_INTERVAL:-2}"
export MAX_RETRIES="${MAX_RETRIES:-3}"
export RETRY_DELAY="${RETRY_DELAY:-5}"

# Feature flags
export AUTO_FIX_ENABLED="${AUTO_FIX_ENABLED:-true}"
export VERBOSE="${VERBOSE:-false}"
export DEBUG="${DEBUG:-false}"
export NO_COLOR="${NO_COLOR:-}"

# Service ports (defaults)
export POSTGRES_PORT="${POSTGRES_PORT:-5432}"
export HASURA_PORT="${HASURA_PORT:-8080}"
export MINIO_PORT="${MINIO_PORT:-9000}"
export REDIS_PORT="${REDIS_PORT:-6379}"
export NGINX_HTTP_PORT="${NGINX_HTTP_PORT:-80}"
export NGINX_HTTPS_PORT="${NGINX_HTTPS_PORT:-443}"

# Required environment variables
export REQUIRED_ENV_VARS=(
  "PROJECT_NAME"
  "BASE_DOMAIN"
  "POSTGRES_PASSWORD"
  "HASURA_GRAPHQL_ADMIN_SECRET"
)
