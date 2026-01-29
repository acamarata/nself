#!/usr/bin/env bash
# test-compliance.sh - Compliance integration tests
# Part of nself v0.7.0 - Sprint 9: CSA-006

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../../lib/compliance/framework.sh"
source "$SCRIPT_DIR/../../lib/compliance/reports.sh"

printf "\n=== Compliance Integration Tests ===\n\n"

# Test 1: Initialize compliance
printf "Test 1: Initialize compliance... "
compliance_init && printf "✓\n" || printf "✗\n"

# Test 2: Set retention policy
printf "Test 2: Set retention policy... "
compliance_set_retention "logs" 90 "GDPR requirement" "gdpr" 2>/dev/null && printf "✓\n" || printf "✗\n"

# Test 3: Get compliance status
printf "Test 3: Get compliance status... "
status=$(compliance_get_status 2>/dev/null)
[[ -n "$status" ]] && printf "✓\n" || printf "✗\n"

# Test 4: Security scan
printf "Test 4: Run security scan... "
scan=$(security_scan 2>/dev/null)
[[ "$scan" != "null" ]] && printf "✓\n" || printf "✗\n"

printf "\n=== Test Summary ===\n"
printf "Total: 4 tests\n"
printf "Sprint 9: Compliance tests complete!\n\n"
