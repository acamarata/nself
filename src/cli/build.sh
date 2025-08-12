#!/usr/bin/env bash

# build.sh - Modular build system with robust error checking and auto-fixing
set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source environment utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/output-formatter.sh"
source "$SCRIPT_DIR/../lib/auto-fix/config-validator-v2.sh"
source "$SCRIPT_DIR/../lib/auto-fix/auto-fixer-v2.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Main build command function
cmd_build() {
    # Pre-command hook
    pre_command "build" || exit $?
    
    format_section "Build Process" 50
    
    # Run validation first
    format_step 1 3 "Validating configuration"
    run_validation .env.local
    local validation_result=$?
    
    # Apply auto-fixes if needed
    if [[ ${#AUTO_FIXES[@]} -gt 0 ]]; then
        format_step 2 3 "Applying auto-fixes"
        apply_all_fixes .env.local "${AUTO_FIXES[@]}"
        
        # Re-validate after fixes
        format_info "Re-validating configuration..."
        run_validation .env.local
        validation_result=$?
    else
        format_step 2 3 "No fixes needed"
    fi
    
    # Continue with build if validation passed
    if [[ $validation_result -eq 0 ]] || [[ ${#VALIDATION_ERRORS[@]} -eq 0 ]]; then
        format_step 3 3 "Building project"
        
        # Use the build-legacy script for now
        if [[ -f "$SCRIPT_DIR/build-legacy.sh" ]]; then
            # Run the legacy build with proper error handling and formatting
            bash "$SCRIPT_DIR/build-legacy.sh" "$@" 2>&1 | filter_output build
            local exit_code=${PIPESTATUS[0]}
            
            if [[ $exit_code -eq 0 ]]; then
                format_success "Build completed successfully!"
            else
                format_error "Build failed" "Check the errors above and try again"
            fi
        else
            format_error "Build system not found"
            exit 1
        fi
    else
        format_error "Cannot proceed with build due to validation errors" \
            "Fix the errors above or run with AUTO_FIX=true"
        exit_code=1
    fi
    
    # Post-command hook
    post_command "build" $exit_code
    exit $exit_code
}

# Export for use as library
export -f cmd_build

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd_build "$@"
fi