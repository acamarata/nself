#!/usr/bin/env bash
# nself.sh - Main wrapper for nself commands

# Use error handling but allow sourcing to fail gracefully
set -o pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Preserve the CLI script directory before sourcing other files
CLI_SCRIPT_DIR="$SCRIPT_DIR"

# Source configuration and utilities with error handling
# Path adjusted for new structure: src/cli -> src/lib
for file in \
  "$CLI_SCRIPT_DIR/../lib/config/defaults.sh" \
  "$CLI_SCRIPT_DIR/../lib/config/constants.sh" \
  "$CLI_SCRIPT_DIR/../lib/utils/display.sh" \
  "$CLI_SCRIPT_DIR/../lib/utils/output-formatter.sh" \
  "$CLI_SCRIPT_DIR/../lib/auto-fix/config-validator-v2.sh" \
  "$CLI_SCRIPT_DIR/../lib/auto-fix/auto-fixer-v2.sh"; do
  if [[ -f "$file" ]]; then
    source "$file"
  fi
done

# Restore the CLI script directory after sourcing
SCRIPT_DIR="$CLI_SCRIPT_DIR"

# Simple fallback if display.sh didn't load
if ! declare -f log_error >/dev/null; then
  log_error() { echo -e "\033[0;31m[ERROR]\033[0m $1" >&2; }
  log_info() { echo -e "\033[0;34m[INFO]\033[0m $1"; }
fi

# Main router function
main() {
  local command="${1:-help}"
  shift || true

  # CRITICAL: Prevent running nself in its own repository
  # Check multiple indicators to ensure we're not in nself source
  if [[ -f "bin/nself" ]] && [[ -d "src/cli" ]] && [[ -d "src/lib" ]] && [[ -f "install.sh" ]]; then
    log_error "FATAL: Cannot run nself commands in the nself source repository!"
    echo ""
    log_info "nself must be run in a separate project directory."
    log_info "To create a test project:"
    echo ""
    echo "  mkdir -p ~/test-project && cd ~/test-project"
    echo "  nself init"
    echo ""
    log_info "Or use the test directory:"
    echo "  mkdir -p ~/.nself/test && cd ~/.nself/test"
    echo "  nself init"
    exit 1
  fi

  # Additional safety check - look for nself source markers
  if [[ -f "src/cli/nself.sh" ]] || [[ -f "src/VERSION" && -d "src/templates" ]]; then
    log_error "FATAL: This appears to be the nself source directory!"
    log_error "Please run nself commands in a separate project directory."
    exit 1
  fi

  # Handle special flags
  case "$command" in
  -v)
    # -v flag should show just version number
    command="version"
    set -- "--short" "$@"
    ;;
  --version)
    # --version shows full verbose output
    command="version"
    ;;
  --help | -h)
    command="help"
    ;;
  esac

  # Map command to file
  local command_file=""

  # Check for command file in cli directory
  if [[ -f "$SCRIPT_DIR/${command}.sh" ]]; then
    command_file="$SCRIPT_DIR/${command}.sh"
  fi

  # Check if command exists
  if [[ -z "$command_file" ]] || [[ ! -f "$command_file" ]]; then
    log_error "Unknown command: $command"
    echo "Run 'nself help' to see available commands"
    return 1
  fi

  # Execute command - check if it uses cmd_ function pattern or direct execution
  local cmd_function="cmd_${command//-/_}"

  # Check if the file uses cmd_ function pattern
  if grep -q "^$cmd_function()" "$command_file" 2>/dev/null; then
    # Source the file and call the function
    source "$command_file"
    local result=$?
    if [[ $result -ne 0 ]]; then
      # Source failed
      return $result
    fi
    "$cmd_function" "$@"
  else
    # File executes directly - just run it with bash
    bash "$command_file" "$@"
  fi
}

# Run main function
main "$@"
