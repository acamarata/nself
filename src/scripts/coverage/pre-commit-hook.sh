#!/usr/bin/env bash
# Pre-commit hook to verify coverage requirements
#
# Installation:
#   cp src/scripts/coverage/pre-commit-hook.sh .git/hooks/pre-commit
#   chmod +x .git/hooks/pre-commit
#
# Or symlink:
#   ln -sf ../../src/scripts/coverage/pre-commit-hook.sh .git/hooks/pre-commit

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Check if we should skip coverage check
if [[ "${SKIP_COVERAGE_CHECK:-false}" == "true" ]]; then
    printf "${YELLOW}⚠${NC} Coverage check skipped (SKIP_COVERAGE_CHECK=true)\n"
    exit 0
fi

# Check if coverage verification script exists
VERIFY_SCRIPT="${PROJECT_ROOT}/src/scripts/coverage/verify-coverage.sh"
if [[ ! -f "$VERIFY_SCRIPT" ]]; then
    printf "${YELLOW}⚠${NC} Coverage verification script not found, skipping\n"
    exit 0
fi

# Print header
printf "\n"
printf "${BLUE}ℹ${NC} Verifying test coverage before commit...\n"
printf "\n"

# Run coverage verification
if ! "$VERIFY_SCRIPT" 2>&1; then
    printf "\n"
    printf "${RED}✗${NC} Coverage verification failed!\n"
    printf "\n"
    printf "Your changes do not maintain the required coverage level.\n"
    printf "\n"
    printf "Options:\n"
    printf "  1. Add tests to cover your changes\n"
    printf "  2. Run: ./src/scripts/coverage/collect-coverage.sh\n"
    printf "  3. View uncovered code: open coverage/reports/html/index.html\n"
    printf "\n"
    printf "To commit anyway (NOT recommended):\n"
    printf "  SKIP_COVERAGE_CHECK=true git commit\n"
    printf "\n"
    exit 1
fi

printf "\n"
printf "${GREEN}✓${NC} Coverage verification passed!\n"
printf "\n"

exit 0
