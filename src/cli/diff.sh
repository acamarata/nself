#!/usr/bin/env bash
# diff.sh - Show configuration differences

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Command function
cmd_diff() {
    local file1="${1:-.env.local}"
    local file2="${2:-.env.example}"
    
    if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_diff_help
        return 0
    fi
    
    show_header "Configuration Differences"
    
    # Check if files exist
    if [[ ! -f "$file1" ]]; then
        log_error "File not found: $file1"
        return 1
    fi
    
    if [[ ! -f "$file2" ]]; then
        log_warning "Comparison file not found: $file2"
        log_info "Creating example file for comparison..."
        
        # Create example from current env
        grep -E '^[A-Z_]+=' "$file1" | sed 's/=.*/=EXAMPLE/' > "$file2"
    fi
    
    # Show differences
    log_info "Comparing: $file1 vs $file2"
    echo
    
    if command -v diff >/dev/null 2>&1; then
        diff -u "$file2" "$file1" | tail -n +3 || true
    else
        log_error "diff command not available"
        return 1
    fi
    
    echo
    
    # Show summary
    local vars1=$(grep -c '^[A-Z_]+=' "$file1" 2>/dev/null || echo 0)
    local vars2=$(grep -c '^[A-Z_]+=' "$file2" 2>/dev/null || echo 0)
    
    log_info "Summary:"
    echo "  $file1: $vars1 variables"
    echo "  $file2: $vars2 variables"
    
    # Check for missing required variables
    local missing=$(comm -13 <(grep '^[A-Z_]+=' "$file1" | cut -d= -f1 | sort) <(grep '^[A-Z_]+=' "$file2" | cut -d= -f1 | sort) 2>/dev/null)
    
    if [[ -n "$missing" ]]; then
        echo
        log_warning "Missing variables in $file1:"
        echo "$missing" | sed 's/^/  - /'
    fi
}

# Show help
show_diff_help() {
    echo "Usage: nself diff [file1] [file2]"
    echo
    echo "Show differences between configuration files"
    echo
    echo "Arguments:"
    echo "  file1    First file (default: .env.local)"
    echo "  file2    Second file (default: .env.example)"
    echo
    echo "Examples:"
    echo "  nself diff                          # Compare .env.local vs .env.example"
    echo "  nself diff .env.local .env.prod     # Compare local vs production"
}

# Export for use as library
export -f cmd_diff

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    pre_command "diff" || exit $?
    cmd_diff "$@"
    exit_code=$?
    post_command "diff" $exit_code
    exit $exit_code
fi
