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
  if [ -f ".env.example" ]; then
    echo_info "No .env or .env.dev found. Copying .env.example to .env"
    cp .env.example .env
    ENV_FILE=".env"
  elif [ -f ".env" ]; then
    ENV_FILE=".env"
  elif [ -f ".env.dev" ]; then
    ENV_FILE=".env.dev"
  else
    echo_error "No .env, .env.dev, or .env.example file found."
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

# Function to create docker-compose.yml
create_docker_compose() {
  echo_info "Creating docker-compose.yml..."

  # Pre-create required folders with proper permissions
  mkdir -p ./data/postgres
  mkdir -p ./data/minio
  mkdir -p ./data/traefik
  mkdir -p ./data/certificates
  
  # Set proper permissions
  chmod -R 777 ./data
  
  # Create traefik configuration directory and files
  mkdir -p ./data/traefik/dynamic
  mkdir -p ./data/traefik/static

  # Create MinIO configuration and structure
  mkdir -p ./data/minio/config
  mkdir -p ./data/minio/data
  mkdir -p ./data/minio/buckets

  # Create MinIO package.json
  cat <<'EOF' > ./data/minio/package.json
{
  "name": "nself-minio",
  "version": "1.0.0",
  "description": "MinIO configuration for Nself",
  "main": "index.js",
  "scripts": {
    "start": "node index.js"
  },
  "dependencies": {
    "minio": "^7.1.3"
  }
}
EOF

  # Create MinIO bucket configuration
  cat <<'EOF' > ./data/minio/config/buckets.json
{
  "buckets": [
    {
      "name": "public",
      "policy": "public-read",
      "versioning": false
    },
    {
      "name": "private",
      "policy": "private",
      "versioning": true
    }
  ]
}
EOF

  # Create MinIO initialization script
  cat <<'EOF' > ./data/minio/config/init.sh
#!/bin/bash

# Wait for MinIO to be ready
until curl -s http://localhost:9000/minio/health/live; do
  echo "Waiting for MinIO to be ready..."
  sleep 1
done

# Create buckets
mc alias set myminio http://localhost:9000 ${MINIO_ROOT_USER:-minioadmin} ${MINIO_ROOT_PASSWORD:-minioadmin}

# Create public bucket
mc mb myminio/public
mc policy set public myminio/public

# Create private bucket
mc mb myminio/private
mc policy set private myminio/private

echo "MinIO initialization completed"
EOF

  # Make init script executable
  chmod +x ./data/minio/config/init.sh

  # Create MinIO client configuration
  cat <<'EOF' > ./data/minio/config/mc-config.json
{
  "version": "10",
  "aliases": {
    "myminio": {
      "url": "http://localhost:9000",
      "accessKey": "${MINIO_ROOT_USER:-minioadmin}",
      "secretKey": "${MINIO_ROOT_PASSWORD:-minioadmin}",
      "api": "s3v4",
      "path": "auto"
    }
  }
}
EOF

  # Create basic traefik configuration
  cat <<'EOF' > ./data/traefik/static/traefik.yml
entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"

api:
  dashboard: true
  insecure: false

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
  file:
    directory: "/etc/traefik/dynamic"
    watch: true

certificatesResolvers:
  default:
    acme:
      email: "your-email@example.com"
      storage: "/certificates/acme.json"
      httpChallenge:
        entryPoint: web
EOF

  # Create dynamic configuration
  cat <<'EOF' > ./data/traefik/dynamic/middleware.yml
http:
  middlewares:
    auth:
      basicAuth:
        usersFile: "/etc/traefik/.htpasswd"
EOF

  # Create .htpasswd file for basic auth
  echo "admin:\$apr1\$ruca84Hq\$STdCQ4m5.kB3cWm8uG1U9/" > ./data/traefik/.htpasswd
  chmod 644 ./data/traefik/.htpasswd

  # Create docker-compose.yml
  cat <<'EOF' > docker-compose.yml
services:
  traefik:
    image: traefik:v2.10
    container_name: traefik
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./data/traefik:/etc/traefik
      - ./data/certificates:/certificates
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.traefik.rule=Host(`traefik.local.nself.org`)"
      - "traefik.http.routers.traefik.entrypoints=websecure"
      - "traefik.http.routers.traefik.service=api@internal"
      - "traefik.http.services.traefik.loadbalancer.server.port=8080"
      - "traefik.http.routers.traefik.middlewares=auth"
      - "traefik.http.middlewares.auth.basicauth.usersfile=/etc/traefik/.htpasswd"

  postgres:
    image: postgres:15-alpine
    container_name: postgres
    restart: unless-stopped
    environment:
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_DB: ${POSTGRES_DB:-postgres}
    volumes:
      - ./data/postgres:/var/lib/postgresql/data
    networks:
      - nself_network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  hasura:
    image: hasura/graphql-engine:v2.35.0
    container_name: hasura
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      HASURA_GRAPHQL_DATABASE_URL: postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-postgres}@postgres:5432/${POSTGRES_DB:-postgres}
      HASURA_GRAPHQL_ENABLE_CONSOLE: "true"
      HASURA_GRAPHQL_DEV_MODE: "true"
      HASURA_GRAPHQL_ENABLED_LOG_TYPES: startup, http-log, webhook-log, websocket-log, query-log
      HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET:-admin-secret}
      HASURA_GRAPHQL_JWT_SECRET: '{"type":"HS256", "key":"${HASURA_GRAPHQL_JWT_SECRET:-your-jwt-secret}"}'
      HASURA_GRAPHQL_UNAUTHORIZED_ROLE: anonymous
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.hasura-console.rule=Host(`console.local.nself.org`)"
      - "traefik.http.routers.hasura-console.entrypoints=websecure"
      - "traefik.http.services.hasura-console.loadbalancer.server.port=8080"
      - "traefik.http.routers.hasura-api.rule=Host(`api.local.nself.org`)"
      - "traefik.http.routers.hasura-api.entrypoints=websecure"
      - "traefik.http.services.hasura-api.loadbalancer.server.port=8080"

  auth:
    image: nhost/hasura-auth:latest
    container_name: auth
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      AUTH_DATABASE_URL: postgres://${POSTGRES_USER:-postgres}:${POSTGRES_PASSWORD:-postgres}@postgres:5432/${POSTGRES_DB:-postgres}
      AUTH_HASURA_GRAPHQL_URL: http://hasura:8080/v1/graphql
      AUTH_HASURA_GRAPHQL_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET:-admin-secret}
      AUTH_SMTP_HOST: ${AUTH_SMTP_HOST:-smtp.gmail.com}
      AUTH_SMTP_PORT: ${AUTH_SMTP_PORT:-587}
      AUTH_SMTP_USER: ${AUTH_SMTP_USER:-}
      AUTH_SMTP_PASS: ${AUTH_SMTP_PASS:-}
      AUTH_SMTP_SENDER: ${AUTH_SMTP_SENDER:-}
      AUTH_SITE_URL: ${AUTH_SITE_URL:-http://localhost:3000}
      AUTH_ADDITIONAL_REDIRECT_URLS: ${AUTH_ADDITIONAL_REDIRECT_URLS:-}
      AUTH_JWT_SECRET: ${AUTH_JWT_SECRET:-your-jwt-secret}
      AUTH_ANONYMOUS_USERS_ENABLED: ${AUTH_ANONYMOUS_USERS_ENABLED:-true}
      AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED: ${AUTH_EMAIL_SIGNIN_EMAIL_VERIFIED_REQUIRED:-false}
      AUTH_ACCESS_TOKEN_EXPIRES_IN: ${AUTH_ACCESS_TOKEN_EXPIRES_IN:-3600}
      AUTH_REFRESH_TOKEN_EXPIRES_IN: ${AUTH_REFRESH_TOKEN_EXPIRES_IN:-2592000}
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.auth.rule=Host(`auth.local.nself.org`)"
      - "traefik.http.routers.auth.entrypoints=websecure"
      - "traefik.http.services.auth.loadbalancer.server.port=4000"

  minio:
    image: minio/minio:latest
    container_name: minio
    restart: unless-stopped
    environment:
      MINIO_ROOT_USER: ${STORAGE_S3_ACCESS_KEY:-minioadmin}
      MINIO_ROOT_PASSWORD: ${STORAGE_S3_SECRET_KEY:-minioadmin}
    volumes:
      - ./data/minio/data:/data
      - ./data/minio/config:/root/.mc
    command: server /data --console-address ":9001"
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.minio.rule=Host(`storage.local.nself.org`)"
      - "traefik.http.routers.minio.entrypoints=websecure"
      - "traefik.http.services.minio.loadbalancer.server.port=9000"
      - "traefik.http.routers.minio-console.rule=Host(`minio-console.local.nself.org`)"
      - "traefik.http.routers.minio-console.entrypoints=websecure"
      - "traefik.http.services.minio-console.loadbalancer.server.port=9001"

  minio-client:
    image: minio/mc:latest
    container_name: minio-client
    depends_on:
      - minio
    volumes:
      - ./data/minio/config:/root/.mc
      - ./data/minio/config/init.sh:/init.sh
    entrypoint: ["/bin/sh", "-c"]
    command: ["/init.sh"]
    networks:
      - nself_network

  functions:
    image: nhost/functions:latest
    container_name: functions
    restart: unless-stopped
    volumes:
      - ./functions:/app/functions
    environment:
      NHOST_FUNCTIONS_URL: http://functions:3000
      NHOST_HASURA_URL: http://hasura:8080/v1/graphql
      NHOST_HASURA_ADMIN_SECRET: ${HASURA_GRAPHQL_ADMIN_SECRET:-admin-secret}
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.functions.rule=Host(`functions.local.nself.org`)"
      - "traefik.http.routers.functions.entrypoints=websecure"
      - "traefik.http.services.functions.loadbalancer.server.port=3000"

  mailhog:
    image: mailhog/mailhog:latest
    container_name: mailhog
    restart: unless-stopped
    ports:
      - "1025:1025"  # SMTP server
      - "8025:8025"  # Web UI
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.mailhog.rule=Host(`mailhog.local.nself.org`)"
      - "traefik.http.routers.mailhog.entrypoints=websecure"
      - "traefik.http.services.mailhog.loadbalancer.server.port=8025"

  app1:
    image: nginx:alpine
    container_name: app1
    restart: unless-stopped
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.app1.rule=Host(`app1.local.nself.org`)"
      - "traefik.http.routers.app1.entrypoints=websecure"
      - "traefik.http.services.app1.loadbalancer.server.port=80"

  app2-api:
    image: nginx:alpine
    container_name: app2-api
    restart: unless-stopped
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.app2-api.rule=Host(`api.app2.local.nself.org`)"
      - "traefik.http.routers.app2-api.entrypoints=websecure"
      - "traefik.http.services.app2-api.loadbalancer.server.port=80"

  app3-api:
    image: nginx:alpine
    container_name: app3-api
    restart: unless-stopped
    networks:
      - nself_network
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.app3-api.rule=Host(`api.app3.local.nself.org`)"
      - "traefik.http.routers.app3-api.entrypoints=websecure"
      - "traefik.http.services.app3-api.loadbalancer.server.port=80"

networks:
  nself_network:
    name: nself_network
    driver: bridge
EOF
}

# ----------------------------
# Main Functions
# ----------------------------

# Function to create directories and files
create_directories_and_files() {
  echo_info "Creating directory structure..."

  # Root level directories
  # mkdir -p .traefik
  mkdir -p .nself/traefik/htpasswd
  touch .nself/traefik/htpasswd/.htpasswd
  mkdir -p data
  mkdir -p emails
  mkdir -p functions
  mkdir -p metadata
  mkdir -p migrations
  mkdir -p seeds
  mkdir -p services

  # Load environment variables to get HOSTS
  load_env

  # Create docker-compose.yml
  create_docker_compose

  # Collect hostnames from *_ROUTE variables
  ROUTE_HOSTS=($(collect_route_hosts))

  # Collect hostnames from OTHER_ROUTES
  OTHER_HOSTS=($(collect_other_routes_hosts))

  # Combine all hostnames into a single array
  ALL_HOSTS=("${ROUTE_HOSTS[@]}" "${OTHER_HOSTS[@]}")

  # Remove duplicate hostnames
  UNIQUE_HOSTS=($(printf "%s\n" "${ALL_HOSTS[@]}" | sort -u))

  if [ ${#UNIQUE_HOSTS[@]} -eq 0 ]; then
    echo_error "No hostnames found in *_ROUTE variables or OTHER_ROUTES."
    exit 1
  fi

  # Create the traefik.yml file
  # echo_info "Creating traefik.yml file..."
  # {
  #   echo "# .traefik/traefik.yml"
  #   echo ""
  #   echo "entryPoints:"
  #   echo "  web:"
  #   echo "    address: \":80\""
  #   echo "  websecure:"
  #   echo "    address: \":443\""
  #   echo ""
  #   echo "providers:"
  #   echo "  file:"
  #   echo "    directory: /dynamic"
  #   echo "    watch: true"
  #   echo ""
  #   echo "api:"
  #   echo "  dashboard: true"
  #   echo ""
  #   echo "# Development Environment: Static Certificates"
  #   echo "certificatesResolvers:"
  #   echo "  default:"
  #   echo "    static:"
  #   echo "      certificates:"

  #   for host in "${UNIQUE_HOSTS[@]}"; do
  #     echo "        - certFile: \"/certs/${host}.pem\""
  #     echo "          keyFile: \"/certs/${host}-key.pem\""
  #   done
  # } > .traefik/traefik.yml

  # Create email templates
#   echo_info "Creating email templates..."
#   cat <<EOF > emails/confirm.html
# <!-- emails/confirm.html -->

# <!DOCTYPE html>
# <html>
# <head>
#   <title>Confirm Your Email</title>
# </head>
# <body>
#   <h1>Welcome!</h1>
#   <p>Please confirm your email by clicking the link below:</p>
#   <!-- Confirmation link goes here -->
# </body>
# </html>
# EOF

#   cat <<EOF > emails/magic-link.html
# <!-- emails/magic-link.html -->

# <!DOCTYPE html>
# <html>
# <head>
#   <title>Your Magic Link</title>
# </head>
# <body>
#   <h1>Hello!</h1>
#   <p>Click the link below to sign in:</p>
#   <!-- Magic link goes here -->
# </body>
# </html>
# EOF

#   cat <<EOF > emails/passwordless.html
# <!-- emails/passwordless.html -->

# <!DOCTYPE html>
# <html>
# <head>
#   <title>Passwordless Sign-In</title>
# </head>
# <body>
#   <h1>Welcome!</h1>
#   <p>Use the link below to sign in without a password:</p>
#   <!-- Passwordless sign-in link goes here -->
# </body>
# </html>
# EOF

#   cat <<EOF > emails/reset-password.html
# <!-- emails/reset-password.html -->

# <!DOCTYPE html>
# <html>
# <head>
#   <title>Reset Your Password</title>
# </head>
# <body>
#   <h1>Password Reset</h1>
#   <p>Click the link below to reset your password:</p>
#   <!-- Reset password link goes here -->
# </body>
# </html>
# EOF

  # Create seed SQL files
#   echo_info "Creating seed SQL files..."
#   cat <<EOF > seeds/0001_seed.sql
# -- seeds/0001_seed.sql

# -- Example: Additional seed data
# -- INSERT INTO products (name, price) VALUES ('Sample Product', 19.99);
# EOF

  # Create metadata tables.yaml
#   echo_info "Creating metadata/tables.yaml..."
#   cat <<EOF > metadata/tables.yaml
# # metadata/tables.yaml

# # Example: Define tables metadata
# tables:
#   - name: users
#     columns:
#       - name: id
#         type: uuid
#         default: gen_random_uuid()
#       - name: name
#         type: text
#       - name: email
#         type: text
#         unique: true
# EOF

#   # Create example service scripts
#   echo_info "Creating example service scripts..."

#   cat <<EOF > services/example.js
# // services/example.js

# const axios = require('axios');

# // Function to fetch weather data
# async function fetchWeather() {
#   try {
#     const response = await axios.get('https://api.weatherapi.com/v1/current.json?key=YOUR_API_KEY&q=London');
#     console.log('Weather Data:', response.data);
#   } catch (error) {
#     console.error('Error fetching weather data:', error);
#   }
# }

# // Fetch weather data every hour
# setInterval(fetchWeather, 60 * 60 * 1000);

# // Initial fetch
# fetchWeather();
# EOF

#   cat <<EOF > services/example.py
# # services/example.py

# import requests
# import time

# def fetch_weather():
#     try:
#         response = requests.get('https://api.weatherapi.com/v1/current.json', params={
#             'key': 'YOUR_API_KEY',
#             'q': 'London'
#         })
#         data = response.json()
#         print('Weather Data:', data)
#     except Exception as e:
#         print('Error fetching weather data:', e)

# // Fetch weather data every hour
# while True:
#     fetch_weather()
#     time.sleep(60 * 60)  # Sleep for one hour
# EOF

#   # Initialize the functions directory with a sample Node.js function
#   echo_info "Creating example function content..."

#   cd functions

#   # Create package.json
#   cat <<EOF > package.json
# {
#   "name": "nproj-functions",
#   "version": "1.0.0",
#   "description": "Nself Functions",
#   "main": "hello.js",
#   "scripts": {
#     "start": "node hello.js"
#   },
#   "dependencies": {
#     "axios": "^1.3.6"
#   },
#   "author": "",
#   "license": "ISC"
# }
# EOF

#   # Create package-lock.json
#   cat <<'EOF' > package-lock.json
# {}
# EOF

#   # Create a sample hello world function
#   cat <<EOF > hello.js
# // functions/hello.js

# const axios = require('axios');

# exports.handler = async (event, context) => {
#   return {
#     statusCode: 200,
#     body: JSON.stringify({ message: 'Hello, World!' }),
#   };
# };
# EOF

#   cd ..

  # Create README.md
  echo_info "Creating README.md..."
  cat <<EOF > README_NSELF.md
# Nhost Self-hosted Project

This project is a self-hosted instance of Nhost, providing GraphQL APIs, authentication, storage, and more.

## Getting Started

1. **Initialize the project:**
   \`\`\`bash
   nself init
   \`\`\`

2. **Configure environment variables:**
   Edit the \`.env.dev\` file with your settings, ensuring the \`HOSTS\` variable is correctly set. For example:
   \`\`\`env
   HOSTS=dashboard.nproj.run,graphql.nproj.run,auth.nproj.run
   # Add more hosts as needed, separated by commas
   \`\`\`

3. **Start the services:**
   \`\`\`bash
   nself up
   \`\`\`

4. ** Stop the service:**
   \`\`\`bash
   nself down
   \`\`\`
## Project Structure

- \`.env.dev or .env\`: Environment configuration file.
- \`emails/\`: Email templates.
- \`functions/\`: Serverless functions.
- \`services/\`: User-created services and scripts.
- \`docker-compose.yml\`: Docker Compose configuration file.

## Docker Compose

The \`docker-compose.yml\` file is generated based on the environment settings and starts all necessary services.

## Additional Resources

- [Nhost Documentation](https://docs.nhost.io/)
- [Hasura Documentation](https://hasura.io/docs/)
EOF
}

# ----------------------------
# Execute the script
# ----------------------------

create_directories_and_files
