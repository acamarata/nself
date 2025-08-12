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
    show_header "Nself - Self-Hosted Infrastructure Manager"
    
    # Get version from VERSION file
    local version="unknown"
    if [[ -f "$SCRIPT_DIR/../VERSION" ]]; then
        version=$(cat "$SCRIPT_DIR/../VERSION" 2>/dev/null || echo "unknown")
    fi
    echo "Version: ${NSELF_VERSION:-$version}"
    echo
    echo "Usage: nself <command> [options]"
    echo
    
    show_section "Core Commands"
    echo "  init          Initialize a new project"
    echo "  build         Build project structure and Docker images"
    echo "  up            Start all services"
    echo "  down          Stop all services"
    echo "  restart       Restart all services"
    echo "  status        Show service status"
    echo "  logs          View service logs"
    echo
    
    show_section "Management Commands"
    echo "  doctor        Run system diagnostics"
    echo "  db            Database operations"
    echo "  email         Email service configuration"
    echo "  urls          Show service URLs"
    echo "  prod          Configure for production deployment"
    echo "  trust         Manage SSL certificates"
    echo
    
    show_section "Development Commands"
    echo "  diff          Show configuration differences"
    echo "  reset         Reset project to clean state"
    echo
    
    show_section "Tool Commands"
    echo "  scaffold      Create new service from template"
    echo "  validate-env  Validate environment configuration"
    echo "  hot_reload    Enable hot reload for development"
    echo
    
    show_section "Other Commands"
    echo "  update        Update nself to latest version"
    echo "  version       Show version information"
    echo "  help          Show this help message"
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
