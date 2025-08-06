#!/bin/bash

# install.sh - Installation script for nself CLI

set -e

# Variables
NSELF_DIR="$HOME/.nself"
BIN_DIR="$NSELF_DIR/bin"
TEMPLATES_DIR="$NSELF_DIR/bin/templates"
CERTS_DIR="$NSELF_DIR/bin/certs"
REPO_URL="https://github.com/acamarata/nself"
REPO_RAW_URL="https://raw.githubusercontent.com/acamarata/nself/main"

# Color functions
echo_info() {
  echo -e "\033[1;34m[INFO]\033[0m $1"
}

echo_success() {
  echo -e "\033[1;32m[SUCCESS]\033[0m $1"
}

echo_error() {
  echo -e "\033[1;31m[ERROR]\033[0m $1" >&2
}

# Progress spinner
show_spinner() {
  local pid=$1
  local message=$2
  local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  local i=0
  
  printf "\033[1;36m%s\033[0m" "$message"
  
  while kill -0 $pid 2>/dev/null; do
    i=$(( (i+1) %10 ))
    printf "\r\033[1;36m%s %s\033[0m" "$message" "${spin:$i:1}"
    sleep 0.1
  done
  
  wait $pid
  local result=$?
  
  if [ $result -eq 0 ]; then
    printf "\r\033[1;32m✓\033[0m %s\n" "$message"
  else
    printf "\r\033[1;31m✗\033[0m %s\n" "$message"
  fi
  
  return $result
}

# Check if command exists
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Install curl if not present
install_curl() {
  echo_info "Installing curl..."
  
  if command -v apt >/dev/null 2>&1; then
    sudo apt update && sudo apt install -y curl
  elif command -v yum >/dev/null 2>&1; then
    sudo yum install -y curl
  elif command -v dnf >/dev/null 2>&1; then
    sudo dnf install -y curl
  elif command -v pacman >/dev/null 2>&1; then
    sudo pacman -Sy --noconfirm curl
  elif command -v zypper >/dev/null 2>&1; then
    sudo zypper install -y curl
  elif command -v brew >/dev/null 2>&1; then
    brew install curl
  else
    echo_error "Package manager not supported. Please install curl manually."
    exit 1
  fi
  
  echo_success "Curl installed successfully!"
}

# Install Docker
install_docker() {
  echo_info "Docker not found. Installing Docker..."
  
  OS="$(uname -s)"
  
  if [ "$OS" = "Linux" ]; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    rm get-docker.sh
    
    # Add current user to docker group
    sudo usermod -aG docker "$USER"
    echo_info "Added $USER to docker group. You may need to log out and back in."
  elif [ "$OS" = "Darwin" ]; then
    echo_error "Please install Docker Desktop from https://www.docker.com/products/docker-desktop"
    echo_error "Then rerun this installation script."
    exit 1
  else
    echo_error "Unsupported OS: $OS"
    exit 1
  fi
  
  echo_success "Docker installed successfully!"
}

# Install Docker Compose
install_docker_compose() {
  echo_info "Installing Docker Compose..."
  
  # Try to install via Docker plugin first (preferred method)
  if command_exists docker; then
    if docker compose version >/dev/null 2>&1; then
      echo_success "Docker Compose plugin already installed!"
      return 0
    fi
  fi
  
  # Install standalone docker-compose
  OS="$(uname -s)"
  ARCH="$(uname -m)"
  
  if [ "$OS" = "Linux" ] || [ "$OS" = "Darwin" ]; then
    COMPOSE_URL="https://github.com/docker/compose/releases/latest/download/docker-compose-${OS}-${ARCH}"
    
    echo_info "Downloading Docker Compose from $COMPOSE_URL..."
    sudo curl -L "$COMPOSE_URL" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    
    echo_success "Docker Compose installed successfully!"
  else
    echo_error "Unsupported OS for Docker Compose: $OS"
    exit 1
  fi
}

# Download NSELF files
download_nself_files() {
  echo ""
  echo_info "📦 Installing nself CLI"
  echo ""
  
  # Create directories
  (
    mkdir -p "$BIN_DIR"
    mkdir -p "$TEMPLATES_DIR"
    mkdir -p "$CERTS_DIR"
  ) &
  show_spinner $! "  Creating directories"
  
  # List of files to download
  declare -a BIN_FILES=(
    "nself.sh"
    "build.sh"
    "compose.sh"
    "success.sh"
    "db.sh"
    "services.sh"
    "services-compose.sh"
    "services-compose-inline.sh"
    "update.sh"
    "VERSION"
  )
  
  # Create temporary directory for downloads
  TMP_DIR=$(mktemp -d)
  trap "rm -rf $TMP_DIR" EXIT
  
  # Download all files in background
  (
    # Download checksum file first (if available)
    CHECKSUM_URL="$REPO_RAW_URL/checksums.sha256"
    CHECKSUM_FILE="$TMP_DIR/checksums.sha256"
    curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_FILE" 2>/dev/null || true
    
    # Download bin files
    for file in "${BIN_FILES[@]}"; do
      TMP_FILE="$TMP_DIR/$file"
      if ! curl -fsSL "$REPO_RAW_URL/bin/$file" -o "$TMP_FILE" 2>/dev/null; then
        exit 1
      fi
      # Move to final location
      mv "$TMP_FILE" "$BIN_DIR/$file"
      chmod +x "$BIN_DIR/$file"
    done
  ) &
  
  show_spinner $! "  Downloading nself files"
  
  if [ $? -ne 0 ]; then
    echo_error "Failed to download files"
    exit 1
  fi
  
  # Download template files
  (curl -fsSL "$REPO_RAW_URL/bin/templates/.env.example" -o "$TEMPLATES_DIR/.env.example" 2>/dev/null) &
  show_spinner $! "  Downloading templates"
}

# Main installation
main() {
  echo
  echo_info "🚀 Starting NSELF installation..."
  echo
  
  # Check for curl
  if ! command_exists curl; then
    install_curl
  fi
  
  # Check for Docker
  if ! command_exists docker; then
    install_docker
  else
    echo_info "Docker is already installed."
  fi
  
  # Check for Docker Compose
  if ! docker compose version >/dev/null 2>&1 && ! command_exists docker-compose; then
    install_docker_compose
  else
    echo_info "Docker Compose is already installed."
  fi
  
  # Download NSELF files
  download_nself_files
  
  # Add to PATH
  if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo_info "Adding NSELF to PATH..."
    
    # Detect shell
    if [ -n "$ZSH_VERSION" ]; then
      SHELL_RC="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
      SHELL_RC="$HOME/.bashrc"
    else
      SHELL_RC="$HOME/.profile"
    fi
    
    # Add to PATH
    echo '' >> "$SHELL_RC"
    echo '# NSELF CLI' >> "$SHELL_RC"
    echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$SHELL_RC"
    
    export PATH="$BIN_DIR:$PATH"
    
    echo_info "Added to PATH in $SHELL_RC"
  fi
  
  # Create global symlink
  echo_info "Creating global nself command..."
  if sudo ln -sf "$BIN_DIR/nself.sh" /usr/local/bin/nself; then
    echo_success "Global nself command created!"
  else
    echo_error "Failed to create global command. You can still use nself from: $BIN_DIR/nself.sh"
  fi
  
  # Display success message
  echo
  echo_success "✨ NSELF installation complete!"
  echo
  echo_info "To get started:"
  echo_info "  1. Create a new directory for your project"
  echo_info "  2. Run 'nself init' to initialize"
  echo_info "  3. Edit .env.local to configure your project"
  echo_info "  4. Run 'nself build' to generate files"
  echo_info "  5. Run 'nself up' to start services"
  echo
  echo_info "Documentation: $REPO_URL"
  echo
  
  # Check if we need to reload shell
  if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo_info "⚠️  Please run 'source $SHELL_RC' or restart your terminal to use nself"
  fi
}

# Run main installation
main "$@"