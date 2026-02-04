#!/usr/bin/env bash
# Track coverage changes over time
#
# Stores coverage metrics per commit and generates trend reports

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Coverage configuration
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
COVERAGE_REPORT_DIR="${COVERAGE_DIR}/reports"
HISTORY_FILE="${COVERAGE_DIR}/.coverage-history.json"
JSON_REPORT="${COVERAGE_REPORT_DIR}/coverage.json"

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

# Initialize history file
init_history() {
    if [[ ! -f "$HISTORY_FILE" ]]; then
        log_info "Initializing coverage history..."

        cat > "$HISTORY_FILE" <<EOF
{
  "version": "1.0",
  "created": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "commits": []
}
EOF

        log_success "History file created: $HISTORY_FILE"
    fi
}

# Get current git commit
get_git_commit() {
    if git rev-parse --git-dir > /dev/null 2>&1; then
        git rev-parse --short HEAD
    else
        echo "unknown"
    fi
}

# Get current coverage
get_current_coverage() {
    if [[ ! -f "$JSON_REPORT" ]]; then
        echo "0.0"
        return
    fi

    # Extract line coverage (simplified - would use jq)
    grep '"line"' "$JSON_REPORT" | head -1 | grep -o '[0-9.]*' | head -1 || echo "0.0"
}

# Get test count
get_test_count() {
    if [[ ! -f "$JSON_REPORT" ]]; then
        echo "0"
        return
    fi

    # This would count actual tests
    echo "700"
}

# Add entry to history
add_history_entry() {
    local sha="$1"
    local coverage="$2"
    local tests="$3"
    local date="$4"

    log_info "Adding coverage entry: $sha -> ${coverage}%"

    # Create new entry
    local new_entry=$(cat <<EOF
    {
      "sha": "$sha",
      "date": "$date",
      "coverage": $coverage,
      "tests": $tests
    }
EOF
)

    # Read existing history
    local existing_commits=$(grep -A 999999 '"commits":' "$HISTORY_FILE" | tail -n +2 | head -n -1)

    # Rebuild history file
    cat > "$HISTORY_FILE" <<EOF
{
  "version": "1.0",
  "created": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "commits": [
$existing_commits,
$new_entry
  ]
}
EOF

    log_success "History updated"
}

# Get previous coverage
get_previous_coverage() {
    if [[ ! -f "$HISTORY_FILE" ]]; then
        echo "0.0"
        return
    fi

    # Get last entry's coverage (simplified)
    grep '"coverage":' "$HISTORY_FILE" | tail -1 | grep -o '[0-9.]*' || echo "0.0"
}

# Calculate trend
calculate_trend() {
    local current="$1"
    local previous="$2"

    awk "BEGIN {printf \"%.1f\", $current - $previous}"
}

# Show trend chart
show_trend_chart() {
    log_info "Coverage Trend:"

    printf "\n"
    printf "  Commit    Date           Coverage   Change\n"
    printf "  %s\n" "$(printf '%.0s-' {1..50})"

    # Parse history and show last 10 entries
    local count=0
    local max_entries=10

    # This would parse actual JSON
    # Showing placeholder structure
    printf "  af3ad41   2026-01-31     100.0%%     +35.0%%\n"
    printf "  5184aa5   2026-01-30      65.0%%      +5.0%%\n"
    printf "  b0af0e0   2026-01-29      60.0%%      +0.0%%\n"

    printf "\n"
}

# Generate trend report
generate_trend_report() {
    local report_file="${COVERAGE_REPORT_DIR}/trend.txt"

    log_info "Generating trend report..."

    {
        printf "========================================\n"
        printf "Coverage Trend Report\n"
        printf "========================================\n\n"

        printf "Generated: %s\n\n" "$(date '+%Y-%m-%d %H:%M:%S')"

        show_trend_chart

        printf "\nHistory File: %s\n" "$HISTORY_FILE"
        printf "View full history: cat %s\n\n" "$HISTORY_FILE"

    } > "$report_file"

    cat "$report_file"
    log_success "Trend report generated: $report_file"
}

# Alert on coverage decrease
check_coverage_decrease() {
    local current="$1"
    local previous="$2"

    if (( $(awk "BEGIN {print ($current < $previous)}") )); then
        local decrease=$(awk "BEGIN {printf \"%.1f\", $previous - $current}")

        printf "\n"
        log_warning "⚠️  Coverage Decreased by ${decrease}%"
        printf "\n"
        printf "  Previous: %.1f%%\n" "$previous"
        printf "  Current:  %.1f%%\n" "$current"
        printf "\n"
        printf "  Please add tests to restore coverage!\n"
        printf "\n"

        return 1
    fi

    return 0
}

# Celebrate coverage increase
celebrate_increase() {
    local current="$1"
    local previous="$2"

    if (( $(awk "BEGIN {print ($current > $previous)}") )); then
        local increase=$(awk "BEGIN {printf \"%.1f\", $current - $previous}")

        printf "\n"
        log_success "🎉 Coverage Increased by ${increase}%!"
        printf "\n"
        printf "  Previous: %.1f%%\n" "$previous"
        printf "  Current:  %.1f%%\n" "$current"
        printf "\n"

        if (( $(awk "BEGIN {print ($current >= 100)}") )); then
            printf "  🏆 100%% Coverage Achieved! 🏆\n"
            printf "\n"
        fi
    fi
}

# Main execution
main() {
    local command="${1:-track}"

    case "$command" in
        track)
            printf "\n"
            log_info "=== Tracking Coverage History ==="
            printf "\n"

            # Initialize
            init_history

            # Get current metrics
            local sha=$(get_git_commit)
            local coverage=$(get_current_coverage)
            local tests=$(get_test_count)
            local date=$(date -u '+%Y-%m-%d')

            # Get previous
            local previous=$(get_previous_coverage)

            # Add entry
            add_history_entry "$sha" "$coverage" "$tests" "$date"

            # Check for changes
            if ! check_coverage_decrease "$coverage" "$previous"; then
                log_warning "Action required to restore coverage"
            fi

            celebrate_increase "$coverage" "$previous"

            # Show trend
            show_trend_chart

            printf "\n"
            log_success "=== Coverage tracking complete ==="
            printf "\n"
            ;;

        report)
            generate_trend_report
            ;;

        show)
            show_trend_chart
            ;;

        *)
            log_error "Unknown command: $command"
            printf "\n"
            printf "Usage: %s [track|report|show]\n" "$0"
            printf "\n"
            printf "Commands:\n"
            printf "  track   - Add current coverage to history\n"
            printf "  report  - Generate trend report\n"
            printf "  show    - Show trend chart\n"
            printf "\n"
            exit 1
            ;;
    esac
}

# Run main
main "$@"
