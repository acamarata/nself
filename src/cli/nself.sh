#!/usr/bin/env bash
# nself.sh - Main wrapper for NSELF commands

# Don't use strict mode until after sourcing
set -eo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source configuration and utilities with error handling
# Path adjusted for new structure: src/cli -> src/lib
for file in \
    "$SCRIPT_DIR/../lib/config/defaults.sh" \
    "$SCRIPT_DIR/../lib/config/constants.sh" \
    "$SCRIPT_DIR/../lib/utils/display.sh" \
    "$SCRIPT_DIR/../lib/utils/output-formatter.sh" \
    "$SCRIPT_DIR/../lib/auto-fix/config-validator-v2.sh" \
    "$SCRIPT_DIR/../lib/auto-fix/auto-fixer-v2.sh"; do
    if [[ -f "$file" ]]; then
        source "$file"
    fi
done

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
        --version|-v)
            command="version"
            ;;
        --help|-h)
            command="help"
            ;;
    esac
    
    # Map command to file (check multiple locations)
    local command_file=""
    
    # First check for direct command file in cli directory
    if [[ -f "$SCRIPT_DIR/${command}.sh" ]]; then
        command_file="$SCRIPT_DIR/${command}.sh"
    # Then check tools subdirectories (now in src/tools)
    elif [[ -f "$SCRIPT_DIR/../tools/dev/${command}.sh" ]]; then
        command_file="$SCRIPT_DIR/../tools/dev/${command}.sh"
    elif [[ -f "$SCRIPT_DIR/../tools/scaffold/${command}.sh" ]]; then
        command_file="$SCRIPT_DIR/../tools/scaffold/${command}.sh"
    elif [[ -f "$SCRIPT_DIR/../tools/validate/${command}.sh" ]]; then
        command_file="$SCRIPT_DIR/../tools/validate/${command}.sh"
    # Handle special naming cases
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
        echo "Run 'nself help' to see available commands"
        return 1
    fi
    
    # Execute command - prefer cmd_ function if exists
    local cmd_function="cmd_${command//-/_}"
    
    # Source the command file
    source "$command_file"
    
    # Execute the command
    if declare -f "$cmd_function" >/dev/null 2>&1; then
        "$cmd_function" "$@"
    else
        # Fallback: execute the file directly
        bash "$command_file" "$@"
    fi
}

# Run main function
main "$@"