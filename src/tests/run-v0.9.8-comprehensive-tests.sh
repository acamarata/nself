#!/usr/bin/env bash
# run-v0.9.8-comprehensive-tests.sh - Run all v0.9.8 comprehensive test suites
# This script runs all 410 new comprehensive tests added for v0.9.8

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
GREEN="\033[32m"
RED="\033[31m"
YELLOW="\033[33m"
BLUE="\033[34m"
RESET="\033[0m"

# Test suite files
TEST_SUITES=(
  "$SCRIPT_DIR/integration/test-billing-comprehensive.sh"
  "$SCRIPT_DIR/integration/test-oauth-providers-comprehensive.sh"
  "$SCRIPT_DIR/integration/test-whitelabel-comprehensive.sh"
  "$SCRIPT_DIR/integration/test-backup-restore-comprehensive.sh"
  "$SCRIPT_DIR/integration/test-rate-limit-comprehensive.sh"
)

# Test suite descriptions
SUITE_NAMES=(
  "Billing System (150 tests)"
  "OAuth Providers (80 tests)"
  "White-Label System (100 tests)"
  "Backup/Restore (50 tests)"
  "Rate Limiting (30 tests)"
)

# Counters
TOTAL_SUITES=${#TEST_SUITES[@]}
SUITES_PASSED=0
SUITES_FAILED=0
START_TIME=$(date +%s)

printf "\n"
printf "${BLUE}╔════════════════════════════════════════════════════════════╗${RESET}\n"
printf "${BLUE}║${RESET}  ${GREEN}nself v0.9.8 Comprehensive Test Suite Runner${RESET}           ${BLUE}║${RESET}\n"
printf "${BLUE}╚════════════════════════════════════════════════════════════╝${RESET}\n"
printf "\n"

printf "Running ${BLUE}${TOTAL_SUITES}${RESET} comprehensive test suites (${BLUE}410${RESET} total tests)...\n\n"

# Run each test suite
for i in "${!TEST_SUITES[@]}"; do
  suite="${TEST_SUITES[$i]}"
  name="${SUITE_NAMES[$i]}"

  if [[ ! -f "$suite" ]]; then
    printf "${RED}✗${RESET} %s - ${RED}FILE NOT FOUND${RESET}\n" "$name"
    SUITES_FAILED=$((SUITES_FAILED + 1))
    continue
  fi

  printf "${YELLOW}▶${RESET} Running: %s\n" "$name"

  # Run test suite and capture exit code
  if bash "$suite" >/dev/null 2>&1; then
    printf "${GREEN}✓${RESET} %s - ${GREEN}PASSED${RESET}\n\n" "$name"
    SUITES_PASSED=$((SUITES_PASSED + 1))
  else
    printf "${RED}✗${RESET} %s - ${RED}FAILED${RESET}\n\n" "$name"
    SUITES_FAILED=$((SUITES_FAILED + 1))
  fi
done

# Calculate execution time
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Print summary
printf "${BLUE}═══════════════════════════════════════════════════════════${RESET}\n"
printf "${BLUE}Test Suite Summary${RESET}\n"
printf "${BLUE}═══════════════════════════════════════════════════════════${RESET}\n"
printf "\n"
printf "Total Suites:    %d\n" "$TOTAL_SUITES"
printf "Passed:          ${GREEN}%d${RESET}\n" "$SUITES_PASSED"
printf "Failed:          ${RED}%d${RESET}\n" "$SUITES_FAILED"
printf "Execution Time:  %d seconds\n" "$DURATION"
printf "\n"

# Calculate success rate
SUCCESS_RATE=$(awk "BEGIN {printf \"%.1f\", ($SUITES_PASSED / $TOTAL_SUITES) * 100}")
printf "Success Rate:    %.1f%%\n" "$SUCCESS_RATE"
printf "\n"

# Exit with appropriate code
if [[ $SUITES_FAILED -eq 0 ]]; then
  printf "${GREEN}✓ All v0.9.8 comprehensive test suites passed!${RESET}\n\n"
  exit 0
else
  printf "${RED}✗ %d test suite(s) failed${RESET}\n\n" "$SUITES_FAILED"
  printf "To see detailed output, run individual test suites:\n"
  for suite in "${TEST_SUITES[@]}"; do
    printf "  bash %s\n" "$suite"
  done
  printf "\n"
  exit 1
fi
