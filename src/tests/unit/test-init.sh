#!/usr/bin/env bash
# test-init-unit.sh - Comprehensive unit tests for init modules
#
# This script tests individual functions in isolation using mocks and stubs

set -euo pipefail

# Test framework setup
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Colors for output
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
RESET="\033[0m"

# Test directories
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INIT_LIB_DIR="$TEST_DIR/../../lib/init"
TEMP_TEST_DIR=""

# ============================================================================
# Test Framework Functions
# ============================================================================

setup_test_env() {
  TEMP_TEST_DIR=$(mktemp -d 2>/dev/null) || TEMP_TEST_DIR="/tmp/nself-test-$$"
  mkdir -p "$TEMP_TEST_DIR"
  cd "$TEMP_TEST_DIR"

  # Mock environment variables
  export OS="test"
  export SUPPORTS_COLOR=true
  export SUPPORTS_UNICODE=true
  export CHECK_MARK="✓"
  export COLOR_GREEN="\033[32m"
  export COLOR_RESET="\033[0m"
}

teardown_test_env() {
  cd /
  [[ -n "$TEMP_TEST_DIR" ]] && rm -rf "$TEMP_TEST_DIR"
}

assert_equals() {
  local expected="$1"
  local actual="$2"
  local message="${3:-Values should be equal}"

  if [[ "$expected" == "$actual" ]]; then
    return 0
  else
    echo -e "${RED}✗ $message${RESET}"
    echo "  Expected: $expected"
    echo "  Actual: $actual"
    return 1
  fi
}

assert_file_exists() {
  local file="$1"
  local message="${2:-File should exist: $file}"

  if [[ -f "$file" ]]; then
    return 0
  else
    echo -e "${RED}✗ $message${RESET}"
    return 1
  fi
}

assert_file_permissions() {
  local file="$1"
  local expected_perms="$2"
  local message="${3:-File should have permissions $expected_perms}"

  local actual_perms
  if [[ -f "$file" ]]; then
    actual_perms=$(stat -c "%a" "$file" 2>/dev/null || stat -f "%OLp" "$file" 2>/dev/null)
    if [[ "$actual_perms" == "$expected_perms" ]]; then
      return 0
    fi
  fi

  echo -e "${RED}✗ $message${RESET}"
  echo "  File: $file"
  echo "  Expected permissions: $expected_perms"
  echo "  Actual permissions: ${actual_perms:-file not found}"
  return 1
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="${3:-String should contain substring}"

  if [[ "$haystack" == *"$needle"* ]]; then
    return 0
  else
    echo -e "${RED}✗ $message${RESET}"
    echo "  Looking for: $needle"
    echo "  In: ${haystack:0:100}..."
    return 1
  fi
}

run_test() {
  local test_name="$1"
  local test_function="$2"

  TESTS_RUN=$((TESTS_RUN + 1))
  echo -n "Testing $test_name... "

  # Setup clean environment for each test
  setup_test_env

  # Run the test
  if $test_function; then
    echo -e "${GREEN}PASSED${RESET}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
  else
    echo -e "${RED}FAILED${RESET}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
  fi

  # Cleanup
  teardown_test_env
}

skip_test() {
  local test_name="$1"
  local reason="${2:-No reason given}"

  TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
  echo -e "${YELLOW}SKIPPING${RESET} $test_name: $reason"
}

# ============================================================================
# Unit Tests for config.sh
# ============================================================================

test_config_constants() {
  source "$INIT_LIB_DIR/config.sh"

  assert_equals "0" "$INIT_E_SUCCESS" "Success error code should be 0"
  assert_equals "73" "$INIT_E_CANTCREAT" "Can't create error code should be 73"
  assert_equals "600" "$INIT_PERM_PRIVATE" "Private permissions should be 600"
  assert_equals "644" "$INIT_PERM_PUBLIC" "Public permissions should be 644"
}

test_config_arrays() {
  source "$INIT_LIB_DIR/config.sh"

  # Test gitignore required array
  local found_env=false
  if [[ -n "${INIT_GITIGNORE_REQUIRED+x}" ]]; then
    for entry in "${INIT_GITIGNORE_REQUIRED[@]}"; do
      if [[ "$entry" == ".env" ]]; then
        found_env=true
        break
      fi
    done
  fi

  assert_equals "true" "$found_env" ".env should be in required gitignore entries"
}

# ============================================================================
# Unit Tests for platform.sh
# ============================================================================

test_platform_detection() {
  source "$INIT_LIB_DIR/platform.sh"

  # Test OS detection
  OSTYPE="linux-gnu"
  detect_platform
  assert_equals "linux" "$OS" "Should detect Linux"

  OSTYPE="darwin20"
  detect_platform
  assert_equals "macos" "$OS" "Should detect macOS"

  OSTYPE="msys"
  detect_platform
  assert_equals "windows-bash" "$OS" "Should detect Windows bash"
}

test_safe_echo() {
  source "$INIT_LIB_DIR/platform.sh"

  # Test safe_echo output
  local output=$(safe_echo "test\nline")
  assert_contains "$output" "test" "safe_echo should output text"

  # Test with color codes
  local color_output=$(safe_echo "${COLOR_GREEN}test${COLOR_RESET}")
  assert_contains "$color_output" "test" "safe_echo should handle colors"
}

test_terminal_capabilities() {
  source "$INIT_LIB_DIR/platform.sh"

  # Test Unicode detection
  TERM="xterm-256color"
  LANG="en_US.UTF-8"
  detect_terminal
  assert_equals "true" "$SUPPORTS_UNICODE" "Should support Unicode in xterm with UTF-8"

  # Test color detection
  assert_equals "true" "$SUPPORTS_COLOR" "Should support colors in xterm-256color"

  # Test NO_COLOR override
  NO_COLOR=1
  detect_terminal
  assert_equals "false" "$SUPPORTS_COLOR" "Should respect NO_COLOR env var"
  unset NO_COLOR
}

# ============================================================================
# Unit Tests for atomic-ops.sh
# ============================================================================

test_atomic_copy() {
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/atomic-ops.sh"

  # Create test file
  echo "test content" > test_source.txt

  # Test atomic copy
  atomic_copy "test_source.txt" "test_dest.txt" "644"

  assert_file_exists "test_dest.txt" "Atomic copy should create destination"

  # Check content
  local content=$(cat test_dest.txt)
  assert_equals "test content" "$content" "Content should match source"

  # Test that it was tracked
  local tracked=false
  for file in "${CREATED_FILES[@]}"; do
    if [[ "$file" == "test_dest.txt" ]]; then
      tracked=true
      break
    fi
  done
  assert_equals "true" "$tracked" "Created file should be tracked"
}

test_rollback() {
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/atomic-ops.sh"

  # Create and track files
  echo "file1" > file1.txt
  CREATED_FILES=("file1.txt")

  echo "original" > file2.txt
  echo "modified" > file2.txt.nself-backup
  MODIFIED_FILES=("file2.txt")

  # Test rollback
  rollback_changes

  assert_equals "false" "$(test -f file1.txt && echo true || echo false)" \
    "Created files should be removed"

  assert_file_exists "file2.txt" "Modified files should be restored"
  local content=$(cat file2.txt 2>/dev/null || echo "missing")
  assert_equals "modified" "$content" "File should be restored from backup"
}

# ============================================================================
# Unit Tests for templates.sh
# ============================================================================

test_find_templates_dir() {
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/platform.sh"
  source "$INIT_LIB_DIR/templates.sh"

  # Mock script directory
  local test_script_dir="$TEMP_TEST_DIR/cli"
  mkdir -p "$test_script_dir/../templates"

  local found_dir=$(find_templates_dir "$test_script_dir" 2>/dev/null || echo "not_found")
  assert_contains "$found_dir" "templates" "Should find templates directory"
}

test_verify_template_files() {
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/platform.sh"
  source "$INIT_LIB_DIR/templates.sh"

  # Create test templates
  mkdir -p templates
  touch templates/.env templates/.env.example

  # Test verification passes
  verify_template_files "templates" ".env" ".env.example"
  assert_equals "0" "$?" "Should verify existing templates"

  # Test verification fails
  verify_template_files "templates" "nonexistent" 2>/dev/null
  assert_equals "78" "$?" "Should fail for missing templates"
}

# ============================================================================
# Unit Tests for gitignore.sh
# ============================================================================

test_gitignore_has_entry() {
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/platform.sh"
  source "$INIT_LIB_DIR/gitignore.sh"

  # Create test gitignore
  cat > .gitignore << 'EOF'
.env
*.log
node_modules/
EOF

  gitignore_has_entry ".env"
  assert_equals "0" "$?" "Should find exact match"

  gitignore_has_entry "*.log"
  assert_equals "0" "$?" "Should find wildcard entry"

  gitignore_has_entry "missing"
  assert_equals "1" "$?" "Should not find missing entry"
}

test_create_gitignore() {
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/platform.sh"
  source "$INIT_LIB_DIR/gitignore.sh"

  # Initialize tracking arrays
  CREATED_FILES=()

  # Test creation
  create_gitignore >/dev/null 2>&1

  assert_file_exists ".gitignore" "Should create .gitignore"

  # Check required entries
  local content=$(cat .gitignore)
  assert_contains "$content" ".env" "Should contain .env"
  assert_contains "$content" ".env.secrets" "Should contain .env.secrets"
  assert_contains "$content" "node_modules/" "Should contain node_modules/"
}

# ============================================================================
# Unit Tests for validation.sh
# ============================================================================

test_check_dependencies() {
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/platform.sh"
  source "$INIT_LIB_DIR/validation.sh"

  # Test with all commands available (should pass on most systems)
  check_dependencies 2>/dev/null
  local result=$?

  # We expect this to pass on CI systems
  if command -v git >/dev/null 2>&1; then
    assert_equals "0" "$result" "Should pass when dependencies exist"
  else
    skip_test "check_dependencies" "Git not available in test environment"
  fi
}

test_validate_env_mode() {
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/validation.sh"

  validate_env_mode "dev"
  assert_equals "0" "$?" "Should accept 'dev' mode"

  validate_env_mode "prod"
  assert_equals "0" "$?" "Should accept 'prod' mode"

  validate_env_mode "invalid" 2>/dev/null
  assert_equals "1" "$?" "Should reject invalid mode"
}

# ============================================================================
# Integration Tests
# ============================================================================

test_init_integration() {
  # Source all modules
  source "$INIT_LIB_DIR/config.sh"
  source "$INIT_LIB_DIR/platform.sh"
  source "$INIT_LIB_DIR/atomic-ops.sh"
  source "$INIT_LIB_DIR/templates.sh"
  source "$INIT_LIB_DIR/gitignore.sh"
  source "$INIT_LIB_DIR/validation.sh"

  # Mock templates directory
  mkdir -p "$TEMP_TEST_DIR/templates"
  echo "# Test .env" > "$TEMP_TEST_DIR/templates/.env"
  echo "# Test .env.example" > "$TEMP_TEST_DIR/templates/.env.example"

  # Initialize arrays
  CREATED_FILES=()
  MODIFIED_FILES=()

  # Test basic template copy
  copy_basic_templates "$TEMP_TEST_DIR/templates" true

  assert_file_exists ".env" "Should create .env"
  assert_file_exists ".env.example" "Should create .env.example"
}

# ============================================================================
# Run All Tests
# ============================================================================

echo "╔══════════════════════════════════════════════════════════╗"
echo "║           nself init Unit Test Suite                     ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo ""

# Config tests
echo -e "${BLUE}Testing config.sh...${RESET}"
run_test "config constants" test_config_constants
run_test "config arrays" test_config_arrays

# Platform tests
echo -e "\n${BLUE}Testing platform.sh...${RESET}"
run_test "platform detection" test_platform_detection
run_test "safe_echo function" test_safe_echo
run_test "terminal capabilities" test_terminal_capabilities

# Atomic ops tests
echo -e "\n${BLUE}Testing atomic-ops.sh...${RESET}"
run_test "atomic copy" test_atomic_copy
run_test "rollback changes" test_rollback

# Templates tests
echo -e "\n${BLUE}Testing templates.sh...${RESET}"
run_test "find templates directory" test_find_templates_dir
run_test "verify template files" test_verify_template_files

# Gitignore tests
echo -e "\n${BLUE}Testing gitignore.sh...${RESET}"
run_test "gitignore has entry" test_gitignore_has_entry
run_test "create gitignore" test_create_gitignore

# Validation tests
echo -e "\n${BLUE}Testing validation.sh...${RESET}"
run_test "check dependencies" test_check_dependencies
run_test "validate env mode" test_validate_env_mode

# Integration tests
echo -e "\n${BLUE}Testing integration...${RESET}"
run_test "init integration" test_init_integration

# Summary
echo ""
echo "╔══════════════════════════════════════════════════════════╗"
echo "║                    Test Summary                          ║"
echo "╚══════════════════════════════════════════════════════════╝"
echo -e "Tests run: $TESTS_RUN"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${RESET}"
if [[ $TESTS_FAILED -gt 0 ]]; then
  echo -e "Tests failed: ${RED}$TESTS_FAILED${RESET}"
fi
if [[ $TESTS_SKIPPED -gt 0 ]]; then
  echo -e "Tests skipped: ${YELLOW}$TESTS_SKIPPED${RESET}"
fi

# Exit code
if [[ $TESTS_FAILED -gt 0 ]]; then
  exit 1
else
  echo -e "\n${GREEN}All tests passed!${RESET}"
  exit 0
fi