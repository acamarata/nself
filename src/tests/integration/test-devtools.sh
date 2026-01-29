#!/usr/bin/env bash
# test-devtools.sh - Developer tools integration tests
# Part of nself v0.7.0 - Sprint 10: DEV-005

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../../lib/dev/tools.sh"
source "$SCRIPT_DIR/../../lib/dev/config-manager.sh"

printf "\n=== Developer Tools Integration Tests ===\n\n"

# Test 1: Initialize dev environment
printf "Test 1: Initialize dev environment... "
dev_init >/dev/null 2>&1 && printf "✓\n" || printf "✗\n"

# Test 2: Generate mocks
printf "Test 2: Generate mock data... "
mocks=$(dev_generate_mocks "users" 5 2>/dev/null)
count=$(echo "$mocks" | jq 'length')
[[ $count -eq 5 ]] && printf "✓\n" || printf "✗\n"

# Test 3: Config validation
printf "Test 3: Config validation... "
config_create_template "minimal" "/tmp/test.env" >/dev/null 2>&1
config_validate "/tmp/test.env" >/dev/null 2>&1 && printf "✓\n" || printf "✗\n"

# Test 4: Config template creation
printf "Test 4: Create config template... "
config_create_template "development" "/tmp/dev.env" >/dev/null 2>&1 && printf "✓\n" || printf "✗\n"

# Cleanup
rm -f /tmp/test.env /tmp/dev.env 2>/dev/null

printf "\n=== Test Summary ===\n"
printf "Total: 4 tests\n"
printf "Sprint 10: Developer tools tests complete!\n\n"
