#!/usr/bin/env bash
# init.sh - Initialize a new nself project with environment files

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source minimal display utilities for colors and formatting
for file in \
    "$SCRIPT_DIR/../lib/utils/display.sh" \
    "$SCRIPT_DIR/../lib/utils/output-formatter.sh"; do
    if [[ -f "$file" ]]; then
        source "$file" 2>/dev/null || true
    fi
done

# Fallback colors and functions if display utilities not loaded
if ! type -t log_error >/dev/null 2>&1; then
    log_error() { echo "${RED}✗${RESET} $*" >&2; }
    log_info() { echo "${BLUE}ℹ${RESET} $*" >&2; }
    log_success() { echo "${GREEN}✓${RESET} $*" >&2; }
    log_secondary() { echo "${BLUE}✓${RESET} $*" >&2; }
fi

# Define colors - use if available, otherwise empty
GREEN="${COLOR_GREEN:-}"
RED="${COLOR_RED:-}"
YELLOW="${COLOR_YELLOW:-}"
BLUE="${COLOR_BLUE:-}"
CYAN="${COLOR_CYAN:-}"
RESET="${COLOR_RESET:-}"
BOLD="${COLOR_BOLD:-}"
DIM="${COLOR_DIM:-}"

# Command function
cmd_init() {
    # Check not in nself source directory
    if [[ -f "bin/nself" ]] && [[ -d "src/cli" ]] && [[ -f "install.sh" ]]; then
        echo ""
        log_error "Cannot run nself commands in the nself source repository!"
        echo ""
        echo "Please run from a separate project directory:"
        echo "  mkdir ~/myproject && cd ~/myproject"
        echo "  nself init"
        return 1
    fi
    
    # Check if project already exists (docker-compose.yml indicates built project)
    if [[ -f "docker-compose.yml" ]]; then
        echo ""
        log_error "Existing project detected (docker-compose.yml found)"
        echo ""
        log_info "This project has already been built."
        echo "Use 'nself status' to check services"
        echo "Use 'nself reset' if you want to start fresh"
        return 1
    fi
    
    # Show welcome banner
    echo ""
    echo "${BLUE}╔══════════════════════════════════════════════════════════╗${RESET}"
    echo "${BLUE}║                                                          ║${RESET}"
    echo "${BLUE}║   Welcome to ${CYAN}nself${BLUE} - Modern Full-Stack Platform          ║${RESET}"
    echo "${BLUE}║   Build production ready applications with ease          ║${RESET}"
    echo "${BLUE}║                                                          ║${RESET}"
    echo "${BLUE}╚══════════════════════════════════════════════════════════╝${RESET}"
    echo ""
    
    # Check if already initialized and backup if needed
    if [[ -f ".env.local" ]] || [[ -f ".env.example" ]]; then
        if [[ -f ".env.local" ]]; then
            mv .env.local .env.local.backup
        fi
        if [[ -f ".env.example" ]]; then
            mv .env.example .env.example.backup
        fi
        log_secondary "Existing env files backed up with .backup suffix"
    fi
    
    # Get templates directory - check multiple possible locations
    local TEMPLATES_DIR=""
    
    # Check if we're running from installed location (bin/nself)
    if [[ -d "$SCRIPT_DIR/../../bin/templates" ]]; then
        TEMPLATES_DIR="$SCRIPT_DIR/../../bin/templates"
    # Check if we're running from source (src/cli/init.sh)
    elif [[ -d "$SCRIPT_DIR/../templates" ]]; then
        TEMPLATES_DIR="$SCRIPT_DIR/../templates"
    else
        log_error "Cannot find templates directory"
        echo "Please ensure nself is properly installed"
        return 1
    fi
    
    # Verify template files exist
    if [[ ! -f "$TEMPLATES_DIR/.env.example" ]] || [[ ! -f "$TEMPLATES_DIR/.env.local" ]]; then
        log_error "Template files not found in $TEMPLATES_DIR"
        echo "Please ensure nself is properly installed"
        return 1
    fi
    
    # Copy template files
    cp "$TEMPLATES_DIR/.env.example" .env.example
    echo "${GREEN}✓${RESET} Created .env.example (reference documentation)"
    
    cp "$TEMPLATES_DIR/.env.local" .env.local
    echo "${GREEN}✓${RESET} Created .env.local (your configuration file)"
    echo ""
    
    # Quick Tips section
    echo "${CYAN}➞ Quick Tips${RESET}"
    echo ""
    echo "${YELLOW}⚡${RESET} Run ${BLUE}nself build${RESET} immediately - it works with defaults!"
    echo "${YELLOW}⚡${RESET} Use ${BLUE}nself prod${RESET} to generate secure production passwords"
    echo "${YELLOW}⚡${RESET} Check ${DIM}.env.example${RESET} for all available options"
    echo "${YELLOW}⚡${RESET} Run ${BLUE}nself help${RESET} for complete command documentation"
    echo ""
    
    # Next Steps section
    echo "${CYAN}➞ Next Steps${RESET}"
    echo ""
    echo "${BLUE}1.${RESET} Edit .env.local to customize (optional)"
    echo "   ${DIM}Only add what you want to change - defaults handle the rest${RESET}"
    echo ""
    echo "${BLUE}2.${RESET} nself build - Generate all project files"
    echo "   ${DIM}Creates Docker configs, service templates, and more${RESET}"
    echo ""
    echo "${BLUE}3.${RESET} nself up - Start your backend"
    echo "   ${DIM}Launches PostgreSQL, Hasura, and all configured services${RESET}"
    echo ""
    
    return 0
}

# Export for use as library
export -f cmd_init

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_init "$@"
fi