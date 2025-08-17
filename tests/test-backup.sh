#!/usr/bin/env bash

# test-backup.sh - Test suite for backup functionality

set -e

# Test configuration
TEST_DIR="/tmp/nself-test-$$"
BACKUP_DIR="$TEST_DIR/backups"

# Setup test environment
setup_test_env() {
  mkdir -p "$TEST_DIR"
  cd "$TEST_DIR"
  
  # Create mock .env.local
  cat > .env.local <<EOF
PROJECT_NAME=test
BASE_DOMAIN=test.local
POSTGRES_PASSWORD=testpass
EOF
  
  # Set backup directory
  export BACKUP_DIR="$BACKUP_DIR"
}

# Cleanup test environment
cleanup_test_env() {
  cd /
  rm -rf "$TEST_DIR"
}

# Test backup create
test_backup_create() {
  echo "Testing backup create..."
  
  # Source backup script
  source /Users/admin/Sites/nself/src/cli/backup.sh
  
  # Create backup
  if cmd_backup create full test-backup; then
    echo "✓ Backup created successfully"
  else
    echo "✗ Backup creation failed"
    return 1
  fi
  
  # Check if backup exists
  if [[ -f "$BACKUP_DIR/test-backup" ]] || [[ -f "$BACKUP_DIR/test-backup.tar.gz" ]]; then
    echo "✓ Backup file exists"
  else
    echo "✗ Backup file not found"
    return 1
  fi
}

# Test backup list
test_backup_list() {
  echo "Testing backup list..."
  
  # List backups
  local output=$(cmd_backup list 2>&1)
  
  if echo "$output" | grep -q "test-backup"; then
    echo "✓ Backup appears in list"
  else
    echo "✗ Backup not in list"
    return 1
  fi
}

# Test backup verify
test_backup_verify() {
  echo "Testing backup verify..."
  
  # Find backup name
  local backup_name=$(ls "$BACKUP_DIR" | grep -E "\.tar" | head -1)
  
  if [[ -n "$backup_name" ]]; then
    if cmd_backup verify "$backup_name"; then
      echo "✓ Backup verification passed"
    else
      echo "✗ Backup verification failed"
      return 1
    fi
  else
    echo "⚠ No backup to verify"
  fi
}

# Test backup schedule
test_backup_schedule() {
  echo "Testing backup schedule..."
  
  # Schedule daily backup
  if cmd_backup schedule daily; then
    echo "✓ Backup scheduled successfully"
    
    # Check crontab
    if crontab -l 2>/dev/null | grep -q "nself-backup"; then
      echo "✓ Cron job created"
    else
      echo "✗ Cron job not found"
      return 1
    fi
  else
    echo "✗ Backup scheduling failed"
    return 1
  fi
  
  # Clean up cron
  crontab -l | grep -v "nself-backup" | crontab - 2>/dev/null || true
}

# Test backup prune
test_backup_prune() {
  echo "Testing backup prune..."
  
  # Create old backup (simulate)
  touch -t 202301010000 "$BACKUP_DIR/old-backup.tar.gz" 2>/dev/null || true
  
  # Prune old backups
  if cmd_backup prune 30; then
    echo "✓ Backup prune executed"
  else
    echo "✗ Backup prune failed"
    return 1
  fi
}

# Main test runner
main() {
  echo "Running backup tests..."
  echo "========================"
  
  # Setup
  setup_test_env
  
  # Run tests
  local failed=0
  
  test_backup_create || ((failed++))
  test_backup_list || ((failed++))
  test_backup_verify || ((failed++))
  test_backup_schedule || ((failed++))
  test_backup_prune || ((failed++))
  
  # Cleanup
  cleanup_test_env
  
  # Summary
  echo "========================"
  if [[ $failed -eq 0 ]]; then
    echo "All backup tests passed!"
    exit 0
  else
    echo "$failed backup test(s) failed"
    exit 1
  fi
}

# Run tests
main "$@"