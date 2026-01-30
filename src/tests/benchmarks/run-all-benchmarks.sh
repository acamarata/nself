#!/usr/bin/env bash
#
# run-all-benchmarks.sh - Run All nself Benchmarks
#
# Runs all performance benchmarks in sequence and generates
# a consolidated report.
#
# Usage:
#   ./run-all-benchmarks.sh [--scale small|medium|large]
#   ./run-all-benchmarks.sh --help
#

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output (using printf, not echo -e)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse arguments first
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
  SHOW_HELP=true
  SCALE="medium"
else
  SCALE="${1:-medium}"
  SHOW_HELP=false
fi

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
CONSOLIDATED_REPORT="${SCRIPT_DIR}/results/all-benchmarks-${TIMESTAMP}.txt"

# Scale-specific parameters
case "$SCALE" in
  small)
    BILLING_SIZE="small"
    TENANT_COUNT=10
    CONNECTION_COUNT=100
    ;;
  medium)
    BILLING_SIZE="medium"
    TENANT_COUNT=100
    CONNECTION_COUNT=1000
    ;;
  large)
    BILLING_SIZE="large"
    TENANT_COUNT=1000
    CONNECTION_COUNT=10000
    ;;
  *)
    printf "${RED}Invalid scale: ${SCALE}${NC}\n"
    printf "Valid options: small, medium, large\n"
    exit 1
    ;;
esac

# Ensure results directory exists
mkdir -p "${SCRIPT_DIR}/results"

# Helper functions
print_header() {
  local title="$1"
  printf "\n${CYAN}=====================================================================${NC}\n"
  printf "${CYAN}  %s${NC}\n" "$title"
  printf "${CYAN}=====================================================================${NC}\n\n"
}

print_section() {
  local title="$1"
  printf "\n${BLUE}>>> %s${NC}\n" "$title"
}

# Show help
show_help() {
  cat <<EOF
Usage: $0 [SCALE]

Run All Performance Benchmarks for nself

ARGUMENTS:
  SCALE               Deployment scale (default: medium)
                      Options: small, medium, large

OPTIONS:
  --help              Show this help message

EXAMPLES:
  $0 small            Run small-scale benchmarks
  $0 medium           Run medium-scale benchmarks
  $0 large            Run large-scale benchmarks

SCALE CONFIGURATIONS:
  small:
    - Billing:        small deployment (10 users)
    - Tenants:        10 tenants
    - Connections:    100 concurrent connections

  medium:
    - Billing:        medium deployment (50 users)
    - Tenants:        100 tenants
    - Connections:    1000 concurrent connections

  large:
    - Billing:        large deployment (100 users)
    - Tenants:        1000 tenants
    - Connections:    10000 concurrent connections

BENCHMARKS RUN:
  1. Billing System Performance
  2. White-Label System Performance
  3. Multi-Tenant System Performance
  4. Real-Time System Performance

RESULTS:
  Individual results in: benchmarks/results/
  Consolidated report:   benchmarks/results/all-benchmarks-TIMESTAMP.txt

DURATION:
  Approximate runtime:
    small:    ~2 minutes
    medium:   ~5 minutes
    large:    ~15 minutes

EOF
}

# Show help if requested (check SHOW_HELP variable set at top)
if [[ "${SHOW_HELP:-false}" == "true" ]]; then
  show_help
  exit 0
fi

# Main benchmark execution
main() {
  print_header "nself Complete Performance Benchmark Suite"

  printf "Configuration:\n"
  printf "  Scale:              %s\n" "$SCALE"
  printf "  Billing Size:       %s\n" "$BILLING_SIZE"
  printf "  Tenant Count:       %d\n" "$TENANT_COUNT"
  printf "  Connections:        %d\n" "$CONNECTION_COUNT"
  printf "  Results File:       %s\n\n" "$CONSOLIDATED_REPORT"

  # Initialize consolidated report
  {
    printf "nself Performance Benchmark Suite\n"
    printf "==================================\n\n"
    printf "Scale:              %s\n" "$SCALE"
    printf "Test Date:          %s\n" "$(date)"
    printf "Timestamp:          %s\n\n" "$TIMESTAMP"
  } > "$CONSOLIDATED_REPORT"

  local start_time=$(date +%s)
  local failed=0

  # Benchmark 1: Billing System
  print_section "1/4 Running Billing System Benchmark..."
  if ./billing-benchmarks.sh "$BILLING_SIZE" >> "$CONSOLIDATED_REPORT" 2>&1; then
    printf "${GREEN}✓ Billing benchmark complete${NC}\n"
  else
    printf "${RED}✗ Billing benchmark failed${NC}\n"
    failed=$((failed + 1))
  fi

  # Benchmark 2: White-Label System
  print_section "2/4 Running White-Label System Benchmark..."
  if ./whitelabel-benchmarks.sh "$TENANT_COUNT" >> "$CONSOLIDATED_REPORT" 2>&1; then
    printf "${GREEN}✓ White-label benchmark complete${NC}\n"
  else
    printf "${RED}✗ White-label benchmark failed${NC}\n"
    failed=$((failed + 1))
  fi

  # Benchmark 3: Multi-Tenant System
  print_section "3/4 Running Multi-Tenant System Benchmark..."
  if ./tenant-benchmarks.sh "$TENANT_COUNT" >> "$CONSOLIDATED_REPORT" 2>&1; then
    printf "${GREEN}✓ Multi-tenant benchmark complete${NC}\n"
  else
    printf "${RED}✗ Multi-tenant benchmark failed${NC}\n"
    failed=$((failed + 1))
  fi

  # Benchmark 4: Real-Time System
  print_section "4/4 Running Real-Time System Benchmark..."
  if ./realtime-benchmarks.sh "$CONNECTION_COUNT" >> "$CONSOLIDATED_REPORT" 2>&1; then
    printf "${GREEN}✓ Real-time benchmark complete${NC}\n"
  else
    printf "${RED}✗ Real-time benchmark failed${NC}\n"
    failed=$((failed + 1))
  fi

  local end_time=$(date +%s)
  local duration=$((end_time - start_time))

  # Summary
  print_header "Benchmark Suite Complete"

  if [[ $failed -eq 0 ]]; then
    printf "${GREEN}✓ All benchmarks passed!${NC}\n\n"
  else
    printf "${RED}✗ %d benchmark(s) failed${NC}\n\n" "$failed"
  fi

  printf "Summary:\n"
  printf "  Duration:           %d seconds\n" "$duration"
  printf "  Failed Tests:       %d/4\n" "$failed"
  printf "  Consolidated Report: %s\n\n" "$CONSOLIDATED_REPORT"

  # Extract key metrics from individual results
  print_section "Key Performance Metrics"

  printf "\n${YELLOW}Billing System:${NC}\n"
  grep -E "Usage Tracking|Quota Checks|Invoice Generation" \
    "${SCRIPT_DIR}/results/billing-benchmark-summary.txt" 2>/dev/null | head -3 || printf "  (results not available)\n"

  printf "\n${YELLOW}White-Label System:${NC}\n"
  grep -E "Asset Loading|CSS Rendering|Theme Switching" \
    "${SCRIPT_DIR}/results/whitelabel-benchmark-summary.txt" 2>/dev/null | head -3 || printf "  (results not available)\n"

  printf "\n${YELLOW}Multi-Tenant System:${NC}\n"
  grep -E "Isolated Queries|RLS Prevention|Tenant Switching" \
    "${SCRIPT_DIR}/results/tenant-benchmark-summary.txt" 2>/dev/null | head -3 || printf "  (results not available)\n"

  printf "\n${YELLOW}Real-Time System:${NC}\n"
  grep -E "Connection Throughput|Message Latency|Presence Updates" \
    "${SCRIPT_DIR}/results/realtime-benchmark-summary.txt" 2>/dev/null | head -3 || printf "  (results not available)\n"

  printf "\n${BLUE}View individual summaries:${NC}\n"
  printf "  Billing:     %s\n" "${SCRIPT_DIR}/results/billing-benchmark-summary.txt"
  printf "  White-label: %s\n" "${SCRIPT_DIR}/results/whitelabel-benchmark-summary.txt"
  printf "  Multi-tenant: %s\n" "${SCRIPT_DIR}/results/tenant-benchmark-summary.txt"
  printf "  Real-time:   %s\n" "${SCRIPT_DIR}/results/realtime-benchmark-summary.txt"

  printf "\n${GREEN}Benchmark suite complete!${NC}\n"

  # Return exit code based on failures
  return $failed
}

# Run benchmarks
main
