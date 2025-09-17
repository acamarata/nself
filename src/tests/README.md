# nself Test Suite

## Structure

```
src/tests/
├── unit/                 # Unit tests for individual functions
│   └── test-init.sh      # Unit tests for init command
├── integration/          # Integration tests for workflows
│   └── test-init-integration.sh
├── helpers/              # Test utilities and mocks
│   └── mock-helpers.sh   # Mock commands and stubs
├── test_framework.sh     # Core test framework
├── run-all-tests.sh      # Main test runner
├── run-init-tests.sh     # Init-specific test runner
└── test-*.sh             # Legacy/feature tests
```

## Running Tests

### Run all tests
```bash
cd src/tests
./run-all-tests.sh
```

### Run only unit tests (quick)
```bash
./run-all-tests.sh --quick
```

### Run specific test category
```bash
./run-all-tests.sh --filter init
```

### Run with verbose output
```bash
./run-all-tests.sh --verbose
```

### Run init tests specifically
```bash
./run-init-tests.sh
```

## CI/CD

Tests are automatically run via GitHub Actions:
- `.github/workflows/test-init.yml` - Init command tests
- Runs on: Ubuntu, macOS
- Bash versions: 3.2, latest

## Test Framework

The test framework (`test_framework.sh`) provides:
- `assert_equals` - Check equality
- `assert_not_equals` - Check inequality
- `assert_contains` - Check string contains substring
- `assert_file_exists` - Check file exists
- `assert_file_contains` - Check file content
- `assert_file_permissions` - Check file permissions
- `run` - Run command and capture output
- `describe` - Describe test context

## Mock Helpers

The mock helpers (`helpers/mock-helpers.sh`) provide:
- Command mocking (git, docker, stat, etc.)
- Stub functions for external dependencies
- Spy functions to track calls
- Input/output mocking
- Test environment setup

## Writing Tests

### Unit Test Example
```bash
test_my_function() {
  describe "My function test"
  
  # Setup
  source "$LIB_DIR/my-module.sh"
  
  # Test
  result=$(my_function "input")
  
  # Assert
  assert_equals "expected" "$result" "Should return expected value"
}
```

### Integration Test Example
```bash
test_workflow() {
  describe "Complete workflow test"
  
  # Setup temp environment
  local temp_dir="/tmp/test-$$"
  mkdir -p "$temp_dir"
  cd "$temp_dir"
  
  # Run workflow
  run bash "$CLI_DIR/command.sh" --option
  
  # Verify results
  assert_file_exists "output.txt"
  
  # Cleanup
  cd /
  rm -rf "$temp_dir"
}
```

## Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always clean up temp files/directories
3. **Descriptive**: Use clear test descriptions
4. **Fast**: Unit tests should be fast (<1s)
5. **Reliable**: Tests should not be flaky
6. **Portable**: Tests must work on Linux/macOS
7. **Bash 3.2**: Compatible with Bash 3.2+

## Coverage

### Init Command
- ✓ Unit tests for all modules
- ✓ Integration tests for workflows
- ✓ Cross-platform compatibility
- ✓ Error handling and recovery
- ✓ File permissions
- ✓ Git integration

### Other Commands
- ☐ build
- ☐ start
- ☐ stop
- ☐ status
- ☐ logs
- ☐ reset
- ☐ update

## Troubleshooting

### Tests fail on macOS
- Check Bash version: `bash --version`
- Ensure GNU coreutils installed: `brew install coreutils`

### Permission errors
- Check file permissions: `ls -la`
- Run with proper user (not root)

### Path issues
- Tests assume running from `src/tests` directory
- Use absolute paths when possible