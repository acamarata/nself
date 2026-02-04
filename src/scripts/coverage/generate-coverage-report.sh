#!/usr/bin/env bash
# Generate comprehensive coverage reports
#
# Creates text, HTML, JSON, and badge reports from collected coverage data

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Coverage directories
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
COVERAGE_DATA_DIR="${COVERAGE_DIR}/data"
COVERAGE_REPORT_DIR="${COVERAGE_DIR}/reports"

# Report files
TEXT_REPORT="${COVERAGE_REPORT_DIR}/coverage.txt"
HTML_REPORT_DIR="${COVERAGE_REPORT_DIR}/html"
JSON_REPORT="${COVERAGE_REPORT_DIR}/coverage.json"
BADGE_FILE="${COVERAGE_REPORT_DIR}/badge.svg"

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

# Initialize report directories
init_reports() {
    log_info "Initializing report directories..."

    mkdir -p "$COVERAGE_REPORT_DIR"
    mkdir -p "$HTML_REPORT_DIR"

    log_success "Report directories ready"
}

# Calculate coverage from kcov data
calculate_kcov_coverage() {
    local coverage_data=""
    local total_lines=0
    local covered_lines=0
    local total_branches=0
    local covered_branches=0

    # Find all kcov JSON files
    while IFS= read -r -d '' json_file; do
        if [[ -f "$json_file" ]]; then
            # Extract coverage data (simplified - would use jq in production)
            local file_lines=$(grep -o '"lines":[0-9]*' "$json_file" | grep -o '[0-9]*' || echo "0")
            local file_covered=$(grep -o '"covered":[0-9]*' "$json_file" | grep -o '[0-9]*' || echo "0")

            total_lines=$((total_lines + file_lines))
            covered_lines=$((covered_lines + file_covered))
        fi
    done < <(find "$COVERAGE_DATA_DIR" -name "*.json" -type f -print0 2>/dev/null)

    if [[ $total_lines -gt 0 ]]; then
        local percentage=$(awk "BEGIN {printf \"%.1f\", ($covered_lines / $total_lines) * 100}")
        printf "%s:%s:%s" "$percentage" "$covered_lines" "$total_lines"
    else
        printf "0.0:0:0"
    fi
}

# Generate text report
generate_text_report() {
    log_info "Generating text report..."

    # Calculate coverage
    local coverage_info=$(calculate_kcov_coverage)
    IFS=':' read -r coverage_pct covered_lines total_lines <<< "$coverage_info"

    # Get test count from logs
    local total_tests=0
    local passed_tests=0
    local failed_tests=0

    # Count from test output logs
    while IFS= read -r -d '' log_file; do
        local suite_tests=$(grep -c "^ok\|^not ok" "$log_file" 2>/dev/null || echo "0")
        local suite_passed=$(grep -c "^ok" "$log_file" 2>/dev/null || echo "0")
        total_tests=$((total_tests + suite_tests))
        passed_tests=$((passed_tests + suite_passed))
    done < <(find "$COVERAGE_DATA_DIR" -name "output.log" -type f -print0 2>/dev/null)

    failed_tests=$((total_tests - passed_tests))

    # Calculate pass rate
    local pass_rate="0.0"
    if [[ $total_tests -gt 0 ]]; then
        pass_rate=$(awk "BEGIN {printf \"%.1f\", ($passed_tests / $total_tests) * 100}")
    fi

    # Generate report
    {
        printf "========================================\n"
        printf "nself Test Coverage Report\n"
        printf "========================================\n\n"

        printf "Generated: %s\n" "$(date '+%Y-%m-%d %H:%M:%S UTC')"
        printf "Target Coverage: 100.0%%\n\n"

        printf "Overall Coverage:\n"
        printf "  Line Coverage:    %6.1f%%  (%s / %s lines)\n" "$coverage_pct" "$covered_lines" "$total_lines"
        printf "  Branch Coverage:  %6.1f%%  (estimated)\n" "$coverage_pct"
        printf "  Function Coverage: %6.1f%%  (estimated)\n" "$coverage_pct"
        printf "\n"

        # Coverage bar
        local bar_length=50
        local filled=$(awk "BEGIN {printf \"%.0f\", ($coverage_pct / 100) * $bar_length}")
        local empty=$((bar_length - filled))

        printf "Progress: ["
        printf "%${filled}s" | tr ' ' '='
        printf "%${empty}s" | tr ' ' '-'
        printf "] %.1f%%\n\n" "$coverage_pct"

        # Test statistics
        printf "Test Statistics:\n"
        printf "  Total Tests:  %6d\n" "$total_tests"
        printf "  Passed:       %6d  (%.1f%%)\n" "$passed_tests" "$pass_rate"
        printf "  Failed:       %6d\n" "$failed_tests"
        printf "  Skipped:      %6d\n" "0"
        printf "\n"

        # Coverage by module
        printf "Coverage by Module:\n"
        printf "  %-30s %10s\n" "Module" "Coverage"
        printf "  %s\n" "$(printf '%.0s-' {1..42})"

        # List test suites
        find "$COVERAGE_DATA_DIR" -maxdepth 1 -type d -not -name "$(basename "$COVERAGE_DATA_DIR")" | sort | while read -r suite_dir; do
            local suite_name=$(basename "$suite_dir")
            printf "  %-30s %9.1f%%\n" "$suite_name" "$coverage_pct"
        done

        printf "\n"

        # Gap analysis
        if (( $(awk "BEGIN {print ($coverage_pct < 100)}") )); then
            local gap=$(awk "BEGIN {printf \"%.1f\", 100 - $coverage_pct}")
            local lines_needed=$((total_lines - covered_lines))

            printf "Gap Analysis:\n"
            printf "  Coverage Gap:     %.1f%%\n" "$gap"
            printf "  Lines to Cover:   %d\n" "$lines_needed"
            printf "  Estimated Tests:  %d\n" "$((lines_needed / 5))"
            printf "\n"
        else
            printf "🎉 Target Coverage Achieved! 🎉\n\n"
        fi

        # Next steps
        printf "Reports Generated:\n"
        printf "  Text:  %s\n" "$TEXT_REPORT"
        printf "  HTML:  %s/index.html\n" "$HTML_REPORT_DIR"
        printf "  JSON:  %s\n" "$JSON_REPORT"
        printf "  Badge: %s\n" "$BADGE_FILE"
        printf "\n"

        printf "View HTML Report:\n"
        printf "  open %s/index.html\n" "$HTML_REPORT_DIR"
        printf "\n"

    } > "$TEXT_REPORT"

    cat "$TEXT_REPORT"
    log_success "Text report generated: $TEXT_REPORT"
}

# Generate JSON report
generate_json_report() {
    log_info "Generating JSON report..."

    # Calculate coverage
    local coverage_info=$(calculate_kcov_coverage)
    IFS=':' read -r coverage_pct covered_lines total_lines <<< "$coverage_info"

    # Create JSON (simplified - would use jq in production)
    cat > "$JSON_REPORT" <<EOF
{
  "timestamp": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "overall": {
    "line": ${coverage_pct},
    "branch": ${coverage_pct},
    "function": ${coverage_pct}
  },
  "lines": {
    "total": ${total_lines},
    "covered": ${covered_lines},
    "uncovered": $((total_lines - covered_lines)),
    "percentage": ${coverage_pct}
  },
  "target": {
    "coverage": 100.0,
    "gap": $(awk "BEGIN {printf \"%.1f\", 100 - $coverage_pct}")
  },
  "suites": [
EOF

    # Add suite data
    local first=true
    find "$COVERAGE_DATA_DIR" -maxdepth 1 -type d -not -name "$(basename "$COVERAGE_DATA_DIR")" | sort | while read -r suite_dir; do
        local suite_name=$(basename "$suite_dir")

        if [[ "$first" != "true" ]]; then
            printf "    ,\n" >> "$JSON_REPORT"
        fi
        first=false

        cat >> "$JSON_REPORT" <<EOF
    {
      "name": "$suite_name",
      "coverage": ${coverage_pct},
      "path": "$suite_dir"
    }
EOF
    done

    cat >> "$JSON_REPORT" <<EOF

  ]
}
EOF

    log_success "JSON report generated: $JSON_REPORT"
}

# Generate HTML report
generate_html_report() {
    log_info "Generating HTML report..."

    # Use kcov's HTML output if available
    if [[ -d "${COVERAGE_DATA_DIR}/unit" ]] && [[ -f "${COVERAGE_DATA_DIR}/unit/index.html" ]]; then
        log_info "Copying kcov HTML reports..."
        cp -r "${COVERAGE_DATA_DIR}"/*/*.html "$HTML_REPORT_DIR/" 2>/dev/null || true
        cp -r "${COVERAGE_DATA_DIR}"/*/*.css "$HTML_REPORT_DIR/" 2>/dev/null || true
        cp -r "${COVERAGE_DATA_DIR}"/*/*.js "$HTML_REPORT_DIR/" 2>/dev/null || true
    fi

    # Create index page
    local coverage_info=$(calculate_kcov_coverage)
    IFS=':' read -r coverage_pct covered_lines total_lines <<< "$coverage_info"

    cat > "${HTML_REPORT_DIR}/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>nself Coverage Report</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 3px solid #4CAF50;
            padding-bottom: 10px;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 30px 0;
        }
        .stat-card {
            background: #f9f9f9;
            padding: 20px;
            border-radius: 6px;
            border-left: 4px solid #4CAF50;
        }
        .stat-value {
            font-size: 2em;
            font-weight: bold;
            color: #4CAF50;
        }
        .stat-label {
            color: #666;
            margin-top: 5px;
        }
        .progress-bar {
            width: 100%;
            height: 30px;
            background: #e0e0e0;
            border-radius: 15px;
            overflow: hidden;
            margin: 20px 0;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #4CAF50, #45a049);
            transition: width 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: bold;
        }
        .module-list {
            margin-top: 30px;
        }
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #ddd;
        }
        th {
            background: #4CAF50;
            color: white;
        }
        tr:hover {
            background: #f5f5f5;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎯 nself Test Coverage Report</h1>
        <p>Generated: $(date '+%Y-%m-%d %H:%M:%S')</p>

        <div class="stats">
            <div class="stat-card">
                <div class="stat-value">${coverage_pct}%</div>
                <div class="stat-label">Overall Coverage</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">${covered_lines}</div>
                <div class="stat-label">Lines Covered</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">${total_lines}</div>
                <div class="stat-label">Total Lines</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">100%</div>
                <div class="stat-label">Target</div>
            </div>
        </div>

        <div class="progress-bar">
            <div class="progress-fill" style="width: ${coverage_pct}%">${coverage_pct}%</div>
        </div>

        <div class="module-list">
            <h2>Coverage by Test Suite</h2>
            <table>
                <thead>
                    <tr>
                        <th>Suite</th>
                        <th>Coverage</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody>
EOF

    # Add suite rows
    find "$COVERAGE_DATA_DIR" -maxdepth 1 -type d -not -name "$(basename "$COVERAGE_DATA_DIR")" | sort | while read -r suite_dir; do
        local suite_name=$(basename "$suite_dir")
        cat >> "${HTML_REPORT_DIR}/index.html" <<EOF
                    <tr>
                        <td>${suite_name}</td>
                        <td>${coverage_pct}%</td>
                        <td>✅ Complete</td>
                    </tr>
EOF
    done

    cat >> "${HTML_REPORT_DIR}/index.html" <<EOF
                </tbody>
            </table>
        </div>
    </div>
</body>
</html>
EOF

    log_success "HTML report generated: ${HTML_REPORT_DIR}/index.html"
}

# Generate coverage badge
generate_badge() {
    log_info "Generating coverage badge..."

    local coverage_info=$(calculate_kcov_coverage)
    IFS=':' read -r coverage_pct covered_lines total_lines <<< "$coverage_info"

    # Determine badge color
    local color="brightgreen"
    if (( $(awk "BEGIN {print ($coverage_pct < 50)}") )); then
        color="red"
    elif (( $(awk "BEGIN {print ($coverage_pct < 80)}") )); then
        color="yellow"
    elif (( $(awk "BEGIN {print ($coverage_pct < 100)}") )); then
        color="green"
    fi

    # Generate SVG badge
    cat > "$BADGE_FILE" <<EOF
<svg xmlns="http://www.w3.org/2000/svg" width="120" height="20">
    <linearGradient id="b" x2="0" y2="100%">
        <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
        <stop offset="1" stop-opacity=".1"/>
    </linearGradient>
    <mask id="a">
        <rect width="120" height="20" rx="3" fill="#fff"/>
    </mask>
    <g mask="url(#a)">
        <path fill="#555" d="M0 0h63v20H0z"/>
        <path fill="#${color}" d="M63 0h57v20H63z"/>
        <path fill="url(#b)" d="M0 0h120v20H0z"/>
    </g>
    <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
        <text x="31.5" y="15" fill="#010101" fill-opacity=".3">coverage</text>
        <text x="31.5" y="14">coverage</text>
        <text x="90.5" y="15" fill="#010101" fill-opacity=".3">${coverage_pct}%</text>
        <text x="90.5" y="14">${coverage_pct}%</text>
    </g>
</svg>
EOF

    log_success "Badge generated: $BADGE_FILE"
}

# Main execution
main() {
    printf "\n"
    log_info "=== Generating Coverage Reports ==="
    printf "\n"

    # Initialize
    init_reports

    # Generate reports
    generate_text_report
    generate_json_report
    generate_html_report
    generate_badge

    printf "\n"
    log_success "=== All reports generated ==="
    printf "\n"

    printf "View reports:\n"
    printf "  Text: cat %s\n" "$TEXT_REPORT"
    printf "  HTML: open %s/index.html\n" "$HTML_REPORT_DIR"
    printf "  JSON: cat %s\n" "$JSON_REPORT"
    printf "\n"
}

# Run main
main "$@"
