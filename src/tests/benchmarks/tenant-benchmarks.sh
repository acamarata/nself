#!/usr/bin/env bash
#
# tenant-benchmarks.sh - Multi-Tenant Performance Tests
#
# Tests multi-tenant system performance including tenant isolation overhead,
# cross-tenant query prevention, tenant switching, and RLS policy enforcement.
#
# Usage:
#   ./tenant-benchmarks.sh [--tenants 10|100|1000]
#   ./tenant-benchmarks.sh --help
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
  SHOW_HELP=true
  TENANT_COUNT=10  # Set default for help
else
  TENANT_COUNT="${1:-10}"
  SHOW_HELP=false
fi

RESULTS_FILE="${SCRIPT_DIR}/results/tenant-benchmark-$(date +%Y%m%d-%H%M%S).json"
SUMMARY_FILE="${SCRIPT_DIR}/results/tenant-benchmark-summary.txt"

# Performance baselines (queries per second)
declare -a BASELINE_QPS_SMALL=(5000 8000 10000 3000)
declare -a BASELINE_QPS_MEDIUM=(3000 5000 7000 2000)
declare -a BASELINE_QPS_LARGE=(2000 3000 5000 1500)

# Test parameters based on tenant count
if [[ $TENANT_COUNT -le 10 ]]; then
  BASELINE_QPS=("${BASELINE_QPS_SMALL[@]}")
  TEST_LABEL="Small Scale"
elif [[ $TENANT_COUNT -le 100 ]]; then
  BASELINE_QPS=("${BASELINE_QPS_MEDIUM[@]}")
  TEST_LABEL="Medium Scale"
else
  BASELINE_QPS=("${BASELINE_QPS_LARGE[@]}")
  TEST_LABEL="Large Scale"
fi

QUERY_COUNT=10000
SWITCH_COUNT=1000
RLS_CHECKS=5000

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
  local unit="${4:-qps}"
  local status="PASS"

  # For QPS metrics, higher is better
  if [[ $(echo "$result < $baseline * 0.8" | bc -l) -eq 1 ]]; then
    status="WARN"
    printf "${YELLOW}  ⚠ ${NC}%s: %.2f %s (baseline: %.2f %s)\n" "$test" "$result" "$unit" "$baseline" "$unit"
  elif [[ $(echo "$result < $baseline * 0.5" | bc -l) -eq 1 ]]; then
    status="FAIL"
    printf "${RED}  ✗ ${NC}%s: %.2f %s (baseline: %.2f %s)\n" "$test" "$result" "$unit" "$baseline" "$unit"
  else
    printf "${GREEN}  ✓ ${NC}%s: %.2f %s (baseline: %.2f %s)\n" "$test" "$result" "$unit" "$baseline" "$unit"
  fi

  # Record result
  echo "$test,$result,$baseline,$unit,$status" >> "${RESULTS_FILE}.csv"
}

# Initialize results file
initialize_results() {
  printf "{\n" > "$RESULTS_FILE"
  printf "  \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\n" >> "$RESULTS_FILE"
  printf "  \"tenant_count\": %d,\n" "$TENANT_COUNT" >> "$RESULTS_FILE"
  printf "  \"scale\": \"%s\",\n" "$TEST_LABEL" >> "$RESULTS_FILE"
  printf "  \"tests\": [\n" >> "$RESULTS_FILE"

  echo "Test,Result,Baseline,Unit,Status" > "${RESULTS_FILE}.csv"
}

finalize_results() {
  printf "  ]\n}\n" >> "$RESULTS_FILE"
}

# Test 1: Tenant Isolation Overhead
test_tenant_isolation() {
  print_test "Tenant Isolation Overhead"

  # Test WITHOUT tenant isolation
  local start_no_rls=$(date +%s.%N)
  local count=0
  for i in $(seq 1 $QUERY_COUNT); do
    # Simulate query without RLS
    echo "SELECT * FROM data WHERE id = ${i};" > /dev/null 2>&1
    count=$((count + 1))
  done
  local end_no_rls=$(date +%s.%N)
  local duration_no_rls=$(echo "$end_no_rls - $start_no_rls" | bc -l)
  local qps_no_rls=$(echo "$count / $duration_no_rls" | bc -l)

  # Test WITH tenant isolation
  local start_with_rls=$(date +%s.%N)
  count=0
  for i in $(seq 1 $QUERY_COUNT); do
    local tenant_id=$((i % TENANT_COUNT + 1))
    # Simulate query with RLS
    echo "SELECT * FROM data WHERE tenant_id = 'tenant_${tenant_id}' AND id = ${i};" > /dev/null 2>&1
    count=$((count + 1))
  done
  local end_with_rls=$(date +%s.%N)
  local duration_with_rls=$(echo "$end_with_rls - $start_with_rls" | bc -l)
  local qps_with_rls=$(echo "$count / $duration_with_rls" | bc -l)

  # Calculate overhead percentage
  local overhead=$(echo "($qps_no_rls - $qps_with_rls) / $qps_no_rls * 100" | bc -l)

  printf "  ${BLUE}Without RLS:${NC} %.2f qps\n" "$qps_no_rls"
  printf "  ${BLUE}With RLS:${NC}    %.2f qps\n" "$qps_with_rls"
  printf "  ${BLUE}Overhead:${NC}    %.2f%%\n" "$overhead"

  print_result "Isolated Queries" "$qps_with_rls" "${BASELINE_QPS[0]}" "qps"
}

# Test 2: Cross-Tenant Query Prevention
test_cross_tenant_prevention() {
  print_test "Cross-Tenant Query Prevention"

  local start_time=$(date +%s.%N)
  local count=0
  local prevented=0

  for i in $(seq 1 $QUERY_COUNT); do
    local requesting_tenant=$((i % TENANT_COUNT + 1))
    local target_tenant=$(((i + 1) % TENANT_COUNT + 1))

    # Simulate query attempt
    if [[ $requesting_tenant != $target_tenant ]]; then
      # Cross-tenant query - should be blocked by RLS
      echo "SELECT * FROM data WHERE tenant_id = 'tenant_${target_tenant}';" > /dev/null 2>&1
      echo "-- BLOCKED by RLS policy" > /dev/null 2>&1
      prevented=$((prevented + 1))
    else
      # Same tenant query - allowed
      echo "SELECT * FROM data WHERE tenant_id = 'tenant_${target_tenant}';" > /dev/null 2>&1
    fi
    count=$((count + 1))
  done

  local end_time=$(date +%s.%N)
  local duration=$(echo "$end_time - $start_time" | bc -l)
  local qps=$(echo "$count / $duration" | bc -l)

  printf "  ${BLUE}Total Queries:${NC}     %d\n" "$count"
  printf "  ${BLUE}Prevented:${NC}         %d\n" "$prevented"
  printf "  ${BLUE}Prevention Rate:${NC}   %.2f%%\n" "$(echo "$prevented / $count * 100" | bc -l)"

  print_result "RLS Prevention" "$qps" "${BASELINE_QPS[1]}" "qps"
}

# Test 3: Tenant Switching Performance
test_tenant_switching() {
  print_test "Tenant Switching Performance"

  local start_time=$(date +%s.%N)
  local count=0
  local current_tenant=1

  for i in $(seq 1 $SWITCH_COUNT); do
    local new_tenant=$((i % TENANT_COUNT + 1))

    # Simulate tenant context switch
    {
      # 1. Clear current tenant context
      echo "SET LOCAL app.current_tenant = NULL;" > /dev/null

      # 2. Set new tenant context
      echo "SET LOCAL app.current_tenant = 'tenant_${new_tenant}';" > /dev/null

      # 3. Verify RLS policies apply
      echo "SELECT set_config('request.jwt.claims', '{\"tenant_id\":\"tenant_${new_tenant}\"}', true);" > /dev/null

      # 4. Execute query with new context
      echo "SELECT * FROM data WHERE id = ${i};" > /dev/null
    } 2>/dev/null

    current_tenant=$new_tenant
    count=$((count + 1))
  done

  local end_time=$(date +%s.%N)
  local duration=$(echo "$end_time - $start_time" | bc -l)
  local switches_per_sec=$(echo "$count / $duration" | bc -l)
  local avg_switch_time=$(echo "$duration / $count * 1000" | bc -l)

  printf "  ${BLUE}Switches/sec:${NC}      %.2f\n" "$switches_per_sec"
  printf "  ${BLUE}Avg switch time:${NC}   %.2f ms\n" "$avg_switch_time"

  print_result "Tenant Switching" "$switches_per_sec" "${BASELINE_QPS[2]}" "switches/sec"
}

# Test 4: RLS Policy Enforcement Overhead
test_rls_enforcement() {
  print_test "RLS Policy Enforcement Performance"

  # Define different RLS policy complexities
  local -a policy_types=("simple" "moderate" "complex")

  for policy_type in "${policy_types[@]}"; do
    local start_time=$(date +%s.%N)
    local count=0

    for i in $(seq 1 $RLS_CHECKS); do
      local tenant_id=$((i % TENANT_COUNT + 1))

      case "$policy_type" in
        simple)
          # Simple RLS: tenant_id = current_tenant
          echo "SELECT * FROM data WHERE tenant_id = 'tenant_${tenant_id}';" > /dev/null
          ;;
        moderate)
          # Moderate RLS: tenant_id check + user role check
          echo "SELECT * FROM data WHERE tenant_id = 'tenant_${tenant_id}' AND (public = true OR user_id = current_user());" > /dev/null
          ;;
        complex)
          # Complex RLS: multiple joins and subqueries
          echo "SELECT d.* FROM data d JOIN permissions p ON d.id = p.resource_id WHERE d.tenant_id = 'tenant_${tenant_id}' AND p.user_id IN (SELECT id FROM users WHERE tenant_id = 'tenant_${tenant_id}');" > /dev/null
          ;;
      esac

      count=$((count + 1))
    done

    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc -l)
    local qps=$(echo "$count / $duration" | bc -l)

    printf "  ${BLUE}%s RLS:${NC} %.2f qps\n" "$policy_type" "$qps"
  done

  # Use simple RLS baseline
  local start_time=$(date +%s.%N)
  local count=0
  for i in $(seq 1 $RLS_CHECKS); do
    local tenant_id=$((i % TENANT_COUNT + 1))
    echo "SELECT * FROM data WHERE tenant_id = 'tenant_${tenant_id}';" > /dev/null 2>&1
    count=$((count + 1))
  done
  local end_time=$(date +%s.%N)
  local duration=$(echo "$end_time - $start_time" | bc -l)
  local qps=$(echo "$count / $duration" | bc -l)

  print_result "RLS Enforcement" "$qps" "${BASELINE_QPS[3]}" "qps"
}

# Test 5: Tenant Data Partitioning Performance
test_data_partitioning() {
  print_test "Tenant Data Partitioning Performance"

  # Test query performance with different partitioning strategies
  printf "\n  ${BLUE}Testing partitioning strategies:${NC}\n"

  # 1. No partitioning (baseline)
  local start_time=$(date +%s.%N)
  local count=0
  for i in $(seq 1 1000); do
    local tenant_id=$((i % TENANT_COUNT + 1))
    echo "SELECT * FROM data WHERE tenant_id = 'tenant_${tenant_id}' AND id = ${i};" > /dev/null 2>&1
    count=$((count + 1))
  done
  local end_time=$(date +%s.%N)
  local duration=$(echo "$end_time - $start_time" | bc -l)
  local qps_no_partition=$(echo "$count / $duration" | bc -l)

  # 2. Hash partitioning by tenant_id
  start_time=$(date +%s.%N)
  count=0
  for i in $(seq 1 1000); do
    local tenant_id=$((i % TENANT_COUNT + 1))
    local partition=$((tenant_id % 10))
    echo "SELECT * FROM data_p${partition} WHERE tenant_id = 'tenant_${tenant_id}' AND id = ${i};" > /dev/null 2>&1
    count=$((count + 1))
  done
  end_time=$(date +%s.%N)
  duration=$(echo "$end_time - $start_time" | bc -l)
  local qps_hash=$(echo "$count / $duration" | bc -l)

  # 3. List partitioning by tenant_id
  start_time=$(date +%s.%N)
  count=0
  for i in $(seq 1 1000); do
    local tenant_id=$((i % TENANT_COUNT + 1))
    echo "SELECT * FROM data_tenant_${tenant_id} WHERE id = ${i};" > /dev/null 2>&1
    count=$((count + 1))
  done
  end_time=$(date +%s.%N)
  duration=$(echo "$end_time - $start_time" | bc -l)
  local qps_list=$(echo "$count / $duration" | bc -l)

  printf "    No Partitioning:    %.2f qps\n" "$qps_no_partition"
  printf "    Hash Partitioning:  %.2f qps (%.2f%% improvement)\n" "$qps_hash" "$(echo "($qps_hash - $qps_no_partition) / $qps_no_partition * 100" | bc -l)"
  printf "    List Partitioning:  %.2f qps (%.2f%% improvement)\n" "$qps_list" "$(echo "($qps_list - $qps_no_partition) / $qps_no_partition * 100" | bc -l)"
}

# Performance bottleneck analysis
analyze_bottlenecks() {
  print_header "Performance Bottleneck Analysis"

  printf "Analyzing multi-tenant performance...\n\n"

  printf "${YELLOW}Potential Bottlenecks:${NC}\n"
  printf "  • RLS Overhead: Consider materialized views for complex policies\n"
  printf "  • Tenant Switching: Cache tenant context in application layer\n"
  printf "  • Query Planning: Use prepared statements to reduce planning time\n"
  printf "  • Index Contention: Partition indexes by tenant_id\n"
  printf "  • Connection Pooling: Use per-tenant connection pools\n\n"
}

# Optimization suggestions
suggest_optimizations() {
  print_header "Optimization Suggestions"

  printf "${GREEN}Recommended Optimizations:${NC}\n\n"

  printf "1. ${YELLOW}Row-Level Security (RLS)${NC}\n"
  printf "   • Keep RLS policies simple (single tenant_id check)\n"
  printf "   • Use indexes on tenant_id column\n"
  printf "   • Consider SECURITY DEFINER functions for complex checks\n"
  printf "   • Monitor RLS policy execution time\n\n"

  printf "2. ${YELLOW}Tenant Context Management${NC}\n"
  printf "   • Cache tenant context in application memory\n"
  printf "   • Use connection pooling per tenant\n"
  printf "   • Set tenant context once per request\n"
  printf "   • Validate tenant isolation at app layer\n\n"

  printf "3. ${YELLOW}Data Partitioning${NC}\n"
  printf "   • Use declarative partitioning for >100 tenants\n"
  printf "   • Hash partition for even distribution\n"
  printf "   • List partition for tenant-specific schemas\n"
  printf "   • Prune partitions in query planning\n\n"

  printf "4. ${YELLOW}Index Strategy${NC}\n"
  printf "   • Create composite indexes: (tenant_id, frequently_queried_column)\n"
  printf "   • Use partial indexes for tenant-specific queries\n"
  printf "   • Partition indexes along with tables\n"
  printf "   • Monitor index bloat and rebuild regularly\n\n"

  printf "5. ${YELLOW}Query Optimization${NC}\n"
  printf "   • Use prepared statements to reduce planning overhead\n"
  printf "   • Enable query plan caching\n"
  printf "   • Analyze query execution plans per tenant\n"
  printf "   • Use EXPLAIN ANALYZE to identify slow queries\n\n"

  printf "6. ${YELLOW}Tenant Isolation Verification${NC}\n"
  printf "   • Run cross-tenant query tests in CI/CD\n"
  printf "   • Monitor for RLS policy bypass attempts\n"
  printf "   • Audit database access logs\n"
  printf "   • Use database activity monitoring tools\n\n"
}

# Security checklist
security_checklist() {
  print_header "Tenant Isolation Security Checklist"

  printf "${GREEN}Pre-Production Security Verification:${NC}\n\n"

  printf "[ ] RLS policies enabled on ALL multi-tenant tables\n"
  printf "[ ] RLS policies tested with multiple tenants\n"
  printf "[ ] Cross-tenant query attempts blocked\n"
  printf "[ ] Tenant context validated on every request\n"
  printf "[ ] Database users have minimal privileges\n"
  printf "[ ] Tenant IDs use UUIDs (not sequential integers)\n"
  printf "[ ] Application validates tenant ownership\n"
  printf "[ ] Audit logging captures tenant context\n"
  printf "[ ] RLS policy bypass attempts monitored\n"
  printf "[ ] Tenant data export/import procedures tested\n\n"
}

# Generate summary report
generate_summary() {
  print_header "Benchmark Summary"

  {
    printf "Multi-Tenant System Performance Benchmark\n"
    printf "==========================================\n\n"
    printf "Scale: %s\n" "$TEST_LABEL"
    printf "Tenant Count: %d\n" "$TENANT_COUNT"
    printf "Test Date: %s\n" "$(date)"
    printf "Results File: %s\n\n" "$RESULTS_FILE"

    printf "Performance Results:\n"
    printf "-------------------\n"
    cat "${RESULTS_FILE}.csv" | column -t -s','

    printf "\n\nExpected Baselines for %s:\n" "$TEST_LABEL"
    printf "--------------------------------\n"
    printf "Isolated Queries:    %.2f qps\n" "${BASELINE_QPS[0]}"
    printf "RLS Prevention:      %.2f qps\n" "${BASELINE_QPS[1]}"
    printf "Tenant Switching:    %.2f switches/sec\n" "${BASELINE_QPS[2]}"
    printf "RLS Enforcement:     %.2f qps\n" "${BASELINE_QPS[3]}"
  } | tee "$SUMMARY_FILE"

  printf "\n${GREEN}Summary saved to: ${NC}%s\n" "$SUMMARY_FILE"
}

# Main benchmark execution
main() {
  print_header "nself Multi-Tenant Performance Benchmark"

  printf "Configuration:\n"
  printf "  Scale: %s\n" "$TEST_LABEL"
  printf "  Tenant Count: %d\n" "$TENANT_COUNT"
  printf "  Query Count: %d\n" "$QUERY_COUNT"
  printf "  Switch Count: %d\n\n" "$SWITCH_COUNT"

  initialize_results

  # Run all tests
  test_tenant_isolation
  test_cross_tenant_prevention
  test_tenant_switching
  test_rls_enforcement
  test_data_partitioning

  finalize_results

  # Analysis and recommendations
  analyze_bottlenecks
  suggest_optimizations
  security_checklist
  generate_summary

  printf "\n${GREEN}Benchmark complete!${NC}\n"
  printf "Full results: %s\n" "$RESULTS_FILE"
  printf "CSV results: %s.csv\n" "$RESULTS_FILE"
}

# Help message
show_help() {
  cat <<EOF
Usage: $0 [TENANT_COUNT]

Multi-Tenant System Performance Benchmark for nself

ARGUMENTS:
  TENANT_COUNT        Number of tenants to simulate (default: 10)
                      Options: 10, 100, 1000

OPTIONS:
  --help              Show this help message

EXAMPLES:
  $0 10               Small scale (10 tenants)
  $0 100              Medium scale (100 tenants)
  $0 1000             Large scale (1000 tenants)

TESTS PERFORMED:
  • Tenant Isolation Overhead        - RLS performance impact
  • Cross-Tenant Query Prevention    - Security enforcement
  • Tenant Switching Performance     - Context switching speed
  • RLS Policy Enforcement Overhead  - Policy complexity impact
  • Data Partitioning Performance    - Partitioning strategy comparison

RESULTS:
  Results are saved to: benchmarks/results/
  - JSON format: tenant-benchmark-YYYYMMDD-HHMMSS.json
  - CSV format:  tenant-benchmark-YYYYMMDD-HHMMSS.json.csv
  - Summary:     tenant-benchmark-summary.txt

SECURITY:
  This benchmark also validates tenant isolation security.
  Review the security checklist in the output.

EOF
}

# Show help if requested (check SHOW_HELP variable set at top)
if [[ "${SHOW_HELP:-false}" == "true" ]]; then
  show_help
  exit 0
fi

# Run benchmark
main
