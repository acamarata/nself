#!/usr/bin/env bash
# help.sh - Show help information

# Source shared utilities (only if not already sourced by wrapper)
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
[[ -z "${DISPLAY_SOURCED:-}" ]] && source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"
[[ -z "${CONSTANTS_SOURCED:-}" ]] && source "$SCRIPT_DIR/../lib/config/constants.sh"

# Command function
cmd_help() {
  local command="${1:-}"

  if [[ -n "$command" ]]; then
    # Show help for specific command
    show_command_help "$command"
  else
    # Show general help
    show_general_help
  fi
}

# Show general help
show_general_help() {
  show_header "nself - Self-Hosted Infrastructure Manager"
  # Get version from VERSION file
  local version="unknown"
  if [[ -f "$SCRIPT_DIR/../VERSION" ]]; then
    version=$(cat "$SCRIPT_DIR/../VERSION" 2>/dev/null || echo "unknown")
  fi
  echo "Version: $version"
  echo "Usage: nself <command> [options]"

  show_section "Core Commands"
  echo -e "  ${COLOR_BLUE}init${COLOR_RESET}          Initialize a new project"
  echo -e "  ${COLOR_BLUE}build${COLOR_RESET}         Build project structure and Docker images"
  echo -e "  ${COLOR_BLUE}start${COLOR_RESET}         Start all services"
  echo -e "  ${COLOR_BLUE}stop${COLOR_RESET}          Stop all services"
  echo -e "  ${COLOR_BLUE}restart${COLOR_RESET}       Restart all services"
  echo -e "  ${COLOR_BLUE}status${COLOR_RESET}        Show service status"
  echo -e "  ${COLOR_BLUE}logs${COLOR_RESET}          View service logs"

  show_section "Management Commands"
  echo -e "  ${COLOR_BLUE}doctor${COLOR_RESET}        Run system diagnostics"
  echo -e "  ${COLOR_BLUE}db${COLOR_RESET}            Database operations"
  echo -e "  ${COLOR_BLUE}email${COLOR_RESET}         Email service configuration"
  echo -e "  ${COLOR_BLUE}urls${COLOR_RESET}          Show service URLs"
  echo -e "  ${COLOR_BLUE}prod${COLOR_RESET}          Configure for production deployment"
  echo -e "  ${COLOR_BLUE}trust${COLOR_RESET}         Manage SSL certificates"

  show_section "Development Commands"
  echo -e "  ${COLOR_BLUE}diff${COLOR_RESET}          Show configuration differences"
  echo -e "  ${COLOR_BLUE}reset${COLOR_RESET}         Reset project to clean state"

  show_section "Tool Commands"
  echo -e "  ${COLOR_BLUE}scaffold${COLOR_RESET}      Create new service from template"
  echo -e "  ${COLOR_BLUE}validate-env${COLOR_RESET}  Validate environment configuration"
  echo -e "  ${COLOR_BLUE}hot_reload${COLOR_RESET}    Enable hot reload for development"

  show_section "Other Commands"
  echo -e "  ${COLOR_BLUE}update${COLOR_RESET}        Update nself to latest version"
  echo -e "  ${COLOR_BLUE}version${COLOR_RESET}       Show version information"
  echo -e "  ${COLOR_BLUE}help${COLOR_RESET}          Show this help message"
  echo
  echo "For command-specific help: nself help <command>"
  echo "                      or: nself <command> --help"
}

# Show command-specific help
show_command_help() {
  local command="$1"
  local command_file=""

  # Find command file (check multiple locations)
  if [[ -f "$SCRIPT_DIR/${command}.sh" ]]; then
    command_file="$SCRIPT_DIR/${command}.sh"
  elif [[ -f "$SCRIPT_DIR/../tools/dev/${command}.sh" ]]; then
    command_file="$SCRIPT_DIR/../tools/dev/${command}.sh"
  elif [[ -f "$SCRIPT_DIR/../tools/scaffold/${command}.sh" ]]; then
    command_file="$SCRIPT_DIR/../tools/scaffold/${command}.sh"
  elif [[ -f "$SCRIPT_DIR/../tools/validate/${command}.sh" ]]; then
    command_file="$SCRIPT_DIR/../tools/validate/${command}.sh"
  elif [[ "$command" == "hot-reload" ]] && [[ -f "$SCRIPT_DIR/../tools/dev/hot_reload.sh" ]]; then
    command_file="$SCRIPT_DIR/../tools/dev/hot_reload.sh"
  elif [[ "$command" == "hot_reload" ]] && [[ -f "$SCRIPT_DIR/../tools/dev/hot_reload.sh" ]]; then
    command_file="$SCRIPT_DIR/../tools/dev/hot_reload.sh"
  elif [[ "$command" == "validate-env" ]] && [[ -f "$SCRIPT_DIR/../tools/validate/validate-env.sh" ]]; then
    command_file="$SCRIPT_DIR/../tools/validate/validate-env.sh"
  fi

  # Check if command exists
  if [[ -z "$command_file" ]] || [[ ! -f "$command_file" ]]; then
    log_error "Unknown command: $command"
    echo
    echo "Run 'nself help' to see available commands"
    return 1
  fi

  # Try to run the command with --help
  if bash "$command_file" --help 2>/dev/null; then
    return 0
  else
    # Fallback: show basic info
    echo "Help for: nself $command"
    echo
    echo "Run: nself $command --help"
    echo "Or check the documentation"
  fi
}

# Export for use as library
export -f cmd_help

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "help" || exit $?
  cmd_help "$@"
  exit_code=$?
  post_command "help" $exit_code
  exit $exit_code
fi
