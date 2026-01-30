#!/usr/bin/env bash
# Quick validation test for CLI output library
# Verifies basic functionality without interactive elements

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../../lib/utils/cli-output.sh"

echo "Testing CLI Output Library..."
echo

# Test basic messages
cli_success "Success test"
cli_error "Error test" 2>&1
cli_warning "Warning test" 2>&1
cli_info "Info test"
echo

# Test sections
cli_section "Test Section"
cli_header "Test Header"
echo

# Test box
cli_box "Test box"
echo

# Test list
cli_list_item "Item 1"
cli_list_numbered 1 "First"
echo

# Test table
cli_table_header "Col1" "Col2"
cli_table_row "A" "B"
cli_table_footer "Col1" "Col2"
echo

# Test summary
cli_summary "Test Complete" "Item 1" "Item 2"

# Test banner
cli_banner "Test Banner"

# Test utilities
cli_separator
cli_center "Centered" 60
cli_indent "Indented" 1
echo

# Test compatibility checks
echo "Checking Bash 3.2 compatibility..."

# Check for echo -e (excluding comments)
if grep -v '^#' "${SCRIPT_DIR}/../../lib/utils/cli-output.sh" | grep -q 'echo -e'; then
  echo "FAIL: Found non-portable echo -e in code"
  exit 1
fi

if grep -q 'declare -A' "${SCRIPT_DIR}/../../lib/utils/cli-output.sh"; then
  echo "FAIL: Found Bash 4+ associative arrays"
  exit 1
fi

if grep -q '\${[^}]*,,[^}]*}' "${SCRIPT_DIR}/../../lib/utils/cli-output.sh"; then
  echo "FAIL: Found Bash 4+ lowercase expansion"
  exit 1
fi

echo "✓ All compatibility checks passed"
echo
echo "✓ All quick tests passed!"
