#!/usr/bin/env bash
# Collect test coverage from all test suites
#
# This script runs all test suites with coverage tracking enabled
# and aggregates the results into unified coverage reports.

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Coverage configuration
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
COVERAGE_DATA_DIR="${COVERAGE_DIR}/data"
COVERAGE_REPORT_DIR="${COVERAGE_DIR}/reports"
COVERAGE_FILE="${COVERAGE_DATA_DIR}/coverage.dat"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Initialize coverage directories
init_coverage() {
    log_info "Initializing coverage tracking..."

    mkdir -p "$COVERAGE_DIR"
    mkdir -p "$COVERAGE_DATA_DIR"
    mkdir -p "$COVERAGE_REPORT_DIR"

    # Clean previous coverage data
    rm -f "$COVERAGE_DATA_DIR"/*.dat
    rm -f "$COVERAGE_DATA_DIR"/*.info

    export COVERAGE_ENABLED=true
    export COVERAGE_FILE="$COVERAGE_FILE"

    log_success "Coverage directories initialized"
}

# Check for coverage tools
check_coverage_tools() {
    log_info "Checking coverage tools..."

    local has_kcov=false
    local has_bashcov=false

    if command -v kcov >/dev/null 2>&1; then
        has_kcov=true
        log_success "kcov found: $(kcov --version 2>&1 | head -1)"
    else
        log_warning "kcov not found (optional)"
    fi

    if command -v bashcov >/dev/null 2>&1; then
        has_bashcov=true
        log_success "bashcov found: $(bashcov --version 2>&1 | head -1)"
    else
        log_warning "bashcov not found (optional)"
    fi

    if [[ "$has_kcov" == "false" ]] && [[ "$has_bashcov" == "false" ]]; then
        log_warning "No coverage tools found, using manual instrumentation"
        return 1
    fi

    return 0
}

# Run test suite with coverage
run_with_coverage() {
    local suite="$1"
    local suite_path="$2"

    log_info "Running ${suite} tests with coverage..."

    local suite_coverage_dir="${COVERAGE_DATA_DIR}/${suite}"
    mkdir -p "$suite_coverage_dir"

    # Use kcov if available
    if command -v kcov >/dev/null 2>&1; then
        log_info "Using kcov for ${suite}..."

        # Run with kcov
        kcov \
            --exclude-pattern=/usr,/tmp,/var \
            --exclude-line='NOCOV' \
            --include-path="${PROJECT_ROOT}/src" \
            "$suite_coverage_dir" \
            "$suite_path" 2>&1 | tee "${suite_coverage_dir}/output.log"

        local exit_code=$?

        if [[ $exit_code -eq 0 ]]; then
            log_success "${suite} tests completed with coverage"
        else
            log_error "${suite} tests failed (exit code: $exit_code)"
            return $exit_code
        fi
    else
        # Fallback: manual instrumentation
        log_info "Using manual instrumentation for ${suite}..."

        COVERAGE_SUITE="$suite" \
        COVERAGE_OUTPUT="${suite_coverage_dir}/manual-coverage.dat" \
            bash "$suite_path" 2>&1 | tee "${suite_coverage_dir}/output.log"

        local exit_code=$?

        if [[ $exit_code -eq 0 ]]; then
            log_success "${suite} tests completed"
        else
            log_error "${suite} tests failed (exit code: $exit_code)"
            return $exit_code
        fi
    fi

    return 0
}

# Run all test suites
run_all_suites() {
    log_info "Running all test suites..."

    local failed_suites=()
    local total_suites=0
    local passed_suites=0

    # Unit tests
    if [[ -d "${PROJECT_ROOT}/src/tests/unit" ]]; then
        total_suites=$((total_suites + 1))
        if run_with_coverage "unit" "${PROJECT_ROOT}/src/tests/run-init-tests.sh"; then
            passed_suites=$((passed_suites + 1))
        else
            failed_suites+=("unit")
        fi
    fi

    # Integration tests
    if [[ -d "${PROJECT_ROOT}/src/tests/integration" ]]; then
        total_suites=$((total_suites + 1))
        if [[ -f "${PROJECT_ROOT}/src/tests/integration/run-all.sh" ]]; then
            if run_with_coverage "integration" "${PROJECT_ROOT}/src/tests/integration/run-all.sh"; then
                passed_suites=$((passed_suites + 1))
            else
                failed_suites+=("integration")
            fi
        else
            log_warning "Integration test runner not found"
        fi
    fi

    # Security tests
    if [[ -d "${PROJECT_ROOT}/src/tests/security" ]]; then
        total_suites=$((total_suites + 1))
        if [[ -f "${PROJECT_ROOT}/src/tests/security/run-all.sh" ]]; then
            if run_with_coverage "security" "${PROJECT_ROOT}/src/tests/security/run-all.sh"; then
                passed_suites=$((passed_suites + 1))
            else
                failed_suites+=("security")
            fi
        else
            log_warning "Security test runner not found"
        fi
    fi

    # E2E tests
    if [[ -f "${PROJECT_ROOT}/src/tests/e2e-comprehensive.sh" ]]; then
        total_suites=$((total_suites + 1))
        if run_with_coverage "e2e" "${PROJECT_ROOT}/src/tests/e2e-comprehensive.sh"; then
            passed_suites=$((passed_suites + 1))
        else
            failed_suites+=("e2e")
        fi
    fi

    # Summary
    printf "\n"
    log_info "Test Suite Summary:"
    printf "  Total:  %d\n" "$total_suites"
    printf "  Passed: %d\n" "$passed_suites"
    printf "  Failed: %d\n" "${#failed_suites[@]}"

    if [[ ${#failed_suites[@]} -gt 0 ]]; then
        printf "\n"
        log_error "Failed suites: ${failed_suites[*]}"
        return 1
    fi

    log_success "All test suites passed!"
    return 0
}

# Merge coverage data
merge_coverage_data() {
    log_info "Merging coverage data..."

    local merged_file="${COVERAGE_DATA_DIR}/merged-coverage.info"

    if command -v lcov >/dev/null 2>&1; then
        # Use lcov to merge
        find "$COVERAGE_DATA_DIR" -name "*.info" -type f > "${COVERAGE_DATA_DIR}/file-list.txt"

        if [[ -s "${COVERAGE_DATA_DIR}/file-list.txt" ]]; then
            lcov --add-tracefile $(cat "${COVERAGE_DATA_DIR}/file-list.txt" | tr '\n' ' ') \
                 --output-file "$merged_file" 2>/dev/null || true

            if [[ -f "$merged_file" ]]; then
                log_success "Coverage data merged to: $merged_file"
            fi
        fi
    else
        log_warning "lcov not found, skipping merge"
    fi
}

# Generate coverage summary
generate_summary() {
    log_info "Generating coverage summary..."

    local summary_file="${COVERAGE_REPORT_DIR}/summary.txt"

    {
        printf "========================================\n"
        printf "nself Test Coverage Summary\n"
        printf "========================================\n"
        printf "Generated: %s\n\n" "$(date '+%Y-%m-%d %H:%M:%S')"

        printf "Coverage Data Location:\n"
        printf "  %s\n\n" "$COVERAGE_DATA_DIR"

        printf "Test Suites Executed:\n"
        find "$COVERAGE_DATA_DIR" -maxdepth 1 -type d -not -name "$(basename "$COVERAGE_DATA_DIR")" | while read -r suite_dir; do
            local suite_name=$(basename "$suite_dir")
            printf "  ✓ %s\n" "$suite_name"
        done

        printf "\n"
        printf "Next Steps:\n"
        printf "  1. Run ./src/scripts/coverage/generate-coverage-report.sh to create reports\n"
        printf "  2. Run ./src/scripts/coverage/verify-coverage.sh to check requirements\n"
        printf "  3. Open coverage/reports/html/index.html for detailed view\n"

    } > "$summary_file"

    cat "$summary_file"
    log_success "Summary saved to: $summary_file"
}

# Main execution
main() {
    printf "\n"
    log_info "=== nself Coverage Collection ==="
    printf "\n"

    # Initialize
    init_coverage

    # Check tools
    check_coverage_tools || log_warning "Limited coverage functionality available"

    # Run all test suites
    if ! run_all_suites; then
        log_error "Some test suites failed"
        exit 1
    fi

    # Merge coverage data
    merge_coverage_data

    # Generate summary
    generate_summary

    printf "\n"
    log_success "=== Coverage collection complete ==="
    printf "\n"
}

# Run main
main "$@"
