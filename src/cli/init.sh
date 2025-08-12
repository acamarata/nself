#!/usr/bin/env bash
# init.sh - Initialize a new nself project with environment files only

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source files with error handling
for file in \
    "$SCRIPT_DIR/../lib/utils/display.sh" \
    "$SCRIPT_DIR/../lib/utils/output-formatter.sh" \
    "$SCRIPT_DIR/../lib/utils/platform.sh" \
    "$SCRIPT_DIR/../lib/utils/error-templates.sh" \
    "$SCRIPT_DIR/../lib/utils/preflight.sh" \
    "$SCRIPT_DIR/../lib/config/smart-defaults.sh" \
    "$SCRIPT_DIR/../lib/auto-fix/config-validator-v2.sh" \
    "$SCRIPT_DIR/../lib/auto-fix/auto-fixer-v2.sh"; do
    if [[ -f "$file" ]]; then
        source "$file"
    else
        echo "Warning: Missing file $file" >&2
    fi
done

# Command function
cmd_init() {
    # Run pre-flight checks
    if ! preflight_init; then
        return 1
    fi
    
    show_welcome_message
    
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
# ╔══════════════════════════════════════════════════════════╗
# ║               NSELF PROJECT CONFIGURATION                ║
# ║                                                          ║
# ║   Minimal config - nself uses smart defaults for rest.  ║
# ║   See .env.example for ALL available options.           ║
# ╚══════════════════════════════════════════════════════════╝

# Project name (lowercase, no spaces)
PROJECT_NAME=myproject

# Optional: Uncomment and modify these for production
# ─────────────────────────────────────────────────────

# ENV=prod                              # Switch to production mode
# BASE_DOMAIN=yourdomain.com           # Your custom domain

# Security (generate with: openssl rand -base64 32)
# POSTGRES_PASSWORD=changeme
# HASURA_GRAPHQL_ADMIN_SECRET=changeme
# HASURA_JWT_KEY=changeme-minimum-32-characters-long
# HASURA_JWT_TYPE=HS256

# Optional Services (uncomment to enable)
# ────────────────────────────────────────
# REDIS_ENABLED=true                    # Redis caching
# FUNCTIONS_ENABLED=true                # Serverless functions
# DASHBOARD_ENABLED=true                # Admin dashboard

# Microservices (uncomment to enable)
# ────────────────────────────────────
# SERVICES_ENABLED=true
# NESTJS_ENABLED=true
# NESTJS_SERVICES=api,webhooks
# BULLMQ_ENABLED=true
# BULLMQ_WORKERS=email-worker,payment-processor
# BULLMQ_DASHBOARD_ENABLED=true
# GOLANG_ENABLED=true
# GOLANG_SERVICES=gateway
# PYTHON_ENABLED=true
# PYTHON_SERVICES=ml-model

# For more options, see .env.example or run: nself help config
EOF
        log_success "Created .env.local with defaults"
    fi
    
    # Validate the created configuration
    format_info "Validating initial configuration..."
    VALIDATION_ERRORS=()
    VALIDATION_WARNINGS=()
    AUTO_FIXES=()
    
    run_validation .env.local
    
    # Apply auto-fixes if needed
    if [[ ${#AUTO_FIXES[@]} -gt 0 ]]; then
        format_info "Applying automatic fixes..."
        apply_all_fixes .env.local "${AUTO_FIXES[@]}"
    fi
    
    # Show platform-specific information
    format_section "System Information" 40
    echo "Platform: ${BOLD}$PLATFORM${RESET}"
    echo "Architecture: ${BOLD}$ARCH${RESET}"
    echo "Docker: ${BOLD}$(docker --version 2>/dev/null | cut -d' ' -f3 | cut -d',' -f1 || echo 'Not installed')${RESET}"
    echo "Memory: ${BOLD}$(get_available_memory_gb)GB available${RESET}"
    echo "Disk: ${BOLD}$(get_available_disk_gb)GB available${RESET}"
    
    show_success_banner "Project initialized successfully!"
    
    format_section "Configuration Files Created" 40
    echo "${GREEN}✓${RESET} ${BOLD}.env.local${RESET}   - Your project configuration"
    echo "${GREEN}✓${RESET} ${BOLD}.env.example${RESET} - Reference documentation"
    
    format_section "Next Steps" 40
    echo "${BLUE}1.${RESET} ${BOLD}Edit .env.local${RESET} to customize (optional)"
    echo "   ${DIM}Only add what you want to change - defaults handle the rest${RESET}"
    echo ""
    echo "${BLUE}2.${RESET} ${BOLD}nself build${RESET} - Generate all project files"
    echo "   ${DIM}Creates Docker configs, service templates, and more${RESET}"
    echo ""
    echo "${BLUE}3.${RESET} ${BOLD}nself up${RESET} - Start your backend"
    echo "   ${DIM}Launches PostgreSQL, Hasura, and all configured services${RESET}"
    
    format_section "Quick Tips" 40
    echo "${YELLOW}⚡${RESET} Run ${BOLD}nself build${RESET} immediately - it works with defaults!"
    echo "${YELLOW}⚡${RESET} Use ${BOLD}nself prod${RESET} to generate secure production passwords"
    echo "${YELLOW}⚡${RESET} Check ${BOLD}.env.example${RESET} for all available options"
    echo "${YELLOW}⚡${RESET} Run ${BOLD}nself help${RESET} for complete command documentation"
    
    return 0
}

# Export for use as library
export -f cmd_init

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_init "$@"
fi