#!/usr/bin/env bash
set -euo pipefail

# build.sh - nself build command wrapper
# This is now a thin wrapper that delegates to modular components

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
LIB_DIR="$SCRIPT_DIR/../lib/build"

# Check if build library exists
if [[ ! -d "$LIB_DIR" ]]; then
  echo "Error: Build library not found at $LIB_DIR" >&2
  exit 1
fi

# Debug output
if [[ "${DEBUG:-}" == "true" ]]; then
  echo "DEBUG: LIB_DIR=$LIB_DIR" >&2
  echo "DEBUG: Loading core.sh..." >&2
fi

# Source the core build module
if [[ -f "$LIB_DIR/core.sh" ]]; then
  source "$LIB_DIR/core.sh"
else
  echo "Error: Core build module not found at $LIB_DIR/core.sh" >&2
  exit 1
fi

# Display utilities are already sourced via core.sh

# Show help for build command
show_build_help() {
  echo "nself build - Generate project infrastructure and configuration"
  echo ""
  echo "Usage: nself build [OPTIONS]"
  echo ""
  echo "Description:"
  echo "  Generates Docker Compose files, SSL certificates, nginx configuration,"
  echo "  and all necessary infrastructure based on your .env settings."
  echo ""
  echo "Options:"
  echo "  -f, --force         Force rebuild of all components"
  echo "  -h, --help          Show this help message"
  echo "  -v, --verbose       Show detailed output"
  echo "  --debug             Enable debug mode"
  echo ""
  echo "Examples:"
  echo "  nself build                    # Build with current configuration"
  echo "  nself build --force            # Force rebuild everything"
  echo "  nself build --debug            # Build with debug output"
  echo ""
  echo "Files Generated:"
  echo "  • docker-compose.yml           • nginx/ configuration"
  echo "  • SSL certificates             • Database initialization"
  echo "  • Service templates            • Environment validation"
  echo ""
  echo "Notes:"
  echo "  • Automatically detects configuration changes"
  echo "  • Only rebuilds what's necessary (unless --force)"
  echo "  • Validates configuration before building"
  echo "  • Creates trusted SSL certificates for HTTPS"
}

# Main build command function
cmd_build() {
  local force_rebuild=false
  local verbose=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
      -f | --force)
        force_rebuild=true
        shift
        ;;
      -h | --help)
        show_build_help
        return 0
        ;;
      -v | --verbose)
        verbose=true
        export VERBOSE=true
        shift
        ;;
      --debug)
        export DEBUG=true
        export VERBOSE=true
        verbose=true
        shift
        ;;
      *)
        echo "Error: Unknown option: $1" >&2
        echo "Use 'nself build --help' for usage information" >&2
        return 1
        ;;
    esac
  done

  # Get project name from env or use basename
  local project_name="${PROJECT_NAME:-$(basename "$PWD")}"
  local env="${ENV:-dev}"

  # Run the orchestrated build
  local build_result
  orchestrate_build "$project_name" "$env" "$force_rebuild" "$verbose"
  build_result=$?

  return $build_result
}

# Export the main command
export -f cmd_build

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_build "$@"
fi