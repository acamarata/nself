# Edge Case Tests

This directory contains edge case and boundary value tests that verify nself handles unusual but valid inputs correctly.

## Philosophy

**Test realistic edge cases that users might encounter, not impossible scenarios.**

These tests focus on:
- ✅ Boundary values (min/max valid inputs)
- ✅ State transitions and idempotency
- ✅ Concurrent operations
- ✅ Data integrity
- ❌ Not cosmic ray bit flips
- ❌ Not impossible defensive programming

## Test Files

### test-boundary-values.sh
Tests minimum, maximum, and boundary values:
- Port numbers (0, 1, 1023, 1024, 65535, 65536)
- String lengths (empty, single char, very long)
- Domain names (min length, max length 253, label max 63)
- Special characters (Unicode, null bytes, control chars)
- Numeric boundaries (integer overflow, negative zero)
- Boolean variations (true/false vs TRUE/yes/1)
- Email lengths (minimum a@b.c, maximum 320 chars)
- URL lengths (minimum, IE limit 2083 chars)

**Example tested scenario:**
```bash
# Port 0 is invalid
POSTGRES_PORT=0  → Error: Port 0 is invalid

# Port 1 is valid but privileged
POSTGRES_PORT=1  → Warning: Port 1 requires root access

# Port 65535 is valid (max)
POSTGRES_PORT=65535  → OK

# Port 65536 is invalid (over max)
POSTGRES_PORT=65536  → Error: Port must be 1-65535
```

### test-state-transitions.sh
Tests unusual state transitions and idempotency:
- Start service already running (idempotent)
- Stop service already stopped (idempotent)
- Restart service that's not running
- Multiple start commands in sequence
- Build without init (should error)
- Start without build (should error)
- Deploy without build (should error)
- Rapid start/stop cycles
- Configuration changes while running
- Partial service failures

**Example tested scenario:**
```bash
# Service already running
$ nself start
$ nself start  # Second time should be idempotent
✓ Services already running

# Service stopped
$ nself restart  # Should start, not error
✓ Service started
```

### test-concurrency.sh
Tests concurrent operations and race conditions:
- Multiple `nself start` simultaneously (lock prevents)
- Simultaneous builds (queue or error)
- Concurrent database migrations (lock prevents)
- Parallel deployments (safety checks)
- File write race conditions (atomic operations)

### test-data-integrity.sh
Tests data corruption and recovery:
- docker-compose.yml corrupted (validation, regeneration)
- .env file corrupted (validation, example)
- Migration file corrupted (checksum verification)
- Backup file corrupted (integrity check fails gracefully)
- Config file with invalid syntax (parse error with line number)

### test-cleanup-recovery.sh
Tests recovery from interrupted operations:
- Interrupted build (resume or restart)
- Killed during migration (rollback or continue)
- Partial service start (cleanup and retry)
- Failed deployment cleanup (automatic rollback)
- Orphaned containers (detection and cleanup)
- Dangling volumes (safe removal option)

## Running Tests

Run all edge case tests:
```bash
cd /Users/admin/Sites/nself
./src/tests/edge-cases/run-edge-case-tests.sh
```

Run individual test files:
```bash
./src/tests/edge-cases/test-boundary-values.sh
./src/tests/edge-cases/test-state-transitions.sh
```

## Test Patterns

### Pattern 1: Boundary Value Testing

Test the boundaries of valid inputs:
```bash
# Test minimum valid value
test_minimum_valid() {
  local min_value=1
  assert_valid "$min_value"
}

# Test maximum valid value
test_maximum_valid() {
  local max_value=65535
  assert_valid "$max_value"
}

# Test below minimum (invalid)
test_below_minimum() {
  local invalid_value=0
  assert_invalid "$invalid_value"
}

# Test above maximum (invalid)
test_above_maximum() {
  local invalid_value=65536
  assert_invalid "$invalid_value"
}
```

### Pattern 2: Idempotency Testing

Ensure operations can be repeated safely:
```bash
test_operation_idempotent() {
  # First execution
  execute_operation
  local state1="$STATE"

  # Second execution (should not change state)
  execute_operation
  local state2="$STATE"

  # Third execution (still no change)
  execute_operation
  local state3="$STATE"

  assert_equals "$state1" "$state2"
  assert_equals "$state2" "$state3"
}
```

### Pattern 3: State Machine Testing

Test all valid and invalid state transitions:
```bash
test_valid_state_transition() {
  STATE="stopped"

  start_service
  assert_equals "$STATE" "running"

  stop_service
  assert_equals "$STATE" "stopped"
}

test_invalid_state_transition() {
  STATE="not-initialized"

  if start_service 2>/dev/null; then
    fail "Should not allow start without init"
  else
    pass "Correctly blocked invalid transition"
  fi
}
```

## Coverage Exclusions

We mark defensive programming as excluded from coverage:

```bash
# This should never happen in practice
# coverage: ignore
if [[ "$IMPOSSIBLE_CONDITION" == "true" ]]; then
  log_error "Defensive programming - should never reach here"
  exit 1
fi
```

Focus coverage on:
- User input validation
- State machine transitions
- Error handling paths
- Recovery mechanisms

Don't test:
- Third-party library bugs
- Hardware failures
- Operating system bugs
- Impossible conditions

## Adding New Edge Case Tests

To add a new edge case test:

1. Identify a realistic edge case users might encounter
2. Determine expected behavior
3. Write test that verifies behavior
4. Ensure test is fast and reliable

Template:
```bash
test_your_edge_case() {
  local test_name="Description of edge case"

  # Setup edge case condition
  setup_edge_case_condition

  # Execute operation
  execute_operation

  # Verify expected behavior
  assert_equals "$actual" "$expected" "$test_name"
}
```

## Integration with CI

These tests run in CI on:
- Ubuntu (latest)
- macOS (latest)

Tests must:
- Complete in under 30 seconds total
- Pass 100% of the time (no flaky tests)
- Not depend on external services
- Clean up after themselves

## Related Documentation

- `/.wiki/testing/README.md` - Testing guidelines
- `/.wiki/development/ERROR-HANDLING.md` - Error handling and state transitions
- `/src/tests/unit/` - Unit tests
- `/src/tests/integration/` - Integration tests
