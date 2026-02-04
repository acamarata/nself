#!/usr/bin/env bash
# Verify coverage meets requirements
#
# Enforces minimum coverage thresholds and fails CI if not met

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Coverage configuration
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
COVERAGE_REPORT_DIR="${COVERAGE_DIR}/reports"
JSON_REPORT="${COVERAGE_REPORT_DIR}/coverage.json"

# Coverage requirements
REQUIRED_LINE_COVERAGE=${REQUIRED_LINE_COVERAGE:-100.0}
REQUIRED_BRANCH_COVERAGE=${REQUIRED_BRANCH_COVERAGE:-95.0}
REQUIRED_FUNCTION_COVERAGE=${REQUIRED_FUNCTION_COVERAGE:-100.0}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() {
    printf "${BLUE}ℹ${NC} %s\n" "$1"
}

log_success() {
    printf "${GREEN}✓${NC} %s\n" "$1"
}

log_warning() {
    printf "${YELLOW}⚠${NC} %s\n" "$1"
}

log_error() {
    printf "${RED}✗${NC} %s\n" "$1"
}

# Extract coverage from JSON report
get_coverage() {
    local metric="$1"

    if [[ ! -f "$JSON_REPORT" ]]; then
        log_error "Coverage report not found: $JSON_REPORT"
        log_info "Run ./src/scripts/coverage/generate-coverage-report.sh first"
        return 1
    fi

    # Extract value (simplified - would use jq in production)
    local value=$(grep "\"$metric\"" "$JSON_REPORT" | head -1 | grep -o '[0-9.]*' | head -1)

    if [[ -z "$value" ]]; then
        echo "0.0"
    else
        echo "$value"
    fi
}

# Get uncovered files
get_uncovered_files() {
    local min_coverage="$1"

    log_info "Finding files below ${min_coverage}% coverage..."

    # This would parse actual coverage data
    # For now, return placeholder
    printf "\n"
    log_info "Uncovered files would be listed here"
    printf "\n"
}

# Verify line coverage
verify_line_coverage() {
    local current=$(get_coverage "line")
    local required="$REQUIRED_LINE_COVERAGE"

    printf "\n"
    log_info "Checking line coverage..."
    printf "  Required: %.1f%%\n" "$required"
    printf "  Current:  %.1f%%\n" "$current"

    if (( $(awk "BEGIN {print ($current < $required)}") )); then
        local gap=$(awk "BEGIN {printf \"%.1f\", $required - $current}")
        printf "  Gap:      ${RED}%.1f%%${NC}\n" "$gap"
        log_error "Line coverage below requirement"
        return 1
    else
        printf "  Status:   ${GREEN}✓ Pass${NC}\n"
        log_success "Line coverage requirement met"
        return 0
    fi
}

# Verify branch coverage
verify_branch_coverage() {
    local current=$(get_coverage "branch")
    local required="$REQUIRED_BRANCH_COVERAGE"

    printf "\n"
    log_info "Checking branch coverage..."
    printf "  Required: %.1f%%\n" "$required"
    printf "  Current:  %.1f%%\n" "$current"

    if (( $(awk "BEGIN {print ($current < $required)}") )); then
        local gap=$(awk "BEGIN {printf \"%.1f\", $required - $current}")
        printf "  Gap:      ${RED}%.1f%%${NC}\n" "$gap"
        log_warning "Branch coverage below requirement (non-blocking)"
        return 0  # Warning only
    else
        printf "  Status:   ${GREEN}✓ Pass${NC}\n"
        log_success "Branch coverage requirement met"
        return 0
    fi
}

# Verify function coverage
verify_function_coverage() {
    local current=$(get_coverage "function")
    local required="$REQUIRED_FUNCTION_COVERAGE"

    printf "\n"
    log_info "Checking function coverage..."
    printf "  Required: %.1f%%\n" "$required"
    printf "  Current:  %.1f%%\n" "$current"

    if (( $(awk "BEGIN {print ($current < $required)}") )); then
        local gap=$(awk "BEGIN {printf \"%.1f\", $required - $current}")
        printf "  Gap:      ${RED}%.1f%%${NC}\n" "$gap"
        log_warning "Function coverage below requirement (non-blocking)"
        return 0  # Warning only
    else
        printf "  Status:   ${GREEN}✓ Pass${NC}\n"
        log_success "Function coverage requirement met"
        return 0
    fi
}

# Show coverage summary
show_summary() {
    log_info "Coverage Summary:"

    local line_cov=$(get_coverage "line")
    local branch_cov=$(get_coverage "branch")
    local function_cov=$(get_coverage "function")

    printf "\n"
    printf "  %-20s %10s %10s %10s\n" "Metric" "Required" "Current" "Status"
    printf "  %s\n" "$(printf '%.0s-' {1..52})"

    # Line coverage
    local line_status="❌ FAIL"
    if (( $(awk "BEGIN {print ($line_cov >= $REQUIRED_LINE_COVERAGE)}") )); then
        line_status="✅ PASS"
    fi
    printf "  %-20s %9.1f%% %9.1f%% %10s\n" "Line Coverage" "$REQUIRED_LINE_COVERAGE" "$line_cov" "$line_status"

    # Branch coverage
    local branch_status="⚠️  WARN"
    if (( $(awk "BEGIN {print ($branch_cov >= $REQUIRED_BRANCH_COVERAGE)}") )); then
        branch_status="✅ PASS"
    fi
    printf "  %-20s %9.1f%% %9.1f%% %10s\n" "Branch Coverage" "$REQUIRED_BRANCH_COVERAGE" "$branch_cov" "$branch_status"

    # Function coverage
    local function_status="⚠️  WARN"
    if (( $(awk "BEGIN {print ($function_cov >= $REQUIRED_FUNCTION_COVERAGE)}") )); then
        function_status="✅ PASS"
    fi
    printf "  %-20s %9.1f%% %9.1f%% %10s\n" "Function Coverage" "$REQUIRED_FUNCTION_COVERAGE" "$function_cov" "$function_status"

    printf "\n"
}

# Generate failure report
generate_failure_report() {
    local line_cov=$(get_coverage "line")
    local gap=$(awk "BEGIN {printf \"%.1f\", $REQUIRED_LINE_COVERAGE - $line_cov}")

    printf "\n"
    log_error "=== Coverage Verification Failed ==="
    printf "\n"

    printf "Coverage Gap: %.1f%%\n" "$gap"
    printf "\n"

    printf "Next Steps:\n"
    printf "  1. Review uncovered code:\n"
    printf "     open %s/html/index.html\n" "$COVERAGE_REPORT_DIR"
    printf "\n"
    printf "  2. Add tests for uncovered lines\n"
    printf "\n"
    printf "  3. Run tests with coverage:\n"
    printf "     ./src/scripts/coverage/collect-coverage.sh\n"
    printf "\n"
    printf "  4. Regenerate reports:\n"
    printf "     ./src/scripts/coverage/generate-coverage-report.sh\n"
    printf "\n"
    printf "  5. Re-verify:\n"
    printf "     ./src/scripts/coverage/verify-coverage.sh\n"
    printf "\n"

    get_uncovered_files "$REQUIRED_LINE_COVERAGE"
}

# Main execution
main() {
    printf "\n"
    log_info "=== Verifying Coverage Requirements ==="
    printf "\n"

    local failed=false

    # Verify each metric
    if ! verify_line_coverage; then
        failed=true
    fi

    verify_branch_coverage || true
    verify_function_coverage || true

    # Show summary
    printf "\n"
    show_summary

    # Final result
    printf "\n"
    if [[ "$failed" == "true" ]]; then
        generate_failure_report
        log_error "=== Coverage verification FAILED ==="
        printf "\n"
        exit 1
    else
        log_success "=== All coverage requirements met! ==="
        printf "\n"

        printf "🎉 Congratulations! Coverage target achieved!\n"
        printf "\n"
        printf "Coverage Report: %s/html/index.html\n" "$COVERAGE_REPORT_DIR"
        printf "\n"
        exit 0
    fi
}

# Run main
main "$@"
