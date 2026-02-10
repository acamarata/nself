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
  echo "  --json          Output in JSON format (for tooling)"
  echo "  -h, --help      Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself -v                # Show version number only"
  echo "  nself --version         # Show detailed information"
  echo "  nself version           # Show detailed information (default)"
  echo "  nself version --short   # Show version number only"
  echo "  nself version --json    # JSON output for integrations"
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

  # Check if called with --json flag for machine-readable output
  if [[ "$arg" == "--json" ]]; then
    local os=$(uname -s)
    local arch=$(uname -m)
    local docker_version=""
    local compose_version=""
    local git_commit=""
    local install_path="$CLI_SCRIPT_DIR"

    # Get Docker version
    if command -v docker >/dev/null 2>&1; then
      docker_version=$(docker --version 2>/dev/null | cut -d' ' -f3 | tr -d ',')
    fi

    # Get Docker Compose version
    if docker compose version >/dev/null 2>&1; then
      compose_version=$(docker compose version --short 2>/dev/null)
    fi

    # Get git commit if in a git repo
    if command -v git >/dev/null 2>&1 && [[ -d "$CLI_SCRIPT_DIR/../../.git" ]]; then
      git_commit=$(git -C "$CLI_SCRIPT_DIR/../.." rev-parse --short HEAD 2>/dev/null || echo "")
    fi

    printf '{\n'
    printf '  "version": "%s",\n' "$version"
    printf '  "commit": "%s",\n' "$git_commit"
    printf '  "platform": "%s/%s",\n' "$os" "$arch"
    printf '  "installPath": "%s",\n' "$install_path"
    printf '  "docker": "%s",\n' "$docker_version"
    printf '  "compose": "%s",\n' "$compose_version"
    printf '  "shell": "%s"\n' "$BASH_VERSION"
    printf '}\n'
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
  # Help is read-only - bypass init/env guards
  for _arg in "$@"; do
    if [[ "$_arg" == "--help" ]] || [[ "$_arg" == "-h" ]]; then
      show_version_help
      exit 0
    fi
  done
  pre_command "version" || exit $?
  cmd_version "$@"
  exit_code=$?
  post_command "version" $exit_code
  exit $exit_code
fi
