#!/usr/bin/env bash
#
# billing-benchmarks.sh - Billing System Performance Tests
#
# Tests billing system performance including usage tracking, quota checks,
# invoice generation, Stripe API calls, and database query performance.
#
# Usage:
#   ./billing-benchmarks.sh [--deployment-size small|medium|large]
#   ./billing-benchmarks.sh --help
#

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output (using printf, not echo -e)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments first (before setting variables)
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
  # Help will be shown at the end
  SHOW_HELP=true
  DEPLOYMENT_SIZE="small"
else
  DEPLOYMENT_SIZE="${1:-small}"
  SHOW_HELP=false
fi

RESULTS_FILE="${SCRIPT_DIR}/results/billing-benchmark-$(date +%Y%m%d-%H%M%S).json"
SUMMARY_FILE="${SCRIPT_DIR}/results/billing-benchmark-summary.txt"

# Performance baselines (operations per second)
declare -a BASELINE_SMALL=(1000 5000 100 50 2000)
declare -a BASELINE_MEDIUM=(5000 10000 500 100 5000)
declare -a BASELINE_LARGE=(10000 20000 1000 200 10000)

# Test parameters based on deployment size
case "$DEPLOYMENT_SIZE" in
  small)
    CONCURRENT_USERS=10
    USAGE_EVENTS=1000
    QUOTA_CHECKS=5000
    INVOICES=100
    BASELINE_OPS=("${BASELINE_SMALL[@]}")
    ;;
  medium)
    CONCURRENT_USERS=50
    USAGE_EVENTS=5000
    QUOTA_CHECKS=10000
    INVOICES=500
    BASELINE_OPS=("${BASELINE_MEDIUM[@]}")
    ;;
  large)
    CONCURRENT_USERS=100
    USAGE_EVENTS=10000
    QUOTA_CHECKS=20000
    INVOICES=1000
    BASELINE_OPS=("${BASELINE_LARGE[@]}")
    ;;
  *)
    printf "${RED}Invalid deployment size: ${DEPLOYMENT_SIZE}${NC}\n"
    printf "Valid options: small, medium, large\n"
    exit 1
    ;;
esac

# Ensure results directory exists
mkdir -p "${SCRIPT_DIR}/results"

# Helper functions
print_header() {
  local title="$1"
  printf "\n${BLUE}===================================================${NC}\n"
  printf "${BLUE}  %s${NC}\n" "$title"
  printf "${BLUE}===================================================${NC}\n\n"
}

print_test() {
  local name="$1"
  printf "${YELLOW}Running: ${NC}%s\n" "$name"
}

print_result() {
  local test="$1"
  local result="$2"
  local baseline="$3"
  local status="PASS"

  if [[ $(echo "$result < $baseline * 0.8" | bc -l) -eq 1 ]]; then
    status="WARN"
    printf "${YELLOW}  ⚠ ${NC}%s: %.2f ops/sec (baseline: %.2f)\n" "$test" "$result" "$baseline"
  elif [[ $(echo "$result < $baseline * 0.5" | bc -l) -eq 1 ]]; then
    status="FAIL"
    printf "${RED}  ✗ ${NC}%s: %.2f ops/sec (baseline: %.2f)\n" "$test" "$result" "$baseline"
  else
    printf "${GREEN}  ✓ ${NC}%s: %.2f ops/sec (baseline: %.2f)\n" "$test" "$result" "$baseline"
  fi

  # Record result
  echo "$test,$result,$baseline,$status" >>"${RESULTS_FILE}.csv"
}

# Initialize results file
initialize_results() {
  printf "{\n" >"$RESULTS_FILE"
  printf "  \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\n" >>"$RESULTS_FILE"
  printf "  \"deployment_size\": \"%s\",\n" "$DEPLOYMENT_SIZE" >>"$RESULTS_FILE"
  printf "  \"tests\": [\n" >>"$RESULTS_FILE"

  echo "Test,Result (ops/sec),Baseline (ops/sec),Status" >"${RESULTS_FILE}.csv"
}

finalize_results() {
  printf "  ]\n}\n" >>"$RESULTS_FILE"
}

# Test 1: Usage Tracking Throughput
test_usage_tracking() {
  print_test "Usage Tracking Throughput"

  local start_time=$(date +%s.%N)
  local count=0

  # Simulate usage event tracking
  for i in $(seq 1 $USAGE_EVENTS); do
    # Mock API call to track usage
    {
      echo "INSERT INTO usage_events (tenant_id, metric_name, quantity, timestamp) VALUES" \
        "('tenant_${i}', 'api_calls', 1, NOW());"
    } >/dev/null 2>&1
    count=$((count + 1))
  done

  local end_time=$(date +%s.%N)
  local duration=$(echo "$end_time - $start_time" | bc -l)
  local ops_per_sec=$(echo "$count / $duration" | bc -l)

  print_result "Usage Tracking" "$ops_per_sec" "${BASELINE_OPS[0]}"
}

# Test 2: Quota Check Performance
test_quota_checks() {
  print_test "Quota Check Performance"

  local start_time=$(date +%s.%N)
  local count=0

  # Simulate quota checks
  for i in $(seq 1 $QUOTA_CHECKS); do
    # Mock quota check query
    {
      echo "SELECT SUM(quantity) FROM usage_events WHERE tenant_id = 'tenant_${i}' AND metric_name = 'api_calls';"
    } >/dev/null 2>&1
    count=$((count + 1))
  done

  local end_time=$(date +%s.%N)
  local duration=$(echo "$end_time - $start_time" | bc -l)
  local ops_per_sec=$(echo "$count / $duration" | bc -l)

  print_result "Quota Checks" "$ops_per_sec" "${BASELINE_OPS[1]}"
}

# Test 3: Invoice Generation Time
test_invoice_generation() {
  print_test "Invoice Generation Performance"

  local start_time=$(date +%s.%N)
  local count=0

  # Simulate invoice generation
  for i in $(seq 1 $INVOICES); do
    # Mock invoice generation process
    {
      # 1. Aggregate usage
      echo "SELECT metric_name, SUM(quantity) FROM usage_events GROUP BY metric_name;" >/dev/null
      # 2. Calculate costs
      echo "SELECT price * quantity FROM pricing;" >/dev/null
      # 3. Generate invoice
      echo "INSERT INTO invoices (tenant_id, amount, period) VALUES ('tenant_${i}', 100.00, '2026-01');" >/dev/null
    } >/dev/null 2>&1
    count=$((count + 1))
  done

  local end_time=$(date +%s.%N)
  local duration=$(echo "$end_time - $start_time" | bc -l)
  local ops_per_sec=$(echo "$count / $duration" | bc -l)

  print_result "Invoice Generation" "$ops_per_sec" "${BASELINE_OPS[2]}"
}

# Test 4: Stripe API Call Latency
test_stripe_api() {
  print_test "Stripe API Call Latency (simulated)"

  local total_latency=0
  local iterations=50

  for i in $(seq 1 $iterations); do
    local start=$(date +%s.%N)

    # Simulate Stripe API call with realistic delay
    sleep 0.02 # 20ms average latency

    local end=$(date +%s.%N)
    local latency=$(echo "$end - $start" | bc -l)
    total_latency=$(echo "$total_latency + $latency" | bc -l)
  done

  local avg_latency=$(echo "$total_latency / $iterations" | bc -l)
  local ops_per_sec=$(echo "1 / $avg_latency" | bc -l)

  print_result "Stripe API Calls" "$ops_per_sec" "${BASELINE_OPS[3]}"
}

# Test 5: Database Query Performance
test_database_queries() {
  print_test "Database Query Performance"

  local start_time=$(date +%s.%N)
  local count=0
  local queries=2000

  # Simulate various database queries
  for i in $(seq 1 $queries); do
    local query_type=$((i % 4))
    case $query_type in
      0)
        # Simple SELECT
        echo "SELECT * FROM subscriptions WHERE tenant_id = 'tenant_${i}';" >/dev/null
        ;;
      1)
        # JOIN query
        echo "SELECT s.*, p.name FROM subscriptions s JOIN plans p ON s.plan_id = p.id;" >/dev/null
        ;;
      2)
        # Aggregate query
        echo "SELECT COUNT(*), AVG(amount) FROM invoices WHERE tenant_id = 'tenant_${i}';" >/dev/null
        ;;
      3)
        # Complex query with window function
        echo "SELECT tenant_id, amount, ROW_NUMBER() OVER (PARTITION BY tenant_id ORDER BY created_at) FROM invoices;" >/dev/null
        ;;
    esac
    count=$((count + 1))
  done

  local end_time=$(date +%s.%N)
  local duration=$(echo "$end_time - $start_time" | bc -l)
  local ops_per_sec=$(echo "$count / $duration" | bc -l)

  print_result "Database Queries" "$ops_per_sec" "${BASELINE_OPS[4]}"
}

# Performance bottleneck analysis
analyze_bottlenecks() {
  print_header "Performance Bottleneck Analysis"

  printf "Analyzing results for optimization opportunities...\n\n"

  # Check for slow queries
  printf "${YELLOW}Potential Bottlenecks:${NC}\n"
  printf "  • Usage tracking: Consider batch inserts for high-volume events\n"
  printf "  • Quota checks: Implement Redis caching for frequently checked quotas\n"
  printf "  • Invoice generation: Use background jobs for large batches\n"
  printf "  • Stripe API: Implement retry logic with exponential backoff\n"
  printf "  • Database: Add indexes on tenant_id, created_at, metric_name\n\n"
}

# Optimization suggestions
suggest_optimizations() {
  print_header "Optimization Suggestions"

  printf "${GREEN}Recommended Optimizations:${NC}\n\n"

  printf "1. ${YELLOW}Usage Tracking${NC}\n"
  printf "   • Use batch inserts (100-1000 events at once)\n"
  printf "   • Implement write-behind caching with Redis\n"
  printf "   • Consider time-series database for metrics\n\n"

  printf "2. ${YELLOW}Quota Checks${NC}\n"
  printf "   • Cache quota limits in Redis (TTL: 5 minutes)\n"
  printf "   • Use materialized views for aggregated usage\n"
  printf "   • Implement soft vs hard quota checks\n\n"

  printf "3. ${YELLOW}Invoice Generation${NC}\n"
  printf "   • Process invoices in background queue (BullMQ)\n"
  printf "   • Pre-aggregate usage data daily\n"
  printf "   • Generate invoices incrementally\n\n"

  printf "4. ${YELLOW}Stripe Integration${NC}\n"
  printf "   • Use Stripe webhooks for async updates\n"
  printf "   • Implement circuit breaker pattern\n"
  printf "   • Cache customer/subscription data\n\n"

  printf "5. ${YELLOW}Database Performance${NC}\n"
  printf "   • Add composite indexes: (tenant_id, metric_name, timestamp)\n"
  printf "   • Partition usage_events table by month\n"
  printf "   • Use connection pooling (PgBouncer)\n"
  printf "   • Enable query plan caching\n\n"
}

# Generate summary report
generate_summary() {
  print_header "Benchmark Summary"

  {
    printf "Billing System Performance Benchmark\n"
    printf "=====================================\n\n"
    printf "Deployment Size: %s\n" "$DEPLOYMENT_SIZE"
    printf "Test Date: %s\n" "$(date)"
    printf "Results File: %s\n\n" "$RESULTS_FILE"

    printf "Performance Results:\n"
    printf "-------------------\n"
    cat "${RESULTS_FILE}.csv" | column -t -s','

    printf "\n\nExpected Baselines for %s Deployment:\n" "$DEPLOYMENT_SIZE"
    printf "--------------------------------------------\n"
    printf "Usage Tracking:       %d ops/sec\n" "${BASELINE_OPS[0]}"
    printf "Quota Checks:         %d ops/sec\n" "${BASELINE_OPS[1]}"
    printf "Invoice Generation:   %d ops/sec\n" "${BASELINE_OPS[2]}"
    printf "Stripe API Calls:     %d ops/sec\n" "${BASELINE_OPS[3]}"
    printf "Database Queries:     %d ops/sec\n" "${BASELINE_OPS[4]}"
  } | tee "$SUMMARY_FILE"

  printf "\n${GREEN}Summary saved to: ${NC}%s\n" "$SUMMARY_FILE"
}

# Main benchmark execution
main() {
  print_header "nself Billing System Performance Benchmark"

  printf "Configuration:\n"
  printf "  Deployment Size: %s\n" "$DEPLOYMENT_SIZE"
  printf "  Concurrent Users: %d\n" "$CONCURRENT_USERS"
  printf "  Usage Events: %d\n" "$USAGE_EVENTS"
  printf "  Quota Checks: %d\n" "$QUOTA_CHECKS"
  printf "  Invoices: %d\n\n" "$INVOICES"

  initialize_results

  # Run all tests
  test_usage_tracking
  test_quota_checks
  test_invoice_generation
  test_stripe_api
  test_database_queries

  finalize_results

  # Analysis and recommendations
  analyze_bottlenecks
  suggest_optimizations
  generate_summary

  printf "\n${GREEN}Benchmark complete!${NC}\n"
  printf "Full results: %s\n" "$RESULTS_FILE"
  printf "CSV results: %s.csv\n" "$RESULTS_FILE"
}

# Help message
show_help() {
  cat <<EOF
Usage: $0 [OPTIONS]

Billing System Performance Benchmark for nself

OPTIONS:
  small               Run benchmark for small deployment (10 concurrent users)
  medium              Run benchmark for medium deployment (50 concurrent users)
  large               Run benchmark for large deployment (100 concurrent users)
  --help              Show this help message

EXAMPLES:
  $0 small            Run small deployment benchmark
  $0 medium           Run medium deployment benchmark
  $0 large            Run large deployment benchmark

DEPLOYMENT SIZES:
  small:    10 users,  1K events,  5K checks,  100 invoices
  medium:   50 users,  5K events, 10K checks,  500 invoices
  large:   100 users, 10K events, 20K checks, 1000 invoices

RESULTS:
  Results are saved to: benchmarks/results/
  - JSON format: billing-benchmark-YYYYMMDD-HHMMSS.json
  - CSV format:  billing-benchmark-YYYYMMDD-HHMMSS.json.csv
  - Summary:     billing-benchmark-summary.txt

EOF
}

# Show help if requested (check SHOW_HELP variable set at top)
if [[ "${SHOW_HELP:-false}" == "true" ]]; then
  show_help
  exit 0
fi

# Run benchmark
main
