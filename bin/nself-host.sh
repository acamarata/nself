#!/bin/bash

# nself-host.sh - Configure Traefik routing and invoke certificate management

set -e

# ----------------------------
# Helper Functions
# ----------------------------

echo_info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

echo_error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

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

to_lowercase() {
  echo "$1" | tr '[:upper:]' '[:lower:]'
}

ends_with() {
  string="$1"
  suffix="$2"
  case "$string" in
    *"$suffix") return 0 ;;
    *) return 1 ;;
  esac
}

get_internal_port() {
  service_name="$1"
  case "$service_name" in
    console)
      echo "3000"
      ;;
    graphql)
      echo "8080"
      ;;
    auth)
      echo "4000"
      ;;
    storage)
      echo "9000"
      ;;
    functions)
      echo "1337"
      ;;
    dashboard)
      echo "3030"
      ;;
    traefik)
      echo "8080"
      ;;
    mailhog)
      echo "1025"
      ;;
    haraka)
      echo "25"
      ;;
    redis)
      echo "6379"
      ;;
    meilisearch)
      echo "7700"
      ;;
    prometheus)
      echo "9090"
      ;;
    grafana)
      echo "3000"
      ;;
    smtp)
      echo "587"
      ;;
    *)
      echo "80"  # Default port if service is not recognized
      ;;
  esac
}

# ----------------------------
# Main Logic
# ----------------------------

load_env

# Determine Environment
ENVIRONMENT="${ENV:-dev}"  # Default to development if ENV is not set

# Collect all hostnames
HOSTS=()

# Predefined services and their domains
predefined_services=("Console" "GraphQL" "Auth" "Storage" "Functions" "Dashboard" "Traefik" "MailHog")
predefined_domains=(
  "console.${PROJECT_NAME}.run"
  "graphql.${PROJECT_NAME}.run"
  "auth.${PROJECT_NAME}.run"
  "storage.${PROJECT_NAME}.run"
  "functions.${PROJECT_NAME}.run"
  "dashboard.${PROJECT_NAME}.run"
  "traefik.${PROJECT_NAME}.run"
  "mailhog.${PROJECT_NAME}.run"
)

for i in "${!predefined_services[@]}"; do
  HOSTS+=("${predefined_domains[i]}")
done

# Handle Optional Routes
OPTIONAL_ROUTES=(
  "MAILHOG_ROUTE"
  "HARAKA_ROUTE"
  "REDIS_ROUTE"
  "MEILISEARCH_ROUTE"
  "PROMETHEUS_ROUTE"
  "GRAFANA_ROUTE"
  "SMTP_ROUTE"
  # Add more optional routes here as needed
)

for ROUTE_VAR in "${OPTIONAL_ROUTES[@]}"; do
  ROUTE_VALUE=${!ROUTE_VAR}
  if [ -n "$ROUTE_VALUE" ]; then
    HOSTNAME=$(echo "$ROUTE_VALUE" | cut -d':' -f1)
    if ends_with "$HOSTNAME" ".${PROJECT_NAME}.run"; then
      FULL_DOMAIN="$HOSTNAME"
    else
      FULL_DOMAIN="${HOSTNAME}.${PROJECT_NAME}.run"
    fi
    HOSTS+=("$FULL_DOMAIN")
  fi
done

# Handle OTHER_ROUTES
if [ -n "${OTHER_ROUTES}" ]; then
  while IFS= read -r line; do
    line=$(echo "$line" | xargs)
    if [[ -z "$line" || "$line" == \#* ]]; then
      continue
    fi
    DOMAIN=$(echo "$line" | cut -d'=' -f1)
    HOSTS+=("$DOMAIN")
  done <<< "$OTHER_ROUTES"
fi

# Invoke certificate management script with hostnames
# bash "$(dirname "$0")/nself-cert.sh" "${HOSTS[@]}"

# Function to generate Traefik Dynamic Configuration
generate_dynamic_yml() {
  echo_info "Generating Traefik dynamic.yml file..."

  mkdir -p .traefik/dynamic

  DYNAMIC_YML=".traefik/dynamic/dynamic.yml"

  # Start with the base configuration
  cat <<EOF > "$DYNAMIC_YML"
http:
  middlewares:
    redirect-to-https:
      redirectScheme:
        scheme: https
        permanent: true
  routers:
    http_redirect:
      rule: "HostRegexp(\`{host:.+}\`)"
      entryPoints:
        - web
      middlewares:
        - redirect-to-https
      service: noop
  services:
    noop:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:80"
EOF

  # Predefined services and their corresponding Host routes
  for i in "${!predefined_services[@]}"; do
    service="${predefined_services[i]}"
    domain="${predefined_domains[i]}"
    service_lower=$(to_lowercase "$service")
    internal_port=$(get_internal_port "$service_lower")

    if [ "$service" == "GraphQL" ]; then
      # Collect remote schema URLs from environment variables
      remote_schemas=()
      for var in $(compgen -v | grep '^HASURA_REMOTE_SCHEMA_[0-9]\+_URL$'); do
        remote_schemas+=("${!var}")
      done

      # Build Host rules for GraphQL (primary + remote schemas)
      host_rules="Host(\`${domain}\`)"
      for rs in "${remote_schemas[@]}"; do
        if ends_with "$rs" ".${PROJECT_NAME}.run"; then
          host_rules+=" || Host(\`${rs}\`)"
        else
          host_rules+=" || Host(\`${rs}.${PROJECT_NAME}.run\`)"
        fi
      done

      if [ "$ENVIRONMENT" == "prod" ]; then
        # Production: Use Let's Encrypt
        cat <<EOF >> "$DYNAMIC_YML"

  routers:
    graphql_router:
      rule: "${host_rules}"
      service: graphql_service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

  services:
    graphql_service:
      loadBalancer:
        servers:
          - url: "http://hasura:${internal_port}"
EOF
      else
        # Development: Use local certificates
        cat <<EOF >> "$DYNAMIC_YML"

  routers:
    graphql_router:
      rule: "${host_rules}"
      service: graphql_service
      entryPoints:
        - websecure
      tls:
        certResolver: default
        domains:
          - main: "${domain}"
            sans:
              - "*.${PROJECT_NAME}.run"

  services:
    graphql_service:
      loadBalancer:
        servers:
          - url: "http://hasura:${internal_port}"
EOF
      fi
    else
      if [ "$ENVIRONMENT" == "prod" ]; then
        # Production: Use Let's Encrypt
        cat <<EOF >> "$DYNAMIC_YML"

  routers:
    ${service_lower}_router:
      rule: "Host(\`${domain}\`) || PathPrefix(\`/${service_lower}\`)"
      service: ${service_lower}_service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

  services:
    ${service_lower}_service:
      loadBalancer:
        servers:
          - url: "http://${service_lower}:${internal_port}"
EOF
      else
        # Development: Use local certificates
        cat <<EOF >> "$DYNAMIC_YML"

  routers:
    ${service_lower}_router:
      rule: "Host(\`${domain}\`) || PathPrefix(\`/${service_lower}\`)"
      service: ${service_lower}_service
      entryPoints:
        - websecure
      tls:
        certResolver: default
        domains:
          - main: "${domain}"
            sans:
              - "*.${PROJECT_NAME}.run"

  services:
    ${service_lower}_service:
      loadBalancer:
        servers:
          - url: "http://${service_lower}:${internal_port}"
EOF
      fi
    fi
  done

  # Handle Optional Routes
  for ROUTE_VAR in "${OPTIONAL_ROUTES[@]}"; do
    ROUTE_VALUE=${!ROUTE_VAR}
    if [ -n "$ROUTE_VALUE" ]; then
      SERVICE_NAME=$(echo "$ROUTE_VAR" | sed 's/_ROUTE$//' | tr '[:upper:]' '[:lower:]')
      HOSTNAME=$(echo "$ROUTE_VALUE" | cut -d':' -f1)
      PORT=$(echo "$ROUTE_VALUE" | cut -d':' -f2)

      if ends_with "$HOSTNAME" ".${PROJECT_NAME}.run"; then
        FULL_DOMAIN="$HOSTNAME"
      else
        FULL_DOMAIN="${HOSTNAME}.${PROJECT_NAME}.run"
      fi

      if [ "$ENVIRONMENT" == "prod" ]; then
        # Production: Use Let's Encrypt
        cat <<EOF >> "$DYNAMIC_YML"

  routers:
    ${SERVICE_NAME}_router:
      rule: "Host(\`${FULL_DOMAIN}\`) || PathPrefix(\`/${SERVICE_NAME}\`)"
      service: ${SERVICE_NAME}_service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

  services:
    ${SERVICE_NAME}_service:
      loadBalancer:
        servers:
          - url: "http://${SERVICE_NAME}:${PORT}"
EOF
      else
        # Development: Use local certificates
        cat <<EOF >> "$DYNAMIC_YML"

  routers:
    ${SERVICE_NAME}_router:
      rule: "Host(\`${FULL_DOMAIN}\`) || PathPrefix(\`/${SERVICE_NAME}\`)"
      service: ${SERVICE_NAME}_service
      entryPoints:
        - websecure
      tls:
        certResolver: default
        domains:
          - main: "${FULL_DOMAIN}"
            sans:
              - "*.${PROJECT_NAME}.run"

  services:
    ${SERVICE_NAME}_service:
      loadBalancer:
        servers:
          - url: "http://${SERVICE_NAME}:${PORT}"
EOF
      fi
    fi
  done

  # Handle OTHER_ROUTES
  if [ -n "${OTHER_ROUTES}" ]; then
    while IFS= read -r line; do
      line=$(echo "$line" | xargs)
      if [[ -z "$line" || "$line" == \#* ]]; then
        continue
      fi

      DOMAIN=$(echo "$line" | cut -d'=' -f1)
      TARGET=$(echo "$line" | cut -d'=' -f2)
      SERVICE_NAME=$(echo "$TARGET" | cut -d':' -f1 | tr '[:upper:]' '[:lower:]')
      HOSTNAME=$(echo "$TARGET" | cut -d':' -f1)
      PORT=$(echo "$TARGET" | cut -d':' -f2)

      if ends_with "$HOSTNAME" ".${PROJECT_NAME}.run"; then
        FULL_DOMAIN="$HOSTNAME"
      else
        FULL_DOMAIN="${HOSTNAME}.${PROJECT_NAME}.run"
      fi

      if [ "$ENVIRONMENT" == "prod" ]; then
        # Production: Use Let's Encrypt
        cat <<EOF >> "$DYNAMIC_YML"

  routers:
    ${SERVICE_NAME}_router:
      rule: "Host(\`${DOMAIN}\`) || PathPrefix(\`/${SERVICE_NAME}\`)"
      service: ${SERVICE_NAME}_service
      entryPoints:
        - websecure
      tls:
        certResolver: letsencrypt

  services:
    ${SERVICE_NAME}_service:
      loadBalancer:
        servers:
          - url: "http://${SERVICE_NAME}:${PORT}"
EOF
      else
        # Development: Use local certificates
        cat <<EOF >> "$DYNAMIC_YML"

  routers:
    ${SERVICE_NAME}_router:
      rule: "Host(\`${DOMAIN}\`) || PathPrefix(\`/${SERVICE_NAME}\`)"
      service: ${SERVICE_NAME}_service
      entryPoints:
        - websecure
      tls:
        certResolver: default
        domains:
          - main: "${DOMAIN}"
            sans:
              - "*.${PROJECT_NAME}.run"

  services:
    ${SERVICE_NAME}_service:
      loadBalancer:
        servers:
          - url: "http://${SERVICE_NAME}:${PORT}"
EOF
      fi
    done <<< "$OTHER_ROUTES"
  fi

  echo_info "dynamic.yml generated."
}

# Function to ensure host entries in /etc/hosts
ensure_host_entries() {
  predefined_services=("Console" "GraphQL" "Auth" "Storage" "Functions" "Dashboard" "Traefik" "MailHog")
  predefined_domains=(
    "console.${PROJECT_NAME}.run"
    "graphql.${PROJECT_NAME}.run"
    "auth.${PROJECT_NAME}.run"
    "storage.${PROJECT_NAME}.run"
    "functions.${PROJECT_NAME}.run"
    "dashboard.${PROJECT_NAME}.run"
    "traefik.${PROJECT_NAME}.run"
    "mailhog.${PROJECT_NAME}.run"
  )

  # Collect entries to add
  ENTRIES_TO_ADD=()

  for i in "${!predefined_services[@]}"; do
    domain="${predefined_domains[i]}"
    if ! grep -q "127.0.0.1 ${domain}" /etc/hosts; then
      ENTRIES_TO_ADD+=("127.0.0.1 ${domain}")
    fi
  done

  # Handle Additional Routes
  if [ -n "$OTHER_ROUTES" ]; then
    while IFS= read -r line; do
      line=$(echo "$line" | xargs)
      if [[ -z "$line" || "$line" == \#* ]]; then
        continue
      fi
      DOMAIN=$(echo "$line" | cut -d'=' -f1)
      if ! grep -q "127.0.0.1 ${DOMAIN}" /etc/hosts; then
        ENTRIES_TO_ADD+=("127.0.0.1 ${DOMAIN}")
      fi
    done <<< "$OTHER_ROUTES"
  fi

  # Handle Optional Routes
  OPTIONAL_ROUTES=(
    "MAILHOG_ROUTE"
    "HARAKA_ROUTE"
    "REDIS_ROUTE"
    "MEILISEARCH_ROUTE"
    "PROMETHEUS_ROUTE"
    "GRAFANA_ROUTE"
    "SMTP_ROUTE"
    # Add more optional routes here as needed
  )

  for ROUTE_VAR in "${OPTIONAL_ROUTES[@]}"; do
    ROUTE_VALUE=${!ROUTE_VAR}
    if [ -n "$ROUTE_VALUE" ]; then
      HOSTNAME=$(echo "$ROUTE_VALUE" | cut -d':' -f1)
      DOMAIN="${HOSTNAME}.${PROJECT_NAME}.run"
      if ! grep -q "127.0.0.1 ${DOMAIN}" /etc/hosts; then
        ENTRIES_TO_ADD+=("127.0.0.1 ${DOMAIN}")
      fi
    fi
  done

  # Add entries if any
  if [ ${#ENTRIES_TO_ADD[@]} -gt 0 ]; then
    echo_info "Adding host entries to /etc/hosts..."
    printf "%s\n" "${ENTRIES_TO_ADD[@]}" | sudo tee -a /etc/hosts >/dev/null
    echo_info "Host entries added."
  else
    echo_info "No new host entries to add."
  fi
}

# Function to setup HTTP to HTTPS redirection
setup_redirect() {
  echo_info "Setting up HTTP to HTTPS redirection..."
  # The redirect is already handled in dynamic.yml with the redirect-to-https middleware and http_redirect router
}

# Function to adjust Traefik static configuration for development
adjust_static_traefik_yml() {
  TRAEFIK_YML=".traefik/traefik.yml"

  if [ "$ENVIRONMENT" == "dev" ]; then
    echo_info "Configuring Traefik for development environment to use local certificates."

    # Append TLS settings to traefik.yml if not present
    if ! grep -q "certificatesResolvers:" "$TRAEFIK_YML"; then
      cat <<EOF >> "$TRAEFIK_YML"

certificatesResolvers:
  default:
    static:
      certificates:
        - certFile: "/certs/${HOSTS[0]}.pem"
          keyFile: "/certs/${HOSTS[0]}-key.pem"
EOF
    fi
  fi
}

# Generate Traefik dynamic configuration
generate_dynamic_yml

# Adjust Traefik static configuration based on environment
adjust_static_traefik_yml

# Ensure host entries are correctly mapped
ensure_host_entries

# Setup HTTP to HTTPS redirection
setup_redirect

echo_info "Traefik dynamic configuration complete."
