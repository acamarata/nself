#!/bin/bash

# nself-succ.sh - Display success message with accessible routes

set -e

# ----------------------------
# Helper Functions
# ----------------------------

# Function to print colored text
color_echo() {
  local color="$1"
  local text="$2"
  case "$color" in
    green)
      echo -e "\033[1;32m${text}\033[0m"
      ;;
    blue)
      echo -e "\033[1;94m${text}\033[0m"
      ;;
    cyan)
      echo -e "\033[1;36m${text}\033[0m"
      ;;
    *)
      echo "$text"
      ;;
  esac
}

# Function to convert string to lowercase
to_lowercase() {
  echo "$1" | tr '[:upper:]' '[:lower:]'
}

# Function to display a success message with accessible routes
success_message() {
  echo ""
  color_echo blue "Nself CLI has successfully set up your project!"
  color_echo blue "You can access services at the following URLs:"
  echo ""

  # Predefined services
  predefined_services=("Console" "GraphQL" "Auth" "Storage" "Functions" "Dashboard" "Traefik" "MailHog")
  predefined_domains=(
    "${HASURA_CONSOLE_ROUTE}"
    "${HASURA_GRAPHQL_ROUTE}"
    "${AUTH_ROUTE}"
    "${STORAGE_ROUTE}"
    "${FUNCTIONS_ROUTE}"
    "${DASHBOARD_ROUTE}"
    "${TRAEFIK_ROUTE}"
    "${MAILHOG_ROUTE}"
    # "${TOP_DOMAIN}"
    # "${MINIO_ROUTE}"
  )

  for i in "${!predefined_services[@]}"; do
    service="${predefined_services[i]}"
    domain="${predefined_domains[i]}"

    if [ "$service" == "GraphQL" ]; then
      # Collect remote schema URLs from environment variables
      remote_schemas=()
      for var in $(compgen -v | grep '^HASURA_REMOTE_SCHEMA_[0-9]\+_URL$'); do
        remote_schemas+=("${!var}")
      done

      # Prepare GraphQL URLs
      graphql_urls="https://${domain}"

      # for rs in "${remote_schemas[@]}"; do
      #   graphql_urls+="\n             https://${rs}"
      # done

      # Print GraphQL service with multiple URLs, properly aligned
      printf "  \033[1;36m%-10s\033[0m \033[1;32m%s\033[0m\n" "$service:" "$(echo -e "$graphql_urls")"
    else
      # Print service with its URL
      printf "  \033[1;36m%-10s\033[0m \033[1;32mhttps://%s\n" "$service:" "$domain"
    fi
  done

  # Optional services
  optional_services=("Haraka" "Redis" "Meilisearch" "Prometheus" "Grafana" "SMTP")
  optional_routes=("$HARAKA_ROUTE" "$REDIS_ROUTE" "$MEILISEARCH_ROUTE" "$PROMETHEUS_ROUTE" "$GRAFANA_ROUTE" "$SMTP_ROUTE")

  for i in "${!optional_services[@]}"; do
    service="${optional_services[i]}"
    route="${optional_routes[i]}"
    if [ -n "$route" ]; then
      service_lower=$(to_lowercase "$service")
      printf "  \033[1;36m%-10s\033[0m \033[1;32mhttps://%s\n" "$service:" "$route" "$service_lower"
      echo ""
    fi
  done

  # OTHER_ROUTES
  # if [ -n "$OTHER_ROUTES" ]; then
  #   echo ""
  #   color_echo blue "Additional Routes:"
  #   while IFS= read -r line; do
  #     # Trim leading/trailing whitespace
  #     line=$(echo "$line" | xargs)
  #     # Skip empty lines and lines starting with #
  #     if [[ -z "$line" || "$line" == \#* ]]; then
  #       continue
  #     fi

  #     DOMAIN=$(echo "$line" | cut -d'=' -f1)
  #     TARGET=$(echo "$line" | cut -d'=' -f2)

  #     if [ -n "$DOMAIN" ] && [ -n "$TARGET" ]; then
  #       color_echo cyan "  - \033[1;32m${DOMAIN}\033[0m \033[1;36mor \033[1;32m${TARGET}\033[0m"
  #     fi
  #   done <<< "$OTHER_ROUTES"

  #   echo ""
  #   #color_echo cyan "Alternatively, http://localhost/service-name"
  # fi

  echo "✅ Your application is accessible at: https://${FUNCTIONS_ROUTE}"
}

# ----------------------------
# Main Execution
# ----------------------------

success_message
