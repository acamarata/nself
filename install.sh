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

echo_warning() {
  echo -e "\033[1;33m[WARNING]\033[0m $1"
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

# Check if nself is already installed
check_existing_installation() {
  local installed=false
  local installed_version=""
  local installed_path=""
  
  # Check for nself in PATH
  if command_exists nself; then
    installed=true
    installed_path=$(which nself)
    # Try to get version
    installed_version=$(nself version 2>/dev/null || echo "unknown")
  elif [ -f "$BIN_DIR/nself.sh" ]; then
    installed=true
    installed_path="$BIN_DIR/nself.sh"
    # Try to get version from VERSION file
    if [ -f "$BIN_DIR/VERSION" ]; then
      installed_version=$(cat "$BIN_DIR/VERSION" 2>/dev/null || echo "unknown")
    else
      installed_version="unknown"
    fi
  fi
  
  if [ "$installed" = true ]; then
    echo ""
    echo_info "🔍 Found existing nself installation"
    echo_info "   Path: $installed_path"
    echo_info "   Version: $installed_version"
    
    # Check for updates with spinner
    local latest_version=""
    (
      latest_version=$(curl -fsSL "$REPO_RAW_URL/bin/VERSION" 2>/dev/null || echo "")
      echo "$latest_version" > /tmp/nself_latest_version.tmp
    ) &
    show_spinner $! "   Checking for updates"
    
    latest_version=$(cat /tmp/nself_latest_version.tmp 2>/dev/null | tr -d '[:space:]')
    rm -f /tmp/nself_latest_version.tmp
    
    if [ -n "$latest_version" ] && [ "$latest_version" != "$installed_version" ]; then
      echo ""
      echo_warning "🆕 A newer version is available: $latest_version"
      echo_info "   Current: $installed_version → Latest: $latest_version"
      echo ""
      printf "Would you like to update nself now? [Y/n] "
      read -r REPLY
      if [[ -z "$REPLY" || "$REPLY" =~ ^[Yy]$ ]]; then
        echo_info "🔄 Updating nself..."
        return 0  # Proceed with installation/update
      else
        echo_info "Keeping current version: $installed_version"
        echo_info "You can update later by running: nself update"
        exit 0
      fi
    elif [ -n "$latest_version" ]; then
      echo_success "✅ You have the latest version installed!"
      echo_info "   No updates needed. All good! 🎉"
      exit 0
    else
      echo_warning "⚠️  Could not check for updates (network issue?)"
      printf "Would you like to reinstall nself? [y/N] "
      read -r REPLY
      if [[ "$REPLY" =~ ^[Yy]$ ]]; then
        return 0  # Proceed with reinstallation
      else
        echo_info "Keeping current installation"
        exit 0
      fi
    fi
  fi
  
  return 0  # No existing installation, proceed
}

# Check system requirements
check_requirements() {
  local missing_deps=()
  local warnings=()
  
  echo ""
  echo_info "📋 Checking system requirements..."
  echo ""
  
  # Check OS
  OS="$(uname -s)"
  ARCH="$(uname -m)"
  
  echo_info "System: $OS ($ARCH)"
  
  if [ "$OS" != "Linux" ] && [ "$OS" != "Darwin" ]; then
    echo_error "Unsupported operating system: $OS"
    echo_error "nself requires Linux or macOS"
    exit 1
  fi
  
  # Check curl
  if ! command_exists curl; then
    missing_deps+=("curl")
  else
    echo_success "✓ curl is installed"
  fi
  
  # Check Docker
  if ! command_exists docker; then
    missing_deps+=("docker")
    echo_warning "✗ Docker is not installed"
  else
    echo_success "✓ Docker is installed ($(docker --version 2>/dev/null | cut -d' ' -f3 | cut -d',' -f1))"
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
      warnings+=("Docker daemon is not running. Please start Docker.")
    fi
  fi
  
  # Check Docker Compose
  if docker compose version >/dev/null 2>&1; then
    echo_success "✓ Docker Compose is installed (plugin)"
  elif command_exists docker-compose; then
    echo_success "✓ Docker Compose is installed (standalone)"
  else
    missing_deps+=("docker-compose")
    echo_warning "✗ Docker Compose is not installed"
  fi
  
  # Check git (optional but recommended)
  if command_exists git; then
    echo_success "✓ git is installed"
  else
    warnings+=("git is not installed (optional but recommended)")
  fi
  
  # Check disk space
  local available_space
  if [ "$OS" = "Darwin" ]; then
    available_space=$(df -h / | awk 'NR==2 {print $4}' | sed 's/Gi//')
  else
    available_space=$(df -h / | awk 'NR==2 {print $4}' | sed 's/G//')
  fi
  
  if [ "${available_space%%.*}" -lt 5 ] 2>/dev/null; then
    warnings+=("Low disk space: ${available_space}GB available (recommend at least 5GB)")
  else
    echo_success "✓ Sufficient disk space (${available_space} available)"
  fi
  
  echo ""
  
  # Show warnings
  if [ ${#warnings[@]} -gt 0 ]; then
    echo_warning "⚠️  Warnings:"
    for warning in "${warnings[@]}"; do
      echo_warning "   - $warning"
    done
    echo ""
  fi
  
  # Handle missing dependencies
  if [ ${#missing_deps[@]} -gt 0 ]; then
    echo_error "❌ Missing required dependencies:"
    for dep in "${missing_deps[@]}"; do
      echo_error "   - $dep"
    done
    echo ""
    echo_info "Would you like to install the missing dependencies?"
    read -p "Install missing dependencies? [Y/n] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
      return 0  # Will install missing deps
    else
      echo_error "Cannot proceed without required dependencies."
      echo_info "Please install the missing dependencies manually and run this script again."
      exit 1
    fi
  fi
  
  return 0
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
  echo ""
  echo_info "📦 Docker Installation Required"
  echo ""
  
  OS="$(uname -s)"
  
  if [ "$OS" = "Linux" ]; then
    echo_info "Installing Docker for Linux..."
    echo ""
    
    # Download and install with spinner
    (
      curl -fsSL https://get.docker.com -o /tmp/get-docker.sh 2>/dev/null
      sudo sh /tmp/get-docker.sh >/dev/null 2>&1
      rm -f /tmp/get-docker.sh
    ) &
    show_spinner $! "  Downloading and installing Docker"
    
    # Add current user to docker group
    echo_info "Adding $USER to docker group..."
    sudo usermod -aG docker "$USER"
    
    echo ""
    echo_success "✓ Docker installed successfully!"
    echo_warning "⚠️  You may need to log out and back in for group permissions to take effect"
    echo ""
  elif [ "$OS" = "Darwin" ]; then
    echo_warning "Docker Desktop is required for macOS"
    echo ""
    echo_info "📥 Please install Docker Desktop:"
    echo_info "   1. Visit: https://www.docker.com/products/docker-desktop"
    echo_info "   2. Download Docker Desktop for Mac"
    echo_info "   3. Install and start Docker Desktop"
    echo_info "   4. Run this installer again"
    echo ""
    exit 1
  else
    echo_error "Unsupported OS: $OS"
    exit 1
  fi
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
  
  # Backup existing installation if updating
  if [ -d "$BIN_DIR" ]; then
    BACKUP_DIR="$NSELF_DIR/backup_$(date +%Y%m%d_%H%M%S)"
    echo_info "Creating backup at $BACKUP_DIR"
    (
      mkdir -p "$BACKUP_DIR"
      cp -r "$BIN_DIR" "$BACKUP_DIR/" 2>/dev/null || true
    ) &
    show_spinner $! "  Backing up existing installation"
  fi
  
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
    "urls.sh"
    "VERSION"
  )
  
  # Create temporary directory for downloads
  TMP_DIR=$(mktemp -d)
  trap "rm -rf $TMP_DIR" EXIT
  
  # Download all files in background
  (
    # Download VERSION first to check compatibility
    VERSION_URL="$REPO_RAW_URL/bin/VERSION"
    VERSION_FILE="$TMP_DIR/VERSION"
    if ! curl -fsSL "$VERSION_URL" -o "$VERSION_FILE" 2>/dev/null; then
      echo "Failed to download VERSION file" >&2
      exit 1
    fi
    
    NEW_VERSION=$(cat "$VERSION_FILE" | tr -d '[:space:]')
    
    # Version compatibility check (future-proof)
    if [ -f "$BIN_DIR/VERSION" ]; then
      OLD_VERSION=$(cat "$BIN_DIR/VERSION" | tr -d '[:space:]')
      # Add any version-specific migration logic here in future versions
      # For now, all versions are compatible
    fi
    
    # Download bin files
    for file in "${BIN_FILES[@]}"; do
      TMP_FILE="$TMP_DIR/$file"
      if [ "$file" != "VERSION" ]; then  # Already downloaded VERSION
        if ! curl -fsSL "$REPO_RAW_URL/bin/$file" -o "$TMP_FILE" 2>/dev/null; then
          echo "Failed to download $file" >&2
          exit 1
        fi
      fi
    done
    
    # Verify all files downloaded successfully before moving
    for file in "${BIN_FILES[@]}"; do
      if [ ! -f "$TMP_DIR/$file" ]; then
        echo "Missing file: $file" >&2
        exit 1
      fi
    done
    
    # Move all files atomically
    for file in "${BIN_FILES[@]}"; do
      mv "$TMP_DIR/$file" "$BIN_DIR/$file"
      chmod +x "$BIN_DIR/$file"
    done
  ) &
  
  show_spinner $! "  Downloading nself files"
  
  if [ $? -ne 0 ]; then
    echo_error "Failed to download files"
    
    # Attempt rollback if backup exists
    if [ -d "$BACKUP_DIR" ]; then
      echo_warning "Attempting to restore from backup..."
      cp -r "$BACKUP_DIR/bin/"* "$BIN_DIR/" 2>/dev/null && \
        echo_success "Restored from backup" || \
        echo_error "Rollback failed. Manual intervention required."
    fi
    exit 1
  fi
  
  # Download template files
  (curl -fsSL "$REPO_RAW_URL/bin/templates/.env.example" -o "$TEMPLATES_DIR/.env.example" 2>/dev/null) &
  show_spinner $! "  Downloading templates"
  
  # Clean up old backups (keep last 3)
  if [ -d "$NSELF_DIR" ]; then
    ls -dt "$NSELF_DIR"/backup_* 2>/dev/null | tail -n +4 | xargs rm -rf 2>/dev/null || true
  fi
}

# Main installation
main() {
  echo
  echo "╔══════════════════════════════════════════════╗"
  echo "║           NSELF CLI Installer                ║"
  echo "║      Self-hosted Nhost Stack Manager         ║"
  echo "╚══════════════════════════════════════════════╝"
  echo
  
  # Check for existing installation
  check_existing_installation
  
  # Check system requirements
  check_requirements
  
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
  
  # Verify installation
  if [ -f "$BIN_DIR/nself.sh" ] && [ -f "$BIN_DIR/VERSION" ]; then
    local installed_version=$(cat "$BIN_DIR/VERSION" 2>/dev/null || echo "unknown")
    echo_success "✓ nself $installed_version installed successfully!"
  else
    echo_error "Installation verification failed"
    exit 1
  fi
  
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
  echo "╔══════════════════════════════════════════════╗"
  echo "║      ✨ NSELF Installation Complete! ✨      ║"
  echo "╚══════════════════════════════════════════════╝"
  echo
  echo_info "📚 Quick Start Guide:"
  echo "   1. Create a new directory for your project"
  echo "   2. Run 'nself init' to initialize"
  echo "   3. Edit .env.local to configure your project"
  echo "   4. Run 'nself build' to generate files"
  echo "   5. Run 'nself up' to start services"
  echo
  echo_info "📖 Documentation: $REPO_URL"
  echo_info "💡 Run 'nself help' for available commands"
  echo_info "🔄 Run 'nself update' to check for updates"
  echo
  
  # Check if we need to reload shell
  if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
    echo_info "⚠️  Please run 'source $SHELL_RC' or restart your terminal to use nself"
  fi
}

# Run main installation
main "$@"