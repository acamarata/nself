#!/usr/bin/env bash
# test-observability.sh - Observability integration tests
# Part of nself v0.7.0 - Sprint 7: OBS-005

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../../lib/observability/metrics.sh"
source "$SCRIPT_DIR/../../lib/observability/logging.sh"
source "$SCRIPT_DIR/../../lib/observability/tracing.sh"
source "$SCRIPT_DIR/../../lib/observability/health.sh"

printf "\n=== Observability Integration Tests ===\n\n"

# Test 1: Initialize observability
printf "Test 1: Initialize metrics... "
metrics_init && printf "✓\n" || printf "✗\n"

printf "Test 2: Initialize logging... "
logging_init && printf "✓\n" || printf "✗\n"

printf "Test 3: Initialize tracing... "
tracing_init && printf "✓\n" || printf "✗\n"

printf "Test 4: Initialize health checks... "
health_init && printf "✓\n" || printf "✗\n"

# Test 5: Record custom metric
printf "Test 5: Record metric... "
metrics_record "test_counter" 42 "counter" '{"env":"test"}' 2>/dev/null && printf "✓\n" || printf "✗\n"

# Test 6: Log entry
printf "Test 6: Create log entry... "
log_info "test_logger" "Test log message" '{"test":true}' 2>/dev/null && printf "✓\n" || printf "✗\n"

# Test 7: Create trace
printf "Test 7: Create trace... "
trace_id=$(trace_start "test_service" "test_operation" '{}' 2>/dev/null)
[[ -n "$trace_id" ]] && printf "✓\n" || printf "✗\n"

# Test 8: Health check
printf "Test 8: Health check... "
result=$(health_check_all 2>/dev/null)
[[ "$result" != "[]" ]] && printf "✓\n" || printf "✗\n"

printf "\n=== Test Summary ===\n"
printf "Total: 8 tests\n"
printf "Sprint 7: Observability tests complete!\n\n"
