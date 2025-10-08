#!/usr/bin/env bash
set -euo pipefail

# version.sh - Show version information

# Get script directory with absolute path
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_DIR="$CLI_SCRIPT_DIR"

# Source shared utilities
[[ -z "${DISPLAY_SOURCED:-}" ]] && source "$CLI_SCRIPT_DIR/../lib/utils/display.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Show help for version command
show_version_help() {
  echo "Usage: nself version [OPTIONS]"
  echo "       nself -v | --version"
  echo ""
  echo "Display nself version information"
  echo ""
  echo "Options:"
  echo "  --short         Show version number only"
  echo "  -h, --help      Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself -v                # Show version number only (0.3.9)"
  echo "  nself --version         # Show detailed information"
  echo "  nself version           # Show detailed information (default)"
  echo "  nself version --short   # Show version number only"
}

# Read version from VERSION file
get_version() {
  # Check src/VERSION file (new location)
  if [[ -f "$CLI_SCRIPT_DIR/../VERSION" ]]; then
    cat "$CLI_SCRIPT_DIR/../VERSION"
  elif [[ -f "$CLI_SCRIPT_DIR/../../src/VERSION" ]]; then
    cat "$CLI_SCRIPT_DIR/../../src/VERSION"
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

  # Check if called with --short flag for minimal output
  if [[ "$arg" == "--short" ]]; then
    echo "$version"
    return 0
  fi

  # Default behavior: show verbose output
  # "nself version" shows full info
  # "nself -v" needs to be handled by detecting how we were called
  show_command_header "nself v${version}" "Version Information"
  echo
  echo "Installation:"
  echo "  Location:    $CLI_SCRIPT_DIR"
  echo "  Config:      ${ENV_FILE:-.env.local}"
  echo
  echo "System:"
  echo "  OS:          $(uname -s)"
  echo "  Arch:        $(uname -m)"
  echo "  Shell:       $BASH_VERSION"

  # Check Docker version
  if command -v docker >/dev/null 2>&1; then
    echo "  Docker:      $(docker --version | cut -d' ' -f3 | tr -d ',')"
  fi

  # Check Docker Compose version
  if docker compose version >/dev/null 2>&1; then
    echo "  Compose:     $(docker compose version --short)"
  fi
  echo
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
