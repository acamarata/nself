#!/bin/bash

# install.sh - Installation script for nself CLI

set -e

# Variables
NSELF_DIR="$HOME/.nself"
BIN_DIR="$NSELF_DIR/bin"
UNITY_DIR="$NSELF_DIR/unity"
# please change this repository
REPO_RAW_URL="https://raw.githubusercontent.com/acamarata/nself/main/bin"

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

install_curl() {
    echo "Installing curl..."
    
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
    else
        echo "❌ Package manager not supported. Install curl manually."
        exit 1
    fi

    echo "✅ Curl installed successfully!"
}

# Function to install Docker
install_docker() {
  echo_info "Docker not found. Installing Docker..."

  OS="$(uname -s)"
  ARCH="$(uname -m)"

  if command -v curl >/dev/null 2>&1; then
      echo "✅ Curl is already installed: $(curl --version | head -n 1)"
  else
      echo "⚠️ Curl is not installed!"
      read -p "Do you want to install curl? (y/n): " choice
      case "$choice" in
          y|Y ) install_curl ;;
          n|N ) echo "❌ Curl installation skipped."; exit 1 ;;
          * ) echo "❌ Invalid choice. Exiting."; exit 1 ;;
      esac
  fi

  if [ "$OS" = "Linux" ]; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    rm get-docker.sh
  elif [ "$OS" = "Darwin" ]; then
    echo_error "Please install Docker Desktop from https://www.docker.com/products/docker-desktop and rerun the install script."
    exit 1
  else
    echo_error "Unsupported OS: $OS"
    exit 1
  fi

  echo_info "Docker installed successfully."
}

# Function to install Last version of Docker Compose
install_docker_compose() {
  echo_info "Docker Compose not found. Installing Docker Compose..."

  # DOCKER_COMPOSE_VERSION="2.20.2"

  OS="$(uname -s)"
  ARCH="$(uname -m)"

  if [ "$OS" = "Linux" ] || [ "$OS" = "Darwin" ]; then
    # sudo curl -L "https://github.com/docker/compose/releases/download/v${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    # Create a symlink if necessary
    if ! command_exists docker-compose; then
      sudo ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose 2>/dev/null || true
    fi
  else
    echo_error "Unsupported OS for Docker Compose installation: $OS"
    exit 1
  fi

  echo_info "Docker Compose installed successfully."
}

# Start installation
echo_info "Starting nself installation..."

# Check and install Docker if necessary
if ! command_exists docker; then
  install_docker
else
  echo_info "Docker is already installed."
fi

# Convert version to comparable format (remove dots)
DC_VERSION_NUMB=$(echo "$DC_VERSION" | awk -F. '{ printf("%d%02d%02d", $1,$2,$3) }')

if ! command_exists docker-compose ; then
  DC_INSTALLED=true
  install_docker_compose
else
  # Check and install Docker Compose if necessary
  DC_VERSION=$(docker-compose version --short)
  MIN_VERSION_NUM=21200

  if [ "$DC_VERSION_NUMB" -lt "$MIN_VERSION_NUM" ]; then
    DC_INSTALLED=true
  fi
  echo_info "Docker Compose is already installed."
fi

if [ "$DC_INSTALLED" = true ]; then
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    echo "docker-compose installation/update complete."
    echo "New version: $(docker-compose version --short)"
fi

# Create nself directories
echo_info "Creating nself directories at $BIN_DIR..."
mkdir -p "$BIN_DIR"
mkdir -p "$UNITY_DIR"
# Download necessary files
# declare -A files_to_download=(
#   [".env.example"]="$REPO_RAW_URL/.env.example"
#   ["docker-compose.template.yml"]="$REPO_RAW_URL/docker-compose.template.yml"
#   ["nself.sh"]="$REPO_RAW_URL/nself.sh"
#   ["VERSION"]="$REPO_RAW_URL/VERSION"
# )

# for file in "${!files_to_download[@]}"; do
#   url="${files_to_download[$file]}"
#   dest="$BIN_DIR/$file"

#   echo_info "Downloading $file from $url..."
#   if curl -fsSL "$url" -o "$dest"; then
#     echo_info "$file downloaded successfully."
#     if [ "$file" = "nself.sh" ]; then
#       chmod +x "$dest"
#     fi
#   else
#     echo_error "Failed to download $file from $url."
#     exit 1
#   fi
# done

# Copy shell to bin
cp bin/* $BIN_DIR
cp -R unity/. $UNITY_DIR

echo "✅ All files, including hidden ones, have been copied."

# Add bin directory to PATH if not already
if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
  echo_info "Adding $BIN_DIR to PATH..."
  SHELL_PROFILE=""
  if [ -n "$ZSH_VERSION" ]; then
    SHELL_PROFILE="$HOME/.zshrc"
  elif [ -n "$BASH_VERSION" ]; then
    SHELL_PROFILE="$HOME/.bashrc"
  else
    SHELL_PROFILE="$HOME/.profile"
  fi

  if ! grep -Fxq "export PATH=\"$BIN_DIR:\$PATH\"" "$SHELL_PROFILE"; then
    echo 'export PATH="$HOME/.nself/bin:$PATH"' >> "$SHELL_PROFILE"
    echo_info "$BIN_DIR added to PATH in $SHELL_PROFILE."
    # Export PATH in current session
    export PATH="$BIN_DIR:$PATH"
  else
    echo_info "$BIN_DIR is already in PATH."
  fi
else
  echo_info "$BIN_DIR is already in PATH."
fi

# Symlink nself.sh to /usr/local/bin/nself
echo_info "Creating symlink for nself..."
if sudo ln -sf "$BIN_DIR/nself.sh" /usr/local/bin/nself; then
  echo_info "Symlink created successfully."
else
  echo_error "Failed to create symlink for nself."
  exit 1
fi

echo_info "nself CLI installed successfully!"
echo_info "You can now use the 'nself' command from your terminal."
