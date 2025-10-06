#!/usr/bin/env bash
# update.sh - Update nself to the latest version

# Don't use set -e - we handle all errors explicitly
set -uo pipefail

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
  if ! latest_json=$(curl -sL --max-time 30 --connect-timeout 10 --retry 2 "$github_api" 2>/dev/null); then
    stop_loading $LOADING_PID ""
    log_error "Failed to check for updates"
    log_info "Please check your internet connection"
    return 1
  fi

  # Parse version from JSON response with robust extraction
  local latest_version
  latest_version=$(echo "$latest_json" | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)".*/\1/')

  # Remove 'v' prefix if present
  latest_version="${latest_version#v}"

  # Validate version format
  if [[ -z "$latest_version" ]] || [[ ! "$latest_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    stop_loading $LOADING_PID ""
    log_error "Could not determine valid version from GitHub"
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

  # Check required commands
  local missing_deps=()
  for cmd in curl tar rsync; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      missing_deps+=("$cmd")
    fi
  done

  if [[ ${#missing_deps[@]} -gt 0 ]]; then
    log_error "Missing required commands: ${missing_deps[*]}"
    log_info "Install missing dependencies:"
    [[ " ${missing_deps[*]} " =~ " curl " ]] && log_info "  macOS: brew install curl"
    [[ " ${missing_deps[*]} " =~ " rsync " ]] && log_info "  macOS: brew install rsync | Linux: apt/yum install rsync"
    [[ " ${missing_deps[*]} " =~ " tar " ]] && log_info "  tar is usually pre-installed"
    return 1
  fi

  # Get latest release info with timeout
  local latest_json
  if ! latest_json=$(curl -sL --max-time 30 --connect-timeout 10 --retry 2 --retry-delay 1 "$github_api" 2>/dev/null); then
    log_error "Failed to fetch release information"
    log_info "Please check your internet connection"
    return 1
  fi

  # Get download URL
  local asset_url
  asset_url=$(echo "$latest_json" | grep -o '"tarball_url"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)".*/\1/')

  if [[ -z "$asset_url" ]] || [[ ! "$asset_url" =~ ^https:// ]]; then
    log_error "No valid download URL found in latest release"
    return 1
  fi

  # Get latest version with validation
  local latest_version
  latest_version=$(echo "$latest_json" | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)".*/\1/')

  # Remove 'v' prefix if present
  latest_version="${latest_version#v}"

  # Validate version format
  if [[ -z "$latest_version" ]] || [[ ! "$latest_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    log_error "Invalid version format from GitHub: '$latest_version'"
    return 1
  fi

  # Create temporary directory with secure permissions
  local tmp_dir
  tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t nself_update.XXXXXX)
  local archive_file="$tmp_dir/nself_latest.tar.gz"
  local extract_dir="$tmp_dir/extracted"

  # Download with timeout and retries
  log_info "Downloading nself $latest_version..."
  if ! curl -L --max-time 300 --connect-timeout 10 --retry 3 --retry-delay 2 --progress-bar "$asset_url" -o "$archive_file" 2>&1; then
    log_error "Failed to download update"
    rm -rf "$tmp_dir"
    return 1
  fi

  # Verify downloaded file
  if [[ ! -f "$archive_file" ]]; then
    log_error "Download failed - file not found"
    rm -rf "$tmp_dir"
    return 1
  fi

  # Verify it's a valid gzip file
  if ! gzip -t "$archive_file" 2>/dev/null; then
    log_error "Downloaded file is corrupted or invalid"
    rm -rf "$tmp_dir"
    return 1
  fi

  # Extract
  log_info "Extracting update..."
  mkdir -p "$extract_dir"
  if ! tar -xzf "$archive_file" -C "$extract_dir" 2>/dev/null; then
    log_error "Failed to extract update"
    rm -rf "$tmp_dir"
    return 1
  fi

  # GitHub tarballs extract to a subdirectory like "owner-repo-hash/"
  # Find it and use it as source
  local extracted_dir
  extracted_dir=$(find "$extract_dir" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | head -1)

  if [[ ! -d "$extracted_dir" ]]; then
    log_error "Could not find extracted directory"
    rm -rf "$tmp_dir"
    return 1
  fi

  # Update installation
  log_info "Installing update..."
  # Get root installation directory (up two levels from src/cli/)
  local install_dir="$(dirname "$(dirname "$SCRIPT_DIR")")"

  # Backup current version file
  if [[ -f "$version_file" ]]; then
    cp "$version_file" "$version_file.backup"
  fi

  # Validate extracted structure
  if [[ ! -d "$extracted_dir/src" ]]; then
    log_error "Invalid archive structure - missing src directory"
    rm -rf "$tmp_dir"
    return 1
  fi

  # Update all directories that install.sh installs (mirrors install.sh behavior)
  log_info "Updating installation files..."

  # Update directories
  for dir in bin src docs; do
    if [[ -d "$extracted_dir/$dir" ]]; then
      log_info "Updating $dir/..."
      if ! rsync -a --delete "$extracted_dir/$dir/" "$install_dir/$dir/"; then
        log_error "Failed to update $dir/"
        # Restore backup if available
        if [[ -f "$version_file.backup" ]]; then
          mv "$version_file.backup" "$version_file"
        fi
        rm -rf "$tmp_dir"
        return 1
      fi
    fi
  done

  # Update root files
  for file in LICENSE README.md; do
    if [[ -f "$extracted_dir/$file" ]]; then
      cp "$extracted_dir/$file" "$install_dir/" 2>/dev/null || true
    fi
  done

  log_success "All files updated successfully"

  # Update version file
  echo "$latest_version" >"$version_file"

  # Verify critical files were updated
  log_info "Verifying update..."
  local verification_failed=false

  # Check bin/nself exists and is executable
  if [[ ! -x "$install_dir/bin/nself" ]]; then
    log_error "Verification failed: bin/nself not executable"
    verification_failed=true
  fi

  # Check VERSION file updated
  if [[ -f "$version_file" ]]; then
    local updated_version=$(cat "$version_file")
    if [[ "$updated_version" != "$latest_version" ]]; then
      log_error "Verification failed: VERSION mismatch"
      verification_failed=true
    fi
  else
    log_error "Verification failed: VERSION file missing"
    verification_failed=true
  fi

  if [[ "$verification_failed" == "true" ]]; then
    log_error "Update verification failed"
    # Restore backup if available
    if [[ -f "$version_file.backup" ]]; then
      mv "$version_file.backup" "$version_file"
    fi
    rm -rf "$tmp_dir"
    return 1
  fi

  log_success "Update verified successfully"

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

# Export for use as library
export -f cmd_update

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_update "$@"
fi
