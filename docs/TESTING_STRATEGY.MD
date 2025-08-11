# NSELF Testing Strategy

## Overview

This document defines the comprehensive testing strategy for NSELF, enabling modular development where tested functions can be relied upon while developing new features.

## Testing Philosophy

### Core Principles
1. **Test Early, Test Often**: Write tests before or alongside code
2. **Modular Testing**: Each function tested in isolation
3. **Build on Trust**: Once a function passes tests, rely on it
4. **Fail Fast**: Catch issues at the smallest unit level
5. **Comprehensive Coverage**: Test happy paths, edge cases, and error conditions

### Benefits of Modular Testing
- **Confidence**: Know that individual parts work correctly
- **Faster Debugging**: Issues isolated to specific functions
- **Easier Refactoring**: Tests ensure functionality preserved
- **Parallel Development**: Team members can work on different modules
- **Documentation**: Tests serve as usage examples

## Test Structure

```
bin/tests/
├── unit/                    # Individual function tests
│   ├── utils/               # Utility function tests
│   │   ├── test_display.sh
│   │   ├── test_error.sh
│   │   ├── test_validation.sh
│   │   └── test_docker.sh
│   ├── commands/            # Command function tests
│   │   ├── test_init.sh
│   │   ├── test_up.sh
│   │   └── test_build.sh
│   └── services/            # Service function tests
│       ├── test_postgres.sh
│       ├── test_hasura.sh
│       └── test_minio.sh
├── integration/             # Multi-component tests
│   ├── test_workflows.sh
│   ├── test_error_recovery.sh
│   └── test_service_orchestration.sh
├── sanity/                  # Quick validation tests
│   ├── test_prerequisites.sh
│   ├── test_environment.sh
│   └── test_dependencies.sh
├── fixtures/                # Test data and mocks
│   ├── mock_data/
│   ├── test_configs/
│   └── stubs/
├── run_tests.sh             # Main test runner
└── test_framework.sh        # Testing utilities
```

## Test Types

### 1. Unit Tests
**Purpose**: Test individual functions in isolation

```bash
#!/bin/bash
# unit/utils/test_validation.sh

source "$(dirname "$0")/../../test_framework.sh"
source "$(dirname "$0")/../../../shared/utils/validation.sh"

test_is_valid_domain() {
    # Test valid domains
    assert_true "is_valid_domain 'example.com'" "Valid domain"
    assert_true "is_valid_domain 'sub.example.com'" "Valid subdomain"
    
    # Test invalid domains
    assert_false "is_valid_domain 'example'" "Missing TLD"
    assert_false "is_valid_domain '.com'" "Missing domain"
    assert_false "is_valid_domain 'ex ample.com'" "Contains space"
}

test_is_valid_port() {
    # Test valid ports
    assert_true "is_valid_port 80" "Standard HTTP port"
    assert_true "is_valid_port 8080" "Common alt port"
    assert_true "is_valid_port 65535" "Maximum port"
    
    # Test invalid ports
    assert_false "is_valid_port 0" "Port zero"
    assert_false "is_valid_port 65536" "Port too high"
    assert_false "is_valid_port 'abc'" "Non-numeric port"
}

run_tests
```

### 2. Integration Tests
**Purpose**: Test component interactions

```bash
#!/bin/bash
# integration/test_service_startup.sh

source "$(dirname "$0")/../test_framework.sh"

test_database_and_hasura_integration() {
    # Start PostgreSQL
    start_postgres_service || fail "PostgreSQL failed to start"
    
    # Wait for PostgreSQL to be ready
    wait_for_postgres || fail "PostgreSQL not ready"
    
    # Start Hasura
    start_hasura_service || fail "Hasura failed to start"
    
    # Test connection
    test_hasura_connects_to_postgres || fail "Hasura cannot connect to PostgreSQL"
    
    # Cleanup
    stop_hasura_service
    stop_postgres_service
    
    pass "Database and Hasura integration successful"
}

run_tests
```

### 3. Sanity Tests
**Purpose**: Quick validation before running full test suite

```bash
#!/bin/bash
# sanity/test_prerequisites.sh

source "$(dirname "$0")/../test_framework.sh"

test_docker_available() {
    if ! command -v docker >/dev/null 2>&1; then
        skip "Docker not installed"
    fi
    
    if ! docker info >/dev/null 2>&1; then
        fail "Docker daemon not running"
    fi
    
    pass "Docker is available"
}

test_required_ports_free() {
    local required_ports=(5432 8080 9000 6379)
    
    for port in "${required_ports[@]}"; do
        if is_port_in_use "$port"; then
            fail "Required port $port is in use"
        fi
    done
    
    pass "All required ports are free"
}

run_tests
```

## Test Framework

### Core Testing Functions

```bash
#!/bin/bash
# test_framework.sh - Testing utility functions

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Assertions
assert_equals() {
    local actual="$1"
    local expected="$2"
    local message="${3:-Assertion}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [[ "$actual" == "$expected" ]]; then
        pass "$message"
    else
        fail "$message: expected '$expected', got '$actual'"
    fi
}

assert_true() {
    local command="$1"
    local message="${2:-Assertion}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if eval "$command"; then
        pass "$message"
    else
        fail "$message: command returned false"
    fi
}

assert_false() {
    local command="$1"
    local message="${2:-Assertion}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if ! eval "$command"; then
        pass "$message"
    else
        fail "$message: command returned true"
    fi
}

assert_contains() {
    local haystack="$1"
    local needle="$2"
    local message="${3:-Assertion}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [[ "$haystack" == *"$needle"* ]]; then
        pass "$message"
    else
        fail "$message: '$needle' not found in output"
    fi
}

assert_file_exists() {
    local file="$1"
    local message="${2:-File should exist}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    if [[ -f "$file" ]]; then
        pass "$message"
    else
        fail "$message: $file not found"
    fi
}

# Test results
pass() {
    local message="$1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "${COLOR_GREEN}✓${COLOR_RESET} $message"
}

fail() {
    local message="$1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "${COLOR_RED}✗${COLOR_RESET} $message"
    return 1
}

skip() {
    local message="$1"
    TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
    echo -e "${COLOR_YELLOW}⊘${COLOR_RESET} $message (skipped)"
}

# Test runner
run_tests() {
    # Find all test functions
    local test_functions=($(declare -F | awk '{print $3}' | grep '^test_'))
    
    echo "Running ${#test_functions[@]} tests..."
    echo
    
    for test_func in "${test_functions[@]}"; do
        echo "→ $test_func"
        $test_func
    done
    
    # Summary
    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Test Results:"
    echo "  Passed:  $TESTS_PASSED"
    echo "  Failed:  $TESTS_FAILED"
    echo "  Skipped: $TESTS_SKIPPED"
    echo "  Total:   $TESTS_RUN"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    [[ $TESTS_FAILED -eq 0 ]]
}
```

## Testing Patterns

### 1. Function Testing Pattern
```bash
# Test individual function with various inputs
test_function_name() {
    # Setup
    local test_data="test input"
    
    # Execute
    local result=$(function_to_test "$test_data")
    
    # Assert
    assert_equals "$result" "expected output" "Function produces correct output"
    
    # Cleanup
    cleanup_test_data
}
```

### 2. Error Handling Testing
```bash
test_error_handling() {
    # Test function handles errors gracefully
    local result=$(function_that_might_fail "bad input" 2>&1) || true
    
    assert_contains "$result" "ERROR" "Should show error message"
    assert_false "function_that_might_fail 'bad input'" "Should return non-zero"
}
```

### 3. Mock/Stub Pattern
```bash
# Create mock function for testing
mock_docker_command() {
    echo "mock output"
    return 0
}

test_with_mock() {
    # Override real command with mock
    alias docker='mock_docker_command'
    
    # Test function that uses docker
    local result=$(function_using_docker)
    
    # Assert
    assert_equals "$result" "expected" "Function works with mock"
    
    # Restore
    unalias docker
}
```

### 4. Fixture Testing
```bash
test_with_fixtures() {
    # Load test data
    local config_file="fixtures/test_configs/valid.env"
    
    # Test configuration loading
    load_configuration "$config_file" || fail "Failed to load fixture"
    
    # Verify loaded values
    assert_equals "$BASE_DOMAIN" "test.local" "Domain loaded correctly"
}
```

## Test Organization

### Naming Conventions
- Test files: `test_<module>.sh`
- Test functions: `test_<functionality>()`
- Fixtures: Descriptive names indicating content
- Mocks: `mock_<original_name>()`

### Test Grouping
```bash
# Group related tests in same file
# test_validation.sh contains:
test_is_valid_domain()
test_is_valid_email()
test_is_valid_port()
test_is_valid_url()
```

### Test Dependencies
```bash
# Declare dependencies at file top
# Requires: docker, network connectivity
# Depends on: shared/utils/docker.sh

test_docker_operations() {
    require_command "docker" || skip "Docker not available"
    require_network || skip "Network not available"
    
    # Run tests
}
```

## Continuous Integration

### Pre-commit Testing
```bash
#!/bin/bash
# .git/hooks/pre-commit

# Run sanity tests before commit
bin/tests/run_tests.sh --sanity || {
    echo "Sanity tests failed. Commit aborted."
    exit 1
}
```

### Full Test Suite
```bash
#!/bin/bash
# run_tests.sh - Main test runner

# Parse options
while [[ $# -gt 0 ]]; do
    case $1 in
        --unit)
            RUN_UNIT=true
            shift
            ;;
        --integration)
            RUN_INTEGRATION=true
            shift
            ;;
        --sanity)
            RUN_SANITY=true
            shift
            ;;
        --all)
            RUN_ALL=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Default to all if nothing specified
if [[ -z "$RUN_UNIT" && -z "$RUN_INTEGRATION" && -z "$RUN_SANITY" ]]; then
    RUN_ALL=true
fi

# Run selected test suites
if [[ "$RUN_ALL" == "true" || "$RUN_SANITY" == "true" ]]; then
    echo "Running sanity tests..."
    run_test_suite "sanity"
fi

if [[ "$RUN_ALL" == "true" || "$RUN_UNIT" == "true" ]]; then
    echo "Running unit tests..."
    run_test_suite "unit"
fi

if [[ "$RUN_ALL" == "true" || "$RUN_INTEGRATION" == "true" ]]; then
    echo "Running integration tests..."
    run_test_suite "integration"
fi
```

## Test Coverage

### Coverage Goals
- **Unit Tests**: 90% function coverage
- **Integration Tests**: All critical paths
- **Sanity Tests**: All prerequisites
- **Error Paths**: 80% error handling coverage

### Coverage Reporting
```bash
# Generate coverage report
generate_coverage_report() {
    local total_functions=$(grep -r "^function\|^[a-z_]*(" bin/ | wc -l)
    local tested_functions=$(grep -r "^test_" bin/tests/ | wc -l)
    local coverage=$((tested_functions * 100 / total_functions))
    
    echo "Test Coverage: $coverage%"
    echo "Functions: $tested_functions/$total_functions"
}
```

## Best Practices

### 1. Test Independence
- Each test should be independent
- No reliance on test execution order
- Clean up after each test
- Use fresh test data

### 2. Clear Test Names
```bash
# Good test names
test_valid_domain_accepts_subdomains()
test_invalid_port_returns_error()
test_docker_startup_with_custom_config()

# Bad test names
test1()
test_function()
test_thing_works()
```

### 3. Descriptive Assertions
```bash
# Good assertions with context
assert_equals "$result" "expected" "Function should handle empty input"

# Bad assertions without context
assert_equals "$result" "expected"
```

### 4. Test Documentation
```bash
# Document complex test scenarios
test_complex_workflow() {
    # This test verifies that when a service fails to start,
    # the system properly rolls back all changes and notifies
    # the user with actionable error messages
    
    # Setup: Create invalid configuration
    # Execute: Attempt service startup
    # Verify: Rollback occurred
    # Verify: Error message shown
}
```

## Debugging Tests

### Verbose Mode
```bash
# Run tests with verbose output
DEBUG=true bin/tests/run_tests.sh

# Or for specific test
DEBUG=true bin/tests/unit/utils/test_validation.sh
```

### Test Isolation
```bash
# Run single test function
bin/tests/run_single_test.sh test_is_valid_domain
```

### Test Fixtures
```bash
# Create reproducible test environment
setup_test_environment() {
    export TEST_MODE=true
    export NSELF_CONFIG_DIR="/tmp/nself_test_$$"
    mkdir -p "$NSELF_CONFIG_DIR"
    trap "rm -rf $NSELF_CONFIG_DIR" EXIT
}
```

## Summary

This testing strategy enables:
- **Modular Development**: Build with confidence
- **Early Bug Detection**: Catch issues at unit level
- **Faster Development**: Less debugging time
- **Better Code Quality**: Tests enforce good design
- **Documentation**: Tests show how to use functions

By following this strategy, we can refactor and extend NSELF with confidence, knowing that our foundation is solid and tested.