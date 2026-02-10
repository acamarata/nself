#!/usr/bin/env bash
# run-all-tests.sh - Comprehensive test runner for nself
#
# This script runs all test suites in the project

set -euo pipefail

# Test directories
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UNIT_DIR="$TEST_DIR/unit"
INTEGRATION_DIR="$TEST_DIR/integration"
HELPERS_DIR="$TEST_DIR/helpers"

# Source test framework
source "$TEST_DIR/test_framework.sh"

# Command line options
VERBOSE=false
QUICK=false
FILTER=""
SHOW_HELP=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -v | --verbose)
      VERBOSE=true
      shift
      ;;
    -q | --quick)
      QUICK=true
      shift
      ;;
    -f | --filter)
      FILTER="$2"
      shift 2
      ;;
    -h | --help)
      SHOW_HELP=true
      shift
      ;;
    *)
      echo "Unknown option: $1"
      SHOW_HELP=true
      shift
      ;;
  esac
done

# Show help
if [[ "$SHOW_HELP" == true ]]; then
  cat <<EOF
Usage: $0 [OPTIONS]

Options:
  -v, --verbose       Show detailed test output
  -q, --quick         Skip integration tests
  -f, --filter PATTERN Run only tests matching pattern
  -h, --help          Show this help message

Examples:
  $0                  Run all tests
  $0 --quick          Run only unit tests
  $0 --filter init    Run only init-related tests
EOF
  exit 0
fi

# Print header
printf "${COLOR_BLUE}╔════════════════════════════════════════════════╗${COLOR_RESET}\n"
printf "${COLOR_BLUE}║       nself Comprehensive Test Suite           ║${COLOR_RESET}\n"
printf "${COLOR_BLUE}╚════════════════════════════════════════════════╝${COLOR_RESET}\n"
echo ""

# Function to run a test file
run_test_file() {
  local test_file="$1"
  local test_name="$(basename "$test_file" .sh)"

  # Apply filter if set
  if [[ -n "$FILTER" ]] && [[ ! "$test_name" =~ $FILTER ]]; then
    return 0
  fi

  TESTS_RUN=$((TESTS_RUN + 1))

  printf "${COLOR_MAGENTA}▶ Running %s${COLOR_RESET}\n" "$test_name"

  local exit_code=0

  if [[ "$VERBOSE" == true ]]; then
    bash "$test_file" || exit_code=$?
  else
    # Capture output and only show on failure
    local output
    output=$(bash "$test_file" 2>&1) || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
      printf "  ${COLOR_GREEN}✓ Passed${COLOR_RESET}\n"
    else
      printf "  ${COLOR_RED}✗ Failed (exit code %d)${COLOR_RESET}\n" "$exit_code"
      printf "%s\n" "$output" | sed 's/^/    /'
    fi
  fi

  if [[ $exit_code -eq 0 ]]; then
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi

  return $exit_code
}

# Run unit tests
if [[ -d "$UNIT_DIR" ]]; then
  printf "${COLOR_BLUE}=== Unit Tests ===${COLOR_RESET}\n"
  echo ""

  for test_file in "$UNIT_DIR"/test-*.sh; do
    if [[ -f "$test_file" ]]; then
      run_test_file "$test_file" || true
    fi
  done
  echo ""
fi

# Run integration tests (unless --quick)
if [[ "$QUICK" != true ]] && [[ -d "$INTEGRATION_DIR" ]]; then
  printf "${COLOR_BLUE}=== Integration Tests ===${COLOR_RESET}\n"
  echo ""

  for test_file in "$INTEGRATION_DIR"/test-*.sh; do
    if [[ -f "$test_file" ]]; then
      run_test_file "$test_file" || true
    fi
  done
  echo ""
fi

# Run legacy tests
printf "${COLOR_BLUE}=== Legacy Tests ===${COLOR_RESET}\n"
echo ""

for test_file in "$TEST_DIR"/test-*.sh; do
  if [[ -f "$test_file" ]]; then
    run_test_file "$test_file" || true
  fi
done
echo ""

# Run specific command tests
printf "${COLOR_BLUE}=== Command-Specific Tests ===${COLOR_RESET}\n"
echo ""

# Test init command specifically
if [[ -z "$FILTER" ]] || [[ "init" =~ $FILTER ]]; then
  if [[ -f "$TEST_DIR/run-init-tests.sh" ]]; then
    TESTS_RUN=$((TESTS_RUN + 1))
    printf "${COLOR_MAGENTA}▶ Running init tests${COLOR_RESET}\n"
    if bash "$TEST_DIR/run-init-tests.sh" --quick >/dev/null 2>&1; then
      TESTS_PASSED=$((TESTS_PASSED + 1))
      printf "  ${COLOR_GREEN}✓ Init tests passed${COLOR_RESET}\n"
    else
      TESTS_FAILED=$((TESTS_FAILED + 1))
      printf "  ${COLOR_RED}✗ Init tests failed${COLOR_RESET}\n"
    fi
  fi
fi
echo ""

# Print summary (ignore return code - we handle exit explicitly below)
print_test_summary || true

# Exit with appropriate code based on failure count
if [[ $TESTS_FAILED -gt 0 ]]; then
  printf "${COLOR_RED}Some tests failed. Please review the output above.${COLOR_RESET}\n"
  exit 1
else
  printf "${COLOR_GREEN}All tests passed successfully!${COLOR_RESET}\n"
  exit 0
fi
