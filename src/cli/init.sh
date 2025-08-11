#!/usr/bin/env bash
# init.sh - Initialize a new nself project with environment files only

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/config/smart-defaults.sh"

# Command function
cmd_init() {
    show_header "NSELF PROJECT INITIALIZATION"
    
    # Safety check: Don't run in nself repository
    if [[ -f "bin/nself" ]] && [[ -d "src/lib" ]] && [[ -d "docs" ]]; then
        log_error "Cannot initialize in the nself repository!"
        echo ""
        log_info "Please create a separate project directory:"
        log_info "  mkdir ~/myproject && cd ~/myproject"
        log_info "  nself init"
        return 1
    fi
    
    # Check if already initialized
    if [[ -f ".env.local" ]]; then
        log_warning "Project already initialized (.env.local exists)"
        read -p "Overwrite existing configuration? [y/N]: " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Initialization cancelled"
            return 1
        fi
        # Backup existing files
        [[ -f ".env.local" ]] && mv .env.local .env.local.old
        [[ -f ".env" ]] && mv .env .env.old
        [[ -f ".env.example" ]] && mv .env.example .env.example.old
        log_info "Existing files backed up with .old suffix"
    fi
    
    # Get templates directory
    local TEMPLATES_DIR="$SCRIPT_DIR/../../bin/templates"
    
    # Fallback to src/templates if bin/templates doesn't exist
    if [[ ! -d "$TEMPLATES_DIR" ]]; then
        TEMPLATES_DIR="$SCRIPT_DIR/../templates"
    fi
    
    # Copy .env.example (reference file)
    if [[ -f "$TEMPLATES_DIR/.env.example" ]]; then
        cp "$TEMPLATES_DIR/.env.example" .env.example
        log_success "Created .env.example (reference documentation)"
    fi
    
    # Copy minimal .env.local template
    if [[ -f "$TEMPLATES_DIR/.env.local" ]]; then
        cp "$TEMPLATES_DIR/.env.local" .env.local
        log_success "Created .env.local (your configuration file)"
    else
        # Create a minimal .env.local if template doesn't exist
        cat > .env.local << 'EOF'
#####################################
# nself Project Configuration
# 
# Add ONLY the values you want to override here.
# nself uses smart defaults for everything else.
# 
# See .env.example for all available options and their defaults.
# Full docs: https://nself.org/docs/configuration
#####################################

# Uncomment and modify only what you need:

# PROJECT_NAME=myproject
# BASE_DOMAIN=local.nself.org

# Enable optional services:
# REDIS_ENABLED=true
# DASHBOARD_ENABLED=true
# FUNCTIONS_ENABLED=true

# Enable microservices:
# SERVICES_ENABLED=true
# NESTJS_ENABLED=true
# NESTJS_SERVICES=api,worker
EOF
        log_success "Created .env.local with defaults"
    fi
    
    echo ""
    log_success "Project initialized successfully!"
    echo ""
    echo "📝 Configuration files created:"
    echo "   .env.local   - Your project configuration (edit this)"
    echo "   .env.example - Reference for all available options"
    echo ""
    echo "🚀 Next steps:"
    echo "   1. Edit .env.local to customize your project (optional)"
    echo "   2. Run: nself build  (generates all project files)"
    echo "   3. Run: nself up     (starts your backend)"
    echo ""
    echo "💡 Tips:"
    echo "   - You can run 'nself build' immediately - it works with defaults"
    echo "   - Only add to .env.local what you want to change"
    echo "   - Run 'nself prod' to generate production passwords"
    
    return 0
}

# Export for use as library
export -f cmd_init

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_init "$@"
fi