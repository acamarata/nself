#!/usr/bin/env bash
# test-init-integration.sh - Integration tests for nself init command
#
# These tests verify the full init workflow in realistic scenarios

set -euo pipefail

# Source test framework
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$TEST_DIR/../test_framework.sh"
source "$TEST_DIR/../helpers/mock-helpers.sh"

# Paths
CLI_DIR="$TEST_DIR/../../cli"
LIB_DIR="$TEST_DIR/../../lib/init"

# ============================================================================
# Integration Tests
# ============================================================================

test_fresh_project_init() {
  describe "Fresh project initialization"
  
  # Create temp directory
  local temp_dir="/tmp/nself-test-fresh-$$"
  mkdir -p "$temp_dir"
  cd "$temp_dir"
  
  # Run init
  run bash "$CLI_DIR/init.sh" --quiet
  
  # Verify all files created
  assert_file_exists ".env" "Should create .env file"
  assert_file_exists ".env.example" "Should create .env.example file"
  assert_file_exists ".gitignore" "Should create .gitignore file"
  
  # Verify .env contents
  assert_file_contains ".env" "PROJECT_NAME=" ".env should contain PROJECT_NAME"
  assert_file_contains ".env" "NODE_ENV=" ".env should contain NODE_ENV"
  
  # Verify .gitignore contents
  assert_file_contains ".gitignore" ".env" ".gitignore should include .env"
  assert_file_contains ".gitignore" "node_modules/" ".gitignore should include node_modules/"
  
  # Verify permissions
  assert_file_permissions ".env" "600" ".env should have 600 permissions"
  assert_file_permissions ".env.example" "644" ".env.example should have 644 permissions"
  
  # Cleanup
  cd /
  rm -rf "$temp_dir"
}

test_reinit_with_force() {
  describe "Re-initialization with --force flag"
  
  local temp_dir="/tmp/nself-test-force-$$"
  mkdir -p "$temp_dir"
  cd "$temp_dir"
  
  # First init
  run bash "$CLI_DIR/init.sh" --quiet
  
  # Modify .env
  echo "CUSTOM_VAR=test" >> .env
  local original_content=$(cat .env)
  
  # Try reinit without force (should fail)
  run_expect_fail bash "$CLI_DIR/init.sh" --quiet
  
  # Verify .env unchanged
  local current_content=$(cat .env)
  assert_equals "$original_content" "$current_content" ".env should remain unchanged without --force"
  
  # Reinit with force
  run bash "$CLI_DIR/init.sh" --force --quiet
  
  # Verify .env was recreated (custom var gone)
  assert_file_not_contains ".env" "CUSTOM_VAR" "Custom variables should be removed after force reinit"
  
  # Cleanup
  cd /
  rm -rf "$temp_dir"
}

test_wizard_mode() {
  describe "Wizard mode initialization"
  
  local temp_dir="/tmp/nself-test-wizard-$$"
  mkdir -p "$temp_dir"
  cd "$temp_dir"
  
  # Create input for wizard
  cat > wizard_input.txt << 'EOF'
MyTestProject
dev
y
y
n
n
y
EOF
  
  # Run wizard with piped input
  run bash "$CLI_DIR/init.sh" --wizard < wizard_input.txt
  
  # Verify project name in .env
  assert_file_contains ".env" "PROJECT_NAME=MyTestProject" "Project name should be set from wizard"
  assert_file_contains ".env" "NODE_ENV=dev" "Environment should be dev from wizard"
  
  # Cleanup
  cd /
  rm -rf "$temp_dir"
}

test_gitignore_update() {
  describe "Gitignore update functionality"
  
  local temp_dir="/tmp/nself-test-gitignore-$$"
  mkdir -p "$temp_dir"
  cd "$temp_dir"
  
  # Create existing .gitignore with some entries
  cat > .gitignore << 'EOF'
# Existing entries
*.log
*.tmp
EOF
  
  # Run init
  run bash "$CLI_DIR/init.sh" --quiet
  
  # Verify original entries preserved
  assert_file_contains ".gitignore" "*.log" "Original .gitignore entries should be preserved"
  assert_file_contains ".gitignore" "*.tmp" "Original .gitignore entries should be preserved"
  
  # Verify new entries added
  assert_file_contains ".gitignore" ".env" "Required .env entry should be added"
  assert_file_contains ".gitignore" "node_modules/" "Required node_modules/ entry should be added"
  
  # Verify no duplicates
  local env_count=$(grep -c "^\.env$" .gitignore)
  assert_equals "1" "$env_count" "Should not have duplicate .env entries"
  
  # Cleanup
  cd /
  rm -rf "$temp_dir"
}

test_error_handling() {
  describe "Error handling and recovery"
  
  local temp_dir="/tmp/nself-test-error-$$"
  mkdir -p "$temp_dir"
  cd "$temp_dir"
  
  # Make directory read-only to cause error
  chmod 555 .
  
  # Try to run init (should fail gracefully)
  run_expect_fail bash "$CLI_DIR/init.sh" --quiet
  
  # Restore permissions
  chmod 755 .
  
  # Verify no partial files created
  assert_file_not_exists ".env" "No .env should be created on error"
  assert_file_not_exists ".env.example" "No .env.example should be created on error"
  
  # Now run successfully
  run bash "$CLI_DIR/init.sh" --quiet
  
  # Verify all files created
  assert_file_exists ".env" "Should create .env after fixing permissions"
  assert_file_exists ".env.example" "Should create .env.example after fixing permissions"
  
  # Cleanup
  cd /
  rm -rf "$temp_dir"
}

test_git_repository_init() {
  describe "Initialization in git repository"
  
  local temp_dir="/tmp/nself-test-git-$$"
  mkdir -p "$temp_dir"
  cd "$temp_dir"
  
  # Initialize git repo
  git init --initial-branch=main >/dev/null 2>&1 || git init >/dev/null 2>&1
  
  # Run init
  run bash "$CLI_DIR/init.sh" --quiet
  
  # Check git status
  local untracked=$(git status --porcelain 2>/dev/null | grep "^??" | wc -l)
  
  # .env should not be tracked (in .gitignore)
  run git status --porcelain .env
  assert_equals "" "$TEST_OUTPUT" ".env should be ignored by git"
  
  # .env.example should be untracked
  run git status --porcelain .env.example
  assert_contains "$TEST_OUTPUT" "??" ".env.example should be untracked"
  
  # Cleanup
  cd /
  rm -rf "$temp_dir"
}

test_parallel_execution() {
  describe "Parallel execution safety"
  
  local temp_dir="/tmp/nself-test-parallel-$$"
  mkdir -p "$temp_dir"
  cd "$temp_dir"
  
  # Run multiple inits in parallel
  (
    bash "$CLI_DIR/init.sh" --quiet &
    bash "$CLI_DIR/init.sh" --quiet &
    bash "$CLI_DIR/init.sh" --quiet &
    wait
  ) 2>/dev/null
  
  # Verify files exist and are valid
  assert_file_exists ".env" "Should have .env after parallel execution"
  assert_file_exists ".env.example" "Should have .env.example after parallel execution"
  
  # Verify file integrity
  assert_file_contains ".env" "PROJECT_NAME=" ".env should be intact after parallel execution"
  
  # Cleanup
  cd /
  rm -rf "$temp_dir"
}

# ============================================================================
# Run Tests
# ============================================================================

run_integration_tests() {
  print_test_header "nself init Integration Tests"
  
  # Run all test functions
  test_fresh_project_init
  test_reinit_with_force
  test_wizard_mode
  test_gitignore_update
  test_error_handling
  test_git_repository_init
  test_parallel_execution
  
  print_test_summary
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  run_integration_tests
fi