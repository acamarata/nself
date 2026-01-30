#!/usr/bin/env bash
# Test nself commands in a temporary directory to avoid polluting the source tree

set -euo pipefail

# Colors for output
COLOR_GREEN="\033[0;32m"
COLOR_RED="\033[0;31m"
COLOR_YELLOW="\033[0;33m"
COLOR_BLUE="\033[0;34m"
COLOR_RESET="\033[0m"

# Get the nself source directory
NSELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Create a unique temp directory
TEMP_DIR="/tmp/nself-test-$$"
echo -e "${COLOR_BLUE}Creating temp directory: $TEMP_DIR${COLOR_RESET}"
mkdir -p "$TEMP_DIR"
cd "$TEMP_DIR"

# Cleanup function
cleanup() {
  local exit_code=$?
  echo
  echo -e "${COLOR_YELLOW}Cleaning up temp directory...${COLOR_RESET}"
  cd /
  rm -rf "$TEMP_DIR"
  if [ $exit_code -eq 0 ]; then
    echo -e "${COLOR_GREEN}✓ Test completed successfully${COLOR_RESET}"
  else
    echo -e "${COLOR_RED}✗ Test failed with exit code $exit_code${COLOR_RESET}"
  fi
  exit $exit_code
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Function to run nself commands
run_nself() {
  local cmd="$1"
  shift
  printf "${COLOR_BLUE}Running: nself $cmd %s${COLOR_RESET}\n" "$*"
  bash "$NSELF_DIR/src/cli/${cmd}.sh" "$@"
}

# Test sequence
echo -e "${COLOR_GREEN}=== Testing nself commands in temp directory ===${COLOR_RESET}"
echo

# Initialize with demo config
echo -e "${COLOR_BLUE}1. Initializing demo project...${COLOR_RESET}"
cp "$NSELF_DIR/src/templates/demo/.env" .env
echo -e "${COLOR_GREEN}✓ Demo config copied${COLOR_RESET}"
echo

# Build the project
echo -e "${COLOR_BLUE}2. Building project...${COLOR_RESET}"
run_nself build
echo

# Check urls command
echo -e "${COLOR_BLUE}3. Testing urls command...${COLOR_RESET}"
run_nself urls
echo

# Check conflict detection
echo -e "${COLOR_BLUE}4. Testing conflict detection...${COLOR_RESET}"
run_nself urls --check-conflicts
echo

# Test with intentional conflict
echo -e "${COLOR_BLUE}5. Testing with intentional conflict...${COLOR_RESET}"
echo "CS_1_ROUTE=api" >>.env
if run_nself urls --check-conflicts 2>&1 | grep -q "conflict"; then
  echo -e "${COLOR_GREEN}✓ Conflict detection working${COLOR_RESET}"
else
  echo -e "${COLOR_RED}✗ Conflict detection failed${COLOR_RESET}"
fi

echo
echo -e "${COLOR_GREEN}=== All tests completed ===${COLOR_RESET}"
