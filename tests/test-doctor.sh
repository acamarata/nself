#!/usr/bin/env bash

# test-doctor.sh - Test suite for doctor functionality

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
POSTGRES_PASSWORD=testpass
HASURA_GRAPHQL_ADMIN_SECRET=testsecret
EOF
}

# Cleanup test environment
cleanup_test_env() {
  cd /
  rm -rf "$TEST_DIR"
}

# Test doctor basic run
test_doctor_basic() {
  echo "Testing doctor basic functionality..."
  
  # Source doctor script
  source /Users/admin/Sites/nself/src/cli/doctor-v2.sh
  
  # Run doctor (may have warnings but shouldn't crash)
  if cmd_doctor 2>&1 | grep -q "System Requirements"; then
    echo "✓ Doctor command runs successfully"
  else
    echo "✗ Doctor command failed"
    return 1
  fi
}

# Test SSL checks
test_ssl_checks() {
  echo "Testing SSL certificate checks..."
  
  # Create mock SSL directory
  mkdir -p nginx/ssl
  
  # Create mock certificate
  openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout nginx/ssl/test.key \
    -out nginx/ssl/test.crt \
    -subj "/C=US/ST=Test/L=Test/O=Test/CN=test.local" \
    2>/dev/null
  
  # Run SSL check
  if check_ssl_certificates 2>&1 | grep -q "SSL"; then
    echo "✓ SSL checks completed"
  else
    echo "✗ SSL checks failed"
    return 1
  fi
}

# Test DNS checks
test_dns_checks() {
  echo "Testing DNS configuration checks..."
  
  # Run DNS check
  if check_dns_configuration 2>&1 | grep -q "DNS"; then
    echo "✓ DNS checks completed"
  else
    echo "✗ DNS checks failed"
    return 1
  fi
}

# Test kernel parameter checks
test_kernel_checks() {
  echo "Testing kernel parameter checks..."
  
  # Run kernel check
  if check_kernel_parameters 2>&1 | grep -q "Kernel"; then
    echo "✓ Kernel checks completed"
  else
    echo "✗ Kernel checks failed"
    return 1
  fi
}

# Test network diagnostics
test_network_diagnostics() {
  echo "Testing network diagnostics..."
  
  # Run network check
  if check_network_diagnostics 2>&1 | grep -q "Network"; then
    echo "✓ Network diagnostics completed"
  else
    echo "✗ Network diagnostics failed"
    return 1
  fi
}

# Test resource usage checks
test_resource_checks() {
  echo "Testing resource usage checks..."
  
  # Run resource check
  if check_resource_usage 2>&1 | grep -q "Resource"; then
    echo "✓ Resource checks completed"
  else
    echo "✗ Resource checks failed"
    return 1
  fi
}

# Main test runner
main() {
  echo "Running doctor tests..."
  echo "========================"
  
  # Setup
  setup_test_env
  
  # Run tests
  local failed=0
  
  test_doctor_basic || ((failed++))
  test_ssl_checks || ((failed++))
  test_dns_checks || ((failed++))
  test_kernel_checks || ((failed++))
  test_network_diagnostics || ((failed++))
  test_resource_checks || ((failed++))
  
  # Cleanup
  cleanup_test_env
  
  # Summary
  echo "========================"
  if [[ $failed -eq 0 ]]; then
    echo "All doctor tests passed!"
    exit 0
  else
    echo "$failed doctor test(s) failed"
    exit 1
  fi
}

# Run tests
main "$@"