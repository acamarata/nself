# Error Scenario Tests

This directory contains realistic error scenario tests that verify nself handles common user errors gracefully with clear, actionable error messages.

## Philosophy

**Test what users actually encounter, not impossible edge cases.**

These tests focus on:
- ✅ Real errors users will encounter
- ✅ Clear, actionable error messages
- ✅ Cross-platform compatibility
- ✅ Recovery and fix instructions
- ❌ Not defensive programming for impossible scenarios
- ❌ Not library bugs or hardware failures

## Test Files

### test-installation-errors.sh
Tests errors during installation and initial setup:
- Docker not installed
- Docker daemon not running
- Insufficient permissions
- Disk space issues
- Incompatible Docker versions
- Port conflicts
- Missing dependencies (curl, git)

**Example tested scenario:**
```bash
# User tries to start nself without Docker running
$ nself start
Error: Docker is not running

Fix:
  macOS: open -a Docker
  Linux: sudo systemctl start docker
```

### test-configuration-errors.sh
Tests configuration file and validation errors:
- Missing .env file
- Invalid environment variable format
- Port numbers out of range
- Invalid domain names
- Conflicting settings
- Missing required variables
- Encrypted .env corruption

**Example tested scenario:**
```bash
# User has invalid port in .env
POSTGRES_PORT=99999

Error: Invalid port number: 99999
Port numbers must be between 1 and 65535

Fix:
  POSTGRES_PORT=5432
```

### test-service-failures.sh
Tests service startup and runtime failures:
- Port already in use
- Container fails to start
- Dependency not ready
- Health check timeout
- Missing Docker image
- Build failures
- Out of memory
- Disk full
- Network errors
- Volume permission errors

**Example tested scenario:**
```bash
# User's postgres port is already in use
Error: Port 5432 is already in use by another process

Fix:
  1. Find the process:
     lsof -i :5432

  2. Kill the process:
     kill -9 <PID>

  Or change the port in .env:
     POSTGRES_PORT=5433
```

## Running Tests

Run all error tests:
```bash
cd /Users/admin/Sites/nself
./src/tests/errors/run-error-tests.sh
```

Run individual test files:
```bash
./src/tests/errors/test-installation-errors.sh
./src/tests/errors/test-configuration-errors.sh
./src/tests/errors/test-service-failures.sh
```

## Test Coverage Goals

These tests ensure:
1. ✅ Every error has a clear title
2. ✅ Every error explains what went wrong
3. ✅ Every error provides specific fix instructions
4. ✅ Commands are provided (copy-paste ready)
5. ✅ Platform-specific instructions (macOS vs Linux)
6. ✅ No cryptic error codes or stack traces
7. ✅ Errors don't crash the program
8. ✅ Proper exit codes returned

## Error Message Quality Checklist

Every error message must have:
- [ ] **Clear title** - What went wrong
- [ ] **Problem description** - Why it happened
- [ ] **Fix instructions** - How to resolve (numbered steps)
- [ ] **Commands** - Actual commands user can run
- [ ] **Verification** - How to verify the fix worked

Bad error message:
```
Error: Operation failed (code: 0x4E2)
```

Good error message:
```
Docker is not running

Problem:
  The 'docker' command could not connect to the Docker daemon.

Fix:
  Start Docker Desktop:
    macOS: open -a Docker
    Linux: sudo systemctl start docker

  Verify Docker is running:
    docker ps
```

## Adding New Error Tests

To add a new error scenario test:

1. Identify a real error users encounter
2. Write the expected error message first
3. Write a test that verifies the message quality
4. Ensure cross-platform compatibility

Template:
```bash
test_your_error_scenario() {
  local test_name="Description of error"

  # Expected error message
  local output
  output=$(your_error_function "params")

  # Verify message quality
  assert_contains "$output" "Error title" "$test_name: Title"
  assert_contains "$output" "Problem:" "$test_name: Problem section"
  assert_contains "$output" "Fix:" "$test_name: Fix section"
  assert_contains "$output" "command-to-run" "$test_name: Actionable command"
}
```

## Integration with CI

These tests run in CI on:
- Ubuntu (latest)
- macOS (latest)

Tests verify:
- Error messages are cross-platform
- No `echo -e` usage (use `printf`)
- No Bash 4+ features
- All commands are portable

## Related Documentation

- `/docs/ERROR-HANDLING.md` - Error handling guidelines
- `/src/lib/utils/error-messages.sh` - Error message library
- `/src/tests/unit/test-error-messages.sh` - Unit tests for error library
