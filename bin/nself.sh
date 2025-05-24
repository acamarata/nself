#!/bin/bash

# nself.sh - Main CLI tool for managing self-hosted Nhost stack

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
UNITY_DIR="$SCRIPT_DIR/../unity"

# ----------------------------
# Variables
# ----------------------------
VERSION_FILE="$SCRIPT_DIR/VERSION"
REPO_RAW_URL="https://raw.githubusercontent.com/acamarata/nself/main/bin"
LOCAL_VERSION=""
LATEST_VERSION=""

# Helper script paths
NSELF_INIT_SCRIPT="$SCRIPT_DIR/nself-init.sh"
NSELF_YAML_SCRIPT="$SCRIPT_DIR/nself-yaml.sh"
NSELF_SEED_SCRIPT="$SCRIPT_DIR/nself-seed.sh"

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

# Function to check if a command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Function to read local version
read_local_version() {
  if [ -f "$VERSION_FILE" ]; then
    LOCAL_VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
  else
    echo_error "VERSION file not found in $SCRIPT_DIR."
    exit 1
  fi
}

# Function to fetch latest version from GitHub
fetch_latest_version() {
  # Attempt to fetch the latest VERSION file
  LATEST_VERSION=$(curl -fsSL "$REPO_RAW_URL/VERSION" 2>/dev/null || echo "")
  # Remove any whitespace
  LATEST_VERSION=$(echo "$LATEST_VERSION" | tr -d '[:space:]')
}

# Function to compare versions
# Returns 0 if latest > local, 1 otherwise
is_newer_version() {
  # Remove leading 'v' if present
  local_ver=${LOCAL_VERSION#v}
  latest_ver=${LATEST_VERSION#v}

  # Convert versions to arrays
  IFS='.' read -r -a local_parts <<< "$local_ver"
  IFS='.' read -r -a latest_parts <<< "$latest_ver"

  # Compare each part
  for i in 0 1 2; do
    local_part=${local_parts[i]:-0}
    latest_part=${latest_parts[i]:-0}
    if (( 10#$latest_part > 10#$local_part )); then
      return 0
    elif (( 10#$latest_part < 10#$local_part )); then
      return 1
    fi
  done

  return 1
}

# Function to check for updates and notify user
check_for_updates() {
  read_local_version
  fetch_latest_version

  if [ -z "$LATEST_VERSION" ]; then
    # Failed to fetch latest version; do not notify
    return
  fi

  if is_newer_version; then
    echo_info "A new version of nself is available: $LATEST_VERSION. You can run 'nself update' to upgrade."
  fi
}

# Function to update nself
update_nself() {
  echo "disable update, because local version."
  # echo_info "Checking for updates..."

  # read_local_version
  # fetch_latest_version

  # if [ -z "$LATEST_VERSION" ]; then
  #   echo_error "Failed to fetch the latest version information. Please check your network connection."
  #   exit 1
  # fi

  # if is_newer_version; then
  #   echo_info "Updating nself from version $LOCAL_VERSION to $LATEST_VERSION..."
  #   # Download the latest nself.sh
  #   if curl -fsSL "$REPO_RAW_URL/nself.sh" -o "$SCRIPT_DIR/nself.sh"; then
  #     chmod +x "$SCRIPT_DIR/nself.sh"
  #     # Download the latest VERSION file
  #     if curl -fsSL "$REPO_RAW_URL/VERSION" -o "$VERSION_FILE"; then
  #       echo_info "nself updated successfully to version $LATEST_VERSION!"
  #     else
  #       echo_error "Failed to download the latest VERSION file."
  #       exit 1
  #     fi
  #   else
  #     echo_error "Failed to download the latest nself.sh script."
  #     exit 1
  #   fi
  # else
  #   echo_info "nself is already up to date."
  # fi
}

# Function to load environment variables from .env.dev or .env
load_env() {
  if [ -f ".env" ]; then
    ENV_FILE=".env"
  elif [ -f ".env.dev" ]; then
    ENV_FILE=".env.dev"
  else
    echo_error "No .env or .env.dev file found."
    exit 1
  fi

  # Export variables from the env file
  set -o allexport
  source "$ENV_FILE"
  set +o allexport

  # Ensure PROJECT_NAME is set
  PROJECT_NAME=${PROJECT_NAME:-project}
  export PROJECT_NAME

  # Ensure COMPOSE_PROJECT_NAME is set and exported
  COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-nproj}
  export COMPOSE_PROJECT_NAME
}

# Function to initialize the project
init_project() {
  if [ -f ".env" ] || [ -f ".env.dev" ]; then
    echo_error ".env or .env.dev already exists in this directory."
    exit 1
  fi

  if [ ! -d "$UNITY_DIR" ]; then
    echo "Error: 'unity' starter directory not found in $UNITY_DIR."
    exit 1
  fi

  if [ ! -f "$UNITY_DIR/.env.example" ]; then
    echo_error ".env.example not found in $UNITY_DIR."
    exit 1
  fi

  if [ ! -f "$NSELF_INIT_SCRIPT" ]; then
    echo_error "nself-init.sh not found in $SCRIPT_DIR."
    exit 1
  fi

  echo_info "Initializing project..."

  bash "$SCRIPT_DIR/nself-unity.sh"
  

  # Copy .env.example to .env.dev
  echo_info "Creating example .env..."
  cp "$UNITY_DIR/.env.example" ".env"

  
  # Run the init helper script
  bash "$NSELF_INIT_SCRIPT"

  echo_info "✓ Initialization complete."
  echo_info "Please modify your .env or .env.dev file..."
  echo_info "Then run 'nself up' to start services!"
}

# Function to generate docker-compose.yml from template
generate_docker_compose() {
  if [ ! -f "$NSELF_YAML_SCRIPT" ]; then
    echo_error "nself-yaml.sh not found in $SCRIPT_DIR."
    exit 1
  fi

  bash "$SCRIPT_DIR/nself-unity.sh"
  # echo_info "Generating docker-compose.yml..."
  # bash "$NSELF_YAML_SCRIPT"
}

# Function to start the services
start_services() {
  if [ ! -f "docker-compose.yml" ]; then
    generate_docker_compose
  fi

  echo_info "Starting services with docker-compose..."
  echo  "docker-compose --env-file $ENV_FILE up -d"

  rm -f .nself/traefik/htpasswd/.htpasswd
  printf "admin:$(openssl passwd -apr1 ${METRICS_PASSWORD})\n" >> .nself/traefik/htpasswd/.htpasswd

  NETWORK_NAME="unity"

  # Check if the network exists
  if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
      echo "⚠️ Docker network '$NETWORK_NAME' not found. Creating it now..."
      docker network create "$NETWORK_NAME"
      echo "✅ Docker network '$NETWORK_NAME' has been created."
  else
      echo "✅ Docker network '$NETWORK_NAME' already exists."
  fi


  #  Create Volume
  VOLUMES=(
    "$FUNCTION_VOLUME"
    "$PROJECT_DATA_VOLUME"
    "$INIT_DB_VOLUME"
    "$PROJECT_DB_VOLUME"
    "$MAILHOG_VOLUME"
    "$MINIO_VOLUME"
    "$MINIO1_VOLUME"
    "$MINIO2_VOLUME"
    "$MINIO3_VOLUME"
    "$MINIO4_VOLUME"
    "$STORAGE_VOLUME"
  )

  # Create each volume
  for volume in "${VOLUMES[@]}"; do
      if [ -n "$volume" ]; then
          docker volume create "$volume"
          echo "✅ Created volume: $volume"
      fi
  done

  if docker-compose --env-file "$ENV_FILE" up -d; then
    bash "$SCRIPT_DIR/nself-succ.sh"
  else
    echo_error "Failed to start Docker services."
    exit 1
  fi
}

# Function to stop the services
stop_services() {
  if [ ! -f "docker-compose.yml" ]; then
    echo_error "docker-compose.yml not found."
    exit 1
  fi

  echo_info "Stopping services with docker-compose..."
  if docker-compose --env-file "$ENV_FILE" down; then
    echo_info "Services have been stopped."
  else
    echo_error "Failed to stop Docker services."
    exit 1
  fi
}

# Function to reset the environment
reset_environment() {
  # Ensure environment variables are loaded
  if [ -f ".env" ] || [ -f ".env.dev" ]; then
    load_env
  else
    echo_error "No environment configuration found. Please ensure .env or .env.dev exists."
    exit 1
  fi

  # Stop and remove Docker containers and named volumes
  echo_info "Removing Docker containers and volumes..."
  docker-compose --env-file "$ENV_FILE" down -v || true

  docker system prune -af
  # Remove Docker volumes (optional, if any leftover)
  echo_info "Removing any remaining unused Docker volumes..."
  # docker volume prune -f || true
  docker volume rm $(docker volume ls -q | grep '$PROJECT_NAME')

  # Delete docker-compose.yml
  if [ -f "docker-compose.yml" ]; then
    echo_info "Deleting docker-compose.yml..."
    rm -f "docker-compose.yml"
  else
    echo_info "docker-compose.yml does not exist. Skipping deletion."
  fi

  # Delete seeds/initdb.sql
  if [ -f "seeds/initdb.sql" ]; then
    echo_info "Deleting seeds/initdb.sql..."
    rm -f "seeds/initdb.sql"
  else
    echo_info "seeds/initdb.sql does not exist. Skipping deletion."
  fi

  echo_info "Environment reset. Run 'nself up' to start services again."
}

# Function to display usage
usage() {
  # Define color codes
  C1='\033[1;94m'
  C2='\033[1;36m'
  RESET='\033[0m'
  
  echo ""
  echo -e "${C1}Nself — Nhost Self-hosted CLI${RESET}"
  echo ""
  echo -e "${C2}Usage: nself [command]${RESET}"
  echo ""
  echo -e "${C2}Commands:${RESET}"
  echo -e "${C2}  init         Initialize project by creating starter folders and files${RESET}"
  echo -e "${C2}  up           Start Docker services, building a compose file if needed${RESET}"
  echo -e "${C2}  down         Stop the Docker services${RESET}"
  # echo -e "${C2}  reset        Stops and deletes containers, volumes, yml, and seed sql${RESET}"
  #echo -e "${C2}  delete       Stops and deletes containers, volumes, and initial files${RESET}"
  # echo -e "${C2}  update       Update nself to the latest version if newer is available${RESET}"
  echo -e "${C2}  help         Display this help message${RESET}"
  echo -e "${C2}  -h, --help   Display this help message${RESET}"
  echo -e "${C2}  --version, -v Show the current version of nself${RESET}"
  echo ""
}

# Function to display version
show_version() {
  if [ -f "$VERSION_FILE" ]; then
    VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
    echo "nself version $VERSION"
  else
    echo_error "VERSION file not found in $SCRIPT_DIR."
    exit 1
  fi
}

# ----------------------------
# Main Logic
# ----------------------------
COMMAND="$1"

case "$COMMAND" in
  init)
    check_for_updates
    init_project
    ;;
  up)
    # Check for required environment files
    if [ ! -f ".env" ] && [ ! -f ".env.dev" ] && [ ! -f "$UNITY_DIR/.env.example" ]; then
      echo_error "No environment configuration found. Please run 'nself init' to initialize your project."
      exit 1
    elif [ -f "$UNITY_DIR/.env.example" ] && [ ! -f ".env" ] && [ ! -f ".env.dev" ]; then
      echo_error ".env.example exists but no .env or .env.dev file found. Please create and configure one of them."
      echo_info "You can run 'nself init' to copy .env.example to .env.dev."
      exit 1
    fi

    check_for_updates

    if [ -f ".env" ] || [ -f ".env.dev" ]; then
      load_env
      if [ ! -f "docker-compose.yml" ]; then
        generate_docker_compose
      fi
      start_services
    else
      echo_error "Environment configuration is missing. Please run 'nself init' to initialize your project."
      exit 1
    fi
    ;;
  down)
    # Check for required environment files
    if [ ! -f ".env" ] && [ ! -f ".env.dev" ]; then
      echo_error "No environment configuration found. Please ensure .env or .env.dev exists."
      exit 1
    fi

    load_env
    stop_services
    ;;
  reset)
    # Check for required environment files
    if [ ! -f ".env" ] && [ ! -f ".env.dev" ]; then
      echo_error "No environment configuration found. Please ensure .env or .env.dev exists."
      exit 1
    fi

    load_env
    reset_environment
    ;;
  # update)
  #   update_nself
    # ;;
  help|-h)
    usage
    ;;
  --version|-v)
    show_version
    ;;
  "")
    usage
    ;;
  *)
    echo_error "Unknown command: $COMMAND"
    usage
    exit 1
    ;;
esac
