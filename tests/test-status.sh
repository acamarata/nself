#!/usr/bin/env bash

# test-status.sh - Test suite for status functionality

set -e

# Test configuration
TEST_DIR="/tmp/nself-test-$$"

# Setup test environment
setup_test_env() {
  mkdir -p "$TEST_DIR"
  cd "$TEST_DIR"
  
  # Create mock .env.local
  cat > .env.local <<EOF
PROJECT_NAME=test
BASE_DOMAIN=test.local
EOF
  
  # Create mock docker-compose.yml
  cat > docker-compose.yml <<EOF
version: '3.8'
services:
  postgres:
    image: postgres:14
  nginx:
    image: nginx:latest
EOF
}

# Cleanup test environment
cleanup_test_env() {
  cd /
  rm -rf "$TEST_DIR"
}

# Test status basic
test_status_basic() {
  echo "Testing status basic functionality..."
  
  # Source status script
  source /Users/admin/Sites/nself/src/cli/status-v2.sh
  
  # Run status (may show no services but shouldn't crash)
  if cmd_status 2>&1 | grep -q "Service"; then
    echo "✓ Status command runs successfully"
  else
    echo "✗ Status command failed"
    return 1
  fi
}

# Test status verbose
test_status_verbose() {
  echo "Testing status verbose mode..."
  
  # Run status with verbose flag
  if cmd_status -v 2>&1 | grep -q "Service"; then
    echo "✓ Status verbose mode works"
  else
    echo "✗ Status verbose mode failed"
    return 1
  fi
}

# Test status JSON output
test_status_json() {
  echo "Testing status JSON output..."
  
  # Set output format
  export OUTPUT_FORMAT="json"
  
  # Run status
  local output=$(cmd_status 2>&1)
  
  # Check for JSON structure
  if echo "$output" | grep -q "timestamp"; then
    echo "✓ Status JSON output works"
  else
    echo "✗ Status JSON output failed"
    return 1
  fi
  
  # Reset format
  unset OUTPUT_FORMAT
}

# Test health monitoring
test_health_monitoring() {
  echo "Testing health monitoring..."
  
  # Mock service status file
  export SERVICE_STATUS_FILE="$TEST_DIR/service-status"
  echo "postgres=healthy" > "$SERVICE_STATUS_FILE"
  echo "nginx=running" >> "$SERVICE_STATUS_FILE"
  
  # Run health summary
  if display_health_summary 2>&1 | grep -q "Health"; then
    echo "✓ Health monitoring works"
  else
    echo "✗ Health monitoring failed"
    return 1
  fi
}

# Test service URL display
test_service_urls() {
  echo "Testing service URL display..."
  
  # Run URL display
  if display_service_urls 2>&1 | grep -q "Service URLs"; then
    echo "✓ Service URL display works"
  else
    echo "✗ Service URL display failed"
    return 1
  fi
}

# Main test runner
main() {
  echo "Running status tests..."
  echo "========================"
  
  # Setup
  setup_test_env
  
  # Run tests
  local failed=0
  
  test_status_basic || ((failed++))
  test_status_verbose || ((failed++))
  test_status_json || ((failed++))
  test_health_monitoring || ((failed++))
  test_service_urls || ((failed++))
  
  # Cleanup
  cleanup_test_env
  
  # Summary
  echo "========================"
  if [[ $failed -eq 0 ]]; then
    echo "All status tests passed!"
    exit 0
  else
    echo "$failed status test(s) failed"
    exit 1
  fi
}

# Run tests
main "$@"