#!/usr/bin/env bash
# run-tests.sh - Canonical test entrypoint for the nself project
#
# This is the official entrypoint for running nself tests.
# It delegates to run-all-tests.sh with all arguments forwarded.
#
# Usage:
#   bash src/tests/run-tests.sh [OPTIONS]
#
# Options:
#   -v, --verbose        Show detailed test output
#   -q, --quick          Skip integration tests (unit tests only)
#   -f, --filter PATTERN Run only tests matching PATTERN
#   -h, --help           Show this help message
#
# Examples:
#   bash src/tests/run-tests.sh              # Run all tests
#   bash src/tests/run-tests.sh --quick      # Run only unit tests
#   bash src/tests/run-tests.sh -f init      # Run init-related tests
#   bash src/tests/run-tests.sh --verbose    # Run with detailed output
#
# For CI/CD usage, this is the script that workflows should invoke.
# For more details, see: .wiki/testing/QUICK-TEST-REFERENCE.md

set -euo pipefail

# Resolve the directory this script lives in (portable, no readlink -f)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# The actual test runner that does the work
TEST_RUNNER="$SCRIPT_DIR/run-all-tests.sh"

# Handle --help before delegating so the wrapper itself documents the interface
for arg in "$@"; do
  if [ "$arg" = "-h" ] || [ "$arg" = "--help" ]; then
    printf "nself Test Suite - Canonical Entrypoint\n"
    printf "========================================\n\n"
    printf "Usage: %s [OPTIONS]\n\n" "$0"
    printf "Options:\n"
    printf "  -v, --verbose        Show detailed test output\n"
    printf "  -q, --quick          Skip integration tests (unit tests only)\n"
    printf "  -f, --filter PATTERN Run only tests matching PATTERN\n"
    printf "  -h, --help           Show this help message\n"
    printf "\nTest Modes:\n"
    printf "  (default)            Run all tests: unit, integration, legacy, command-specific\n"
    printf "  --quick              Run unit tests only (faster feedback loop)\n"
    printf "  --filter <pattern>   Run tests whose names match the given pattern\n"
    printf "\nExamples:\n"
    printf "  %s                   Run the full test suite\n" "$0"
    printf "  %s --quick           Run unit tests only\n" "$0"
    printf "  %s -f init           Run only init-related tests\n" "$0"
    printf "  %s -v                Run all tests with verbose output\n" "$0"
    printf "  %s -q -f billing     Run unit tests matching 'billing'\n" "$0"
    printf "\nThis script delegates to run-all-tests.sh.\n"
    printf "For full documentation, see: .wiki/testing/QUICK-TEST-REFERENCE.md\n"
    exit 0
  fi
done

# Verify the actual test runner exists
if [ ! -f "$TEST_RUNNER" ]; then
  printf "Error: Test runner not found at %s\n" "$TEST_RUNNER" >&2
  printf "Ensure you are running from the nself project root.\n" >&2
  exit 1
fi

# Delegate to the actual test runner with all arguments forwarded
exec bash "$TEST_RUNNER" "$@"
