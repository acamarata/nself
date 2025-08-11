#!/bin/bash

# install.sh - Smart installation script for nself CLI
# Handles: fresh install, upgrades, migrations, different installation modes
#
# NOTE: This script must be self-contained since it runs before nself is installed.
#       Output functions mirror those in src/lib/utils/display.sh but are
#       duplicated here for independence. See docs/OUTPUT_FORMATTING.MD for standards.

set -e

# ========================================================================
# CONFIGURATION
# ========================================================================

# Default installation settings
DEFAULT_INSTALL_MODE="user"  # user, system, docker, portable
DEFAULT_INSTALL_DIR="$HOME/.nself"
DEFAULT_BRANCH="main"
DEFAULT_REPO="acamarata/nself"
NSELF_VERSION="${NSELF_VERSION:-}"  # Allow version override

# Parse command line arguments
INSTALL_MODE="${1:-$DEFAULT_INSTALL_MODE}"
INSTALL_DIR="${2:-$DEFAULT_INSTALL_DIR}"
FORCE_REINSTALL="${FORCE_REINSTALL:-false}"
SKIP_BACKUP="${SKIP_BACKUP:-false}"
SKIP_PATH="${SKIP_PATH:-false}"
VERBOSE="${VERBOSE:-false}"

# Repository URLs
REPO_URL="https://github.com/${DEFAULT_REPO}"
GITHUB_API="https://api.github.com/repos/${DEFAULT_REPO}"

# Get the version to install (latest release or specified version)
get_install_version() {
  if [[ -n "$NSELF_VERSION" ]]; then
    echo "$NSELF_VERSION"
  else
    # Fetch latest release tag from GitHub
    local latest_tag=$(curl -s "${GITHUB_API}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -n "$latest_tag" ]]; then
      echo "$latest_tag"
    else
      # Fallback to main branch if no releases
      echo "main"
    fi
  fi
}

INSTALL_VERSION=$(get_install_version)
REPO_RAW_URL="https://raw.githubusercontent.com/${DEFAULT_REPO}/${INSTALL_VERSION}"

# Installation paths
case "$INSTALL_MODE" in
  system)
    INSTALL_DIR="/usr/local/nself"
    BIN_LINK="/usr/local/bin/nself"
    NEEDS_SUDO=true
    ;;
  docker)
    INSTALL_DIR="/opt/nself"
    BIN_LINK="/usr/bin/nself"
    NEEDS_SUDO=false
    ;;
  portable)
    INSTALL_DIR="${INSTALL_DIR:-./nself}"
    BIN_LINK=""
    NEEDS_SUDO=false
    ;;
  user|*)
    INSTALL_DIR="${INSTALL_DIR:-$HOME/.nself}"
    BIN_LINK=""
    NEEDS_SUDO=false
    ;;
esac

BIN_DIR="$INSTALL_DIR/bin"
SRC_DIR="$INSTALL_DIR/src"
BACKUP_DIR="$HOME/.nself-backup"
TEMP_DIR=$(mktemp -d -t nself-install-XXXXXX)

# Cleanup on exit
trap 'rm -rf "$TEMP_DIR"' EXIT

# ========================================================================
# OUTPUT FUNCTIONS
# ========================================================================
# NOTE: These use echo_* prefix instead of log_* to avoid confusion with
#       system logging. Once installed, nself uses log_* functions from display.sh

# Color support detection
if [[ -t 1 ]] && [[ -n "${TERM:-}" ]] && command -v tput >/dev/null 2>&1; then
  RED=$(tput setaf 1)
  GREEN=$(tput setaf 2)
  YELLOW=$(tput setaf 3)
  BLUE=$(tput setaf 4)
  MAGENTA=$(tput setaf 5)
  CYAN=$(tput setaf 6)
  BOLD=$(tput bold)
  RESET=$(tput sgr0)
else
  RED=""
  GREEN=""
  YELLOW=""
  BLUE=""
  MAGENTA=""
  CYAN=""
  BOLD=""
  RESET=""
fi

echo_header() {
  local title="$1"
  local width=72
  local title_len=${#title}
  local padding=$(( (width - title_len - 2) / 2 ))
  local right_padding=$(( width - title_len - 2 - padding ))
  
  echo ""
  echo "${BOLD}╔$(printf '═%.0s' $(seq 1 $width))╗${RESET}"
  printf "${BOLD}║%*s%s%*s║${RESET}\n" $padding "" "$title" $right_padding ""
  echo "${BOLD}╚$(printf '═%.0s' $(seq 1 $width))╝${RESET}"
  echo ""
}

echo_section() {
  local title="$1"
  echo ""
  echo "${BOLD}${title}${RESET}"
  echo "$(printf '─%.0s' $(seq 1 ${#title}))"
}

echo_info() {
  echo "${BLUE}[INFO]${RESET} $1"
}

echo_success() {
  echo "${GREEN}[SUCCESS]${RESET} $1"
}

echo_warning() {
  echo "${YELLOW}[WARNING]${RESET} $1"
}

echo_error() {
  echo "${RED}[ERROR]${RESET} $1" >&2
}

echo_debug() {
  [[ "$VERBOSE" == "true" ]] && echo "${MAGENTA}[DEBUG]${RESET} $1"
}

# Progress spinner (follows OUTPUT_FORMATTING.MD standard)
show_spinner() {
  local pid=$1
  local message=$2
  local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  local i=0
  
  if [[ -t 1 ]]; then
    # Use BLUE for spinner per standard
    printf "${BLUE}%s${RESET}" "$message"
    
    while kill -0 $pid 2>/dev/null; do
      i=$(( (i+1) %10 ))
      printf "\r${BLUE}${spin:$i:1}${RESET} %s" "$message"
      sleep 0.1
    done
    
    wait $pid
    local result=$?
    
    if [ $result -eq 0 ]; then
      printf "\r${GREEN}✓${RESET} %s\n" "$message"
    else
      printf "\r${RED}✗${RESET} %s\n" "$message"
    fi
    
    return $result
  else
    echo "$message"
    wait $pid
    return $?
  fi
}

confirm() {
  local prompt="${1:-Continue?}"
  local default="${2:-n}"
  
  if [[ "$default" == "y" ]]; then
    prompt="$prompt [Y/n]: "
    default_val=0
  else
    prompt="$prompt [y/N]: "
    default_val=1
  fi
  
  read -p "$prompt" -n 1 -r
  echo
  
  if [[ -z "$REPLY" ]]; then
    return $default_val
  elif [[ "$REPLY" =~ ^[Yy]$ ]]; then
    return 0
  else
    return 1
  fi
}

# ========================================================================
# UTILITY FUNCTIONS
# ========================================================================

command_exists() {
  command -v "$1" >/dev/null 2>&1
}

get_sudo() {
  if [[ "$NEEDS_SUDO" == "true" ]] && [[ "$EUID" -ne 0 ]]; then
    echo "sudo"
  else
    echo ""
  fi
}

run_cmd() {
  local sudo=$(get_sudo)
  echo_debug "Running: $sudo $*"
  $sudo "$@"
}

detect_os() {
  case "$(uname -s)" in
    Darwin*)  OS="macos" ;;
    Linux*)   OS="linux" ;;
    CYGWIN*)  OS="windows" ;;
    MINGW*)   OS="windows" ;;
    *)        OS="unknown" ;;
  esac
  echo_debug "Detected OS: $OS"
}

detect_arch() {
  case "$(uname -m)" in
    x86_64)   ARCH="amd64" ;;
    aarch64)  ARCH="arm64" ;;
    arm64)    ARCH="arm64" ;;
    armv7l)   ARCH="arm" ;;
    *)        ARCH="unknown" ;;
  esac
  echo_debug "Detected architecture: $ARCH"
}

version_compare() {
  # Compare two version strings
  # Returns: 0 if equal, 1 if $1 > $2, 2 if $1 < $2
  local v1="$1"
  local v2="$2"
  
  if [[ "$v1" == "$v2" ]]; then
    return 0
  fi
  
  # Sort versions and check which is higher
  local sorted=$(echo -e "$v1\n$v2" | sort -V | head -n1)
  
  if [[ "$sorted" == "$v1" ]]; then
    return 2  # v1 < v2
  else
    return 1  # v1 > v2
  fi
}

get_installed_version() {
  local version="unknown"
  
  # Try multiple methods to get version
  if command_exists nself; then
    version=$(nself version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -n1 || echo "unknown")
  elif [[ -f "$BIN_DIR/VERSION" ]]; then
    version=$(cat "$BIN_DIR/VERSION" 2>/dev/null || echo "unknown")
  elif [[ -f "$INSTALL_DIR/VERSION" ]]; then
    version=$(cat "$INSTALL_DIR/VERSION" 2>/dev/null || echo "unknown")
  fi
  
  echo "$version"
}

get_latest_version() {
  local version
  version=$(curl -fsSL "$REPO_RAW_URL/src/config/VERSION" 2>/dev/null || \
           curl -fsSL "$REPO_RAW_URL/VERSION" 2>/dev/null || \
           echo "unknown")
  echo "$version"
}

# ========================================================================
# INSTALLATION DETECTION
# ========================================================================

detect_existing_installation() {
  echo_header "Checking for Existing Installation"
  
  local found_installations=()
  
  # Check standard locations
  local locations=(
    "$HOME/.nself"
    "/usr/local/nself"
    "/opt/nself"
    "./nself"
  )
  
  for loc in "${locations[@]}"; do
    if [[ -d "$loc" ]] && [[ -f "$loc/bin/nself.sh" || -f "$loc/bin/nself" ]]; then
      found_installations+=("$loc")
      echo_info "Found installation at: $loc"
    fi
  done
  
  # Check PATH for nself
  if command_exists nself; then
    local nself_path=$(which nself)
    echo_info "Found nself in PATH: $nself_path"
    
    # Get the installation directory from the command
    local nself_dir=$(dirname $(dirname $(readlink -f "$nself_path" || echo "$nself_path")))
    if [[ -d "$nself_dir" ]] && [[ ! " ${found_installations[@]} " =~ " ${nself_dir} " ]]; then
      found_installations+=("$nself_dir")
    fi
  fi
  
  if [[ ${#found_installations[@]} -eq 0 ]]; then
    echo_success "No existing installation found - proceeding with fresh install"
    return 1
  else
    # Check version of primary installation
    local installed_version=$(get_installed_version)
    local latest_version=$(get_latest_version)
    
    echo ""
    echo_info "Currently installed: v${installed_version}"
    echo_info "Latest available: v${latest_version}"
    
    # Determine if this is an upgrade or reinstall
    if [[ "$installed_version" != "unknown" ]] && [[ "$latest_version" != "unknown" ]]; then
      version_compare "$installed_version" "$latest_version"
      local cmp_result=$?
      
      if [[ $cmp_result -eq 0 ]]; then
        echo_success "You have the latest version installed"
        
        if [[ "$FORCE_REINSTALL" != "true" ]]; then
          if ! confirm "Reinstall anyway?" "n"; then
            echo_info "Installation cancelled"
            exit 0
          fi
        fi
      elif [[ $cmp_result -eq 2 ]]; then
        echo_warning "An update is available: v${installed_version} → v${latest_version}"
        
        # Check for breaking changes (major version difference)
        local installed_major="${installed_version%%.*}"
        local latest_major="${latest_version%%.*}"
        
        if [[ "$installed_major" != "$latest_major" ]]; then
          echo ""
          echo_warning "⚠️  BREAKING CHANGES DETECTED ⚠️"
          echo_warning "This is a major version upgrade ($installed_major.x → $latest_major.x)"
          echo_warning "Your current installation will be backed up"
          echo ""
          
          if ! confirm "Proceed with upgrade?" "y"; then
            echo_info "Upgrade cancelled"
            exit 0
          fi
        fi
      else
        echo_warning "Installed version ($installed_version) is newer than latest ($latest_version)"
        
        if ! confirm "Downgrade to v${latest_version}?" "n"; then
          echo_info "Installation cancelled"
          exit 0
        fi
      fi
    fi
    
    return 0
  fi
}

# ========================================================================
# BACKUP FUNCTIONS
# ========================================================================

backup_existing_installation() {
  if [[ "$SKIP_BACKUP" == "true" ]]; then
    echo_info "Skipping backup (--skip-backup specified)"
    return 0
  fi
  
  if [[ ! -d "$INSTALL_DIR" ]]; then
    return 0
  fi
  
  echo_header "Backing Up Existing Installation"
  
  local timestamp=$(date +%Y%m%d_%H%M%S)
  local version=$(get_installed_version)
  local backup_name="nself_${version}_${timestamp}"
  local backup_path="$BACKUP_DIR/$backup_name"
  
  echo_info "Creating backup at: $backup_path"
  
  (
    mkdir -p "$BACKUP_DIR"
    cp -r "$INSTALL_DIR" "$backup_path"
    
    # Save installation metadata
    cat > "$backup_path/.backup_info" << EOF
Backup Date: $(date)
Version: $version
Original Path: $INSTALL_DIR
Installation Mode: $INSTALL_MODE
EOF
  ) &
  show_spinner $! "Backing up installation"
  
  echo_success "Backup created: $backup_path"
  
  # Clean old backups (keep last 3)
  echo_info "Cleaning old backups..."
  local backup_count=$(ls -1 "$BACKUP_DIR" 2>/dev/null | wc -l)
  if [[ $backup_count -gt 3 ]]; then
    ls -1t "$BACKUP_DIR" | tail -n +4 | while read old_backup; do
      echo_debug "Removing old backup: $old_backup"
      rm -rf "$BACKUP_DIR/$old_backup"
    done
  fi
}

# ========================================================================
# PREREQUISITE CHECKS
# ========================================================================

check_prerequisites() {
  echo_header "Checking Prerequisites"
  
  local errors=0
  
  # Check OS
  detect_os
  if [[ "$OS" == "unknown" ]]; then
    echo_error "Unsupported operating system"
    ((errors++))
  else
    echo_success "Operating system: $OS"
  fi
  
  # Check architecture
  detect_arch
  if [[ "$ARCH" == "unknown" ]]; then
    echo_warning "Unknown architecture - installation may not work correctly"
  else
    echo_success "Architecture: $ARCH"
  fi
  
  # Check for required commands
  local required_commands=("curl" "tar" "bash")
  for cmd in "${required_commands[@]}"; do
    if command_exists "$cmd"; then
      echo_success "Found required command: $cmd"
    else
      echo_error "Missing required command: $cmd"
      ((errors++))
    fi
  done
  
  # Check for optional but recommended commands
  local optional_commands=("git" "docker")
  for cmd in "${optional_commands[@]}"; do
    if command_exists "$cmd"; then
      echo_success "Found optional command: $cmd"
    else
      echo_warning "Missing optional command: $cmd (some features may not work)"
    fi
  done
  
  # Check disk space
  local available_space=$(df "$HOME" | awk 'NR==2 {print $4}')
  if [[ $available_space -lt 100000 ]]; then  # Less than 100MB
    echo_error "Insufficient disk space (need at least 100MB)"
    ((errors++))
  else
    echo_success "Sufficient disk space available"
  fi
  
  # Check permissions for installation directory
  local parent_dir=$(dirname "$INSTALL_DIR")
  if [[ ! -w "$parent_dir" ]]; then
    if [[ "$NEEDS_SUDO" != "true" ]]; then
      echo_error "No write permission for $parent_dir"
      echo_info "Try: sudo $0 system  # for system-wide installation"
      ((errors++))
    fi
  else
    echo_success "Write permission for installation directory"
  fi
  
  if [[ $errors -gt 0 ]]; then
    echo ""
    echo_error "Prerequisites check failed with $errors error(s)"
    exit 1
  fi
  
  echo ""
  echo_success "All prerequisites met"
}

# ========================================================================
# DOWNLOAD FUNCTIONS
# ========================================================================

download_nself() {
  echo_header "Downloading nself"
  
  echo_info "Source: $REPO_URL"
  echo_info "Target: $TEMP_DIR"
  
  # Try git first (preserves history and is faster)
  if command_exists git; then
    (
      git clone --depth 1 --branch "$INSTALL_VERSION" "$REPO_URL.git" "$TEMP_DIR/nself" 2>/dev/null
    ) &
    
    if show_spinner $! "Downloading via git"; then
      echo_success "Downloaded successfully"
      return 0
    else
      echo_warning "Git clone failed, trying alternative method..."
    fi
  fi
  
  # Fallback to tar download
  local tar_url="$REPO_URL/archive/refs/heads/${DEFAULT_BRANCH}.tar.gz"
  
  (
    curl -fsSL "$tar_url" | tar -xz -C "$TEMP_DIR" --strip-components=1
  ) &
  
  if show_spinner $! "Downloading via curl"; then
    echo_success "Downloaded successfully"
    return 0
  else
    echo_error "Failed to download nself"
    exit 1
  fi
}

# ========================================================================
# INSTALLATION FUNCTIONS
# ========================================================================

install_files() {
  echo_header "Installing Files"
  
  local source_dir="$TEMP_DIR/nself"
  [[ -d "$source_dir" ]] || source_dir="$TEMP_DIR"
  
  # Create installation directory
  echo_info "Creating directory: $INSTALL_DIR"
  run_cmd mkdir -p "$INSTALL_DIR"
  
  # Copy files
  echo_info "Copying files..."
  (
    # Copy bin directory (should only contain the shim)
    if [[ -d "$source_dir/bin" ]]; then
      run_cmd cp -r "$source_dir/bin" "$INSTALL_DIR/"
    fi
    
    # Copy src directory (contains all the logic)
    if [[ -d "$source_dir/src" ]]; then
      run_cmd cp -r "$source_dir/src" "$INSTALL_DIR/"
    fi
    
    # Copy docs directory
    if [[ -d "$source_dir/docs" ]]; then
      run_cmd cp -r "$source_dir/docs" "$INSTALL_DIR/"
    fi
    
    # Copy VERSION file from its new location
    for version_file in "$source_dir/src/config/VERSION" "$source_dir/VERSION"; do
      if [[ -f "$version_file" ]]; then
        # Keep VERSION in src/config where it belongs
        run_cmd mkdir -p "$INSTALL_DIR/src/config"
        run_cmd cp "$version_file" "$INSTALL_DIR/src/config/VERSION"
        break
      fi
    done
    
    # Copy LICENSE and README
    for file in LICENSE README.md; do
      [[ -f "$source_dir/$file" ]] && run_cmd cp "$source_dir/$file" "$INSTALL_DIR/"
    done
  ) &
  show_spinner $! "Installing files"
  
  # Set permissions
  echo_info "Setting permissions..."
  (
    run_cmd chmod -R 755 "$INSTALL_DIR"
    # Make the bin shim executable
    run_cmd chmod +x "$INSTALL_DIR/bin/nself" 2>/dev/null || true
    # Make all CLI scripts executable
    run_cmd chmod +x "$INSTALL_DIR/src/cli/"*.sh 2>/dev/null || true
    # Make all tool scripts executable
    run_cmd find "$INSTALL_DIR/src/tools" -name "*.sh" -exec chmod +x {} \; 2>/dev/null || true
  ) &
  show_spinner $! "Setting permissions"
  
  echo_success "Files installed to: $INSTALL_DIR"
}

setup_path() {
  if [[ "$SKIP_PATH" == "true" ]] || [[ "$INSTALL_MODE" == "portable" ]]; then
    return 0
  fi
  
  echo_header "Setting Up PATH"
  
  # For system installation, create symlink
  if [[ -n "$BIN_LINK" ]]; then
    echo_info "Creating symlink: $BIN_LINK"
    run_cmd ln -sf "$BIN_DIR/nself" "$BIN_LINK"
    echo_success "System-wide installation complete"
    return 0
  fi
  
  # For user installation, update shell configuration
  local shell_configs=()
  local current_shell=$(basename "$SHELL")
  
  case "$current_shell" in
    bash)  shell_configs+=("$HOME/.bashrc" "$HOME/.bash_profile") ;;
    zsh)   shell_configs+=("$HOME/.zshrc") ;;
    fish)  shell_configs+=("$HOME/.config/fish/config.fish") ;;
    *)     shell_configs+=("$HOME/.profile") ;;
  esac
  
  local path_line="export PATH=\"$BIN_DIR:\$PATH\""
  local added_to=()
  
  for config in "${shell_configs[@]}"; do
    if [[ -f "$config" ]]; then
      # Check if already in PATH
      if grep -q "$BIN_DIR" "$config" 2>/dev/null; then
        echo_info "PATH already configured in $config"
      else
        echo "" >> "$config"
        echo "# Added by nself installer on $(date)" >> "$config"
        echo "$path_line" >> "$config"
        added_to+=("$config")
        echo_success "Added to PATH in: $config"
      fi
    fi
  done
  
  if [[ ${#added_to[@]} -gt 0 ]]; then
    echo ""
    echo_warning "PATH has been updated in: ${added_to[*]}"
    echo_warning "Run this to use nself immediately:"
    echo ""
    echo "    ${CYAN}source ${added_to[0]}${RESET}"
    echo ""
    echo "Or start a new terminal session"
  fi
}

# ========================================================================
# MIGRATION FUNCTIONS
# ========================================================================

migrate_configuration() {
  local old_version="$1"
  local new_version="$2"
  
  echo_header "Migrating Configuration"
  
  echo_info "Migrating from v${old_version} to v${new_version}"
  
  # Version-specific migrations
  local old_major="${old_version%%.*}"
  local new_major="${new_version%%.*}"
  
  if [[ "$old_major" == "0" ]] && [[ "$new_major" == "0" ]]; then
    # 0.x to 0.y migration
    local old_minor=$(echo "$old_version" | cut -d. -f2)
    local new_minor=$(echo "$new_version" | cut -d. -f2)
    
    if [[ $old_minor -lt 3 ]] && [[ $new_minor -ge 3 ]]; then
      echo_info "Migrating from pre-0.3.0 structure..."
      
      # Specific migrations for 0.2.x → 0.3.x
      # - Directory structure changed
      # - Command files reorganized
      # - Configuration format updated
      
      echo_success "Migration completed for 0.3.0"
    fi
  fi
  
  # Copy user configurations if they exist
  local user_configs=(
    "$HOME/.nself/config.json"
    "$HOME/.nself/settings.json"
    "$HOME/.nself/.env"
  )
  
  for config in "${user_configs[@]}"; do
    if [[ -f "$config" ]]; then
      local config_name=$(basename "$config")
      echo_info "Preserving user configuration: $config_name"
      cp "$config" "$INSTALL_DIR/" 2>/dev/null || true
    fi
  done
}

# ========================================================================
# VERIFICATION
# ========================================================================

verify_installation() {
  echo_header "Verifying Installation"
  
  local errors=0
  
  # Check main executable exists
  if [[ -f "$BIN_DIR/nself" ]] && [[ -f "$SRC_DIR/cli/nself.sh" ]]; then
    echo_success "Main executable found"
  else
    echo_error "Main executable not found"
    ((errors++))
  fi
  
  # Check if nself is accessible
  if [[ "$INSTALL_MODE" != "portable" ]]; then
    if command_exists nself || [[ -f "$BIN_LINK" ]]; then
      echo_success "nself is accessible from PATH"
      
      # Try to get version
      local version=$("$BIN_DIR/nself" version 2>/dev/null || echo "unknown")
      echo_success "Installed version: $version"
    else
      echo_warning "nself not in PATH yet (restart terminal or source shell config)"
    fi
  fi
  
  # Check critical directories
  local required_dirs=("bin" "src/cli" "src/lib" "src/templates")
  for dir in "${required_dirs[@]}"; do
    if [[ -d "$INSTALL_DIR/$dir" ]]; then
      echo_success "Required directory exists: $dir"
    else
      echo_error "Missing required directory: $dir"
      ((errors++))
    fi
  done
  
  if [[ $errors -gt 0 ]]; then
    echo ""
    echo_error "Installation verification failed with $errors error(s)"
    
    # Offer to restore backup
    if [[ -d "$BACKUP_DIR" ]] && [[ $(ls -1 "$BACKUP_DIR" 2>/dev/null | wc -l) -gt 0 ]]; then
      echo ""
      if confirm "Restore from backup?" "y"; then
        restore_from_backup
      fi
    fi
    
    exit 1
  fi
  
  echo ""
  echo_success "Installation verified successfully"
}

# ========================================================================
# RESTORE FUNCTIONS
# ========================================================================

restore_from_backup() {
  echo_header "Restoring from Backup"
  
  if [[ ! -d "$BACKUP_DIR" ]]; then
    echo_error "No backups found"
    return 1
  fi
  
  # List available backups
  local backups=($(ls -1t "$BACKUP_DIR" 2>/dev/null))
  
  if [[ ${#backups[@]} -eq 0 ]]; then
    echo_error "No backups found"
    return 1
  fi
  
  echo_info "Available backups:"
  local i=1
  for backup in "${backups[@]}"; do
    local info_file="$BACKUP_DIR/$backup/.backup_info"
    if [[ -f "$info_file" ]]; then
      local backup_date=$(grep "Backup Date:" "$info_file" | cut -d: -f2-)
      local backup_version=$(grep "Version:" "$info_file" | cut -d: -f2 | tr -d ' ')
      echo "  $i) $backup (v${backup_version},${backup_date})"
    else
      echo "  $i) $backup"
    fi
    ((i++))
  done
  
  echo ""
  read -p "Select backup number (1-${#backups[@]}): " selection
  
  if [[ ! "$selection" =~ ^[0-9]+$ ]] || [[ $selection -lt 1 ]] || [[ $selection -gt ${#backups[@]} ]]; then
    echo_error "Invalid selection"
    return 1
  fi
  
  local selected_backup="${backups[$((selection-1))]}"
  echo_info "Restoring from: $selected_backup"
  
  # Remove current installation
  echo_info "Removing current installation..."
  run_cmd rm -rf "$INSTALL_DIR"
  
  # Restore backup
  echo_info "Restoring backup..."
  run_cmd cp -r "$BACKUP_DIR/$selected_backup" "$INSTALL_DIR"
  
  echo_success "Restored from backup successfully"
}

# ========================================================================
# UNINSTALL FUNCTION
# ========================================================================

uninstall_nself() {
  echo_header "Uninstalling nself"
  
  if ! confirm "Are you sure you want to uninstall nself?" "n"; then
    echo_info "Uninstall cancelled"
    exit 0
  fi
  
  # Remove installation directory
  if [[ -d "$INSTALL_DIR" ]]; then
    echo_info "Removing $INSTALL_DIR..."
    run_cmd rm -rf "$INSTALL_DIR"
  fi
  
  # Remove symlinks
  if [[ -n "$BIN_LINK" ]] && [[ -L "$BIN_LINK" ]]; then
    echo_info "Removing symlink $BIN_LINK..."
    run_cmd rm -f "$BIN_LINK"
  fi
  
  # Remove from PATH
  echo_info "Removing from PATH configurations..."
  local shell_configs=("$HOME/.bashrc" "$HOME/.bash_profile" "$HOME/.zshrc" "$HOME/.profile")
  
  for config in "${shell_configs[@]}"; do
    if [[ -f "$config" ]] && grep -q "$BIN_DIR" "$config"; then
      # Remove nself PATH entries
      sed -i.bak "/$BIN_DIR/d" "$config"
      sed -i.bak "/nself installer/d" "$config"
      echo_success "Removed from $config"
    fi
  done
  
  # Ask about backups
  if [[ -d "$BACKUP_DIR" ]]; then
    if confirm "Remove all backups?" "n"; then
      echo_info "Removing backups..."
      rm -rf "$BACKUP_DIR"
    else
      echo_info "Keeping backups in $BACKUP_DIR"
    fi
  fi
  
  echo ""
  echo_success "nself has been uninstalled"
  echo_info "Thank you for using nself!"
}

# ========================================================================
# MAIN INSTALLATION FLOW
# ========================================================================

print_banner() {
  echo ""
  echo "${BOLD}${CYAN}╔════════════════════════════════════════════════════════════════╗${RESET}"
  echo "${BOLD}${CYAN}║                                                                ║${RESET}"
  echo "${BOLD}${CYAN}║${RESET}  ${BOLD}${GREEN}  _  _   ___  ___  _     ___${RESET}   ${CYAN}Self-Hosted Infrastructure${RESET}   ${BOLD}${CYAN}║${RESET}"
  echo "${BOLD}${CYAN}║${RESET}  ${BOLD}${GREEN} | \\| | / __|| __|| |   | __|${RESET}  ${CYAN}Made Simple${RESET}                 ${BOLD}${CYAN}║${RESET}"
  echo "${BOLD}${CYAN}║${RESET}  ${BOLD}${GREEN} | .  | \\__ \\| _| | |__ | _|${RESET}                                ${BOLD}${CYAN}║${RESET}"
  echo "${BOLD}${CYAN}║${RESET}  ${BOLD}${GREEN} |_|\\_| |___/|___||____||_|${RESET}   ${CYAN}v$(get_latest_version)${RESET}                      ${BOLD}${CYAN}║${RESET}"
  echo "${BOLD}${CYAN}║                                                                ║${RESET}"
  echo "${BOLD}${CYAN}╚════════════════════════════════════════════════════════════════╝${RESET}"
  echo ""
}

print_help() {
  echo "Usage: $0 [mode] [directory] [options]"
  echo ""
  echo "Installation modes:"
  echo "  user      Install for current user (default)"
  echo "  system    Install system-wide (requires sudo)"
  echo "  docker    Install for Docker container"
  echo "  portable  Install to current/specified directory"
  echo ""
  echo "Options:"
  echo "  --force           Force reinstall even if up to date"
  echo "  --skip-backup     Don't backup existing installation"
  echo "  --skip-path       Don't modify PATH"
  echo "  --verbose         Show detailed output"
  echo "  --uninstall       Uninstall nself"
  echo "  --help            Show this help"
  echo ""
  echo "Examples:"
  echo "  $0                    # Install for current user"
  echo "  $0 system             # Install system-wide"
  echo "  $0 portable ./tools   # Install to ./tools/nself"
  echo "  $0 --uninstall        # Remove nself"
  echo ""
  echo "Environment variables:"
  echo "  FORCE_REINSTALL=true  Force reinstallation"
  echo "  SKIP_BACKUP=true      Skip backup"
  echo "  VERBOSE=true          Verbose output"
}

main() {
  # Handle special flags
  case "${1:-}" in
    --help|-h)
      print_help
      exit 0
      ;;
    --uninstall)
      uninstall_nself
      exit 0
      ;;
    --force)
      FORCE_REINSTALL=true
      shift
      ;;
    --skip-backup)
      SKIP_BACKUP=true
      shift
      ;;
    --skip-path)
      SKIP_PATH=true
      shift
      ;;
    --verbose|-v)
      VERBOSE=true
      shift
      ;;
  esac
  
  # Re-parse after handling flags
  INSTALL_MODE="${1:-$DEFAULT_INSTALL_MODE}"
  INSTALL_DIR="${2:-$DEFAULT_INSTALL_DIR}"
  
  # Print banner
  print_banner
  
  echo_info "Installation mode: ${BOLD}$INSTALL_MODE${RESET}"
  echo_info "Installation directory: ${BOLD}$INSTALL_DIR${RESET}"
  echo ""
  
  # Run installation steps
  check_prerequisites
  
  # Check for existing installation and handle upgrade/backup
  if detect_existing_installation; then
    local old_version=$(get_installed_version)
    backup_existing_installation
    
    # Remove old installation
    echo_info "Removing old installation..."
    run_cmd rm -rf "$INSTALL_DIR"
  fi
  
  download_nself
  install_files
  
  # Migrate if this was an upgrade
  if [[ -n "${old_version:-}" ]] && [[ "$old_version" != "unknown" ]]; then
    local new_version=$(get_latest_version)
    if [[ "$old_version" != "$new_version" ]]; then
      migrate_configuration "$old_version" "$new_version"
    fi
  fi
  
  setup_path
  verify_installation
  
  # Print success message
  echo ""
  echo_header "Installation Complete! 🎉"
  
  echo "${GREEN}nself has been successfully installed!${RESET}"
  echo ""
  echo "Next steps:"
  echo "  1. Restart your terminal or run: ${CYAN}source ~/.bashrc${RESET}"
  echo "  2. Verify installation: ${CYAN}nself version${RESET}"
  echo "  3. Get started: ${CYAN}nself help${RESET}"
  echo ""
  echo "Quick start:"
  echo "  ${CYAN}mkdir myproject && cd myproject${RESET}"
  echo "  ${CYAN}nself init${RESET}"
  echo "  ${CYAN}nself up${RESET}"
  echo ""
  echo "Documentation: ${BLUE}https://github.com/${DEFAULT_REPO}/wiki${RESET}"
  echo "Support: ${BLUE}https://github.com/${DEFAULT_REPO}/issues${RESET}"
  echo ""
  
  # Show backup location if created
  if [[ -d "$BACKUP_DIR" ]] && [[ $(ls -1 "$BACKUP_DIR" 2>/dev/null | wc -l) -gt 0 ]]; then
    echo_info "Previous installation backed up to: $BACKUP_DIR"
  fi
}

# ========================================================================
# ENTRY POINT
# ========================================================================

# Run main function
main "$@"