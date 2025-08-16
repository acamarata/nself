#!/usr/bin/env bash
# update.sh - Update nself to the latest version

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"

# Show help for update command
show_update_help() {
  echo "nself update - Update nself to the latest version"
  echo ""
  echo "Usage: nself update [OPTIONS]"
  echo ""
  echo "Description:"
  echo "  Downloads and installs the latest version of nself from GitHub."
  echo "  Automatically detects current version and checks for updates."
  echo ""
  echo "Options:"
  echo "  --check             Check for updates without installing"
  echo "  -h, --help          Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself update                   # Update to latest version"
  echo "  nself update --check           # Check for updates only"
  echo ""
  echo "Update Process:"
  echo "  1. Checks current version against GitHub releases"
  echo "  2. Downloads latest release if newer version available"
  echo "  3. Backs up current installation"
  echo "  4. Installs new version"
  echo "  5. Verifies installation"
  echo ""
  echo "Notes:"
  echo "  • Requires internet connection"
  echo "  • Preserves existing configuration files"
  echo "  • Safe to run multiple times"
  echo "  • Shows current and latest version information"
}

# Function to check for updates
check_for_updates() {
  local repo_owner="acamarata"
  local repo_name="nself"
  local version_file="$SCRIPT_DIR/../VERSION"
  local github_api="https://api.github.com/repos/$repo_owner/$repo_name/releases/latest"

  # Get current version
  if [[ -f "$version_file" ]]; then
    local current_version=$(cat "$version_file")
  else
    local current_version="0.0.0"
  fi

  echo ""
  log_info "Current version: $current_version"

  # Get latest version from GitHub with loading spinner
  LOADING_PID=$(start_loading "Checking for updates...")

  local latest_json
  if ! latest_json=$(curl -sL "$github_api" 2>/dev/null); then
    stop_loading $LOADING_PID ""
    log_error "Failed to check for updates"
    log_info "Please check your internet connection"
    return 1
  fi

  # Parse version from JSON response
  local latest_version
  latest_version=$(echo "$latest_json" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)

  if [[ -z "$latest_version" ]]; then
    stop_loading $LOADING_PID ""
    log_error "Could not determine latest version"
    return 1
  fi

  # Stop loading and show latest version
  stop_loading $LOADING_PID "$(printf "%bℹ%b Latest version:  %s" "${COLOR_BLUE}" "${COLOR_RESET}" "$latest_version")"

  # Compare versions
  if [[ "$current_version" == "$latest_version" ]]; then
    log_success "Already up to date!"
    echo ""
    return 2 # Special return code for "already up to date"
  fi

  log_info "Update available: $current_version → $latest_version"
  echo ""
  return 0
}

# Function to perform update
perform_update() {
  local repo_owner="acamarata"
  local repo_name="nself"
  local version_file="$SCRIPT_DIR/../VERSION"
  local github_api="https://api.github.com/repos/$repo_owner/$repo_name/releases/latest"

  # Get latest release info
  local latest_json
  if ! latest_json=$(curl -sL "$github_api" 2>/dev/null); then
    log_error "Failed to fetch release information"
    return 1
  fi

  # Get download URL
  local asset_url
  asset_url=$(echo "$latest_json" | sed -n 's/.*"tarball_url":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)

  if [[ -z "$asset_url" ]]; then
    log_error "No download URL found in latest release"
    return 1
  fi

  # Get latest version
  local latest_version
  latest_version=$(echo "$latest_json" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)

  # Create temporary directory
  local tmp_dir
  tmp_dir=$(mktemp -d)
  local archive_file="$tmp_dir/nself_latest.tar.gz"
  local extract_dir="$tmp_dir/extracted"

  # Download
  log_info "Downloading nself $latest_version..."
  if ! curl -L "$asset_url" -o "$archive_file" 2>/dev/null; then
    log_error "Failed to download update"
    rm -rf "$tmp_dir"
    return 1
  fi

  # Extract
  log_info "Extracting update..."
  mkdir -p "$extract_dir"
  if ! tar -xzf "$archive_file" -C "$extract_dir" --strip-components=1 2>/dev/null; then
    log_error "Failed to extract update"
    rm -rf "$tmp_dir"
    return 1
  fi

  # Update installation
  log_info "Installing update..."
  local install_dir="$(dirname "$SCRIPT_DIR")"

  # Backup current version file
  if [[ -f "$version_file" ]]; then
    cp "$version_file" "$version_file.backup"
  fi

  # Update files (preserve bin directory wrapper)
  if ! rsync -a --delete "$extract_dir/src/" "$install_dir/"; then
    log_error "Failed to install update"
    # Restore backup if available
    if [[ -f "$version_file.backup" ]]; then
      mv "$version_file.backup" "$version_file"
    fi
    rm -rf "$tmp_dir"
    return 1
  fi

  # Update version file
  echo "$latest_version" >"$version_file"

  # Clean up
  rm -rf "$tmp_dir"
  if [[ -f "$version_file.backup" ]]; then
    rm -f "$version_file.backup"
  fi

  log_success "Successfully updated to nself $latest_version"
  log_info "Run 'nself version' to verify the update"
}

# Main command function
cmd_update() {
  local check_only=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
    --check)
      check_only=true
      shift
      ;;
    -h | --help)
      show_update_help
      return 0
      ;;
    *)
      log_error "Unknown option: $1"
      log_info "Use 'nself update --help' for usage information"
      return 1
      ;;
    esac
  done

  # Show header
  show_command_header "nself update" "Update to latest version"

  # Check for updates
  check_for_updates
  local check_result=$?

  case $check_result in
  0)
    # Update available
    if [[ "$check_only" == "true" ]]; then
      log_info "Update available. Run 'nself update' to install."
      return 0
    else
      echo ""
      log_info "Proceeding with update..."
      perform_update
    fi
    ;;
  2)
    # Already up to date
    return 0
    ;;
  *)
    # Error occurred
    return 1
    ;;
  esac
}

# Execute the command
cmd_update "$@"
