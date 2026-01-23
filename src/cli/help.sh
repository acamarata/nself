#!/usr/bin/env bash
set -euo pipefail

# help.sh - Show help information

# Get script directory with absolute path
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_DIR="$CLI_SCRIPT_DIR"

# Source shared utilities
[[ -z "${DISPLAY_SOURCED:-}" ]] && source "$CLI_SCRIPT_DIR/../lib/utils/display.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"
[[ -z "${CONSTANTS_SOURCED:-}" ]] && source "$CLI_SCRIPT_DIR/../lib/config/constants.sh"

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
  # Get version from VERSION file
  local version="unknown"
  if [[ -f "$SCRIPT_DIR/../VERSION" ]]; then
    version=$(cat "$SCRIPT_DIR/../VERSION" 2>/dev/null || echo "unknown")
  fi

  show_command_header "nself v${version}" "Self-Hosted Infrastructure Manager"
  echo
  echo "Usage: nself <command> [options]"

  show_section "Core Commands"
  echo -e "  ${COLOR_BLUE}init${COLOR_RESET}          Initialize a new project"
  echo -e "  ${COLOR_BLUE}build${COLOR_RESET}         Build project structure and Docker images"
  echo -e "  ${COLOR_BLUE}start${COLOR_RESET}         Start all services"
  echo -e "  ${COLOR_BLUE}stop${COLOR_RESET}          Stop all services"
  echo -e "  ${COLOR_BLUE}restart${COLOR_RESET}       Restart all services"
  echo
  echo -e "  ${COLOR_BLUE}reset${COLOR_RESET}         Reset project to clean state"
  echo -e "  ${COLOR_BLUE}clean${COLOR_RESET}         Clean up Docker resources"
  echo -e "  ${COLOR_BLUE}restore${COLOR_RESET}       Restore configuration from backup"

  show_section "Status Commands"
  echo -e "  ${COLOR_BLUE}status${COLOR_RESET}        Show service status"
  echo -e "  ${COLOR_BLUE}logs${COLOR_RESET}          View service logs"
  echo -e "  ${COLOR_BLUE}exec${COLOR_RESET}          Execute commands in containers"
  echo -e "  ${COLOR_BLUE}urls${COLOR_RESET}          Show service URLs"
  echo
  echo -e "  ${COLOR_BLUE}doctor${COLOR_RESET}        Run system diagnostics"
  echo -e "  ${COLOR_BLUE}version${COLOR_RESET}       Show version information"
  echo -e "  ${COLOR_BLUE}update${COLOR_RESET}        Update nself to latest version"
  echo -e "  ${COLOR_BLUE}help${COLOR_RESET}          Show this help message"

  show_section "Management Commands"
  echo -e "  ${COLOR_BLUE}ssl${COLOR_RESET}           Manage SSL certificates"
  echo -e "  ${COLOR_BLUE}trust${COLOR_RESET}         Trust local SSL certificates"
  echo
  echo -e "  ${COLOR_BLUE}admin${COLOR_RESET}         Admin UI management"
  echo -e "  ${COLOR_DIM}email${COLOR_RESET}         ${COLOR_DIM}Email service configuration (» 0.4.1)${COLOR_RESET}"
  echo -e "  ${COLOR_DIM}search${COLOR_RESET}        ${COLOR_DIM}Search service management (» 0.4.1)${COLOR_RESET}"
  echo -e "  ${COLOR_DIM}functions${COLOR_RESET}     ${COLOR_DIM}Serverless functions setup (» 0.4.1)${COLOR_RESET}"
  echo -e "  ${COLOR_DIM}mlflow${COLOR_RESET}        ${COLOR_DIM}MLflow ML experiment tracking (» 0.4.1)${COLOR_RESET}"
  echo
  echo -e "  ${COLOR_DIM}metrics${COLOR_RESET}       ${COLOR_DIM}Complete monitoring stack (» 0.4.2)${COLOR_RESET}"
  echo -e "  ${COLOR_DIM}monitor${COLOR_RESET}       ${COLOR_DIM}Access monitoring dashboards (» 0.4.2)${COLOR_RESET}"
  echo
  show_section "Database Commands"
  printf "  ${COLOR_BLUE}db${COLOR_RESET}            Database tools (migrate, seed, mock, backup, restore, schema, types)\n"

  show_section "Deployment Commands"
  printf "  ${COLOR_BLUE}env${COLOR_RESET}           Environment management (local/staging/prod)\n"
  printf "  ${COLOR_BLUE}deploy${COLOR_RESET}        Deploy to staging or production\n"
  printf "  ${COLOR_BLUE}prod${COLOR_RESET}          Production configuration and hardening\n"
  printf "  ${COLOR_BLUE}staging${COLOR_RESET}       Staging environment management\n"
  printf "  ${COLOR_BLUE}sync${COLOR_RESET}          Sync data between environments\n"

  show_section "Provider Commands"
  printf "  ${COLOR_BLUE}providers${COLOR_RESET}     Configure cloud provider credentials\n"
  printf "  ${COLOR_BLUE}provision${COLOR_RESET}     Provision infrastructure on any provider\n"
  echo
  printf "  ${COLOR_DIM}scale${COLOR_RESET}         ${COLOR_DIM}Scaling management (» 0.4.6)${COLOR_RESET}\n"

  show_section "Utility Commands"
  printf "  ${COLOR_BLUE}ci${COLOR_RESET}            CI/CD workflow generation (GitHub/GitLab)\n"
  printf "  ${COLOR_BLUE}completion${COLOR_RESET}    Shell completion (bash/zsh/fish)\n"
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
