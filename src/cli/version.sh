#!/usr/bin/env bash
# version.sh - Show version information

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Show help for version command
show_version_help() {
  echo "Usage: nself version [OPTIONS]"
  echo "       nself -v | --version"
  echo ""
  echo "Display nself version information"
  echo ""
  echo "Options:"
  echo "  --verbose       Show detailed version and system information"
  echo "  -h, --help      Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself version           # Show version"
  echo "  nself -v                # Show version (shorthand)"
  echo "  nself --version         # Show version (longhand)"
  echo "  nself version --verbose # Show detailed information"
}

# Read version from VERSION file
get_version() {
  # Check src/VERSION file (new location)
  if [[ -f "$SCRIPT_DIR/../VERSION" ]]; then
    cat "$SCRIPT_DIR/../VERSION"
  elif [[ -f "$SCRIPT_DIR/../../src/VERSION" ]]; then
    cat "$SCRIPT_DIR/../../src/VERSION"
  else
    echo "0.3.0"
  fi
}

# Command function
cmd_version() {
  local arg="${1:-}"
  local version=$(get_version)

  # Handle help flag
  if [[ "$arg" == "-h" ]] || [[ "$arg" == "--help" ]]; then
    show_version_help
    return 0
  fi

  if [[ "$arg" == "--verbose" ]]; then
    show_header "nself Version Information"
    echo "Version:     $version"
    echo "Location:    $SCRIPT_DIR"
    echo "Config:      ${ENV_FILE:-.env.local}"
    echo
    echo "System Information:"
    echo "  OS:        $(uname -s)"
    echo "  Arch:      $(uname -m)"
    echo "  Shell:     $BASH_VERSION"

    # Check Docker version
    if command -v docker >/dev/null 2>&1; then
      echo "  Docker:    $(docker --version | cut -d' ' -f3 | tr -d ',')"
    fi

    # Check Docker Compose version
    if docker compose version >/dev/null 2>&1; then
      echo "  Compose:   $(docker compose version --short)"
    fi
    echo
  else
    # Simple standard format matching common CLI tools
    echo "nself $version"
  fi
}

# Export for use as library
export -f cmd_version

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "version" || exit $?
  cmd_version "$@"
  exit_code=$?
  post_command "version" $exit_code
  exit $exit_code
fi
