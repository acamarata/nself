#!/usr/bin/env bash

# Smart Defaults System for nself
# Provides default values for all configuration options
# Priority: .env > .env.local > defaults

# Apply smart defaults for any missing environment variables
apply_smart_defaults() {
  # Core Settings
  : ${ENV:=dev}
  : ${PROJECT_NAME:=myproject}
  : ${BASE_DOMAIN:=local.nself.org}
  : ${DB_ENV_SEEDS:=true}

  # PostgreSQL
  : ${POSTGRES_VERSION:=16-alpine}
  : ${POSTGRES_HOST:=postgres}
  : ${POSTGRES_PORT:=5432}
  : ${POSTGRES_DB:=nhost}
  : ${POSTGRES_USER:=postgres}
  : ${POSTGRES_PASSWORD:=postgres-dev-password}
  : ${POSTGRES_EXTENSIONS:=uuid-ossp}

  # Hasura
  : ${HASURA_VERSION:=v2.44.0}
  : ${HASURA_GRAPHQL_ADMIN_SECRET:=hasura-admin-secret-dev}

  # JWT Configuration - Support both new simple format and legacy JSON format
  : ${HASURA_JWT_KEY:=development-secret-key-minimum-32-characters-long}
  : ${JWT_KEY:=$HASURA_JWT_KEY}
  : ${HASURA_JWT_TYPE:=HS256}

  # If HASURA_GRAPHQL_JWT_SECRET is not set, construct it from the simple variables
  if [[ -z "${HASURA_GRAPHQL_JWT_SECRET:-}" ]]; then
    HASURA_GRAPHQL_JWT_SECRET="{\"type\":\"${HASURA_JWT_TYPE}\",\"key\":\"${HASURA_JWT_KEY}\"}"
  fi

  # Set console/dev mode based on ENV
  if [[ "$ENV" == "prod" ]]; then
    : ${HASURA_GRAPHQL_ENABLE_CONSOLE:=false}
    : ${HASURA_GRAPHQL_DEV_MODE:=false}
  else
    : ${HASURA_GRAPHQL_ENABLE_CONSOLE:=true}
    : ${HASURA_GRAPHQL_DEV_MODE:=true}
  fi

  : ${HASURA_GRAPHQL_ENABLE_TELEMETRY:=false}
  : ${HASURA_GRAPHQL_CORS_DOMAIN:=*}
  : ${HASURA_ROUTE:=api.${BASE_DOMAIN}}

  # Auth
  : ${AUTH_VERSION:=0.36.0}
  : ${AUTH_HOST:=auth}
  : ${AUTH_PORT:=4000}
  : ${AUTH_CLIENT_URL:=http://localhost:3000}
  : ${AUTH_JWT_REFRESH_TOKEN_EXPIRES_IN:=2592000}
  : ${AUTH_JWT_ACCESS_TOKEN_EXPIRES_IN:=900}
  : ${AUTH_WEBAUTHN_ENABLED:=false}
  : ${AUTH_ROUTE:=auth.${BASE_DOMAIN}}

  # Email (Development defaults to MailPit)
  : ${AUTH_SMTP_HOST:=mailpit}
  : ${AUTH_SMTP_PORT:=1025}
  : ${AUTH_SMTP_USER:=""}
  : ${AUTH_SMTP_PASS:=""}
  : ${AUTH_SMTP_SECURE:=false}
  : ${AUTH_SMTP_SENDER:=noreply@${BASE_DOMAIN}}

  # Storage
  : ${STORAGE_VERSION:=0.6.1}
  : ${STORAGE_ROUTE:=storage.${BASE_DOMAIN}}
  : ${STORAGE_CONSOLE_ROUTE:=storage-console.${BASE_DOMAIN}}
  : ${MINIO_VERSION:=latest}
  : ${MINIO_PORT:=9000}
  : ${MINIO_ROOT_USER:=minioadmin}
  : ${MINIO_ROOT_PASSWORD:=minioadmin}
  : ${S3_ACCESS_KEY:=storage-access-key-dev}
  : ${S3_SECRET_KEY:=storage-secret-key-dev}
  : ${S3_BUCKET:=nhost}
  : ${S3_REGION:=us-east-1}

  # Nginx
  : ${NGINX_VERSION:=alpine}
  : ${NGINX_HTTP_PORT:=80}
  : ${NGINX_HTTPS_PORT:=443}

  # SSL
  : ${SSL_MODE:=local}

  # Optional Services (all disabled by default)
  : ${FUNCTIONS_ENABLED:=false}
  : ${FUNCTIONS_ROUTE:=functions.${BASE_DOMAIN}}
  : ${DASHBOARD_ENABLED:=false}
  : ${DASHBOARD_VERSION:=latest}
  : ${DASHBOARD_ROUTE:=dashboard.${BASE_DOMAIN}}
  : ${REDIS_ENABLED:=false}
  : ${REDIS_VERSION:=7-alpine}
  : ${REDIS_PORT:=6379}
  : ${REDIS_PASSWORD:=""}
  
  # MLflow - ML Experiment Tracking
  : ${MLFLOW_ENABLED:=false}
  : ${MLFLOW_VERSION:=2.9.2}
  : ${MLFLOW_PORT:=5000}
  : ${MLFLOW_ROUTE:=mlflow.${BASE_DOMAIN}}
  : ${MLFLOW_DB_NAME:=mlflow}
  : ${MLFLOW_ARTIFACTS_BUCKET:=mlflow-artifacts}
  : ${MLFLOW_AUTH_ENABLED:=false}
  : ${MLFLOW_AUTH_USERNAME:=admin}
  : ${MLFLOW_AUTH_PASSWORD:=mlflow-admin-password}

  # Email Provider
  : ${EMAIL_PROVIDER:=mailpit}
  : ${MAILPIT_SMTP_PORT:=1025}
  : ${MAILPIT_UI_PORT:=8025}
  : ${MAILPIT_ROUTE:=mail.${BASE_DOMAIN}}
  : ${EMAIL_FROM:=noreply@${BASE_DOMAIN}}

  # Microservices (all disabled by default)
  : ${SERVICES_ENABLED:=false}
  : ${NESTJS_ENABLED:=false}
  : ${NESTJS_SERVICES:=""}
  : ${NESTJS_USE_TYPESCRIPT:=true}
  : ${NESTJS_PORT_START:=3100}
  : ${BULLMQ_ENABLED:=false}
  : ${BULLMQ_WORKERS:=""}
  : ${BULLMQ_DASHBOARD_ENABLED:=false}
  : ${BULLMQ_DASHBOARD_PORT:=4200}
  : ${BULLMQ_DASHBOARD_ROUTE:=queues.${BASE_DOMAIN}}
  : ${GOLANG_ENABLED:=false}
  : ${GOLANG_SERVICES:=""}
  : ${GOLANG_PORT_START:=3200}
  : ${PYTHON_ENABLED:=false}
  : ${PYTHON_SERVICES:=""}
  : ${PYTHON_FRAMEWORK:=fastapi}
  : ${PYTHON_PORT_START:=3300}
  : ${NESTJS_RUN_ENABLED:=false}
  : ${NESTJS_RUN_PORT:=3400}

  # Advanced/Internal
  : ${HASURA_METADATA_DATABASE_URL:=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}}
  : ${DOCKER_NETWORK:=${PROJECT_NAME}_network}
  : ${HASURA_PORT:=8080}
  : ${HASURA_CONSOLE_PORT:=9695}
  : ${FUNCTIONS_PORT:=4300}
  : ${DASHBOARD_PORT:=4500}
  : ${STORAGE_PORT:=5001}
  : ${S3_ENDPOINT:=http://minio:${MINIO_PORT}}
  : ${FILES_ROUTE:=files.${BASE_DOMAIN}}
  : ${MAIL_ROUTE:=mail.${BASE_DOMAIN}}

  # Export all variables
  export ENV PROJECT_NAME BASE_DOMAIN DB_ENV_SEEDS
  export POSTGRES_VERSION POSTGRES_HOST POSTGRES_PORT POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD POSTGRES_EXTENSIONS
  export HASURA_VERSION HASURA_GRAPHQL_ADMIN_SECRET HASURA_GRAPHQL_JWT_SECRET
  export HASURA_JWT_KEY HASURA_JWT_TYPE
  export HASURA_GRAPHQL_ENABLE_CONSOLE HASURA_GRAPHQL_DEV_MODE HASURA_GRAPHQL_ENABLE_TELEMETRY
  export HASURA_GRAPHQL_CORS_DOMAIN HASURA_ROUTE
  export AUTH_VERSION AUTH_HOST AUTH_PORT AUTH_CLIENT_URL
  export AUTH_JWT_REFRESH_TOKEN_EXPIRES_IN AUTH_JWT_ACCESS_TOKEN_EXPIRES_IN
  export AUTH_WEBAUTHN_ENABLED AUTH_ROUTE
  export AUTH_SMTP_HOST AUTH_SMTP_PORT AUTH_SMTP_USER AUTH_SMTP_PASS AUTH_SMTP_SECURE AUTH_SMTP_SENDER
  export STORAGE_VERSION STORAGE_ROUTE STORAGE_CONSOLE_ROUTE
  export MINIO_VERSION MINIO_PORT MINIO_ROOT_USER MINIO_ROOT_PASSWORD
  export S3_ACCESS_KEY S3_SECRET_KEY S3_BUCKET S3_REGION
  export NGINX_VERSION NGINX_HTTP_PORT NGINX_HTTPS_PORT
  export SSL_MODE
  export FUNCTIONS_ENABLED FUNCTIONS_ROUTE
  export DASHBOARD_ENABLED DASHBOARD_VERSION DASHBOARD_ROUTE
  export REDIS_ENABLED REDIS_VERSION REDIS_PORT REDIS_PASSWORD
  export MLFLOW_ENABLED MLFLOW_VERSION MLFLOW_PORT MLFLOW_ROUTE
  export MLFLOW_DB_NAME MLFLOW_ARTIFACTS_BUCKET MLFLOW_AUTH_ENABLED
  export MLFLOW_AUTH_USERNAME MLFLOW_AUTH_PASSWORD
  export EMAIL_PROVIDER MAILPIT_SMTP_PORT MAILPIT_UI_PORT MAILPIT_ROUTE EMAIL_FROM
  export SERVICES_ENABLED
  export NESTJS_ENABLED NESTJS_SERVICES NESTJS_USE_TYPESCRIPT NESTJS_PORT_START
  export BULLMQ_ENABLED BULLMQ_WORKERS BULLMQ_DASHBOARD_ENABLED BULLMQ_DASHBOARD_PORT BULLMQ_DASHBOARD_ROUTE
  export GOLANG_ENABLED GOLANG_SERVICES GOLANG_PORT_START
  export PYTHON_ENABLED PYTHON_SERVICES PYTHON_FRAMEWORK PYTHON_PORT_START
  export NESTJS_RUN_ENABLED NESTJS_RUN_PORT
  export HASURA_METADATA_DATABASE_URL DOCKER_NETWORK
  export HASURA_PORT HASURA_CONSOLE_PORT FUNCTIONS_PORT DASHBOARD_PORT STORAGE_PORT
  export S3_ENDPOINT FILES_ROUTE MAIL_ROUTE
}

# Load environment files with proper priority
load_env_with_defaults() {
  # IMPORTANT: Strict priority order - only ONE env file is used!
  # Priority: .env > .env.local > .env.dev > defaults

  # Check for .env first (production - highest priority)
  if [[ -f ".env" ]]; then
    set -a
    source .env
    set +a
    # Apply defaults for missing values
    apply_smart_defaults
    # STOP - ignore all other env files
    return
  fi

  # Check for .env.local (development)
  if [[ -f ".env.local" ]]; then
    # Use env.sh utility if available for proper loading
    if [[ -f "$SCRIPT_DIR/../utils/env.sh" ]]; then
      source "$SCRIPT_DIR/../utils/env.sh"
      load_env_with_priority
    else
      set -a
      source .env.local
      set +a
    fi
    # Apply defaults for missing values
    apply_smart_defaults
    # STOP - ignore .env.dev
    return
  fi

  # Check for .env.dev (team defaults)
  if [[ -f ".env.dev" ]]; then
    set -a
    source .env.dev
    set +a
    # Apply defaults for missing values
    apply_smart_defaults
    return
  fi

  # No env files found - use only defaults
  apply_smart_defaults

  # Re-construct JWT secret if using simple format
  if [[ -z "${HASURA_GRAPHQL_JWT_SECRET:-}" ]] && [[ -n "${HASURA_JWT_KEY:-}" ]]; then
    : ${HASURA_JWT_TYPE:=HS256}
    HASURA_GRAPHQL_JWT_SECRET="{\"type\":\"${HASURA_JWT_TYPE}\",\"key\":\"${HASURA_JWT_KEY}\"}"
  fi

  # Re-apply computed values that depend on other vars
  : ${HASURA_ROUTE:=api.${BASE_DOMAIN}}
  : ${AUTH_ROUTE:=auth.${BASE_DOMAIN}}
  : ${STORAGE_ROUTE:=storage.${BASE_DOMAIN}}
  : ${STORAGE_CONSOLE_ROUTE:=storage-console.${BASE_DOMAIN}}
  : ${FUNCTIONS_ROUTE:=functions.${BASE_DOMAIN}}
  : ${DASHBOARD_ROUTE:=dashboard.${BASE_DOMAIN}}
  : ${MAILPIT_ROUTE:=mail.${BASE_DOMAIN}}
  : ${BULLMQ_DASHBOARD_ROUTE:=queues.${BASE_DOMAIN}}
  : ${MLFLOW_ROUTE:=mlflow.${BASE_DOMAIN}}
  : ${AUTH_SMTP_SENDER:=noreply@${BASE_DOMAIN}}
  : ${EMAIL_FROM:=noreply@${BASE_DOMAIN}}
  : ${FILES_ROUTE:=files.${BASE_DOMAIN}}
  : ${MAIL_ROUTE:=mail.${BASE_DOMAIN}}
  : ${DOCKER_NETWORK:=${PROJECT_NAME}_network}
  : ${HASURA_METADATA_DATABASE_URL:=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}}
  : ${S3_ENDPOINT:=http://minio:${MINIO_PORT}}

  # Export computed values
  export HASURA_ROUTE AUTH_ROUTE STORAGE_ROUTE STORAGE_CONSOLE_ROUTE
  export FUNCTIONS_ROUTE DASHBOARD_ROUTE MAILPIT_ROUTE BULLMQ_DASHBOARD_ROUTE MLFLOW_ROUTE
  export AUTH_SMTP_SENDER EMAIL_FROM FILES_ROUTE MAIL_ROUTE
  export DOCKER_NETWORK HASURA_METADATA_DATABASE_URL S3_ENDPOINT
  export HASURA_GRAPHQL_JWT_SECRET HASURA_JWT_KEY HASURA_JWT_TYPE
}

export -f apply_smart_defaults
export -f load_env_with_defaults
