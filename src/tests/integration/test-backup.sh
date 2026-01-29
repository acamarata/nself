#!/usr/bin/env bash
# test-backup.sh - Backup & DR integration tests
# Part of nself v0.7.0 - Sprint 8: BDR-005

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../../lib/backup/automated.sh"
source "$SCRIPT_DIR/../../lib/backup/recovery.sh"

printf "\n=== Backup & DR Integration Tests ===\n\n"

# Test 1: Initialize backup system
printf "Test 1: Initialize backup... "
backup_init && printf "✓\n" || printf "✗\n"

# Test 2: Create backup
printf "Test 2: Create full backup... "
result=$(backup_create "full" "true" "false" "false" 2>/dev/null)
backup_id=$(echo "$result" | jq -r '.backup_id' 2>/dev/null)
[[ -n "$backup_id" && "$backup_id" != "null" ]] && printf "✓\n" || printf "✗\n"

# Test 3: Verify backup
printf "Test 3: Verify backup... "
backup_verify "$backup_id" >/dev/null 2>&1 && printf "✓\n" || printf "✗\n"

# Test 4: List backups
printf "Test 4: List backups... "
list=$(backup_list 10 2>/dev/null)
[[ "$list" != "[]" ]] && printf "✓\n" || printf "✗\n"

# Test 5: Schedule backup
printf "Test 5: Create backup schedule... "
backup_schedule_create "test_daily" "full" "daily" 7 2>/dev/null && printf "✓\n" || printf "✗\n"

printf "\n=== Test Summary ===\n"
printf "Total: 5 tests\n"
printf "Sprint 8: Backup & DR tests complete!\n\n"
