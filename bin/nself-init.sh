#!/bin/bash

# nself-init.sh - Helper script to create project directory structure and initial files

set -e

# ----------------------------
# Resolve Script Directory
# ----------------------------
SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do
  DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
SCRIPT_DIR="$(cd -P "$(dirname "$SOURCE")" >/dev/null 2>&1 && pwd)"

# Create necessary directories
mkdir -p .nself/traefik/htpasswd

# Create .htpasswd file with default credentials
echo "Creating .htpasswd file..."
printf "admin:$(openssl passwd -apr1 metrics_password)\n" > .nself/traefik/htpasswd/.htpasswd

# Create other necessary directories
mkdir -p services/{auth,storage,functions}
mkdir -p data
mkdir -p emails

echo "✅ Project structure created successfully!"

# ----------------------------
# Helper Functions
# ----------------------------

# Function to print informational messages
echo_info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

# Function to print error messages
echo_error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

# Function to load environment variables
load_env() {
  if [ -f ".env" ]; then
    ENV_FILE=".env"
  elif [ -f ".env.dev" ]; then
    ENV_FILE=".env.dev"
  else
    echo_error "No .env or .env.dev file found."
    exit 1
  fi

  echo_info "Loading environment variables from $ENV_FILE"
  set -o allexport
  source "$ENV_FILE"
  set +o allexport
}

# Function to collect hostnames from *_ROUTE variables
collect_route_hosts() {
  ROUTE_HOSTS=()
  for var in $(compgen -v | grep '_ROUTE$'); do
    ROUTE_VALUE="${!var}"
    if [ -n "$ROUTE_VALUE" ]; then
      HOSTNAME=$(echo "$ROUTE_VALUE" | cut -d':' -f1)
      ROUTE_HOSTS+=("$HOSTNAME")
    fi
  done
  echo "${ROUTE_HOSTS[@]}"
}

# Function to collect hostnames from OTHER_ROUTES
collect_other_routes_hosts() {
  OTHER_HOSTS=()
  if [ -n "$OTHER_ROUTES" ]; then
    while IFS= read -r line; do
      # Remove leading/trailing whitespace
      line=$(echo "$line" | xargs)
      # Skip empty lines and comments
      if [[ -z "$line" || "$line" == \#* ]]; then
        continue
      fi
      DOMAIN=$(echo "$line" | cut -d'=' -f1)
      OTHER_HOSTS+=("$DOMAIN")
    done <<< "$OTHER_ROUTES"
  fi
  echo "${OTHER_HOSTS[@]}"
}

# Function to split comma-separated string into array
split_hosts() {
  IFS=',' read -ra ADDR <<< "$1"
  echo "${ADDR[@]}"
}

# ----------------------------
# Main Functions
# ----------------------------

# Function to create directories and files
create_directories_and_files() {
  echo_info "Creating directory structure..."

  # Root level directories
  mkdir -p .nself/traefik/htpasswd
  mkdir -p data
  mkdir -p emails
  mkdir -p functions
  mkdir -p services/{auth,storage,functions}

  # Create .htpasswd file
  echo_info "Creating .htpasswd file..."
  printf "admin:$(openssl passwd -apr1 metrics_password)\n" > .nself/traefik/htpasswd/.htpasswd
  chmod 644 .nself/traefik/htpasswd/.htpasswd

  # Create .env file if it doesn't exist
  if [ ! -f ".env" ]; then
    echo_info "Creating .env file..."
    cat <<EOF > .env
# Project Configuration
PROJECT_NAME=nproj
PROJECT_DOMAIN=localhost

# Route Configuration
HASURA_ROUTE=hasura.localhost
AUTH_ROUTE=auth.localhost
STORAGE_ROUTE=storage.localhost
FUNCTIONS_ROUTE=functions.localhost
METRICS_ROUTE=metrics.localhost
MAILHOG_ROUTE=mailhog.localhost

# Database Configuration
POSTGRES_DB=postgres
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_PORT=5434

# Hasura Configuration
HASURA_GRAPHQL_ADMIN_SECRET=nhost-admin-secret
HASURA_GRAPHQL_JWT_SECRET={"type":"HS256","key":"myjwtsecretthats32characterslong"}
HASURA_GRAPHQL_UNAUTHORIZED_ROLE=public

# Auth Configuration
AUTH_ACCESS_TOKEN_EXPIRES_IN=900
AUTH_REFRESH_TOKEN_EXPIRES_IN=2592000
AUTH_MFA_ENABLED=false
AUTH_MFA_TOTP_ISSUER=nproj
AUTH_PASSWORD_MIN_LENGTH=8
AUTH_PASSWORD_REQUIRE_SPECIAL=false
AUTH_EMAIL_VERIFICATION_REQUIRED=true

# Storage Configuration
STORAGE_BUCKET_NAME=nproj-bucket
STORAGE_ACCESS_KEY=storage_access_key
STORAGE_SECRET_KEY=storage_secret_key
STORAGE_MAX_FILE_SIZE=10485760
STORAGE_PUBLIC_ACCESS=true
STORAGE_PORT=9000

# Functions Configuration
FUNCTIONS_TIMEOUT=30
FUNCTIONS_MEMORY_LIMIT=128
FUNCTIONS_PORT=1337
FUNCTIONS_API_ROUTE=/functions

# Metrics Configuration
METRICS_PASSWORD=metrics_password

# Docker Volume Settings
FUNCTION_VOLUME=nproj_function
PROJECT_DATA_VOLUME=nproj_data
INIT_DB_VOLUME=nproj_init_db
PROJECT_DB_VOLUME=nproj_db
MAILHOG_VOLUME=nproj_mailhog
MINIO_VOLUME=nproj_minio
MINIO1_VOLUME=nproj_minio1
MINIO2_VOLUME=nproj_minio2
MINIO3_VOLUME=nproj_minio3
MINIO4_VOLUME=nproj_minio4
STORAGE_VOLUME=nproj_storage
EOF
  fi

  # Create docker-compose.yml
  echo_info "Creating docker-compose.yml..."
  cat <<EOF > docker-compose.yml
services:
  postgres:
    image: nhost/postgres:16.4-202401126-1
    environment:
      POSTGRES_DB: \${POSTGRES_DB:-postgres}
      POSTGRES_USER: \${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD:-postgres}
    ports:
      - "\${POSTGRES_PORT:-5434}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - postgres_lib:/var/lib/postgresql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

  hasura:
    image: nhost/graphql-engine:v2.44.0-ce
    ports:
      - "1337:8080"
    depends_on:
      - postgres
    environment:
      HASURA_GRAPHQL_DATABASE_URL: postgres://postgres:postgres@postgres:5432/postgres
      HASURA_GRAPHQL_ADMIN_SECRET: \${HASURA_GRAPHQL_ADMIN_SECRET:-nhost-admin-secret}
      HASURA_GRAPHQL_JWT_SECRET: \${HASURA_GRAPHQL_JWT_SECRET:-'{"type":"HS256","key":"myjwtsecretthats32characterslong"}'}
      HASURA_GRAPHQL_UNAUTHORIZED_ROLE: \${HASURA_GRAPHQL_UNAUTHORIZED_ROLE:-public}
      HASURA_GRAPHQL_ENABLE_CONSOLE: true
      HASURA_GRAPHQL_LOG_LEVEL: info
      HASURA_GRAPHQL_ENABLE_CORS: true
      HASURA_GRAPHQL_CORS_DOMAIN: "*"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped

  auth:
    image: nhost/hasura-auth:0.36.1
    depends_on:
      - hasura
      - postgres
    ports:
      - "4000:4000"
    environment:
      HASURA_ENDPOINT: http://hasura:8080
      AUTH_ACCESS_TOKEN_EXPIRES_IN: \${AUTH_ACCESS_TOKEN_EXPIRES_IN:-900}
      AUTH_REFRESH_TOKEN_EXPIRES_IN: \${AUTH_REFRESH_TOKEN_EXPIRES_IN:-2592000}
      AUTH_MFA_ENABLED: \${AUTH_MFA_ENABLED:-false}
      AUTH_MFA_TOTP_ISSUER: \${AUTH_MFA_TOTP_ISSUER:-nproj}
      AUTH_PASSWORD_MIN_LENGTH: \${AUTH_PASSWORD_MIN_LENGTH:-8}
      AUTH_PASSWORD_REQUIRE_SPECIAL: \${AUTH_PASSWORD_REQUIRE_SPECIAL:-false}
      AUTH_EMAIL_VERIFICATION_REQUIRED: \${AUTH_EMAIL_VERIFICATION_REQUIRED:-true}
      HASURA_GRAPHQL_JWT_SECRET: \${HASURA_GRAPHQL_JWT_SECRET:-'{"type":"HS256","key":"myjwtsecretthats32characterslong"}'}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped

  storage:
    image: nhost/hasura-storage:0.6.1
    depends_on:
      - hasura
    ports:
      - "9000:9000"
    environment:
      STORAGE_BUCKET_NAME: \${STORAGE_BUCKET_NAME:-nproj-bucket}
      STORAGE_ACCESS_KEY: \${STORAGE_ACCESS_KEY:-storage_access_key}
      STORAGE_SECRET_KEY: \${STORAGE_SECRET_KEY:-storage_secret_key}
      STORAGE_MAX_FILE_SIZE: \${STORAGE_MAX_FILE_SIZE:-10485760}
      STORAGE_PUBLIC_ACCESS: \${STORAGE_PUBLIC_ACCESS:-true}
      STORAGE_PORT: \${STORAGE_PORT:-9000}
    command: serve
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 5
    restart: unless-stopped

  functions:
    image: nhost/functions:1.2.0
    depends_on:
      - auth
      - postgres
    ports:
      - "4001:1337"
    environment:
      FUNCTIONS_TIMEOUT: \${FUNCTIONS_TIMEOUT:-30}
      FUNCTIONS_MEMORY_LIMIT: \${FUNCTIONS_MEMORY_LIMIT:-128}
      FUNCTIONS_PORT: \${FUNCTIONS_PORT:-1337}
      FUNCTIONS_API_ROUTE: \${FUNCTIONS_API_ROUTE:-/functions}
    volumes:
      - ./functions:/usr/src/app
    working_dir: /usr/src/app
    restart: unless-stopped

volumes:
  postgres_data:
  postgres_lib:
EOF

  echo_info "✅ Project structure created successfully!"
}

# ----------------------------
# Execute the script
# ----------------------------

create_directories_and_files
