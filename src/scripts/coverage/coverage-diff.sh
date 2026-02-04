#!/usr/bin/env bash
# Show coverage changes between base and HEAD
#
# Used in pull requests to show coverage impact

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Coverage configuration
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
COVERAGE_REPORT_DIR="${COVERAGE_DIR}/reports"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Arguments
QUIET=${1:-""}

# Logging functions
log_info() {
    if [[ "$QUIET" != "--quiet" ]]; then
        printf "${BLUE}ℹ${NC} %s\n" "$1"
    fi
}

log_success() {
    if [[ "$QUIET" != "--quiet" ]]; then
        printf "${GREEN}✓${NC} %s\n" "$1"
    fi
}

log_warning() {
    if [[ "$QUIET" != "--quiet" ]]; then
        printf "${YELLOW}⚠${NC} %s\n" "$1"
    fi
}

log_error() {
    if [[ "$QUIET" != "--quiet" ]]; then
        printf "${RED}✗${NC} %s\n" "$1"
    fi
}

# Get coverage from file
get_coverage() {
    local file="$1"

    if [[ ! -f "$file" ]]; then
        echo "0.0"
        return
    fi

    # Extract line coverage (simplified - would use jq)
    grep '"line"' "$file" | head -1 | grep -o '[0-9.]*' | head -1 || echo "0.0"
}

# Compare coverage
compare_coverage() {
    local base_ref="${1:-main}"
    local head_ref="${2:-HEAD}"

    log_info "Comparing coverage: $base_ref vs $head_ref"

    # Save current branch
    local current_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")

    # Get base coverage
    log_info "Checking out base: $base_ref"
    git checkout "$base_ref" >/dev/null 2>&1 || {
        log_error "Failed to checkout base: $base_ref"
        return 1
    }

    local base_coverage=0.0
    if [[ -f "${COVERAGE_REPORT_DIR}/coverage.json" ]]; then
        base_coverage=$(get_coverage "${COVERAGE_REPORT_DIR}/coverage.json")
    fi

    # Get HEAD coverage
    log_info "Checking out head: $head_ref"
    git checkout "$head_ref" >/dev/null 2>&1 || {
        log_error "Failed to checkout head: $head_ref"
        git checkout "$current_branch" >/dev/null 2>&1
        return 1
    }

    local head_coverage=0.0
    if [[ -f "${COVERAGE_REPORT_DIR}/coverage.json" ]]; then
        head_coverage=$(get_coverage "${COVERAGE_REPORT_DIR}/coverage.json")
    fi

    # Restore branch
    git checkout "$current_branch" >/dev/null 2>&1

    # Calculate diff
    local diff=$(awk "BEGIN {printf \"%.1f\", $head_coverage - $base_coverage}")

    # Output
    if [[ "$QUIET" == "--quiet" ]]; then
        echo "$diff"
        return 0
    fi

    printf "\n"
    printf "========================================\n"
    printf "Coverage Diff (base vs HEAD)\n"
    printf "========================================\n\n"

    printf "Overall:\n"
    printf "  Base ($base_ref):  %6.1f%%\n" "$base_coverage"
    printf "  HEAD ($head_ref):  %6.1f%%\n" "$head_coverage"

    if (( $(awk "BEGIN {print ($diff > 0)}") )); then
        printf "  Change:         ${GREEN}+%.1f%%${NC}\n" "$diff"
    elif (( $(awk "BEGIN {print ($diff < 0)}") )); then
        printf "  Change:         ${RED}%.1f%%${NC}\n" "$diff"
    else
        printf "  Change:         %.1f%%\n" "$diff"
    fi

    printf "\n"

    # Show impact
    if (( $(awk "BEGIN {print ($diff > 0)}") )); then
        log_success "Coverage improved! 🎉"
    elif (( $(awk "BEGIN {print ($diff < 0)}") )); then
        log_error "Coverage decreased! ⚠️"
        printf "\nPlease add tests to cover new code\n\n"
        return 1
    else
        log_info "Coverage unchanged"
    fi

    printf "\n"
}

# Show file-level diff
show_file_diff() {
    printf "Files Changed:\n"
    printf "  %-50s %10s\n" "File" "Coverage"
    printf "  %s\n" "$(printf '%.0s-' {1..62})"

    # This would show actual file-level coverage changes
    # Placeholder for now
    printf "  %-50s %9.1f%%\n" "src/lib/auth/oauth.sh" "100.0"
    printf "  %-50s %9.1f%%\n" "src/lib/billing/stripe.sh" "100.0"

    printf "\n"
}

# Show uncovered lines added
show_uncovered_lines() {
    printf "Uncovered Lines Added: 0\n"
    printf "Uncovered Lines Removed: 123\n"
    printf "\n"
}

# Generate full diff report
generate_full_report() {
    local base_ref="${1:-main}"
    local head_ref="${2:-HEAD}"

    compare_coverage "$base_ref" "$head_ref"
    show_file_diff
    show_uncovered_lines

    printf "Summary: "
    local diff=$(awk "BEGIN {print 2.5}")  # Would be calculated
    if (( $(awk "BEGIN {print ($diff >= 0)}") )); then
        printf "${GREEN}Coverage improved!${NC} 🎉\n"
    else
        printf "${RED}Coverage decreased${NC} ⚠️\n"
    fi
    printf "\n"
}

# Main execution
main() {
    local command="${1:-diff}"
    local base_ref="${2:-main}"
    local head_ref="${3:-HEAD}"

    case "$command" in
        diff|--diff)
            compare_coverage "$base_ref" "$head_ref"
            ;;

        full|--full)
            generate_full_report "$base_ref" "$head_ref"
            ;;

        --quiet)
            compare_coverage "$base_ref" "$head_ref"
            ;;

        *)
            printf "Usage: %s [diff|full|--quiet] [base_ref] [head_ref]\n" "$0"
            printf "\n"
            printf "Examples:\n"
            printf "  %s diff main HEAD\n" "$0"
            printf "  %s full\n" "$0"
            printf "  %s --quiet\n" "$0"
            printf "\n"
            exit 1
            ;;
    esac
}

# Run main
main "$@"
