#!/usr/bin/env bash
# cli-output-demo.sh - Demonstration of CLI output library features
#
# This script showcases all available functions and their visual output.
# Run this to see what each function produces.
#
# Usage: bash examples/cli-output-demo.sh

set -euo pipefail

# Source the library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../src/lib/utils/cli-output.sh"

# =============================================================================
# DEMO SECTIONS
# =============================================================================

demo_basic_messages() {
  cli_header "Basic Messages"

  cli_info "This demonstrates the basic message types:"
  cli_blank

  cli_success "This is a success message - used when operations complete"
  cli_error "This is an error message - used when operations fail"
  cli_warning "This is a warning message - used for non-critical issues"
  cli_info "This is an info message - used for general information"
  cli_message "This is a plain message - no icon, no color"
  cli_bold "This is bold text - used for emphasis"
  cli_dim "This is dimmed text - used for less important information"

  cli_blank
  cli_info "Debug messages only show when DEBUG=true:"
  DEBUG=true cli_debug "This is a debug message"
}

demo_sections() {
  cli_header "Sections and Headers"

  cli_info "Use these to organize output hierarchically:"
  cli_blank

  cli_section "Configuration Phase"
  cli_info "Section headers are good for major subsections"

  cli_section "Build Phase"
  cli_info "They provide visual separation and context"

  cli_blank
  cli_info "For multi-step processes, use step indicators:"
  cli_step 1 5 "Downloading dependencies"
  cli_step 2 5 "Running tests"
  cli_step 3 5 "Building artifacts"
  cli_step 4 5 "Running linters"
  cli_step 5 5 "Packaging application"
}

demo_boxes() {
  cli_header "Boxes"

  cli_info "Simple boxes with automatic width:"
  cli_blank

  cli_box "Default info box"
  cli_box "Success box" "success"
  cli_box "Error box" "error"
  cli_box "Warning box" "warning"

  cli_info "Detailed boxes with title and content:"
  cli_blank

  cli_box_detailed "Important Notice" "This is a detailed box that can contain longer text. The content will automatically wrap to fit within the standard 60-character width, making it perfect for displaying important information or instructions."

  cli_box_detailed "Quick Tip" "Use boxes to highlight important information that users shouldn't miss."
}

demo_lists() {
  cli_header "Lists"

  cli_section "Bullet Lists"
  cli_info "Use bullet lists for unordered items:"
  cli_list_item "PostgreSQL database"
  cli_list_item "Hasura GraphQL engine"
  cli_list_item "Authentication service"
  cli_list_item "Nginx reverse proxy"

  cli_section "Numbered Lists"
  cli_info "Use numbered lists for ordered steps:"
  cli_list_numbered 1 "Initialize the project"
  cli_list_numbered 2 "Configure environment"
  cli_list_numbered 3 "Build Docker containers"
  cli_list_numbered 4 "Start services"
  cli_list_numbered 5 "Verify deployment"

  cli_section "Checklists"
  cli_info "Use checklists to show task completion:"
  cli_list_checked "Docker installed and running"
  cli_list_checked "Environment variables configured"
  cli_list_unchecked "SSL certificates generated"
  cli_list_unchecked "Database migrations applied"
}

demo_tables() {
  cli_header "Tables"

  cli_info "Tables automatically calculate column widths:"
  cli_blank

  cli_table_header "Service" "Status" "Port" "Health"
  cli_table_row "postgres" "running" "5432" "healthy"
  cli_table_row "hasura" "running" "8080" "healthy"
  cli_table_row "auth" "running" "4000" "healthy"
  cli_table_row "nginx" "running" "443" "healthy"
  cli_table_row "redis" "stopped" "6379" "n/a"
  cli_table_footer "Service" "Status" "Port" "Health"
}

demo_progress() {
  cli_header "Progress Indicators"

  cli_section "Progress Bars"
  cli_info "Show progress for long-running tasks:"
  cli_blank

  local i
  for i in 0 10 25 50 75 90 100; do
    cli_progress "Building Docker images" $i 100
    sleep 0.3
  done

  cli_blank
  cli_section "Spinners"
  cli_info "Use spinners for indeterminate operations:"
  cli_blank

  local spinner_pid
  spinner_pid=$(cli_spinner_start "Pulling Docker images")
  sleep 3
  cli_spinner_stop "$spinner_pid" "Docker images pulled successfully"

  spinner_pid=$(cli_spinner_start "Running database migrations")
  sleep 2
  cli_spinner_stop "$spinner_pid" "Migrations applied successfully"
}

demo_special() {
  cli_header "Special Output"

  cli_section "Banners"
  cli_info "Use banners for major events or welcome messages:"
  cli_blank

  cli_banner "nself v1.0.0" "Modern Full-Stack Platform"

  cli_blank
  cli_section "Summaries"
  cli_info "Use summaries to conclude complex operations:"
  cli_blank

  cli_summary "Build Complete" \
    "Build time: 2m 34s" \
    "Services started: 5" \
    "Containers created: 25" \
    "Warnings: 0" \
    "Errors: 0"

  cli_blank
  cli_section "Separators"
  cli_info "Use separators to divide content:"
  cli_separator
  cli_info "Standard 60-character separator above"
  cli_separator 40
  cli_info "Custom 40-character separator above"
}

demo_utilities() {
  cli_header "Utility Functions"

  cli_section "Indentation"
  cli_info "Use indentation to show hierarchy:"
  cli_blank

  cli_message "Root level"
  cli_indent "First level" 1
  cli_indent "Second level" 2
  cli_indent "Third level" 3

  cli_blank
  cli_section "Centering"
  cli_info "Center text within a specific width:"
  cli_blank

  cli_separator
  cli_center "Centered Title" 60
  cli_separator

  cli_blank
  cli_section "Blank Lines"
  cli_info "Control spacing with blank lines:"
  cli_message "Before"
  cli_blank 3
  cli_message "After (3 blank lines)"

  cli_blank
  cli_section "Color Stripping"
  cli_info "Remove colors for file logging:"
  cli_blank

  local colored_text
  colored_text=$(cli_success "This has colors")
  cli_message "Colored: $colored_text"

  local stripped_text
  stripped_text=$(echo "$colored_text" | cli_strip_colors)
  cli_message "Stripped: $stripped_text"
}

demo_practical_example() {
  cli_header "Practical Example: Build Command"

  cli_info "This simulates a typical nself build command output:"
  cli_blank

  # Phase 1: Validation
  cli_section "Validation"
  local spinner_pid
  spinner_pid=$(cli_spinner_start "Validating environment")
  sleep 1
  cli_spinner_stop "$spinner_pid" "Environment validation complete"

  cli_list_checked "Docker installed"
  cli_list_checked "Docker Compose available"
  cli_list_checked "Environment file exists"
  cli_list_checked "Ports available"

  # Phase 2: Build
  cli_section "Build"
  cli_step 1 4 "Generating configuration files"
  sleep 0.5
  cli_success "Configuration files generated"

  cli_step 2 4 "Building Docker images"
  local i
  for i in 0 33 66 100; do
    cli_progress "Building images" $i 100
    sleep 0.3
  done

  cli_step 3 4 "Creating Docker network"
  sleep 0.3
  cli_success "Network created: nself_network"

  cli_step 4 4 "Generating SSL certificates"
  sleep 0.3
  cli_success "SSL certificates generated"

  # Phase 3: Summary
  cli_blank
  cli_summary "Build Complete" \
    "Total time: 2m 15s" \
    "Docker images: 25" \
    "Configuration files: 12" \
    "SSL certificates: Generated"

  cli_blank
  cli_box "Next Steps: Run 'nself start' to launch services" "success"
}

demo_error_handling() {
  cli_header "Error Handling Example"

  cli_info "This shows how to handle errors effectively:"
  cli_blank

  cli_section "Attempting Connection"
  spinner_pid=$(cli_spinner_start "Connecting to database")
  sleep 2
  cli_spinner_stop "$spinner_pid" "Connection attempt finished"

  # Simulate error
  cli_error "Failed to connect to database"
  cli_warning "Database may not be running"
  cli_blank
  cli_info "Troubleshooting steps:"
  cli_list_numbered 1 "Check if Docker is running"
  cli_list_numbered 2 "Verify database service is started"
  cli_list_numbered 3 "Check database logs for errors"
  cli_blank

  cli_box "Run 'nself logs postgres' to see database logs" "info"
}

demo_no_color() {
  cli_header "NO_COLOR Support"

  cli_info "The library respects the NO_COLOR environment variable"
  cli_blank

  cli_message "Current output is with colors enabled"
  cli_blank

  cli_info "To disable colors, set NO_COLOR=1:"
  cli_dim "$ export NO_COLOR=1"
  cli_dim "$ nself build"
  cli_blank

  cli_info "Try running this demo with NO_COLOR=1:"
  cli_dim "$ NO_COLOR=1 bash examples/cli-output-demo.sh"
}

# =============================================================================
# MAIN DEMO RUNNER
# =============================================================================

main() {
  # Welcome banner
  cli_banner "CLI Output Library Demo" "Showcasing all available functions"

  cli_info "This demo shows all available output functions"
  cli_info "Scroll through to see examples of each type"
  cli_blank

  # Run demos
  demo_basic_messages
  sleep 1

  demo_sections
  sleep 1

  demo_boxes
  sleep 1

  demo_lists
  sleep 1

  demo_tables
  sleep 1

  demo_progress
  sleep 1

  demo_special
  sleep 1

  demo_utilities
  sleep 1

  demo_practical_example
  sleep 1

  demo_error_handling
  sleep 1

  demo_no_color

  # Final message
  cli_blank 2
  cli_separator
  cli_success "Demo complete!"
  cli_info "For more information, see: docs/development/CLI-OUTPUT-LIBRARY.md"
  cli_separator
  cli_blank
}

# Run the demo
main "$@"
