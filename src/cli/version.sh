#!/usr/bin/env bash
# version.sh - Show version information

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

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
    local verbose="${1:-}"
    local version=$(get_version)
    
    if [[ "$verbose" == "--verbose" ]] || [[ "$verbose" == "-v" ]]; then
        show_header "Nself Version Information"
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
        echo "nself version $version"
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
