#!/usr/bin/env bash
# run-all-realtime-tests.sh - Run all real-time system tests
# Part of nself v0.8.0 - Sprint 16: Real-Time Collaboration
#
# Usage:
#   ./run-all-realtime-tests.sh              # Run all tests
#   ./run-all-realtime-tests.sh --verbose    # Verbose output
#   ./run-all-realtime-tests.sh --quick      # Skip WebSocket tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test configuration
VERBOSE=false
QUICK=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --verbose | -v)
      VERBOSE=true
      shift
      ;;
    --quick | -q)
      QUICK=true
      shift
      ;;
    --help | -h)
      printf "Usage: %s [options]\n" "$(basename "$0")"
      printf "\nOptions:\n"
      printf "  --verbose, -v    Show verbose output\n"
      printf "  --quick, -q      Skip WebSocket tests (database only)\n"
      printf "  --help, -h       Show this help\n"
      exit 0
      ;;
    *)
      printf "Unknown option: %s\n" "$1" >&2
      exit 1
      ;;
  esac
done

# Colors
RED='\033[31m'
GREEN='\033[32m'
BLUE='\033[34m'
YELLOW='\033[33m'
NC='\033[0m'

print_header() {
  printf "\n${BLUE}=== %s ===${NC}\n\n" "$1"
}

print_success() {
  printf "${GREEN}✓${NC} %s\n" "$1"
}

print_failure() {
  printf "${RED}✗${NC} %s\n" "$1"
}

print_info() {
  printf "${YELLOW}ℹ${NC} %s\n" "$1"
}

# Track overall results
total_suites=0
passed_suites=0
failed_suites=0

run_test_suite() {
  local test_file="$1"
  local test_name="$2"

  total_suites=$((total_suites + 1))

  print_header "$test_name"

  if [[ ! -f "$test_file" ]]; then
    print_failure "Test file not found: $test_file"
    failed_suites=$((failed_suites + 1))
    return 1
  fi

  if [[ ! -x "$test_file" ]]; then
    print_info "Making test executable: $test_file"
    chmod +x "$test_file"
  fi

  # Run test
  if [[ "$VERBOSE" == "true" ]]; then
    "$test_file"
  else
    "$test_file" 2>&1 | grep -E "(✓|✗|Test Summary|passed|failed|complete)" || true
  fi

  local exit_code=${PIPESTATUS[0]}

  if [[ $exit_code -eq 0 ]]; then
    print_success "$test_name completed successfully"
    passed_suites=$((passed_suites + 1))
    return 0
  else
    print_failure "$test_name failed with exit code $exit_code"
    failed_suites=$((failed_suites + 1))
    return 1
  fi
}

# ============================================================================
# Main Test Execution
# ============================================================================

main() {
  print_header "Real-Time System Test Suite"

  # Check prerequisites
  print_info "Checking prerequisites..."

  # Check PostgreSQL
  if ! docker ps --filter 'name=postgres' --format '{{.Names}}' | grep -q postgres; then
    print_failure "PostgreSQL not running"
    printf "\nStart PostgreSQL with: nself start\n"
    exit 1
  fi
  print_success "PostgreSQL is running"

  # Check migration
  local migration_check
  migration_check=$(docker exec "$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)" \
    psql -U postgres -d nself -t -A -c "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = 'realtime');" 2>/dev/null || printf "f")

  if [[ "$migration_check" != "t" ]]; then
    print_failure "Realtime schema not found - migration 012 may not be applied"
    printf "\nApply migration with:\n"
    printf "  docker exec <container> psql -U postgres -d nself -f /path/to/012_create_realtime_system.sql\n"
    exit 1
  fi
  print_success "Realtime schema exists"

  printf "\n"

  # ============================================================================
  # Test Suite 1: Database Layer Tests
  # ============================================================================

  run_test_suite "$SCRIPT_DIR/test-realtime.sh" "Database Layer Tests (45 tests)"

  # ============================================================================
  # Test Suite 2: WebSocket Tests (Optional)
  # ============================================================================

  if [[ "$QUICK" == "false" ]]; then
    print_header "WebSocket Tests"
    print_info "Checking for WebSocket test helpers..."

    if [[ -f "$SCRIPT_DIR/websocket-test-helpers.sh" ]]; then
      print_success "WebSocket test helpers found"

      # Check if wscat is available
      if command -v wscat >/dev/null 2>&1; then
        print_success "wscat is available"
        print_info "WebSocket tests can be enabled when WebSocket server is implemented"
      else
        print_info "wscat not found - WebSocket tests skipped"
        print_info "Install with: npm install -g wscat"
      fi
    else
      print_info "WebSocket test helpers not found - skipping"
    fi
  else
    print_info "Quick mode - skipping WebSocket tests"
  fi

  # ============================================================================
  # Final Summary
  # ============================================================================

  print_header "Overall Test Summary"

  printf "Test Suites:\n"
  printf "  Total: %d\n" "$total_suites"
  printf "  Passed: %d\n" "$passed_suites"
  printf "  Failed: %d\n" "$failed_suites"

  if [[ $failed_suites -eq 0 ]]; then
    printf "\n"
    print_success "All test suites passed!"
    printf "\n${GREEN}Sprint 16: Real-Time Collaboration - Tests Complete!${NC}\n\n"
    exit 0
  else
    printf "\n"
    print_failure "Some test suites failed"
    printf "\nRun with --verbose for detailed output\n"
    exit 1
  fi
}

# Run main function
main
